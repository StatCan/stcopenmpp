// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"errors"
	"strconv"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// cellIdValue is dimensions item as id and value of input parameter or output table.
type cellIdValue struct {
	DimIds []int       // dimensions enum ids or int values if dimension type simple
	IsNull bool        // if true then value is NULL
	Value  interface{} // value: int64, bool, float64 or string
}

// cellCodeValue is dimensions item as code and value of input parameter or output table.
// Value is enum code if parameter is enum-based.
type cellCodeValue struct {
	Dims   []string    // dimensions as enum code or string converted built-in type
	IsNull bool        // if true then value is NULL
	Value  interface{} // value: int64, bool, float64 or string
}

// calculation maps between run id and label (digest) and calculations id and name
type CalcMaps struct {
	RunIdToLabel map[int]string // map of run id's to run label (digest)
	CalcIdToName map[int]string // map of calculation id to name
}

// return empty value of calculation maps
func EmptyCalcMaps() CalcMaps {
	return CalcMaps{
		RunIdToLabel: map[int]string{},
		CalcIdToName: map[int]string{},
	}
}

// CsvConverter provide methods to convert parameters or output table data to row []string for csv file.
type CsvIntKeysConverter interface {
	CsvConverter // convert parameter row or output table row to row []string for csv file
	CellIntKeys  // provide a method to get row keys as []int, for example, get sub id and dimension ids
}

// CsvConverter provide methods to convert parameter row, output table row or microdata row from or to row []string for csv file.
// Double format string is used for values or if value type is float, double, long double.
// If dimension type is enum based then csv row is enum code and cell.DimIds is enum id.
// If parameter type is enum based then cell value is enum id and csv row value is enum code.
type CsvConverter interface {
	// return file name of csv file to store parameter or output table rows
	CsvFileName() (string, error)

	// return true if csv converter is using enum id's for dimensions or attributes
	IsUseEnumId() bool

	// return first line of csv file with column names: expr_name,dim0,dim1,expr_value.
	// if IsIdCsv is true: expr_id,dim0,dim1,expr_value
	// if isAllAcc is true: sub_id,dim0,dim1,acc0,acc1,acc2
	CsvHeader() ([]string, error)

	// return converter from cell of parameter, output table or microdata to csv id's row []string.
	// converter simply sprint() dimension id's and value into []string buffer.
	// converter return isNotEmpty flag if cell value is not empty.
	ToCsvIdRow() (func(interface{}, []string) (bool, error), error)

	// return converter from cell of parameter, output table or microdata to csv row []string.
	// it does convert from enum id to code for all dimensions into []string buffer.
	// if this is a enum-based parameter value then it is also converted to enum code.
	// converter return isNotEmpty flag if cell value is not empty.
	ToCsvRow() (func(interface{}, []string) (bool, error), error)
}

// Provide locale-specific methods to convert parameter row, output table row or microdata row to []string row for csv file.
// For example convert dimension enum id to enum label in specific language or use locale format for numeric values: 1234.56 => 1 234,56
type CsvLocaleConverter interface {
	// return converter from cell of parameter, output table or microdata to csv row []string.
	// If dimension type is enum based then csv row is enum label.
	// If parameter type is enum based then csv row value is enum label.
	// Value and dimesions of built-in types converted to locale-specific strings, e.g.: 1234.56 => 1 234,56
	ToCsvRow() (func(interface{}, []string) (bool, error), error)

	// return first line of csv file with column names, for example: expr_name,Age,City,expr_value.
	CsvHeader() ([]string, error)
}

// Convert from csv row []string to parameter, output table or microdata cell (dimensions and value or microdata key and attributes value).
type CsvToCellConverter interface {
	// return converter from csv row []string to parameter, output table or microdata cell (dimensions and value or microdata key and attributes value)
	ToCell() (func(row []string) (interface{}, error), error)
}

// CellIntKeys provide method to get a copy of cell keys as []int for parameter or output table row,
// for example, return [parameter row sub id and dimension ids].
type CellIntKeys interface {

	// KeyIds return converter to copy row primary key into key []int.
	// Row primary keys are:
	//   parameter: (sub_id, dimension ids)
	//   accumulators: (acc_id, sub_id, dimension ids)
	//   all accumulators: (sub_id, dimension ids)
	//   expressions: (expr_id, dimension ids)
	KeyIds(name string) (func(interface{}, []int) error, error)
}

// CellToCodeConverter provide methods to convert parameters or output table row from enum id to enum code.
// If dimension type is enum based then dimensions enum ids can be converted to enum code.
// If dimension type is simple (bool or int) then dimension value converted to string.
// If parameter type is enum based then cell value enum id converted to enum code.
type CellToCodeConverter interface {

	// IdToCodeCell return converter from id cell to code cell.
	// Cell is dimensions and value of parameter or output table.
	// It does convert from enum id to code for all dimensions and enum-based parameter value.
	IdToCodeCell(modelDef *ModelMeta, name string) (func(interface{}) (interface{}, error), error)
}

// CellToIdConverter provide methods to convert parameters or output table row from enum code to enum id.
// If dimension type is enum based then dimensions enum codes converted to enum ids.
// If dimension type is simple (bool or int) then dimension code converted from string to dimension type.
// If parameter type is enum based then cell value enum code converted to enum id.
type CellToIdConverter interface {

	// CodeToIdCell return converter from code cell to id cell.
	// Cell is dimensions and value of parameter or output table.
	// It does convert from enum code to id for all dimensions and enum-based parameter value.
	CodeToIdCell(modelDef *ModelMeta, name string) (func(interface{}) (interface{}, error), error)
}

// Return converter from dimension item id to code.
// It is also used for parameter values if parameter type is enum-based.
// If dimension is enum-based then from enum id to enum code or to the "all" total enum code;
// If dimension is simple integer type then use Itoa(integer id) as code;
// If dimension is boolean then 0=>false, (1 or -1)=>true else error
func (typeOf *TypeMeta) itemIdToCode(msgName string, isTotalEnabled bool) (func(itemId int) (string, error), error) {

	var cvt func(itemId int) (string, error)

	switch {
	case !typeOf.IsBuiltIn(): // enum dimension: find enum code by id

		cvt = func(itemId int) (string, error) {

			if isTotalEnabled && itemId == typeOf.TotalEnumId { // check is it total item
				return TotalEnumCode, nil
			}
			if !typeOf.IsRange { // enum dimension: find enum code by id

				for j := range typeOf.Enum {
					if itemId == typeOf.Enum[j].EnumId {
						return typeOf.Enum[j].Name, nil
					}
				}
			} else { // range dimension: item id the same as code

				if typeOf.MinEnumId <= itemId && itemId <= typeOf.MaxEnumId {
					return strconv.Itoa(itemId), nil
				}
			}
			return "", errors.New("invalid value: " + strconv.Itoa(itemId) + " of: " + msgName)
		}

	case typeOf.IsBool(): // boolean dimension: 0=>false, (1 or -1)=>true else error

		cvt = func(itemId int) (string, error) {
			switch itemId {
			case 0:
				return "false", nil
			case 1, -1:
				return "true", nil
			}
			if isTotalEnabled && itemId == typeOf.TotalEnumId { // check is it total item
				return TotalEnumCode, nil
			}
			return "", errors.New("invalid value: " + strconv.Itoa(itemId) + " of: " + msgName)
		}

	case typeOf.IsInt(): // integer dimension

		cvt = func(itemId int) (string, error) {
			return strconv.Itoa(itemId), nil
		}

	default:
		return nil, errors.New("invalid (not supported) type: " + typeOf.Name + " of: " + msgName)
	}

	return cvt, nil
}

// Return converter from dimension item id to language-specific label.
// If language code is empty then it returns itemIdToCode converter from item id to item code
// It is also used for parameter values if parameter type is enum-based.
// If dimension is enum-based then from enum id to enum description or to the "all" total enum label;
// If dimension is simple integer type then use Itoa(integer id) as code;
// If dimension is boolean then 0=>false, (1 or -1)=>true else error
func (typeOf *TypeMeta) itemIdToLabel(lang string, enumTxt []TypeEnumTxtRow, msgLst []LangMsg, msgName string, isTotalEnabled bool) (func(itemId int) (string, error), error) {

	if lang == "" {
		return typeOf.itemIdToCode(msgName, isTotalEnabled) // language is empty: retrun converter from id to enum code
	}
	var cvt func(itemId int) (string, error)

	// for boolean type or enum based types which are not ranges create map of enum id to label
	labelMap := make(map[int]string, len(typeOf.Enum))

	if typeOf.IsBool() || !typeOf.IsBuiltIn() && !typeOf.IsRange {

		// add item code into map as default label
		for j := range typeOf.Enum {
			labelMap[typeOf.Enum[j].EnumId] = typeOf.Enum[j].Name
		}
		// replace labels: use description where exists for specified language
		for j := range enumTxt {
			if enumTxt[j].ModelId == typeOf.ModelId && enumTxt[j].TypeId == typeOf.TypeId && enumTxt[j].LangCode == lang {
				labelMap[enumTxt[j].EnumId] = enumTxt[j].Descr
			}
		}
	}

	// find language-specific total label
	allLabel := TotalEnumCode

	if isTotalEnabled {
		for j := range msgLst {
			if msgLst[j].LangCode == lang {
				if lbl, ok := msgLst[j].Msg[TotalEnumCode]; ok {
					allLabel = lbl
				}
			}
		}
	}

	prt := message.NewPrinter(language.Make(lang)) // printer to format built-in types

	switch {
	case typeOf.IsBool() || !typeOf.IsBuiltIn() && !typeOf.IsRange: // boolean or enum dimension: find label by enum id

		cvt = func(itemId int) (string, error) {

			if lbl, ok := labelMap[itemId]; ok {
				return lbl, nil
			}
			if isTotalEnabled && itemId == typeOf.TotalEnumId { // check is it total item
				return allLabel, nil
			}
			return "", errors.New("invalid value: " + strconv.Itoa(itemId) + " of: " + msgName)
		}

	case typeOf.IsRange: // range dimension: item id the same as label

		cvt = func(itemId int) (string, error) {

			if typeOf.MinEnumId <= itemId && itemId <= typeOf.MaxEnumId {
				return prt.Sprintf("%d", itemId), nil
			}
			if isTotalEnabled && itemId == typeOf.TotalEnumId { // check is it total item
				return allLabel, nil
			}
			return "", errors.New("invalid value: " + strconv.Itoa(itemId) + " of: " + msgName)
		}

	case typeOf.IsInt(): // integer dimension

		cvt = func(itemId int) (string, error) {
			return prt.Sprintf("%d", itemId), nil
		}

	default:
		return nil, errors.New("invalid (not supported) type: " + typeOf.Name + " of: " + msgName)
	}

	return cvt, nil
}
