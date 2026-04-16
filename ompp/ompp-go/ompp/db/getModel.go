// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"errors"
	"strconv"
)

// GetModelList return list of the models: model_dic table rows.
func GetModelList(dbConn *sql.DB) ([]ModelDicRow, error) {

	var modelRs []ModelDicRow

	err := SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_name, M.model_digest, M.model_type,"+
			" M.model_ver, M.create_dt, L.lang_code"+
			" FROM model_dic M"+
			" INNER JOIN lang_lst L ON (L.lang_id = M.default_lang_id)"+
			" ORDER BY 1",
		func(rows *sql.Rows) error {
			var r ModelDicRow
			if err := rows.Scan(
				&r.ModelId, &r.Name, &r.Digest, &r.Type,
				&r.Version, &r.CreateDateTime, &r.DefaultLangCode); err != nil {
				return err
			}
			modelRs = append(modelRs, r)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return modelRs, nil
}

// GetModelRow return model_dic table row by model id.
func GetModelRow(dbConn *sql.DB, modelId int) (*ModelDicRow, error) {

	// select model_dic row
	var modelRow = ModelDicRow{ModelId: modelId}

	err := SelectFirst(dbConn,
		"SELECT"+
			" M.model_id, M.model_name, M.model_digest, M.model_type,"+
			" M.model_ver, M.create_dt, L.lang_code"+
			" FROM model_dic M"+
			" INNER JOIN lang_lst L ON (L.lang_id = M.default_lang_id)"+
			" WHERE M.model_id = "+strconv.Itoa(modelId)+
			" ORDER BY 1",
		func(row *sql.Row) error {
			return row.Scan(
				&modelRow.ModelId, &modelRow.Name, &modelRow.Digest, &modelRow.Type,
				&modelRow.Version, &modelRow.CreateDateTime, &modelRow.DefaultLangCode)
		})
	switch {
	case err == sql.ErrNoRows:
		return nil, nil // model not found
	case err != nil:
		return nil, err
	}

	return &modelRow, nil
}

// GetModelId return model id if exists.
//
// Model selected by name and/or digest, i.e.: ("modelOne", "abcd20120817160459148")
// if digest is empty then first model with min(model_id) is used
func GetModelId(dbConn *sql.DB, name, digest string) (bool, int, error) {

	// model not found: model name and digest empty
	if name == "" && digest == "" {
		return false, 0, nil
	}

	// select model_id by name and/or digest
	// if digest is empty then first model with min(model_id) is used
	q := "SELECT M.model_id FROM model_dic M"
	if name != "" && digest != "" {
		q += " WHERE M.model_name = " + ToQuoted(name) +
			" AND M.model_digest = " + ToQuoted(digest)
	}
	if name == "" && digest != "" {
		q += " WHERE M.model_digest = " + ToQuoted(digest)
	}
	if name != "" && digest == "" {
		q += " WHERE M.model_name = " + ToQuoted(name) +
			" AND M.model_id = (SELECT MIN(MMD.model_id) FROM model_dic MMD WHERE MMD.model_name = " + ToQuoted(name) + ")"
	}
	q += " ORDER BY 1"

	mId := 0
	err := SelectFirst(dbConn, q,
		func(row *sql.Row) error {
			return row.Scan(&mId)
		})
	switch {
	case err == sql.ErrNoRows:
		return false, 0, nil
	case err != nil:
		return false, 0, err
	}

	return true, mId, nil
}

// GetModel return model metadata: parameters and output tables definition.
//
// Model selected by name and/or digest, i.e.: ("modelOne", "abcd201208171604590148")
// if digest is empty then first model with min(model_id) is used
func GetModel(dbConn *sql.DB, name, digest string) (*ModelMeta, error) {

	if name == "" && digest == "" {
		return nil, errors.New("invalid (empty) model name and model digest")
	}

	// find model id
	isExist, mId, err := GetModelId(dbConn, name, digest)
	if err != nil {
		return nil, err
	}
	if !isExist {
		return nil, errors.New("model " + name + " " + digest + " not found")
	}

	return GetModelById(dbConn, mId)
}

// GetModelById return model metadata: parameters and output tables definition.
//
// Model selected by model id, which expected to be positive.
func GetModelById(dbConn *sql.DB, modelId int) (*ModelMeta, error) {

	// validate parameters
	if modelId <= 0 {
		return nil, errors.New("invalid model id")
	}

	// select model_dic row
	modelRow, err := GetModelRow(dbConn, modelId)
	if err != nil {
		return nil, err
	}
	if modelRow == nil {
		return nil, errors.New("model not found, id: " + strconv.Itoa(modelId))
	}

	return getModel(dbConn, modelRow)
}

// getModel return model metadata by modelRow (model_dic row).
func getModel(dbConn *sql.DB, modelRow *ModelDicRow) (*ModelMeta, error) {

	// select db rows from type_dic join to model_type_dic
	meta := &ModelMeta{
		Model:       *modelRow,
		Type:        []TypeMeta{},
		Param:       []ParamMeta{},
		Table:       []TableMeta{},
		Entity:      []EntityMeta{},
		Group:       []GroupMeta{},
		EntityGroup: []EntityGroupMeta{},
	}
	smId := strconv.Itoa(meta.Model.ModelId)

	err := SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_type_id, H.type_hid, H.type_name, H.type_digest, H.dic_id, H.total_enum_id"+
			" FROM type_dic H"+
			" INNER JOIN model_type_dic M ON (M.type_hid = H.type_hid)"+
			" WHERE M.model_id = "+smId+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r TypeDicRow
			if err := rows.Scan(
				&r.ModelId, &r.TypeId, &r.TypeHid, &r.Name, &r.Digest, &r.DicId, &r.TotalEnumId); err != nil {
				return err
			}
			r.IsRange = r.DicId == rangeDicId
			meta.Type = append(meta.Type, TypeMeta{TypeDicRow: r})
			return nil
		})
	if err != nil {
		return nil, err
	}
	if len(meta.Type) <= 0 {
		return nil, errors.New("no model types found")
	}

	// for each type find min, max and count of enum id's
	nTypeIdx := 0
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_type_id, MIN(E.enum_id), MAX(E.enum_id), COUNT(E.enum_id)"+
			" FROM type_dic H"+
			" INNER JOIN model_type_dic M ON (M.type_hid = H.type_hid)"+
			" INNER JOIN type_enum_lst E ON (E.type_hid = H.type_hid)"+
			" WHERE M.model_id = "+smId+
			" GROUP BY M.model_id, M.model_type_id"+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var modelId, typeId, minE, maxE, countE int
			if err := rows.Scan(
				&modelId, &typeId, &minE, &maxE, &countE); err != nil {
				return err
			}
			for ; nTypeIdx < len(meta.Type); nTypeIdx++ {
				if meta.Type[nTypeIdx].ModelId == modelId && meta.Type[nTypeIdx].TypeId == typeId {
					meta.Type[nTypeIdx].MinEnumId = minE
					meta.Type[nTypeIdx].MaxEnumId = maxE
					meta.Type[nTypeIdx].sizeOf = countE
					break
				}
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	for nTypeIdx = 0; nTypeIdx < len(meta.Type); nTypeIdx++ {
		if !meta.Type[nTypeIdx].IsBuiltIn() && meta.Type[nTypeIdx].sizeOf <= 0 {
			return nil, errors.New("invalid (empty) type, no enums found:" + " " + meta.Type[nTypeIdx].Name + " " + "model:" + " " + modelRow.Name + " " + modelRow.Digest)
		}
	}

	// select db rows from type_enum_lst join to model_type_dic
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_type_id, D.enum_id, D.enum_name"+
			" FROM type_enum_lst D"+
			" INNER JOIN model_type_dic M ON (M.type_hid = D.type_hid)"+
			" WHERE M.model_id = "+smId+
			" ORDER BY 1, 2, 3",
		func(rows *sql.Rows) error {
			var r TypeEnumRow
			if err := rows.Scan(
				&r.ModelId, &r.TypeId, &r.EnumId, &r.Name); err != nil {
				return err
			}

			k, ok := meta.TypeByKey(r.TypeId) // find type master row
			if !ok {
				return errors.New("type " + strconv.Itoa(r.TypeId) + " not found for " + r.Name)
			}

			if meta.Type[k].IsRange { // skip range enums, store range as [min, max] id's
				return nil
			}
			meta.Type[k].Enum = append(meta.Type[k].Enum, r) // this not a range: append enum item
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from parameter_dic join to model_parameter_dic
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_parameter_id, D.parameter_hid, D.parameter_name, D.parameter_digest,"+
			" D.parameter_rank, T.model_type_id, D.is_extendable, M.is_hidden, D.num_cumulated,"+
			" D.db_run_table, D.db_set_table, D.import_digest"+
			" FROM parameter_dic D"+
			" INNER JOIN model_parameter_dic M ON (M.parameter_hid = D.parameter_hid)"+
			" INNER JOIN model_type_dic T ON (T.type_hid = D.type_hid AND T.model_id = M.model_id)"+
			" WHERE M.model_id = "+smId+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r ParamDicRow
			nExt := 0
			nHidden := 0
			if err := rows.Scan(
				&r.ModelId, &r.ParamId, &r.ParamHid, &r.Name, &r.Digest,
				&r.Rank, &r.TypeId, &nExt, &nHidden, &r.NumCumulated,
				&r.DbRunTable, &r.DbSetTable, &r.ImportDigest); err != nil {
				return err
			}
			r.IsExtendable = nExt != 0 // oracle: smallint is float64
			r.IsHidden = nHidden != 0  // oracle: smallint is float64

			k, ok := meta.TypeByKey(r.TypeId) // find parameter type
			if !ok {
				return errors.New("type " + strconv.Itoa(r.TypeId) + " not found for " + r.Name)
			}

			meta.Param = append(meta.Param, ParamMeta{
				ParamDicRow: r, Dim: []ParamDimsRow{}, Import: []ParamImportRow{}, typeOf: &meta.Type[k]})
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from model_parameter_import
	err = SelectRows(dbConn,
		"SELECT"+
			" I.model_id, I.model_parameter_id, I.from_name, I.from_model_name, I.is_sample_dim"+
			" FROM model_parameter_import I"+
			" WHERE I.model_id = "+smId+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r ParamImportRow
			nSample := 0
			if err := rows.Scan(
				&r.ModelId, &r.ParamId, &r.FromName, &r.FromModel, &nSample); err != nil {
				return err
			}
			r.IsSampleDim = nSample != 0 // oracle: smallint is float64

			idx, ok := meta.ParamByKey(r.ParamId) // find parameter row for that dimension
			if !ok {
				return errors.New("parameter " + strconv.Itoa(r.ParamId) + " not found")
			}

			meta.Param[idx].Import = append(meta.Param[idx].Import, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from parameter_dims join to model_parameter_dic
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_parameter_id, D.dim_id, D.dim_name, T.model_type_id"+
			" FROM parameter_dims D"+
			" INNER JOIN model_parameter_dic M ON (M.parameter_hid = D.parameter_hid)"+
			" INNER JOIN model_type_dic T ON (T.type_hid = D.type_hid AND T.model_id = M.model_id)"+
			" WHERE M.model_id = "+smId+
			" ORDER BY 1, 2, 3",
		func(rows *sql.Rows) error {
			var r ParamDimsRow
			if err := rows.Scan(
				&r.ModelId, &r.ParamId, &r.DimId, &r.Name, &r.TypeId); err != nil {
				return err
			}

			idx, ok := meta.ParamByKey(r.ParamId) // find parameter row for that dimension
			if !ok {
				return errors.New("parameter " + strconv.Itoa(r.ParamId) + " not found for " + r.Name)
			}
			k, ok := meta.TypeByKey(r.TypeId) // find parameter type
			if !ok {
				return errors.New("type " + strconv.Itoa(r.TypeId) + " not found for " + r.Name)
			}
			r.typeOf = &meta.Type[k]
			r.sizeOf = r.typeOf.sizeOf

			meta.Param[idx].Dim = append(meta.Param[idx].Dim, r)
			return nil
		})
	if err != nil {
		return nil, err
	}
	// set db column name for parameter dimnesions
	for k := range meta.Param {
		meta.Param[k].updateParameterColumnNames()
	}

	// select db rows from table_dic join to model_table_dic
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_table_id, D.table_hid, D.table_name,"+
			" D.table_digest, D.table_rank, D.is_sparse, M.is_user,"+
			" D.db_expr_table, D.db_acc_table, D.db_acc_all_view, M.expr_dim_pos,"+
			" M.is_hidden, D.import_digest"+
			" FROM table_dic D"+
			" INNER JOIN model_table_dic M ON (M.table_hid = D.table_hid)"+
			" WHERE M.model_id = "+smId+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r TableDicRow
			nSparse := 0
			nUser := 0
			nHide := 0
			if err := rows.Scan(
				&r.ModelId, &r.TableId, &r.TableHid, &r.Name,
				&r.Digest, &r.Rank, &nSparse, &nUser,
				&r.DbExprTable, &r.DbAccTable, &r.DbAccAllView, &r.ExprPos,
				&nHide, &r.ImportDigest); err != nil {
				return err
			}
			r.IsSparse = nSparse != 0 // oracle: smallint is float64
			r.IsUser = nUser != 0     // oracle: smallint is float64
			r.IsHidden = nHide != 0   // oracle: smallint is float64

			meta.Table = append(meta.Table, TableMeta{TableDicRow: r})
			return nil
		})
	if err != nil {
		return nil, err
	}
	if len(meta.Table) <= 0 {
		return nil, errors.New("no model output tables found")
	}

	// select db rows from table_dims join to model_table_dic
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_table_id, D.dim_id, D.dim_name, T.model_type_id, D.is_total, D.dim_size"+
			" FROM table_dims D"+
			" INNER JOIN model_table_dic M ON (M.table_hid = D.table_hid)"+
			" INNER JOIN model_type_dic T ON (T.type_hid = D.type_hid AND T.model_id = M.model_id)"+
			" WHERE M.model_id = "+smId+
			" ORDER BY 1, 2, 3",
		func(rows *sql.Rows) error {
			var r TableDimsRow
			nTotal := 0
			if err := rows.Scan(
				&r.ModelId, &r.TableId, &r.DimId, &r.Name, &r.TypeId, &nTotal, &r.DimSize); err != nil {
				return err
			}
			r.IsTotal = nTotal != 0 // oracle: smallint is float64

			idx, ok := meta.OutTableByKey(r.TableId) // find table row for that dimension
			if !ok {
				return errors.New("output table " + strconv.Itoa(r.TableId) + " not found for " + r.Name)
			}
			k, ok := meta.TypeByKey(r.TypeId) // find parameter type
			if !ok {
				return errors.New("type " + strconv.Itoa(r.TypeId) + " not found for " + r.Name)
			}
			r.typeOf = &meta.Type[k]

			meta.Table[idx].Dim = append(meta.Table[idx].Dim, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from table_acc join to model_table_dic
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_table_id, D.acc_id, D.acc_name, D.is_derived, D.acc_src, D.acc_sql"+
			" FROM table_acc D"+
			" INNER JOIN model_table_dic M ON (M.table_hid = D.table_hid)"+
			" WHERE M.model_id = "+smId+
			" ORDER BY 1, 2, 3",
		func(rows *sql.Rows) error {
			var r TableAccRow
			nIs := 0
			if err := rows.Scan(
				&r.ModelId, &r.TableId, &r.AccId, &r.Name, &nIs, &r.SrcAcc, &r.AccSql); err != nil {
				return err
			}
			r.IsDerived = nIs != 0 // oracle: smallint is float64

			idx, ok := meta.OutTableByKey(r.TableId) // find table row for that accumulator
			if !ok {
				return errors.New("output table " + strconv.Itoa(r.TableId) + " not found for " + r.Name)
			}

			meta.Table[idx].Acc = append(meta.Table[idx].Acc, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from table_expr join to model_table_dic
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_table_id, D.expr_id, D.expr_name, D.expr_decimals, D.expr_src, D.expr_sql"+
			" FROM table_expr D"+
			" INNER JOIN model_table_dic M ON (M.table_hid = D.table_hid)"+
			" WHERE M.model_id = "+smId+
			" ORDER BY 1, 2, 3",
		func(rows *sql.Rows) error {
			var r TableExprRow
			if err := rows.Scan(
				&r.ModelId, &r.TableId, &r.ExprId, &r.Name, &r.Decimals, &r.SrcExpr, &r.ExprSql); err != nil {
				return err
			}

			idx, ok := meta.OutTableByKey(r.TableId) // find table row for that expression
			if !ok {
				return errors.New("output table " + strconv.Itoa(r.TableId) + " not found for " + r.Name)
			}

			meta.Table[idx].Expr = append(meta.Table[idx].Expr, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// set db column name for output tables dimnesions, expressions, accumulators and entity attributes
	for k := range meta.Table {
		meta.Table[k].updateTableColumnNames()
	}

	// select db rows from entity_dic join to model_entity_dic table
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_entity_id, ET.entity_hid, ET.entity_name, ET.entity_digest"+
			" FROM entity_dic ET"+
			" INNER JOIN model_entity_dic M ON (M.entity_hid = ET.entity_hid)"+
			" WHERE M.model_id = "+smId+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r EntityDicRow
			if err := rows.Scan(&r.ModelId, &r.EntityId, &r.EntityHid, &r.Name, &r.Digest); err != nil {
				return err
			}

			meta.Entity = append(meta.Entity, EntityMeta{EntityDicRow: r, Attr: []EntityAttrRow{}})
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from entity_attr join to model_entity_dic table
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_entity_id, A.attr_id, A.attr_name, T.model_type_id, A.is_internal"+
			" FROM entity_attr A"+
			" INNER JOIN model_entity_dic M ON (M.entity_hid = A.entity_hid)"+
			" INNER JOIN model_type_dic T ON (T.type_hid = A.type_hid AND T.model_id = M.model_id)"+
			" WHERE M.model_id = "+smId+
			" ORDER BY 1, 2, 3",
		func(rows *sql.Rows) error {
			var r EntityAttrRow
			nInternal := 0
			if err := rows.Scan(
				&r.ModelId, &r.EntityId, &r.AttrId, &r.Name, &r.TypeId, &nInternal); err != nil {
				return err
			}
			r.IsInternal = nInternal != 0 // oracle: smallint is float64

			idx, ok := meta.EntityByKey(r.EntityId) // find entity row for that attribute
			if !ok {
				return errors.New("entity " + strconv.Itoa(r.EntityId) + " not found for " + r.Name)
			}
			k, ok := meta.TypeByKey(r.TypeId) // find attribute type
			if !ok {
				return errors.New("type " + strconv.Itoa(r.TypeId) + " not found for " + r.Name)
			}
			r.typeOf = &meta.Type[k]

			meta.Entity[idx].Attr = append(meta.Entity[idx].Attr, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// set db column name for entity attributes: attr0, attr1
	for k := range meta.Entity {
		meta.Entity[k].updateEntityColumnNames()
	}

	// select db rows from group_lst
	err = SelectRows(dbConn,
		"SELECT"+
			" model_id, group_id, is_parameter, group_name, is_hidden"+
			" FROM group_lst"+
			" WHERE model_id = "+smId+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r GroupLstRow
			nParam := 0
			nHidden := 0
			if err := rows.Scan(
				&r.ModelId, &r.GroupId, &nParam, &r.Name, &nHidden); err != nil {
				return err
			}
			r.IsParam = nParam != 0   // oracle: smallint is float64
			r.IsHidden = nHidden != 0 // oracle: smallint is float64
			meta.Group = append(meta.Group,
				GroupMeta{GroupLstRow: r, GroupPc: []GroupPcRow{}})
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from group_pc and append it as a child of group_lst row
	if grpCount := len(meta.Group); grpCount > 0 {
		nGrp := 0
		err = SelectRows(dbConn,
			"SELECT"+
				" model_id, group_id, child_pos, child_group_id, leaf_id"+
				" FROM group_pc"+
				" WHERE model_id = "+smId+
				" ORDER BY 1, 2, 3",
			func(rows *sql.Rows) error {
				var r GroupPcRow
				var cgId, leafId sql.NullInt64
				if err := rows.Scan(
					&r.ModelId, &r.GroupId, &r.ChildPos, &cgId, &leafId); err != nil {
					return err
				}
				if cgId.Valid {
					r.ChildGroupId = int(cgId.Int64)
				} else {
					r.ChildGroupId = -1
				}
				if leafId.Valid {
					r.ChildLeafId = int(leafId.Int64)
				} else {
					r.ChildLeafId = -1
				}

				// if parent group id not the same then find next parent group index
				if nGrp < grpCount && r.GroupId != meta.Group[nGrp].GroupId {
					for ; nGrp < grpCount; nGrp++ {
						if r.GroupId == meta.Group[nGrp].GroupId {
							break
						}
					}
				}
				// append to parent group, if exist
				if nGrp < grpCount {
					meta.Group[nGrp].GroupPc = append(meta.Group[nGrp].GroupPc, r)
				}
				return nil
			})
		if err != nil {
			return nil, err
		}
	}

	// select db rows from entity_group_lst
	err = SelectRows(dbConn,
		"SELECT"+
			" model_id, model_entity_id, group_id, group_name, is_hidden"+
			" FROM entity_group_lst"+
			" WHERE model_id = "+smId+
			" ORDER BY 1, 2, 3",
		func(rows *sql.Rows) error {
			var r EntityGroupLstRow
			nHidden := 0
			if err := rows.Scan(
				&r.ModelId, &r.EntityId, &r.GroupId, &r.Name, &nHidden); err != nil {
				return err
			}
			r.IsHidden = nHidden != 0 // oracle: smallint is float64
			meta.EntityGroup = append(meta.EntityGroup,
				EntityGroupMeta{EntityGroupLstRow: r, GroupPc: []EntityGroupPcRow{}})
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from entity_group_pc and append it as a child of entity_group_lst row
	if grpCount := len(meta.EntityGroup); grpCount > 0 {
		nGrp := 0
		err = SelectRows(dbConn,
			"SELECT"+
				" model_id, model_entity_id, group_id, child_pos, child_group_id, attr_id"+
				" FROM entity_group_pc"+
				" WHERE model_id = "+smId+
				" ORDER BY 1, 2, 3, 4",
			func(rows *sql.Rows) error {
				var r EntityGroupPcRow
				var cgId, attrId sql.NullInt64
				if err := rows.Scan(
					&r.ModelId, &r.EntityId, &r.GroupId, &r.ChildPos, &cgId, &attrId); err != nil {
					return err
				}
				if cgId.Valid {
					r.ChildGroupId = int(cgId.Int64)
				} else {
					r.ChildGroupId = -1
				}
				if attrId.Valid {
					r.AttrId = int(attrId.Int64)
				} else {
					r.AttrId = -1
				}

				// if parent entity id and group id not the same then find next parent group index
				if nGrp < grpCount && (r.EntityId != meta.EntityGroup[nGrp].EntityId || r.GroupId != meta.EntityGroup[nGrp].GroupId) {
					for ; nGrp < grpCount; nGrp++ {
						if r.EntityId == meta.EntityGroup[nGrp].EntityId && r.GroupId == meta.EntityGroup[nGrp].GroupId {
							break
						}
					}
				}
				// append to parent group, if exist
				if nGrp < grpCount {
					meta.EntityGroup[nGrp].GroupPc = append(meta.EntityGroup[nGrp].GroupPc, r)
				}
				return nil
			})
		if err != nil {
			return nil, err
		}
	}

	// update internal members used to link arrays to each other and simplify search:
	// type indexes, dimension indexes, type size, parameter and output table size
	if err := meta.updateInternals(); err != nil {
		return nil, err
	}
	return meta, nil
}
