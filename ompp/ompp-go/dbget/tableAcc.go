// Copyright OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"path/filepath"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// get output table native (not derived) accumulators and write run results into csv or tsv file.
func tableAcc(srcDb *sql.DB, modelId int, runOpts *config.RunOptions) error {

	// find model run
	msg, run, err := findRun(srcDb, modelId, runOpts.String(runArgKey), runOpts.Int(runIdArgKey, 0), runOpts.Bool(runFirstArgKey), runOpts.Bool(runLastArgKey))
	if err != nil {
		return helper.ErrorNew("Error at get model run:", msg, err)
	}
	if run == nil {
		return helper.ErrorNew("Error: model run not found")
	}
	if run.Status != db.DoneRunStatus {
		return helper.ErrorNew("Error: model run not completed successfully:", run.Name)
	}

	// get model metadata
	meta, err := db.GetModelById(srcDb, modelId)
	if err != nil {
		return helper.ErrorNew("Error at get model metadata by id:", modelId, ":", err)
	}

	// write output table accumulators to csv or tsv file
	name := runOpts.String(tableArgKey)
	fp := ""

	if theCfg.isConsole {
		omppLog.Log("Do", theCfg.action, name)
	} else {

		fp = theCfg.fileName
		if fp == "" {
			fp = name + ".acc" + extByKind()
		}
		fp = filepath.Join(theCfg.dir, fp)

		omppLog.Log("Do", theCfg.action, ":", fp)
	}

	return tableRunAcc(srcDb, meta, name, run.RunId, runOpts, fp)
}

// read output table native (not derived) accumulators and write run results into csv or tsv file.
// Csv file header: acc_name,sub_id,dim0,....,value
func tableRunAcc(srcDb *sql.DB, meta *db.ModelMeta, name string, runId int, runOpts *config.RunOptions, path string) error {

	if name == "" {
		return helper.ErrorNew("Invalid (empty) output table name")
	}
	if meta == nil {
		return helper.ErrorNew("Invalid (empty) model metadata")
	}
	_, ok := meta.OutTableByName(name)
	if !ok {
		return helper.ErrorNew("Error: model output table not found:", name)
	}

	// make csv header
	// create converter from db cell into csv row []string
	var err error
	hdr := []string{}
	var cvtRow func(interface{}, []string) (bool, error)

	cvtAcc := &db.CellAccConverter{CellTableConverter: db.CellTableConverter{
		ModelDef:    meta,
		Name:        name,
		IsIdCsv:     theCfg.isIdCsv,
		DoubleFmt:   theCfg.doubleFmt,
		IsNoZeroCsv: runOpts.Bool(noZeroArgKey),
		IsNoNullCsv: runOpts.Bool(noNullArgKey),
	}}

	tblLt := db.ReadTableLayout{
		ReadLayout: db.ReadLayout{
			Name:   name,
			FromId: runId,
		},
		IsAccum:    true,
		IsAllAccum: false,
	}

	if theCfg.isNoLang || theCfg.isIdCsv {

		hdr, err = cvtAcc.CsvHeader()
		if err != nil {
			return helper.ErrorNew("Failed to make output table csv header:", name, ":", err)
		}
		if theCfg.isIdCsv {
			cvtRow, err = cvtAcc.ToCsvIdRow()
		} else {
			cvtRow, err = cvtAcc.ToCsvRow()
		}
		if err != nil {
			return helper.ErrorNew("Failed to create output table converter to csv:", name, ":", err)
		}

	} else { // get language-specific metadata

		langDef, err := db.GetLanguages(srcDb)
		if err != nil {
			return helper.ErrorNew("Error at get language-specific metadata:", err)
		}

		// make list of model translated strings: merge common.message.ini and lang_word
		msgLst := db.NewLangMsg(langDef.Lang, nil)

		if cmIni, e := config.ReadCommonMessageIni(theCfg.binDir, theCfg.encodingName); e == nil {
			msgLst = db.AppendLangMsgFromIni(msgLst, cmIni)
		}

		// model language-specific lables for dimensions, items and tables
		txt, err := db.GetModelText(srcDb, meta.Model.ModelId, theCfg.lang, true)
		if err != nil {
			return helper.ErrorNew("Error at get model text metadata:", err)
		}

		cvtLoc := &db.CellAccLocaleConverter{
			CellAccConverter: *cvtAcc,
			Lang:             theCfg.lang,
			MsgDef:           msgLst,
			DimsTxt:          txt.TableDimsTxt,
			EnumTxt:          txt.TypeEnumTxt,
			AccTxt:           txt.TableAccTxt,
		}

		hdr, err = cvtLoc.CsvHeader()
		if err != nil {
			return helper.ErrorNew("Failed to make output table csv header:", name, ":", err)
		}
		cvtRow, err = cvtLoc.ToCsvRow()
		if err != nil {
			return helper.ErrorNew("Failed to create output table converter to csv:", name, ":", err)
		}
	}

	// start csv output to file or console
	f, csvWr, err := createCsvWriter(path)
	if err != nil {
		return err
	}
	isFile := f != nil

	defer func() {
		if isFile {
			f.Close()
		}
	}()

	// write csv header
	if err := csvWr.Write(hdr); err != nil {
		return helper.ErrorNew("Error at csv write:", name, ":", err)
	}

	// convert cell into []string and write line into csv file
	cs := make([]string, len(hdr))

	cvtWr := func(c interface{}) (bool, error) {

		// if converter return empty line then skip it
		isNotEmpty := false
		var e2 error = nil

		if isNotEmpty, e2 = cvtRow(c, cs); e2 != nil {
			return false, e2
		}
		if !isNotEmpty {
			return true, nil
		}

		e2 = csvWr.Write(cs)
		return e2 == nil, e2
	}

	// read output table accumulators
	_, err = db.ReadOutputTableTo(srcDb, meta, &tblLt, cvtWr)
	if err != nil {
		return helper.ErrorNew("Error at output table output:", name, ":", err)
	}

	csvWr.Flush() // flush csv to response

	return nil
}
