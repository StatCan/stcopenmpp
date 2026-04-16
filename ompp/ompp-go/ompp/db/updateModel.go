// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"errors"
	"strconv"
)

// UpdateModel insert new model metadata in database, return true if model inserted or false if already exist.
//
// If model with same digest already exist then call simply return existing model metadata (no changes made in database).
// Parameters and output tables Hid's and db table names updated with actual database values
// If new model inserted then modelDef updated with actual id's (model id, parameter Hid...)
// If parameter (output table) not exist then create db tables for parameter values (output table values)
// If db table names is "" empty then make db table names for parameter values (output table values)
func UpdateModel(dbConn *sql.DB, dbFacet Facet, modelDef *ModelMeta) (bool, error) {

	// validate parameters
	if modelDef == nil {
		return false, errors.New("invalid (empty) model metadata")
	}
	if modelDef.Model.Name == "" || modelDef.Model.Digest == "" {
		return false, errors.New("invalid (empty) model name or model digest")
	}

	// check if model already exist
	isExist, mId, err := GetModelId(dbConn, "", modelDef.Model.Digest)
	if err != nil {
		return isExist, err
	}
	if isExist {
		md, err := GetModelById(dbConn, mId) // read existing model definition
		if err != nil {
			return isExist, err
		}
		*modelDef = *md
		return isExist, err
	}
	// else
	// model not exist: do insert and update in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return isExist, err
	}
	if err = doInsertModel(trx, dbFacet, modelDef); err != nil {
		trx.Rollback()
		return isExist, err
	}
	trx.Commit()
	return isExist, nil
}

// doInsertModel insert new model metadata in database.
// It does update as part of transaction
// Parameters, output tables, entities Hid's and db table names updated with actual database values
// If new model inserted then modelDef updated with actual id's (model id, parameter Hid...)
// If parameter (output table) not exist then create db tables for parameter values (output table values)
// If db table names is "" empty or too long then make db table names for parameter values (output table values)
func doInsertModel(trx *sql.Tx, dbFacet Facet, modelDef *ModelMeta) error {

	// find default model language id by code
	var dlId int
	err := TrxSelectFirst(trx,
		"SELECT lang_id FROM lang_lst WHERE lang_code = "+ToQuoted(modelDef.Model.DefaultLangCode),
		func(row *sql.Row) error {
			return row.Scan(&dlId)
		})
	switch {
	case err == sql.ErrNoRows:
		return errors.New("invalid default model language: " + modelDef.Model.DefaultLangCode)
	case err != nil:
		return err
	}

	// get new model id
	err = TrxUpdate(trx,
		"UPDATE id_lst SET id_value = id_value + 1 WHERE id_key = 'model_id'")
	if err != nil {
		return err
	}
	err = TrxSelectFirst(trx,
		"SELECT id_value FROM id_lst WHERE id_key = 'model_id'",
		func(row *sql.Row) error {
			return row.Scan(&modelDef.Model.ModelId)
		})
	switch {
	case err == sql.ErrNoRows:
		return errors.New("invalid destination database, likely not an openM++ database")
	case err != nil:
		return err
	}
	smId := strconv.Itoa(modelDef.Model.ModelId)

	// insert model_dic row with new model id
	// INSERT INTO model_dic
	//   (model_id, model_name, model_digest, model_type, model_ver, create_dt, default_lang_id)
	// VALUES
	//   (1234, 'modelOne', '1234abcd', 0, '1.0.0.0', '2012-08-17 16:04:59.148', 0)
	err = TrxUpdate(trx,
		"INSERT INTO model_dic"+
			" (model_id, model_name, model_digest, model_type, model_ver, create_dt, default_lang_id)"+
			" VALUES ("+
			smId+", "+
			toQuotedMax(modelDef.Model.Name, nameDbMax)+", "+
			toQuotedMax(modelDef.Model.Digest, codeDbMax)+", "+
			strconv.Itoa(modelDef.Model.Type)+", "+
			toQuotedMax(modelDef.Model.Version, codeDbMax)+", "+
			toQuotedMax(modelDef.Model.CreateDateTime, codeDbMax)+", "+
			strconv.Itoa(dlId)+")")
	if err != nil {
		return err
	}

	// for each type:
	// if type not exist then insert into type_dic, type_enum_lst
	// update type Hid with actual db value
	// insert into model_type_dic to append this type to the model
	for idx := range modelDef.Type {

		modelDef.Type[idx].ModelId = modelDef.Model.ModelId // update model id with db value

		// get new type Hid
		// UPDATE id_lst SET id_value =
		//   CASE
		//     WHEN 0 = (SELECT COUNT(*) FROM type_dic WHERE type_digest = 'abcdef')
		//       THEN id_value + 1
		//     ELSE id_value
		//   END
		// WHERE id_key = 'type_hid'
		err = TrxUpdate(trx,
			"UPDATE id_lst SET id_value ="+
				" CASE"+
				" WHEN 0 = (SELECT COUNT(*) FROM type_dic WHERE type_digest = "+ToQuoted(modelDef.Type[idx].Digest)+")"+
				" THEN id_value + 1"+
				" ELSE id_value"+
				" END"+
				" WHERE id_key = 'type_hid'")
		if err != nil {
			return err
		}

		// check if this type already exist
		modelDef.Type[idx].TypeHid = 0
		err = TrxSelectFirst(trx,
			"SELECT type_hid FROM type_dic WHERE type_digest = "+ToQuoted(modelDef.Type[idx].Digest),
			func(row *sql.Row) error {
				return row.Scan(&modelDef.Type[idx].TypeHid)
			})
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		// if type not exists then insert into type_dic and type_enum_lst
		if modelDef.Type[idx].TypeHid <= 0 {

			// get new type Hid
			err = TrxSelectFirst(trx,
				"SELECT id_value FROM id_lst WHERE id_key = 'type_hid'",
				func(row *sql.Row) error {
					return row.Scan(&modelDef.Type[idx].TypeHid)
				})
			switch {
			case err == sql.ErrNoRows:
				return errors.New("invalid destination database, likely not an openM++ database")
			case err != nil:
				return err
			}
			sHid := strconv.Itoa(modelDef.Type[idx].TypeHid)

			// INSERT INTO type_dic
			//   (type_hid, type_name, type_digest, dic_id, total_enum_id)
			// VALUES
			//   (26, 'age', '20128171604590121', 2, 4)
			err = TrxUpdate(trx,
				"INSERT INTO type_dic (type_hid, type_name, type_digest, dic_id, total_enum_id)"+
					" VALUES ("+
					sHid+", "+
					toQuotedMax(modelDef.Type[idx].Name, nameDbMax)+", "+
					toQuotedMax(modelDef.Type[idx].Digest, codeDbMax)+", "+
					strconv.Itoa(modelDef.Type[idx].DicId)+", "+
					strconv.Itoa(modelDef.Type[idx].TotalEnumId)+")")
			if err != nil {
				return err
			}

			// INSERT INTO type_enum_lst (type_hid, enum_id, enum_name) VALUES (26, 0, '10')
			if !modelDef.Type[idx].IsRange || len(modelDef.Type[idx].Enum) > 0 {

				for j := range modelDef.Type[idx].Enum {

					modelDef.Type[idx].Enum[j].ModelId = modelDef.Model.ModelId // update model id with db value

					err = TrxUpdate(trx,
						"INSERT INTO type_enum_lst (type_hid, enum_id, enum_name) VALUES ("+
							sHid+", "+
							strconv.Itoa(modelDef.Type[idx].Enum[j].EnumId)+", "+
							toQuotedMax(modelDef.Type[idx].Enum[j].Name, nameDbMax)+")")
					if err != nil {
						return err
					}
				}
			} else {

				for nId := modelDef.Type[idx].MinEnumId; nId <= modelDef.Type[idx].MaxEnumId; nId++ {

					err = TrxUpdate(trx,
						"INSERT INTO type_enum_lst (type_hid, enum_id, enum_name) VALUES ("+
							sHid+", "+
							strconv.Itoa(nId)+", "+
							"'"+strconv.Itoa(nId)+"')")
					if err != nil {
						return err
					}
				}
			}
		}

		// append type into model type list, if not in the list
		//
		// INSERT INTO model_type_dic (model_id, model_type_id, type_hid)
		// SELECT 1234, 31, D.type_hid
		// FROM type_dic D
		// WHERE D.type_digest = '20128171604590121'
		// AND NOT EXISTS
		// (
		//   SELECT * FROM model_type_dic E WHERE E.model_id = 1234 AND E.model_type_id = 31
		// )
		err = TrxUpdate(trx,
			"INSERT INTO model_type_dic (model_id, model_type_id, type_hid)"+
				" SELECT "+
				smId+", "+
				strconv.Itoa(modelDef.Type[idx].TypeId)+", "+
				" D.type_hid"+
				" FROM type_dic D"+
				" WHERE D.type_digest = "+ToQuoted(modelDef.Type[idx].Digest)+
				" AND NOT EXISTS"+
				" (SELECT * FROM model_type_dic E"+
				" WHERE E.model_id = "+smId+
				" AND E.model_type_id = "+strconv.Itoa(modelDef.Type[idx].TypeId)+
				" )")
		if err != nil {
			return err
		}
	}

	// for each parameter:
	// if parameter not exist then insert into parameter_dic, parameter_dims
	// update parameter Hid with actual db value
	// insert into model_parameter_dic to append this parameter to the model
	// if parameter not exist then create db tables for parameter values
	// if db table names is "" empty then make db table names for parameter values
	for idx := range modelDef.Param {

		modelDef.Param[idx].ModelId = modelDef.Model.ModelId // update model id with db value

		// get new parameter Hid
		// UPDATE id_lst SET id_value =
		//   CASE
		//     WHEN 0 = (SELECT COUNT(*) FROM parameter_dic WHERE parameter_digest = '978abf5')
		//       THEN id_value + 1
		//     ELSE id_value
		//   END
		// WHERE id_key = 'parameter_hid'
		err = TrxUpdate(trx,
			"UPDATE id_lst SET id_value ="+
				" CASE"+
				" WHEN 0 = (SELECT COUNT(*) FROM parameter_dic WHERE parameter_digest = "+ToQuoted(modelDef.Param[idx].Digest)+")"+
				" THEN id_value + 1"+
				" ELSE id_value"+
				" END"+
				" WHERE id_key = 'parameter_hid'")
		if err != nil {
			return err
		}

		// check if this parameter already exist
		modelDef.Param[idx].ParamHid = 0
		err = TrxSelectFirst(trx,
			"SELECT parameter_hid FROM parameter_dic WHERE parameter_digest = "+ToQuoted(modelDef.Param[idx].Digest),
			func(row *sql.Row) error {
				return row.Scan(&modelDef.Param[idx].ParamHid)
			})
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		// if parameter not exists then insert into parameter_dic and parameter_dims
		// if db table names is "" empty then make db table names for parameter values
		// if parameter not exist then create db tables for parameter values
		if modelDef.Param[idx].ParamHid <= 0 {

			// get new parameter Hid
			err = TrxSelectFirst(trx,
				"SELECT id_value FROM id_lst WHERE id_key = 'parameter_hid'",
				func(row *sql.Row) error {
					return row.Scan(&modelDef.Param[idx].ParamHid)
				})
			switch {
			case err == sql.ErrNoRows:
				return errors.New("invalid destination database, likely not an openM++ database")
			case err != nil:
				return err
			}

			// update parameter values db table names, if empty or too long for current database
			if modelDef.Param[idx].DbRunTable == "" ||
				len(modelDef.Param[idx].DbRunTable) > maxTableNameSize ||
				modelDef.Param[idx].DbSetTable == "" ||
				len(modelDef.Param[idx].DbSetTable) > maxTableNameSize {

				p, s := makeDbTablePrefixSuffix(modelDef.Param[idx].Name, modelDef.Param[idx].Digest)
				modelDef.Param[idx].DbRunTable = p + "_p" + s
				modelDef.Param[idx].DbSetTable = p + "_w" + s
			}

			// INSERT INTO parameter_dic
			//   (parameter_hid, parameter_name, parameter_digest, db_run_table,
			//   db_set_table, parameter_rank, type_hid, is_extendable, num_cumulated)
			// VALUES
			//   (4, 'ageSex', '978abf5', 'ageSex', '2012817', 2, 14, 1, 0)
			err = TrxUpdate(trx,
				"INSERT INTO parameter_dic"+
					" (parameter_hid, parameter_name, parameter_digest, db_run_table,"+
					" db_set_table, parameter_rank, type_hid, is_extendable,"+
					" num_cumulated, import_digest)"+
					" VALUES ("+
					strconv.Itoa(modelDef.Param[idx].ParamHid)+", "+
					toQuotedMax(modelDef.Param[idx].Name, nameDbMax)+", "+
					toQuotedMax(modelDef.Param[idx].Digest, codeDbMax)+", "+
					ToQuoted(modelDef.Param[idx].DbRunTable)+", "+
					ToQuoted(modelDef.Param[idx].DbSetTable)+", "+
					strconv.Itoa(modelDef.Param[idx].Rank)+", "+
					strconv.Itoa(modelDef.Param[idx].typeOf.TypeHid)+", "+
					toBoolSqlConst(modelDef.Param[idx].IsExtendable)+", "+
					strconv.Itoa(modelDef.Param[idx].NumCumulated)+", "+
					toQuotedMax(modelDef.Param[idx].ImportDigest, codeDbMax)+
					")")
			if err != nil {
				return err
			}

			// INSERT INTO parameter_dims (parameter_hid, dim_id, dim_name, type_hid) VALUES (4, 0, 'dim0', 26)
			for j := range modelDef.Param[idx].Dim {

				modelDef.Param[idx].Dim[j].ModelId = modelDef.Model.ModelId // update model id with db value

				err = TrxUpdate(trx,
					"INSERT INTO parameter_dims (parameter_hid, dim_id, dim_name, type_hid) VALUES ("+
						strconv.Itoa(modelDef.Param[idx].ParamHid)+", "+
						strconv.Itoa(modelDef.Param[idx].Dim[j].DimId)+", "+
						ToQuoted(modelDef.Param[idx].Dim[j].Name)+", "+
						strconv.Itoa(modelDef.Param[idx].Dim[j].typeOf.TypeHid)+")")
				if err != nil {
					return err
				}
			}

			// create parameter tables: parameter run values and parameter workset values
			rSql, wSql, err := sqlCreateParamTable(dbFacet, &modelDef.Param[idx])
			if err != nil {
				return err
			}
			err = TrxUpdate(trx, rSql)
			if err != nil {
				return err
			}
			err = TrxUpdate(trx, wSql)
			if err != nil {
				return err
			}
		}

		// append parameter into model parameter list, if not in the list
		//
		// INSERT INTO model_parameter_dic (model_id, model_parameter_id, parameter_hid, is_hidden)
		// SELECT 1234, 0, D.parameter_hid, 1
		// FROM parameter_dic D
		// WHERE D.parameter_digest = '978abf5'
		// AND NOT EXISTS
		// (
		//   SELECT * FROM model_parameter_dic E WHERE E.model_id = 1234 AND E.model_parameter_id = 0
		// )
		err = TrxUpdate(trx,
			"INSERT INTO model_parameter_dic (model_id, model_parameter_id, parameter_hid, is_hidden)"+
				" SELECT "+
				smId+", "+
				strconv.Itoa(modelDef.Param[idx].ParamId)+", "+
				" D.parameter_hid, "+
				toBoolSqlConst(modelDef.Param[idx].IsHidden)+
				" FROM parameter_dic D"+
				" WHERE D.parameter_digest = "+ToQuoted(modelDef.Param[idx].Digest)+
				" AND NOT EXISTS"+
				" (SELECT * FROM model_parameter_dic E"+
				" WHERE E.model_id = "+smId+
				" AND E.model_parameter_id = "+strconv.Itoa(modelDef.Param[idx].ParamId)+
				" )")
		if err != nil {
			return err
		}

		// INSERT INTO model_parameter_import
		//   (model_id, model_parameter_id, from_name, from_model_name, is_sample_dim)
		// VALUES
		//   (1234, 101, 'ageSex', 'modelOne', 0)
		for j := range modelDef.Param[idx].Import {

			modelDef.Param[idx].Import[j].ModelId = modelDef.Model.ModelId // update model id with db value

			err = TrxUpdate(trx,
				"INSERT INTO model_parameter_import (model_id, model_parameter_id, from_name, from_model_name, is_sample_dim)"+
					" VALUES ("+
					smId+", "+
					strconv.Itoa(modelDef.Param[idx].ParamId)+", "+
					ToQuoted(modelDef.Param[idx].Import[j].FromName)+", "+
					ToQuoted(modelDef.Param[idx].Import[j].FromModel)+", "+
					toBoolSqlConst(modelDef.Param[idx].Import[j].IsSampleDim)+")")
			if err != nil {
				return err
			}
		}
	}

	// for each output table:
	// if output table not exist then insert into table_dic, table_dims, table_acc, table_expr
	// update table Hid with actual db value
	// insert into model_table_dic to append this output table to the model
	// if output table not exist then create db tables for output table values
	// if db table names is "" empty then make db table names for output table values
	for idx := range modelDef.Table {

		modelDef.Table[idx].ModelId = modelDef.Model.ModelId // update model id with db value

		// get new output table Hid
		// UPDATE id_lst SET id_value =
		//   CASE
		//     WHEN 0 = (SELECT COUNT(*) FROM table_dic WHERE table_digest = '0887a6494df')
		//      THEN id_value + 1
		//     ELSE id_value
		//   END
		// WHERE id_key = 'table_hid'
		err = TrxUpdate(trx,
			"UPDATE id_lst SET id_value ="+
				" CASE"+
				" WHEN 0 = (SELECT COUNT(*) FROM table_dic WHERE table_digest = "+ToQuoted(modelDef.Table[idx].Digest)+")"+
				" THEN id_value + 1"+
				" ELSE id_value"+
				" END"+
				" WHERE id_key = 'table_hid'")
		if err != nil {
			return err
		}

		// check if this output table already exist
		modelDef.Table[idx].TableHid = 0
		err = TrxSelectFirst(trx,
			"SELECT table_hid FROM table_dic WHERE table_digest = "+ToQuoted(modelDef.Table[idx].Digest),
			func(row *sql.Row) error {
				return row.Scan(&modelDef.Table[idx].TableHid)
			})
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		// if output table not exists then insert into table_dic, table_dims, table_acc, table_expr
		// if output table not exist then create db tables for output table values
		// if db table names is "" empty then make db table names for output table values
		if modelDef.Table[idx].TableHid <= 0 {

			// get new output table Hid
			err = TrxSelectFirst(trx,
				"SELECT id_value FROM id_lst WHERE id_key = 'table_hid'",
				func(row *sql.Row) error {
					return row.Scan(&modelDef.Table[idx].TableHid)
				})
			switch {
			case err == sql.ErrNoRows:
				return errors.New("invalid destination database, likely not an openM++ database")
			case err != nil:
				return err
			}

			// update output table values db table names, if empty or too long for current database
			if modelDef.Table[idx].DbExprTable == "" ||
				len(modelDef.Table[idx].DbExprTable) > maxTableNameSize ||
				modelDef.Table[idx].DbAccTable == "" ||
				len(modelDef.Table[idx].DbAccTable) > maxTableNameSize {

				p, s := makeDbTablePrefixSuffix(modelDef.Table[idx].Name, modelDef.Table[idx].Digest)
				modelDef.Table[idx].DbExprTable = p + "_v" + s
				modelDef.Table[idx].DbAccTable = p + "_a" + s
				modelDef.Table[idx].DbAccAllView = p + "_d" + s
			}

			// INSERT INTO table_dic
			//   (table_hid, table_name, table_digest, table_rank,
			//   is_sparse, db_expr_table, db_acc_table, db_acc_all_view,
			//   import_digest)
			// VALUES
			//   (2, 'salarySex', '0887a6494df', 'salarySex', '2012820', 2, 1)
			err = TrxUpdate(trx,
				"INSERT INTO table_dic"+
					" (table_hid, table_name, table_digest, table_rank,"+
					" is_sparse, db_expr_table, db_acc_table, db_acc_all_view,"+
					" import_digest)"+
					" VALUES ("+
					strconv.Itoa(modelDef.Table[idx].TableHid)+", "+
					toQuotedMax(modelDef.Table[idx].Name, nameDbMax)+", "+
					toQuotedMax(modelDef.Table[idx].Digest, codeDbMax)+", "+
					strconv.Itoa(modelDef.Table[idx].Rank)+", "+
					toBoolSqlConst(modelDef.Table[idx].IsSparse)+", "+
					ToQuoted(modelDef.Table[idx].DbExprTable)+", "+
					ToQuoted(modelDef.Table[idx].DbAccTable)+", "+
					ToQuoted(modelDef.Table[idx].DbAccAllView)+", "+
					toQuotedMax(modelDef.Table[idx].ImportDigest, codeDbMax)+
					")")
			if err != nil {
				return err
			}

			// INSERT INTO table_dims
			//   (table_hid, dim_id, dim_name, type_hid, is_total, dim_size)
			// VALUES
			//   (2, 0, 'dim0', 28, 0, 3)
			for j := range modelDef.Table[idx].Dim {

				modelDef.Table[idx].Dim[j].ModelId = modelDef.Model.ModelId // update model id with db value

				err = TrxUpdate(trx,
					"INSERT INTO table_dims (table_hid, dim_id, dim_name, type_hid, is_total, dim_size)"+
						" VALUES ("+
						strconv.Itoa(modelDef.Table[idx].TableHid)+", "+
						strconv.Itoa(modelDef.Table[idx].Dim[j].DimId)+", "+
						ToQuoted(modelDef.Table[idx].Dim[j].Name)+", "+
						strconv.Itoa(modelDef.Table[idx].Dim[j].typeOf.TypeHid)+", "+
						toBoolSqlConst(modelDef.Table[idx].Dim[j].IsTotal)+", "+
						strconv.Itoa(modelDef.Table[idx].Dim[j].DimSize)+")")
				if err != nil {
					return err
				}
			}

			// INSERT INTO table_acc (table_hid, acc_id, acc_name, is_derived, acc_expr)
			// VALUES (2, 4, 'acc4', 1, 'acc0 + acc1')
			for j := range modelDef.Table[idx].Acc {

				modelDef.Table[idx].Acc[j].ModelId = modelDef.Model.ModelId // update model id with db value

				err = TrxUpdate(trx,
					"INSERT INTO table_acc (table_hid, acc_id, acc_name, is_derived, acc_src, acc_sql)"+
						" VALUES ("+
						strconv.Itoa(modelDef.Table[idx].TableHid)+", "+
						strconv.Itoa(modelDef.Table[idx].Acc[j].AccId)+", "+
						ToQuoted(modelDef.Table[idx].Acc[j].Name)+", "+
						toBoolSqlConst(modelDef.Table[idx].Acc[j].IsDerived)+", "+
						ToQuoted(modelDef.Table[idx].Acc[j].SrcAcc)+", "+
						ToQuoted(modelDef.Table[idx].Acc[j].AccSql)+")")
				if err != nil {
					return err
				}
			}

			// INSERT INTO table_expr
			//   (table_hid, expr_id, expr_name, expr_decimals, expr_src, expr_sql)
			// VALUES
			//   (2, 0, 'expr0', 4, 'OM_AVG(acc0)', '....sql....')
			for j := range modelDef.Table[idx].Expr {

				modelDef.Table[idx].Expr[j].ModelId = modelDef.Model.ModelId // update model id with db value

				err = TrxUpdate(trx,
					"INSERT INTO table_expr (table_hid, expr_id, expr_name, expr_decimals, expr_src, expr_sql)"+
						" VALUES ("+
						strconv.Itoa(modelDef.Table[idx].TableHid)+", "+
						strconv.Itoa(modelDef.Table[idx].Expr[j].ExprId)+", "+
						ToQuoted(modelDef.Table[idx].Expr[j].Name)+", "+
						strconv.Itoa(modelDef.Table[idx].Expr[j].Decimals)+", "+
						ToQuoted(modelDef.Table[idx].Expr[j].SrcExpr)+", "+
						ToQuoted(modelDef.Table[idx].Expr[j].ExprSql)+")")
				if err != nil {
					return err
				}
			}

			// create db tables: output table expression(s) values and accumulator(s) values
			eSql, aSql := sqlCreateOutTable(dbFacet, &modelDef.Table[idx])
			err = TrxUpdate(trx, eSql)
			if err != nil {
				return err
			}
			err = TrxUpdate(trx, aSql)
			if err != nil {
				return err
			}

			// create db views: output table all accumulators view
			avSql := sqlCreateAccAllView(dbFacet, &modelDef.Table[idx])
			err = TrxUpdate(trx, avSql)
			if err != nil {
				return err
			}
		}

		// append into model output table list, if not in the list
		//
		// INSERT INTO model_table_dic (model_id, model_table_id, table_hid, is_user, expr_dim_pos, is_hidden)
		// SELECT 1234, 0, D.table_hid, 0, 1
		// FROM table_dic D
		// WHERE D.table_digest = '0887a6494df'
		// AND NOT EXISTS
		// (
		//   SELECT * FROM model_table_dic E WHERE E.model_id = 1234 AND E.model_table_id = 0
		// )
		err = TrxUpdate(trx,
			"INSERT INTO model_table_dic (model_id, model_table_id, table_hid, is_user, expr_dim_pos, is_hidden)"+
				" SELECT "+
				smId+", "+
				strconv.Itoa(modelDef.Table[idx].TableId)+", "+
				" D.table_hid, "+
				toBoolSqlConst(modelDef.Table[idx].IsUser)+", "+
				strconv.Itoa(modelDef.Table[idx].ExprPos)+", "+
				toBoolSqlConst(modelDef.Table[idx].IsHidden)+
				" FROM table_dic D"+
				" WHERE D.table_digest = "+ToQuoted(modelDef.Table[idx].Digest)+
				" AND NOT EXISTS"+
				" (SELECT * FROM model_table_dic E"+
				" WHERE E.model_id = "+smId+
				" AND E.model_table_id = "+strconv.Itoa(modelDef.Table[idx].TableId)+
				" )")
		if err != nil {
			return err
		}
	}

	// for each entity:
	// if entity not exist then insert into entity_dic, entity_attr
	// update entity Hid with actual db value
	// insert into model_entity_dic to append this entity to the model
	for idx := range modelDef.Entity {

		modelDef.Entity[idx].ModelId = modelDef.Model.ModelId // update model id with db value

		// get new entity Hid
		// UPDATE id_lst SET id_value =
		//   CASE
		//     WHEN 0 = (SELECT COUNT(*) FROM entity_dic WHERE entity_digest = '978abf5')
		//       THEN id_value + 1
		//     ELSE id_value
		//   END
		// WHERE id_key = 'entity_hid'
		err = TrxUpdate(trx,
			"UPDATE id_lst SET id_value ="+
				" CASE"+
				" WHEN 0 = (SELECT COUNT(*) FROM entity_dic WHERE entity_digest = "+ToQuoted(modelDef.Entity[idx].Digest)+")"+
				" THEN id_value + 1"+
				" ELSE id_value"+
				" END"+
				" WHERE id_key = 'entity_hid'")
		if err != nil {
			return err
		}

		// check if this entity already exist
		modelDef.Entity[idx].EntityHid = 0
		err = TrxSelectFirst(trx,
			"SELECT entity_hid FROM entity_dic WHERE entity_digest = "+ToQuoted(modelDef.Entity[idx].Digest),
			func(row *sql.Row) error {
				return row.Scan(&modelDef.Entity[idx].EntityHid)
			})
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		// if entity not exists then insert into entity_dic and entity_attr
		if modelDef.Entity[idx].EntityHid <= 0 {

			// get new entity Hid
			err = TrxSelectFirst(trx,
				"SELECT id_value FROM id_lst WHERE id_key = 'entity_hid'",
				func(row *sql.Row) error {
					return row.Scan(&modelDef.Entity[idx].EntityHid)
				})
			switch {
			case err == sql.ErrNoRows:
				return errors.New("invalid destination database, likely not an openM++ database")
			case err != nil:
				return err
			}

			// INSERT INTO entity_dic (entity_hid, entity_name, entity_digest) VALUES (101, 'Person', '7890abcd')
			err = TrxUpdate(trx,
				"INSERT INTO entity_dic"+
					" (entity_hid, entity_name, entity_digest)"+
					" VALUES ("+
					strconv.Itoa(modelDef.Entity[idx].EntityHid)+", "+
					toQuotedMax(modelDef.Entity[idx].Name, nameDbMax)+", "+
					toQuotedMax(modelDef.Entity[idx].Digest, codeDbMax)+
					")")
			if err != nil {
				return err
			}

			// INSERT INTO entity_attr
			//   (entity_hid, attr_id, attr_name, type_hid, is_internal)
			// VALUES
			//   (101, 2, 'Age', 7, 1)
			for j := range modelDef.Entity[idx].Attr {

				modelDef.Entity[idx].Attr[j].ModelId = modelDef.Model.ModelId // update model id with db value

				err = TrxUpdate(trx,
					"INSERT INTO entity_attr"+
						" (entity_hid, attr_id, attr_name, type_hid, is_internal)"+
						" VALUES ("+
						strconv.Itoa(modelDef.Entity[idx].EntityHid)+", "+
						strconv.Itoa(modelDef.Entity[idx].Attr[j].AttrId)+", "+
						ToQuoted(modelDef.Entity[idx].Attr[j].Name)+", "+
						strconv.Itoa(modelDef.Entity[idx].Attr[j].typeOf.TypeHid)+", "+
						toBoolSqlConst(modelDef.Entity[idx].Attr[j].IsInternal)+
						")")
				if err != nil {
					return err
				}
			}
		}

		// append entity into model list, if not in the list
		//
		// INSERT INTO model_entity_dic (model_id, model_entity_id, entity_hid)
		// SELECT 1234, 1, D.entity_hid
		// FROM entity_dic D
		// WHERE D.entity_digest = '7890abcd'
		// AND NOT EXISTS
		// (
		//   SELECT * FROM model_entity_dic E WHERE E.model_id = 1234 AND E.model_entity_id = 1
		// )
		err = TrxUpdate(trx,
			"INSERT INTO model_entity_dic (model_id, model_entity_id, entity_hid)"+
				" SELECT "+
				smId+", "+
				strconv.Itoa(modelDef.Entity[idx].EntityId)+", "+
				" D.entity_hid"+
				" FROM entity_dic D"+
				" WHERE D.entity_digest = "+ToQuoted(modelDef.Entity[idx].Digest)+
				" AND NOT EXISTS"+
				" (SELECT * FROM model_entity_dic E"+
				" WHERE E.model_id = "+smId+
				" AND E.model_entity_id = "+strconv.Itoa(modelDef.Entity[idx].EntityId)+
				" )")
		if err != nil {
			return err
		}
	}

	// for all groups of parameters or output tables check constarints: group id is unique and group name is unique
	grpIdCounts := make(map[int]int)
	grpNameCounts := make(map[string]int)
	for idx := range modelDef.Group {
		if _, isExist := grpNameCounts[modelDef.Group[idx].Name]; !isExist {
			grpNameCounts[modelDef.Group[idx].Name] = 1
		} else {
			return errors.New("invalid (duplicate) group name: " + modelDef.Group[idx].Name)
		}
		if _, isExist := grpIdCounts[modelDef.Group[idx].GroupId]; !isExist {
			grpIdCounts[modelDef.Group[idx].GroupId] = 1
		} else {
			return errors.New("invalid (duplicate) group id: " + strconv.Itoa(modelDef.Group[idx].GroupId))
		}
	}

	// for each group of parameters or output tables:
	// insert into group_lst table and child rows into group_pc table
	// update model_id with actual db value
	// convert id to string or return "NULL" if id is negative
	idOrNullIfNegative := func(id int) string {
		if id < 0 {
			return "NULL"
		}
		return strconv.Itoa(id)
	}

	for idx := range modelDef.Group {

		modelDef.Group[idx].ModelId = modelDef.Model.ModelId // update model id with db value

		// INSERT INTO group_lst
		//   (model_id, group_id, is_parameter, group_name, is_hidden)
		// VALUES
		//   (1234, 4, 1, 'Geo_group', 0)
		err = TrxUpdate(trx,
			"INSERT INTO group_lst"+
				" (model_id, group_id, is_parameter, group_name, is_hidden)"+
				" VALUES ("+
				smId+", "+
				strconv.Itoa(modelDef.Group[idx].GroupId)+", "+
				toBoolSqlConst(modelDef.Group[idx].IsParam)+", "+
				toQuotedMax(modelDef.Group[idx].Name, nameDbMax)+", "+
				toBoolSqlConst(modelDef.Group[idx].IsHidden)+")")
		if err != nil {
			return err
		}

		// insert child rows into group_pc table
		for pcIdx := range modelDef.Group[idx].GroupPc {

			modelDef.Group[idx].GroupPc[pcIdx].ModelId = modelDef.Model.ModelId // update model id with db value

			// insert group members rows into group_pc
			// if child group id < 0 or leaf id < then treat it as NULL value
			//
			// INSERT INTO group_pc
			//   (model_id, group_id, child_pos, child_group_id, leaf_id)
			// VALUES
			//   (1234, 4, 1, NULL, 101)
			err = TrxUpdate(trx,
				"INSERT INTO group_pc"+
					" (model_id, group_id, child_pos, child_group_id, leaf_id)"+
					" VALUES ("+
					smId+", "+
					strconv.Itoa(modelDef.Group[idx].GroupPc[pcIdx].GroupId)+", "+
					strconv.Itoa(modelDef.Group[idx].GroupPc[pcIdx].ChildPos)+", "+
					idOrNullIfNegative(modelDef.Group[idx].GroupPc[pcIdx].ChildGroupId)+", "+
					idOrNullIfNegative(modelDef.Group[idx].GroupPc[pcIdx].ChildLeafId)+")")
			if err != nil {
				return err
			}
		}
	}

	// for all entity attribute groups check constarints: (entity id, group id) is unique and group name is unique
	type egt struct {
		gId int
		eId int
	}
	egIdCounts := map[egt]int{}
	egNameCounts := map[string]int{}

	for idx := range modelDef.EntityGroup {
		if _, isExist := egNameCounts[modelDef.EntityGroup[idx].Name]; !isExist {
			egNameCounts[modelDef.EntityGroup[idx].Name] = 1
		} else {
			return errors.New("invalid (duplicate) entity group name: " + modelDef.EntityGroup[idx].Name)
		}
		eg := egt{
			eId: modelDef.EntityGroup[idx].EntityId,
			gId: modelDef.EntityGroup[idx].GroupId,
		}
		if _, isExist := egIdCounts[eg]; !isExist {
			egIdCounts[eg] = 1
		} else {
			return errors.New("invalid (duplicate) entity id and group id: " + strconv.Itoa(modelDef.EntityGroup[idx].EntityId) + " " + strconv.Itoa(modelDef.EntityGroup[idx].GroupId))
		}
	}

	// for each entity attribute group:
	// insert into entity_group_lst table and child rows into entity_group_pc table
	// update model_id with actual db value
	for idx := range modelDef.EntityGroup {

		modelDef.EntityGroup[idx].ModelId = modelDef.Model.ModelId // update model id with db value

		// INSERT INTO entity_group_lst
		//   (model_id, model_entity_id, group_id, group_name, is_hidden)
		// VALUES
		//   (1234, 4, 1, 'Geo_group', 0)
		err = TrxUpdate(trx,
			"INSERT INTO entity_group_lst"+
				" (model_id, model_entity_id, group_id, group_name, is_hidden)"+
				" VALUES ("+
				smId+", "+
				strconv.Itoa(modelDef.EntityGroup[idx].EntityId)+", "+
				strconv.Itoa(modelDef.EntityGroup[idx].GroupId)+", "+
				toQuotedMax(modelDef.EntityGroup[idx].Name, nameDbMax)+", "+
				toBoolSqlConst(modelDef.EntityGroup[idx].IsHidden)+")")
		if err != nil {
			return err
		}

		// insert child rows into entity_group_pc table
		for pcIdx := range modelDef.EntityGroup[idx].GroupPc {

			modelDef.EntityGroup[idx].GroupPc[pcIdx].ModelId = modelDef.Model.ModelId // update model id with db value

			// insert group members rows into entity_group_pc
			// if child group id < 0 or leaf id < then treat it as NULL value
			//
			// INSERT INTO entity_group_pc
			//   (model_id, model_entity_id, group_id, child_pos, child_group_id, attr_id)
			// VALUES
			//   (1234, 4, 1, 0, NULL, 101)
			err = TrxUpdate(trx,
				"INSERT INTO entity_group_pc"+
					" (model_id, model_entity_id, group_id, child_pos, child_group_id, attr_id)"+
					" VALUES ("+
					smId+", "+
					strconv.Itoa(modelDef.EntityGroup[idx].GroupPc[pcIdx].EntityId)+", "+
					strconv.Itoa(modelDef.EntityGroup[idx].GroupPc[pcIdx].GroupId)+", "+
					strconv.Itoa(modelDef.EntityGroup[idx].GroupPc[pcIdx].ChildPos)+", "+
					idOrNullIfNegative(modelDef.EntityGroup[idx].GroupPc[pcIdx].ChildGroupId)+", "+
					idOrNullIfNegative(modelDef.EntityGroup[idx].GroupPc[pcIdx].AttrId)+")")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// sqlCreateParamTable return create table for parameter run values and workset values:
//
// CREATE TABLE ageSex_p20120817
// (
// run_id      INT      NOT NULL,
// sub_id      SMALLINT NOT NULL,
// dim0        INT      NOT NULL,
// dim1        INT      NOT NULL,
// param_value FLOAT    NOT NULL, -- can be NULL
// PRIMARY KEY (run_id, sub_id, dim0, dim1)
// )
//
// CREATE TABLE ageSex_w20120817
// (
// set_id      INT      NOT NULL,
// sub_id      SMALLINT NOT NULL,
// dim0        INT      NOT NULL,
// dim1        INT      NOT NULL,
// param_value FLOAT    NOT NULL, -- can be NULL
// PRIMARY KEY (set_id, sub_id, dim0, dim1)
// )
func sqlCreateParamTable(dbFacet Facet, param *ParamMeta) (string, string, error) {

	colPart := ""
	for k := range param.Dim {
		colPart += param.Dim[k].colName + " INT NOT NULL, "
	}

	tname, err := param.typeOf.sqlColumnType(dbFacet)
	if err != nil {
		return "", "", nil
	}
	if param.IsExtendable {
		colPart += "param_value " + tname + " NULL, " // "extendable" parameter means value can be NULL
	} else {
		colPart += "param_value " + tname + " NOT NULL, "
	}

	keyPart := ""
	for k := range param.Dim {
		keyPart += ", " + param.Dim[k].colName
	}

	rSql := dbFacet.createTableIfNotExist(param.DbRunTable, "("+
		"run_id INT NOT NULL, "+
		"sub_id SMALLINT NOT NULL, "+
		colPart+
		"PRIMARY KEY (run_id, sub_id"+keyPart+")"+
		")")
	wSql := dbFacet.createTableIfNotExist(param.DbSetTable, "("+
		"set_id INT NOT NULL, "+
		"sub_id SMALLINT NOT NULL, "+
		colPart+
		"PRIMARY KEY (set_id, sub_id"+keyPart+")"+
		")")
	return rSql, wSql, nil
}

// sqlCreateOutTable return create table for output table expressions and accumulators:
//
// CREATE TABLE salarySex_v20120820
// (
// run_id     INT      NOT NULL,
// expr_id    SMALLINT NOT NULL,
// dim0       INT      NOT NULL,
// dim1       INT      NOT NULL,
// expr_value FLOAT    NULL,
// PRIMARY KEY (run_id, expr_id, dim0, dim1)
// )
//
// CREATE TABLE salarySex_a20120820
// (
// run_id    INT      NOT NULL,
// acc_id    SMALLINT NOT NULL,
// sub_id    SMALLINT NOT NULL,
// dim0      INT      NOT NULL,
// dim1      INT      NOT NULL,
// acc_value FLOAT    NULL,
// PRIMARY KEY (run_id, acc_id, sub_id, dim0, dim1)
// )
func sqlCreateOutTable(dbFacet Facet, meta *TableMeta) (string, string) {

	colPart := ""
	for k := range meta.Dim {
		colPart += meta.Dim[k].colName + " INT NOT NULL, "
	}

	keyPart := ""
	for k := range meta.Dim {
		keyPart += ", " + meta.Dim[k].colName
	}

	eSql := dbFacet.createTableIfNotExist(meta.DbExprTable, "("+
		"run_id  INT NOT NULL, "+
		"expr_id SMALLINT NOT NULL, "+
		colPart+
		"expr_value "+dbFacet.floatType()+" NULL, "+
		"PRIMARY KEY (run_id, expr_id"+keyPart+")"+
		")")
	aSql := dbFacet.createTableIfNotExist(meta.DbAccTable, "("+
		"run_id INT NOT NULL, "+
		"acc_id SMALLINT NOT NULL, "+
		"sub_id SMALLINT NOT NULL, "+
		colPart+
		"acc_value "+dbFacet.floatType()+" NULL, "+
		"PRIMARY KEY (run_id, acc_id, sub_id"+keyPart+")"+
		")")
	return eSql, aSql
}

// sqlCreateAccAllView return create view for all accumulators view:
//
// CREATE VIEW IF NOT EXISTS T04_FertilityRatesByAgeGroup_d10612268
// AS
// WITH va1 AS
// (
//
//	SELECT
//	  run_id, sub_id, dim0, dim1, acc_value
//	FROM salarySex_a_2012820
//	WHERE acc_id = 1
//
// )
// SELECT
//
//	A.run_id,
//	A.sub_id,
//	A.dim0       AS "Year",
//	A.dim1       AS "Age Group",
//	A.acc_value  AS "Acc0",
//	A1.acc_value AS "Average Income",
//	(
//	  A.acc_value / CASE WHEN ABS(A1.acc_value) > 1.0e-37 THEN A1.acc_value ELSE NULL END
//	) AS "Expr0"
//
// FROM salarySex_a_2012820 A
// INNER JOIN va1 A1 ON (A1.run_id = A.run_id AND A1.sub_id = A.sub_id AND A1.dim0 = A.dim0 AND A1.dim1 = A.dim1)
// WHERE A.acc_id = 0;
func sqlCreateAccAllView(dbFacet Facet, meta *TableMeta) string {

	sql := withPartOfAllAccView(meta)
	if sql != "" {
		sql += " "
	}
	sql += mainSelectOfAllAccView(meta, false) + ";"

	return dbFacet.createViewIfNotExist(meta.DbAccAllView, sql)
}

// sqlAccAllViewAsWith return all accumulators view as WITH cte sql.
// All dimesions, accumulators and expressions columns are AS db column names: dim0, dim1, acc0, expr0
//
// WITH va1 AS
// (
//
//	SELECT .... FROM salarySex_a_2012820 WHERE acc_id = 1
//
// ),
// v_all_acc AS
// (
//
//	SELECT .... FROM salarySex_a_2012820 A INNER JOIN va1 A1 ON (....) WHERE A.acc_id = 0
//
// )
func sqlAccAllViewAsWith(meta *TableMeta) string {

	sql := withPartOfAllAccView(meta)
	if sql != "" {
		sql += ","
	} else {
		sql += "WITH"
	}
	sql += " v_all_acc AS (" + mainSelectOfAllAccView(meta, true) + ")"

	return sql
}

// withPartOfAllAccView return WITH part of all accumulators view.
//
// WITH va1 AS
// (
//
//	SELECT
//	  run_id, sub_id, dim0, dim1, acc_value
//	FROM salarySex_a_2012820
//	WHERE acc_id = 1
//
// )
// SELECT .... FROM salarySex_a_2012820 A INNER JOIN va1 A1 ON (....) WHERE A.acc_id = 0
func withPartOfAllAccView(meta *TableMeta) string {

	// start from WITH for native accumulators CTE, excluding first accumulator
	sql := ""
	for k := range meta.Acc {

		if k < 1 || meta.Acc[k].IsDerived {
			continue // skip first accumulator and all derived accumultors
		}

		if sql == "" {
			sql = "WITH"
		} else {
			sql += ","
		}
		sql += " va" + strconv.Itoa(k) + " AS (" + meta.Acc[k].AccSql + ")"
	}

	return sql
}

// mainSelectOfAllAccView return main SELECT part of all accumulators view.
//
// If isColumnNames is true then use internal db column names: dim0, dim1, acc0, expr0
// else use model definition names: "Year", "Age Group", "Acc0", "Average Income"
//
// WITH va1 AS (....)
// SELECT
//
//	A.run_id,
//	A.sub_id,
//	A.dim0       AS "Year",
//	A.dim1       AS "Age Group",
//	A.acc_value  AS "acc0",
//	A1.acc_value AS "acc1",
//	(
//	  A.acc_value / CASE WHEN ABS(A1.acc_value) > 1.0e-37 THEN A1.acc_value ELSE NULL END
//	) AS "Expr0"
//
// FROM salarySex_a_2012820 A
// INNER JOIN va1 A1 ON (A1.run_id = A.run_id AND A1.sub_id = A.sub_id AND A1.dim0 = A.dim0 AND A1.dim1 = A.dim1)
// WHERE A.acc_id = 0
func mainSelectOfAllAccView(meta *TableMeta, isColumnNames bool) string {

	// start main SELECT body with run id, sub id and dimensions
	sql := "SELECT A.run_id, A.sub_id"

	for k := range meta.Dim {
		sql += ", A." + meta.Dim[k].colName
		if !isColumnNames {
			sql += " AS \"" + meta.Dim[k].Name + "\""
		}
	}

	// append accumulators: A.acc_value AS acc0, A1.acc_value AS "Year", (A.acc_value + A1.acc_value) AS expr0
	for k := range meta.Acc {

		if !meta.Acc[k].IsDerived && k < 1 {
			sql += ", A.acc_value" // first native accumulator alias A.
		}
		if !meta.Acc[k].IsDerived && k >= 1 {
			sql += ", A" + strconv.Itoa(k) + ".acc_value" // all other native accumulators
		}
		if meta.Acc[k].IsDerived {
			sql += ", (" + meta.Acc[k].AccSql + ")" // derived accumulator expression
		}

		if !isColumnNames {
			sql += " AS \"" + meta.Acc[k].Name + "\""
		} else {
			sql += " AS " + meta.Acc[k].colName
		}
	}

	// from accumulator table inner join all CTE for native accumulators
	sql += " FROM " + meta.DbAccTable + " A"

	for k := range meta.Acc {

		if k < 1 || meta.Acc[k].IsDerived {
			continue // skip first accumulator and all derived accumultors
		}
		alias := "A" + strconv.Itoa(k)

		sql += " INNER JOIN va" + strconv.Itoa(k) + " " + alias + " ON (" +
			alias + ".run_id = A.run_id AND " + alias + ".sub_id = A.sub_id"

		for j := range meta.Dim {
			sql += " AND " + alias + "." + meta.Dim[j].colName + " = A." + meta.Dim[j].colName
		}
		sql += ")"
	}

	sql += " WHERE A.acc_id = " + strconv.Itoa(meta.Acc[0].AccId)

	return sql
}

// outTableCreateAccAllView return SELEST for all accumulators view.
// If isColumnNames is true then use internal db column names: dim0, dim1, acc0, expr0
// else use model definition names: "Year", "Age Group", "Acc0", "Average Income"
//
// WITH va1 AS
// (
//
//	SELECT
//	  run_id, sub_id, dim0, dim1, acc_value
//	FROM salarySex_a_2012820
//	WHERE acc_id = 1
//
// )
// SELECT
//
//	A.run_id,
//	A.sub_id,
//	A.dim0       AS "Year",
//	A.dim1       AS "Age Group",
//	A.acc_value  AS "acc0",
//	A1.acc_value AS "acc1",
//	(
//	  A.acc_value / CASE WHEN ABS(A1.acc_value) > 1.0e-37 THEN A1.acc_value ELSE NULL END
//	) AS "Expr0"
//
// FROM salarySex_a_2012820 A
// INNER JOIN va1 A1 ON (A1.run_id = A.run_id AND A1.sub_id = A.sub_id AND A1.dim0 = A.dim0 AND A1.dim1 = A.dim1)
// WHERE A.acc_id = 0
func outTableSelectAccAllView(meta *TableMeta, isColumnNames bool) string {

	// start from WITH for native accumulators CTE, excluding first accumulator
	sql := ""
	for k := range meta.Acc {

		if k < 1 || meta.Acc[k].IsDerived {
			continue // skip first accumulator and all derived accumultors
		}

		if sql == "" {
			sql = "WITH"
		} else {
			sql += ","
		}
		sql += " va" + strconv.Itoa(k) + " (" + meta.Acc[k].AccSql + ")"
	}

	// start main SELECT body with run id, sub id and dimensions
	if sql != "" {
		sql += " "
	}
	sql += "SELECT A.run_id, A.sub_id"

	for k := range meta.Dim {
		sql += ", A." + meta.Dim[k].colName
		if !isColumnNames {
			sql += " AS \"" + meta.Dim[k].Name + "\""
		}
	}

	// append accumulators: A.acc_value AS acc0, A1.acc_value AS "Year", (A.acc_value + A1.acc_value) AS expr0
	for k := range meta.Acc {

		if !meta.Acc[k].IsDerived && k < 1 {
			sql += ", A.acc_value" // first native accumulator alias A.
		}
		if !meta.Acc[k].IsDerived && k >= 1 {
			sql += ", A" + strconv.Itoa(k) + ".acc_value" // all other native accumulators
		}
		if meta.Acc[k].IsDerived {
			sql += ", (" + meta.Acc[k].AccSql + ")" // derived accumulator expression
		}

		if !isColumnNames {
			sql += " AS \"" + meta.Acc[k].Name + "\""
		} else {
			sql += " AS " + meta.Acc[k].colName
		}
	}

	// from accumulator table inner join all CTE for native accumulators
	sql += " FROM " + meta.DbAccTable + " A"

	for k := range meta.Acc {

		if k < 1 || meta.Acc[k].IsDerived {
			continue // skip first accumulator and all derived accumultors
		}
		alias := "A" + strconv.Itoa(k)

		sql += " INNER JOIN va" + strconv.Itoa(k) + " " + alias + " ON (" +
			alias + ".run_id = A.run_id AND " + alias + ".sub_id = A.sub_id"

		for j := range meta.Dim {
			sql += " AND " + alias + "." + meta.Dim[j].colName + " = A." + meta.Dim[j].colName
		}
		sql += ")"
	}

	sql += " WHERE A.acc_id = " + strconv.Itoa(meta.Acc[0].AccId)

	return sql
}
