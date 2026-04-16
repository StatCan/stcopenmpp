// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// copy workset from database into text json and csv files
func dbToTextWorkset(modelName string, modelDigest string, runOpts *config.RunOptions) error {

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

	// "root" directory for workset metadata
	// later this "root" combined with modelName.set.name or modelName.set.id
	outDir := ""
	if runOpts.IsExist(paramDirArgKey) {
		outDir = filepath.Clean(runOpts.String(paramDirArgKey))
	} else {
		if setId > 0 {
			outDir = filepath.Join(runOpts.String(outputDirArgKey), modelName+".set."+strconv.Itoa(setId))
		} else {
			outDir = filepath.Join(runOpts.String(outputDirArgKey), modelName+".set."+setName)
		}
	}
	fileCreated := make(map[string]bool)

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
		if wsRow, err = db.GetWorksetByName(srcDb, modelDef.Model.ModelId, setName); err != nil {
			return err
		}
		if wsRow == nil {
			return helper.ErrorNew("Workset not found:", setName)
		}
	}

	wm, err := db.GetWorksetFull(srcDb, wsRow, "") // get full workset metadata
	if err != nil {
		return err
	}

	// check: workset must be readonly
	if !wm.Set.IsReadonly {
		return helper.ErrorNew("workset must be readonly:", wsRow.SetId, wsRow.Name)
	}

	// create new output directory for workset metadata
	if !theCfg.isKeepOutputDir {
		if ok := dirDeleteAndLog(outDir); !ok {
			return helper.ErrorNew("Error: unable to delete:", outDir)
		}
	}
	if err = os.MkdirAll(outDir, 0750); err != nil {
		return err
	}

	// for single run or single workset output to text
	// do not use of run and set id's in directory names by default
	// only use id's in directory names if:
	// dbcopy option use id name = true or user specified run id or workset id
	isUseIdNames := runOpts.Bool(useIdNamesArgKey)

	// write workset metadata into json and parameter values into csv files
	if err = toWorksetText(srcDb, modelDef, wm, outDir, fileCreated, isUseIdNames); err != nil {
		return err
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

// toWorksetListText write all readonly worksets into csv files, each set in separate subdirectory
func toWorksetListText(
	dbConn *sql.DB,
	modelDef *db.ModelMeta,
	outDir string,
	fileCreated map[string]bool,
	isUseIdNames bool) error {

	// get all readonly worksets
	wl, err := db.GetWorksetFullList(dbConn, modelDef.Model.ModelId, true, "")
	if err != nil {
		return err
	}

	// read all workset parameters and dump it into csv files
	for k := range wl {
		err = toWorksetText(dbConn, modelDef, &wl[k], outDir, fileCreated, isUseIdNames)
		if err != nil {
			return err
		}
	}
	return nil
}

// toWorksetText write workset metadata into json file
// and parameters into csv files, in separate subdirectory
// by default file name and directory name include set id: modelName.set.1234.SetName
// user can explicitly disable it by IdNames=false
func toWorksetText(
	dbConn *sql.DB,
	modelDef *db.ModelMeta,
	meta *db.WorksetMeta,
	outDir string,
	fileCreated map[string]bool,
	isUseIdNames bool) error {

	// convert db rows into "public" format
	setId := meta.Set.SetId
	omppLog.Log("Workset", setId, meta.Set.Name)

	pub, err := meta.ToPublic(dbConn, modelDef)
	if err != nil {
		return err
	}

	// create workset subdir under output dir
	var csvName string
	if !isUseIdNames {
		csvName = "set." + helper.CleanFileName(pub.Name)
	} else {
		csvName = "set." + strconv.Itoa(setId) + "." + helper.CleanFileName(pub.Name)
	}
	csvDir := filepath.Join(outDir, csvName)

	if !theCfg.isKeepOutputDir {
		if ok := dirDeleteAndLog(csvDir); !ok {
			return helper.ErrorNew("Error: unable to delete:", csvDir)
		}
	}
	if err = os.MkdirAll(csvDir, 0750); err != nil {
		return err
	}

	// write all parameters into csv files
	nP := len(pub.Param)
	omppLog.Log("  Parameters:", nP)
	logT := time.Now().Unix()

	for j := 0; j < nP; j++ {

		cvtParam := &db.CellParamConverter{
			ModelDef:  modelDef,
			Name:      pub.Param[j].Name,
			IsIdCsv:   theCfg.isIdCsv,
			DoubleFmt: theCfg.doubleFmt,
		}
		paramLt := db.ReadParamLayout{
			ReadLayout: db.ReadLayout{
				Name:   pub.Param[j].Name,
				FromId: setId,
			},
			IsFromSet: true,
		}

		logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s", j, nP, paramLt.Name))

		err = toCellCsvFile(dbConn, modelDef, paramLt, cvtParam, fileCreated, csvDir, "", "")
		if err != nil {
			return err
		}
	}

	// save model workset metadata into json
	if err := helper.ToJsonFile(filepath.Join(outDir, modelDef.Model.Name+"."+csvName+".json"), pub); err != nil {
		return err
	}
	return nil
}
