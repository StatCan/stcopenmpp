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

// copy model run from source database to destination database
func dbToDbRun(modelName string, modelDigest string, runOpts *config.RunOptions) error {

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
	dstDb, dbFacet, err := db.Open(csOut, dnOut, true)
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

	// find source model run metadata by id, run digest or name
	runId, runDigest, runName, isFirst, isLast := runIdDigestNameFromOptions(runOpts)
	if runId < 0 || runId == 0 && runName == "" && runDigest == "" && !isFirst && !isLast {
		return helper.ErrorFmt("dbcopy invalid argument(s) run id: %s, run name: %s, run digest: %s",
			runOpts.String(runIdArgKey), runOpts.String(runNameArgKey), runOpts.String(runDigestArgKey))
	}
	runRow, e := findModelRunByIdDigestName(srcDb, srcModel.Model.ModelId, runId, runDigest, runName, isFirst, isLast)
	if e != nil {
		return e
	}
	if runRow == nil {
		return helper.ErrorNew("Model run not found:", runOpts.String(runIdArgKey), runOpts.String(runNameArgKey), runOpts.String(runDigestArgKey))
	}

	// check is this run belong to the source model
	if runRow.ModelId != srcModel.Model.ModelId {
		return helper.ErrorFmt("model run %d %s %s does not belong to model %s %s", runRow.RunId, runRow.Name, runRow.RunDigest, modelName, modelDigest)
	}

	// run must be completed: status success, error or exit
	if !db.IsRunCompleted(runRow.Status) {
		return helper.ErrorNew("model run not completed:", runRow.RunId, runRow.Name, runRow.RunDigest)
	}

	// get full model run metadata
	meta, err := db.GetRunFullText(srcDb, runRow, true, "")
	if err != nil {
		return err
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

	// convert model run db rows into "public" format
	pub, err := meta.ToPublic(srcModel)
	if err != nil {
		return err
	}

	// if model digest validation disabled
	if theCfg.isNoDigestCheck {
		pub.ModelDigest = ""
	}

	// copy source model run metadata, parameter values, output results into destination database
	_, err = copyRunDbToDb(srcDb, dstDb, dbFacet, srcModel, dstModel, meta.Run.RunId, pub, dstLang)
	if err != nil {
		return err
	}

	return nil
}

// copyRunListDbToDb do copy all model runs parameters and output tables from source to destination database
// Double format is used for float model types digest calculation, if non-empty format supplied
func copyRunListDbToDb(
	srcDb *sql.DB, dstDb *sql.DB, dbFacet db.Facet, srcModel *db.ModelMeta, dstModel *db.ModelMeta, dstLang *db.LangMeta) error {

	// source: get all successfully completed model runs in all languages
	srcRl, err := db.GetRunFullTextList(srcDb, srcModel.Model.ModelId, true, "")
	if err != nil {
		return err
	}
	if len(srcRl) <= 0 {
		return nil
	}

	// copy all run metadata, run parameters, output accumulators and expressions from source to destination
	// model run "public" format is used
	for k := range srcRl {

		// convert model db rows into "public"" format
		pub, err := srcRl[k].ToPublic(srcModel)
		if err != nil {
			return err
		}

		// save into destination database
		_, err = copyRunDbToDb(srcDb, dstDb, dbFacet, srcModel, dstModel, srcRl[k].Run.RunId, pub, dstLang)
		if err != nil {
			return err
		}
	}
	return nil
}

// copyRunDbToDb do copy model run metadata, run parameters and output tables from source to destination database
// it return destination run id (run id in destination database)
func copyRunDbToDb(
	srcDb *sql.DB, dstDb *sql.DB, dbFacet db.Facet, srcModel *db.ModelMeta, dstModel *db.ModelMeta, srcId int, pub *db.RunPub, dstLang *db.LangMeta) (int, error) {

	// validate parameters
	if pub == nil {
		return 0, helper.ErrorNew("invalid (empty) source model run metadata, source run not found or not exists")
	}

	// destination: convert from "public" format into destination db rows
	dstRun, err := pub.FromPublic(dstModel)
	if err != nil {
		return 0, err
	}

	// destination: save model run metadata
	isExist, err := dstRun.UpdateRun(dstDb, dstModel, dstLang, theCfg.doubleFmt)
	if err != nil {
		return 0, err
	}
	dstId := dstRun.Run.RunId
	if isExist { // exit if model run already exist
		omppLog.LogFmt("Model run %d %s already exists as %d", srcId, pub.Name, dstId)
		return dstId, nil
	}

	// copy all run parameters, output accumulators and expressions from source to destination
	omppLog.LogFmt("Model run from %d %s to %d", srcId, pub.Name, dstId)
	nP := len(srcModel.Param)
	omppLog.Log("  Parameters:", nP)
	logT := time.Now().Unix()

	// copy all parameters values for that model run
	for j := range srcModel.Param {

		// source: read parameter values
		paramLt := db.ReadParamLayout{
			ReadLayout: db.ReadLayout{
				Name:   srcModel.Param[j].Name,
				FromId: srcId,
			},
		}
		logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s", j, nP, paramLt.Name))

		cLst := list.New()

		_, err := db.ReadParameterTo(srcDb, srcModel, &paramLt, func(src interface{}) (bool, error) {
			cLst.PushBack(src)
			return true, nil
		})
		if err != nil {
			return 0, err
		}
		if cLst.Len() <= 0 { // parameter data must exist for all parameters
			return 0, helper.ErrorFmt("missing run parameter values %s run id: %d", paramLt.Name, paramLt.FromId)
		}

		// destination: insert parameter values in model run
		dstParamLt := db.WriteParamLayout{
			WriteLayout: db.WriteLayout{
				Name: dstModel.Param[j].Name,
				ToId: dstId,
			},
			SubCount:  dstRun.Param[j].SubCount,
			IsToRun:   true,
			DoubleFmt: theCfg.doubleFmt,
		}

		if err = db.WriteParameterFrom(dstDb, dstModel, &dstParamLt, makeFromList(cLst)); err != nil {
			return 0, err
		}
	}

	// copy all output tables values for that model run, if the table included in run results
	nT := len(srcModel.Table)
	omppLog.Log("  Tables:", nT)

	for j := range srcModel.Table {

		// check if table exist in model run results
		var isFound bool
		for k := range pub.Table {
			isFound = pub.Table[k].Name == srcModel.Table[j].Name
			if isFound {
				break
			}
		}
		if !isFound {
			continue // skip table: it is suppressed and not in run results
		}

		// source: read output table accumulator
		tblLt := db.ReadTableLayout{
			ReadLayout: db.ReadLayout{
				Name:   srcModel.Table[j].Name,
				FromId: srcId,
			},
			IsAccum: true,
		}
		logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s", j, nT, tblLt.Name))

		acLst := list.New()

		_, err = db.ReadOutputTableTo(srcDb, srcModel, &tblLt, func(src interface{}) (bool, error) {
			acLst.PushBack(src)
			return true, nil
		})
		if err != nil {
			return 0, err
		}

		// source: read output table expression values
		tblLt.IsAccum = false
		ecLst := list.New()

		_, err = db.ReadOutputTableTo(srcDb, srcModel, &tblLt, func(src interface{}) (bool, error) {
			ecLst.PushBack(src)
			return true, nil
		})
		if err != nil {
			return 0, err
		}

		// insert output table values (accumulators and expressions) in model run
		dstTblLt := db.WriteTableLayout{
			WriteLayout: db.WriteLayout{
				Name: dstModel.Table[j].Name,
				ToId: dstId,
			},
			SubCount:  dstRun.Run.SubCount,
			DoubleFmt: theCfg.doubleFmt,
		}

		err = db.WriteOutputTableFrom(dstDb, dstModel, &dstTblLt, makeFromList(acLst), makeFromList(ecLst))
		if err != nil {
			return 0, err
		}
	}

	// copy entity microdata values from source run into destination
	nMd := len(pub.Entity)

	if nMd > 0 {

		omppLog.Log("  Microdata:", nMd)

		for j := 0; j < nMd; j++ {

			// source: read microdata values
			microLt := db.ReadMicroLayout{
				ReadLayout: db.ReadLayout{
					Name:   pub.Entity[j].Name,
					FromId: srcId},
				GenDigest: pub.Entity[j].GenDigest,
			}
			logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s", j, nMd, microLt.Name))

			cLst := list.New()

			_, err := db.ReadMicrodataTo(srcDb, srcModel, &microLt, func(src interface{}) (bool, error) {
				cLst.PushBack(src)
				return true, nil
			})
			if err != nil {
				return 0, err
			}
			if cLst.Len() != pub.Entity[j].RowCount {
				return 0, helper.ErrorFmt("missing run microdata values %s run id: %d", microLt.Name, microLt.FromId)
			}

			// destination: insert microdata values into model run
			dstMicroLt := db.WriteMicroLayout{
				WriteLayout: db.WriteLayout{
					Name: pub.Entity[j].Name,
					ToId: dstId,
				},
				DoubleFmt: theCfg.doubleFmt,
			}

			if err = db.WriteMicrodataFrom(dstDb, dbFacet, dstModel, dstRun, &dstMicroLt, makeFromList(cLst)); err != nil {
				return 0, err
			}
		}
	}

	return dstId, nil
}
