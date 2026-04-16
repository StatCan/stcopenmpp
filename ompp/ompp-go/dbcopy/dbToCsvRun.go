// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// toRunCsv write model run metadata, parameters and output tables into csv files, in separate subdirectory
func toRunCsv(
	dbConn *sql.DB,
	modelDef *db.ModelMeta,
	meta *db.RunMeta,
	outDir string,
	isUseIdNames bool,
	isAllInOne bool,
	fileCreated map[string]bool,
) error {

	// create run subdir under model dir
	runId := meta.Run.RunId
	omppLog.Log("Model run", runId, meta.Run.Name)

	// make output directory as one of:
	// all_model_runs, run.Name_Of_the_Run, run.NN.Name_Of_the_Run
	var csvTop string
	if isAllInOne {
		csvTop = filepath.Join(outDir, "all_model_runs")
	} else {
		if !isUseIdNames {
			csvTop = filepath.Join(outDir, "run."+helper.CleanFileName(meta.Run.Name))
		} else {
			csvTop = filepath.Join(outDir, "run."+strconv.Itoa(runId)+"."+helper.CleanFileName(meta.Run.Name))
		}
	}
	dirSuffix := ""
	if theCfg.isNoZeroCsv {
		dirSuffix = dirSuffix + ".no-zero"
	}
	if theCfg.isNoNullCsv {
		dirSuffix = dirSuffix + ".no-null"
	}
	paramCsvDir := filepath.Join(csvTop, "parameters")
	tableCsvDir := filepath.Join(csvTop, "output-tables"+dirSuffix)
	microCsvDir := filepath.Join(csvTop, "microdata"+dirSuffix)
	nMd := len(meta.EntityGen)

	if e := os.MkdirAll(paramCsvDir, 0750); e != nil {
		return e
	}
	if e := os.MkdirAll(tableCsvDir, 0750); e != nil {
		return e
	}
	if !theCfg.isNoMicrodata && nMd > 0 {
		if e := os.MkdirAll(microCsvDir, 0750); e != nil {
			return e
		}
	}

	// if this is "all-in-one" output then first column is run id or run name
	var firstCol, firstVal string
	if isAllInOne {
		if theCfg.isIdCsv {
			firstCol = "run_id"
			firstVal = strconv.Itoa(runId)
		} else {
			firstCol = "run_name"
			firstVal = meta.Run.Name
		}
	}

	// write all parameters into csv file
	nP := len(modelDef.Param)
	omppLog.Log("  Parameters: ", nP)
	logT := time.Now().Unix()

	for j := 0; j < nP; j++ {

		cvtParam := &db.CellParamConverter{
			ModelDef:  modelDef,
			Name:      modelDef.Param[j].Name,
			IsIdCsv:   theCfg.isIdCsv,
			DoubleFmt: theCfg.doubleFmt,
		}
		paramLt := db.ReadParamLayout{ReadLayout: db.ReadLayout{
			Name:   modelDef.Param[j].Name,
			FromId: runId,
		}}

		logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s", j, nP, paramLt.Name))

		err := toCellCsvFile(dbConn, modelDef, paramLt, cvtParam, fileCreated, paramCsvDir, firstCol, firstVal)
		if err != nil {
			return err
		}
	}

	// write each run parameter value notes into parameterName.LANG.md file
	if !isAllInOne {
		for j := range meta.Param {

			paramName := ""
			for i := range meta.Param[j].Txt {

				if meta.Param[j].Txt[i].LangCode != "" && meta.Param[j].Txt[i].Note != "" {

					// find parameter by name if this is a first note for that parameter
					if paramName == "" {
						k, ok := modelDef.ParamByHid(meta.Param[j].ParamHid)
						if !ok {
							return helper.ErrorNew("parameter not found by Hid:", meta.Param[j].ParamHid)
						}
						paramName = modelDef.Param[k].Name
					}

					// write notes into parameterName.LANG.md file
					err := toDotMdFile(
						paramCsvDir,
						paramName+"."+meta.Param[j].Txt[i].LangCode,
						meta.Param[j].Txt[i].Note)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	// write output tables into csv file, if the table included in run results
	nT := len(modelDef.Table)
	omppLog.Log("  Tables:", nT)

	for j := 0; j < nT; j++ {

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

		// write output table expression values into csv file
		tblLt := db.ReadTableLayout{
			ReadLayout: db.ReadLayout{
				Name:   modelDef.Table[j].Name,
				FromId: runId,
			},
		}
		ctc := db.CellTableConverter{
			ModelDef:    modelDef,
			Name:        modelDef.Table[j].Name,
			IsIdCsv:     theCfg.isIdCsv,
			DoubleFmt:   theCfg.doubleFmt,
			IsNoZeroCsv: theCfg.isNoZeroCsv,
			IsNoNullCsv: theCfg.isNoNullCsv,
		}
		cvtExpr := &db.CellExprConverter{CellTableConverter: ctc}
		cvtAcc := &db.CellAccConverter{CellTableConverter: ctc}
		cvtAll := &db.CellAllAccConverter{CellTableConverter: ctc}

		logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s", j, nT, tblLt.Name))

		err := toCellCsvFile(dbConn, modelDef, tblLt, cvtExpr, fileCreated, tableCsvDir, firstCol, firstVal)
		if err != nil {
			return err
		}

		// write output table accumulators into csv file
		if !theCfg.isNoAccCsv {

			tblLt.IsAccum = true
			tblLt.IsAllAccum = false

			logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s accumulators", j, nT, tblLt.Name))

			err = toCellCsvFile(dbConn, modelDef, tblLt, cvtAcc, fileCreated, tableCsvDir, firstCol, firstVal)
			if err != nil {
				return err
			}

			// write all accumulators view into csv file
			tblLt.IsAccum = true
			tblLt.IsAllAccum = true

			logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s all accumulators", j, nT, tblLt.Name))

			err = toCellCsvFile(dbConn, modelDef, tblLt, cvtAll, fileCreated, tableCsvDir, firstCol, firstVal)
			if err != nil {
				return err
			}
		}
	}

	// write microdata into csv file, if there is any microdata for that model run and microadata write enabled
	if !theCfg.isNoMicrodata && nMd > 0 {

		omppLog.Log("  Microdata:", nMd)

		for j := 0; j < nMd; j++ {

			eId := meta.EntityGen[j].EntityId
			eIdx, isFound := modelDef.EntityByKey(eId)
			if !isFound {
				return helper.ErrorNew("error: entity not found by Id:", eId, meta.EntityGen[j].GenDigest)
			}

			cvtMicro := &db.CellMicroConverter{CellEntityConverter: db.CellEntityConverter{
				ModelDef:    modelDef,
				Name:        modelDef.Entity[eIdx].Name,
				EntityGen:   &meta.EntityGen[j],
				IsIdCsv:     theCfg.isIdCsv,
				DoubleFmt:   theCfg.doubleFmt,
				IsNoZeroCsv: theCfg.isNoZeroCsv,
				IsNoNullCsv: theCfg.isNoNullCsv,
			}}
			microLt := db.ReadMicroLayout{
				ReadLayout: db.ReadLayout{
					Name:   modelDef.Entity[eIdx].Name,
					FromId: runId,
				},
				GenDigest: meta.EntityGen[j].GenDigest,
			}

			logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s", j, nMd, microLt.Name))

			err := toCellCsvFile(dbConn, modelDef, microLt, cvtMicro, fileCreated, microCsvDir, firstCol, firstVal)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
