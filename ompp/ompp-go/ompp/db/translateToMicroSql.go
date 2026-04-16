// Copyright (c) 2021 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"errors"
	"slices"
	"strconv"
)

// Translate microdata aggregation into sql query.
func translateToMicroAggrSql(
	modelDef *ModelMeta, entity *EntityMeta, entityGen *EntityGenMeta, readLt *ReadLayout, calcLt *CalculateLayout, groupBy []string, runIds []int,
) (string, error) {

	// make sql:
	// WITH cte array
	// SELECT main sql for calculation
	// WHERE run id IN (....)
	// AND dimension filters
	// ORDER BY 1, 2,....
	// default order by: run_id, calculation id, group by attributes

	aggrCols, err := makeMicroAggrCols(entity, entityGen, groupBy)
	if err != nil {
		return "", err
	}
	paramCols := makeParamCols(modelDef.Param)

	// validate filter names: it must be name of attribute or name of calculated attribute
	for k := range readLt.Filter {

		isOk := calcLt.Name == readLt.Filter[k].Name

		for j := 0; !isOk && j < len(entity.Attr); j++ {
			isOk = entity.Attr[j].Name == readLt.Filter[k].Name
		}
		if !isOk {
			return "", errors.New("Error: entity " + entity.Name + " does not have attribute " + readLt.Filter[k].Name)
		}
	}

	// translate calculation to sql
	mainSql, _, err := partialTranslateToMicroSql(modelDef, entity, entityGen, aggrCols, paramCols, readLt, calcLt, runIds)
	if err != nil {
		return "", err
	}

	cteSql, err := makeMicroCteAggrSql(entity, entityGen, aggrCols, readLt.FromId, runIds)
	if err != nil {
		return "", errors.New("Error at " + entity.Name + " " + calcLt.Calculate + ": " + err.Error())
	}
	pCteSql, err := makeParamCteSql(paramCols, readLt.FromId, runIds)
	if err != nil {
		return "", errors.New("Error at " + entity.Name + " " + calcLt.Calculate + ": " + err.Error())
	}
	if pCteSql != "" {
		cteSql += ", " + pCteSql
	}

	return "WITH " + cteSql + mainSql + makeOrderBy(len(groupBy), readLt.OrderBy, 2), nil
}

// create list of microdata columns, set column names, group by flag and aggregatable flag for float and int attributes
func makeMicroAggrCols(entity *EntityMeta, entityGen *EntityGenMeta, groupBy []string) ([]aggrColumn, error) {

	// aggregation expression columns: only native numeric attributes can be aggregated
	aggrCols := make([]aggrColumn, len(entityGen.GenAttr))

	for k := range entityGen.GenAttr {

		// find entity attribute
		aIdx, ok := entity.AttrByKey(entityGen.GenAttr[k].AttrId)
		if !ok {
			return []aggrColumn{}, errors.New("model entity attribute not found by id: " + entity.Name + ": " + strconv.Itoa(entityGen.GenAttr[k].AttrId))
		}

		isGroup := false
		for j := 0; !isGroup && j < len(groupBy); j++ {
			isGroup = groupBy[j] == entity.Attr[aIdx].Name
		}

		aggrCols[k] = aggrColumn{
			name:    entity.Attr[aIdx].Name,
			colName: entity.Attr[aIdx].colName,
			isGroup: isGroup,
			isAggr:  entity.Attr[aIdx].typeOf.IsFloat() || entity.Attr[aIdx].typeOf.IsInt(), // only numeric attributes can be aggregated
		}
	}

	// validate group by attributes
	for k := range groupBy {

		isOk := false
		for j := 0; !isOk && j < len(aggrCols); j++ {
			isOk = groupBy[k] == aggrCols[j].name
		}
		if !isOk {
			return []aggrColumn{}, errors.New("Entity " + entity.Name + " does not have attribute " + groupBy[k])
		}
	}

	return aggrCols, nil
}

// Translate microdata aggregation to sql query, apply dimension filters and selected run id's.
// Return main sql and run comparison flag.
//
// It can be a multiple runs comparison and base run id is layout.FromId
// Or simple expression calculation inside of single run, in that case layout.FromId and runIds[] are merged.
// Only simple functions allowed in expression calculation.
func partialTranslateToMicroSql(
	modelDef *ModelMeta, entity *EntityMeta, entityGen *EntityGenMeta, aggrCols []aggrColumn, paramCols map[string]paramColumn, readLt *ReadLayout, calcLt *CalculateLayout, runIds []int,
) (
	string, bool, error,
) {

	// translate microdata aggregation expression into sql query
	//
	// no comparison, aggregation for each run: OM_SUM(Income - 0.5 * OM_AVG(Pension) * param.Extra)
	//
	// WITH atts  (run_id, entity_key, attr1, attr2, attr3, attr8) AS (.... WHERE RE.run_id IN (219, 221, 222)),
	//      par_103 (run_id, param_value) AS  (.... AVG(param_value) FROM Extra.... WHERE RP.run_id IN (219, 221, 222) GROUP BY RP.run_id)
	// SELECT
	//   A.run_id, CalcId AS calc_id, A.attr1, A.attr2, A.calc_value
	// FROM
	// (
	//   SELECT
	//     M1.run_id, M1.attr1, M1.attr2,
	//     SUM(M1.attr3 - 0.5 * L1E1.ex1 * M1P103.param_value) AS calc_value
	//   FROM atts M1
	//   INNER JOIN par_103 M1P103 ON (M1P103.run_id = M1.run_id)
	//   INNER JOIN ( .... ) ON (run_id, attr1, attr2)
	//   GROUP BY M1.run_id, M1.attr1, M1.attr2
	// ) A
	// WHERE A.attr1 = .....
	//
	// microdata run comparison: OM_AVG( Income[variant] - (Pension[base] + Salary[base]) )
	//
	// WITH abase   (run_id, entity_key, attr1, attr2, attr4, attr8)   AS (.... WHERE RE.run_id = 219),
	//      avar    (run_id, entity_key, attr1, attr2, attr3)          AS (.... WHERE RE.run_id IN (221, 222)),
	//      abv (run_id, attr1, attr2, attr4_base, attr8_base, attr3_var) AS (....  FROM abase B INNER JOIN avar V ON (V.entity_key = B.entity_key) ),
	//      pbase_103 (param_value)         AS  (SELECT AVG(param_value) ....WHERE RP.run_id = 219),
	//      pvar_103  (run_id, param_value) AS  (.... WHERE RP..run_id IN (221, 222) GROUP BY RP.run_id)
	// SELECT
	//   A.run_id, CalcId AS calc_id, A.attr1, A.attr2, A.calc_value
	// FROM
	// (
	//   SELECT
	//     M1.run_id, M1.attr1, M1.attr2,
	//     AVG(M1.attr3_var - (M1.attr8_base + M1.attr4_base)) AS calc_value
	//   FROM abv M1
	//   INNER JOIN pbase_103 M1PB103
	//   INNER JOIN pvar_103  M1PV103 ON (M1PV103.run_id = M1.run_id)
	//   INNER JOIN ( .... ) ON (run_id, attr1, attr2)
	//   GROUP BY M1.run_id, M1.attr1, M1.attr2
	// ) A
	// WHERE A.attr1 = .....
	//
	mainSql, isRunCompare, err := translateMicroCalcToSql(entity, entityGen, aggrCols, paramCols, calcLt.CalcId, calcLt.Calculate)
	if err != nil {
		return "", false, errors.New("Error at " + entity.Name + " " + calcLt.Calculate + ": " + err.Error())
	}

	iDbl, ok := modelDef.TypeOfDouble()
	if !ok {
		return "", false, errors.New("double type not found, entity " + entity.Name)
	}

	// append attribute enum code filters and value filters, if specified: A.attr1 = 'M' AND (calc_value < 1234 AND calc_id = 12001)
	where := ""

	for k := range readLt.Filter {

		var err error
		f := ""

		if calcLt.Name == readLt.Filter[k].Name { // check if this is a filter by calculated value

			f, err = makeWhereValueFilter(
				&readLt.Filter[k], "", "calc_value", "calc_id", calcLt.CalcId, &modelDef.Type[iDbl], readLt.Filter[k].Name, "entity "+entity.Name)
			if err != nil {
				return "", false, err
			}
		}
		if f == "" { // if not a filter by value then it can be filter by dimension

			aix := -1
			for j := range entity.Attr {
				if entity.Attr[j].Name == readLt.Filter[k].Name {
					aix = j
					break
				}
			}
			if aix >= 0 {

				f, err = makeWhereFilter(
					&readLt.Filter[k], "A", entity.Attr[aix].colName, entity.Attr[aix].typeOf, false, entity.Attr[aix].Name, "entity "+entity.Name)
				if err != nil {
					return "", false, errors.New("Error at " + entity.Name + " " + calcLt.Calculate + ": " + err.Error())
				}
			}
		}
		// use filter: it is a filter by attribute name or by current calculated column name
		if f != "" {
			if where == "" {
				where = f
			} else {
				where += " AND " + f
			}
		}
	}

	// append attribute enum id filters, if specified
	for k := range readLt.FilterById {

		// find attribute index by name
		aix := -1
		for j := range entity.Attr {
			if entity.Attr[j].Name == readLt.FilterById[k].Name {
				aix = j
				break
			}
		}
		if aix < 0 {
			return "", false, errors.New("Error at " + entity.Name + " " + calcLt.Calculate + ": entity " + entity.Name + " does not have attribute " + readLt.FilterById[k].Name)
		}

		f, err := makeWhereIdFilter(
			&readLt.FilterById[k], "A", entity.Attr[aix].colName, entity.Attr[aix].typeOf, entity.Attr[aix].Name, "entity "+entity.Name)
		if err != nil {
			return "", false, errors.New("Error at " + entity.Name + " " + calcLt.Calculate + ": " + err.Error())
		}

		if where == "" {
			where = f
		} else {
			where += " AND " + f
		}
	}

	// append WHERE to main sql query if where filters not empty
	if where != "" {
		mainSql += " WHERE " + where
	}

	return mainSql, isRunCompare, nil
}

// Translate microdata aggregation into main sql query.
// Calculation must return a single value as a result of aggregation, ex.: AVG(attr1).
//
// Return sql SELECT for value calculation and run comparison flag:
// if true then it is multiple runs comparison else expression calculation inside of a single run(s).
//
// It aggregation for one or multiple runs: OM_SUM(Income - 0.5 * OM_AVG(Pension) * param.Extra)
//
//		WITH atts (run_id, entity_key, attr1, attr2, attr3, attr8) AS (.... WHERE RE.run_id IN (219, 221, 222))
//	         par_103 (run_id, param_value) AS  (.... AVG(param_value) FROM Extra.... WHERE RP.run_id IN (219, 221, 222) GROUP BY RP.run_id)
//		SELECT
//		  A.run_id, CalcId AS calc_id, A.attr1, A.attr2, A.calc_value
//		FROM
//		(...., SUM(M1.attr3 - 0.5 * L1E1.ex1 * M1P103.param_value) AS calc_value
//		  GROUP BY M1.run_id, M1.attr1, M1.attr2
//		) A
//
// microdata run comparison: OM_AVG( Income[variant] - (Pension[base] + Salary[base]) + (param.Extra[variant] - param.Extra[base]) )
//
//			WITH abase (run_id, entity_key, attr1, attr2, attr4, attr8)   AS (.... WHERE RE.run_id = 219),
//			     avar  (run_id, entity_key, attr1, attr2, attr3)          AS (.... WHERE RE.run_id IN (221, 222)),
//			     abv (run_id, attr1, attr2, attr4_base, attr8_base, attr3_var) AS (....  FROM abase B INNER JOIN avar V ON (V.entity_key = B.entity_key) ),
//		      pbase_103 (param_base)        AS  (SELECT AVG(param_value) ....WHERE RP.run_id = 219),
//		      pvar_103  (run_id, param_var) AS  (.... WHERE RP..run_id IN (221, 222) GROUP BY RP.run_id)
//			SELECT
//			  A.run_id, CalcId AS calc_id, A.attr1, A.attr2, A.calc_value
//			FROM
//			(...., AVG(M1.attr3_var - (M1.attr8_base + M1.attr4_base) + (M1PV103.param_var - M1PB103.param_base)) AS calc_value
//			  FROM abv M1
//	          INNER JOIN pbase_103 M1PB103
//	          INNER JOIN pvar_103  M1PV103 ON (M1PV103.run_id = M1.run_id)
//			  GROUP BY M1.run_id, M1.attr1, M1.attr2
//			) A
func translateMicroCalcToSql(
	entity *EntityMeta, entityGen *EntityGenMeta, aggrCols []aggrColumn, paramCols map[string]paramColumn, calcId int, calculateExpr string,
) (
	string, bool, error,
) {

	// clean source calculation from cr lf and unsafe sql quotes
	// return error if unsafe sql or comment found outside of 'quotes', ex.: -- ; DELETE INSERT UPDATE...
	// clean source calculation from cr lf and unsafe sql quotes
	// return error if unsafe sql or comment found outside of 'quotes', ex.: -- ; DELETE INSERT UPDATE...
	startExpr := cleanSourceExpr(calculateExpr)
	err := errorIfUnsafeSqlOrComment(startExpr)
	if err != nil {
		return "", false, err
	}

	// translate (substitute) all simple functions: OM_DIV_BY OM_IF...
	startExpr, err = translateAllSimpleFnc(startExpr)
	if err != nil {
		return "", false, err
	}

	// produce attribute column name:
	// if it is not a run comparison:      attr3 => L1A4.attr3
	// or for base and variant attributes: L1A4.attr3_var, L1A4.attr4_base, L1A4.attr8_base
	isAttrSimple := false
	isAttrBase := false
	isAttrVar := false

	makeAttrColName := func(
		name string, nameIdx int, isSimple, isVar bool, firstAlias string, levelAlias string, isFirst bool,
	) string {

		if !isSimple {
			if isVar {
				isAttrVar = true
				aggrCols[nameIdx].isVar = true
				return firstAlias + "." + aggrCols[nameIdx].colName + "_var" // variant run attribute: attr1[variant] => L1A4.attr1_var
			}
			isAttrBase = true
			aggrCols[nameIdx].isBase = true
			return firstAlias + "." + aggrCols[nameIdx].colName + "_base" // base run attribute: attr1[base] => L1A4.attr1_base
		}
		// else: isSimple name, not a name[base] or name[variant]
		isAttrSimple = true
		aggrCols[nameIdx].isSimple = true
		return firstAlias + "." + aggrCols[nameIdx].colName // not a run comparison: attr2 => L1A4.attr2
	}

	// translate parameter names by replacing it with CTE alias and CTE parameter value name:
	//	param.Name          => M1P103.param_value
	//	param.Name[base]    => M1PB103.param_base
	//	param.Name[variant] => M1PV103.param_var
	// also return INNER JOIN between parameter CTE view and main table:
	//  INNER JOIN par_103   M1P103 ON (M1P103.run_id = M1.run_id)
	//  INNER JOIN pbase_103 M1PB103
	//  INNER JOIN pvar_103  M1PV103 ON (M1P103.run_id = M1.run_id)
	isParamSimple := false
	isParamBase := false
	isParamVar := false

	makeParamColName := func(colKey string, isSimple, isVar bool, alias string) (string, string, error) {

		pCol, ok := paramCols[colKey]
		if !ok {
			return "", "", errors.New("Error: parameter not found: " + colKey)
		}
		if !pCol.isNumber || pCol.paramRow == nil {
			return "", "", errors.New("Error: parameter must a be numeric scalar: " + colKey)
		}

		sqlName := ""
		innerJoin := ""
		sHid := strconv.Itoa(pCol.paramRow.ParamHid)
		if isSimple {
			isParamSimple = true
			pCol.isSimple = true
			pa := alias + "P" + sHid
			sqlName = pa + ".param_value" // not a run comparison: param.Name => M1P103.param_value
			innerJoin = "INNER JOIN par_" + sHid + " " + pa + " ON (" + pa + ".run_id = " + alias + ".run_id)"
		} else {
			if isVar {
				isParamVar = true
				pCol.isVar = true
				pa := alias + "PV" + strconv.Itoa(pCol.paramRow.ParamHid)
				sqlName = pa + ".param_var" // variant run parameter: param.Name[variant] => M1PV103.param_var
				innerJoin = "INNER JOIN pvar_" + sHid + " " + pa + " ON (" + pa + ".run_id = " + alias + ".run_id)"
			} else {
				isParamBase = true
				pCol.isBase = true
				pa := alias + "PB" + strconv.Itoa(pCol.paramRow.ParamHid)
				sqlName = pa + ".param_base" // base run parameter: param.Name[base] => M1PB103.param_base
				innerJoin = "INNER JOIN pbase_" + sHid + " " + pa
			}
		}
		paramCols[colKey] = pCol

		return sqlName, innerJoin, nil
	}

	// parse aggregation expression
	levelArr, err := parseAggrCalculation(aggrCols, paramCols, startExpr, makeAttrColName, makeParamColName)
	if err != nil {
		return "", false, err
	}

	// validate attribute names:
	// all names must be either with suffixes: attr3[base], attr4[variant] or in simple form: attr3, attr4
	// if it is run comparison then both [base] and [variant] forms must be used, it cannot be only [base] or only [variant]

	if isAttrSimple && (isAttrBase || isAttrVar) ||
		!isAttrSimple && (isAttrBase && !isAttrVar || !isAttrBase && isAttrVar) {
		return "", false, errors.New("invalid (or mixed forms) of attribute names used for aggregation of: " + entity.Name + ": " + calculateExpr)
	}
	if !isAttrSimple && !isAttrBase && !isAttrVar {
		return "", false, errors.New("error: there are no attribute names found for aggregation of: " + entity.Name + ": " + calculateExpr)
	}
	isCompare := isAttrBase && isAttrVar

	// validate parameter names:
	// if it is run comparison then parameter name cannot be simple else parameter name cannot be [base] or [variant]
	if isCompare && isParamSimple {
		return "", false, errors.New("invalid use of parameter name in microdata run comparison: " + entity.Name + ": " + calculateExpr)
	}
	if !isCompare && (isParamBase || isParamVar) {
		return "", false, errors.New("invalid use of parameter run comparison name in microdata aggregation: " + entity.Name + ": " + calculateExpr)
	}

	// build main part of aggregation sql from parser state
	mainSql, err := makeMicroMainAggrSql(entity, entityGen, aggrCols, paramCols, calcId, levelArr, isCompare)
	if err != nil {
		return "", false, err
	}

	return mainSql, isCompare, nil
}

// Build main part of aggregation sql from parser state.
func makeMicroMainAggrSql(
	entity *EntityMeta, entityGen *EntityGenMeta, aggrCols []aggrColumn, paramCols map[string]paramColumn, calcId int, levelArr []levelDef, isRunCompare bool,
) (
	string, error,
) {

	// no comparison, microdata aggregation for each run: OM_SUM(Income - 0.5 * OM_AVG(Pension))
	//
	// -- WITH atts (run_id, entity_key, attr1, attr2, attr3, attr8) AS (....)
	// -- par_103 (run_id, param_value) AS  (.... AVG(param_value) FROM Extra.... WHERE RP.run_id IN (219, 221, 222) GROUP BY RP.run_id)
	//
	// SELECT
	//   A.run_id, CalcId AS calc_id, A.attr1, A.attr2, A.calc_value
	// FROM
	// (
	//   SELECT
	//     M1.run_id, M1.attr1, M1.attr2,
	//     SUM(M1.attr3 - 0.5 * L1E1.ex1 * M1P103.param_value) AS calc_value
	//   FROM atts M1
	//   INNER JOIN par_103 M1P103 ON (M1P103.run_id = M1.run_id)
	//   INNER JOIN
	//   (
	//     SELECT
	//       M2.run_id, M2.attr1, M2.attr2,
	//       AVG(M2.attr8) AS ex1
	//     FROM atts M2
	//     GROUP BY M2.run_id, M2.attr1, M2.attr2
	//   ) L1A1
	//   ON (L1A1.run_id = M1.run_id AND L1A1.attr1 = M1.attr1 AND L1A1.attr2 = M1.attr2)
	//   GROUP BY M1.run_id, M1.attr1, M1.attr2
	// ) A
	//
	// microdata run comparison: OM_AVG( Income[variant] - (Pension[base] + Salary[base]) + (param.Extra[variant] - param.Extra[base]) )
	//
	// -- WITH abase (run_id, entity_key, attr1, attr2, attr4, attr8) AS (....),
	// -- avar (run_id, entity_key, attr1, attr2, attr3) AS (....),
	// -- abv (run_id, attr1, attr2, attr3_var, attr4_base, attr8_base) AS (....),
	// -- pbase_103 (param_base)        AS  (SELECT AVG(param_value) ....WHERE RP.run_id = 219),
	// -- pvar_103  (run_id, param_var) AS  (.... WHERE RP..run_id IN (221, 222) GROUP BY RP.run_id)
	//
	// SELECT
	//   A.run_id, CalcId AS calc_id, A.attr1, A.attr2, A.calc_value
	// FROM
	// (
	//   SELECT
	//     M1.run_id, M1.attr1, M1.attr2,
	//     AVG(M1.attr3_var - (M1.attr8_base + M1.attr4_base) + (M1PV103.param_var - M1PB103.param_base)) AS calc_value
	//   FROM abv M1
	//   INNER JOIN pbase_103 M1PB103
	//   INNER JOIN pvar_103  M1PV103 ON (M1PV103.run_id = M1.run_id)
	//   GROUP BY M1.run_id, M1.attr1, M1.attr2
	// ) A
	//
	vSrc := "atts"
	if isRunCompare {
		vSrc = "abv"
	}

	// SELECT  A.run_id, CalcId AS calc_id, A.attr1, A.attr2, A.calc_value FROM (
	//
	mainSql := "SELECT A.run_id, " + strconv.Itoa(calcId) + " AS calc_id"

	for _, c := range aggrCols {
		if c.isGroup {
			mainSql += ", A." + c.colName
		}
	}
	mainSql += ", A.calc_value FROM ( "

	// main aggregation sql body
	for nLev, lv := range levelArr {

		//   SELECT
		//     M1.run_id, M1.attr1, M1.attr2,
		//     SUM(M1.attr3 - 0.5 * L1E1.ex1 * M1P103.param_value) AS calc_value
		//   FROM atts M1
		//   INNER JOIN par_103 M1P103 ON (M1P103.run_id = M1.run_id)
		//   INNER JOIN
		//   (
		mainSql += "SELECT " + lv.fromAlias + ".run_id"

		for _, c := range aggrCols {
			if c.isGroup {
				mainSql += ", " + lv.fromAlias + "." + c.colName
			}
		}

		for _, expr := range lv.exprArr {
			mainSql += ", " + expr.sqlExpr
			if expr.colName != "" {
				mainSql += " AS " + expr.colName
			}
		}

		mainSql += " FROM " + vSrc + " " + lv.fromAlias

		slices.Sort(lv.paramJoinArr)

		for _, pj := range lv.paramJoinArr {
			mainSql += " " + pj
		}

		if nLev < len(levelArr)-1 { // if not lowest level then continue INNER JOIN down to the next level
			mainSql += " INNER JOIN ("
		}
	}

	// for each level except of the lowest append:
	//     GROUP BY M2.run_id, M2.attr1, M2.attr2
	//   ) L1A1
	//   ON (L1A1.run_id = M1.run_id AND L1A1.attr1 = M1.attr1 AND L1A1.attr2 = M1.attr2)
	//
	for nLev := len(levelArr) - 1; nLev >= 0; nLev-- {

		mainSql += " GROUP BY " + levelArr[nLev].fromAlias + ".run_id"

		for _, c := range aggrCols {
			if c.isGroup {
				mainSql += ", " + levelArr[nLev].fromAlias + "." + c.colName
			}
		}

		if nLev > 0 {

			mainSql += ") " + levelArr[nLev].innerAlias +
				" ON (" + levelArr[nLev].innerAlias + ".run_id = " + levelArr[nLev-1].fromAlias + ".run_id"

			for _, c := range aggrCols {
				if c.isGroup {
					mainSql += " AND " + levelArr[nLev].innerAlias + "." + c.colName + " = " + levelArr[nLev-1].fromAlias + "." + c.colName
				}
			}

			mainSql += ")"
		}
	}
	mainSql += " ) A"

	return mainSql, nil
}

// Build CTE part of aggregation sql from the list of aggregated attributes.
func makeMicroCteAggrSql(
	entity *EntityMeta, entityGen *EntityGenMeta, aggrCols []aggrColumn, fromId int, runIds []int,
) (
	string, error,
) {

	// list of column names for CTE header and CTE body, add group by attributes in the order of attributes
	cHdr := "run_id, entity_key"
	cBody := "RE.run_id, C.entity_key"
	isAnySimple := false
	isAnyBase := false
	isAnyVar := false

	for _, c := range aggrCols {
		if c.isGroup {
			cHdr += ", " + c.colName
			cBody += ", C." + c.colName
		}
		if c.isSimple {
			isAnySimple = true
		}
		if c.isBase {
			isAnyBase = true
		}
		if c.isVar {
			isAnyVar = true
		}
	}
	if isAnyBase != isAnyVar {
		return "", errors.New("invalid (or mixed forms) of attribute names used for microdata comparison of: " + entity.Name)
	}

	// CTE: run comparison attributes and / or aggreagtion without comparison
	cteSql := ""

	if isAnySimple { // no comparison, select attributes from the list of model runs

		// atts (run_id, entity_key, attr1, attr2, attr3, attr8) AS
		// (
		//   SELECT
		//     RE.run_id, C.entity_key, C.attr1, C.attr2, C.attr4, C.attr8
		//   FROM Person_gfa43c687 C
		//   INNER JOIN run_entity RE ON (RE.base_run_id = C.run_id AND RE.entity_gen_hid = 201)
		//   WHERE RE.run_id IN (219, 221, 222)
		// )
		//
		cteSql = "atts (" + cHdr

		for k := range aggrCols {
			if aggrCols[k].isSimple {
				cteSql += ", " + aggrCols[k].colName
			}
		}

		cteSql += ") AS (SELECT " + cBody

		for k := range aggrCols {
			if aggrCols[k].isSimple {
				cteSql += ", C." + aggrCols[k].colName
			}
		}

		cteSql += " FROM " + entityGen.DbEntityTable + " C" +
			" INNER JOIN run_entity RE ON (RE.base_run_id = C.run_id AND RE.entity_gen_hid = " + strconv.Itoa(entityGen.GenHid) + ")" +
			" WHERE"

		if len(runIds) <= 0 {
			cteSql += " RE.run_id = " + strconv.Itoa(fromId)
		} else {
			cteSql += " RE.run_id IN (" + strconv.Itoa(fromId)

			for _, rId := range runIds {
				if rId != fromId {
					cteSql += ", " + strconv.Itoa(rId)
				}
			}
			cteSql += ")"
		}
		cteSql += ")"
	}

	if isAnyBase && isAnyVar { // run comparison: select from variant runs join to base run

		// microdata run comparison: OM_AVG( Income[variant] - (Pension[base] + Salary[base]) )
		//
		// abase (run_id, entity_key, attr1, attr2, attr4, attr8) AS
		// (
		//   SELECT
		// 	   RE.run_id, C.entity_key, C.attr1, C.attr2, C.attr4, C.attr8
		//   FROM Person_gfa43c687 C
		//   INNER JOIN run_entity RE ON (RE.base_run_id = C.run_id AND RE.entity_gen_hid = 201)
		//   WHERE RE.run_id = 219
		// ),
		// avar (run_id, entity_key, attr1, attr2, attr3) AS
		// (
		//   SELECT
		// 	   RE.run_id, C.entity_key, C.attr1, C.attr2, C.attr3
		//   FROM Person_gfa43c687 C
		//   INNER JOIN run_entity RE ON (RE.base_run_id = C.run_id AND RE.entity_gen_hid = 201)
		//   WHERE RE.run_id IN (221, 222)
		// ),
		// abv (run_id, attr1, attr2, attr4_base, attr8_base, attr3_var)
		// AS
		// (
		//   SELECT
		// 	   V.run_id, V.attr1, V.attr2, B.attr4 AS attr4_base, B.attr8 AS attr8_base, V.attr3 AS attr3_var
		//   FROM abase B
		//   INNER JOIN avar V ON (V.entity_key = B.entity_key)
		// )
		//
		if cteSql == "" {
			cteSql = "abase (" + cHdr
		} else {
			cteSql += ", abase (" + cHdr
		}

		for k := range aggrCols {
			if aggrCols[k].isBase {
				cteSql += ", " + aggrCols[k].colName
			}
		}

		cteSql += ") AS (SELECT " + cBody

		for k := range aggrCols {
			if aggrCols[k].isBase {
				cteSql += ", C." + aggrCols[k].colName
			}
		}

		cteSql += " FROM " + entityGen.DbEntityTable + " C" +
			" INNER JOIN run_entity RE ON (RE.base_run_id = C.run_id AND RE.entity_gen_hid = " + strconv.Itoa(entityGen.GenHid) + ")" +
			" WHERE RE.run_id = " + strconv.Itoa(fromId) +
			")"

		// variant runs
		cteSql += ", avar (" + cHdr

		for k := range aggrCols {
			if aggrCols[k].isVar {
				cteSql += ", " + aggrCols[k].colName
			}
		}

		cteSql += ") AS (SELECT " + cBody

		for k := range aggrCols {
			if aggrCols[k].isVar {
				cteSql += ", C." + aggrCols[k].colName
			}
		}

		cteSql += " FROM " + entityGen.DbEntityTable + " C" +
			" INNER JOIN run_entity RE ON (RE.base_run_id = C.run_id AND RE.entity_gen_hid = " + strconv.Itoa(entityGen.GenHid) + ")" +
			" WHERE RE.run_id IN ("

		isF := true
		for _, rId := range runIds {
			if rId != fromId {
				if isF {
					isF = false
					cteSql += strconv.Itoa(rId)
				} else {
					cteSql += ", " + strconv.Itoa(rId)
				}
			}
		}
		cteSql += "))"

		// inner join of base and variant runs by entity_key
		cteSql += ", abv (run_id"

		for _, c := range aggrCols {
			if c.isGroup {
				cteSql += ", " + c.colName
			}
		}
		for k := range aggrCols {
			if aggrCols[k].isBase {
				cteSql += ", " + aggrCols[k].colName + "_base"
			}
		}
		for k := range aggrCols {
			if aggrCols[k].isVar {
				cteSql += ", " + aggrCols[k].colName + "_var"
			}
		}

		cteSql += ") AS (SELECT V.run_id"

		for _, c := range aggrCols {
			if c.isGroup {
				cteSql += ", V." + c.colName
			}
		}
		for k := range aggrCols {
			if aggrCols[k].isBase {
				cteSql += ", B." + aggrCols[k].colName + " AS " + aggrCols[k].colName + "_base"
			}
		}
		for k := range aggrCols {
			if aggrCols[k].isVar {
				cteSql += ", V." + aggrCols[k].colName + " AS " + aggrCols[k].colName + "_var"
			}
		}

		cteSql += " FROM abase B" +
			" INNER JOIN avar V ON (V.entity_key = B.entity_key)" +
			")"
	}

	return cteSql, nil
}
