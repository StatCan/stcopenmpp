// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// CellParam is value of input parameter.
type CellParam struct {
	cellIdValue     // dimensions as enum id's and value
	SubId       int // parameter subvalue id
}

// CellCodeParam is value of input parameter.
// Dimension(s) items are enum codes, not enum ids.
type CellCodeParam struct {
	cellCodeValue     // dimensions as enum codes and value
	SubId         int // parameter subvalue id
}

// CellParamConverter is a converter for input parameter to implement CsvConverter interface.
type CellParamConverter struct {
	ModelDef  *ModelMeta // model metadata
	Name      string     // parameter name
	IsIdCsv   bool       // if true then use enum id's else use enum codes
	DoubleFmt string     // if not empty then format string is used to sprintf if value type is float, double, long double
	theParam  *ParamMeta // if not nil then parameter found
}

// Converter for input parameter to implement CsvLocaleConverter interface.
type CellParamLocaleConverter struct {
	CellParamConverter
	Lang    string            // language code, expected to compatible with BCP 47 language tag
	DimsTxt []ParamDimsTxtRow // parameter dimension text rows: parameter_dims_txt join to model_parameter_dic
	EnumTxt []TypeEnumTxtRow  // type enum text rows: type_enum_txt join to model_type_dic
}

// return true if csv converter is using enum id's for dimensions
func (cellCvt *CellParamConverter) IsUseEnumId() bool { return cellCvt.IsIdCsv }

// CsvFileName return file name of csv file to store parameter rows
func (cellCvt *CellParamConverter) CsvFileName() (string, error) {

	// find parameter by name
	_, err := cellCvt.paramByName()
	if err != nil {
		return "", err
	}

	// make csv file name
	if cellCvt.IsIdCsv {
		return cellCvt.Name + ".id.csv", nil
	}
	return cellCvt.Name + ".csv", nil
}

// Return first line for csv file: column names. For example: sub_id,dim0,dim1,param_value.
func (cellCvt *CellParamConverter) CsvHeader() ([]string, error) {

	// find parameter by name
	param, err := cellCvt.paramByName()
	if err != nil {
		return []string{}, err
	}

	// make first line columns
	h := make([]string, param.Rank+2)

	h[0] = "sub_id"
	for k := range param.Dim {
		h[k+1] = param.Dim[k].Name
	}
	h[param.Rank+1] = "param_value"

	return h, nil
}

// Return first line for csv file: column names. For example: sub_id,Age,Sex,param_value.
func (cellCvt *CellParamLocaleConverter) CsvHeader() ([]string, error) {

	// default column headers
	h, err := cellCvt.CellParamConverter.CsvHeader()
	if err != nil {
		return []string{}, err
	}

	// replace dimension name with description, where it exists
	if cellCvt.Lang != "" {

		dm := map[int]string{} // map id to dimension description

		// find parameter by name
		param, err := cellCvt.paramByName()
		if err != nil {
			return []string{}, err
		}
		for j := range cellCvt.DimsTxt {
			if cellCvt.DimsTxt[j].ModelId == param.ModelId && cellCvt.DimsTxt[j].ParamId == param.ParamId && cellCvt.DimsTxt[j].LangCode == cellCvt.Lang {
				dm[cellCvt.DimsTxt[j].DimId] = cellCvt.DimsTxt[j].Descr
			}
		}
		for k := range param.Dim {
			if d, ok := dm[param.Dim[k].DimId]; ok {
				h[k+1] = d
			}
		}
	}
	return h, nil
}

// Return converter to copy primary key: (sub id, dimension ids) into key []int.
//
// Converter will return error if len(key) not equal to row key size.
func (cellCvt *CellParamConverter) KeyIds(name string) (func(interface{}, []int) error, error) {

	cvt := func(src interface{}, key []int) error {

		cell, ok := src.(CellParam)
		if !ok {
			return errors.New("invalid type, expected: CellParam (internal error): " + name)
		}

		n := len(cell.DimIds)
		if len(key) != n+1 {
			return errors.New("invalid size of key buffer, expected: " + strconv.Itoa(n+1) + ": " + name)
		}

		key[0] = cell.SubId

		for k, e := range cell.DimIds {
			key[k+1] = e
		}
		return nil
	}

	return cvt, nil
}

// Return converter from parameter cell (sub id, dimensions, value) to csv id's row []string.
//
// Converter return isNotEmpty flag, it is always true if there were no error during conversion.
// Converter simply does Sprint() for each sub-value id, dimension item id and value.
// Converter will return error if len(row) not equal to number of fields in csv record.
func (cellCvt *CellParamConverter) ToCsvIdRow() (func(interface{}, []string) (bool, error), error) {

	// find parameter by name
	param, err := cellCvt.paramByName()
	if err != nil {
		return nil, err
	}

	// for float model types use format if specified
	isUseFmt := param.typeOf.IsFloat() && cellCvt.DoubleFmt != ""

	cvt := func(src interface{}, row []string) (bool, error) {

		cell, ok := src.(CellParam)
		if !ok {
			return false, errors.New("invalid type, expected: CellParam (internal error): " + cellCvt.Name)
		}

		n := len(cell.DimIds)
		if len(row) != n+2 {
			return false, errors.New("invalid size of csv row buffer, expected: " + strconv.Itoa(n+2) + ": " + cellCvt.Name)
		}

		row[0] = fmt.Sprint(cell.SubId)

		for k, e := range cell.DimIds {
			row[k+1] = fmt.Sprint(e)
		}

		// use "null" string for db NULL values and format for model float types
		if cell.IsNull {
			row[n+1] = "null"
		} else {
			if isUseFmt {
				row[n+1] = fmt.Sprintf(cellCvt.DoubleFmt, cell.Value)
			} else {
				row[n+1] = fmt.Sprint(cell.Value)
			}
		}
		return true, nil
	}
	return cvt, nil
}

// Return converter from parameter cell (sub id, dimensions, value) to csv row []string.
//
// Converter return isNotEmpty flag, it is always true if there were no error during conversion.
// Converter will return error if len(row) not equal to number of fields in csv record.
// If dimension type is enum based then csv row is enum code.
// If parameter type is enum based then csv row value is enum code.
func (cellCvt *CellParamConverter) ToCsvRow() (func(interface{}, []string) (bool, error), error) {

	// find parameter by name
	param, err := cellCvt.paramByName()
	if err != nil {
		return nil, err
	}

	// for each dimension create converter from item id to code
	fd := make([]func(itemId int) (string, error), param.Rank)

	for k := 0; k < param.Rank; k++ {
		f, err := param.Dim[k].typeOf.itemIdToCode(cellCvt.Name+"."+param.Dim[k].Name, false)
		if err != nil {
			return nil, err
		}
		fd[k] = f
	}

	// if parameter value type is float then use format, if not empty
	isUseFmt := param.typeOf.IsFloat() && cellCvt.DoubleFmt != ""

	// if parameter value type is enum-based then convert from enum id to code
	isUseEnum := !param.typeOf.IsBuiltIn()
	var fv func(itemId int) (string, error)

	if isUseEnum {
		f, err := param.typeOf.itemIdToCode(cellCvt.Name, false)
		if err != nil {
			return nil, err
		}
		fv = f
	}

	cvt := func(src interface{}, row []string) (bool, error) {

		cell, ok := src.(CellParam)
		if !ok {
			return false, errors.New("invalid type, expected: parameter cell (internal error): " + cellCvt.Name)
		}

		n := len(cell.DimIds)
		if len(row) != n+2 {
			return false, errors.New("invalid size of csv row buffer, expected: " + strconv.Itoa(n+2) + ": " + cellCvt.Name)
		}

		row[0] = fmt.Sprint(cell.SubId)

		// convert dimension item id to code
		for k, e := range cell.DimIds {
			v, err := fd[k](e)
			if err != nil {
				return false, err
			}
			row[k+1] = v
		}

		// convert cell value:
		// if float then use format, if enum then find code by id, default: Sprint(value)
		// use "null" string for db NULL values and format for model float types
		switch {
		case cell.IsNull:
			row[n+1] = "null"

		case isUseFmt:
			row[n+1] = fmt.Sprintf(cellCvt.DoubleFmt, cell.Value)

		case isUseEnum:
			// depending on sql + driver it can be different type
			iv, ok := helper.ToIntValue(cell.Value)
			if !ok {
				return false, errors.New("invalid parameter value type, expected: integer enum: " + cellCvt.Name)
			}

			v, err := fv(int(iv))
			if err != nil {
				return false, err
			}
			row[n+1] = v

		default:
			row[n+1] = fmt.Sprint(cell.Value)
		}

		return true, nil
	}

	return cvt, nil
}

// Return converter from parameter cell (sub id, dimensions, value) to language-specific csv []string row of enum labels and value.
//
// Converter return isNotEmpty flag, it is always true if there were no error during conversion.
// Converter will return error if len(row) not equal to number of fields in csv record.
// If dimension type is enum based then csv row is enum label.
// If parameter type is enum based then csv row value is enum label.
// Value and dimesions of built-in types converted to locale-specific strings, e.g.: 1234.56 => 1 234,56
func (cellCvt *CellParamLocaleConverter) ToCsvRow() (func(interface{}, []string) (bool, error), error) {

	// find parameter by name
	param, err := cellCvt.paramByName()
	if err != nil {
		return nil, err
	}

	// for each dimension create converter from item id to label
	fd := make([]func(itemId int) (string, error), param.Rank)

	for k := 0; k < param.Rank; k++ {
		f, err := param.Dim[k].typeOf.itemIdToLabel(cellCvt.Lang, cellCvt.EnumTxt, nil, cellCvt.Name+"."+param.Dim[k].Name, false)
		if err != nil {
			return nil, err
		}
		fd[k] = f
	}

	// if parameter value type is float then use format, if not empty
	isUseFmt := param.typeOf.IsFloat() && cellCvt.DoubleFmt != ""

	// if parameter value type is enum-based then convert from enum id to label
	isUseEnum := !param.typeOf.IsBuiltIn()
	isUseBool := param.typeOf.IsBool()

	var fv func(itemId int) (string, error)

	if isUseEnum || isUseBool {
		f, err := param.typeOf.itemIdToLabel(cellCvt.Lang, cellCvt.EnumTxt, nil, cellCvt.Name, false)
		if err != nil {
			return nil, err
		}
		fv = f
	}

	prt := message.NewPrinter(language.Make(cellCvt.Lang)) // printer to format built-in types

	cvt := func(src interface{}, row []string) (bool, error) {

		cell, ok := src.(CellParam)
		if !ok {
			return false, errors.New("invalid type, expected: parameter cell (internal error): " + cellCvt.Name)
		}

		n := len(cell.DimIds)
		if len(row) != n+2 {
			return false, errors.New("invalid size of csv row buffer, expected: " + strconv.Itoa(n+2) + ": " + cellCvt.Name)
		}

		row[0] = prt.Sprint(cell.SubId) // convert sub-value id to local-specific string

		// convert dimension item id to code
		for k, e := range cell.DimIds {
			v, err := fd[k](e)
			if err != nil {
				return false, err
			}
			row[k+1] = v
		}

		// convert cell value:
		// if float then use format, if enum then find code by id, default: Sprint(value)
		// use "null" string for db NULL values and format for model float types
		switch {
		case cell.IsNull:
			row[n+1] = "null"

		case isUseFmt:
			row[n+1] = prt.Sprintf(cellCvt.DoubleFmt, cell.Value)

		case isUseEnum:
			// depending on sql + driver it can be different type
			iv, ok := helper.ToIntValue(cell.Value)
			if !ok {
				return false, errors.New("invalid parameter value type, expected: integer enum: " + cellCvt.Name)
			}

			v, err := fv(int(iv))
			if err != nil {
				return false, err
			}
			row[n+1] = v

		case isUseBool:
			isVal, ok := cell.Value.(bool)
			if !ok {
				return false, errors.New("invalid parameter value type, expected: boolean value: " + cellCvt.Name)
			}
			var iv int
			if isVal {
				iv = 1
			}

			v, err := fv(iv)
			if err != nil {
				return false, err
			}
			row[n+1] = v

		default:
			row[n+1] = prt.Sprint(cell.Value)
		}

		return true, nil
	}

	return cvt, nil
}

// Return closure to convert csv row []string to parameter cell (sub id, dimensions, value).
//
// It does return error if len(row) not equal to number of fields in cell db-record.
// If dimension type is enum based then csv row is enum code and it is converted into cell.DimIds (into dimension type type enum ids).
// If parameter type is enum based then csv row value is enum code and it is converted into value enum id.
func (cellCvt *CellParamConverter) ToCell() (func(row []string) (interface{}, error), error) {

	// find parameter by name
	param, err := cellCvt.paramByName()
	if err != nil {
		return nil, err
	}

	// for each dimension create converter from item code to id
	fd := make([]func(src string) (int, error), param.Rank)

	for k := 0; k < param.Rank; k++ {
		f, err := param.Dim[k].typeOf.itemCodeToId(cellCvt.Name+"."+param.Dim[k].Name, false)
		if err != nil {
			return nil, err
		}
		fd[k] = f
	}

	// cell value converter: float, bool, string or integer by default
	var fc func(src string) (interface{}, error)
	var fe func(src string) (int, error)
	var ff func(src string) (bool, float64, error)
	isFloat := param.typeOf.IsFloat()
	isEnum := !param.typeOf.IsBuiltIn()
	isNullable := param.IsExtendable // only extended parameter value can be NULL

	switch {
	case isEnum:
		f, err := param.typeOf.itemCodeToId(cellCvt.Name, false)
		if err != nil {
			return nil, err
		}
		fe = f
	case isFloat:
		ff = func(src string) (bool, float64, error) {

			if src == "" || src == "null" {
				if isNullable {
					return true, 0.0, nil
				}
				// else parameter is not nullable
				return true, 0.0, errors.New("invalid parameter value, it cannot be NULL: " + cellCvt.Name)
			}
			vf, e := strconv.ParseFloat(src, 64)
			if e != nil {
				return false, 0.0, e
			}
			return false, vf, nil
		}
	case param.typeOf.IsBool():
		fc = func(src string) (interface{}, error) { return strconv.ParseBool(src) }
	case param.typeOf.IsString():
		fc = func(src string) (interface{}, error) { return src, nil }
	case param.typeOf.IsInt():
		fc = func(src string) (interface{}, error) { return strconv.Atoi(src) }
	default:
		return nil, errors.New("invalid (not supported) parameter type: " + cellCvt.Name)
	}

	// do conversion
	cvt := func(row []string) (interface{}, error) {

		// make conversion buffer and check input csv row size
		cell := CellParam{cellIdValue: cellIdValue{DimIds: make([]int, param.Rank)}}

		n := len(cell.DimIds)
		if len(row) != n+2 {
			return nil, errors.New("invalid size of csv row, expected: " + strconv.Itoa(n+2) + ": " + cellCvt.Name)
		}

		// subvalue number
		nSub, err := strconv.Atoi(row[0])
		if err != nil {
			return nil, err
		}
		/* validation done at writing
		if subCount < 1 || subCount == 1 && nSub != defaultSubId {
			return nil, errors.New("invalid sub-value id: " + strconv.Itoa(nSub) + " parameter: " + name)
		}
		*/
		cell.SubId = nSub

		// convert dimensions: enum code to enum id or integer value for simple type dimension
		for k := range cell.DimIds {
			i, err := fd[k](row[k+1])
			if err != nil {
				return nil, err
			}
			cell.DimIds[k] = i
		}

		// value conversion, float value can be NULL
		var v interface{}
		var isNull bool
		switch {
		case isEnum:
			isNull = false
			v, err = fe(row[n+1])
		case isFloat:
			isNull, v, err = ff(row[n+1])
		default:
			isNull = false
			v, err = fc(row[n+1])
		}
		if err != nil {
			return nil, err
		}
		cell.IsNull = isNull
		cell.Value = v

		return cell, nil
	}

	return cvt, nil
}

// Return converter from parameter cell of ids: (sub id, dimensions, value)
// to cell of codes: (sub id, dimensions as enum code, value)
//
// If dimension type is enum based then dimensions enum ids can be converted to enum code.
// If dimension type is simple (bool or int) then dimension value converted to string.
// If parameter type is enum based then cell value enum id converted to enum code.
func (cellCvt *CellParamConverter) IdToCodeCell(modelDef *ModelMeta, name string) (func(interface{}) (interface{}, error), error) {

	// find parameter by name
	param, err := cellCvt.paramByName()
	if err != nil {
		return nil, err
	}

	// for each dimension create converter from item id to code
	fd := make([]func(itemId int) (string, error), param.Rank)

	for k := 0; k < param.Rank; k++ {
		f, err := param.Dim[k].typeOf.itemIdToCode(name+"."+param.Dim[k].Name, false)
		if err != nil {
			return nil, err
		}
		fd[k] = f
	}

	// if parameter value type is enum-based then convert from enum id to code
	isUseEnum := !param.typeOf.IsBuiltIn()
	var fv func(itemId int) (string, error)

	if isUseEnum {
		f, err := param.typeOf.itemIdToCode(name, false)
		if err != nil {
			return nil, err
		}
		fv = f
	}

	// create cell converter
	cvt := func(src interface{}) (interface{}, error) {

		srcCell, ok := src.(CellParam)
		if !ok {
			return nil, errors.New("invalid type, expected: parameter cell (internal error): " + name)
		}
		if len(srcCell.DimIds) != param.Rank {
			return nil, errors.New("invalid cell rank: " + strconv.Itoa(len(srcCell.DimIds)) + ", expected: " + strconv.Itoa(param.Rank) + ": " + name)
		}

		dstCell := CellCodeParam{
			cellCodeValue: cellCodeValue{
				Dims:   make([]string, param.Rank),
				IsNull: srcCell.IsNull,
			},
			SubId: srcCell.SubId,
		}

		// convert dimension item id to code
		for k := range srcCell.DimIds {
			v, err := fd[k](srcCell.DimIds[k])
			if err != nil {
				return nil, err
			}
			dstCell.Dims[k] = v
		}

		// convert cell value:
		// if not enum then copy value else find code by id
		if !isUseEnum {
			dstCell.Value = srcCell.Value
		} else {

			// depending on sql + driver it can be different type
			iv, ok := helper.ToIntValue(srcCell.Value)
			if !ok {
				return nil, errors.New("invalid parameter value type, expected: integer enum: " + name)
			}

			v, err := fv(int(iv)) // find value code by id
			if err != nil {
				return nil, err
			}
			dstCell.Value = v
		}

		return dstCell, nil // converted OK
	}

	return cvt, nil
}

// Return converter from parameter cell of codes: (sub id, dimensions as enum code, value)
// to cell of ids: (sub id, dimensions, value)
//
// If dimension type is enum based then dimensions enum codes converted to enum ids.
// If dimension type is simple (bool or int) then dimension code converted from string to dimension type.
// If parameter type is enum based then cell value enum code converted to enum id.
func (cellCvt *CellParamConverter) CodeToIdCell(modelDef *ModelMeta, name string) (func(interface{}) (interface{}, error), error) {

	// find parameter by name
	param, err := cellCvt.paramByName()
	if err != nil {
		return nil, err
	}

	// for each dimension create converter from item code to id
	fd := make([]func(itemCode string) (int, error), param.Rank)

	for k := 0; k < param.Rank; k++ {
		f, err := param.Dim[k].typeOf.itemCodeToId(name+"."+param.Dim[k].Name, false)
		if err != nil {
			return nil, err
		}
		fd[k] = f
	}

	// if parameter value type is enum-based then convert from enum code to id
	isUseEnum := !param.typeOf.IsBuiltIn()
	var fv func(itemCode string) (int, error)

	if isUseEnum {
		f, err := param.typeOf.itemCodeToId(name, false)
		if err != nil {
			return nil, err
		}
		fv = f
	}

	// create cell converter
	cvt := func(src interface{}) (interface{}, error) {

		srcCell, ok := src.(CellCodeParam)
		if !ok {
			return nil, errors.New("invalid type, expected: parameter code cell (internal error): " + name)
		}
		if len(srcCell.Dims) != param.Rank {
			return nil, errors.New("invalid cell rank: " + strconv.Itoa(len(srcCell.Dims)) + ", expected: " + strconv.Itoa(param.Rank) + ": " + name)
		}

		dstCell := CellParam{
			cellIdValue: cellIdValue{
				DimIds: make([]int, param.Rank),
				IsNull: srcCell.IsNull,
			},
			SubId: srcCell.SubId,
		}

		// convert dimension item code to id
		for k := range srcCell.Dims {
			v, err := fd[k](srcCell.Dims[k])
			if err != nil {
				return nil, err
			}
			dstCell.DimIds[k] = v
		}

		// convert cell value:
		// if not enum then copy value else find id by code
		if !isUseEnum {
			dstCell.Value = srcCell.Value
		} else {
			sv, ok := srcCell.Value.(string)
			if !ok {
				return nil, errors.New("invalid parameter value type, expected: string enum code: " + name)
			}
			v, err := fv(sv) // find value id by code
			if err != nil {
				return nil, err
			}
			dstCell.Value = v
		}

		return dstCell, nil // converted OK
	}

	return cvt, nil
}

// return parameter metadata by parameter name
func (cellCvt *CellParamConverter) paramByName() (*ParamMeta, error) {

	if cellCvt.theParam != nil {
		return cellCvt.theParam, nil // parameter already found
	}

	// validate parameters
	if cellCvt.ModelDef == nil {
		return nil, errors.New("invalid (empty) model metadata, look like model not found")
	}
	if cellCvt.Name == "" {
		return nil, errors.New("invalid (empty) parameter name")
	}

	// find parameter by name
	idx, ok := cellCvt.ModelDef.ParamByName(cellCvt.Name)
	if !ok {
		return nil, errors.New("parameter not found: " + cellCvt.Name)
	}
	cellCvt.theParam = &cellCvt.ModelDef.Param[idx]

	return cellCvt.theParam, nil
}
