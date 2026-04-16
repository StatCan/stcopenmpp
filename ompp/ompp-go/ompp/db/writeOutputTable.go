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
)

// WriteOutputTableFrom insert output table values (accumulators or expressions) into model run from accFrom() and exprFrom() readers.
//
// Model run must exist and be in completed state (i.e. success or error state).
// Model run should not already contain output table values: it can be inserted only once in model run and cannot be updated after.
// Accumulators and expressions values must come in the order of primary key otherwise digest calculated incorrectly.
// Double format is used for float model types digest calculation, if non-empty format supplied.
func WriteOutputTableFrom(
	dbConn *sql.DB, modelDef *ModelMeta, layout *WriteTableLayout, accFrom func() (interface{}, error), exprFrom func() (interface{}, error),
) error {

	// validate parameters
	if modelDef == nil {
		return errors.New("invalid (empty) model metadata, look like model not found")
	}
	if layout == nil {
		return errors.New("invalid (empty) write layout")
	}
	if layout.Name == "" {
		return errors.New("invalid (empty) output table name")
	}
	if layout.ToId <= 0 {
		return errors.New("invalid destination run id: " + strconv.Itoa(layout.ToId))
	}
	if accFrom == nil || exprFrom == nil {
		return errors.New("invalid (empty) output table values")
	}

	// find output table id by name
	var meta *TableMeta
	if k, ok := modelDef.OutTableByName(layout.Name); ok {
		meta = &modelDef.Table[k]
	} else {
		return errors.New("output table not found: " + layout.Name)
	}

	// do insert or update output table in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	if err = doWriteOutputTableFrom(trx, modelDef, meta, layout.ToId, layout.DoubleFmt, accFrom, exprFrom); err != nil {
		trx.Rollback()
		return err
	}

	trx.Commit()
	return nil
}

// doWriteOutputTableFrom insert output table values (accumulators or expressions) into model run.
// It does insert as part of transaction
// Model run must exist and be in completed state (i.e. success or error state).
// Model run should not already contain output table values: it can be inserted only once in model run and cannot be updated after.
// Double format is used for float model types digest calculation, if non-empty format supplied
func doWriteOutputTableFrom(
	trx *sql.Tx, modelDef *ModelMeta, meta *TableMeta, runId int, doubleFmt string, accFrom func() (interface{}, error), exprFrom func() (interface{}, error),
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

	// check if output table values not already exist for that run
	sHid := strconv.Itoa(meta.TableHid)
	n := 0
	err = TrxSelectFirst(trx,
		"SELECT COUNT(*) FROM run_table"+" WHERE run_id = "+srId+" AND table_hid = "+sHid,
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
		return errors.New("model run with id: " + srId + " already contain output table values " + meta.Name)
	}

	// insert into run_table with current run id as base run id
	err = TrxUpdate(trx,
		"INSERT INTO run_table (run_id, table_hid, base_run_id, value_digest)"+
			" VALUES ("+
			srId+", "+sHid+", "+srId+", NULL)")
	if err != nil {
		return err
	}

	// create output table digest calculator and start accumulator(s) digest
	hMd5, digestAcc, isOrderBy, err := digestAccumulatorsFrom(modelDef, meta, doubleFmt)
	if err != nil {
		return err
	}

	// insert output table accumulators into model run
	// prepare put() closure to convert each accumulator cell into parameters of insert sql statement
	accSql := makeSqlAccValueInsert(meta, runId)
	put := putAccInsertFrom(meta, accFrom, digestAcc)

	if err = TrxUpdateStatement(trx, accSql, put); err != nil {
		return err
	}
	// check if all rows ordered by primary key, digest is incorrect otherwise
	if isOrderBy == nil || !*isOrderBy {
		return errors.New("invalid digest due to incorrect accumulator(s) rows order: " + meta.Name)
	}

	// start expression(s) digest calculation
	digestExpr, isOrderBy, err := digestExpressionsFrom(modelDef, meta, doubleFmt, hMd5)
	if err != nil {
		return err
	}

	// insert output table expressions into model run
	// prepare put() closure to convert each expression cell into parameters of insert sql statement
	exprSql := makeSqlExprValueInsert(meta, runId)
	put = putExprInsertFrom(meta, exprFrom, digestExpr)

	if err = TrxUpdateStatement(trx, exprSql, put); err != nil {
		return err
	}
	// check if all rows ordered by primary key, digest is incorrect otherwise
	if isOrderBy == nil || !*isOrderBy {
		return errors.New("invalid digest due to incorrect expression(s) rows order: " + meta.Name)
	}

	// update output table digest with actual value
	dgst := fmt.Sprintf("%x", hMd5.Sum(nil))

	err = TrxUpdate(trx,
		"UPDATE run_table SET value_digest = "+ToQuoted(dgst)+
			" WHERE run_id = "+srId+
			" AND table_hid ="+sHid)
	if err != nil {
		return err
	}

	// find base run by digest, it must exist
	nBase := 0
	err = TrxSelectFirst(trx,
		"SELECT MIN(run_id) FROM run_table"+
			" WHERE table_hid = "+sHid+
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

	// if output table values already exist then update base run id
	// and remove duplicate values
	if runId != nBase {

		err = TrxUpdate(trx,
			"UPDATE run_table SET base_run_id = "+strconv.Itoa(nBase)+
				" WHERE run_id = "+srId+
				" AND table_hid = "+sHid)
		if err != nil {
			return err
		}
		err = TrxUpdate(trx, "DELETE FROM "+meta.DbExprTable+" WHERE run_id = "+srId)
		if err != nil {
			return err
		}
		err = TrxUpdate(trx, "DELETE FROM "+meta.DbAccTable+" WHERE run_id = "+srId)
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

// digestAccumulatorsFrom start output table digest calculation and return closure to add accumulator(s) row to digest.
func digestAccumulatorsFrom(
	modelDef *ModelMeta, meta *TableMeta, doubleFmt string,
) (hash.Hash, func(interface{}) error, *bool, error) {

	// start from name and metadata digest
	hMd5 := md5.New()
	_, err := hMd5.Write([]byte("table_name,table_digest\n"))
	if err != nil {
		return nil, nil, nil, err
	}
	_, err = hMd5.Write([]byte(meta.Name + "," + meta.Digest + "\n"))
	if err != nil {
		return nil, nil, nil, err
	}

	// create accumulator(s) row digester append digest of accumulator(s) cells
	cvtAcc := &CellAccConverter{
		CellTableConverter: CellTableConverter{
			ModelDef:  modelDef,
			Name:      meta.Name,
			IsIdCsv:   true,
			DoubleFmt: doubleFmt,
		},
	}

	digestRow, isOrderBy, err := digestIntKeysCellsFrom(hMd5, modelDef, meta.Name, cvtAcc)
	if err != nil {
		return nil, nil, nil, err
	}

	return hMd5, digestRow, isOrderBy, nil
}

// digestExpressionsFrom append output expression(s) header to digest and return closure to add expression(s) row to digest.
func digestExpressionsFrom(
	modelDef *ModelMeta, meta *TableMeta, doubleFmt string, hSum hash.Hash,
) (func(interface{}) error, *bool, error) {

	// create expression(s) row digester append digest of expression(s) cells
	// append digest of expression(s) cells
	cvtExpr := &CellExprConverter{
		CellTableConverter: CellTableConverter{
			ModelDef:  modelDef,
			Name:      meta.Name,
			IsIdCsv:   true,
			DoubleFmt: doubleFmt,
		},
	}

	digestRow, isOrderBy, err := digestIntKeysCellsFrom(hSum, modelDef, meta.Name, cvtExpr)
	if err != nil {
		return nil, nil, err
	}

	return digestRow, isOrderBy, nil
}

// make sql to insert output table expressions into model run
func makeSqlExprValueInsert(meta *TableMeta, runId int) string {

	// INSERT INTO salarySex_v2012820
	//   (run_id, expr_id, dim0, dim1, expr_value)
	// VALUES
	//   (2, ?, ?, ?, ?)
	q := "INSERT INTO " + meta.DbExprTable + " (run_id, expr_id, "
	for k := range meta.Dim {
		q += meta.Dim[k].colName + ", "
	}

	q += "expr_value) VALUES (" + strconv.Itoa(runId) + ", ?, "

	for k := 0; k < len(meta.Dim); k++ {
		q += "?, "
	}
	q += "?)"

	return q
}

// prepare put() closure to convert each cell of output expression into parameters of insert sql statement until from() return not nil CellExpr value.
func putExprInsertFrom(
	meta *TableMeta, from func() (interface{}, error), digestFrom func(interface{}) error,
) func() (bool, []interface{}, error) {

	// for each cell of output expressions put into row of sql statement parameters
	row := make([]interface{}, meta.Rank+2)

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
		cell, ok := c.(CellExpr)
		if !ok {
			return false, nil, errors.New("invalid type, expected: output table expression cell (internal error)")
		}

		n := len(cell.DimIds)
		if len(row) != n+2 {
			return false, nil, errors.New("invalid output table expression row size, expected: " + strconv.Itoa(n+2))
		}

		// set sql statement parameter values: expression id, dimensions enum
		row[0] = cell.ExprId

		for k, e := range cell.DimIds {
			row[k+1] = e
		}

		// cell value is nullable
		row[n+1] = sql.NullFloat64{Float64: cell.Value.(float64), Valid: !cell.IsNull}

		// append row digest to output table digest
		err = digestFrom(cell)
		if err != nil {
			return false, nil, err
		}

		return true, row, nil // return current row to sql statement
	}

	return put
}

// make sql to insert output table accumulators into model run
func makeSqlAccValueInsert(meta *TableMeta, runId int) string {

	// INSERT INTO salarySex_a2012820
	//   (run_id, acc_id, sub_id, dim0, dim1, acc_value)
	// VALUES
	//   (2, ?, ?, ?, ?, ?)
	q := "INSERT INTO " + meta.DbAccTable + " (run_id, acc_id, sub_id, "
	for k := range meta.Dim {
		q += meta.Dim[k].colName + ", "
	}

	q += "acc_value) VALUES (" + strconv.Itoa(runId) + ", ?, ?, "

	for k := 0; k < len(meta.Dim); k++ {
		q += "?, "
	}
	q += "?)"

	return q
}

// prepare put() closure to convert each cell of accumulators into parameters of insert sql statement until from() return not nil CellAcc value.
func putAccInsertFrom(
	meta *TableMeta, from func() (interface{}, error), digestFrom func(interface{}) error,
) func() (bool, []interface{}, error) {

	// for each cell of accumulators put into row of sql statement parameters
	row := make([]interface{}, meta.Rank+3)

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
		cell, ok := c.(CellAcc)
		if !ok {
			return false, nil, errors.New("invalid type, expected: output table accumulator cell (internal error)")
		}

		n := len(cell.DimIds)
		if len(row) != n+3 {
			return false, nil, errors.New("invalid output accumulator table row size, expected: " + strconv.Itoa(n+3))
		}

		// set sql statement parameter values: accumulator id and subvalue number, dimensions enum
		row[0] = cell.AccId
		row[1] = cell.SubId

		for k, e := range cell.DimIds {
			row[k+2] = e
		}

		// cell value is nullable
		row[n+2] = sql.NullFloat64{Float64: cell.Value.(float64), Valid: !cell.IsNull}

		// append row digest to output table digest
		err = digestFrom(cell)
		if err != nil {
			return false, nil, err
		}

		return true, row, nil // return current row to sql statement
	}

	return put
}
