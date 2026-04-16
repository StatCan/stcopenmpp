// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"errors"
	"fmt"
	"strconv"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// CellExpr is value of output table expression.
type CellExpr struct {
	cellIdValue     // dimensions as enum id's and value
	ExprId      int // output table expression id
}

// CellCodeExpr is value of output table expression.
// Dimension(s) items are enum codes, not enum ids.
type CellCodeExpr struct {
	cellCodeValue     // dimensions as enum codes and value
	ExprId        int // output table expression id
}

// CellTableConverter is a parent for for output table converters.
type CellTableConverter struct {
	ModelDef    *ModelMeta // model metadata
	Name        string     // output table name
	theTable    *TableMeta // if not nil then output table already found
	IsIdCsv     bool       // if true then use enum id's else use enum codes
	DoubleFmt   string     // if not empty then format string is used to sprintf if value type is float, double, long double
	IsNoZeroCsv bool       // if true then do not write zero values into csv output
	IsNoNullCsv bool       // if true then do not write NULL values into csv output
}

// CellExprConverter is a converter for output table expression to implement CsvConverter interface.
type CellExprConverter struct {
	CellTableConverter // model metadata and output table name
}

// Converter for output table expression to implement CsvLocaleConverter interface.
type CellExprLocaleConverter struct {
	CellExprConverter
	Lang    string            // language code, expected to compatible with BCP 47 language tag
	MsgDef  []LangMsg         // translated strings in multiple languages
	DimsTxt []TableDimsTxtRow // output table dimension text rows: table_dims_txt join to model_table_dic
	EnumTxt []TypeEnumTxtRow  // type enum text rows: type_enum_txt join to model_type_dic
	ExprTxt []TableExprTxtRow // output table expression text rows: table_expr_txt join to model_table_dic
}

// return true if csv converter is using enum id's for dimensions
func (cellCvt *CellExprConverter) IsUseEnumId() bool { return cellCvt.IsIdCsv }

// Return file name of csv file to store output table expression rows
func (cellCvt *CellExprConverter) CsvFileName() (string, error) {

	// find output table by name
	_, err := cellCvt.tableByName()
	if err != nil {
		return "", err
	}

	// make csv file name
	if cellCvt.IsIdCsv {
		return cellCvt.Name + ".id.csv", nil
	}
	return cellCvt.Name + ".csv", nil
}

// Return first line for csv file: column names.
// For example: expr_name,dim0,dim1,expr_value
// or if IsIdCsv is true: expr_id,dim0,dim1,expr_value
func (cellCvt *CellExprConverter) CsvHeader() ([]string, error) {

	// find output table by name
	table, err := cellCvt.tableByName()
	if err != nil {
		return []string{}, err
	}

	// make first line columns
	h := make([]string, table.Rank+2)

	if cellCvt.IsIdCsv {
		h[0] = "expr_id"
	} else {
		h[0] = "expr_name"
	}
	for k := range table.Dim {
		h[k+1] = table.Dim[k].Name
	}
	h[table.Rank+1] = "expr_value"

	return h, nil
}

// Return first line for csv file: column names.
// For example: expr_name,Age,Sex,expr_value
func (cellCvt *CellExprLocaleConverter) CsvHeader() ([]string, error) {

	// default column headers
	h, err := cellCvt.CellExprConverter.CsvHeader()
	if err != nil {
		return []string{}, err
	}

	// replace dimension name with description, where it exists
	if cellCvt.Lang != "" {

		dm := map[int]string{} // map id to dimension description

		table, err := cellCvt.tableByName() // find output table by name
		if err != nil {
			return []string{}, err
		}
		for j := range cellCvt.DimsTxt {
			if cellCvt.DimsTxt[j].ModelId == table.ModelId && cellCvt.DimsTxt[j].TableId == table.TableId && cellCvt.DimsTxt[j].LangCode == cellCvt.Lang {
				dm[cellCvt.DimsTxt[j].DimId] = cellCvt.DimsTxt[j].Descr
			}
		}
		for k := range table.Dim {
			if d, ok := dm[table.Dim[k].DimId]; ok {
				h[k+1] = d
			}
		}
	}
	return h, nil
}

// Return converter to copy primary key: (expr_id, dimension ids) into key []int.
//
// Converter will return error if len(key) not equal to row key size.
func (cellCvt *CellExprConverter) KeyIds(name string) (func(interface{}, []int) error, error) {

	cvt := func(src interface{}, key []int) error {

		cell, ok := src.(CellExpr)
		if !ok {
			return errors.New("invalid type, expected: CellExpr (internal error): " + name)
		}

		n := len(cell.DimIds)
		if len(key) != n+1 {
			return errors.New("invalid size of key buffer, expected: " + strconv.Itoa(n+1) + ": " + name)
		}

		key[0] = cell.ExprId

		for k, e := range cell.DimIds {
			key[k+1] = e
		}
		return nil
	}

	return cvt, nil
}

// Return converter from output table cell (expr_id, dimensions, value) to csv id's row []string.
//
// Converter return isNotEmpty flag: false if IsNoZero or IsNoNull is set and cell value is empty or zero.
// Converter simply does Sprint() for each dimension item id, expression id and value.
// Converter will return error if len(row) not equal to number of fields in csv record.
func (cellCvt *CellExprConverter) ToCsvIdRow() (func(interface{}, []string) (bool, error), error) {

	// find output table by name
	_, err := cellCvt.tableByName()
	if err != nil {
		return nil, err
	}

	// return converter from id based cell to csv string array
	cvt := func(src interface{}, row []string) (bool, error) {

		cell, ok := src.(CellExpr)
		if !ok {
			return false, errors.New("invalid type, expected: CellExpr (internal error): " + cellCvt.Name)
		}

		n := len(cell.DimIds)
		if len(row) != n+2 {
			return false, errors.New("invalid size of csv row buffer, expected: " + strconv.Itoa(n+2) + ": " + cellCvt.Name)
		}

		row[0] = fmt.Sprint(cell.ExprId)

		for k, e := range cell.DimIds {
			row[k+1] = fmt.Sprint(e)
		}

		// use "null" string for db NULL values and format for model float types
		isNotEmpty := true

		if cell.IsNull {
			row[n+1] = "null"
			isNotEmpty = !cellCvt.IsNoNullCsv
		} else {

			if cellCvt.IsNoZeroCsv {
				fv, ok := cell.Value.(float64)
				isNotEmpty = ok && fv != 0
			}

			if cellCvt.DoubleFmt != "" {
				row[n+1] = fmt.Sprintf(cellCvt.DoubleFmt, cell.Value)
			} else {
				row[n+1] = fmt.Sprint(cell.Value)
			}
		}
		return isNotEmpty, nil
	}

	return cvt, nil
}

// Return converter from output table cell (expr_id, dimensions, value)
// to csv row []string (expr_name, dimensions, value).
//
// Converter return isNotEmpty flag: false if IsNoZero or IsNoNull is set and cell value is empty or zero.
// If dimension type is enum based then csv row is enum code.
// Converter return error if len(row) not equal to number of fields in csv record.
func (cellCvt *CellExprConverter) ToCsvRow() (func(interface{}, []string) (bool, error), error) {

	// find output table by name
	table, err := cellCvt.tableByName()
	if err != nil {
		return nil, err
	}

	// for each dimension create converter from item id to code
	fd := make([]func(itemId int) (string, error), len(table.Dim))

	for k := range table.Dim {
		f, err := table.Dim[k].typeOf.itemIdToCode(cellCvt.Name+"."+table.Dim[k].Name, table.Dim[k].IsTotal)
		if err != nil {
			return nil, err
		}
		fd[k] = f
	}

	cvt := func(src interface{}, row []string) (bool, error) {

		cell, ok := src.(CellExpr)
		if !ok {
			return false, errors.New("invalid type, expected: output table expression cell (internal error): " + cellCvt.Name)
		}

		n := len(cell.DimIds)
		if len(row) != n+2 {
			return false, errors.New("invalid size of csv row buffer, expected: " + strconv.Itoa(n+2) + ": " + cellCvt.Name)
		}

		row[0] = table.Expr[cell.ExprId].Name

		// convert dimension item id to code
		for k, e := range cell.DimIds {
			v, err := fd[k](e)
			if err != nil {
				return false, err
			}
			row[k+1] = v
		}

		// use "null" string for db NULL values and format for model float types
		isNotEmpty := true

		if cell.IsNull {
			row[n+1] = "null"
			isNotEmpty = !cellCvt.IsNoNullCsv
		} else {

			if cellCvt.IsNoZeroCsv {
				fv, ok := cell.Value.(float64)
				isNotEmpty = ok && fv != 0
			}

			if cellCvt.DoubleFmt != "" {
				row[n+1] = fmt.Sprintf(cellCvt.DoubleFmt, cell.Value)
			} else {
				row[n+1] = fmt.Sprint(cell.Value)
			}
		}
		return isNotEmpty, nil
	}

	return cvt, nil
}

// Return converter from output table cell (expr_id, dimensions, value)
// to language-specific csv []string row of dimension enum labels and value.
//
// Converter return isNotEmpty flag: false if IsNoZero or IsNoNull is set and cell value is empty or zero.
// If dimension type is enum based then csv row is enum label.
// Value and dimesions of built-in types converted to locale-specific strings, e.g.: 1234.56 => 1 234,56
// Converter return error if len(row) not equal to number of fields in csv record.
func (cellCvt *CellExprLocaleConverter) ToCsvRow() (func(interface{}, []string) (bool, error), error) {

	// find output table by name
	table, err := cellCvt.tableByName()
	if err != nil {
		return nil, err
	}

	// for each dimension create converter from item id to label
	fd := make([]func(itemId int) (string, error), len(table.Dim))

	for k := range table.Dim {
		f, err := table.Dim[k].typeOf.itemIdToLabel(cellCvt.Lang, cellCvt.EnumTxt, cellCvt.MsgDef, cellCvt.Name+"."+table.Dim[k].Name, table.Dim[k].IsTotal)
		if err != nil {
			return nil, err
		}
		fd[k] = f
	}

	idToLabel, err := cellCvt.exprIdToLabel() // converter from expression id to language-specific label
	if err != nil {
		return nil, err
	}

	// format value locale-specific strings, e.g.: 1234.56 => 1 234,56
	prt := message.NewPrinter(language.Make(cellCvt.Lang))

	cvt := func(src interface{}, row []string) (bool, error) {

		cell, ok := src.(CellExpr)
		if !ok {
			return false, errors.New("invalid type, expected: output table expression cell (internal error): " + cellCvt.Name)
		}

		n := len(cell.DimIds)
		if len(row) != n+2 {
			return false, errors.New("invalid size of csv row buffer, expected: " + strconv.Itoa(n+2) + ": " + cellCvt.Name)
		}

		row[0], err = idToLabel(cell.ExprId)
		if err != nil {
			return false, err
		}

		// convert dimension item id to label
		for k, e := range cell.DimIds {
			v, err := fd[k](e)
			if err != nil {
				return false, err
			}
			row[k+1] = v
		}

		// use "null" string for db NULL values and format for model float types
		isNotEmpty := true

		if cell.IsNull {
			row[n+1] = "null"
			isNotEmpty = !cellCvt.IsNoNullCsv
		} else {

			if cellCvt.IsNoZeroCsv {
				fv, ok := cell.Value.(float64)
				isNotEmpty = ok && fv != 0
			}

			if cellCvt.DoubleFmt != "" {
				row[n+1] = prt.Sprintf(cellCvt.DoubleFmt, cell.Value)
			} else {
				row[n+1] = prt.Sprint(cell.Value)
			}
		}
		return isNotEmpty, nil
	}

	return cvt, nil
}

// Return closure to convert csv row []string to output table expression cell (dimensions and value).
//
// Converter return error if len(row) not equal to number of fields in cell db-record.
// If dimension type is enum based then csv row is enum code and it is converted into cell.DimIds (into dimension type type enum ids).
func (cellCvt *CellExprConverter) ToCell() (func(row []string) (interface{}, error), error) {

	// find output table by name
	table, err := cellCvt.tableByName()
	if err != nil {
		return nil, err
	}

	// for each dimension create converter from item code to id
	fd := make([]func(src string) (int, error), table.Rank)

	for k := 0; k < table.Rank; k++ {
		f, err := table.Dim[k].typeOf.itemCodeToId(cellCvt.Name+"."+table.Dim[k].Name, table.Dim[k].IsTotal)
		if err != nil {
			return nil, err
		}
		fd[k] = f
	}

	// do conversion
	cvt := func(row []string) (interface{}, error) {

		// make conversion buffer and check input csv row size
		cell := CellExpr{cellIdValue: cellIdValue{DimIds: make([]int, table.Rank)}}

		n := len(cell.DimIds)
		if len(row) != n+2 {
			return nil, errors.New("invalid size of csv row, expected: " + strconv.Itoa(n+2) + ": " + cellCvt.Name)
		}

		// expression id by name
		cell.ExprId = -1
		for k := range table.Expr {
			if row[0] == table.Expr[k].Name {
				cell.ExprId = k
				break
			}
		}
		if cell.ExprId < 0 {
			return nil, errors.New("invalid expression name: " + row[0] + " output table: " + cellCvt.Name)
		}

		// convert dimensions: enum code to enum id or integer value for simple type dimension
		for k := range cell.DimIds {
			i, err := fd[k](row[k+1])
			if err != nil {
				return nil, err
			}
			cell.DimIds[k] = i
		}

		// value conversion
		cell.IsNull = row[n+1] == "" || row[n+1] == "null"

		if cell.IsNull {
			cell.Value = 0.0
		} else {
			v, err := strconv.ParseFloat(row[n+1], 64)
			if err != nil {
				return nil, err
			}
			cell.Value = v
		}
		return cell, nil
	}

	return cvt, nil
}

// Return converter from output table cell of ids: (expr_id, dimensions enum ids, value)
// to cell of codes: (expr_id, dimensions as enum codes, value).
//
// If dimension type is enum based then dimensions enum ids can be converted to enum code.
// If dimension type is simple (bool or int) then dimension value converted to string.
func (cellCvt *CellExprConverter) IdToCodeCell(modelDef *ModelMeta, name string) (func(interface{}) (interface{}, error), error) {

	// find output table by name
	table, err := cellCvt.tableByName()
	if err != nil {
		return nil, err
	}

	// for each dimension create converter from item id to code
	fd := make([]func(itemId int) (string, error), table.Rank)

	for k := 0; k < table.Rank; k++ {
		f, err := table.Dim[k].typeOf.itemIdToCode(name+"."+table.Dim[k].Name, table.Dim[k].IsTotal)
		if err != nil {
			return nil, err
		}
		fd[k] = f
	}

	// create cell converter
	cvt := func(src interface{}) (interface{}, error) {

		srcCell, ok := src.(CellExpr)
		if !ok {
			return nil, errors.New("invalid type, expected: output table expression cell (internal error): " + name)
		}
		if len(srcCell.DimIds) != table.Rank {
			return nil, errors.New("invalid cell rank: " + strconv.Itoa(len(srcCell.DimIds)) + ", expected: " + strconv.Itoa(table.Rank) + ": " + name)
		}

		dstCell := CellCodeExpr{
			cellCodeValue: cellCodeValue{
				Dims:   make([]string, table.Rank),
				IsNull: srcCell.IsNull,
				Value:  srcCell.Value,
			},
			ExprId: srcCell.ExprId,
		}

		// convert dimension item id to code
		for k := range srcCell.DimIds {
			v, err := fd[k](srcCell.DimIds[k])
			if err != nil {
				return nil, err
			}
			dstCell.Dims[k] = v
		}

		return dstCell, nil // converted OK
	}

	return cvt, nil
}

// return output table metadata by output table name
func (cellCvt *CellTableConverter) tableByName() (*TableMeta, error) {

	if cellCvt.theTable != nil {
		return cellCvt.theTable, nil // output table already found
	}

	// validate parameters
	if cellCvt.ModelDef == nil {
		return nil, errors.New("invalid (empty) model metadata, look like model not found")
	}
	if cellCvt.Name == "" {
		return nil, errors.New("invalid (empty) output table name")
	}

	// find output table by name
	idx, ok := cellCvt.ModelDef.OutTableByName(cellCvt.Name)
	if !ok {
		return nil, errors.New("output table not found: " + cellCvt.Name)
	}
	cellCvt.theTable = &cellCvt.ModelDef.Table[idx]

	return cellCvt.theTable, nil
}

// Return converter from expression id to language-specific label.
// Converter return expression description by expression id and language.
// If language code or description is empty then return expression name
func (cellCvt *CellExprLocaleConverter) exprIdToLabel() (func(itemId int) (string, error), error) {

	// find output table by name
	table, err := cellCvt.tableByName()
	if err != nil {
		return nil, err
	}
	labelMap := make(map[int]string, len(table.Expr))

	// add expression name into map as default label
	for j := range table.Expr {
		labelMap[table.Expr[j].ExprId] = table.Expr[j].Name
	}

	// replace labels: use description where exists for specified language
	if cellCvt.Lang != "" {
		for j := range cellCvt.ExprTxt {
			if cellCvt.ExprTxt[j].ModelId == table.ModelId && cellCvt.ExprTxt[j].TableId == table.TableId && cellCvt.ExprTxt[j].LangCode == cellCvt.Lang {
				labelMap[cellCvt.ExprTxt[j].ExprId] = cellCvt.ExprTxt[j].Descr
			}
		}
	}

	cvt := func(exprId int) (string, error) {

		if lbl, ok := labelMap[exprId]; ok {
			return lbl, nil
		}
		return "", errors.New("invalid value: " + strconv.Itoa(exprId) + " of: " + cellCvt.Name)
	}

	return cvt, nil
}
