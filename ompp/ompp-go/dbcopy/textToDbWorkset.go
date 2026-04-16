// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"errors"
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

// copy workset from text json and csv files into database
func textToDbWorkset(modelName string, modelDigest string, runOpts *config.RunOptions) error {

	// validate parameters
	if modelName == "" {
		return errors.New("invalid (empty) model name")
	}

	// get workset name and id
	setName := runOpts.String(setNameArgKey)
	setId := runOpts.Int(setIdArgKey, 0)

	if setId < 0 || setId == 0 && setName == "" {
		return helper.ErrorFmt("dbcopy invalid argument(s) for set id: %s and/or set name: %s", runOpts.String(setIdArgKey), runOpts.String(setNameArgKey))
	}

	// root for workset data: input directory or name of input.zip
	// it is parameter directory (if specified) or input directory/modelName.set.id
	// for csv files this "root" combined with sub-directory: root/set.id.setName or root/set.setName
	inpDir := ""
	if runOpts.IsExist(paramDirArgKey) {
		inpDir = filepath.Clean(runOpts.String(paramDirArgKey))
	} else {
		if setId > 0 {
			inpDir = filepath.Join(runOpts.String(inputDirArgKey), modelName+".set."+strconv.Itoa(setId))
		} else {
			inpDir = filepath.Join(runOpts.String(inputDirArgKey), modelName+".set."+setName)
		}
	}

	// unzip if required and use unzipped directory as "root" input diretory
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

	// get workset metadata json path and csv directory by set id or set name or both
	var metaPath string
	var csvDir string

	if runOpts.IsExist(setNameArgKey) && runOpts.IsExist(setIdArgKey) { // both: set id and name

		metaPath = filepath.Join(inpDir,
			modelName+".set."+strconv.Itoa(setId)+"."+helper.CleanFileName(setName)+".json")

		if _, err := os.Stat(metaPath); err != nil { // clear path to indicate metadata json file does not exist
			metaPath = ""
		}

		csvDir = filepath.Join(inpDir,
			"set."+strconv.Itoa(setId)+"."+helper.CleanFileName(setName))

		if _, err := os.Stat(csvDir); err != nil { // clear path to indicate csv directory does not exist
			csvDir = ""
		}

	} else { // set id or set name only

		// make path search patterns for metadata json and csv directory
		var cp string
		if runOpts.IsExist(setNameArgKey) && !runOpts.IsExist(setIdArgKey) { // set name only
			cp = "set.*" + helper.CleanFileName(setName)
		}
		if !runOpts.IsExist(setNameArgKey) && runOpts.IsExist(setIdArgKey) { // set id only
			cp = "set." + strconv.Itoa(setId) + ".*"
		}
		mp := modelName + "." + cp + ".json"

		// find path to metadata json by pattern
		fl, err := filepath.Glob(inpDir + "/" + mp)
		if err != nil {
			return err
		}
		if len(fl) >= 1 {
			metaPath = fl[0]
			if len(fl) > 1 {
				omppLog.Log("found multiple workset metadata json files, using:", filepath.Base(metaPath))
			}
		}

		// csv directory:
		// if metadata json file exist then check if csv directory for that json file
		if metaPath != "" {

			d, f := filepath.Split(metaPath)
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
	}

	// check results: metadata json file or csv directory must exist
	if metaPath == "" && csvDir == "" {
		return errors.New("no workset metadata json file and no csv directory, workset: " + strconv.Itoa(setId) + " " + setName)
	}

	// open source database connection and check is it valid
	dn := runOpts.String(toDbDriverArgKey)
	if dn == "" && runOpts.IsExist(dbDriverArgKey) {
		dn = runOpts.String(dbDriverArgKey)
	}
	cs, dn := db.IfEmptyMakeDefault(modelName, runOpts.String(toSqliteArgKey), runOpts.String(toDbConnStrArgKey), dn)

	dstDb, _, err := db.Open(cs, dn, true)
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
	dstSetName := runOpts.String(setNewNameArgKey)

	dstId, err := fromWorksetTextToDb(dstDb, modelDef, langDef, setName, dstSetName, metaPath, csvDir)
	if err != nil {
		return err
	}
	if dstId <= 0 {
		return errors.New("workset not found or empty: " + strconv.Itoa(setId) + " " + setName)
	}

	return nil
}

// fromWorksetTextListToDb read all worksets parameters from csv and json files,
// convert it to db cells and insert into database
// update set id's and base run id's with actual id in database
func fromWorksetTextListToDb(
	dbConn *sql.DB, modelDef *db.ModelMeta, langDef *db.LangMeta, inpDir string,
) error {

	// get list of workset json files
	fl, err := filepath.Glob(inpDir + "/" + modelDef.Model.Name + ".set.*.json")
	if err != nil {
		return err
	}
	if len(fl) <= 0 {
		return nil // no worksets
	}

	// for each file:
	// read workset metadata, update workset in target database
	// read csv files from workset csv subdir and update parameter values
	for k := range fl {

		// check if workset subdir exist
		d, f := filepath.Split(fl[k])
		csvDir := strings.TrimSuffix(strings.TrimPrefix(f, modelDef.Model.Name+"."), ".json")

		if len(csvDir) <= 4 { // expected csv directory: set.4.q or set.q
			csvDir = ""
		} else {
			csvDir = filepath.Join(d, csvDir)
			if _, err := os.Stat(csvDir); err != nil {
				csvDir = ""
			}
		}

		// update or insert workset metadata and parameters from csv if csv directory exist
		_, err := fromWorksetTextToDb(dbConn, modelDef, langDef, "", "", fl[k], csvDir)
		if err != nil {
			return err
		}
	}

	return nil
}

// fromWorksetTextToDb read workset metadata from json file,
// read all parameters from csv files, convert it to db cells and insert into database
// update set id's and base run id's with actual id in destination database
// it return source workset id (set id from metadata json file) and destination set id
func fromWorksetTextToDb(
	dbConn *sql.DB,
	modelDef *db.ModelMeta,
	langDef *db.LangMeta,
	srcSetName string,
	dstSetName string,
	metaPath string,
	csvDir string,
) (int, error) {

	// if no metadata file and no csv directory then exit: nothing to do
	if metaPath == "" && csvDir == "" {
		return 0, nil // no workset
	}

	// get workset metadata:
	// model name and set name must be specified as parameter or inside of metadata json
	var pub db.WorksetPub

	if metaPath == "" && csvDir != "" { // no metadata json file, only csv directory
		pub.Name = srcSetName
		pub.ModelName = modelDef.Model.Name
	}

	if metaPath != "" { // read metadata json file

		isExist, err := helper.FromJsonFile(metaPath, &pub)
		if err != nil {
			return 0, err
		}

		if !isExist { // metadata from json is empty

			if csvDir == "" { // if metadata json empty and no csv directory then exit: no data
				return 0, nil
			}
			// metadata empty but there is csv directory: use expected model name and set name
			pub.Name = srcSetName
			pub.ModelName = modelDef.Model.Name
		}
	}
	if pub.Name == "" {
		return 0, errors.New("workset name is empty and metadata json file not found or empty")
	}
	srcSetName = pub.Name

	if theCfg.isNoDigestCheck {
		pub.ModelDigest = "" // model digest validation disabled
	}

	// if only csv directory specified:
	//   make list of parameters based on csv file names
	//   assume only one parameter sub-value in csv file
	if metaPath == "" && csvDir != "" {

		fl, err := filepath.Glob(csvDir + "/*.csv")
		if err != nil {
			return 0, err
		}
		pub.Param = make([]db.ParamRunSetPub, len(fl))

		for j := range fl {
			fn := filepath.Base(fl[j])
			fn = fn[:len(fn)-4] // remove .csv extension
			pub.Param[j].Name = fn
			pub.Param[j].SubCount = 1 // only one sub-value
		}
	}

	// save workset metadata as "read-write" and after importing all parameters set it as "readonly"
	// save workset metadata parameters list, make it empty and use add parameters to update metadata and values from csv
	isReadonly := pub.IsReadonly
	pub.IsReadonly = false
	paramLst := append([]db.ParamRunSetPub{}, pub.Param...)
	pub.Param = []db.ParamRunSetPub{}

	// rename destination workset
	if dstSetName != "" {
		pub.Name = dstSetName
	}

	// destination: convert from "public" format into destination db rows
	// display warning if base run not found in destination database
	ws, err := pub.FromPublic(dbConn, modelDef)
	if err != nil {
		return 0, err
	}
	if ws.Set.BaseRunId <= 0 && pub.BaseRunDigest != "" {
		omppLog.LogFmt("Warning: workset %s, base run not found by digest %s", ws.Set.Name, pub.BaseRunDigest)
	}

	// if destination workset exists then make it read-write and delete all existing parameters from workset
	wsRow, err := db.GetWorksetByName(dbConn, modelDef.Model.ModelId, ws.Set.Name)
	if err != nil {
		return 0, err
	}
	if wsRow != nil {
		err = db.UpdateWorksetReadonly(dbConn, wsRow.SetId, false) // make destination workset read-write
		if err != nil {
			return 0, errors.New("failed to clear workset read-only status: " + strconv.Itoa(wsRow.SetId) + " " + wsRow.Name + " " + err.Error())
		}
		err = db.DeleteWorksetAllParameters(dbConn, wsRow.SetId) // delete all parameters from workset
		if err != nil {
			return 0, errors.New("failed to delete workset " + strconv.Itoa(wsRow.SetId) + " " + wsRow.Name + " " + err.Error())
		}
	}

	// create empty workset metadata or update existing workset metadata
	err = ws.UpdateWorkset(dbConn, modelDef, true, langDef)
	if err != nil {
		return 0, err
	}
	dstId := ws.Set.SetId // actual set id from destination database

	// read all workset parameters and copy into destination database
	omppLog.Log("Workset ", srcSetName, " into: ", dstId, " "+ws.Set.Name)
	nP := len(paramLst)
	omppLog.Log("  Parameters:", nP)
	logT := time.Now().Unix()

	// read all workset parameters from csv files
	for j := range paramLst {

		// read parameter values from csv file
		logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s", j, nP, paramLst[j].Name))

		cvtParam := db.CellParamConverter{
			ModelDef:  modelDef,
			Name:      paramLst[j].Name,
			IsIdCsv:   false,
			DoubleFmt: theCfg.doubleFmt,
		}

		err = updateWorksetParamFromCsvFile(dbConn, modelDef, ws, &paramLst[j], csvDir, langDef, cvtParam)
		if err != nil {
			return 0, err
		}
	}

	// update workset readonly status with actual value
	err = db.UpdateWorksetReadonly(dbConn, dstId, isReadonly)
	if err != nil {
		return 0, err
	}

	return dstId, nil
}

// updateWorksetParamFromCsvFile read parameter csv file values insert it into db parameter value table and update workset parameter metadata
func updateWorksetParamFromCsvFile(
	dbConn *sql.DB,
	modelDef *db.ModelMeta,
	wsMeta *db.WorksetMeta,
	paramPub *db.ParamRunSetPub,
	csvDir string,
	langDef *db.LangMeta,
	csvCvt db.CellParamConverter,
) error {

	// converter from csv row []string to db cell
	cvt, err := csvCvt.ToCell()
	if err != nil {
		return errors.New("invalid converter from csv row: " + err.Error())
	}

	// open csv file, convert to utf-8 and parse csv into db cells
	// reading from .id.csv files not supported by converters
	fn, err := csvCvt.CsvFileName()
	if err != nil {
		return errors.New("invalid csv file name: " + err.Error())
	}
	chs, err := csvCvt.CsvHeader()
	if err != nil {
		return errors.New("Error at building csv parameter header " + paramPub.Name + ": " + err.Error())
	}
	ch := strings.Join(chs, ",")

	f, err := os.Open(filepath.Join(csvDir, fn))
	if err != nil {
		return helper.ErrorNew("csv file open error:", fn, ":", err)
	}
	defer f.Close()

	from, err := makeFromCsvReader(fn, f, ch, cvt)
	if err != nil {
		return errors.New("fail to create expressions csv reader: " + err.Error())
	}

	// write each csv row into parameter or output table
	_, err = wsMeta.UpdateWorksetParameterFrom(dbConn, modelDef, true, paramPub, langDef, from)
	if err != nil {
		return err
	}

	return nil
}
