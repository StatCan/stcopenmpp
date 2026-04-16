// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"net/http"
	"net/url"

	"github.com/husobee/vestigo"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
)

// homeHandler is static pages handler for front-end UI served on web / root.
// Only GET requests expected.
func homeHandler(w http.ResponseWriter, r *http.Request) {
	setContentType(http.FileServer(http.Dir(theCfg.htmlDir))).ServeHTTP(w, r)
}

// static file download handler from user home/io/download or home/io/upload and subfolders.
// URLs served from home/io directory are:
//
//	https://domain.name/download/file.name
//	https://domain.name/upload/file.name
//
// Only GET requests expected.
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	http.FileServer(http.Dir(theCfg.inOutDir)).ServeHTTP(w, r)
}

// static file download handler from user files directory and subfolders, if userhome specified then it is home/io.
// URLs served from home/io directory are:
//
//	https://domain.name/files/file.name
//	https://domain.name/files/download/file.name
//	https://domain.name/files/upload/file.name
//
// Only GET requests expected.
func filesHandler(w http.ResponseWriter, r *http.Request) {
	http.StripPrefix("/files/", http.FileServer(http.Dir(theCfg.filesDir))).ServeHTTP(w, r)
}

// modelDocHandler is static pages handler for model documentation served /doc URLs.
// Files served from models/doc directory URLs are:
//
//	https://domain.name/doc/any-dir/ModelName.doc.EN.html
//	https://domain.name/doc/any-dir/ModelName.doc.FR.html
//
// Model documentation file path must be specified through ModelName.extra.json file.
// It must be relative to models, for example:
//
//	ModelName.extra.json: any-dir/ModelName.doc.FR.html
//	result in URL:        https://domain.name/doc/any-dir/ModelName.doc.FR.html
//
// Only GET requests expected.
func modelDocHandler(w http.ResponseWriter, r *http.Request) {
	setContentType(http.StripPrefix("/doc/", http.FileServer(http.Dir(theCfg.docDir)))).ServeHTTP(w, r)
}

// serve UI routes /model/*
func uiModelRoutes(router *vestigo.Router) {

	// GET /model/:digest/run-list   => /?model=digest&run-list=
	// GET /model/:digest/set-list   => /?model=digest&set-list=
	// GET /model/:digest/run/:run/table/:name   => /?model=digest&run=run&table=name
	// GET /model/:digest/run/:run/parameter/:name   => /?model=digest&run=run&parameter=name
	// GET /model/:digest/set/:setName/parameter/:name => /?model=digest&set=setName&parameter=name
	// GET /model/:digest/run/:run/entity/:name => /?model=digest&run=run&entity=name
	// GET /model/:digest/run-log/:stamp => /?model=digest&run-log=stamp
	router.Get("/model/:digest/run-list", uiModelRunListHandler, logRequest)
	router.Get("/model/:digest/set-list", uiModelSetListHandler, logRequest)
	router.Get("/model/:digest/run/:run/parameter/:name", uiModelRunParamHandler, logRequest)
	router.Get("/model/:digest/set/:setName/parameter/:name", uiModelSetParamHandler, logRequest)
	router.Get("/model/:digest/run/:run/table/:name", uiModelRunTableHandler, logRequest)
	router.Get("/model/:digest/run/:run/entity/:name", uiModelRunEntityHandler, logRequest)
	router.Get("/model/:digest/run-log/:stamp", uiModelRunLogHandler, logRequest)

	// router.Get("/model/*", homeHandler, logRequest) // any other /model/* serve as UI / root
}

// redirect GET /model/:digest/run-list => /?model=digest&run-list=
func uiModelRunListHandler(w http.ResponseWriter, r *http.Request) {

	u, v, err := uiModelUrl(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	v.Set("run-list", "")
	u.RawQuery = v.Encode()
	http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
}

// redirect GET /model/:digest/set-list => /?model=digest&set-list=
func uiModelSetListHandler(w http.ResponseWriter, r *http.Request) {

	u, v, err := uiModelUrl(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	v.Set("set-list", "")
	u.RawQuery = v.Encode()
	http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
}

// redirect /model/:digest/run/:run/parameter/:name => /?model=digest&run=run&parameter=name
func uiModelRunParamHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "digest")
	rdsn := getRequestParam(r, "run")
	name := getRequestParam(r, "name") // parameter name

	// model digest or name must exists
	meta, ok := theCatalog.ModelMetaByDigestOrName(dn)
	if !ok {
		http.Error(w, helper.Msg("Error: model digest or name not found:", dn), http.StatusBadRequest)
		return
	}

	// check if parameter name exist in the model
	if _, ok := meta.ParamByName(name); !ok {
		http.Error(w, helper.Msg("Error: model parameter not found:", dn, ":", name), http.StatusBadRequest)
		return
	}

	// run must be completed or in progress
	rRow := theCatalog.RunRow(dn, rdsn)
	if rRow == nil || !db.IsRunCompletedOrProgress(rRow.Status) {
		http.Error(w, helper.Msg("Model run not found:", rdsn), http.StatusBadRequest)
		return
	}

	// redirect to /?model=digest&run=run&parameter=name
	u, v, err := uiModelUrl(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	v.Set("run", rdsn)
	v.Set("parameter", name)
	u.RawQuery = v.Encode()
	http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
}

// redirect GET /model/:digest/set/:setName/parameter/:name => /?model=digest&set=setName&parameter=name
func uiModelSetParamHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "digest")
	wsName := getRequestParam(r, "setName")
	name := getRequestParam(r, "name") // parameter name

	// model digest or name must exists
	meta, ok := theCatalog.ModelMetaByDigestOrName(dn)
	if !ok {
		http.Error(w, helper.Msg("Error: model digest or name not found:", dn), http.StatusBadRequest)
		return
	}

	// check if parameter name exist in the model
	if _, ok := meta.ParamByName(name); !ok {
		http.Error(w, helper.Msg("Error: model parameter not found:", dn, ":", name), http.StatusBadRequest)
		return
	}

	// as it is today we are not checking if parameter exists in workset

	// workset must be readonly
	wsRow, ok := theCatalog.WorksetByName(dn, wsName)
	if !ok || wsRow == nil {
		http.Error(w, helper.Msg("Model scenario not found:", wsName), http.StatusBadRequest)
		return
	}
	if !wsRow.IsReadonly {
		http.Error(w, helper.Msg("Model scenario must be read-only:", wsName), http.StatusBadRequest)
		return
	}

	// redirect to /?model=digest&set=setName&parameter=name
	u, v, err := uiModelUrl(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	v.Set("set", wsName)
	v.Set("parameter", name)
	u.RawQuery = v.Encode()
	http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
}

// redirect GET /model/:digest/run/:run/table/:name   => /?model=digest&run=run&table=name
func uiModelRunTableHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "digest")
	rdsn := getRequestParam(r, "run")
	name := getRequestParam(r, "name") // table name

	// model digest or name must exists
	meta, ok := theCatalog.ModelMetaByDigestOrName(dn)
	if !ok {
		http.Error(w, helper.Msg("Error: model digest or name not found:", dn), http.StatusBadRequest)
		return
	}

	// run must be completed sucessfully
	rRow := theCatalog.RunRow(dn, rdsn)
	if rRow == nil {
		http.Error(w, helper.Msg("Model run not found:", rdsn), http.StatusBadRequest)
		return
	}
	if rRow.Status != db.DoneRunStatus {
		http.Error(w, helper.Msg("Error: model run not completed successfully:", rdsn, rRow.Name), http.StatusBadRequest)
		return
	}

	// check if table name exist in the model
	if _, ok := meta.OutTableByName(name); !ok {
		http.Error(w, helper.Msg("Error: model output table not found:", dn, ":", name), http.StatusBadRequest)
		return
	}

	// as it is today we are not checking if output table exists in model run

	// redirect to /?model=digest&run=run&table=name
	u, v, err := uiModelUrl(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	v.Set("run", rdsn)
	v.Set("table", name)
	u.RawQuery = v.Encode()
	http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
}

// redirect GET /model/:digest/run/:run/entity/:name => /?model=digest&run=run&entity=name
func uiModelRunEntityHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "digest")
	rdsn := getRequestParam(r, "run")
	name := getRequestParam(r, "name") // entity name

	// model digest or name must exists
	meta, ok := theCatalog.ModelMetaByDigestOrName(dn)
	if !ok {
		http.Error(w, helper.Msg("Error: model digest or name not found:", dn), http.StatusBadRequest)
		return
	}

	// run must be completed sucessfully
	rRow := theCatalog.RunRow(dn, rdsn)
	if rRow == nil {
		http.Error(w, helper.Msg("Model run not found:", rdsn), http.StatusBadRequest)
		return
	}
	if rRow.Status != db.DoneRunStatus {
		http.Error(w, helper.Msg("Error: model run not completed successfully:", rdsn, rRow.Name), http.StatusBadRequest)
		return
	}

	// check if entity name exist in the model
	if _, ok := meta.EntityByName(name); !ok {
		http.Error(w, helper.Msg("Error: model entity not found:", dn, ":", name), http.StatusBadRequest)
		return
	}

	// as it is today we are not checking if entity generation exists in model run

	// redirect to /?model=digest&run=run&entity=name
	u, v, err := uiModelUrl(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	v.Set("run", rdsn)
	v.Set("entity", name)
	u.RawQuery = v.Encode()
	http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
}

// redirect GET /model/:digest/run-log/:stamp => /?model=digest&run-log=stamp
func uiModelRunLogHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "digest")
	stamp := getRequestParam(r, "stamp") // mdel run stamp

	// model run must exist
	rRow := theCatalog.RunRow(dn, stamp)
	if rRow == nil {
		http.Error(w, helper.Msg("Model run not found:", stamp), http.StatusBadRequest)
		return
	}

	// redirect to /?model=digest&run-log=stamp
	u, v, err := uiModelUrl(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	v.Set("run-log", stamp)
	u.RawQuery = v.Encode()
	http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
}

// make parts of model UI url start: /?model=digest
func uiModelUrl(r *http.Request) (*url.URL, url.Values, error) {

	//	GET /model/:digest/run-list
	dn := getRequestParam(r, "digest")

	// model digest or name must exists
	if _, ok := theCatalog.ModelDicByDigestOrName(dn); !ok {
		return nil, url.Values{}, helper.ErrorNew("Model digest or name not found:", dn)
	}

	// set url to UI / root
	u, err := r.URL.Parse("/")
	if err != nil {
		return u, url.Values{}, err
	}

	// append to query model=digest
	v := url.Values{}
	v.Set("model", dn)

	return u, v, nil
}
