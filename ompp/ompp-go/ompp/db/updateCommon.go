// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"errors"
	"fmt"
	"hash"
	"hash/crc32"
	"strconv"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
)

// return prefix and suffix for parameter value db tables or output table value db tables or entity generation db tables.
// For example run parameter db table name ageSex_p2abc4def is paramNameAsPrefix + _p + crc32Suffix.
// Prefix based on parameter name or output table name,
// suffix is 32 chars of md5 or 8 chars of crc32
// There is extra 2 chars: _p, _w, _v, _a, _g in table name between prefix and suffix.
func makeDbTablePrefixSuffix(name string, digest string) (string, string) {

	// if max size of db table name is too short then use crc32(md5) digest
	// isCrc32Name := maxTableNameSize() < 50
	isCrc32Name := true // 2016-08-17: always use short crc32 name suffix

	dbSuffixSize := 32
	if isCrc32Name {
		dbSuffixSize = 8
	}

	dbPrefixSize := maxTableNameSize - (2 + dbSuffixSize)
	if dbPrefixSize < 2 {
		dbPrefixSize = 2
	}

	// make prefix part of db table name by using only [A-Z,a-z,0-9] and _ underscore
	// also shorten source name, ie: ageSexProvince => ageSexPr
	prefix := helper.ToAlphaNumeric(name)
	if len(prefix) > dbPrefixSize {
		prefix = prefix[:dbPrefixSize]
	}

	// make unique suffix of db table name by using digest or crc32(digest)
	suffix := digest
	if isCrc32Name {
		hCrc32 := crc32.NewIEEE()
		hCrc32.Write([]byte(digest))
		suffix = fmt.Sprintf("%x", hCrc32.Sum(nil))
	}

	return prefix, suffix
}

// cvtToSqlDbValue return converter from source value into sql db value, for example parameters value or microdata attributes value.
// converter does type validation,
// for non built-it types validate enum id presense in enum list.
// only float values can be NULL, for any other types NULL values are rejected with error.
func cvtToSqlDbValue(isNullable bool, typeOf *TypeMeta, msgName string) func(bool, interface{}) (interface{}, error) {

	// float type value: check if isNull flag, validate and convert type
	if typeOf.IsFloat() {
		return func(isNull bool, src interface{}) (interface{}, error) {
			if isNull && !isNullable || !isNull && src == nil {
				return nil, errors.New("invalid value, it cannot be NULL " + msgName)
			}
			if isNull {
				return sql.NullFloat64{Float64: 0.0, Valid: false}, nil
			}
			switch v := src.(type) {
			case float64:
				return sql.NullFloat64{Float64: v, Valid: !isNull}, nil
			case float32:
				return sql.NullFloat64{Float64: float64(v), Valid: !isNull}, nil
			case int:
				return sql.NullFloat64{Float64: float64(v), Valid: !isNull}, nil
			case uint:
				return sql.NullFloat64{Float64: float64(v), Valid: !isNull}, nil
			case int64:
				return sql.NullFloat64{Float64: float64(v), Valid: !isNull}, nil
			case uint64:
				return sql.NullFloat64{Float64: float64(v), Valid: !isNull}, nil
			case int32:
				return sql.NullFloat64{Float64: float64(v), Valid: !isNull}, nil
			case uint32:
				return sql.NullFloat64{Float64: float64(v), Valid: !isNull}, nil
			case int16:
				return sql.NullFloat64{Float64: float64(v), Valid: !isNull}, nil
			case uint16:
				return sql.NullFloat64{Float64: float64(v), Valid: !isNull}, nil
			case int8:
				return sql.NullFloat64{Float64: float64(v), Valid: !isNull}, nil
			case uint8:
				return sql.NullFloat64{Float64: float64(v), Valid: !isNull}, nil
			}
			return nil, errors.New("invalid value type, expected: float or double " + msgName)
		}
	}

	// value of integer: check value is not null and validate type
	if typeOf.IsInt() {
		return func(isNull bool, src interface{}) (interface{}, error) {
			if isNull || src == nil {
				return nil, errors.New("invalid value, it cannot be NULL " + msgName)
			}
			switch src.(type) {
			case int:
				return src, nil
			case uint:
				return src, nil
			case int64:
				return src, nil
			case uint64:
				return src, nil
			case int32:
				return src, nil
			case uint32:
				return src, nil
			case int16:
				return src, nil
			case uint16:
				return src, nil
			case int8:
				return src, nil
			case uint8:
				return src, nil
			case float64: // from json or oracle (often)
				return src, nil
			case float32: // from json or oracle (unlikely)
				return src, nil
			}
			return nil, errors.New("invalid value type, expected: integer " + msgName)
		}
	}

	// value of string type: check value is not null and validate type
	if typeOf.IsString() {
		return func(isNull bool, src interface{}) (interface{}, error) {
			if isNull || src == nil {
				return nil, errors.New("invalid value, it cannot be NULL " + msgName)
			}
			switch src.(type) {
			case string:
				return src, nil
			}
			return nil, errors.New("invalid value type, expected: string " + msgName)
		}
	}

	// boolean is a special case because not all drivers correctly handle conversion to smallint
	if typeOf.IsBool() {
		return func(isNull bool, src interface{}) (interface{}, error) {
			if isNull || src == nil {
				return nil, errors.New("invalid value, it cannot be NULL " + msgName)
			}
			if is, ok := src.(bool); ok && is {
				return 1, nil
			}
			return 0, nil
		}
	}

	// enum-based type: enum id must be in enum list
	return func(isNull bool, src interface{}) (interface{}, error) {

		if isNull || src == nil {
			return nil, errors.New("invalid value, it cannot be NULL " + msgName)
		}

		// validate type and convert to int
		iv, ok := helper.ToIntValue(src)
		if !ok {
			return nil, errors.New("invalid value type, expected: integer enum id " + msgName)
		}

		// validate enum id: it must be in enum list or in the range
		if !typeOf.IsRange {
			for j := range typeOf.Enum {
				if iv == typeOf.Enum[j].EnumId {
					return iv, nil
				}
			}
		} else {
			if typeOf.MinEnumId <= iv && iv <= typeOf.MaxEnumId {
				return iv, nil
			}
		}
		return nil, errors.New("invalid value type, enum id not found: " + strconv.Itoa(iv) + " " + msgName)
	}
}

// digestIntKeysCellsFrom append header to hash and return closure
// to add hash of cells (parameter values, accumulators or expressions) to digest.
// It is also return reference to bool flag to indicate all source rows are ordered by primary key.
// If there is no order by primery key then digest calculation is incorrect.
// It is a hash of text values identical to csv file hash, for example:
//
//	acc_id,sub_id,dim0,dim1,acc_value\n
//	0,1,0,0,1234.5678\n
func digestIntKeysCellsFrom(hSum hash.Hash, modelDef *ModelMeta, name string, csvCvt CsvIntKeysConverter) (func(interface{}) error, *bool, error) {

	isOrderBy := true // return true if rows ordered by primary key

	// append header, like: acc_id,sub_id,dim0,dim1,acc_value\n
	cs, err := csvCvt.CsvHeader()
	if err != nil {
		return nil, &isOrderBy, err
	}
	for k := range cs {
		if k != 0 {
			if _, err = hSum.Write([]byte(",")); err != nil {
				return nil, &isOrderBy, err
			}
		}
		if _, err = hSum.Write([]byte(cs[k])); err != nil {
			return nil, &isOrderBy, err
		}
	}
	if _, err = hSum.Write([]byte("\n")); err != nil {
		return nil, &isOrderBy, err
	}

	// rows must be order by primary key, e.g.: acc_id, sub_id, dim0, dim1
	// for correct digest calculation
	// store previous row order by columns to check source rows order
	nOrder := len(cs) - 1
	prevKey := make([]int, nOrder)
	nowKey := make([]int, nOrder)
	isFirst := true

	keyCvt, err := csvCvt.KeyIds(name)
	if err != nil {
		return nil, &isOrderBy, err
	}

	// for each row append dimensions and value to digest
	cvt, err := csvCvt.ToCsvIdRow() // converter from cell id's to csv id's row []string
	if err != nil {
		return nil, &isOrderBy, err
	}

	digestNextRow := func(src interface{}) error {

		// check row order by: if previous row key is less than current ror key
		if nOrder > 0 && isOrderBy {
			if isFirst {
				if e := keyCvt(src, prevKey); e != nil {
					return e
				}
				isFirst = false
			} else {
				if e := keyCvt(src, nowKey); e != nil {
					return e
				}
				for k := 0; isOrderBy && k < nOrder; k++ {
					isOrderBy = nowKey[k] >= prevKey[k]
					nowKey[k] = prevKey[k]
				}
			}
		}

		// convert to strings
		// ignore is not empty flag return value, digest should include all rows
		if _, err := cvt(src, cs); err != nil {
			return err
		}

		// append to digest
		for k := range cs {
			if k != 0 {
				if _, err = hSum.Write([]byte(",")); err != nil {
					return err
				}
			}
			if _, err = hSum.Write([]byte(cs[k])); err != nil {
				return err
			}
		}
		if _, err = hSum.Write([]byte("\n")); err != nil {
			return err
		}

		return nil
	}

	return digestNextRow, &isOrderBy, nil
}

// digestMicrodataCellsFrom append header to hash and return closure
// to add hash of entity microdata cells (microdata key, attributes value) to digest.
// It is a hash of text values identical to csv file hash, for example:
//
//	key,Age,Sex,Income\n
//	1234,45,M,567.89\n
func digestMicrodataCellsFrom(hSum hash.Hash, modelDef *ModelMeta, rowCount *int, csvCvt CsvConverter) (func(interface{}) error, error) {

	// append header, like: key,Age,Sex,Income\n
	cs, err := csvCvt.CsvHeader()
	if err != nil {
		return nil, err
	}

	for k := range cs {
		if k != 0 {
			if _, err = hSum.Write([]byte(",")); err != nil {
				return nil, err
			}
		}
		if _, err = hSum.Write([]byte(cs[k])); err != nil {
			return nil, err
		}
	}
	if _, err = hSum.Write([]byte("\n")); err != nil {
		return nil, err
	}

	// for each row append entity key and attributes value to digest
	cvt, err := csvCvt.ToCsvIdRow() // converter from cell id's to csv row []string
	if err != nil {
		return nil, err
	}

	digestNextRow := func(src interface{}) error {

		// convert to strings
		// ignore is not empty flag return value, digest should include all rows
		if _, err := cvt(src, cs); err != nil {
			return err
		}

		// append to digest
		for k := range cs {
			if k != 0 {
				if _, err = hSum.Write([]byte(",")); err != nil {
					return err
				}
			}
			if _, err = hSum.Write([]byte(cs[k])); err != nil {
				return err
			}
		}
		if _, err = hSum.Write([]byte("\n")); err != nil {
			return err
		}
		*rowCount += 1 // count rows

		return nil
	}

	return digestNextRow, nil
}
