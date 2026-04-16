// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// modelDownloadPostHandler initate creation of model zip archive in home/io/download folder.
// POST /api/download/model/:model
// POST /api/download/model/:model/lang/:lang
// Zip archive is the same as created by dbcopy command line utilty.
// Dimension(s) and enum-based parameters returned as enum codes, not enum id's.
// Json is posted to specify download options.
// If NoAccumulatorsCsv is true then accumulators CSV files are not included in result.
// It is significantly faster to porduce the result archive, we but cannot import it back into the model database,
// it is only to analyze model output values CSV data using some other tools
// If NoMicrodata is true then microdata not included in result.
// If Utf8BomIntoCsv is true then add utf-8 byte order mark into csv files
func modelDownloadPostHandler(w http.ResponseWriter, r *http.Request) {

	// url or query parameters
	dn := getRequestParam(r, "model")      // model digest-or-name
	lang := preferedRequestLang(r, "lang") // get prefered language for dbcopy log messages

	// decode json download options
	opts := struct {
		NoAccumulatorsCsv bool
		NoMicrodata       bool
		Utf8BomIntoCsv    bool
		IdCsv             bool
	}{}
	if !jsonRequestDecode(w, r, false, &opts) {
		return // error at json decode, response done with http error
	}
	if !theCfg.isMicrodata {
		opts.NoMicrodata = true // microdata output disabled
	}

	// find model metadata by digest or name
	mb, ok := theCatalog.modelBasicByDigestOrName(dn)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Model not found:", dn), http.StatusBadRequest)
		return // empty result: model digest not found
	}
	m, ok := theCatalog.ModelDicByDigest(mb.model.Digest)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Model not found:", dn), http.StatusBadRequest)
		return // empty result: model digest not found
	}

	// base part of: output directory name, .zip file name and log file name
	baseName := mb.model.Name
	omppLog.Log("Download of:", baseName)

	// if download.progress.log file exist the retun error: download in progress
	logPath := filepath.Join(theCfg.downloadDir, baseName+".progress.download.log")
	if helper.IsFileExist(logPath) {
		omppLog.Log("Error: download already in progress:", logPath)
		http.Error(w, helper.MsgL(lang, "Model download already in progress:", baseName), http.StatusBadRequest)
		return
	}

	// create new download.progress.log file and write model decsription
	isLog := fileCreateEmpty(false, logPath)
	if !isLog {
		omppLog.Log("Failed to create download log file:", baseName+".progress.download.log")
		http.Error(w, helper.MsgL(lang, "Model download failed:", baseName), http.StatusBadRequest)
		return
	}
	// do not translate hdrMsg strings below
	hdrMsg := []string{
		"---------------",
		"Model Name    : " + m.Name,
		"Model Version : " + m.Version + " " + m.CreateDateTime,
		"Model Digest  : " + m.Digest,
		"Folder        : " + baseName,
		"---------------",
	}
	if !writeToCmdLog(logPath, true, helper.MsgL(lang, "Download of:", baseName)) {
		renameToDownloadErrorLog(logPath, "", nil)
		omppLog.Log("Failed to write into download log file:", baseName+".progress.download.log")
		http.Error(w, helper.MsgL(lang, "Model download failed:", baseName), http.StatusBadRequest)
		return
	}
	if !writeToCmdLog(logPath, false, hdrMsg...) {
		renameToDownloadErrorLog(logPath, "", nil)
		omppLog.Log("Failed to write into download log file:", baseName+".progress.download.log")
		http.Error(w, helper.MsgL(lang, "Model download failed:", baseName), http.StatusBadRequest)
		return
	}

	// create model download files on separate thread
	cmd, cmdMsg := makeModelDownloadCommand(mb, logPath, opts.NoAccumulatorsCsv, opts.NoMicrodata, opts.Utf8BomIntoCsv, opts.IdCsv, lang)

	go makeDownload(baseName, cmd, cmdMsg, logPath)

	// report to the client results location
	w.Header().Set("Content-Location", "/api/download/model/"+dn+"/"+baseName)
}

// runDownloadPostHandler initate creation of model run zip archive in home/io/download folder.
// POST /api/download/model/:model/run/:run
// POST /api/download/model/:model/run/:run/lang/:lang
// Zip archive is the same as created by dbcopy command line utilty.
// Dimension(s) and enum-based parameters returned as enum codes, not enum id's.
// Json is posted to specify download options.
// If NoAccumulatorsCsv is true then accumulators CSV files are not included in result.
// It is significantly faster to porduce the result archive, we but cannot import it back into the model database,
// it is only to analyze model output values CSV data using some other tools
// If NoMicrodata is true then microdata not included in result.
// If Utf8BomIntoCsv is true then add utf-8 byte order mark into csv files
func runDownloadPostHandler(w http.ResponseWriter, r *http.Request) {

	// url or query parameters
	dn := getRequestParam(r, "model")      // model digest-or-name
	rdsn := getRequestParam(r, "run")      // run digest-or-stamp-or-name
	lang := preferedRequestLang(r, "lang") // get prefered language for dbcopy log messages

	// decode json download options
	opts := struct {
		NoAccumulatorsCsv bool
		NoMicrodata       bool
		Utf8BomIntoCsv    bool
		IdCsv             bool
	}{}
	if !jsonRequestDecode(w, r, false, &opts) {
		return // error at json decode, response done with http error
	}
	if !theCfg.isMicrodata {
		opts.NoMicrodata = true // microdata output disabled
	}

	// find model metadata by digest or name
	mb, ok := theCatalog.modelBasicByDigestOrName(dn)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Model not found:", dn), http.StatusBadRequest)
		return // empty result: model digest not found
	}
	m, ok := theCatalog.ModelDicByDigest(mb.model.Digest)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Model not found:", dn), http.StatusBadRequest)
		return // empty result: model digest not found
	}

	// find all model runs by run digest, run stamp or name, check run status: it must be success
	rst, ok := theCatalog.RunRowList(mb.model.Digest, rdsn)
	if !ok || len(rst) <= 0 {
		http.Error(w, helper.MsgL(lang, "Model run not found:", mb.model.Name, dn, rdsn), http.StatusBadRequest)
		return // empty result: model run not found
	}
	if len(rst) > 1 {
		omppLog.Log("Warning: multiple model runs found, using first one of:", mb.model.Name, dn, rdsn)
	}
	r0 := rst[0] // first run, if there are multiple with the same stamp or name

	if r0.Status != db.DoneRunStatus {
		http.Error(w, helper.MsgL(lang, "Model run is not completed successfully:", mb.model.Name, dn, rdsn), http.StatusBadRequest)
		return // empty result: run status must be success
	}

	// base part of: output directory name, .zip file name and log file name
	baseName := mb.model.Name + ".run." + helper.CleanFileName(r0.Name)
	omppLog.Log("Download of:", baseName)

	// if download.progress.log file exist the retun error: download in progress
	logPath := filepath.Join(theCfg.downloadDir, baseName+".progress.download.log")
	if helper.IsFileExist(logPath) {
		omppLog.Log("Error: download already in progress:", logPath)
		http.Error(w, helper.MsgL(lang, "Model run download already in progress:", baseName), http.StatusBadRequest)
		return
	}

	// create new download.progress.log file and write model run decsription
	isLog := fileCreateEmpty(false, logPath)
	if !isLog {
		omppLog.Log("Failed to create download log file:", baseName+".progress.download.log")
		http.Error(w, helper.MsgL(lang, "Model run download failed:", baseName), http.StatusBadRequest)
		return
	}
	// do not translate hdrMsg strings below
	hdrMsg := []string{
		"---------------",
		"Model Name    : " + m.Name,
		"Model Version : " + m.Version + " " + m.CreateDateTime,
		"Model Digest  : " + m.Digest,
		"Run Name      : " + r0.Name,
		"Run Version   : " + strconv.Itoa(r0.RunId) + " " + r0.CreateDateTime,
		"Run Digest    : " + r0.RunDigest,
		"Folder        : " + baseName,
		"---------------",
	}
	if !writeToCmdLog(logPath, true, "Download of:", baseName) {
		renameToDownloadErrorLog(logPath, "", nil)
		omppLog.Log("Failed to write into download log file:", baseName+".progress.download.log")
		http.Error(w, helper.MsgL(lang, "Model run download failed:", baseName), http.StatusBadRequest)
		return
	}
	if !writeToCmdLog(logPath, false, hdrMsg...) {
		renameToDownloadErrorLog(logPath, "", nil)
		omppLog.Log("Failed to write into download log file:", baseName+".progress.download.log")
		http.Error(w, helper.MsgL(lang, "Model run download failed:", baseName), http.StatusBadRequest)
		return
	}

	// create model run download files on separate thread
	cmd, cmdMsg := makeRunDownloadCommand(mb, r0.RunId, logPath, opts.NoAccumulatorsCsv, opts.NoMicrodata, opts.Utf8BomIntoCsv, opts.IdCsv, lang)

	go makeDownload(baseName, cmd, cmdMsg, logPath)

	// report to the client results location
	w.Header().Set("Content-Location", "/api/download/model/"+dn+"/run/"+rdsn+"/"+baseName)
}

// worksetDownloadPostHandler initate creation of model workset zip archive in home/io/download folder.
// POST /api/download/model/:model/workset/:set
// POST /api/download/model/:model/workset/:set/lang/:lang
// Zip archive is the same as created by dbcopy command line utilty.
// Dimension(s) and enum-based parameters returned as enum codes, not enum id's.
// Json is posted to specify download options.
// If Utf8BomIntoCsv is true then add utf-8 byte order mark into csv files
func worksetDownloadPostHandler(w http.ResponseWriter, r *http.Request) {

	// url or query parameters
	dn := getRequestParam(r, "model")      // model digest-or-name
	wsn := getRequestParam(r, "set")       // workset name
	lang := preferedRequestLang(r, "lang") // get prefered language for dbcopy log messages

	// decode json download options
	opts := struct {
		Utf8BomIntoCsv bool
		IdCsv          bool
	}{}

	if !jsonRequestDecode(w, r, false, &opts) {
		return // error at json decode, response done with http error
	}

	// find model metadata by digest or name
	mb, ok := theCatalog.modelBasicByDigestOrName(dn)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Model not found:", dn), http.StatusBadRequest)
		return // empty result: model digest not found
	}
	m, ok := theCatalog.ModelDicByDigest(mb.model.Digest)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Model not found:", dn), http.StatusBadRequest)
		return // empty result: model digest not found
	}

	// find workset by name and check status: it must be read-only
	ws, ok := theCatalog.WorksetByName(dn, wsn)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Model scenario not found:", mb.model.Name, dn, wsn), http.StatusBadRequest)
		return // empty result: workset not found
	}
	if !ws.IsReadonly {
		http.Error(w, helper.MsgL(lang, "Model scenario must be read-only:", mb.model.Name, dn, ws.Name), http.StatusBadRequest)
		return // empty result: workset must be read-only
	}

	// base part of: output directory name, .zip file name and log file name
	baseName := mb.model.Name + ".set." + helper.CleanFileName(ws.Name)
	omppLog.Log("Download of: ", baseName)

	// if download.progress.log file exist the retun error: download in progress
	logPath := filepath.Join(theCfg.downloadDir, baseName+".progress.download.log")
	if helper.IsFileExist(logPath) {
		omppLog.Log("Error: download already in progress:", logPath)
		http.Error(w, helper.MsgL(lang, "Model scenario download already in progress:", baseName), http.StatusBadRequest)
		return
	}

	// create new download.progress.log file and write model scenario decsription
	isLog := fileCreateEmpty(false, logPath)
	if !isLog {
		omppLog.Log("Failed to create download log file:", baseName+".progress.download.log")
		http.Error(w, helper.MsgL(lang, "Model scenario download failed:", baseName), http.StatusBadRequest)
		return
	}
	// do not translate hdrMsg strings below
	hdrMsg := []string{
		"------------------",
		"Model Name       : " + m.Name,
		"Model Version    : " + m.Version + " " + m.CreateDateTime,
		"Model Digest     : " + m.Digest,
		"Scenario Name    : " + ws.Name,
		"Scenario Version : " + ws.UpdateDateTime,
		"Folder           : " + baseName,
		"------------------",
	}
	if !writeToCmdLog(logPath, true, helper.MsgL(lang, "Download of:", baseName)) {
		renameToDownloadErrorLog(logPath, "", nil)
		omppLog.Log("Failed to write into download log file:", baseName+".progress.download.log")
		http.Error(w, helper.MsgL(lang, "Model scenario download failed:", baseName), http.StatusBadRequest)
		return
	}
	if !writeToCmdLog(logPath, false, hdrMsg...) {
		renameToDownloadErrorLog(logPath, "", nil)
		omppLog.Log("Failed to write into download log file:", baseName+".progress.download.log")
		http.Error(w, helper.MsgL(lang, "Model scenario download failed:", baseName), http.StatusBadRequest)
		return
	}

	// create model scenario download files on separate thread
	cmd, cmdMsg := makeWorksetDownloadCommand(mb, ws.Name, logPath, opts.Utf8BomIntoCsv, opts.IdCsv, lang)

	go makeDownload(baseName, cmd, cmdMsg, logPath)

	// report to the client results location
	w.Header().Set("Content-Location", "/api/download/model/"+dn+"/workset/"+wsn+"/"+baseName)
}
