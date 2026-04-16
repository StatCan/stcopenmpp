// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"errors"
	"strconv"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// ParameterCellConverter return parameter value converter between code cell and id's cell.
// If isToId true then from code to id cell else other way around
func (mc *ModelCatalog) ParameterCellConverter(
	isToId bool, dn string, name string,
) (
	func(interface{}) (interface{}, error), bool,
) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		return nil, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Error: model digest or name not found: ", dn)
		return nil, false // return empty result: model not found or error
	}

	// check if parameter name exist in the model
	if _, ok = meta.ParamByName(name); !ok {
		omppLog.Log("Error: model parameter not found:", dn, ":", name)
		return nil, false
	}

	// create converter
	ctc := db.CellParamConverter{
		ModelDef:  meta,
		Name:      name,
		DoubleFmt: theCfg.doubleFmt,
	}
	var cvt func(interface{}) (interface{}, error)
	var err error

	if isToId {
		cvt, err = ctc.CodeToIdCell(meta, name)
	} else {
		cvt, err = ctc.IdToCodeCell(meta, name)
	}
	if err != nil {
		omppLog.Log("Failed to create parameter cell value converter: ", name, ": ", err.Error())
		return nil, false
	}

	return cvt, true
}

// TableToCodeCellConverter return output table value converter from id's cell into code cell.
func (mc *ModelCatalog) TableToCodeCellConverter(dn string, name string, isAcc, isAllAcc bool) (func(interface{}) (interface{}, error), bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		return nil, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Error: model digest or name not found: ", dn)
		return nil, false // return empty result: model not found or error
	}

	// check if output table name exist in the model
	if _, ok = meta.OutTableByName(name); !ok {
		omppLog.Log("Error: model output table not found: ", dn, ": ", name)
		return nil, false
	}

	// create converter
	ctc := db.CellTableConverter{
		ModelDef:  meta,
		Name:      name,
		IsIdCsv:   false,
		DoubleFmt: theCfg.doubleFmt,
	}
	var cvt func(interface{}) (interface{}, error)
	var err error

	switch {
	case isAcc && isAllAcc:
		csvCvt := db.CellAllAccConverter{
			CellTableConverter: ctc,
			ValueName:          "",
		}
		cvt, err = csvCvt.IdToCodeCell(meta, name)
	case isAcc:
		csvCvt := db.CellAccConverter{
			CellTableConverter: ctc,
		}
		cvt, err = csvCvt.IdToCodeCell(meta, name)
	default:
		csvCvt := db.CellExprConverter{
			CellTableConverter: ctc,
		}
		cvt, err = csvCvt.IdToCodeCell(meta, name)
	}
	if err != nil {
		omppLog.Log("Failed to create output table cell id's to code converter: ", name, ": ", err.Error())
		return nil, false
	}

	return cvt, true
}

// TableToCodeCalcCellConverter return output table calculated value converter from id's cell into code cell and run id(s).
// Function accept base run digest-or-stamp-or-name and optional list of variant runs digest-or-stamp-or-name.
// All runs must be completed successfully.
func (mc *ModelCatalog) TableToCodeCalcCellConverter(
	dn string, rdsn string, tableName string, calcLt []db.CalculateTableLayout, runLst []string,
) (func(interface{}) (interface{}, error), int, []int, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		return nil, 0, nil, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Error: model digest or name not found: ", dn)
		return nil, 0, nil, false // return empty result: model not found or error
	}

	// check if output table name exist in the model
	if _, ok = meta.OutTableByName(tableName); !ok {
		omppLog.Log("Error: model output table not found: ", dn, ": ", tableName)
		return nil, 0, nil, false
	}

	// create converter
	ctc := db.CellTableCalcConverter{
		CellTableConverter: db.CellTableConverter{
			ModelDef:  meta,
			Name:      tableName,
			IsIdCsv:   false,
			DoubleFmt: theCfg.doubleFmt,
		},
		CalcMaps: db.EmptyCalcMaps(),
	}
	if e := ctc.SetCalcIdNameMap(calcLt); e != nil {
		omppLog.Log("Failed to create output table cell id's to code converter: ", tableName, ": ", e.Error())
		return nil, 0, nil, false
	}

	cvt, err := ctc.IdToCodeCell(meta, tableName)
	if err != nil {
		omppLog.Log("Failed to create output table cell id's to code converter: ", tableName, ": ", err.Error())
		return nil, 0, nil, false
	}

	// validate all runs: it must be completed successfully
	// set run id's maps in the convereter
	baseRunId, runIds, ok := mc.setRunIdDigestMap(meta.Model.Digest, rdsn, runLst, &ctc.CalcMaps)

	return cvt, baseRunId, runIds, ok
}

// TableAggrExprCalculateLayout return calculate layout
// either for all expressions by aggregation name: sum avg count min max var sd se cv
// or from comma separated list of aggregation exprission(s), for example: OM_AVG(acc0) , OM_SD(acc1)
func (mc *ModelCatalog) TableAggrExprCalculateLayout(dn string, name string, aggr string) ([]db.CalculateTableLayout, bool) {

	// check aggregation operation
	fnc := ""
	switch aggr {
	case "sum":
		fnc = "OM_SUM"
	case "avg":
		fnc = "OM_AVG"
	case "count":
		fnc = "OM_COUNT"
	case "min":
		fnc = "OM_MIN"
	case "max":
		fnc = "OM_MAX"
	case "var":
		fnc = "OM_VAR"
	case "sd":
		fnc = "OM_SD"
	case "se":
		fnc = "OM_SE"
	case "cv":
		fnc = "OM_CV"
	default: // comma separated list of expressions

		calcLt := []db.CalculateTableLayout{}

		ce := helper.ParseCsvLine(aggr, 0)
		for j := range ce {

			if ce[j] != "" {
				calcLt = append(calcLt, db.CalculateTableLayout{
					CalculateLayout: db.CalculateLayout{
						Calculate: ce[j],
						CalcId:    j + db.CALCULATED_ID_OFFSET,
						Name:      "ex_" + strconv.Itoa(j+db.CALCULATED_ID_OFFSET),
					},
					IsAggr: true,
				})
			}
		}
		if len(calcLt) <= 0 {
			omppLog.Log("Error: invalid (empty) calculation expression")
			return []db.CalculateTableLayout{}, false
		}

		return calcLt, true
	}
	if fnc == "" {
		omppLog.Log("Error: invalid aggregation name: ", aggr, ": ", dn, ": ", name)
		return []db.CalculateTableLayout{}, false
	}

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		return []db.CalculateTableLayout{}, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Error: model digest or name not found: ", dn)
		return []db.CalculateTableLayout{}, false // return empty result: model not found or error
	}

	// find output table by name
	idx, ok := meta.OutTableByName(name)
	if !ok {
		omppLog.Log("Error: model output table not found: ", dn, ": ", name)
		return []db.CalculateTableLayout{}, false
	}
	table := &meta.Table[idx]

	// calculate output table expression and aggregation expression
	// for example: if Expr1 = AVG(acc1) and aggregation function is SD()
	// then append to calculation: Expr1 and SD(acc1)
	calcLt := []db.CalculateTableLayout{}

	for _, ex := range table.Expr {

		// append output table expression
		calcLt = append(calcLt, db.CalculateTableLayout{
			CalculateLayout: db.CalculateLayout{
				Calculate: ex.Name,
				CalcId:    ex.ExprId,
				Name:      ex.Name,
			},
			IsAggr: false,
		})

		for _, acc := range table.Acc {

			// find derived accumulator with the same name as expression name
			if acc.IsDerived && acc.Name == ex.Name {

				calcLt = append(calcLt, db.CalculateTableLayout{
					CalculateLayout: db.CalculateLayout{
						Calculate: fnc + "(" + acc.SrcAcc + ")",
						CalcId:    ex.ExprId + db.CALCULATED_ID_OFFSET,
						Name:      fnc + "_" + ex.Name,
					},
					IsAggr: true,
				})
				break
			}
		}
	}

	return calcLt, true
}

// TableExprCompareLayout return calculate layout
// either for all expressions by name: diff ratio percent
// or from comma separated list of exprission(s), for example: expr0 , 7 + expr1[variant] + expr2[base]
func (mc *ModelCatalog) TableExprCompareLayout(dn string, name string, cmp string) ([]db.CalculateTableLayout, bool) {

	// check comparison expression
	var fnc func(expr string) string
	switch cmp {
	case "diff":
		fnc = func(expr string) string {
			return expr + "[variant] - " + expr + "[base]"
		}
	case "ratio":
		fnc = func(expr string) string {
			return expr + "[variant] / " + expr + "[base]"
		}
	case "percent":
		fnc = func(expr string) string {
			return "100 * (" + expr + "[variant] - " + expr + "[base]" + ") / " + expr + "[base]"
		}
	default: // comma separated list of expressions

		calcLt := []db.CalculateTableLayout{}

		ce := helper.ParseCsvLine(cmp, 0)
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
			}
		}
		if len(calcLt) <= 0 {
			omppLog.Log("Error: invalid (empty) comparison expression")
			return []db.CalculateTableLayout{}, false
		}

		return calcLt, true
	}
	if fnc == nil {
		omppLog.Log("Error: invalid comparison name: ", cmp, ": ", dn, ": ", name)
		return []db.CalculateTableLayout{}, false
	}

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		return []db.CalculateTableLayout{}, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Error: model digest or name not found: ", dn)
		return []db.CalculateTableLayout{}, false // return empty result: model not found or error
	}

	// find output table by name
	idx, ok := meta.OutTableByName(name)
	if !ok {
		omppLog.Log("Error: model output table not found: ", dn, ": ", name)
		return []db.CalculateTableLayout{}, false
	}
	table := &meta.Table[idx]

	// calculate output table expression and comparison expression
	// for example: if table has Expr0 and Expr1 values and comparion is DIFF
	// then append to calculation: Expr0, Expr0[variant] - Expr0[base], Expr1, Expr1[variant] - Expr1[base]
	calcLt := []db.CalculateTableLayout{}

	for _, ex := range table.Expr {

		// append output table expression
		calcLt = append(calcLt, db.CalculateTableLayout{
			CalculateLayout: db.CalculateLayout{
				Calculate: ex.Name,
				CalcId:    ex.ExprId,
				Name:      ex.Name,
			},
			IsAggr: false,
		})

		// append comparison
		calcLt = append(calcLt, db.CalculateTableLayout{
			CalculateLayout: db.CalculateLayout{
				Calculate: fnc(ex.Name),
				CalcId:    ex.ExprId + db.CALCULATED_ID_OFFSET,
				Name:      cmp + "_" + ex.Name,
			},
			IsAggr: false,
		})
	}

	return calcLt, true
}

// set run id to digest maps for calculated cell converter.
// Function accept base run digest-or-stamp-or-name and optional list of variant runs digest-or-stamp-or-name.
// All runs must be completed successfully.
// Return base run id and optional list of run id's and Ok boolen flag.
func (mc *ModelCatalog) setRunIdDigestMap(digest string, rdsn string, runLst []string, cm *db.CalcMaps) (int, []int, bool) {

	// find model run id by digest-or-stamp-or-name
	r, ok := mc.CompletedRunByDigestOrStampOrName(digest, rdsn)
	if !ok {
		return 0, nil, false // return empty result: run select error
	}
	if r.Status != db.DoneRunStatus {
		omppLog.Log("Error: model run not completed successfully: ", rdsn, ": ", r.Status)
		return 0, nil, false
	}
	baseRunId := r.RunId // source run id

	cm.RunIdToLabel[r.RunId] = r.RunDigest // add base run digest to converter

	// check if all additional model runs completed successfully
	runIds := []int{}

	if len(runLst) > 0 {
		rLst, _ := mc.RunRowListByModel(digest)

		for _, rn := range runLst {

			rId := 0
			for k := 0; rId <= 0 && k < len(rLst); k++ {

				if rn == rLst[k].RunDigest || rn == rLst[k].RunStamp || rn == rLst[k].Name {
					rId = rLst[k].RunId
				}
				if rId > 0 {
					if rLst[k].Status != db.DoneRunStatus {
						omppLog.Log("Error: model run not completed successfully: ", rLst[k].RunDigest, ": ", rLst[k].Status)
						return 0, nil, false
					}
					cm.RunIdToLabel[rLst[k].RunId] = rLst[k].RunDigest // add run digest to converter
				}
			}
			if rId <= 0 {
				omppLog.Log("Warning: model run not found: ", rn)
				continue
			}
			// else: model run completed successfully, include run id into the list of additional runs

			isFound := false
			for k := 0; !isFound && k < len(runIds); k++ {
				isFound = rId == runIds[k]
			}
			if !isFound {
				runIds = append(runIds, rId)
			}
		}
	}

	return baseRunId, runIds, true
}

// MicrodataCellConverter return microdata value converter between code cell and id's cell.
// If isToId true then from code to id cell else other way around.
// Return model run id, entity generation digest, microdata value converter and boolean Ok flag.
func (mc *ModelCatalog) MicrodataCellConverter(
	isToId bool, dn string, rdsn string, name string,
) (
	int, string, func(interface{}) (interface{}, error), bool,
) {

	// validate parameters and return empty results on empty input
	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		return 0, "", nil, false
	}
	if rdsn == "" {
		omppLog.Log("Error: invalid (empty) model run digest, stamp and name")
		return 0, "", nil, false
	}
	if name == "" {
		omppLog.Log("Error: invalid (empty) model entity name")
		return 0, "", nil, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Error: model digest or name not found: ", dn)
		return 0, "", nil, false // return empty result: model not found or error
	}

	// get run_lst db row by digest, stamp or run name
	r, ok := mc.CompletedRunByDigestOrStampOrName(dn, rdsn)
	if !ok {
		return 0, "", nil, false // run not found or not completed
	}
	if r.Status != db.DoneRunStatus {
		omppLog.Log("Error: model run not completed successfully: ", rdsn, ": ", r.Status)
		return 0, "", nil, false
	}

	// find entity generation by entity name
	_, entGen, ok := mc.EntityGenByName(dn, r.RunId, name)
	if !ok {
		return r.RunId, "", nil, false // entity generation not found
	}

	// create converter
	cvtMicro := &db.CellMicroConverter{CellEntityConverter: db.CellEntityConverter{
		ModelDef:  meta,
		Name:      name,
		EntityGen: entGen,
		IsIdCsv:   isToId,
		DoubleFmt: theCfg.doubleFmt,
	}}
	var cvt func(interface{}) (interface{}, error)
	var err error

	if isToId {
		cvt, err = cvtMicro.CodeToIdCell(meta, name)
	} else {
		cvt, err = cvtMicro.IdToCodeCell(meta, name)
	}
	if err != nil {
		omppLog.Log("Failed to create microdata cell value converter: ", name, ": ", err.Error())
		return r.RunId, "", nil, false
	}

	return r.RunId, entGen.GenDigest, cvt, true
}

// ParameterToCsvConverter return csv header as string array, parameter csv converter and boolean Ok flag.
func (mc *ModelCatalog) ParameterToCsvConverter(dn string, isCode bool, name string) ([]string, func(interface{}, []string) (bool, error), bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		return []string{}, nil, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Error: model digest or name not found: ", dn)
		return []string{}, nil, false // return empty result: model not found or error
	}

	// check if parameter name exist in the model
	if _, ok = meta.ParamByName(name); !ok {
		omppLog.Log("Error: model parameter not found:", dn, ":", name)
		return []string{}, nil, false // return empty result: parameter not found or error
	}

	// make csv header
	csvCvt := db.CellParamConverter{
		ModelDef:  meta,
		Name:      name,
		IsIdCsv:   !isCode,
		DoubleFmt: theCfg.doubleFmt,
	}

	hdr, err := csvCvt.CsvHeader()
	if err != nil {
		omppLog.Log("Failed to make parameter csv header: ", dn, ": ", name, ": ", err.Error())
		return []string{}, nil, false
	}

	// create converter from db cell into csv row []string
	var cvt func(interface{}, []string) (bool, error)

	if isCode {
		cvt, err = csvCvt.ToCsvRow()
	} else {
		cvt, err = csvCvt.ToCsvIdRow()
	}
	if err != nil {
		omppLog.Log("Failed to create parameter converter to csv: ", dn, ": ", name, ": ", err.Error())
		return []string{}, nil, false
	}

	return hdr, cvt, true
}

// TableToCsvConverter return csv header as starting array, output table cell to csv converter and and boolean Ok flag.
func (mc *ModelCatalog) TableToCsvConverter(dn string, isCode bool, name string, isAcc, isAllAcc bool) ([]string, func(interface{}, []string) (bool, error), bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		return []string{}, nil, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Error: model digest or name not found: ", dn)
		return []string{}, nil, false // return empty result: model not found or error
	}

	// check if output table name exist in the model
	if _, ok = meta.OutTableByName(name); !ok {
		omppLog.Log("Error: model output table not found: ", dn, ": ", name)
		return []string{}, nil, false // return empty result: output table not found or error
	}

	// set cell conveter to csv
	ctc := db.CellTableConverter{
		ModelDef:  meta,
		Name:      name,
		IsIdCsv:   !isCode,
		DoubleFmt: theCfg.doubleFmt,
	}
	var csvCvt db.CsvConverter

	switch {
	case isAcc && isAllAcc:
		csvCvt = &db.CellAllAccConverter{
			CellTableConverter: ctc,
			ValueName:          "",
		}
	case isAcc:
		csvCvt = &db.CellAccConverter{
			CellTableConverter: ctc,
		}
	default:
		csvCvt = &db.CellExprConverter{
			CellTableConverter: ctc,
		}
	}

	// make csv header
	hdr, err := csvCvt.CsvHeader()
	if err != nil {
		omppLog.Log("Failed to make output table csv header: ", dn, ": ", name, ": ", err.Error())
		return []string{}, nil, false
	}

	// create converter from db cell into csv row []string
	var cvt func(interface{}, []string) (bool, error)

	if isCode {
		cvt, err = csvCvt.ToCsvRow()
	} else {
		cvt, err = csvCvt.ToCsvIdRow()
	}
	if err != nil {
		omppLog.Log("Failed to create output table converter to csv: ", dn, ": ", name, ": ", err.Error())
		return []string{}, nil, false
	}

	return hdr, cvt, true
}

// TableToCalcCsvConverter return csv header as starting array,  output table calculated value to csv converter and and boolean Ok flag.
// Function accept base run digest-or-stamp-or-name and optional list of variant runs digest-or-stamp-or-name.
// All runs must be completed successfully.
func (mc *ModelCatalog) TableToCalcCsvConverter(
	dn string, rdsn string, isCode bool, tableName string, calcLt []db.CalculateTableLayout, runLst []string,
) ([]string, func(interface{}, []string) (bool, error), int, []int, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		return []string{}, nil, 0, nil, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Error: model digest or name not found: ", dn)
		return []string{}, nil, 0, nil, false // return empty result: model not found or error
	}

	// check if output table name exist in the model
	if _, ok = meta.OutTableByName(tableName); !ok {
		omppLog.Log("Error: model output table not found: ", dn, ": ", tableName)
		return []string{}, nil, 0, nil, false // return empty result: output table not found or error
	}

	// create cell conveter to csv
	ctc := db.CellTableCalcConverter{
		CellTableConverter: db.CellTableConverter{
			ModelDef:  meta,
			Name:      tableName,
			IsIdCsv:   !isCode,
			DoubleFmt: theCfg.doubleFmt,
		},
		CalcMaps: db.EmptyCalcMaps(),
	}
	if e := ctc.SetCalcIdNameMap(calcLt); e != nil {
		omppLog.Log("Failed to create output table converter to csv: ", dn, ": ", tableName, ": ", e.Error())
		return []string{}, nil, 0, nil, false // return empty result: invalid calculation name or id
	}

	// validate all runs: it must be completed successfully
	// set run id's maps in the convereter
	baseRunId, runIds, ok := mc.setRunIdDigestMap(meta.Model.Digest, rdsn, runLst, &ctc.CalcMaps)
	if !ok {
		omppLog.Log("Failed to create output table converter to csv, invalid run digest or model run not completed: ", dn, ": ", tableName)
		return []string{}, nil, 0, nil, false // return empty result: output table not found or error
	}

	// make csv header
	hdr, err := ctc.CsvHeader()
	if err != nil {
		omppLog.Log("Failed to make output table csv header: ", dn, ": ", tableName, ": ", err.Error())
		return []string{}, nil, 0, nil, false
	}

	// create converter from db cell into csv row []string
	var cvt func(interface{}, []string) (bool, error)

	if isCode {
		cvt, err = ctc.ToCsvRow()
	} else {
		cvt, err = ctc.ToCsvIdRow()
	}
	if err != nil {
		omppLog.Log("Failed to create output table converter to csv: ", dn, ": ", tableName, ": ", err.Error())
		return []string{}, nil, 0, nil, false
	}

	return hdr, cvt, baseRunId, runIds, true
}

// MicrodataToCsvConverter return model run id, entity generation digest,
// csv header as starting array, microdata cell to csv converter and boolean Ok flag.
func (mc *ModelCatalog) MicrodataToCsvConverter(
	dn string, isCode bool, rdsn, name string,
) (
	int, string, []string, func(interface{}, []string) (bool, error), bool,
) {

	// validate parameters and return empty results on empty input
	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		return 0, "", []string{}, nil, false
	}
	if rdsn == "" {
		omppLog.Log("Error: invalid (empty) model run digest, stamp and name")
		return 0, "", []string{}, nil, false
	}
	if name == "" {
		omppLog.Log("Error: invalid (empty) model entity name")
		return 0, "", []string{}, nil, false
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Error: model digest or name not found: ", dn)
		return 0, "", []string{}, nil, false // return empty result: model not found or error
	}

	// get run_lst db row by digest, stamp or run name
	r, ok := mc.CompletedRunByDigestOrStampOrName(dn, rdsn)
	if !ok {
		return 0, "", []string{}, nil, false // run not found or not completed
	}
	if r.Status != db.DoneRunStatus {
		omppLog.Log("Error: model run not completed successfully: ", rdsn, ": ", r.Status)
		return r.RunId, "", []string{}, nil, false
	}

	// find entity generation by entity name
	_, entGen, ok := mc.EntityGenByName(dn, r.RunId, name)
	if !ok {
		return 0, "", []string{}, nil, false // entity generation not found
	}

	// make csv header
	cvtMicro := &db.CellMicroConverter{CellEntityConverter: db.CellEntityConverter{
		ModelDef:  meta,
		Name:      name,
		EntityGen: entGen,
		IsIdCsv:   !isCode,
		DoubleFmt: theCfg.doubleFmt,
	}}

	hdr, err := cvtMicro.CsvHeader()
	if err != nil {
		omppLog.Log("Failed to make microdata csv header: ", dn, ": ", name, ": ", err.Error())
		return r.RunId, "", []string{}, nil, false
	}

	// create converter from db cell into csv row []string
	var cvt func(interface{}, []string) (bool, error)

	if isCode {
		cvt, err = cvtMicro.ToCsvRow()
	} else {
		cvt, err = cvtMicro.ToCsvIdRow()
	}
	if err != nil {
		omppLog.Log("Failed to create microdata converter to csv: ", dn, ": ", name, ": ", err.Error())
		return r.RunId, "", []string{}, nil, false
	}

	return r.RunId, entGen.GenDigest, hdr, cvt, true
}

// MicrodataCalcToCsvConverter validate group by attributes return model run id, run id variants, entity generation digest
// csv header as starting array, microdata cell to csv converter or error.
func (mc *ModelCatalog) MicrodataCalcToCsvConverter(
	dn string, isCode bool, rdsn string, runLst []string, entityName string, calcLt *db.CalculateMicroLayout,
) (
	int, []int, string, *db.CellMicroCalcConverter, error,
) {

	// validate parameters and return empty results on empty input
	if dn == "" {
		return 0, []int{}, "", nil, errors.New("Error: invalid (empty) model digest and name")
	}
	if rdsn == "" {
		return 0, []int{}, "", nil, errors.New("Error: invalid (empty) model run digest, stamp and name")
	}
	if entityName == "" {
		return 0, []int{}, "", nil, errors.New("Error: invalid (empty) model entity name")
	}
	if len(calcLt.GroupBy) <= 0 {
		return 0, []int{}, "", nil, errors.New("Error: invalid (empty) microdata group by attributes: " + dn + ": " + entityName)
	}
	if len(calcLt.Calculation) <= 0 {
		return 0, []int{}, "", nil, errors.New("Error: invalid (empty) microdata calculation expression(s): " + dn + ": " + entityName)
	}

	// get model metadata and database connection
	meta, _, ok := mc.modelMeta(dn)
	if !ok {
		return 0, []int{}, "", nil,
			errors.New("Error: model digest or name not found: " + dn) // return empty result: model not found or error
	}

	// create cell conveter to csv
	cvtMicro := db.CellMicroCalcConverter{
		CellEntityConverter: db.CellEntityConverter{
			ModelDef:  meta,
			Name:      entityName,
			IsIdCsv:   !isCode,
			DoubleFmt: theCfg.doubleFmt,
		},
		CalcMaps: db.EmptyCalcMaps(),
		GroupBy:  calcLt.GroupBy,
	}
	if e := cvtMicro.SetCalcIdNameMap(calcLt.Calculation); e != nil {
		return 0, []int{}, "", nil,
			errors.New("Failed to create microdata aggregation converter to csv: " + dn + ": " + entityName + ": " + e.Error()) // return empty result: invalid calculation name or id
	}

	// validate all runs: it must be completed successfully
	// set run id's maps in the convereter
	baseRunId, runIds, ok := mc.setRunIdDigestMap(meta.Model.Digest, rdsn, runLst, &cvtMicro.CalcMaps)
	if !ok {
		return 0, []int{}, "", nil,
			errors.New("Failed to create microdata aggregation converter to csv, invalid run digest or model run not completed: " + dn + ": " + entityName) // return empty result: model run not found or not susscessful
	}

	// find entity generation by entity name and validate entity generation: it must exist for base run and all variant runs
	_, entGen, attrs, reLst, err := mc.EntityGenAttrsRunList(dn, baseRunId, entityName)
	if err != nil {
		return 0, []int{}, "", nil, err // entity generation not found
	}
	cvtMicro.EntityGen = entGen

	for k := 0; k < len(runIds); k++ {
		isFound := false
		for j := 0; !isFound && j < len(reLst); j++ {
			isFound = reLst[j].RunId == runIds[k]
		}
		if !isFound {
			return 0, []int{}, "", nil,
				errors.New("Failed to create microdata aggregation converter to csv, invalid run id: " + strconv.Itoa(runIds[k]) + ": " + dn + ": " + entityName) // return empty result: model run does not have this entity generation
		}
	}

	// validate group by attributes
	for k := 0; k < len(calcLt.GroupBy); k++ {

		isFound := false
		for j := 0; !isFound && j < len(attrs); j++ {
			isFound = attrs[j].Name == calcLt.GroupBy[k]
		}
		if !isFound {
			return 0, []int{}, "", nil, errors.New("Invalid group by attribute: " + entityName + "." + calcLt.GroupBy[k])
		}
	}

	return baseRunId, runIds, entGen.GenDigest, &cvtMicro, nil
}
