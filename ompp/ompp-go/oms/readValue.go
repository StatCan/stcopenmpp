// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// ReadParameterTo select "page" of parameter values from workset or model run and pass each row into cvtWr().
// Parameter identified by model digest-or-name, run digest-or-stamp-or-name or set name, parameter name.
// Page of values is a rows from parameter value table started at zero based offset row
// and up to max page size rows, if page size <= 0 then all values returned.
// Parameter values can be read-only (select from run or read-only workset) or read-write (read-write workset).
// Rows can be filtered and ordered (see db.ReadParamLayout for details).
func (mc *ModelCatalog) ReadParameterTo(dn, src string, layout *db.ReadParamLayout, cvtWr func(src interface{}) (bool, error)) (*db.ReadPageLayout, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		return nil, false
	}
	if layout.Name == "" {
		omppLog.Log("Error: invalid (empty) output table name")
		return nil, false
	}

	// get model metadata and database connection
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return nil, false
	}

	// check if parameter name exist in the model
	if _, ok = meta.ParamByName(layout.Name); !ok {
		omppLog.Log("Warning: parameter not found: ", layout.Name)
		return nil, false // return empty result: parameter not found or error
	}

	// find workset id by name or run id by name-or-digest
	if layout.IsFromSet {

		w, err := db.GetWorksetByName(dbConn, meta.Model.ModelId, src)
		if err != nil {
			return nil, false // return empty result: workset select error
		}

		layout.FromId = w.SetId // source workset id

	} else {

		// get run_lst db row by digest, stamp or run name
		rst, err := db.GetRunByDigestStampName(dbConn, meta.Model.ModelId, src)
		if err != nil {
			omppLog.Log("Error at get run status:", meta.Model.Name, ":", src, ":", err.Error())
			return nil, false // return empty result: run select error
		}
		if rst == nil {
			omppLog.Log("Warning: run not found: ", meta.Model.Name, ": ", src)
			return nil, false // return empty result: run_lst row not found
		}

		layout.FromId = rst.RunId // source run id
	}

	// read parameter page
	lt, err := db.ReadParameterTo(dbConn, meta, layout, cvtWr)
	if err != nil {
		omppLog.Log("Error at read parameter: ", dn, ": ", layout.Name, ": ", err.Error())
		return nil, false // return empty result: values select error
	}

	return lt, true
}

// ReadOutTableTo select "page" of output table values from model run and pass each row into cvtWr().
// Output table identified by model digest-or-name, run digest-or-stamp-or-name and output table name.
// Page of values is a rows from output table expression or accumulator table or all accumulators view.
// Page started at zero based offset row and up to max page size rows, if page size <= 0 then all values returned.
// Values can be from expression table, accumulator table or "all accumulators" view.
// Rows can be filtered and ordered (see db.ReadTableLayout for details).
func (mc *ModelCatalog) ReadOutTableTo(dn, rdsn string, layout *db.ReadTableLayout, cvtWr func(src interface{}) (bool, error)) (*db.ReadPageLayout, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		return nil, false
	}
	if layout.Name == "" {
		omppLog.Log("Error: invalid (empty) output table name")
		return nil, false
	}

	// get model metadata and database connection
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return nil, false
	}

	// check if output table name exist in the model
	if _, ok = meta.OutTableByName(layout.Name); !ok {
		omppLog.Log("Warning: output table not found: ", layout.Name)
		return nil, false // return empty result: output table not found or error
	}

	// find model run id by digest-or-stamp-or-name
	r, ok := mc.CompletedRunByDigestOrStampOrName(dn, rdsn)
	if !ok {
		return nil, false // return empty result: run select error
	}
	if r.Status != db.DoneRunStatus {
		omppLog.Log("Warning: model run not completed successfully: ", rdsn, ": ", r.Status)
		return nil, false
	}
	layout.FromId = r.RunId // source run id

	// read output table page
	lt, err := db.ReadOutputTableTo(dbConn, meta, layout, cvtWr)
	if err != nil {
		omppLog.Log("Error at read output table: ", dn, ": ", layout.Name, ": ", err.Error())
		return nil, false // return empty result: values select error
	}

	return lt, true
}

// ReadOutTableCalculateTo select "page" of calculated output table values from model run(s) and pass each row into cvtWr().
//
// It can calculate multiple values based on expressions and/or accumulators aggregation.
// Optional list of run id's can be supplied to read more than one run from output table.
// Output table values identified by model digest-or-name run id(s) and output table name.
// Page of values is a rows from output table expression or accumulator table or all accumulators view.
// Page started at zero based offset row and up to max page size rows, if page size <= 0 then all values returned.
// Values can be from expression table, accumulator table or "all accumulators" view.
// Rows can be filtered and ordered (see db.ReadTableLayout for details).
func (mc *ModelCatalog) ReadOutTableCalculateTo(
	dn, rdsn string, layout *db.ReadTableLayout, calcLt []db.CalculateTableLayout, runIds []int, cvtWr func(src interface{}) (bool, error),
) (*db.ReadPageLayout, bool) {

	// if model digest-or-name is empty then return empty results
	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		return nil, false
	}
	if layout.Name == "" {
		omppLog.Log("Error: invalid (empty) output table name")
		return nil, false
	}
	if len(calcLt) <= 0 {
		omppLog.Log("Error: invalid (empty) output table calculation expression(s): ", layout.Name)
		return nil, false
	}
	for k := range calcLt {
		if len(calcLt[k].Calculate) <= 0 {
			omppLog.Log("Error: invalid (empty) output table calculation expression(s): ", layout.Name)
			return nil, false
		}
	}

	// get model metadata and database connection
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return nil, false
	}

	// check if output table name exist in the model
	if _, ok = meta.OutTableByName(layout.Name); !ok {
		omppLog.Log("Warning: output table not found: ", layout.Name)
		return nil, false // return empty result: output table not found or error
	}

	// find model run id by digest-or-stamp-or-name
	r, ok := mc.CompletedRunByDigestOrStampOrName(dn, rdsn)
	if !ok {
		return nil, false // return empty result: run select error
	}
	if r.Status != db.DoneRunStatus {
		omppLog.Log("Warning: model run not completed successfully: ", rdsn, ": ", r.Status)
		return nil, false
	}
	layout.FromId = r.RunId // source run id

	// read output table page
	lt, err := db.ReadOutputTableCalculteTo(dbConn, meta, layout, calcLt, runIds, cvtWr)
	if err != nil {
		omppLog.Log("Error at read output table: ", dn, ": ", layout.Name, ": ", err.Error())
		return nil, false // return empty result: values select error
	}

	return lt, true
}

// ReadMicrodataTo select "page" of microdata values from model run and pass each row into cvtWr().
// Microdata identified by model digest-or-name, run digest-or-stamp-or-name and entity name.
// Page of values is a rows from microdata value table started at zero based offset row
// and up to max page size rows, if page size <= 0 then all values returned.
// Rows can be filtered and ordered (see db.ReadMicroLayout for details).
func (mc *ModelCatalog) ReadMicrodataTo(dn, rdsn string, layout *db.ReadMicroLayout, cvtWr func(src interface{}) (bool, error)) (*db.ReadPageLayout, bool) {

	// validate parameters and return empty results on empty input
	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		return nil, false
	}
	if layout.Name == "" {
		omppLog.Log("Error: invalid (empty) model entity name")
		return nil, false
	}

	// get model metadata and database connection
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return nil, false
	}

	// find entity generation by entity name
	if _, ok := meta.EntityByName(layout.Name); !ok {
		omppLog.Log("Warning: model entity not found: ", layout.Name)
		return nil, false
	}

	// if run id not defiened then find model run id by digest-or-stamp-or-name
	if layout.FromId <= 0 {

		r, ok := mc.CompletedRunByDigestOrStampOrName(dn, rdsn)
		if !ok {
			return nil, false // return empty result: run select error
		}
		if r.Status != db.DoneRunStatus {
			omppLog.Log("Warning: model run not completed successfully: ", rdsn, ": ", r.Status)
			return nil, false
		}
		layout.FromId = r.RunId // source run id
	}

	// if generation digest undefined then find entity generation by entity name and run id
	if layout.GenDigest == "" {

		_, entGen, ok := mc.EntityGenByName(dn, layout.FromId, layout.Name)
		if !ok {
			return nil, false // entity generation not found
		}
		layout.GenDigest = entGen.GenDigest
	}

	// read microdata values page
	lt, err := db.ReadMicrodataTo(dbConn, meta, layout, cvtWr)
	if err != nil {
		omppLog.Log("Error at read microdata: ", dn, ": ", layout.Name, ": ", layout.GenDigest, ": ", err.Error())
		return nil, false // return empty result: values select error
	}

	return lt, true
}

// ReadMicrodataCalculateTo select "page" of aggreagted microdata values from model run(s) and pass each row into cvtWr().
//
// It can calculate multiple values based on the list of aggregations. All aggregated values group by the same attributes.
// Optional list of run id's can be supplied to read more than one run from output table.
// Page started at zero based offset row and up to max page size rows, if page size <= 0 then all values returned.
// Rows can be filtered and ordered (see db.ReadLayout for details).
func (mc *ModelCatalog) ReadMicrodataCalculateTo(
	dn, rdsn string, layout *db.ReadMicroLayout, calcLt *db.CalculateMicroLayout, runIds []int, cvtWr func(src interface{}) (bool, error),
) (*db.ReadPageLayout, bool) {

	// validate parameters and return empty results on empty input
	if dn == "" {
		omppLog.Log("Error: invalid (empty) model digest and name")
		return nil, false
	}
	if layout.Name == "" {
		omppLog.Log("Error: invalid (empty) model entity name")
		return nil, false
	}
	if len(calcLt.GroupBy) <= 0 {
		omppLog.Log("Error: invalid (empty) microdata group by attributes: ", dn, ": ", layout.Name)
		return nil, false
	}
	if len(calcLt.Calculation) <= 0 {
		omppLog.Log("Error: invalid (empty) microdata calculation expression(s): ", dn, ": ", layout.Name)
		return nil, false
	}

	// get model metadata and database connection
	meta, dbConn, ok := mc.modelMeta(dn)
	if !ok {
		omppLog.Log("Warning: model digest or name not found:", dn)
		return nil, false
	}

	// find entity generation by entity name
	if _, ok := meta.EntityByName(layout.Name); !ok {
		omppLog.Log("Warning: model entity not found: ", layout.Name)
		return nil, false
	}

	// if run id not defiened then find model run id by digest-or-stamp-or-name
	if layout.FromId <= 0 {

		r, ok := mc.CompletedRunByDigestOrStampOrName(dn, rdsn)
		if !ok {
			return nil, false // return empty result: run select error
		}
		if r.Status != db.DoneRunStatus {
			omppLog.Log("Warning: model run not completed successfully: ", rdsn, ": ", r.Status)
			return nil, false
		}
		layout.FromId = r.RunId // source run id
	}

	// if generation digest undefined then find entity generation by entity name and run id
	if layout.GenDigest == "" {

		_, entGen, ok := mc.EntityGenByName(dn, layout.FromId, layout.Name)
		if !ok {
			return nil, false // entity generation not found
		}
		layout.GenDigest = entGen.GenDigest
	}

	// read microdata values page
	lt, err := db.ReadMicrodataCalculateTo(dbConn, meta, layout, calcLt, runIds, cvtWr)
	if err != nil {
		omppLog.Log("Error at read microdata: ", dn, ": ", layout.Name, ": ", layout.GenDigest, ": ", err.Error())
		return nil, false // return empty result: values select error
	}

	return lt, true
}
