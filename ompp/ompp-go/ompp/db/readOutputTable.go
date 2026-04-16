// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"errors"
	"strconv"
)

// ReadOutputTableTo read output table page (dimensions and values) from model run results and process each row by cvtTo().
//
// If layout.IsAccum true then select accumulator(s) else output expression value(s)
// If layout.ValueName not empty then select only that expression (accumulator) else all expressions (accumulators)
func ReadOutputTableTo(dbConn *sql.DB, modelDef *ModelMeta, layout *ReadTableLayout, cvtTo func(src interface{}) (bool, error)) (*ReadPageLayout, error) {

	// validate parameters
	if modelDef == nil {
		return nil, errors.New("invalid (empty) model metadata, look like model not found")
	}
	if layout == nil {
		return nil, errors.New("invalid (empty) output table read layout")
	}
	if layout.Name == "" {
		return nil, errors.New("invalid (empty) output table name")
	}

	// find output table id by name
	var table *TableMeta
	if k, ok := modelDef.OutTableByName(layout.Name); ok {
		table = &modelDef.Table[k]
	} else {
		return nil, errors.New("output table not found: " + layout.Name)
	}

	// find expression or accumulator id by name
	// if this is select from all accumulators view then find db internal column name
	valId := -1
	valAccCol := ""

	if layout.ValueName != "" {

		if layout.IsAccum { // find accumulator

			for i := range table.Acc {
				if table.Acc[i].Name == layout.ValueName {
					valId = table.Acc[i].AccId
					valAccCol = table.Acc[i].colName
				}
			}
			if valId < 0 || valAccCol == "" {
				return nil, errors.New("output table accumulator not found: " + layout.Name + " " + layout.ValueName)
			}

		} else { // find expression

			for i := range table.Expr {
				if table.Expr[i].Name == layout.ValueName {
					valId = table.Expr[i].ExprId
				}
			}
			if valId < 0 {
				return nil, errors.New("output table expression not found: " + layout.Name + " " + layout.ValueName)
			}
		}
	}

	// number of accumulator value columns: acc_value or acc0, acc1, acc2...
	accCount := 1
	if layout.IsAccum && layout.IsAllAccum && layout.ValueName == "" {
		accCount = len(table.Acc)
	}

	// check if model run exist and model run completed
	runRow, err := GetRun(dbConn, layout.FromId)
	if err != nil {
		return nil, err
	}
	if runRow == nil {
		return nil, errors.New("model run not found, id: " + strconv.Itoa(layout.FromId))
	}
	if runRow.Status != DoneRunStatus {
		return nil, errors.New("model run not completed successfully, id: " + strconv.Itoa(layout.FromId))
	}

	// make sql to select output table expression(s) from model run:
	//
	//   SELECT expr_id, dim0, dim1, expr_value
	//   FROM salarySex_v2012_820
	//   WHERE run_id =
	//   (
	//     SELECT base_run_id FROM run_table WHERE run_id = 2 AND table_hid = 12345
	//   )
	//   AND expr_id = 3
	//   AND dim1 IN (10, 20, 30, 40)
	//   ORDER BY 1, 2, 3
	//
	// or accumulator(s):
	//
	//   SELECT acc_id, sub_id, dim0, dim1, acc_value
	//   FROM salarySex_a2012_820
	//   WHERE run_id =
	//   (
	//     SELECT base_run_id FROM run_table WHERE run_id = 2 AND table_hid = 12345
	//   )
	//   AND acc_id = 4
	//   AND sub_id = 7
	//   AND dim1 IN (10, 20, 30, 40)
	//   ORDER BY 1, 2, 3, 4
	//
	// or all accumulators view:
	//
	//   WITH va1 AS
	//   (
	//     SELECT
	//       run_id, sub_id, dim0, dim1, acc_value
	//     FROM salarySex_a_2012820
	//     WHERE acc_id = 1
	//   ),
	//   v_all_acc AS
	//   (
	//     SELECT
	//       A.run_id,
	//       A.sub_id,
	//       A.dim0,
	//       A.dim1,
	//       A.acc_value  AS acc0,
	//       A1.acc_value AS acc1,
	//       (
	//         A.acc_value / CASE WHEN ABS(A1.acc_value) > 1.0e-37 THEN A1.acc_value ELSE NULL END
	//       ) AS expr0
	//     FROM salarySex_a_2012820 A
	//     INNER JOIN va1 A1 ON (A1.run_id = A.run_id AND A1.sub_id = A.sub_id AND A1.dim0 = A.dim0 AND A1.dim1 = A.dim1)
	//     WHERE A.acc_id = 0
	//   )
	//   SELECT
	//     run_id, sub_id, dim0, dim1, acc0, acc1, expr0
	//   FROM va_all_acc
	//   WHERE run_id =
	//   (
	//     SELECT base_run_id FROM run_table WHERE run_id = 2 AND table_hid = 12345
	//   )
	//   AND sub_id = 7
	//   AND dim1 IN (10, 20, 30, 40)
	//   ORDER BY 1, 2, 3, 4
	//
	q := ""
	if layout.IsAllAccum {
		q = sqlAccAllViewAsWith(table) + " "
	}

	q += "SELECT"

	if layout.IsAccum {
		if !layout.IsAllAccum {
			q += " acc_id,"
		}
		q += " sub_id"
	} else {
		q += " expr_id"
	}

	for k := range table.Dim {
		q += ", " + table.Dim[k].colName
	}

	if !layout.IsAccum {
		q += ", expr_value FROM " + table.DbExprTable
	} else {
		if !layout.IsAllAccum {
			q += ", acc_value FROM " + table.DbAccTable
		} else {
			if valAccCol != "" {
				q += ", " + valAccCol
			} else {
				for k := range table.Acc {
					q += ", " + table.Acc[k].colName
				}
			}
			q += " FROM v_all_acc"
		}
	}

	q += " WHERE run_id =" +
		" (SELECT base_run_id FROM run_table" +
		" WHERE run_id = " + strconv.Itoa(layout.FromId) +
		" AND table_hid = " + strconv.Itoa(table.TableHid) + ")"

	if !layout.IsAllAccum && valId >= 0 {
		if layout.IsAccum {
			q += " AND acc_id = " + strconv.Itoa(valId)
		} else {
			q += " AND expr_id = " + strconv.Itoa(valId)
		}
	}

	// append sub-value id filter
	if layout.IsAccum && layout.IsSubId {
		q += " AND sub_id = " + strconv.Itoa(layout.SubId)
	}

	// append dimension enum code filters, if specified
	iDbl, ok := modelDef.TypeOfDouble()
	if !ok {
		return nil, errors.New("double type not found, output table " + table.Name)
	}

	for k := range layout.Filter {

		// filter by expression value or accumulator value or find dimension index by name
		var err error
		f := ""

		if !layout.IsAccum {

			eix := -1
			for j := range table.Expr {
				if table.Expr[j].Name == layout.Filter[k].Name {
					eix = j
					break
				}
			}
			if eix >= 0 {
				f, err = makeWhereValueFilter(
					&layout.Filter[k], "", "expr_value", "expr_id", table.Expr[eix].ExprId, &modelDef.Type[iDbl], layout.Filter[k].Name, "output table "+table.Name)
				if err != nil {
					return nil, err
				}
			}
		} else {

			aix := -1
			for j := range table.Acc {
				if (!table.Acc[j].IsDerived || layout.IsAllAccum) && table.Acc[j].Name == layout.Filter[k].Name {
					aix = j
					break
				}
			}
			if aix >= 0 {
				if !layout.IsAllAccum {

					f, err = makeWhereValueFilter(
						&layout.Filter[k], "", "acc_value", "acc_id", table.Acc[aix].AccId, &modelDef.Type[iDbl], layout.Filter[k].Name, "output table "+table.Name)
					if err != nil {
						return nil, err
					}
				} else {

					f, err = makeWhereFilter(
						&layout.Filter[k], "", table.Acc[aix].Name, &modelDef.Type[iDbl], false, layout.Filter[k].Name, "output table "+table.Name)
					if err != nil {
						return nil, err
					}
				}
			}
		}
		if f == "" { // if not a filter by value then it must be filter by dimension

			dix := -1
			for j := range table.Dim {
				if table.Dim[j].Name == layout.Filter[k].Name {
					dix = j
					break
				}
			}
			if dix < 0 {
				return nil, errors.New("output table " + table.Name + " does not have dimension " + layout.Filter[k].Name)
			}

			f, err = makeWhereFilter(
				&layout.Filter[k], "", table.Dim[dix].colName, table.Dim[dix].typeOf, table.Dim[dix].IsTotal, table.Dim[dix].Name, "output table "+table.Name)
			if err != nil {
				return nil, err
			}
		}

		q += " AND " + f
	}

	// append dimension enum id filters, if specified
	for k := range layout.FilterById {

		// find dimension index by name
		dix := -1
		for j := range table.Dim {
			if table.Dim[j].Name == layout.FilterById[k].Name {
				dix = j
				break
			}
		}
		if dix < 0 {
			return nil, errors.New("output table " + table.Name + " does not have dimension " + layout.FilterById[k].Name)
		}

		f, err := makeWhereIdFilter(
			&layout.FilterById[k], "", table.Dim[dix].colName, table.Dim[dix].typeOf, table.Dim[dix].Name, "output table "+table.Name)
		if err != nil {
			return nil, err
		}

		q += " AND " + f
	}

	// append order by expr_id or acc_id, sub_id or sub_id
	nExtraCol := 1
	if layout.IsAccum && !layout.IsAllAccum {
		nExtraCol = 2 // extra columns: acc_id, sub_id
	}
	q += makeOrderBy(table.Rank, layout.OrderBy, nExtraCol)

	// prepare db-row conversion buffer:
	// acc_id, sub_id, expr_id, dimensions, value or []values
	var n1, n2 int
	d := make([]int, table.Rank)
	var vf sql.NullFloat64
	fa := make([]sql.NullFloat64, accCount)
	var scanBuf []interface{}

	scanBuf = append(scanBuf, &n1)
	if layout.IsAccum && !layout.IsAllAccum {
		scanBuf = append(scanBuf, &n2)
	}
	for k := 0; k < table.Rank; k++ {
		scanBuf = append(scanBuf, &d[k])
	}
	if !layout.IsAccum || !layout.IsAllAccum {
		scanBuf = append(scanBuf, &vf)
	} else {
		for k := 0; k < accCount; k++ {
			scanBuf = append(scanBuf, &fa[k])
		}
	}

	// make new cell from conversion buffer
	makeCell := func() interface{} {

		if layout.IsAccum {

			if !layout.IsAllAccum {
				var ca = CellAcc{cellIdValue: cellIdValue{DimIds: make([]int, table.Rank)}}
				ca.AccId = n1
				ca.SubId = n2
				copy(ca.DimIds, d)
				ca.IsNull = !vf.Valid
				ca.Value = 0.0
				if !ca.IsNull {
					ca.Value = vf.Float64
				}
				return ca // return accumulator cell
			}
			// else all accumulators

			var cl = CellAllAcc{
				DimIds: make([]int, table.Rank),
				IsNull: make([]bool, accCount),
				Value:  make([]float64, accCount)}

			cl.SubId = n1
			copy(cl.DimIds, d)

			for k := 0; k < accCount; k++ {
				cl.IsNull[k] = !fa[k].Valid
				cl.Value[k] = 0.0
				if !cl.IsNull[k] {
					cl.Value[k] = fa[k].Float64
				}
			}
			return cl // return all-accumulators cell
		}
		// else output table expression

		var ce = CellExpr{cellIdValue: cellIdValue{DimIds: make([]int, table.Rank)}}
		ce.ExprId = n1
		copy(ce.DimIds, d)
		ce.IsNull = !vf.Valid
		ce.Value = 0.0
		if !ce.IsNull {
			ce.Value = vf.Float64
		}
		return ce // return table expression cell
	}

	// if full page requested:
	// select rows into the list buffer and write rows from the list into output stream
	if layout.IsFullPage {

		// make a list of output cells
		cLst, lt, e := SelectToList(dbConn, q, layout.ReadPageLayout,
			func(rows *sql.Rows) (interface{}, error) {

				if e := rows.Scan(scanBuf...); e != nil {
					return nil, e
				}
				return makeCell(), nil
			})
		if e != nil {
			return nil, e
		}

		// write page into output stream
		for c := cLst.Front(); c != nil; c = c.Next() {

			if _, e := cvtTo(c.Value); e != nil {
				return nil, e
			}
		}

		return lt, nil // done: return output page layout
	}
	// else: select rows and write it into output stream without buffering

	// adjust page layout: starting offset and page size
	nStart := layout.Offset
	if nStart < 0 {
		nStart = 0
	}
	nSize := layout.Size
	if nSize < 0 {
		nSize = 0
	}
	var nRow int64

	lt := ReadPageLayout{
		Offset:     nStart,
		Size:       0,
		IsLastPage: false,
	}

	// select cells:
	// expr_id or or sub_id or acc_id and sub_id, dimension(s) enum ids
	// value or all accumulator values and null status
	err = SelectRowsTo(dbConn, q,
		func(rows *sql.Rows) (bool, error) {

			// if page size is limited then select only a page of rows
			nRow++
			if nSize > 0 && nRow > nStart+nSize {
				return false, nil
			}
			if nRow <= nStart {
				return true, nil
			}

			// select next row
			if e := rows.Scan(scanBuf...); e != nil {
				return false, e
			}
			lt.Size++

			// make new cell from scan conversion buffer and pass it to the writer
			return cvtTo(makeCell())
		})
	if err != nil {
		return nil, err
	}

	// check for the empty result page or last page
	if lt.Size <= 0 {
		lt.Offset = nRow
	}
	lt.IsLastPage = nSize <= 0 || nSize > 0 && nRow <= nStart+nSize

	return &lt, nil
}

// ReadOutputTableCalculteTo read calculated output table page (run id, expression(s) id, dimensions and values) and process each row by cvtTo().
//
// It can calculate multiple values based on expressions and/or accumulators aggregation.
// Optional runIds list of run id's can be supplied to read more than one run from output table.
func ReadOutputTableCalculteTo(
	dbConn *sql.DB, modelDef *ModelMeta, layout *ReadTableLayout, calcLt []CalculateTableLayout, runIds []int, cvtTo func(src interface{}) (bool, error),
) (*ReadPageLayout, error) {

	// validate parameters
	if modelDef == nil {
		return nil, errors.New("invalid (empty) model metadata, look like model not found")
	}
	if layout == nil {
		return nil, errors.New("invalid (empty) output table read layout")
	}
	if layout.Name == "" {
		return nil, errors.New("invalid (empty) output table name")
	}
	if len(calcLt) <= 0 {
		return nil, errors.New("invalid (empty) output table calculation expression(s): " + layout.Name)
	}

	// find output table id by name
	var table *TableMeta
	if k, ok := modelDef.OutTableByName(layout.Name); ok {
		table = &modelDef.Table[k]
	} else {
		return nil, errors.New("output table not found: " + layout.Name)
	}

	// make sql to select calculated output table expression(s) from model run(s)
	q, err := translateTableCalcToSql(modelDef, table, &layout.ReadLayout, calcLt, runIds)
	if err != nil {
		return nil, err
	}

	// prepare db-row scan conversion buffer: run_id, expression id, dimensions, value
	var runId int
	var calcId int
	d := make([]int, table.Rank)
	var vf sql.NullFloat64
	var scanBuf []interface{}

	scanBuf = append(scanBuf, &runId)
	scanBuf = append(scanBuf, &calcId)

	for k := 0; k < table.Rank; k++ {
		scanBuf = append(scanBuf, &d[k])
	}
	scanBuf = append(scanBuf, &vf)

	// make new cell from conversion buffer
	makeCell := func() interface{} {
		c := CellTableCalc{
			cellIdValue: cellIdValue{DimIds: make([]int, table.Rank)},
			CalcId:      calcId,
			RunId:       runId,
		}
		copy(c.DimIds, d)
		c.IsNull = !vf.Valid
		c.Value = 0.0
		if !c.IsNull {
			c.Value = vf.Float64
		}
		return c
	}

	// if full page requested:
	// select rows into the list buffer and write rows from the list into output stream
	if layout.IsFullPage {

		// make a list of output cells
		cLst, lt, e := SelectToList(dbConn, q, layout.ReadPageLayout,
			func(rows *sql.Rows) (interface{}, error) {

				if e := rows.Scan(scanBuf...); e != nil {
					return nil, e
				}
				return makeCell(), nil
			})
		if e != nil {
			return nil, e
		}

		// write page into output stream
		for c := cLst.Front(); c != nil; c = c.Next() {

			if _, e := cvtTo(c.Value); e != nil {
				return nil, e
			}
		}

		return lt, nil // done: return output page layout
	}
	// else: select rows and write it into output stream without buffering

	// adjust page layout: starting offset and page size
	nStart := layout.Offset
	if nStart < 0 {
		nStart = 0
	}
	nSize := layout.Size
	if nSize < 0 {
		nSize = 0
	}
	var nRow int64

	lt := ReadPageLayout{
		Offset:     nStart,
		Size:       0,
		IsLastPage: false,
	}

	// select cells:
	// expr_id or or sub_id or acc_id and sub_id, dimension(s) enum ids
	// value or all accumulator values and null status
	err = SelectRowsTo(dbConn, q,
		func(rows *sql.Rows) (bool, error) {

			// if page size is limited then select only a page of rows
			nRow++
			if nSize > 0 && nRow > nStart+nSize {
				return false, nil
			}
			if nRow <= nStart {
				return true, nil
			}

			// select next row
			if e := rows.Scan(scanBuf...); e != nil {
				return false, e
			}
			lt.Size++

			// make new cell from scan conversion buffer and pass it to the writer
			return cvtTo(makeCell())
		})
	if err != nil {
		return nil, err
	}

	// check for the empty result page or last page
	if lt.Size <= 0 {
		lt.Offset = nRow
	}
	lt.IsLastPage = nSize <= 0 || nSize > 0 && nRow <= nStart+nSize

	return &lt, nil
}
