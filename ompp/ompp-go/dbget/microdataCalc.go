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

// aggregate and compare model runs microdata, write results into csv or json files.
func microdataCompare(srcDb *sql.DB, modelId int, runOpts *config.RunOptions) error {

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
			return helper.ErrorNew("Error: Model run not found:", src)
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

	// get microdata entity, group by attributes and calcultion expression(s)
	entityName := runOpts.String(entityArgKey)
	if entityName == "" {
		return helper.ErrorNew("Invalid (empty) microdata entity name")
	}
	groupBy := helper.ParseCsvLine(runOpts.String(groupByArgKey), ',')
	if len(groupBy) <= 0 {
		return helper.ErrorNew("Invalid (empty) microdata group by attributes")
	}
	cLst := helper.ParseCsvLine(runOpts.String(aggrArgKey), ',')
	if len(cLst) <= 0 {
		return helper.ErrorNew("Invalid (empty) microdata aggregation expression(s)")
	}

	// set aggregation expressions
	calcLt := db.CalculateMicroLayout{
		Calculation: []db.CalculateLayout{},
		GroupBy:     groupBy,
	}
	cn := helper.ParseCsvLine(runOpts.String(aggrNameArgKey), ',') // list of names, if not empty

	for j := range cLst {

		if cLst[j] != "" {
			calcLt.Calculation = append(calcLt.Calculation, db.CalculateLayout{
				Calculate: cLst[j],
				CalcId:    j + db.CALCULATED_ID_OFFSET,
				Name:      "ex_" + strconv.Itoa(j+db.CALCULATED_ID_OFFSET),
			})
			if j < len(cn) && cn[j] != "" {
				calcLt.Calculation[j].Name = cn[j]
			}
		}
	}

	// get model metadata and find entity
	meta, err := db.GetModelById(srcDb, modelId)
	if err != nil {
		return helper.ErrorNew("Error at get model metadata by id:", modelId, err)
	}

	// find model entity by entity name
	eIdx, ok := meta.EntityByName(entityName)
	if !ok {
		return helper.ErrorNew("Error: model entity not found:" + entityName)
	}
	ent := &meta.Entity[eIdx]

	// create cell conveter to csv
	cvtMicro := &db.CellMicroCalcConverter{
		CellEntityConverter: db.CellEntityConverter{
			ModelDef:    meta,
			Name:        entityName,
			IsIdCsv:     theCfg.isIdCsv,
			DoubleFmt:   theCfg.doubleFmt,
			IsNoZeroCsv: runOpts.Bool(noZeroArgKey),
			IsNoNullCsv: runOpts.Bool(noNullArgKey),
		},
		CalcMaps: db.EmptyCalcMaps(),
		GroupBy:  calcLt.GroupBy,
	}
	if e := cvtMicro.SetCalcIdNameMap(calcLt.Calculation); e != nil {
		return helper.ErrorNew("Failed to create microdata aggregation converter to csv:", entityName, e)
	}

	// set run id to name map in the convereter
	cvtMicro.CalcMaps.RunIdToLabel[baseRun.RunId] = baseRun.Name // add base run name

	runIds := make([]int, len(varRunLst))
	for k := 0; k < len(varRunLst); k++ {
		cvtMicro.CalcMaps.RunIdToLabel[varRunLst[k].RunId] = varRunLst[k].Name // add names of variant runs
		runIds[k] = varRunLst[k].RunId
	}

	// find entity generation by entity name and validate entity generation: it must exist for base run and all variant runs
	//
	// get list of entity generations for base model run
	egLst, err := db.GetEntityGenList(srcDb, baseRun.RunId)
	if err != nil {
		return helper.ErrorNew("Error at get run entities:", entityName, ":", baseRun.RunId, ":", err)
	}

	// find entity generation by entity id, as it is today model run has only one entity generation for each entity
	gIdx := -1
	for k := range egLst {

		if egLst[k].EntityId == ent.EntityId {
			gIdx = k
			break
		}
	}
	if gIdx < 0 {
		return helper.ErrorNew("Error: model run entity generation not found:", entityName, ":", baseRun.RunId)
	}
	entGen := &egLst[gIdx] // entity generation exists in the base run

	// collect generation attribues
	attrs := make([]db.EntityAttrRow, len(entGen.GenAttr))

	for k, ga := range entGen.GenAttr {

		aIdx, ok := ent.AttrByKey(ga.AttrId)
		if !ok {
			return helper.ErrorNew("entity attribute not found by id:", ga.AttrId, entityName)
		}
		attrs[k] = ent.Attr[aIdx]
	}

	// find all run_entity rows for that entity generation
	runEnt, err := db.GetRunEntityGenByModel(srcDb, modelId)
	if err != nil {
		return helper.ErrorNew("Error at get run entities by model id:", modelId, ":", err)
	}

	n := 0
	for k := 0; k < len(runEnt); k++ {
		if runEnt[k].GenHid == entGen.GenHid {
			runEnt[n] = runEnt[k]
			n++
		}
	}
	runEnt = runEnt[:n]

	// validate entity generation: it in the base run and must exist for all variant runs
	cvtMicro.EntityGen = entGen

	for k := 0; k < len(runIds); k++ {
		isFound := false
		for j := 0; !isFound && j < len(runEnt); j++ {
			isFound = runEnt[j].RunId == runIds[k]
		}
		if !isFound {
			return helper.ErrorNew("Failed to create microdata aggregation converter to csv, entity not found in the run:", runIds[k], ":", entityName)
		}
	}

	// validate group by attributes
	for k := 0; k < len(calcLt.GroupBy); k++ {

		isFound := false
		for j := 0; !isFound && j < len(attrs); j++ {
			isFound = attrs[j].Name == calcLt.GroupBy[k]
		}
		if !isFound {
			return helper.ErrorNew("Invalid group by attribute:", entityName+"."+calcLt.GroupBy[k])
		}
	}

	// read microdata values, page size =0: read all values
	microLt := db.ReadMicroLayout{
		ReadLayout: db.ReadLayout{
			Name:           entityName,
			FromId:         baseRun.RunId,
			ReadPageLayout: db.ReadPageLayout{Offset: 0, Size: 0},
		},
		GenDigest: entGen.GenDigest,
	}

	// make csv header
	// create converter from db cell into csv row []string
	hdr := []string{}
	var cvtRow func(interface{}, []string) (bool, error)

	if theCfg.isNoLang || theCfg.isIdCsv {

		hdr, err = cvtMicro.CsvHeader()
		if err != nil {
			return helper.ErrorNew("Failed to make microdata csv header:", entityName, ":", err)
		}
		if theCfg.isIdCsv {
			cvtRow, err = cvtMicro.ToCsvIdRow()
		} else {
			cvtRow, err = cvtMicro.ToCsvRow()
			hdr[0] = "run_name" // first column is a run name
		}
		if err != nil {
			return helper.ErrorNew("Failed to create microdata converter to csv:", entityName, ":", err)
		}

	} else { // get language-specific metadata

		txt, err := db.GetModelText(srcDb, meta.Model.ModelId, theCfg.lang, true)
		if err != nil {
			return helper.ErrorNew("Error at get language-specific metadata:", err)
		}

		cvtLoc := &db.CellMicroCalcLocaleConverter{
			CellMicroCalcConverter: *cvtMicro,
			Lang:                   theCfg.lang,
			EnumTxt:                txt.TypeEnumTxt,
			AttrTxt:                txt.EntityAttrTxt,
		}

		hdr, err = cvtLoc.CsvHeader()
		if err != nil {
			return helper.ErrorNew("Failed to make microdata csv header:", entityName, ":", err)
		}
		cvtRow, err = cvtLoc.ToCsvRow()
		if err != nil {
			return helper.ErrorNew("Failed to create microdata converter to csv:", entityName, ":", err)
		}
	}

	// start csv output to file or console
	fp := ""
	if theCfg.isConsole {
		omppLog.Log("Do", theCfg.action, entityName)
	} else {

		fp = theCfg.fileName
		if fp == "" {
			fp = entityName + extByKind()
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
		return helper.ErrorNew("Error at csv write:", entityName, ":", err)
	}

	// convert microdata cell into []string and write line into csv file
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

	// read microdata values page
	_, err = db.ReadMicrodataCalculateTo(srcDb, meta, &microLt, &calcLt, runIds, cvtWr)
	if err != nil {
		return helper.ErrorNew("Error at microdata run aggregation output:", entityName, ":", microLt.GenDigest, ":", err)
	}

	csvWr.Flush() // flush csv to output stream

	return nil
}
