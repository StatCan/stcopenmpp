// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
)

// RenameTask do rename task if new name is not empty "" string
func RenameTask(dbConn *sql.DB, taskId int, newTaskName string) (bool, error) {

	if newTaskName == "" {
		return false, nil // exit if new name is empty: nothing to do
	}

	// validate parameters
	if taskId <= 0 {
		return false, errors.New("invalid task id: " + strconv.Itoa(taskId))
	}

	// check if new name is unique
	err := SelectFirst(dbConn,
		"SELECT COUNT(*) FROM task_lst WHERE task_id <> "+strconv.Itoa(taskId)+" AND task_name = "+ToQuoted(newTaskName),
		func(row *sql.Row) error {
			nCnt := 0
			if err := row.Scan(&nCnt); err != nil {
				return err
			}
			if nCnt != 0 {
				return errors.New("failed to update: task name must be unique: " + newTaskName)
			}
			return nil
		})
	if err != nil {
		return false, err
	}

	// do update in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return false, err
	}
	err = TrxUpdate(trx,
		"UPDATE task_lst SET task_name = "+toQuotedMax(newTaskName, nameDbMax)+" WHERE task_id = "+strconv.Itoa(taskId))
	if err != nil {
		trx.Rollback()
		return false, err
	}
	trx.Commit()

	return true, nil
}

// UpdateTaskFull delete existing and insert new modeling task and task run history in database.
//
// Model id, task id, run id, set id updated with actual database id's.
// Task input worksets (new content of task_set table) must exist in workset_lst table.
func (meta *TaskMeta) UpdateTaskFull(dbConn *sql.DB, modelDef *ModelMeta, langDef *LangMeta) error {

	// validate parameters
	if modelDef == nil {
		return errors.New("invalid (empty) model metadata")
	}
	if langDef == nil {
		return errors.New("invalid (empty) language list")
	}
	if meta.Task.ModelId != modelDef.Model.ModelId {
		return errors.New("model task: " + strconv.Itoa(meta.Task.TaskId) + " " + meta.Task.Name + " invalid model id " + strconv.Itoa(meta.Task.ModelId) + " expected: " + strconv.Itoa(modelDef.Model.ModelId))
	}

	// do update in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	if err = doReplaceTaskFull(trx, modelDef, meta, langDef); err != nil {
		trx.Rollback()
		return err
	}
	trx.Commit()

	return nil
}

// doReplaceTaskFull delete existing and insert new tasks and task run history in database.
// It does update as part of transaction
// Model id, task id, run id, set id updated with actual database id's.
func doReplaceTaskFull(trx *sql.Tx, modelDef *ModelMeta, meta *TaskMeta, langDef *LangMeta) error {

	// insert new or update existing task_lst master row by task name
	isNew, err := doCreateTaskRow(trx, modelDef, meta)
	if err != nil {
		return err
	}
	stId := strconv.Itoa(meta.Task.TaskId)

	if isNew {
		// insert new rows into task body tables: task_txt, task_set, task_run_lst, task_run_set
		if err = doInsertTaskBody(trx, modelDef, meta, langDef); err != nil {
			return err
		}
	} else {

		// delete existing task_run_set, task_run_lst, task_set, task_txt
		err = TrxUpdate(trx, "DELETE FROM task_run_set WHERE task_id ="+stId)
		if err != nil {
			return err
		}
		err = TrxUpdate(trx, "DELETE FROM task_run_lst WHERE task_id ="+stId)
		if err != nil {
			return err
		}
		err = TrxUpdate(trx, "DELETE FROM task_set WHERE task_id ="+stId)
		if err != nil {
			return err
		}
		err = TrxUpdate(trx, "DELETE FROM task_txt WHERE task_id ="+stId)
		if err != nil {
			return err
		}

		// insert new rows into task body tables: task_txt, task_set, task_run_lst, task_run_set
		if err = doInsertTaskBody(trx, modelDef, meta, langDef); err != nil {
			return err
		}

		// restore actual task name, if required
		// UPDATE task_lst SET task_name = 'my task' WHERE task_id = 88
		err = TrxUpdate(trx,
			"UPDATE task_lst SET task_name = "+toQuotedMax(meta.Task.Name, nameDbMax)+" WHERE task_id = "+stId)
		if err != nil {
			return err
		}
	}

	return nil
}

// doInsertTaskBody insert new rows into task body tables: task_txt, task_set, task_run_lst, task_run_set
// It does update as part of transaction
// Task id and task run id updated with actual database id's.
func doInsertTaskBody(trx *sql.Tx, modelDef *ModelMeta, meta *TaskMeta, langDef *LangMeta) error {

	stId := strconv.Itoa(meta.Task.TaskId)

	// update task text (description and notes)
	for j := range meta.Txt {

		// update task id
		meta.Txt[j].TaskId = meta.Task.TaskId

		// if language code valid then insert into task_txt
		if lId, ok := langDef.IdByCode(meta.Txt[j].LangCode); ok {

			err := TrxUpdate(trx,
				"INSERT INTO task_txt (task_id, lang_id, descr, note) VALUES ("+
					stId+", "+
					strconv.Itoa(lId)+", "+
					toQuotedMax(meta.Txt[j].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(meta.Txt[j].Note, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	// update task input: task workset list
	for j := range meta.Set {
		err := TrxUpdate(trx,
			"INSERT INTO task_set (task_id, set_id) VALUES ("+
				stId+", "+
				strconv.Itoa(meta.Set[j])+")")
		if err != nil {
			return err
		}
	}

	// task run history and status
	for k := range meta.TaskRun {

		// get new task run id
		id := 0
		err := TrxUpdate(trx, "UPDATE id_lst SET id_value = id_value + 1 WHERE id_key = 'run_id_set_id'")
		if err != nil {
			return err
		}
		err = TrxSelectFirst(trx,
			"SELECT id_value FROM id_lst WHERE id_key = 'run_id_set_id'",
			func(row *sql.Row) error {
				return row.Scan(&id)
			})
		switch {
		case err == sql.ErrNoRows:
			return errors.New("invalid destination database, likely not an openM++ database")
		case err != nil:
			return err
		}
		meta.TaskRun[k].TaskRunId = id            // update task run id
		meta.TaskRun[k].TaskId = meta.Task.TaskId // update task id

		// insert into task_run_lst
		err = TrxUpdate(trx,
			"INSERT INTO task_run_lst (task_run_id, task_id, run_name, sub_count, create_dt, status, update_dt, run_stamp)"+
				" VALUES ("+
				strconv.Itoa(id)+", "+
				stId+", "+
				toQuotedMax(meta.TaskRun[k].Name, nameDbMax)+", "+
				strconv.Itoa(meta.TaskRun[k].SubCount)+", "+
				ToQuoted(meta.TaskRun[k].CreateDateTime)+", "+
				ToQuoted(meta.TaskRun[k].Status)+", "+
				ToQuoted(meta.TaskRun[k].UpdateDateTime)+", "+
				toQuotedMax(meta.TaskRun[k].RunStamp, codeDbMax)+")")
		if err != nil {
			return err
		}

		// update task run history body: set task run id and task id
		for j := range meta.TaskRun[k].TaskRunSet {

			meta.TaskRun[k].TaskRunSet[j].TaskRunId = id            // update task run id
			meta.TaskRun[k].TaskRunSet[j].TaskId = meta.Task.TaskId // update task id

			// insert into task_run_set
			err := TrxUpdate(trx,
				"INSERT INTO task_run_set (task_run_id, run_id, set_id, task_id)"+
					" VALUES ("+
					strconv.Itoa(meta.TaskRun[k].TaskRunSet[j].TaskRunId)+", "+
					strconv.Itoa(meta.TaskRun[k].TaskRunSet[j].RunId)+", "+
					strconv.Itoa(meta.TaskRun[k].TaskRunSet[j].SetId)+", "+
					stId+")")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ReplaceTaskDef delete existing and insert new modeling task definition: task metadata and task input worksets.
//
// It does replace: task_txt and task_set db rows.
// It does not change anything in task run history: task_run_lst and task_run_set tables.
// If task not exist then new task created.
// Task input worksets (new content of task_set table) must exist in workset_lst.set_id.
// Model id and task id updated with actual database id's.
func (meta *TaskMeta) ReplaceTaskDef(dbConn *sql.DB, modelDef *ModelMeta, langDef *LangMeta) error {

	// validate parameters
	if modelDef == nil {
		return errors.New("invalid (empty) model metadata")
	}
	if langDef == nil {
		return errors.New("invalid (empty) language list")
	}
	if meta.Task.ModelId != modelDef.Model.ModelId {
		return errors.New("model task: " + strconv.Itoa(meta.Task.TaskId) + " " + meta.Task.Name +
			" invalid model id " + strconv.Itoa(meta.Task.ModelId) +
			" expected: " + strconv.Itoa(modelDef.Model.ModelId))
	}

	// do update in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	if err = doReplaceTask(trx, modelDef, meta, langDef); err != nil {
		trx.Rollback()
		return err
	}
	trx.Commit()

	return nil
}

// MergeTaskDef update existing and insert new modeling task definition: task metadata and task input worksets.
//
// It does replace: task_txt and task_set db rows.
// It does not change anything in task run history: task_run_lst and task_run_set tables.
// If task not exist then new task created.
// Task input worksets (new content of task_set table) must exist in workset_lst.set_id.
// Model id and task id updated with actual database id's.
func (meta *TaskMeta) MergeTaskDef(dbConn *sql.DB, modelDef *ModelMeta, langDef *LangMeta) error {

	// validate parameters
	if modelDef == nil {
		return errors.New("invalid (empty) model metadata")
	}
	if langDef == nil {
		return errors.New("invalid (empty) language list")
	}
	if meta.Task.ModelId != modelDef.Model.ModelId {
		return errors.New("model task: " + strconv.Itoa(meta.Task.TaskId) + " " + meta.Task.Name +
			" invalid model id " + strconv.Itoa(meta.Task.ModelId) +
			" expected: " + strconv.Itoa(modelDef.Model.ModelId))
	}

	// do update in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	if err = doMergeTask(trx, modelDef, meta, langDef); err != nil {
		trx.Rollback()
		return err
	}
	trx.Commit()

	return nil
}

// doReplaceTask delete existing and insert new modeling task definition: task metadata and task input worksets.
// It does update as part of transaction.
// Model id and task id updated with actual database id's.
func doReplaceTask(trx *sql.Tx, modelDef *ModelMeta, meta *TaskMeta, langDef *LangMeta) error {

	// insert new or update existing task_lst master row by task name
	isNew, err := doCreateTaskRow(trx, modelDef, meta)
	if err != nil {
		return err
	}

	// delete existing and insert new task text description and notes and task sets: task_txt and task_set rows
	err = doReplaceTaskDef(trx, modelDef, meta, langDef)
	if err != nil {
		return err
	}

	// restore actual task name, if required
	if !isNew {
		// UPDATE task_lst SET task_name = 'my task' WHERE task_id = 88
		err = TrxUpdate(trx,
			"UPDATE task_lst SET task_name = "+toQuotedMax(meta.Task.Name, nameDbMax)+" WHERE task_id = "+strconv.Itoa(meta.Task.TaskId))
		if err != nil {
			return err
		}
	}
	return nil
}

// doMergeTask update existing and insert new modeling task definition: task metadata and task input worksets.
// It does update as part of transaction.
// Model id and task id updated with actual database id's.
func doMergeTask(trx *sql.Tx, modelDef *ModelMeta, meta *TaskMeta, langDef *LangMeta) error {

	// insert new or update existing task_lst master row by task name
	isNew, err := doCreateTaskRow(trx, modelDef, meta)
	if err != nil {
		return err
	}

	// insert new or update existing task text description and notes and task sets: task_txt and task_set rows
	err = doMergeTaskDef(trx, modelDef, meta, langDef)
	if err != nil {
		return err
	}

	// restore actual task name, if required
	if !isNew {
		// UPDATE task_lst SET task_name = 'my task' WHERE task_id = 88
		err = TrxUpdate(trx,
			"UPDATE task_lst SET task_name = "+toQuotedMax(meta.Task.Name, nameDbMax)+" WHERE task_id = "+strconv.Itoa(meta.Task.TaskId))
		if err != nil {
			return err
		}
	}
	return nil
}

// doCreateTaskRow insert new or update existing task_lst master row by task name.
// It does update as part of transaction.
// Return true if new task_lst row inserted
func doCreateTaskRow(trx *sql.Tx, modelDef *ModelMeta, meta *TaskMeta) (bool, error) {

	// task name cannot be empty
	if meta.Task.Name == "" {
		return false, errors.New("invalid (empty) task name, id: " + strconv.Itoa(meta.Task.TaskId))
	}
	meta.Task.ModelId = modelDef.Model.ModelId // update model id
	smId := strconv.Itoa(modelDef.Model.ModelId)

	// get new task id if task not exist
	//
	// UPDATE id_lst SET id_value =
	//   CASE
	//     WHEN 0 = (SELECT COUNT(*) FROM task_lst WHERE model_id = 1 AND task_name = 'task_44')
	//       THEN id_value + 1
	//     ELSE id_value
	//   END
	// WHERE id_key = 'run_id_set_id'
	err := TrxUpdate(trx,
		"UPDATE id_lst SET id_value ="+
			" CASE"+
			" WHEN 0 ="+
			" (SELECT COUNT(*) FROM task_lst"+
			" WHERE model_id = "+smId+" AND task_name = "+ToQuoted(meta.Task.Name)+
			" )"+
			" THEN id_value + 1"+
			" ELSE id_value"+
			" END"+
			" WHERE id_key = 'run_id_set_id'")
	if err != nil {
		return false, err
	}

	// check if this task already exist
	taskId := 0
	err = TrxSelectFirst(trx,
		"SELECT task_id FROM task_lst WHERE model_id = "+smId+" AND task_name = "+ToQuoted(meta.Task.Name),
		func(row *sql.Row) error {
			return row.Scan(&taskId)
		})
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	// if task not exist then insert new task master row
	isNew := taskId <= 0
	if isNew {

		// get new task id
		err = TrxSelectFirst(trx,
			"SELECT id_value FROM id_lst WHERE id_key = 'run_id_set_id'",
			func(row *sql.Row) error {
				return row.Scan(&taskId)
			})
		switch {
		case err == sql.ErrNoRows:
			return false, errors.New("invalid destination database, likely not an openM++ database")
		case err != nil:
			return false, err
		}

		// INSERT INTO task_lst (task_id, model_id, task_name) VALUES (88, 11, 'modelOne task')
		err = TrxUpdate(trx,
			"INSERT INTO task_lst (task_id, model_id, task_name) VALUES ("+
				strconv.Itoa(taskId)+", "+
				smId+", "+
				toQuotedMax(meta.Task.Name, nameDbMax)+")")
		if err != nil {
			return false, err
		}

	} else { // lock task: update with unique name

		// UPDATE task_lst SET task_name = 'task_88_2014-08-17 16:57:04.0123' WHERE task_id = 88
		err := TrxUpdate(trx,
			"UPDATE task_lst"+
				" SET task_name = "+ToQuoted("task_"+strconv.Itoa(taskId)+"_"+helper.MakeDateTime(time.Now()))+
				" WHERE task_id = "+strconv.Itoa(taskId))
		if err != nil {
			return false, err
		}
	}
	meta.Task.TaskId = taskId // update task id with actual value

	return isNew, nil
}

// doReplaceTaskDef delete existing and insert new task text description and notes and task sets: task_txt and task_set rows.
// It does update as part of transaction.
// Model id and task id updated with actual database id's.
func doReplaceTaskDef(trx *sql.Tx, modelDef *ModelMeta, meta *TaskMeta, langDef *LangMeta) error {

	// delete existing task text and task input sets
	stId := strconv.Itoa(meta.Task.TaskId)

	err := TrxUpdate(trx, "DELETE FROM task_set WHERE task_id ="+stId)
	if err != nil {
		return err
	}
	err = TrxUpdate(trx, "DELETE FROM task_txt WHERE task_id ="+stId)
	if err != nil {
		return err
	}

	// update task text (description and notes)
	for j := range meta.Txt {

		// update task id
		meta.Txt[j].TaskId = meta.Task.TaskId

		// if language code valid then insert into task_txt
		if lId, ok := langDef.IdByCode(meta.Txt[j].LangCode); ok {

			err = TrxUpdate(trx,
				"INSERT INTO task_txt (task_id, lang_id, descr, note) VALUES ("+
					stId+", "+
					strconv.Itoa(lId)+", "+
					toQuotedMax(meta.Txt[j].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(meta.Txt[j].Note, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	// update task input: task workset list
	for j := range meta.Set {
		err = TrxUpdate(trx,
			"INSERT INTO task_set (task_id, set_id) VALUES ("+
				stId+", "+
				strconv.Itoa(meta.Set[j])+")")
		if err != nil {
			return err
		}
	}
	return nil
}

// doMergeTaskDef update existing and insert new task text description and notes and task sets: task_txt and task_set rows.
// It does update as part of transaction.
// Model id and task id updated with actual database id's.
func doMergeTaskDef(trx *sql.Tx, modelDef *ModelMeta, meta *TaskMeta, langDef *LangMeta) error {

	// update task text (description and notes)
	stId := strconv.Itoa(meta.Task.TaskId)

	for j := range meta.Txt {

		// update task id
		meta.Txt[j].TaskId = meta.Task.TaskId

		// if language code valid then insert into task_txt
		if lId, ok := langDef.IdByCode(meta.Txt[j].LangCode); ok {

			err := TrxUpdate(trx, "DELETE FROM task_txt WHERE task_id ="+stId+" AND lang_id = "+strconv.Itoa(lId))
			if err != nil {
				return err
			}
			err = TrxUpdate(trx,
				"INSERT INTO task_txt (task_id, lang_id, descr, note) VALUES ("+
					stId+", "+
					strconv.Itoa(lId)+", "+
					toQuotedMax(meta.Txt[j].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(meta.Txt[j].Note, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	// update task input: task workset list
	for j := range meta.Set {
		err := TrxUpdate(trx,
			"INSERT INTO task_set (task_id, set_id)"+
				" SELECT "+stId+", "+strconv.Itoa(meta.Set[j])+
				" FROM workset_lst W"+
				" WHERE W.set_id = "+strconv.Itoa(meta.Set[j])+
				" AND NOT EXISTS ("+
				" SELECT * FROM task_set E WHERE E.task_id ="+stId+" AND E.set_id = "+strconv.Itoa(meta.Set[j])+
				")")
		if err != nil {
			return err
		}
	}
	return nil
}
