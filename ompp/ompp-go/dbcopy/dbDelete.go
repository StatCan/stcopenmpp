// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// delete model from database
func dbDeleteModel(modelName string, modelDigest string, runOpts *config.RunOptions) error {

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

	// delete model metadata and drop model tables
	omppLog.Log("Delete:", modelName, modelDigest)

	err = db.DeleteModel(srcDb, modelId)
	if err != nil {
		return helper.ErrorNew("model delete failed", modelName, modelDigest, ":", err)
	}
	return nil
}

// delete model run metadata, parameters run values and outpurt tables run values from database
func dbDeleteRun(modelName string, modelDigest string, runOpts *config.RunOptions) error {

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
	if !db.IsRunCompleted(runRow.Status) && runRow.Status != db.DeleteRunStatus {
		return helper.ErrorNew("model run not completed:", runRow.RunId, runRow.Name, runRow.RunDigest)
	}

	// delete model run metadata, parameters run values and output tables run values from database
	omppLog.Log("Delete model run:", runRow.RunId, runRow.Name, runRow.RunDigest)

	err = db.DeleteRun(srcDb, runRow.RunId)
	if err != nil {
		return helper.ErrorNew("failed to delete model run", runRow.RunId, ":", runRow.Name, runRow.RunDigest, ":", err)
	}
	return nil
}

// delete workset metadata and workset parameter values from database
func dbDeleteWorkset(modelName string, modelDigest string, runOpts *config.RunOptions) error {

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

	// check: workset must be read-write in order to delete
	if wsRow.IsReadonly {
		return helper.ErrorNew("workset is read-only:", wsRow.SetId, wsRow.Name)
	}

	// delete workset metadata and workset parameter values from database
	omppLog.Log("Delete workset:", wsRow.SetId, wsRow.Name)

	err = db.DeleteWorkset(srcDb, wsRow.SetId)
	if err != nil {
		return helper.ErrorNew("failed to delete workset", wsRow.SetId, wsRow.Name, err)
	}
	return nil
}

// delete modeling task and task run history from database
func dbDeleteTask(modelName string, modelDigest string, runOpts *config.RunOptions) error {

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

	// delete modeling task and task run history from database
	omppLog.Log("Delete task:", taskRow.TaskId, taskRow.Name)

	err = db.DeleteTask(srcDb, taskRow.TaskId)
	if err != nil {
		return helper.ErrorNew("failed to delete modeling task", taskRow.TaskId, taskRow.Name+" ", err)
	}
	return nil
}
