// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"bufio"
	"cmp"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/statcan/stcopenmpp/ompp/ompp-go/ompp/helper"
	"github.com/statcan/stcopenmpp/ompp/ompp-go/ompp/omppLog"
)

// reload models catalog: rescan models directory tree and reload model.sqlite.
//
//	POST /api/admin/all-models/refresh
func allModelsRefreshHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// model directory required to build list of model sqlite files
	modelLogDir, _ := theCatalog.getModelLogDir()
	modelDir, _ := theCatalog.getModelDir()
	if modelDir == "" {
		omppLog.Log("Failed to refresh models catalog: path to model directory cannot be empty")
		http.Error(w, helper.MsgL(lang, "Failed to refresh models catalog: path to model directory cannot be empty"), http.StatusBadRequest)
		return
	}
	omppLog.Log("Model directory:", modelDir)

	// refresh models catalog
	if _, err := theCatalog.refreshSqlite(modelDir, modelLogDir); err != nil {
		omppLog.LogNoLT(err)
		http.Error(w, helper.MsgL(lang, "Failed to refersh models catalog:", modelDir), http.StatusBadRequest)
		return
	}

	// refresh run state catalog
	if err := theRunCatalog.refreshCatalog(theCfg.etcDir, nil); err != nil {
		omppLog.LogNoLT(err)
		http.Error(w, helper.MsgL(lang, "Failed to refersh model runs catalog"), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Location", "/api/admin/all-models/refresh/"+filepath.ToSlash(modelDir))
	w.Header().Set("Content-Type", "text/plain")
}

// clean models catalog: close all model.sqlite connections and clean models catalog
//
//	POST /api/admin/all-models/close
func allModelsCloseHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// close models catalog
	modelDir, _ := theCatalog.getModelDir()

	if err := theCatalog.closeAll(); err != nil {
		omppLog.LogNoLT(err)
		http.Error(w, helper.MsgL(lang, "Failed to close models catalog:", modelDir), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Location", "/api/admin/all-models/close/"+filepath.ToSlash(modelDir))
	w.Header().Set("Content-Type", "text/plain")
}

// close model.sqlite connection and clean model from catalog
//
//	POST /api/admin/model/:model/close
//
// Model identified by digest-or-name.
// If multiple models with same name exist then result is undefined.
func modelCloseHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	lang := preferedRequestLang(r, "") // get prefered language for messages

	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		http.Error(w, helper.MsgL(lang, "Invalid (empty) model digest and name"), http.StatusBadRequest)
		return
	}

	// close model and remove from catalog
	if _, _, _, err := theCatalog.closeModel(dn); err != nil {
		omppLog.LogNoLT(err)
		http.Error(w, helper.MsgL(lang, "Failed to close model", ": ", dn), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Location", "/api/admin/model/"+dn+"/close")
	w.Header().Set("Content-Type", "text/plain")
}

// delete all model files from disk
//
//	POST /api/admin/model/:model/delete
//
// Model identified by digest-or-name.
// If multiple models with same name exist then result is undefined.
func modelDeleteHandler(w http.ResponseWriter, r *http.Request) {

	dn := getRequestParam(r, "model")
	lang := preferedRequestLang(r, "") // get prefered language for messages

	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		http.Error(w, helper.MsgL(lang, "Invalid (empty) model digest and name"), http.StatusBadRequest)
		return
	}

	// close model and delete all model files from disk
	if err := theCatalog.deleteModel(dn); err != nil {
		omppLog.LogNoLT(err)
		http.Error(w, helper.MsgL(lang, "Failed to delete model:", dn), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Location", "/api/admin/model/"+dn+"/delete")
	w.Header().Set("Content-Type", "text/plain")
}

// open SQLite db file and get all models from it.
//
//	POST /api/admin/db-file-open/:path
//
// Path to model database must be relative to models/bin root.
// Slashes / or back \ slashes in the path must be replaced with * star.
// If model(s) with the same digest already open then method return an error.
func modelOpenDbFileHandler(w http.ResponseWriter, r *http.Request) {

	dbPath := getRequestParam(r, "path")
	lang := preferedRequestLang(r, "") // get prefered language for messages

	if dbPath == "" {
		omppLog.Log("Error: invalid (empty) path to model database file")
		http.Error(w, helper.MsgL(lang, "Invalid (empty) path to model database file"), http.StatusBadRequest)
		return
	}
	dbPath = strings.ReplaceAll(dbPath, "*", "/") // restore slashed / path

	// make db path relative to models/bin root
	// and check if model database file is already open: it should not be in the list of model db files
	mbinDir, _ := theCatalog.getModelDir()
	srcPath := path.Join(mbinDir, dbPath)

	mbs := theCatalog.allModels()
	if slices.IndexFunc(mbs, func(mb modelBasic) bool { return mb.relPath == srcPath }) >= 0 {
		http.Error(w, helper.MsgL(lang, "Error: model database file already open:", dbPath), http.StatusBadRequest)
		return
	}

	// open db file and add models to catalog
	n, err := theCatalog.loadModelDbFile(srcPath)
	if err != nil {
		omppLog.LogNoLT(err)
		http.Error(w, helper.MsgL(lang, "Failed to open model db file:", dbPath), http.StatusBadRequest)
		return
	}
	if n <= 0 {
		omppLog.LogNoLT(err)
		http.Error(w, helper.MsgL(lang, "Error: invalid (empty) model db file:", dbPath), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Location", "/api/admin/db-file-open/"+filepath.ToSlash(dbPath))
	w.Header().Set("Content-Type", "text/plain")
}

// pause or resume jobs queue processing by this oms instance
//
//	POST /api/admin/jobs-pause/:pause
func jobsPauseHandler(w http.ResponseWriter, r *http.Request) {
	doJobsPause(jobQueuePausedPath(theCfg.omsName), "/api/admin/jobs-pause/", w, r)
}

// Pause or resume jobs queue processing by this oms instance all by all oms instances
//
//	POST /api/admin/jobs-pause/:pause
//	POST /api/admin-all/jobs-pause/:pause
func doJobsPause(filePath, msgPath string, w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// url or query parameters: pause or resume boolean flag
	sp := getRequestParam(r, "pause")
	isPause, err := strconv.ParseBool(sp)
	if sp == "" || err != nil {
		http.Error(w, helper.MsgL(lang, "Invalid (or empty) jobs pause flag, expected true or false"), http.StatusBadRequest)
		return
	}

	// create jobs paused state file or remove it to resume queue processing
	isOk := false
	if isPause {
		isOk = fileCreateEmpty(false, filePath)
	} else {
		isOk = fileDeleteAndLog(false, filePath)
	}
	if !isOk {
		isPause = !isPause // operation failed
	}

	// Content-Location: /api/admin/jobs-pause/true
	w.Header().Set("Content-Location", msgPath+strconv.FormatBool(isPause))
	w.Header().Set("Content-Type", "text/plain")
}

// async start of model database cleanup and retrun LogFileName on success
//
//	POST /api/admin/db-cleanup/:path
//	POST /api/admin/db-cleanup/:path/name/:name
//	POST /api/admin/db-cleanup/:path/name/:name/digest/:digest
//	POST /api/admin/db-cleanup/:path/lang/:lang
//
// Relative path to model database file is required, slash / in the path must be replaced with * star.
// Model name and digest are optional parameters.
// Cleanup is done on separate thread by db cleanup script, defined in disk.ini [Common] DbCleanup.
// Model database must be closed, for example by: POST /api/admin/model/:model/close.
func modelDbCleanupHandler(w http.ResponseWriter, r *http.Request) {

	// if disk space use control disabled then do nothing
	if !theCfg.isDiskUse {
		w.Header().Set("Content-Location", "/api/admin/db-cleanup/none")
		w.Header().Set("Content-Type", "text/plain")
	}

	// validate parameters: path to database file is required
	dbPath := getRequestParam(r, "path")
	name := getRequestParam(r, "name")
	digest := getRequestParam(r, "digest")
	lang := preferedRequestLang(r, "lang") // get prefered language for dbcopy log messages

	if dbPath == "" {
		omppLog.Log("Error: invalid (empty) path to model database file")
		http.Error(w, helper.MsgL(lang, "Invalid (empty) path to model database file"), http.StatusBadRequest)
		return
	}
	dbPath = strings.ReplaceAll(dbPath, "*", "/") // restore slashed / path

	// check if database cleanup script defined
	// check if database file is exists and belong to current oms instance: it must be in the list of instance database files
	diskUse, dbUse := theRunCatalog.getDiskUse()

	if diskUse.dbCleanupCmd == "" {
		omppLog.Log("Error: db cleanup script is not defined in disk.ini")
		http.Error(w, helper.MsgL(lang, "Error: db cleanup script is not defined in disk.ini"), http.StatusInternalServerError)
		return
	}
	if i := slices.IndexFunc(
		dbUse, func(du dbDiskUse) bool { return du.DbPath == dbPath }); i < 0 || i >= len(dbUse) {
		http.Error(w, helper.MsgL(lang, "Error: model database not found", name, digest), http.StatusBadRequest)
		return
	}

	// check if model database is closed: it should not be in the list of model db files
	mbs := theCatalog.allModels()

	if i := slices.IndexFunc(mbs, func(mb modelBasic) bool { return mb.relPath == dbPath }); i >= 0 && i < len(mbs) {
		http.Error(w, helper.MsgL(lang, "Error: model database must be closed", name, digest), http.StatusBadRequest)
		return
	}

	// join db path with models/bin root
	srcPath := dbPath
	if mr, isOk := theCatalog.getModelDir(); isOk {
		srcPath = filepath.Join(mr, dbPath)
	}
	srcPath = filepath.Clean(srcPath)

	// make log file name and path
	ln := filepath.Base(dbPath)
	if ln == "." || ln == "/" || ln == "\\" {
		ln = "no-name"
	}
	ld, _ := theCatalog.getModelLogDir()

	cmdLog := newCmdLog("db-cleanup", ln, ld) // create new batch process log file

	// start database cleanup
	go func(cmdPath, mDbPath, mName, mDigest, msgLang string) {

		// make db cleanup command
		if mName == "" && (mDigest != "" || msgLang != "") {
			mName = "no-name"
		}
		if mDigest == "" && msgLang != "" {
			mDigest = "no-digest"
		}
		cArgs := []string{
			mDbPath,
			mName,
			mDigest,
		}
		if msgLang != "" {
			cArgs = append(cArgs, msgLang)
		}
		cmd := exec.Command(cmdPath, cArgs...)

		// connect console output to output log file
		outPipe, err := cmd.StdoutPipe()
		if err != nil {
			omppLog.Log("Error at join to stdout log", ": ", cmdLog.logPath, ": ", err)
			return
		}
		errPipe, err := cmd.StderrPipe()
		if err != nil {
			omppLog.Log("Error at join to stderr log", ": ", cmdLog.logPath, ": ", err)
			return
		}
		outDoneC := make(chan bool, 1)
		errDoneC := make(chan bool, 1)
		logTck := time.NewTicker(logTickTimeout * time.Millisecond)

		// start console output listners
		doLog := func(r io.Reader, done chan<- bool) {
			sc := bufio.NewScanner(r)
			for sc.Scan() {
				cmdLog.toLog(false, sc.Text())
			}
			done <- true
			close(done)
		}
		go doLog(outPipe, outDoneC)
		go doLog(errPipe, errDoneC)

		// start db cleanup
		omppLog.Log(strings.Join(cmd.Args, " "))
		cmdLog.toLog(true, strings.Join(cmd.Args, " "))

		err = cmd.Start()
		if err != nil {
			omppLog.Log("Error at", ": ", cmdLog.logPath, ": ", err)
			cmdLog.toLogError(true, err.Error())
			return
		}
		// else db cleanup started: wait until completed

		// wait until stdout and stderr closed
		for outDoneC != nil || errDoneC != nil {
			select {
			case _, ok := <-outDoneC:
				if !ok {
					outDoneC = nil
				}
			case _, ok := <-errDoneC:
				if !ok {
					errDoneC = nil
				}
			case <-logTck.C:
			}
		}

		// wait for db cleanup to be completed
		e := cmd.Wait()
		if e != nil {
			omppLog.Log("Error at: ", cmd.Args)
			cmdLog.toLogError(true, e.Error())
			return
		}
		// else:
		// completed OK
		cmdLog.toLog(true, "Done.")
		if !cmdLog.isLogOk {
			omppLog.Log("Warning: db cleanup log output may be incomplete")
		}

		// refresh disk usage
		refreshDiskScanC <- true

	}(diskUse.dbCleanupCmd, srcPath, name, digest, lang)

	// db cleanup is starting now: return path to log file
	jsonResponse(w, r, struct {
		LogFileName string
		IsError     bool
	}{
		LogFileName: cmdLog.logName,
		IsError:     cmdLog.isCmdErr || cmdLogIsErrorName(cmdLog.logPath),
	})
}

// copy model files from models library, using url parameter: path to model.publish.lst
//
//	POST /api/admin/copy-model/:path
//	POST /api/admin/copy-model/:path/lang/:lang
//
// If model with same digest already exist in current oms then copy may fail and result is undefined.
// This method does not support non-default documentaion path, it is expected to be modesl/doc
func copyModelPathHandler(w http.ResponseWriter, r *http.Request) {

	// get request parameters and check if copy model enabled
	pubLst := getRequestParam(r, "path")
	pubLst = strings.ReplaceAll(pubLst, "*", "/") // restore slashed / path

	lang := preferedRequestLang(r, "lang") // get prefered language for log messages

	mcf, mlb := theCatalog.toPublicConfig()
	if !mlb.IsCopy {
		http.Error(w, helper.MsgL(lang, "Copy model disabled"), http.StatusBadRequest)
		return
	}

	// check if model.publish.lst exists in model source directory
	srcPubLst := filepath.Join(mlb.srcRoot, mcf.ModelDir, pubLst)
	if pubLst == "" || !helper.IsFileExist(srcPubLst) {
		omppLog.Log("Error: invalid (empty) path to model publish list", ":", srcPubLst)
		http.Error(w, helper.MsgL(lang, "Invalid (empty) path to model publish list"), http.StatusBadRequest)
		return
	}
	pubLst = filepath.Join(mcf.ModelDir, pubLst)
	pubLst = filepath.ToSlash(pubLst)

	// if model name in json body is empty then use file name of model.publish.lst
	nameVer := filepath.Base(pubLst)
	if len(nameVer) > len(".publish.lst") {
		nameVer = nameVer[0 : len(nameVer)-len(".publish.lst")]
	}

	// BIN_DIR cannot be current dot. or file system root
	binD := filepath.Dir(pubLst)
	if binD == "." || binD == "/" || binD == "\\" || binD == "C:\\" {
		binD = ""
	}

	// do model copy, assume default path for DOC_DIR and LOG_DIR
	doCopyModel(pubLst, nameVer, "", binD, "", "", lang, w, r)
}

// copy model files from models library using json post body
//
//	POST /api/admin/copy-model
//	POST /api/admin/copy-model/lang/:lang
//
// Request json body must contain model.publish.lst file and model name or name-version.
// Model digest is highly recommended to avoid publishing model with the same digest.
// If model digest specified in json body then existing model with the same digest will be deleted.
//
// If model with same digest already exist in current oms then copy may fail and result is undefined.
func copyModelPostHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "lang") // get prefered language for log messages

	mcf, mlb := theCatalog.toPublicConfig()
	if !mlb.IsCopy {
		http.Error(w, helper.MsgL(lang, "Copy model disabled"), http.StatusBadRequest)
		return
	}

	// decode json options for copy model
	opts := struct {
		PublishLst  string // path/to/ModelName.publish.lst in models library
		NameVersion string // model name or name-version
		ModelDigest string // if not empty then model digest (highly recommended)
		DocDir      string // if not empty then model documentation folder from model.extra.json
		BinDir      string // if not empty then model bin folder
		LogDir      string // if not empty then model log folder
	}{}
	if !jsonRequestDecode(w, r, true, &opts) {
		return // error at json decode, response done with http error
	}

	// check if model.publish.lst exists in model source directory
	pubLst := filepath.Join(mcf.ModelDir, opts.PublishLst)
	srcPubLst := filepath.Join(mlb.srcRoot, pubLst)

	if opts.PublishLst == "" || !helper.IsFileExist(srcPubLst) {
		omppLog.Log("Error: invalid (empty) path to model publish list", ":", srcPubLst)
		http.Error(w, helper.MsgL(lang, "Invalid (empty) path to model publish list"), http.StatusBadRequest)
		return
	}
	pubLst = filepath.ToSlash(pubLst)

	// model name is required
	if opts.NameVersion == "" {
		omppLog.Log("Error: invalid (empty) model name-version")
		http.Error(w, helper.MsgL(lang, "Error: invalid (empty) model name-version"), http.StatusBadRequest)
		return
	}

	// BIN_DIR, DOC_DIR, LOG_DIR directories cannot be current dot. or file system root
	binD := opts.BinDir
	if binD != "" {
		binD = filepath.Join(mcf.ModelDir, binD)
	} else {
		binD = filepath.Dir(pubLst) // if not posted in json body then it must be directory/of/model.publist.lst
	}
	if binD == "." || binD == "/" || binD == "\\" || binD == "C:\\" {
		binD = ""
	}
	docD := opts.DocDir
	if docD == "." || docD == "/" || docD == "\\" || docD == "C:\\" {
		docD = ""
	}
	logD := opts.LogDir
	if logD == "." || logD == "/" || logD == "\\" || logD == "C:\\" {
		logD = ""
	}

	// do model copy
	doCopyModel(pubLst, opts.NameVersion, opts.ModelDigest, binD, docD, logD, lang, w, r)
}

// Copy model files from models library located in srcRoot directory.
// Model files listed in modelName.publish.lst file in models library.
// If model digest specified in json body then existing model with the same digest will be deleted.
// If model with same digest already exist in current oms then copy may fail and result is undefined.
func doCopyModel(pubLst, nameVer, digest, binDir, docDir, logDir string, lang string, w http.ResponseWriter, r *http.Request) {

	mcf, mlb := theCatalog.toPublicConfig()
	if !mlb.IsCopy {
		http.Error(w, helper.MsgL(lang, "Copy model disabled"), http.StatusBadRequest)
		return
	}

	srcPubLst := filepath.Join(mlb.srcRoot, pubLst)
	if pubLst == "" || !helper.IsFileExist(srcPubLst) {
		omppLog.Log("Error: invalid (empty) path to model publish list", ":", srcPubLst)
		http.Error(w, helper.MsgL(lang, "Invalid (empty) path to model publish list"), http.StatusBadRequest)
		return
	}
	pubLst = filepath.ToSlash(pubLst)

	// if model with the same digest alreday published then delete it
	var errCopy error
	cmdLog := newCmdLog("copy-model", nameVer, mcf.ModelLogDir) // create new batch process log file

	if digest != "" {
		if mb, ok := theCatalog.modelBasicByDigestOrName(digest); ok {

			m := helper.MsgL(lang, "Delete model:", digest, mb.model.Name)
			omppLog.Log("Delete model:", digest, mb.model.Name)
			cmdLog.toLog(true, m)

			if errCopy = theCatalog.deleteModel(digest); errCopy != nil {
				omppLog.LogNoLT(errCopy)
				m = helper.MsgL(lang, "Failed to delete model:", digest, mb.model.Name)
				cmdLog.toLogError(true, m)
				http.Error(w, m, http.StatusBadRequest)
				return
			}
		}
	}
	m := helper.MsgL(lang, "Copy model:", pubLst, nameVer)
	cmdLog.toLog(true, m)

	// do model copy
	go func() {

		// make model copy command:
		// model-copy.sh RiskPaths.publish.lst ../archive ../my-work RiskPaths v3.2.1
		cArgs := []string{
			pubLst,
			mlb.srcRoot,
			theCfg.rootDir,
			nameVer,
		}
		cmd := exec.Command(mlb.copyCmd, cArgs...)

		// set BIN_DIR, DOC_DIR, LOG_DIR environment
		if binDir != "" {
			cmd.Env = append(cmd.Environ(), "BIN_DIR="+binDir)
		}
		if docDir != "" {
			cmd.Env = append(cmd.Environ(), "DOC_DIR="+filepath.Join(mcf.ModelDocDir, docDir))
		}
		if logDir != "" {
			cmd.Env = append(cmd.Environ(), "LOG_DIR="+filepath.Join(mcf.ModelLogDir, logDir))
		}

		// connect console output to output log file
		outPipe, e := cmd.StdoutPipe()
		if e != nil {
			errCopy = e
			omppLog.Log("Error at join to stdout log", ": ", cmdLog.logPath, ": ", errCopy)
			return
		}
		errPipe, e := cmd.StderrPipe()
		if e != nil {
			errCopy = e
			omppLog.Log("Error at join to stderr log", ": ", cmdLog.logPath, ": ", errCopy)
			return
		}
		outDoneC := make(chan bool, 1)
		errDoneC := make(chan bool, 1)
		logTck := time.NewTicker(logTickTimeout * time.Millisecond)

		// start console output listners
		doLog := func(r io.Reader, done chan<- bool) {
			sc := bufio.NewScanner(r)
			for sc.Scan() {
				cmdLog.toLog(true, sc.Text())
			}
			done <- true
			close(done)
		}
		go doLog(outPipe, outDoneC)
		go doLog(errPipe, errDoneC)

		// start model copy
		omppLog.Log(strings.Join(cmd.Args, " "))
		cmdLog.toLog(true, strings.Join(cmd.Args, " "))

		errCopy = cmd.Start()
		if errCopy != nil {
			omppLog.Log("Error at", ": ", errCopy)
			cmdLog.toLogError(true, errCopy.Error())
			return
		}
		// else model copy started: wait until completed

		// wait until stdout and stderr closed
		for outDoneC != nil || errDoneC != nil {
			select {
			case _, ok := <-outDoneC:
				if !ok {
					outDoneC = nil
				}
			case _, ok := <-errDoneC:
				if !ok {
					errDoneC = nil
				}
			case <-logTck.C:
			}
		}

		// wait for model copy to be completed
		errCopy = cmd.Wait()
		if errCopy != nil {
			omppLog.Log("Error at: ", cmd.Args)
			cmdLog.toLogError(true, errCopy.Error())
			return
		}
		// else:
		// batch completed OK
		// refresh models catalog
		omppLog.Log("Refresh models catalog", mcf.ModelDir)
		cmdLog.toLog(true, helper.MsgL(lang, "Refresh models catalog"))

		_, errCopy = theCatalog.refreshSqlite(mcf.ModelDir, mcf.ModelLogDir)
		if errCopy != nil {
			omppLog.LogNoLT(errCopy)
			cmdLog.toLogError(true, "Failed to refresh models catalog")
			return
		}
		// else:
		// completed OK
		cmdLog.toLog(true, helper.MsgL(lang, "Done."))
		if !cmdLog.isLogOk {
			omppLog.Log("Warning: model copy log output may be incomplete")
		}
	}()

	// model copy is starting now: return path to log file
	em := ""
	if errCopy != nil {
		em = errCopy.Error()
	}
	jsonResponse(w, r, struct {
		LogFileName string
		IsError     bool
		ErrorMsg    string
	}{
		LogFileName: cmdLog.logName,
		IsError:     errCopy != nil || cmdLog.isCmdErr,
		ErrorMsg:    em,
	})
}

// get list of all db cleanup log files
//
//	GET /api/admin/db-cleanup/log-all
func dbCleanupAllLogGetHandler(w http.ResponseWriter, r *http.Request) {
	batchAllLogGetHandler("db-cleanup", w, r)
}

// get list of all copy model log files
//
//	GET /api/admin/copy-model/log-all
func copyModelAllLogGetHandler(w http.ResponseWriter, r *http.Request) {
	batchAllLogGetHandler("copy-model", w, r)
}

// get db cleanup log file content by name
//
//	GET /api/admin/db-cleanup/log/:name
func dbCleanupFileLogGetHandler(w http.ResponseWriter, r *http.Request) {
	batchFileLogGetHandler("db-cleanup", w, r)
}

// get copy model log file content by name
//
//	GET /api/admin/copy-model/log/:name
func copyModelFileLogGetHandler(w http.ResponseWriter, r *http.Request) {
	batchFileLogGetHandler("copy-model", w, r)
}

// get list of all batch process log files
//
//	GET /api/admin/db-cleanup/log-all
//	GET /api/admin/copy-model/log-all
func batchAllLogGetHandler(prefix string, w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	type fi struct {
		BaseName    string // base name: db name or model name
		LogStamp    string // log file date-time stamp
		LogFileName string // db-cleanup.2024_03_05_00_30_37_568.modelOne.sqlite.console.txt
		IsError     bool   // if true then it is an error log file name: db-cleanup.2024_03_05_00_30_37_568.modelOne.sqlite.error.txt
	}

	logDir, isLog := theCatalog.getModelLogDir()
	if !isLog {
		jsonResponse(w, r, []fi{}) // log is not enabled: empty response
		return
	}

	// get list of models/log/db-cleanup.*.txt files
	fiLst := []fi{}

	pl, err := filepath.Glob(logDir + string(filepath.Separator) + prefix + ".*.txt")
	if err != nil {
		http.Error(w, helper.MsgL(lang, "Error at batch process log files list"), http.StatusInternalServerError)
		return
	}
	for _, p := range pl {

		ts, base, isErrLog, fn := parseBatchLogPath(prefix, p)

		if ts != "" && base != "" {
			fiLst = append(fiLst, fi{
				BaseName:    base,
				LogStamp:    ts,
				LogFileName: fn,
				IsError:     isErrLog,
			})
		}
	}
	// sort by time stamp and base name (model name)
	slices.SortStableFunc(fiLst, func(left fi, right fi) int {
		return cmp.Or(strings.Compare(left.LogStamp, right.LogStamp), cmp.Compare(left.BaseName, right.BaseName))
	})

	jsonResponse(w, r, fiLst)
}

// batch process log file name and state
type cmdLog struct {
	logName  string // log file name, e.g.: copy-model.2022_08_09_23_45_06_777.RiskPaths.console.txt
	logPath  string // log file path, e.g.: log/copy-model.2022_08_09_23_45_06_777.RiskPaths.console.txt
	isLogOk  bool   // if false then log write failed
	isCmdErr bool   // if true then batch process set error flag or log file path suffix is.error.txt
}

// create new batch process log file
func newCmdLog(prefix, baseName string, logDir string) *cmdLog {

	log := cmdLog{}

	log.logName, log.logPath = batchLogNamePath(prefix, baseName, false, logDir)

	log.isLogOk = fileCreateEmpty(false, log.logPath)
	if !log.isLogOk {
		omppLog.Log("Error at creating log file", ": ", log.logPath)
	}
	return &log
}

// return log file path suffix is.error.txt
func cmdLogIsErrorName(logPath string) bool { return strings.HasSuffix(logPath, ".error.txt") }

// append message into batch process log file
func (log *cmdLog) toLog(isTs bool, m string) {
	if log.isLogOk {
		log.isLogOk = writeToCmdLog(log.logPath, isTs, m)
	}
	log.isCmdErr = log.isCmdErr || cmdLogIsErrorName(log.logPath)
}

// rename batch process log file from name.console.txt to name.error.txt
// and append message into the log file
func (log *cmdLog) toLogError(isTs bool, m string) {
	if log.isLogOk {
		if strings.HasSuffix(log.logPath, ".console.txt") {
			lp := log.logPath[:len(log.logPath)-len(".console.txt")] + ".error.txt"

			if e := os.Rename(log.logPath, lp); e == nil {
				log.logPath = lp
			}
		}
		log.isLogOk = writeToCmdLog(log.logPath, isTs, m)
	}
	log.isCmdErr = true
}

// get batch process log file content by name
//
//	GET /api/admin/db-cleanup/log/:name
//	GET /api/admin/copy-model/log/:name
func batchFileLogGetHandler(prefix string, w http.ResponseWriter, r *http.Request) {

	// check log file: it must be db cleanup log file
	logName := getRequestParam(r, "name")
	lang := preferedRequestLang(r, "") // get prefered language for messages

	ts, base, isErrLog, fn := parseBatchLogPath(prefix, logName)
	if ts == "" || base == "" {
		http.Error(w, helper.MsgL(lang, "Invalid batch process log file name", logName), http.StatusBadRequest)
		return
	}

	// response: db cleanup log file info and content
	st := struct {
		BaseName    string   // base name: db name or model name
		LogStamp    string   // log file date-time stamp
		LogFileName string   // copy-model.2022_08_09_23_45_06_777.RiskPaths.console.txt
		IsError     bool     // if true then it is an error log file name: copy-model.2022_08_09_23_45_06_777.RiskPaths.error.txt
		Size        int64    // bytes, log file size
		ModTs       int64    // unix milliseconds, log file update time
		Lines       []string // log file content
	}{
		Lines: []string{},
	}

	// check if log file exists in models/log directory
	logDir, isLog := theCatalog.getModelLogDir()
	if !isLog {
		jsonResponse(w, r, []string{}) // log is not enabled: empty response
		return
	}
	logPath := filepath.Join(logDir, fn)

	fi, err := helper.FileStat(logPath)
	if err != nil && !isErrLog { // file renamed from .console.txt to .error.txt

		afn := fn[:len(fn)-len(".console.txt")] + ".error.txt"
		logPath = filepath.Join(logDir, afn)

		var e error
		if fi, e = helper.FileStat(logPath); e == nil { // if log file exist then read from alternative log file
			err = nil
			fn = afn // .error.txt file name
			isErrLog = true
		}
	}
	if err != nil { // log file not found, retrun first error
		http.Error(w, helper.MsgL(lang, "Error at reading batch process log file:", err), http.StatusBadRequest)
		return
	}

	// read log file content and return result
	st.BaseName = base
	st.LogStamp = ts
	st.LogFileName = fn
	st.IsError = isErrLog
	st.Size = fi.Size()
	st.ModTs = fi.ModTime().UnixMilli()
	st.Lines, _ = readLogFile(logPath)

	jsonResponse(w, r, st)
}

// Return batch process log file name and file path, for example:
// db-cleanup.2022_07_08_23_03_27_555.RiskPaths.console.txt
// db-cleanup.2024_03_05_00_30_37_568.modelOne.sqlite.error.txt
// copy-model.2022_08_09_23_45_06_777.RiskPaths.console.txt
// copy-model.2022_08_09_23_45_06_777.RiskPaths.error.txt
func batchLogNamePath(prefix, baseName string, isErrLog bool, logDir string) (string, string) {

	ts, _ := theCatalog.getNewTimeStamp()
	fn := prefix + "." + ts + "." + baseName

	if !isErrLog {
		fn = prefix + "." + ts + "." + baseName + ".console.txt"
	} else {
		fn = prefix + "." + ts + "." + baseName + ".error.txt"
	}
	return fn, filepath.Join(logDir, fn)
}

// parse batch process log path, it can be:
//
//	log/db-cleanup.2022_07_08_23_03_27_555.RiskPaths.console.txt
//	log/db-cleanup.2024_03_05_00_30_37_568.modelOne.sqlite.error.txt
//	log/copy-model.2022_08_09_23_45_06_777.RiskPaths.console.txt
//	log/copy-model.2022_08_09_23_45_06_777.RiskPaths.error.txt
//
// Remove directory, remove prefix. , remove .console.txt or .error.txt extension.
// Return date-time stamp, base name, extension .error.txt flag and file name, for example:
//
//	srcPath:
//		log/copy-model.2022_08_09_23_45_06_777.RiskPaths.console.txt
//	return:
//		2022_08_09_23_45_06_777, RiskPaths, false, copy-model.2022_08_09_23_45_06_777.RiskPaths.console.txt
func parseBatchLogPath(prefix, srcPath string) (string, string, bool, string) {

	_, fn := filepath.Split(srcPath)

	pd := prefix + "."
	if !strings.HasPrefix(fn, pd) {
		return "", "", true, ""
	}
	p := fn[len(pd):] // remove prefix.

	isErrLog := strings.HasSuffix(fn, ".error.txt")
	if isErrLog {
		p = p[:len(p)-len(".error.txt")]
	} else {
		if !strings.HasSuffix(fn, ".console.txt") {
			return "", "", true, "" // invalid extension: it must be .console.txt or .error.txt
		}
		p = p[:len(p)-len(".console.txt")]
	}

	// check result: it must 2 non-empty parts and first must be a time stamp
	sp := strings.SplitN(p, ".", 2)

	if len(sp) < 2 || !helper.IsUnderscoreTimeStamp(sp[0]) || sp[1] == "" {
		return "", "", true, "" // source file path is not db cleanup log file
	}
	return sp[0], sp[1], isErrLog, fn
}
