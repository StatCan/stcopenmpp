// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"errors"
	"slices"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// UpdateRunStatus updates run status to the one of run completed: s=success, x=exit, e=error.
func (mc *ModelCatalog) UpdateRunStatus(dn, rdsn string, status string) (bool, error) {

	// if model digest-or-name or run digest-or-name name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return false, nil
	}
	if rdsn == "" {
		omppLog.Log("Warning: invalid (empty) model run digest, stamp and name")
		return false, nil
	}
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return false, nil
	}

	// find model run by digest, stamp or run name
	r, err := db.GetRunByDigestStampName(dbConn, meta.Model.ModelId, rdsn)
	if err != nil {
		omppLog.Log("Error at get model run: ", dn, ": ", rdsn, ": ", err.Error())
		return false, err
	}
	if r == nil {
		return false, nil // return OK: model run not found
	}

	// update run status
	err = db.UpdateRunStatus(dbConn, r.RunId, status)
	if err != nil {
		omppLog.Log("Error at run status update: ", dn, ": ", rdsn, ": ", err.Error())
		return false, err
	}
	return true, nil
}

// Start a separate thread to delete model run including output table values, input parameters and microdata.
func (mc *ModelCatalog) DeleteRunStart(dn, rdsn string) (bool, error) {

	// if model digest-or-name or run digest-or-name name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return false, nil
	}
	if rdsn == "" {
		omppLog.Log("Warning: invalid (empty) model run digest, stamp and name")
		return false, nil
	}
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return false, nil
	}

	// find model run by digest, stamp or run name
	r, err := db.GetRunByDigestStampName(dbConn, meta.Model.ModelId, rdsn)
	if err != nil {
		omppLog.Log("Error at get model run: ", dn, ": ", rdsn, ": ", err.Error())
		return false, err
	}
	if r == nil {
		return false, nil // return OK: model run not found
	}

	// do delete run from database in background
	go func(dbc *sql.DB, runId int, dn, rdsn string) {

		e := db.DeleteRun(dbc, runId)
		if e != nil {
			omppLog.Log("Error at delete model run: ", dn, ": ", rdsn, ": ", e.Error())
		} else {
			omppLog.Log("Deleted model run: ", dn, ": ", rdsn)
		}
	}(dbConn, r.RunId, dn, rdsn)

	return true, nil
}

// Start a separate thread to delete list of model runs including output table values, input parameters and microdata.
func (mc *ModelCatalog) DeleteRunListStart(dn string, rdsnLst []string) (bool, error) {

	// if model digest-or-name or run digest-or-name name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return false, nil
	}
	if len(rdsnLst) <= 0 {
		return false, nil // return OK: empty list of model runs
	}
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return false, nil
	}

	// for each run find run id by digest, stamp or run name
	rIds := make([]int, 0, len(rdsnLst))
	rm := make(map[int]string, len(rdsnLst))

	for _, rdsn := range rdsnLst {

		rLst, err := db.GetRunListByDigestStampName(dbConn, meta.Model.ModelId, rdsn)
		if err != nil {
			omppLog.Log("Error at get model run: ", dn, ": ", rdsn, ": ", err.Error())
			return false, err
		}

		for k := range rLst {
			rIds = append(rIds, rLst[k].RunId)
			rm[rLst[k].RunId] = rdsn
		}
	}

	// do delete runs from database in background
	go func(dbc *sql.DB, modelDn string, runIds []int, rdsnMap map[int]string) {

		slices.Sort(rIds) // delete runs in descending order

		n := 0
		for k := len(runIds) - 1; k >= 0; k-- {

			if runIds[k] <= 0 || k < len(runIds)-1 && runIds[k] == runIds[k+1] {
				continue // skip invalid run id or duplicate run id
			}

			e := db.DeleteRun(dbc, runIds[k])
			if e != nil {
				omppLog.Log("Error at delete model run: ", modelDn, ": ", runIds[k], " ", rdsnMap[runIds[k]], ": ", e.Error())
				return
			}
			n++
		}
		omppLog.Log("Deleted multiple model runs: ", n, ": ", modelDn)

	}(dbConn, dn, rIds, rm)

	return true, nil
}

// UpdateRunText do merge run text (run description and notes) and run parameter notes.
func (mc *ModelCatalog) UpdateRunText(rp *db.RunPub) (bool, string, string, error) {

	// validate parameters
	if rp == nil {
		omppLog.Log("Error: invalid (empty) model run data")
		return false, "", "", errors.New("Error: invalid (empty) model run data")
	}

	// if model digest-or-name or run digest-or-name is empty then return empty results
	dn := rp.ModelDigest
	if dn == "" {
		dn = rp.ModelName
	}
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return false, "", "", nil
	}

	rdsn := rp.RunDigest
	if rdsn == "" {
		rdsn = rp.RunStamp
	}
	if rdsn == "" {
		rdsn = rp.Name
	}
	if rdsn == "" {
		omppLog.Log("Warning: invalid (empty) model run digest, stamp and name")
		return false, "", "", nil
	}

	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return false, "", "", nil
	}

	langMeta := mc.modelLangMeta(dn)
	if langMeta == nil {
		omppLog.Log("Error: invalid (empty) model language list: ", dn)
		return false, "", "", errors.New("Error: invalid (empty) model language list: " + dn)
	}

	// find model run by digest, stamp or run name
	r, err := db.GetRunByDigestStampName(dbConn, meta.Model.ModelId, rdsn)
	if err != nil {
		omppLog.Log("Error at get model run: ", dn, ": ", rdsn, ": ", err.Error())
		return false, dn, rdsn, err
	}
	if r == nil {
		return false, dn, rdsn, nil // return OK: model run not found
	}

	// validate: model run must be completed
	if !db.IsRunCompleted(r.Status) {
		omppLog.Log("Failed to update model run, it is not completed: ", dn, ": ", rdsn)
		return false, dn, rdsn, errors.New("Failed to update model run, it is not completed: " + dn + ": " + rdsn)
	}

	// convert run from "public" into db rows
	rm, err := rp.FromPublic(meta)
	if err != nil {
		omppLog.Log("Error at model run conversion: ", dn, ": ", rdsn, ": ", err.Error())
		return false, dn, rdsn, err
	}

	// match languages from request into model languages
	for k := range rm.Txt {
		lc := mc.languageCodeMatch(dn, rm.Txt[k].LangCode)
		if lc != "" {
			rm.Txt[k].LangCode = lc
		}
	}
	for k := range rm.Param {
		for j := range rm.Param[k].Txt {
			lc := mc.languageCodeMatch(dn, rm.Param[k].Txt[j].LangCode)
			if lc != "" {
				rm.Param[k].Txt[j].LangCode = lc
			}
		}
	}

	// update model run text and run parameter notes
	err = rm.UpdateRunText(dbConn, meta, r.RunId, langMeta)
	if err != nil {
		omppLog.Log("Error at update model run: ", dn, ": ", rdsn, ": ", err.Error())
		return false, dn, rdsn, err
	}

	return true, dn, rdsn, nil
}

// UpdateRunParameterText do merge (insert or update) parameters run value notes.
func (mc *ModelCatalog) UpdateRunParameterText(dn, rdsn string, pvtLst []db.ParamRunSetTxtPub) (bool, error) {

	// validate parameters
	if pvtLst == nil || len(pvtLst) <= 0 {
		omppLog.Log("Warning: empty list of run parameters to update value notes")
		return false, nil
	}
	if dn == "" {
		return false, errors.New("Error: invalid (empty) model digest or name")
	}
	if rdsn == "" {
		return false, errors.New("Error: invalid (empty) model run digest or stamp or name")
	}

	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		return false, errors.New("Error: model digest or name not found: " + dn)
	}

	langMeta := mc.modelLangMeta(dn)
	if langMeta == nil {
		return false, errors.New("Error: invalid (empty) model language list: " + dn)
	}

	// validate parameters by name: it must be model parameter
	for k := range pvtLst {
		if _, ok = meta.ParamByName(pvtLst[k].Name); !ok {
			return false, errors.New("Model parameter not found: " + dn + ": " + pvtLst[k].Name)
		}
	}

	// find model run by digest, stamp or run name
	r, err := db.GetRunByDigestStampName(dbConn, meta.Model.ModelId, rdsn)
	if err != nil {
		return false, errors.New("Model run not found: " + dn + ": " + rdsn + ": " + err.Error())
	}
	if r == nil {
		return false, errors.New("Model run not found: " + dn + ": " + rdsn)
	}

	// validate: model run must be completed
	if !db.IsRunCompleted(r.Status) {
		return false, errors.New("Failed to update model run, it is not completed: " + dn + ": " + rdsn)
	}

	// match languages from request into model languages
	for j := range pvtLst {
		for k := range pvtLst[j].Txt {
			lc := mc.languageCodeMatch(dn, pvtLst[j].Txt[k].LangCode)
			if lc != "" {
				pvtLst[j].Txt[k].LangCode = lc
			}
		}
	}

	// update run parameter notes
	err = db.UpdateRunParameterText(dbConn, meta, r.RunId, pvtLst, langMeta)
	if err != nil {
		return false, errors.New("Error at update run parameter notes: " + dn + ": " + rdsn + ": " + err.Error())
	}

	return true, nil
}
