// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"

	ps "github.com/keybase/go-ps"
)

const jobScanInterval = 1123                    // timeout in msec, sleep interval between scanning all job directories
const computeStartStopInterval = 3373           // timeout in msec, interval between start or stop computational servers, must be at least 2 * jobScanInterval
const jobQueueScanInterval = 107                // timeout in msec, sleep interval between getting next job from the queue
const jobOuterScanInterval = 5021               // timeout in msec, sleep interval between scanning active job directory
const serverTimeoutDefault = 60                 // time in seconds to start or stop compute server
const minJobTickMs int64 = 1597707959000        // unix milliseconds of 2020-08-17 23:45:59
const minCmdTimeSec int64 = minJobTickMs / 1000 // unix seconds of 2020-08-17 23:45:59
const jobPositionDefault = 20220817             // queue job position by default, e.g. if queue is empty
const maxComputeErrorsDefault = 8               // default errors threshold for compute server or cluster

/*
scan active job directory to find active model run files without run state.
It can be a result of oms restart or server reboot.

if active job file found and no run state then
create run job from active file
add it to the list of "outer" jobs (active jobs without run state)

for each job in the outer list

	find model process by pid and executable name
	if process exist then wait until it done
	check if file still exist
	read run_lst row
	if no run_lst row then move job file to history as error
	else
	  if run state is not completed then update run state as error
	  and move file to history according to status
*/
func scanOuterJobs(doneC <-chan bool) {
	if !theCfg.isJobControl {
		return // job control disabled
	}

	// map active job file path to file content (run job), it is only job where no run state in RunCatalog
	outerJobs := map[string]RunJob{}

	activeDir := filepath.Join(theCfg.jobDir, "active")
	nActive := len(activeDir)
	ptrn := activeDir + string(filepath.Separator) + "*-#-" + theCfg.omsName + "-#-*.json"

	for {
		// find active job files
		fLst := filesByPattern(ptrn, "Error at active job files search")
		if len(fLst) <= 0 {
			if isExitSleep(jobOuterScanInterval, doneC) {
				return
			}
			continue // no active jobs
		}

		// find new active jobs since last scan which do not exist in run state list of RunCatalog
		for k := range fLst {

			if _, ok := outerJobs[fLst[k]]; ok {
				continue // this file already in the outer jobs list
			}

			// get submission stamp, model name, digest and process id from active job file name
			stamp, _, mn, dgst, _, _, _, _, pid := parseActivePath(fLst[k][nActive+1:])
			if stamp == "" || mn == "" || dgst == "" || pid <= 0 {
				continue // file name is not an active job file name
			}

			// find run state by model digest and submission stamp
			isFound, _ := theRunCatalog.getRunStateBySubmitStamp(dgst, stamp)
			if isFound {
				continue // this is an active job under oms control
			}

			// run state not found: create run state from active job file
			var jc RunJob
			isOk, err := helper.FromJsonFile(fLst[k], &jc)
			if err != nil {
				omppLog.LogNoLT(err)
			}
			if !isOk || err != nil {
				moveActiveJobToHistory(fLst[k], "", false, stamp, mn, dgst, "no-model-run-time-stamp", 0, 0) // invalid file content: move to history with unknown status
				continue
			}

			// add job into outer jobs list
			outerJobs[fLst[k]] = jc
		}

		// for outer jobs find process by pid and executable name
		// if process completed then move job file into the history
		for fp, jc := range outerJobs {

			proc, err := ps.FindProcess(jc.Pid)

			if err == nil && proc != nil &&
				strings.HasSuffix(strings.ToLower(jc.CmdPath), strings.ToLower(proc.Executable())) {
				continue // model still running
			}

			// check if job file not exist then remove it from the outer job list
			if !helper.IsFileExist(fp) {
				delete(outerJobs, fp)
				continue
			}

			// get run_lst row and move to job history according to status
			// model process does not exist, run status must completed: s=success, x=exit, e=error
			// if model status is not completed then it is an error
			var rStat string
			rp, ok := theCatalog.RunStatus(jc.ModelDigest, jc.RunStamp)
			if ok && rp != nil {
				rStat = rp.Status
				if !db.IsRunCompleted(rStat) {
					rStat = db.ErrorRunStatus
					_, e := theCatalog.UpdateRunStatus(jc.ModelDigest, jc.RunStamp, db.ErrorRunStatus)
					if e != nil {
						omppLog.LogNoLT(e)
					}
				}
			}
			moveActiveJobToHistory(fp, rStat, false, jc.SubmitStamp, jc.ModelName, jc.ModelDigest, jc.RunStamp, 0, 0)
			delete(outerJobs, fp)
		}

		// wait for doneC or sleep
		if isExitSleep(jobOuterScanInterval, doneC) {
			return
		}
	}
}

// scan model run queue, start model runs, start or stop MPI servers
func scanRunJobs(doneC <-chan bool) {
	if !theCfg.isJobControl {
		return // job control disabled: no queue
	}

	lastStartStopTs := time.Now().UnixMilli() // last time when start and stop of computational servers done

	for {
		// get job from the queue and run
		if isFound, job, qPath, hf, compHostUse, e := theRunCatalog.selectJobFromQueue(); e == nil {

			if isFound {
				_, e = theRunCatalog.runModel(job, qPath, hf, compHostUse)
				if e != nil {
					omppLog.LogNoLT(e)
				}
			}
		} else { // error at select job from queue: there is a problem with that job, remove it from the queue

			omppLog.LogNoLT(e)
			if qPath != "" {
				moveJobQueueToFailed(qPath, job.SubmitStamp, job.ModelName, job.ModelDigest, "", false) // can not run this job: remove from the queue
			}
		}

		// check if this is a time to do start or stop computational servers:
		// interval must be at least double of job files scan to make sure server state files updated
		if lastStartStopTs+computeStartStopInterval < time.Now().UnixMilli() {

			// start computational servers or clusters
			startNames, maxStartTime, startExes, startArgs := theRunCatalog.selectToStartCompute()

			for k := range startNames {
				if startExes[k] != "" {
					go doStartStopCompute(startNames[k], "start", startExes[k], startArgs[k], maxStartTime)
				} else {
					doStartOnceCompute(startNames[k]) // special case: server always ready
				}
			}

			// stop computational servers or clusters
			stopNames, maxStopTime, stopExes, stopArgs := theRunCatalog.selectToStopCompute()

			for k := range stopNames {
				if stopExes[k] != "" {
					go doStartStopCompute(stopNames[k], "stop", stopExes[k], stopArgs[k], maxStopTime)
				} else {
					doStopCleanupCompute(stopNames[k]) // special case: server never stop
				}
			}

			lastStartStopTs = time.Now().UnixMilli()
		}

		// wait for doneC or sleep
		if isExitSleep(jobQueueScanInterval, doneC) {
			return
		}
	}
}

// start (or stop) computational server or cluster and create (or delete) ready state file
func doStartStopCompute(name, state, exe string, args []string, maxTime int64) {

	omppLog.LogNoLT(state, ":", name)

	// create server start or stop file to signal server state
	sf := createCompStateFile(name, state)
	if sf == "" {
		omppLog.Log("FAILED to create state file:", state, " ", name)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(maxTime)*time.Millisecond)
	defer cancel()

	// make a command, run it and return combined output
	out, err := exec.CommandContext(ctx, exe, args...).CombinedOutput()
	if len(out) > 0 {
		omppLog.LogNoLT(string(out))
	}

	// create or delete server ready file and delete state file
	isOk := false
	if err == nil {

		readyPath := compReadyPath(name)
		if state == "start" {
			isOk = fileCreateEmpty(false, readyPath)
			if !isOk {
				omppLog.Log("FAILED to create server ready file:", readyPath)
			}
		} else {
			isOk = fileDeleteAndLog(false, readyPath)
			if !isOk {
				omppLog.Log("FAILED to delete server ready file:", readyPath)
			}
		}

		if isOk {
			okStart := deleteCompStateFiles(name, "start")
			okStop := deleteCompStateFiles(name, "stop")
			okErr := deleteCompStateFiles(name, "error")
			isOk = okStart && okStop && okErr
		}
		if isOk {
			omppLog.Log("Done:", state, " ", name)
		}

	} else {
		if createCompStateFile(name, "error") == "" {
			omppLog.Log("FAILED to create error state file:", name)
		}
		if ctx.Err() == context.DeadlineExceeded {
			omppLog.Log("ERROR server timeout:", state, " ", name)
		}
		omppLog.Log("Error:", err)
		omppLog.Log("FAILED:", state, " ", name)
	}

	// remove this server from startup or shutdown list
	if state == "start" {
		theRunCatalog.startupCompleted(isOk, name)
	} else {
		theRunCatalog.shutdownCompleted(isOk, name)
	}
}

// start computational server, special case: if server always ready then only create ready state file
func doStartOnceCompute(name string) {

	omppLog.Log("Start:", name)

	readyPath := compReadyPath(name)
	isOk := fileCreateEmpty(false, readyPath)
	if !isOk {
		omppLog.Log("FAILED to create server ready file:", readyPath)
	}

	if isOk {
		okStart := deleteCompStateFiles(name, "start")
		okStop := deleteCompStateFiles(name, "stop")
		okErr := deleteCompStateFiles(name, "error")
		isOk = okStart && okStop && okErr
	}
	if isOk {
		omppLog.Log("Done start of:", name)
	}
}

// stop computational server, special case: if server never stop then only cleanup state files
func doStopCleanupCompute(name string) {

	deleteCompStateFiles(name, "start")
	deleteCompStateFiles(name, "stop")
	deleteCompStateFiles(name, "error")
}
