// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// copy modeling task metadata and run history from source database to destination database
func dbToDbTask(modelName string, modelDigest string, runOpts *config.RunOptions) error {

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

	// validate source and destination
	csInp, dnInp := db.IfEmptyMakeDefaultReadOnly(modelName, runOpts.String(fromSqliteArgKey), runOpts.String(dbConnStrArgKey), runOpts.String(dbDriverArgKey))
	csOut, dnOut := db.IfEmptyMakeDefault(modelName, runOpts.String(toSqliteArgKey), runOpts.String(toDbConnStrArgKey), runOpts.String(toDbDriverArgKey))

	if csInp == csOut && dnInp == dnOut {
		return helper.ErrorNew("source same as destination: cannot overwrite model in database")
	}

	// open source database connection and check is it valid
	srcDb, _, err := db.Open(csInp, dnInp, false)
	if err != nil {
		return err
	}
	defer srcDb.Close()

	if err := db.CheckOpenmppSchemaVersion(srcDb); err != nil {
		return err
	}

	// open destination database and check is it valid
	dstDb, dbFacet, err := db.Open(csOut, dnOut, true)
	if err != nil {
		return err
	}
	defer dstDb.Close()

	if err := db.CheckOpenmppSchemaVersion(dstDb); err != nil {
		return err
	}

	// source: get model metadata
	srcModel, err := db.GetModel(srcDb, modelName, modelDigest)
	if err != nil {
		return err
	}
	modelName = srcModel.Model.Name // set model name: it can be empty and only model digest specified

	// get task metadata by id or name
	var taskRow *db.TaskRow
	if taskId > 0 {
		if taskRow, err = db.GetTask(srcDb, taskId); err != nil {
			return err
		}
		if taskRow == nil {
			return helper.ErrorNew("modeling task not found, task id:", taskId)
		}
	} else {
		if taskRow, err = db.GetTaskByName(srcDb, srcModel.Model.ModelId, taskName); err != nil {
			return err
		}
		if taskRow == nil {
			return helper.ErrorNew("modeling task not found:", taskName)
		}
	}

	meta, err := db.GetTaskFull(srcDb, taskRow, true, "") // get task full metadata, including task run history
	if err != nil {
		return err
	}

	// destination: get model metadata
	dstModel, err := db.GetModel(dstDb, modelName, modelDigest)
	if err != nil {
		return err
	}

	// destination: get list of languages
	dstLang, err := db.GetLanguages(dstDb)
	if err != nil {
		return err
	}

	// copy to destiantion model runs from task run history
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

			// convert model run db rows into "public" format
			runPub, err := rm.ToPublic(srcModel)
			if err != nil {
				return err
			}
			if theCfg.isNoDigestCheck {
				runPub.ModelDigest = "" // model digest validation disabled
			}

			// copy source model run metadata, parameter values, output results into destination database
			_, err = copyRunDbToDb(srcDb, dstDb, dbFacet, srcModel, dstModel, rm.Run.RunId, runPub, dstLang)
			if err != nil {
				return err
			}
		}
	}

	// find workset by set id and save it's metadata to json and workset parameters to csv
	var wsIdLst []int
	var isSetNotFound, isSetNotReadOnly bool

	var fws = func(dbConn *sql.DB, setId int) error {

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

		// convert workset db rows into "public" format
		setPub, err := wm.ToPublic(srcDb, srcModel)
		if err != nil {
			return err
		}
		if theCfg.isNoDigestCheck {
			setPub.ModelDigest = "" // model digest validation disabled
		}

		// copy source workset metadata and parameters into destination database
		_, err = copyWorksetDbToDb(srcDb, dstDb, srcModel, dstModel, wm.Set.SetId, setPub, dstLang)
		if err != nil {
			return err
		}
		return nil
	}

	// save task body worksets
	for k := range meta.Set {
		if err = fws(srcDb, meta.Set[k]); err != nil {
			return err
		}
	}

	// save worksets from model run history
	for j := range meta.TaskRun {
		for k := range meta.TaskRun[j].TaskRunSet {
			if err = fws(srcDb, meta.TaskRun[j].TaskRunSet[k].SetId); err != nil {
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

	// convert task db rows into "public" format
	pub, err := meta.ToPublic(srcDb, srcModel)
	if err != nil {
		return err
	}

	// rename destination task
	if runOpts.IsExist(taskNewNameArgKey) {
		pub.Name = runOpts.String(taskNewNameArgKey)
	}

	// copy source task metadata into destination database
	_, err = copyTaskDbToDb(srcDb, dstDb, srcModel, dstModel, meta.Task.TaskId, pub, dstLang)
	if err != nil {
		return err
	}
	return nil
}

// copyTaskListDbToDb do copy all modeling tasks from source to destination database
func copyTaskListDbToDb(
	srcDb *sql.DB, dstDb *sql.DB, srcModel *db.ModelMeta, dstModel *db.ModelMeta, dstLang *db.LangMeta) error {

	// source: get all modeling tasks metadata in all languages
	srcTl, err := db.GetTaskFullList(srcDb, srcModel.Model.ModelId, true, "")
	if err != nil {
		return err
	}
	if len(srcTl) <= 0 {
		return nil
	}

	// copy task metadata from source to destination database by using "public" format
	for k := range srcTl {

		// convert task metadata db rows into "public"" format
		pub, err := srcTl[k].ToPublic(srcDb, srcModel)
		if err != nil {
			return err
		}

		// save into destination database
		_, err = copyTaskDbToDb(srcDb, dstDb, srcModel, dstModel, srcTl[k].Task.TaskId, pub, dstLang)
		if err != nil {
			return err
		}
	}
	return nil
}

// copyTaskDbToDb do copy modeling task metadata and task run history from source to destination database
// it return destination task id (task id in destination database)
func copyTaskDbToDb(
	srcDb *sql.DB, dstDb *sql.DB, srcModel *db.ModelMeta, dstModel *db.ModelMeta, srcId int, pub *db.TaskPub, dstLang *db.LangMeta) (int, error) {

	// validate parameters
	if pub == nil {
		return 0, helper.ErrorNew("invalid (empty) source modeling task metadata, source task not found or not exists")
	}

	// destination: convert from "public" format into destination db rows
	dstTask, isSetNotFound, isTaskRunNotFound, err := pub.FromPublic(dstDb, dstModel, true)
	if err != nil {
		return 0, err
	}
	if isSetNotFound {
		omppLog.LogFmt("Warning: task %s workset(s) not found, copy of task incomplete", dstTask.Task.Name)
	}
	if isTaskRunNotFound {
		omppLog.LogFmt("Warning: task %s worksets or model runs not found, copy of task run history incomplete", dstTask.Task.Name)
	}

	// destination: save modeling task metadata
	err = dstTask.UpdateTaskFull(dstDb, dstModel, dstLang)
	if err != nil {
		return 0, err
	}
	dstId := dstTask.Task.TaskId
	omppLog.LogFmt("Modeling task from %d %s to %d", srcId, pub.Name, dstId)

	return dstId, nil
}
