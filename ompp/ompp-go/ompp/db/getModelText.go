// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"errors"
	"strconv"
)

// GetModelTextById return model_dic_txt table rows by model id.
//
// If langCode not empty then only specified language selected else all languages.
func GetModelTextRowById(dbConn *sql.DB, modelId int, langCode string) ([]ModelTxtRow, error) {

	// select db rows from model_dic_txt
	txtLst := []ModelTxtRow{}

	q := "SELECT" +
		" M.model_id, M.lang_id, L.lang_code, M.descr, M.note" +
		" FROM model_dic_txt M" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE M.model_id = " + strconv.Itoa(modelId)
	if langCode != "" {
		q += " AND L.lang_code = " + ToQuoted(langCode)
	}
	q += " ORDER BY 1, 2"

	err := SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var r ModelTxtRow
			var lId int
			var note sql.NullString
			if err := rows.Scan(
				&r.ModelId, &lId, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			txtLst = append(txtLst, r)
			return nil
		})
	if err != nil {
		return nil, err
	}
	return txtLst, nil
}

// GetModelText return model text metadata: description and notes.
// If langCode not empty then only specified language selected else all languages.
// If isPack is true then return empty text metadata for range types
func GetModelText(dbConn *sql.DB, modelId int, langCode string, isPack bool) (*ModelTxtMeta, error) {

	// select model name and digest by id
	meta := &ModelTxtMeta{
		ModelTxt:       []ModelTxtRow{},
		TypeTxt:        []TypeTxtRow{},
		TypeEnumTxt:    []TypeEnumTxtRow{},
		ParamTxt:       []ParamTxtRow{},
		ParamDimsTxt:   []ParamDimsTxtRow{},
		TableTxt:       []TableTxtRow{},
		TableDimsTxt:   []TableDimsTxtRow{},
		TableAccTxt:    []TableAccTxtRow{},
		TableExprTxt:   []TableExprTxtRow{},
		EntityTxt:      []EntityTxtRow{},
		EntityAttrTxt:  []EntityAttrTxtRow{},
		GroupTxt:       []GroupTxtRow{},
		EntityGroupTxt: []EntityGroupTxtRow{},
	}

	err := SelectFirst(dbConn,
		"SELECT model_name, model_digest FROM model_dic WHERE model_id = "+strconv.Itoa(modelId),
		func(row *sql.Row) error {
			return row.Scan(&meta.ModelName, &meta.ModelDigest)
		})
	switch {
	case err == sql.ErrNoRows:
		return nil, errors.New("model not found, invalid model id: " + strconv.Itoa(modelId))
	case err != nil:
		return nil, err
	}

	// make where clause parts:
	// WHERE M.model_id = 1234 AND L.lang_code = 'EN'
	where := " WHERE M.model_id = " + strconv.Itoa(modelId)
	if langCode != "" {
		where += " AND L.lang_code = " + ToQuoted(langCode)
	}

	// select db rows from model_dic_txt
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.lang_id, L.lang_code, M.descr, M.note"+
			" FROM model_dic_txt M"+
			" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)"+
			where+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r ModelTxtRow
			var lId int
			var note sql.NullString
			if err := rows.Scan(
				&r.ModelId, &lId, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			meta.ModelTxt = append(meta.ModelTxt, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from type_dic_txt join to model_type_dic

	rangeT := map[int]bool{} // map type id to is a range flag

	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_type_id, T.lang_id, H.dic_id, L.lang_code, T.descr, T.note"+
			" FROM type_dic_txt T"+
			" INNER JOIN model_type_dic M ON (M.type_hid = T.type_hid)"+
			" INNER JOIN type_dic H ON (H.type_hid = T.type_hid)"+
			" INNER JOIN lang_lst L ON (L.lang_id = T.lang_id)"+
			where+
			" ORDER BY 1, 2, 3",
		func(rows *sql.Rows) error {
			var r TypeTxtRow
			var lId, dicId int
			var note sql.NullString
			if err := rows.Scan(
				&r.ModelId, &r.TypeId, &lId, &dicId, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			rangeT[r.TypeId] = dicId == rangeDicId

			meta.TypeTxt = append(meta.TypeTxt, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from type_enum_txt join to model_type_dic
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_type_id, T.enum_id, T.lang_id, L.lang_code, T.descr, T.note"+
			" FROM type_enum_txt T"+
			" INNER JOIN model_type_dic M ON (M.type_hid = T.type_hid)"+
			" INNER JOIN lang_lst L ON (L.lang_id = T.lang_id)"+
			where+
			" ORDER BY 1, 2, 3, 4",
		func(rows *sql.Rows) error {
			var r TypeEnumTxtRow
			var lId int
			var note sql.NullString
			if err := rows.Scan(
				&r.ModelId, &r.TypeId, &r.EnumId, &lId, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			// skip empty text: if description and notes are empty
			// skip range text: if pack range flag set then skip range text, it is always empty
			if r.Descr == "" && r.Note == "" || isPack && rangeT[r.TypeId] {
				return nil
			}
			meta.TypeEnumTxt = append(meta.TypeEnumTxt, r) // append enum item: it is not a range type or do not pack range types
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from parameter_dic_txt join to model_parameter_dic
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_parameter_id, T.lang_id, L.lang_code, T.descr, T.note"+
			" FROM parameter_dic_txt T"+
			" INNER JOIN model_parameter_dic M ON (M.parameter_hid = T.parameter_hid)"+
			" INNER JOIN lang_lst L ON (L.lang_id = T.lang_id)"+
			where+
			" ORDER BY 1, 2, 3",
		func(rows *sql.Rows) error {
			var r ParamTxtRow
			var lId int
			var note sql.NullString
			if err := rows.Scan(
				&r.ModelId, &r.ParamId, &lId, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			meta.ParamTxt = append(meta.ParamTxt, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from parameter_dims_txt join to model_parameter_dic
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_parameter_id, T.dim_id, T.lang_id, L.lang_code, T.descr, T.note"+
			" FROM parameter_dims_txt T"+
			" INNER JOIN model_parameter_dic M ON (M.parameter_hid = T.parameter_hid)"+
			" INNER JOIN lang_lst L ON (L.lang_id = T.lang_id)"+
			where+
			" ORDER BY 1, 2, 3, 4",
		func(rows *sql.Rows) error {
			var r ParamDimsTxtRow
			var lId int
			var note sql.NullString
			if err := rows.Scan(
				&r.ModelId, &r.ParamId, &r.DimId, &lId, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			meta.ParamDimsTxt = append(meta.ParamDimsTxt, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from table_dic_txt join to model_table_dic
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_table_id, T.lang_id, L.lang_code, T.descr, T.note, T.expr_descr, T.expr_note"+
			" FROM table_dic_txt T"+
			" INNER JOIN model_table_dic M ON (M.table_hid = T.table_hid)"+
			" INNER JOIN lang_lst L ON (L.lang_id = T.lang_id)"+
			where+
			" ORDER BY 1, 2, 3",
		func(rows *sql.Rows) error {
			var r TableTxtRow
			var lId int
			var note, exnote sql.NullString
			if err := rows.Scan(
				&r.ModelId, &r.TableId, &lId, &r.LangCode, &r.Descr, &note, &r.ExprDescr, &exnote); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			if exnote.Valid {
				r.ExprNote = exnote.String
			}
			meta.TableTxt = append(meta.TableTxt, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from table_dims_txt join to model_table_dic
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_table_id, T.dim_id, T.lang_id, L.lang_code, T.descr, T.note"+
			" FROM table_dims_txt T"+
			" INNER JOIN model_table_dic M ON (M.table_hid = T.table_hid)"+
			" INNER JOIN lang_lst L ON (L.lang_id = T.lang_id)"+
			where+
			" ORDER BY 1, 2, 3, 4",
		func(rows *sql.Rows) error {
			var r TableDimsTxtRow
			var lId int
			var note sql.NullString
			if err := rows.Scan(
				&r.ModelId, &r.TableId, &r.DimId, &lId, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			meta.TableDimsTxt = append(meta.TableDimsTxt, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from table_acc_txt join to model_table_dic
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_table_id, T.acc_id, T.lang_id, L.lang_code, T.descr, T.note"+
			" FROM table_acc_txt T"+
			" INNER JOIN model_table_dic M ON (M.table_hid = T.table_hid)"+
			" INNER JOIN lang_lst L ON (L.lang_id = T.lang_id)"+
			where+
			" ORDER BY 1, 2, 3, 4",
		func(rows *sql.Rows) error {
			var r TableAccTxtRow
			var lId int
			var note sql.NullString
			if err := rows.Scan(
				&r.ModelId, &r.TableId, &r.AccId, &lId, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			meta.TableAccTxt = append(meta.TableAccTxt, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from table_expr_txt join to model_table_dic
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_table_id, T.expr_id, T.lang_id, L.lang_code, T.descr, T.note"+
			" FROM table_expr_txt T"+
			" INNER JOIN model_table_dic M ON (M.table_hid = T.table_hid)"+
			" INNER JOIN lang_lst L ON (L.lang_id = T.lang_id)"+
			where+
			" ORDER BY 1, 2, 3, 4",
		func(rows *sql.Rows) error {
			var r TableExprTxtRow
			var lId int
			var note sql.NullString
			if err := rows.Scan(
				&r.ModelId, &r.TableId, &r.ExprId, &lId, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			meta.TableExprTxt = append(meta.TableExprTxt, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from entity_dic_txt join to model_entity_dic table
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_entity_id, T.lang_id, L.lang_code, T.descr, T.note"+
			" FROM entity_dic_txt T"+
			" INNER JOIN model_entity_dic M ON (M.entity_hid = T.entity_hid)"+
			" INNER JOIN lang_lst L ON (L.lang_id = T.lang_id)"+
			where+
			" ORDER BY 1, 2, 3",
		func(rows *sql.Rows) error {
			var r EntityTxtRow
			var lId int
			var note sql.NullString
			if err := rows.Scan(
				&r.ModelId, &r.EntityId, &lId, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			meta.EntityTxt = append(meta.EntityTxt, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from entity_attr_txt join to model_entity_dic table
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_entity_id, T.attr_id, T.lang_id, L.lang_code, T.descr, T.note"+
			" FROM entity_attr_txt T"+
			" INNER JOIN model_entity_dic M ON (M.entity_hid = T.entity_hid)"+
			" INNER JOIN lang_lst L ON (L.lang_id = T.lang_id)"+
			where+
			" ORDER BY 1, 2, 3, 4",
		func(rows *sql.Rows) error {
			var r EntityAttrTxtRow
			var lId int
			var note sql.NullString
			if err := rows.Scan(
				&r.ModelId, &r.EntityId, &r.AttrId, &lId, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			meta.EntityAttrTxt = append(meta.EntityAttrTxt, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from group_txt
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.group_id, M.lang_id, L.lang_code, M.descr, M.note"+
			" FROM group_txt M"+
			" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)"+
			where+
			" ORDER BY 1, 2, 3",
		func(rows *sql.Rows) error {
			var r GroupTxtRow
			var lId int
			var note sql.NullString
			if err := rows.Scan(
				&r.ModelId, &r.GroupId, &lId, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			meta.GroupTxt = append(meta.GroupTxt, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// select db rows from entity_group_txt
	err = SelectRows(dbConn,
		"SELECT"+
			" M.model_id, M.model_entity_id, M.group_id, M.lang_id, L.lang_code, M.descr, M.note"+
			" FROM entity_group_txt M"+
			" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)"+
			where+
			" ORDER BY 1, 2, 3, 4",
		func(rows *sql.Rows) error {
			var r EntityGroupTxtRow
			var lId int
			var note sql.NullString
			if err := rows.Scan(
				&r.ModelId, &r.EntityId, &r.GroupId, &lId, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			meta.EntityGroupTxt = append(meta.EntityGroupTxt, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	return meta, nil
}
