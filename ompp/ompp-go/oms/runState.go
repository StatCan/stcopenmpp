// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
)

const modelRunsScanInterval = 4021 // timeout in msec, sleep interval between scanning run list in database

// find model run state by model digest and submission stamp, if not found then return false and empty RunState
func (rsc *RunCatalog) getRunStateBySubmitStamp(digest, submitStamp string) (bool, RunState) {
	if digest == "" || submitStamp == "" {
		return false, RunState{}
	}
	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	// find model run state by submit stamp
	ml, isFound := rsc.modelRuns[digest]
	if !isFound {
		return false, RunState{} // model digest not found
	}

	var rsl *runStateLog
	for _, r := range ml {

		if r.SubmitStamp == submitStamp {
			rsl = r
			break
		}
	}
	if rsl == nil {
		return false, RunState{} // submit stamp not found or invalid run state
	}

	return true, rsl.RunState
}

// find runStateLog by model digest and run stamp or submit stamp
// internal: use only inside of lock
func (rsc *RunCatalog) findRunStateLog(modelDigest, stamp string) *runStateLog {

	if modelDigest == "" || stamp == "" { // empty key: return not found
		return nil
	}
	ml, isFound := rsc.modelRuns[modelDigest]
	if !isFound {
		return nil // model digest not found
	}

	rsl, isFound := ml[stamp]
	if !isFound { // if run stamp not found then search by submit stamp

		for _, r := range ml {

			if isFound = r.SubmitStamp == stamp; isFound { // check if it is a submit stamp, not a run stamp
				rsl = r
				break
			}
		}
	}
	if !isFound || rsl == nil {
		return nil // not found by run stamp or submit stamp: return empty result
	}
	return rsl
}

// add new model run state and create model run log file
func (rsc *RunCatalog) createRunStateLog(rState *RunState) {
	if rState == nil || rState.ModelDigest == "" || rState.RunStamp == "" {
		return // invalid run state
	}

	// create log file or truncate existing
	if rState.IsLog {
		f, err := os.Create(rState.logPath)
		if err != nil {
			rState.IsLog = false
			return
		}
		defer f.Close()
	}

	// add new entry to run state log list
	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	rsl := rsc.newIfNotExistRunStateLog(rState)
	if rsl != nil {
		rsl.RunState = *rState // update run state pid, kill channel and cmdPath
	}
}

// create new or return existing runStateLog by model digest and run stamp
// internal: use only inside of lock
func (rsc *RunCatalog) newIfNotExistRunStateLog(rState *RunState) *runStateLog {

	if rState == nil || rState.ModelDigest == "" || rState.RunStamp == "" {
		return nil // invalid run state: return empty result
	}

	// find model run state by model digest and run stamp
	_, isFound := rsc.modelRuns[rState.ModelDigest]
	if !isFound {
		rsc.modelRuns[rState.ModelDigest] = map[string]*runStateLog{}
	}

	rsl := rsc.modelRuns[rState.ModelDigest][rState.RunStamp]
	if rsl == nil {
		rsl = &runStateLog{
			RunState:   *rState,
			logUsedTs:  time.Now().Unix(),
			logLineLst: make([]string, 0, 128),
		}
		rsc.modelRuns[rState.ModelDigest][rState.RunStamp] = rsl
	}

	return rsl
}

// updateRunStateProcess set process info if isFinal is false or clear it if isFinal is true
func (rsc *RunCatalog) updateRunStateProcess(rState *RunState, isFinal bool) {
	if rState == nil || rState.ModelDigest == "" || rState.RunStamp == "" {
		return // invalid run state
	}
	tNow := time.Now()

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	// find model run state by digest and run stamp
	rsl := rsc.newIfNotExistRunStateLog(rState)
	if rsl == nil {
		return // invalid run state
	}

	// update run state and set or clear process info
	rsl.UpdateDateTime = helper.MakeDateTime(tNow)
	rsl.IsFinal = isFinal
	rsl.SubmitStamp = rState.SubmitStamp
	rsl.TaskRunName = rState.TaskRunName

	if isFinal {
		rsl.killC = nil
	} else {
		rsl.pid = rState.pid
		rsl.killC = rState.killC
	}
	if rState.cmdPath != "" {
		rsl.cmdPath = rState.cmdPath
	}
}

// updateRunStateLog does model run state update and append to model log lines array
func (rsc *RunCatalog) updateRunStateLog(rState *RunState, isFinal bool, msg string) {
	if rState == nil || rState.ModelDigest == "" || rState.RunStamp == "" {
		return // invalid run state
	}
	tNow := time.Now()

	// write into model console log file
	if rState.IsLog && msg != "" {

		f, err := os.OpenFile(rState.logPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			rState.IsLog = false
		} else {
			defer f.Close()

			_, err = f.WriteString(msg)
			if err == nil {
				if runtime.GOOS == "windows" { // adjust newline for windows
					_, err = f.WriteString("\r\n")
				} else {
					_, err = f.WriteString("\n")
				}
			}
			if err != nil {
				rState.IsLog = false
			}
		}
	}

	// update run state list
	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	// find model run state by digest and run stamp
	rsl := rsc.newIfNotExistRunStateLog(rState)
	if rsl == nil {
		return // invalid run state
	}

	// update run state and append new log line if not empty
	rsl.UpdateDateTime = helper.MakeDateTime(tNow)
	rsl.IsFinal = isFinal
	rsl.SubmitStamp = rState.SubmitStamp
	rsl.TaskRunName = rState.TaskRunName

	if isFinal {
		rsl.killC = nil
		rsl.isKill = rsl.isKill || rState.isKill
	} else {
		rsl.pid = rState.pid
		rsl.killC = rState.killC
	}
	if rState.cmdPath != "" {
		rsl.cmdPath = rState.cmdPath
	}

	rsl.IsLog = rState.IsLog // it can be updated by write log error
	if msg != "" {
		rsl.logUsedTs = tNow.Unix()
		rsl.logLineLst = append(rsl.logLineLst, msg)
	}
}

// scan model run list in database and model run log files and update model run list
func scanModelRuns(doneC <-chan bool) {

	for {
		// update model run config: run templates and run presets
		theRunCatalog.updateEtcModelConfig(theCfg.etcDir)

		// get current models from run catalog and main catalog
		rbs := theRunCatalog.allModels()
		dLst := theCatalog.allModelDigests()
		sort.Strings(dLst)

		// for each model read from database run_lst rows and insert or update model runs list
		for dgst, it := range rbs {

			// skip model if digest does not exist in main catalog, ie: catalog closed
			if i := sort.SearchStrings(dLst, dgst); i < 0 || i >= len(dLst) || dLst[i] != dgst {
				continue
			}

			// get list of the model runs
			rl, ok := theCatalog.RunRowListByModel(dgst)
			if !ok || len(rl) <= 0 {
				continue // no model runs (or ignore get runs error)
			}

			// append new runs to model run list
			rsLst := make([]RunState, len(rl))

			for k := range rl {

				rsLst[k] = RunState{
					ModelName:      it.name,
					ModelDigest:    dgst,
					RunStamp:       rl[k].RunStamp,
					IsFinal:        db.IsRunCompleted(rl[k].Status),
					UpdateDateTime: rl[k].UpdateDateTime,
					RunName:        rl[k].Name,
				}
				if helper.IsUnderscoreTimeStamp(rl[k].RunStamp) {
					rsLst[k].SubmitStamp = rl[k].RunStamp
				} else {
					rsLst[k].SubmitStamp = helper.ToUnderscoreTimeStamp(rl[k].CreateDateTime)
				}
			}

			// model log directory path must be not empty to search for log files
			if !it.isLogDir {
				continue
			}
			if it.logDir == "" {
				it.logDir = "." // assume current directory if log directory not specified but eanbled by isLogDir
			}
			it.logDir = filepath.ToSlash(it.logDir) // use / path separator

			// get list of model run log files
			ptrn := it.logDir + "/" + it.name + "." + "*.log"
			fLst := filesByPattern(ptrn, "Error at log files search")

			// replace path separators by / and sort file paths list
			for k := range fLst {
				fLst[k] = filepath.ToSlash(fLst[k])
			}
			sort.Strings(fLst) // sort by file path

			// search new model runs to find log file path
			// append run state to run state list ordered by run id
			for k := 0; k < len(rsLst); k++ {

				// search for modelName.runStamp.console.log
				fn := it.logDir + "/" + it.name + "." + rsLst[k].RunStamp + ".console.log"
				n := sort.SearchStrings(fLst, fn)
				isFound := n < len(fLst) && fLst[n] == fn

				// search for modelName.runStamp.log
				if !isFound {
					fn = it.logDir + "/" + it.name + "." + rsLst[k].RunStamp + ".log"
					n = sort.SearchStrings(fLst, fn)
					isFound = n < len(fLst) && fLst[n] == fn
				}

				if isFound {
					rsLst[k].IsLog = true
					rsLst[k].LogFileName = filepath.Base(fn)
					rsLst[k].logPath = fn
				}
			}

			// add new log file names to model log list and update most recent run id for that model
			theRunCatalog.updateModelRuns(dgst, rsLst)
		}

		// clear old entries entries in run state list
		theRunCatalog.clearModelRunsLog()

		// wait for doneC or sleep
		if isExitSleep(modelRunsScanInterval, doneC) {
			return
		}
	}
}

// update model runs list with db run rows data:
// add new db run state to model runs list if not exist
// if run already exist then replace run state with more recent db run data
// if existing run not in current db run list then remove it if state is final and kill channel is empty nil
func (rsc *RunCatalog) updateModelRuns(digest string, runStateLst []RunState) {
	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	// add new runs to model run history
	nowTs := time.Now().Unix()

	for _, r := range runStateLst {

		if r.RunStamp == "" { // skip: run stamp should not be empty
			continue
		}

		if ml, ok := rsc.modelRuns[digest]; ok { // skip model if not exist

			if rsl, isLog := ml[r.RunStamp]; !isLog { // if run state not exist then insert new run state

				ml[r.RunStamp] = &runStateLog{RunState: r, logUsedTs: nowTs, logLineLst: []string{}}

			} else { // if run state already exist then update existing values

				if rsl.SubmitStamp == "" {
					rsl.SubmitStamp = r.SubmitStamp
				}
				if !rsl.IsFinal {
					rsl.IsFinal = r.IsFinal
				}
				if rsl.UpdateDateTime < r.UpdateDateTime {
					rsl.UpdateDateTime = r.UpdateDateTime
				}
				if rsl.RunName == "" {
					rsl.RunName = r.RunName
				}
				if rsl.TaskRunName == "" {
					rsl.TaskRunName = r.TaskRunName
				}
				if !rsl.IsLog {
					rsl.IsLog = r.IsLog
				}
				if rsl.LogFileName == "" {
					rsl.LogFileName = r.LogFileName
				}
				if rsl.logPath == "" {
					rsl.logPath = r.logPath
				}
			}
		}
	}

	// remove runs from the list if db run not exist and status is final and kill channel is empty nil
	for stamp, r := range rsc.modelRuns[digest] {

		if !r.IsFinal || r.killC != nil { // skip: model still running
			continue
		}

		// check if this run stamp exist in run_lst
		isExist := false
		for k := 0; !isExist && k < len(runStateLst); k++ {
			isExist = runStateLst[k].RunStamp == stamp
		}
		if !isExist {
			delete(rsc.modelRuns[digest], stamp) // model run not forund in database and model is not running
		}
	}
}

// remove old log file lines from run state list
func (rsc *RunCatalog) clearModelRunsLog() {
	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	dt := time.Now().AddDate(0, 0, -1).Unix()

	for _, ml := range rsc.modelRuns {
		for _, r := range ml {
			if r.logUsedTs < dt && len(r.logLineLst) > 0 {
				r.logLineLst = []string{} // clear log lines if content was not used recently
			}
		}
	}
}
