// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"encoding/csv"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// worksetReadonlyUpdateHandler update workset read-only status by model digest-or-name and workset name:
// POST /api/model/:model/workset/:set/readonly/:readonly
// If multiple models with same name exist then result is undefined.
// If no such workset exist in database then empty result returned.
func worksetReadonlyUpdateHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	wsn := getRequestParam(r, "set")
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// convert readonly flag
	isReadonly, ok := getBoolRequestParam(r, "readonly")
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid value of workset read-only flag", wsn), http.StatusBadRequest)
		return
	}

	// update workset read-only status
	digest, ws, ok, err := theCatalog.UpdateWorksetReadonly(dn, wsn, isReadonly)
	if err != nil {
		http.Error(w, helper.MsgL(lang, "Error at updating workset read-only flag", wsn), http.StatusBadRequest)
		return
	}
	if ok {
		w.Header().Set("Content-Location", "/api/model/"+digest+"/workset/"+ws.Name)
	} else {
		ws = &db.WorksetRow{}
	}
	jsonResponse(w, r, ws)
}

// worksetCreateHandler creates new workset and append parameter(s) from json request:
// PUT  /api/workset-create
// Json keys: model digest or name and workset name.
// If multiple models with same name exist then result is undefined, it is recommended to use model digest.
// If workset name is empty in json request then automatically generate unique workset name.
// If workset with same name already exist for that model then return error.
// Json content: workset "public" metadata and optioanl parameters list.
// Workset and parameters "public" metadata expected to be same as return of GET /api/model/:model/workset/:set/text-all
// Each parameter (if present) must contain "public" metadata and parameter values.
// Parameter values must include all values and can be either:
// JSON cell values, identical to output of read parameter "page": POST /api/model/:model/workset/:set/parameter/value
// or copy directions, for example run digest to copy value from.
// Dimension(s) and enum-based parameters expected to be enum codes, not enum id's.
func worksetCreateHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// decode json workset "public" metadata
	var wp db.WorksetCreatePub
	if !jsonRequestDecode(w, r, true, &wp) {
		return // error at json decode, response done with http error
	}
	dn := wp.ModelDigest
	if dn == "" {
		dn = wp.ModelName
	}

	// if workset name is empty then automatically generate name
	wsn := wp.Name
	if wsn == "" {
		ts, _ := theCatalog.getNewTimeStamp()
		wsn = "set_" + ts
	}

	// return error if workset already exist or unable to get workset status
	ws, ok := theCatalog.WorksetByName(dn, wsn)
	if ok {
		omppLog.LogFmt("Error: workset already exist: %s model: %s", wsn, dn)
		http.Error(w, helper.FmtL(lang, "Error: workset already exist: %s model: %s", wsn, dn), http.StatusBadRequest)
		return
	}
	if !ok && ws == nil {
		omppLog.Log("Failed to create workset:", dn, ":", wsn)
		http.Error(w, helper.MsgL(lang, "Failed to create workset:", dn, ":", wsn), http.StatusBadRequest)
		return
	}
	// else workset not exist

	// insert workset metadata with empty list of parameters and read-write status
	newWp := db.WorksetPub{
		WorksetHdrPub: db.WorksetHdrPub{
			ModelName:      wp.ModelName,
			ModelDigest:    wp.ModelDigest,
			Name:           wsn,
			BaseRunDigest:  wp.BaseRunDigest,
			IsReadonly:     false,
			UpdateDateTime: wp.UpdateDateTime,
			Txt:            wp.Txt,
		},
		Param: []db.ParamRunSetPub{},
	}

	ok, _, wsRow, err := theCatalog.UpdateWorkset(true, &newWp)
	if err != nil {
		http.Error(w, helper.MsgL(lang, "Failed to create workset metadata", dn, ":", wsn, ":", err), http.StatusBadRequest)
		return
	}
	if !ok {
		http.Error(w, helper.MsgL(lang, "Failed to create workset metadata", dn, ":", wsn), http.StatusBadRequest)
		return
	}

	// append parameters metadata and values, each parameter will be inserted in separate transaction
	for k := range wp.Param {
		switch wp.Param[k].Kind {
		case "run":
			if e := theCatalog.CopyParameterToWsFromRun(dn, wsn, wp.Param[k].Name, false, wp.Param[k].From); e != nil {
				http.Error(w, helper.MsgL(lang, "Failed to copy parameter from model run", wsn, ":", wp.Param[k].Name, ":", wp.Param[k].From, ":", e), http.StatusBadRequest)
				return
			}
			continue
		case "set":
			if e := theCatalog.CopyParameterBetweenWs(dn, wsn, wp.Param[k].Name, false, wp.Param[k].From); e != nil {
				http.Error(w, helper.MsgL(lang, "Failed to copy parameter from workset", wsn, ":", wp.Param[k].Name, ":", wp.Param[k].From, ":", e), http.StatusBadRequest)
				return
			}
			continue
		}
		// default case: source of parameter must be a "value" or empty
		if wp.Param[k].Kind != "" && wp.Param[k].Kind != "value" || len(wp.Param[k].Value) <= 0 {
			http.Error(w, helper.MsgL(lang, "Invalid (or empty) workset parameter values", wsn, ":", wp.Param[k].Name), http.StatusBadRequest)
			return
		}
		if _, e := theCatalog.UpdateWorksetParameter(true, &newWp, &wp.Param[k].ParamRunSetPub, wp.Param[k].Value); e != nil {
			http.Error(w, helper.MsgL(lang, "Failed to update workset parameter", wsn, ":", wp.Param[k].Name, ":", e), http.StatusBadRequest)
			return
		}
	}

	// if required make workset read-only
	if wp.IsReadonly {
		theCatalog.UpdateWorksetReadonly(dn, wsn, wp.IsReadonly)
	}
	w.Header().Set("Content-Location", "/api/model/"+dn+"/workset/"+wsn) // respond with workset location
	jsonResponse(w, r, wsRow)
}

// worksetReplaceHandler replace workset and all parameters from multipart-form:
// PUT /api/workset-replace
// Expected multipart form parts:
// first workset part with workset metadata in json
// and multiple parts file-csv=parameterName.csv.
// Json keys: model digest or name and workset name.
// Json content: workset "public" metadata.
// If multiple models with same name exist then result is undefined, it is recommended to use model digest.
// If workset name is empty in json request then automatically generate unique workset name.
// If no such workset exist in database then new workset created.
// If workset already exist then it is delete-insert operation:
// existing metadata, parameter list, parameter metadata and parameter values deleted from database
// and new metadata, parameters metadata and parameters values inserted.
func worksetReplaceHandler(w http.ResponseWriter, r *http.Request) {
	worksetUpdateHandler(true, w, r)
}

// worksetMergeHandler merge workset metadata and parameters metadata and values from multipart-form:
// PATCH /api/workset-merge
// Expected multipart form parts:
// first workset part with workset metadata in json
// and optional multiple parts file-csv=parameterName.csv.
// Json keys: model digest or name and workset name.
// Json content: workset "public" metadata.
// If multiple models with same name exist then result is undefined, it is recommended to use model digest.
// If no such workset exist in database then new workset created.
// If workset name is empty in json request then automatically generate unique workset name.
// If workset already exist then merge operation existing workset metadata with new.
// If workset not exist then create new workset.
// Merge parameter list: if parameter exist then merge metadata.
// If new parameter values supplied then replace paramter values.
// If parameter not already exist in workset then parameter values must be supplied.
func worksetMergeHandler(w http.ResponseWriter, r *http.Request) {
	worksetUpdateHandler(false, w, r)
}

// worksetUpdateHandler replace or merge workset metadata and parameters from multipart-form:
// Expected multipart form parts:
// first workset part with workset metadata in json
// and optional multiple parts file-csv=parameterName.csv.
// Json keys: model digest or name and workset name.
// Json content: workset "public" metadata.
// If parameter not already exist in workset then parameter values must be supplied.
// It is an error to add parameter metadata without parameter values.
func worksetUpdateHandler(isReplace bool, w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// parse multipart form: first part must be workset metadata
	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, helper.MsgL(lang, "Error at multipart form open"), http.StatusBadRequest)
		return
	}

	var newWp db.WorksetPub
	if !jsonMultipartDecode(w, mr, "workset", &newWp) {
		return // error at json decode, response done with http error
	}
	dn := newWp.ModelDigest
	if dn == "" {
		dn = newWp.ModelName
	}

	// if workset name is empty then automatically generate name
	if newWp.Name == "" {
		ts, _ := theCatalog.getNewTimeStamp()
		newWp.Name = "set_" + ts
	}

	// get existing workset metadata
	oldWp, _, err := theCatalog.WorksetTextFull(dn, newWp.Name, true, nil)
	if err != nil {
		http.Error(w, helper.MsgL(lang, "Failed to get existing workset metadata", dn, ":", newWp.Name), http.StatusBadRequest)
		return
	}

	// make starting list of parameters as new parameters which already exist in workset
	newParamLst := append([]db.ParamRunSetPub{}, newWp.Param...)
	newWp.Param = make([]db.ParamRunSetPub, 0, len(newParamLst))

	for k := range newParamLst {
		for j := range oldWp.Param {

			// if this new parameter already exist in workset then keep it
			if newParamLst[k].Name == oldWp.Param[j].Name {
				newWp.Param = append(newWp.Param, newParamLst[k])
				break
			}
		}
	}

	// update workset metadata, postpone read-only status until update completed
	isReadonly := newWp.IsReadonly
	newWp.IsReadonly = false

	ok, _, wsRow, err := theCatalog.UpdateWorkset(isReplace, &newWp)
	if err != nil {
		http.Error(w, helper.MsgL(lang, "Failed to update workset metadata", dn, ":", newWp.Name, ":", err), http.StatusBadRequest)
		return
	}
	if !ok {
		http.Error(w, helper.MsgL(lang, "Failed to update workset metadata", dn, ":", newWp.Name), http.StatusBadRequest)
		return
	}

	// each parameter will be replaced in separate transaction
	// decode multipart form csv files
	pim := make(map[int]bool) // if index of parameter name in the map then csv supplied for that parameter
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break // end of posted data
		}
		if err != nil {
			http.Error(w, helper.MsgL(lang, "Failed to get next part of multipart form", dn, ":", newWp.Name), http.StatusBadRequest)
			return
		}

		// skip non parameter-csv data
		if part.FormName() != "parameter-csv" {
			part.Close()
			continue
		}

		// validate: parameter name must be in the list of workset parameters
		ext := path.Ext(part.FileName())
		if ext != ".csv" {
			part.Close()
			http.Error(w, helper.MsgL(lang, "Error: parameter file must have .csv extension", newWp.Name, ":", part.FileName()), http.StatusBadRequest)
			return
		}
		name := strings.TrimSuffix(path.Base(part.FileName()), ext)

		np := -1
		for k := range newParamLst {
			if name == newParamLst[k].Name {
				np = k
				break
			}
		}
		if np < 0 {
			part.Close()
			http.Error(w, helper.MsgL(lang, "Error: parameter must be in workset parameters list:", newWp.Name+":", name), http.StatusBadRequest)
			return
		}

		// read csv values and update parameter
		csvRd := csv.NewReader(part)
		csvRd.TrimLeadingSpace = true
		csvRd.ReuseRecord = true

		_, err = theCatalog.UpdateWorksetParameterCsv(isReplace, &newWp, &newParamLst[np], csvRd)
		part.Close() // done with csv parameter data
		if err != nil {
			http.Error(w, helper.MsgL(lang, "Failed to update workset parameter", newWp.Name, ":", name, ":", err), http.StatusBadRequest)
			return
		}
		pim[np] = true // parameter metadata and csv values updated
	}

	// update parameter(s) metadata where parameter csv values not supplied
	for k := range newParamLst {

		if pim[k] {
			continue // parameter metadata already updated together with csv parameter values
		}

		// update only parameter metadata
		_, err = theCatalog.UpdateWorksetParameterCsv(isReplace, &newWp, &newParamLst[k], nil)
		if err != nil {
			http.Error(w, helper.MsgL(lang, "Failed to update workset parameter", newWp.Name, ":", newParamLst[k].Name, ":", err), http.StatusBadRequest)
			return
		}
	}

	// if required make workset read-only
	if isReadonly {
		theCatalog.UpdateWorksetReadonly(dn, newWp.Name, isReadonly)
	}

	w.Header().Set("Content-Location", "/api/model/"+dn+"/workset/"+newWp.Name) // respond with workset location
	jsonResponse(w, r, wsRow)
}

// Delete workset and workset parameters:
// DELETE /api/model/:model/workset/:set
// If multiple models with same name exist then result is undefined.
// If no such workset exist in database then no error, empty operation.
func worksetDeleteHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	wsn := getRequestParam(r, "set")
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// delete workset
	ok, err := theCatalog.DeleteWorkset(dn, wsn)
	if err != nil {
		http.Error(w, helper.MsgL(lang, "Workset delete failed", dn, ":", wsn), http.StatusBadRequest)
		return
	}
	if ok {
		w.Header().Set("Content-Location", "/api/model/"+dn+"/workset/"+wsn)
		w.Header().Set("Content-Type", "text/plain")
	}
}

// Delete multiple worksets and workset parameters:
// POST /api/model/:model/delete-worksets
// Request body contains array of workset names to delete.
// Model identified by digest-or-name.
// If multiple models with same name exist then result is undefined.
// If no such worksets exist in database then no error, empty operation.
func worksetListDeleteHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// decode json array of workset names
	var nameLst []string
	if !jsonRequestDecode(w, r, true, &nameLst) {
		return // error at json decode, response done with http error
	}

	// delete all worksets
	n := 0

	for _, name := range nameLst {

		if name == "" { // skip empty names
			continue
		}

		// set read-write status
		_, _, _, err := theCatalog.UpdateWorksetReadonly(dn, name, false)
		if err != nil {
			http.Error(w, helper.MsgL(lang, "Error at clear workset read-only", dn, ":", name), http.StatusBadRequest)
			return
		}

		// delete workset
		ok, err := theCatalog.DeleteWorkset(dn, name)
		if err != nil {
			http.Error(w, helper.MsgL(lang, "Workset delete failed", dn, ":", name), http.StatusBadRequest)
			return
		}
		if ok {
			n++
		}
	}
	omppLog.Log("Deleted multiple worksets:", n, ":", dn)

	w.Header().Set("Content-Location", "/api/model/"+dn+"/delete-worksets/"+strconv.Itoa(n))
	w.Header().Set("Content-Type", "text/plain")
}

// parameterPageUpdateHandler update a "page" of workset parameter values.
// PATCH /api/model/:model/workset/:set/parameter/:name/new/value
// Dimension(s) and enum-based parameters expected to be as enum codes.
// Input parameter "page" json expected to be identical to output of read parameter "page".
func parameterPageUpdateHandler(w http.ResponseWriter, r *http.Request) {
	doUpdateParameterPageHandler(w, r, true)
}

// parameterIdPageUpdateHandler update a "page" of workset parameter values.
// PATCH /api/model/:model/workset/:set/parameter/:name/new/value-id
// Dimension(s) and enum-based parameters expected to be as enum id, not enum codes.
// Input parameter "page" json expected to be identical to output of read parameter "page".
func parameterIdPageUpdateHandler(w http.ResponseWriter, r *http.Request) {
	doUpdateParameterPageHandler(w, r, false)
}

// doUpdateParameterPageHandler update a "page" of workset parameter values.
// Page is part of parameter values defined by zero-based "start" row number and row count.
// Dimension(s) and enum-based parameters can be as enum codes or enum id's.
func doUpdateParameterPageHandler(w http.ResponseWriter, r *http.Request, isCode bool) {

	// url or query parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	wsn := getRequestParam(r, "set")   // workset name
	name := getRequestParam(r, "name") // parameter name
	lang := preferedRequestLang(r, "") // get prefered language for messages

	var from func() (interface{}, error) = nil

	// decode json body into array of cells
	// make closure to process each cell, convert to id cells, if required
	if !isCode {

		var cArr []db.CellParam
		if !jsonRequestDecode(w, r, true, &cArr) {
			return // error at json decode, response done with http error
		}
		if len(cArr) <= 0 {
			http.Error(w, helper.FmtL(lang, "Workset parameter update failed %s parameter empty: %s", wsn, name), http.StatusInternalServerError)
			return
		}

		// iterate over cell array
		k := -1
		from = func() (interface{}, error) {
			k++
			if k >= len(cArr) {
				return nil, nil // end of data
			}
			return cArr[k], nil
		}
	} else {

		// decode code cells from json body
		var cArr []db.CellCodeParam
		if !jsonRequestDecode(w, r, true, &cArr) {
			return // error at json decode, response done with http error
		}
		if len(cArr) <= 0 {
			http.Error(w, helper.FmtL(lang, "Workset parameter update failed %s parameter empty: %s", wsn, name), http.StatusInternalServerError)
			return
		}

		// convert from enum code cells to id cells
		cvt, ok := theCatalog.ParameterCellConverter(true, dn, name)
		if !ok {
			http.Error(w, helper.MsgL(lang, "Workset parameter update failed", wsn, ":", name), http.StatusBadRequest)
			return
		}

		// iterate over cell array and convert each cell from code to enum id
		k := -1
		from = func() (interface{}, error) {
			k++
			if k >= len(cArr) {
				return nil, nil // end of data
			}

			c, err := cvt(cArr[k])
			if err != nil {
				return nil, err
			}
			return c, nil
		}
	}

	// update parameter values
	err := theCatalog.UpdateWorksetParameterPage(dn, wsn, name, from)
	if err != nil {
		omppLog.LogNoLT(err)
		http.Error(w, helper.MsgL(lang, "Workset parameter update failed", wsn, ":", name), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Location", "/api/model/"+dn+"/workset/"+wsn+"/parameter/"+name) // respond with workset parameter location
	w.Header().Set("Content-Type", "text/plain")
}

// worksetParameterDeleteHandler delete workset parameter:
// DELETE /api/model/:model/workset/:set/parameter/:name
// If multiple models with same name exist then result is undefined.
// If no such parameter or workset exist in database then no error, empty operation.
func worksetParameterDeleteHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	wsn := getRequestParam(r, "set")
	name := getRequestParam(r, "name")
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// delete workset parameter
	ok, err := theCatalog.DeleteWorksetParameter(dn, wsn, name)
	if err != nil {
		http.Error(w, helper.MsgL(lang, "Workset parameter delete failed", wsn, ":", name), http.StatusBadRequest)
		return
	}
	if ok {
		w.Header().Set("Content-Location", "/api/model/"+dn+"/workset/"+wsn+"/parameter/"+name)
		w.Header().Set("Content-Type", "text/plain")
	}
}

// worksetParameterRunCopyHandler do copy (insert new) parameter into workset from model run:
// PUT /api/model/:model/workset/:set/copy/parameter/:name/from-run/:run
// If multiple models with same name exist then result is undefined.
// If such parameter already exist in destination workset then return error.
// Destination workset must be in read-write state.
// Run must be completed, run status one of: s=success, x=exit, e=error.
func worksetParameterRunCopyHandler(w http.ResponseWriter, r *http.Request) {
	worksetParameterRunCopy(false, w, r)
}

// worksetParameterRunMergeHandler do merge (insert or update) parameter into workset from model run:
// PATCH  /api/model/:model/workset/:set/merge/parameter/:name/from-run/:run
// If multiple models with same name exist then result is undefined.
// If such parameter already exist in destination workset
// then it is updated by delete old and insert new values and metadata.
// Destination workset must be in read-write state.
// Run must be completed, run status one of: s=success, x=exit, e=error.
func worksetParameterRunMergeHandler(w http.ResponseWriter, r *http.Request) {
	worksetParameterRunCopy(true, w, r)
}

// worksetParameterRunCopy do copy parameter from model run into workset.
// if isReplace is true then it insert new else merge: insert new or update existing.
func worksetParameterRunCopy(isReplace bool, w http.ResponseWriter, r *http.Request) {

	// url or query parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	wsn := getRequestParam(r, "set")   // workset name
	name := getRequestParam(r, "name") // parameter name
	rdsn := getRequestParam(r, "run")  // source run digest or stamp or name
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// copy workset parameter from model run
	err := theCatalog.CopyParameterToWsFromRun(dn, wsn, name, isReplace, rdsn)
	if err != nil {
		omppLog.LogNoLT(err)
		http.Error(w, helper.FmtL(lang, "Workset parameter copy failed %s: %s from run: %s", wsn, name, rdsn), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Location", "/api/model/"+dn+"/workset/"+wsn+"/parameter/"+name)
	w.Header().Set("Content-Type", "text/plain")
}

// worksetParameterCopyFromWsHandler do copy (insert new) parameter from one workset to another:
// PUT /api/model/:model/workset/:set/copy/parameter/:name/from-workset/:from-set
// If multiple models with same name exist then result is undefined.
// If such parameter already exist in destination workset then return error.
// Destination workset must be in read-write state, source workset must be read-only.
func worksetParameterCopyFromWsHandler(w http.ResponseWriter, r *http.Request) {
	worksetParameterCopyFromWs(false, w, r)
}

// worksetParameterMergeFromWsHandler do merge (insert or update) parameter from one workset to another:
// PATCH /api/model/:model/workset/:set/merge/parameter/:name/from-workset/:from-set
// If multiple models with same name exist then result is undefined.
// If such parameter already exist in destination workset
// then it is updated by delete old and insert new values and metadata.
// Destination workset must be in read-write state, source workset must be read-only.
func worksetParameterMergeFromWsHandler(w http.ResponseWriter, r *http.Request) {
	worksetParameterCopyFromWs(true, w, r)
}

// worksetParameterCopyFromWs does copy parameter from one workset to another.
// if isReplace is true then it does insert new else merge: insert new or update existing.
func worksetParameterCopyFromWs(isReplace bool, w http.ResponseWriter, r *http.Request) {

	// url or query parameters
	dn := getRequestParam(r, "model")           // model digest-or-name
	dstWsName := getRequestParam(r, "set")      // workset name
	name := getRequestParam(r, "name")          // parameter name
	srcWsName := getRequestParam(r, "from-set") // source run digest or name
	lang := preferedRequestLang(r, "")          // get prefered language for messages

	// copy workset parameter from other workset
	err := theCatalog.CopyParameterBetweenWs(dn, dstWsName, name, isReplace, srcWsName)
	if err != nil {
		omppLog.LogNoLT(err)
		http.Error(w, helper.FmtL(lang, "Workset parameter copy failed %s: %s from run: %s", dstWsName, name, srcWsName), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Location", "/api/model/"+dn+"/workset/"+dstWsName+"/parameter/"+name)
	w.Header().Set("Content-Type", "text/plain")
}

// worksetParameterTextMergeHandler do merge (insert or update) workset parameter(s) value notes, array of parameters expected.
// PATCH /api/model/:model/workset/:set/parameter-text
// Model can be identified by digest or name and model run also identified by run digest-or-stamp-or-name.
// If multiple models with same name exist then result is undefined.
// If workset does not exist or is not in read-write state then return error.
// If parameter not exist in workset then return error.
// Input json must be array of ParamRunSetTxtPub,
// if parameters text array is empty then nothing updated, it is empty operation return is success
func worksetParameterTextMergeHandler(w http.ResponseWriter, r *http.Request) {

	// url or query parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	wsn := getRequestParam(r, "set")   // workset name
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// decode json parameter value notes
	var pvtLst []db.ParamRunSetTxtPub
	if !jsonRequestDecode(w, r, true, &pvtLst) {
		return // error at json decode, response done with http error
	}

	// update workset parameter value notes in model catalog
	ok, err := theCatalog.UpdateWorksetParameterText(dn, wsn, pvtLst)
	if err != nil {
		omppLog.LogNoLT(err)
		http.Error(w, helper.MsgL(lang, "Workset parameter(s) value notes update failed", dn, ":", wsn, ":", err), http.StatusBadRequest)
		return
	}
	if ok {
		w.Header().Set("Content-Location", "/api/model/"+dn+"/workset/"+wsn+"/parameter-text")
		w.Header().Set("Content-Type", "text/plain")
	}
}
