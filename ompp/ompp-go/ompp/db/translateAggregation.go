// Copyright (c) 2021 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"errors"
	"strconv"
	"strings"
)

// aggregation expression and expression column name
type aggrExprColumn struct {
	colName string // column name, ie: ex2
	srcExpr string // source expression, ie: OM_AVG(acc0)
	sqlExpr string // sql expresiion, ie: AVG(M2.acc0)
}

// accumulator or attribute column used for aggregation calculation or group by attribute
type aggrColumn struct {
	name     string // accumulator or attribute name VARCHAR(255) NOT NULL
	colName  string // db column name: acc1 or attr1
	isGroup  bool   // if true then it is group by column of microdata attributes
	isAggr   bool   // if true then it can be aggregated: accumulator is native (not a derived) or attribute has native type
	isBase   bool   // if true then attribute is used for the base run in comparison
	isVar    bool   // if true then attribute is used for the variant run in comparison
	isSimple bool   // if true then attribute is used in aggregation without comparison
}

// scalar parameter column used as value in calculated expression
type paramColumn struct {
	isNumber bool         // if true then parameter is numeric scalar: zero rank integer or float
	isBase   bool         // if true then parameter is used for the base run in comparison
	isVar    bool         // if true then parameter is used for the variant run in comparison
	isSimple bool         // if true then parameter is used in aggregation without comparison
	paramRow *ParamDicRow // db row of parameter_dic join to model_parameter_dic table
}

// Parsed aggregation expressions for each nesting level
type levelDef struct {
	level          int              // nesting level
	fromAlias      string           // from table alias
	innerAlias     string           // inner join table alias
	nextInnerAlias string           // next level inner join table alias
	exprArr        []aggrExprColumn // column names and expressions
	paramJoinArr   []string         // parameters inner join: inner join between parameter CTE and main table
	firstAgcIdx    int              // first used aggregation column index (accumulator or attribute index)
	agcUsageArr    []bool           // contains true if aggregation column (accumulator or attribute) used at current level
}

// level parse state
type levelParseState struct {
	*levelDef                       // current level
	nextExprNumber int              // number of aggregation epxpressions
	nextExprArr    []aggrExprColumn // aggregation expressions for the next level
}

// Parse output table accumulators calculation.
func parseAggrCalculation(
	aggrCols []aggrColumn,
	paramCols map[string]paramColumn,
	calculateExpr string,
	makeAggrColName func(string, int, bool, bool, string, string, bool) string,
	makeParamColName func(string, bool, bool, string) (string, string, error),
) (
	[]levelDef, error,
) {

	// start with source expression and column name
	nLevel := 1

	levelArr := []levelDef{
		levelDef{
			level:          nLevel,
			fromAlias:      "M" + strconv.Itoa(nLevel),
			innerAlias:     "T" + strconv.Itoa(nLevel),
			nextInnerAlias: "T" + strconv.Itoa(nLevel+1),
			exprArr: []aggrExprColumn{{
				colName: "calc_value",
				srcExpr: calculateExpr,
			}},
			paramJoinArr: []string{},
			agcUsageArr:  make([]bool, len(aggrCols)),
		}}
	lps := &levelParseState{
		levelDef:       &levelArr[len(levelArr)-1],
		nextExprNumber: 1,
		nextExprArr:    []aggrExprColumn{},
	}

	// return error if this is a top level and any all aggregation column exist in source expression.
	errorIfAnyAggrCol := func(src string) error {

		if nLevel > 1 || src == "" {
			return nil
		}
		var err error

		nStart := 0
		for nEnd := 0; nStart >= 0 && nEnd >= 0; {

			nStart, nEnd, err = nextUnquoted(src, nStart)
			if err != nil {
				return err
			}
			if nStart < 0 || nEnd < 0 { // end of source formula
				break
			}

			//  for each accumulator name check if name exist in that unquoted part of sql
			for k := 0; k < len(aggrCols); k++ {

				if findNamePos(src[nStart:nEnd], aggrCols[k].name) >= 0 {
					return errors.New("error: top level columns must be inside of aggregation function: " + aggrCols[k].name + " : " + calculateExpr)
				}
			}
			nStart = nEnd // to the next 'unquoted part' of calculation string
		}
		return nil
	}

	// until any function expressions exist on current level repeat translation:
	//
	//	OM_SUM(acc0 - 0.5 * OM_AVG(acc0))
	//  =>
	//  SUM(acc0 - 0.5 * T2.ex2)
	//	  => function: SUM(argument)
	//	  => argument: acc0 - 0.5 * OM_AVG(acc0)
	//	     => push OM_* functions as expression to the next level:
	//	        => current level sql column: OM_AVG(acc0) => T2.ex2
	//	        => next level expression:    OM_AVG(acc0)
	for {

		for nL := range lps.exprArr {

			// parse until source expression not completed
			sqlExpr := ""

			for src := lps.exprArr[nL].srcExpr; len(src) > 0; {

				// find most left (top level) aggregation function
				fncName, namePos, arg, nAfter, err := findFirstFnc(src, aggrFncLst)
				if err != nil {
					return []levelDef{}, err
				}
				if fncName == "" { // no any functions found, done with with this source expression

					// if this is the first level then it should be aggregation columns (accumulators) outside of aggregation functions
					if nLevel <= 1 {
						if err = errorIfAnyAggrCol(src); err != nil {
							return []levelDef{}, err
						}
					}
					sqlExpr += src // copy source expression into output
					break
				}

				// translate (substitute) aggregation function by sql fragment
				t, err := lps.translateAggregationFnc(fncName, arg, src)
				if err != nil {
					return []levelDef{}, err
				}

				// if this is the first level then it should be aggregation columns (accumulators) outside of aggregation functions
				if nLevel <= 1 && namePos > 0 {
					if err = errorIfAnyAggrCol(src[:namePos]); err != nil {
						return []levelDef{}, err
					}
				}
				// replace source
				sqlExpr += src[:namePos] + t
				if nAfter >= len(src) {
					break // done with source expression
				}
				src = src[nAfter:]
			}
			lps.exprArr[nL].sqlExpr = sqlExpr

			// accumultors first pass: collect accumulators usage in current sql expression
			var err error = nil

			nStart := 0
			for nEnd := 0; nStart >= 0 && nEnd >= 0; {

				nStart, nEnd, err = nextUnquoted(sqlExpr, nStart)
				if err != nil {
					return []levelDef{}, err
				}
				if nStart < 0 || nEnd < 0 { // end of source formula
					break
				}

				//  for each accumulator name check if name exist in that unquoted part of sql
				for k := 0; k < len(aggrCols); k++ {

					if findNamePos(sqlExpr[nStart:nEnd], aggrCols[k].name) >= 0 {
						lps.agcUsageArr[k] = true
					}
				}

				nStart = nEnd // to the next 'unquoted part' of calculation string
			}
		}

		// accumulators second pass: translate accumulators for all sql expressions
		// replace accumulators names by column name in joined accumulator table:
		//   acc0 => M1.acc_value
		//   acc2 => L1A2.acc2
		for ne := range lps.exprArr {

			var e error
			if lps.exprArr[ne].sqlExpr, e = lps.processAggrColumns(lps.exprArr[ne].sqlExpr, aggrCols, makeAggrColName); e != nil {
				return []levelDef{}, e
			}
		}

		// find parameter names and replace with column name:
		//   param.Name          => M1P103.param_value
		//   param.Name[base]    => M1PB103.param_base
		//   param.Name[variant] => M1PV103.param_var
		for ne := range lps.exprArr {

			var e error
			if lps.exprArr[ne].sqlExpr, e = lps.processParamColumns(lps.exprArr[ne].sqlExpr, paramCols, makeParamColName); e != nil {
				return []levelDef{}, e
			}
		}

		// if any expressions pushed to the next level then continue parsing
		if len(lps.nextExprArr) <= 0 {
			break
		}
		// else push aggregation expressions to the next level

		nLevel++
		levelArr = append(levelArr,
			levelDef{
				level:          nLevel,
				fromAlias:      "M" + strconv.Itoa(nLevel),
				innerAlias:     "T" + strconv.Itoa(nLevel),
				nextInnerAlias: "T" + strconv.Itoa(nLevel+1),
				exprArr:        append([]aggrExprColumn{}, lps.nextExprArr...),
				paramJoinArr:   []string{},
				agcUsageArr:    make([]bool, len(aggrCols)),
			})

		lps.levelDef = &levelArr[len(levelArr)-1]
		lps.nextExprArr = []aggrExprColumn{}
	}

	return levelArr, nil
}

// push OM_ function to next aggregation level and return column name
func (lps *levelParseState) pushToNextLevel(fncExpr string) string {

	colName := "ex" + strconv.Itoa(lps.nextExprNumber)
	lps.nextExprNumber++

	lps.nextExprArr = append(lps.nextExprArr,
		aggrExprColumn{
			colName: colName,
			srcExpr: fncExpr,
		})
	return colName
}

// Translate accumulator names by inserting table alias.
// If this is the first accumulator at this level then do: acc1 => M2.acc_value
// else use joined accumulator table: L1A4.acc4
func (lps *levelParseState) processAggrColumns(
	expr string, aggrCols []aggrColumn, makeColName func(string, int, bool, bool, string, string, bool) string,
) (
	string, error,
) {

	// find index of first used native (not a derived) accumulator
	// or index of first used aggregatin attribute
	lps.firstAgcIdx = -1
	for k, isUsed := range lps.agcUsageArr {
		if isUsed && aggrCols[k].isAggr {
			lps.firstAgcIdx = k
			break
		}
	}
	if lps.firstAgcIdx < 0 {
		return expr, nil // return source expression as is: no accumulators used in that expression
	}

	// for each 'unquoted' part of expression check if there is any table accumulator name
	// substitute each accumulator name with corresponding sql column name
	var err error = nil
	nStart := 0

	for nEnd := 0; nStart >= 0 && nEnd >= 0; {

		nStart, nEnd, err = nextUnquoted(expr, nStart)
		if err != nil {
			return "", err
		}
		if nStart < 0 || nEnd < 0 { // end of source expression
			break
		}

		// substitute first occurence of accumulator name with sql column name
		// for example: acc0 => M1.acc_value or acc4 => L1A4.acc4
		isFound := false

		for k := 0; !isFound && k < len(aggrCols); k++ {

			if !lps.agcUsageArr[k] || !aggrCols[k].isAggr { // only native accumulators can be aggregated and only native types attributes
				continue
			}

			n := findNamePos(expr[nStart:nEnd], aggrCols[k].name)
			if n >= 0 {
				isFound = true

				isFirstAgc := k == lps.firstAgcIdx
				firstAlias := lps.fromAlias
				levelAlias := "L" + strconv.Itoa(lps.level) + "A" + strconv.Itoa(k)

				nLen := len(aggrCols[k].name)
				isBase := false
				isVar := false

				switch {
				case strings.HasPrefix(expr[nStart+n+nLen:], "[variant]"):
					isVar = true
					nLen += len("[variant]")

				case strings.HasPrefix(expr[nStart+n+nLen:], "[base]"):
					isBase = true
					nLen += len("[base]")
				}

				col := makeColName(aggrCols[k].name, k, (!isBase && !isVar), isVar, firstAlias, levelAlias, isFirstAgc)

				expr = expr[:nStart+n] + col + expr[nStart+n+nLen:]
			}
		}

		if !isFound {
			nStart = nEnd // to the next 'unquoted part' of calculation string
		}
	}

	return expr, nil
}

// Translate parameter names by replacing it with CTE alias and CTE parameter value name:
//
//	param.Name          => M1P103.param_value
//	param.Name[base]    => M1PB103.param_base
//	param.Name[variant] => M1PV103.param_var
func (lps *levelParseState) processParamColumns(
	expr string, paramCols map[string]paramColumn, makeColName func(string, bool, bool, string) (string, string, error),
) (
	string, error,
) {

	// for each 'unquoted' part of expression check if there is any param.AnyName
	// if it is a paramter then substitute param.AnyName with corresponding sql column name
	var err error = nil
	nStart := 0

	for nEnd := 0; nStart >= 0 && nEnd >= 0; {

		nStart, nEnd, err = nextUnquoted(expr, nStart)
		if err != nil {
			return "", err
		}
		if nStart < 0 || nEnd < 0 { // end of source expression
			break
		}

		// substitute first occurence of accumulator name with sql column name
		// for example: acc0 => M1.acc_value or acc4 => L1A4.acc4
		isFound := false

		for pn := range paramCols {

			n := findNamePos(expr[nStart:nEnd], pn)
			if n >= 0 {
				isFound = true

				nLen := len(pn)
				isBase := false
				isVar := false

				switch {
				case strings.HasPrefix(expr[nStart+n+nLen:], "[variant]"):
					isVar = true
					nLen += len("[variant]")
				case strings.HasPrefix(expr[nStart+n+nLen:], "[base]"):
					isBase = true
					nLen += len("[base]")
				}

				cname, pJoin, e := makeColName(pn, (!isBase && !isVar), isVar, lps.fromAlias)
				if e != nil {
					return "", e
				}
				expr = expr[:nStart+n] + cname + expr[nStart+n+nLen:]

				isNew := true
				for k := 0; isNew && k < len(lps.paramJoinArr); k++ {
					isNew = lps.paramJoinArr[k] != pJoin
				}
				if isNew {
					lps.paramJoinArr = append(lps.paramJoinArr, pJoin)
				}

				break // parameter found
			}
		}

		if !isFound {
			nStart = nEnd // to the next 'unquoted part' of calculation string
		}
	}

	return expr, nil
}

// Translate aggregation function into sql expression:
//
//	OM_AVG(acc0) => AVG(acc0)
//
//	OM_SUM(acc0 - 0.5 * OM_AVG(acc0)) => SUM(acc0 - 0.5 * T2.ex2)
//
//	OM_VAR(acc0)
//	=>
//	OM_SUM((acc0 - OM_AVG(acc0)) * (acc0 - OM_AVG(acc0))) / (OM_COUNT(acc0) – 1)
//	=>
//	SUM((M1.acc0 - T2.ex2) * (acc0 - T2.ex2)) / (COUNT(acc0) – 1)
func (lps *levelParseState) translateAggregationFnc(name, arg string, src string) (string, error) {

	if len(arg) <= 0 {
		return "", errors.New("invalid (empty) function argument: " + name + " : " + src)
	}

	// translate function argument
	//   argument: acc0 - 0.5 * OM_AVG(acc0)
	//   return:   acc0 - 0.5 * T2.ex2
	//   push to the next level: OM_AVG(acc0)
	sqlArg, err := lps.translateArg(arg)
	if err != nil {
		return "", err
	}

	switch name {
	case "OM_AVG":
		return "AVG(" + sqlArg + ")", nil

	case "OM_SUM":
		return "SUM(" + sqlArg + ")", nil

	case "OM_COUNT":
		return "COUNT(" + sqlArg + ")", nil

	case "OM_MIN":
		return "MIN(" + sqlArg + ")", nil

	case "OM_MAX":
		return "MAX(" + sqlArg + ")", nil

	case "OM_COUNT_IF":
		return "COUNT(CASE WHEN " + sqlArg + " THEN 1 ELSE NULL END)", nil

	case "OM_VAR":
		// SUM((arg - T2.ex2) * (arg - T2.ex2)) / (COUNT(arg) - 1)
		// (COUNT(arg) - 1) =>
		//   CASE WHEN ABS( (COUNT(arg) - 1) ) > 1.0e-37 THEN (COUNT(arg) - 1) ELSE NULL END

		avgCol := lps.pushToNextLevel("OM_AVG(" + arg + ")")
		return "SUM((" +
				"(" + sqlArg + ") - " + lps.nextInnerAlias + "." + avgCol + ") * ((" + sqlArg + ") - " + lps.nextInnerAlias + "." + avgCol +
				"))" +
				" / CASE WHEN ABS( COUNT(" + sqlArg + ") - 1 ) > 1.0e-37 THEN COUNT(" + sqlArg + ") - 1 ELSE NULL END",
			nil

	case "OM_SD": // SQRT(var)

		avgCol := lps.pushToNextLevel("OM_AVG(" + arg + ")")
		return "SQRT(" +
				"SUM((" +
				"(" + sqlArg + ") - " + lps.nextInnerAlias + "." + avgCol + ") * ((" + sqlArg + ") - " + lps.nextInnerAlias + "." + avgCol +
				"))" +
				" / CASE WHEN ABS( COUNT(" + sqlArg + ") - 1 ) > 1.0e-37 THEN COUNT(" + sqlArg + ") - 1 ELSE NULL END" +
				" )",
			nil

	case "OM_SE": // SQRT(var / COUNT(arg))

		avgCol := lps.pushToNextLevel("OM_AVG(" + arg + ")")
		return "SQRT(" +
				"SUM((" +
				"(" + sqlArg + ") - " + lps.nextInnerAlias + "." + avgCol + ") * ((" + sqlArg + ") - " + lps.nextInnerAlias + "." + avgCol +
				"))" +
				" / CASE WHEN ABS( COUNT(" + sqlArg + ") - 1 ) > 1.0e-37 THEN COUNT(" + sqlArg + ") - 1 ELSE NULL END" +
				" / CASE WHEN ABS( COUNT(" + sqlArg + ") ) > 1.0e-37 THEN COUNT(" + sqlArg + ") ELSE NULL END" +
				" )",
			nil

	case "OM_CV": // 100 * ( SQRT(var) / AVG(arg) )

		avgCol := lps.pushToNextLevel("OM_AVG(" + arg + ")")
		return "100 * (" +
				" SQRT(" +
				"SUM((" +
				"(" + sqlArg + ") - " + lps.nextInnerAlias + "." + avgCol + ") * ((" + sqlArg + ") - " + lps.nextInnerAlias + "." + avgCol +
				"))" +
				" / CASE WHEN ABS( COUNT(" + sqlArg + ") - 1 ) > 1.0e-37 THEN COUNT(" + sqlArg + ") - 1 ELSE NULL END" +
				" )" +
				" / CASE WHEN ABS( AVG(" + sqlArg + ") ) > 1.0e-37 THEN AVG(" + sqlArg + ") ELSE NULL END" +
				" )",
			nil
	}
	return "", errors.New("unknown non-aggregation function: " + name + " : " + src)
}

// Translate function argument into sql fragment and push nested OM_ functions to next aggregation level:
//
//	argument: acc0 - 0.5 * OM_AVG(acc0)
//	return:   acc0 - 0.5 * T2.ex2
//	push to the next level: OM_AVG(acc0)
func (lps *levelParseState) translateArg(arg string) (string, error) {

	// parse until source expression not completed
	// push all top level aggregation functions to the next level and substitute with joined column name:
	//   curent level column:    T2.ex2
	//   push to the next level:
	//     column name:          T2.ex2
	//     expression:           OM_AVG(acc0)

	expr := arg

	for {
		// find most left (top level) aggregation function
		fncName, namePos, _, nAfter, err := findFirstFnc(expr, aggrFncLst)
		if err != nil {
			return "", err
		}
		if fncName == "" { // all done: no any functions found
			break
		}

		// push nested function to the next level: OM_AVG(acc0)
		// and replace current level with column name i.e.: T2.ex2
		if nAfter >= len(expr) {
			colName := lps.pushToNextLevel(expr[namePos:])
			expr = expr[:namePos] + lps.nextInnerAlias + "." + colName
		} else {
			colName := lps.pushToNextLevel(expr[namePos:nAfter])
			expr = expr[:namePos] + lps.nextInnerAlias + "." + colName + expr[nAfter:]
		}
	}
	// done with functions:
	//   argument: acc0 - 0.5 * OM_AVG(acc0)
	//   return:   acc0 - 0.5 * T2.ex2

	return expr, nil
}
