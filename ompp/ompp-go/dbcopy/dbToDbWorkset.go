// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"container/list"
	"database/sql"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// copy workset from source database to destination database
func dbToDbWorkset(modelName string, modelDigest string, runOpts *config.RunOptions) error {

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

	// validate source and destination
	csInp, dnInp := db.IfEmptyMakeDefaultReadOnly(modelName, runOpts.String(fromSqliteArgKey), runOpts.String(dbConnStrArgKey), runOpts.String(dbDriverArgKey))
	csOut, dnOut := db.IfEmptyMakeDefault(modelName, runOpts.String(toSqliteArgKey), runOpts.String(toDbConnStrArgKey), runOpts.String(toDbDriverArgKey))

	if csInp == csOut && dnInp == dnOut {
		return helper.ErrorNew("source same as destination: cannot overwrite model in database")
	}

	// open source database connection and check is it valid
	srcDb, _, err := db.Open(csInp, dnInp, false)
	if err != nil {
		return err
	}
	defer srcDb.Close()

	if err := db.CheckOpenmppSchemaVersion(srcDb); err != nil {
		return err
	}

	// open destination database and check is it valid
	dstDb, _, err := db.Open(csOut, dnOut, true)
	if err != nil {
		return err
	}
	defer dstDb.Close()

	if err := db.CheckOpenmppSchemaVersion(dstDb); err != nil {
		return err
	}

	// source: get model metadata
	srcModel, err := db.GetModel(srcDb, modelName, modelDigest)
	if err != nil {
		return err
	}
	modelName = srcModel.Model.Name // set model name: it can be empty and only model digest specified

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
		if wsRow, err = db.GetWorksetByName(srcDb, srcModel.Model.ModelId, setName); err != nil {
			return err
		}
		if wsRow == nil {
			return helper.ErrorNew("Workset not found:", setName)
		}
	}

	srcWs, err := db.GetWorksetFull(srcDb, wsRow, "") // get full workset metadata
	if err != nil {
		return err
	}

	// check: workset must be readonly
	if !srcWs.Set.IsReadonly {
		return helper.ErrorNew("workset must be readonly:", wsRow.SetId, wsRow.Name)
	}

	// destination: get model metadata
	dstModel, err := db.GetModel(dstDb, modelName, modelDigest)
	if err != nil {
		return err
	}

	// destination: get list of languages
	dstLang, err := db.GetLanguages(dstDb)
	if err != nil {
		return err
	}

	// convert workset db rows into "public" format
	pub, err := srcWs.ToPublic(srcDb, srcModel)
	if err != nil {
		return err
	}

	// if model digest validation disabled
	if theCfg.isNoDigestCheck {
		pub.ModelDigest = ""
	}

	// rename destination workset
	if runOpts.IsExist(setNewNameArgKey) {
		pub.Name = runOpts.String(setNewNameArgKey)
	}

	// copy source workset metadata and parameters into destination database
	_, err = copyWorksetDbToDb(srcDb, dstDb, srcModel, dstModel, srcWs.Set.SetId, pub, dstLang)
	if err != nil {
		return err
	}
	return nil
}

// copyWorksetListDbToDb do copy all readonly worksets parameters from source to destination database
func copyWorksetListDbToDb(
	srcDb *sql.DB, dstDb *sql.DB, srcModel *db.ModelMeta, dstModel *db.ModelMeta, dstLang *db.LangMeta) error {

	// source: get all readonly worksets in all languages
	srcWl, err := db.GetWorksetFullList(srcDb, srcModel.Model.ModelId, true, "")
	if err != nil {
		return err
	}
	if len(srcWl) <= 0 {
		return nil
	}

	// copy worksets from source to destination database by using "public" format
	for k := range srcWl {

		// convert workset db rows into "public"" format
		pub, err := srcWl[k].ToPublic(srcDb, srcModel)
		if err != nil {
			return err
		}

		// save into destination database
		_, err = copyWorksetDbToDb(srcDb, dstDb, srcModel, dstModel, srcWl[k].Set.SetId, pub, dstLang)
		if err != nil {
			return err
		}
	}
	return nil
}

// copyWorksetDbToDb do copy workset metadata and parameters from source to destination database
// it return destination set id (set id in destination database)
func copyWorksetDbToDb(
	srcDb *sql.DB, dstDb *sql.DB, srcModel *db.ModelMeta, dstModel *db.ModelMeta, srcId int, pub *db.WorksetPub, dstLang *db.LangMeta) (int, error) {

	// validate parameters
	if pub == nil {
		return 0, helper.ErrorNew("invalid (empty) source workset metadata, source workset not found or not exists")
	}

	// save workset metadata as "read-write" and after importing all parameters set it as "readonly"
	// save workset metadata parameters list, make it empty and use add parameters to update metadata and values from csv
	isReadonly := pub.IsReadonly
	pub.IsReadonly = false
	paramLst := append([]db.ParamRunSetPub{}, pub.Param...)
	pub.Param = []db.ParamRunSetPub{}

	// destination: convert from "public" format into destination db rows
	// display warning if base run not found in destination database
	dstWs, err := pub.FromPublic(dstDb, dstModel)
	if err != nil {
		return 0, err
	}
	if dstWs.Set.BaseRunId <= 0 && pub.BaseRunDigest != "" {
		omppLog.LogFmt("Warning: workset %s, base run not found by digest %s", dstWs.Set.Name, pub.BaseRunDigest)
	}

	// if destination workset exists then make it read-write and delete all existing parameters from workset
	wsRow, err := db.GetWorksetByName(dstDb, dstModel.Model.ModelId, pub.Name)
	if err != nil {
		return 0, err
	}
	if wsRow != nil {
		err = db.UpdateWorksetReadonly(dstDb, wsRow.SetId, false) // make destination workset read-write
		if err != nil {
			return 0, helper.ErrorNew("failed to clear workset read-only status:", wsRow.SetId, wsRow.Name, err)
		}
		err = db.DeleteWorksetAllParameters(dstDb, wsRow.SetId) // delete all parameters from workset
		if err != nil {
			return 0, helper.ErrorNew("failed to delete workset", wsRow.SetId, wsRow.Name, err)
		}
	}

	// create empty workset metadata or update existing workset metadata
	err = dstWs.UpdateWorkset(dstDb, dstModel, true, dstLang)
	if err != nil {
		return 0, err
	}
	dstId := dstWs.Set.SetId // actual set id from destination database

	// read all workset parameters and copy into destination database
	omppLog.LogFmt("Workset %s from id %d to %d", dstWs.Set.Name, srcId, dstId)
	nP := len(paramLst)
	omppLog.Log("  Parameters:", nP)
	logT := time.Now().Unix()

	paramLt := &db.ReadParamLayout{ReadLayout: db.ReadLayout{FromId: srcId}, IsFromSet: true}

	// write parameter into destination database
	for j := range paramLst {

		// source: read workset parameter values
		paramLt.Name = paramLst[j].Name
		cLst := list.New()

		logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s", j, nP, paramLt.Name))

		_, err := db.ReadParameterTo(srcDb, srcModel, paramLt, func(src interface{}) (bool, error) {
			cLst.PushBack(src)
			return true, nil
		})
		if err != nil {
			return 0, err
		}
		if cLst.Len() <= 0 { // parameter data must exist for all parameters
			return 0, helper.ErrorFmt("missing workset parameter values %s set id: %d", paramLt.Name, paramLt.FromId)
		}

		// destination: insert or update parameter values in workset
		_, err = dstWs.UpdateWorksetParameterFrom(dstDb, dstModel, true, &paramLst[j], dstLang, makeFromList(cLst))
		if err != nil {
			return 0, err
		}
	}

	// update workset readonly status with actual value
	err = db.UpdateWorksetReadonly(dstDb, dstId, isReadonly)
	if err != nil {
		return 0, err
	}

	return dstId, nil
}
