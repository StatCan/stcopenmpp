// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"crypto/md5"
	"errors"
	"fmt"
	"strconv"
)

// updateInternals language metadata internal members.
// It must be called after restoring from json.
func (meta *LangMeta) updateInternals() {

	meta.idIndex = make(map[int]int, len(meta.Lang))
	meta.codeIndex = make(map[string]int, len(meta.Lang))

	for k := range meta.Lang {
		meta.idIndex[meta.Lang[k].langId] = k     // index of lang_id in result []language slice
		meta.codeIndex[meta.Lang[k].LangCode] = k // index of lang_code in result []language slice
	}
}

// updateParameterColumnNames sets internal db column names for parameter dimensions: dim0, dim1
func (param *ParamMeta) updateParameterColumnNames() {

	for i := range param.Dim {
		param.Dim[i].colName = "dim" + strconv.Itoa(param.Dim[i].DimId)
	}
}

// updateTableColumnNames sets internal db column names for output table dimensions, expressions and accumulators.
// For example: dim0, dim1, acc0, expr1.
// Accumulator db column name for native accumulator is acc1 = "acc" + id,
// for derived accumulators it is the same as expression db column name: expr0 = "expr" + expression id.
func (table *TableMeta) updateTableColumnNames() {

	// dimensions column name: dim0
	for i := range table.Dim {
		table.Dim[i].colName = "dim" + strconv.Itoa(table.Dim[i].DimId)
	}

	// expressions column name: expr0
	for i := range table.Expr {
		table.Expr[i].colName = "expr" + strconv.Itoa(table.Expr[i].ExprId)
	}

	// accumulators column name: acc1 for native and expr0 for derived
	for i := range table.Acc {

		if !table.Acc[i].IsDerived {
			table.Acc[i].colName = "acc" + strconv.Itoa(table.Acc[i].AccId) // native accumulator
		} else {

			// find expresiion with the same name as derived accumulator name
			table.Acc[i].colName = "expr" + strconv.Itoa(table.Acc[i].AccId) // by default make unique name using accumulator id: expr8
			for j := range table.Expr {
				if table.Acc[i].Name == table.Expr[j].Name {
					table.Acc[i].colName = table.Expr[j].colName
					break
				}
			}
		}
	}
}

// updateEntityColumnNames sets internal db column names for entity attributes: attr0, attr1
func (entity *EntityMeta) updateEntityColumnNames() {

	for i := range entity.Attr {
		entity.Attr[i].colName = "attr" + strconv.Itoa(entity.Attr[i].AttrId)
	}
}

// updateInternals model metadata internal members.
// It must be called after restoring from json.
// It does recalculate digest of type, parameter, output table, entity, model if digest is "" empty.
func (meta *ModelMeta) updateInternals() error {

	hMd5 := md5.New()
	hMd5Import := md5.New()
	isDigestUpdated := false

	// update type digest, if it is empty
	for idx := range meta.Type {

		if !meta.Type[idx].IsRange {
			meta.Type[idx].sizeOf = len(meta.Type[idx].Enum)
		} else {
			meta.Type[idx].sizeOf = 1 + meta.Type[idx].MaxEnumId - meta.Type[idx].MinEnumId
		}

		if meta.Type[idx].Digest != "" { // digest already defined, skip
			continue
		}

		// for built-in types use _name_ as digest, ie: _int_ or _Time_
		if meta.Type[idx].IsBuiltIn() {
			meta.Type[idx].Digest = "_" + meta.Type[idx].Name + "_"
			continue
		}
		// else: model-specific type with empty "" digest

		// digest type header
		hMd5.Reset()
		_, err := hMd5.Write([]byte("type_name,dic_id,total_enum_id\n"))
		if err != nil {
			return err
		}
		_, err = hMd5.Write([]byte(
			meta.Type[idx].Name + "," + strconv.Itoa(meta.Type[idx].DicId) + "," + strconv.Itoa(meta.Type[idx].TotalEnumId) + "\n"))
		if err != nil {
			return err
		}

		// digest type enums
		_, err = hMd5.Write([]byte("enum_id,enum_name\n"))
		if err != nil {
			return err
		}
		if !meta.Type[idx].IsRange {

			for k := range meta.Type[idx].Enum {
				_, err := hMd5.Write([]byte(
					strconv.Itoa(meta.Type[idx].Enum[k].EnumId) + "," + meta.Type[idx].Enum[k].Name + "\n"))
				if err != nil {
					return err
				}
			}
		} else {

			for k := 0; k < meta.Type[idx].sizeOf; k++ {
				sId := strconv.Itoa(k + meta.Type[idx].MinEnumId)
				_, err := hMd5.Write(
					[]byte(sId + "," + sId + "\n"))
				if err != nil {
					return err
				}
			}
		}

		meta.Type[idx].Digest = fmt.Sprintf("%x", hMd5.Sum(nil)) // set type digest string
		isDigestUpdated = true
	}

	// update parameter type and size (row count for all dimensions)
	// update parameter dimensions: type and size (count of all enums)
	// update parameter digest and import digest, if digest is empty
	for idx := range meta.Param {

		// update parameter type
		k, ok := meta.TypeByKey(meta.Param[idx].TypeId)
		if !ok {
			return errors.New("type " + strconv.Itoa(meta.Param[idx].TypeId) + " not found for " + meta.Param[idx].Name)
		}
		meta.Param[idx].typeOf = &meta.Type[k]

		if meta.Param[idx].Rank != len(meta.Param[idx].Dim) {
			return errors.New("incorrect rank of parameter " + meta.Param[idx].Name)
		}

		// update parameter size: row count for all dimensions
		// update parameter dimensions: type and size (count of all enums)
		meta.Param[idx].sizeOf = 1
		for i := range meta.Param[idx].Dim {

			j, ok := meta.TypeByKey(meta.Param[idx].Dim[i].TypeId)
			if !ok {
				return errors.New("type " + strconv.Itoa(meta.Param[idx].Dim[i].TypeId) + " not found for " + meta.Param[idx].Name)
			}
			meta.Param[idx].Dim[i].typeOf = &meta.Type[j]
			meta.Param[idx].Dim[i].sizeOf = meta.Param[idx].Dim[i].typeOf.sizeOf

			if meta.Param[idx].Dim[i].sizeOf > 0 {
				meta.Param[idx].sizeOf *= meta.Param[idx].Dim[i].sizeOf
			}

		}
		meta.Param[idx].updateParameterColumnNames() // set dimensions db column name

		// update parameter digest and import digest, if digest is empty
		if meta.Param[idx].Digest == "" {

			// digest parameter header: name, rank, value type digest
			hMd5.Reset()
			_, err := hMd5.Write([]byte("parameter_name,parameter_rank,type_digest\n"))
			if err != nil {
				return err
			}
			_, err = hMd5.Write([]byte(meta.Param[idx].Name + "," + strconv.Itoa(meta.Param[idx].Rank) + "," + meta.Param[idx].typeOf.Digest + "\n"))
			if err != nil {
				return err
			}

			// import digest same as full parameter digest but does not include parameter name
			hMd5Import.Reset()
			_, err = hMd5Import.Write([]byte("parameter_rank,type_digest\n"))
			if err != nil {
				return err
			}
			_, err = hMd5Import.Write([]byte(strconv.Itoa(meta.Param[idx].Rank) + "," + meta.Param[idx].typeOf.Digest + "\n"))
			if err != nil {
				return err
			}

			// digest parameter dimensions: id, name, size and dimension type digest
			_, err = hMd5.Write([]byte("dim_id,dim_name,dim_size,type_digest\n"))
			if err != nil {
				return err
			}
			_, err = hMd5Import.Write([]byte("dim_id,dim_name,dim_size,type_digest\n"))
			if err != nil {
				return err
			}

			for k := range meta.Param[idx].Dim {
				bt := []byte(
					strconv.Itoa(meta.Param[idx].Dim[k].DimId) + "," +
						meta.Param[idx].Dim[k].Name + "," +
						strconv.Itoa(meta.Param[idx].Dim[k].sizeOf) + "," +
						meta.Param[idx].Dim[k].typeOf.Digest + "\n")

				_, err = hMd5.Write(bt)
				if err != nil {
					return err
				}
				_, err = hMd5Import.Write(bt)
				if err != nil {
					return err
				}
			}

			meta.Param[idx].Digest = fmt.Sprintf("%x", hMd5.Sum(nil))
			meta.Param[idx].ImportDigest = fmt.Sprintf("%x", hMd5Import.Sum(nil))
			isDigestUpdated = true
		}
	}

	// update output table size (row count for all dimensions)
	// update output table dimensions: type and size (count of all enums)
	// update output table digest and import digest, if digest is empty
	for idx := range meta.Table {

		if meta.Table[idx].Rank != len(meta.Table[idx].Dim) {
			return errors.New("incorrect rank of output table " + meta.Table[idx].Name)
		}

		// update output table size (row count for all dimensions)
		// update output table dimensions: type and size (count of all enums)
		meta.Table[idx].sizeOf = 1
		for i := range meta.Table[idx].Dim {

			j, ok := meta.TypeByKey(meta.Table[idx].Dim[i].TypeId)
			if !ok {
				return errors.New("type " + strconv.Itoa(meta.Table[idx].Dim[i].TypeId) + " not found for " + meta.Table[idx].Name)
			}
			meta.Table[idx].Dim[i].typeOf = &meta.Type[j]

			if meta.Table[idx].Dim[i].DimSize > 0 {
				meta.Table[idx].sizeOf *= meta.Table[idx].Dim[i].DimSize
			}
		}
		meta.Table[idx].updateTableColumnNames() // set db column name for dimensions, expressions and accumulators

		// update output table digest and import digest, if digest is empty
		if meta.Table[idx].Digest == "" {

			// digest output table header: name, rank
			hMd5.Reset()
			_, err := hMd5.Write([]byte("table_name,table_rank\n"))
			if err != nil {
				return err
			}
			_, err = hMd5.Write([]byte(meta.Table[idx].Name + "," + strconv.Itoa(meta.Table[idx].Rank) + "\n"))
			if err != nil {
				return err
			}

			// table import digest same as parameter import digest
			hMd5Import.Reset()
			_, err = hMd5Import.Write([]byte("parameter_rank,type_digest\n"))
			if err != nil {
				return err
			}
			_, err = hMd5Import.Write([]byte(strconv.Itoa(meta.Table[idx].Rank) + ",_double_\n"))
			if err != nil {
				return err
			}

			// digest output table dimensions: id, name, size and dimension type digest
			_, err = hMd5.Write([]byte("dim_id,dim_name,dim_size,type_digest\n"))
			if err != nil {
				return err
			}
			_, err = hMd5Import.Write([]byte("dim_id,dim_name,dim_size,type_digest\n"))
			if err != nil {
				return err
			}

			for k := range meta.Table[idx].Dim {
				bt := []byte(
					strconv.Itoa(meta.Table[idx].Dim[k].DimId) + "," +
						meta.Table[idx].Dim[k].Name + "," +
						strconv.Itoa(meta.Table[idx].Dim[k].DimSize) + "," +
						meta.Table[idx].Dim[k].typeOf.Digest + "\n")

				_, err = hMd5.Write(bt)
				if err != nil {
					return err
				}
				_, err = hMd5Import.Write(bt)
				if err != nil {
					return err
				}
			}
			meta.Table[idx].ImportDigest = fmt.Sprintf("%x", hMd5Import.Sum(nil)) // done with import digest

			// digest output table accumulators: id, name, expression
			_, err = hMd5.Write([]byte("acc_id,acc_name,acc_src\n"))
			if err != nil {
				return err
			}
			for k := range meta.Table[idx].Acc {
				_, err := hMd5.Write([]byte(
					strconv.Itoa(meta.Table[idx].Acc[k].AccId) + "," + meta.Table[idx].Acc[k].Name + "," + meta.Table[idx].Acc[k].SrcAcc + "\n"))
				if err != nil {
					return err
				}
			}

			// digest output table expressions: id, name, source expression
			_, err = hMd5.Write([]byte("expr_id,expr_name,expr_src\n"))
			if err != nil {
				return err
			}
			for k := range meta.Table[idx].Expr {
				_, err := hMd5.Write([]byte(
					strconv.Itoa(meta.Table[idx].Expr[k].ExprId) + "," + meta.Table[idx].Expr[k].Name + "," + meta.Table[idx].Expr[k].SrcExpr + "\n"))
				if err != nil {
					return err
				}
			}

			meta.Table[idx].Digest = fmt.Sprintf("%x", hMd5.Sum(nil)) // set output table digest string
			isDigestUpdated = true
		}
	}

	// update model digest if it is "" empty, it does not include entities digest
	if isDigestUpdated || meta.Model.Digest == "" {

		// digest model header: model name, type, version
		hMd5.Reset()
		_, err := hMd5.Write([]byte("model_name,model_type,model_ver\n"))
		if err != nil {
			return err
		}
		_, err = hMd5.Write([]byte(
			meta.Model.Name + "," + strconv.Itoa(meta.Model.Type) + "," + meta.Model.Version + "\n"))
		if err != nil {
			return err
		}

		// add digests of all model types
		_, err = hMd5.Write([]byte("type_digest\n"))
		if err != nil {
			return err
		}
		for k := range meta.Type {
			_, err := hMd5.Write([]byte(meta.Type[k].Digest + "\n"))
			if err != nil {
				return err
			}
		}

		// add digests of all model parameters
		_, err = hMd5.Write([]byte("parameter_digest\n"))
		if err != nil {
			return err
		}
		for k := range meta.Param {
			_, err := hMd5.Write([]byte(meta.Param[k].Digest + "\n"))
			if err != nil {
				return err
			}
		}

		// add digests of all model output tables
		_, err = hMd5.Write([]byte("table_digest\n"))
		if err != nil {
			return err
		}
		for k := range meta.Table {
			_, err := hMd5.Write([]byte(meta.Table[k].Digest + "\n"))
			if err != nil {
				return err
			}
		}

		meta.Model.Digest = fmt.Sprintf("%x", hMd5.Sum(nil)) // set model digest string
	}

	// update entity attributes type
	// update entity digest, if digest is empty
	for idx := range meta.Entity {

		// update entity attributes type
		for i := range meta.Entity[idx].Attr {

			j, ok := meta.TypeByKey(meta.Entity[idx].Attr[i].TypeId)
			if !ok {
				return errors.New("type " + strconv.Itoa(meta.Entity[idx].Attr[i].TypeId) + " not found for " + meta.Entity[idx].Name)
			}
			meta.Entity[idx].Attr[i].typeOf = &meta.Type[j]
		}
		meta.Entity[idx].updateEntityColumnNames() // set attributes db column name

		// update entity digest, if digest is empty
		if meta.Entity[idx].Digest == "" {

			// make digest header as entity name
			hMd5.Reset()
			_, err := hMd5.Write([]byte("entity_name\n"))
			if err != nil {
				return err
			}
			_, err = hMd5.Write([]byte(meta.Entity[idx].Name + "\n"))
			if err != nil {
				return err
			}

			// digest entity attributes: name and attribute type digest
			_, err = hMd5.Write([]byte("attr_name,type_digest\n"))
			if err != nil {
				return err
			}

			for k := range meta.Entity[idx].Attr {
				bt := []byte(meta.Entity[idx].Attr[k].Name + "," + meta.Entity[idx].Attr[k].typeOf.Digest + "\n")

				_, err = hMd5.Write(bt)
				if err != nil {
					return err
				}
			}

			meta.Entity[idx].Digest = fmt.Sprintf("%x", hMd5.Sum(nil))
		}
	}

	return nil
}
