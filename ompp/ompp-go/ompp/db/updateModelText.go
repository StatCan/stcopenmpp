// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"errors"
	"strconv"
)

// delete existing and insert new model text (description and notes) in database.
//
// Model id, type Hid, parameter Hid, table Hid, language id updated with actual database id's
func UpdateModelText(dbConn *sql.DB, modelDef *ModelMeta, langDef *LangMeta, modelTxt *ModelTxtMeta) error {

	// validate parameters
	if modelTxt == nil {
		return nil // source is empty: nothing to do, exit
	}
	if modelDef == nil {
		return errors.New("invalid (empty) model metadata")
	}
	if langDef == nil {
		return errors.New("invalid (empty) language list")
	}
	if modelTxt.ModelName != modelDef.Model.Name || modelTxt.ModelDigest != modelDef.Model.Digest {
		return errors.New("invalid model name " + modelTxt.ModelName + " or digest " + modelTxt.ModelDigest + " expected: " + modelDef.Model.Name + " " + modelDef.Model.Digest)
	}

	// do update in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	if err = doUpdateModelText(trx, modelDef, langDef, modelTxt); err != nil {
		trx.Rollback()
		return err
	}
	trx.Commit()
	return nil
}

// delete existing and insert new model text (description and notes) in database.
// It does update as part of transaction
// Model id, type Hid, parameter Hid, entity Hid, table Hid, language id updated with actual database id's
func doUpdateModelText(trx *sql.Tx, modelDef *ModelMeta, langDef *LangMeta, modelTxt *ModelTxtMeta) error {

	// update model_dic_txt and ids
	smId := strconv.Itoa(modelDef.Model.ModelId)
	for idx := range modelTxt.ModelTxt {

		modelTxt.ModelTxt[idx].ModelId = modelDef.Model.ModelId // update model id

		// if language code valid then delete and insert into model_dic_txt
		if lId, ok := langDef.IdByCode(modelTxt.ModelTxt[idx].LangCode); ok {

			err := TrxUpdate(trx,
				"DELETE FROM model_dic_txt WHERE model_id = "+smId+" AND lang_id = "+strconv.Itoa(lId))
			if err != nil {
				return err
			}
			err = TrxUpdate(trx,
				"INSERT INTO model_dic_txt (model_id, lang_id, descr, note) VALUES ("+
					smId+", "+
					strconv.Itoa(lId)+", "+
					toQuotedMax(modelTxt.ModelTxt[idx].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(modelTxt.ModelTxt[idx].Note, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	// update type_dic_txt and ids
	prevHid := -1
	for idx := range modelTxt.TypeTxt {

		modelTxt.TypeTxt[idx].ModelId = modelDef.Model.ModelId // update model id

		// find type Hid
		k, ok := modelDef.TypeByKey(modelTxt.TypeTxt[idx].TypeId)
		if !ok {
			return errors.New("invalid type id " + strconv.Itoa(modelTxt.TypeTxt[idx].TypeId))
		}
		hId := modelDef.Type[k].TypeHid

		// delete existing enum text and type text
		if hId != prevHid {
			err := TrxUpdate(trx, "DELETE FROM type_enum_txt WHERE type_hid = "+strconv.Itoa(hId))
			if err != nil {
				return err
			}
			err = TrxUpdate(trx, "DELETE FROM type_dic_txt WHERE type_hid = "+strconv.Itoa(hId))
			if err != nil {
				return err
			}
			prevHid = hId
		}

		// if language code valid then delete and insert into type_dic_txt
		if lId, ok := langDef.IdByCode(modelTxt.TypeTxt[idx].LangCode); ok {

			err := TrxUpdate(trx,
				"INSERT INTO type_dic_txt (type_hid, lang_id, descr, note) VALUES ("+
					strconv.Itoa(hId)+", "+
					strconv.Itoa(lId)+", "+
					toQuotedMax(modelTxt.TypeTxt[idx].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(modelTxt.TypeTxt[idx].Note, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	// update type_enum_txt and ids
	for idx := range modelTxt.TypeEnumTxt {

		modelTxt.TypeEnumTxt[idx].ModelId = modelDef.Model.ModelId // update model id

		// skip if description and notes are empty
		if modelTxt.TypeEnumTxt[idx].Descr == "" && modelTxt.TypeEnumTxt[idx].Note == "" {
			continue
		}

		// find type Hid
		k, ok := modelDef.TypeByKey(modelTxt.TypeEnumTxt[idx].TypeId)
		if !ok {
			return errors.New("invalid type id " + strconv.Itoa(modelTxt.TypeEnumTxt[idx].TypeId))
		}
		hId := modelDef.Type[k].TypeHid

		// if language code valid then delete and insert into type_enum_txt
		if lId, ok := langDef.IdByCode(modelTxt.TypeEnumTxt[idx].LangCode); ok {

			err := TrxUpdate(trx,
				"INSERT INTO type_enum_txt (type_hid, enum_id, lang_id, descr, note) VALUES ("+
					strconv.Itoa(hId)+", "+
					strconv.Itoa(modelTxt.TypeEnumTxt[idx].EnumId)+", "+
					strconv.Itoa(lId)+", "+
					toQuotedMax(modelTxt.TypeEnumTxt[idx].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(modelTxt.TypeEnumTxt[idx].Note, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	// update parameter_dic_txt and ids
	for idx := range modelTxt.ParamTxt {

		modelTxt.ParamTxt[idx].ModelId = modelDef.Model.ModelId // update model id

		// find parameter Hid
		hId := modelDef.ParamHidById(modelTxt.ParamTxt[idx].ParamId)
		if hId <= 0 {
			return errors.New("invalid parameter id " + strconv.Itoa(modelTxt.ParamTxt[idx].ParamId))
		}

		// if language code valid then delete and insert into parameter_dic_txt
		if lId, ok := langDef.IdByCode(modelTxt.ParamTxt[idx].LangCode); ok {

			err := TrxUpdate(trx,
				"DELETE FROM parameter_dic_txt"+
					" WHERE parameter_hid = "+strconv.Itoa(hId)+
					" AND lang_id = "+strconv.Itoa(lId))
			if err != nil {
				return err
			}
			err = TrxUpdate(trx,
				"INSERT INTO parameter_dic_txt (parameter_hid, lang_id, descr, note) VALUES ("+
					strconv.Itoa(hId)+", "+
					strconv.Itoa(lId)+", "+
					toQuotedMax(modelTxt.ParamTxt[idx].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(modelTxt.ParamTxt[idx].Note, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	// update parameter_dims_txt and ids
	for idx := range modelTxt.ParamDimsTxt {

		modelTxt.ParamDimsTxt[idx].ModelId = modelDef.Model.ModelId // update model id

		// find parameter Hid
		hId := modelDef.ParamHidById(modelTxt.ParamDimsTxt[idx].ParamId)
		if hId <= 0 {
			return errors.New("invalid parameter id " + strconv.Itoa(modelTxt.ParamDimsTxt[idx].ParamId))
		}

		// if language code valid then delete and insert into parameter_dims_txt
		if lId, ok := langDef.IdByCode(modelTxt.ParamDimsTxt[idx].LangCode); ok {

			err := TrxUpdate(trx,
				"DELETE FROM parameter_dims_txt"+
					" WHERE parameter_hid = "+strconv.Itoa(hId)+
					" AND dim_id = "+strconv.Itoa(modelTxt.ParamDimsTxt[idx].DimId)+
					" AND lang_id = "+strconv.Itoa(lId))
			if err != nil {
				return err
			}
			err = TrxUpdate(trx,
				"INSERT INTO parameter_dims_txt (parameter_hid, dim_id, lang_id, descr, note) VALUES ("+
					strconv.Itoa(hId)+", "+
					strconv.Itoa(modelTxt.ParamDimsTxt[idx].DimId)+", "+
					strconv.Itoa(lId)+", "+
					toQuotedMax(modelTxt.ParamDimsTxt[idx].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(modelTxt.ParamDimsTxt[idx].Note, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	// update table_dic_txt and ids
	for idx := range modelTxt.TableTxt {

		modelTxt.TableTxt[idx].ModelId = modelDef.Model.ModelId // update model id

		// find output table Hid
		hId := modelDef.OutTableHidById(modelTxt.TableTxt[idx].TableId)
		if hId <= 0 {
			return errors.New("invalid output table id " + strconv.Itoa(modelTxt.TableTxt[idx].TableId))
		}

		// if language code valid then delete and insert into table_dic_txt
		if lId, ok := langDef.IdByCode(modelTxt.TableTxt[idx].LangCode); ok {

			err := TrxUpdate(trx,
				"DELETE FROM table_dic_txt"+
					" WHERE table_hid = "+strconv.Itoa(hId)+
					" AND lang_id = "+strconv.Itoa(lId))
			if err != nil {
				return err
			}
			err = TrxUpdate(trx,
				"INSERT INTO table_dic_txt (table_hid, lang_id, descr, note, expr_descr, expr_note) VALUES ("+
					strconv.Itoa(hId)+", "+
					strconv.Itoa(lId)+", "+
					toQuotedMax(modelTxt.TableTxt[idx].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(modelTxt.TableTxt[idx].Note, noteDbMax)+", "+
					toQuotedMax(modelTxt.TableTxt[idx].ExprDescr, descrDbMax)+", "+
					toQuotedOrNullMax(modelTxt.TableTxt[idx].ExprNote, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	// update table_dims_txt and ids
	for idx := range modelTxt.TableDimsTxt {

		modelTxt.TableDimsTxt[idx].ModelId = modelDef.Model.ModelId // update model id

		// find output table Hid
		hId := modelDef.OutTableHidById(modelTxt.TableDimsTxt[idx].TableId)
		if hId <= 0 {
			return errors.New("invalid output table id " + strconv.Itoa(modelTxt.TableDimsTxt[idx].TableId))
		}

		// if language code valid then delete and insert into table_dims_txt
		if lId, ok := langDef.IdByCode(modelTxt.TableDimsTxt[idx].LangCode); ok {

			err := TrxUpdate(trx,
				"DELETE FROM table_dims_txt"+
					" WHERE table_hid = "+strconv.Itoa(hId)+
					" AND dim_id = "+strconv.Itoa(modelTxt.TableDimsTxt[idx].DimId)+
					" AND lang_id = "+strconv.Itoa(lId))
			if err != nil {
				return err
			}
			err = TrxUpdate(trx,
				"INSERT INTO table_dims_txt (table_hid, dim_id, lang_id, descr, note) VALUES ("+
					strconv.Itoa(hId)+", "+
					strconv.Itoa(modelTxt.TableDimsTxt[idx].DimId)+", "+
					strconv.Itoa(lId)+", "+
					toQuotedMax(modelTxt.TableDimsTxt[idx].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(modelTxt.TableDimsTxt[idx].Note, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	// update table_acc_txt and ids
	for idx := range modelTxt.TableAccTxt {

		modelTxt.TableAccTxt[idx].ModelId = modelDef.Model.ModelId // update model id

		// find output table Hid
		hId := modelDef.OutTableHidById(modelTxt.TableAccTxt[idx].TableId)
		if hId <= 0 {
			return errors.New("invalid output table id " + strconv.Itoa(modelTxt.TableAccTxt[idx].TableId))
		}

		// if language code valid then delete and insert into table_acc_txt
		if lId, ok := langDef.IdByCode(modelTxt.TableAccTxt[idx].LangCode); ok {

			err := TrxUpdate(trx,
				"DELETE FROM table_acc_txt"+
					" WHERE table_hid = "+strconv.Itoa(hId)+
					" AND acc_id = "+strconv.Itoa(modelTxt.TableAccTxt[idx].AccId)+
					" AND lang_id = "+strconv.Itoa(lId))
			if err != nil {
				return err
			}
			err = TrxUpdate(trx,
				"INSERT INTO table_acc_txt (table_hid, acc_id, lang_id, descr, note) VALUES ("+
					strconv.Itoa(hId)+", "+
					strconv.Itoa(modelTxt.TableAccTxt[idx].AccId)+", "+
					strconv.Itoa(lId)+", "+
					toQuotedMax(modelTxt.TableAccTxt[idx].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(modelTxt.TableAccTxt[idx].Note, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	// update table_expr_txt and ids
	for idx := range modelTxt.TableExprTxt {

		modelTxt.TableExprTxt[idx].ModelId = modelDef.Model.ModelId // update model id

		// find output table Hid
		hId := modelDef.OutTableHidById(modelTxt.TableExprTxt[idx].TableId)
		if hId <= 0 {
			return errors.New("invalid output table id " + strconv.Itoa(modelTxt.TableExprTxt[idx].TableId))
		}

		// if language code valid then delete and insert into table_expr_txt
		if lId, ok := langDef.IdByCode(modelTxt.TableExprTxt[idx].LangCode); ok {

			err := TrxUpdate(trx,
				"DELETE FROM table_expr_txt"+
					" WHERE table_hid = "+strconv.Itoa(hId)+
					" AND expr_id = "+strconv.Itoa(modelTxt.TableExprTxt[idx].ExprId)+
					" AND lang_id = "+strconv.Itoa(lId))
			if err != nil {
				return err
			}
			err = TrxUpdate(trx,
				"INSERT INTO table_expr_txt (table_hid, expr_id, lang_id, descr, note) VALUES ("+
					strconv.Itoa(hId)+", "+
					strconv.Itoa(modelTxt.TableExprTxt[idx].ExprId)+", "+
					strconv.Itoa(lId)+", "+
					toQuotedMax(modelTxt.TableExprTxt[idx].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(modelTxt.TableExprTxt[idx].Note, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	// update entity_dic_txt and ids
	for idx := range modelTxt.EntityTxt {

		modelTxt.EntityTxt[idx].ModelId = modelDef.Model.ModelId // update model id

		// find entity Hid
		hId := modelDef.EntityHidById(modelTxt.EntityTxt[idx].EntityId)
		if hId <= 0 {
			return errors.New("invalid entity id " + strconv.Itoa(modelTxt.EntityTxt[idx].EntityId))
		}

		// if language code valid then delete and insert into entity_dic_txt
		if lId, ok := langDef.IdByCode(modelTxt.EntityTxt[idx].LangCode); ok {

			err := TrxUpdate(trx,
				"DELETE FROM entity_dic_txt"+
					" WHERE entity_hid = "+strconv.Itoa(hId)+
					" AND lang_id = "+strconv.Itoa(lId))
			if err != nil {
				return err
			}
			err = TrxUpdate(trx,
				"INSERT INTO entity_dic_txt (entity_hid, lang_id, descr, note) VALUES ("+
					strconv.Itoa(hId)+", "+
					strconv.Itoa(lId)+", "+
					toQuotedMax(modelTxt.EntityTxt[idx].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(modelTxt.EntityTxt[idx].Note, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	// update entity_attr_txt and ids
	for idx := range modelTxt.EntityAttrTxt {

		modelTxt.EntityAttrTxt[idx].ModelId = modelDef.Model.ModelId // update model id

		// find entity Hid
		hId := modelDef.EntityHidById(modelTxt.EntityAttrTxt[idx].EntityId)
		if hId <= 0 {
			return errors.New("invalid entity id " + strconv.Itoa(modelTxt.EntityAttrTxt[idx].EntityId))
		}

		// if language code valid then delete and insert into entity_attr_txt
		if lId, ok := langDef.IdByCode(modelTxt.EntityAttrTxt[idx].LangCode); ok {

			err := TrxUpdate(trx,
				"DELETE FROM entity_attr_txt"+
					" WHERE entity_hid = "+strconv.Itoa(hId)+
					" AND attr_id = "+strconv.Itoa(modelTxt.EntityAttrTxt[idx].AttrId)+
					" AND lang_id = "+strconv.Itoa(lId))
			if err != nil {
				return err
			}
			err = TrxUpdate(trx,
				"INSERT INTO entity_attr_txt (entity_hid, attr_id, lang_id, descr, note) VALUES ("+
					strconv.Itoa(hId)+", "+
					strconv.Itoa(modelTxt.EntityAttrTxt[idx].AttrId)+", "+
					strconv.Itoa(lId)+", "+
					toQuotedMax(modelTxt.EntityAttrTxt[idx].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(modelTxt.EntityAttrTxt[idx].Note, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	// update group_txt and ids
	for idx := range modelTxt.GroupTxt {

		modelTxt.GroupTxt[idx].ModelId = modelDef.Model.ModelId // update model id
		sGrpId := strconv.Itoa(modelTxt.GroupTxt[idx].GroupId)

		// if language code valid then delete and insert into group_txt
		if lId, ok := langDef.IdByCode(modelTxt.GroupTxt[idx].LangCode); ok {

			err := TrxUpdate(trx,
				"DELETE FROM group_txt WHERE model_id = "+smId+" AND group_id = "+sGrpId+" AND lang_id = "+strconv.Itoa(lId))
			if err != nil {
				return err
			}
			err = TrxUpdate(trx,
				"INSERT INTO group_txt (model_id, group_id, lang_id, descr, note)"+
					" VALUES ("+
					smId+", "+
					sGrpId+", "+
					strconv.Itoa(lId)+", "+
					toQuotedMax(modelTxt.GroupTxt[idx].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(modelTxt.GroupTxt[idx].Note, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	// update entity_group_txt and ids
	for idx := range modelTxt.EntityGroupTxt {

		modelTxt.EntityGroupTxt[idx].ModelId = modelDef.Model.ModelId // update model id
		sEntId := strconv.Itoa(modelTxt.EntityGroupTxt[idx].EntityId)
		sGrpId := strconv.Itoa(modelTxt.EntityGroupTxt[idx].GroupId)

		// if language code valid then delete and insert into entity_group_txt
		if lId, ok := langDef.IdByCode(modelTxt.EntityGroupTxt[idx].LangCode); ok {

			err := TrxUpdate(trx,
				"DELETE FROM entity_group_txt WHERE model_id = "+smId+" AND model_entity_id = "+sEntId+" AND group_id = "+sGrpId+" AND lang_id = "+strconv.Itoa(lId))
			if err != nil {
				return err
			}
			err = TrxUpdate(trx,
				"INSERT INTO entity_group_txt (model_id, model_entity_id, group_id, lang_id, descr, note)"+
					" VALUES ("+
					smId+", "+
					sEntId+", "+
					sGrpId+", "+
					strconv.Itoa(lId)+", "+
					toQuotedMax(modelTxt.EntityGroupTxt[idx].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(modelTxt.EntityGroupTxt[idx].Note, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	return nil
}
