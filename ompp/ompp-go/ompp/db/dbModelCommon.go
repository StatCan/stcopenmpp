// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
)

// Clone return deep copy of source model metadata
func (src *ModelMeta) Clone() (*ModelMeta, error) {

	var dst ModelMeta

	if err := helper.DeepCopy(src, &dst); err != nil {
		return nil, err
	}
	if err := dst.updateInternals(); err != nil {
		return nil, err
	}
	return &dst, nil
}

// Clone return deep copy of source model text metadata
func (src *ModelTxtMeta) Clone() (*ModelTxtMeta, error) {

	var dst ModelTxtMeta

	if err := helper.DeepCopy(src, &dst); err != nil {
		return nil, err
	}
	return &dst, nil
}

// Clone return deep copy of source language metadata
func (src *LangMeta) Clone() (*LangMeta, error) {

	dst := &LangMeta{}

	if err := helper.DeepCopy(src, dst); err != nil {
		return nil, err
	}

	// copy language id: it is non-public and update internals
	for k := range src.Lang {
		dst.Lang[k].langId = src.Lang[k].langId
	}
	dst.updateInternals() // update internals

	return dst, nil
}

// Clone return deep copy of source model words
func (src *ModelWordMeta) Clone() (*ModelWordMeta, error) {

	var dst ModelWordMeta

	if err := helper.DeepCopy(src, &dst); err != nil {
		return nil, err
	}
	return &dst, nil
}

// FromJson restore model metadata list from json string bytes
func (dst *ModelMeta) FromJson(srcJson []byte) (bool, error) {

	isExist, err := helper.FromJson(srcJson, dst)
	if err != nil {
		return false, err
	}
	if !isExist {
		return false, nil
	}
	dst.updateInternals()
	return true, nil
}

// FromJson restore language list from json string bytes
func (dst *LangMeta) FromJson(srcJson []byte) (bool, error) {

	isExist, err := helper.FromJson(srcJson, dst)
	if err != nil {
		return false, err
	}
	if !isExist {
		return false, nil
	}
	dst.updateInternals()
	return true, nil
}

// IdByCode return language id by language code or first language if code not found
func (langDef *LangMeta) IdByCode(langCode string) (int, bool) {
	if i, ok := langDef.codeIndex[langCode]; ok {
		return langDef.Lang[i].langId, true
	}
	return langDef.Lang[0].langId, false
}

// CodeIdId return language code by language id or first language if id not found
func (langDef *LangMeta) CodeById(langId int) (string, bool) {
	if i, ok := langDef.idIndex[langId]; ok {
		return langDef.Lang[i].LangCode, true
	}
	return langDef.Lang[0].LangCode, false
}

// TypeByKey return index of type by key: typeId
func (modelDef *ModelMeta) TypeByKey(typeId int) (int, bool) {

	n := len(modelDef.Type)
	k := sort.Search(n, func(i int) bool {
		return modelDef.Type[i].TypeId >= typeId
	})
	return k, (k >= 0 && k < n && modelDef.Type[k].TypeId == typeId)
}

// return double type index, it sia type of output value table and calculated value
func (modelDef *ModelMeta) TypeOfDouble() (int, bool) {

	for k := range modelDef.Type {
		if modelDef.Type[k].Digest == "_double_" {
			return k, true
		}
	}
	return len(modelDef.Type), false
}

// ParamByKey return index of parameter by key: paramId
func (modelDef *ModelMeta) ParamByKey(paramId int) (int, bool) {

	n := len(modelDef.Param)
	k := sort.Search(n, func(i int) bool {
		return modelDef.Param[i].ParamId >= paramId
	})
	return k, (k >= 0 && k < n && modelDef.Param[k].ParamId == paramId)
}

// ParamByName return index of parameter by name
func (modelDef *ModelMeta) ParamByName(name string) (int, bool) {

	for k := range modelDef.Param {
		if modelDef.Param[k].Name == name {
			return k, true
		}
	}
	return len(modelDef.Param), false
}

// ParamByHid return index of parameter by parameter Hid
func (modelDef *ModelMeta) ParamByHid(paramHid int) (int, bool) {

	for k := range modelDef.Param {
		if modelDef.Param[k].ParamHid == paramHid {
			return k, true
		}
	}
	return len(modelDef.Param), false
}

// ParamHidById return parameter Hid by id or -1 if not found
func (modelDef *ModelMeta) ParamHidById(paramId int) int {

	if k, ok := modelDef.ParamByKey(paramId); ok {
		return modelDef.Param[k].ParamHid
	}
	return -1
}

// DimByKey return index of parameter dimension by key: dimId
func (param *ParamMeta) DimByKey(dimId int) (int, bool) {

	n := len(param.Dim)
	k := sort.Search(n, func(i int) bool {
		return param.Dim[i].DimId >= dimId
	})
	return k, (k >= 0 && k < n && param.Dim[k].DimId == dimId)
}

// OutTableByKey return index of output table by key: tableId
func (modelDef *ModelMeta) OutTableByKey(tableId int) (int, bool) {

	n := len(modelDef.Table)
	k := sort.Search(n, func(i int) bool {
		return modelDef.Table[i].TableId >= tableId
	})
	return k, (k >= 0 && k < n && modelDef.Table[k].TableId == tableId)
}

// OutTableByName return index of output table by name
func (modelDef *ModelMeta) OutTableByName(name string) (int, bool) {

	for k := range modelDef.Table {
		if modelDef.Table[k].Name == name {
			return k, true
		}
	}
	return len(modelDef.Table), false
}

// OutTableByHid return index of output table by table Hid
func (modelDef *ModelMeta) OutTableByHid(tableHid int) (int, bool) {

	for k := range modelDef.Table {
		if modelDef.Table[k].TableHid == tableHid {
			return k, true
		}
	}
	return len(modelDef.Table), false
}

// OutTableHidById return output table Hid by id or -1 if not found
func (modelDef *ModelMeta) OutTableHidById(tableId int) int {

	if k, ok := modelDef.OutTableByKey(tableId); ok {
		return modelDef.Table[k].TableHid
	}
	return -1
}

// DimByKey return index of output table dimension by key: dimId
func (table *TableMeta) DimByKey(dimId int) (int, bool) {

	n := len(table.Dim)
	k := sort.Search(n, func(i int) bool {
		return table.Dim[i].DimId >= dimId
	})
	return k, (k >= 0 && k < n && table.Dim[k].DimId == dimId)
}

// EntityByKey return index of entity by key: entityId
func (modelDef *ModelMeta) EntityByKey(entityId int) (int, bool) {

	n := len(modelDef.Entity)
	k := sort.Search(n, func(i int) bool {
		return modelDef.Entity[i].EntityId >= entityId
	})
	return k, (k >= 0 && k < n && modelDef.Entity[k].EntityId == entityId)
}

// EntityByName return index of entity by name
func (modelDef *ModelMeta) EntityByName(name string) (int, bool) {

	for k := range modelDef.Entity {
		if modelDef.Entity[k].Name == name {
			return k, true
		}
	}
	return len(modelDef.Entity), false
}

// EntityHidById return entity Hid by id or -1 if not found
func (modelDef *ModelMeta) EntityHidById(entityId int) int {

	if k, ok := modelDef.EntityByKey(entityId); ok {
		return modelDef.Entity[k].EntityHid
	}
	return -1
}

// AttrByKey return index of entity attribute by key: attrId
func (entity *EntityMeta) AttrByKey(attrId int) (int, bool) {

	n := len(entity.Attr)
	k := sort.Search(n, func(i int) bool {
		return entity.Attr[i].AttrId >= attrId
	})
	return k, (k >= 0 && k < n && entity.Attr[k].AttrId == attrId)
}

// AttrByName return index of entity attribute by name
func (entity *EntityMeta) AttrByName(name string) (int, bool) {

	for k := range entity.Attr {
		if entity.Attr[k].Name == name {
			return k, true
		}
	}
	return len(entity.Attr), false
}

// EntityGenByEntityId return index of entity generation by model entity id.
// As it is today model do not insert more than one generation for each entity into model run, but there is no such costraint in db schema.
func (run *RunMeta) EntityGenByEntityId(entityId int) (int, bool) {

	for k := range run.EntityGen {
		if run.EntityGen[k].EntityId == entityId {
			return k, true
		}
	}
	return len(run.EntityGen), false
}

// EntityGenByDigest return index of entity generation by generation digest
func (run *RunMeta) EntityGenByDigest(digest string) (int, bool) {

	for k := range run.EntityGen {
		if run.EntityGen[k].GenDigest == digest {
			return k, true
		}
	}
	return len(run.EntityGen), false
}

// IsBool return true if model type is boolean.
func (typeRow *TypeDicRow) IsBool() bool { return strings.ToLower(typeRow.Name) == "bool" }

// IsString return true if model type is string.
func (typeRow *TypeDicRow) IsString() bool { return strings.ToLower(typeRow.Name) == "file" }

// IsFloat return true if model type is any of float family.
func (typeRow *TypeDicRow) IsFloat() bool {
	switch strings.ToLower(typeRow.Name) {
	case "float", "double", "ldouble", "time", "real":
		return true
	}
	return false
}

// IsFloat32 return true if model type is float 32 bit.
func (typeRow *TypeDicRow) IsFloat32() bool {
	return strings.ToLower(typeRow.Name) == "float"
}

// IsInt return true if model type is integer (not float, string or boolean).
// If type is not a built-in then it must be integer enums.
func (typeRow *TypeDicRow) IsInt() bool {
	return typeRow.IsBuiltIn() && !typeRow.IsBool() && !typeRow.IsString() && !typeRow.IsFloat()
}

// IsBuiltIn return true if model type is built-in, ie: int, double, logical.
func (typeRow *TypeDicRow) IsBuiltIn() bool { return typeRow.TypeId <= maxBuiltInTypeId }

// sqlColumnType return sql column type, ie: VARCHAR(255)
func (typeRow *TypeDicRow) sqlColumnType(dbFacet Facet) (string, error) {

	// model specific types: it must be enum
	if !typeRow.IsBuiltIn() {
		return "INT", nil
	}

	// built-in types (ordered as in omc grammar for clarity)
	switch strings.ToLower(typeRow.Name) {

	// C++ ambiguous integral type
	// (in C/C++, the signedness of char is not specified)
	case "char":
		return "SMALLINT", nil

	// C++ signed integral types
	case "schar", "short":
		return "SMALLINT", nil

	// C++ signed integral types
	case "int":
		return "INT", nil

	// C++ signed integral types
	case "long", "llong":
		return dbFacet.bigintType(), nil

	// C++ unsigned integral types (including bool)
	case "bool", "uchar":
		return "SMALLINT", nil

	// C++ unsigned integral types (including bool)
	case "ushort":
		return "INT", nil

	// C++ unsigned integral types (including bool)
	case "uint", "ulong", "ullong":
		return dbFacet.bigintType(), nil

	// C++ floating point types
	case "float", "double", "ldouble":
		return dbFacet.floatType(), nil

	// Changeable numeric types
	case "time", "real":
		return dbFacet.floatType(), nil

	// Changeable numeric types
	case "integer", "counter":
		return "INT", nil

	// path to a file (a string)
	case "file":
		return dbFacet.textType(4096), nil
		// go 1.7 max path:
		// linux:  syscall.PathMax
		// win:    syscall.MAX_PATH
	}

	return "", errors.New("invalid type id: " + strconv.Itoa(typeRow.TypeId))
}

// itemCodeToId return converter from dimension item code to id.
// It is also used for parameter values if parameter type is enum-based.
// If dimension is enum-based then from enum code to enum id or to the total enum id;
// If dimension is simple integer type then parse integer;
// If dimension is boolean then false=>0, true=>1
func (typeOf *TypeMeta) itemCodeToId(msgName string, isTotalEnabled bool) (func(src string) (int, error), error) {

	var cvt func(src string) (int, error)

	switch {
	case !typeOf.IsBuiltIn(): // enum dimension: find enum id by code

		cvt = func(src string) (int, error) {

			if isTotalEnabled && src == TotalEnumCode { // check is it total item
				return typeOf.TotalEnumId, nil
			}
			if !typeOf.IsRange { // enum dimension: find enum id by code

				for j := range typeOf.Enum {
					if src == typeOf.Enum[j].Name {
						return typeOf.Enum[j].EnumId, nil
					}
				}
			} else { // range dimension: item id the same as code

				nId, err := strconv.Atoi(src)
				if err != nil {
					return 0, errors.New("invalid value: " + src + " of: " + msgName)
				}
				if typeOf.MinEnumId <= nId && nId <= typeOf.MaxEnumId {
					return nId, nil
				}
			}
			return 0, errors.New("invalid value: " + src + " of: " + msgName)
		}

	case typeOf.IsBool(): // boolean dimension: false=>0, true=>1

		cvt = func(src string) (int, error) {

			if isTotalEnabled && src == TotalEnumCode { // check is it total item
				return typeOf.TotalEnumId, nil
			}
			// convert boolean enum codes to id's
			is, err := strconv.ParseBool(src)
			if err != nil {
				return 0, errors.New("invalid value: " + src + " of: " + msgName)
			}
			if is {
				return 1, nil
			}
			return 0, nil
		}

	case typeOf.IsInt(): // integer dimension

		cvt = func(src string) (int, error) {
			i, err := strconv.Atoi(src)
			if err != nil {
				return 0, errors.New("invalid value: " + src + " of: " + msgName)
			}
			return i, nil
		}

	default:
		return nil, errors.New("invalid (not supported) type: " + typeOf.Name + " of: " + msgName)
	}

	return cvt, nil
}

// IsRunCompleted return true if run status one of: s=success, x=exit, e=error
func IsRunCompleted(status string) bool {
	return status == DoneRunStatus || status == ExitRunStatus || status == ErrorRunStatus
}

// IsRunCompleted return true if run status one of: s=success, x=exit, e=error, p=progress
func IsRunCompletedOrProgress(status string) bool {
	return status == DoneRunStatus || status == ExitRunStatus || status == ErrorRunStatus || status == ProgressRunStatus
}

// NameOfRunStatus return short name by run run status code: s=success, x=exit, e=error
func NameOfRunStatus(status string) string {
	switch status {
	case InitRunStatus:
		return "init"
	case ProgressRunStatus:
		return "progress"
	case WaitRunStatus:
		return "wait"
	case DoneRunStatus:
		return "success"
	case ExitRunStatus:
		return "exit"
	case ErrorRunStatus:
		return "error"
	case DeleteRunStatus:
		return "delete"
	}
	return "unknown"
}
