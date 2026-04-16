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

// CellMicro is a row of entity microdata: entity_key and attribute values,
// if attribute type is enum based value is enum id
type CellMicro struct {
	Key  uint64      // microdata key, part of row primary key: entity_key
	Attr []attrValue // attributes value: built-in type value or enum id
}

// CellCodeMicro is a row of entity microdata: entity_key and attribute values,
// if attribute type is enum based value is enum code
type CellCodeMicro struct {
	Key  uint64      // microdata key, part of row primary key: entity_key
	Attr []attrValue // attributes value: built-in type value or enum code
}

// Entity attribute value: value of built-in type or enum value.
// If attribute type is enum based then value is enum id or enum code, depending on where attrValue is used,
// for CellMicro.[]Attr it is enum id's and for CellCodeMicro.[]Attr it is enum codes.
type attrValue struct {
	IsNull bool        // if true then value is NULL
	Value  interface{} // value: int64, bool, float64 or string
}

// CellMicroConverter  is a parent for for entity microdata converters.
type CellEntityConverter struct {
	ModelDef    *ModelMeta      // model metadata
	Name        string          // model entity name
	EntityGen   *EntityGenMeta  // model run entity generation
	IsIdCsv     bool            // if true then use enum id's else use enum codes
	DoubleFmt   string          // if not empty then format string is used to sprintf if value type is float, double, long double
	IsNoZeroCsv bool            // if true then do not write zero values into csv output
	IsNoNullCsv bool            // if true then do not write NULL values into csv output
	theEntity   *EntityMeta     // if not nil then entity found
	theAttrs    []EntityAttrRow // if not empty then entity generation attributes
}

// CellMicroConverter is a converter for entity microdata row to implement CsvConverter interface.
type CellMicroConverter struct {
	CellEntityConverter // model metadata, entity generation and and attributes
}

// Converter for entity microdata to implement CsvLocaleConverter interface.
type CellMicroLocaleConverter struct {
	CellMicroConverter
	Lang          string             // language code, expected to compatible with BCP 47 language tag
	EnumTxt       []TypeEnumTxtRow   // type enum text rows: type_enum_txt join to model_type_dic
	AttrTxt       []EntityAttrTxtRow // entity attributes text rows: entity_attr_txt join to model_entity_dic table
	theAttrLabels map[int]string     // map entity generation attribute id to language-specific label
}

// return true if csv converter is using enum id's for attributes
func (cellCvt *CellMicroConverter) IsUseEnumId() bool { return cellCvt.IsIdCsv }

// CsvFileName return file name of csv file to store entity microdata rows
func (cellCvt *CellMicroConverter) CsvFileName() (string, error) {

	// find entity by name
	_, err := cellCvt.entityByName()
	if err != nil {
		return "", err
	}

	// make csv file name
	if cellCvt.IsIdCsv {
		return cellCvt.Name + ".id.csv", nil
	}
	return cellCvt.Name + ".csv", nil
}

// CsvHeader return first line for csv file: column names. For example: key,AgeGroup,Income.
func (cellCvt *CellMicroConverter) CsvHeader() ([]string, error) {

	// find entity metadata by entity name and attributes by generation Hid
	_, attrs, err := cellCvt.entityAttrs()
	if err != nil {
		return []string{}, err
	}

	// make first line columns
	h := make([]string, 1+len(attrs))
	h[0] = "key"

	for k, ea := range attrs {
		h[k+1] = ea.Name
	}
	return h, nil
}

// CsvHeader return first line for csv file: column names. For example: key,Age Group,Taxable Income.
func (cellCvt *CellMicroLocaleConverter) CsvHeader() ([]string, error) {

	// default column headers
	h, err := cellCvt.CellMicroConverter.CsvHeader()
	if err != nil {
		return []string{}, err
	}

	// replace attribute name with description, where it exists
	if cellCvt.Lang != "" {

		// find entity metadata by entity name and attributes by generation Hid
		_, attrs, err := cellCvt.entityAttrs()
		if err != nil {
			return []string{}, err
		}

		am, err := cellCvt.attrLabel() // attribute labels
		if err != nil {
			return []string{}, err
		}

		for k, ea := range attrs {
			if d, ok := am[ea.AttrId]; ok {
				h[k+1] = d
			}
		}
	}
	return h, nil
}

// ToCsvIdRow return converter from microdata cell: (microdata key, attributes as enum id or built-in type value) to csv id's row []string.
//
// Converter return isNotEmpty flag: false if IsNoZero or IsNoNull is set and all attributes values are empty or zero,
// only attributes of type float or integer or string are considered as "value" attributes.
// Converter simply does Sprint() for key and each attribute value.
// If value is NULL then empty "" string used.
// Converter will return error if len(row) not equal to number of fields in csv record.
func (cellCvt *CellMicroConverter) ToCsvIdRow() (func(interface{}, []string) (bool, error), error) {

	// find entity metadata by entity name and attributes by generation Hid
	_, attrs, err := cellCvt.entityAttrs()
	if err != nil {
		return nil, err
	}
	nAttr := len(attrs)

	// convert attributes value to string using Sprint() or Sprintf(double format)
	fd := make([]func(v interface{}) string, nAttr)

	for k, ea := range attrs {

		// for float attributes use format if specified
		if cellCvt.DoubleFmt != "" && ea.typeOf.IsFloat() {
			fd[k] = func(v interface{}) string { return fmt.Sprintf(cellCvt.DoubleFmt, v) }
		} else {
			fd[k] = func(v interface{}) string { return fmt.Sprint(v) }
		}
	}

	// return converter for microdata key and attribute values
	cvt := func(src interface{}, row []string) (bool, error) {

		cell, ok := src.(CellMicro)
		if !ok {
			return false, errors.New("invalid type, expected: CellMicro (internal error): " + cellCvt.Name)
		}

		n := len(cell.Attr)
		if n != nAttr || len(row) != n+1 {
			return false, errors.New("invalid size of csv row buffer, expected: " + strconv.Itoa(nAttr+1) + ": " + cellCvt.Name)
		}

		// check for empty data: if all values are NULLs or zeros and no null or no zero flag is set
		isAllEmpty, e := cellCvt.isAllEmpty(cell, attrs)
		if e != nil {
			return false, e
		}

		// convert attributes
		row[0] = fmt.Sprint(cell.Key) // first column is entity microdata key

		for k, a := range cell.Attr {

			// use "null" string for db NULL values
			if a.IsNull || a.Value == nil {
				row[k+1] = "null"
			} else {
				row[k+1] = fd[k](a.Value)
			}
		}
		return !isAllEmpty, nil
	}
	return cvt, nil
}

// Return converter from microdata cell: (microdata key, attributes as enum id or built-in type value) to csv row []string.
//
// Converter return isNotEmpty flag: false if IsNoZero or IsNoNull is set and all values of float or integer or string type are empty or zero.
// Converter simply does Sprint() for key and each attribute value.
// If attribute type is float and double format is not empty "" string then converter does Sprintf(using double format).
// If attribute type is enum based then converter return enum code for attribute enum id.
// If value is NULL then "null" string used.
// Converter will return error if len(row) not equal to number of fields in csv record.
func (cellCvt *CellMicroConverter) ToCsvRow() (func(interface{}, []string) (bool, error), error) {

	// find entity metadata by entity name and attributes by generation Hid
	_, attrs, err := cellCvt.entityAttrs()
	if err != nil {
		return nil, err
	}
	nAttr := len(attrs)

	// convert attributes value to string:
	// for built-in attribute type use Sprint() or Sprintf(double format)
	// for enum attribute type return enum code by enum id
	fd := make([]func(v interface{}) (string, error), nAttr)

	for k, ea := range attrs {

		if ea.typeOf.IsBuiltIn() { // built-in attribute type: format value by Sprint()

			// for float attributes use format if specified
			if cellCvt.DoubleFmt != "" && ea.typeOf.IsFloat() {
				fd[k] = func(v interface{}) (string, error) { return fmt.Sprintf(cellCvt.DoubleFmt, v), nil }
			} else {
				fd[k] = func(v interface{}) (string, error) { return fmt.Sprint(v), nil }
			}
		} else { // enum based attribute type: find and return enum code by enum id

			msgName := cellCvt.Name + "." + ea.Name // for error message, ex: Person.Income
			f, err := ea.typeOf.itemIdToCode(msgName, false)
			if err != nil {
				return nil, err
			}

			fd[k] = func(v interface{}) (string, error) { // convereter return enum code by enum id

				// depending on sql + driver it can be different type
				if iv, ok := helper.ToIntValue(v); ok {
					return f(iv)
				} else {
					return "", errors.New("invalid attribute value, must be integer enum id: " + msgName)
				}
			}
		}
	}

	// return converter for microdata key and attribute values
	cvt := func(src interface{}, row []string) (bool, error) {

		cell, ok := src.(CellMicro)
		if !ok {
			return false, errors.New("invalid type, expected: CellMicro (internal error): " + cellCvt.Name)
		}

		n := len(cell.Attr)
		if n != nAttr || len(row) != n+1 {
			return false, errors.New("invalid size of csv row buffer, expected: " + strconv.Itoa(nAttr+1) + ": " + cellCvt.Name)
		}

		// check for empty data: if all values are NULLs or zeros and no null or no zero flag is set
		isAllEmpty, e := cellCvt.isAllEmpty(cell, attrs)
		if e != nil {
			return false, e
		}

		// convert attributes
		row[0] = fmt.Sprint(cell.Key) // first column is entity microdata key

		for k, a := range cell.Attr {

			// use "null" string for db NULL values
			if a.IsNull || a.Value == nil {
				row[k+1] = "null"
			} else {
				if s, e := fd[k](a.Value); e != nil { // use attribute value converter
					return false, e
				} else {
					row[k+1] = s
				}
			}
		}
		return !isAllEmpty, nil
	}
	return cvt, nil
}

// Return converter from microdata cell: (microdata key, attributes as enum id or built-in type value)
// to language-specific csv []string row of dimension enum labels and value.
//
// Converter return isNotEmpty flag: false if IsNoZero or IsNoNull is set and all values of float or integer or string type are empty or zero.
// Microdata row key and attribute values of built-in type converted to locale-specific strings, e.g.: 1234.56 => 1 234,56.
// If value is NULL then "null" string used.
// If attribute type is enum based then csv value is enum label.
// Converter will return error if len(row) not equal to number of fields in csv record.
func (cellCvt *CellMicroLocaleConverter) ToCsvRow() (func(interface{}, []string) (bool, error), error) {

	// find entity metadata by entity name and attributes by generation Hid
	_, attrs, err := cellCvt.entityAttrs()
	if err != nil {
		return nil, err
	}
	nAttr := len(attrs)

	// for built-in attribute types format value locale-specific strings, e.g.: 1234.56 => 1 234,56
	prt := message.NewPrinter(language.Make(cellCvt.Lang))

	// for enum attribute type return enum label by enum id
	fd := make([]func(v interface{}) (string, error), nAttr)

	for k, ea := range attrs {

		if ea.typeOf.IsBuiltIn() { // built-in attribute type: format value by Sprint()

			// for float attributes use format if specified
			if cellCvt.DoubleFmt != "" && ea.typeOf.IsFloat() {
				fd[k] = func(v interface{}) (string, error) { return prt.Sprintf(cellCvt.DoubleFmt, v), nil }
			} else {
				fd[k] = func(v interface{}) (string, error) { return prt.Sprint(v), nil }
			}
		} else { // enum based attribute type: find and return enum label by enum id

			msgName := cellCvt.Name + "." + ea.Name // for error message, ex: Person.Income
			f, err := ea.typeOf.itemIdToLabel(cellCvt.Lang, cellCvt.EnumTxt, nil, msgName, false)

			if err != nil {
				return nil, err
			}

			fd[k] = func(v interface{}) (string, error) { // convereter return enum label by enum id

				// depending on sql + driver it can be different type
				if iv, ok := helper.ToIntValue(v); ok {
					return f(iv)
				} else {
					return "", errors.New("invalid attribute value, must be integer enum id: " + msgName)
				}
			}
		}
	}

	// return converter for microdata key and attribute values
	cvt := func(src interface{}, row []string) (bool, error) {

		cell, ok := src.(CellMicro)
		if !ok {
			return false, errors.New("invalid type, expected: CellMicro (internal error): " + cellCvt.Name)
		}

		n := len(cell.Attr)
		if n != nAttr || len(row) != n+1 {
			return false, errors.New("invalid size of csv row buffer, expected: " + strconv.Itoa(nAttr+1) + ": " + cellCvt.Name)
		}

		// check for empty data: if all values are NULLs or zeros and no null or no zero flag is set
		isAllEmpty, e := cellCvt.isAllEmpty(cell, attrs)
		if e != nil {
			return false, e
		}

		// convert attributes
		row[0] = prt.Sprint(cell.Key) // first column is entity microdata key

		for k, a := range cell.Attr {

			// use "null" string for db NULL values
			if a.IsNull || a.Value == nil {
				row[k+1] = "null"
			} else {
				if s, e := fd[k](a.Value); e != nil { // use attribute value converter
					return false, e
				} else {
					row[k+1] = s
				}
			}
		}
		return !isAllEmpty, nil
	}
	return cvt, nil
}

// check for empty data: if all values are NULLs or zeros and no null or no zero flag is set.
// only attributes of type float or integer or string are considered as "value" attributes.
func (cellCvt *CellMicroConverter) isAllEmpty(cell CellMicro, attrs []EntityAttrRow) (bool, error) {

	isAll := cellCvt.IsNoZeroCsv || cellCvt.IsNoNullCsv

	for k, a := range cell.Attr {

		if !isAll {
			break
		}

		if !attrs[k].typeOf.IsBuiltIn() ||
			!attrs[k].typeOf.IsFloat() && !attrs[k].typeOf.IsInt() && !attrs[k].typeOf.IsString() {
			continue // only float or intger or string attributes are considered as values
		}

		if a.IsNull || a.Value == nil {
			isAll = cellCvt.IsNoNullCsv
		} else {

			isAll = cellCvt.IsNoZeroCsv

			if isAll {
				switch {
				case attrs[k].typeOf.IsFloat():
					fv, ok := a.Value.(float64)
					isAll = ok && fv == 0
				case attrs[k].typeOf.IsString():
					sv, ok := a.Value.(string)
					isAll = ok && sv == ""
				case attrs[k].typeOf.IsInt():
					iv, ok := helper.ToIntValue(a.Value)
					isAll = ok && iv == 0
				default:
					return false, errors.New("invalid (not supported) entity attribute type: " + cellCvt.Name + "." + attrs[k].Name)
				}
			}
		}
	}
	return isAll, nil
}

// CsvToCell return closure to convert csv row []string to microdata cell (key, attributes value).
//
// If attribute type is enum based then csv row contains enum code and it is converted into cell attribute enum id.
// It does return error if len(row) not equal to number of fields in cell db-record.
// Converter will return error if len(row) not equal to number of fields in csv record.
func (cellCvt *CellMicroConverter) ToCell() (func(row []string) (interface{}, error), error) {

	// find entity metadata by entity name and attributes by generation Hid
	_, attrs, err := cellCvt.entityAttrs()
	if err != nil {
		return nil, err
	}
	nAttr := len(attrs)

	// convert attributes string to value:
	// for built-in attribute type use ParseFloat(), ParseBool() or Atoi()
	// for enum attribute type return enum id by code
	fd := make([]func(src string) (interface{}, error), nAttr)

	for k, ea := range attrs {

		msgName := cellCvt.Name + "." + ea.Name // for error message, ex: Person.Income

		// attribute type to create converter

		switch {
		case !ea.typeOf.IsBuiltIn(): // enum based attribute type: find and return enum id by enum code

			f, err := ea.typeOf.itemCodeToId(msgName, false)
			if err != nil {
				return nil, err
			}

			fd[k] = func(src string) (interface{}, error) { // convereter return enum id by code
				return f(src)
			}

		case ea.typeOf.IsFloat(): // float types, only 64 or 32 bits supported

			n := 64
			if ea.typeOf.IsFloat32() {
				n = 32
			}

			fd[k] = func(src string) (interface{}, error) {
				vf, e := strconv.ParseFloat(src, n)
				if e != nil {
					return 0.0, e
				}
				return vf, nil
			}

		case ea.typeOf.IsBool():
			fd[k] = func(src string) (interface{}, error) { return strconv.ParseBool(src) }
		case ea.typeOf.IsString():
			fd[k] = func(src string) (interface{}, error) { return src, nil }
		case ea.typeOf.IsInt():
			fd[k] = func(src string) (interface{}, error) { return strconv.Atoi(src) }
		default:
			return nil, errors.New("invalid (not supported) entity attribute type: " + msgName)
		}
	}

	// return converter from csv strings into microdata key and attribute values
	cvt := func(row []string) (interface{}, error) {

		cell := CellMicro{Attr: make([]attrValue, nAttr)}

		n := len(cell.Attr)
		if n != nAttr || len(row) != n+1 {
			return nil, errors.New("invalid size of csv row, expected: " + strconv.Itoa(nAttr+1) + ": " + cellCvt.Name)
		}

		// convert microdata key, it is uint 64 bit
		if row[0] == "" || row[0] == "null" {
			return nil, errors.New("invalid microdata key, it cannot be NULL: " + cellCvt.Name)
		}

		mKey, err := strconv.ParseUint(row[0], 10, 64)
		if err != nil {
			return nil, err
		}
		cell.Key = mKey

		// convert attributes
		for k := 0; k < nAttr; k++ {

			cell.Attr[k].IsNull = row[k+1] == "" || row[k+1] == "null"

			if !cell.Attr[k].IsNull {
				v, e := fd[k](row[k+1])
				if e != nil {
					return nil, err
				}
				cell.Attr[k].Value = v
			}
		}

		return cell, nil
	}
	return cvt, nil
}

// IdToCodeCell return converter
// from microdata cell of ids: (entity key, attributes as built-in values or enum id's)
// into cell of codes: (entity key, attributes as built-in values or enum codes)
//
// If attribute type is enum based then attribute enum id converted to enum code.
// If attribute type is built-in (bool, int, float) then return attribute value as is, no conversion.
func (cellCvt *CellMicroConverter) IdToCodeCell(modelDef *ModelMeta, _ string) (func(interface{}) (interface{}, error), error) {

	// find entity metadata by entity name and attributes by generation Hid
	_, attrs, err := cellCvt.entityAttrs()
	if err != nil {
		return nil, err
	}
	nAttr := len(attrs)

	// convert attributes value to string if attribute is enum based: return enum code by enum id
	// do not convert built-in attribute type, converter function is nil
	fa := make([]func(v interface{}) (string, error), nAttr)

	for k, ea := range attrs {

		if ea.typeOf.IsBuiltIn() {

			fa[k] = nil // built-in attribute type: do not convert

		} else { // enum based attribute type: find and return enum code by enum id

			msgName := cellCvt.Name + "." + ea.Name // for error message, ex: Person.Income
			f, err := ea.typeOf.itemIdToCode(msgName, false)
			if err != nil {
				return nil, err
			}

			fa[k] = func(v interface{}) (string, error) { // convereter return enum code by enum id

				// depending on sql + driver it can be different type
				if iv, ok := helper.ToIntValue(v); ok {
					return f(iv)
				} else {
					return "", errors.New("invalid attribute value, must be integer enum id: " + msgName)
				}
			}
		}
	}

	// create cell converter
	cvt := func(src interface{}) (interface{}, error) {

		srcCell, ok := src.(CellMicro)
		if !ok {
			return nil, errors.New("invalid type, expected: microdata cell (internal error): " + cellCvt.Name)
		}
		if len(srcCell.Attr) != nAttr {
			return nil, errors.New("invalid number of attributes, expected: " + strconv.Itoa(nAttr) + ": " + cellCvt.Name)
		}

		dstCell := CellCodeMicro{
			Key:  srcCell.Key,
			Attr: make([]attrValue, nAttr),
		}

		// convert attributes enum id's to enum codes, copy built-in values as is
		for k, a := range srcCell.Attr {

			if a.IsNull || a.Value == nil {
				dstCell.Attr[k] = attrValue{IsNull: true, Value: nil}
			} else {
				if fa[k] == nil {
					dstCell.Attr[k].Value = a.Value // converter not defined for built-in types: use value as is
				} else {
					if s, e := fa[k](a.Value); e != nil { // use attribute value converter
						return nil, err
					} else {
						dstCell.Attr[k].Value = s
					}
				}
			}
		}

		return dstCell, nil // converted OK
	}

	return cvt, nil
}

// CodeToIdCell return converter
// from parameter cell of codes: (entity key, attributes as built-in values or enum codes)
// to cell of ids: (entity key, attributes as built-in values or enum id's)
//
// If attribute type is enum based then attribute enum codes converted to enum ids.
// If attribute type is built-in (bool, int, float) then attribute value converted from string to attribute type.
func (cellCvt *CellMicroConverter) CodeToIdCell(modelDef *ModelMeta, _ string) (func(interface{}) (interface{}, error), error) {

	// find entity metadata by entity name and attributes by generation Hid
	_, attrs, err := cellCvt.entityAttrs()
	if err != nil {
		return nil, err
	}
	nAttr := len(attrs)

	// convert attributes value from string if attribute is enum based: return enum id by code
	// do not convert built-in attribute type, converter function is nil
	fa := make([]func(src string) (interface{}, error), nAttr)

	for k, ea := range attrs {

		if ea.typeOf.IsBuiltIn() {

			fa[k] = nil // built-in attribute type: do not convert

		} else { // enum based attribute type: find and return enum code by enum id

			msgName := cellCvt.Name + "." + ea.Name // for error message, ex: Person.Income

			f, err := ea.typeOf.itemCodeToId(msgName, false)
			if err != nil {
				return nil, err
			}

			fa[k] = func(src string) (interface{}, error) { // convereter return enum id by code
				return f(src)
			}

		}
	}

	// create cell converter
	cvt := func(src interface{}) (interface{}, error) {

		srcCell, ok := src.(CellCodeMicro)
		if !ok {
			return nil, errors.New("invalid type, expected: parameter code cell (internal error): " + cellCvt.Name)
		}
		if len(srcCell.Attr) != nAttr {
			return nil, errors.New("invalid number of attributes, expected: " + strconv.Itoa(nAttr) + ": " + cellCvt.Name)
		}

		dstCell := CellCodeMicro{
			Key:  srcCell.Key,
			Attr: make([]attrValue, nAttr),
		}

		// convert attributes enum codes to enum id's or built-in values from string
		for k, a := range srcCell.Attr {

			if a.IsNull || a.Value == nil {
				dstCell.Attr[k] = attrValue{IsNull: true, Value: nil}
			} else {
				if fa[k] == nil {
					dstCell.Attr[k].Value = a.Value // converter not defined for built-in types: use value as is
				} else {

					sv, ok := a.Value.(string)
					if !ok {
						return nil, errors.New("invalid microdata attribute value type, string expected: " + cellCvt.Name)
					}

					v, e := fa[k](sv) // use attribute value converter from string
					if e != nil {
						return nil, e
					}
					dstCell.Attr[k].Value = v
				}
			}
		}

		return dstCell, nil // converted OK
	}

	return cvt, nil
}

// return entity metadata by entity name
func (cellCvt *CellEntityConverter) entityByName() (*EntityMeta, error) {

	if cellCvt.theEntity != nil {
		return cellCvt.theEntity, nil // entity already found
	}

	// validate parameters
	if cellCvt.ModelDef == nil {
		return nil, errors.New("invalid (empty) model metadata, look like model not found")
	}
	if cellCvt.Name == "" {
		return nil, errors.New("invalid (empty) entity name")
	}

	// find entity index by name
	idx, ok := cellCvt.ModelDef.EntityByName(cellCvt.Name)
	if !ok {
		return nil, errors.New("entity not found: " + cellCvt.Name)
	}
	cellCvt.theEntity = &cellCvt.ModelDef.Entity[idx]

	return cellCvt.theEntity, nil
}

// return entity metadata by entity name and entity generation attributes by generation Hid
func (cellCvt *CellEntityConverter) entityAttrs() (*EntityMeta, []EntityAttrRow, error) {

	if cellCvt.theEntity != nil && len(cellCvt.theAttrs) > 0 {
		return cellCvt.theEntity, cellCvt.theAttrs, nil // attributes already found
	}
	// validate parameters
	if cellCvt.EntityGen == nil {
		return nil, []EntityAttrRow{}, errors.New("invalid (empty) entity generation metadata, look like model run not found or there is no microdata: " + cellCvt.Name)
	}

	// find entity by name and entity generation by Hid
	ent, err := cellCvt.entityByName()
	if err != nil {
		return nil, []EntityAttrRow{}, err
	}

	// collect generation attribues
	attrs := make([]EntityAttrRow, len(cellCvt.EntityGen.GenAttr))

	for k, ga := range cellCvt.EntityGen.GenAttr {

		aIdx, ok := ent.AttrByKey(ga.AttrId)
		if !ok {
			return nil, []EntityAttrRow{}, errors.New("entity attribute not found by id: " + strconv.Itoa(ga.AttrId) + " " + cellCvt.Name)
		}
		attrs[k] = ent.Attr[aIdx]
	}
	cellCvt.theAttrs = attrs

	return ent, cellCvt.theAttrs, nil
}

// return map of entity generation attribute id to language-specific label.
// Label is an attribute description in specific language.
// If language code or description is empty then label is attribute name
func (cellCvt *CellMicroLocaleConverter) attrLabel() (map[int]string, error) {

	if cellCvt.theAttrLabels != nil && len(cellCvt.theAttrLabels) > 0 {
		return cellCvt.theAttrLabels, nil // attribute labels are already found
	}

	// find entity metadata by entity name and attributes by generation Hid
	ent, attrs, err := cellCvt.entityAttrs()
	if err != nil {
		return nil, err
	}
	labelMap := make(map[int]string, len(attrs))

	// add attribute name into map as default label
	for j := range attrs {
		labelMap[attrs[j].AttrId] = attrs[j].Name
	}

	// replace labels: use description where exists for specified language
	if cellCvt.Lang != "" {
		for j := range cellCvt.AttrTxt {
			if cellCvt.AttrTxt[j].ModelId == ent.ModelId && cellCvt.AttrTxt[j].EntityId == ent.EntityId && cellCvt.AttrTxt[j].LangCode == cellCvt.Lang {
				if _, ok := labelMap[cellCvt.AttrTxt[j].AttrId]; ok {
					labelMap[cellCvt.AttrTxt[j].AttrId] = cellCvt.AttrTxt[j].Descr
				}
			}
		}
	}
	cellCvt.theAttrLabels = labelMap

	return cellCvt.theAttrLabels, nil
}
