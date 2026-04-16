// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"errors"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// Get model run job from the queue.
// Return job to run, bool job found flag, queue job file path, list of server names to run the job, list of resources to use on each server
func (rsc *RunCatalog) selectJobFromQueue() (bool, *RunJob, string, hostIni, []computeUse, error) {

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	if rsc.IsQueuePaused || len(rsc.queueKeys) <= 0 {
		return false, nil, "", hostIni{}, []computeUse{}, nil // queue is paused or empty
	}

	// resource available to run MPI jobs from global MPI queue or from localhost queue of this oms instance jobs
	qLocal := ComputeRes{
		Cpu: rsc.LocalRes.Cpu - rsc.LocalActiveRes.Cpu,
		Mem: rsc.LocalRes.Mem - rsc.LocalActiveRes.Mem,
	}
	compUse := []computeUse{}

	// find first job in localhost non-MPI queue or global MPI queue
	// check if there are enough available resources to run the job
	stamp := ""

	for _, qKey := range rsc.queueKeys {

		jc, ok := rsc.queueJobs[qKey]

		if !ok || jc.isError || jc.IsOverLimit || jc.isPaused || !jc.isFirst {
			continue // skip invalid job, paused job, job resources exceeding limits or it is not the first job in queue
		}

		isSel := false
		for j := 0; !isSel && j < len(rsc.selectedKeys); j++ {
			isSel = rsc.selectedKeys[j] == qKey
		}
		if isSel {
			continue // this job is already selected to run
		}

		// check avaliable resource: cpu cores and memory
		if !jc.IsMpi {

			if rsc.LocalRes.Cpu > 0 && jc.Res.Cpu > 0 && qLocal.Cpu < jc.Res.Cpu {
				continue // localhost cpu cores limited and job cpu limit set to non zero and localhost available cpu less than required cpu
			}
			if rsc.LocalRes.Mem > 0 && jc.Res.Mem > 0 && qLocal.Mem < jc.Res.Mem {
				continue // localhost memory limited and job memory limit set to non zero and localhost available memory less than required memory
			}

			stamp = qKey // localhost first job and enough resources available to run
			break
		}
		// else: it is MPI first job

		if len(rsc.computeState) <= 0 { // no computational servers, use localhost to run MPI jobs

			isMpiLimit := rsc.LocalRes.Cpu > 0 // MPI cores limited if localhost CPU is limited

			if isMpiLimit && jc.Res.Cpu > 0 && qLocal.Cpu < jc.Res.Cpu {
				continue // job cpu limit set to non zero and localhost cpu less than required cpu
			}
			if isMpiLimit && rsc.LocalRes.Mem > 0 && jc.Res.Mem > 0 && qLocal.Mem < jc.Res.Mem {
				continue // job memory limit is set to non zero and localhost available memory less than required memory
			}

			stamp = qKey // localhost first MPI job and enough resources available to run
			break
		}
		// else: it is MPI first job to run on cluster

		// check if all servers in host ini are ready
		if len(rsc.first.hostUse) <= 0 || rsc.first.res.ThreadCount <= 0 || jc.Res.ThreadCount <= 0 {

			jc.isError = true
			rsc.queueJobs[qKey] = jc
			e := errors.New("ERROR: computational resources not found to run the model: " +
				strconv.Itoa(jc.Res.Cpu) + ": " + strconv.Itoa(jc.Res.Mem) + ": " + strconv.Itoa(rsc.first.res.ThreadCount) + ": " + qKey)
			return false, &jc.RunJob, jc.filePath, hostIni{}, []computeUse{}, e
		}

		for _, hcu := range rsc.first.hostUse {

			cs, ok := rsc.computeState[hcu.CompName]
			if !ok || cs.State != "ready" {
				return false, nil, "", hostIni{}, []computeUse{}, nil // server is not ready, return to wait
			}
		}
		compUse = append([]computeUse{}, rsc.first.hostUse...) // all srevers are ready to use

		stamp = qKey // first MPI job in queue

		jc.Res = rsc.first.res // actual resources
		rsc.queueJobs[qKey] = jc
		break
	}
	if stamp == "" {
		return false, nil, "", hostIni{}, []computeUse{}, nil // queue is empty or all jobs already selected to run
	}
	qj := rsc.queueJobs[stamp] // job found

	// job found: copy run request from the queue
	rsc.selectedKeys = append(rsc.selectedKeys, stamp)

	job := qj.RunJob

	job.Opts = make(map[string]string, len(qj.Opts))
	for key, val := range qj.Opts {
		job.Opts[key] = val
	}

	job.Env = make(map[string]string, len(qj.Env))
	for key, val := range qj.Env {
		job.Env[key] = val
	}

	job.Tables = make([]string, len(qj.Tables))
	copy(job.Tables, qj.Tables)

	job.RunNotes = make(
		[]struct {
			LangCode string // model language code
			Note     string // run notes
		},
		len(qj.RunNotes))
	copy(job.RunNotes, qj.RunNotes)

	return true, &job, qj.filePath, rsc.hostFile, compUse, nil
}

// Return copy of submission stamps and job control items for queue, active and history model run jobs
func (rsc *RunCatalog) getRunJobs() ([]string, []RunJob, []string, []RunJob, []string, []historyJobFile) {

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	// jobs queue: sort in order of submission stamps, which user may change through UI
	qKeys := make([]string, len(rsc.queueKeys))
	qJobs := make([]RunJob, len(rsc.queueKeys))
	for k, stamp := range rsc.queueKeys {
		qKeys[k] = stamp
		qJobs[k] = rsc.queueJobs[stamp].RunJob
	}

	// active jobs: sort by submission time
	aKeys := make([]string, len(rsc.activeJobs))
	n := 0
	for stamp := range rsc.activeJobs {
		aKeys[n] = stamp
		n++
	}
	sort.Strings(aKeys)

	aJobs := make([]RunJob, len(aKeys))
	for k, stamp := range aKeys {
		aJobs[k] = rsc.activeJobs[stamp].RunJob
	}

	// history jobs: sort by submission time
	hKeys := make([]string, len(rsc.historyJobs))
	n = 0
	for stamp := range rsc.historyJobs {
		hKeys[n] = stamp
		n++
	}
	sort.Strings(hKeys)

	hJobs := make([]historyJobFile, len(hKeys))
	for k, stamp := range hKeys {
		hJobs[k] = rsc.historyJobs[stamp]
	}

	return qKeys, qJobs, aKeys, aJobs, hKeys, hJobs
}

// Return active job control item and is found boolean flag
func (rsc *RunCatalog) getActiveJobItem(submitStamp string) (runJobFile, bool) {

	if submitStamp == "" {
		return runJobFile{}, false // empty job submission stamp: return empty result
	}

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	if aj, ok := rsc.activeJobs[submitStamp]; ok {
		return aj, true
	}
	return runJobFile{}, false // not found
}

// Return queue job control item and is found boolean flag
func (rsc *RunCatalog) getQueueJobItem(submitStamp string) (queueJobFile, bool) {

	if submitStamp == "" {
		return queueJobFile{}, false // empty job submission stamp: return empty result
	}

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	if qj, ok := rsc.queueJobs[submitStamp]; ok {
		return qj, true
	}
	return queueJobFile{}, false // not found
}

// Return history job control item and is found boolean flag
func (rsc *RunCatalog) getHistoryJobItem(submitStamp string) (historyJobFile, bool) {

	if submitStamp == "" {
		return historyJobFile{}, false // empty job submission stamp: return empty result
	}

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	if hj, ok := rsc.historyJobs[submitStamp]; ok {
		return hj, true
	}
	return historyJobFile{}, false // not found
}

// write new run request into job queue file, return queue job file path
func (rsc *RunCatalog) addJobToQueue(job *RunJob) (string, error) {
	if !theCfg.isJobControl {
		return "", nil // job control disabled
	}

	fp := jobQueuePath(
		job.SubmitStamp, job.ModelName, job.ModelDigest, job.IsMpi, rsc.nextJobPosition(), job.Res.ProcessCount, job.Res.ThreadCount, job.Res.ProcessMemMb, job.Res.ThreadMemMb,
	)

	err := helper.ToJsonIndentFile(fp, job)
	if err != nil {
		omppLog.Log(err)
		fileDeleteAndLog(true, fp) // on error remove file, if any file created
		return "", err
	}

	return "", nil
}

// Return next job position in the queue, it is not a queue index but "ticket number" to establish queue jobs order
func (rsc *RunCatalog) nextJobPosition() int {
	if !theCfg.isJobControl {
		return 0 // job control disabled
	}

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	rsc.jobLastPosition++

	if rsc.jobLastPosition <= jobPositionDefault {
		rsc.jobLastPosition = jobPositionDefault + 1
	}
	return rsc.jobLastPosition
}

// move job into the specified queue index position.
// Top of the queue position is zero, negative position treated as zero.
// If position number exceeds queue length then job moved to the bottom of the queue.
// Return false if job not found in the queue
func (rsc *RunCatalog) moveJobInQueue(submitStamp string, index int) (bool, [][2]string) {

	if submitStamp == "" {
		return false, [][2]string{} // empty job submission stamp: return empty result
	}

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	// find current job position in the queue, excluding jobs selected to run
	isFound := false
	n := 0
	for n = range rsc.queueKeys {
		isFound = rsc.queueKeys[n] == submitStamp
		if isFound {
			break
		}
	}
	if isFound {
		isSel := false
		for j := 0; !isSel && j < len(rsc.selectedKeys); j++ {
			isSel = rsc.selectedKeys[j] == submitStamp
		}
		isFound = !isSel
	}
	if !isFound {
		return false, [][2]string{} // job not found in the queue
	}

	// position must be between zero at the last position in the queue
	nPos := index
	fPos := jobPositionDefault

	isFirst := nPos <= 0
	if isFirst {
		nPos = 0
		rsc.jobFirstPosition--
		fPos = rsc.jobFirstPosition
	}

	isLast := !isFirst && nPos >= len(rsc.queueKeys)-1
	if isLast {
		nPos = len(rsc.queueKeys) - 1
		fPos = rsc.jobLastPosition + 1
		rsc.jobLastPosition = fPos + 1
	}
	if nPos == n {
		return true, [][2]string{} // job is already at this position
	}

	// list of files to rename
	moveLst := [][2]string{}

	fJ, isFrom := rsc.queueJobs[submitStamp]
	toJ, isTo := rsc.queueJobs[rsc.queueKeys[nPos]]

	if !isFirst && !isLast {

		if isFrom && isTo {
			moveLst = append(moveLst, [2]string{
				fJ.filePath,
				jobQueuePath(fJ.SubmitStamp, fJ.ModelName, fJ.ModelDigest, fJ.IsMpi, toJ.position, fJ.Res.ProcessCount, fJ.Res.ThreadCount, fJ.Res.ProcessMemMb, fJ.Res.ThreadMemMb),
			})
		}
	} else { // move source file to the top (before the first position) or to the bottom (after last postion)

		moveLst = append(moveLst, [2]string{
			fJ.filePath,
			jobQueuePath(fJ.SubmitStamp, fJ.ModelName, fJ.ModelDigest, fJ.IsMpi, fPos, fJ.Res.ProcessCount, fJ.Res.ThreadCount, fJ.Res.ProcessMemMb, fJ.Res.ThreadMemMb),
		})
	}

	// move down to the queue: shift items up
	if nPos > n {

		for k := n; k < nPos && k < len(rsc.queueKeys)-1; k++ {

			// skip files selected to run
			fromKey := rsc.queueKeys[k+1]

			isSel := false
			for j := 0; !isSel && j < len(rsc.selectedKeys); j++ {
				isSel = rsc.selectedKeys[j] == fromKey
			}
			if isSel {
				continue
			}

			if !isFirst && !isLast {

				fJ, isFrom = rsc.queueJobs[fromKey]
				toJ, isTo = rsc.queueJobs[rsc.queueKeys[k]]
				if isFrom && isTo {
					moveLst = append(moveLst, [2]string{
						fJ.filePath,
						jobQueuePath(fJ.SubmitStamp, fJ.ModelName, fJ.ModelDigest, fJ.IsMpi, toJ.position, fJ.Res.ProcessCount, fJ.Res.ThreadCount, fJ.Res.ProcessMemMb, fJ.Res.ThreadMemMb),
					})
				}
			}

			rsc.queueKeys[k] = fromKey
		}

	} else { // move up in the queue: shift items down

		for k := n - 1; k >= nPos && k >= 0; k-- {

			// skip files selected to run
			fromKey := rsc.queueKeys[k]

			isSel := false
			for j := 0; !isSel && j < len(rsc.selectedKeys); j++ {
				isSel = rsc.selectedKeys[j] == fromKey
			}
			if isSel {
				continue
			}

			if !isFirst && !isLast {

				fJ, isFrom = rsc.queueJobs[fromKey]
				toJ, isTo = rsc.queueJobs[rsc.queueKeys[k+1]]
				if isFrom && isTo {
					moveLst = append(moveLst, [2]string{
						fJ.filePath,
						jobQueuePath(fJ.SubmitStamp, fJ.ModelName, fJ.ModelDigest, fJ.IsMpi, toJ.position, fJ.Res.ProcessCount, fJ.Res.ThreadCount, fJ.Res.ProcessMemMb, fJ.Res.ThreadMemMb),
					})
				}
			}

			rsc.queueKeys[k+1] = fromKey
		}
	}
	rsc.queueKeys[nPos] = submitStamp

	return true, moveLst
}

// Select computational servers or clusters to startup.
// Return server names, startup timeout and for each server startup exe names and startup arguments
func (rsc *RunCatalog) selectToStartCompute() ([]string, int64, []string, [][]string) {

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	// do not start anything if:
	// this instance is not a leader oms instance or queue is paused or no computational servers to start
	nowTs := time.Now().UnixMilli()

	if !rsc.isLeader || rsc.IsAllQueuePaused || rsc.lastStartStopTs+computeStartStopInterval > nowTs {
		return []string{}, 1, []string{}, [][]string{}
	}

	// remove from startup list servers which are no longer exist
	n := 0
	for _, name := range rsc.startupNames {

		if _, ok := rsc.computeState[name]; ok {
			rsc.startupNames[n] = name // server stil exist
			n++
		}
	}
	rsc.startupNames = rsc.startupNames[:n]

	// append to startup list servers for the first job in global queue
	srvLst := []string{}
	exeLst := []string{}
	argLst := [][]string{}

	for _, cu := range rsc.first.hostUse {

		if cs, ok := rsc.computeState[cu.CompName]; !ok || cs.State != "" {
			continue // server not exists or not in power off state
		}

		isAct := false
		for k := 0; !isAct && k < len(rsc.startupNames); k++ {
			isAct = rsc.startupNames[k] == cu.CompName
		}
		for k := 0; !isAct && k < len(rsc.shutdownNames); k++ {
			isAct = rsc.shutdownNames[k] == cu.CompName
		}
		if isAct {
			continue // this server already in startup or shutdown list
		}

		// for each server find startup executable and arguments
		srvLst = append(srvLst, cu.CompName)
		exeLst = append(exeLst, rsc.computeState[cu.CompName].startExe)

		args := make([]string, len(rsc.computeState[cu.CompName].startArgs))
		copy(args, rsc.computeState[cu.CompName].startArgs)
		argLst = append(argLst, args)
	}

	// append to servers starup list and return list of servers to start
	rsc.startupNames = append(rsc.startupNames, srvLst...)

	return srvLst, rsc.maxStartTime, exeLst, argLst
}

// Select computational servers or clusters to shutdown.
// Return server names, shutdown timeout and for each server shutdown exe names and shutdown arguments
func (rsc *RunCatalog) selectToStopCompute() ([]string, int64, []string, [][]string) {

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	// do not stop anything: if idle time is unlimited or this instance is not a leader oms instance
	nowTs := time.Now().UnixMilli()

	if rsc.maxIdleTime <= 0 || !rsc.isLeader || rsc.lastStartStopTs+computeStartStopInterval > nowTs {
		return []string{}, 1, []string{}, [][]string{}
	}

	// remove from shutdown list servers which are no longer exist
	n := 0
	for _, name := range rsc.shutdownNames {
		if _, ok := rsc.computeState[name]; ok {
			rsc.shutdownNames[n] = name // server stil exist
			n++
		}
	}
	rsc.shutdownNames = rsc.shutdownNames[:n]

	// find servers where there no model runs for more than idle time interval
	srvLst := []string{}
	exeLst := []string{}
	argLst := [][]string{}

	for _, cs := range rsc.computeState {

		isAct := false
		for k := 0; !isAct && k < len(rsc.shutdownNames); k++ {
			isAct = rsc.shutdownNames[k] == cs.Name
		}
		for k := 0; !isAct && k < len(rsc.startupNames); k++ {
			isAct = rsc.startupNames[k] == cs.Name
		}
		for k := 0; !isAct && k < len(rsc.first.hostUse); k++ {
			isAct = rsc.first.hostUse[k].CompName == cs.Name
		}
		if isAct {
			// this server already in shutdown or startup list
			// or in the list of servers for the first job
			continue
		}

		// if no model runs for more than idle time in milliseconds and server started more than idle time in milliseconds
		if cs.State == "ready" && cs.LastUsedTs+rsc.maxIdleTime < nowTs {

			srvLst = append(srvLst, cs.Name)
			exeLst = append(exeLst, cs.stopExe)

			args := make([]string, len(cs.stopArgs))
			copy(args, cs.stopArgs)
			argLst = append(argLst, args)

			rsc.shutdownNames = append(rsc.shutdownNames, cs.Name)
		}
	}

	return srvLst, rsc.maxStopTime, exeLst, argLst
}

// on sucess reset server error count or increase it on error and remove server name from startup list
func (rsc *RunCatalog) startupCompleted(isOkStart bool, name string) {

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	// update server state
	rsc.lastStartStopTs = time.Now().UnixMilli()

	if cs, ok := rsc.computeState[name]; ok {
		if isOkStart && cs.LastUsedTs < rsc.lastStartStopTs {
			cs.LastUsedTs = rsc.lastStartStopTs
			rsc.computeState[name] = cs
		}
	}

	// remove server from startup list
	n := 0
	for k := range rsc.startupNames {
		if rsc.startupNames[k] != name {
			rsc.startupNames[n] = rsc.startupNames[k]
			n++
		}
	}
	rsc.startupNames = rsc.startupNames[:n]
}

// on sucess reset server error count or increase it on error remove server name from shutdown list
func (rsc *RunCatalog) shutdownCompleted(isOkStop bool, name string) {

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	// update server state
	rsc.lastStartStopTs = time.Now().UnixMilli()

	if cs, ok := rsc.computeState[name]; ok {
		if isOkStop && cs.LastUsedTs < rsc.lastStartStopTs {
			cs.LastUsedTs = rsc.lastStartStopTs
			rsc.computeState[name] = cs
		}
	}

	// remove server from shutdown list
	n := 0
	for k := range rsc.shutdownNames {
		if rsc.shutdownNames[k] != name {
			rsc.shutdownNames[n] = rsc.shutdownNames[k]
			n++
		}
	}
	rsc.shutdownNames = rsc.shutdownNames[:n]
}

// update run catalog with current job control files
func (rsc *RunCatalog) updateRunJobs(
	jsState JobServiceState,
	computeState map[string]computeItem,
	firstHostUse jobHostUse,
	cfgRes []modelCfgRes,
	queueJobs map[string]queueJobFile,
	activeJobs map[string]runJobFile,
	historyJobs map[string]historyJobFile,
	omsActive map[string]omsUsage,
	activeRuns []runUsage,
	runCompUsage []runComputeUse,
	queueRqs []queueRequest,

) *jobControlState {

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	jNextPos := rsc.jobLastPosition

	rsc.JobServiceState = jsState

	if rsc.jobLastPosition < jNextPos {
		rsc.jobLastPosition = jNextPos
	}

	// copy state of computational resources
	for name := range rsc.computeState {
		_, ok := computeState[name]
		if !ok {
			delete(rsc.computeState, name) // remove: server or cluster not does exist anymore
		}
	}
	for name, cs := range computeState {
		if cs.LastUsedTs < rsc.computeState[name].LastUsedTs {
			cs.LastUsedTs = rsc.computeState[name].LastUsedTs
		}
		rsc.computeState[name] = cs
	}

	// if first job changed or if there is a new host allocation then update computational host usage
	if rsc.first.oms != firstHostUse.oms || rsc.first.stamp != firstHostUse.stamp {

		rsc.first.oms = firstHostUse.oms
		rsc.first.stamp = firstHostUse.stamp
		rsc.first.res = firstHostUse.res
		rsc.first.hostUse = rsc.first.hostUse[:0]
		rsc.first.hostUse = append(rsc.first.hostUse, firstHostUse.hostUse...)

	} else { // this job already first, check if old host allocation still valid

		isOk := len(rsc.first.hostUse) == 0 && len(firstHostUse.hostUse) == 0 ||
			len(rsc.first.hostUse) != 0 && len(firstHostUse.hostUse) != 0

		// check if all hosts allocated before still have enough resources to run the job
		for k := 0; isOk && k < len(rsc.first.hostUse); k++ {

			cs, ok := rsc.computeState[rsc.first.hostUse[k].CompName]
			isOk = ok && cs.State != "error" && cs.TotalRes.Cpu >= rsc.first.hostUse[k].Cpu && cs.TotalRes.Mem >= rsc.first.hostUse[k].Mem
		}
		if !isOk {
			rsc.first.hostUse = rsc.first.hostUse[:0]
			rsc.first.hostUse = append(rsc.first.hostUse, firstHostUse.hostUse...)
		}
	}

	// copy model resources requirements
	clear(rsc.cfgRes)
	binRoot, _ := theCatalog.getModelDir()
	br := filepath.ToSlash(binRoot)

	for dgst, mb := range rsc.models {

		sp := filepath.ToSlash(filepath.Join(mb.binDir, mb.name))

		for _, rs := range cfgRes {
			if sp == path.Join(br, rs.Path) {
				rsc.cfgRes[dgst] = rs
				break
			}
		}
	}

	// update queue jobs and collect all new submission stamps
	for stamp := range rsc.queueJobs {
		jf, ok := queueJobs[stamp]
		if !ok || jf.isError {
			delete(rsc.queueJobs, stamp) // remove: job file not exists
		}
	}

	for stamp, jf := range queueJobs {
		if _, ok := rsc.models[jf.ModelDigest]; !ok {
			continue // skip: model digest is not the models list
		}
		if jf.isError {
			continue // skip: model job error
		}
		rsc.queueJobs[stamp] = jf
	}

	// remove queue submission stamps which are no longer exists in the queue
	n := 0
	for _, stamp := range rsc.queueKeys {
		if _, ok := queueJobs[stamp]; ok {
			rsc.queueKeys[n] = stamp
			n++
		}
	}
	rsc.queueKeys = rsc.queueKeys[:n]

	// find new submission stamps from the queue
	n = len(queueJobs) - n
	if n > 0 {

		qKeys := make([]string, n)
		k := 0
		for stamp := range queueJobs {

			isFound := false
			for j := 0; !isFound && j < len(rsc.queueKeys); j++ {
				isFound = rsc.queueKeys[j] == stamp
			}
			if !isFound {
				qKeys[k] = stamp
				k++
			}
		}

		// sort new jobs by time stamps: first come first served and append at the end of existing queue
		sort.Strings(qKeys)
		rsc.queueKeys = append(rsc.queueKeys, qKeys...)
	}

	// update active model run jobs
	for stamp := range rsc.activeJobs {
		jf, ok := activeJobs[stamp]
		if !ok || jf.isError {
			delete(rsc.activeJobs, stamp) // remove: job file not exists
		}
	}

	for stamp, jf := range activeJobs {
		if _, ok := rsc.models[jf.ModelDigest]; !ok {
			continue // skip: model digest is not the models list
		}
		if jf.isError {
			continue // skip: model job error
		}
		rsc.activeJobs[stamp] = jf
	}

	// update model run job history
	for stamp := range rsc.historyJobs {
		jh, ok := historyJobs[stamp]
		if !ok || jh.isError {
			delete(rsc.historyJobs, stamp) // remove: job file not exist
		}
	}

	for stamp, jh := range historyJobs {
		if !jh.isError {
			rsc.historyJobs[stamp] = jh
		}
	}

	// cleanup selected to run jobs list: remove if submission stamp not exist in queue files list
	n = 0
	for _, stamp := range rsc.selectedKeys {
		if _, ok := queueJobs[stamp]; ok {
			rsc.selectedKeys[n] = stamp // job file still exist in the queue
			n++
		}
	}
	rsc.selectedKeys = rsc.selectedKeys[:n]

	// return job control state
	jsc := jobControlState{
		Queue: make([]string, len(rsc.queueKeys)),
	}
	copy(jsc.Queue, rsc.queueKeys)

	// if this is a global admin oms instance then update model runs state and resources usage
	if theCfg.isAdminAll {

		// for all oms instances: state and resources usage
		rsc.OmsActive = rsc.OmsActive[:0]
		for name, u := range omsActive {
			rsc.OmsActive = append(rsc.OmsActive, omsState{Oms: name, omsUsage: u})
		}

		rsc.ActiveRuns = rsc.ActiveRuns[:0] // for active model runs: computational resources usage
		rsc.ActiveRuns = append(rsc.ActiveRuns, activeRuns...)

		rsc.RunCompUsage = rsc.RunCompUsage[:0] // for all active model runs computational resources usage on each server
		rsc.RunCompUsage = append(rsc.RunCompUsage, runCompUsage...)

		rsc.QueueRuns = rsc.QueueRuns[:0] // all queued runs oms, model and computational resources
		rsc.QueueRuns = append(rsc.QueueRuns, queueRqs...)
	}

	return &jsc
}
