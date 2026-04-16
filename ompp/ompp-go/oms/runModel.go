// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"bufio"
	"errors"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// runModel starts new model run and return run stamp.
// if run stamp not specified as input parameter then use unique timestamp.
// Model run console output redirected to log file: models/log/modelName.runStamp.console.log
func (rsc *RunCatalog) runModel(job *RunJob, queueJobPath string, hfCfg hostIni, compUse []computeUse) (*RunState, error) {

	// new run state
	ts, tNow := theCatalog.getNewTimeStamp()

	rs := &RunState{
		ModelName:      job.ModelName,
		ModelDigest:    job.ModelDigest,
		RunStamp:       helper.CleanFileName(job.RunStamp),
		SubmitStamp:    job.SubmitStamp,
		UpdateDateTime: helper.MakeDateTime(tNow),
	}
	if rs.RunStamp == "" {
		rs.RunStamp = helper.CleanFileName(job.Opts["OpenM.RunStamp"])
	}
	if rs.RunStamp == "" {
		rs.RunStamp = ts
	}

	// set directories: work directory and bin model.exe directory
	// if bin directory is relative then it must be relative to oms root directory
	// re-base it to model work directory
	binRoot, _ := theCatalog.getModelDir()

	mb, ok := theCatalog.modelBasicByDigestOrName(rs.ModelDigest)
	if !ok {
		err := errors.New("Model not found: " + rs.ModelName + ": " + rs.ModelDigest)
		omppLog.Log("Model run error: ", err)
		moveJobQueueToFailed(queueJobPath, rs.SubmitStamp, rs.ModelName, rs.ModelDigest, rs.RunStamp, false)
		rs.IsFinal = true
		return rs, err // exit with error: model failed to start
	}
	binDir := mb.binDir

	wDir := binDir
	if job.Dir != "" {
		wDir = filepath.Join(binRoot, job.Dir)
	}

	binDir, err := filepath.Abs(binDir) // relative to work dir fails on Windows if models are at \\UNC\path
	if err != nil {
		binDir = binRoot
	}

	// make file path for model console log output
	if mb.isLogDir {
		if mb.logDir == "" {
			mb.logDir, mb.isLogDir = theCatalog.getModelLogDir()
		}
	}
	if mb.isLogDir {
		if mb.logDir == "" {
			mb.logDir = binDir
		}
		rs.IsLog = mb.isLogDir
		rs.LogFileName = rs.ModelName + "." + rs.RunStamp + ".console.log"
		rs.logPath = filepath.Join(mb.logDir, rs.LogFileName)
	}

	// make model run arguments and create ini file if required
	mArgs, iniPath, err := makeRunArgsIni(mb.binDir, wDir, mb.logDir, job, rs)
	if err != nil {
		omppLog.Log("Model run error: ", err)
		moveJobQueueToFailed(queueJobPath, rs.SubmitStamp, rs.ModelName, rs.ModelDigest, rs.RunStamp, false)
		rs.IsFinal = true
		return rs, err
	}

	// use job control resources if not explicitly disabled and create hostfile
	hfPath := ""
	if job.IsMpi && !job.Mpi.IsNotByJob {

		hfPath, err = createHostFile(job, hfCfg, compUse)

		if err != nil {
			omppLog.Log("Model run error: ", err)
			moveJobQueueToFailed(queueJobPath, rs.SubmitStamp, rs.ModelName, rs.ModelDigest, rs.RunStamp, false)
			rs.IsFinal = true
			return rs, err
		}
	}

	// cleanup helpers
	delComputeUse := func(cuLst []computeUse) {
		for _, cu := range cuLst {
			if cu.filePath != "" {
				fileDeleteAndLog(false, cu.filePath)
			}
		}
	}
	cleanAndReturn := func(e error, rState *RunState, qPath string, cuLst []computeUse) (*RunState, error) {
		omppLog.Log("Error at starting model: ", e)
		delComputeUse(cuLst)
		moveJobQueueToFailed(qPath, rState.SubmitStamp, rState.ModelName, rState.ModelDigest, rState.RunStamp, false)
		rState.IsFinal = true
		return rState, errors.New("Error at starting model " + rState.ModelName + ": " + e.Error())
	}

	// assume model exe name is the same as model name
	mExe := helper.CleanFileName(rs.ModelName)

	cmd, err := rsc.makeCommand(mExe, binDir, wDir, mb.dbPath, mArgs, &job.RunRequest, job.Res.ProcessCount, hfPath)
	if err != nil {
		omppLog.Log("Error at starting model: ", err)
		moveJobQueueToFailed(queueJobPath, rs.SubmitStamp, rs.ModelName, rs.ModelDigest, rs.RunStamp, false)
		rs.IsFinal = true
		return rs, errors.New("Error at starting model " + rs.ModelName + ": " + err.Error())
	}

	// create job usage file for each computational server
	isErr := false
	for k := 0; !isErr && k < len(compUse); k++ {

		compUse[k].filePath = compUsedPath(compUse[k].CompName, rs.SubmitStamp, compUse[k].Cpu, compUse[k].Mem)
		isErr = !fileCreateEmpty(false, compUse[k].filePath)
	}
	if isErr {
		omppLog.Log("Error at starting model: ", rs.ModelName, " ", rs.ModelDigest, " ", rs.SubmitStamp)
		delComputeUse(compUse)
		moveJobQueueToFailed(queueJobPath, rs.SubmitStamp, rs.ModelName, rs.ModelDigest, rs.RunStamp, false)
		rs.IsFinal = true
		return rs, errors.New("Error at starting model " + rs.ModelName + " " + rs.ModelDigest)
	}

	// connect console output to log line array
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return cleanAndReturn(err, rs, queueJobPath, compUse)
	}
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return cleanAndReturn(err, rs, queueJobPath, compUse)
	}
	outDoneC := make(chan bool, 1)
	errDoneC := make(chan bool, 1)
	rs.killC = make(chan bool, 1)
	logTck := time.NewTicker(logTickTimeout * time.Millisecond)

	// append console output to log lines array
	doLog := func(rState *RunState, r io.Reader, done chan<- bool) {
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			rsc.updateRunStateLog(rState, false, sc.Text())
		}
		done <- true
		close(done)
	}

	// run state initialized: append it to the run state list
	// create model run log file
	rsc.createRunStateLog(rs)

	// start console output listners
	go doLog(rs, outPipe, outDoneC)
	go doLog(rs, errPipe, errDoneC)

	// start the model
	omppLog.Log("Run model: ", mExe, " in directory: ", wDir)
	if rs.logPath != "" {
		omppLog.Log("Run model: ", mExe, " log: ", rs.logPath)
	}
	cmdLine := strings.Join(cmd.Args, " ")
	omppLog.Log(cmdLine)
	rs.cmdPath = cmd.Path
	rsc.updateRunStateProcess(rs, false)

	err = cmd.Start()
	if err != nil {
		omppLog.Log("Model run error: ", err)
		delComputeUse(compUse)
		moveJobQueueToFailed(queueJobPath, rs.SubmitStamp, rs.ModelName, rs.ModelDigest, rs.RunStamp, false)
		rsc.updateRunStateLog(rs, true, err.Error())
		rs.IsFinal = true
		return rs, err // exit with error: model failed to start
	}
	// else model started
	rs.pid = cmd.Process.Pid
	cmdStart := time.Now().Unix() // model started, Unix seconds
	rsc.updateRunStateProcess(rs, false)

	// move job file form queue to active
	activeJobPath, _ := moveJobToActive(queueJobPath, rs, job.Res, rs.RunStamp, binDir, wDir, iniPath, hfPath, cmdLine)

	//  wait until run completed or terminated
	go func(rState *RunState, cmd *exec.Cmd, jobPath string, cuLst []computeUse) {

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
			case isKill, ok := <-rState.killC:
				if !ok {
					rState.killC = nil
				}
				if isKill && ok {
					omppLog.Log("Kill run: ", rState.ModelName, " ", rState.ModelDigest, " ", rState.RunName, " ", rState.RunStamp)
					rState.isKill = true
					if e := cmd.Process.Kill(); e != nil {
						omppLog.Log(e)
					}
				}
			case <-logTck.C:
			}
		}

		// wait for model run to be completed
		e := cmd.Wait()
		cmdStop := time.Now().Unix()

		if e != nil {
			omppLog.Log("Model run error: ", e)
			delComputeUse(cuLst)
			rsc.updateRunStateLog(rState, true, e.Error())
			moveActiveJobToHistory(jobPath, db.ErrorRunStatus, rState.isKill, rState.SubmitStamp, rState.ModelName, rState.ModelDigest, rState.RunStamp, cmdStart, cmdStop)
			_, e = theCatalog.UpdateRunStatus(rState.ModelDigest, rState.RunStamp, db.ErrorRunStatus)
			if e != nil {
				omppLog.Log(e)
			}
			return
		}
		// else: completed OK
		rsc.updateRunStateLog(rState, true, "")
		delComputeUse(cuLst)
		moveActiveJobToHistory(jobPath, db.DoneRunStatus, false, rState.SubmitStamp, rState.ModelName, rState.ModelDigest, rState.RunStamp, cmdStart, cmdStop)

	}(rs, cmd, activeJobPath, compUse)

	return rs, nil
}

// makeCommand return command to run the model.
// If template file name specified then template processing results used to create command line.
// If this is MPI model run then tempalate is requred
// MPI run template can be model specific: "mpi.ModelName.template.txt" or default: "mpi.ModelRun.template.txt".
func (rsc *RunCatalog) makeCommand(mExe, binDir, workDir, dbPath string, mArgs []string, req *RunRequest, procCount int, hfPath string) (*exec.Cmd, error) {

	// check is it MPI model run, to run MPI model template is required
	// if template not specified then try to use default template
	if req.IsMpi && req.Template == "" {

		// search for model-specific MPI template
		mtn := "mpi." + req.ModelName + ".template.txt"
		isDef := false

		for _, tn := range theRunCatalog.mpiTemplates {
			if tn == mtn {
				req.Template = mtn
			}
			if tn == defaultMpiTemplate {
				isDef = true
			}
		}

		// if model-specific MPI template not found then use default MPI template to run the model
		if req.Template == "" && isDef {
			req.Template = defaultMpiTemplate
		}
	}
	isTmpl := req.Template != ""

	// if there is no template then
	//  set work directory
	// 	append request environment variables to model environment
	//  make command line:
	//	 ./modelExe -OpenM.LogToFile true ...etc...

	if !isTmpl {
		if binDir == "" || binDir == "." || binDir == "./" {
			mExe = "./" + mExe
		} else {
			mExe = filepath.Join(binDir, mExe)
		}
		cmd := exec.Command(mExe, mArgs...)

		cmd.Dir = workDir

		if len(req.Env) > 0 {
			env := os.Environ()
			for key, val := range req.Env {
				if key != "" && val != "" {
					env = append(env, key+"="+val)
				}
			}
			cmd.Env = env
		}
		return cmd, nil
	}
	// else
	// process template to get exe name and arguments

	// parse template
	tmpl, err := template.ParseFiles(filepath.Join(rsc.etcDir, req.Template))
	if err != nil {
		return nil, err
	}

	// set template parameters
	wd, err := filepath.Abs(workDir)
	if err != nil {
		return nil, err
	}
	np := procCount
	if req.IsMpi && req.Mpi.IsNotOnRoot {
		np++
	}

	d := struct {
		ModelName string            // model name
		ExeStem   string            // base part of model exe name, usually modelName
		Dir       string            // work directory to run the model
		BinDir    string            // bin directory where model.exe is located
		DbPath    string            // path to sqlite database file: models/bin/model.sqlite
		MpiNp     int               // number of MPI processes
		HostFile  string            // if not empty then absolute path to hostfile
		Args      []string          // model command line arguments
		Env       map[string]string // environment variables to run the model
	}{
		ModelName: req.ModelName,
		ExeStem:   mExe,
		Dir:       wd,
		BinDir:    binDir,
		DbPath:    dbPath,
		MpiNp:     np,
		HostFile:  hfPath,
		Args:      mArgs,
		Env:       req.Env,
	}

	// execute template and convert results in array of text lines
	var b strings.Builder

	err = tmpl.Execute(&b, d)
	if err != nil {
		return nil, err
	}
	tLines := strings.Split(strings.ReplaceAll(b.String(), "\r", "\n"), "\n")

	// from template processing results get:
	//   exe name as first non-empty line
	//   use all other non-empty lines as command line arguments
	cExe := ""
	cArgs := []string{}

	for k := range tLines {

		cl := strings.TrimSpace(tLines[k])
		if cl == "" {
			continue
		}
		if cExe == "" {
			cExe = cl
		} else {
			cArgs = append(cArgs, cl)
		}
	}
	if cExe == "" {
		return nil, errors.New("Error: empty template processing results, cannot run the model: " + req.ModelName)
	}

	// make command
	cmd := exec.Command(cExe, cArgs...)
	return cmd, nil
}

// StopModelRun kill model run by run stamp
// or remove run request from the queue by submit stamp or by run stamp.
// Return submission stamp, job file path and two flags: if model run found and if model is runniing now
func (rsc *RunCatalog) stopModelRun(modelDigest string, stamp string) (bool, string, string, bool) {

	tNow := time.Now()

	rsc.rscLock.Lock()
	defer rsc.rscLock.Unlock()

	// find model run state by digest and run stamp
	rsl := rsc.findRunStateLog(modelDigest, stamp)

	if rsl == nil { // if model run stamp and submit stamp not found then check if there is a job file in the queue

		if qj, ok := rsc.queueJobs[stamp]; ok {
			return true, stamp, qj.filePath, false // job file found in the queue
		}
		return false, "", "", false // no model run stamp and no submit stamp found
	}

	// find model in the active job list or if not active then find it in job queue
	jobPath := ""
	if aj, ok := rsc.activeJobs[rsl.SubmitStamp]; ok {
		jobPath = aj.filePath
	} else {
		if qj, ok := rsc.queueJobs[rsl.SubmitStamp]; ok {
			jobPath = qj.filePath
		}
	}

	rsl.UpdateDateTime = helper.MakeDateTime(tNow)

	// kill model run if model is running
	if rsl.killC != nil {
		rsl.killC <- true
		rsl.isKill = true
		return true, rsl.SubmitStamp, jobPath, true
	}
	// else remove request from the queue
	rsl.IsFinal = true

	return true, rsl.SubmitStamp, jobPath, false
}

// make model run command line arguments and create ini file if required
func makeRunArgsIni(binDir, workDir, logDir string, job *RunJob, rs *RunState) ([]string, string, error) {

	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return []string{}, "", err
	}

	// make model run command line arguments, starting from process run stamp and log options
	mArgs := []string{}
	mArgs = append(mArgs, "-OpenM.RunStamp", rs.RunStamp)
	mArgs = append(mArgs, "-OpenM.LogToConsole", "true")
	mArgs = append(mArgs, "-OpenM.LogToFile", "false")
	iniPath := ""

	importDbLcDot := strings.ToLower("-ImportDb.")
	microdataLcDot := strings.ToLower("-Microdata.")
	dotRunDescrLc := strings.ToLower(".RunDescription")

	entAttrs := theCatalog.entityAttrsByDigest(rs.ModelDigest)
	descrNotes := []db.DescrNote{}

	// append model run options from run request
	for krq, val := range job.Opts {

		if len(krq) < 1 { // skip empty run options
			continue
		}

		// command line argument key starts with "-" ex: "-OpenM.Threads"
		key := krq
		if krq[0] != '-' {
			key = "-" + krq
		}

		// save run name and task run name to return as part of run state
		if strings.EqualFold(key, "-OpenM.RunName") {
			rs.RunName = val
		}
		if strings.EqualFold(key, "-OpenM.TaskRunName") {
			rs.TaskRunName = val
		}
		if strings.EqualFold(key, "-OpenM.RunStamp") {
			continue // skip: use it run stamp from run state
		}
		// thread count MUST be specified using request Threads
		if strings.EqualFold(key, "-OpenM.Threads") {
			continue // skip number of threads option: use request Threads value instead
		}
		// MPI "not on root" flag
		if strings.EqualFold(key, "-OpenM.NotOnRoot") {
			continue // skip  MPI "not on root" flag: use request Mpi.IsNotOnRoot boolean instead
		}
		if strings.EqualFold(key, "-OpenM.LogToConsole") {
			continue // skip log to console input run option: it is already on
		}
		if strings.EqualFold(key, "-OpenM.LogToFile") {
			continue // skip log to file input run option: replaced by console output
		}
		if strings.EqualFold(key, "-OpenM.LogFilePath") {
			continue // skip log file path input run option: replaced by console output
		}
		if strings.EqualFold(key, "-OpenM.Database") || strings.EqualFold(key, "-db") ||
			strings.EqualFold(key, "-OpenM.Sqlite") || strings.EqualFold(key, "-OpenM.SqliteFromBin") {
			continue // database connection string not allowed as run option
		}
		if strings.HasPrefix(strings.ToLower(key), importDbLcDot) {
			continue // import database connection string not allowed as run option
		}

		// directory value: substitute OM_USER_FILES if requierd and sanitize: it must be relative to oms root
		if strings.EqualFold(key, "-OpenM.iniFile") || strings.EqualFold(key, "-ini") ||
			strings.EqualFold(key, "-OpenM.ParamDir") || strings.EqualFold(key, "-p") ||
			strings.EqualFold(key, "-Microdata.CsvDir") {

			if !filepath.IsLocal(val) {
				return []string{}, "", errors.New("invalid directory: " + val)
			}

			if theCfg.filesDir != "" {

				isUfd := true
				switch {
				case strings.HasPrefix(val, "OM_USER_FILES"):
					val = strings.Replace(val, "OM_USER_FILES", theCfg.filesDir, 1)
				case strings.HasPrefix(val, "$OM_USER_FILES"):
					val = strings.Replace(val, "$OM_USER_FILES", theCfg.filesDir, 1)
				case strings.HasPrefix(val, "{OM_USER_FILES}"):
					val = strings.Replace(val, "{OM_USER_FILES}", theCfg.filesDir, 1)
				case strings.HasPrefix(val, "${OM_USER_FILES}"):
					val = strings.Replace(val, "${OM_USER_FILES}", theCfg.filesDir, 1)
				case strings.HasPrefix(val, "%OM_USER_FILES%"):
					val = strings.Replace(val, "%OM_USER_FILES%", theCfg.filesDir, 1)
				default:
					isUfd = false
				}
				// if directory is relative to user files directory then make it relative to working directory
				if isUfd {

					val, err = filepath.Abs(val)
					if err != nil {
						return []string{}, "", errors.New("invalid directory: " + val)
					}
					val, err = filepath.Rel(absWorkDir, val)
					if err != nil {
						return []string{}, "", errors.New("invalid directory: " + val)
					}
				}
			}

			val = helper.CleanFilePath(val) // cleanup path
		}

		if strings.EqualFold(key, "-OpenM.iniFile") || strings.EqualFold(key, "-ini") {
			iniPath = val
			continue // ini file path as run option: merge of ini content may be required
		}

		// use ini file if run description specified
		if strings.HasSuffix(strings.ToLower(key), dotRunDescrLc) {

			if 1+len(dotRunDescrLc) >= len(key) {
				return []string{}, iniPath, errors.New("invalid run description key: " + key)
			}
			lc := key[1:(len(key) - len(dotRunDescrLc))] // language code

			idx := -1
			for k := range descrNotes {
				if descrNotes[k].LangCode == lc {
					idx = k
					break
				}
			}
			if idx < 0 {
				idx = len(descrNotes)
				descrNotes = append(descrNotes, db.DescrNote{LangCode: lc})
			}
			descrNotes[idx].Descr = val

			continue // use ini-file for run description
		}

		// if this is microdata run option then microdata must be enabled
		// do not allow microdata options which are part of Microdata run request:
		//   -Microdata.ToDb -Microdata.UseInternal
		//   -Microdata.All  -Microdata.anyEntityName
		if strings.HasPrefix(strings.ToLower(key), microdataLcDot) {

			if !theCfg.isMicrodata {
				return []string{}, iniPath, errors.New("microdata not allowed: " + rs.ModelName + ": " + rs.ModelDigest)
			}
			subKey := key[len(microdataLcDot):]

			if strings.EqualFold(subKey, "All") || strings.EqualFold(subKey, "ToDb") || strings.EqualFold(subKey, "UseInternal") {
				return []string{}, iniPath, errors.New("incorrect use of run option: " + key + ": " + rs.ModelName + ": " + rs.ModelDigest)
			}

			for k := range entAttrs {

				if subKey == entAttrs[k].Name {
					return []string{}, iniPath, errors.New("incorrect use of run option: " + key + ": " + rs.ModelName + ": " + rs.ModelDigest)
				}
			}
		}

		mArgs = append(mArgs, key, val) // append command line argument key and value
	}

	// append threads number if required
	if job.Res.ThreadCount > 1 {
		mArgs = append(mArgs, "-OpenM.Threads", strconv.Itoa(job.Res.ThreadCount))
	}
	if job.IsMpi && job.Mpi.IsNotOnRoot {
		mArgs = append(mArgs, "-OpenM.NotOnRoot")
	}

	// create model run ini file if required to pass model run options
	// merge ini file options with job model run options
	//

	// if list of tables to retain is not empty then put the list into ini-file:
	//
	//   [Tables]
	//   Retain   = ageSexIncome, AdditionalTables
	//
	iniKeyVal := map[string]string{}

	if len(job.Tables) > 0 {
		iniKeyVal["Tables.Retain"] = strings.Join(job.Tables, ", ")
	}

	// if list of microdata to retain is not empty then put the list into ini-file:
	//
	//   [Microdata]
	//   ToDb        = true
	//   UseInternal = true
	//   Person      = age,income
	//   Other       = All
	//
	if theCfg.isMicrodata && job.Microdata.IsToDb && len(entAttrs) > 0 && len(job.Microdata.Entity) > 0 {

		iniKeyVal["Microdata.ToDb"] = "true"
		if job.Microdata.IsInternal {
			iniKeyVal["Microdata.UseInternal"] = "true"
		}

		// for each entity check if All attributes included or attributes must be specified as comma separated list
		for k := range job.Microdata.Entity {

			// find entity name in the list of model entities
			eIdx := -1
			for j := range entAttrs {
				if entAttrs[j].Name == job.Microdata.Entity[k].Name {
					eIdx = j
					break
				}
			}
			if eIdx < 0 || eIdx >= len(entAttrs) {
				return []string{}, iniPath, errors.New("invalid microdata entity: " + job.Microdata.Entity[k].Name + ": " + rs.ModelName + ": " + rs.ModelDigest)
			}

			// check if all entity attributes included in run microdata
			na := len(job.Microdata.Entity[k].Attr)
			isAll := na == 1 && job.Microdata.Entity[k].Attr[0] == "All"

			if !isAll {

				attrs := make([]string, na)
				copy(attrs, job.Microdata.Entity[k].Attr)
				sort.Strings(attrs)

				for j := range entAttrs[eIdx].Attr {

					if !job.Microdata.IsInternal && entAttrs[eIdx].Attr[j].IsInternal {
						continue // skip: this model run does not using internal attributes
					}

					n := sort.SearchStrings(attrs, entAttrs[eIdx].Attr[j].Name)
					isAll = n >= 0 && n < na && attrs[n] == entAttrs[eIdx].Attr[j].Name
					if !isAll {
						break
					}
				}
			}

			// append entity attributes to ini file content: EntityName = All or EntityName = AttrA, AttrB
			if isAll {
				iniKeyVal["Microdata."+job.Microdata.Entity[k].Name] = "All"
			} else {
				iniKeyVal["Microdata."+job.Microdata.Entity[k].Name] = strings.Join(job.Microdata.Entity[k].Attr, ",")
			}
		}
	}

	// if run description or notes specified then use ini-file:
	//
	// [EN]
	// RunDescription = "model run #7 with 50,000 cases"
	// RunNotesPath   = run_notes-in-english.md
	//

	// save run notes into the file(s) and append that file path to the ini-file content
	for _, rn := range job.RunNotes {
		if rn.Note == "" {
			continue
		}
		if !rs.IsLog {
			return []string{}, iniPath, errors.New("Unable to save run notes: " + rs.ModelName + ": " + rs.ModelDigest)
		}

		p, e := filepath.Abs(filepath.Join(logDir, rs.RunStamp+".run_notes."+rn.LangCode+".md"))
		if e == nil {
			e = os.WriteFile(p, []byte(rn.Note), 0644)
		}
		if e == nil {
			p, e = filepath.Rel(absWorkDir, p)
		}
		if e != nil {
			return []string{}, iniPath, e
		}

		// store path to notes .md file instead of actual notes
		dnIdx := -1
		for k := range descrNotes {
			if descrNotes[k].LangCode == rn.LangCode {
				dnIdx = k
				break
			}
		}
		if dnIdx < 0 {
			dnIdx = len(descrNotes)
			descrNotes = append(descrNotes, db.DescrNote{LangCode: rn.LangCode})
		}
		descrNotes[dnIdx].Note = p
	}

	// validate run description and notes language code
	if len(descrNotes) > 0 {

		langLst, _ := theCatalog.LangListByDigestOrName(rs.ModelDigest)

		for k := range descrNotes {

			isOk := false
			for j := range langLst {
				isOk = langLst[j].LangCode == descrNotes[k].LangCode
				if isOk {
					break
				}
			}
			if !isOk {
				return []string{}, iniPath, errors.New("invalid language code: " + descrNotes[k].LangCode + ": " + rs.ModelName + ": " + rs.ModelDigest)
			}

			if descrNotes[k].Descr != "" {
				iniKeyVal[descrNotes[k].LangCode+".RunDescription"] = helper.QuoteForIni(descrNotes[k].Descr)
			}
			if descrNotes[k].Note != "" {
				iniKeyVal[descrNotes[k].LangCode+".RunNotesPath"] = descrNotes[k].Note // path to notes .md file
			}
		}
	}

	// if ini file file path specified as run option then read ini file and merge with other run options
	eaIni := []helper.IniEntry{}

	if iniPath != "" {

		if len(iniKeyVal) <= 0 { // nothing to merge, add ini file model run option

			mArgs = append(mArgs, "-OpenM.iniFile", iniPath) // append ini file path to command line arguments

		} else { // else merge ini file content

			p, e := filepath.Abs(filepath.Join(binDir, iniPath))
			if e != nil {
				return []string{}, iniPath, e
			}
			var e2 error

			eaIni, e2 = config.NewIni(p, theCfg.encodingName)
			if e2 != nil {
				return []string{}, iniPath, e2
			}

			for _, e := range eaIni {
				key := e.Section + "." + e.Key
				if _, ok := iniKeyVal[key]; !ok { // if option not exist then insert new option from ini file
					iniKeyVal[key] = e.Val
				}
			}
		}
	}

	// microdata disabled: remove microdata from ini file
	if !theCfg.isMicrodata {
		maps.DeleteFunc(iniKeyVal, func(key string, val string) bool { return strings.HasPrefix(strings.ToLower(key), "microdata.") })
	}

	// create ini file and append -ini fileName.ini to model run options
	if len(iniKeyVal) > 0 {

		for key, val := range iniKeyVal {
			eaIni = helper.MergeSectionKeyIniEntry(eaIni, key, val)

		}

		iniContent := ""
		sc := ""

		for _, e := range eaIni {

			if !strings.EqualFold(e.Section, sc) { // append new [section]
				sc = e.Section
				iniContent += "[" + sc + "]\n"
			}
			iniContent += e.Key + " = " + helper.QuoteForIni(e.Val) + "\n"
		}

		iniPath = rs.RunStamp + "." + rs.ModelName + ".ini" // new ini file path, relative to log directory

		p, e := filepath.Abs(filepath.Join(logDir, iniPath))
		if e == nil {
			e = os.WriteFile(p, []byte(iniContent), 0644)
		}
		if e == nil {
			p, e = filepath.Rel(absWorkDir, p)
		}
		if e != nil {
			return []string{}, iniPath, e
		}
		mArgs = append(mArgs, "-ini", p) // append ini file path to command line arguments
	}

	return mArgs, iniPath, nil
}
