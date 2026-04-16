// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"net/http"
	"strconv"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// return service configuration: model catalog, run catalog, job service and disk use configuartion.
//
//	GET /api/service/config
func serviceConfigHandler(w http.ResponseWriter, r *http.Request) {

	st := struct {
		OmsName        string             // service instance name
		DoubleFmt      string             // format to convert float or double value to string
		LoginUrl       string             // user login URL for UI
		LogoutUrl      string             // user logout URL for UI
		AllowUserHome  bool               // if true then store user settings in home directory
		AllowDownload  bool               // if true then allow download from home/io/download directory
		AllowUpload    bool               // if true then allow upload from home/io/upload directory
		AllowFiles     bool               // if true then allow user files, if home directory specified then files directory: home/io
		AllowMicrodata bool               // if true then allow model run microdata
		IsJobControl   bool               // if true then job control enabled
		IsModelDoc     bool               // if true then model documentation is enabled
		IsDiskUse      bool               // if true then storage usage control enabled
		IsDiskCleanup  bool               // if true then disk cleanup enabled
		IsAdminAll     bool               // if true then it is global admin service
		JobServicePub                     // jobs service state: paused, resources usage and limits
		DiskUse        diskUseConfig      // disk use config
		Env            map[string]string  // server config environmemt variables for UI
		UiExtra        string             // UI extra config from etc/ui.extra.json
		ModelCatalog   ModelCatalogConfig // "public" state of model catalog
		RunCatalog     RunCatalogConfig   // "public" state of model run catalog
	}{
		OmsName:        theCfg.omsName,
		DoubleFmt:      theCfg.doubleFmt,
		AllowUserHome:  theCfg.isHome,
		AllowDownload:  theCfg.downloadDir != "",
		AllowUpload:    theCfg.uploadDir != "",
		AllowFiles:     theCfg.filesDir != "",
		AllowMicrodata: theCfg.isMicrodata,
		IsJobControl:   theCfg.isJobControl,
		IsAdminAll:     theCfg.isAdminAll,
		JobServicePub:  theRunCatalog.getJobServicePub(),
		IsModelDoc:     theCfg.docDir != "",
		IsDiskUse:      theCfg.isDiskUse,
		Env:            theCfg.env,
		UiExtra:        theCfg.uiExtra,
		ModelCatalog:   theCatalog.toPublicConfig(),
		RunCatalog:     *theRunCatalog.toPublicConfig(),
	}
	if theCfg.isDiskUse {
		_, st.DiskUse = theRunCatalog.getDiskUseStatus()
		st.IsDiskCleanup = st.DiskUse.dbCleanupCmd != "" && theCfg.dbcopyPath != ""
	}

	jsonResponse(w, r, st)
}

// return job service state: model runs queue, active runs and run history
//
//	GET /api/service/state
func serviceStateHandler(w http.ResponseWriter, r *http.Request) {

	// it can be inconsistent between job service state, compute state and jobs state
	st := struct {
		IsJobControl  bool             // if true then job control enabled
		IsAdminAll    bool             // if true then it is global admin service
		JobServicePub                  // jobs service state: paused, resources usage and limits
		Queue         []RunJob         // list of model run jobs in the queue
		Active        []RunJob         // list of active (currently running) model run jobs
		History       []historyJobFile // history of model runs
		ComputeState  []computePub     // state of computational servers or clusters
		IsDiskUse     bool             // if true then storage usage control enabled
		IsDiskCleanup bool             // if true then disk cleanup enabled
		IsDiskOver    bool             // if true then storage use reach the limit
		diskUseConfig                  // storage use settings
	}{
		IsJobControl: theCfg.isJobControl,
		IsAdminAll:   theCfg.isAdminAll,
		Queue:        []RunJob{},
		Active:       []RunJob{},
		History:      []historyJobFile{},
		ComputeState: []computePub{},
		IsDiskUse:    theCfg.isDiskUse,
	}

	if theCfg.isJobControl {

		st.JobServicePub = theRunCatalog.getJobServicePub()
		st.ComputeState = theRunCatalog.getComputePub()

		qKeys, qJobs, aKeys, aJobs, hKeys, hJobs := theRunCatalog.getRunJobs()

		// clean active and queue job details, do not expose path or environment
		st.Queue = make([]RunJob, len(qKeys))
		for k := range qKeys {
			st.Queue[k] = qJobs[k]
			st.Queue[k].Env = map[string]string{}
		}

		st.Active = make([]RunJob, len(aKeys))
		for k := range aKeys {
			st.Active[k] = aJobs[k]
			st.Active[k].Pid = 0
			st.Active[k].CmdPath = ""
			st.Active[k].CmdLine = ""
			st.Active[k].LogPath = ""
			st.Active[k].BinDir = ""
			st.Active[k].WorkDir = ""
			// st.Active[k].IniPath = ""
			st.Active[k].HostFilePath = ""
			st.Active[k].Env = map[string]string{}
		}

		st.History = make([]historyJobFile, len(hKeys))
		for k := range hKeys {
			st.History[k] = hJobs[k]
		}
	}

	if theCfg.isDiskUse {
		st.IsDiskOver, st.diskUseConfig = theRunCatalog.getDiskUseStatus()
		st.IsDiskCleanup = st.dbCleanupCmd != "" && theCfg.dbcopyPath != ""
	}

	jsonResponse(w, r, st)
}

// return disk use state: summary of disk use and list of model database files size.
//
//	GET /api/service/disk-use
func serviceDiskUseHandler(w http.ResponseWriter, r *http.Request) {

	st := struct {
		IsDiskUse bool         // if true then storage usage control enabled
		DiskUse   diskUseState // storage space use state
		DbDiskUse []dbDiskUse  // model db file disk usage
	}{
		IsDiskUse: theCfg.isDiskUse,
		DbDiskUse: []dbDiskUse{},
	}

	if theCfg.isDiskUse {
		st.DiskUse, st.DbDiskUse = theRunCatalog.getDiskUse()
	}
	jsonResponse(w, r, st)
}

// refersh disk use state: scan disk usage now.
//
//	POST /api/service/disk-use/refresh
func serviceRefreshDiskUseHandler(w http.ResponseWriter, r *http.Request) {

	if theCfg.isDiskUse {
		refreshDiskScanC <- true
	}
	// respond with disk usage active status
	w.Header().Set("Content-Location", "/api/service/disk-use/refresh/"+strconv.FormatBool(theCfg.isDiskUse))
}

// job control state, log file content and run progress
type runJobState struct {
	JobStatus string      // if not empty then job run status: success, error, exit
	RunJob                // job control state: job control file content
	RunStatus []db.RunPub // if not empty then run_lst and run_progerss from db
	Lines     []string    // log file content
}

// return empty value of job control state
func emptyRunJobState(submitStamp string) runJobState {
	return runJobState{
		RunJob:    emptyRunJob(submitStamp),
		RunStatus: []db.RunPub{},
		Lines:     []string{},
	}
}

// return empty value of run job
func emptyRunJob(submitStamp string) RunJob {
	return RunJob{
		SubmitStamp: submitStamp,
		RunRequest: RunRequest{
			Opts:   map[string]string{},
			Env:    map[string]string{},
			Tables: []string{},
			Microdata: struct {
				IsToDb     bool
				IsInternal bool
				Entity     []struct {
					Name string
					Attr []string
				}
			}{
				Entity: []struct {
					Name string
					Attr []string
				}{},
			},
			RunNotes: []struct {
				LangCode string
				Note     string
			}{}},
	}
}

// return empty value of past job
func emptyPastRunJob(submitStamp string) PastRunJob {
	return PastRunJob{RunJob: emptyRunJob(submitStamp)}
}

// return active job state, run log file content and, if model run exists in database then also run progress
//
//	GET /api/service/job/active/:job
func jobActiveHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// url or query parameters: submission stamp
	submitStamp := getRequestParam(r, "job")
	if submitStamp == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) submission stamp"), http.StatusBadRequest)
		return
	}

	// find job state in run catalog
	aj, isOk := theRunCatalog.getActiveJobItem(submitStamp)

	if !isOk || aj.isError {
		jsonResponse(w, r, emptyRunJobState(submitStamp)) // job not found or job control file error
		return
	}

	// get job control state, read log file and run progress, if it is available
	isOk, st := getJobState(aj.filePath)
	if !isOk {
		jsonResponse(w, r, emptyRunJobState(submitStamp)) // unable to read job control file
		return
	}

	jsonResponse(w, r, st) // return final result
}

// return queue job state
//
//	GET /api/service/job/queue/:job
func jobQueueHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// url or query parameters: submission stamp
	submitStamp := getRequestParam(r, "job")
	if submitStamp == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) submission stamp"), http.StatusBadRequest)
		return
	}

	// find job state in run catalog
	qj, isOk := theRunCatalog.getQueueJobItem(submitStamp)

	if !isOk || qj.isError {
		jsonResponse(w, r, emptyRunJobState(submitStamp)) // job not found or job control file error
		return
	}

	// get job control state, log file and run progress are always empty
	isOk, st := getJobState(qj.filePath)
	if !isOk {
		jsonResponse(w, r, emptyRunJobState(submitStamp)) // unable to read job control file
		return
	}

	jsonResponse(w, r, st) // return final result
}

// return history job state, run log file content and, if model run exists in database then also return run progress
//
//	GET /api/service/job/history/:job
func jobHistoryHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// url or query parameters: submission stamp
	submitStamp := getRequestParam(r, "job")
	if submitStamp == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) submission stamp"), http.StatusBadRequest)
		return
	}

	// find job state in run catalog
	hj, isOk := theRunCatalog.getHistoryJobItem(submitStamp)

	if !isOk || hj.isError {
		jsonResponse(w, r, emptyRunJobState(submitStamp)) // job not found or job control file error
		return
	}

	// get job control state, read log file and run progress, if it is available
	isOk, st := getJobState(hj.filePath)
	if !isOk {
		jsonResponse(w, r, emptyRunJobState(submitStamp)) // unable to read job control file
		return
	}
	st.JobStatus = hj.JobStatus

	jsonResponse(w, r, st) // return final result
}

// returns job control file content, run log file content and, if model run exists in database then also return run progress
// it is clear (set to empty) server-only part of job state: pid, exe path, log path and environment
func getJobState(filePath string) (bool, *runJobState) {

	st := emptyRunJobState("")

	// read job control file
	var jc RunJob
	isOk, err := helper.FromJsonFile(filePath, &jc)
	if err != nil {
		omppLog.LogNoLT(err)
	}
	if !isOk || err != nil {
		return false, &st
	}

	// set job state and clear server-only part of the job state: pid, exe path, log path and environment
	st.RunJob = jc
	st.Pid = 0
	st.CmdPath = ""
	st.CmdLine = ""
	st.LogPath = ""
	st.BinDir = ""
	st.WorkDir = ""
	// st.IniPath = ""
	st.HostFilePath = ""
	st.Env = map[string]string{}

	setRunJobDefaults(&st.RunJob) // set default values for empty fields instead of nil

	// read log file content
	if jc.LogPath != "" {
		st.Lines, _ = readLogFile(jc.LogPath)
	}

	// get run progress if model run exist in database
	if jc.ModelDigest != "" && jc.RunStamp != "" {
		if _, isOk = theCatalog.ModelDicByDigest(jc.ModelDigest); isOk { // if model exist
			if rst, isOk := theCatalog.RunStatusList(jc.ModelDigest, jc.RunStamp); isOk { // get run status
				st.RunStatus = rst
			}
		}
	}

	return true, &st // return final result
}

// set job state default empty values instead of missing null values
func setRunJobDefaults(jc *RunJob) {
	if len(jc.Opts) == 0 {
		jc.Opts = map[string]string{}
	}
	if jc.Tables == nil {
		jc.Tables = []string{}
	}
	if jc.Microdata.Entity == nil {
		jc.Microdata.Entity = []struct {
			Name string
			Attr []string
		}{}
	}
	if jc.RunNotes == nil {
		jc.RunNotes = []struct {
			LangCode string
			Note     string
		}{}
	}
}

// move job into the specified queue index position.
//
//	PUT /api/service/job/move/:pos/:job
//
// Top of the queue position is zero, negative position treated as zero.
// If position number exceeds queue length then job moved to the bottom of the queue.
func jobMoveHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// url or query parameters: position and submission stamp
	sp := getRequestParam(r, "pos")
	nPos, err := strconv.Atoi(sp)
	if sp == "" || err != nil {
		http.Error(w, helper.MsgL(lang, "Invalid (or empty) job queue position"), http.StatusBadRequest)
		return
	}

	submitStamp := getRequestParam(r, "job")
	if submitStamp == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) submission stamp"), http.StatusBadRequest)
		return
	}

	// move job in the queue and rename files in the queue
	isOk, fileMoveLst := theRunCatalog.moveJobInQueue(submitStamp, nPos)

	for _, fm := range fileMoveLst {
		fileMoveAndLog(false, fm[0], fm[1])
	}

	w.Header().Set("Content-Type", "text/plain")
	if !isOk {
		w.Header().Set("Content-Location", "service/job/move/false/"+sp+"/"+submitStamp)
		return
	}
	// else: job moved into the specified queue position
	w.Header().Set("Content-Location", "service/job/move/true/"+sp+"/"+submitStamp)
}

// delete only job history json file, it does not delete model run.
//
//	DELETE /api/service/job/delete/history/:job
func jobHistoryDeleteHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// url or query parameters: submission stamp
	submitStamp := getRequestParam(r, "job")
	if submitStamp == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) submission stamp"), http.StatusBadRequest)
		return
	}

	// find job history in run catalog
	hj, isOk := theRunCatalog.getHistoryJobItem(submitStamp)
	if isOk {

		isOk = fileDeleteAndLog(true, hj.filePath)
		if !isOk {
			http.Error(w, helper.MsgL(lang, "Unable to delete job file"), http.StatusInternalServerError)
			return
		}
	}
	// job history file deleted or job history not found
	w.Header().Set("Content-Location", "/api/service/job/delete/history/"+submitStamp)
}

// delete all successful or not successful jobs history json files.
// it does not delete model runs.
//
//	DELETE /api/service/job/delete/history-all/:success
func jobHistoryAllDeleteHandler(w http.ResponseWriter, r *http.Request) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// url or query parameters: successful or not boolean flag
	sp := getRequestParam(r, "success")
	isSuccess, err := strconv.ParseBool(sp)
	if sp == "" || err != nil {
		http.Error(w, helper.MsgL(lang, "Invalid (or empty) history delete flag, expected true or false"), http.StatusBadRequest)
		return
	}

	doJobHistoryAllDelete(isSuccess, w, lang)
}

// delete all successful or not successful jobs history json files, it does not delete model runs.
func doJobHistoryAllDelete(isSuccess bool, w http.ResponseWriter, lang string) {

	nDel := 0
	if theCfg.isJobControl {

		_, _, _, _, hKeys, hJobs := theRunCatalog.getRunJobs()

		for k := range hKeys {

			if isSuccess && hJobs[k].JobStatus != "success" || !isSuccess && hJobs[k].JobStatus == "success" {
				continue
			}
			if isOk := fileDeleteAndLog(true, hJobs[k].filePath); !isOk {
				http.Error(w, helper.MsgL(lang, "Unable to delete job file", hJobs[k].SubmitStamp), http.StatusInternalServerError)
				return
			}
			nDel++
		}
	}

	// all job history file deleted
	if isSuccess {
		w.Header().Set("Content-Location", "/api/service/job/delete/history-all-success/"+strconv.Itoa(nDel))
		return
	} // else
	w.Header().Set("Content-Location", "/api/service/job/delete/history-all-not-success/"+strconv.Itoa(nDel))
}
