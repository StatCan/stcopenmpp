// Copyright (c) 2021 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"container/list"
	"database/sql"
	"errors"
	"sort"
	"strconv"
)

const CALCULATED_ID_OFFSET = 12000 // calculated exprssion id offset, for example for Expr1 calculated expression id is 12001

// CalculateOutputTable read output table page (dimensions and values) and calculate extra measure(s).
//
// If calcLt.IsAggr true then do accumulator(s) aggregation else calculate expression value(s), ex: Expr1[variant] - Expr1[base].
func CalculateOutputTable(dbConn *sql.DB, modelDef *ModelMeta, tableLt *ReadCalculteTableLayout, runIds []int) (*list.List, *ReadPageLayout, error) {

	// validate parameters
	if modelDef == nil {
		return nil, nil, errors.New("invalid (empty) model metadata, look like model not found")
	}
	if tableLt == nil {
		return nil, nil, errors.New("invalid (empty) output table layout")
	}
	if tableLt.Name == "" {
		return nil, nil, errors.New("invalid (empty) output table name")
	}
	if len(tableLt.Calculation) <= 0 {
		return nil, nil, errors.New("invalid (empty) calculation expression(s)")
	}

	// find output table id by name
	var table *TableMeta
	if k, ok := modelDef.OutTableByName(tableLt.Name); ok {
		table = &modelDef.Table[k]
	} else {
		return nil, nil, errors.New("output table not found: " + tableLt.Name)
	}

	// translate calculation to sql
	q, err := translateTableCalcToSql(modelDef, table, &tableLt.ReadLayout, tableLt.Calculation, runIds)
	if err != nil {
		return nil, nil, err
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

	// select cells:
	// run_id, calculation id, dimension(s) enum ids, value null status
	cLst, lt, err := SelectToList(dbConn, q, tableLt.ReadPageLayout,
		func(rows *sql.Rows) (interface{}, error) {

			if err := rows.Scan(scanBuf...); err != nil {
				return nil, err
			}

			// make new cell from conversion buffer
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
			return c, nil
		})
	if err != nil {
		return nil, nil, err
	}

	return cLst, lt, nil
}

// Translate all output table calculations to sql query, apply dimension filters, selected run id's and order by.
// It can be a multiple runs comparison and base run id is layout.FromId.
// Or simple expression calculation inside of single run or accumulators aggregation inside of single run,
// in that case layout.FromId and runIds[] are merged.
func translateTableCalcToSql(modelDef *ModelMeta, table *TableMeta, readLt *ReadLayout, calcLt []CalculateTableLayout, runIds []int) (string, error) {

	// translate each calculation to sql: CTE and main sql query
	cteSql := []string{}
	mainSql := []string{}
	paramCols := makeParamCols(modelDef.Param)

	// validate filter names: it must be name of dimension or name of calculated expression
	for k := range readLt.Filter {

		isOk := false
		for j := 0; !isOk && j < len(calcLt); j++ {
			isOk = calcLt[j].Name == readLt.Filter[k].Name
		}
		for j := 0; !isOk && j < len(table.Dim); j++ {
			isOk = table.Dim[j].Name == readLt.Filter[k].Name
		}
		if !isOk {
			return "", errors.New("Error: output table " + table.Name + " does not have dimension " + readLt.Filter[k].Name)
		}
	}

	// translate all calculations to sql
	for k := range calcLt {

		cte := []string{}
		mSql := ""
		cteAcc := ""
		var err error

		if !calcLt[k].IsAggr {
			cte, mSql, _, err = partialTranslateToExprSql(modelDef, table, paramCols, readLt, &calcLt[k].CalculateLayout, runIds)
		} else {
			cteAcc, mSql, err = partialTranslateToAccSql(modelDef, table, paramCols, readLt, &calcLt[k].CalculateLayout, runIds)
			if err == nil {
				cte = []string{cteAcc}
			}
		}
		if err != nil {
			return "", err
		}

		// merge main body SQL, expected to be unique, skip duplicates
		isFound := false
		for j := 0; !isFound && j < len(mainSql); j++ {
			isFound = mSql == mainSql[j]
		}
		if isFound {
			continue // skip duplicate SQL, it is the same source expression
		}
		mainSql = append(mainSql, mSql)

		// merge CTE sql's, skip identical CTE
		for _, c := range cte {

			isFound = false
			for j := 0; !isFound && j < len(cteSql); j++ {
				isFound = c == cteSql[j]
			}
			if !isFound {
				cteSql = append(cteSql, c)
			}
		}
	}

	// make sql:
	// WITH cte array
	// SELECT main sql for calculation 1
	// WHERE run id IN (....)
	// AND dimension filters
	// UNION ALL
	// SELECT main sql for calculation 2
	// WHERE run id IN (....)
	// AND dimension filters
	// ORDER BY 1, 2,....

	sql := "WITH "
	for k := range cteSql {
		if k > 0 {
			sql += ", "
		}
		sql += cteSql[k]
	}

	pCteSql, err := makeParamCteSql(paramCols, readLt.FromId, runIds)
	if err != nil {
		return "", err
	}
	if pCteSql != "" {
		sql += ", " + pCteSql
	}

	for k := range mainSql {
		if k > 0 {
			sql = sql + " UNION ALL " + mainSql[k]
		} else {
			sql = sql + " " + mainSql[k]
		}
	}

	// append ORDER BY, default order by: run_id, expression id, dimensions
	sql += makeOrderBy(table.Rank, readLt.OrderBy, 2)

	return sql, nil
}

// CalculateMicrodata aggregates microdata using group by attributes as dimensions and calculate aggregated measure(s).
func CalculateMicrodata(dbConn *sql.DB, modelDef *ModelMeta, microLt *ReadCalculteMicroLayout, runIds []int) (*list.List, *ReadPageLayout, error) {

	// validate parameters
	if modelDef == nil {
		return nil, nil, errors.New("invalid (empty) model metadata, look like model not found")
	}
	if microLt == nil {
		return nil, nil, errors.New("invalid (empty) microdata calculate layout")
	}
	if microLt.Name == "" {
		return nil, nil, errors.New("invalid (empty) microdata entity name")
	}
	if len(microLt.Calculation) <= 0 {
		return nil, nil, errors.New("invalid (empty) calculation expression(s)")
	}

	// find entity by name
	var entity *EntityMeta

	if k, ok := modelDef.EntityByName(microLt.Name); ok {
		entity = &modelDef.Entity[k]
	} else {
		return nil, nil, errors.New("entity not found: " + microLt.Name)
	}

	// find entity generation by entity id, as it is today model run has only one entity generation for each entity
	egLst, err := GetEntityGenList(dbConn, microLt.FromId)
	if err != nil {
		return nil, nil, errors.New("entity generation not found: " + microLt.Name + ": " + strconv.Itoa(microLt.FromId))
	}
	var entityGen *EntityGenMeta

	for k := range egLst {

		if egLst[k].EntityId == entity.EntityId {
			entityGen = &egLst[k]
			break
		}
	}
	if entityGen == nil {
		return nil, nil, errors.New("Error: entity generation not found: " + microLt.Name + ": " + strconv.Itoa(microLt.FromId))
	}

	// find group by microdata attributes by name
	aGroupBy := []EntityAttrRow{}

	for _, ga := range entityGen.GenAttr {

		aIdx, ok := entity.AttrByKey(ga.AttrId)
		if !ok {
			return nil, nil, errors.New("entity attribute not found by id: " + strconv.Itoa(ga.AttrId) + " " + entity.Name)
		}

		isFound := false
		for j := 0; !isFound && j < len(microLt.GroupBy); j++ {

			if microLt.GroupBy[j] != entity.Attr[aIdx].Name {
				continue
			}
			aGroupBy = append(aGroupBy, entity.Attr[aIdx])

			// group by attributes must boolean or not built-in
			if entity.Attr[aIdx].typeOf.IsBuiltIn() && !entity.Attr[aIdx].typeOf.IsBool() {
				return nil, nil, errors.New("invalid type of entity group by attribute not found by: " + entity.Name + "." + microLt.GroupBy[j] + " : " + entity.Attr[aIdx].typeOf.Name)
			}

		}
	}

	// check: all group by attributes must be found
	for _, name := range microLt.GroupBy {

		isFound := false
		for k := 0; !isFound && k < len(aGroupBy); k++ {
			isFound = aGroupBy[k].Name == name
		}
		if !isFound {
			return nil, nil, errors.New("entity group by attribute not found by: " + entity.Name + "." + name)
		}
	}

	// translate calculation to sql
	q, err := translateMicroToSql(modelDef, entity, entityGen, &microLt.ReadLayout, &microLt.CalculateMicroLayout, runIds)
	if err != nil {
		return nil, nil, err
	}

	// prepare db-row scan conversion buffer: run_id, calculation id, group by attributes, value
	// and define conversion function to make new cell from scan buffer
	scanBuf, fc, err := scanSqlRowToCellMicroCalc(entity, aGroupBy)
	if err != nil {
		return nil, nil, err
	}

	// select cells:
	// run_id, calculation id, group by attributes, value and null status
	cLst, lt, err := SelectToList(dbConn, q, microLt.ReadPageLayout,
		func(rows *sql.Rows) (interface{}, error) {

			if e := rows.Scan(scanBuf...); e != nil {
				return nil, e
			}

			// make new cell from conversion buffer
			c := CellMicroCalc{Attr: make([]attrValue, len(aGroupBy)+1)}

			if e := fc(&c); e != nil {
				return nil, e
			}

			return c, nil
		})
	if err != nil {
		return nil, nil, err
	}

	return cLst, lt, nil
}

// Translate all microdata aggregations to sql query, apply group by, dimension filters, selected run id's and order by.
// It can be a multiple runs comparison and base run id is layout.FromId.
// Or simple aggreagtion inside of single run, in that case layout.FromId and runIds[] are merged.
func translateMicroToSql(modelDef *ModelMeta, entity *EntityMeta, entityGen *EntityGenMeta, readLt *ReadLayout, calcLt *CalculateMicroLayout, runIds []int) (string, error) {

	// translate each calculation to sql: CTE and main sql query
	mainSql := []string{}

	aggrCols, err := makeMicroAggrCols(entity, entityGen, calcLt.GroupBy)
	if err != nil {
		return "", err
	}
	paramCols := makeParamCols(modelDef.Param)

	// validate filter names: it must be name of attribute or name of calculated attribute
	for k := range readLt.Filter {

		isOk := false
		for j := 0; !isOk && j < len(calcLt.Calculation); j++ {
			isOk = calcLt.Calculation[j].Name == readLt.Filter[k].Name
		}
		for j := 0; !isOk && j < len(entity.Attr); j++ {
			isOk = entity.Attr[j].Name == readLt.Filter[k].Name
		}
		if !isOk {
			return "", errors.New("Error: entity " + entity.Name + " does not have attribute " + readLt.Filter[k].Name)
		}
	}

	// translate all calculations to sql
	for k := range calcLt.Calculation {

		mSql, _, err := partialTranslateToMicroSql(modelDef, entity, entityGen, aggrCols, paramCols, readLt, &calcLt.Calculation[k], runIds)
		if err != nil {
			return "", err
		}

		// merge main body SQL, expected to be unique, skip duplicates
		isFound := false
		for j := 0; !isFound && j < len(mainSql); j++ {
			isFound = mSql == mainSql[j]
		}
		if isFound {
			continue // skip duplicate SQL, it is the same source expression
		}
		mainSql = append(mainSql, mSql)
	}

	cteSql, err := makeMicroCteAggrSql(entity, entityGen, aggrCols, readLt.FromId, runIds)
	if err != nil {
		return "", errors.New("Error at making CTE for aggregation of " + entity.Name + ": " + err.Error())
	}
	pCteSql, err := makeParamCteSql(paramCols, readLt.FromId, runIds)
	if err != nil {
		return "", errors.New("Error at making CTE for parameters of " + entity.Name + ": " + err.Error())
	}
	if pCteSql != "" {
		cteSql += ", " + pCteSql
	}

	// make sql:
	// WITH cte array
	// SELECT main sql for aggregation 1
	// WHERE attribute filters
	// UNION ALL
	// SELECT main sql for aggregation 2
	// WHERE attribute filters
	// ORDER BY 1, 2,....

	sql := "WITH " + cteSql

	for k := range mainSql {
		if k > 0 {
			sql = sql + " UNION ALL " + mainSql[k]
		} else {
			sql = sql + " " + mainSql[k]
		}
	}

	// append ORDER BY, default order by: run_id, calculation id, group by attributes
	sql += makeOrderBy(len(calcLt.GroupBy), readLt.OrderBy, 2)

	return sql, nil
}

// make parameter columns: map of ["param.Name"] to parameter metadata and is number flag, which is true if parameter is a float or integer scalar
func makeParamCols(paramMeta []ParamMeta) map[string]paramColumn {

	pc := make(map[string]paramColumn, len(paramMeta))

	for k := 0; k < len(paramMeta); k++ {

		pc["param."+paramMeta[k].Name] = paramColumn{
			paramRow: &paramMeta[k].ParamDicRow,
			isNumber: paramMeta[k].ParamDicRow.Rank == 0 && (paramMeta[k].typeOf.IsFloat() || paramMeta[k].typeOf.IsInt()),
		}
	}
	return pc
}

// Build CTE part of calculation sql from the list of parameter columns.
//
// No comparison, simple use of parameter scalar value:
//
//	par_103 (run_id, param_value) AS  (.... AVG(param_value) FROM Extra.... WHERE RP.run_id IN (219, 221, 222) GROUP BY RP.run_id)
//
// run comparison:
//
//	pbase_103 (param_base)        AS  (SELECT AVG(param_value) ....WHERE RP.run_id = 219),
//	pvar_103  (run_id, param_var) AS  (.... WHERE RP..run_id IN (221, 222) GROUP BY RP.run_id)
func makeParamCteSql(paramCols map[string]paramColumn, fromId int, runIds []int) (string, error) {

	if fromId <= 0 {
		if len(runIds) <= 0 {
			return "", errors.New("Error: unable to make paramters CTE, run list is empty")
		}
		fromId = runIds[0]
	}

	extraIds := make([]int, 0, len(runIds)) // list of additional run Id's, different from base run id
	for _, rId := range runIds {
		if rId != fromId {
			extraIds = append(extraIds, rId)
		}
	}
	sort.Ints(extraIds)

	extraLst := ""
	for _, rId := range extraIds {
		if extraLst != "" {
			extraLst += ", "
		}
		extraLst += strconv.Itoa(rId)
	}

	// for each parameter make CTE for simple, [base] and [variant] use of parameter scalar
	lastId := -1
	cte := ""

	for {

		// walk through parameters in the order of parameter id's
		minId := -1
		minKey := ""
		for pKey, pCol := range paramCols {

			if !pCol.isSimple && !pCol.isBase && !pCol.isVar {
				continue // skip: parameter not used
			}
			if pCol.paramRow.ParamId > lastId && (minId < 0 || pCol.paramRow.ParamId < minId) {
				minId = pCol.paramRow.ParamId
				minKey = pKey
			}
		}
		if minKey == "" {
			break // done with all parameters
		}
		lastId = minId

		pCol := paramCols[minKey]
		if !pCol.isNumber || pCol.paramRow == nil {
			return "", errors.New("Error: parameter must a be numeric scalar: " + minKey)
		}
		sHid := strconv.Itoa(pCol.paramRow.ParamHid)

		// simple use of sacalr parameter, no run comparison:
		//
		// par_103 (run_id, param_value) AS
		// (
		//   SELECT
		//     RP.run_id, AVG(C.param_value)
		//   FROM StartingSeed_p_2012819 C
		//   INNER JOIN run_parameter RP ON (RP.base_run_id = C.run_id AND RP.parameter_hid = 103)
		//   WHERE RP.run_id = 219
		//   GROUP BY RP.run_id
		// )
		if pCol.isSimple {
			if cte != "" {
				cte += ", "
			}
			cte += "par_" + sHid + " (run_id, param_value) AS" +
				" (" +
				"SELECT RP.run_id, AVG(C.param_value)" +
				" FROM " + pCol.paramRow.DbRunTable + " C" +
				" INNER JOIN run_parameter RP ON (RP.base_run_id = C.run_id AND RP.parameter_hid = " + sHid + ")" +
				" WHERE RP.run_id"

			if extraLst == "" {
				cte += " = " + strconv.Itoa(fromId)
			} else {
				cte += " IN (" + strconv.Itoa(fromId) + ", " + extraLst + ")"
			}
			cte += " GROUP BY RP.run_id" +
				")"
		}

		// run comparison [base] run parameter
		//
		// pbase_103 (param_base) AS
		// (
		//   SELECT
		//     AVG(C.param_value)
		//   FROM StartingSeed_p_2012819 C
		//   INNER JOIN run_parameter RP ON (RP.base_run_id = C.run_id AND RP.parameter_hid = 103)
		//   WHERE RP.run_id = 219
		// )
		if pCol.isBase {
			if cte != "" {
				cte += ", "
			}
			cte += "pbase_" + sHid + " (param_base) AS" +
				" (" +
				"SELECT AVG(C.param_value)" +
				" FROM " + pCol.paramRow.DbRunTable + " C" +
				" INNER JOIN run_parameter RP ON (RP.base_run_id = C.run_id AND RP.parameter_hid = " + sHid + ")" +
				" WHERE RP.run_id = " + strconv.Itoa(fromId) +
				")"
		}

		// run comparison [variant] value of scalar parameter
		//
		// pvar_103 (run_id, param_var) AS
		// (
		//   SELECT
		//     RP.run_id, AVG(C.param_value)
		//   FROM StartingSeed_p_2012819 C
		//   INNER JOIN run_parameter RP ON (RP.base_run_id = C.run_id AND RP.parameter_hid = 103)
		//   WHERE RP.run_id IN (221, 222)
		//   GROUP BY RP.run_id
		// )
		if pCol.isVar {
			if cte != "" {
				cte += ", "
			}
			if extraLst == "" {
				return "", errors.New("Invalid (empty) list of variant runs to get a parameter: " + minKey)
			}
			cte += "pvar_" + sHid + " (run_id, param_var) AS" +
				" (" +
				"SELECT RP.run_id, AVG(C.param_value)" +
				" FROM " + pCol.paramRow.DbRunTable + " C" +
				" INNER JOIN run_parameter RP ON (RP.base_run_id = C.run_id AND RP.parameter_hid = " + sHid + ")" +
				" WHERE RP.run_id IN (" + extraLst + ")" +
				" GROUP BY RP.run_id" +
				")"
		}
	}

	return cte, nil
}
