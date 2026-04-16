// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"encoding/json"
	"net/http"

	"github.com/stcopenmpp/ompp-go/ompp"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// Get model metadata, including language-specific text:
// GET /api/model/:model/text
// GET /api/model/:model/text/lang/:lang
// Model digest-or-name must specified, if multiple models with same name exist only one is returned.
// If optional lang specified then result in that language else in browser language or model default.
func modelTextHandler(w http.ResponseWriter, r *http.Request) {
	doModelTextHandler(w, r, false)
}

// Get model metadata, including language-specific text with packed range types:
// GET /api/model/:model/pack/text
// GET /api/model/:model/pack/text/lang/:lang
// Model digest-or-name must specified, if multiple models with same name exist only one is returned.
// If optional lang specified then result in that language else in browser language or model default.
func modelTextPackHandler(w http.ResponseWriter, r *http.Request) {
	doModelTextHandler(w, r, true)
}

// Get model metadata, including language-specific text.
// If isPack is true then return "packed" range types as [min, max] enum id's, not as full enum array.
// Model digest-or-name must specified, if multiple models with same name exist only one is returned.
// If optional lang specified then result in that language else in browser language or model default.
func doModelTextHandler(w http.ResponseWriter, r *http.Request, isPack bool) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// find model metadata in model catalog and get language-specific model metadata.
	// It can be in default model language or empty if no model text db rows exist.
	var me ompp.ModelMetaEncoder

	getText := func(mc *ModelCatalog, dn string, mdRow *db.ModelDicRow, lc string, lcd string) bool {

		mc.theLock.Lock()
		defer mc.theLock.Unlock()

		imdl, ok := mc.indexByDigest(mdRow.Digest)
		if !ok {
			omppLog.Log("Warning: model digest or name not found:", dn)
			return false // return empty result: model not found or error
		}
		if e := me.New(mc.modelLst[imdl].meta, mc.modelLst[imdl].txtMeta, lc, lcd); e != nil {
			omppLog.Log("Error: invalid (empty) model metadata")
			return false
		}
		return true
	}

	//
	// actual http handler
	//

	dn := getRequestParam(r, "model")
	rqLangTags := getRequestLang(r, "lang") // get optional language argument and languages accepted by browser

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		http.Error(w, helper.MsgL(lang, "Invalid (empty) model digest and name"), http.StatusBadRequest)
		return
	}

	// find model in catalog
	mdRow, ok := theCatalog.ModelDicByDigestOrName(dn)
	if !ok {
		omppLog.Log("Error: model digest or name not found:", dn)
		http.Error(w, helper.MsgL(lang, "Model digest or name not found:", dn), http.StatusBadRequest)
		return
	}

	// if language-specific model metadata not loaded then read it from database
	if ok := theCatalog.loadModelText(mdRow.Digest); !ok {
		omppLog.Log("Error: Model text metadata not found:", dn)
		http.Error(w, helper.MsgL(lang, "Model text metadata not found:", dn), http.StatusBadRequest)
		return
	}

	// match preferred languages and model languages
	lc := theCatalog.languageTagMatch(mdRow.Digest, rqLangTags)
	lcd, _, _ := theCatalog.modelLangs(mdRow.Digest)
	if lc == "" && lcd == "" {
		omppLog.Log("Error: invalid (empty) model default language:", dn)
		http.Error(w, helper.MsgL(lang, "Invalid (empty) model default language:", dn), http.StatusBadRequest)
		return
	}

	isOk := getText(&theCatalog, dn, &mdRow, lc, lcd)
	if !isOk {
		http.Error(w, helper.MsgL(lang, "Model not found:", mdRow.Name, dn), http.StatusBadRequest)
		return
	}

	// write json response
	jsonSetHeaders(w, r)

	err := me.DoEncode(isPack, json.NewEncoder(w))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
