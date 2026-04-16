// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"strconv"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
)

// toTaskListCsv writes all successfully completed tasks and tasks run history into csv files.
func toTaskListCsv(dbConn *sql.DB, modelId int, outDir string) error {

	// get all modeling tasks and successfully completed tasks run history
	tl, err := db.GetTaskFullList(dbConn, modelId, true, "")
	if err != nil {
		return err
	}

	// write modeling task rows into csv
	row := make([]string, 3)

	idx := 0
	err = toCsvFile(
		outDir,
		"task_lst.csv",
		[]string{"task_id", "model_id", "task_name"},
		func() (bool, []string, error) {
			if 0 <= idx && idx < len(tl) {
				row[0] = strconv.Itoa(tl[idx].Task.TaskId)
				row[1] = strconv.Itoa(tl[idx].Task.ModelId)
				row[2] = tl[idx].Task.Name
				idx++
				return false, row, nil
			}
			return true, row, nil // end of task rows
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "task_lst.csv", err)
	}

	// write task text rows into csv
	row = make([]string, 4)

	idx = 0
	j := 0
	err = toCsvFile(
		outDir,
		"task_txt.csv",
		[]string{"task_id", "lang_code", "descr", "note"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(tl) { // end of task rows
				return true, row, nil
			}

			// if end of current task texts then find next task with text rows
			if j < 0 || j >= len(tl[idx].Txt) {
				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(tl) { // end of task rows
						return true, row, nil
					}
					if len(tl[idx].Txt) > 0 {
						break
					}
				}
			}

			// make task text []string row
			row[0] = strconv.Itoa(tl[idx].Txt[j].TaskId)
			row[1] = tl[idx].Txt[j].LangCode
			row[2] = tl[idx].Txt[j].Descr

			if tl[idx].Txt[j].Note == "" { // empty "" string is NULL
				row[3] = "NULL"
			} else {
				row[3] = tl[idx].Txt[j].Note
			}
			j++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "task_txt.csv", err)
	}

	// write task body (task sets) rows into csv
	row = make([]string, 2)

	idx = 0
	j = 0
	err = toCsvFile(
		outDir,
		"task_set.csv",
		[]string{"task_id", "set_id"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(tl) { // end of task rows
				return true, row, nil
			}

			// if end of current task sets then find next task with set rows
			if j < 0 || j >= len(tl[idx].Set) {
				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(tl) { // end of task rows
						return true, row, nil
					}
					if len(tl[idx].Set) > 0 {
						break
					}
				}
			}

			// make task set []string row
			row[0] = strconv.Itoa(tl[idx].Task.TaskId)
			row[1] = strconv.Itoa(tl[idx].Set[j])
			j++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "task_set.csv", err)
	}

	// write task run history rows into csv
	row = make([]string, 8)

	idx = 0
	j = 0
	err = toCsvFile(
		outDir,
		"task_run_lst.csv",
		[]string{"task_run_id", "task_id", "run_name", "sub_count", "create_dt", "status", "update_dt", "run_stamp"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(tl) { // end of task rows
				return true, row, nil
			}

			// if end of current task run history then find next task with run history rows
			if j < 0 || j >= len(tl[idx].TaskRun) {
				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(tl) { // end of task rows
						return true, row, nil
					}
					if len(tl[idx].TaskRun) > 0 {
						break
					}
				}
			}

			// make task run history []string row
			row[0] = strconv.Itoa(tl[idx].TaskRun[j].TaskRunId)
			row[1] = strconv.Itoa(tl[idx].TaskRun[j].TaskId)
			row[2] = tl[idx].TaskRun[j].Name
			row[3] = strconv.Itoa(tl[idx].TaskRun[j].SubCount)
			row[4] = tl[idx].TaskRun[j].CreateDateTime
			row[5] = tl[idx].TaskRun[j].Status
			row[6] = tl[idx].TaskRun[j].UpdateDateTime
			row[7] = tl[idx].TaskRun[j].RunStamp
			j++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "task_run_lst.csv", err)
	}

	// write task run history body (set id, run id pair) rows into csv
	row = make([]string, 4)

	idx = 0
	tri := 0
	j = 0
	err = toCsvFile(
		outDir,
		"task_run_set.csv",
		[]string{"task_run_id", "run_id", "set_id", "task_id"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(tl) { // end of task rows
				return true, row, nil
			}

			// if end of current task run history body then find next task with run history body rows
			if tri < 0 || tri >= len(tl[idx].TaskRun) || j < 0 || j >= len(tl[idx].TaskRun[tri].TaskRunSet) {

				j = 0
				for {
					if 0 <= tri && tri < len(tl[idx].TaskRun) {
						tri++
					}
					if tri < 0 || tri >= len(tl[idx].TaskRun) {
						idx++
						tri = 0
					}
					if idx < 0 || idx >= len(tl) { // end of task rows
						return true, row, nil
					}
					if tri >= len(tl[idx].TaskRun) { // end of run history rows for that task
						continue
					}
					if len(tl[idx].TaskRun[tri].TaskRunSet) > 0 {
						break
					}
				}
			}

			// make task run history body []string row
			row[0] = strconv.Itoa(tl[idx].TaskRun[tri].TaskRunSet[j].TaskRunId)
			row[1] = strconv.Itoa(tl[idx].TaskRun[tri].TaskRunSet[j].RunId)
			row[2] = strconv.Itoa(tl[idx].TaskRun[tri].TaskRunSet[j].SetId)
			row[3] = strconv.Itoa(tl[idx].TaskRun[tri].TaskRunSet[j].TaskId)
			j++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "task_run_set.csv", err)
	}

	return nil
}
