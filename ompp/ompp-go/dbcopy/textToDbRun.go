// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// copy model run from text json and csv files into database
func textToDbRun(modelName string, modelDigest string, runOpts *config.RunOptions) error {

	// validate parameters
	if modelName == "" {
		return helper.ErrorNew("invalid (empty) model name")
	}

	// get model run id and/or digest and/or name
	runId := runOpts.Int(runIdArgKey, 0)
	runDigest := runOpts.String(runDigestArgKey)
	runName := runOpts.String(runNameArgKey)
	if runId < 0 || runId == 0 && runDigest == "" && runName == "" {
		return helper.ErrorNew("dbcopy invalid argument(s) for model run:", runOpts.String(runIdArgKey), runOpts.String(runNameArgKey), runOpts.String(runDigestArgKey))
	}

	// root directory of run data is input directory or name of input.zip, result is one of:
	// input/modelName.run.id
	// input/modelName.run.runName
	// input/modelName.run.runDigest
	// for csv files this "root" combined subdirectory: root/run.id.runName or root/run.runName
	inpDir := ""
	switch {
	case runId > 0:
		inpDir = filepath.Join(runOpts.String(inputDirArgKey), modelName+".run."+strconv.Itoa(runId))
	case runDigest != "":
		inpDir = filepath.Join(runOpts.String(inputDirArgKey), modelName+".run."+helper.CleanFileName(runDigest))
	default:
		// if not run id and not digest then run name
		inpDir = filepath.Join(runOpts.String(inputDirArgKey), modelName+".run."+helper.CleanFileName(runName))
	}

	// unzip if required and use unzipped directory as "root" input directory
	if runOpts.Bool(zipArgKey) {
		base := filepath.Base(inpDir)
		omppLog.Log("Unpack", base+".zip")

		outDir := runOpts.String(outputDirArgKey)
		if outDir == "" {
			outDir = inpDir
		}
		if err := helper.UnpackZip(inpDir+".zip", !theCfg.isKeepOutputDir, outDir); err != nil {
			return err
		}
		inpDir = filepath.Join(outDir, base)
	}

	// get model run metadata json path and csv directory by run id or run name or both
	var metaPath string

	if runOpts.IsExist(runNameArgKey) && runOpts.IsExist(runIdArgKey) { // both: run id and name

		metaPath = filepath.Join(inpDir,
			modelName+".run."+strconv.Itoa(runId)+"."+helper.CleanFileName(runName)+".json")

	} else { // only run id or run name and/or run digest

		// make path search patterns for metadata json and csv directory
		var mp string
		switch {
		case runOpts.IsExist(runNameArgKey) && !runOpts.IsExist(runIdArgKey): // run name and not run id
			mp = modelName + ".run.*" + helper.CleanFileName(runName) + ".json"
		case !runOpts.IsExist(runNameArgKey) && runOpts.IsExist(runIdArgKey): // run id and not run name
			mp = modelName + ".run." + strconv.Itoa(runId) + ".*.json"
		default:
			// run digest and no run name or run id
			mp = modelName + ".run.*.json"
		}

		// find path to metadata json by pattern
		fl, err := filepath.Glob(inpDir + "/" + mp)
		if err != nil {
			return err
		}
		if len(fl) <= 0 {
			return helper.ErrorNew("no metadata json file found for model run:", runId, runName, runDigest)
		}
		metaPath = fl[0]
		if len(fl) > 1 {
			omppLog.Log("found multiple model run metadata json files, using:", filepath.Base(metaPath))
		}
	}

	// check results: metadata json file or csv directory must exist
	if metaPath == "" {
		return helper.ErrorNew("no metadata json file found for model run:", runId, runName, runDigest)
	}
	if _, err := os.Stat(metaPath); err != nil {
		return helper.ErrorNew("no metadata json file found for model run:", runId, runName, runDigest)
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

	// read from metadata json and csv files and update target database
	dstId, err := fromRunTextToDb(dstDb, dbFacet, modelDef, langDef, runName, metaPath)
	if err != nil {
		return err
	}
	if dstId <= 0 {
		return helper.ErrorNew("model run not found or empty:", runId, runName, runDigest)
	}

	return nil
}

// fromRunTextListToDb read all model runs (metadata, parameters, output tables)
// from csv and json files, convert it to db cells and insert into database.
// Double format is used for float model types digest calculation, if non-empty format supplied
func fromRunTextListToDb(
	dbConn *sql.DB,
	dbFacet db.Facet,
	modelDef *db.ModelMeta,
	langDef *db.LangMeta,
	inpDir string,

) error {

	// get list of model run json files
	fl, err := filepath.Glob(inpDir + "/" + modelDef.Model.Name + ".run.*.json")
	if err != nil {
		return err
	}
	if len(fl) <= 0 {
		return nil // no model runs
	}

	// for each file:
	// read model run metadata, update model in target database
	// read csv files from run csv subdir, update run parameters values and output tables values
	// update model run digest
	for k := range fl {

		_, err := fromRunTextToDb(dbConn, dbFacet, modelDef, langDef, "", fl[k])
		if err != nil {
			return err
		}
	}

	return nil
}

// fromRunTextToDb read model run metadata from json file,
// read from csv files parameter values and output tables values,
// convert it to db cells and insert into database,
// and finally update model run digest.
// Double format is used for float model types digest calculation, if non-empty format supplied
// it return source run id (run id from metadata json file) and destination run id
func fromRunTextToDb(
	dbConn *sql.DB,
	dbFacet db.Facet,
	modelDef *db.ModelMeta,
	langDef *db.LangMeta,
	srcName string,
	metaPath string,
) (int, error) {

	// if no metadata file then exit: nothing to do
	if metaPath == "" {
		return 0, nil // no model run metadata
	}

	// get model run metadata
	// model name and set name must be specified as parameter or inside of metadata json
	var pub db.RunPub
	isExist, err := helper.FromJsonFile(metaPath, &pub)
	if err != nil {
		return 0, err
	}
	if !isExist {
		return 0, nil // no model run
	}

	// check if run subdir exist
	d, f := filepath.Split(metaPath)
	c := strings.TrimSuffix(strings.TrimPrefix(f, pub.ModelName+"."), ".json")
	pDir := filepath.Join(c, "parameters")
	tDir := filepath.Join(c, "output-tables")
	mDir := filepath.Join(c, "microdata")
	nMd := len(pub.Entity)

	paramCsvDir := filepath.Join(d, pDir)
	if _, err := os.Stat(paramCsvDir); err != nil {
		return 0, helper.ErrorNew("csv parameters directory not found:" + pDir)
	}
	tableCsvDir := filepath.Join(d, tDir)
	if _, err := os.Stat(tableCsvDir); err != nil {
		return 0, helper.ErrorNew("csv output tables directory not found:" + tDir)
	}
	microCsvDir := filepath.Join(d, mDir)
	if nMd > 0 {
		if _, err := os.Stat(microCsvDir); err != nil {
			return 0, helper.ErrorNew("csv microdata directory not found:" + mDir)
		}
	}

	// run name: use run name from json metadata if json metadata not empty, else use supplied run name
	if pub.Name != "" && srcName != pub.Name {
		srcName = pub.Name
	}

	if theCfg.isNoDigestCheck {
		pub.ModelDigest = "" // model digest validation disabled
	}

	// destination: convert from "public" format into destination db rows
	meta, err := pub.FromPublic(modelDef)
	if err != nil {
		return 0, err
	}

	// save model run
	isExist, err = meta.UpdateRun(dbConn, modelDef, langDef, theCfg.doubleFmt)
	if err != nil {
		return 0, err
	}
	dstId := meta.Run.RunId
	if isExist { // exit if model run already exist
		omppLog.LogFmt("Model run %s already exists as %d", srcName, dstId)
		return dstId, nil
	}

	omppLog.LogFmt("Model run from %s into id: %d", srcName, dstId)

	// restore run parameters: all model parameters must be included in the run
	nP := len(modelDef.Param)
	omppLog.Log("  Parameters:", nP)
	logT := time.Now().Unix()

	for j := range modelDef.Param {

		// read parameter values from csv file
		logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s", j, nP, modelDef.Param[j].Name))

		// insert parameter values in model run
		paramLt := db.WriteParamLayout{
			WriteLayout: db.WriteLayout{
				Name: modelDef.Param[j].Name,
				ToId: dstId,
			},
			SubCount:  meta.Param[j].SubCount,
			DoubleFmt: theCfg.doubleFmt,
			IsToRun:   true,
		}
		cvtParam := db.CellParamConverter{
			ModelDef:  modelDef,
			Name:      modelDef.Param[j].Name,
			IsIdCsv:   false,
			DoubleFmt: theCfg.doubleFmt,
		}

		err = writeParamFromCsvFile(dbConn, modelDef, paramLt, paramCsvDir, cvtParam)
		if err != nil {
			omppLog.Log("Error at:", paramLt.Name, ":", err)
			omppLog.Log("Cleanup on error: delete model run", srcName, dstId)

			// delete model run on error to rollback results of UpdateRun() call above
			e := db.DeleteRun(dbConn, dstId)
			if e != nil {
				omppLog.LogFmt("Failed to delete model run %s id: %d: %v", srcName, dstId, e)
			}
			return 0, err // return original error
		}
	}

	// restore run output tables accumulators and expressions, if the table included in run results
	nT := len(modelDef.Table)
	omppLog.Log("  Tables:", nT)

	for j := range modelDef.Table {

		// check if table exist in model run results
		var isFound bool
		for k := range meta.Table {
			isFound = meta.Table[k].TableHid == modelDef.Table[j].TableHid
			if isFound {
				break
			}
		}
		if !isFound {
			continue // skip table: it is suppressed and not in run results
		}

		// read output table accumulator(s) values from csv file
		tblLt := db.WriteTableLayout{
			WriteLayout: db.WriteLayout{
				Name: modelDef.Table[j].Name,
				ToId: dstId,
			},
			SubCount:  meta.Run.SubCount,
			DoubleFmt: theCfg.doubleFmt,
		}
		ctc := db.CellTableConverter{
			ModelDef:  modelDef,
			Name:      modelDef.Table[j].Name,
			IsIdCsv:   false,
			DoubleFmt: theCfg.doubleFmt,
		}
		cvtExpr := db.CellExprConverter{CellTableConverter: ctc}
		cvtAcc := db.CellAccConverter{CellTableConverter: ctc}

		logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s", j, nT, tblLt.Name))

		err := writeTableFromCsvFiles(dbConn, modelDef, tblLt, tableCsvDir, cvtExpr, cvtAcc)
		if err != nil {
			if err != nil {
				omppLog.Log("Error at:", tblLt.Name, ":", err)
				omppLog.Log("Cleanup on error: delete model run", srcName, dstId)

				// delete model run on error to rollback results of UpdateRun() call above
				e := db.DeleteRun(dbConn, dstId)
				if e != nil {
					omppLog.LogFmt("Failed to delete model run %s id: %d: %v", srcName, dstId, e)
				}
				return 0, err // return original error
			}
		}
	}

	// update model run digest
	if meta.Run.ValueDigest == "" {

		svd, err := db.UpdateRunValueDigest(dbConn, dstId)
		if err != nil {
			return 0, err
		}
		meta.Run.ValueDigest = svd
	}

	// read entity microdata values from csv file
	if nMd > 0 {

		omppLog.Log("  Microdata:", nMd)

		for j := 0; j < nMd; j++ {

			// read microdata values from csv file
			microLt := db.WriteMicroLayout{
				WriteLayout: db.WriteLayout{
					Name: pub.Entity[j].Name,
					ToId: dstId,
				},
				DoubleFmt: theCfg.doubleFmt,
			}
			cvtMicro := db.CellMicroConverter{CellEntityConverter: db.CellEntityConverter{
				ModelDef:  modelDef,
				Name:      pub.Entity[j].Name,
				EntityGen: &meta.EntityGen[j], // entity generation converted from "public" and has the same item order
				IsIdCsv:   false,
				DoubleFmt: theCfg.doubleFmt,
			}}

			logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s", j, nMd, microLt.Name))

			err := writeMicroFromCsvFile(dbConn, dbFacet, modelDef, meta, microLt, microCsvDir, cvtMicro)
			if err != nil {
				omppLog.Log("Error at:", pub.Entity[j].Name, ":", err)
				omppLog.Log("Cleanup on error: delete model run", srcName, dstId)

				// delete model run on error to rollback results of UpdateRun() call above
				e := db.DeleteRun(dbConn, dstId)
				if e != nil {
					omppLog.Log("Failed to delete model run %s id: %d: %v", srcName, dstId, e)
				}
				return 0, err // return original error
			}
		}
	}

	return dstId, nil
}
