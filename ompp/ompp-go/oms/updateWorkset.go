// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"encoding/csv"
	"errors"
	"io"
	"sort"
	"strings"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// UpdateWorksetReadonly update workset read-only status by model digest-or-name and workset name.
func (mc *ModelCatalog) UpdateWorksetReadonly(dn, wsn string, isReadonly bool) (string, *db.WorksetRow, bool, error) {

	// if model digest-or-name or workset name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return "", &db.WorksetRow{}, false, nil
	}
	if wsn == "" {
		omppLog.Log("Warning: invalid (empty) workset name")
		return "", &db.WorksetRow{}, false, nil
	}
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return "", &db.WorksetRow{}, false, nil // return empty result: model not found or error
	}

	// find workset in database
	w, err := db.GetWorksetByName(dbConn, meta.Model.ModelId, wsn)
	if err != nil {
		omppLog.Log("Error at get workset status: ", dn, ": ", wsn, ": ", err.Error())
		return "", &db.WorksetRow{}, false, err // return empty result: workset select error
	}
	if w == nil {
		omppLog.Log("Warning: workset not found: ", dn, ": ", wsn)
		return "", &db.WorksetRow{}, false, nil // return empty result: workset_lst row not found
	}

	// update workset readonly status
	err = db.UpdateWorksetReadonly(dbConn, w.SetId, isReadonly)
	if err != nil {
		omppLog.Log("Error at update workset status: ", dn, ": ", wsn, ": ", err.Error())
		return "", &db.WorksetRow{}, false, err // return empty result: workset select error
	}

	// get workset status
	w, err = db.GetWorkset(dbConn, w.SetId)
	if err != nil {
		omppLog.Log("Error at get workset status: ", dn, ": ", w.SetId, ": ", err.Error())
		return "", &db.WorksetRow{}, false, err // return empty result: workset select error
	}
	if w == nil {
		omppLog.Log("Warning workset status not found: ", dn, ": ", wsn)
		return "", &db.WorksetRow{}, false, nil // return empty result: workset_lst row not found
	}

	return meta.Model.Digest, w, true, nil
}

// UpdateWorkset update workset metadata: create new workset, replace existsing or merge metadata.
// Return: isUpdated true/false flag, isEraseParam true/false warining flag, workset_lst db row and error
func (mc *ModelCatalog) UpdateWorkset(isReplace bool, wp *db.WorksetPub) (bool, bool, *db.WorksetRow, error) {

	// if model digest-or-name or workset name is empty then return empty results
	dn := wp.ModelDigest
	if dn == "" {
		dn = wp.ModelName
	}
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return false, false, nil, nil
	}
	if wp.Name == "" {
		omppLog.Log("Warning: invalid (empty) workset name")
		return false, false, nil, nil
	}

	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Error: model digest or name not found: ", dn)
		return false, false, nil, errors.New("Error: model digest or name not found: " + dn)
	}
	langMeta := mc.modelLangMeta(dn)
	if langMeta == nil {
		omppLog.Log("Error: invalid (empty) model language list: ", dn)
		return false, false, nil, errors.New("Error: invalid (empty) model language list: " + dn)
	}

	// find workset in database: it must be read-write if exists
	w, err := db.GetWorksetByName(dbConn, meta.Model.ModelId, wp.Name)
	if err != nil {
		omppLog.Log("Error at get workset status: ", dn, ": ", wp.Name, ": ", err.Error())
		return false, false, nil, err
	}
	if w != nil && w.IsReadonly {
		omppLog.Log("Failed to update read-only workset: ", dn, ": ", wp.Name)
		return false, false, nil, errors.New("Failed to update read-only workset: " + dn + ": " + wp.Name)
	}

	// if workset does not exist then clean paramters list to create empty workset
	isEraseParam := w == nil && len(wp.Param) > 0
	if isEraseParam {
		wp.Param = []db.ParamRunSetPub{}
		omppLog.Log("Warning: existing workset not found, create new empty workset (without parameters): ", wp.Name)
	}

	// convert workset from "public" into db rows
	wm, err := wp.FromPublic(dbConn, meta)
	if err != nil {
		omppLog.Log("Error at workset json conversion: ", dn, ": ", wp.Name, ": ", err.Error())
		return false, isEraseParam, nil, err
	}

	// check if base run exist
	if wp.BaseRunDigest != "" && wm.Set.BaseRunId <= 0 {
		omppLog.Log("Error at update workset, base run not found: ", dn, ": ", wp.Name, ": ", wp.BaseRunDigest)
		return false, isEraseParam, nil, errors.New("Failed to update workset, base run not found: " + dn + ": " + wp.Name + ": " + wp.BaseRunDigest)
	}

	// match languages from request into model languages
	for k := range wm.Txt {
		lc := mc.languageCodeMatch(dn, wm.Txt[k].LangCode)
		if lc != "" {
			wm.Txt[k].LangCode = lc
		}
	}
	for k := range wm.Param {
		for j := range wm.Param[k].Txt {
			lc := mc.languageCodeMatch(dn, wm.Param[k].Txt[j].LangCode)
			if lc != "" {
				wm.Param[k].Txt[j].LangCode = lc
			}
		}
	}

	// update workset metadata
	err = wm.UpdateWorkset(dbConn, meta, isReplace, langMeta)
	if err != nil {
		omppLog.Log("Error at update workset: ", dn, ": ", wp.Name, ": ", err.Error())
		return false, isEraseParam, nil, err
	}

	// get updated workset status: it must exist
	w, err = db.GetWorksetByName(dbConn, meta.Model.ModelId, wp.Name)
	if err != nil {
		omppLog.Log("Error at update workset: ", dn, ": ", wp.Name, ": ", err.Error())
		return false, isEraseParam, nil, err
	}
	if w == nil {
		omppLog.Log("Error at update workset, it does not exist: ", dn, ": ", wp.Name)
		return false, isEraseParam, nil, errors.New("Failed to update workset, it does not exist: " + dn + ": " + wp.Name)
	}

	return true, isEraseParam, w, nil
}

// DeleteWorkset do delete workset, including parameter values from database.
func (mc *ModelCatalog) DeleteWorkset(dn, wsn string) (bool, error) {

	// if model digest-or-name or workset name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return false, nil
	}
	if wsn == "" {
		omppLog.Log("Warning: invalid (empty) workset name")
		return false, nil
	}
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Error: model digest or name not found: ", dn)
		return false, errors.New("Error: model digest or name not found: " + dn)
	}

	// find workset in database
	w, err := db.GetWorksetByName(dbConn, meta.Model.ModelId, wsn)
	if err != nil {
		omppLog.Log("Error at get workset status: ", dn, ": ", wsn, ": ", err.Error())
		return false, err
	}
	if w == nil {
		return false, nil // return OK: workset not found
	}

	// delete workset from database
	err = db.DeleteWorkset(dbConn, w.SetId)
	if err != nil {
		omppLog.Log("Error at delete workset: ", dn, ": ", wsn, ": ", err.Error())
		return false, err
	}

	return true, nil
}

// UpdateWorksetParameter replace or merge parameter metadata into workset and replace parameter values.
func (mc *ModelCatalog) UpdateWorksetParameter(
	isReplace bool, wp *db.WorksetPub, param *db.ParamRunSetPub, cArr []db.CellCodeParam,
) (
	bool, error) {

	// if model digest-or-name or workset name is empty then return empty results
	dn := wp.ModelDigest
	if dn == "" {
		dn = wp.ModelName
	}
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return false, nil
	}
	if wp.Name == "" {
		omppLog.Log("Warning: invalid (empty) workset name")
		return false, nil
	}
	if len(cArr) <= 0 {
		return false, errors.New("workset: " + wp.Name + " parameter empty: " + param.Name)
	}

	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Error: model digest or name not found: ", dn)
		return false, errors.New("Error: model digest or name not found: " + dn)
	}

	langMeta := mc.modelLangMeta(dn)
	if langMeta == nil {
		omppLog.Log("Error: invalid (empty) model language list: ", dn)
		return false, errors.New("Error: invalid (empty) model language list: " + dn)
	}

	// convert workset from "public" into db rows
	wm, err := wp.FromPublic(dbConn, meta)
	if err != nil {
		omppLog.Log("Error at workset json conversion: ", dn, ": ", wp.Name, ": ", err.Error())
		return false, err
	}

	// match languages from request into model languages
	for j := range param.Txt {
		lc := mc.languageCodeMatch(dn, param.Txt[j].LangCode)
		if lc != "" {
			param.Txt[j].LangCode = lc
		}
	}

	// convert cell from emun codes to enum id's
	csvCvt := db.CellParamConverter{
		ModelDef:  meta,
		Name:      param.Name,
		DoubleFmt: theCfg.doubleFmt,
	}
	cvt, e := csvCvt.CodeToIdCell(meta, param.Name)
	if e != nil {
		return false, errors.New("Failed to create parameter cell value converter: " + param.Name + " : " + e.Error())
	}

	// for each row convert parameter cell from code to enum id
	k := -1
	from := func() (interface{}, error) {
		k++
		if k >= len(cArr) {
			return nil, nil // end of data
		}

		c, e := cvt(cArr[k])
		if e != nil {
			return nil, errors.New("Failed to convert value of parameter: " + param.Name + " : " + e.Error())
		}
		return c, nil
	}

	// update workset parameter metadata and parameter values
	hId, err := wm.UpdateWorksetParameterFrom(dbConn, meta, isReplace, param, langMeta, from)
	if err != nil {
		omppLog.Log("Error at update workset: ", dn, ": ", wp.Name, ": ", err.Error())
		return false, err
	}
	return hId > 0, nil // return success and true if parameter was found
}

// UpdateWorksetParameterText do merge (insert or update) parameters value notes.
func (mc *ModelCatalog) UpdateWorksetParameterText(dn, wsn string, pvtLst []db.ParamRunSetTxtPub) (bool, error) {

	// validate parameters
	if pvtLst == nil || len(pvtLst) <= 0 {
		omppLog.Log("Warning: empty list of run parameters to update value notes")
		return false, nil
	}
	if dn == "" {
		return false, errors.New("Error: invalid (empty) model digest or name")
	}
	if wsn == "" {
		return false, errors.New("Error: invalid (empty) workset name")
	}

	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		return false, errors.New("Error: model digest or name not found: " + dn)
	}

	langMeta := mc.modelLangMeta(dn)
	if langMeta == nil {
		return false, errors.New("Error: invalid (empty) model language list: " + dn)
	}

	// find workset in database: it must be read-write if exists
	w, err := db.GetWorksetByName(dbConn, meta.Model.ModelId, wsn)
	if err != nil {
		return false, errors.New("Workset not found: " + dn + ": " + wsn + ": " + err.Error())
	}
	if w == nil {
		return false, errors.New("Workset not found: " + dn + ": " + wsn)
	}
	if w.IsReadonly {
		return false, errors.New("Failed to update read-only workset: " + dn + ": " + wsn)
	}

	// get workset parameters list
	hIds, _, _, err := db.GetWorksetParamList(dbConn, w.SetId)

	// check: parameter must in workset parametrs list
	// match languages from request into model languages
	for j := range pvtLst {

		np, ok := meta.ParamByName(pvtLst[j].Name)
		if !ok {
			return false, errors.New("Model parameter not found: " + dn + ": " + pvtLst[j].Name)
		}

		i := sort.SearchInts(hIds, meta.Param[np].ParamHid)
		if i < 0 || i >= len(hIds) || hIds[i] != meta.Param[np].ParamHid {
			return false, errors.New("Workset parameter not found: " + dn + ": " + wsn + ": " + pvtLst[j].Name)
		}

		for k := range pvtLst[j].Txt {
			lc := mc.languageCodeMatch(dn, pvtLst[j].Txt[k].LangCode)
			if lc != "" {
				pvtLst[j].Txt[k].LangCode = lc
			}
		}
	}

	// update workset parameter notes
	err = db.UpdateWorksetParameterText(dbConn, meta, wsn, pvtLst, langMeta)
	if err != nil {
		return false, errors.New("Error at update workset parameter notes: " + dn + ": " + wsn + ": " + err.Error())
	}

	return true, nil
}

// UpdateWorksetParameterCsv replace or merge parameter metadata into workset and replace parameter values from csv reader.
func (mc *ModelCatalog) UpdateWorksetParameterCsv(
	isReplace bool, wp *db.WorksetPub, param *db.ParamRunSetPub, csvRd *csv.Reader,
) (
	bool, error) {

	// if model digest-or-name or workset name is empty then return empty results
	dn := wp.ModelDigest
	if dn == "" {
		dn = wp.ModelName
	}
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return false, nil
	}
	if wp.Name == "" {
		omppLog.Log("Warning: invalid (empty) workset name")
		return false, nil
	}

	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Error: model digest or name not found: ", dn)
		return false, errors.New("Error: model digest or name not found: " + dn)
	}

	langMeta := mc.modelLangMeta(dn)
	if langMeta == nil {
		omppLog.Log("Error: invalid (empty) model language list: ", dn)
		return false, errors.New("Error: invalid (empty) model language list: " + dn)
	}

	// convert workset from "public" into db rows
	wm, err := wp.FromPublic(dbConn, meta)
	if err != nil {
		omppLog.Log("Error at workset json conversion: ", dn, ": ", wp.Name, ": ", err.Error())
		return false, err
	}

	// match languages from request into model languages
	for j := range param.Txt {
		lc := mc.languageCodeMatch(dn, param.Txt[j].LangCode)
		if lc != "" {
			param.Txt[j].LangCode = lc
		}
	}

	// if csv file exist then read csv file and convert and append lines into cell list
	var from func() (interface{}, error) = nil

	if csvRd != nil {

		// converter from csv row []string to db cell
		csvCvt := db.CellParamConverter{
			ModelDef:  meta,
			Name:      param.Name,
			DoubleFmt: theCfg.doubleFmt,
		}
		cvt, err := csvCvt.ToCell()
		if err != nil {
			return false, errors.New("invalid converter from csv row: " + err.Error())
		}

		// validate header line
		fhs, e := csvRd.Read()
		switch {
		case e == io.EOF:
			return false, errors.New("Inavlid (empty) csv parameter values " + param.Name)
		case e != nil:
			return false, errors.New("Failed to read csv parameter values " + param.Name + ": " + e.Error())
		}
		if chs, e := csvCvt.CsvHeader(); e != nil {
			return false, errors.New("Error at building csv parameter header " + param.Name)
		} else {
			fh := strings.Join(fhs, ",")
			if strings.HasPrefix(fh, string(helper.Utf8bom)) {
				fh = fh[len(helper.Utf8bom):]
			}
			ch := strings.Join(chs, ",")
			if fh != ch {
				return false, errors.New("Invalid csv parameter header " + param.Name + ": " + fh + " expected: " + ch)
			}
		}

		// convert each line into cell (id cell)
		from = func() (interface{}, error) {
			row, err := csvRd.Read()
			switch {
			case err == io.EOF:
				return nil, nil // eof
			case err != nil:
				return nil, errors.New("Failed to read csv parameter values " + param.Name)
			}

			// convert and append cell to cell list
			c, err := cvt(row)
			if err != nil {
				return nil, errors.New("Failed to convert csv parameter values " + param.Name)
			}
			return c, nil
		}

	}

	// update workset parameter metadata and parameter values
	hId, err := wm.UpdateWorksetParameterFrom(dbConn, meta, isReplace, param, langMeta, from)
	if err != nil {
		omppLog.Log("Error at update workset: ", dn, ": ", wp.Name, ": ", err.Error())
		return false, err
	}
	return hId > 0, nil // return success and true if parameter was found
}

// UpdateWorksetParameterPage merge "page" of parameter values into workset.
// Parameter must be already in workset and identified by model digest-or-name, set name, parameter name.
func (mc *ModelCatalog) UpdateWorksetParameterPage(dn, wsn, name string, from func() (interface{}, error)) error {

	// if model digest-or-name, set name or paramete name is empty then return empty results
	if dn == "" {
		return errors.New("Invalid (empty) model digest and name")
	}
	if wsn == "" {
		return errors.New("Invalid (empty) workset name. Model: " + dn)
	}
	if name == "" {
		return errors.New("Invalid (empty) parameter name. Model: " + dn + " workset: " + wsn)
	}

	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		return errors.New("Error: model digest or name not found: " + dn)
	}

	langMeta := mc.modelLangMeta(dn)
	if langMeta == nil {
		return errors.New("Error: invalid (empty) model language list: " + dn)
	}

	// find parameter Hid in model parameters list
	pHid := 0
	if k, ok := meta.ParamByName(name); ok {
		pHid = meta.Param[k].ParamHid
	} else {
		return errors.New("Parameter " + name + " not found in model: " + dn)
	}

	// find workset id by name
	ws, ok := mc.WorksetByName(dn, wsn)
	if !ok {
		return errors.New("Workset " + wsn + " not found in model: " + dn)
	}
	layout := db.WriteParamLayout{
		WriteLayout: db.WriteLayout{Name: name, ToId: ws.SetId},
		IsToRun:     false,
		IsPage:      true,
		DoubleFmt:   theCfg.doubleFmt,
	}

	// parameter must be in workset already
	nSub, _, err := db.GetWorksetParam(dbConn, ws.SetId, pHid)
	if err != nil {
		return errors.New("Error at get workset parameters list: " + wsn + ": " + err.Error())
	}
	if nSub <= 0 {
		return helper.ErrorFmt("Workset: %s must contain parameter: %s", wsn, name)
	}
	layout.SubCount = nSub

	// write parameter values
	return db.WriteParameterFrom(dbConn, meta, &layout, from)
}

// DeleteWorksetParameter do delete workset parameter metadata and values from database.
func (mc *ModelCatalog) DeleteWorksetParameter(dn, wsn, name string) (bool, error) {

	// if model digest-or-name or workset name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return false, nil
	}
	if wsn == "" {
		omppLog.Log("Warning: invalid (empty) workset name")
		return false, nil
	}
	if name == "" {
		omppLog.Log("Warning: invalid (empty) workset parameter name")
		return false, nil
	}

	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return false, nil // return empty result: model not found or error
	}

	// delete workset from database
	hId, err := db.DeleteWorksetParameter(dbConn, meta.Model.ModelId, wsn, name)
	if err != nil {
		omppLog.Log("Error at update workset: ", dn, ": ", wsn, ": ", err.Error())
		return false, err
	}
	return hId > 0, nil // return success and true if parameter was found
}

// CopyParameterToWsFromRun copy parameter metadata and values into workset from model run.
// If isReplace is true and parameter already exist in destination workset then error returned
// If isReplace is false then existing parameter values and metadata deleted and new inserted from model run.
// Destination workset must be in read-write state.
// Source model run must be completed, run status one of: s=success, x=exit, e=error.
func (mc *ModelCatalog) CopyParameterToWsFromRun(dn, wsn, name string, isReplace bool, rdsn string) error {

	// validate parameters
	if dn == "" {
		return errors.New("Parameter copy failed: invalid (empty) model digest and name")
	}
	if wsn == "" {
		return errors.New("Parameter copy failed: invalid (empty) workset name")
	}
	if name == "" {
		return errors.New("Parameter copy failed: invalid (empty) parameter name")
	}
	if rdsn == "" {
		return errors.New("Parameter copy failed: invalid (empty) model run digest or stamp or name")
	}

	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		return errors.New("Model digest or name not found: " + dn)
	}

	// find parameter Hid in model parameters list
	pHid := 0
	if k, ok := meta.ParamByName(name); ok {
		pHid = meta.Param[k].ParamHid
	} else {
		return errors.New("Parameter " + name + " not found in model: " + dn)
	}

	// find workset by name: it must be read-write
	ws, ok := mc.WorksetByName(dn, wsn)
	if !ok || ws == nil {
		return errors.New("Workset not found or error at get workset status: " + dn + ": " + wsn)
	}
	if ws.IsReadonly {
		return errors.New("Parameter copy failed, destination workset is read-only: " + wsn + ": " + name)
	}

	// if it is not merge and parameter already then return error
	if !isReplace {
		nSub, _, e := db.GetWorksetParam(dbConn, ws.SetId, pHid)
		if e != nil {
			return errors.New("Error at get workset parameters list: " + wsn + ": " + e.Error())
		}
		if nSub > 0 {
			return errors.New("Parameter copy failed, workset already contains parameter: " + wsn + ": " + name)
		}
	}

	// find run by digest or stamp or name: it must be completed
	r, ok := mc.CompletedRunByDigestOrStampOrName(dn, rdsn)
	if !ok || r == nil {
		return errors.New("Model not found or not completed: " + dn + ": " + rdsn)
	}

	// copy parameter into workset from model run
	err := db.CopyParameterFromRun(dbConn, meta, ws, name, isReplace, r)
	if err != nil {
		return errors.New("Parameter copy failed: " + wsn + ": " + name + ": " + err.Error())
	}
	return nil
}

// CopyParameterBetweenWs copy parameter metadata and values into workset from other workset.
// If isReplace is true and parameter already exist in destination workset then error returned
// If isReplace is false then existing parameter values and metadata deleted and new inserted from source workset.
// Destination workset must be in read-write state.
// Source workset must be read-only.
func (mc *ModelCatalog) CopyParameterBetweenWs(dn, dstWsName, name string, isReplace bool, srcWsName string) error {

	// validate parameters
	if dn == "" {
		return errors.New("Parameter copy failed: invalid (empty) model digest and name")
	}
	if dstWsName == "" {
		return errors.New("Parameter copy failed: invalid (empty) destination workset name")
	}
	if name == "" {
		return errors.New("Parameter copy failed: invalid (empty) parameter name")
	}
	if srcWsName == "" {
		return errors.New("Parameter copy failed: invalid (empty) source workset name")
	}

	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		return errors.New("Model digest or name not found: " + dn)
	}

	// find parameter Hid in model parameters list
	pHid := 0
	if k, ok := meta.ParamByName(name); ok {
		pHid = meta.Param[k].ParamHid
	} else {
		return errors.New("Parameter " + name + " not found in model: " + dn)
	}

	// find destination workset by name: it must be read-write
	dstWs, ok := mc.WorksetByName(dn, dstWsName)
	if !ok || dstWs == nil {
		return errors.New("Workset not found or error at get workset status: " + dn + ": " + dstWsName)
	}
	if dstWs.IsReadonly {
		return errors.New("Parameter copy failed, destination workset is read-only: " + dstWsName + ": " + name)
	}

	// if it is not merge and parameter already then return error
	if !isReplace {
		nSub, _, e := db.GetWorksetParam(dbConn, dstWs.SetId, pHid)
		if e != nil {
			return errors.New("Error at get workset parameters list: " + dstWsName + ": " + e.Error())
		}
		if nSub > 0 {
			return errors.New("Parameter copy failed, workset already contains parameter: " + dstWsName + ": " + name)
		}
	}

	// find source workset by name: it must be read-only
	srcWs, ok := mc.WorksetByName(dn, srcWsName)
	if !ok || srcWs == nil {
		return errors.New("Workset not found or error at get workset status: " + dn + ": " + srcWsName)
	}
	if !srcWs.IsReadonly {
		return errors.New("Parameter copy failed, source workset must be read-only: " + srcWsName + ": " + name)
	}

	// copy parameter from one workset to another
	err := db.CopyParameterFromWorkset(dbConn, meta, dstWs, name, isReplace, srcWs)
	if err != nil {
		return errors.New("Parameter copy failed: " + dstWsName + ": " + name + ": " + err.Error())
	}
	return nil
}
