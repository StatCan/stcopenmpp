// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
	"golang.org/x/text/language"
)

// CompletedRunByDigestOrStampOrName select run_lst db row by digest-or-stamp-or-name and model digest or name.
// Run must be completed, run status one of: s=success, x=exit, e=error.
func (mc *ModelCatalog) CompletedRunByDigestOrStampOrName(dn string, rdsn string) (*db.RunRow, bool) {

	r := mc.RunRow(dn, rdsn)
	if r == nil {
		return nil, false
	}

	// run must be completed
	if !db.IsRunCompleted(r.Status) {
		omppLog.Log("Warning: run is not completed:", dn, ":", rdsn, ":", r.Status)
		return nil, false // return empty result: run_lst row not found
	}

	return r, true
}

// RunRow select run_lst db row by digest-or-stamp-or-name and model digest or name.
func (mc *ModelCatalog) RunRow(dn string, rdsn string) *db.RunRow {

	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return nil
	}
	if rdsn == "" {
		omppLog.Log("Warning: invalid (empty) run digest or stamp or name")
		return nil
	}
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return nil // return empty result: model not found or error
	}

	// get run_lst db row by digest, stamp or run name
	r, err := db.GetRunByDigestStampName(dbConn, meta.Model.ModelId, rdsn)
	if err != nil {
		omppLog.Log("Error at get run status:", meta.Model.Name, ":", rdsn, ":", err.Error())
		return nil // return empty result: run select error
	}
	if r == nil {
		omppLog.Log("Warning: run not found:", meta.Model.Name, ":", rdsn)
		return nil // return empty result: run_lst row not found
	}

	return r
}

// RunStatus return run_lst db row and run_progress db rows by model digest-or-name and run digest-or-stamp-or-name.
func (mc *ModelCatalog) RunStatus(dn, rdsn string) (*db.RunPub, bool) {

	// if model digest-or-name or run digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return &db.RunPub{}, false
	}
	if rdsn == "" {
		omppLog.Log("Warning: invalid (empty) run digest or stamp or name")
		return &db.RunPub{}, false
	}
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return &db.RunPub{}, false // return empty result: model not found or error
	}

	// get run_lst db row by digest, stamp or run name
	r, err := db.GetRunByDigestStampName(dbConn, meta.Model.ModelId, rdsn)
	if err != nil {
		omppLog.Log("Error at get run status:", dn, ":", rdsn, ":", err.Error())
		return &db.RunPub{}, false // return empty result: run select error
	}
	if r == nil {
		// omppLog.Log("Warning run status not found: ", dn, ": ", rdsn)
		return &db.RunPub{}, false // return empty result: run_lst row not found
	}

	// get run sub-values progress for that run id
	rpRs, err := db.GetRunProgress(dbConn, r.RunId)
	if err != nil {
		omppLog.Log("Error at get run progress:", dn, ":", rdsn, ":", err.Error())
		return &db.RunPub{}, false // return empty result: run progress select error
	}

	// convert to "public" format
	rp, err := (&db.RunMeta{Run: *r, Progress: rpRs}).ToPublic(meta)
	if err != nil {
		omppLog.Log("Error at run status conversion:", dn, ": ", rdsn, ": ", err.Error())
		return &db.RunPub{}, false // return empty result
	}

	return rp, true
}

// RunStatusList return list of run_lst rows joined to run_progress by model digest-or-name and run digest-or-stamp-or-name.
func (mc *ModelCatalog) RunStatusList(dn, rdsn string) ([]db.RunPub, bool) {

	// if model digest-or-name or run digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return []db.RunPub{}, false
	}
	if rdsn == "" {
		omppLog.Log("Warning: invalid (empty) run digest or stamp or name")
		return []db.RunPub{}, false
	}
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return []db.RunPub{}, false
	}

	// get run_lst db row by digest, stamp or run name
	rLst, err := db.GetRunListByDigestStampName(dbConn, meta.Model.ModelId, rdsn)
	if err != nil {
		omppLog.Log("Error at get run status:", dn, ":", rdsn, ":", err.Error())
		return []db.RunPub{}, false // return empty result: run select error
	}
	if len(rLst) <= 0 {
		// omppLog.Log("Warning run status not found: ", dn, ": ", rdsn)
		return []db.RunPub{}, false // return empty result: run_lst row not found
	}

	// for each run_lst row join run_progress rows
	rpLst := []db.RunPub{}

	for n := range rLst {
		// get run sub-values progress for that run id
		rpRs, err := db.GetRunProgress(dbConn, rLst[n].RunId)
		if err != nil {
			omppLog.Log("Error at get run progress:", dn, ":", rdsn, ":", err.Error())
			return []db.RunPub{}, false // return empty result: run progress select error
		}

		// convert to "public" format
		rp, err := (&db.RunMeta{Run: rLst[n], Progress: rpRs}).ToPublic(meta)
		if err != nil {
			omppLog.Log("Error at run status conversion:", dn, ":", rdsn, ":", err.Error())
			return []db.RunPub{}, false // return empty result
		}
		rpLst = append(rpLst, *rp)
	}
	return rpLst, true
}

// FirstOrLastRunStatus return first or last or last completed run_lst db row and run_progress db rows by model digest-or-name.
func (mc *ModelCatalog) FirstOrLastRunStatus(dn string, isFirst, isCompleted bool) (*db.RunPub, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return &db.RunPub{}, false
	}
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return &db.RunPub{}, false
	}

	// get first or last or last completed run_lst db row
	rst := &db.RunRow{}
	var err error

	if isFirst {
		rst, err = db.GetFirstRun(dbConn, meta.Model.ModelId)
	} else {
		if !isCompleted {
			rst, err = db.GetLastRun(dbConn, meta.Model.ModelId)
		} else {
			rst, err = db.GetLastCompletedRun(dbConn, meta.Model.ModelId)
		}
	}
	if err != nil {
		omppLog.Log("Error at get run status:", dn, ":", err.Error())
		return &db.RunPub{}, false // return empty result: run select error
	}
	if rst == nil {
		// omppLog.Log("Warning: there is no run status not found for the model: ", dn)
		return &db.RunPub{}, false // return empty result: run_lst row not found
	}

	// get run sub-values progress for that run id
	rpRs, err := db.GetRunProgress(dbConn, rst.RunId)
	if err != nil {
		omppLog.Log("Error at get run progress:", dn, ":", err.Error())
		return &db.RunPub{}, false // return empty result: run progress select error
	}

	// convert to "public" format
	rp, err := (&db.RunMeta{Run: *rst, Progress: rpRs}).ToPublic(meta)
	if err != nil {
		omppLog.Log("Error at run status conversion:", dn, ":", err.Error())
		return &db.RunPub{}, false // return empty result
	}

	return rp, true
}

// RunRowList return list of run_lst db rows by model digest-or-name and run digest-stamp-or-name sorted by run_id.
// If there are multiple rows with same run stamp or run digest then multiple rows returned.
func (mc *ModelCatalog) RunRowList(dn string, rdsn string) ([]db.RunRow, bool) {

	// if model digest is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return []db.RunRow{}, false
	}
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return []db.RunRow{}, false
	}

	// get run_lst db rows by digest, stamp or run name
	rLst, err := db.GetRunListByDigestStampName(dbConn, meta.Model.ModelId, rdsn)
	if err != nil {
		omppLog.Log("Error at get run status:", dn, ":", rdsn, ":", err.Error())
		return []db.RunRow{}, false // return empty result: run select error
	}
	return rLst, true
}

// RunRowListByModel return list of run_lst db rows by model digest-or-name, sorted by run_id.
func (mc *ModelCatalog) RunRowListByModel(dn string) ([]db.RunRow, bool) {

	// if model digest is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return []db.RunRow{}, false
	}
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return []db.RunRow{}, false
	}

	// get run list
	rl, err := db.GetRunList(dbConn, meta.Model.ModelId)
	if err != nil {
		omppLog.Log("Error at get run list:", dn, ":", err.Error())
		return []db.RunRow{}, false // return empty result: run select error
	}
	return rl, true
}

// RunPubList return list of run_lst db rows in "public" format by model digest-or-name.
// No text info returned (no description and notes).
func (mc *ModelCatalog) RunPubList(dn string) ([]db.RunPub, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return []db.RunPub{}, false
	}
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return []db.RunPub{}, false
	}

	rl, err := db.GetRunList(dbConn, meta.Model.ModelId)
	if err != nil {
		omppLog.Log("Error at get run list:", dn, ":", err.Error())
		return []db.RunPub{}, false // return empty result: run select error
	}
	if len(rl) <= 0 {
		return []db.RunPub{}, true // return empty result: run_lst rows not found for that model
	}

	// for each run_lst convert it to "public" run format
	rpl := make([]db.RunPub, len(rl))

	for ni := range rl {

		p, err := (&db.RunMeta{Run: rl[ni]}).ToPublic(meta)
		if err != nil {
			omppLog.Log("Error at run conversion:", dn, ":", err.Error())
			return []db.RunPub{}, false // return empty result: conversion error
		}
		if p != nil {
			rpl[ni] = *p
		}
	}

	return rpl, true
}

// RunListText return list of run_lst and run_txt db rows by model digest-or-name.
// Text (description and notes) are in preferred language if text in such language exists.
func (mc *ModelCatalog) RunListText(dn string, preferredLang []language.Tag) ([]db.RunPub, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return []db.RunPub{}, false
	}
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return []db.RunPub{}, false
	}

	// match request preferred language
	lc := mc.languageTagMatch(dn, preferredLang)
	if lc == "" {
		omppLog.Log("Warning: invalid (empty) model default language or model not found:", dn)
		return []db.RunPub{}, false // return empty result: model default language cannot be empty
	}

	// get run_txt db row for each run_lst using matched preferred language
	rl, rt, err := db.GetRunListText(dbConn, meta.Model.ModelId, lc)
	if err != nil {
		omppLog.Log("Error at get run list:", dn, ":", err.Error())
		return []db.RunPub{}, false // return empty result: run select error
	}
	if len(rl) <= 0 {
		return []db.RunPub{}, true // return empty result: run_lst rows not found for that model
	}

	// for each run_lst find run_txt row if exist and convert to "public" run format
	rpl := make([]db.RunPub, len(rl))

	nt := 0
	for ni := range rl {

		// find text row for current master row by run id
		isFound := false
		for ; nt < len(rt); nt++ {
			isFound = rt[nt].RunId == rl[ni].RunId
			if rt[nt].RunId >= rl[ni].RunId {
				break // text found or text missing: text run id ahead of master run id
			}
		}

		// convert to "public" format
		var p *db.RunPub
		var err error

		if isFound && nt < len(rt) {
			p, err = (&db.RunMeta{Run: rl[ni], Txt: []db.RunTxtRow{rt[nt]}}).ToPublic(meta)
		} else {
			p, err = (&db.RunMeta{Run: rl[ni]}).ToPublic(meta)
		}
		if err != nil {
			omppLog.Log("Error at run conversion:", dn, ":", err.Error())
			return []db.RunPub{}, false // return empty result: conversion error
		}
		if p != nil {
			rpl[ni] = *p
		}
	}

	return rpl, true
}

// RunFull return full run metadata (without text) by model digest-or-name and run digest-or-stamp-or-name.
// It does not return run text metadata: decription and notes from run_txt and run_parameter_txt tables.
func (mc *ModelCatalog) RunFull(dn, rdsn string) (*db.RunPub, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return &db.RunPub{}, false
	}
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return &db.RunPub{}, false
	}

	// get run_lst db row by digest, stamp or run name
	r, err := db.GetRunByDigestStampName(dbConn, meta.Model.ModelId, rdsn)
	if err != nil {
		omppLog.Log("Error at get run db row:", dn, ": ", rdsn, ": ", err.Error())
		return &db.RunPub{}, false // return empty result: run select error
	}
	if r == nil {
		omppLog.Log("Warning run db row not found:", dn, ": ", rdsn)
		return &db.RunPub{}, false // return empty result: run_lst row not found
	}

	// get full metadata db rows
	rm, err := db.GetRunFull(dbConn, r)
	if err != nil {
		omppLog.Log("Error at get run:", dn, ":", r.Name, ":", err.Error())
		return &db.RunPub{}, false // return empty result: run select error
	}

	// convert to "public" model run format
	rp, err := rm.ToPublic(meta)
	if err != nil {
		omppLog.Log("Error at completed run conversion:", dn, ":", r.Name, ":", err.Error())
		return &db.RunPub{}, false // return empty result: conversion error
	}

	return rp, true
}

// RunTextFull return full run metadata (including text) by model digest-or-name and run digest-or-stamp-or-name.
// It does not return non-completed runs (run in progress).
// Run completed if run status one of: s=success, x=exit, e=error.
// Text (description and notes) can be in preferred language or all languages.
// If preferred language requested and it is not found in db then return empty text results.
func (mc *ModelCatalog) RunTextFull(dn, rdsn string, isAllLang bool, preferredLang []language.Tag) (*db.RunPub, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return &db.RunPub{}, false
	}
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return &db.RunPub{}, false
	}

	// get run_lst db row by digest, stamp or run name
	r, err := db.GetRunByDigestStampName(dbConn, meta.Model.ModelId, rdsn)
	if err != nil {
		omppLog.Log("Error at get run db row:", dn, ": ", rdsn, ": ", err.Error())
		return &db.RunPub{}, false // return empty result: run select error
	}
	if r == nil {
		omppLog.Log("Warning run db row not found:", dn, ": ", rdsn)
		return &db.RunPub{}, false // return empty result: run_lst row not found
	}

	// get full metadata db rows using matched preferred language or in all languages
	lc := ""
	if !isAllLang {
		lc = mc.languageTagMatch(dn, preferredLang)
		if lc == "" {
			omppLog.Log("Warning: invalid (empty) model default language or model not found:", dn)
			return &db.RunPub{}, false // return empty result: model default language cannot be empty
		}
	}

	rm, err := db.GetRunFullText(dbConn, r, false, lc)
	if err != nil {
		omppLog.Log("Error at get run text:", dn, ":", r.Name, ":", err.Error())
		return &db.RunPub{}, false // return empty result: run select error
	}

	// convert to "public" model run format
	rp, err := rm.ToPublic(meta)
	if err != nil {
		omppLog.Log("Error at completed run conversion:", dn, ":", r.Name, ":", err.Error())
		return &db.RunPub{}, false // return empty result: conversion error
	}

	return rp, true
}
