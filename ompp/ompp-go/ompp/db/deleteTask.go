// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"errors"
	"strconv"
)

// Delete modeling task and task run history from database.
// Only task itself is deleted, model runs and worksets are not affected.
func DeleteTask(dbConn *sql.DB, taskId int) error {

	// validate parameters
	if taskId <= 0 {
		return errors.New("invalid task id: " + strconv.Itoa(taskId))
	}

	// delete inside of transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	if err := doDeleteTask(trx, taskId); err != nil {
		trx.Rollback()
		return err
	}
	trx.Commit()
	return nil
}

// delete modeling task and task run history from database.
// Only task itself is deleted from task_lst and other task_ tables, model runs and worksets are not affected.
// It does update as part of transaction
func doDeleteTask(trx *sql.Tx, taskId int) error {

	// update task master record to prevent task use
	stId := strconv.Itoa(taskId)
	err := TrxUpdate(trx, "UPDATE task_lst SET task_name = 'deleted' WHERE task_id = "+stId)
	if err != nil {
		return err
	}

	// delete modeling task and task run history
	err = TrxUpdate(trx, "DELETE FROM task_run_set WHERE task_id = "+stId)
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM task_run_lst WHERE task_id = "+stId)
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM task_set WHERE task_id = "+stId)
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM task_txt WHERE task_id = "+stId)
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM task_lst WHERE task_id = "+stId)
	if err != nil {
		return err
	}

	return nil
}
