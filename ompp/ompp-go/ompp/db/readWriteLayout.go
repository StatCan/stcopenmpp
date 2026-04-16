// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"errors"
	"sort"
	"strconv"
	"strings"
)

// WriteLayout describes parameters or output tables values for insert or update.
//
// Name is a parameter or output table name to read.
type WriteLayout struct {
	Name string // parameter name or output table name
	ToId int    // run id or set id to write parameter or output table values
}

// WriteParamLayout describes parameter values for insert or update.
//
// Double format string is used for digest calculation if value type if float or double.
type WriteParamLayout struct {
	WriteLayout        // common write layout: parameter name, run or set id
	SubCount    int    // sub-values count
	IsToRun     bool   // if true then write into into model run else into workset
	IsPage      bool   // if true then write only page of data else all parameter values
	DoubleFmt   string // used for float model types digest calculation
}

// WriteTableLayout describes output table values for insert or update.
//
// Double format string is used for digest calculation if value type if float or double.
type WriteTableLayout struct {
	WriteLayout        // common write layout: output table name and run id
	SubCount    int    // sub-values count
	DoubleFmt   string // used for float model types digest calculation
}

// WriteMicroLayout describes source and size of data page to read entity microdata.
//
// Double format string is used for digest calculation if value type if float or double.
type WriteMicroLayout struct {
	WriteLayout        // common write layout: entity name and run id
	DoubleFmt   string // used for float model types digest calculation
}

// ReadLayout describes source and size of data page to read input parameter, output table values or microdata.
//
// Row filters combined by AND and allow to select dimension or attribute items,
// it can be enum codes or enum id's, ex.: dim0 = 'CA' AND dim1 IN (2010, 2011, 2012)
//
// Order by applied to output columns.
// Because dimension or attribute columns always contain enum id's,
// therefore result ordered by id's and not by enum codes.
// Columns list depending on output table or parameter query:
//
// parameter values:
//
//	SELECT sub_id, dim0, dim1, param_value FROM parameterTable ORDER BY...
//
// output table expressions:
//
//	SELECT expr_id, dim0, dim1, expr_value FROM outputTable ORDER BY...
//
// output table accumulators:
//
//	SELECT acc_id, sub_id, dim0, dim1, acc_value FROM outputTable ORDER BY...
//
// all-accumulators view:
//
//	SELECT sub_id, dim0, dim1, acc0_value, acc1_value... FROM outputTable ORDER BY...
//
// entity microdata table:
//
//	SELECT entity_key, attr0, attr1,... FROM microdataTable ORDER BY...
type ReadLayout struct {
	Name           string           // parameter name, output table name or entity microdata name
	FromId         int              // run id or set id to select input parameter, output table values or microdata from
	ReadPageLayout                  // read page first row offset, size and last page flag
	Filter         []FilterColumn   // dimension or attribute or value filters, final WHERE does join all filters by AND
	FilterById     []FilterIdColumn // dimension or attribute filters by enum ids, final WHERE does join filters by AND
	OrderBy        []OrderByColumn  // order by columnns, if empty then dimension id ascending order is used
}

// ReadParamLayout describes source and size of data page to read input parameter values.
//
// It can read parameter values from model run results or from input working set (workset).
// If this is read from workset then it can be read-only or read-write (editable) workset.
type ReadParamLayout struct {
	ReadLayout           // parameter name, run id or set id page size, where filters and order by
	IsFromSet       bool // if true then select from workset else from model run
	IsEditSet       bool // if true then workset must be editable (readonly = false)
	ReadSubIdLayout      // sub-value id filter: select rows with only one sub-value id
}

// ReadTableLayout describes source and size of data page to read output table values.
//
// If ValueName is not empty then only accumulator or output expression
// with that name selected (i.e: "acc1" or "expr4") else all output table accumulators (expressions) selected.
type ReadTableLayout struct {
	ReadLayout             // output table name, run id, page size, where filters and order by
	ValueName       string // if not empty then expression or accumulator name to select
	IsAccum         bool   // if true then select output table accumulator else expression
	IsAllAccum      bool   // if true then select from all accumulators view else from accumulators table
	ReadSubIdLayout        // sub-value id filter: select rows with only one sub-value id
}

// ReadMicroLayout describes source and size of data page to read entity microdata.
//
// Only one entity generation digest expected for each run id + entity name, but there is no such constarint in db schema.
type ReadMicroLayout struct {
	ReadLayout        // entity name, run id, page size, where filters and order by
	GenDigest  string // entity generation digest
}

// ReadSubIdLayout supply sub-value id filter to select rows with only single sub_id from output table or input parameter values.
type ReadSubIdLayout struct {
	IsSubId bool // if true then select only single sub-value id
	SubId   int  // sub-value id to select rows from output table or parameter
}

// ReadPageLayout describes first row offset and size of data page to read input parameter or output table values.
type ReadPageLayout struct {
	Offset     int64 // first row to return from select, zero-based ofsset
	Size       int64 // max row count to select, if <= 0 then all rows
	IsLastPage bool  // output last page flag: return true if it was a last page of rows
	IsFullPage bool  // input last page flag: if true then adjust offset to return full last page
}

// ReadCompareTableLayout to compare output table runs with base run using multiple comparison expressions and/or calculation measures.
//
// Comparison expression(s) must contain [base] and [variant] expression(s), ex.: Expr0[base] - Expr0[variant].
// Calculation measure(s) can include table exprissions, ex.: Expr0 + Expr1
// or aggregation of table accumulators, ex.: OM_SUM(acc0) / OM_COUNT(acc0)
type ReadCompareTableLayout struct {
	ReadCalculteTableLayout          // output table, base run and comparison expressions or calculations
	Runs                    []string // runs to compare: list of digest, stamp or name
}

// ReadCalculteTableLayout describe table read layout and additional measures to calculte.
type ReadCalculteTableLayout struct {
	ReadLayout                         // output table name, run id, page size, where filters and order by
	Calculation []CalculateTableLayout // additional measures to calculate
}

// CalculateLayout describes calculation of output table values.
// It can be comparison calculation for multiple model runs, ex.: Expr0[base] - Expr0[variant].
type CalculateTableLayout struct {
	CalculateLayout      // expression to calculate and layout
	IsAggr          bool // if true then select output table accumulator else expression
}

// CalculateLayout describes calculation expression for parameters, output table values or microdata entity.
// It can be comparison calculation for multiple model runs, ex.: Expr0[base] - Expr0[variant].
type CalculateLayout struct {
	Calculate string // expression to calculate, ex.: Expr0[base] - Expr0[variant]
	CalcId    int    // calculated expression id, calc_id column in csv,     ex.: 0, 12001, 24001
	Name      string // calculated expression name, calc_name column in csv, ex.: Expr0, AVG_Expr0, RATIO_Expro0
}

// ReadCompareMicroLayout to compare microdata runs with base run using multiple comparison aggregations and/or calculation aggregations.
//
// Comparison aggregation must contain [base] and [variant] attribute(s), ex.: OM_AVG(Income[base] - Income[variant]).
// Calculation aggregation is attribute(s) aggregation expression, ex.: OM_MAX(Income) / OM_MIN(Salary).
type ReadCompareMicroLayout struct {
	ReadCalculteMicroLayout          // aggregation measures and group by attributes
	Runs                    []string // runs to compare: list of digest, stamp or name
}

// ReadCalculteMicroLayout describe microdata generation read layout, aggregation measures and group by attributes.
type ReadCalculteMicroLayout struct {
	ReadLayout           // entity name, run id, page size, where filters and order by
	CalculateMicroLayout // microdata aggregations
}

// CalculateMicroLayout describes aggregations of microdata.
//
// It can be comparison aggregations and/or calculation aggregations.
// Comparison aggregation must contain [base] and [variant] attribute(s), ex.: OM_AVG(Income[base] - Income[variant]).
// Calculation aggregation is attribute(s) aggregation expression, ex.: OM_MAX(Income) / OM_MIN(Salary).
type CalculateMicroLayout struct {
	Calculation []CalculateLayout // aggregation measures, ex.: OM_MIN(Salary), OM_AVG(Income[base] - Income[variant])
	GroupBy     []string          // attributes to group by
}

// FilterOp is enum type for filter operators in select where conditions
type FilterOp string

// Select filter operators for dimension enum values or attribute values.
const (
	InAutoOpFilter  FilterOp = "IN_AUTO" // auto convert IN list filter into equal or BETWEEN if possible
	InOpFilter      FilterOp = "IN"      // dimension enum ids in: dim2 IN (11, 22, 33)
	EqOpFilter      FilterOp = "="       // dimension equal: dim1 = 12
	NeOpFilter      FilterOp = "!="      // dimension equal: dim1 <> 12
	GtOpFilter      FilterOp = ">"       // value greater than: attr1 > 12
	GeOpFilter      FilterOp = ">="      // value greater or equal: attr1 >= 12
	LtOpFilter      FilterOp = "<"       // value less than: attr1 < 12
	LeOpFilter      FilterOp = "<="      // value less or equal: attr1 <= 12
	BetweenOpFilter FilterOp = "BETWEEN" // dimension enum ids between: dim3 BETWEEN 44 AND 88
)

// FilterColumn define dimension or attribute column and condition to filter enum codes to build select where
type FilterColumn struct {
	Name   string   // dimension or attribute name
	Op     FilterOp // filter operator: equal, IN, BETWEEN
	Values []string // enum code(s): one, two or many values depending on filter condition
}

// FilterIdColumn define dimension or attribute column and condition to filter enum ids to build select where
type FilterIdColumn struct {
	Name    string   // dimension or attribute name
	Op      FilterOp // filter operator: equal, IN, BETWEEN
	EnumIds []int    // enum id(s): one, two or many ids depending on filter condition
}

// OrderByColumn define column to order by rows selected from parameter or output table.
type OrderByColumn struct {
	IndexOne int  // one-based column index
	IsDesc   bool // if true then descending order
}

// makeOrderBy return ORDER BY clause either from explicitly specified column list
// or by default: 1,...,rank
// or if prefixIdColumns > 0 then order by 1,..., prefixIdColumns,..., prefixIdColumns+rank+1
func makeOrderBy(rank int, orderBy []OrderByColumn, prefixIdColumns int) string {

	if len(orderBy) > 0 { // if order by excplicitly specified

		q := " ORDER BY "
		for k, co := range orderBy {
			if k > 0 {
				q += ", "
			}
			q += strconv.Itoa(co.IndexOne)
			if co.IsDesc {
				q += " DESC"
			}
		}
		return q
	}
	// else
	if rank > 0 || prefixIdColumns > 0 { // default: order by acc_id, sub_id, dimensions

		q := " ORDER BY "
		for k := 1; k <= prefixIdColumns; k++ {
			if k > 1 {
				q += ", "
			}
			q += strconv.Itoa(k)
		}
		for k := 1; k <= rank; k++ {
			if k > 1 || prefixIdColumns > 0 {
				q += ", "
			}
			q += strconv.Itoa(prefixIdColumns + k)
		}
		return q
	}

	return ""
}

// makeWhereFilter convert dimension or attribute enum codes to enum ids and return filter condition, eg: dim1 IN (1, 2, 3, 4)
func makeWhereFilter(
	flt *FilterColumn, alias string, colName string, typeOf *TypeMeta, isTotalEnabled bool, msgName string, msgParent string,
) (string, error) {

	// validate number of enum ids in enum list
	nFlt := len(flt.Values)
	if nFlt <= 0 ||
		nFlt != 1 && (flt.Op == EqOpFilter || flt.Op == NeOpFilter || flt.Op == GtOpFilter || flt.Op == GeOpFilter || flt.Op == LtOpFilter || flt.Op == LeOpFilter) ||
		nFlt != 2 && flt.Op == BetweenOpFilter {
		return "", errors.New("invalid number of arguments to filter " + msgParent + " " + msgName + ": " + strconv.Itoa(nFlt))
	}

	// for boolean or enum-based dimensions or attributes make filter by id
	if typeOf.IsBool() || !typeOf.IsBuiltIn() {

		// convert enum codes to ids
		cvt, err := typeOf.itemCodeToId(msgName, isTotalEnabled)
		if err != nil {
			return "", err
		}
		fltId := FilterIdColumn{
			Name:    flt.Name,
			Op:      flt.Op,
			EnumIds: make([]int, nFlt),
		}
		for k := range flt.Values {
			id, err := cvt(flt.Values[k])
			if err != nil {
				return "", err
			}
			fltId.EnumIds[k] = id
		}

		return makeWhereIdFilter(&fltId, alias, colName, typeOf, msgName, msgParent)
	}

	// else make filter for other types: int, float, strings
	return makeCodeWhereFiler(alias, colName, typeOf.IsString(), flt, msgName, msgParent)
}

// return filter by enum code or value comparison, e.g.: E.attr4 < 1234 or E.dim0 IN ('a', 'b', 'c')
func makeCodeWhereFiler(alias, colName string, isStrType bool, flt *FilterColumn, msgName, msgParent string) (string, error) {

	// use sql-quotes for string type
	var vals []string

	if !isStrType {
		vals = flt.Values
	} else {
		vals = make([]string, len(flt.Values))
		for k := range flt.Values {
			vals[k] = ToQuoted(flt.Values[k])
		}
	}

	// make dimension or attribute filter
	q := ""
	if alias != "" {
		q += alias + "."
	}
	q += colName
	switch flt.Op {
	case EqOpFilter: // AND dim1 = 2
		q += " = " + vals[0]
	case NeOpFilter: // AND dim1 <> 2
		q += " <> " + vals[0]
	case GtOpFilter: // AND attr1 > 2
		q += " > " + vals[0]
	case GeOpFilter: // AND attr1 > 2
		q += " >= " + vals[0]
	case LtOpFilter: // AND attr1 > 2
		q += " < " + vals[0]
	case LeOpFilter: // AND attr1 > 2
		q += " <= " + vals[0]
	case InOpFilter, InAutoOpFilter: // AND dim1 IN (10, 20, 30)
		q += " IN (" + strings.Join(vals, ",") + ")"
	case BetweenOpFilter: // AND dim1 BETWEEN 100 AND 200
		q += " BETWEEN " + vals[0] + " AND " + vals[1]
	default:
		return "", errors.New("invalid filter operation to read " + msgParent + " " + msgName)
	}
	return q, nil

}

// return filter by value of parameter, expression, accumulator or attribute, eg: (E.expr_value < 2 AND E.expr_id = 1)
func makeWhereValueFilter(
	flt *FilterColumn, alias string, colName string, idColName string, idValue int, typeOf *TypeMeta, msgName string, msgParent string,
) (string, error) {

	// if there is no id column then result is identical to regular where filter
	if idColName == "" {
		return makeWhereFilter(flt, alias, colName, typeOf, false, msgName, msgParent)
	}
	// else: it is (expr_value and expr_id) or (acc_value and acc_id)

	// validate number of enum ids in enum list
	nFlt := len(flt.Values)
	if nFlt <= 0 ||
		nFlt != 1 && (flt.Op == EqOpFilter || flt.Op == NeOpFilter || flt.Op == GtOpFilter || flt.Op == GeOpFilter || flt.Op == LtOpFilter || flt.Op == LeOpFilter) ||
		nFlt != 2 && flt.Op == BetweenOpFilter {
		return "", errors.New("invalid number of arguments to filter " + msgParent + " " + msgName + ": " + strconv.Itoa(nFlt))
	}

	// for boolean or enum-based parameters or attributes make filter by id
	q := "("

	if typeOf.IsBool() || !typeOf.IsBuiltIn() {

		// convert enum codes to ids
		cvt, err := typeOf.itemCodeToId(msgName, false)
		if err != nil {
			return "", err
		}
		fltId := FilterIdColumn{
			Name:    flt.Name,
			Op:      flt.Op,
			EnumIds: make([]int, nFlt),
		}
		for k := range flt.Values {
			id, err := cvt(flt.Values[k])
			if err != nil {
				return "", err
			}
			fltId.EnumIds[k] = id
		}

		c, err := makeWhereIdFilter(&fltId, alias, colName, typeOf, msgName, msgParent)
		if err != nil {
			return "", err
		}
		q += c

	} else { // make filter for other types: int, float, strings

		c, err := makeCodeWhereFiler(alias, colName, typeOf.IsString(), flt, msgName, msgParent)
		if err != nil {
			return "", err
		}
		q += c
	}

	// append id column condition: AND E.expr_id = 1)
	q += " AND "
	if alias != "" {
		q += alias + "."
	}
	q += idColName + " = " + strconv.Itoa(idValue) + ")"

	return q, nil
}

// makeWhereIdFilter return dimension or attribute filter condition for enum ids, eg: dim1 IN (1, 2, 3, 4)
// It is also can be equal or BETWEEN fitler.
func makeWhereIdFilter(
	flt *FilterIdColumn, alias string, colName string, typeOf *TypeMeta, msgName string, msgParent string) (string, error) {

	// validate number of enum ids in enum list
	nFlt := len(flt.EnumIds)
	if nFlt <= 0 ||
		nFlt != 1 && (flt.Op == EqOpFilter || flt.Op == NeOpFilter || flt.Op == GtOpFilter || flt.Op == GeOpFilter || flt.Op == LtOpFilter || flt.Op == LeOpFilter) ||
		nFlt != 2 && flt.Op == BetweenOpFilter {
		return "", errors.New("invalid number of arguments to filter " + msgParent + " " + msgName + ": " + strconv.Itoa(nFlt))
	}

	sort.Ints(flt.EnumIds) // sort enum id's for fast search

	emin := flt.EnumIds[0]
	emax := flt.EnumIds[nFlt-1]
	if emin > emax {
		emin, emax = emax, emin
	}
	op := flt.Op
	neVal := flt.EnumIds[0]

	// if filter condition is "auto" and only single enum supplied then use = equal filter
	// else use BETWEEN if all enum values between supplied min and max (no holes)
	// use IN filter by default
	if op == InAutoOpFilter {

		if typeOf.sizeOf <= 0 {
			return "", errors.New("auto filter cannot be applied to " + msgParent + " " + msgName)
		}

		switch {
		case nFlt == 1:
			op = EqOpFilter // single value: use equal

		case nFlt == typeOf.sizeOf-1: // single value excluded: use NE

			n := 0
			if !typeOf.IsRange {

				for k := 0; k < len(typeOf.Enum); k++ {

					j := sort.Search(nFlt, func(i int) bool {
						return flt.EnumIds[i] >= typeOf.Enum[k].EnumId
					})
					if j < 0 || j >= nFlt || flt.EnumIds[j] != typeOf.Enum[k].EnumId {
						n++
						if n > 1 {
							break // this is not NE filter, more than one value not included
						}
						neVal = typeOf.Enum[k].EnumId // found a value for not equal condition
					}
				}
			} else {

				for _, e := range flt.EnumIds {
					if e < typeOf.MinEnumId || typeOf.MaxEnumId < e {
						n++
						if n > 1 {
							break // this is not NE filter, more than one value not included
						}
						neVal = e // found a value for not equal condition
					}
				}
			}
			if n <= 1 {
				op = NeOpFilter // it is a NE filter: only one value excluded from the list of enums
			}

		default: // multiple values: check if BETWEEN possible else use IN

			op = InOpFilter // use IN by default

			for _, e := range flt.EnumIds {
				if e < emin {
					emin = e
				}
				if e > emax {
					emax = e
				}
			}

			// check if all type enums between min and max (no holes)
			if !typeOf.IsRange {
				if len(typeOf.Enum) > 0 {

					isHole := true
					for k := 0; k < len(typeOf.Enum); k++ {

						if typeOf.Enum[k].EnumId < emin {
							continue
						}
						if typeOf.Enum[k].EnumId > emax {
							break
						}
						// between min and max
						isHole = true
						for _, e := range flt.EnumIds {
							if e == typeOf.Enum[k].EnumId {
								isHole = false
								break
							}
						}
						if isHole {
							break // current type enum id is not between min and max filter
						}
					}
					if !isHole {
						op = BetweenOpFilter // all type enum ids is between filter min and max enum ids
					}
				}
			} else { // it is a range dimension
				// all filter id's must be sequential int numbers, no holes
				// filter id's are sorted already
				isHole := false
				ePrev := flt.EnumIds[0]

				for k := 1; k < len(flt.EnumIds); k++ {
					isHole = flt.EnumIds[k] != ePrev+1
					if isHole {
						break // current filter enum id is not previous + 1: hole found
					}
					ePrev = flt.EnumIds[k]
				}
				if !isHole {
					op = BetweenOpFilter // all filter enum id's are sequential integers: it is a bteween filter
				}
			}
		}
	}

	// make dimension or attribute filter
	q := ""
	if alias != "" {
		q += alias + "."
	}
	q += colName
	switch op {
	case EqOpFilter: // AND dim1 = 2
		q += " = " + strconv.Itoa(flt.EnumIds[0])
	case NeOpFilter: // AND dim1 <> 2
		q += " <> " + strconv.Itoa(neVal)
	case GtOpFilter: // AND attr1 > 2
		q += " > " + strconv.Itoa(flt.EnumIds[0])
	case GeOpFilter: // AND attr1 > 2
		q += " >= " + strconv.Itoa(flt.EnumIds[0])
	case LtOpFilter: // AND attr1 > 2
		q += " < " + strconv.Itoa(flt.EnumIds[0])
	case LeOpFilter: // AND attr1 > 2
		q += " <= " + strconv.Itoa(flt.EnumIds[0])
	case InOpFilter: // AND dim1 IN (10, 20, 30)
		q += " IN ("
		for k, e := range flt.EnumIds {
			if k > 0 {
				q += ", "
			}
			q += strconv.Itoa(e)
		}
		q += ")"
	case BetweenOpFilter: // AND dim1 BETWEEN 100 AND 200
		q += " BETWEEN " + strconv.Itoa(emin) + " AND " + strconv.Itoa(emax)
	default:
		return "", errors.New("invalid filter operation to read " + msgParent + " " + msgName)
	}
	return q, nil
}
