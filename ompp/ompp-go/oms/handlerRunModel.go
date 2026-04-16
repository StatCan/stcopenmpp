// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/text/language"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// run the model identified by model digest-or-name with specified run options.
//
//	POST /api/run
//
// Json RunRequest structure is posted to specify model digest-or-name, run stamp and othe run options.
// If multiple models with same name exist then result is undefined.
// Model run console output redirected to log file: models/log/modelName.runStamp.console.log
func runModelHandler(w http.ResponseWriter, r *http.Request) {

	// decode json request body
	var req RunRequest
	if !jsonRequestDecode(w, r, true, &req) {
		return // error at json decode, response done with http error
	}
	if req.Opts == nil {
		req.Opts = map[string]string{}
	}
	if req.Env == nil {
		req.Env = map[string]string{}
	}

	// if log messages language not specified then use browser preferred language
	lang, ok := req.Opts["OpenM.MessageLanguage"]

	if !ok {
		if rqLangTags, _, e := language.ParseAcceptLanguage(r.Header.Get("Accept-Language")); e == nil {
			if len(rqLangTags) > 0 && rqLangTags[0] != language.Und {
				lang = rqLangTags[0].String()
			}
		}
	}
	if lang != "" {
		req.Opts["OpenM.MessageLanguage"] = lang // model run log language
	}

	// block model run if disk space usage exceed the limits
	if isOver, _ := theRunCatalog.getDiskUseStatus(); isOver {
		http.Error(w, helper.MsgL(lang, "Disk space usage exceeds quota, model run disabled"), http.StatusBadRequest)
		return
	}

	// find model metadata by digest or name
	dn := req.ModelDigest
	if dn == "" {
		dn = req.ModelName
	}
	m, ok := theCatalog.ModelDicByDigestOrName(dn)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Model not found:", dn), http.StatusBadRequest)
		return // empty result: model digest not found
	}
	req.ModelDigest = m.Digest
	req.ModelName = m.Name

	// adjust MPI options:
	// IsMpi is the same as number of processes > 0
	if req.Mpi.Np < 0 {
		req.Mpi.Np = 0
	}
	if req.Mpi.Np > 0 {
		req.IsMpi = true
	}
	if req.IsMpi && req.Mpi.Np <= 0 {
		req.Mpi.Np = 1
	}
	if req.IsMpi && !theCfg.isJobControl {
		req.Mpi.IsNotByJob = true // if job control disabled then model run cannot use job control
	}

	// get submit stamp
	submitStamp, tNow := theCatalog.getNewTimeStamp()

	job := RunJob{
		SubmitStamp: submitStamp,
		RunRequest:  req,
	}

	// get number of modelling cpu
	// for backward compatibility: check if number of threads specified using run options
	job.Res, job.Mpi.IsNotOnRoot, ok = resFromRequest(req)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Model start failed:", dn), http.StatusBadRequest)
		return
	}
	job.Threads = job.Res.ThreadCount

	// if job control disabled then start model run
	if !theCfg.isJobControl {

		rs, err := theRunCatalog.runModel(&job, "", hostIni{}, []computeUse{}) // no job control: use empty arguments
		if err != nil {
			omppLog.LogNoLT(err)
			http.Error(w, helper.MsgL(lang, "Model start failed:", dn), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Location", "/api/model/"+job.ModelDigest+"/run/"+rs.RunStamp)
		jsonResponse(w, r, rs)
		return
	}
	// else append run request to the queue and return submit stamp

	_, err := theRunCatalog.addJobToQueue(&job)
	if err != nil {
		http.Error(w, helper.MsgL(lang, "Model run submission failed:", dn), http.StatusBadRequest)
		return
	}
	rStamp := helper.CleanFileName(job.RunStamp)

	w.Header().Set("Content-Location", "/api/model/"+job.ModelDigest+"/run/"+rStamp)
	jsonResponse(w, r,
		&RunState{
			ModelName:      job.ModelName,
			ModelDigest:    job.ModelDigest,
			RunStamp:       rStamp,
			SubmitStamp:    submitStamp,
			UpdateDateTime: helper.MakeDateTime(tNow),
		})
}

// return cpu modelling count, MPI not-on-root flag and error flag
func resFromRequest(req RunRequest) (RunRes, bool, bool) {

	// get number of threads and MPI NotOnRoot flag
	nTh := req.Threads
	isNotOnRoot := req.Mpi.IsNotOnRoot

	for krq, val := range req.Opts {

		// backward compatibility: check if number of threads specified using run options
		var err error

		if req.Threads <= 1 &&
			(strings.EqualFold(krq, "-OpenM.Threads") || strings.EqualFold(krq, "OpenM.Threads")) {

			nTh, err = strconv.Atoi(val) // must be >= 1
			if err != nil || nTh < 1 {
				omppLog.LogNoLT(err)
				return RunRes{}, false, false
			}
		}

		// backward compatibility: check MPI "not on root" specified using run options
		if !isNotOnRoot {

			// get MPI "not on root" flag: do not run modelling on root MPI process
			if strings.EqualFold(krq, "-OpenM.NotOnRoot") || strings.EqualFold(krq, "OpenM.NotOnRoot") {

				if val == "" {
					isNotOnRoot = true // empty boolean option value treated as true
				}
				isNotOnRoot, err = strconv.ParseBool(val)
				if err != nil {
					omppLog.LogNoLT(err)
					return RunRes{}, false, false
				}
			}
		}
	}

	// number of modelling processes, requested cpu count and memory
	nProc := req.Mpi.Np
	if nProc <= 0 {
		nProc = 1
	}
	np := nProc
	if np > 1 && isNotOnRoot {
		np--
	}
	if nTh <= 0 {
		nTh = 1
	}
	cfgRes := theRunCatalog.getCfgRes(req.ModelDigest)
	nMem := memoryRunSize(np, nTh, cfgRes.ProcessMemMb, cfgRes.ThreadMemMb)

	res := RunRes{
		ComputeRes: ComputeRes{
			Cpu: np * nTh,
			Mem: nMem,
		},
		ProcessCount: np,
		ThreadCount:  nTh,
		ProcessMemMb: cfgRes.ProcessMemMb,
		ThreadMemMb:  cfgRes.ThreadMemMb,
	}

	return res, isNotOnRoot, true
}

// kill model run by model digest-or-name and run stamp or remove model run request from queue by submit stamp.
//
//	PUT /api/run/stop/model/:model/stamp/:stamp
//
// If multiple models with same name exist then result is undefined.
func stopModelHandler(w http.ResponseWriter, r *http.Request) {

	// url or query parameters:  model digest-or-name, page offset and page size
	dn := getRequestParam(r, "model")
	stamp := getRequestParam(r, "stamp")
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// find model metadata by digest or name
	m, ok := theCatalog.ModelDicByDigestOrName(dn)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Model not found:", dn), http.StatusBadRequest)
		return // empty result: model digest not found
	}
	modelDigest := m.Digest

	// kill model run by run stamp or
	// remove run request from the queue by submit stamp or by run stamp
	isFound, submitStamp, jobPath, isRunning := theRunCatalog.stopModelRun(modelDigest, stamp)

	if !isRunning {
		moveJobQueueToFailed(jobPath, submitStamp, m.Name, m.Digest, stamp, true) // model was not running, move job control file to history
	}

	// write model run key as response
	w.Header().Set("Content-Location", "/api/model/"+modelDigest+"/run/"+stamp+"/"+strconv.FormatBool(isFound))
	w.Header().Set("Content-Type", "text/plain")
}

// return model run status and log by model digest-or-name and run-or-submit stamp.
//
//	GET /api/run/log/model/:model/stamp/:stamp
//	GET /api/run/log/model/:model/stamp/:stamp/start/:start/count/:count
//
// If multiple models with same name exist then result is undefined.
// Model run log is same as console output and include stdout and stderr.
// Run log can be returned by page defined by zero-based "start" line and line count.
// If count <= 0 then all log lines until eof returned, complete current log: start=0, count=0
func runLogPageHandler(w http.ResponseWriter, r *http.Request) {

	// url or query parameters: model digest-or-name, page offset and page size
	dn := getRequestParam(r, "model")
	stamp := getRequestParam(r, "stamp")
	lang := preferedRequestLang(r, "") // get prefered language for messages

	start, ok := getIntRequestParam(r, "start", 0)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid value of start log start line", dn), http.StatusBadRequest)
		return
	}
	count, ok := getIntRequestParam(r, "count", 0)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid value of log line count", dn), http.StatusBadRequest)
		return
	}

	// find model metadata by digest or name
	m, ok := theCatalog.ModelDicByDigestOrName(dn)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Model not found:", dn), http.StatusBadRequest)
		return // empty result: model digest not found
	}
	modelDigest := m.Digest
	modelName := m.Name

	// get current run status and page of log lines
	lrp, e := theRunCatalog.readModelRunLog(modelDigest, stamp, start, count)
	if e != nil {
		omppLog.LogNoLT(e)
		http.Error(w, helper.MsgL(lang, "Model run status read failed:", modelName, ":", dn), http.StatusBadRequest)
		return
	}

	// return model run status and log content
	jsonResponse(w, r, lrp)
}
