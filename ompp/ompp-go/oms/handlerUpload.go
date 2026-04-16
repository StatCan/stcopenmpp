// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// post of model run zip archive in home/io/upload folder.
//
// POST /api/upload/model/:model/run
// POST /api/upload/model/:model/run/:run
// POST /api/upload/model/:model/run/lang/:lang
//
// Zip archive is the same as created by dbcopy command line utilty.
// Dimension(s) and enum-based parameters returned as enum codes, not enum id's.
func runUploadPostHandler(w http.ResponseWriter, r *http.Request) {
	// url or query parameters
	dn := getRequestParam(r, "model")      // model digest-or-name
	rName := getRequestParam(r, "run")     // run name
	lang := preferedRequestLang(r, "lang") // get prefered language for dbcopy log messages

	// block upload if disk space usage exceed the limits
	if isOver, _ := theRunCatalog.getDiskUseStatus(); isOver {
		http.Error(w, helper.LTL(lang, "Disk space usage exceeds quota, upload disabled"), http.StatusBadRequest)
		return
	}

	// find model metadata by digest or name
	mb, ok := theCatalog.modelBasicByDigestOrName(dn)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Model not found:", dn), http.StatusBadRequest)
		return // empty result: model digest not found
	}

	// parse multipart form: only single part expected with run.zip file attached
	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, helper.LTL(lang, "Error at multipart form open"), http.StatusBadRequest)
		return
	}

	// open next part
	part, err := mr.NextPart()
	if err == io.EOF {
		http.Error(w, helper.LTL(lang, "Invalid (empty) next part of multipart form"), http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, helper.MsgL(lang, "Failed to get next part of multipart form:", err), http.StatusBadRequest)
		return
	}
	defer part.Close()

	// check file name: it should be modelName.run.RunName.zip
	// if run name not specified in URL the get it from file name
	fName := part.FileName()
	ext := path.Ext(fName)
	baseName := strings.TrimSuffix(fName, ext)
	mpn := mb.model.Name + ".run."
	runName := strings.TrimPrefix(baseName, mpn)

	if baseName == "" || baseName == "." || baseName == ".." ||
		runName == "" || runName == "." || runName == ".." ||
		fName != helper.CleanFileName(fName) {
		http.Error(w, helper.MsgL(lang, "Error: invalid (or empty) file name:", fName), http.StatusBadRequest)
		return
	}
	if ext != ".zip" || !strings.HasPrefix(baseName, mpn) {
		http.Error(w, helper.MsgL(lang, "Error: file name must be:", mpn+"Name.zip"), http.StatusBadRequest)
		return
	}
	if rName != "" && runName != rName {
		http.Error(w, helper.MsgL(lang, "Error: invalid file name, expected:", mpn+rName+".zip"), http.StatusBadRequest)
		return
	}

	// if upload.progress.log file exist the retun error: upload in progress
	omppLog.Log("Upload of:", fName)

	logPath := filepath.Join(theCfg.uploadDir, baseName+".progress.upload.log")
	if helper.IsFileExist(logPath) {
		omppLog.Log("Error: upload already in progress:", logPath)
		http.Error(w, helper.MsgL(lang, "Model run upload already in progress:", baseName), http.StatusBadRequest)
		return
	}

	// create new upload.progress.log file and write model run decsription
	isLog := fileCreateEmpty(false, logPath)
	if !isLog {
		omppLog.Log("Failed to create upload log file:", baseName+".progress.upload.log")
		http.Error(w, helper.MsgL(lang, "Model run upload failed:", baseName), http.StatusBadRequest)
		return
	}
	// do not translate hdrMsg strings below
	hdrMsg := []string{
		"------------------",
		"Upload           : " + fName,
		"Model Name       : " + mb.model.Name,
		"Model Version    : " + mb.model.Version + " " + mb.model.CreateDateTime,
		"Model Digest     : " + mb.model.Digest,
		"Run Name         : " + runName,
		"Folder           : " + baseName,
		"------------------",
	}
	if !writeToCmdLog(logPath, true, helper.MsgL(lang, "Upload of:", baseName)) {
		renameToUploadErrorLog(logPath, "", nil)
		omppLog.Log("Failed to write into upload log file:", baseName+".progress.upload.log")
		http.Error(w, helper.MsgL(lang, "Model run upload failed:", baseName), http.StatusBadRequest)
		return
	}
	if !writeToCmdLog(logPath, false, hdrMsg...) {
		renameToUploadErrorLog(logPath, "", nil)
		omppLog.Log("Failed to write into upload log file:", baseName+".progress.upload.log")
		http.Error(w, helper.MsgL(lang, "Model run upload failed:", baseName), http.StatusBadRequest)
		return
	}

	// save run.zip into upload directory
	saveToPath := filepath.Join(theCfg.uploadDir, fName)

	err = helper.SaveTo(saveToPath, part)
	if err != nil {
		omppLog.Log("Error: unable to write into", saveToPath, err)
		http.Error(w, helper.MsgL(lang, "Error: unable to write into", fName), http.StatusInternalServerError)
		return
	}

	// create model run upload files on separate thread
	cmd, cmdMsg := makeRunUploadCommand(mb, runName, logPath, lang)

	go makeUpload(baseName, cmd, cmdMsg, logPath)

	// report to the client results location
	w.Header().Set("Content-Location", "/api/upload/model/"+dn+"/run/"+runName+"/"+baseName)
}

// post of model workset zip archive in home/io/upload folder.
//
// POST /api/upload/model/:model/workset
// POST /api/upload/model/:model/workset/:set
// POST /api/upload/model/:model/workset/lang/:lang
//
// Zip archive is the same as created by dbcopy command line utilty.
// Dimension(s) and enum-based parameters returned as enum codes, not enum id's.
// Posted multi-part form can have optional "workset-upload-options" part with json upload options
// Upload option NoDigestCheck=true do suppress model digest verification:
// model digest in source zip is ignored, only model name is used and that allows to upload worksets into different model version.
func worksetUploadPostHandler(w http.ResponseWriter, r *http.Request) {
	// url or query parameters
	dn := getRequestParam(r, "model")      // model digest-or-name
	wsn := getRequestParam(r, "set")       // workset name
	lang := preferedRequestLang(r, "lang") // get prefered language for dbcopy log messages

	// block upload if disk space usage exceed the limits
	if isOver, _ := theRunCatalog.getDiskUseStatus(); isOver {
		http.Error(w, helper.LTL(lang, "Disk space usage exceeds quota, upload disabled"), http.StatusBadRequest)
		return
	}

	// find model metadata by digest or name
	mb, ok := theCatalog.modelBasicByDigestOrName(dn)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Model not found:", dn), http.StatusBadRequest)
		return // empty result: model digest not found
	}

	// parse multipart form: only single part expected with set.zip file attached
	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, helper.LTL(lang, "Error at multipart form open"), http.StatusBadRequest)
		return
	}

	// open first part: it can be workset-upload-options part or workset.zip file
	part, err := mr.NextPart()
	if err == io.EOF {
		http.Error(w, helper.LTL(lang, "Invalid (empty) next part of multipart form"), http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, helper.MsgL(lang, "Failed to get next part of multipart form:", err), http.StatusBadRequest)
		return
	}

	// if this is workset-upload-options then decode json
	isNoDigestCheck := false
	if part.FormName() == "workset-upload-options" {

		opts := struct{ NoDigestCheck bool }{}

		err = json.NewDecoder(part).Decode(&opts)
		if err != nil && err != io.EOF {
			http.Error(w, helper.LTL(lang, "Json decode error at 'workset-upload-options' part of multipart form"), http.StatusBadRequest)
			part.Close()
			return
		}
		isNoDigestCheck = opts.NoDigestCheck

		// open next part: workset.zip file
		part.Close()

		part, err = mr.NextPart()
		if err == io.EOF {
			http.Error(w, helper.LTL(lang, "Invalid (empty) next part of multipart form"), http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, helper.MsgL(lang, "Failed to get next part of multipart form:", err), http.StatusBadRequest)
			return
		}
	}
	defer part.Close()

	// check file name: it should be modelName.set.WorksetName.zip
	// if workset name not specified in URL the get it from file name
	fName := part.FileName()
	ext := path.Ext(fName)
	baseName := strings.TrimSuffix(fName, ext)
	mpn := mb.model.Name + ".set."
	setName := strings.TrimPrefix(baseName, mpn)

	if baseName == "" || baseName == "." || baseName == ".." ||
		setName == "" || setName == "." || setName == ".." ||
		fName != helper.CleanFileName(fName) {
		http.Error(w, helper.MsgL(lang, "Error: invalid (or empty) file name:", fName), http.StatusBadRequest)
		return
	}
	if ext != ".zip" || !strings.HasPrefix(baseName, mpn) {
		http.Error(w, helper.MsgL(lang, "Error: file name must be:", mpn+"Name.zip"), http.StatusBadRequest)
		return
	}
	if wsn != "" && setName != wsn {
		http.Error(w, helper.MsgL(lang, "Error: invalid file name, expected:", mpn+wsn+".zip"), http.StatusBadRequest)
		return
	}

	// if upload.progress.log file exist the retun error: upload in progress
	omppLog.Log("Upload of:", fName)

	logPath := filepath.Join(theCfg.uploadDir, baseName+".progress.upload.log")
	if helper.IsFileExist(logPath) {
		omppLog.Log("Error: upload already in progress:", logPath)
		http.Error(w, helper.MsgL(lang, "Model scenario upload already in progress:", baseName), http.StatusBadRequest)
		return
	}

	// create new upload.progress.log file and write model scenario decsription
	isLog := fileCreateEmpty(false, logPath)
	if !isLog {
		omppLog.Log("Failed to create upload log file:", baseName+".progress.upload.log")
		http.Error(w, helper.MsgL(lang, "Model scenario upload failed:", baseName), http.StatusBadRequest)
		return
	}
	// do not translate hdrMsg strings below
	hdrMsg := []string{
		"------------------",
		"Upload           : " + fName,
		"Model Name       : " + mb.model.Name,
		"Model Version    : " + mb.model.Version + " " + mb.model.CreateDateTime,
		"Model Digest     : " + mb.model.Digest,
		"Scenario Name    : " + setName,
		"Folder           : " + baseName,
		"------------------",
	}
	if !writeToCmdLog(logPath, true, helper.MsgL(lang, "Upload of:", baseName)) {
		renameToUploadErrorLog(logPath, "", nil)
		omppLog.Log("Failed to write into upload log file:", baseName+".progress.upload.log")
		http.Error(w, helper.MsgL(lang, "Model scenario upload failed:", baseName), http.StatusBadRequest)
		return
	}
	if !writeToCmdLog(logPath, false, hdrMsg...) {
		renameToUploadErrorLog(logPath, "", nil)
		omppLog.Log("Failed to write into upload log file:", baseName+".progress.upload.log")
		http.Error(w, helper.MsgL(lang, "Model scenario upload failed:", baseName), http.StatusBadRequest)
		return
	}

	// save set.zip into upload directory
	saveToPath := filepath.Join(theCfg.uploadDir, fName)

	helper.SaveTo(saveToPath, part)
	if err != nil {
		omppLog.Log("Error: unable to write into", saveToPath, err)
		http.Error(w, helper.MsgL(lang, "Error: unable to write into", fName), http.StatusInternalServerError)
		return
	}

	// create model scenario upload files on separate thread
	cmd, cmdMsg := makeWorksetUploadCommand(mb, setName, logPath, isNoDigestCheck, lang)

	go makeUpload(baseName, cmd, cmdMsg, logPath)

	// report to the client results location
	w.Header().Set("Content-Location", "/api/upload/model/"+dn+"/workset/"+setName+"/"+baseName)
}
