// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// worksetParameterPageReadHandler read a "page" of parameter values from workset.
// POST /api/model/:model/workset/:set/parameter/value
// Dimension(s) and enum-based parameters returned as enum codes.
func worksetParameterPageReadHandler(w http.ResponseWriter, r *http.Request) {
	doReadParameterPageHandler(w, r, "set", true, true)
}

// worksetParameterIdPageReadHandler read a "page" of parameter values from workset.
// POST /api/model/:model/workset/:set/parameter/value-id
// Dimension(s) and enum-based parameters returned as enum id, not enum codes.
func worksetParameterIdPageReadHandler(w http.ResponseWriter, r *http.Request) {
	doReadParameterPageHandler(w, r, "set", true, false)
}

// runParameterPageReadHandler read a "page" of parameter values from model run.
// POST /api/model/:model/run/:run/parameter/value
// Dimension(s) and enum-based parameters returned as enum codes.
func runParameterPageReadHandler(w http.ResponseWriter, r *http.Request) {
	doReadParameterPageHandler(w, r, "run", false, true)
}

// runParameterIdPageReadHandler read a "page" of parameter values from model run.
// POST /api/model/:model/run/:run/parameter/value-id
// Dimension(s) and enum-based parameters returned as enum id, not enum codes.
func runParameterIdPageReadHandler(w http.ResponseWriter, r *http.Request) {
	doReadParameterPageHandler(w, r, "run", false, false)
}

// doReadParameterPageHandler read a "page" of parameter values from workset or model run.
// Json is posted to specify parameter name, "page" size and other read arguments,
// see db.ReadParamLayout for more details.
// Page is part of parameter values defined by zero-based "start" row number and row count.
// If row count <= 0 then all rows returned.
// Dimension(s) and enum-based parameters returned as enum codes or enum id's
func doReadParameterPageHandler(w http.ResponseWriter, r *http.Request, srcArg string, isSet, isCode bool) {

	// url parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	src := getRequestParam(r, srcArg)  // workset name or run digest-or-name
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// decode json request body
	var layout db.ReadParamLayout
	if !jsonRequestDecode(w, r, true, &layout) {
		return // error at json decode, response done with http error
	}
	layout.IsFromSet = isSet // overwrite json value, it was likely default

	// get converter from id's cell into code cell
	var cvtCell func(interface{}) (interface{}, error)
	if isCode {
		ok := false
		cvtCell, ok = theCatalog.ParameterCellConverter(false, dn, layout.Name)
		if !ok {
			http.Error(w, helper.MsgL(lang, "Error at parameter read:", layout.Name), http.StatusBadRequest)
			return
		}
	}

	// write to response: page layout and page data
	jsonSetHeaders(w, r) // start response with set json headers, i.e. content type

	w.Write([]byte("{\"Page\":[")) // start of data page and start of json output array

	enc := json.NewEncoder(w)
	cvtWr := jsonCellWriter(w, enc, cvtCell)

	// read parameter page into json array response, convert enum id's to code if requested
	lt, ok := theCatalog.ReadParameterTo(dn, src, &layout, cvtWr)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at parameter read:", src, ":", layout.Name), http.StatusBadRequest)
		return
	}
	w.Write([]byte{']'}) // end of data page array

	// continue response with output page layout: offset, size, last page flag
	w.Write([]byte(",\"Layout\":"))

	err := json.NewEncoder(w).Encode(lt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write([]byte("}")) // end of data page and end of json
}

// runTablePageReadHandler read a "page" of output table values
// from expression table, accumulator table or "all-accumulators" view of model run.
// POST /api/model/:model/run/:run/table/value
// Dimension items returned as enum codes or, if dimension type simple as string values
func runTablePageReadHandler(w http.ResponseWriter, r *http.Request) {
	doReadTablePageHandler(w, r, true)
}

// runTableIdPageReadHandler read a "page" of output table values
// from expression table, accumulator table or "all-accumulators" view of model run.
// POST /api/model/:model/run/:run/table/value-id
// Dimension(s) returned as enum id, not enum codes.
func runTableIdPageReadHandler(w http.ResponseWriter, r *http.Request) {
	doReadTablePageHandler(w, r, false)
}

// doReadTablePageHandler read a "page" of output table values
// from expression table, accumulator table or "all-accumulators" view of model run.
// Json is posted to specify table name, "page" size and other read arguments,
// see db.ReadTableLayout for more details.
// Page is part of output table values defined by zero-based "start" row number and row count.
// If row count <= 0 then all rows returned.
// Dimension items returned enum id's or as enum codes and for dimension type simple as string values.
func doReadTablePageHandler(w http.ResponseWriter, r *http.Request, isCode bool) {

	// url parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	rdsn := getRequestParam(r, "run")  // run digest-or-stamp-or-name
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// decode json request body
	var layout db.ReadTableLayout
	if !jsonRequestDecode(w, r, true, &layout) {
		return // error at json decode, response done with http error
	}

	// if required get converter from id's cell into code cell
	var cvtCell func(interface{}) (interface{}, error)
	if isCode {
		ok := false
		cvtCell, ok = theCatalog.TableToCodeCellConverter(dn, layout.Name, layout.IsAccum, layout.IsAllAccum)
		if !ok {
			http.Error(w, helper.MsgL(lang, "Error at output table read:", layout.Name), http.StatusBadRequest)
			return
		}
	}

	// write to response: page layout and page data
	jsonSetHeaders(w, r) // start response with set json headers, i.e. content type

	w.Write([]byte("{\"Page\":[")) // start of data page and start of json output array

	enc := json.NewEncoder(w)
	cvtWr := jsonCellWriter(w, enc, cvtCell)

	// read output table page into json array response, convert enum id's to code if requested
	lt, ok := theCatalog.ReadOutTableTo(dn, rdsn, &layout, cvtWr)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at run output table read:", rdsn, ":", layout.Name), http.StatusBadRequest)
		return
	}
	w.Write([]byte{']'}) // end of data page array

	// continue response with output page layout: offset, size, last page flag
	w.Write([]byte(",\"Layout\":"))

	err := json.NewEncoder(w).Encode(lt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write([]byte("}")) // end of data page and end of json
}

// runTableCalcPageReadHandler read a "page" of output table expressions and calculate of additional measures.
// POST /api/model/:model/run/:run/table/calc
// Dimension items returned as enum codes or, if dimension type simple as string values
func runTableCalcPageReadHandler(w http.ResponseWriter, r *http.Request) {
	doReadTableCalcPageHandler(w, r, true)
}

// runTableCalcIdPageReadHandler read a "page" of output table expressions and calculate of additional measures.
// POST /api/model/:model/run/:run/table/calc-id
// Dimension(s) returned as enum id, not enum codes.
func runTableCalcIdPageReadHandler(w http.ResponseWriter, r *http.Request) {
	doReadTableCalcPageHandler(w, r, false)
}

// doTableCalcGetPageHandler calculate a "page" of additional measures for output table using expressions or by aggregating accumulators.
// Json is posted to specify table name, "page" size and additional measures calculations,
// see db.ReadCalculteTableLayout for more details.
// Page is part of output table values defined by zero-based "start" row number and row count.
// If row count <= 0 then all rows returned.
// Dimension items returned enum id's or as enum codes and for dimension type simple as string values.
func doReadTableCalcPageHandler(w http.ResponseWriter, r *http.Request, isCode bool) {

	// url parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	rdsn := getRequestParam(r, "run")  // run digest-or-stamp-or-name
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// decode json request body
	var layout db.ReadCalculteTableLayout
	if !jsonRequestDecode(w, r, true, &layout) {
		return // error at json decode, response done with http error
	}

	// if required get converter from id's cell into code cell
	var cvtCell func(interface{}) (interface{}, error)
	runIds := []int{}
	ok := false
	if isCode {
		cvtCell, _, runIds, ok = theCatalog.TableToCodeCalcCellConverter(dn, rdsn, layout.Name, layout.Calculation, nil)
		if !ok {
			http.Error(w, helper.MsgL(lang, "Error at run output table calculate:", layout.Name), http.StatusBadRequest)
			return
		}
	}

	// write to response: page layout and page data
	jsonSetHeaders(w, r) // start response with set json headers, i.e. content type

	w.Write([]byte("{\"Page\":[")) // start of data page and start of json output array

	enc := json.NewEncoder(w)
	cvtWr := jsonCellWriter(w, enc, cvtCell)

	// calculate output table measure and read measure page into json array response, convert enum id's to code if requested
	lt, ok := theCatalog.ReadOutTableCalculateTo(
		dn, rdsn, &db.ReadTableLayout{ReadLayout: layout.ReadLayout}, layout.Calculation, runIds, cvtWr,
	)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at run output table calculate:", rdsn, ":", layout.Name), http.StatusBadRequest)
		return
	}

	w.Write([]byte{']'}) // end of data page array

	// continue response with output page layout: offset, size, last page flag
	w.Write([]byte(",\"Layout\":"))

	err := json.NewEncoder(w).Encode(lt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write([]byte("}")) // end of data page and end of json
}

// runTableComparePageReadHandler compare model runs and return a "page" of comparison expressions and/or calculated additional measures.
// POST /api/model/:model/run/:run/table/compare
// Dimension items returned as enum codes or, if dimension type simple as string values
func runTableComparePageReadHandler(w http.ResponseWriter, r *http.Request) {
	doReadTableComparePageHandler(w, r, true)
}

// runTableCompareIdPageReadHandler compare model runs and return a "page" of comparison expressions and/or calculated additional measures.
// POST /api/model/:model/run/:run/table/compare-id
// Dimension(s) returned as enum id, not enum codes.
func runTableCompareIdPageReadHandler(w http.ResponseWriter, r *http.Request) {
	doReadTableComparePageHandler(w, r, false)
}

// doReadTableComparePageHandler compare model runs with base run and return a "page" of comparison values or calculated additional measures.
// Json is posted to specify table name, "page" size and additional measures calculations,
// see db.ReadCompareTableLayout for more details.
// Page is part of output table values defined by zero-based "start" row number and row count.
// If row count <= 0 then all rows returned.
// Dimension items returned enum id's or as enum codes and for dimension type simple as string values.
func doReadTableComparePageHandler(w http.ResponseWriter, r *http.Request, isCode bool) {

	// url parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	rdsn := getRequestParam(r, "run")  // base run digest-or-stamp-or-name
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// decode json request body
	var layout db.ReadCompareTableLayout
	if !jsonRequestDecode(w, r, true, &layout) {
		return // error at json decode, response done with http error
	}

	// check if base run compeleted successfully
	rBase, ok := theCatalog.CompletedRunByDigestOrStampOrName(dn, rdsn)
	if !ok {
		omppLog.Log("Error at table compare: base run not found or not completed successfully:", rdsn)
		http.Error(w, helper.FmtL(lang, "Error at run output table compare: %s: base run must be completed successfully: %s", layout.Name, rdsn), http.StatusBadRequest)
		return
	}
	if rBase.Status != db.DoneRunStatus {
		omppLog.Log("Error at table compare: base run not completed successfully:", rdsn, ":", rBase.Status)
		http.Error(w, helper.FmtL(lang, "Error at run output table compare: %s: base run must be completed successfully: %s", layout.Name, rdsn), http.StatusBadRequest)
		return
	}
	layout.FromId = rBase.RunId // set base run

	// if required get converter from id's cell into code cell
	// validate all runs: it must be completed successfully
	var cvtCell func(interface{}) (interface{}, error)
	var runIds []int
	ok = false
	if isCode {
		cvtCell, _, runIds, ok = theCatalog.TableToCodeCalcCellConverter(dn, rdsn, layout.Name, layout.Calculation, layout.Runs)
		if !ok {
			http.Error(w, helper.MsgL(lang, "Error at run output table compare:", layout.Name), http.StatusBadRequest)
			return
		}
	} else {
		runIds, ok = isSuccessAllRuns(dn, layout.Runs)
		if !ok {
			omppLog.Log("Error at table compare: all runs must be completed successfully:", dn, ":", layout.Name)
			http.Error(w, helper.FmtL(lang, "Error at run output table compare: %s: all runs must be completed successfully: %s", layout.Name, rdsn), http.StatusBadRequest)
			return
		}
	}
	if len(runIds) <= 0 {
		omppLog.Log("Warning at table compare: only base run found, no runs to comparte with:", dn, ":", layout.Name)
	}

	// write to response: page layout and page data
	jsonSetHeaders(w, r) // start response with set json headers, i.e. content type

	w.Write([]byte("{\"Page\":[")) // start of data page and start of json output array

	enc := json.NewEncoder(w)
	cvtWr := jsonCellWriter(w, enc, cvtCell)

	// calculate output table measure and read measure page into json array response, convert enum id's to code if requested
	lt, ok := theCatalog.ReadOutTableCalculateTo(
		dn, rdsn, &db.ReadTableLayout{ReadLayout: layout.ReadLayout}, layout.Calculation, runIds, cvtWr,
	)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at run output table compare:", rdsn, ":", layout.Name), http.StatusBadRequest)
		return
	}

	w.Write([]byte{']'}) // end of data page array

	// continue response with output page layout: offset, size, last page flag
	w.Write([]byte(",\"Layout\":"))

	err := json.NewEncoder(w).Encode(lt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write([]byte("}")) // end of data page and end of json
}

// check if all runs completed successfully and return run id's for all existing runs, skip runs which do exist.
func isSuccessAllRuns(digest string, runLst []string) ([]int, bool) {

	// check if all additional model runs completed successfully
	runIds := []int{}

	if len(runLst) > 0 {
		rLst, _ := theCatalog.RunRowListByModel(digest)

		for _, rn := range runLst {

			rId := 0
			for k := 0; rId <= 0 && k < len(rLst); k++ {

				if rn == rLst[k].RunDigest || rn == rLst[k].RunStamp || rn == rLst[k].Name {
					rId = rLst[k].RunId
				}
				if rId > 0 {
					if rLst[k].Status != db.DoneRunStatus {
						omppLog.Log("Warning: model run not completed successfully:", rLst[k].RunDigest, ":", rLst[k].Status)
						return []int{}, false
					}
				}
			}
			if rId <= 0 {
				omppLog.Log("Warning: model run not found:", rn)
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

	return runIds, true
}

// worksetParameterPageGetHandler read a "page" of parameter values from workset.
//
//	GET /api/model/:model/workset/:set/parameter/:name/value
//	GET /api/model/:model/workset/:set/parameter/:name/value/start/:start
//	GET /api/model/:model/workset/:set/parameter/:name/value/start/:start/count/:count
//
// Dimension(s) and enum-based parameters returned as enum codes.
func worksetParameterPageGetHandler(w http.ResponseWriter, r *http.Request) {
	doParameterGetPageHandler(w, r, "set", true, true)
}

// runParameterPageGetHandler read a "page" of parameter values from model run results.
//
//	GET /api/model/:model/run/:run/parameter/:name/value
//	GET /api/model/:model/run/:run/parameter/:name/value/start/:start
//	GET /api/model/:model/run/:run/parameter/:name/value/start/:start/count/:count
//
// Dimension(s) and enum-based parameters returned as enum codes.
func runParameterPageGetHandler(w http.ResponseWriter, r *http.Request) {
	doParameterGetPageHandler(w, r, "run", false, true)
}

// doParameterGetPageHandler read a "page" of parameter values from workset or model run.
// Page is part of parameter values defined by zero-based "start" row number and row count.
// If row count <= 0 then all rows returned.
// Dimension(s) and enum-based parameters returned as enum codes or enum id's.
func doParameterGetPageHandler(w http.ResponseWriter, r *http.Request, srcArg string, isSet, isCode bool) {

	// url or query parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	src := getRequestParam(r, srcArg)  // workset name or run digest-or-stamp-or-name
	name := getRequestParam(r, "name") // parameter name
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// url or query parameters: page offset and page size
	start, ok := getInt64RequestParam(r, "start", 0)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid value of start row number to read", name), http.StatusBadRequest)
		return
	}
	count, ok := getInt64RequestParam(r, "count", 0)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid value of max row count to read", name), http.StatusBadRequest)
		return
	}

	// setup read layout
	layout := db.ReadParamLayout{
		ReadLayout: db.ReadLayout{
			Name:           name,
			ReadPageLayout: db.ReadPageLayout{Offset: start, Size: count},
		},
		IsFromSet: isSet,
	}

	// if required get converter from id's cell into code cell
	var cvtCell func(interface{}) (interface{}, error)
	if isCode {
		cvtCell, ok = theCatalog.ParameterCellConverter(false, dn, name)
		if !ok {
			http.Error(w, helper.MsgL(lang, "Error at parameter read:", name), http.StatusBadRequest)
			return
		}
	}

	// write to response
	jsonSetHeaders(w, r) // start response with set json headers, i.e. content type

	w.Write([]byte{'['}) // start of json output array

	enc := json.NewEncoder(w)
	cvtWr := jsonCellWriter(w, enc, cvtCell)

	// read parameter page into json array response, convert enum id's to code if requested
	_, ok = theCatalog.ReadParameterTo(dn, src, &layout, cvtWr)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at parameter read:", src, ":", layout.Name), http.StatusBadRequest)
		return
	}
	w.Write([]byte{']'}) // end of json output array
}

// runTableExprPageGetHandler read a "page" of output table expression(s) values from model run results.
// GET /api/model/:model/run/:run/table/:name/expr
// GET /api/model/:model/run/:run/table/:name/expr/start/:start
// GET /api/model/:model/run/:run/table/:name/expr/start/:start/count/:count
// Enum-based dimension items returned as enum codes.
func runTableExprPageGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableGetPageHandler(w, r, false, false, true)
}

// runTableAccPageGetHandler read a "page" of output table accumulator(s) values from model run results.
// GET /api/model/:model/run/:run/table/:name/acc/start/:start
// GET /api/model/:model/run/:run/table/:name/acc/start/:start/count/:count
// Enum-based dimension items returned as enum codes.
func runTableAccPageGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableGetPageHandler(w, r, true, false, true)
}

// runTableAllAccPageGetHandler read a "page" of output table accumulator(s) values
// from "all-accumulators" view of model run results.
// GET /api/model/:model/run/:run/table/:name/all-acc
// GET /api/model/:model/run/:run/table/:name/all-acc/start/:start
// GET /api/model/:model/run/:run/table/:name/all-acc/start/:start/count/:count
// Enum-based dimension items returned as enum codes.
func runTableAllAccPageGetHandler(w http.ResponseWriter, r *http.Request) {
	doTableGetPageHandler(w, r, true, true, true)
}

// doTableGetPageHandler read a "page" of values from
// output table expressions, accumulators or "all-accumulators" views.
// Page is part of output table values defined by zero-based "start" row number and row count.
// If row count <= 0 then all rows returned.
// Enum-based dimension items returned as enum id or as enum codes.
func doTableGetPageHandler(w http.ResponseWriter, r *http.Request, isAcc, isAllAcc, isCode bool) {

	// url or query parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	rdsn := getRequestParam(r, "run")  // run digest-or-stamp-or-name
	name := getRequestParam(r, "name") // output table name
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// url or query parameters: page offset and page size
	start, ok := getInt64RequestParam(r, "start", 0)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid value of start row number to read", name), http.StatusBadRequest)
		return
	}
	count, ok := getInt64RequestParam(r, "count", 0)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid value of max row count to read", name), http.StatusBadRequest)
		return
	}

	// setup read layout
	layout := db.ReadTableLayout{
		ReadLayout: db.ReadLayout{
			Name:           name,
			ReadPageLayout: db.ReadPageLayout{Offset: start, Size: count},
		},
		IsAccum:    isAcc,
		IsAllAccum: isAllAcc,
	}

	// if required get converter from id's cell into code cell
	var cvtCell func(interface{}) (interface{}, error)
	if isCode {
		cvtCell, ok = theCatalog.TableToCodeCellConverter(dn, layout.Name, layout.IsAccum, layout.IsAllAccum)
		if !ok {
			http.Error(w, helper.MsgL(lang, "Error at run output table read:", name), http.StatusBadRequest)
			return
		}
	}

	// write to response: page layout and page data
	jsonSetHeaders(w, r) // start response with set json headers, i.e. content type

	w.Write([]byte{'['}) // start of json output array

	enc := json.NewEncoder(w)
	cvtWr := jsonCellWriter(w, enc, cvtCell)

	// read output table page into json array response, convert enum id's to code if requested
	_, ok = theCatalog.ReadOutTableTo(dn, rdsn, &layout, cvtWr)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at run output table read:", rdsn, ":", layout.Name), http.StatusBadRequest)
		return
	}
	w.Write([]byte{']'}) // end of json output array
}

// runTableCalcPageGetHandler for all output table expressions calculate a "page" of additional measures.
//
// Measures calculated either as aggregation for each expresion: SUM AVG COUNT MIN MAX VAR SD SE CV
// or as comma separated list of arbitrary calcialtion expressions, for example: OM_AVG(acc0),OM_SD(acc1)
//
//	GET /api/model/:model/run/:run/table/:name/calc/:calc
//	GET /api/model/:model/run/:run/table/:name/calc/:calc/start/:start
//	GET /api/model/:model/run/:run/table/:name/calc/:calc/start/:start/count/:count
//
// Calculation applied to derived accumulator with the same name as expression name.
//
// Page is part of output table values defined by zero-based "start" row number and row count.
// If row count <= 0 then all rows returned.
// Enum-based dimension items returned as enum codes.
func runTableCalcPageGetHandler(w http.ResponseWriter, r *http.Request) {

	// url or query parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	rdsn := getRequestParam(r, "run")  // run digest-or-stamp-or-name
	name := getRequestParam(r, "name") // output table name
	calc := getRequestParam(r, "calc") // calculation function name: sum avg count min max var sd se cv
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// validate parameters: page offset, page size and calculation expression
	if calc == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) calculation expression", calc), http.StatusBadRequest)
		return
	}
	start, ok := getInt64RequestParam(r, "start", 0)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid value of start row number to read", name), http.StatusBadRequest)
		return
	}
	count, ok := getInt64RequestParam(r, "count", 0)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid value of max row count to read", name), http.StatusBadRequest)
		return
	}

	// setup read layout and calculate layout
	tableLt := db.ReadTableLayout{
		ReadLayout: db.ReadLayout{
			Name:           name,
			ReadPageLayout: db.ReadPageLayout{Offset: start, Size: count},
		},
	}

	calcLt, ok := theCatalog.TableAggrExprCalculateLayout(dn, name, calc)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid calculation expression", calc), http.StatusBadRequest)
		return
	}

	// get converter from id's cell into code cell
	cvtCell, _, runIds, ok := theCatalog.TableToCodeCalcCellConverter(dn, rdsn, tableLt.Name, calcLt, nil)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Failed to create output table csv converter:", name), http.StatusBadRequest)
		return
	}

	// write to response: page layout and page data
	jsonSetHeaders(w, r) // start response with set json headers, i.e. content type

	w.Write([]byte{'['}) // start of json output array

	enc := json.NewEncoder(w)
	cvtWr := jsonCellWriter(w, enc, cvtCell)

	// calculate output table measure and read measure page into json array response, convert enum id's to code if requested
	_, ok = theCatalog.ReadOutTableCalculateTo(dn, rdsn, &tableLt, calcLt, runIds, cvtWr)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at run output table read:", rdsn, ":", name), http.StatusBadRequest)
		return
	}
	w.Write([]byte{']'}) // end of json output array
}

// runTableComparePageGetHandler compare model runs and return a "page" of comparison measures.
//
// It is either calculation for each expression: DIFF RATIO PERCENT or multiple arbitrary calculations.
// For example, RATIO is: expr0[variant] / expr0[base], expr1[variant] / expr1[base],....
// Or arbitrary comma separated expression(s): expr0 , expr1[variant] + expr2[base] , ....
// Variant runs can be a comma separated list of run digests or run stamps or run names.
// If run name contains comma then name must be "double quoted" or 'single quoted'.
// For example: "Year 1995, 1996", 'Age [30, 40]'
//
// GET /api/model/:model/run/:run/table/:name/compare/:compare/variant/:variant
// GET /api/model/:model/run/:run/table/:name/compare/:compare/variant/:variant/start/:start
// GET /api/model/:model/run/:run/table/:name/compare/:compare/variant/:variant/start/:start/count/:count
//
// Page is part of output table values defined by zero-based "start" row number and row count.
// If row count <= 0 then all rows returned.
// Enum-based dimension items returned as enum codes.
func runTableComparePageGetHandler(w http.ResponseWriter, r *http.Request) {

	// url or query parameters
	dn := getRequestParam(r, "model")        // model digest-or-name
	rdsn := getRequestParam(r, "run")        // base run digest-or-stamp-or-name
	name := getRequestParam(r, "name")       // output table name
	compare := getRequestParam(r, "compare") // comparison function name: diff ratio percent
	vr := getRequestParam(r, "variant")      // variant run digest-or-stamp-or-name
	lang := preferedRequestLang(r, "")       // get prefered language for messages

	// validate parameters: page offset, page size and calculation expression
	if compare == "" {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) comparison expression"), http.StatusBadRequest)
		return
	}
	vRdsn := helper.ParseCsvLine(vr, 0)
	if len(vRdsn) <= 0 {
		http.Error(w, helper.MsgL(lang, "Invalid or empty list of runs to compare"), http.StatusBadRequest)
		return
	}
	start, ok := getInt64RequestParam(r, "start", 0)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid value of start row number to read", name), http.StatusBadRequest)
		return
	}
	count, ok := getInt64RequestParam(r, "count", 0)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid value of max row count to read", name), http.StatusBadRequest)
		return
	}

	// setup read layout and calculate layout
	tableLt := db.ReadTableLayout{
		ReadLayout: db.ReadLayout{
			Name:           name,
			ReadPageLayout: db.ReadPageLayout{Offset: start, Size: count},
		},
	}

	calcLt, ok := theCatalog.TableExprCompareLayout(dn, name, compare)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid comparison expression", compare), http.StatusBadRequest)
		return
	}

	// get converter from id's cell into code cell
	cvtCell, _, runIds, ok := theCatalog.TableToCodeCalcCellConverter(dn, rdsn, tableLt.Name, calcLt, vRdsn)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Failed to create output table converter:", name), http.StatusBadRequest)
		return
	}

	// write to response: page layout and page data
	jsonSetHeaders(w, r) // start response with set json headers, i.e. content type

	w.Write([]byte{'['}) // start of json output array

	enc := json.NewEncoder(w)
	cvtWr := jsonCellWriter(w, enc, cvtCell)

	// calculate output table measure and read measure page into json array response, convert enum id's to code if requested
	_, ok = theCatalog.ReadOutTableCalculateTo(dn, rdsn, &tableLt, calcLt, runIds, cvtWr)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at run output table read:", rdsn, ":", name), http.StatusBadRequest)
		return
	}
	w.Write([]byte{']'}) // end of json output array
}

// runMicrodataPageReadHandler read a "page" of microdata values from model run.
// POST /api/model/:model/run/:run/microdata/value
// Enum-based microdata attributes returned as enum codes.
func runMicrodataPageReadHandler(w http.ResponseWriter, r *http.Request) {
	doReadMicrodataPageHandler(w, r, true)
}

// runMicrodataIdPageReadHandler read a "page" of microdata values from model run.
// POST /api/model/:model/run/:run/microdata/value-id
// Enum-based microdata attributes returned as enum id, not enum codes.
func runMicrodataIdPageReadHandler(w http.ResponseWriter, r *http.Request) {
	doReadMicrodataPageHandler(w, r, false)
}

// doReadMicrodataPageHandler read a "page" of microdata values from model run.
// Json is posted to specify entity name, "page" size and other read arguments,
// see db.ReadMicroLayout for more details.
// If generation digest not specified in ReadMicroLayout then it found by entity name and run digest.
// Page of values is a rows from microdata value table started at zero based offset row
// and up to max page size rows, if page size <= 0 then all values returned.
// Enum-based microdata attributes returned as enum codes or enum id's.
func doReadMicrodataPageHandler(w http.ResponseWriter, r *http.Request, isCode bool) {

	// url parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	rdsn := getRequestParam(r, "run")  // run digest-or-stamp-or-name
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// return error if microdata disabled
	if !theCfg.isMicrodata {
		http.Error(w, helper.MsgL(lang, "Error: microdata not allowed:", dn, rdsn), http.StatusBadRequest)
		return
	}

	// decode json request body
	var layout db.ReadMicroLayout
	if !jsonRequestDecode(w, r, true, &layout) {
		return // error at json decode, response done with http error
	}

	// get converter from id's cell into code cell
	var cvtCell func(interface{}) (interface{}, error)
	if isCode {

		ok := false
		genDigest := ""
		_, genDigest, cvtCell, ok = theCatalog.MicrodataCellConverter(false, dn, rdsn, layout.Name)
		if !ok {
			http.Error(w, helper.MsgL(lang, "Error at run microdata read:", layout.Name), http.StatusBadRequest)
			return
		}
		if layout.GenDigest != "" && layout.GenDigest != genDigest {
			http.Error(w, helper.FmtL(lang, "Error at run microdata read, generation digest not found: %s expected: %s: %s", layout.GenDigest, genDigest, layout.Name), http.StatusBadRequest)
			return
		}
	}

	// write to response: page layout and page data
	jsonSetHeaders(w, r) // start response with set json headers, i.e. content type

	w.Write([]byte("{\"Page\":[")) // start of data page and start of json output array

	enc := json.NewEncoder(w)
	cvtWr := jsonCellWriter(w, enc, cvtCell)

	// read microdata page into json array response, convert enum id's to code if requested
	lt, ok := theCatalog.ReadMicrodataTo(dn, rdsn, &layout, cvtWr)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at run microdata read:", rdsn, ":", layout.Name), http.StatusBadRequest)
		return
	}
	w.Write([]byte{']'}) // end of data page array

	// continue response with output page layout: offset, size, last page flag
	w.Write([]byte(",\"Layout\":"))

	err := json.NewEncoder(w).Encode(lt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write([]byte("}")) // end of data page and end of json
}

// runMicrodataPageGetHandler read a "page" of microdata values from model run results.
// GET /api/model/:model/run/:run/microdata/:name/value
// GET /api/model/:model/run/:run/microdata/:name/value/start/:start
// GET /api/model/:model/run/:run/microdata/:name/value/start/:start/count/:count
// Enum-based microdata attributes returned as enum codes.
func runMicrodataPageGetHandler(w http.ResponseWriter, r *http.Request) {
	doMicrodataGetPageHandler(w, r, true)
}

// doMicrodataGetPageHandler read a "page" of microdata values from model run.
// Page of values is a rows from microdata value table started at zero based offset row
// and up to max page size rows, if page size <= 0 then all values returned.
// Enum-based microdata attributes returned as enum codes or enum id's.
func doMicrodataGetPageHandler(w http.ResponseWriter, r *http.Request, isCode bool) {

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

	// url or query parameters: page offset and page size
	start, ok := getInt64RequestParam(r, "start", 0)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid value of start row number to read", name), http.StatusBadRequest)
		return
	}
	count, ok := getInt64RequestParam(r, "count", 0)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid value of max row count to read", name), http.StatusBadRequest)
		return
	}

	// setup read layout
	layout := db.ReadMicroLayout{
		ReadLayout: db.ReadLayout{
			Name:           name,
			ReadPageLayout: db.ReadPageLayout{Offset: start, Size: count},
		},
	}

	// get converter from id's cell into code cell
	var cvtCell func(interface{}) (interface{}, error)
	if isCode {

		ok := false
		genDigest := ""
		_, genDigest, cvtCell, ok = theCatalog.MicrodataCellConverter(false, dn, rdsn, layout.Name)
		if !ok {
			http.Error(w, helper.MsgL(lang, "Error at run microdata read:", name), http.StatusBadRequest)
			return
		}
		layout.GenDigest = genDigest
	}

	// write to response
	jsonSetHeaders(w, r) // start response with set json headers, i.e. content type

	w.Write([]byte{'['}) // start of json output array

	enc := json.NewEncoder(w)
	cvtWr := jsonCellWriter(w, enc, cvtCell)

	// read microdata page into json array response, convert enum id's to code if requested
	_, ok = theCatalog.ReadMicrodataTo(dn, rdsn, &layout, cvtWr)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at run microdata read:", rdsn, ":", layout.Name), http.StatusBadRequest)
		return
	}
	w.Write([]byte{']'}) // end of json output array
}

// runMicrodataCalcPageReadHandler read a "page" of microdata values from model run.
// POST /api/model/:model/run/:run/microdata/calc
// It can be multiple aggregations of value attributes (float of integer type), group by dimension attributes (enum-based or bool type).
// For example: GroupBy: [AgeGroup, Sex] and Calculation: [OM_AVG(Income), OM_MAX(Salary+Pension)]
// Enum-based microdata attributes returned as enum codes.
func runMicrodataCalcPageReadHandler(w http.ResponseWriter, r *http.Request) {

	// url parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	rdsn := getRequestParam(r, "run")  // run digest-or-stamp-or-name
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// return error if microdata disabled
	if !theCfg.isMicrodata {
		http.Error(w, helper.MsgL(lang, "Error: microdata not allowed:", dn, rdsn), http.StatusBadRequest)
		return
	}

	// decode json request body
	var layout db.ReadCalculteMicroLayout
	if !jsonRequestDecode(w, r, true, &layout) {
		return // error at json decode, response done with http error
	}

	doReadMicrodataCalcPageHandler(w, r, dn, rdsn, &layout, []string{}, true)
}

// runMicrodataIdPageReadHandler read a "page" of microdata values from model run.
// POST /api/model/:model/run/:run/microdata/calc-id
// It can be multiple aggregations of value attributes (float of integer type), group by dimension attributes (enum-based or bool type).
// For example: GroupBy: [AgeGroup, Sex] and Calculation: [OM_AVG(Income), OM_MAX(Salary+Pension)]
// Enum-based microdata attributes returned as enum id, not enum codes.
func runMicrodataCalcIdPageReadHandler(w http.ResponseWriter, r *http.Request) {

	// url parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	rdsn := getRequestParam(r, "run")  // run digest-or-stamp-or-name
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// return error if microdata disabled
	if !theCfg.isMicrodata {
		http.Error(w, helper.MsgL(lang, "Error: microdata not allowed:", dn, rdsn), http.StatusBadRequest)
		return
	}

	// decode json request body
	var layout db.ReadCalculteMicroLayout
	if !jsonRequestDecode(w, r, true, &layout) {
		return // error at json decode, response done with http error
	}

	doReadMicrodataCalcPageHandler(w, r, dn, rdsn, &layout, []string{}, false)
}

// doReadMicrodataCalcPageHandler read a "page" of microdata values from model run.
// Json is posted to specify entity name, "page" size and other read arguments,
// see db.ReadCalculteMicroLayout for more details.
// If generation digest not specified in ReadCalculteMicroLayout then it found by entity name and run digest.
// Page of values is a rows from microdata value table started at zero based offset row
// and up to max page size rows, if page size <= 0 then all values returned.
// Enum-based microdata attributes returned as enum codes or enum id's.
func doReadMicrodataCalcPageHandler(w http.ResponseWriter, r *http.Request, dn string, rdsn string, layout *db.ReadCalculteMicroLayout, varLst []string, isCode bool) {

	lang := preferedRequestLang(r, "") // get prefered language for messages

	// return error if microdata disabled
	if !theCfg.isMicrodata {
		http.Error(w, helper.MsgL(lang, "Error: microdata not allowed:", dn, rdsn), http.StatusBadRequest)
		return
	}
	if len(layout.GroupBy) <= 0 {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) microdata group by attributes:", dn, ":", layout.Name), http.StatusBadRequest)
		return
	}
	if len(layout.Calculation) <= 0 {
		http.Error(w, helper.MsgL(lang, "Invalid (empty) microdata calculation expression(s):", dn, ":", layout.Name), http.StatusBadRequest)
		return
	}

	// get base run id, run variants, entity generation digest and microdata cell converter
	baseRunId, runIds, genDigest, cvtMicro, err := theCatalog.MicrodataCalcToCsvConverter(dn, isCode, rdsn, varLst, layout.Name, &layout.CalculateMicroLayout)
	if err != nil {
		omppLog.Log("Failed to create microdata csv converter:", rdsn, ":", layout.Name, ":", err)
		http.Error(w, helper.MsgL(lang, "Failed to create microdata csv converter:", rdsn, ":", layout.Name), http.StatusBadRequest)
		return
	}
	layout.FromId = baseRunId

	// get converter from id's cell into code cell
	var cvtCell func(interface{}) (interface{}, error)
	if isCode {

		cvtCell, err = cvtMicro.IdToCodeCell(cvtMicro.ModelDef, layout.Name)
		if err != nil {
			omppLog.Log("Failed to create microdata cell value converter:", dn, ":", layout.Name, ":", err)
			http.Error(w, helper.MsgL(lang, "Failed to create microdata cell value converter:", rdsn, ":", layout.Name), http.StatusBadRequest)
		}
	}

	// read microdata values, page size =0: read all values
	microLt := db.ReadMicroLayout{
		ReadLayout: layout.ReadLayout,
		GenDigest:  genDigest,
	}

	// write to response: page layout and page data
	jsonSetHeaders(w, r) // start response with set json headers, i.e. content type

	w.Write([]byte("{\"Page\":[")) // start of data page and start of json output array

	enc := json.NewEncoder(w)
	cvtWr := jsonCellWriter(w, enc, cvtCell)

	// read microdata page into json array response, convert enum id's to code if requested
	lt, ok := theCatalog.ReadMicrodataCalculateTo(dn, rdsn, &microLt, &layout.CalculateMicroLayout, runIds, cvtWr)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at run microdata read:", rdsn, ":", layout.Name), http.StatusBadRequest)
		return
	}
	w.Write([]byte{']'}) // end of data page array

	// continue response with output page layout: offset, size, last page flag
	w.Write([]byte(",\"Layout\":"))

	err = json.NewEncoder(w).Encode(lt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write([]byte("}")) // end of data page and end of json
}

// runMicrodataComparePageReadHandler read a "page" of microdata comparison between base and variant(s) model runs.
// POST /api/model/:model/run/:run/microdata/compare
// It can be multiple comparison of value attributes (float of integer type) and / or aggregation of value attributes.
// All comparisons and aggregations grouped by dimension attributes (enum-based or bool type).
// It can be multiple aggregations of value attributes (float of integer type), group by dimension attributes (enum-based or bool type).
// For example: GroupBy: [AgeGroup, Sex] and Calculation: [OM_AVG(Income[variant]-Income[base]) , OM_MAX(Salary+Pension)]
// Enum-based microdata attributes returned as enum codes.
func runMicrodataComparePageReadHandler(w http.ResponseWriter, r *http.Request) {

	// url parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	rdsn := getRequestParam(r, "run")  // run digest-or-stamp-or-name
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// return error if microdata disabled
	if !theCfg.isMicrodata {
		http.Error(w, helper.MsgL(lang, "Error: microdata not allowed:", dn, rdsn), http.StatusBadRequest)
		return
	}

	// decode json request body
	var layout db.ReadCompareMicroLayout
	if !jsonRequestDecode(w, r, true, &layout) {
		return // error at json decode, response done with http error
	}

	doReadMicrodataCalcPageHandler(w, r, dn, rdsn, &layout.ReadCalculteMicroLayout, layout.Runs, true)
}

// runMicrodataCompareIdPageReadHandler read a "page" of microdata comparison between base and variant(s) model runs.
// POST /api/model/:model/run/:run/microdata/compare-id
// It can be multiple comparison of value attributes (float of integer type) and / or aggregation of value attributes.
// All comparisons and aggregations grouped by dimension attributes (enum-based or bool type).
// It can be multiple aggregations of value attributes (float of integer type), group by dimension attributes (enum-based or bool type).
// For example: GroupBy: [AgeGroup, Sex] and Calculation: [OM_AVG(Income[variant]-Income[base]) , OM_MAX(Salary+Pension)]
// Enum-based microdata attributes returned as enum id, not enum codes.
func runMicrodataCompareIdPageReadHandler(w http.ResponseWriter, r *http.Request) {

	// url parameters
	dn := getRequestParam(r, "model")  // model digest-or-name
	rdsn := getRequestParam(r, "run")  // run digest-or-stamp-or-name
	lang := preferedRequestLang(r, "") // get prefered language for messages

	// return error if microdata disabled
	if !theCfg.isMicrodata {
		http.Error(w, helper.MsgL(lang, "Error: microdata not allowed:", dn, rdsn), http.StatusBadRequest)
		return
	}

	// decode json request body
	var layout db.ReadCompareMicroLayout
	if !jsonRequestDecode(w, r, true, &layout) {
		return // error at json decode, response done with http error
	}

	doReadMicrodataCalcPageHandler(w, r, dn, rdsn, &layout.ReadCalculteMicroLayout, layout.Runs, false)
}

// runMicrodataCalcPageGetHandler aggregate a "page" of microdata values from model run results.
// GET /api/model/:model/run/:run/microdata/:name/group-by/:group-by/calc/:calc
// GET /api/model/:model/run/:run/microdata/:name/group-by/:group-by/calc/:calc/start/:start
// GET /api/model/:model/run/:run/microdata/:name/group-by/:group-by/calc/:calc/start/:start/count/:count
// It can be multiple aggregations of value attributes (float of integer type), group by dimension attributes (enum-based or bool type).
// For example: microdata/Person/group-by/AgeGroup,Sex/calc/OM_AVG(Income),OM_MAX(Salary+Pension)/csv
// If run name contains comma then name must be "double quoted" or 'single quoted'.
// For example: "Year 1995, 1996" or: 'Age [30, 40]'
// Enum-based microdata attributes returned as enum codes.
func runMicrodataCalcPageGetHandler(w http.ResponseWriter, r *http.Request) {
	doMicrodataCalcGetPageHandler(w, r, true, "calc", false)
}

// runMicrodataComparePageGetHandler microdata comparison "page" from model run results.
// GET /api/model/:model/run/:run/microdata/:name/group-by/:group-by/compare/:compare/variant/:variant
// GET /api/model/:model/run/:run/microdata/:name/group-by/:group-by/compare/:compare/variant/:variant/start/:start
// GET /api/model/:model/run/:run/microdata/:name/group-by/:group-by/compare/:compare/variant/:variant/start/:start/count/:count
// It can be multiple comparison of value attributes (float of integer type) and / or aggregation of value attributes.
// All comparisons and aggregations grouped by dimension attributes (enum-based or bool type).
// For example: microdata/Person/group-by/AgeGroup,Sex/compare/OM_AVG(Income[variant]-Income[base]),OM_MAX(Salary+Pension)
// If run name contains comma then name must be "double quoted" or 'single quoted'.
// For example: "Year 1995, 1996" or: 'Age [30, 40]'
// Enum-based microdata attributes returned as enum codes.
func runMicrodataComparePageGetHandler(w http.ResponseWriter, r *http.Request) {
	doMicrodataCalcGetPageHandler(w, r, true, "compare", true)
}

// doMicrodataCalcGetPageHandler aggreagte a "page" of microdata values from model run.
// Page of values is a rows from microdata value table started at zero based offset row
// and up to max page size rows, if page size <= 0 then all values returned.
// Enum-based microdata attributes returned as enum codes or enum id's.
func doMicrodataCalcGetPageHandler(w http.ResponseWriter, r *http.Request, isCode bool, calcKey string, isVar bool) {

	// url or query parameters
	dn := getRequestParam(r, "model")                                 // model digest-or-name
	rdsn := getRequestParam(r, "run")                                 // run digest-or-stamp-or-name
	name := getRequestParam(r, "name")                                // entity name
	groupBy := helper.ParseCsvLine(getRequestParam(r, "group-by"), 0) // group by list of attributes, comma-separated
	cLst := helper.ParseCsvLine(getRequestParam(r, calcKey), 0)       // list of aggregations of value attribute(s), comma-separated
	lang := preferedRequestLang(r, "")                                // get prefered language for messages

	// url or query parameters: page offset and page size
	start, ok := getInt64RequestParam(r, "start", 0)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid value of start row number to read", name), http.StatusBadRequest)
		return
	}
	count, ok := getInt64RequestParam(r, "count", 0)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Invalid value of max row count to read", name), http.StatusBadRequest)
		return
	}

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
		http.Error(w, helper.MsgL(lang, "Invalid (empty) microdata aggregation(s)", name), http.StatusBadRequest)
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
		omppLog.Log("Failed to create microdata csv converter:", rdsn, ":", name, ":", err)
		http.Error(w, helper.MsgL(lang, "Failed to create microdata csv converter:", rdsn, ":", name), http.StatusBadRequest)
		return
	}

	// read microdata values, page size =0: read all values
	microLt := db.ReadMicroLayout{
		ReadLayout: db.ReadLayout{
			Name:           name,
			FromId:         baseRunId,
			ReadPageLayout: db.ReadPageLayout{Offset: start, Size: count},
		},
		GenDigest: genDigest,
	}

	// get converter from id's cell into code cell
	var cvtCell func(interface{}) (interface{}, error)
	if isCode {

		cvtCell, err = cvtMicro.IdToCodeCell(cvtMicro.ModelDef, name)
		if err != nil {
			omppLog.Log("Failed to create microdata cell value converter:", dn, ":", name, ":", err)
			http.Error(w, helper.MsgL(lang, "Failed to create microdata cell value converter:", rdsn, ":", name), http.StatusBadRequest)
		}
	}

	// write to response
	jsonSetHeaders(w, r) // start response with set json headers, i.e. content type

	w.Write([]byte{'['}) // start of json output array

	enc := json.NewEncoder(w)
	cvtWr := jsonCellWriter(w, enc, cvtCell)

	// read microdata page into json array response, convert enum id's to code if requested
	_, ok = theCatalog.ReadMicrodataCalculateTo(dn, rdsn, &microLt, &calcLt, runIds, cvtWr)
	if !ok {
		http.Error(w, helper.MsgL(lang, "Error at run microdata read:", rdsn, ":", microLt.Name), http.StatusBadRequest)
		return
	}
	w.Write([]byte{']'}) // end of json output array
}
