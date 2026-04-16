// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"errors"
	"strconv"
)

// ReadParameterTo read input parameter rows (sub id, dimensions, value) from workset or model run results and process each row by cvtTo().
func ReadParameterTo(dbConn *sql.DB, modelDef *ModelMeta, layout *ReadParamLayout, cvtTo func(src interface{}) (bool, error)) (*ReadPageLayout, error) {

	// validate parameters
	if modelDef == nil {
		return nil, errors.New("invalid (empty) model metadata, look like model not found")
	}
	if layout == nil {
		return nil, errors.New("invalid (empty) parameter read layout")
	}
	if layout.Name == "" {
		return nil, errors.New("invalid (empty) parameter name")
	}

	// find parameter id by name
	var param *ParamMeta
	if k, ok := modelDef.ParamByName(layout.Name); ok {
		param = &modelDef.Param[k]
	} else {
		return nil, errors.New("parameter not found: " + layout.Name)
	}

	// if this is workset parameter then:
	//   if source workset exist
	//   check readonly status: if layout.IsEditSet then read-only must be false
	//   if parameter not in workset then select base run id, it must be >0
	var srcRunId int
	var isWsParam bool

	if !layout.IsFromSet {
		srcRunId = layout.FromId // this is parameter from existing run
	} else {

		// validate workset: it must exist
		setRow, err := GetWorkset(dbConn, layout.FromId)
		if err != nil {
			return nil, err
		}
		if setRow == nil {
			return nil, errors.New("workset not found, id: " + strconv.Itoa(layout.FromId))
		}

		// workset readonly status must be compatible with (oposite to) "edit workset" status
		if layout.IsEditSet && setRow.IsReadonly {
			return nil, errors.New("cannot edit parameter " + param.Name + " from read-only workset, id: " + strconv.Itoa(layout.FromId))
		}

		// check is this workset contain the parameter
		err = SelectFirst(dbConn,
			"SELECT COUNT(*) FROM workset_parameter"+
				" WHERE set_id = "+strconv.Itoa(layout.FromId)+
				" AND parameter_hid = "+strconv.Itoa(param.ParamHid),
			func(row *sql.Row) error {
				var n int
				if err := row.Scan(&n); err != nil {
					return err
				}
				isWsParam = n != 0
				return nil
			})
		switch {
		case err == sql.ErrNoRows: // unknown error: should never be there
			return nil, errors.New("cannot count parameter " + param.Name + " in workset, id: " + strconv.Itoa(layout.FromId))
		case err != nil:
			return nil, err
		}

		// if parameter not in that workset then workset must have base run
		if !isWsParam {
			if setRow.BaseRunId <= 0 {
				return nil, errors.New("workset does not contain parameter " + param.Name + " and not run-based, workset id: " + strconv.Itoa(layout.FromId))
			}
			srcRunId = setRow.BaseRunId
		}
	}

	// if parameter from run (or from workset base run) then:
	//   check if model run exist and model run completed is completed or in progress
	if !isWsParam {
		runRow, err := GetRun(dbConn, srcRunId)
		if err != nil {
			return nil, err
		}
		if runRow == nil {
			return nil, errors.New("model run not found, id: " + strconv.Itoa(srcRunId))
		}
		if !IsRunCompleted(runRow.Status) && runRow.Status != ProgressRunStatus {
			return nil, errors.New("model run not completed, id: " + strconv.Itoa(srcRunId))
		}
	}

	// make sql to select parameter from model run or workset:
	//   SELECT sub_id, dim0, dim1, param_value
	//   FROM ageSex_p2012_817
	//   WHERE run_id = (SELECT base_run_id FROM run_parameter WHERE run_id = 1234 AND parameter_hid = 1)
	//   AND sub_id = 7
	//   AND dim1 IN (1, 2, 3, 4)
	//   ORDER BY 1, 2, 3
	// or:
	//   SELECT sub_id, dim0, dim1, param_value
	//   FROM ageSex_w2012_817
	//   AND sub_id = 7
	//   WHERE set_id = 9876
	//   AND dim1 IN (1, 2, 3, 4)
	//   ORDER BY 1, 2, 3
	q := "SELECT sub_id, "
	for k := range param.Dim {
		q += param.Dim[k].colName + ", "
	}
	q += "param_value FROM "

	if isWsParam {
		q += param.DbSetTable +
			" WHERE set_id = " + strconv.Itoa(layout.FromId)
	} else {
		q += param.DbRunTable +
			" WHERE run_id =" +
			" (SELECT base_run_id FROM run_parameter" +
			" WHERE run_id = " + strconv.Itoa(srcRunId) +
			" AND parameter_hid = " + strconv.Itoa(param.ParamHid) + ")"
	}

	// append sub-value id filter
	if layout.IsSubId {
		q += " AND sub_id = " + strconv.Itoa(layout.SubId)
	}

	// append dimension enum code filters, if specified
	for k := range layout.Filter {

		// filter parameter value or find dimension index by name
		var err error
		f := ""

		if layout.Filter[k].Name == "param_value" {

			f, err = makeWhereValueFilter(
				&layout.Filter[k], "", "param_value", "", 0, param.typeOf, "param_value", "parameter "+param.Name)
			if err != nil {
				return nil, err
			}
		} else {

			dix := -1
			for j := range param.Dim {
				if param.Dim[j].Name == layout.Filter[k].Name {
					dix = j
					break
				}
			}
			if dix < 0 {
				return nil, errors.New("parameter " + param.Name + " does not have dimension " + layout.Filter[k].Name)
			}
			f, err = makeWhereFilter(
				&layout.Filter[k], "", param.Dim[dix].colName, param.Dim[dix].typeOf, false, param.Dim[dix].Name, "parameter "+param.Name)
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
		for j := range param.Dim {
			if param.Dim[j].Name == layout.FilterById[k].Name {
				dix = j
				break
			}
		}
		if dix < 0 {
			return nil, errors.New("parameter " + param.Name + " does not have dimension " + layout.FilterById[k].Name)
		}

		f, err := makeWhereIdFilter(
			&layout.FilterById[k], "", param.Dim[dix].colName, param.Dim[dix].typeOf, param.Dim[dix].Name, "parameter "+param.Name)
		if err != nil {
			return nil, err
		}

		q += " AND " + f
	}

	// append order by
	q += makeOrderBy(param.Rank, layout.OrderBy, 1)

	// prepare db-row scan conversion buffer: sub_id, dimensions, value
	// and define conversion function to make new cell from scan buffer
	scanBuf, fc := scanSqlRowToCellParam(param)

	// if full page requested:
	// select rows into the list buffer and write rows from the list into output stream
	if layout.IsFullPage {

		// make a list of output cells
		cLst, lt, e := SelectToList(dbConn, q, layout.ReadPageLayout,
			func(rows *sql.Rows) (interface{}, error) {

				if e := rows.Scan(scanBuf...); e != nil {
					return nil, e
				}

				// make new cell from conversion buffer
				c := CellParam{cellIdValue: cellIdValue{DimIds: make([]int, param.Rank)}}

				if e := fc(&c); e != nil {
					return nil, e
				}

				return c, nil
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

	// select parameter cells: (sub id, dimension(s) enum ids, parameter value)
	err := SelectRowsTo(dbConn, q,
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

			// make new cell from conversion buffer
			c := CellParam{cellIdValue: cellIdValue{DimIds: make([]int, param.Rank)}}

			if e := fc(&c); e != nil {
				return false, e
			}

			return cvtTo(c) // process cell
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

// trxReadParameterTo read input parameter rows (sub id, dimensions, value) from workset or model run results and process each row by cvtTo().
func trxReadParameterTo(trx *sql.Tx, param *ParamMeta, query string, cvtTo func(src interface{}) error) error {

	// select parameter cells: (sub id, dimension(s) enum ids, parameter value)
	scanBuf, fc := scanSqlRowToCellParam(param)

	err := TrxSelectRows(trx, query,
		func(rows *sql.Rows) error {

			// select next row
			if e := rows.Scan(scanBuf...); e != nil {
				return e
			}

			// make new cell from conversion buffer
			var c = CellParam{cellIdValue: cellIdValue{DimIds: make([]int, param.Rank)}}
			if e := fc(&c); e != nil {
				return e
			}

			return cvtTo(c) // process cell
		})
	return err
}

// prepare to scan sql rows and convert each row to CellParam
// retun scan buffer to be popualted by rows.Scan() and closure to that buffer into CellParam
func scanSqlRowToCellParam(param *ParamMeta) ([]interface{}, func(*CellParam) error) {

	var nSub int
	d := make([]int, param.Rank)
	var v interface{}
	var vs string
	var vf sql.NullFloat64
	var cvt func(c *CellParam) error
	var scanBuf []interface{}

	scanBuf = append(scanBuf, &nSub)
	for k := 0; k < param.Rank; k++ {
		scanBuf = append(scanBuf, &d[k])
	}

	switch {
	case param.typeOf.IsBool():
		scanBuf = append(scanBuf, &v)
		cvt = func(c *CellParam) error {
			c.SubId = nSub
			copy(c.DimIds, d)
			c.IsNull = false // logical parameter expected to be NOT NULL
			is := false
			switch vn := v.(type) {
			case nil: // 2018: unexpected today, may be in the future
				is = false
				c.IsNull = true
			case bool:
				is = vn
			case int64:
				is = vn != 0
			case uint64:
				is = vn != 0
			case int32:
				is = vn != 0
			case uint32:
				is = vn != 0
			case int16:
				is = vn != 0
			case uint16:
				is = vn != 0
			case int8:
				is = vn != 0
			case uint8:
				is = vn != 0
			case uint:
				is = vn != 0
			case float32: // oracle (very unlikely)
				is = vn != 0.0
			case float64: // oracle (often)
				is = vn != 0.0
			case int:
				is = vn != 0
			default:
				return errors.New("invalid parameter value type, expected: integer")
			}
			c.Value = is
			return nil
		}

	case param.typeOf.IsString():
		scanBuf = append(scanBuf, &vs)
		cvt = func(c *CellParam) error {
			c.SubId = nSub
			copy(c.DimIds, d)
			c.IsNull = false
			c.Value = vs
			return nil
		}

	case param.typeOf.IsFloat():
		scanBuf = append(scanBuf, &vf)
		cvt = func(c *CellParam) error {
			c.SubId = nSub
			copy(c.DimIds, d)
			c.IsNull = !vf.Valid
			c.Value = 0.0
			if !c.IsNull {
				c.Value = vf.Float64
			}
			return nil
		}

	default:
		scanBuf = append(scanBuf, &v)
		cvt = func(c *CellParam) error { c.SubId = nSub; copy(c.DimIds, d); c.IsNull = false; c.Value = v; return nil }
	}

	return scanBuf, cvt
}
