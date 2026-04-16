// Copyright OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"path/filepath"
	"strconv"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// Compare output table expression(s) between model runs or aggregate output tables sub-values.
// Calculate non-aggregation value(s), for example, difference or ratio and write run results into csv or tsv file.
// Aggregate output table sub-values: calculate new measure and write run results into csv or tsv file.
func tableCompare(srcDb *sql.DB, modelId int, runOpts *config.RunOptions) error {

	// find base model run
	msg, baseRun, err := findRun(srcDb, modelId, runOpts.String(runArgKey), runOpts.Int(runIdArgKey, 0), runOpts.Bool(runFirstArgKey), runOpts.Bool(runLastArgKey))
	if err != nil {
		return helper.ErrorNew("Error at get base model run:", msg, err)
	}
	if baseRun != nil {
		if baseRun.Status != db.DoneRunStatus {
			return helper.ErrorNew("Error: base model run not completed successfully:", msg)
		}
	} else {
		if runOpts.String(runArgKey) != "" || runOpts.Int(runIdArgKey, 0) != 0 || runOpts.Bool(runFirstArgKey) || runOpts.Bool(runLastArgKey) {
			return helper.ErrorNew("Error: base model run not found")
		}
	}

	// make list of variant model runs
	varRunLst := []*db.RunRow{}

	// check variant run search results and push to vrarints list
	pushToVar := func(src string, m string, r *db.RunRow) error {

		if src != "" && r == nil {
			return helper.ErrorNew("Error: model run not found:", src)
		}
		if r.Status != db.DoneRunStatus {
			return helper.ErrorNew("Error: model run not completed successfully:", m)
		}
		if baseRun == nil { // if base run not specified then use first run as base run
			baseRun = r
			return nil
		}
		// else: add to the list of variant runs
		if r.RunDigest == baseRun.RunDigest {
			omppLog.Log("Warning: skip this model run, it is the same as base run:", src)
			return nil

		}

		// check if variant not already exist in the list of variants
		isFound := false
		for j := 0; !isFound && j < len(varRunLst); j++ {
			isFound = varRunLst[j].RunDigest == r.RunDigest
		}
		if !isFound {
			varRunLst = append(varRunLst, r)
		}
		return nil
	}

	// get variant runs from comma separarted list of digest, stamp or name
	if rdsnLst := helper.ParseCsvLine(runOpts.String(withRunsArgKey), ','); len(rdsnLst) > 0 {

		for _, rdsn := range rdsnLst {

			m, r, e := findRun(srcDb, modelId, rdsn, 0, false, false)
			if e != nil {
				return helper.ErrorNew("Error at get model run:", m, e)
			}
			if e = pushToVar(rdsn, m, r); e != nil {
				return e
			}
		}
	}
	// get variant runs from comma separarted list of run id's
	if idLst := helper.ParseCsvLine(runOpts.String(withRunIdsArgKey), ','); len(idLst) > 0 {

		for _, sId := range idLst {

			if sId == "" {
				continue
			}
			rId, e := strconv.Atoi(sId)
			if e != nil || rId <= 0 {
				return helper.ErrorNew("Invalid model run id:", sId)
			}

			m, r, e := findRun(srcDb, modelId, "", rId, false, false)
			if e != nil {
				return helper.ErrorNew("Error at get model run:", m, e)
			}
			if e = pushToVar(sId, m, r); e != nil {
				return e
			}
		}
	}
	// check if first run must be used as variant run
	if runOpts.Bool(withRunFirstArgKey) {

		m, r, e := findRun(srcDb, modelId, "", 0, true, false)
		if e != nil {
			return helper.ErrorNew("Error at get first model run:", m, e)
		}
		if e = pushToVar(m, m, r); e != nil {
			return e
		}
	}
	// check if last run must be used as variant run
	if runOpts.Bool(withRunLastArgKey) {

		m, r, e := findRun(srcDb, modelId, "", 0, false, true)
		if e != nil {
			return helper.ErrorNew("Error at get last model run:", m, e)
		}
		if e = pushToVar(m, m, r); e != nil {
			return e
		}
	}

	// check: base model run must exist
	if baseRun == nil {
		return helper.ErrorNew("Error: base model run not found")
	}

	// get model metadata and check if table exists in the model
	meta, err := db.GetModelById(srcDb, modelId)
	if err != nil {
		return helper.ErrorNew("Error at get model metadata by id:", modelId, ":", err)
	}
	name := runOpts.String(tableArgKey)

	if _, ok := meta.OutTableByName(name); !ok {
		return helper.ErrorNew("Error: model output table not found:", name)
	}

	// set  calculate layout: calculation and aggregation expressions
	calcLt := []db.CalculateTableLayout{}

	ce := helper.ParseCsvLine(runOpts.String(calcArgKey), ',')
	cn := helper.ParseCsvLine(runOpts.String(calcNameArgKey), ',')
	for j := range ce {

		if ce[j] != "" {
			calcLt = append(calcLt, db.CalculateTableLayout{
				CalculateLayout: db.CalculateLayout{
					Calculate: ce[j],
					CalcId:    j + db.CALCULATED_ID_OFFSET,
					Name:      "ex_" + strconv.Itoa(j+db.CALCULATED_ID_OFFSET),
				},
				IsAggr: false,
			})
			if j < len(cn) && cn[j] != "" {
				calcLt[j].Name = cn[j]
			}
		}
	}
	n := len(ce)

	ce = helper.ParseCsvLine(runOpts.String(aggrArgKey), ',')
	cn = helper.ParseCsvLine(runOpts.String(aggrNameArgKey), ',')
	for j := range ce {

		if ce[j] != "" {
			calcLt = append(calcLt, db.CalculateTableLayout{
				CalculateLayout: db.CalculateLayout{
					Calculate: ce[j],
					CalcId:    n + j + db.CALCULATED_ID_OFFSET,
					Name:      "ex_" + strconv.Itoa(n+j+db.CALCULATED_ID_OFFSET),
				},
				IsAggr: true,
			})
			if j < len(cn) && cn[j] != "" {
				calcLt[n+j].Name = cn[j]
			}
		}
	}
	if len(calcLt) <= 0 {
		return helper.ErrorNew("Error: invalid (empty) calculation and aggregation expression", runOpts.String(calcArgKey), runOpts.String(aggrArgKey))
	}

	// create cell converter to csv
	cvtTable := db.CellTableCalcConverter{
		CellTableConverter: db.CellTableConverter{
			ModelDef:    meta,
			Name:        name,
			IsIdCsv:     theCfg.isIdCsv,
			IsNoNullCsv: runOpts.Bool(noNullArgKey),
			IsNoZeroCsv: runOpts.Bool(noZeroArgKey),
			DoubleFmt:   theCfg.doubleFmt,
		},
		CalcMaps: db.EmptyCalcMaps(),
	}
	if e := cvtTable.SetCalcIdNameMap(calcLt); e != nil {
		return helper.ErrorNew("Failed to create output table converter to csv:", meta.Model.Name, name)
	}

	// set run id to name map in the convereter
	cvtTable.CalcMaps.RunIdToLabel[baseRun.RunId] = baseRun.Name // add base run name

	runIds := make([]int, len(varRunLst))
	for k := 0; k < len(varRunLst); k++ {
		cvtTable.CalcMaps.RunIdToLabel[varRunLst[k].RunId] = varRunLst[k].Name // add names of variant runs
		runIds[k] = varRunLst[k].RunId
	}

	// setup read layout: page size =0, read all values
	tableLt := db.ReadTableLayout{
		ReadLayout: db.ReadLayout{
			Name:   name,
			FromId: baseRun.RunId,
		},
	}

	// make csv header
	// create converter from db cell into csv row []string
	hdr := []string{}
	var cvtRow func(interface{}, []string) (bool, error)

	if theCfg.isNoLang || theCfg.isIdCsv {

		hdr, err = cvtTable.CsvHeader()
		if err != nil {
			return helper.ErrorNew("Failed to make output table csv header:", name, ":", err)
		}
		if theCfg.isIdCsv {
			cvtRow, err = cvtTable.ToCsvIdRow()
		} else {
			cvtRow, err = cvtTable.ToCsvRow()
			hdr[0] = "run_name" // first column is a run name
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
			return helper.ErrorNew("Error at get language-specific metadata:", err)
		}

		cvtLoc := &db.CellTableCalcLocaleConverter{
			CellTableCalcConverter: cvtTable,
			Lang:                   theCfg.lang,
			MsgDef:                 msgLst,
			DimsTxt:                txt.TableDimsTxt,
			EnumTxt:                txt.TypeEnumTxt,
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

	// write output table values to csv or tsv file
	fp := ""

	if theCfg.isConsole {
		omppLog.Log("Do", theCfg.action, name)
	} else {

		fp = theCfg.fileName
		if fp == "" {
			fp = name + extByKind()
		}
		fp = filepath.Join(theCfg.dir, fp)

		omppLog.Log("Do", theCfg.action, ":", fp)
	}

	f, csvWr, err := createCsvWriter(fp)
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

	// convert output table cell into []string and write line into csv file
	cs := make([]string, len(hdr))

	cvtWr := func(c interface{}) (bool, error) {

		// if converter return empty line then skip it
		isNotEmpty := true
		var e2 error = nil

		if isNotEmpty, e2 = cvtRow(c, cs); e2 != nil {
			return false, e2
		}
		if isNotEmpty {
			if e2 = csvWr.Write(cs); e2 != nil {
				return false, e2
			}
		}
		return true, nil
	}

	// read output table page
	_, err = db.ReadOutputTableCalculteTo(srcDb, meta, &tableLt, calcLt, runIds, cvtWr)
	if err != nil {
		return helper.ErrorNew("Error at output table aggregation output:", name, ":", err)
	}

	csvWr.Flush() // flush csv to output stream

	return nil
}
