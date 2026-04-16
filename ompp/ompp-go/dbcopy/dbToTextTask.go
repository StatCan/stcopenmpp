// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// copy modeling task metadata and run history from database into text json and csv files
func dbToTextTask(modelName string, modelDigest string, runOpts *config.RunOptions) error {

	// get task name and id
	taskName := runOpts.String(taskNameArgKey)
	taskId := runOpts.Int(taskIdArgKey, 0)

	// conflicting options: use task id if positive else use task name
	if runOpts.IsExist(taskNameArgKey) && runOpts.IsExist(taskIdArgKey) {
		if taskId > 0 {
			omppLog.LogFmt("dbcopy options conflict. Using task id: %d, not a task name: %s", taskId, taskName)
			taskName = ""
		} else {
			omppLog.LogFmt("dbcopy options conflict. Using task name: %s, not a task id: %d", taskName, taskId)
			taskId = 0
		}
	}

	if taskId < 0 || taskId == 0 && taskName == "" {
		return helper.ErrorFmt("dbcopy invalid argument(s) for task id: %s and/or task name: %s", runOpts.String(taskIdArgKey), runOpts.String(taskNameArgKey))
	}

	// open source database connection and check is it valid
	cs, dn := db.IfEmptyMakeDefaultReadOnly(modelName, runOpts.String(fromSqliteArgKey), runOpts.String(dbConnStrArgKey), runOpts.String(dbDriverArgKey))

	srcDb, _, err := db.Open(cs, dn, false)
	if err != nil {
		return err
	}
	defer srcDb.Close()

	if err := db.CheckOpenmppSchemaVersion(srcDb); err != nil {
		return err
	}

	// get model metadata
	modelDef, err := db.GetModel(srcDb, modelName, modelDigest)
	if err != nil {
		return err
	}
	modelName = modelDef.Model.Name // set model name: it can be empty and only model digest specified

	// get task metadata by id or name
	var taskRow *db.TaskRow
	var outDir string
	if taskId > 0 {
		if taskRow, err = db.GetTask(srcDb, taskId); err != nil {
			return err
		}
		if taskRow == nil {
			return helper.ErrorNew("modeling task not found, task id:", taskId)
		}
		outDir = filepath.Join(runOpts.String(outputDirArgKey), modelName+".task."+strconv.Itoa(taskId))
	} else {
		if taskRow, err = db.GetTaskByName(srcDb, modelDef.Model.ModelId, taskName); err != nil {
			return err
		}
		if taskRow == nil {
			return helper.ErrorNew("modeling task not found:", taskName)
		}
		outDir = filepath.Join(runOpts.String(outputDirArgKey), modelName+".task."+taskName)
	}

	meta, err := db.GetTaskFull(srcDb, taskRow, true, "") // get task full metadata, including task run history
	if err != nil {
		return err
	}

	// use of run and set id's in directory names:
	// if true then always use id's in the names, false never use it
	// by default: only if name conflict
	isUseIdNames := false
	if runOpts.IsExist(useIdNamesArgKey) {
		isUseIdNames = runOpts.Bool(useIdNamesArgKey)

	} else { // default: check if run names in conflict, workset names are unique by db constraint

		rIdLst := make([]int, 0, len(meta.TaskRun)*len(meta.Set))
		rNameLst := make([]string, 0, len(meta.TaskRun)*len(meta.Set))

		for j := range meta.TaskRun {
		nxtRun:
			for k := range meta.TaskRun[j].TaskRunSet {

				// check is this run id already processed
				runId := meta.TaskRun[j].TaskRunSet[k].RunId
				for i := range rIdLst {
					if runId == rIdLst[i] {
						continue nxtRun // skip this run id, it is already processed
					}
				}
				rIdLst = append(rIdLst, runId)

				// find model run metadata by id
				runRow, err := db.GetRun(srcDb, runId)
				if err != nil {
					return err
				}

				// check if there is any other run with the same name
				if runRow != nil {

					for _, name := range rNameLst {
						isUseIdNames = name == runRow.Name
						if isUseIdNames {
							break // other run with the same name found
						}
					}
					if isUseIdNames {
						break
					}
					rNameLst = append(rNameLst, runRow.Name)
				}
			}
			if isUseIdNames {
				break
			}
		}
	}

	// create new output directory for task metadata
	if !theCfg.isKeepOutputDir {
		if ok := dirDeleteAndLog(outDir); !ok {
			return helper.ErrorNew("Error: unable to delete:", outDir)
		}
	}
	if err = os.MkdirAll(outDir, 0750); err != nil {
		return err
	}
	fileCreated := make(map[string]bool)

	// write task metadata into json file
	if err = toTaskJson(srcDb, modelDef, meta, outDir, isUseIdNames); err != nil {
		return err
	}

	// save runs from model run history
	var runIdLst []int
	var isRunNotFound, isRunNotCompleted bool

	for j := range meta.TaskRun {
	nextRun:
		for k := range meta.TaskRun[j].TaskRunSet {

			// check is this run id already processed
			runId := meta.TaskRun[j].TaskRunSet[k].RunId
			for i := range runIdLst {
				if runId == runIdLst[i] {
					continue nextRun
				}
			}
			runIdLst = append(runIdLst, runId)

			// find model run metadata by id
			runRow, err := db.GetRun(srcDb, runId)
			if err != nil {
				return err
			}
			if runRow == nil {
				isRunNotFound = true
				continue // skip: run not found
			}

			// run must be completed: status success, error or exit
			if !db.IsRunCompleted(runRow.Status) {
				isRunNotCompleted = true
				continue // skip: run not completed
			}

			rm, err := db.GetRunFullText(srcDb, runRow, true, "") // get full model run metadata
			if err != nil {
				return err
			}

			// write model run metadata into json, parameters and output result values into csv files
			if err = toRunText(srcDb, modelDef, rm, outDir, "", fileCreated, isUseIdNames); err != nil {
				return err
			}
		}
	}

	// find workset by set id and save it's metadata to json and workset parameters to csv
	var wsIdLst []int
	var isSetNotFound, isSetNotReadOnly bool

	var fws = func(dbConn *sql.DB, setId int, fileCreated map[string]bool) error {

		// check is workset already processed
		for i := range wsIdLst {
			if setId == wsIdLst[i] {
				return nil
			}
		}
		wsIdLst = append(wsIdLst, setId)

		// get workset by id
		wsRow, err := db.GetWorkset(dbConn, setId)
		if err != nil {
			return err
		}
		if wsRow == nil { // exit: workset not found
			isSetNotFound = true
			return nil
		}
		if !wsRow.IsReadonly { // exit: workset not readonly
			isSetNotReadOnly = true
			return nil
		}

		wm, err := db.GetWorksetFull(dbConn, wsRow, "") // get full workset metadata
		if err != nil {
			return err
		}

		// write workset metadata into json and parameter values into csv files
		if err = toWorksetText(dbConn, modelDef, wm, outDir, fileCreated, isUseIdNames); err != nil {
			return err
		}
		return nil
	}

	// save task body worksets
	for k := range meta.Set {
		if err = fws(srcDb, meta.Set[k], fileCreated); err != nil {
			return err
		}
	}

	// save worksets from model run history
	for j := range meta.TaskRun {
		for k := range meta.TaskRun[j].TaskRunSet {
			if err = fws(srcDb, meta.TaskRun[j].TaskRunSet[k].SetId, fileCreated); err != nil {
				return err
			}
		}
	}

	// display warnings if any worksets not found or not readonly
	// display warnings if any model runs not exists or not completed
	if isSetNotFound {
		omppLog.LogFmt("Warning: task %s workset(s) not found, copy of task incomplete", meta.Task.Name)
	}
	if isSetNotReadOnly {
		omppLog.LogFmt("Warning: task %s workset(s) not readonly, copy of task incomplete", meta.Task.Name)
	}
	if isRunNotFound {
		omppLog.LogFmt("Warning: task %s model run(s) not found, copy of task run history incomplete", meta.Task.Name)
	}
	if isRunNotCompleted {
		omppLog.LogFmt("Warning: task %s model run(s) not completed, copy of task run history incomplete", meta.Task.Name)
	}

	// pack worksets metadata json and csv files into zip
	if runOpts.Bool(zipArgKey) {
		zipPath, err := helper.PackZip(outDir, !theCfg.isKeepOutputDir, "")
		if err != nil {
			return err
		}
		omppLog.Log("Packed", zipPath)
	}
	return nil
}

// toTaskListJson convert all successfully completed tasks and tasks run history to json and write into json files
func toTaskListJson(dbConn *sql.DB, modelDef *db.ModelMeta, outDir string, isUseIdNames bool) error {

	// get all modeling tasks and successfully completed tasks run history
	tl, err := db.GetTaskFullList(dbConn, modelDef.Model.ModelId, true, "")
	if err != nil {
		return err
	}

	// read each task metadata and write into json files
	for k := range tl {
		if err := toTaskJson(dbConn, modelDef, &tl[k], outDir, isUseIdNames); err != nil {
			return err
		}
	}
	return nil
}

// toTaskJson convert modeling task and task run history to json and write into json file
func toTaskJson(dbConn *sql.DB, modelDef *db.ModelMeta, meta *db.TaskMeta, outDir string, isUseIdNames bool) error {

	// convert db rows into "public" format
	omppLog.Log("Modeling task", meta.Task.TaskId, meta.Task.Name)

	pub, err := meta.ToPublic(dbConn, modelDef)
	if err != nil {
		return err
	}

	// save modeling task metadata into json
	var fname string
	if !isUseIdNames {
		fname = modelDef.Model.Name + ".task." + helper.CleanFileName(meta.Task.Name) + ".json"
	} else {
		fname = modelDef.Model.Name + ".task." + strconv.Itoa(meta.Task.TaskId) + "." + helper.CleanFileName(meta.Task.Name) + ".json"
	}

	return helper.ToJsonFile(filepath.Join(outDir, fname), pub)
}
