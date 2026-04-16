// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"errors"
	"strconv"
)

// GetTask return modeling task rows by id: task_lst table row and set ids from task_set
func GetTask(dbConn *sql.DB, taskId int) (*TaskRow, error) {
	return getTaskRow(dbConn,
		"SELECT K.task_id, K.model_id, K.task_name FROM task_lst K"+
			" WHERE K.task_id ="+strconv.Itoa(taskId))
}

// GetTaskByName return modeling task rows by id: task_lst table row and set ids from task_set
func GetTaskByName(dbConn *sql.DB, modelId int, name string) (*TaskRow, error) {
	return getTaskRow(dbConn,
		"SELECT K.task_id, K.model_id, K.task_name FROM task_lst K"+
			" WHERE K.task_id = "+
			" ("+
			" SELECT MIN(M.task_id) FROM task_lst M"+
			" WHERE M.model_id = "+strconv.Itoa(modelId)+" AND M.task_name ="+ToQuoted(name)+
			" )")
}

// GetTaskList return list of model tasks: task_lst rows.
func GetTaskList(dbConn *sql.DB, modelId int) ([]TaskRow, error) {

	// model not found: model id must be positive
	if modelId <= 0 {
		return nil, nil
	}

	// get list of modeling task for that model id
	q := "SELECT K.task_id, K.model_id, K.task_name FROM task_lst K" +
		" WHERE K.model_id =" + strconv.Itoa(modelId) +
		" ORDER BY 1"

	taskRs, err := getTaskLst(dbConn, q)
	if err != nil {
		return nil, err
	}
	if len(taskRs) <= 0 { // no tasks found
		return nil, nil
	}
	return taskRs, nil
}

// GetTaskListText return list of modeling tasks with description and notes: task_lst and task_txt rows.
//
// If langCode not empty then only specified language selected else all languages
func GetTaskListText(dbConn *sql.DB, modelId int, langCode string) ([]TaskRow, []TaskTxtRow, error) {

	// model not found: model id must be positive
	if modelId <= 0 {
		return nil, nil, nil
	}

	// get list of modeling task for that model id
	q := "SELECT K.task_id, K.model_id, K.task_name FROM task_lst K" +
		" WHERE K.model_id =" + strconv.Itoa(modelId) +
		" ORDER BY 1"

	taskRs, err := getTaskLst(dbConn, q)
	if err != nil {
		return nil, nil, err
	}
	if len(taskRs) <= 0 { // no tasks found
		return nil, nil, err
	}

	// get tasks description and notes by model id
	q = "SELECT M.task_id, M.lang_id, L.lang_code, M.descr, M.note" +
		" FROM task_txt M" +
		" INNER JOIN task_lst K ON (K.task_id = M.task_id)" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE K.model_id = " + strconv.Itoa(modelId)
	if langCode != "" {
		q += " AND L.lang_code = " + ToQuoted(langCode)
	}
	q += " ORDER BY 1, 2"

	txtRs, err := getTaskText(dbConn, q)
	if err != nil {
		return nil, nil, err
	}
	return taskRs, txtRs, nil
}

// GetTaskSetIds return modeling task set id's by task id from task_set db table
func GetTaskSetIds(dbConn *sql.DB, taskId int) ([]int, error) {

	var idRs []int

	err := SelectRows(dbConn,
		"SELECT TS.set_id FROM task_set TS WHERE TS.task_id = "+strconv.Itoa(taskId)+" ORDER BY 1",
		func(rows *sql.Rows) error {
			var id int
			if err := rows.Scan(&id); err != nil {
				return err
			}
			idRs = append(idRs, id)
			return nil
		})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return idRs, nil
}

// getTaskRow return task_lst table row.
func getTaskRow(dbConn *sql.DB, query string) (*TaskRow, error) {

	var taskRow TaskRow

	err := SelectFirst(dbConn, query,
		func(row *sql.Row) error {
			if err := row.Scan(
				&taskRow.TaskId, &taskRow.ModelId, &taskRow.Name); err != nil {
				return err
			}
			return nil
		})
	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, err
	}

	return &taskRow, nil
}

// getTaskLst return task_lst table rows.
func getTaskLst(dbConn *sql.DB, query string) ([]TaskRow, error) {

	// get list of modeling task for that model id
	var taskRs []TaskRow

	err := SelectRows(dbConn, query,
		func(rows *sql.Rows) error {
			var r TaskRow
			if err := rows.Scan(&r.TaskId, &r.ModelId, &r.Name); err != nil {
				return err
			}
			taskRs = append(taskRs, r)
			return nil
		})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return taskRs, nil
}

// getTaskSetLst return modeling tasks id map to set ids from task_set table
func getTaskSetLst(dbConn *sql.DB, query string) (map[int][]int, error) {

	var tsRs = make(map[int][]int)

	err := SelectRows(dbConn, query,
		func(rows *sql.Rows) error {
			var taskId, setId int
			if err := rows.Scan(&taskId, &setId); err != nil {
				return err
			}
			tsRs[taskId] = append(tsRs[taskId], setId)
			return nil
		})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return tsRs, nil
}

// GetTaskRunSetText return additinal modeling task text description and notes for all task worksets and model runs: workset_txt, run_txt table rows.
//
// It includes all input worksets text and model runs text.
// If isSuccess true then return only successfully completed runs
// else return all runs: success, error, exit, progress.
// If langCode not empty then only specified language selected else all languages.
func GetTaskRunSetText(dbConn *sql.DB, taskId int, isSuccess bool, langCode string) (*TaskRunSetTxt, error) {

	tp := TaskRunSetTxt{
		SetTxt: map[string][]DescrNote{},
		RunTxt: map[string][]DescrNote{}}

	// select description and notes for task input worksets from task_set table
	q := "SELECT M.set_id, M.lang_id, H.set_name, L.lang_code, M.descr, M.note" +
		" FROM task_set TS" +
		" INNER JOIN workset_lst H ON (H.set_id = TS.set_id)" +
		" INNER JOIN workset_txt M ON (M.set_id = H.set_id)" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE TS.task_id = " + strconv.Itoa(taskId)
	if langCode != "" {
		q += " AND L.lang_code = " + ToQuoted(langCode)
	}
	q += " ORDER BY 1, 2"

	err := SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var setId, lId int
			var name string
			var r DescrNote
			var note sql.NullString
			if err := rows.Scan(&setId, &lId, &name, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			// if such language not exist for that workset name then append language, description, notes to that name
			if v, ok := tp.SetTxt[name]; !ok {
				tp.SetTxt[name] = append(tp.SetTxt[name], r)
			} else {
				isExist := false
				for k := range v {
					isExist = v[k].LangCode == r.LangCode
					if isExist {
						break
					}
				}
				if !isExist {
					tp.SetTxt[name] = append(v, r)
				}
			}
			return nil
		})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// select description and notes for task run worksets from task_run_set table
	var statusFilter string
	if isSuccess {
		statusFilter = " AND RL.status = " + ToQuoted(DoneRunStatus)
	} else {
		statusFilter = " AND RL.status IN (" +
			ToQuoted(DoneRunStatus) + ", " + ToQuoted(ErrorRunStatus) + ", " + ToQuoted(ExitRunStatus) + ", " + ToQuoted(ProgressRunStatus) + ")"
	}

	q = "SELECT M.set_id, M.lang_id, WS.set_name, L.lang_code, M.descr, M.note" +
		" FROM task_run_lst TRL" +
		" INNER JOIN task_run_set TRS ON (TRS.task_run_id = TRL.task_run_id)" +
		" INNER JOIN run_lst RL ON (RL.run_id = TRS.run_id)" +
		" INNER JOIN workset_lst WS ON (WS.set_id = TRS.set_id)" +
		" INNER JOIN workset_txt M ON (M.set_id = WS.set_id)" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE TRL.task_id = " + strconv.Itoa(taskId)

	q += statusFilter

	if langCode != "" {
		q += " AND L.lang_code = " + ToQuoted(langCode)
	}
	q += " ORDER BY 1, 2"

	err = SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var setId, lId int
			var name string
			var r DescrNote
			var note sql.NullString
			if err := rows.Scan(&setId, &lId, &name, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			// if such language not exist for that workset name then append language, description, notes to that name
			if v, ok := tp.SetTxt[name]; !ok {
				tp.SetTxt[name] = append(tp.SetTxt[name], r)
			} else {
				isExist := false
				for k := range v {
					isExist = v[k].LangCode == r.LangCode
					if isExist {
						break
					}
				}
				if !isExist {
					tp.SetTxt[name] = append(v, r)
				}
			}
			return nil
		})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// select description and notes for model runs from task_run_set table
	q = "SELECT M.run_id, M.lang_id, RL.run_name, RL.run_digest, L.lang_code, M.descr, M.note" +
		" FROM task_run_lst TRL" +
		" INNER JOIN task_run_set TRS ON (TRS.task_run_id = TRL.task_run_id)" +
		" INNER JOIN run_lst RL ON (RL.run_id = TRS.run_id)" +
		" INNER JOIN run_txt M ON (M.run_id = RL.run_id)" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE TRL.task_id = " + strconv.Itoa(taskId)

	q += statusFilter

	if langCode != "" {
		q += " AND L.lang_code = " + ToQuoted(langCode)
	}
	q += " ORDER BY 1, 2"

	err = SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var runId, lId int
			var dn string
			var r DescrNote
			var digest, note sql.NullString
			if err := rows.Scan(&runId, &lId, &dn, &digest, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if digest.Valid {
				dn = digest.String
			}
			if note.Valid {
				r.Note = note.String
			}
			// if such language not exist for that run digest-or-name then append language, description, notes to that run
			if v, ok := tp.RunTxt[dn]; !ok {
				tp.RunTxt[dn] = append(tp.RunTxt[dn], r)
			} else {
				isExist := false
				for k := range v {
					isExist = v[k].LangCode == r.LangCode
					if isExist {
						break
					}
				}
				if !isExist {
					tp.RunTxt[dn] = append(v, r)
				}
			}
			return nil
		})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return &tp, nil
}

// getTaskText return modeling task description and notes: task_txt table rows.
func getTaskText(dbConn *sql.DB, query string) ([]TaskTxtRow, error) {

	// select db rows from task_txt
	var txtLst []TaskTxtRow

	err := SelectRows(dbConn, query,
		func(rows *sql.Rows) error {
			var r TaskTxtRow
			var lId int
			var note sql.NullString
			if err := rows.Scan(
				&r.TaskId, &lId, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			txtLst = append(txtLst, r)
			return nil
		})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return txtLst, nil
}

// GetTaskRun return modeling task run status: task_run_lst table row.
func GetTaskRun(dbConn *sql.DB, taskRunId int) (*TaskRunRow, error) {
	return getTaskRunRow(dbConn,
		"SELECT task_run_id, task_id, run_name, sub_count, create_dt, status, update_dt, run_stamp"+
			" FROM task_run_lst"+
			" WHERE task_run_id = "+strconv.Itoa(taskRunId))
}

// GetTaskRunByStamp return modeling task run status by task id and task run stamp: task_run_lst table row.
// If there is multiple task runs with this stamp then run with min(task_run_id) returned
func GetTaskRunByStamp(dbConn *sql.DB, taskId int, stamp string) (*TaskRunRow, error) {
	return getTaskRunRow(dbConn,
		"SELECT R.task_run_id, R.task_id, R.run_name, R.sub_count, R.create_dt, R.status, R.update_dt, R.run_stamp"+
			" FROM task_run_lst R"+
			" WHERE R.task_run_id ="+
			" ("+
			" SELECT MIN(M.task_run_id) FROM task_run_lst M"+
			" WHERE M.task_id = "+strconv.Itoa(taskId)+
			" AND M.run_stamp = "+ToQuoted(stamp)+
			")")
}

// GetTaskRunByName return modeling task run status by task id and task run name: task_run_lst table row.
// If there is multiple task runs with this name then run with min(task_run_id) returned
func GetTaskRunByName(dbConn *sql.DB, taskId int, name string) (*TaskRunRow, error) {
	return getTaskRunRow(dbConn,
		"SELECT R.task_run_id, R.task_id, R.run_name, R.sub_count, R.create_dt, R.status, R.update_dt, R.run_stamp"+
			" FROM task_run_lst R"+
			" WHERE R.task_run_id ="+
			" ("+
			" SELECT MIN(M.task_run_id) FROM task_run_lst M"+
			" WHERE M.task_id = "+strconv.Itoa(taskId)+
			" AND M.run_name = "+ToQuoted(name)+
			")")
}

// GetTaskRunByStampOrName return modeling task run status by task id and task run stamp or task run name: task_run_lst table row.
//
// It does select task run row by task id and stamp, if not found then by task id and run name.
// If there is multiple task runs with this stamp or name then run with min(task_run_id) returned.
func GetTaskRunByStampOrName(dbConn *sql.DB, taskId int, trsn string) (*TaskRunRow, error) {

	tr, err := GetTaskRunByStamp(dbConn, taskId, trsn)
	if err == nil && tr == nil {
		tr, err = GetTaskRunByName(dbConn, taskId, trsn)
	}
	return tr, err
}

// GetTaskRunListByStampOrName return modeling task run rows from task_run_lst table by task id and task run stamp or task run name.
//
// It does select task run rows by task id and stamp, if not found then by task id and run name.
// If there is multiple task runs with this stamp or name then multiple rows returned.
func GetTaskRunListByStampOrName(dbConn *sql.DB, taskId int, trsn string) ([]TaskRunRow, error) {

	runRs, err := getTaskRunLst(dbConn,
		"SELECT R.task_run_id, R.task_id, R.run_name, R.sub_count, R.create_dt, R.status, R.update_dt, R.run_stamp"+
			" FROM task_run_lst R"+
			" WHERE R.task_id = "+strconv.Itoa(taskId)+
			" AND R.run_stamp = "+ToQuoted(trsn)+
			" ORDER BY 1")

	if err == nil && len(runRs) <= 0 {
		runRs, err = getTaskRunLst(dbConn,
			"SELECT R.task_run_id, R.task_id, R.run_name, R.sub_count, R.create_dt, R.status, R.update_dt, R.run_stamp"+
				" FROM task_run_lst R"+
				" WHERE R.task_id = "+strconv.Itoa(taskId)+
				" AND R.run_name = "+ToQuoted(trsn)+
				" ORDER BY 1")
	}
	return runRs, err
}

// GetTaskRunSetRows return task run body (pairs of run id and set id) by task run id: task_run_set table rows.
func GetTaskRunSetRows(dbConn *sql.DB, taskRunId int) ([]TaskRunSetRow, error) {
	return getTaskRunSetLst(dbConn,
		"SELECT TRS.task_run_id, TRS.run_id, TRS.set_id, TRS.task_id"+
			" FROM task_run_lst M"+
			" INNER JOIN task_run_set TRS ON (TRS.task_run_id = M.task_run_id)"+
			" WHERE M.task_run_id = "+strconv.Itoa(taskRunId)+
			" ORDER BY 1, 2")
}

// GetTaskFirstRun return first run of the modeling task: task_run_lst table row.
func GetTaskFirstRun(dbConn *sql.DB, taskId int) (*TaskRunRow, error) {
	return getTaskRunRow(dbConn,
		"SELECT"+
			" R.task_run_id, R.task_id, R.run_name, R.sub_count, R.create_dt, R.status, R.update_dt, R.run_stamp"+
			" FROM task_run_lst R"+
			" WHERE R.task_run_id ="+
			" (SELECT MIN(M.task_run_id) FROM task_run_lst M WHERE M.task_id = "+strconv.Itoa(taskId)+")")
}

// GetTaskLastRun return last run of the modeling task: task_run_lst table row.
func GetTaskLastRun(dbConn *sql.DB, taskId int) (*TaskRunRow, error) {
	return getTaskRunRow(dbConn,
		"SELECT"+
			" R.task_run_id, R.task_id, R.run_name, R.sub_count, R.create_dt, R.status, R.update_dt, R.run_stamp"+
			" FROM task_run_lst R"+
			" WHERE R.task_run_id ="+
			" (SELECT MAX(M.task_run_id) FROM task_run_lst M WHERE M.task_id = "+strconv.Itoa(taskId)+")")
}

// GetTaskLastCompletedRun return last completed run of the modeling task: task_run_lst table row.
//
// Task run completed if run status one of: s=success, x=exit, e=error
func GetTaskLastCompletedRun(dbConn *sql.DB, taskId int) (*TaskRunRow, error) {
	return getTaskRunRow(dbConn,
		"SELECT"+
			" R.task_run_id, R.task_id, R.run_name, R.sub_count, R.create_dt, R.status, R.update_dt, R.run_stamp"+
			" FROM task_run_lst R"+
			" WHERE R.task_run_id ="+
			" ("+
			" SELECT MAX(M.task_run_id) FROM task_run_lst M"+
			" WHERE M.task_id = "+strconv.Itoa(taskId)+
			" AND M.status IN ("+ToQuoted(DoneRunStatus)+", "+ToQuoted(ErrorRunStatus)+", "+ToQuoted(ExitRunStatus)+")"+
			" )")
}

// GetTaskRunList return model run history: master row from task_lst and all rows from task_run_lst, task_run_set.
func GetTaskRunList(dbConn *sql.DB, taskRow *TaskRow) (*TaskMeta, error) {

	// validate parameters
	if taskRow == nil {
		return nil, errors.New("invalid (empty) task row, it may be task not found")
	}

	// task meta header: task_lst master row and empty details
	meta := &TaskMeta{TaskDef: TaskDef{
		Task: *taskRow,
		Txt:  []TaskTxtRow{},
		Set:  []int{},
	}}

	// get task run history and status
	runRs, err := getTaskRunLst(dbConn,
		"SELECT M.task_run_id, M.task_id, M.run_name, M.sub_count, M.create_dt, M.status, M.update_dt, M.run_stamp"+
			" FROM task_run_lst M"+
			" WHERE M.task_id = "+strconv.Itoa(taskRow.TaskId)+
			" ORDER BY 1")
	if err != nil {
		return nil, err
	}

	meta.TaskRun = make([]taskRunItem, len(runRs))
	ri := make(map[int]int, len(runRs)) // map (task run id) => index in task run array

	for k := range runRs {
		ri[runRs[k].TaskRunId] = k
		meta.TaskRun[k].TaskRunRow = runRs[k]
	}

	// select run results for the tasks
	runSetRs, err := getTaskRunSetLst(dbConn,
		"SELECT TRS.task_run_id, TRS.run_id, TRS.set_id, TRS.task_id"+
			" FROM task_run_lst M"+
			" INNER JOIN task_run_set TRS ON (TRS.task_run_id = M.task_run_id)"+
			" WHERE M.task_id = "+strconv.Itoa(taskRow.TaskId)+
			" ORDER BY 1, 2")
	if err != nil {
		return nil, err
	}

	for k := range runSetRs {
		if i, ok := ri[runSetRs[k].TaskRunId]; ok {
			meta.TaskRun[i].TaskRunSet = append(meta.TaskRun[i].TaskRunSet, runSetRs[k])
		}
	}

	return meta, nil
}

// getTaskRunRow return task_run_lst table row.
func getTaskRunRow(dbConn *sql.DB, query string) (*TaskRunRow, error) {

	var r TaskRunRow

	err := SelectFirst(dbConn, query,
		func(row *sql.Row) error {
			if err := row.Scan(
				&r.TaskRunId, &r.TaskId, &r.Name, &r.SubCount, &r.CreateDateTime, &r.Status, &r.UpdateDateTime, &r.RunStamp); err != nil {
				return err
			}
			return nil
		})
	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, err
	}

	return &r, nil
}

// getTaskRunLst return list of modeling task runs: task_run_lst table rows.
func getTaskRunLst(dbConn *sql.DB, query string) ([]TaskRunRow, error) {

	var trRs []TaskRunRow

	err := SelectRows(dbConn, query,
		func(rows *sql.Rows) error {
			var r TaskRunRow
			if err := rows.Scan(
				&r.TaskRunId, &r.TaskId, &r.Name, &r.SubCount, &r.CreateDateTime, &r.Status, &r.UpdateDateTime, &r.RunStamp); err != nil {
				return err
			}
			trRs = append(trRs, r)
			return nil
		})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return trRs, nil
}

// getTaskRunLst return list of modeling task run body: task_run_set table rows.
func getTaskRunSetLst(dbConn *sql.DB, query string) ([]TaskRunSetRow, error) {

	var trsRs []TaskRunSetRow

	err := SelectRows(dbConn, query,
		func(rows *sql.Rows) error {
			var r TaskRunSetRow
			if err := rows.Scan(&r.TaskRunId, &r.RunId, &r.SetId, &r.TaskId); err != nil {
				return err
			}
			trsRs = append(trsRs, r)
			return nil
		})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return trsRs, nil
}

// GetTaskFull return modeling task metadata, description, notes and run history
// from db-tables: task_lst, task_txt, task_set, task_run_lst, task_run_set.
//
// If isSuccess true then return only successfully completed runs
// else return all runs: success, error, exit, progress.
// If langCode not empty then only specified language selected else all languages
func GetTaskFull(dbConn *sql.DB, taskRow *TaskRow, isSuccess bool, langCode string) (*TaskMeta, error) {

	// validate parameters
	if taskRow == nil {
		return nil, errors.New("invalid (empty) task row, it may be task not found")
	}

	// where filters
	taskWhere := " WHERE K.task_id = " + strconv.Itoa(taskRow.TaskId)
	statusFilter := ""
	if isSuccess {
		statusFilter = " AND H.status = " + ToQuoted(DoneRunStatus)
	} else {
		statusFilter = " AND H.status IN (" +
			ToQuoted(DoneRunStatus) + ", " + ToQuoted(ErrorRunStatus) + ", " + ToQuoted(ExitRunStatus) + ", " + ToQuoted(ProgressRunStatus) + ")"
	}

	var langFilter string
	if langCode != "" {
		langFilter = " AND L.lang_code = " + ToQuoted(langCode)
	}

	// task meta header: task_lst master row
	meta := &TaskMeta{TaskDef: TaskDef{Task: *taskRow}}

	// get tasks description and notes by model id
	q := "SELECT M.task_id, M.lang_id, L.lang_code, M.descr, M.note" +
		" FROM task_txt M" +
		" INNER JOIN task_lst K ON (K.task_id = M.task_id)" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		taskWhere +
		langFilter +
		" ORDER BY 1, 2"

	txtRs, err := getTaskText(dbConn, q)
	if err != nil {
		return nil, err
	}
	meta.Txt = txtRs

	// get list of set ids for the task
	setRs, err := GetTaskSetIds(dbConn, taskRow.TaskId)
	if err != nil {
		return nil, err
	}
	meta.Set = setRs

	// get task run history and status
	q = "SELECT H.task_run_id, H.task_id, H.run_name, H.sub_count, H.create_dt, H.status, H.update_dt, H.run_stamp" +
		" FROM task_run_lst H" +
		" INNER JOIN task_lst K ON (K.task_id = H.task_id)" +
		taskWhere +
		statusFilter +
		" ORDER BY 1"

	runRs, err := getTaskRunLst(dbConn, q)
	if err != nil {
		return nil, err
	}

	meta.TaskRun = make([]taskRunItem, len(runRs))
	ri := make(map[int]int, len(runRs)) // map (task run id) => index in task run array

	for k := range runRs {
		ri[runRs[k].TaskRunId] = k
		meta.TaskRun[k].TaskRunRow = runRs[k]
	}

	// select run results for the tasks
	q = "SELECT M.task_run_id, M.run_id, M.set_id, M.task_id" +
		" FROM task_run_set M" +
		" INNER JOIN task_run_lst H ON (H.task_run_id = M.task_run_id)" +
		" INNER JOIN task_lst K ON (K.task_id = H.task_id)" +
		taskWhere +
		statusFilter +
		" ORDER BY 1, 2"

	runSetRs, err := getTaskRunSetLst(dbConn, q)
	if err != nil {
		return nil, err
	}

	for k := range runSetRs {
		if i, ok := ri[runSetRs[k].TaskRunId]; ok {
			meta.TaskRun[i].TaskRunSet = append(meta.TaskRun[i].TaskRunSet, runSetRs[k])
		}
	}

	return meta, nil
}

// GetTaskFullList return list of modeling tasks metadata, description, notes and run history
// from db-tables: task_lst, task_txt, task_set, task_run_lst, task_run_set.
//
// If isSuccess true then return only successfully completed runs
// else return all runs: success, error, exit, progress.
// If langCode not empty then only specified language selected else all languages
func GetTaskFullList(dbConn *sql.DB, modelId int, isSuccess bool, langCode string) ([]TaskMeta, error) {

	// where filters
	var statusFilter string
	if isSuccess {
		statusFilter = " AND H.status = " + ToQuoted(DoneRunStatus)
	} else {
		statusFilter = " AND H.status IN (" +
			ToQuoted(DoneRunStatus) + ", " + ToQuoted(ErrorRunStatus) + ", " + ToQuoted(ExitRunStatus) + ", " + ToQuoted(ProgressRunStatus) + ")"
	}

	var langFilter string
	if langCode != "" {
		langFilter = " AND L.lang_code = " + ToQuoted(langCode)
	}

	// get list of modeling task for that model id
	smId := strconv.Itoa(modelId)

	q := "SELECT K.task_id, K.model_id, K.task_name FROM task_lst K" +
		" WHERE K.model_id =" + smId +
		" ORDER BY 1"

	taskRs, err := getTaskLst(dbConn, q)
	if err != nil {
		return nil, err
	}

	// get tasks description and notes by model id
	q = "SELECT M.task_id, M.lang_id, L.lang_code, M.descr, M.note" +
		" FROM task_txt M" +
		" INNER JOIN task_lst K ON (K.task_id = M.task_id)" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE K.model_id = " + smId +
		langFilter +
		" ORDER BY 1, 2"

	txtRs, err := getTaskText(dbConn, q)
	if err != nil {
		return nil, err
	}

	// get list of set ids for each task
	q = "SELECT M.task_id, M.set_id" +
		" FROM task_set M" +
		" INNER JOIN task_lst K ON (K.task_id = M.task_id)" +
		" WHERE K.model_id = " + smId +
		" ORDER BY 1, 2"

	setRs, err := getTaskSetLst(dbConn, q)
	if err != nil {
		return nil, err
	}

	// get task run history and status
	q = "SELECT H.task_run_id, H.task_id, H.run_name, H.sub_count, H.create_dt, H.status, H.update_dt, H.run_stamp" +
		" FROM task_run_lst H" +
		" INNER JOIN task_lst K ON (K.task_id = H.task_id)" +
		" WHERE K.model_id = " + smId +
		statusFilter +
		" ORDER BY 1"

	runRs, err := getTaskRunLst(dbConn, q)
	if err != nil {
		return nil, err
	}

	// select run results for the tasks
	q = "SELECT M.task_run_id, M.run_id, M.set_id, M.task_id" +
		" FROM task_run_set M" +
		" INNER JOIN task_run_lst H ON (H.task_run_id = M.task_run_id)" +
		" INNER JOIN task_lst K ON (K.task_id = H.task_id)" +
		" WHERE K.model_id = " + smId +
		statusFilter +
		" ORDER BY 1, 2"

	runSetRs, err := getTaskRunSetLst(dbConn, q)
	if err != nil {
		return nil, err
	}

	// convert to output result: join run pieces in struct by task id
	tl := make([]TaskMeta, len(taskRs))
	im := make(map[int]int) // map [task id] => index of task_lst row

	for k := range taskRs {
		taskId := taskRs[k].TaskId
		tl[k].Task = taskRs[k]
		tl[k].Set = setRs[taskId]
		im[taskId] = k
	}
	for k := range txtRs {
		if i, ok := im[txtRs[k].TaskId]; ok {
			tl[i].Txt = append(tl[i].Txt, txtRs[k])
		}
	}
	for k := range runRs {
		if i, ok := im[runRs[k].TaskId]; ok {
			tl[i].TaskRun = append(tl[i].TaskRun, taskRunItem{TaskRunRow: runRs[k]})
		}
	}
	for k := range runSetRs {
		// find task run id in the list af task runs for the task
		// and append task pair of (run id, set id) to that task run
		if i, ok := im[runSetRs[k].TaskId]; ok {
			for j := range tl[i].TaskRun {
				if tl[i].TaskRun[j].TaskRunRow.TaskRunId == runSetRs[k].TaskRunId {
					tl[i].TaskRun[j].TaskRunSet = append(tl[i].TaskRun[j].TaskRunSet, runSetRs[k])
					break
				}
			}
		}
	}

	return tl, nil
}
