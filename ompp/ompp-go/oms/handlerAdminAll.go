// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"net/http"
	"path"
	"path/filepath"
	"strconv"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// pause or resume jobs queue processing by all oms instances
//
//	POST /api/admin-all/jobs-pause/:pause
func jobsAllPauseHandler(w http.ResponseWriter, r *http.Request) {
	doJobsPause(jobAllQueuePausedPath(), "/api/admin-all/jobs-pause/", w, r)
}

// for all oms instances: get state, computational and disk resources usage
//
//	GET /api/admin-all/state
func adminAllStateGetHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	if !theCfg.isAdminAll {
		http.Error(w, helper.MsgL(lang, "Forbidden: disabled on the server"), http.StatusForbidden)
		return
	}

	ast, jsp, csp := theRunCatalog.getAllAdminPubState()

	st := struct {
		IsAdminAll    bool         // if true then allow global administrative routes: /admin-all/
		IsJobControl  bool         // if true then job control enabled
		IsJobPast     bool         // if true then job past shadow history enabled
		IsDiskUse     bool         // if true then storage usage control enabled
		JobServicePub              // jobs service state: paused, resources usage and limits
		ComputeState  []computePub // state of computational servers or clusters
		adminState                 // model run state and resources usage: for global admin only
	}{
		IsAdminAll:    theCfg.isAdminAll,
		IsJobControl:  theCfg.isJobControl,
		IsJobPast:     theCfg.isJobPast,
		IsDiskUse:     theCfg.isDiskUse,
		JobServicePub: jsp,
		ComputeState:  csp,
		adminState:    ast,
	}
	jsonResponse(w, r, st)
}

// for global admin: get queue job state content by oms instance name and submit stamp
//
//	GET /api/admin-all/job/queue/user/:user/stamp/:stamp/state
func adminAllJobQueueStateHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	if !theCfg.isAdminAll {
		http.Error(w, helper.MsgL(lang, "Forbidden: disabled on the server"), http.StatusForbidden)
		return
	}

	// url or query parameters: oms user name and submission stamp
	oms := getRequestParam(r, "user")
	if oms == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) oms user name"), http.StatusBadRequest)
		return
	}
	submitStamp := getRequestParam(r, "stamp")
	if submitStamp == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) submission stamp"), http.StatusBadRequest)
		return
	}

	// find queue job request by oms name and submit stamp
	qr, ok := theRunCatalog.findQueueRequest(oms, submitStamp)
	if !ok {
		jsonResponse(w, r, emptyRunJob(submitStamp)) // queue request not found, it may be completed
		return
	}
	// read job control state
	ok, jc := readJobStateFile(qr.filePath)
	if !ok {
		jsonResponse(w, r, emptyRunJob(submitStamp)) // unable to read job control file
		return
	}

	jsonResponse(w, r, jc) // return final result
}

// for global admin: get active job state content by oms instance name and submit stamp
//
//	GET /api/admin-all/job/active/user/:user/stamp/:stamp/state
func adminAllJobActiveStateHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	if !theCfg.isAdminAll {
		http.Error(w, helper.MsgL(lang, "Forbidden: disabled on the server"), http.StatusForbidden)
		return
	}

	// url or query parameters: oms user name and submission stamp
	oms := getRequestParam(r, "user")
	if oms == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) oms user name"), http.StatusBadRequest)
		return
	}
	submitStamp := getRequestParam(r, "stamp")
	if submitStamp == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) submission stamp"), http.StatusBadRequest)
		return
	}

	// find active job run usage by oms name and submit stamp
	ru, ok := theRunCatalog.findActiveRunUsage(oms, submitStamp)
	if !ok {
		jsonResponse(w, r, emptyRunJob(submitStamp)) // job not found, it may be completed
		return
	}
	// read job control state
	ok, jc := readJobStateFile(ru.filePath)
	if !ok {
		jsonResponse(w, r, emptyRunJob(submitStamp)) // unable to read job control file
		return
	}
	// if log log file path is empty for that active run then set log file in active runs list
	if ru.logPath == "" && jc.LogPath != "" {
		theRunCatalog.updateActiveJobLogPath(oms, submitStamp, jc.LogPath)
	}
	jsonResponse(w, r, jc) // return final result
}

// for global admin: get active job log file name and content by oms instance name and submit stamp, response is string array, first line is a file name
//
//	GET /api/admin-all/job/active/user/:user/stamp/:stamp/log
func adminAllJobActiveLogHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	if !theCfg.isAdminAll {
		http.Error(w, helper.MsgL(lang, "Forbidden: disabled on the server"), http.StatusForbidden)
		return
	}

	// url or query parameters: oms user name and submission stamp
	oms := getRequestParam(r, "user")
	if oms == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) oms user name"), http.StatusBadRequest)
		return
	}
	submitStamp := getRequestParam(r, "stamp")
	if submitStamp == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) submission stamp"), http.StatusBadRequest)
		return
	}

	// find active job run usage by oms name and submit stamp
	lf := []string{}

	ru, ok := theRunCatalog.findActiveRunUsage(oms, submitStamp)
	if !ok {
		jsonResponse(w, r, lf) // job not found, it may be completed
		return
	}

	//  if log path is empty then read job control file to get log path from it
	if ru.logPath == "" {
		ok, jc := readJobStateFile(ru.filePath)
		if !ok {
			jsonResponse(w, r, lf) // unable to read job control file
			return
		}

		// update log log file path in active runs list
		if jc.LogPath != "" {
			theRunCatalog.updateActiveJobLogPath(oms, submitStamp, jc.LogPath)
		}
		ru.logPath = jc.LogPath
	}
	if ru.logPath == "" {
		jsonResponse(w, r, lf) // log path is empty, log disabled
		return
	}

	// read log content and return final result
	lf = append(lf, filepath.Base(ru.logPath)) // first line is log file name

	if lines, ok := readLogFile(ru.logPath); ok {
		lf = append(lf, lines...)
	}
	jsonResponse(w, r, lf)
}

// read job control file content
func readJobStateFile(filePath string) (bool, *RunJob) {

	st := emptyRunJob("")
	if filePath == "" {
		return false, &st
	}

	// read job control file
	isOk, err := helper.FromJsonFile(filePath, &st)
	if err != nil {
		omppLog.LogNoLT(err)
	}
	if !isOk || err != nil {
		return false, &st
	}
	setRunJobDefaults(&st) // set default values for empty fields instead of nil

	return true, &st
}

// for global admin: stop queue run: find and delete job control file from the queue
//
//	PUT /api/admin-all/run/stop/queue/user/:user/stamp/:stamp
func adminAllJobStopQueueRunHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	if !theCfg.isAdminAll {
		http.Error(w, helper.MsgL(lang, "Forbidden: disabled on the server"), http.StatusForbidden)
		return
	}

	// url or query parameters: oms user name and submission stamp
	oms := getRequestParam(r, "user")
	if oms == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) oms user name"), http.StatusBadRequest)
		return
	}
	stamp := getRequestParam(r, "stamp")
	if stamp == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) submission stamp"), http.StatusBadRequest)
		return
	}

	// find queue job request by oms name and submit stamp
	qr, isFound := theRunCatalog.findQueueRequest(oms, stamp)
	if !isFound {
		http.Error(w, helper.MsgL(lang, "Model run not found:", oms, stamp), http.StatusBadRequest)
		return // empty result: oms or submission stamp not found in the queue
	}

	isFound = moveJobQueueOmsToFailed(qr.filePath, stamp, oms, qr.ModelName, qr.ModelDigest, stamp, true) // move job control file to history

	// write queue job run key as response
	w.Header().Set("Content-Location", "/api/admin-all/run/stop/queue/"+oms+"/stamp/"+stamp+"/"+strconv.FormatBool(isFound))
	w.Header().Set("Content-Type", "text/plain")
}

// for global admin: get past jobs files tree
//
//	GET /api/admin-all/job/past/file-tree
func adminAllJobPastTreeHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	if !theCfg.isAdminAll {
		http.Error(w, helper.MsgL(lang, "Forbidden: disabled on the server"), http.StatusForbidden)
		return
	}

	if !theCfg.isJobPast {
		jsonResponse(w, r, []PathItem{}) // job shadow history disabled: return empty result
		return
	}

	// walk past directory and return path array, do not build tree
	pd := filepath.Join(theCfg.jobDir, "past")
	pathLst, err := filesWalk(pd, "", "", false, lang)
	if err != nil {
		http.Error(w, helper.MsgL(lang, "Error at directory walk:", pd), http.StatusBadRequest)
		return
	}

	// parse shadow history path list: past/yyyy_mm/ shadow history job file name, for example:
	//
	// 2022_07/2022_07_08_23_03_27_555-#-_4040-#-RiskPaths-#-d90e1e9a-#-2022_07_04_20_06_10_818-#-mpi-#-cpu-#-8-#-mem-#-4-#-3600-#-success.json
	//
	// submission stamp, oms instance name, model name, digest, run stamp, MPI or local, cpu cores, memory size in GBytes, run time in seconds and run status.
	//
	type pastJob struct {
		YearMonth   string // year-month sub-folder: yyyy_mm
		IsDir       bool   // if true then it is a directory else past job file name
		Oms         string // oms instance name
		SubmitStamp string // submission timestamp
		ModelName   string // model name
		ModelDigest string // model digest
		RunStamp    string // model run stamp, may be auto-generated as timestamp
		ComputeRes         // run total resources: cpu count and memory size
		IsMpi       bool   // if true then it use MPI to run the model
		TotalSec    int64  // seconds, total model run time
		Status      string // run status: success, kill or error
		srcPath     string // source path in / slash form
	}
	pjLst := make([]pastJob, 0, len(pathLst))

	for _, p := range pathLst {

		if p.IsDir { // folder name expected: past/yyyy_mm

			ym := path.Base(p.Path)

			if ym != "" && ym != "." && ym != ".." && ym != "/" {
				pjLst = append(pjLst,
					pastJob{YearMonth: ym, IsDir: true, srcPath: p.Path},
				)
			}
			continue
		}
		// else expected past job control file name

		ym, subStamp, oms, mn, dgst, rStamp, isMpi, cpu, mem, tSec, status := parsePastPath(p.Path)
		if subStamp == "" || oms == "" || mn == "" || dgst == "" || rStamp == "" {
			continue // file name is not a job file name
		}

		pjLst = append(pjLst,
			pastJob{
				YearMonth:   ym,
				IsDir:       false,
				Oms:         oms,
				SubmitStamp: subStamp,
				ModelName:   mn,
				ModelDigest: dgst,
				RunStamp:    rStamp,
				ComputeRes:  ComputeRes{Cpu: cpu, Mem: mem},
				IsMpi:       isMpi,
				TotalSec:    tSec,
				Status:      status,
				srcPath:     p.Path,
			})
	}

	jsonResponse(w, r, pjLst)
}

// for global admin: get past job state content by past folder, oms instance name and submit stamp
//
//	GET /api/admin-all/job/past/folder/:path/user/:user/stamp/:stamp/state
func adminAllJobPastStateHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	if !theCfg.isAdminAll {
		http.Error(w, helper.MsgL(lang, "Forbidden: disabled on the server"), http.StatusForbidden)
		return
	}

	// url or query parameters: past folder path, oms user name and submission stamp
	dir := getRequestParam(r, "path")
	if dir == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) folder path"), http.StatusBadRequest)
		return
	}
	oms := getRequestParam(r, "user")
	if oms == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) oms user name"), http.StatusBadRequest)
		return
	}
	submitStamp := getRequestParam(r, "stamp")
	if submitStamp == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) submission stamp"), http.StatusBadRequest)
		return
	}

	// find and read job control state file
	ok, pj := readPastStateFile(dir, oms, submitStamp)
	if !ok {
		jsonResponse(w, r, emptyPastRunJob(submitStamp)) // unable to read job control file
		return
	}

	jsonResponse(w, r, pj) // return final result
}

// find and read past folder job control file content by past folder, oms instance name and submit stamp
func readPastStateFile(dir, oms, stamp string) (bool, *PastRunJob) {

	st := emptyPastRunJob("")
	if !theCfg.isJobPast || dir == "" || oms == "" || stamp == "" {
		return false, &st
	}

	// find past job control state file
	ptrn := filepath.Join(theCfg.jobDir, "past", dir) + string(filepath.Separator) + stamp + "-#-" + oms + "-#-*.json"

	p := ""
	if fLst, err := filepath.Glob(ptrn); err == nil && len(fLst) > 0 {
		p = fLst[0]
	}
	if p == "" {
		return false, &st
	}

	// read job control file
	isOk, err := helper.FromJsonFile(p, &st)
	if err != nil {
		omppLog.LogNoLT(err)
	}
	if !isOk || err != nil {
		return false, &st
	}
	setRunJobDefaults(&st.RunJob) // set default values for empty fields instead of nil

	return true, &st // return final result
}

// for global admin: get past job log file name and content by past/folder, oms instance name and submit stamp, response is string array, first line is a file name
//
//	GET /api/admin-all/job/past/folder/:path/user/:user/stamp/:stamp/log
func adminAllJobPastLogHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	if !theCfg.isAdminAll {
		http.Error(w, helper.MsgL(lang, "Forbidden: disabled on the server"), http.StatusForbidden)
		return
	}

	// url or query parameters: past folder path, oms user name and submission stamp
	dir := getRequestParam(r, "path")
	if dir == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) folder path"), http.StatusBadRequest)
		return
	}
	oms := getRequestParam(r, "user")
	if oms == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) oms user name"), http.StatusBadRequest)
		return
	}
	submitStamp := getRequestParam(r, "stamp")
	if submitStamp == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) submission stamp"), http.StatusBadRequest)
		return
	}

	// find and read job control state file
	lf := []string{}

	if theCfg.isJobPast {

		ok, pj := readPastStateFile(dir, oms, submitStamp)
		if !ok {
			jsonResponse(w, r, lf) // job not found, it may be completed
			return
		}

		//  if log path is empty then read job control file to get log path from it
		if pj.LogPath == "" {
			jsonResponse(w, r, lf) // job not found, it may be completed
			return
		}

		// read log content and return final result
		lf = append(lf, filepath.Base(pj.LogPath)) // first line is log file name

		if lines, ok := readLogFile(pj.LogPath); ok {
			lf = append(lf, lines...)
		}
	}
	jsonResponse(w, r, lf)
}
