// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// userViewGetHandler return user views by model name or digest:
// GET /api/user/view/model/:model
// If multiple models with same name exist only one is returned.
// If no model view file in user home directory then response is 200 OK with is empty {} json payload
func userViewGetHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	if !theCfg.isHome {
		http.Error(w, helper.MsgL(lang, "Forbidden: model view reading disabled on the server"), http.StatusForbidden)
		return
	}

	dn := getRequestParam(r, "model")

	// find model by digest or name
	m, ok := theCatalog.ModelDicByDigestOrName(dn)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error: model not found", dn), http.StatusNotFound)
		return // not found error: model not found in model catalog
	}

	// open model.view.json file from user home directory
	// if model.view.json not exist then return empty object {} response
	fileName := m.Name + ".view.json"
	bt, err := os.ReadFile(filepath.Join(theCfg.homeDir, fileName))
	if err != nil {
		if os.IsNotExist(err) {
			jsonResponseBytes(w, r, []byte{})
			return // empty result: model view file not exist
		}
		omppLog.Log("Error: unable to read from", fileName, err)
		http.Error(w, helper.MsgL(lang, "Error: unable to read from", fileName), http.StatusInternalServerError)
		return // model view file read error
	}

	// send file content as json response body
	jsonResponseBytes(w, r, bt)
}

// userViewPutHandler write user views json body into home/user/modelName.view.json file:
// PUT  /api/user/view/model/:model
// Model parameter can be a model name or digest.
// If multiple models with same name exist only one is used.
// If file name already exist in home directory it is truncated.
func userViewPutHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	if !theCfg.isHome {
		http.Error(w, helper.MsgL(lang, "Forbidden: model view saving disabled on the server"), http.StatusForbidden)
		return
	}

	dn := getRequestParam(r, "model")

	// find model by digest or name
	m, ok := theCatalog.ModelDicByDigestOrName(dn)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error: model not found", dn), http.StatusNotFound)
		return // not found error: model not found in model catalog
	}

	// copy request body into home/user/model.view.json file
	_ = jsonRequestToFile(w, r, filepath.Join(theCfg.homeDir, m.Name+".view.json"))
}

// userViewDeleteHandler delete model.view.json file from user home directory:
// DELETE /api/user/view/model/:model
// Model parameter can be a model name or digest.
// If multiple models with same name exist only one is used.
func userViewDeleteHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	if !theCfg.isHome {
		http.Error(w, helper.MsgL(lang, "Forbidden: model view saving disabled on the server"), http.StatusForbidden)
		return
	}

	dn := getRequestParam(r, "model")

	// find model by digest or name
	m, ok := theCatalog.ModelDicByDigestOrName(dn)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error: model not found", dn), http.StatusNotFound)
		return // not found error: model not found in model catalog
	}

	// delete model views file from home directory
	fName := m.Name + ".view.json"
	err := os.Remove(filepath.Join(theCfg.homeDir, fName))
	if err != nil {
		if !os.IsNotExist(err) {
			omppLog.Log("Error: unable to delete file", fName, err)
			http.Error(w, helper.MsgL(lang, "Error: unable to delete file", fName), http.StatusInternalServerError)
			return // model view file read error
		}
	}
	// on success
	w.Header().Set("Content-Location", "/api/user/view/model/"+dn)
	w.Header().Set("Content-Type", "text/plain")
}
