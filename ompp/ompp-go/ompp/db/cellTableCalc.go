// Copyright (c) 2021 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"errors"
	"fmt"
	"strconv"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// CellTableCalc is value of output table calculated expression.
type CellTableCalc struct {
	cellIdValue     // dimensions as enum id's and value
	CalcId      int // calculated expression id
	RunId       int // model run id
}

// CellCodeTableCalc is value of output table calculated expression.
// Dimension(s) items are enum codes, not enum ids.
type CellCodeTableCalc struct {
	cellCodeValue        // dimensions as enum codes and value
	CalcName      string // calculated expression name
	RunDigest     string // model run digest
}

// CellTableCalcConverter is a converter for output table calculated cell to implement CsvConverter interface.
type CellTableCalcConverter struct {
	CellTableConverter // model metadata and output table name
	CalcMaps           // map between runs digest and id and calculations name and id
}

// Converter for output table expression to implement CsvLocaleConverter interface.
type CellTableCalcLocaleConverter struct {
	CellTableCalcConverter
	Lang    string            // language code, expected to compatible with BCP 47 language tag
	MsgDef  []LangMsg         // translated strings in multiple languages
	DimsTxt []TableDimsTxtRow // output table dimension text rows: table_dims_txt join to model_table_dic
	EnumTxt []TypeEnumTxtRow  // type enum text rows: type_enum_txt join to model_type_dic
}

// Set calculation Id to name maps
func (cellCvt *CellTableCalcConverter) SetCalcIdNameMap(calcLt []CalculateTableLayout) error {

	cellCvt.CalcIdToName = map[int]string{}

	for k, c := range calcLt {

		if c.Name == "" {
			return errors.New("invalid (empty) calculation name at index: [" + strconv.Itoa(k) + "], id: " + strconv.Itoa(c.CalcId) + ": " + cellCvt.Name)
		}
		cellCvt.CalcIdToName[c.CalcId] = c.Name
	}
	return nil
}

// return true if csv converter is using enum id's for dimensions
func (cellCvt *CellTableCalcConverter) IsUseEnumId() bool { return cellCvt.IsIdCsv }

// Return file name of csv file to store output table calculated rows
func (cellCvt *CellTableCalcConverter) CsvFileName() (string, error) {

	// find output table by name
	_, err := cellCvt.tableByName()
	if err != nil {
		return "", err
	}

	// make csv file name
	if cellCvt.IsIdCsv {
		return cellCvt.Name + ".id.calc.csv", nil
	}
	return cellCvt.Name + ".calc.csv", nil
}

// Return first line for csv file: column names, for example: run_digest,calc_id,dim0,dim1,calc_value
func (cellCvt *CellTableCalcConverter) CsvHeader() ([]string, error) {

	// find output table by name
	table, err := cellCvt.tableByName()
	if err != nil {
		return []string{}, err
	}

	// make first line columns
	h := make([]string, table.Rank+3)

	if cellCvt.IsIdCsv {
		h[0] = "run_id"
		h[1] = "calc_id"
	} else {
		h[0] = "run_digest"
		h[1] = "calc_name"
	}
	for k := range table.Dim {
		h[k+2] = table.Dim[k].Name
	}
	h[table.Rank+2] = "calc_value"

	return h, nil
}

// Return first line for csv file: column names.
// For example: run_label,calc_name,Age,Sex,calc_value
func (cellCvt *CellTableCalcLocaleConverter) CsvHeader() ([]string, error) {

	// default column headers
	h, err := cellCvt.CellTableCalcConverter.CsvHeader()
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
				h[k+2] = d
			}
		}
	}
	return h, nil
}

// Return converter to copy primary key: (run_id, calc_id, dimension ids) into key []int.
//
// Converter will return error if len(key) not equal to row key size.
func (cellCvt *CellTableCalcConverter) KeyIds(name string) (func(interface{}, []int) error, error) {

	cvt := func(src interface{}, key []int) error {

		cell, ok := src.(CellTableCalc)
		if !ok {
			return errors.New("invalid type, expected: CellTableCalc (internal error): " + name)
		}

		n := len(cell.DimIds)
		if len(key) != n+2 {
			return errors.New("invalid size of key buffer, expected: " + strconv.Itoa(n+2) + ": " + name)
		}

		key[0] = cell.RunId
		key[1] = cell.CalcId

		for k, e := range cell.DimIds {
			key[k+2] = e
		}
		return nil
	}

	return cvt, nil
}

// Return converter from output table calculated cell (run_id, calc_id, dimensions, calc_value) to csv id's row []string.
//
// Converter return isNotEmpty flag: false if IsNoZero or IsNoNull is set and cell value is empty or zero.
// Converter simply does Sprint() for each dimension item id, run id and value.
// Converter will return error if len(row) not equal to number of fields in csv record.
func (cellCvt *CellTableCalcConverter) ToCsvIdRow() (func(interface{}, []string) (bool, error), error) {

	// find output table by name
	_, err := cellCvt.tableByName()
	if err != nil {
		return nil, err
	}

	// return converter from id based cell to csv string array
	cvt := func(src interface{}, row []string) (bool, error) {

		cell, ok := src.(CellTableCalc)
		if !ok {
			return false, errors.New("invalid type, expected: CellTableCalc (internal error)")
		}

		n := len(cell.DimIds)
		if len(row) != n+3 {
			return false, errors.New("invalid size of csv row buffer, expected: " + strconv.Itoa(n+3) + ": " + cellCvt.Name)
		}

		row[0] = fmt.Sprint(cell.RunId)
		row[1] = fmt.Sprint(cell.CalcId)

		for k, e := range cell.DimIds {
			row[k+2] = fmt.Sprint(e)
		}

		// use "null" string for db NULL values and format for model float types
		isNotEmpty := true

		if cell.IsNull {
			row[n+2] = "null"
			isNotEmpty = !cellCvt.IsNoNullCsv
		} else {

			if cellCvt.IsNoZeroCsv {
				fv, ok := cell.Value.(float64)
				isNotEmpty = ok && fv != 0
			}

			if cellCvt.DoubleFmt != "" {
				row[n+2] = fmt.Sprintf(cellCvt.DoubleFmt, cell.Value)
			} else {
				row[n+2] = fmt.Sprint(cell.Value)
			}
		}
		return isNotEmpty, nil
	}

	return cvt, nil
}

// Return converter from output table calculated cell (run_id, calc_id, dimensions, calc_value)
// to csv row []string (run digest, calc_name, dimensions, calc_value).
//
// Converter return isNotEmpty flag: false if IsNoZero or IsNoNull is set and cell value is empty or zero.
// Converter will return error if len(row) not equal to number of fields in csv record.
// Converter will return error if run_id not exist in the list of model runs (in run_lst table).
// Double format string is used if parameter type is float, double, long double.
// If dimension type is enum based then csv row is enum code.
func (cellCvt *CellTableCalcConverter) ToCsvRow() (func(interface{}, []string) (bool, error), error) {

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

		cell, ok := src.(CellTableCalc)
		if !ok {
			return false, errors.New("invalid type, expected: output table calculated cell (internal error)")
		}

		n := len(cell.DimIds)
		if len(row) != n+3 {
			return false, errors.New("invalid size of csv row buffer, expected: " + strconv.Itoa(n+3) + ": " + cellCvt.Name)
		}

		row[0] = cellCvt.RunIdToLabel[cell.RunId]
		if row[0] == "" {
			return false, errors.New("invalid (missing) run id: " + strconv.Itoa(cell.RunId) + " output table: " + cellCvt.Name)
		}
		row[1] = cellCvt.CalcIdToName[cell.CalcId]
		if row[1] == "" {
			return false, errors.New("invalid (missing) calculation id: " + strconv.Itoa(cell.CalcId) + " output table: " + cellCvt.Name)
		}

		// convert dimension item id to code
		for k, e := range cell.DimIds {
			v, err := fd[k](e)
			if err != nil {
				return false, err
			}
			row[k+2] = v
		}

		// use "null" string for db NULL values and format for model float types
		isNotEmpty := true

		if cell.IsNull {
			row[n+2] = "null"
			isNotEmpty = !cellCvt.IsNoNullCsv
		} else {

			if cellCvt.IsNoZeroCsv {
				fv, ok := cell.Value.(float64)
				isNotEmpty = ok && fv != 0
			}

			if cellCvt.DoubleFmt != "" {
				row[n+2] = fmt.Sprintf(cellCvt.DoubleFmt, cell.Value)
			} else {
				row[n+2] = fmt.Sprint(cell.Value)
			}
		}
		return isNotEmpty, nil
	}

	return cvt, nil
}

// Return converter from output table calculated cell (run_id, calc_id, dimensions, calc_value)
// to language-specific csv []string row of dimension enum labels and value.
//
// Converter return isNotEmpty flag: false if IsNoZero or IsNoNull is set and cell value is empty or zero.
// If dimension type is enum based then csv row is enum label.
// Value and dimesions of built-in types converted to locale-specific strings, e.g.: 1234.56 => 1 234,56
// Converter will return error if len(row) not equal to number of fields in csv record.
func (cellCvt *CellTableCalcLocaleConverter) ToCsvRow() (func(interface{}, []string) (bool, error), error) {

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

	// format value locale-specific strings, e.g.: 1234.56 => 1 234,56
	prt := message.NewPrinter(language.Make(cellCvt.Lang))

	cvt := func(src interface{}, row []string) (bool, error) {

		cell, ok := src.(CellTableCalc)
		if !ok {
			return false, errors.New("invalid type, expected: output table calculated cell (internal error)")
		}

		n := len(cell.DimIds)
		if len(row) != n+3 {
			return false, errors.New("invalid size of csv row buffer, expected: " + strconv.Itoa(n+3) + ": " + cellCvt.Name)
		}

		row[0] = cellCvt.RunIdToLabel[cell.RunId]
		if row[0] == "" {
			return false, errors.New("invalid (missing) run id: " + strconv.Itoa(cell.RunId) + " output table: " + cellCvt.Name)
		}
		row[1] = cellCvt.CalcIdToName[cell.CalcId]
		if row[1] == "" {
			return false, errors.New("invalid (missing) calculation id: " + strconv.Itoa(cell.CalcId) + " output table: " + cellCvt.Name)
		}

		// convert dimension item id to label
		for k, e := range cell.DimIds {
			v, err := fd[k](e)
			if err != nil {
				return false, err
			}
			row[k+2] = v
		}

		// use "null" string for db NULL values and format for model float types
		isNotEmpty := true

		if cell.IsNull {
			row[n+2] = "null"
			isNotEmpty = !cellCvt.IsNoNullCsv
		} else {

			if cellCvt.IsNoZeroCsv {
				fv, ok := cell.Value.(float64)
				isNotEmpty = ok && fv != 0
			}

			if cellCvt.DoubleFmt != "" {
				row[n+2] = prt.Sprintf(cellCvt.DoubleFmt, cell.Value)
			} else {
				row[n+2] = prt.Sprint(cell.Value)
			}
		}
		return isNotEmpty, nil

	}

	return cvt, nil
}

// Return converter from output table calculated cell of ids: (run_id, calc_id, dimensions enum ids, calc_value)
// to cell of codes: (RunDigest, CalcName, dimensions as enum codes, calc_value).
// Output RunDigest value is coming from RunIdToLabel map and it can be not a run digest but other label, e.g. run name or description.
//
// If dimension type is enum based then dimensions enum ids can be converted to enum code.
// If dimension type is simple (bool or int) then dimension value converted to string.
func (cellCvt *CellTableCalcConverter) IdToCodeCell(modelDef *ModelMeta, name string) (func(interface{}) (interface{}, error), error) {

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

		srcCell, ok := src.(CellTableCalc)
		if !ok {
			return nil, errors.New("invalid type, expected: output table calculated cell (internal error): " + name)
		}
		if len(srcCell.DimIds) != table.Rank {
			return nil, errors.New("invalid cell rank: " + strconv.Itoa(len(srcCell.DimIds)) + ", expected: " + strconv.Itoa(table.Rank) + ": " + name)
		}

		dgst := cellCvt.RunIdToLabel[srcCell.RunId]
		if dgst == "" {
			return nil, errors.New("invalid (missing) run id: " + strconv.Itoa(srcCell.RunId) + " output table: " + name)
		}
		cName := cellCvt.CalcIdToName[srcCell.CalcId]
		if cName == "" {
			return nil, errors.New("invalid (missing) calculation id: " + strconv.Itoa(srcCell.CalcId) + " output table: " + name)
		}

		dstCell := CellCodeTableCalc{
			cellCodeValue: cellCodeValue{
				Dims:   make([]string, table.Rank),
				IsNull: srcCell.IsNull,
				Value:  srcCell.Value,
			},
			CalcName:  cName,
			RunDigest: dgst,
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
