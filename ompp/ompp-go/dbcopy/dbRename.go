// Copyright (c) 2020 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// rename model run
func dbRenameRun(modelName string, modelDigest string, runOpts *config.RunOptions) error {

	// new run name argument required and cannot be empty
	newRunName := runOpts.String(runNewNameArgKey)
	if newRunName == "" {
		return helper.ErrorNew("dbcopy invalid (empty or missing) argument of:", runNewNameArgKey)
	}

	// open source database connection and check is it valid
	cs, dn := db.IfEmptyMakeDefault(modelName, runOpts.String(fromSqliteArgKey), runOpts.String(dbConnStrArgKey), runOpts.String(dbDriverArgKey))

	srcDb, _, err := db.Open(cs, dn, false)
	if err != nil {
		return err
	}
	defer srcDb.Close()

	if err := db.CheckOpenmppSchemaVersion(srcDb); err != nil {
		return err
	}

	// find the model by name and/or digest
	isFound, modelId, err := db.GetModelId(srcDb, modelName, modelDigest)
	if err != nil {
		return err
	}
	if !isFound {
		return helper.ErrorFmt("model %s %s not found", modelName, modelDigest)
	}

	// find model run metadata by id, run digest or name
	runId, runDigest, runName, isFirst, isLast := runIdDigestNameFromOptions(runOpts)
	if runId < 0 || runId == 0 && runName == "" && runDigest == "" && !isFirst && !isLast {
		return helper.ErrorFmt("dbcopy invalid argument(s) run id: %s, run name: %s, run digest: %s",
			runOpts.String(runIdArgKey), runOpts.String(runNameArgKey), runOpts.String(runDigestArgKey))
	}
	runRow, e := findModelRunByIdDigestName(srcDb, modelId, runId, runDigest, runName, isFirst, isLast)
	if e != nil {
		return e
	}
	if runRow == nil {
		return helper.ErrorNew("Model run not found:", runOpts.String(runIdArgKey), runOpts.String(runNameArgKey), runOpts.String(runDigestArgKey))
	}

	// check is this run belong to the model
	if runRow.ModelId != modelId {
		return helper.ErrorFmt("model run %d %s %s does not belong to model %s %s", runRow.RunId, runRow.Name, runRow.RunDigest, modelName, modelDigest)
	}

	// run must be completed: status success, error or exit
	if !db.IsRunCompleted(runRow.Status) {
		return helper.ErrorNew("model run not completed:", runRow.RunId, runRow.Name, runRow.RunDigest)
	}

	// rename model run
	omppLog.LogFmt("Rename model run %d % s into: %s", runRow.RunId, runRow.Name, newRunName)

	isFound, err = db.RenameRun(srcDb, runRow.RunId, newRunName)
	if err != nil {
		helper.ErrorNew("failed to rename model run", runRow.RunId, runRow.Name, ":", err)
	}
	if !isFound {
		return helper.ErrorNew("Model run not found:", runRow.RunId, runRow.Name)
	}
	return nil
}

// rename workset
func dbRenameWorkset(modelName string, modelDigest string, runOpts *config.RunOptions) error {

	// new workset name argument required and cannot be empty
	newSetName := runOpts.String(setNewNameArgKey)
	if newSetName == "" {
		return helper.ErrorNew("dbcopy invalid (empty or missing) argument of:", setNewNameArgKey)
	}

	// get workset name and id
	setName := runOpts.String(setNameArgKey)
	setId := runOpts.Int(setIdArgKey, 0)

	// conflicting options: use set id if positive else use set name
	if runOpts.IsExist(setNameArgKey) && runOpts.IsExist(setIdArgKey) {
		if setId > 0 {
			omppLog.LogFmt("dbcopy options conflict. Using set id: %d, not a set name: %s", setId, setName)
			setName = ""
		} else {
			omppLog.LogFmt("dbcopy options conflict. Using set name: %s, not a set id: %d", setName, setId)
			setId = 0
		}
	}

	if setId < 0 || setId == 0 && setName == "" {
		return helper.ErrorFmt("dbcopy invalid argument(s) for set id: %s and/or set name: %s", runOpts.String(setIdArgKey), runOpts.String(setNameArgKey))
	}

	// open source database connection and check is it valid
	cs, dn := db.IfEmptyMakeDefault(modelName, runOpts.String(fromSqliteArgKey), runOpts.String(dbConnStrArgKey), runOpts.String(dbDriverArgKey))

	srcDb, _, err := db.Open(cs, dn, false)
	if err != nil {
		return err
	}
	defer srcDb.Close()

	if err := db.CheckOpenmppSchemaVersion(srcDb); err != nil {
		return err
	}

	// find the model
	isFound, modelId, err := db.GetModelId(srcDb, modelName, modelDigest)
	if err != nil {
		return err
	}
	if !isFound {
		return helper.ErrorFmt("model %s %s not found", modelName, modelDigest)
	}

	// get workset metadata by id or name
	var wsRow *db.WorksetRow
	if setId > 0 {
		if wsRow, err = db.GetWorkset(srcDb, setId); err != nil {
			return err
		}
		if wsRow == nil {
			return helper.ErrorNew("workset not found, set id:", setId)
		}
	} else {
		if wsRow, err = db.GetWorksetByName(srcDb, modelId, setName); err != nil {
			return err
		}
		if wsRow == nil {
			return helper.ErrorNew("Workset not found:", setName)
		}
	}

	// check is this workset belong to the model
	if wsRow.ModelId != modelId {
		return helper.ErrorFmt("workset %d %s does not belong to model %s %s", wsRow.SetId, wsRow.Name, modelName, modelDigest)
	}

	// rename workset (even it is read-only)
	omppLog.LogFmt("Rename workset %d %s into: %s", wsRow.SetId, wsRow.Name, newSetName)

	isFound, err = db.RenameWorkset(srcDb, wsRow.SetId, newSetName, true)
	if err != nil {
		return helper.ErrorNew("failed to rename workset", wsRow.SetId, wsRow.Name, err)
	}
	if !isFound {
		return helper.ErrorNew("Workset not found:", wsRow.SetId, wsRow.Name)
	}

	return nil
}

// rename modeling task
func dbRenameTask(modelName string, modelDigest string, runOpts *config.RunOptions) error {

	// new task name argument required and cannot be empty
	newTaskName := runOpts.String(taskNewNameArgKey)
	if newTaskName == "" {
		return helper.ErrorNew("dbcopy invalid (empty or missing) argument of:", taskNewNameArgKey)
	}

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
	cs, dn := db.IfEmptyMakeDefault(modelName, runOpts.String(fromSqliteArgKey), runOpts.String(dbConnStrArgKey), runOpts.String(dbDriverArgKey))

	srcDb, _, err := db.Open(cs, dn, false)
	if err != nil {
		return err
	}
	defer srcDb.Close()

	if err := db.CheckOpenmppSchemaVersion(srcDb); err != nil {
		return err
	}

	// find the model
	isFound, modelId, err := db.GetModelId(srcDb, modelName, modelDigest)
	if err != nil {
		return err
	}
	if !isFound {
		return helper.ErrorFmt("model %s %s not found", modelName, modelDigest)
	}

	// find modeling task by id or name
	var taskRow *db.TaskRow
	if taskId > 0 {
		if taskRow, err = db.GetTask(srcDb, taskId); err != nil {
			return err
		}
		if taskRow == nil {
			return helper.ErrorNew("modeling task not found, task id:", taskId)
		}
	} else {
		if taskRow, err = db.GetTaskByName(srcDb, modelId, taskName); err != nil {
			return err
		}
		if taskRow == nil {
			return helper.ErrorNew("modeling task not found:", taskName)
		}
	}

	// check is this task belong to the model
	if taskRow.ModelId != modelId {
		return helper.ErrorFmt("modeling task %d % s does not belong to model %s %s", taskRow.TaskId, taskRow.Name, modelName, modelDigest)
	}

	// rename modeling task
	omppLog.LogFmt("Rename task %d %s into: %s", taskRow.TaskId, taskRow.Name, newTaskName)

	isFound, err = db.RenameTask(srcDb, taskRow.TaskId, newTaskName)
	if err != nil {
		return helper.ErrorNew("failed to rename modeling task", taskRow.TaskId, taskRow.Name, err)
	}
	if !isFound {
		return helper.ErrorNew("modeling task not found:", taskRow.TaskId, taskRow.Name)
	}
	return nil
}
