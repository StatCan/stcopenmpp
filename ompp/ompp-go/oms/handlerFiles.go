// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// upload file to user files folder, unzip if file.zip uploaded.
//
//	POST /api/files/file/:path
//	POST /api/files/file?path=....
func filesFileUploadPostHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	p, src, err := getFilesPathParam(r, "path")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// check if folder exists
	saveToPath := filepath.Join(theCfg.filesDir, p)
	dir, fName := filepath.Split(saveToPath)

	ok, err := helper.IsDir(dir)
	if err != nil {
		omppLog.Log("Error: invalid (or empty) directory:", dir, ":", src, err)
		http.Error(w, helper.MsgL(lang, "Invalid (or empty) file path:", src), http.StatusBadRequest)
		return
	}
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid (or empty) folder path:", src), http.StatusBadRequest)
		return
	}

	// parse multipart form: only single part expected with file.name file attached
	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, helper.MsgL(lang, "Error at multipart form open"), http.StatusBadRequest)
		return
	}

	// open next part
	part, err := mr.NextPart()
	if err == io.EOF {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) next part of multipart form"), http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, helper.MsgL(lang, "Failed to get next part of multipart form:", err), http.StatusBadRequest)
		return
	}
	defer part.Close()

	// check file name: it should be the same as url parameter
	fn := part.FileName()
	if fn != fName {
		http.Error(w, helper.MsgL(lang, "Error: invalid (or empty) file name:", fName, "("+fn+")"), http.StatusBadRequest)
		return
	}

	// save file into user files
	omppLog.Log("Upload of:", fName)

	err = helper.SaveTo(saveToPath, part)
	if err != nil {
		omppLog.Log("Error: unable to write into", saveToPath, err)
		http.Error(w, helper.MsgL(lang, "Error: unable to write into", fName), http.StatusInternalServerError)
		return
	}

	// unzip if source file is .zip archive
	ext := filepath.Ext(fName)
	if strings.ToLower(ext) == ".zip" {

		if err = helper.UnpackZip(saveToPath, false, strings.TrimSuffix(saveToPath, ext)); err != nil {
			omppLog.Log("Error: unable to unzip", saveToPath, err)
			http.Error(w, helper.MsgL(lang, "Error: unable to unzip", src), http.StatusInternalServerError)
			return
		}
	}

	// report to the client results location
	w.Header().Set("Content-Location", "/api/files/file/"+src)
}

// return file tree (file path, size, modification time) in user files by ext which is comma separted list of extensions, underscore _ or * extension means any.
//
//	GET /api/files/file-tree/:ext/path/:path
//	GET /api/files/file-tree/:ext/path/
//	GET /api/files/file-tree/:ext/path
//	GET /api/files/file-tree/:ext/path?path=....
func filesTreeGetHandler(w http.ResponseWriter, r *http.Request) {
	doFileTreeGet(theCfg.filesDir, true, "path", true, w, r)
}

// return file tree (file path, size, modification time) by sub-folder name.
//
//	GET /api/download/file-tree/:folder
//	GET /api/upload/file-tree/:folder
//	GET /api/files/file-tree/:ext/path/
//	GET /api/files/file-tree/:ext/path/:path
//
// pathParam is a name of path parameter: "path" or "folder"
// if isAllowEmptyFolder is true then value of path parameter can be empty
// if isExt is true then request should have "ext" extnsion filter parameter
func doFileTreeGet(rootDir string, isAllowEmptyPath bool, pathParam string, isExt bool, w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// optional url or query parameters: sub-folder path and files extension
	src := getRequestParam(r, pathParam)
	if pathParam == "" || src == "" && !isAllowEmptyPath || src == "." || src == ".." {
		http.Error(w, helper.MsgL(lang, "Folder name invalid (or empty):", src), http.StatusBadRequest)
		return
	}
	folder := src

	extCsv := ""
	if isExt {
		extCsv = getRequestParam(r, "ext")
		if extCsv == "" || extCsv == "." || extCsv == ".." {
			http.Error(w, helper.MsgL(lang, "Files extension invalid (or empty):", extCsv), http.StatusBadRequest)
			return
		}
		if extCsv == "_" || extCsv == "*" { // _ extension means * any extension
			extCsv = ""
		}
		if strings.ContainsAny(extCsv, helper.InvalidFileNameChars) {
			http.Error(w, helper.MsgL(lang, "Files extension invalid (or empty):", extCsv), http.StatusBadRequest)
			return
		}
	}

	// if folder path specified then it must be local path, not absolute and should not contain invalid characters
	if folder != "" {

		folder = filepath.Clean(folder)
		if folder == "." || folder == ".." || !filepath.IsLocal(folder) {
			http.Error(w, helper.MsgL(lang, "Folder name invalid (or empty):", src), http.StatusBadRequest)
			return
		}

		// folder path should not contain invalid characters
		if strings.ContainsAny(folder, helper.InvalidFilePathChars) {
			http.Error(w, helper.MsgL(lang, "Folder name invalid (or empty):", src), http.StatusBadRequest)
			return
		}
	}

	// get files tree
	treeLst, err := filesWalk(rootDir, folder, extCsv, true, lang)
	if err != nil {
		omppLog.Log("Error:", err)
		http.Error(w, helper.MsgL(lang, "Error at folder scan:", folder), http.StatusBadRequest)
		return
	}

	jsonResponse(w, r, treeLst)
}

// create folder under user files directory.
//
//	PUT /api/files/folder/:path
func filesFolderCreatePutHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	_, folder, err := getFilesPathParam(r, "path")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// create folder(s) path under user files root
	folderPath := filepath.Join(theCfg.filesDir, folder)

	if err := os.MkdirAll(folderPath, 0750); err != nil {
		omppLog.Log("Error at creating folder:", folderPath, err)
		http.Error(w, helper.MsgL(lang, "Error at creating folder:", folder), http.StatusInsufficientStorage)
		return
	}

	// report to the client results location
	w.Header().Set("Content-Location", "/api/files/folder/"+folder)
}

// delete file or folder from user files directory.
//
//	DELETE /api/files/delete/:path
func filesDeleteHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	_, folder, err := getFilesPathParam(r, "path")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// check: path to be deleted should not be download or upload foleder
	folderPath := filepath.Join(theCfg.filesDir, folder)

	if isReservedFilesPath(folderPath) {
		omppLog.Log("Error: unable to delete:", folderPath)
		http.Error(w, helper.MsgL(lang, "Error: unable to delete:", folder), http.StatusBadRequest)
		return
	}
	// remove folder(s)
	if err := os.RemoveAll(folderPath); err != nil {
		omppLog.Log("Error at deleting folder:", folderPath, err)
		http.Error(w, helper.MsgL(lang, "Error at deleting folder:", folder), http.StatusInsufficientStorage)
		return
	}

	// report to the client deleted location
	w.Header().Set("Content-Location", "/api/files/folder/"+folder)
}

// delete all user files and folders, keep reserved folders and it content: download and upload.
//
//	DELETE /api/files/delete-all
func filesAllDeleteHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// get list of files under user files root
	pLst, err := filepath.Glob(theCfg.filesDir + "/*")
	if err != nil {
		omppLog.Log("Error at user files directory scan:", theCfg.filesDir+"/*", err)
		http.Error(w, helper.MsgL(lang, "Error at user files directory scan"), http.StatusBadRequest)
		return
	}

	// delete files and remove sub-folders except of protected sub-folders: download, upload
	for k := 0; k < len(pLst); k++ {

		if isReservedFilesPath(pLst[k]) {
			continue
		}
		if err = os.RemoveAll(pLst[k]); err != nil {
			omppLog.Log("Error: unable to delete:", pLst[k])
			http.Error(w, helper.MsgL(lang, "Error: unable to delete:", pLst[k]), http.StatusBadRequest)
			return
		}
	}
}

// return true if path is one of "reserved" paths and cannot be deleted: . .. download upload home, etc.
func isReservedFilesPath(path string) bool {
	return path == "." || path == ".." ||
		path == theCfg.downloadDir || path == theCfg.uploadDir ||
		path == theCfg.inOutDir || path == theCfg.filesDir ||
		path == theCfg.homeDir || path == theCfg.rootDir ||
		path == theCfg.jobDir || path == theCfg.docDir ||
		path == theCfg.htmlDir || path == theCfg.etcDir
}

// get and validate path parameter from url or query parameter, return error if it is empty or . or .. or invalid
func getFilesPathParam(r *http.Request, name string) (string, string, error) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// url or query parameter folder required
	src := getRequestParam(r, name)
	if src == "" || src == "." || src == ".." {
		return "", src, helper.ErrorNewL(lang, "Folder name invalid (or empty):", src)
	}

	// folder path should be local, not absolute and shoud not contain invalid characters
	folder := filepath.Clean(src)
	if folder == "." || folder == ".." || !filepath.IsLocal(folder) {
		return "", src, helper.ErrorNewL(lang, "Folder name invalid (or empty):", src)
	}
	if strings.ContainsAny(folder, helper.InvalidFilePathChars) {
		return "", src, helper.ErrorNewL(lang, "Folder name invalid (or empty):", src)
	}

	return folder, src, nil
}
