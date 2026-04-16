// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"crypto/md5"
	"database/sql"
	"errors"
	"fmt"
	"hash"
	"strconv"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
)

// WriteParameterFrom insert or update parameter values in workset or insert parameter values into model run until from() return not nil CellParam value.
//
// If this is model run update (layout.IsToRun is true) then model run must exist and be in completed state (i.e. success or error state).
// Model run should not already contain parameter values: parameter can be inserted only once in model run and cannot be updated after.
//
// If workset already contain parameter values then values updated else inserted.
// If only "page" of workset parameter rows supplied (layout.IsPage is true)
// then each row deleted by primary key before insert else all rows deleted by one delete by set id.
//
// Double format is used for float model types digest calculation, if non-empty format supplied.
func WriteParameterFrom(dbConn *sql.DB, modelDef *ModelMeta, layout *WriteParamLayout, from func() (interface{}, error)) error {

	// validate parameters
	if modelDef == nil {
		return errors.New("invalid (empty) model metadata, look like model not found")
	}
	if layout == nil {
		return errors.New("invalid (empty) write layout")
	}
	if layout.Name == "" {
		return errors.New("invalid (empty) parameter name")
	}
	if layout.ToId <= 0 {
		if layout.IsToRun {
			return errors.New("invalid destination run id: " + strconv.Itoa(layout.ToId))
		}
		return errors.New("invalid destination set id: " + strconv.Itoa(layout.ToId))
	}
	if layout.SubCount <= 0 {
		return errors.New("invalid number of parameter sub-vaules: " + strconv.Itoa(layout.SubCount))
	}
	if from == nil {
		return errors.New("invalid (empty) parameter values")
	}

	// find parameter id by name and default sub-value id for workset parameter
	var param *ParamMeta
	if k, ok := modelDef.ParamByName(layout.Name); ok {
		param = &modelDef.Param[k]
	} else {
		return errors.New("parameter not found: " + layout.Name)
	}

	var defSubId int = 0
	if !layout.IsToRun {
		n, defId, e := GetWorksetParam(dbConn, layout.ToId, param.ParamHid)
		if e != nil {
			return e
		}
		if n <= 0 {
			return errors.New("parameter not found: " + layout.Name + " in workset: " + strconv.Itoa(layout.ToId))
		}
		defSubId = defId
	}

	// do insert or update parameter in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	if layout.IsToRun {
		err = doWriteRunParameterFrom(trx, modelDef, param, layout.ToId, layout.SubCount, from, layout.DoubleFmt)
	} else {
		err = doWriteSetParameterFrom(trx, param, layout.ToId, layout.SubCount, defSubId, layout.IsPage, from, layout.DoubleFmt)
	}
	if err != nil {
		trx.Rollback()
		return err
	}

	trx.Commit()
	return nil
}

// doWriteRunParameterFrom insert parameter values into model run.
// It does insert as part of transaction
// Model run must exist and be in completed state (i.e. success or error state).
// Model run should not already contain parameter values: parameter can be inserted only once in model run and cannot be updated after.
// Double format is used for float model types digest calculation, if non-empty format supplied
func doWriteRunParameterFrom(
	trx *sql.Tx, modelDef *ModelMeta, param *ParamMeta, runId int, subCount int, from func() (interface{}, error), doubleFmt string,
) error {

	// update model run master record to prevent run use
	srId := strconv.Itoa(runId)
	err := TrxUpdate(trx,
		"UPDATE run_lst SET sub_restart = sub_restart - 1 WHERE run_id = "+srId)
	if err != nil {
		return err
	}

	// check if model run exist and status is completed
	st := ""
	err = TrxSelectFirst(trx,
		"SELECT status FROM run_lst WHERE run_id = "+srId,
		func(row *sql.Row) error {
			if err := row.Scan(&st); err != nil {
				return err
			}
			return nil
		})
	switch {
	case err == sql.ErrNoRows:
		return errors.New("model run not found, id: " + srId)
	case err != nil:
		return err
	}
	if !IsRunCompleted(st) {
		return errors.New("model run not completed, id: " + srId)
	}

	// check if parameter values not already exist for that run
	sHid := strconv.Itoa(param.ParamHid)
	n := 0
	err = TrxSelectFirst(trx,
		"SELECT COUNT(*) FROM run_parameter"+" WHERE run_id = "+srId+" AND parameter_hid = "+sHid,
		func(row *sql.Row) error {
			if err := row.Scan(&n); err != nil {
				return err
			}
			return nil
		})
	switch {
	case err != nil && err != sql.ErrNoRows:
		return err
	}
	if n > 0 {
		return errors.New("model run with id: " + srId + " already contain parameter values " + param.Name)
	}

	// insert into run_parameter with current run id as base run id and NULL digest
	err = TrxUpdate(trx,
		"INSERT INTO run_parameter (run_id, parameter_hid, base_run_id, sub_count, value_digest)"+
			" VALUES ("+
			srId+", "+sHid+", "+srId+", "+strconv.Itoa(subCount)+", NULL)")
	if err != nil {
		return err
	}

	// create parameter digest calculator
	hMd5, digestFrom, isOrderBy, err := digestParameterFrom(modelDef, param, doubleFmt)
	if err != nil {
		return err
	}

	// make sql to insert parameter values into model run
	// prepare put() closure to convert each cell into insert sql statement parameters
	q := makeSqlInsertParamValue(param.DbRunTable, "run_id", param.Dim, runId)
	put := putInsertParamFrom(param, subCount, 0, from, digestFrom)

	// execute sql insert using put() above for each row
	if err = TrxUpdateStatement(trx, q, put); err != nil {
		return errors.New("insert parameter failed: " + param.Name + " " + err.Error())
	}

	// for correct digest calculation rows order by must be: sub_id, dim0, dim1,....
	// if rows order by was incorrect then read parameter rows from run values table and recaluculate digest
	if isOrderBy == nil || !*isOrderBy {
		//
		// SELECT sub_id, dim0, dim1, param_value FROM ageSex_w2012_817 WHERE run_id = 1234 ORDER BY 1, 2, 3
		//
		q := "SELECT sub_id, "
		for k := range param.Dim {
			q += param.Dim[k].colName + ", "
		}
		q += "param_value FROM " + param.DbRunTable + " WHERE run_id = " + srId

		q += " ORDER BY 1"
		for k := range param.Dim {
			q += ", " + strconv.Itoa(k+2)
		}

		// select all parameter rows and re-calculate digest
		hMd5, digestFrom, _, err = digestParameterFrom(modelDef, param, doubleFmt)
		if err != nil {
			return err
		}

		err = trxReadParameterTo(trx, param, q, digestFrom)
		if err != nil {
			return errors.New("digest parameter failed: " + param.Name + " " + err.Error())
		}
	}

	// update parameter digest with actual value
	dgst := fmt.Sprintf("%x", hMd5.Sum(nil))

	err = TrxUpdate(trx,
		"UPDATE run_parameter SET value_digest = "+ToQuoted(dgst)+
			" WHERE run_id = "+srId+
			" AND parameter_hid ="+sHid)
	if err != nil {
		return err
	}

	// find base run by digest, it must exist
	nBase := 0
	err = TrxSelectFirst(trx,
		"SELECT MIN(run_id) FROM run_parameter"+
			" WHERE parameter_hid = "+sHid+
			" AND value_digest = "+ToQuoted(dgst),
		func(row *sql.Row) error {
			if err := row.Scan(&nBase); err != nil {
				return err
			}
			return nil
		})
	switch {
	// case err == sql.ErrNoRows: it must exist, at least as newly inserted row above
	case err != nil:
		return err
	}

	// if parameter values already exist then update base run id
	// and remove duplicate values
	if runId != nBase {

		err = TrxUpdate(trx,
			"UPDATE run_parameter SET base_run_id = "+strconv.Itoa(nBase)+
				" WHERE run_id = "+srId+
				" AND parameter_hid = "+sHid)
		if err != nil {
			return err
		}
		err = TrxUpdate(trx, "DELETE FROM "+param.DbRunTable+" WHERE run_id = "+srId)
		if err != nil {
			return err
		}
	}

	// completed OK, restore run_lst values
	err = TrxUpdate(trx,
		"UPDATE run_lst SET sub_restart = sub_restart + 1 WHERE run_id = "+srId)
	if err != nil {
		return err
	}
	return nil
}

// doWriteSetParameterFrom insert or update parameter values in workset.
// It does insert as part of transaction
// If workset already contain parameter values then values updated else inserted.
func doWriteSetParameterFrom(
	trx *sql.Tx, param *ParamMeta, setId int, subCount int, defaultSubId int, isPage bool, from func() (interface{}, error), doubleFmt string,
) error {

	// start workset update
	sId := strconv.Itoa(setId)
	err := TrxUpdate(trx,
		"UPDATE workset_lst"+
			" SET is_readonly = is_readonly + 1, update_dt = "+ToQuoted(helper.MakeDateTime(time.Now()))+
			" WHERE set_id = "+sId)
	if err != nil {
		return err
	}

	// check if workset exist and not readonly
	nRd := 0
	err = TrxSelectFirst(trx,
		"SELECT is_readonly FROM workset_lst WHERE set_id = "+sId,
		func(row *sql.Row) error {
			if err := row.Scan(&nRd); err != nil {
				return err
			}
			return nil
		})
	switch {
	case err == sql.ErrNoRows:
		return errors.New("workset not found, id: " + sId)
	case err != nil:
		return err
	}
	if nRd != 1 {
		return errors.New("cannot update parameter " + param.Name + ", workset is readonly, id: " + sId)
	}

	// delete existing parameter values and insert new values
	if !isPage {
		if err = TrxUpdate(trx, "DELETE FROM "+param.DbSetTable+" WHERE set_id = "+sId); err != nil {
			return err
		}

		// make sql to insert parameter values into workset
		// prepare put() closure to convert each cell into insert sql statement parameters
		sql := makeSqlInsertParamValue(param.DbSetTable, "set_id", param.Dim, setId)
		put := putInsertParamFrom(param, subCount, defaultSubId, from, nil)

		// execute sql insert using put() above for each row
		if err = TrxUpdateStatement(trx, sql, put); err != nil {
			return errors.New("insert parameter failed: " + param.Name + " " + err.Error())
		}

	} else { // page of data: parameter page updated, typically json from UI

		err = doDeleteInsertParamRows(trx, param, setId, subCount, defaultSubId, from, doubleFmt)
		if err != nil {
			return errors.New("update parameter failed: " + param.Name + " " + err.Error())
		}
	}

	// update completed: reset readonly status to "read-write"
	err = TrxUpdate(trx, "UPDATE workset_lst SET is_readonly = 0 WHERE set_id = "+sId)
	if err != nil {
		return err
	}
	return nil
}

// make sql to insert parameter values into model run or workset
func makeSqlInsertParamValue(dbTable string, runSetCol string, dims []ParamDimsRow, toId int) string {

	// INSERT INTO ageSex_w2012817 (set_id, sub_id, dim0, dim1, param_value) VALUES (2, ?, ?, ?, ?)
	q := "INSERT INTO " + dbTable +
		" (" + runSetCol + ", sub_id, "

	for k := range dims {
		q += dims[k].colName + ", "
	}

	q += "param_value) VALUES (" + strconv.Itoa(toId) + ", ?, "

	for k := 0; k < len(dims); k++ {
		q += "?, "
	}
	q += "?)"

	return q
}

// prepare put() closure to convert each cell into insert sql statement parameters until from() return not nil CellParam value.
func putInsertParamFrom(
	param *ParamMeta, subCount int, defaultSubId int, from func() (interface{}, error), digestFrom func(interface{}) error,
) func() (bool, []interface{}, error) {

	// converter from value into db value
	fv := cvtToSqlDbValue(param.IsExtendable, param.typeOf, param.Name)

	// for each cell put into row of sql statement parameters
	row := make([]interface{}, param.Rank+2)

	put := func() (bool, []interface{}, error) {

		// get next input row
		c, err := from()
		if err != nil {
			return false, nil, err
		}
		if c == nil {
			return false, nil, nil // end of data
		}

		// convert and check input row
		cell, ok := c.(CellParam)
		if !ok {
			return false, nil, errors.New("invalid type, expected: parameter cell (internal error)")
		}

		n := len(cell.DimIds)
		if len(row) != n+2 {
			return false, nil, errors.New("invalid size of row buffer, expected: " + strconv.Itoa(n+2))
		}

		// parameter sub-value id and default sub-value id can be any integer
		// however if workset created by openM++ tools (e.g. by omc) then default id =0 and sub-value ids: [0,subCount-1]
		// to validate allow non-zero based sub id only for single sub-values set
		if defaultSubId == 0 && (cell.SubId < 0 || cell.SubId >= subCount) || subCount == 1 && cell.SubId != defaultSubId {
			return false, nil, errors.New("invalid sub-value id: " + strconv.Itoa(cell.SubId) + " parameter: " + param.Name)
		}

		// set sql statement parameter values: sub-value number, dimensions enum, parameter value
		row[0] = cell.SubId

		for k, e := range cell.DimIds {
			row[k+1] = e
		}
		if v, err := fv(cell.IsNull, cell.Value); err == nil {
			row[n+1] = v
		} else {
			return false, nil, err
		}

		// if parameter digest required then append row digest to parameter value digest
		if digestFrom != nil {
			err = digestFrom(cell)
			if err != nil {
				return false, nil, err
			}
		}

		return true, row, nil // return current row to sql statement
	}
	return put
}

// digestParameterFrom start run parameter digest calculation and return closure to add parameter row to digest.
func digestParameterFrom(modelDef *ModelMeta, param *ParamMeta, doubleFmt string) (hash.Hash, func(interface{}) error, *bool, error) {

	// start from name and metadata digest
	hMd5 := md5.New()
	_, err := hMd5.Write([]byte("parameter_name,parameter_digest\n"))
	if err != nil {
		return nil, nil, nil, err
	}
	_, err = hMd5.Write([]byte(param.Name + "," + param.Digest + "\n"))
	if err != nil {
		return nil, nil, nil, err
	}

	// create parameter row digester append digest of parameter cells
	cvtParam := &CellParamConverter{
		ModelDef:  modelDef,
		Name:      param.Name,
		IsIdCsv:   true,
		DoubleFmt: doubleFmt,
	}

	digestRow, isOrderBy, err := digestIntKeysCellsFrom(hMd5, modelDef, param.Name, cvtParam)
	if err != nil {
		return nil, nil, nil, err
	}

	return hMd5, digestRow, isOrderBy, nil
}

// delete and insert parameter rows from workset by specified keys: set id, sub-value number and dimension(s) items id
func doDeleteInsertParamRows(
	trx *sql.Tx, param *ParamMeta, setId int, subCount int, defaultSubId int, from func() (interface{}, error), doubleFmt string,
) error {

	// start of delete sql:
	// DELETE FROM ageSex_w2012817 WHERE set_id = 2 AND sub_id = 0 AND dim0 = 1 AND dim1 = 2
	del := "DELETE FROM " + param.DbSetTable + " WHERE set_id = " + strconv.Itoa(setId)

	// start of insert sql:
	// INSERT INTO ageSex_w2012817 (set_id, sub_id, dim0, dim1, param_value) VALUES (2, 0, 1, 2, 'QC')
	ins := "INSERT INTO " + param.DbSetTable + " (set_id, sub_id, "

	dimCount := len(param.Dim)
	for k := 0; k < dimCount; k++ {
		ins += param.Dim[k].colName + ", "
	}

	ins += "param_value) VALUES (" + strconv.Itoa(setId) + ", "

	// converter from value into sql string to insert, if parameter type is string the return is 'sql quoted value'
	fv := cvtParamValueToSqlString(param, doubleFmt)

	for {
		// get next input row
		c, err := from()
		if err != nil {
			return err
		}
		if c == nil {
			break // end of data
		}

		// convert and check input row
		cell, ok := c.(CellParam)
		if !ok {
			return errors.New("invalid type, expected: parameter cell (internal error)")
		}

		n := len(cell.DimIds)
		if n != dimCount {
			return errors.New("invalid parameter row dimensions count, expected: " + strconv.Itoa(dimCount))
		}

		// make delete sql: add sub-value number and dimensions enum
		delSql := del + " AND sub_id = " + strconv.Itoa(cell.SubId)

		for k := 0; k < dimCount; k++ {
			delSql += " AND " + param.Dim[k].colName + " = " + strconv.Itoa(cell.DimIds[k])
		}

		// parameter sub-value id and default sub-value id can be any integer
		// however if workset created by openM++ tools (e.g. by omc) then default id =0 and sub-value ids: [0,subCount-1]
		// to validate allow non-zero based sub id only for single sub-values set
		if defaultSubId == 0 && (cell.SubId < 0 || cell.SubId >= subCount) || subCount == 1 && cell.SubId != defaultSubId {
			return errors.New("invalid sub-value id: " + strconv.Itoa(cell.SubId) + " parameter: " + param.Name)
		}

		// make insert sql: add sub-value number, dimensions enum, parameter value
		insSql := ins + strconv.Itoa(cell.SubId) + ", "

		for k := 0; k < dimCount; k++ {
			insSql += strconv.Itoa(cell.DimIds[k]) + ", "
		}

		if v, err := fv(cell); err == nil {
			insSql += v + ")"
		} else {
			return err
		}

		// do delete and insert
		if err = TrxUpdate(trx, delSql); err != nil {
			return err
		}
		if err = TrxUpdate(trx, insSql); err != nil {
			return err
		}
	}

	return nil
}

// cvtParamValueToSqlString return converter from value into sql string to insert
// if parameter type is string then return is 'sql quoted value'
// converter does type validation
// for non built-it types validate enum id presense in enum list
// only float parameter values can be NULL, for any other parameter types NULL values rejected.
func cvtParamValueToSqlString(param *ParamMeta, doubleFmt string) func(cell CellParam) (string, error) {

	// float parameter: check if isNull flag, validate and convert type
	// cell value is nullable for extended parameters only
	isNullable := param.IsExtendable

	isUseFmt := param.typeOf.IsFloat() && doubleFmt != "" // for float model types use format if specified
	isUseEnum := !param.typeOf.IsBuiltIn()                // parameter is enum-based: validate enum id, it must be in enum type
	isBool := param.typeOf.IsBool()                       // boolean sql values are 0 or 1
	isStr := param.typeOf.IsString()                      // string value must 'sql quoted'

	cvt := func(cell CellParam) (string, error) {

		// validate if cell value is null and parameter value can be NULL then retun "NULL" string
		if cell.IsNull || cell.Value == nil {
			if !isNullable {
				return "", errors.New("invalid parameter value, it cannot be NULL")
			}
			return "NULL", nil
		}
		if isBool {
			if is, ok := cell.Value.(bool); ok && is {
				return "1", nil
			}
			return "0", nil
		}
		if isStr {
			if sv, ok := cell.Value.(string); !ok {
				return "", errors.New("invalid parameter value, it must be string")
			} else {
				return toQuotedMax(sv, stringDbMax), nil
			}
		}

		// if parameter type is enum based then validate enum id
		if isUseEnum {
			iv, ok := helper.ToIntValue(cell.Value)
			if !ok {
				return "", errors.New("invalid parameter value, expected: integer enum id")
			}

			// validate enum id: it must be in enum list or in the range
			if !param.typeOf.IsRange {
				for j := range param.typeOf.Enum {
					if iv == param.typeOf.Enum[j].EnumId {
						return strconv.Itoa(iv), nil
					}
				}
			} else {
				if param.typeOf.MinEnumId <= iv && iv <= param.typeOf.MaxEnumId {
					return strconv.Itoa(iv), nil
				}
			}
			return "", errors.New("invalid parameter value type, enum id not found: " + strconv.Itoa(iv))
		}

		// format integer and for model float types
		if isUseFmt {
			return fmt.Sprintf(doubleFmt, cell.Value), nil
		} else {
			return fmt.Sprint(cell.Value), nil
		}
	}

	return cvt
}
