// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"encoding/csv"
	"net/http"
	"strconv"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// worksetParameterCsvGetHandler read a parameter values from workset and write it as csv response.
// GET /api/model/:model/workset/:set/parameter/:name/csv
// Dimension(s) and enum-based parameters returned as enum codes.
func worksetParameterCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doParameterGetCsvHandler(w, r, "set", true, true, false)
}

// worksetParameterCsvBomGetHandler read a parameter values from workset and write it as csv response.
// GET /api/model/:model/workset/:set/parameter/:name/csv-bom
// Dimension(s) and enum-based parameters returned as enum codes.
// Response starts from utf-8 BOM bytes.
func worksetParameterCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doParameterGetCsvHandler(w, r, "set", true, true, true)
}

// worksetParameterIdCsvGetHandler read a parameter values from workset and write it as csv response.
// GET /api/model/:model/workset/:set/parameter/:name/csv-id
// Dimension(s) and enum-based parameters returned as enum id's.
func worksetParameterIdCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doParameterGetCsvHandler(w, r, "set", true, false, false)
}

// worksetParameterIdCsvBomGetHandler read a parameter values from workset and write it as csv response.
// GET /api/model/:model/workset/:set/parameter/:name/csv-id-bom
// Dimension(s) and enum-based parameters returned as enum id's.
// Response starts from utf-8 BOM bytes.
func worksetParameterIdCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doParameterGetCsvHandler(w, r, "set", true, false, true)
}

// runParameterCsvGetHandler read a parameter values from model run results and write it as csv response.
// GET /api/model/:model/run/:run/parameter/:name/csv
// Dimension(s) and enum-based parameters returned as enum codes.
func runParameterCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doParameterGetCsvHandler(w, r, "run", false, true, false)
}

// runParameterCsvBomGetHandler read a parameter values from model run results and write it as csv response.
// GET /api/model/:model/run/:run/parameter/:name/csv-bom
// Dimension(s) and enum-based parameters returned as enum codes.
// Response starts from utf-8 BOM bytes.
func runParameterCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doParameterGetCsvHandler(w, r, "run", false, true, true)
}

// runParameterIdCsvGetHandler read a parameter values from model run results and write it as csv response.
// GET /api/model/:model/run/:run/parameter/:name/csv-id
// Dimension(s) and enum-based parameters returned as enum id's.
func runParameterIdCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doParameterGetCsvHandler(w, r, "run", false, false, false)
}

// runParameterIdCsvBomGetHandler read a parameter values from model run results and write it as csv response.
// GET /api/model/:model/run/:run/parameter/:name/csv-id-bom
// Dimension(s) and enum-based parameters returned as enum id's.
// Response starts from utf-8 BOM bytes.
func runParameterIdCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doParameterGetCsvHandler(w, r, "run", false, false, true)
}

// doParameterGetCsvHandler read parameter values from workset or model run and write it as csv response.
// It does read all parameter values, not a "page" of values.
// Dimension(s) and enum-based parameters returned as enum codes or enum id's.
func doParameterGetCsvHandler(w http.ResponseWriter, r *http.Request, srcArg string, isSet, isCode, isBom bool) {

	// url or query parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	src := getRequestParam(r, srcArg)  // workset name or run digest-or-stamp-or-name
	name := getRequestParam(r, "name") // parameter name
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// read parameter values, page size =0: read all values
	layout := db.ReadParamLayout{
		ReadLayout: db.ReadLayout{Name: name}, IsFromSet: isSet,
	}

	// get converter from cell list to csv rows []string
	hdr, cvtRow, ok := theCatalog.ParameterToCsvConverter(dn, isCode, name)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Failed to create parameter csv converter", src, ":", name), http.StatusBadRequest)
		return
	}

	// set response headers: Content-Disposition: attachment; filename=name.csv
	csvSetHeaders(w, name)

	// write csv body
	if isBom {
		if _, err := w.Write(helper.Utf8bom); err != nil {
			http.Error(w, helper.MsgL(lang, "Error at csv write:", src, ":", name), http.StatusBadRequest)
			return
		}
	}

	csvWr := csv.NewWriter(w)

	if err := csvWr.Write(hdr); err != nil {
		http.Error(w, helper.MsgL(lang, "Error at csv write:", src, ":", name), http.StatusBadRequest)
		return
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

	_, ok = theCatalog.ReadParameterTo(dn, src, &layout, cvtWr)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at parameter read:", src, ":", name), http.StatusBadRequest)
		return
	}
	csvWr.Flush() // flush csv to response
}

// runTableExprCsvGetHandler read table expression(s) values from model run results and write it as csv response.
// GET /api/model/:model/run/:run/table/:name/expr/csv
// Dimension(s) returned as enum codes.
func runTableExprCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableGetCsvHandler(w, r, false, false, true, false)
}

// runTableExprCsvBomGetHandler read table expression(s) values from model run results and write it as csv response.
// GET /api/model/:model/run/:run/table/:name/expr/csv-bom
// Dimension(s) returned as enum codes.
// Response starts from utf-8 BOM bytes.
func runTableExprCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableGetCsvHandler(w, r, false, false, true, true)
}

// runTableExprIdCsvGetHandler read table expression(s) values from model run results and write it as csv response.
// GET /api/model/:model/run/:run/table/:name/expr/csv-id
// Dimension(s) returned as enum id's.
func runTableExprIdCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableGetCsvHandler(w, r, false, false, false, false)
}

// runTableExprIdCsvBomGetHandler read table expression(s) values from model run results and write it as csv response.
// GET /api/model/:model/run/:run/table/:name/expr/csv-id-bom
// Dimension(s) returned as enum id's.
// Response starts from utf-8 BOM bytes.
func runTableExprIdCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableGetCsvHandler(w, r, false, false, false, true)
}

// runTableAccCsvGetHandler read table accumultor(s) values from model run results and write it as csv response.
// GET /api/model/:model/run/:run/table/:name/acc/csv
// Dimension(s) returned as enum codes.
func runTableAccCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableGetCsvHandler(w, r, true, false, true, false)
}

// runTableAccCsvBomGetHandler read table accumultor(s) values from model run results and write it as csv response.
// GET /api/model/:model/run/:run/table/:name/acc/csv-bom
// Dimension(s) returned as enum codes.
// Response starts from utf-8 BOM bytes.
func runTableAccCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableGetCsvHandler(w, r, true, false, true, true)
}

// runTableAccIdCsvGetHandler read table accumultor(s) values from model run results and write it as csv response.
// GET /api/model/:model/run/:run/table/:name/acc/csv-id
// Dimension(s) returned as enum id's.
func runTableAccIdCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableGetCsvHandler(w, r, true, false, false, false)
}

// runTableAccIdCsvBomGetHandler read table accumultor(s) values from model run results and write it as csv response.
// GET /api/model/:model/run/:run/table/:name/acc/csv-id-bom
// Dimension(s) returned as enum id's.
// Response starts from utf-8 BOM bytes.
func runTableAccIdCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableGetCsvHandler(w, r, true, false, false, true)
}

// runTableAllAccCsvGetHandler read table "all-accumulators" values
// from model run results and write it as csv response.
// GET /api/model/:model/run/:run/table/:name/all-acc/csv
// Dimension(s) returned as enum codes.
func runTableAllAccCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableGetCsvHandler(w, r, true, true, true, false)
}

// runTableAllAccCsvBomGetHandler read table "all-accumulators" values
// from model run results and write it as csv response.
// GET /api/model/:model/run/:run/table/:name/all-acc/csv-bom
// Dimension(s) returned as enum codes.
// Response starts from utf-8 BOM bytes.
func runTableAllAccCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableGetCsvHandler(w, r, true, true, true, true)
}

// runTableAllAccIdCsvGetHandler read table "all-accumulators" values
// from model run results and write it as csv response.
// GET /api/model/:model/run/:run/table/:name/all-acc/csv-id
// Dimension(s) returned as enum id's.
func runTableAllAccIdCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableGetCsvHandler(w, r, true, true, false, false)
}

// runTableAllAccIdCsvBomGetHandler read table "all-accumulators" values
// from model run results and write it as csv response.
// GET /api/model/:model/run/:run/table/:name/all-acc/csv-id-bom
// Dimension(s) returned as enum id's.
// Response starts from utf-8 BOM bytes.
func runTableAllAccIdCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableGetCsvHandler(w, r, true, true, false, true)
}

// doTableGetCsvHandler read output table expression, accumulator or "all-accumulator" values
// from model run and write it as csv response.
// It does read all output table values, not a "page" of values.
// Dimension(s) and enum-based parameters returned as enum codes or enum id's.
func doTableGetCsvHandler(w http.ResponseWriter, r *http.Request, isAcc, isAllAcc, isCode, isBom bool) {

	// url or query parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	rdsn := getRequestParam(r, "run")  // run digest-or-stamp-or-name
	name := getRequestParam(r, "name") // output table name
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// read output table values, page size =0: read all values
	layout := db.ReadTableLayout{
		ReadLayout: db.ReadLayout{Name: name},
		IsAccum:    isAcc,
		IsAllAccum: isAllAcc,
	}

	// get converter from cell list to csv rows []string
	hdr, cvtRow, ok := theCatalog.TableToCsvConverter(dn, isCode, name, layout.IsAccum, layout.IsAllAccum)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Failed to create output table csv converter:", name), http.StatusBadRequest)
		return
	}

	// set response headers: Content-Disposition: attachment; filename=name.csv
	fn := name
	if isAcc {
		if isAllAcc {
			fn += ".acc-all"
		} else {
			fn += ".acc"
		}
	}
	csvSetHeaders(w, fn)

	// write csv body
	if isBom {
		if _, err := w.Write(helper.Utf8bom); err != nil {
			http.Error(w, helper.MsgL(lang, "Error at csv write:", rdsn, ":", name), http.StatusBadRequest)
			return
		}
	}

	csvWr := csv.NewWriter(w)

	if err := csvWr.Write(hdr); err != nil {
		http.Error(w, helper.MsgL(lang, "Error at csv write:", rdsn, ":", name), http.StatusBadRequest)
		return
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

	_, ok = theCatalog.ReadOutTableTo(dn, rdsn, &layout, cvtWr)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at run output table read:", rdsn, ":", name), http.StatusBadRequest)
		return
	}
	csvWr.Flush() // flush csv to response
}

// runTableCalcCsvGetHandler write into CSV response all output table expressions
// and for each expression additional measure: SUM AVG COUNT MIN MAX VAR SD SE CV.
// GET /api/model/:model/run/:run/table/:name/calc/:calc/csv
// Dimension(s) returned as enum codes.
func runTableCalcCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableCalcGetCsvHandler(w, r, true, false)
}

// runTableCalcCsvBomGetHandler write into CSV response all output table expressions
// and for each expression additional measure: SUM AVG COUNT MIN MAX VAR SD SE CV.
// GET /api/model/:model/run/:run/table/:name/calc/:calc/csv-bom
// Dimension(s) returned as enum codes.
// Response starts from utf-8 BOM bytes.
func runTableCalcCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableCalcGetCsvHandler(w, r, true, true)
}

// runTableCalcIdCsvGetHandler write into CSV response all output table expressions
// and for each expression additional measure: SUM AVG COUNT MIN MAX VAR SD SE CV.
// GET /api/model/:model/run/:run/table/:name/calc/:calc/csv-id
// Dimension(s) returned as enum id's.
func runTableCalcIdCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableCalcGetCsvHandler(w, r, false, false)
}

// runTableCalcIdCsvBomGetHandler write into CSV response all output table expressions
// and for each expression additional measure: SUM AVG COUNT MIN MAX VAR SD SE CV.
// GET /api/model/:model/run/:run/table/:name/calc/:calc/csv-id-bom
// Dimension(s) returned as enum id's.
// Response starts from utf-8 BOM bytes.
func runTableCalcIdCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableCalcGetCsvHandler(w, r, false, true)
}

// doTableCalcGetCsvHandler write into CSV response all output table expressions
// and for each expression additional measure: SUM AVG COUNT MIN MAX VAR SD SE CV.
// It does read all output table values, not a "page" of values.
// Dimension(s) and enum-based parameters returned as enum codes or enum id's.
func doTableCalcGetCsvHandler(w http.ResponseWriter, r *http.Request, isCode, isBom bool) {

	// url or query parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	rdsn := getRequestParam(r, "run")  // run digest-or-stamp-or-name
	name := getRequestParam(r, "name") // output table name
	calc := getRequestParam(r, "calc") // calculation function name: sum avg count min max var sd se cv
	lang := preferedRequestLang(r, "") // get prefered language for messages

	if calc == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) calculation expression", calc), http.StatusBadRequest)
		return
	}

	// setup read layout and calculate layout
	// page size =0, read all values
	tableLt := db.ReadTableLayout{
		ReadLayout: db.ReadLayout{Name: name},
	}

	calcLt, ok := theCatalog.TableAggrExprCalculateLayout(dn, name, calc)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid calculation expression", calc), http.StatusBadRequest)
		return
	}

	// get converter from cell list to csv rows []string
	hdr, cvtRow, _, runIds, ok := theCatalog.TableToCalcCsvConverter(dn, rdsn, isCode, name, calcLt, nil)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Failed to create output table csv converter:", name), http.StatusBadRequest)
		return
	}

	// set response headers: Content-Disposition: attachment; filename=name.csv
	csvSetHeaders(w, name)

	// write csv body
	if isBom {
		if _, err := w.Write(helper.Utf8bom); err != nil {
			http.Error(w, helper.MsgL(lang, "Error at csv write:", rdsn, ":", name), http.StatusBadRequest)
			return
		}
	}

	csvWr := csv.NewWriter(w)

	if err := csvWr.Write(hdr); err != nil {
		http.Error(w, helper.MsgL(lang, "Error at csv write:", rdsn, ":", name), http.StatusBadRequest)
		return
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

	_, ok = theCatalog.ReadOutTableCalculateTo(dn, rdsn, &tableLt, calcLt, runIds, cvtWr)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at run output table read:", rdsn, ":", name), http.StatusBadRequest)
		return
	}
	csvWr.Flush() // flush csv to response
}

// runTableCompareCsvGetHandler write into CSV response output table comparison between base and variant model runs.
// It is either calculation for each expression: DIFF RATIO PERCENT or multiple arbitrary calculations.
// For example, RATIO is: expr0[variant] / expr0[base], expr1[variant] / expr1[base],....
// Or arbitrary comma separated expression(s): expr0 , expr1[variant] + expr2[base] , ....
// Variant runs can be a comma separated list of run digests or run stamps or run names.
// If run name contains comma then name must be "double quoted" or 'single quoted'.
// For example: "Year 1995, 1996", 'Age [30, 40]'
// GET /api/model/:model/run/:run/table/:name/compare/:compare/variant/:variant/csv
// Dimension(s) returned as enum codes.
func runTableCompareCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableCompareGetCsvHandler(w, r, true, false)
}

// runTableCompareCsvBomGetHandler write into CSV response output table comparison between base and variant model runs.
// It is either calculation for each expression: DIFF RATIO PERCENT or multiple arbitrary calculations.
// For example, RATIO is: expr0[variant] / expr0[base], expr1[variant] / expr1[base],....
// Or arbitrary comma separated expression(s): expr0 , expr1[variant] + expr2[base] , ....
// Variant runs can be a comma separated list of run digests or run stamps or run names.
// If run name contains comma then name must be "double quoted" or 'single quoted'.
// For example: "Year 1995, 1996", 'Age [30, 40]'
// GET /api/model/:model/run/:run/table/:name/compare/:compare/variant/:variant/csv-bom
// Dimension(s) returned as enum codes.
// Response starts from utf-8 BOM bytes.
func runTableCompareCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableCompareGetCsvHandler(w, r, true, true)
}

// runTableCompareIdCsvGetHandler write into CSV response output table comparison between base and variant model runs.
// It is either calculation for each expression: DIFF RATIO PERCENT or multiple arbitrary calculations.
// For example, RATIO is: expr0[variant] / expr0[base], expr1[variant] / expr1[base],....
// Or arbitrary comma separated expression(s): expr0 , expr1[variant] + expr2[base] , ....
// Variant runs can be a comma separated list of run digests or run stamps or run names.
// If run name contains comma then name must be "double quoted" or 'single quoted'.
// For example: "Year 1995, 1996", 'Age [30, 40]'
// GET /api/model/:model/run/:run/table/:name/compare/:compare/variant/:variant/csv-id
// Dimension(s) returned as enum id's.
func runTableCompareIdCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableCompareGetCsvHandler(w, r, false, false)
}

// runTableCompareIdCsvBomGetHandler write into CSV response output table comparison between base and variant model runs.
// It is either calculation for each expression: DIFF RATIO PERCENT or multiple arbitrary calculations.
// For example, RATIO is: expr0[variant] / expr0[base], expr1[variant] / expr1[base],....
// Or arbitrary comma separated expression(s): expr0 , expr1[variant] + expr2[base] , ....
// Variant runs can be a comma separated list of run digests or run stamps or run names.
// If run name contains comma then name must be "double quoted" or 'single quoted'.
// For example: "Year 1995, 1996", 'Age [30, 40]'
// GET /api/model/:model/run/:run/table/:name/compare/:compare/variant/:variant/csv-id-bom
// Dimension(s) returned as enum id's.
// Response starts from utf-8 BOM bytes.
func runTableCompareIdCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableCompareGetCsvHandler(w, r, false, true)
}

// doTableCompareGetCsvHandler write into CSV response output table comparison between base and variant model runs.
// It is either calculation for each expression: DIFF RATIO PERCENT or multiple arbitrary calculations.
// For example, RATIO is: expr0[variant] / expr0[base], expr1[variant] / expr1[base],....
// Or arbitrary comma separated expression(s): expr0 , 7 + expr1[variant] + expr2[base] , ....
// Variant runs can be a comma separated list of run digests or run stamps or run names.
// If run name contains comma then name must be "double quoted" or 'single quoted'.
// For example: "Year 1995, 1996", 'Age [30, 40]'
// It does read all output table values, not a "page" of values.
// Dimension(s) and enum-based parameters returned as enum codes if isCode is true or enum id's.
func doTableCompareGetCsvHandler(w http.ResponseWriter, r *http.Request, isCode, isBom bool) {

	// url or query parameters
	dn := getRequestParam(r, "model")        // model digest-or-name
	rdsn := getRequestParam(r, "run")        // base run digest-or-stamp-or-name
	name := getRequestParam(r, "name")       // output table name
	compare := getRequestParam(r, "compare") // comparison function name: diff ratio percent
	vr := getRequestParam(r, "variant")      // comma separated list of variant runs digest-or-stamp-or-name
	lang := preferedRequestLang(r, "")       // get prefered language for messages

	if compare == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) comparison expression"), http.StatusBadRequest)
		return
	}
	vRdsn := helper.ParseCsvLine(vr, 0)
	if len(vRdsn) <= 0 {
		http.Error(w, helper.MsgL(lang, "Invalid or empty list of runs to compare"), http.StatusBadRequest)
		return
	}

	// setup read layout and calculate layout
	// page size =0, read all values
	tableLt := db.ReadTableLayout{
		ReadLayout: db.ReadLayout{Name: name},
	}

	calcLt, ok := theCatalog.TableExprCompareLayout(dn, name, compare)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid comparison expression", compare), http.StatusBadRequest)
		return
	}

	// get converter from cell list to csv rows []string
	hdr, cvtRow, _, runIds, ok := theCatalog.TableToCalcCsvConverter(dn, rdsn, isCode, name, calcLt, vRdsn)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Failed to create output table csv converter:", name), http.StatusBadRequest)
		return
	}

	// set response headers: Content-Disposition: attachment; filename=name.csv
	csvSetHeaders(w, name)

	// write csv body
	if isBom {
		if _, err := w.Write(helper.Utf8bom); err != nil {
			http.Error(w, helper.MsgL(lang, "Error at csv write:", rdsn, ":", name), http.StatusBadRequest)
			return
		}
	}

	csvWr := csv.NewWriter(w)

	if err := csvWr.Write(hdr); err != nil {
		http.Error(w, helper.MsgL(lang, "Error at csv write:", rdsn, ":", name), http.StatusBadRequest)
		return
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

	_, ok = theCatalog.ReadOutTableCalculateTo(dn, rdsn, &tableLt, calcLt, runIds, cvtWr)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at run output table read:", rdsn, ":", name), http.StatusBadRequest)
		return
	}
	csvWr.Flush() // flush csv to response
}

// runMicrodataCsvGetHandler read a microdata values from model run results and write it as csv response.
// GET /api/model/:model/run/:run/microdata/:name/csv
// Enum-based microdata attributes returned as enum codes.
func runMicrodataCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doMicrodataGetCsvHandler(w, r, true, false)
}

// runMicrodataCsvBomGetHandler read a microdata values from model run results and write it as csv response.
// GET /api/model/:model/run/:run/microdata/:name/csv-bom
// Enum-based microdata attributes returned as enum codes.
// Response starts from utf-8 BOM bytes.
func runMicrodataCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doMicrodataGetCsvHandler(w, r, true, true)
}

// runMicrodataIdCsvGetHandler read a microdata values from model run results and write it as csv response.
// GET /api/model/:model/run/:run/microdata/:name/csv-id
// Enum-based microdata attributes returned as enum id's.
func runMicrodataIdCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doMicrodataGetCsvHandler(w, r, false, false)
}

// runMicrodataIdCsvBomGetHandler read a microdata values from model run results and write it as csv response.
// GET /api/model/:model/run/:run/microdata/:name/csv-id-bom
// Enum-based microdata attributes returned as enum id's.
// Response starts from utf-8 BOM bytes.
func runMicrodataIdCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doMicrodataGetCsvHandler(w, r, false, true)
}

// doMicrodataGetCsvHandler read microdata values from model run and write it as csv response.
// It does read all microdata values, not a "page" of values.
// Enum-based microdata attributes returned as enum codes or enum id's.
func doMicrodataGetCsvHandler(w http.ResponseWriter, r *http.Request, isCode, isBom bool) {

	// url or query parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	rdsn := getRequestParam(r, "run")  // run digest-or-stamp-or-name
	name := getRequestParam(r, "name") // entity name
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// return error if microdata disabled
	if !theCfg.isMicrodata {
		http.Error(w, helper.MsgL(lang, "Error: microdata not allowed:", dn, rdsn), http.StatusBadRequest)
		return
	}

	// get converter from cell list to csv rows []string
	runId, genDigest, hdr, cvtRow, ok := theCatalog.MicrodataToCsvConverter(dn, isCode, rdsn, name)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Failed to create microdata csv converter:", rdsn, ":", name), http.StatusBadRequest)
		return
	}

	// read microdata values, page size =0: read all values
	layout := db.ReadMicroLayout{
		ReadLayout: db.ReadLayout{
			Name:   name,
			FromId: runId,
		},
		GenDigest: genDigest,
	}

	// set response headers: Content-Disposition: attachment; filename=name.csv
	csvSetHeaders(w, name)

	// write csv body
	if isBom {
		if _, err := w.Write(helper.Utf8bom); err != nil {
			http.Error(w, helper.MsgL(lang, "Error at csv write:", rdsn, ":", name), http.StatusBadRequest)
			return
		}
	}

	csvWr := csv.NewWriter(w)

	if err := csvWr.Write(hdr); err != nil {
		http.Error(w, helper.MsgL(lang, "Error at csv write:", rdsn, ":", name), http.StatusBadRequest)
		return
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

	_, ok = theCatalog.ReadMicrodataTo(dn, rdsn, &layout, cvtWr)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at microdata read:", rdsn, ":", name), http.StatusBadRequest)
		return
	}
	csvWr.Flush() // flush csv to response
}

// runMicrodataCalcCsvGetHandler aggregate microdata values and write it into csv response.
// GET /api/model/:model/run/:run/microdata/:name/group-by/:group-by/calc/:calc/csv
// It can be multiple aggregations of value attributes (float of integer type), group by dimension attributes (enum-based or bool type).
// For example: microdata/Person/group-by/AgeGroup,Sex/calc/OM_AVG(Income),OM_MAX(Salary+Pension)/csv
// If run name contains comma then name must be "double quoted" or 'single quoted'.
// For example: "Year 1995, 1996" or: 'Age [30, 40]'
// Enum-based microdata attributes returned as enum codes.
func runMicrodataCalcCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doMicrodataCalcGetCsvHandler(w, r, true, false, "calc", false)
}

// runMicrodataCalcCsvBomGetHandler aggregate microdata values and write it into csv response.
// GET /api/model/:model/run/:run/microdata/:name/group-by/:group-by/calc/:calc/csv-bom
// It can be multiple aggregations of value attributes (float of integer type), group by dimension attributes (enum-based or bool type).
// For example: microdata/Person/group-by/AgeGroup,Sex/calc/OM_AVG(Income),OM_MAX(Salary+Pension)/csv
// If run name contains comma then name must be "double quoted" or 'single quoted'.
// For example: "Year 1995, 1996" or: 'Age [30, 40]'
// Enum-based microdata attributes returned as enum codes.
// Response starts from utf-8 BOM bytes.
func runMicrodataCalcCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doMicrodataCalcGetCsvHandler(w, r, true, true, "calc", false)
}

// runMicrodataCalcIdCsvGetHandler aggregate microdata values and write it into csv response.
// GET /api/model/:model/run/:run/microdata/:name/group-by/:group-by/calc/:calc/csv-id
// It can be multiple aggregations of value attributes (float of integer type), group by dimension attributes (enum-based or bool type).
// For example: microdata/Person/group-by/AgeGroup,Sex/calc/OM_AVG(Income),OM_MAX(Salary+Pension)/csv
// If run name contains comma then name must be "double quoted" or 'single quoted'.
// For example: "Year 1995, 1996" or: 'Age [30, 40]'
// Enum-based microdata attributes returned as enum id's.
func runMicrodataCalcIdCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doMicrodataCalcGetCsvHandler(w, r, false, false, "calc", false)
}

// runMicrodataCalcIdCsvBomGetHandler aggregate microdata values and write it into csv response.
// GET /api/model/:model/run/:run/microdata/:name/group-by/:group-by/calc/:calc/csv-id-bom
// It can be multiple aggregations of value attributes (float of integer type), group by dimension attributes (enum-based or bool type).
// For example: microdata/Person/group-by/AgeGroup,Sex/calc/OM_AVG(Income),OM_MAX(Salary+Pension)/csv
// If run name contains comma then name must be "double quoted" or 'single quoted'.
// For example: "Year 1995, 1996" or: 'Age [30, 40]'
// Enum-based microdata attributes returned as enum id's.
// Response starts from utf-8 BOM bytes.
func runMicrodataCalcIdCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doMicrodataCalcGetCsvHandler(w, r, false, true, "calc", false)
}

// doMicrodataCalcGetCsvHandler aggregate microdata values and write it into csv response.
// It can be multiple aggregations of value attributes (float of integer type), group by dimension attributes (enum-based or bool type).
// For example: microdata/Person/group-by/AgeGroup,Sex/calc/OM_AVG(Income),OM_MAX(Salary+Pension)/csv
// If run name contains comma then name must be "double quoted" or 'single quoted'.
// For example: "Year 1995, 1996" or: 'Age [30, 40]'
// Enum-based attributes returned as enum codes or enum id's.
func doMicrodataCalcGetCsvHandler(w http.ResponseWriter, r *http.Request, isCode, isBom bool, calcKey string, isVar bool) {

	// url or query parameters
	dn := getRequestParam(r, "model")                                 // model digest-or-name
	rdsn := getRequestParam(r, "run")                                 // run digest-or-stamp-or-name
	name := getRequestParam(r, "name")                                // entity name
	groupBy := helper.ParseCsvLine(getRequestParam(r, "group-by"), 0) // group by list of attributes, comma-separated
	cLst := helper.ParseCsvLine(getRequestParam(r, calcKey), 0)       // list of aggregations or comparisons, comma-separated
	lang := preferedRequestLang(r, "")                                // get prefered language for messages

	// return error if microdata disabled
	if !theCfg.isMicrodata {
		http.Error(w, helper.MsgL(lang, "Error: microdata not allowed:", dn, rdsn), http.StatusBadRequest)
		return
	}

	if len(groupBy) <= 0 {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) microdata group by attributes", name), http.StatusBadRequest)
		return
	}
	if len(cLst) <= 0 {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) microdata aggregation(s) and comparison(s)", name), http.StatusBadRequest)
		return
	}

	varLst := []string{}
	if isVar {
		varLst = helper.ParseCsvLine(getRequestParam(r, "variant"), 0) // list of aggregations or comparisons, comma-separated
	}

	// set aggregation expressions
	calcLt := db.CalculateMicroLayout{
		Calculation: []db.CalculateLayout{},
		GroupBy:     groupBy,
	}

	for j := range cLst {

		if cLst[j] != "" {
			calcLt.Calculation = append(calcLt.Calculation, db.CalculateLayout{
				Calculate: cLst[j],
				CalcId:    j + db.CALCULATED_ID_OFFSET,
				Name:      "ex_" + strconv.Itoa(j+db.CALCULATED_ID_OFFSET),
			})
		}
	}

	// get base run id, run variants, entity generation digest and microdata cell converter
	baseRunId, runIds, genDigest, cvtMicro, err := theCatalog.MicrodataCalcToCsvConverter(dn, isCode, rdsn, varLst, name, &calcLt)
	if err != nil {
		omppLog.Log("Failed to create microdata csv converter: ", rdsn, ":", name, ":", err)
		http.Error(w, helper.MsgL(lang, "Failed to create microdata csv converter:", rdsn, ":", name), http.StatusBadRequest)
		return
	}

	// read microdata values, page size =0: read all values
	microLt := db.ReadMicroLayout{
		ReadLayout: db.ReadLayout{
			Name:   name,
			FromId: baseRunId,
		},
		GenDigest: genDigest,
	}

	// make csv header
	hdr, err := cvtMicro.CsvHeader()
	if err != nil {
		omppLog.Log("Failed to make microdata csv header: ", dn, ":", name, ":", err)
		http.Error(w, helper.MsgL(lang, "Failed to create microdata csv converter:", rdsn, ":", name), http.StatusBadRequest)
		return
	}

	// create converter from db cell into csv row []string
	var cvtRow func(interface{}, []string) (bool, error)

	if isCode {
		cvtRow, err = cvtMicro.ToCsvRow()
	} else {
		cvtRow, err = cvtMicro.ToCsvIdRow()
	}
	if err != nil {
		omppLog.Log("Failed to create microdata converter to csv: ", dn, ":", name, ":", err)
		http.Error(w, helper.MsgL(lang, "Failed to create microdata csv converter:", rdsn, ":", name), http.StatusBadRequest)
		return
	}

	// set response headers: Content-Disposition: attachment; filename=name.csv
	csvSetHeaders(w, name)

	// write csv body
	if isBom {
		if _, err := w.Write(helper.Utf8bom); err != nil {
			omppLog.Log("Error at csv write: ", dn, ":", name, ":", err)
			http.Error(w, helper.MsgL(lang, "Error at csv write:", rdsn, ":", name), http.StatusBadRequest)
			return
		}
	}

	csvWr := csv.NewWriter(w)

	if err := csvWr.Write(hdr); err != nil {
		omppLog.Log("Error at csv write: ", dn, ":", name, ":", err)
		http.Error(w, helper.MsgL(lang, "Error at csv write:", rdsn, ":", name), http.StatusBadRequest)
		return
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

	_, ok := theCatalog.ReadMicrodataCalculateTo(dn, rdsn, &microLt, &calcLt, runIds, cvtWr)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at microdata aggregation read", rdsn, ":", name), http.StatusBadRequest)
		return
	}
	csvWr.Flush() // flush csv to response
}

// runMicrodataCompareCsvGetHandler write into CSV response microdata comparison between base and variant(s) model runs.
// GET /api/model/:model/run/:run/microdata/:name/group-by/:group-by/compare/:compare/variant/:variant/csv
// It can be multiple comparison of value attributes (float of integer type) and / or aggregation of value attributes.
// All comparisons and aggregations grouped by dimension attributes (enum-based or bool type).
// For example: microdata/Person/group-by/AgeGroup,Sex/compare/OM_AVG(Income[variant]-Income[base]),OM_MAX(Salary+Pension)
// If run name contains comma then name must be "double quoted" or 'single quoted'.
// For example: "Year 1995, 1996" or: 'Age [30, 40]'
// Enum-based microdata attributes returned as enum id's.
func runMicrodataCompareCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doMicrodataCalcGetCsvHandler(w, r, true, false, "compare", true)
}

// runMicrodataCompareCsvGetHandler write into CSV response microdata comparison between base and variant(s) model runs.
// GET /api/model/:model/run/:run/microdata/:name/group-by/:group-by/compare/:compare/variant/:variant/csv-bom
// It can be multiple comparison of value attributes (float of integer type) and / or aggregation of value attributes.
// All comparisons and aggregations grouped by dimension attributes (enum-based or bool type).
// For example: microdata/Person/group-by/AgeGroup,Sex/compare/OM_AVG(Income[variant]-Income[base]),OM_MAX(Salary+Pension)
// If run name contains comma then name must be "double quoted" or 'single quoted'.
// For example: "Year 1995, 1996" or: 'Age [30, 40]'
// Enum-based microdata attributes returned as enum id's.
func runMicrodataCompareCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doMicrodataCalcGetCsvHandler(w, r, true, true, "compare", true)
}

// runMicrodataCompareCsvGetHandler write into CSV response microdata comparison between base and variant(s) model runs.
// GET /api/model/:model/run/:run/microdata/:name/group-by/:group-by/compare/:compare/variant/:variant/csv-id
// It can be multiple comparison of value attributes (float of integer type) and / or aggregation of value attributes.
// All comparisons and aggregations grouped by dimension attributes (enum-based or bool type).
// For example: microdata/Person/group-by/AgeGroup,Sex/compare/OM_AVG(Income[variant]-Income[base]),OM_MAX(Salary+Pension)
// If run name contains comma then name must be "double quoted" or 'single quoted'.
// For example: "Year 1995, 1996" or: 'Age [30, 40]'
// Enum-based microdata attributes returned as enum id's.
func runMicrodataCompareIdCsvGetHandler(w http.ResponseWriter, r *http.Request) {
	doMicrodataCalcGetCsvHandler(w, r, false, false, "compare", true)
}

// runMicrodataCompareCsvGetHandler write into CSV response microdata comparison between base and variant(s) model runs.
// GET /api/model/:model/run/:run/microdata/:name/group-by/:group-by/compare/:compare/variant/:variant/csv-id-bom
// It can be multiple comparison of value attributes (float of integer type) and / or aggregation of value attributes.
// All comparisons and aggregations grouped by dimension attributes (enum-based or bool type).
// For example: microdata/Person/group-by/AgeGroup,Sex/compare/OM_AVG(Income[variant]-Income[base]),OM_MAX(Salary+Pension)
// If run name contains comma then name must be "double quoted" or 'single quoted'.
// For example: "Year 1995, 1996" or: 'Age [30, 40]'
// Enum-based microdata attributes returned as enum id's.
func runMicrodataCompareIdCsvBomGetHandler(w http.ResponseWriter, r *http.Request) {
	doMicrodataCalcGetCsvHandler(w, r, false, true, "compare", true)
}
