// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
	"golang.org/x/text/language"
)

// WorksetByName select workset_lst db row by workset name and model digest or name.
// Return:
//
//	workset_lst db row: nil on error, empty row if workset not found
//	boolean flag: true if workset found, false if not found or error
func (mc *ModelCatalog) WorksetByName(dn string, wsn string) (*db.WorksetRow, bool) {

	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return nil, false
	}
	if wsn == "" {
		omppLog.Log("Warning: invalid (empty) workset name")
		return nil, false
	}
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return nil, false // return empty result: model not found or error
	}

	// get workset_lst db row
	ws, err := db.GetWorksetByName(dbConn, meta.Model.ModelId, wsn)
	if err != nil {
		omppLog.Log("Workset not found or error at get workset status: ", meta.Model.Name, ": ", wsn, ": ", err.Error())
		return nil, false // return empty result: workset select error
	}
	if ws == nil {
		// omppLog.Log("Warning: workset not found: ", meta.Model.Name, ": ", wsn)
		return &db.WorksetRow{}, false // return empty result: workset_lst row not found
	}

	return ws, true
}

// WorksetDefaultStatus return workset_lst db row of default workset by model digest-or-name.
func (mc *ModelCatalog) WorksetDefaultStatus(dn string) (*db.WorksetRow, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return &db.WorksetRow{}, false
	}

	// get model metadata and database connection
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return &db.WorksetRow{}, false
	}

	// get workset_lst db row for default workset
	w, err := db.GetDefaultWorkset(dbConn, meta.Model.ModelId)
	if err != nil {
		omppLog.Log("Error at get default workset status: ", dn, ": ", err.Error())
		return &db.WorksetRow{}, false // return empty result: workset select error
	}
	if w == nil {
		omppLog.Log("Warning default workset status not found: ", dn)
		return &db.WorksetRow{}, false // return empty result: workset_lst row not found
	}

	return w, true
}

// WorksetRowListByModelDigest return list of workset_lst db rows by model digest, sorted by workset_id.
func (mc *ModelCatalog) WorksetRowListByModelDigest(digest string) ([]db.WorksetRow, bool) {

	// if model digest is empty then return empty results
	if digest == "" {
		omppLog.Log("Warning: invalid (empty) model digest")
		return []db.WorksetRow{}, false
	}

	// get model metadata and database connection
	meta, dbConn, ok := mc.modelMeta(digest)
	if !ok {
		omppLog.Log("Warning: model digest not found: ", digest)
		return []db.WorksetRow{}, false
	}

	// get workset list
	wl, err := db.GetWorksetList(dbConn, meta.Model.ModelId)
	if err != nil {
		omppLog.Log("Error at get workset list: ", digest, ": ", err.Error())
		return []db.WorksetRow{}, false // return empty result: workset select error
	}
	return wl, true
}

// WorksetPubList return list of workset_lst db rows by model digest-or-name.
// No text info returned (no description and notes).
func (mc *ModelCatalog) WorksetPubList(dn string) ([]db.WorksetPub, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return []db.WorksetPub{}, false
	}

	// get model metadata and database connection
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return []db.WorksetPub{}, false // return empty result: model not found or error
	}

	// read model workset list
	wl, err := db.GetWorksetList(dbConn, meta.Model.ModelId)
	if err != nil {
		omppLog.Log("Error at get workset list: ", dn, ": ", err.Error())
		return []db.WorksetPub{}, false // return empty result: workset select error
	}
	if len(wl) <= 0 {
		// omppLog.Log("Warning: there is no any worksets found for the model: ", dn)
		return []db.WorksetPub{}, false // return empty result: workset_lst rows not found for that model
	}

	// for each workset_lst convert it to "public" workset format
	wpl := make([]db.WorksetPub, len(wl))

	for ni := range wl {

		p, err := (&db.WorksetMeta{Set: wl[ni]}).ToPublic(dbConn, meta)
		if err != nil {
			omppLog.Log("Error at workset conversion: ", dn, ": ", err.Error())
			return []db.WorksetPub{}, false // return empty result: conversion error
		}
		if p != nil {
			wpl[ni] = *p
		}
	}

	return wpl, true
}

// WorksetListText return list of workset_lst and workset_txt db rows by model digest-or-name.
// Text (description and notes) are in preferred language if text in such language exists.
func (mc *ModelCatalog) WorksetListText(dn string, preferredLang []language.Tag) ([]db.WorksetPub, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return []db.WorksetPub{}, false
	}

	// get model metadata and database connection
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return []db.WorksetPub{}, false // return empty result: model not found or error
	}

	// match request preferred language
	lc := mc.languageTagMatch(dn, preferredLang)
	if lc == "" {
		omppLog.Log("Warning: invalid (empty) model default language or model not found:", dn)
		return []db.WorksetPub{}, false // return empty result: model default language cannot be empty
	}

	// get workset_txt db row for each workset_lst using matched preferred language
	wl, wt, err := db.GetWorksetListText(dbConn, meta.Model.ModelId, lc)
	if err != nil {
		omppLog.Log("Error at get workset list: ", dn, ": ", err.Error())
		return []db.WorksetPub{}, false // return empty result: workset select error
	}
	if len(wl) <= 0 {
		omppLog.Log("Warning: default workset not exist for the model: ", dn) // at least default workset must exist for every model
		return []db.WorksetPub{}, false                                       // return empty result: workset_lst rows not found for that model
	}

	// for each workset_lst find workset_txt row if exist and convert to "public" workset format
	wpl := make([]db.WorksetPub, len(wl))

	nt := 0
	for ni := range wl {

		// find text row for current master row by set id
		isFound := false
		for ; nt < len(wt); nt++ {
			isFound = wt[nt].SetId == wl[ni].SetId
			if wt[nt].SetId >= wl[ni].SetId {
				break // text found or text missing: text set id ahead of master set id
			}
		}

		// convert to "public" format
		var p *db.WorksetPub
		var err error

		if isFound && nt < len(wt) {
			p, err = (&db.WorksetMeta{Set: wl[ni], Txt: []db.WorksetTxtRow{wt[nt]}}).ToPublic(dbConn, meta)
		} else {
			p, err = (&db.WorksetMeta{Set: wl[ni]}).ToPublic(dbConn, meta)
		}
		if err != nil {
			omppLog.Log("Error at workset conversion: ", dn, ": ", err.Error())
			return []db.WorksetPub{}, false // return empty result: conversion error
		}
		if p != nil {
			wpl[ni] = *p
		}
	}

	return wpl, true
}

// WorksetTextFull return full workset metadata by model digest-or-name and workset name.
// Text (description and notes) can be in preferred language or all languages.
// If preferred language requested and it is not found in db then return empty text results.
func (mc *ModelCatalog) WorksetTextFull(dn, wsn string, isAllLang bool, preferredLang []language.Tag) (*db.WorksetPub, bool, error) {

	// if model digest-or-name or workset name is empty then return empty results
	if dn == "" {
		omppLog.Log("Warning: invalid (empty) model digest and name")
		return &db.WorksetPub{}, false, nil
	}
	if wsn == "" {
		omppLog.Log("Warning: invalid (empty) workset name")
		return &db.WorksetPub{}, false, nil
	}

	// get model metadata and database connection
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return &db.WorksetPub{}, false, nil // return empty result: model not found or error
	}

	// get workset_lst db row by name
	w, err := db.GetWorksetByName(dbConn, meta.Model.ModelId, wsn)
	if err != nil {
		omppLog.Log("Error at get workset status: ", dn, ": ", wsn, ": ", err.Error())
		return &db.WorksetPub{}, false, err // return empty result: workset select error
	}
	if w == nil {
		// omppLog.Log("Warning workset status not found: ", dn, ": ", wsn)
		return &db.WorksetPub{}, false, nil // return empty result: workset_lst row not found
	}

	// get full workset metadata using matched preferred language or in all languages
	lc := ""
	if !isAllLang {

		lc = mc.languageTagMatch(dn, preferredLang)
		if lc == "" {
			omppLog.Log("Error: invalid (empty) model default language: ", dn, ": ", w.Name)
			return &db.WorksetPub{}, false, err // return empty result: workset select error
		}
	}

	wm, err := db.GetWorksetFull(dbConn, w, lc)
	if err != nil {
		omppLog.Log("Error at get workset metadata: ", dn, ": ", w.Name, ": ", err.Error())
		return &db.WorksetPub{}, false, err // return empty result: workset select error
	}

	// convert to "public" model workset format
	wp, err := wm.ToPublic(dbConn, meta)
	if err != nil {
		omppLog.Log("Error at workset conversion: ", dn, ": ", w.Name, ": ", err.Error())
		return &db.WorksetPub{}, false, err // return empty result: conversion error
	}

	return wp, true, nil
}
