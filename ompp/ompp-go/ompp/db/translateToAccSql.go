// Copyright (c) 2021 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"errors"
	"slices"
	"strconv"
)

// Translate output table accumulators calculation into sql query.
func translateToAccSql(modelDef *ModelMeta, table *TableMeta, paramMeta []ParamMeta, readLt *ReadLayout, calcLt *CalculateLayout, runIds []int) (string, error) {

	// make sql:
	// WITH cte
	// SELECT main sql for calculation
	// WHERE run id IN (....)
	// AND dimension filters
	// ORDER BY 1, 2,....
	paramCols := makeParamCols(paramMeta)

	// validate filter names: it must be name of dimension or name of calculated expression
	for k := range readLt.Filter {

		isOk := calcLt.Name == readLt.Filter[k].Name

		for j := 0; !isOk && j < len(table.Dim); j++ {
			isOk = table.Dim[j].Name == readLt.Filter[k].Name
		}
		if !isOk {
			return "", errors.New("Error: output table " + table.Name + " does not have dimension " + readLt.Filter[k].Name)
		}
	}

	// translate calculation to sql
	cteSql, mainSql, err := partialTranslateToAccSql(modelDef, table, paramCols, readLt, calcLt, runIds)
	if err != nil {
		return "", err
	}
	pCteSql, err := makeParamCteSql(paramCols, readLt.FromId, runIds)
	if err != nil {
		return "", err
	}
	if pCteSql != "" {
		cteSql += ", " + pCteSql
	}

	sql := "WITH " + cteSql + " " + mainSql

	// append ORDER BY, default order by: run_id, expression id, dimensions
	sql += makeOrderBy(table.Rank, readLt.OrderBy, 2)

	return sql, nil
}

// Translate output table accumulators aggregation to sql query, apply dimension filters and selected run id's.
// Return list of CTE sql's and main sql's.
func partialTranslateToAccSql(
	modelDef *ModelMeta, table *TableMeta, paramCols map[string]paramColumn, readLt *ReadLayout, calcLt *CalculateLayout, runIds []int,
) (
	string, string, error,
) {

	// translate output table aggregation expression into sql query:
	//   WITH asrc (run_id, acc_id, sub_id, dim0, dim1, acc_value ) AS
	//   (
	//     SELECT
	//       BR.run_id, C.acc_id, C.sub_id, C.dim0, C.dim1, C.acc_value
	//     FROM age_acc C
	//     INNER JOIN run_table BR ON (BR.base_run_id = C.run_id AND BR.table_hid = 101)
	//   )
	//   SELECT
	//     A.run_id, CalcId AS calc_id, A.dim0, A.dim1, A.calc_value
	//   FROM
	//   (
	//     SELECT
	//       M1.run_id, M1.dim0, M1.dim1,
	//       SUM(M1.acc_value + 0.5 * T2.ex2) AS calc_value
	//     FROM asrc M1
	//     INNER JOIN ........
	//     WHERE M1.acc_id = 0
	//     GROUP BY M1.run_id, M1.dim0, M1.dim1
	//   ) A
	// WHERE A.run_id IN (103, 104, 105, 106, 107, 108, 109, 110, 111, 112)
	// AND A.dim0 = .....
	// ORDER BY 1, 2, 3, 4
	//
	cteSql, mainSql, err := transalteAccAggrToSql(table, paramCols, calcLt.CalcId, calcLt.Calculate)
	if err != nil {
		return "", "", errors.New("Error at " + table.Name + " " + calcLt.Calculate + ": " + err.Error())
	}

	// make where clause and dimension filters:
	// WHERE A.run_id IN (103, 104, 105, 106, 107, 108, 109, 110, 111, 112)
	// AND A.dim0 = .....

	// append run id's
	where := " WHERE A.run_id IN ("

	if readLt.FromId > 0 {
		isFound := false
		for k := 0; !isFound && k < len(runIds); k++ {
			isFound = runIds[k] == readLt.FromId
		}
		if !isFound {
			where += strconv.Itoa(readLt.FromId)
			if len(runIds) > 0 {
				where += ", "
			}
		}
	}
	for k := 0; k < len(runIds); k++ {
		if k > 0 {
			where += ", "
		}
		where += strconv.Itoa(runIds[k])
	}
	where += ")"

	// append dimension enum code filters and value filter, if specified: A.dim1 = 'M' AND (calc_value < 1234 AND calc_id = 12001)
	iDbl, ok := modelDef.TypeOfDouble()
	if !ok {
		return "", "", errors.New("double type not found, output table " + table.Name)
	}

	for k := range readLt.Filter {

		var err error
		f := ""

		if calcLt.Name == readLt.Filter[k].Name { // check if this is a filter by calculated value

			f, err = makeWhereValueFilter(
				&readLt.Filter[k], "", "calc_value", "calc_id", calcLt.CalcId, &modelDef.Type[iDbl], readLt.Filter[k].Name, "output table "+table.Name)
			if err != nil {
				return "", "", err
			}
		}
		if f == "" { // if not a filter by value then it can be filter by dimension

			dix := -1
			for j := range table.Dim {
				if table.Dim[j].Name == readLt.Filter[k].Name {
					dix = j
					break
				}
			}
			if dix >= 0 {

				f, err = makeWhereFilter(
					&readLt.Filter[k], "A", table.Dim[dix].colName, table.Dim[dix].typeOf, table.Dim[dix].IsTotal, table.Dim[dix].Name, "output table "+table.Name)
				if err != nil {
					return "", "", errors.New("Error at " + table.Name + " " + calcLt.Calculate + ": " + err.Error())
				}
			}
		}
		// use filter: it is a filter by dimension name or by current calculated column name
		if f != "" {
			where += " AND " + f
		}
	}

	// append dimension enum id filters, if specified
	for k := range readLt.FilterById {

		// find dimension index by name
		dix := -1
		for j := range table.Dim {
			if table.Dim[j].Name == readLt.FilterById[k].Name {
				dix = j
				break
			}
		}
		if dix < 0 {
			return "", "", errors.New("Error at " + table.Name + " " + calcLt.Calculate + ": output table " + table.Name + " does not have dimension " + readLt.FilterById[k].Name)
		}

		f, err := makeWhereIdFilter(
			&readLt.FilterById[k], "A", table.Dim[dix].colName, table.Dim[dix].typeOf, table.Dim[dix].Name, "output table "+table.Name)
		if err != nil {
			return "", "", errors.New("Error at " + table.Name + " " + calcLt.Calculate + ": " + err.Error())
		}

		where += " AND " + f
	}

	// append WHERE to main sql query and return result
	mainSql += where

	return cteSql, mainSql, nil
}

// Translate output table aggregation expression into sql query.
// Only native accumulators allowed.
// Calculation must return a single value as a result of aggregation, ex.: AVG(acc_value).
//
//	WITH asrc (run_id, acc_id, sub_id, dim0, dim1, acc_value ) AS
//	(
//	  SELECT
//	    BR.run_id, C.acc_id, C.sub_id, C.dim0, C.dim1, C.acc_value
//	  FROM age_acc C
//	  INNER JOIN run_table BR ON (BR.base_run_id = C.run_id AND BR.table_hid = 101)
//	)
//	SELECT
//	  A.run_id, CalcId AS calc_id, A.dim0, A.dim1, A.calc_value
//	FROM
//	(
//	  SELECT
//	    M1.run_id, M1.dim0, M1.dim1,
//	    SUM(M1.acc_value + 0.5 * T2.ex2) AS calc_value
//	  FROM asrc M1
//	  INNER JOIN ........
//	  WHERE M1.acc_id = 0
//	  GROUP BY M1.run_id, M1.dim0, M1.dim1
//	) A
func transalteAccAggrToSql(table *TableMeta, paramCols map[string]paramColumn, calcId int, calculateExpr string) (string, string, error) {

	// clean source calculation from cr lf and unsafe sql quotes
	// return error if unsafe sql or comment found outside of 'quotes', ex.: -- ; DELETE INSERT UPDATE...
	startExpr := cleanSourceExpr(calculateExpr)
	err := errorIfUnsafeSqlOrComment(startExpr)
	if err != nil {
		return "", "", err
	}

	// translate (substitute) all simple functions: OM_DIV_BY OM_IF...
	startExpr, err = translateAllSimpleFnc(startExpr)
	if err != nil {
		return "", "", err
	}

	// aggregation expression columns: only native (not a derived) accumulators can be aggregated
	aggrCols := make([]aggrColumn, len(table.Acc))

	for k := range table.Acc {
		aggrCols[k] = aggrColumn{
			name:    table.Acc[k].Name,
			colName: table.Acc[k].colName,
			isAggr:  !table.Acc[k].IsDerived, // only native accumulators can be aggregated
		}
	}

	// produce accumulator column name: acc0 => M1.acc_value or acc4 => L1A4.acc4
	makeAccColName := func(
		name string, nameIdx int, isSimple, isVar bool, firstAlias string, levelAccAlias string, isFirstAcc bool,
	) string {

		if isFirstAcc {
			return firstAlias + "." + "acc_value" // first accumulator: acc0 => acc_value
		}
		return levelAccAlias + "." + name // any other accumulator: acc4 => acc4
	}

	// translate parameter names by replacing it with CTE alias and CTE parameter value name:
	//	param.Name          => M1P103.param_value
	// also return INNER JOIN between parameter CTE view and main table:
	//  INNER JOIN par_103   M1P103 ON (M1P103.run_id = M1.run_id)
	// it cannot be run comparison
	makeParamColName := func(colKey string, isSimple, isVar bool, alias string) (string, string, error) {

		pCol, ok := paramCols[colKey]
		if !ok {
			return "", "", errors.New("Error: parameter not found: " + colKey)
		}
		if !pCol.isNumber || pCol.paramRow == nil {
			return "", "", errors.New("Error: parameter must a be numeric scalar: " + colKey)
		}
		if !isSimple || isVar {
			return "", "", errors.New("Error: parameter cannot be a run comparison parameter[base] or parameter[variant]: " + colKey)
		}
		sHid := strconv.Itoa(pCol.paramRow.ParamHid)

		pCol.isSimple = true
		pa := alias + "P" + sHid
		sqlName := pa + ".param_value" // not a run comparison: param.Name => M1P103.param_value
		innerJoin := "INNER JOIN par_" + sHid + " " + pa + " ON (" + pa + ".run_id = " + alias + ".run_id)"

		paramCols[colKey] = pCol

		return sqlName, innerJoin, nil
	}

	// parse aggregation expression
	levelArr, err := parseAggrCalculation(aggrCols, paramCols, startExpr, makeAccColName, makeParamColName)
	if err != nil {
		return "", "", err
	}

	// build output sql from parser state: CTE and main sql query
	cteSql, mainSql, err := makeAccAggrSql(table, calcId, levelArr)
	if err != nil {
		return "", "", err
	}

	return cteSql, mainSql, nil
}

// Build aggregation sql from parser state.
func makeAccAggrSql(table *TableMeta, calcId int, levelArr []levelDef) (string, string, error) {

	// build output sql for expression:
	//
	// OM_SUM(acc0 + 0.5 * OM_AVG(acc1 + param.Extra + acc4 + 0.1 * (OM_MAX(acc0) - OM_MIN(acc1)) ))
	// =>
	//   WITH asrc (run_id, acc_id, sub_id, dim0, dim1, acc_value ) AS
	//   (
	//     SELECT
	//       BR.run_id, C.acc_id, C.sub_id, C.dim0, C.dim1, C.acc_value
	//     FROM age_acc C
	//     INNER JOIN run_table BR ON (BR.base_run_id = C.run_id AND BR.table_hid = 101)
	//   ),
	//   par_103 (run_id, param_value) AS
	//     (.... AVG(param_value) FROM Extra.... WHERE RP.run_id IN (219, 221, 222) GROUP BY RP.run_id)
	//   SELECT
	//     A.run_id, CalcId AS calc_id, A.dim0, A.dim1, A.calc_value
	//   FROM
	//   (
	//     SELECT
	//       M1.run_id, M1.dim0, M1.dim1,
	//       SUM(M1.acc_value + 0.5 * T2.ex2) AS calc_value
	//     FROM asrc M1
	//     INNER JOIN
	//     (
	//       SELECT
	//         M2.run_id, M2.dim0, M2.dim1,
	//         AVG(M2.acc_value + L2A4.acc4 + 0.1 * (T3.ex31 - T3.ex32)) AS ex2
	//       FROM asrc M2
	//       INNER JOIN par_103 M1P103 ON (M1P103.run_id = M2.run_id)
	//       INNER JOIN
	//       (
	//         SELECT run_id, dim0, dim1, sub_id, acc_value AS acc4 FROM asrc WHERE acc_id = 4
	//       ) L2A4
	//       ON (L2A4.run_id = M2.run_id AND L2A4.dim0 = M2.dim0 AND L2A4.dim1 = M2.dim1 AND L2A4.sub_id = M2.sub_id)
	//       INNER JOIN
	//       (
	//         SELECT
	//           M3.run_id, M3.dim0, M3.dim1,
	//           MAX(M3.acc_value) AS ex31,
	//           MIN(L3A1.acc1)    AS ex32
	//         FROM asrc M3
	//         INNER JOIN
	//         (
	//           SELECT run_id, dim0, dim1, sub_id, acc_value AS acc1 FROM asrc WHERE acc_id = 1
	//         ) L3A1
	//         ON (L3A1.run_id = M3.run_id AND L3A1.dim0 = M3.dim0 AND L3A1.dim1 = M3.dim1 AND L3A1.sub_id = M3.sub_id)
	//         WHERE M3.acc_id = 0
	//         GROUP BY M3.run_id, M3.dim0, M3.dim1
	//       ) T3
	//       ON (T3.run_id = M2.run_id AND T3.dim0 = M2.dim0 AND T3.dim1 = M2.dim1)
	//       WHERE M2.acc_id = 1
	//       GROUP BY M2.run_id, M2.dim0, M2.dim1
	//     ) T2
	//     ON (T2.run_id = M1.run_id AND T2.dim0 = M1.dim0 AND T2.dim1 = M1.dim1)
	//     WHERE M1.acc_id = 0
	//     GROUP BY M1.run_id, M1.dim0, M1.dim1
	//   ) A
	//
	cteSql := "asrc (run_id, acc_id, sub_id"
	for _, d := range table.Dim {
		cteSql += ", " + d.colName
	}
	cteSql += ", acc_value) AS" +
		" (" +
		"SELECT BR.run_id, C.acc_id, C.sub_id"
	for _, d := range table.Dim {
		cteSql += ", C." + d.colName
	}
	cteSql += ", C.acc_value" +
		" FROM " + table.DbAccTable + " C" +
		" INNER JOIN run_table BR ON (BR.base_run_id = C.run_id AND BR.table_hid = " + strconv.Itoa(table.TableHid) + ")" +
		")"

	// SELECT A.run_id, CalcId AS calc_id, A.dim0, A.dim1, A.calc_value FROM (
	//
	mainSql := "SELECT A.run_id, " + strconv.Itoa(calcId) + " AS calc_id"

	for _, d := range table.Dim {
		mainSql += ", A." + d.colName
	}
	mainSql += ", A.calc_value FROM ( "

	// main aggregation sql body
	for nLev, lv := range levelArr {

		// select run_id, dim0,...,sub_id, acc_value
		// from accumulator table where acc_id = first accumulator
		//
		mainSql += "SELECT " + lv.fromAlias + ".run_id"

		for _, d := range table.Dim {
			mainSql += ", " + lv.fromAlias + "." + d.colName
		}

		for _, expr := range lv.exprArr {
			mainSql += ", " + expr.sqlExpr
			if expr.colName != "" {
				mainSql += " AS " + expr.colName
			}
		}

		mainSql += " FROM asrc " + lv.fromAlias

		// INNER JOIN parameters CTE ON run_id
		slices.Sort(lv.paramJoinArr)

		for _, pj := range lv.paramJoinArr {
			mainSql += " " + pj
		}

		// INNER JOIN accumulator table for all other accumulators ON run_id, dim0,...,sub_id
		for nAcc, acc := range table.Acc {

			if !lv.agcUsageArr[nAcc] || nAcc == lv.firstAgcIdx { // skip first accumulator and unused accumulators
				continue
			}
			accAlias := "L" + strconv.Itoa(lv.level) + "A" + strconv.Itoa(nAcc)

			mainSql += " INNER JOIN (SELECT run_id, "

			for _, d := range table.Dim {
				mainSql += d.colName + ", "
			}

			mainSql += "sub_id, acc_value AS " + acc.colName +
				" FROM asrc" +
				" WHERE acc_id = " + strconv.Itoa(acc.AccId) +
				") " + accAlias

			mainSql += " ON (" + accAlias + ".run_id = " + lv.fromAlias + ".run_id"

			for _, d := range table.Dim {
				mainSql += " AND " + accAlias + "." + d.colName + " = " + lv.fromAlias + "." + d.colName
			}

			mainSql += " AND " + accAlias + ".sub_id = " + lv.fromAlias + ".sub_id)"
		}

		if nLev < len(levelArr)-1 { // if not lowest level then continue INNER JOIN down to the next level
			mainSql += " INNER JOIN ("
		}
	}

	// for each level except of the lowest append:
	//   WHERE acc_id = first accumulator id
	//   GROUP BY run_id, dim0,...
	//   ) ON (run_id, dim0,...)
	for nLev := len(levelArr) - 1; nLev >= 0; nLev-- {

		firstId := 0
		if levelArr[nLev].firstAgcIdx >= 0 && levelArr[nLev].firstAgcIdx < len(table.Acc) {
			firstId = table.Acc[levelArr[nLev].firstAgcIdx].AccId
		}

		mainSql += " WHERE " + levelArr[nLev].fromAlias + ".acc_id = " + strconv.Itoa(firstId)

		mainSql += " GROUP BY " + levelArr[nLev].fromAlias + ".run_id"

		for _, d := range table.Dim {
			mainSql += ", " + levelArr[nLev].fromAlias + "." + d.colName
		}

		if nLev > 0 {

			mainSql += ") " + levelArr[nLev].innerAlias +
				" ON (" + levelArr[nLev].innerAlias + ".run_id = " + levelArr[nLev-1].fromAlias + ".run_id"

			for _, d := range table.Dim {
				mainSql += " AND " + levelArr[nLev].innerAlias + "." + d.colName + " = " + levelArr[nLev-1].fromAlias + "." + d.colName
			}

			mainSql += ")"
		}
	}
	mainSql += " ) A"

	return cteSql, mainSql, nil
}
