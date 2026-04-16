// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// return .download.log file by name and download status.
//
//	GET /api/download/log/file/:name
//
// Download status is one of: progress ready error or "" if unknown
func fileLogDownloadGetHandler(w http.ResponseWriter, r *http.Request) {
	fileLogUpDownGet("download", theCfg.downloadDir, w, r)
}

// return .upload.log file by name and upload status.
//
//	GET /api/upload/log/file/:name
//
// Upload status is one of: progress ready error or "" if unknown
func fileLogUploadGetHandler(w http.ResponseWriter, r *http.Request) {
	fileLogUpDownGet("upload", theCfg.uploadDir, w, r)
}

// return .up-or-down.log file by name and status.
// Status is one of: progress ready error or "" if unknown
func fileLogUpDownGet(upDown string, upDownDir string, w http.ResponseWriter, r *http.Request) {

	// url or query parameters
	fileName := getRequestParam(r, "name")
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// validate name: it must be a file name only, it cannot be any directory
	if fileName == "" || !strings.HasSuffix(fileName, "."+upDown+".log") || fileName != filepath.Base(helper.CleanFileName(fileName)) {
		http.Error(w, helper.MsgL(lang, "Log file name invalid (or empty):", fileName), http.StatusBadRequest)
		return
	}

	// read file content
	filePath := filepath.Join(upDownDir, fileName)

	if !helper.IsFileExist(filePath) {
		http.Error(w, helper.MsgL(lang, "Log file not found:", fileName), http.StatusBadRequest)
		return
	}

	bt, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, helper.MsgL(lang, "Failed to read log file:", fileName), http.StatusBadRequest)
		return
	}
	fc := string(bt)

	// parse log file content to get folder name, log file kind and keys
	uds := parseUpDownLog(upDown, fileName, fc)
	updateStatUpDownLog(fileName, &uds, upDownDir)

	jsonResponse(w, r, uds) // return log file content and status
}

// return all .download.log files and download status.
//
//	GET /api/download/log-all
//
// Download status is one of: progress ready error or "" if unknown
func allLogDownloadGetHandler(w http.ResponseWriter, r *http.Request) {
	allLogUpDownGet("download", theCfg.downloadDir, w, r)
}

// return all .upload.log files and upload status.
//
//	GET /api/upload/log-all
//
// Upload status is one of: progress ready error or "" if unknown
func allLogUploadGetHandler(w http.ResponseWriter, r *http.Request) {
	allLogUpDownGet("upload", theCfg.uploadDir, w, r)
}

// return all .up-or-down.log files and status.
// Status is one of: progress ready error or "" if unknown
func allLogUpDownGet(upDown string, upDownDir string, w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// find all .up-or-down.log files
	fLst, err := os.ReadDir(upDownDir)
	if err != nil {
		http.Error(w, helper.MsgL(lang, "Error at reading log directory"), http.StatusBadRequest)
		return
	}

	// parse all files
	udsLst := parseUpDownLogFileList(upDown, "", fLst, upDownDir)

	jsonResponse(w, r, udsLst)
}

// return model .download.log files with download status.
//
//	GET /api/download/log/model/:model
//
// Download status is one of: progress ready error or "" if unknown
func modelLogDownloadGetHandler(w http.ResponseWriter, r *http.Request) {
	modelLogUpDownGet("download", theCfg.downloadDir, w, r)
}

// return model .upload.log files with upload status.
//
//	GET /api/upload/log/model/:model
//
// Upload status is one of: progress ready error or "" if unknown
func modelLogUploadGetHandler(w http.ResponseWriter, r *http.Request) {
	modelLogUpDownGet("upload", theCfg.uploadDir, w, r)
}

// return model .download.log or .upload.log files and status.
// Status is one of: progress ready error or "" if unknown
func modelLogUpDownGet(upDown string, upDownDir string, w http.ResponseWriter, r *http.Request) {

	// url or query parameters
	dn := getRequestParam(r, "model") // model digest-or-name
	mName := dn

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// if this model digest then try to find model name in catalog
	mb, ok := theCatalog.modelBasicByDigestOrName(dn)
	if ok {
		mName = mb.model.Name
	}

	// find all .download.log or .upload.log files
	fLst, err := os.ReadDir(upDownDir)
	if err != nil {
		http.Error(w, helper.MsgL(lang, "Error at reading log directory"), http.StatusBadRequest)
		return
	}

	// parse all files
	udsLst := parseUpDownLogFileList(upDown, mName+".", fLst, upDownDir)

	jsonResponse(w, r, udsLst)
}

// return file tree (file path, size, modification time) by folder name.
//
//	GET /api/download/file-tree/:folder
func fileTreeDownloadGetHandler(w http.ResponseWriter, r *http.Request) {
	doFileTreeGet(theCfg.downloadDir, false, "folder", false, w, r)
}

// return file tree (file path, size, modification time) by folder name.
//
//	GET /api/upload/file-tree/:folder
func fileTreeUploadGetHandler(w http.ResponseWriter, r *http.Request) {
	doFileTreeGet(theCfg.uploadDir, false, "folder", false, w, r)
}

// delete download files by folder name.
//
//	DELETE /api/download/delete/:folder
//
// Delete folder, .zip file and .download.log files
func downloadDeleteHandler(w http.ResponseWriter, r *http.Request) {
	upDownDelete("download", theCfg.downloadDir, false, w, r)

}

// starts deleting of download files by folder name.
//
//	DELETE /api/download/start/delete/:folder
//
// Delete started on separate thread and does delete of folder, .zip file and .download.log files
func downloadDeleteAsyncHandler(w http.ResponseWriter, r *http.Request) {
	upDownDelete("download", theCfg.downloadDir, true, w, r)
}

// delete upload files by folder name.
//
//	DELETE /api/upload/delete/:folder
//
// Delete folder, .zip file and .upload.log files
func uploadDeleteHandler(w http.ResponseWriter, r *http.Request) {
	upDownDelete("upload", theCfg.uploadDir, false, w, r)

}

// starts deleting of upload files by folder name.
//
//	DELETE /api/upload/start/delete/:folder
//
// Delete started on separate thread and does delete of folder, .zip file and .upload.log files
func uploadDeleteAsyncHandler(w http.ResponseWriter, r *http.Request) {
	upDownDelete("upload", theCfg.uploadDir, true, w, r)
}

// delete all download files for all models.
// DELETE /api/download/delete-all
// Delete all models deletes folder, .zip file and .download.log files
func downloadAllDeleteHandler(w http.ResponseWriter, r *http.Request) {
	upDownAllDelete("download", theCfg.downloadDir, false, w, r)
}

// start deleting all download files for all models.
//
//	DELETE /api/download/start/delete-all
//
// Delete started on separate thread and fo all models deletes folder, .zip file and .download.log files
func downloadAllDeleteAsyncHandler(w http.ResponseWriter, r *http.Request) {
	upDownAllDelete("download", theCfg.downloadDir, true, w, r)
}

// delete all upload files for all models.
// DELETE /api/upload/delete-all
// Delete all models folder, .zip file and .upload.log files
func uploadAllDeleteHandler(w http.ResponseWriter, r *http.Request) {
	upDownAllDelete("upload", theCfg.uploadDir, false, w, r)
}

// start deleting all upload files for all models.
//
//	DELETE /api/upload/start/delete-all
//
// Delete started on separate thread and for all models deletes folder, .zip file and .upload.log files
func uploadAllDeleteAsyncHandler(w http.ResponseWriter, r *http.Request) {
	upDownAllDelete("upload", theCfg.uploadDir, true, w, r)
}

// Delete all files and folders, on separate thread or blocking current thread
func upDownAllDelete(upDown string, upDownDir string, isAsync bool, w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	if !helper.IsDirExist(upDownDir) {
		http.Error(w, helper.MsgL(lang, "Folder not found:", upDownDir), http.StatusBadRequest)
		return
	}

	// list of files and folders
	pLst, err := filepath.Glob(filepath.Join(upDownDir, "*"))
	if err != nil {
		omppLog.Log("Error at get file list of:", upDownDir, ":", err)
		http.Error(w, helper.MsgL(lang, "Error at folder scan:", upDown), http.StatusBadRequest)
		return
	}

	// delete all files and folders, write progress to log
	doDeleteAll := func(upDown string, upDownDir string, nameLst []string) {

		slices.Sort(nameLst) // sort names: folders are shroter, files after folders

		// create new up-or-down.progress.log file and write delete header
		const logDelAll = "delete-all_"
		logName := logDelAll + helper.MakeTimeStamp(time.Now()) + ".progress." + upDown + ".log"
		logPath := filepath.Join(upDownDir, logName)

		isLog := fileCreateEmpty(false, logPath)
		if !isLog {
			omppLog.Log("Failed to create log file:", logName)
			return
		}
		if isLog {
			isLog = writeToCmdLog(logPath, true, "Start deleting all from: "+upDown+" [ "+strconv.Itoa(len(nameLst))+" ]")
		}
		nErr := 0

		for _, p := range nameLst {

			fi, e := os.Stat(p)
			if e != nil {
				continue // ignore directory entry where file removed after readdir()
			}
			name := fi.Name()

			if strings.HasPrefix(name, logDelAll) {
				continue // skip delete-all log files
			}

			// remove files and directories
			if isLog {
				isLog = writeToCmdLog(logPath, true, "delete: "+name)
			}
			if !fi.IsDir() {
				if e := os.Remove(p); e != nil && !os.IsNotExist(e) {
					if isLog {
						isLog = writeToCmdLog(logPath, true, "Error at delete "+name, e.Error())
					}
					nErr++
				}
			} else {
				if e := os.RemoveAll(p); e != nil && !os.IsNotExist(e) {
					if isLog {
						isLog = writeToCmdLog(logPath, true, "Error at delete "+name, e.Error())
					}
					nErr++
				}
			}
		}

		// last step: remove delete progress log file or rename it on errors
		if nErr == 0 {
			if e := os.Remove(logPath); e != nil && !os.IsNotExist(e) {
				omppLog.LogNoLT(e)
			}
		} else {
			omppLog.Log("Failed to delete from", upDown, ". Errors:", nErr)
			renameToUpDownErrorLog(upDown, logPath, helper.Msg("Errors:", nErr), nil)
		}

		// if disk usage scan active then refersh disk use now
		if theCfg.isDiskUse {
			refreshDiskScanC <- true
		}
	}

	// do delete all files
	nFound := len(pLst)

	if nFound > 0 {
		if isAsync {
			go doDeleteAll(upDown, upDownDir, pLst)
		} else {
			doDeleteAll(upDown, upDownDir, pLst)
		}
	}

	// report to the client results location
	w.Header().Set("Content-Location", "/api/"+upDown+"/delete-all/"+strconv.Itoa(nFound))
}

// upDownDelete delete download or upload files by folder name.
// Delete folder, .zip file and .up-or-down.log files
func upDownDelete(upDown string, upDownDir string, isAsync bool, w http.ResponseWriter, r *http.Request) {

	// url or query parameters
	folder := getRequestParam(r, "folder")
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// delete files
	err := doDeleteUpDown(upDown, upDownDir, isAsync, folder, lang)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// report to the client results location
	w.Header().Set("Content-Location", "/api/"+upDown+"/delete/"+folder)
}

// delete download or upload files by folder name.
// if isAsync is true then start delete on separate thread
func doDeleteUpDown(upDown string, upDownDir string, isAsync bool, folder, lang string) error {

	if folder == "" || folder != filepath.Base(helper.CleanFileName(folder)) {
		return helper.ErrorNewL(lang, "Folder name invalid (or empty):", folder)
	}
	omppLog.Log("Delete:", upDown, folder)

	// create new up-or-down.progress.log file and write delete header
	logPath := filepath.Join(upDownDir, folder+".progress."+upDown+".log")

	isLog := fileCreateEmpty(false, logPath)
	if !isLog {
		omppLog.Log("Failed to create log file:", folder+".progress."+upDown+".log")
		return helper.ErrorNewL(lang, "Delete failed:", folder)
	}
	// do not translate hdrMsg strings below
	hdrMsg := []string{
		"---------------",
		"Delete        : " + folder,
		"Folder        : " + folder,
		"---------------",
	}
	if !writeToCmdLog(logPath, true, "Start deleting: "+folder) {
		renameToUpDownErrorLog(upDown, logPath, "", nil)
		omppLog.Log("Failed to write into log file:", folder+".progress."+upDown+".log")
		return helper.ErrorNewL(lang, "Delete failed:", folder)
	}
	if !writeToCmdLog(logPath, false, hdrMsg...) {
		renameToUpDownErrorLog(upDown, logPath, "", nil)
		omppLog.Log("Failed to write into log file:", folder+".progress."+upDown+".log")
		return helper.ErrorNewL(lang, "Delete failed:", folder)
	}

	// delete files on separate thread
	doDelete := func(baseName, logPath string) {

		// remove files
		basePath := filepath.Join(upDownDir, baseName)

		if !removeUpDownFile(upDown, basePath+".ready."+upDown+".log", logPath, baseName+".ready."+upDown+".log") {
			return
		}
		if !removeUpDownFile(upDown, basePath+".error."+upDown+".log", logPath, baseName+".error."+upDown+".log") {
			return
		}
		if !removeUpDownFile(upDown, basePath+".zip", logPath, baseName+".zip") {
			return
		}
		if !removeUpDownDir(upDown, basePath, logPath, baseName) {
			return
		}
		// last step: remove delete progress log file
		if e := os.Remove(basePath + ".progress." + upDown + ".log"); e != nil && !os.IsNotExist(e) {
			omppLog.LogNoLT(e)
		}
	}

	if isAsync {
		go doDelete(folder, logPath)
	} else {
		doDelete(folder, logPath)
	}
	return nil
}
