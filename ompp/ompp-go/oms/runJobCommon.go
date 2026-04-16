// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// jobDirValid checking job control configuration
// and return flags: is job control enabled, is past sub-directory exists, is disk usage control enabled.
// if job control directory is empty then job control disabled.
// if job control directory not empty then it must have active, state, queue, history subdirectories.
func jobDirValid(jobDir string) (bool, bool, error) {

	if jobDir == "" {
		return false, false, nil // job control disabled
	}
	if !helper.IsDirExist(jobDir) ||
		!helper.IsDirExist(filepath.Join(jobDir, "active")) || !helper.IsDirExist(filepath.Join(jobDir, "state")) ||
		!helper.IsDirExist(filepath.Join(jobDir, "queue")) || !helper.IsDirExist(filepath.Join(jobDir, "history")) {
		return false, false, nil
	}
	isPast := helper.IsDirExist(filepath.Join(jobDir, "past"))

	return true, isPast, nil
}

// Return job control file path if model is running now.
// For example: 2022_07_08_23_03_27_555-#-_4040-#-RiskPaths-#-d90e1e9a-#-2022_07_04_20_06_10_818-#-mpi-#-cpu-#-8-#-mem-#-4-#-8888.json
func jobActivePath(submitStamp, modelName, modelDigest string, runStamp string, isMpi bool, pid int, cpu int, mem int) string {

	ml := "local"
	if isMpi {
		ml = "mpi"
	}
	return filepath.Join(
		theCfg.jobDir,
		"active",
		submitStamp+"-#-"+theCfg.omsName+"-#-"+modelName+"-#-"+modelDigest+"-#-"+runStamp+"-#-"+ml+"-#-cpu-#-"+strconv.Itoa(cpu)+"-#-mem-#-"+strconv.Itoa(mem)+"-#-"+strconv.Itoa(pid)+".json")
}

// Return path job control file path if model run standing is queue.
// For example: 2022_07_05_19_55_38_111-#-_4040-#-RiskPaths-#-d90e1e9a-#-mpi-#-cpu-#-2-#-8-#-mem-#-32-#-512-#-20220817.json
func jobQueuePath(submitStamp, modelName, modelDigest string, isMpi bool, position int, procCpu, threadCpu, procMem, threadMem int) string {

	ml := "local"
	if isMpi {
		ml = "mpi"
	}
	return filepath.Join(
		theCfg.jobDir,
		"queue",
		submitStamp+"-#-"+theCfg.omsName+"-#-"+modelName+"-#-"+modelDigest+"-#-"+ml+
			"-#-cpu-#-"+strconv.Itoa(procCpu)+"-#-"+strconv.Itoa(threadCpu)+"-#-mem-#-"+strconv.Itoa(procMem)+"-#-"+strconv.Itoa(threadMem)+
			"-#-"+strconv.Itoa(position)+".json")
}

// Return job control file path to completed model with run status suffix.
// For example: job/history/2022_07_04_20_06_10_817-#-_4040-#-RiskPaths-#-d90e1e9a-#-2022_07_04_20_06_10_818-#-success.json
func jobHistoryPath(status string, isKill bool, submitStamp, modelName, modelDigest, runStamp string) string {
	return jobHistoryOmsPath(status, isKill, submitStamp, theCfg.omsName, modelName, modelDigest, runStamp)
}

// Return job control file path to completed model with run status suffix.
// For example: job/history/2022_07_04_20_06_10_817-#-_4040-#-RiskPaths-#-d90e1e9a-#-2022_07_04_20_06_10_818-#-success.json
func jobHistoryOmsPath(status string, isKill bool, submitStamp, oms, modelName, modelDigest, runStamp string) string {

	st := db.NameOfRunStatus(status)
	if isKill && status == db.ErrorRunStatus {
		st = "kill"
	}
	if oms == "" {
		oms = theCfg.omsName
	}
	return filepath.Join(
		theCfg.jobDir,
		"history",
		submitStamp+"-#-"+oms+"-#-"+modelName+"-#-"+modelDigest+"-#-"+runStamp+"-#-"+st+".json")
}

// Return parts of job control shadow history file path: past folder, month sub-folder and file name.
// For example:
// job/past, 2022_07, 2022_07_08_23_03_27_555-#-_4040-#-RiskPaths-#-d90e1e9a-#-2022_07_04_20_06_10_818-#-mpi-#-cpu-#-8-#-mem-#-4-#-3600.json
func jobPastPath(status string, isKill bool, submitStamp, modelName, modelDigest, runStamp string, isMpi bool, cpu int, mem int, totalSec int64) (string, string, string) {
	return jobPastOmsPath(status, isKill, submitStamp, theCfg.omsName, modelName, modelDigest, runStamp, isMpi, cpu, mem, totalSec)
}

// Return parts of job control shadow history file path: past folder, month sub-folder and file name.
// For example:
// job/past, 2022_07, 2022_07_08_23_03_27_555-#-_4040-#-RiskPaths-#-d90e1e9a-#-2022_07_04_20_06_10_818-#-mpi-#-cpu-#-8-#-mem-#-4-#-3600.json
func jobPastOmsPath(status string, isKill bool, submitStamp, oms, modelName, modelDigest, runStamp string, isMpi bool, cpu int, mem int, totalSec int64) (string, string, string) {

	d := ""
	if len(submitStamp) >= 7 {
		d = submitStamp[:7]
	}
	if oms == "" {
		oms = theCfg.omsName
	}
	st := db.NameOfRunStatus(status)
	if isKill && status == db.ErrorRunStatus {
		st = "kill"
	}
	ml := "local"
	if isMpi {
		ml = "mpi"
	}

	return filepath.Join(theCfg.jobDir, "past"),
		d,
		submitStamp + "-#-" + oms + "-#-" + modelName + "-#-" + modelDigest + "-#-" + runStamp + "-#-" +
			ml + "-#-cpu-#-" + strconv.Itoa(cpu) + "-#-mem-#-" + strconv.Itoa(mem) + "-#-" + strconv.FormatInt(totalSec, 10) + "-#-" + st + ".json"
}

// Return job state file path e.g.: job/state/_4040.json
func jobStatePath() string {
	return filepath.Join(theCfg.jobDir, "state", theCfg.omsName+".json")
}

// Return this oms instance job queue paused file path e.g.: job/state/jobs.queue-#-_4040-#-paused
func jobQueuePausedPath(oms string) string {
	return filepath.Join(theCfg.jobDir, "state", "jobs.queue-#-"+oms+"-#-paused")
}

// return all job queue paused file path e.g.: job/state/jobs.queue.all.paused
func jobAllQueuePausedPath() string {
	return filepath.Join(theCfg.jobDir, "state", "jobs.queue.all.paused")
}

// Return compute server or cluster ready file path: job/state/comp-ready-#-name
func compReadyPath(name string) string {
	return filepath.Join(theCfg.jobDir, "state", "comp-ready-#-"+name)
}

// Return server used by model run path prefix and file path.
// For example: job/state/comp-used-#-name-#-2022_07_08_23_03_27_555-#-_4040-#-cpu-#-4-#-mem-#-8
func compUsedPath(name, submitStamp string, cpu, mem int) string {
	return filepath.Join(
		theCfg.jobDir,
		"state",
		"comp-used-#-"+name+"-#-"+submitStamp+"-#-"+theCfg.omsName+"-#-cpu-#-"+strconv.Itoa(cpu)+"-#-mem-#-"+strconv.Itoa(mem))
}

// Return disk state use file path: oms instance name, usage in MBytes, status (over / ok), user limit in MBytes, time stamp and clock ticks.
// For example: job/state/disk-#-_4040-#-size-#-100-#-ok-#-120-#-2022_07_08_23_45_12_123-#-125678.json
func diskUseStatePath(totalSize int64, isOver bool, userLimit int64, tickMs int64) string {

	s := "over"
	if !isOver {
		s = "ok"
	}
	mb := int(math.Ceil(float64(totalSize) / (1024.0 * 1024.0)))
	ul := int(math.Floor(float64(userLimit) / (1024.0 * 1024.0)))

	tPart := helper.MakeTimeStamp(time.UnixMilli(tickMs)) + "-#-" + strconv.FormatInt(tickMs, 10)

	return filepath.Join(
		theCfg.jobDir,
		"state",
		"disk-#-"+theCfg.omsName+"-#-size-#-"+strconv.Itoa(mb)+"-#-"+s+"-#-"+strconv.Itoa(ul)+"-#-"+tPart+".json")
}

// parse job file path or job file name:
// remove .json extension and directory prefix.
// Return submission stamp, oms instance name, model name, model digest and the rest of the file name
func parseJobPath(srcPath string) (string, string, string, string, string) {

	// remove job directory and extension, file extension must be .json
	if filepath.Ext(srcPath) != ".json" {
		return "", "", "", "", ""
	}
	p := filepath.Base(srcPath)
	p = p[:len(p)-len(".json")]

	// check result: it must be at least 4 non-empty parts and first must be a time stamp
	sp := strings.SplitN(p, "-#-", 5)
	if len(sp) < 4 || !helper.IsUnderscoreTimeStamp(sp[0]) || sp[1] == "" || sp[2] == "" || sp[3] == "" {
		return "", "", "", "", "" // source file path is not job file
	}

	if len(sp) == 4 {
		return sp[0], sp[1], sp[2], sp[3], "" // only 4 parts, the rest of source file name is empty
	}
	return sp[0], sp[1], sp[2], sp[3], sp[4]
}

// Parse active file path or active file name.
// Return submission stamp, oms instance name, model name, digest, run stamp, MPI or local, cpu count, memory size and process id.
// For example: 2022_07_08_23_03_27_555-#-_4040-#-RiskPaths-#-d90e1e9a-#-2022_07_04_20_06_10_818-#-mpi-#-cpu-#-8-#-mem-#-4-#-8888.json
func parseActivePath(srcPath string) (string, string, string, string, string, bool, int, int, int) {

	// parse common job file part
	subStamp, oms, mn, dgst, p := parseJobPath(srcPath)

	if subStamp == "" || oms == "" || mn == "" || dgst == "" || p == "" {
		return subStamp, oms, "", "", "", false, 0, 0, 0 // source file path is not active or queue job file
	}

	// parse cpu count and memory size, 6 parts expected
	sp := strings.Split(p, "-#-")
	if len(sp) != 7 ||
		!helper.IsUnderscoreTimeStamp(sp[0]) ||
		(sp[1] != "mpi" && sp[1] != "local") ||
		sp[2] != "cpu" || sp[3] == "" ||
		sp[4] != "mem" || sp[5] == "" ||
		sp[6] == "" {
		return subStamp, oms, "", "", "", false, 0, 0, 0 // source file path is not active or queue job file
	}
	isMpi := sp[1] == "mpi"

	// parse and convert cpu count, memory size and process pid
	cpu, err := strconv.Atoi(sp[3])
	if err != nil || cpu <= 0 {
		return subStamp, oms, "", "", "", false, 0, 0, 0 // cpu count must be positive integer
	}
	mem, err := strconv.Atoi(sp[5])
	if err != nil || mem < 0 {
		return subStamp, oms, "", "", "", false, 0, 0, 0 // memory size must be non-negative integer
	}
	pid, err := strconv.Atoi(sp[6])
	if err != nil || pid <= 0 {
		return subStamp, oms, "", "", "", false, 0, 0, 0 // process pid must be positive integer
	}

	return subStamp, oms, mn, dgst, sp[0], isMpi, cpu, mem, pid
}

// Parse queue file path or queue file name.
// Return submission stamp, oms instance name, model name, digest, MPI or local,
// process count, thread count, process memory size im MBytes, thread memory size im MBytes,
// and job position in queue.
// For example: 2022_07_05_19_55_38_111-#-_4040-#-RiskPaths-#-d90e1e9a-#-mpi-#-cpu-#-2-#-8-#-mem-#-32-#-512-#-20220817.json
func parseQueuePath(srcPath string) (string, string, string, string, bool, int, int, int, int, int) {

	// parse common job file part
	subStamp, oms, mn, dgst, p := parseJobPath(srcPath)

	if subStamp == "" || oms == "" || mn == "" || dgst == "" || p == "" {
		return subStamp, oms, "", "", false, 0, 0, 0, 0, 0 // source file path is not active or queue job file
	}

	// parse MPI or local flag, cpu count and memory size and queue position, 8 parts expected
	sp := strings.Split(p, "-#-")
	if len(sp) != 8 ||
		(sp[0] != "mpi" && sp[0] != "local") ||
		sp[1] != "cpu" || sp[2] == "" || sp[3] == "" ||
		sp[4] != "mem" || sp[5] == "" || sp[6] == "" ||
		sp[7] == "" {
		return subStamp, oms, "", "", false, 0, 0, 0, 0, 0 // source file path is not active or queue job file
	}
	isMpi := sp[0] == "mpi"

	// parse and convert process count, thread count, process memory size and thread memory size
	nProc, err := strconv.Atoi(sp[2])
	if err != nil || nProc <= 0 {
		return subStamp, oms, "", "", false, 0, 0, 0, 0, 0 // process count must be positive integer
	}
	nTh, err := strconv.Atoi(sp[3])
	if err != nil || nTh <= 0 {
		return subStamp, oms, "", "", false, 0, 0, 0, 0, 0 // thread count must be positive integer
	}
	procMem, err := strconv.Atoi(sp[5])
	if err != nil || procMem < 0 {
		return subStamp, oms, "", "", false, 0, 0, 0, 0, 0 // process memory size must be non-negative integer
	}
	thMem, err := strconv.Atoi(sp[6])
	if err != nil || thMem < 0 {
		return subStamp, oms, "", "", false, 0, 0, 0, 0, 0 // memory size must be non-negative integer
	}
	pos, err := strconv.Atoi(sp[7])
	if err != nil || pos < 0 {
		return subStamp, oms, "", "", false, 0, 0, 0, 0, 0 // position must be non-negative integer
	}

	return subStamp, oms, mn, dgst, isMpi, nProc, nTh, procMem, thMem, pos
}

// Parse history file path or history file name
// Return submission stamp, oms instance name, model name, digest, run stamp and run status.
// For example: 2022_07_04_20_06_10_817-#-_4040-#-RiskPaths-#-d90e1e9a-#-2022_07_04_20_06_10_818-#-success.json
func parseHistoryPath(srcPath string) (string, string, string, string, string, string) {

	// parse common job file part
	subStamp, oms, mn, dgst, p := parseJobPath(srcPath)

	if subStamp == "" || oms == "" || mn == "" || dgst == "" || len(p) < len("r-#-s") {
		return subStamp, oms, "", "", "", "" // source file path is not history job file
	}

	// get run stamp and status
	sp := strings.Split(p, "-#-")
	if len(sp) != 2 || sp[0] == "" || sp[1] == "" {
		return subStamp, oms, "", "", "", "" // source file path is not history job file
	}

	return subStamp, oms, mn, dgst, sp[0], sp[1]
}

// Parse past shadow history file path and return:
// year-month sub-folder,
// submission stamp, oms instance name, model name, digest, run stamp,
// MPI or local, cpu cores, memory size in GBytes, run time in seconds and run status.
// For example:
// 2022_07/2022_07_08_23_03_27_555-#-_4040-#-RiskPaths-#-d90e1e9a-#-2022_07_04_20_06_10_818-#-mpi-#-cpu-#-8-#-mem-#-4-#-3600-#-success.json
func parsePastPath(srcPath string) (string, string, string, string, string, string, bool, int, int, int64, string) {

	// split month sub-folder and file name
	dir, fName := filepath.Split(srcPath)
	if dir == "" || fName == "" {
		return "", "", "", "", "", "", false, 0, 0, 0, "" // source file path is not shadow history job file
	}

	// bottom directory must be year and month: yyyy_mm, for example: 2022_07
	ym := filepath.Base(dir)
	if len(ym) != 7 || ym[4] != '_' {
		return "", "", "", "", "", "", false, 0, 0, 0, "" // source file parent folder is not yyyy_mm
	}

	// parse common job file part
	subStamp, oms, mn, dgst, p := parseJobPath(fName)

	if subStamp == "" || oms == "" || mn == "" || dgst == "" || len(p) < len("r-#-M-#-c-#-m-#-t-#-s") {
		return ym, subStamp, oms, "", "", "", false, 0, 0, 0, "" // source file path is not shadow history job file
	}

	// parse the rest of file name, it old format, same as history file, 2 parts only:
	//   run stamp run status
	// or new format, 8 parts:
	//   stamp, MPI or local, cpu cores, memory size in GBytes, run time in seconds and run status
	//   for example:
	//   2022_07_04_20_06_10_818-#-mpi-#-cpu-#-8-#-mem-#-4-#-3600-#-success
	rStamp := ""
	status := ""
	isMpi := false
	cpu := 0
	mem := 0
	var tSec int64 = 0
	var err error

	sp := strings.Split(p, "-#-")
	if len(sp) != 8 && len(sp) != 2 {
		return ym, subStamp, oms, "", "", "", false, 0, 0, 0, "" // source file path is not shadow history job file
	}

	if len(sp) == 2 { // file name is in history format

		rStamp = sp[0]
		status = sp[1]

	} else { // file name is in past history format

		if sp[0] == "" ||
			(sp[1] != "mpi" && sp[1] != "local") ||
			sp[2] != "cpu" || sp[3] == "" ||
			sp[4] != "mem" || sp[5] == "" ||
			sp[6] == "" || sp[7] == "" {
			return ym, subStamp, oms, "", "", "", false, 0, 0, 0, "" // source file path is not shadow history job file
		}
		rStamp = sp[0]
		status = sp[7]
		isMpi = sp[1] == "mpi"

		// convert cpu cores, memory size and total run time
		cpu, err = strconv.Atoi(sp[3])
		if err != nil || cpu <= 0 {
			return ym, subStamp, oms, "", "", "", false, 0, 0, 0, "" // cpu count must be positive integer
		}
		mem, err = strconv.Atoi(sp[5])
		if err != nil || mem < 0 {
			return ym, subStamp, oms, "", "", "", false, 0, 0, 0, "" // memory size must be non-negative integer
		}
		tSec, err = strconv.ParseInt(sp[6], 10, 64)
		if err != nil || tSec < 0 {
			return ym, subStamp, oms, "", "", "", false, 0, 0, 0, "" // total run time must be non-negative integer
		}
	}

	if rStamp == "" || status == "" {
		return ym, subStamp, oms, "", "", "", false, 0, 0, 0, "" // source file path is not history job file
	}

	// month sub-folder,
	// submission stamp, oms instance name, model name, digest, run stamp,
	// MPI or local, cpu cores, memory size in GBytes, run time in seconds and run status.
	return ym, subStamp, oms, mn, dgst, rStamp, isMpi, cpu, mem, tSec, status
}

// Parse oms heart beat tick file path: job/state/oms-#-_4040-#-2022_07_08_23_45_12_123-#-1257894000000-#-2022_08_17_21_56_34_321
// Return oms instance name time stamp, clock ticks and last run stamp.
// If oms instance file does not have last run stamp then use current date-time stamp
func parseOmsTickPath(srcPath string) (string, string, int64, string) {

	p := filepath.Base(srcPath) // remove job state directory

	// split file name and check result: it must be 5 non-empty parts with time stamp and last run stamp
	sp := strings.Split(p, "-#-")
	if len(sp) < 4 || sp[0] != "oms" ||
		sp[1] == "" || !helper.IsUnderscoreTimeStamp(sp[2]) || sp[3] == "" {
		return "", "", 0, "" // source file path is not oms heart beat file
	}

	// check if there last run stamp in file name, use current date-time stamp if not
	lastRunStamp := ""
	if len(sp) == 5 {
		if !helper.IsUnderscoreTimeStamp(sp[4]) {
			return "", "", 0, "" // source file path is not oms heart beat file
		}
		lastRunStamp = sp[4]
	} else {
		lastRunStamp = helper.MakeTimeStamp(time.Now())
	}

	// convert clock ticks
	tickMs, err := strconv.ParseInt(sp[3], 10, 64)
	if err != nil || tickMs <= minJobTickMs {
		return "", "", 0, "" // clock ticks must after 2020-08-17 23:45:59
	}

	return sp[1], sp[2], tickMs, lastRunStamp
}

// Parse oms instance job queue paused file path e.g.: job/state/jobs.queue-#-_4040-#-paused
// Return oms instance name.
func parseQueuePausedPath(srcPath string) string {

	p := filepath.Base(srcPath) // remove job state directory

	// split file name and check result: it must be 3 non-empty parts
	sp := strings.Split(p, "-#-")
	if len(sp) != 3 || sp[0] != "jobs.queue" || sp[1] == "" || sp[2] != "paused" {
		return "" // source file path is not job queue paused file
	}

	return sp[1]
}

// parse compute server or cluster ready file path and return server name, e.g.: job/state/comp-ready-#-name
func parseCompReadyPath(srcPath string) string {

	p := filepath.Base(srcPath) // remove job directory

	// split file name and check result: it must be 2 non-empty parts
	sp := strings.Split(p, "-#-")
	if len(sp) != 2 || sp[0] != "comp-ready" || sp[1] == "" {
		return "" // source file path is not compute server ready path
	}

	return sp[1]
}

// parse compute server or cluster state file path and return server name, time stamp and clock ticks.
// State must be one of: start, stop, error.
// For example: job/state/comp-start-#-name-#-2022_07_08_23_45_12_123-#-1257894000000
func parseCompStatePath(srcPath, state string) (string, string, int64) {

	p := filepath.Base(srcPath) // remove job directory

	// split file name and check result: it must be 4 non-empty parts and with time stamp
	sp := strings.Split(p, "-#-")
	if len(sp) != 4 || sp[0] != "comp-"+state || sp[1] == "" || !helper.IsUnderscoreTimeStamp(sp[2]) || sp[3] == "" {
		return "", "", 0 // source file path is not compute server state path
	}

	// convert clock ticks
	tickMs, err := strconv.ParseInt(sp[3], 10, 64)
	if err != nil || tickMs <= minJobTickMs {
		return "", "", 0 // clock ticks must after 2020-08-17 23:45:59
	}

	return sp[1], sp[2], tickMs
}

// Parse compute server used by model run file path
// Return server name, submission stamp, oms instance name, cpu count and memory size.
// For example: job/state/comp-used-#-name-#-2022_07_08_23_03_27_555-#-_4040-#-cpu-#-4-#-mem-#-8
func parseCompUsedPath(srcPath string) (string, string, string, int, int) {

	p := filepath.Base(srcPath) // remove job directory

	// split file name and check result: it must be 8 non-empty parts
	// and include submission stamp, cpu count and memory size
	sp := strings.Split(p, "-#-")
	if len(sp) != 8 || sp[0] != "comp-used" ||
		sp[1] == "" || !helper.IsUnderscoreTimeStamp(sp[2]) || sp[3] == "" ||
		sp[4] != "cpu" || sp[5] == "" || sp[6] != "mem" || sp[7] == "" {
		return "", "", "", 0, 0 // source file path is not compute server used by model run path
	}

	// parse and convert cpu count and memory size
	cpu, err := strconv.Atoi(sp[5])
	if err != nil || cpu <= 0 {
		return "", "", "", 0, 0 // cpu count must be positive integer
	}
	mem, err := strconv.Atoi(sp[7])
	if err != nil || mem < 0 {
		return "", "", "", 0, 0 // memory size must be non-negative integer
	}

	return sp[1], sp[2], sp[3], cpu, mem
}

// Parse disk state use file path: job/state/disk-#-_4040-#-size-#-100-#-ok-#-120-#-2022_07_08_23_45_12_123-#-125678.json
// Return oms instance name, total MB usage, is over status, user limit MB, time stamp and clock ticks, total bytes usage, user limit in bytes
func parseDiskUseStatePath(srcPath string) (string, int64, bool, int64, string, int64, int64, int64) {

	// remove job state directory and extension, file extension must be .json
	if filepath.Ext(srcPath) != ".json" {
		return "", 0, false, 0, "", 0, 0, 0
	}
	p := filepath.Base(srcPath)
	p = p[:len(p)-len(".json")]

	// split file name and check result: it must be 8 non-empty parts with time stamp
	sp := strings.Split(p, "-#-")
	if len(sp) != 8 ||
		sp[0] != "disk" || sp[1] == "" ||
		sp[2] != "size" || sp[3] == "" ||
		sp[4] == "" || sp[5] == "" ||
		!helper.IsUnderscoreTimeStamp(sp[6]) || sp[7] == "" {
		return "", 0, false, 0, "", 0, 0, 0 // source file path is not disk use state file
	}

	// parse total size MBytes
	mb, err := strconv.ParseInt(sp[3], 10, 64)
	if err != nil || mb < 0 {
		return "", 0, false, 0, "", 0, 0, 0 // disk usage must be non-negative integer
	}

	// check status: it must be "ok" or "over"
	isOver := false
	if sp[4] == "over" {
		isOver = true
	} else {
		if sp[4] != "ok" {
			return "", 0, false, 0, "", 0, 0, 0 // status it must be "ok" or "over"
		}
	}

	// parse user limit MBytes
	ul, err := strconv.ParseInt(sp[5], 10, 64)
	if err != nil || ul < 0 {
		return "", 0, false, 0, "", 0, 0, 0 // user limit must be non-negative integer
	}

	// convert clock ticks
	tickMs, err := strconv.ParseInt(sp[7], 10, 64)
	if err != nil || tickMs <= minJobTickMs {
		return "", 0, false, 0, "", 0, 0, 0 // clock ticks must after 2020-08-17 23:45:59
	}

	return sp[1], mb, isOver, ul, sp[6], tickMs, mb * 1024 * 1024, ul * 1024 * 1024
}

// move run job to active state from queue
func moveJobToActive(queueJobPath string, rState *RunState, res RunRes, runStamp, binDir, workDir, iniPath, hfPath, cmdLine string) (string, bool) {
	if !theCfg.isJobControl {
		return "", true // job control disabled
	}

	// read run request from job queue
	var jc RunJob
	isOk, err := helper.FromJsonFile(queueJobPath, &jc)
	if err != nil {
		omppLog.Log(err)
	}
	if !isOk || err != nil {
		fileDeleteAndLog(true, queueJobPath) // invalid file content: remove job control file from queue
		return "", false
	}

	// make absolute path to log file
	if jc.LogPath, err = filepath.Abs(rState.logPath); err != nil { // on error keep log absolute path empty
		jc.LogPath = ""
	}
	// add run stamp, process info, actual run resources and move job control file into active
	jc.RunStamp = runStamp
	jc.Pid = rState.pid
	jc.CmdPath = rState.cmdPath
	jc.CmdLine = cmdLine
	jc.Res = res
	jc.LogFileName = rState.LogFileName
	jc.BinDir = binDir
	jc.WorkDir = workDir
	jc.IniPath = iniPath
	jc.HostFilePath = hfPath

	dst := jobActivePath(rState.SubmitStamp, rState.ModelName, rState.ModelDigest, runStamp, jc.IsMpi, rState.pid, jc.Res.Cpu, jc.Res.Mem)

	fileDeleteAndLog(false, queueJobPath) // remove job control file from queue

	err = helper.ToJsonIndentFile(dst, &jc)
	if err != nil {
		omppLog.Log(err)
		fileDeleteAndLog(true, dst) // on error remove file, if any file created
		return "", false
	}

	return dst, true
}

// move active model run job control file to history
func moveActiveJobToHistory(activePath, status string, isKill bool, submitStamp, modelName, modelDigest, runStamp string, cmdStart, cmdStop int64) bool {
	if !theCfg.isJobControl {
		return true // job control disabled
	}

	// move active job file to history
	hst := jobHistoryPath(status, isKill, submitStamp, modelName, modelDigest, runStamp)

	isOk := fileMoveAndLog(false, activePath, hst)
	if !isOk {
		fileDeleteAndLog(true, activePath) // if move failed then delete job control file from active list
	}

	// remove all compute server usage files
	// for example: job/state/comp-used-#-name-#-2022_07_08_23_03_27_555-#-_4040-#-cpu-#-4-#-mem-#-8
	ptrn := filepath.Join(theCfg.jobDir, "state") + string(filepath.Separator) + "comp-used-#-*-#-" + submitStamp + "-#-" + theCfg.omsName + "-#-cpu-#-*-#-mem-#-*"

	if fLst, err := filepath.Glob(ptrn); err == nil {
		for _, f := range fLst {
			fileDeleteAndLog(false, f)
		}
	}

	if isOk && theCfg.isJobPast { // copy to the shadow history path

		// check start and stop time
		if cmdStart < minCmdTimeSec {
			cmdStart = 0
		}
		if cmdStop < cmdStart {
			cmdStart = 0
			cmdStop = 0
		}
		totalSec := cmdStop - cmdStart

		// get resources from active job file name
		stamp, _, mn, dgst, _, isMpi, cpu, mem, _ := parseActivePath(activePath)

		if stamp != submitStamp || mn != modelName || dgst != modelDigest || cpu < 0 || mem < 0 {
			return false // active job file name is incorrect
		}

		pastDir, monthDir, fn := jobPastPath(status, isKill, submitStamp, modelName, modelDigest, runStamp, isMpi, cpu, mem, totalSec)
		d := filepath.Join(pastDir, monthDir)

		if e := os.MkdirAll(d, 0750); e != nil {
			omppLog.Log(e)
			return false // unable to create output directory
		}

		p := filepath.Join(d, fn)
		if !fileCopy(false, hst, p) {
			return false // fail to copy job file into the past
		}
		if cmdStart < minCmdTimeSec || cmdStop < cmdStart || totalSec < 0 {
			return true // done with past file, it is copied into past directory
		}

		//  add usage time to the file
		//
		var jc RunJob
		ok, err := helper.FromJsonFile(p, &jc) // read completed job info
		if err != nil {
			omppLog.Log(err)
			return false // unexpected json in the past file
		}
		if !ok {
			return true // done with past file, it is copied into past directory
		}

		sd := ""
		if totalSec >= 0 { // format duration to 12:04:05 or to 04:05

			h := totalSec / 3600
			m := (totalSec % 3600) / 60
			s := totalSec % 60

			if h > 0 {
				sd = fmt.Sprintf("%d:%02d:%02d", h, m, s)
			} else {

				sd = fmt.Sprintf("%02d:%02d", m, s)
			}
		}

		// format start and stop time if it is not an empty values
		sb := ""
		se := ""
		if cmdStart > minCmdTimeSec {
			sb = helper.MakeDateTime(time.Unix(cmdStart, 0))
		}
		if cmdStop >= cmdStart {
			se = helper.MakeDateTime(time.Unix(cmdStop, 0))
		}

		pj := PastRunJob{
			RunJob:    jc,
			Submitted: helper.FromUnderscoreTimeStamp(submitStamp),
			Started:   sb,
			Completed: se,
			Duration:  sd,
		}

		err = helper.ToJsonIndentFile(p, &pj) // save result to the past job file
		if err != nil {
			omppLog.Log(err)
		}
	}

	return isOk
}

// move model run request from queue to error if model run fail to start
func moveJobQueueToFailed(queuePath string, submitStamp, modelName, modelDigest, runStamp string, isKill bool) bool {
	return moveJobQueueOmsToFailed(queuePath, submitStamp, theCfg.omsName, modelName, modelDigest, runStamp, isKill)
}

// move model run request from queue to error if model run fail to start
func moveJobQueueOmsToFailed(queuePath string, submitStamp, oms, modelName, modelDigest, runStamp string, isKill bool) bool {
	if !theCfg.isJobControl || queuePath == "" {
		return true // job control disabled
	}
	if !helper.IsUnderscoreTimeStamp(runStamp) {
		runStamp = submitStamp
	}
	if oms == "" {
		oms = theCfg.omsName
	}

	hst := jobHistoryOmsPath(db.ErrorRunStatus, isKill, submitStamp, oms, modelName, modelDigest, runStamp)

	if !fileMoveAndLog(true, queuePath, hst) {
		fileDeleteAndLog(true, queuePath) // if move failed then delete job control file from queue
		return false
	} else {

		if theCfg.isJobPast { // copy to the shadow history path

			stamp, qOms, mn, dgst, isMpi, procCount, thCount, procMem, thMem, _ := parseQueuePath(queuePath)
			if stamp != submitStamp || mn != modelName || dgst != modelDigest {
				return false // file name is not a job queue file name
			}
			if !theCfg.isAdminAll && (oms != qOms || oms != theCfg.omsName) {
				omppLog.Log("Queue oms name must be the same as current instance oms name:", qOms, theCfg.omsName, oms)
				return false // queue oms is not
			}

			pastDir, monthDir, fn := jobPastOmsPath(db.ErrorRunStatus, isKill, submitStamp, oms, modelName, modelDigest, runStamp, isMpi, procCount*thCount, (procMem + thCount*thMem), 0)
			d := filepath.Join(pastDir, monthDir)

			if os.MkdirAll(d, 0750) == nil {
				fileCopy(false, hst, filepath.Join(d, fn))
			}
		}
	}
	return true
}

// read run title from job json file: return run name or task run name or workset name
func getJobRunTitle(filePath string) string {
	if !theCfg.isJobControl {
		return "" // job control disabled
	}

	// read run request from job queue
	var jc RunJob
	isOk, err := helper.FromJsonFile(filePath, &jc)
	if err != nil {
		omppLog.Log(err)
	}
	if !isOk || err != nil {
		return ""
	}

	// find run name or task run or workset name in model run options
	runName := ""
	taskRunName := ""
	wsName := ""
	for krq, val := range jc.Opts {

		// remove "-" from command line argument key ex: "-OpenM.Threads"
		key := krq
		if krq[0] == '-' {
			key = krq[1:]
		}

		if strings.EqualFold(key, "OpenM.RunName") {
			runName = val
		}
		if strings.EqualFold(key, "OpenM.TaskRunName") {
			taskRunName = val
		}
		if strings.EqualFold(key, "OpenM.SetName") {
			wsName = val
		}
	}

	if runName != "" {
		return runName
	}
	if taskRunName != "" {
		return taskRunName
	}
	return wsName
}

// Remove all existing oms heart beat tick files and create new oms heart beat tick file with current timestamp and last run stamp.
// Return oms heart beat file path and true is file created successfully.
// For example: job/state/oms-#-_4040-#-2022_07_08_23_45_12_123-#-1257894000000-#-2022_08_17_21_56_34_321
func makeOmsTick(lastRunStamp string) (string, bool) {

	// create new oms heart beat tick file
	p := filepath.Join(theCfg.jobDir, "state", "oms-#-"+theCfg.omsName)
	tNow := time.Now()
	ts := helper.MakeTimeStamp(tNow)
	if lastRunStamp == "" {
		lastRunStamp = ts
	}
	fnow := p + "-#-" + ts + "-#-" + strconv.FormatInt(tNow.UnixMilli(), 10) + "-#-" + lastRunStamp

	isOk := fileCreateEmpty(false, fnow)

	// delete existing heart beat files for our oms instance
	omsFiles := filesByPattern(p+"-#-*-#-*", "Error at oms heart beat files search")
	for _, f := range omsFiles {
		if f != fnow {
			fileDeleteAndLog(false, f)
		}
	}
	return fnow, isOk
}

// Create new compute server file with current timestamp.
// For example: job/state/comp-start-#-name-#-2022_07_08_23_45_12_123-#-1257894000000
func createCompStateFile(name, state string) string {
	if !theCfg.isJobControl {
		return "" // job control disabled
	}

	ts := time.Now()
	fp := filepath.Join(
		theCfg.jobDir,
		"state",
		"comp-"+state+"-#-"+name+"-#-"+helper.MakeTimeStamp(ts)+"-#-"+strconv.FormatInt(ts.UnixMilli(), 10))

	if !fileCreateEmpty(false, fp) {
		fp = ""
	}
	return fp
}

// Remove all existing compute server state files, log delete errors, return true on success or false or errors.
// Create error state file if any delete failed.
// For example: job/state/comp-error-#-name-#-2022_07_08_23_45_12_123-#-1257894000000
func deleteCompStateFiles(name, state string) bool {

	p := filepath.Join(theCfg.jobDir, "state", "comp-"+state+"-#-"+name)

	isNoError := true

	fl := filesByPattern(p+"-#-*-#-*", "Error at server state files search")
	for _, f := range fl {
		isOk := fileDeleteAndLog(false, f)
		isNoError = isNoError && isOk
		if !isOk {
			createCompStateFile(name, "error")
			omppLog.Log("FAILED to delete state file: ", state, " ", name, " ", p)
		}
	}

	return isNoError
}

// Return true if jobs queue processing is paused for this oms instance
func isPausedJobQueue() bool {
	return helper.IsFileExist(jobQueuePausedPath(theCfg.omsName)) || helper.IsFileExist(jobAllQueuePausedPath())
}

// Return true if jobs queue processing is paused for all oms instances
func isPausedJobAllQueue() bool {
	return helper.IsFileExist(jobAllQueuePausedPath())
}

// read job control state from the file, return empty state on error or if state file not exist
func jobStateRead() (*jobControlState, bool) {

	var jcs jobControlState
	isOk, err := helper.FromJsonFile(jobStatePath(), &jcs)
	if err != nil {
		omppLog.Log(err)
	}
	if !isOk || err != nil {
		return &jobControlState{Queue: []string{}}, false
	}
	return &jcs, true
}

// save job control state into the file, return false on error
func jobStateWrite(jsc jobControlState) bool {
	if !theCfg.isJobControl {
		return false // job control disabled
	}

	err := helper.ToJsonIndentFile(jobStatePath(), jsc)
	if err != nil {
		omppLog.Log(err)
		return false
	}
	return true
}

// save storage usage state into the file, return false on error
func diskUseStateWrite(duState *diskUseState, dbUse []dbDiskUse) bool {
	if !theCfg.isDiskUse || !theCfg.isJobControl {
		return false // job control disabled
	}

	deleteDiskUseStateFiles() // delete all existing disk use state files for current instance

	ds := struct {
		diskUseState
		DbUse []dbDiskUse
	}{
		diskUseState: *duState,
		DbUse:        dbUse,
	}

	err := helper.ToJsonIndentFile(
		diskUseStatePath(duState.TotalSize, duState.IsOver, duState.Limit, duState.UpdateTs),
		&ds)
	if err != nil {
		omppLog.Log(err)
		return false
	}
	return true
}

// Remove all existing disk use state files for current oms instance, log delete errors, return true on success or false or errors.
func deleteDiskUseStateFiles() bool {

	// pattern: job/state/disk-#-_4040-#-size-#-100-#-ok-#-120-#-2022_07_08_23_45_12_123-#-125678.json
	p := filepath.Join(theCfg.jobDir, "state", "disk-#-"+theCfg.omsName+"-#-size-#-*-#-*-#-*-#-*-#-*.json")
	isNoError := true

	fl := filesByPattern(p, "Error at disk use state files search")
	for _, f := range fl {
		isOk := fileDeleteAndLog(false, f)
		isNoError = isNoError && isOk
		if !isOk {
			// createCompStateFile(name, "error")
			omppLog.Log("FAILED to delete disk use state file: ", theCfg.omsName, " ", p)
		}
	}

	return isNoError
}

// return model run memory size in GB from number of processes, thread count, process memory in MBytes and therad memory in MBytes
func memoryRunSize(procCount, threadCount, procMem, threadMem int) int {
	if procCount <= 0 {
		procCount = 1
	}
	if threadCount <= 0 {
		threadCount = 1
	}
	return int(math.Ceil(float64(procCount*(procMem+threadMem*threadCount)) / 1024.0))
}

// Create MPI job hostfile, e.g.: models/log/host-2022_07_08_23_03_27_555-_4040.ini
// Return path to host file and or error if file create failed.
func createHostFile(job *RunJob, hfCfg hostIni, compUse []computeUse) (string, error) {
	if !theCfg.isJobControl {
		return "", nil // job control disabled
	}

	if !job.IsMpi || !hfCfg.isUse || len(compUse) <= 0 { // hostfile is not required
		return "", nil
	}

	/*
	   ; MS-MPI hostfile
	   ;
	   ; cpm:1
	   ; cpc-1:2
	   ; cpc-3:4
	   ;
	   [hostfile]
	   HostFileDir = models\log
	   HostName = @-HOST-@
	   CpuCores = @-CORES-@
	   RootLine = cpm:1
	   HostLine = @-HOST-@:@-CORES-@

	   ; OpenMPI hostfile
	   ;
	   ; cpm   slots=1 max_slots=1
	   ; cpc-1 slots=2
	   ; cpc-3 slots=4
	   ;
	   [hostfile]
	   HostFileDir = models/log
	   HostName = @-HOST-@
	   CpuCores = @-CORES-@
	   RootLine = cpm slots=1 max_slots=1
	   HostLine = @-HOST-@ slots=@-CORES-@
	*/
	// first line is root process host
	ls := []string{}

	if hfCfg.rootLine != "" {
		ls = append(ls, hfCfg.rootLine)
	}

	// for each server substitute host name and cores count
	if job.Res.ThreadCount <= 0 {
		job.Res.ThreadCount = 1
	}

	if hfCfg.hostLine != "" {
		for _, cu := range compUse {

			ln := hfCfg.hostLine

			if hfCfg.hostName != "" {
				ln = strings.ReplaceAll(ln, hfCfg.hostName, cu.CompName)
			}
			if hfCfg.cpuCores != "" {
				if job.Mpi.IsNotByJob {
					// use CPU, threads and memory as is, do not use job
					ln = strings.ReplaceAll(ln, hfCfg.cpuCores, strconv.Itoa(cu.Cpu))
				} else {
					ln = strings.ReplaceAll(ln, hfCfg.cpuCores, strconv.Itoa(cu.Cpu/job.Res.ThreadCount))
				}
			}
			ls = append(ls, ln)
		}
	}

	// write all lines into hostfile: /ompp/models/log/host-2022_07_08_23_03_27_555-_4040.ini
	hfPath := ""
	var err error
	if len(ls) > 0 {

		fn := "host-" + job.SubmitStamp + "-" + theCfg.omsName + ".ini"
		hfPath, err = filepath.Abs(filepath.Join(hfCfg.dir, fn))
		if err == nil {
			err = os.WriteFile(hfPath, []byte(strings.Join(ls, "\n")+"\n"), 0644)
		}
		if err != nil {
			omppLog.Log("Error at write into ", fn, ": ", err)
			return "", err
		}

		omppLog.Log("Run job: ", job.SubmitStamp, " ", job.ModelName, " hostfile: ", hfPath)
		for _, ln := range ls {
			omppLog.Log(ln)
		}
	}

	return hfPath, nil
}
