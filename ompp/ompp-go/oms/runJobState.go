// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// scan job control directories to read and update job lists: queue, active and history
func scanStateJobs(doneC <-chan bool) {
	if !theCfg.isJobControl {
		return // job control disabled
	}

	omsTickPath, _ := makeOmsTick(helper.MakeTimeStamp(time.Now())) // job processing started at this oms instance
	nTick := 0

	// path to job.ini: available resources limits and computational servers configuration
	jobIniPath := filepath.Join(theCfg.jobDir, "job.ini")

	// model run files in queue, active runs, run history
	queuePtrn := filepath.Join(theCfg.jobDir, "queue") + string(filepath.Separator) + "*-#-*-#-*-#-*-#-*-#-cpu-#-*-#-mem-#-*.json"
	activePtrn := filepath.Join(theCfg.jobDir, "active") + string(filepath.Separator) + "*-#-*-#-*-#-*-#-*-#-*-#-cpu-#-*-#-mem-#-*.json"
	historyPtrn := filepath.Join(theCfg.jobDir, "history") + string(filepath.Separator) + "*-#-" + theCfg.omsName + "-#-*.json"

	// oms instances heart beat and oms instance queue paused files:
	// if oms instance file does not updated more than 1 minute then oms instance is dead
	// if oms instance file does not have last run stamp then use current date-time stamp
	// oms instance heart beat tick:  oms-#-_4040-#-2022_07_08_23_45_12_123-#-1257894000000-#-2022_08_17_21_56_34_321
	// oms instance job queue paused: jobs.queue-#-_4040-#-paused
	omsTickPtrn := filepath.Join(theCfg.jobDir, "state") + string(filepath.Separator) + "oms-#-*-#-*-#-*"
	omsPausedPtrn := filepath.Join(theCfg.jobDir, "state") + string(filepath.Separator) + "jobs.queue-#-*-#-paused"

	// disk use file: disk-#-_4040-#-size-#-100-#-ok-#-120-#-2022_07_08_23_45_12_123-#-125678.json
	diskUsePtrn := filepath.Join(theCfg.jobDir, "state", "disk-#-*-#-size-#-*-#-*-#-*-#-*-#-*.json")

	// compute servers or clusters:
	// server ready: comp-ready-#-name
	// server start: comp-start-#-name-#-2022_08_17_22_33_44_567
	// server stop:  comp-stop-#-name-#-2022_08_17_22_33_44_567
	// server error: comp-error-#-name-#-2022_08_17_22_33_44_567
	// server used by model run: comp-used-#-name-#-2022_07_08_23_03_27_555-#-_4040-#-cpu-#-4-#-mem-#-8
	compReadyPtrn := filepath.Join(theCfg.jobDir, "state") + string(filepath.Separator) + "comp-ready-#-*"
	compStartPtrn := filepath.Join(theCfg.jobDir, "state") + string(filepath.Separator) + "comp-start-#-*-#-*"
	compStopPtrn := filepath.Join(theCfg.jobDir, "state") + string(filepath.Separator) + "comp-stop-#-*-#-*"
	compErrorPtrn := filepath.Join(theCfg.jobDir, "state") + string(filepath.Separator) + "comp-error-#-*-#-*"
	compUsedPtrn := filepath.Join(theCfg.jobDir, "state") + string(filepath.Separator) + "comp-used-#-*-#-*-#-*-#-cpu-#-*-#-mem-#-*"

	queueJobs := map[string]queueJobFile{}
	queueRqs := []queueRequest{}
	activeJobs := map[string]runJobFile{}
	historyJobs := map[string]historyJobFile{}
	omsPaused := map[string]bool{}
	omsDiskUsage := map[string]diskUsage{}
	computeState := map[string]computeItem{}
	omsActive := map[string]omsUsage{} // for all oms instances: state, computational and disk resources usage
	activeRuns := []runUsage{}         // for active model runs: computational resources usage
	runCompUsage := []runComputeUse{}  // for all active model runs computational resources usage on each server
	hostByCpu := []string{}            // names of computational servers or clusters sorted by available CPU cores
	hostByMem := []string{}            // names of computational servers or clusters sorted by available memory

	for {
		// get jobs service state and computational resources state: servers or clustres definition
		updateTs := time.Now()
		nowTs := updateTs.UnixMilli()

		jsState, cfgRes := initJobComputeState(jobIniPath, updateTs, computeState)

		queueFiles := filesByPattern(queuePtrn, "Error at queue job files search")
		activeFiles := filesByPattern(activePtrn, "Error at active job files search")
		historyFiles := filesByPattern(historyPtrn, "Error at history job files search")
		omsTickFiles := filesByPattern(omsTickPtrn, "Error at oms heart beat files search")
		omsPausedFiles := filesByPattern(omsPausedPtrn, "Error at queue paused files search")
		diskUseFiles := filesByPattern(diskUsePtrn, "Error at disk use files search")
		compReadyFiles := filesByPattern(compReadyPtrn, "Error at server ready files search")
		compStartFiles := filesByPattern(compStartPtrn, "Error at server start files search")
		compStopFiles := filesByPattern(compStopPtrn, "Error at server stop files search")
		compErrorFiles := filesByPattern(compErrorPtrn, "Error at server errors files search")
		compUsedFiles := filesByPattern(compUsedPtrn, "Error at server usage files search")

		jsState.jobLastPosition = jobPositionDefault + (1 + len(queueFiles))
		jsState.jobFirstPosition = jobPositionDefault - (1 + len(queueFiles))

		// update oms instances paused status
		clear(omsPaused)

		for _, fp := range omsPausedFiles {

			oms := parseQueuePausedPath(fp)
			if oms != "" {
				omsPaused[oms] = true
			}
		}

		// update oms instances disk usage status
		clear(omsDiskUsage)

		for _, fp := range diskUseFiles {

			oms, mb, isOver, ul, _, _, _, _ := parseDiskUseStatePath(fp)
			if oms != "" {
				omsDiskUsage[oms] = diskUsage{
					IsDiskOver:   isOver,
					TotalSizeMb:  mb,
					LimitMb:      ul,
					diskFilePath: fp,
				}
			}
		}

		// update oms instances heart beat status
		// one minute of missing heart beats is an interval to consider oms instance dead
		minOmsStateTs := updateTs.Add(-1 * time.Minute).UnixMilli()
		leaderName := ""

		for _, fp := range omsTickFiles {

			oms, _, ts, rStamp := parseOmsTickPath(fp)
			if oms == "" {
				continue // skip: invalid active run job state file path
			}

			if ts > minOmsStateTs { // oms instance is alive

				u := omsActive[oms]
				omsActive[oms] = omsUsage{
					LastStamp:   rStamp,
					IsPaused:    omsPaused[oms],
					ComputeRes:  u.ComputeRes,
					LocalRes:    u.LocalRes,
					diskUsage:   omsDiskUsage[oms],
					omsFilePath: fp,
				}
				if leaderName == "" || leaderName > oms {
					leaderName = oms
				}
			} else {
				delete(omsActive, oms) // oms instance if no longer active
			}
		}
		jsState.isLeader = theCfg.omsName == leaderName // this oms instance is a leader instance

		// computational resources state
		// for each server or cluster detect current state: ready, start, stop or power off
		// total computational resources (cpu and memory), used resources and avaliable resources
		runCompUsage = updateComputeState(
			computeState,
			omsActive,
			runCompUsage,
			nowTs,
			jsState.maxStartTime,
			jsState.maxStopTime,
			jsState.maxComputeErrors,
			compReadyFiles, compStartFiles, compStopFiles, compErrorFiles, compUsedFiles)

		jsState.MpiErrorRes = ComputeRes{}
		hostByCpu = hostByCpu[:0]
		hostByMem = hostByMem[:0]

		for _, cs := range computeState {
			if cs.State == "error" {
				jsState.MpiErrorRes.Cpu += cs.TotalRes.Cpu
				jsState.MpiErrorRes.Mem += cs.TotalRes.Mem
			} else {
				hostByCpu = append(hostByCpu, cs.Name)
				hostByMem = append(hostByMem, cs.Name)
			}
		}

		// sort computational servers:
		//   if ready then use where more cpu and memory
		//   if not ready then use where less errors and longest unused time
		cmpHost := func(isByMem bool, nameLst []string) func(i, j int) bool {

			return func(i, j int) bool {
				ics := computeState[nameLst[i]]
				iCpu := ics.TotalRes.Cpu - ics.UsedRes.Cpu
				iMem := ics.TotalRes.Mem - ics.UsedRes.Mem
				jcs := computeState[nameLst[j]]
				jCpu := jcs.TotalRes.Cpu - jcs.UsedRes.Cpu
				jMem := jcs.TotalRes.Mem - jcs.UsedRes.Mem

				switch {
				case ics.State == "ready" && jcs.State == "ready":
					if !isByMem {
						return iCpu > jCpu || iCpu == jCpu && iMem > jMem
					}
					return iMem > jMem || iMem == jMem && iCpu > jCpu
				case ics.State == "ready" && jcs.State != "ready":
					return true
				case ics.State != "ready" && jcs.State == "ready":
					return false
				}
				if iCpu != jCpu || iMem != jMem {
					if !isByMem {
						return iCpu > jCpu || iCpu == jCpu && iMem > jMem
					}
					return iMem > jMem || iMem == jMem && iCpu > jCpu
				}
				return ics.ErrorCount < jcs.ErrorCount ||
					ics.ErrorCount == jcs.ErrorCount && ics.LastUsedTs < jcs.LastUsedTs ||
					ics.ErrorCount == jcs.ErrorCount && ics.LastUsedTs == jcs.LastUsedTs && ics.Name < jcs.Name
			}
		}
		sort.SliceStable(hostByCpu, cmpHost(false, hostByCpu))
		sort.SliceStable(hostByMem, cmpHost(true, hostByMem))

		mpiTotalRes := ComputeRes{}
		isMpiLimit := false

		if len(computeState) <= 0 {

			mpiTotalRes = jsState.LocalRes
			isMpiLimit = jsState.LocalRes.Cpu > 0
		} else {

			isMpiLimit = true
			mpiTotalRes.Cpu = jsState.MpiRes.Cpu - jsState.MpiErrorRes.Cpu
			mpiTotalRes.Mem = jsState.MpiRes.Mem - jsState.MpiErrorRes.Mem
		}

		// model runs
		//
		// parse active files, use unlimited resources for already active jobs
		aKeys, aTotal, aOwn, aLocalTotal, aLocal, activeRuns := updateActiveJobs(activeFiles, activeJobs, omsActive, activeRuns)

		// parse queue files and re-build model runs queue
		sort.Strings(queueFiles)

		qKeys, maxPos, minPos, qTotal, qOwn, qLocalTotal, qLocal, firstHostUse := updateQueueJobs(
			queueFiles,
			queueJobs,
			activeJobs,
			mpiTotalRes,
			isMpiLimit,
			jsState.MpiMaxThreads,
			jsState.LocalRes,
			jsState.MaxOwnMpiRes,
			hostByCpu,
			hostByMem,
			computeState,
			omsActive,
			jsState.IsAllQueuePaused,
		)

		// parse history files list
		hKeys := make([]string, 0, len(historyFiles))

		for _, f := range historyFiles {

			// get submission stamp and oms instance
			subStamp, oms, mn, dgst, rStamp, status := parseHistoryPath(f)
			if subStamp == "" || oms == "" {
				continue // file name is not a job file name
			}
			hKeys = append(hKeys, subStamp)

			if _, ok := historyJobs[subStamp]; ok {
				continue // this file already in the history jobs list
			}

			// add job into history jobs list
			historyJobs[subStamp] = historyJobFile{
				filePath:    f,
				isError:     (mn == "" || dgst == "" || rStamp == "" || status == ""),
				SubmitStamp: subStamp,
				ModelName:   mn,
				ModelDigest: dgst,
				RunStamp:    rStamp,
				JobStatus:   status,
				RunTitle:    getJobRunTitle(f),
			}
		}

		// remove from queue files or active files which are in history
		// remove from queue files which are in active
		for stamp := range historyJobs {
			delete(queueJobs, stamp)
			delete(activeJobs, stamp)
		}
		for stamp := range activeJobs {
			delete(queueJobs, stamp)
		}

		// remove existing job entries where files are no longer exist
		sort.Strings(qKeys)
		for stamp := range queueJobs {
			k := sort.SearchStrings(qKeys, stamp)
			if k < 0 || k >= len(qKeys) || qKeys[k] != stamp {
				delete(queueJobs, stamp)
			}
		}
		sort.Strings(aKeys)
		for stamp := range activeJobs {
			k := sort.SearchStrings(aKeys, stamp)
			if k < 0 || k >= len(aKeys) || aKeys[k] != stamp {
				delete(activeJobs, stamp)
			}
		}
		sort.Strings(hKeys)
		for stamp := range historyJobs {
			k := sort.SearchStrings(hKeys, stamp)
			if k < 0 || k >= len(hKeys) || hKeys[k] != stamp {
				delete(historyJobs, stamp)
			}
		}

		// parse queue files to provide global admin queue infromation
		queueRqs = queueRqs[:0]

		if theCfg.isAdminAll {
			for _, f := range queueFiles {

				// get submission stamp, oms instance and queue position
				stamp, oms, mn, dgst, isMpi, procCount, thCount, procMem, thMem, pos := parseQueuePath(f)
				if stamp == "" || oms == "" || mn == "" || dgst == "" {
					continue // file name is not a job file name
				}
				if _, isOk := omsActive[oms]; !isOk { // skip: oms instance inactive
					continue
				}
				queueRqs = append(queueRqs, queueRequest{
					Oms:          oms,
					SubmitStamp:  stamp,
					ModelName:    mn,
					ModelDigest:  dgst,
					IsMpi:        isMpi,
					ProcessCount: procCount,
					ThreadCount:  thCount,
					ProcessMemMb: procMem,
					ThreadMemMb:  thMem,
					Position:     pos,
					filePath:     f,
				})
			}
		}

		// update run catalog with current job control files and save persistent part of jobs state
		jsState.ActiveTotalRes = aTotal
		jsState.ActiveOwnRes = aOwn
		jsState.LocalActiveTotalRes = aLocalTotal
		jsState.LocalActiveRes = aLocal
		jsState.QueueTotalRes = qTotal
		jsState.QueueOwnRes = qOwn
		jsState.LocalQueueTotalRes = qLocalTotal
		jsState.LocalQueueRes = qLocal
		jsState.jobLastPosition = maxPos
		jsState.jobFirstPosition = minPos

		jsc := theRunCatalog.updateRunJobs(jsState, computeState, firstHostUse, cfgRes, queueJobs, activeJobs, historyJobs, omsActive, activeRuns, runCompUsage, queueRqs)
		jobStateWrite(*jsc)

		// update oms heart beat file
		nTick++
		if nTick%7 == 0 {
			omsTickPath, _ = makeOmsTick(omsActive[theCfg.omsName].LastStamp)
		}

		// wait for doneC or sleep
		if isExitSleep(jobScanInterval, doneC) {
			break
		}
	}

	fileDeleteAndLog(true, omsTickPath) // try to remove oms heart beat file, this code may never be executed due to race at shutdown
}

// insert run job into job map: map job file submission stamp to file content (run job).
// update last run stamp for oms instances.
func updateActiveJobs(fLst []string, jobMap map[string]runJobFile, omsActive map[string]omsUsage, activeRuns []runUsage) ([]string, ComputeRes, ComputeRes, ComputeRes, ComputeRes, []runUsage) {

	subStamps := make([]string, 0, len(fLst)) // list of submission stamps
	totalRes := ComputeRes{}
	ownRes := ComputeRes{}
	localTotalRes := ComputeRes{}
	localOwnRes := ComputeRes{}

	// clean oms cpu and memory usage
	for oms, u := range omsActive {
		u.Cpu = 0
		u.Mem = 0
		u.LocalRes.Cpu = 0
		u.LocalRes.Mem = 0
		omsActive[oms] = u
	}

	// parse active files, collect jobs for current oms instance
	// and for all instances update last run stamp and active MPI resource usage (cpu and memory)
	activeRuns = activeRuns[:0]
	for _, f := range fLst {

		// get submission stamp, oms instance and resources
		stamp, oms, mn, dgst, rStamp, isMpi, cpu, mem, pid := parseActivePath(f)
		if stamp == "" || oms == "" || mn == "" || dgst == "" {
			continue // file name is not a job file name
		}

		// update oms resource usage and last run stamp
		if u, ok := omsActive[oms]; !ok {
			continue // skip: oms instance inactive
		} else {

			if rStamp > u.LastStamp {
				u.LastStamp = rStamp
			}
			if isMpi {
				u.Cpu += cpu
				u.Mem += mem
			} else {
				u.LocalRes.Cpu += cpu
				u.LocalRes.Mem += mem
			}
			omsActive[oms] = u
		}

		// append model run into activce runs collection
		if theCfg.isAdminAll {
			activeRuns = append(activeRuns,
				runUsage{
					Oms:         oms,
					SubmitStamp: stamp,
					ComputeRes:  ComputeRes{Cpu: cpu, Mem: mem},
					IsMpi:       isMpi,
					ModelName:   mn,
					ModelDigest: dgst,
					RunStamp:    rStamp,
					pid:         pid,
					filePath:    f,
				})
		}

		// collect total resource usage and oms resource usage
		if isMpi {
			totalRes.Cpu = totalRes.Cpu + cpu
			totalRes.Mem = totalRes.Mem + mem
		} else {
			localTotalRes.Cpu = localTotalRes.Cpu + cpu
			localTotalRes.Mem = localTotalRes.Mem + mem
		}

		if oms != theCfg.omsName {
			continue // done with this job: it is other oms instance
		}

		// this is own job: job to run in current oms instance on MPI cluster or localhost server
		if isMpi {
			ownRes.Cpu = ownRes.Cpu + cpu
			ownRes.Mem = ownRes.Mem + mem
		} else {
			localOwnRes.Cpu = localOwnRes.Cpu + cpu
			localOwnRes.Mem = localOwnRes.Mem + mem
		}

		subStamps = append(subStamps, stamp)

		if jc, ok := jobMap[stamp]; ok {
			jobMap[stamp] = jc
			continue // this file already in the jobs list
		}

		// create run state from job file
		var jc RunJob
		isOk, err := helper.FromJsonFile(f, &jc)
		if err != nil {
			omppLog.LogNoLT(err)
			jobMap[stamp] = runJobFile{filePath: f, isError: true, oms: oms}
		}
		if !isOk || err != nil {
			continue // file not exist or invalid
		}

		jobMap[stamp] = runJobFile{RunJob: jc, filePath: f, oms: oms} // add job into jobs list
	}

	return subStamps, totalRes, ownRes, localTotalRes, localOwnRes, activeRuns
}

// insert run job into queue job map: map job file submission stamp to file content (run job)
func updateQueueJobs(
	fLst []string,
	queueJobs map[string]queueJobFile,
	activeJobs map[string]runJobFile,
	mpiTotalRes ComputeRes,
	isMpiLimit bool,
	mpiMaxTh int,
	localRes ComputeRes,
	maxMpiOwnRes ComputeRes,
	hostByCpu []string,
	hostByMem []string,
	computeState map[string]computeItem,
	omsActive map[string]omsUsage,
	isAllPaused bool,
) (
	[]string, int, int, ComputeRes, ComputeRes, ComputeRes, ComputeRes, jobHostUse) {

	nFiles := len(fLst)
	maxPos := jobPositionDefault + 1
	minPos := jobPositionDefault - 1

	// queue file name parts, position in the queue and resources
	type qFileHdr struct {
		fileIdx  int    // source file index
		oms      string // instance name
		stamp    string // submission stamp
		position int    // queue position: file name part
		allQPos  int    // queue position: one based index in combined queue (queues from all oms instances)
		res      RunRes // resources required to run the model
		isPaused bool   // if true then job queue is paused
		isOver   bool   // if true then resources required are exceeding total resource(s) limit(s)
		isFirst  bool   // if true the it is the first job in global queue
	}
	type omsQ struct {
		omsUsage            // oms last run stamp and resource usage
		q        []qFileHdr // instance queue files
	}
	qAll := make(map[string]omsQ, nFiles) // queue for each oms instance

	// for each oms instance append MPI job files to job queue
	for k, f := range fLst {

		// get submission stamp, oms instance and queue position
		stamp, oms, mn, dgst, isMpi, procCount, thCount, procMem, thMem, pos := parseQueuePath(f)
		if stamp == "" || oms == "" || mn == "" || dgst == "" {
			continue // file name is not a job file name
		}
		if maxPos < pos {
			maxPos = pos
		}
		if minPos > pos {
			minPos = pos
		}
		if !isMpi {
			continue // this is not MPI cluster job
		}
		u, isOk := omsActive[oms]
		if !isOk {
			continue // skip: oms instance inactive
		}
		cpu := procCount * thCount
		mem := memoryRunSize(procCount, thCount, procMem, thMem)

		// check resources quota: MPI cpu and memory available and disk usage not over the limit
		isOver := u.IsDiskOver || (isMpiLimit && cpu > mpiTotalRes.Cpu) || (mpiTotalRes.Mem > 0 && mem > mpiTotalRes.Mem)

		// check resources quota: max cpu memory allowed for each oms instance
		if !isOver && (maxMpiOwnRes.Cpu > 0 || maxMpiOwnRes.Mem > 0) {

			omsCpu := 0
			omsMem := 0
			for stamp := range activeJobs {
				omsCpu += activeJobs[stamp].Res.Cpu
				omsMem += activeJobs[stamp].Res.Mem

				isOver = maxMpiOwnRes.Cpu > 0 && omsCpu >= maxMpiOwnRes.Cpu || maxMpiOwnRes.Mem > 0 && omsMem >= mpiTotalRes.Mem
				if isOver {
					break
				}
			}
		}

		// append to the instance queue
		qOms, ok := qAll[oms]
		if !ok {
			qOms = omsQ{
				omsUsage: u,
				q:        make([]qFileHdr, 0, nFiles),
			}
		}
		qOms.q = append(qOms.q, qFileHdr{
			fileIdx:  k,
			oms:      oms,
			stamp:    stamp,
			position: pos,
			res: RunRes{
				ComputeRes: ComputeRes{
					Cpu: cpu,
					Mem: mem,
				},
				ProcessCount: procCount,
				ThreadCount:  thCount,
				ProcessMemMb: procMem,
				ThreadMemMb:  thMem,
			},
			isPaused: isAllPaused || u.IsPaused,
			isOver:   isOver,
		})
		qAll[oms] = qOms
	}

	// sort each job queue in order of position file name part and submission stamp
	for _, qOms := range qAll {
		sort.SliceStable(qOms.q, func(i, j int) bool {
			return qOms.q[i].position < qOms.q[j].position || qOms.q[i].position == qOms.q[j].position && qOms.q[i].stamp < qOms.q[j].stamp
		})
	}

	// sort oms instance names by:
	//   split instances in two categories depending if current active CPUs is less than quota or not
	//   move forward oms instances where usages less than quota
	// inside of each category sort oms instances by:
	//   last run stamp and oms instance name
	nOms := len(qAll)
	nc := 0 // cpu quota for each oms instance: eqully divide number of CPUs avaliable

	if nOms > 0 && isMpiLimit && mpiTotalRes.Cpu > 0 {

		nc = mpiTotalRes.Cpu / nOms
		if mpiTotalRes.Cpu%nOms > 0 {
			nc++
		}
		if nc < mpiMaxTh {
			nc = mpiMaxTh
		}
		if nc < 1 {
			nc = 1
		}
	}

	omsKeys := make([]string, nOms)
	n := 0
	for oms := range qAll {
		omsKeys[n] = oms
		n++
	}
	sort.SliceStable(omsKeys, func(i, j int) bool {

		iUse := qAll[omsKeys[i]].omsUsage
		jUse := qAll[omsKeys[j]].omsUsage

		if iUse.Cpu < nc && jUse.Cpu >= nc {
			return true
		}
		if iUse.Cpu >= nc && jUse.Cpu < nc {
			return false
		}
		return iUse.LastStamp < jUse.LastStamp || (iUse.LastStamp == jUse.LastStamp && omsKeys[i] < omsKeys[j])
	})

	// order combined queue jobs by:
	//   oms instance last run stamp
	// inside of each oms instance queue jobs are ordered by:
	//   position in the queue (position which user can adjust)
	//   submission stamp

	totalRes := ComputeRes{} // total resources required to serve all queues
	nextQueueIdx := 0        // global queue index position
	isFirstJob := true
	firstHostUse := jobHostUse{hostUse: []computeUse{}}

	for iOms := 0; iOms < nOms; iOms++ {

		qOms := qAll[omsKeys[iOms]]

		for jq := 0; jq < len(qOms.q); jq++ {

			// collect total resource usage
			totalRes.Cpu = totalRes.Cpu + qOms.q[jq].res.Cpu
			totalRes.Mem = totalRes.Mem + qOms.q[jq].res.Mem

			// if current top job is not exceeding available resources then assign global queue index position
			if !qOms.q[jq].isOver {

				// check if there are any server(s) exists to run the job
				srcJhu := jobHostUse{oms: omsKeys[iOms], stamp: qOms.q[jq].stamp, res: qOms.q[jq].res, hostUse: []computeUse{}}

				if qOms.q[jq].res.Mem > 0 {
					qOms.q[jq].isOver, _ = findComputeRes(srcJhu, false, mpiMaxTh, hostByMem, computeState)
				} else {
					qOms.q[jq].isOver, _ = findComputeRes(srcJhu, false, mpiMaxTh, hostByCpu, computeState)
				}

				// if job queue not paused then allocate job to the servers and add servers to startup list
				if !qOms.q[jq].isOver && !qOms.q[jq].isPaused {

					isOver := false
					var dst jobHostUse
					if qOms.q[jq].res.Mem > 0 {
						isOver, dst = findComputeRes(srcJhu, true, mpiMaxTh, hostByMem, computeState)
					} else {
						isOver, dst = findComputeRes(srcJhu, true, mpiMaxTh, hostByCpu, computeState)
					}

					// if this is the first job in global queue then save host ini servers
					if !isOver && isFirstJob {
						isFirstJob = false
						qOms.q[jq].isFirst = true
						firstHostUse = dst
					}
				}

				if !qOms.q[jq].isOver {
					nextQueueIdx++
					qOms.q[jq].allQPos = nextQueueIdx // one based index in global queue
				}
			}
		}

		qAll[omsKeys[iOms]] = qOms
	}

	// update current instance job map, queue files and resources
	// append localhost job queue for current oms instance

	qKeys := make([]string, 0, nFiles) // model run submission stamps for current oms instance
	totalLocal := ComputeRes{}
	ownLocal := ComputeRes{}
	isOmsPaused := isAllPaused || omsActive[theCfg.omsName].IsPaused // if current oms instance is paused
	isOmsDiskOver := omsActive[theCfg.omsName].IsDiskOver            // if current oms instance exceded disk quota
	isFirstJob = true

	for _, f := range fLst {

		// get submission stamp, oms instance and queue position
		stamp, oms, mn, dgst, isMpi, procCount, thCount, procMem, thMem, pos := parseQueuePath(f)
		if stamp == "" || oms == "" || mn == "" || dgst == "" {
			continue // file name is not a job file name
		}

		if isMpi {
			continue // skip: this is MPI model run
		}
		cpu := procCount * thCount
		mem := memoryRunSize(procCount, thCount, procMem, thMem)
		totalLocal.Cpu = totalLocal.Cpu + cpu
		totalLocal.Mem = totalLocal.Mem + mem

		if oms != theCfg.omsName {
			continue // skip: this is from other oms instance
		}
		ownLocal.Cpu = ownLocal.Cpu + cpu
		ownLocal.Mem = ownLocal.Mem + mem

		isOver := isOmsDiskOver || (localRes.Cpu > 0 && ownLocal.Cpu > localRes.Cpu) || (localRes.Mem > 0 && ownLocal.Mem > localRes.Mem)
		qKeys = append(qKeys, stamp)

		// if this file already in the queue jobs map then update resources
		if jc, ok := queueJobs[stamp]; ok {

			jc.filePath = f
			jc.position = pos
			jc.QueuePos = len(qKeys)
			jc.isPaused = isOmsPaused
			jc.IsOverLimit = isOver
			jc.isFirst = !isOver && !isOmsPaused && isFirstJob
			queueJobs[stamp] = jc // update existing job in the queue with current resources info

			if jc.isFirst {
				isFirstJob = false
			}
			continue
		}
		// else create run state from job file and insert into the queue map
		var jc RunJob

		isOk, err := helper.FromJsonFile(f, &jc)
		if err != nil {
			omppLog.LogNoLT(err)
			queueJobs[stamp] = queueJobFile{runJobFile: runJobFile{filePath: f, isError: true, oms: oms}}
		}
		if !isOk || err != nil {
			continue // file does not exist or invalid
		}
		jc.IsOverLimit = isOver
		jc.QueuePos = len(qKeys) // one-based position in local queue of the current oms instance

		// add new job into queue jobs map
		isFirst := !isOver && !isOmsPaused && isFirstJob

		queueJobs[stamp] = queueJobFile{
			runJobFile: runJobFile{RunJob: jc, filePath: f, oms: oms},
			position:   pos,
			isPaused:   isOmsPaused,
			isFirst:    isFirst,
		}
		if isFirst {
			isFirstJob = false
		}
	}

	// update current instance job map, queue files and resources
	// append MPI jobs queue for current oms instance
	ownRes := ComputeRes{}

	ownQ, isOwn := qAll[theCfg.omsName]
	if !isOwn {
		return qKeys, maxPos, minPos, totalRes, ownRes, totalLocal, ownLocal, firstHostUse // there are no MPI jobs for current oms instance
	}

	for _, f := range ownQ.q {

		ownRes.Cpu = ownRes.Cpu + f.res.Cpu
		ownRes.Mem = ownRes.Mem + f.res.Mem

		qKeys = append(qKeys, f.stamp)

		// if this file already in the queue jobs map then update resources
		if jc, ok := queueJobs[f.stamp]; ok {

			jc.filePath = fLst[f.fileIdx]
			jc.position = f.position
			jc.isPaused = isOmsPaused
			jc.IsOverLimit = f.isOver
			jc.isFirst = f.isFirst
			jc.QueuePos = f.allQPos
			jc.Res = f.res
			queueJobs[f.stamp] = jc // update existing job in the queue with current resources info
			continue
		}
		// else create run state from job file and insert into the queue map
		var jc RunJob

		isOk, err := helper.FromJsonFile(fLst[f.fileIdx], &jc)
		if err != nil {
			omppLog.LogNoLT(err)
			queueJobs[f.stamp] = queueJobFile{runJobFile: runJobFile{filePath: fLst[f.fileIdx], isError: true, oms: f.oms}}
		}
		if !isOk || err != nil {
			continue // file does not exist or invalid
		}
		jc.IsOverLimit = f.isOver
		jc.QueuePos = f.allQPos
		jc.Res = f.res

		// add new job into queue jobs map
		queueJobs[f.stamp] = queueJobFile{
			runJobFile: runJobFile{RunJob: jc, filePath: fLst[f.fileIdx], oms: f.oms},
			position:   f.position,
			isPaused:   isOmsPaused,
			isFirst:    f.isFirst,
		}
	}

	return qKeys, maxPos, minPos, totalRes, ownRes, totalLocal, ownLocal, firstHostUse
}

// check if there are any server(s) exists to run the job and find additional servers to start
func findComputeRes(src jobHostUse, isUse bool, mpiMaxTh int, computeHost []string, computeState map[string]computeItem) (bool, jobHostUse) {

	// max threads per process: limited by job.ini MpiMaxThreads value and by Threads value from model run request
	maxTh := src.res.Cpu
	if mpiMaxTh > 0 && maxTh > mpiMaxTh {
		maxTh = mpiMaxTh
	}
	if src.res.ThreadCount > 0 && maxTh > src.res.ThreadCount {
		maxTh = src.res.ThreadCount
	}
	dst := src
	dst.hostUse = []computeUse{}

	// check if model run request not exceed cpu and memory available resources
	minMem := memoryRunSize(1, 1, src.res.ProcessMemMb, src.res.ThreadMemMb) // memory required to run single thread
	cSel := map[string]int{}                                                 // map server name to number of process to run
	nTh := maxTh
	nCpu := 0

	for _, cn := range computeHost {

		if nCpu >= src.res.Cpu {
			break // done: all cores allocated
		}

		if _, isSel := cSel[cn]; isSel {
			continue // this server already selected
		}
		cs := computeState[cn]
		if cs.State == "error" {
			continue // skip: failed server
		}

		// try to allocate cores for at least one process
		nc := cs.TotalRes.Cpu
		aMem := cs.TotalRes.Mem
		if isUse {
			nc = nc - cs.UsedRes.Cpu
			aMem = aMem - cs.UsedRes.Mem
		}
		if nc > nTh {
			nc = nTh
		}

		if nc <= 0 || minMem > 0 && aMem < minMem {
			continue // no free resources on that server
		}

		// limit core usage by memory
		if minMem > 0 {

			for ; nc > 0; nc-- {
				if aMem >= memoryRunSize(1, nc, src.res.ProcessMemMb, src.res.ThreadMemMb) {
					break
				}
			}
		}
		if nc <= 0 {
			continue // not enough cores or memory to run model single process single thread
		}
		nTh = nc // max possible number of threads to run on all servers

		// re-calculate cpu and memory usage for all allocated servers
		// clear previous process allocation for all servers
		for name := range cSel {
			cSel[name] = 0
		}
		cSel[cn] = 0 // add new server

		// allocate to each server max number of model processes with that thread count and memory size
		mp := memoryRunSize(1, nTh, src.res.ProcessMemMb, src.res.ThreadMemMb)
		nCpu = 0

		for name := range cSel {

			c := computeState[name].TotalRes.Cpu
			m := computeState[name].TotalRes.Mem
			if isUse {
				c = c - computeState[name].UsedRes.Cpu
				m = m - computeState[name].UsedRes.Mem
			}

			np := 1
			for ; nCpu+np*nTh < src.res.Cpu; np++ {
				if (np+1)*nTh > c || minMem > 0 && (np+1)*mp > m {
					break
				}
			}
			cSel[name] = np
			nCpu += np * nTh

			if nCpu >= src.res.Cpu { // all cores allocated
				break
			}
		}

		if nCpu >= src.res.Cpu { // all cores allocated
			break
		}
	}

	// check search result: if enough resources found
	isOver := nCpu < src.res.Cpu
	if isOver {
		return true, dst // over limit: not enough resources to run the job
	}

	// update cpu and memory usage
	dst.hostUse = make([]computeUse, 0, len(cSel))

	if isUse {
		mp := memoryRunSize(1, nTh, src.res.ProcessMemMb, src.res.ThreadMemMb)
		procCount := 0

		for j := range computeHost {

			if np, ok := cSel[computeHost[j]]; ok && np > 0 {

				c := np * nTh
				m := np * mp
				dst.hostUse = append(dst.hostUse,
					computeUse{CompName: computeHost[j], ComputeRes: ComputeRes{Cpu: c, Mem: m}},
				)
				procCount += np
			}
		}
		dst.res.ProcessCount = procCount
		dst.res.ThreadCount = nTh
		dst.res.Mem = procCount * mp // actual memory usage
	}

	return isOver, dst
}

// read job service state and computational servers definition from job.ini
func initJobComputeState(jobIniPath string, updateTs time.Time, computeState map[string]computeItem) (JobServiceState, []modelCfgRes) {

	jsState := JobServiceState{
		JobServicePub: JobServicePub{
			IsQueuePaused:     isPausedJobQueue(),
			IsAllQueuePaused:  isPausedJobAllQueue(),
			JobUpdateDateTime: helper.MakeDateTime(updateTs),
		},
		maxStartTime: serverTimeoutDefault,
		maxStopTime:  serverTimeoutDefault,
	}
	cfgRes := []modelCfgRes{}

	// read available resources limits and computational servers configuration from job.ini
	if jobIniPath == "" || !helper.IsFileExist(jobIniPath) {
		return jsState, cfgRes
	}

	opts, err := config.FromIni(jobIniPath, theCfg.encodingName)
	if err != nil {
		omppLog.LogNoLT(err)
		return jsState, cfgRes
	}
	nowTs := updateTs.UnixMilli()

	// total available resources limits and timeouts
	jsState.LocalRes.Cpu = opts.Int("Common.LocalCpu", 0)      // localhost unlimited cpu cores by default
	jsState.LocalRes.Mem = opts.Int("Common.LocalMemory", 0)   // localhost unlimited memory by default
	jsState.MaxOwnMpiRes.Cpu = opts.Int("Common.MpiCpu", 0)    // max MPI cpu cores available for each oms instance, unlimited cpu cores by default
	jsState.MaxOwnMpiRes.Mem = opts.Int("Common.MpiMemory", 0) // max MPI memory cores available for each oms instance, by default

	jsState.maxIdleTime = 1000 * opts.Int64("Common.IdleTimeout", 0) // zero default timeout: never stop servers by default
	jsState.maxStartTime = 1000 * opts.Int64("Common.StartTimeout", serverTimeoutDefault)
	jsState.maxStopTime = 1000 * opts.Int64("Common.StopTimeout", serverTimeoutDefault)
	jsState.maxComputeErrors = opts.Int("Common.MaxErrors", maxComputeErrorsDefault)

	// MPI jobs process, threads and hostfile config
	jsState.MpiMaxThreads = opts.Int("Common.MpiMaxThreads", 0) // max number of modelling threads per MPI process, zero means unlimited
	jsState.hostFile.hostName = opts.String("hostfile.HostName")
	jsState.hostFile.cpuCores = opts.String("hostfile.CpuCores")
	jsState.hostFile.rootLine = opts.String("hostfile.RootLine")
	jsState.hostFile.hostLine = opts.String("hostfile.HostLine")
	jsState.hostFile.dir = opts.String("hostfile.HostFileDir")

	jsState.hostFile.isUse = jsState.hostFile.dir != "" &&
		jsState.hostFile.dir != "." && jsState.hostFile.dir != ".." &&
		jsState.hostFile.dir != "./" && jsState.hostFile.dir != "../" && jsState.hostFile.dir != "/" &&
		(jsState.hostFile.rootLine != "" || jsState.hostFile.hostLine != "")

	if jsState.hostFile.isUse {
		jsState.hostFile.isUse = helper.IsDirExist(jsState.hostFile.dir)
	}

	// default settings for compute servers or clusters
	//
	splitOpts := func(key, sep string) []string {
		v := strings.Split(opts.String(key), sep)
		if len(v) <= 0 || len(v) == 1 && v[0] == "" {
			return []string{}
		}
		return v
	}

	exeStart := opts.String("Common.StartExe")
	exeStop := opts.String("Common.StopExe")
	argsBreak := opts.String("Common.ArgsBreak")
	argsStart := splitOpts("Common.StartArgs", argsBreak)
	argsStop := splitOpts("Common.StopArgs", argsBreak)

	// compute servers or clusters defaults
	srvNames := splitOpts("Common.Servers", ",")

	for k, s := range srvNames {

		srvNames[k] = strings.TrimSpace(s)
		if srvNames[k] == "" {
			continue // skip empty name
		}

		// add or clean existing computational server state
		cs, ok := computeState[srvNames[k]]
		if !ok {
			cs = computeItem{
				computePub: computePub{Name: srvNames[k], LastUsedTs: nowTs},
				startArgs:  []string{},
				stopArgs:   []string{},
			}
		}
		cs.UsedRes.Cpu = 0 // updated as sum of comp-used files
		cs.UsedRes.Mem = 0 // updated as sum of comp-used files
		cs.OwnRes.Cpu = 0  // updated as sum of comp-used files
		cs.OwnRes.Mem = 0  // updated as sum of comp-used files
		cs.ErrorCount = 0  // updated as count of comp-start, comp-stop, comp-error files

		cs.TotalRes.Cpu = opts.Int(cs.Name+".Cpu", 1)    // one cpu core by default
		cs.TotalRes.Mem = opts.Int(cs.Name+".Memory", 0) // unlimited memory by default

		cs.startExe = opts.String(cs.Name + ".StartExe")
		if cs.startExe == "" {
			cs.startExe = exeStart // default executable to start server
		}

		cs.startArgs = splitOpts(cs.Name+".StartArgs", argsBreak)
		if len(cs.startArgs) <= 0 {
			cs.startArgs = append(argsStart, cs.Name) // default start arguments and server name as last argument
		}

		cs.stopExe = opts.String(cs.Name + ".StopExe")
		if cs.stopExe == "" {
			cs.stopExe = exeStop // default executable to stop server
		}

		cs.stopArgs = splitOpts(cs.Name+".StopArgs", argsBreak)
		if len(cs.stopArgs) <= 0 {
			cs.stopArgs = append(argsStop, cs.Name) // default stop arguments and server name as last argument
		}
		computeState[cs.Name] = cs
	}

	// remove compute servers or clusters which are no longer exist
	sort.Strings(srvNames)
	for name := range computeState {
		k := sort.SearchStrings(srvNames, name)
		if k < 0 || k >= len(srvNames) || srvNames[k] != name {
			delete(computeState, name)
		}
	}

	// update total available MPI resources
	for _, cs := range computeState {
		jsState.MpiRes.Cpu += cs.TotalRes.Cpu
		jsState.MpiRes.Mem += cs.TotalRes.Mem
	}

	// model resources requirements
	mpLst := splitOpts("Common.Models", ",")
	cfgRes = make([]modelCfgRes, 0, len(mpLst))

	for k := range mpLst {

		p := strings.TrimSpace(mpLst[k])
		if p == "" {
			continue // skip empty path
		}
		mp := opts.Int(p+".MemoryProcessMb", 0) // unlimited memory by default
		if mp < 0 {
			mp = 0
		}
		mt := opts.Int(p+".MemoryThreadMb", 0) // unlimited memory by default
		if mt < 0 {
			mt = 0
		}
		cfgRes = append(cfgRes, modelCfgRes{Path: p, ProcessMemMb: mp, ThreadMemMb: mt})
	}

	return jsState, cfgRes
}

// Update computational servers or clusters map.
// For each server or cluster find current state (ready, start, stop or power off),
// total computational resources (cpu and memory), used resources and avaliable resources.
func updateComputeState(
	computeState map[string]computeItem,
	omsActive map[string]omsUsage,
	runCompUsage []runComputeUse,
	nowTs int64,
	maxStartTime int64,
	maxStopTime int64,
	maxComputeErrors int,
	compReadyFiles, compStartFiles, compStopFiles, compErrorFiles, compUsedFiles []string,
) []runComputeUse {

	// clear state for server or cluster: initial state is "" power off
	for name, cs := range computeState {

		cs.State = "" // power off state
		computeState[name] = cs
	}

	// ready compute resources: powered on servers or clusters
	for _, f := range compReadyFiles {

		// get server name
		name := parseCompReadyPath(f)
		if name == "" {
			continue // skip: this is not a compute server ready file
		}
		cs, ok := computeState[name]
		if !ok {
			continue // this server does not exist anymore
		}

		// update server state to ready
		cs.State = "ready"
		cs.ErrorCount = 0

		if cs.LastUsedTs <= 0 {
			cs.LastUsedTs = nowTs
		}
		computeState[name] = cs
	}

	// start up servers or clusters: check startup timeout
	for _, f := range compStartFiles {

		// get server name and time stamp
		name, _, ts := parseCompStatePath(f, "start")
		if name == "" {
			continue // skip: this is not a compute server state file
		}
		cs, ok := computeState[name]
		if !ok {
			continue // this server does not exist anymore
		}

		// update server state to start or detect error
		if (cs.State == "" || cs.State == "start" || cs.State == "ready") && (ts+maxStartTime) >= nowTs {
			cs.State = "start"
		} else {
			cs.ErrorCount++ // start timeout error
		}
		computeState[name] = cs
	}

	// stop servers or clusters: check shutdown timeout
	for _, f := range compStopFiles {

		// get server name and time stamp
		name, _, ts := parseCompStatePath(f, "stop")
		if name == "" {
			continue // skip: this is not a compute server state file
		}
		cs, ok := computeState[name]
		if !ok {
			continue // this server does not exist anymore
		}

		// update server state to stop or detect error
		if (cs.State == "ready" || cs.State == "stop" || cs.State == "") && (ts+maxStopTime) >= nowTs {
			cs.State = "stop"
		} else {
			cs.ErrorCount++ // stop timeout error
		}
		computeState[name] = cs
	}

	// server or cluster in "error" state
	for _, f := range compErrorFiles {

		// get server name and time stamp
		name, _, _ := parseCompStatePath(f, "error")
		if name == "" {
			continue // skip: this is not a compute server state file
		}
		cs, ok := computeState[name]
		if !ok {
			continue // this server does not exist anymore
		}

		cs.ErrorCount++ // error logged for that server or cluster

		computeState[name] = cs
	}

	// set error state for server or cluster if error count exceed max error limit
	for name, cs := range computeState {

		if cs.ErrorCount > maxComputeErrors {
			cs.State = "error"
			computeState[name] = cs
		}
	}

	// servers or clusters used for model runs: sum up resources used by current oms instance and all oms instances
	runCompUsage = runCompUsage[:0]

	for _, f := range compUsedFiles {

		// get server name and oms name
		name, stamp, oms, cpu, mem := parseCompUsedPath(f)
		if name == "" || stamp == "" || oms == "" {
			continue // skip: this is not a compute server state file
		}
		if _, ok := omsActive[oms]; !ok {
			continue // oms instance not active
		}
		cs, ok := computeState[name]
		if !ok {
			continue // this server does not exist anymore
		}

		// update resources used by model runs
		cs.UsedRes.Cpu += cpu
		cs.UsedRes.Mem += mem
		if oms == theCfg.omsName {
			cs.OwnRes.Cpu += cpu
			cs.OwnRes.Mem += mem
		}
		if cs.LastUsedTs < nowTs {
			cs.LastUsedTs = nowTs
		}
		computeState[name] = cs

		// if this oms instance has global admin rigths then
		// update computational resouces usage by model runs
		if theCfg.isAdminAll {

			runCompUsage = append(runCompUsage, runComputeUse{
				Oms:         oms,
				SubmitStamp: stamp,
				computeUse: computeUse{
					CompName:   name,
					ComputeRes: ComputeRes{Cpu: cpu, Mem: mem},
					filePath:   f,
				},
			})
		}
	}

	return runCompUsage
}
