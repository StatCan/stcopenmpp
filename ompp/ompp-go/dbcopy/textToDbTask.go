// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// copy modeling task metadata and run history from json files into database
func textToDbTask(modelName string, modelDigest string, runOpts *config.RunOptions) error {

	// validate parameters
	if modelName == "" {
		return helper.ErrorNew("invalid (empty) model name")
	}

	// get modeling task name and id
	taskName := runOpts.String(taskNameArgKey)
	taskId := runOpts.Int(taskIdArgKey, 0)

	if taskId < 0 || taskId == 0 && taskName == "" {
		return helper.ErrorFmt("dbcopy invalid argument(s) for modeling task id: %s and/or name: %s", runOpts.String(taskIdArgKey), runOpts.String(taskNameArgKey))
	}

	// deirectory for task metadata: it is input directory/modelName
	inpDir := ""
	if taskId > 0 {
		inpDir = filepath.Join(runOpts.String(inputDirArgKey), modelName+".task."+strconv.Itoa(taskId))
	} else {
		inpDir = filepath.Join(runOpts.String(inputDirArgKey), modelName+".task."+taskName)
	}

	// get model task metadata json path by task id or task name or both
	var metaPath string

	if runOpts.IsExist(taskNameArgKey) && runOpts.IsExist(taskIdArgKey) { // both: task id and name

		metaPath = filepath.Join(inpDir,
			modelName+".task."+strconv.Itoa(taskId)+"."+helper.CleanFileName(taskName)+".json")

	} else { // task id or task name only

		// make path search patterns for metadata json file
		var mp string
		if runOpts.IsExist(taskNameArgKey) && !runOpts.IsExist(taskIdArgKey) { // task name only
			mp = modelName + ".task.*" + helper.CleanFileName(taskName) + ".json"
		}
		if !runOpts.IsExist(taskNameArgKey) && runOpts.IsExist(taskIdArgKey) { // task id only
			mp = modelName + ".task." + strconv.Itoa(taskId) + ".*.json"
		}

		// find path to metadata json by pattern
		fl, err := filepath.Glob(inpDir + "/" + mp)
		if err != nil {
			return err
		}
		if len(fl) <= 0 {
			return helper.ErrorNew("no metadata json file found for modeling task:", taskId, taskName)
		}
		if len(fl) > 1 {
			omppLog.Log("found multiple modeling task metadata json files, using:", filepath.Base(metaPath))
		}
		metaPath = fl[0]
	}

	// check results: metadata json file must exist
	if metaPath == "" {
		return helper.ErrorNew("no metadata json file found for modeling task:", taskId, taskName)
	}
	if _, err := os.Stat(metaPath); err != nil {
		return helper.ErrorNew("no metadata json file found for modeling task:", taskId, taskName)
	}

	// open source database connection and check is it valid
	dn := runOpts.String(toDbDriverArgKey)
	if dn == "" && runOpts.IsExist(dbDriverArgKey) {
		dn = runOpts.String(dbDriverArgKey)
	}
	cs, dn := db.IfEmptyMakeDefault(modelName, runOpts.String(toSqliteArgKey), runOpts.String(toDbConnStrArgKey), dn)

	dstDb, dbFacet, err := db.Open(cs, dn, true)
	if err != nil {
		return err
	}
	defer dstDb.Close()

	if err := db.CheckOpenmppSchemaVersion(dstDb); err != nil {
		return err
	}

	// get model metadata
	modelDef, err := db.GetModel(dstDb, modelName, modelDigest)
	if err != nil {
		return err
	}

	// get full list of languages
	langDef, err := db.GetLanguages(dstDb)
	if err != nil {
		return err
	}

	// read task metadata from json
	var pub db.TaskPub
	isExist, err := helper.FromJsonFile(metaPath, &pub)
	if err != nil {
		return err
	}
	if !isExist {
		return helper.ErrorNew("modeling task not found or empty:", taskId, taskName)
	}

	// task name: use task name from json metadata, if empty
	if pub.Name != "" && taskName != pub.Name {
		taskName = pub.Name
	}

	// restore model runs from json and/or csv files and insert it into database
	var runLst []string
	var isRunNotFound, isRunNotCompleted bool

	for j := range pub.TaskRun {
	nextRun:
		for k := range pub.TaskRun[j].TaskRunSet {

			// check is this run id already processed
			runDigest := pub.TaskRun[j].TaskRunSet[k].Run.RunDigest
			for i := range runLst {
				if runDigest == runLst[i] {
					continue nextRun
				}
			}
			runLst = append(runLst, runDigest)

			// run name must not be empty in order to find run json metadata and csv files
			runName := pub.TaskRun[j].TaskRunSet[k].Run.Name
			if runName == "" {
				isRunNotFound = true // skip: run name empty
				continue
			}

			// run must be completed: status success, error or exit
			if !db.IsRunCompleted(pub.TaskRun[j].TaskRunSet[k].Run.Status) {
				isRunNotCompleted = true
				continue // skip: run not completed
			}

			// make path search patterns for metadata json and csv directory
			//cp := "run.*" + helper.CleanFileName(runName)
			mp := modelName + ".run.*" + helper.CleanFileName(runName) + ".json"
			var jsonPath, csvDir string

			// find path to metadata json by pattern
			fl, err := filepath.Glob(inpDir + "/" + mp)
			if err != nil {
				return err
			}
			if len(fl) <= 0 {
				isRunNotFound = true // skip: no run metadata
				continue
			}
			jsonPath = fl[0]
			if len(fl) > 1 {
				omppLog.Log("found multiple model run metadata json files, using:", filepath.Base(jsonPath))
			}

			// csv directory: check if csv directory exist for that json file
			d, f := filepath.Split(jsonPath)
			c := strings.TrimSuffix(strings.TrimPrefix(f, modelName+"."), ".json")

			if len(c) <= 4 { // expected csv directory: run.4.r or run.r
				csvDir = ""
			} else {
				csvDir = filepath.Join(d, c)
				if _, err := os.Stat(csvDir); err != nil {
					csvDir = ""
				}
			}

			// check results: metadata json file or csv directory must exist
			if jsonPath == "" || csvDir == "" {
				isRunNotFound = true // skip: no run metadata json file or csv directory
				continue
			}
			if _, err := os.Stat(jsonPath); err != nil {
				isRunNotFound = true // skip: no run metadata json file
				continue
			}
			if _, err := os.Stat(csvDir); err != nil {
				isRunNotFound = true // skip: no run csv directory
				continue
			}

			// read from metadata json and csv files and update target database
			dstId, err := fromRunTextToDb(dstDb, dbFacet, modelDef, langDef, runName, jsonPath)
			if err != nil {
				return err
			}
			if dstId <= 0 {
				isRunNotFound = true // run json file empty
			}
		}
	}

	// restore workset by set name from json and/or csv files and insert it into database
	var wsLst []string
	isSetNotFound := false

	var fws = func(dbConn *sql.DB, setName string) error {

		// check is workset already processed
		for i := range wsLst {
			if setName == wsLst[i] {
				return nil
			}
		}
		wsLst = append(wsLst, setName)

		// make path search patterns for metadata json and csv directory
		cp := "set.*" + helper.CleanFileName(setName)
		mp := modelName + "." + cp + ".json"
		var jsonPath, csvDir string

		// find path to metadata json by pattern
		fl, err := filepath.Glob(inpDir + "/" + mp)
		if err != nil {
			return err
		}
		if len(fl) >= 1 { // set name is unique per model, it is expected to be only one file
			jsonPath = fl[0]
			if len(fl) > 1 {
				omppLog.Log("found multiple workset metadata json files, using:", filepath.Base(jsonPath))
			}
		}

		// csv directory:
		// if metadata json file exist then check if csv directory for that json file
		if jsonPath != "" {

			d, f := filepath.Split(jsonPath)
			c := strings.TrimSuffix(strings.TrimPrefix(f, modelName+"."), ".json")

			if len(c) <= 4 { // expected csv directory: set.4.w or set.w
				csvDir = ""
			} else {
				csvDir = filepath.Join(d, c)
				if _, err := os.Stat(csvDir); err != nil {
					csvDir = ""
				}
			}

		} else { // metadata json file not exist: search for csv directory by pattern

			fl, err := filepath.Glob(inpDir + "/" + cp)
			if err != nil {
				return err
			}
			if len(fl) >= 1 {
				csvDir = fl[0]
				if len(fl) > 1 {
					omppLog.Log("found multiple workset csv directories, using:", filepath.Base(csvDir))
				}
			}
		}

		// check results: metadata json file or csv directory must exist
		if jsonPath == "" && csvDir == "" {
			isSetNotFound = true // exit: no workset json and no csv directory exists
			return nil
		}

		// write workset metadata into json and parameter values into csv files
		dstId, err := fromWorksetTextToDb(dbConn, modelDef, langDef, setName, "", jsonPath, csvDir)
		if err != nil {
			return err
		}
		if dstId <= 0 {
			isSetNotFound = true // workset empty: json empty and csv directory empty
		}
		return nil
	}

	// restore task body worksets
	for k := range pub.Set {
		if err = fws(dstDb, pub.Set[k]); err != nil {
			return err
		}
	}

	// restore worksets from model run history
	for j := range pub.TaskRun {
		for k := range pub.TaskRun[j].TaskRunSet {
			if err = fws(dstDb, pub.TaskRun[j].TaskRunSet[k].SetName); err != nil {
				return err
			}
		}
	}

	// display warnings if any workset not found (files and csv directories not found)
	// display warnings if any model runs not found or not completed
	if isSetNotFound {
		omppLog.LogFmt("Warning: task %s workset(s) not found, copy of task incomplete", pub.Name)
	}
	if isRunNotFound {
		omppLog.LogFmt("Warning: task %s model run(s) not found, copy of task run history incomplete", pub.Name)
	}
	if isRunNotCompleted {
		omppLog.LogFmt("Warning: task %s model run(s) not completed, copy of task run history incomplete", pub.Name)
	}

	// rename destination task
	srcTaskName := pub.Name
	if runOpts.IsExist(taskNewNameArgKey) {
		pub.Name = runOpts.String(taskNewNameArgKey)
	}

	// insert or update modeling task and task run history into database
	dstId, err := fromTaskJsonToDb(dstDb, modelDef, langDef, &pub)
	if err != nil {
		return err
	}
	omppLog.LogFmt("Modeling task from %s into: %d %s", srcTaskName, dstId, pub.Name)

	return nil
}

// fromTaskListJsonToDb reads modeling tasks and tasks run history from json file and insert it into database.
// it does update task id, set id's and run id's with actual id in destination database
func fromTaskListJsonToDb(dbConn *sql.DB, modelDef *db.ModelMeta, langDef *db.LangMeta, inpDir string) error {

	// get list of task json files
	fl, err := filepath.Glob(inpDir + "/" + modelDef.Model.Name + ".task.*.json")
	if err != nil {
		return err
	}
	if len(fl) <= 0 {
		return nil // no modeling tasks
	}

	// for each file: read task metadata, update task in target database
	for k := range fl {

		// read task metadata from json
		var pub db.TaskPub
		isExist, err := helper.FromJsonFile(fl[k], &pub)
		if err != nil {
			return err
		}
		if !isExist {
			continue // skip: no modeling task, file not exist or empty
		}

		// insert or update modeling task and task run history into database
		dstId, err := fromTaskJsonToDb(dbConn, modelDef, langDef, &pub)
		if err != nil {
			return err
		}
		omppLog.LogFmt("Modeling task from %s into id: %d", pub.Name, dstId)
	}

	return nil
}

// fromTaskTextToDb insert or update modeling task and task run history into database.
// it does update task id with actual id in destination database and return it
func fromTaskJsonToDb(
	dbConn *sql.DB, modelDef *db.ModelMeta, langDef *db.LangMeta, pubMeta *db.TaskPub) (int, error) {

	// convert from "public" format into destination db rows
	meta, isSetNotFound, isTaskRunNotFound, err := pubMeta.FromPublic(dbConn, modelDef, true)
	if err != nil {
		return 0, err
	}
	if isSetNotFound {
		omppLog.LogFmt("Warning: task %s workset(s) not found, copy of task incomplete", meta.Task.Name)
	}
	if isTaskRunNotFound {
		omppLog.LogFmt("Warning: task %s worksets or model runs not found, copy of task run history incomplete", meta.Task.Name)
	}

	// save modeling task metadata
	err = meta.UpdateTaskFull(dbConn, modelDef, langDef)
	if err != nil {
		return 0, err
	}
	return meta.Task.TaskId, nil
}
