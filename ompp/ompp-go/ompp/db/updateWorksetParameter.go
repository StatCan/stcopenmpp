// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/helper"
)

// UpdateWorksetParameterFrom add new or replace existing workset parameter.
//
// If parameter already exist in workset then parameter metadata either merged or replaced.
// If parameter not exist in workset then parameter values must be supplied, it cannot be empty.
// If parameter not exist in workset then new parameter metadata and values inserted.
// If from() not nil then parameter values completely replaced, on each call from() must return CellParam for all parameter values.
//
// Set name is used to find workset and set id updated with actual database value.
// Workset must be read-write for replace or merge.
func (meta *WorksetMeta) UpdateWorksetParameterFrom(
	dbConn *sql.DB, modelDef *ModelMeta, isReplaceMeta bool, param *ParamRunSetPub, langDef *LangMeta, from func() (interface{}, error),
) (int, error) {

	// validate parameters
	if modelDef == nil {
		return 0, errors.New("invalid (empty) model metadata")
	}
	if param == nil {
		return 0, errors.New("invalid (empty) parameter metadata")
	}
	if langDef == nil {
		return 0, errors.New("invalid (empty) language list")
	}
	if meta.Set.Name == "" {
		return 0, errors.New("invalid (empty) workset name")
	}
	if meta.Set.ModelId != modelDef.Model.ModelId {
		return 0, errors.New("workset: " + meta.Set.Name + " invalid model id " + strconv.Itoa(meta.Set.ModelId) + " expected: " + strconv.Itoa(modelDef.Model.ModelId))
	}

	// do update in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return 0, err
	}

	// if parameter values supplied then sub-value count must be positive
	isData := from != nil
	if isData && param.SubCount <= 0 {
		return 0, errors.New("parameter sub-value count must be positive: " + strconv.Itoa(param.SubCount) + ": " + param.Name)
	}

	// create, replace or merge workset metadata
	paramHid, err := doUpdateWorksetParameterMeta(trx, modelDef, meta, isReplaceMeta, param, isData, langDef)
	if err != nil {
		trx.Rollback()
		return 0, err
	}

	// if new parameter values supplied then insert or update parameter values in workset
	if paramHid > 0 && isData {

		var pm *ParamMeta
		if k, ok := modelDef.ParamByName(param.Name); ok {
			pm = &modelDef.Param[k]
		} else {
			trx.Rollback()
			return 0, errors.New("parameter not found: " + param.Name)
		}

		err = doWriteSetParameterFrom(trx, pm, meta.Set.SetId, param.SubCount, param.DefaultSubId, false, from, "")
		if err != nil {
			trx.Rollback()
			return 0, err
		}
	}

	trx.Commit()

	return paramHid, nil
}

// UpdateWorksetParameterText merge parameter value notes into workset_parameter_txt table.
//
// Set name is used to find workset and set id updated with actual database value.
// Workset must exist and must be read-write for replace or merge.
//
// Parameter must exist exist in the model otherwise it is an error.
// If parameter not exist in workset then function does nothing (it is empty operation).
// If input array of ParamRunSetTxtPub is empty then it is empty operation and return is success.
func UpdateWorksetParameterText(dbConn *sql.DB, modelDef *ModelMeta, setName string, paramTxtPub []ParamRunSetTxtPub, langDef *LangMeta) error {

	// validate parameters
	if len(paramTxtPub) <= 0 {
		return nil // nothing to be updated: return success
	}
	if modelDef == nil {
		return errors.New("invalid (empty) model metadata")
	}
	if langDef == nil {
		return errors.New("invalid (empty) language list")
	}
	if setName == "" {
		return errors.New("invalid (empty) workset name")
	}

	// convert from public to internal array of parameter value notes
	// and check: parameter name must exist in the model
	paramLst := make([]worksetParam, len(paramTxtPub))
	for k := range paramTxtPub {
		np, ok := modelDef.ParamByName(paramTxtPub[k].Name)
		if !ok {
			return errors.New("model parameter not found: " + paramTxtPub[k].Name)
		}
		paramLst[k].ParamHid = modelDef.Param[np].ParamHid
		paramLst[k].Txt = make([]WorksetParamTxtRow, len(paramTxtPub[k].Txt))

		for j := range paramTxtPub[k].Txt {
			paramLst[k].Txt[j].ParamHid = paramLst[k].ParamHid
			paramLst[k].Txt[j].LangCode = paramTxtPub[k].Txt[j].LangCode
			paramLst[k].Txt[j].Note = paramTxtPub[k].Txt[j].Note
		}
	}

	// do update in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	err = doUpdateWorksetParameterText(trx, modelDef, setName, paramLst, langDef)
	if err != nil {
		trx.Rollback()
		return err
	}
	trx.Commit()

	return nil
}

// doUpdateWorksetParameterMeta insert new or update existing workset parameter metadata.
// It does update as part of transaction.
//
// If parameter already exist in workset then parameter metadata either merged or replaced.
// If parameter not exist in workset then parameter values must be supplied, it cannot be empty.
// If parameter not exist in workset then new parameter metadata and values inserted.
// If list of new parameter values supplied then parameter values completely replaced.
//
// Set name is used to find workset and set id updated with actual database value.
// Workset must be read-write for replace or merge.
func doUpdateWorksetParameterMeta(
	trx *sql.Tx, modelDef *ModelMeta, wm *WorksetMeta, isReplaceMeta bool, param *ParamRunSetPub, isData bool, langDef *LangMeta,
) (int, error) {

	// find model parameter hId by name
	idx, ok := modelDef.ParamByName(param.Name)
	if !ok {
		return 0, errors.New("model: " + modelDef.Model.Name + " parameter " + param.Name + " not found")
	}
	paramHid := modelDef.Param[idx].ParamHid
	spHid := strconv.Itoa(paramHid)

	// "lock" workset to prevent update or use by the model
	wm.Set.ModelId = modelDef.Model.ModelId // update model id

	err := TrxUpdate(trx,
		"UPDATE workset_lst"+
			" SET is_readonly = is_readonly + 1"+
			" WHERE model_id = "+strconv.Itoa(modelDef.Model.ModelId)+" AND set_name = "+ToQuoted(wm.Set.Name))
	if err != nil {
		return 0, err
	}

	// check if workset exist and not readonly
	var setId, nRd int
	err = TrxSelectFirst(trx,
		"SELECT set_id, is_readonly FROM workset_lst"+
			" WHERE model_id = "+strconv.Itoa(modelDef.Model.ModelId)+" AND set_name = "+ToQuoted(wm.Set.Name),
		func(row *sql.Row) error {
			if err := row.Scan(&setId, &nRd); err != nil {
				return err
			}
			return nil
		})
	switch {
	case err == sql.ErrNoRows:
		return 0, errors.New("failed to update: workset not found: " + wm.Set.Name)
	case err != nil:
		return 0, err
	case nRd != 1:
		return 0, errors.New("failed to update: workset is read-only: " + wm.Set.Name)
	}
	wm.Set.SetId = setId // workset exist, id may be different

	// check if parameter exist in workset_parameter
	sId := strconv.Itoa(wm.Set.SetId)

	if isData {
		err = TrxUpdate(trx, "UPDATE workset_parameter"+
			" SET sub_count = "+strconv.Itoa(param.SubCount)+","+
			" default_sub_id = "+strconv.Itoa(param.DefaultSubId)+
			" WHERE set_id = "+sId+" AND parameter_hid = "+spHid)
	} else {
		err = TrxUpdate(trx, "UPDATE workset_parameter"+
			" SET parameter_hid = "+spHid+
			" WHERE set_id = "+sId+" AND parameter_hid = "+spHid)
	}
	if err != nil {
		return 0, err
	}
	err = TrxSelectFirst(trx,
		"SELECT parameter_hid FROM workset_parameter WHERE set_id = "+sId+" AND parameter_hid = "+spHid,
		func(row *sql.Row) error {
			h := 0
			if err := row.Scan(&h); err != nil {
				return err
			}
			return nil

		})
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	isParamExist := err == nil // not sql.ErrNoRows

	// if no data then parameter must exist in workset
	if !isData && !isParamExist {
		return 0, errors.New("parameter: " + param.Name + " must exist in workset: " + wm.Set.Name)
	}

	// if this is merge then insert ot update workset_parameter
	// if this is replace then delete existing workset_parameter_txt and delete-insert workset_parameter
	if !isReplaceMeta {

		// parameter not exist in workset: do insert
		if !isParamExist {
			err = TrxUpdate(trx,
				"INSERT INTO workset_parameter (set_id, parameter_hid, sub_count, default_sub_id) VALUES ("+
					sId+", "+spHid+", "+strconv.Itoa(param.SubCount)+", "+strconv.Itoa(param.DefaultSubId)+")")
			if err != nil {
				return 0, err
			}
		}

	} else { // repalce existing parameter metadata: delete text and delete-insert workset_parameter

		err = TrxUpdate(trx, "DELETE FROM workset_parameter_txt WHERE set_id = "+sId+" AND parameter_hid = "+spHid)
		if err != nil {
			return 0, err
		}
		err = TrxUpdate(trx, "DELETE FROM workset_parameter WHERE set_id = "+sId+" AND parameter_hid = "+spHid)
		if err != nil {
			return 0, err
		}
		err = TrxUpdate(trx,
			"INSERT INTO workset_parameter (set_id, parameter_hid, sub_count, default_sub_id) VALUES ("+
				sId+", "+spHid+", "+strconv.Itoa(param.SubCount)+", "+strconv.Itoa(param.DefaultSubId)+")")
		if err != nil {
			return 0, err
		}
	}

	// delete and insert into workset parameter text: parameter value notes
	for j := range param.Txt {

		if lId, ok := langDef.IdByCode(param.Txt[j].LangCode); ok { // if language code valid

			slId := strconv.Itoa(lId)

			if !isReplaceMeta {
				err = TrxUpdate(trx,
					"DELETE FROM workset_parameter_txt"+
						" WHERE set_id = "+sId+
						" AND parameter_hid = "+spHid+
						" AND lang_id = "+slId)
				if err != nil {
					return 0, err
				}
			}
			err = TrxUpdate(trx,
				"INSERT INTO workset_parameter_txt (set_id, parameter_hid, lang_id, note)"+
					" SELECT "+
					sId+", "+" parameter_hid, "+slId+", "+toQuotedOrNullMax(param.Txt[j].Note, noteDbMax)+
					" FROM workset_parameter"+
					" WHERE set_id = "+sId+
					" AND parameter_hid = "+spHid)
			if err != nil {
				return 0, err
			}
		}
	}

	// "unlock" workset for parameter values write: restore original value of is_readonly=0
	wm.Set.UpdateDateTime = helper.MakeDateTime(time.Now())
	err = TrxUpdate(trx,
		"UPDATE workset_lst"+
			" SET is_readonly = 0,"+
			" update_dt = "+ToQuoted(wm.Set.UpdateDateTime)+
			" WHERE set_id = "+strconv.Itoa(setId))
	if err != nil {
		return 0, err
	}

	return paramHid, nil
}

// doUpdateWorksetParameterText merge workset parameters value notes.
// It does update as part of transaction.
//
// Set name is used to find workset and set id updated with actual database value.
// Workset must exist and must be read-write for replace or merge.
//
// If parameter not exist in workset then function does nothing (it is empty operation).
func doUpdateWorksetParameterText(trx *sql.Tx, modelDef *ModelMeta, setName string, paramLst []worksetParam, langDef *LangMeta) error {

	// "lock" workset to prevent update or use by the model
	err := TrxUpdate(trx,
		"UPDATE workset_lst"+
			" SET is_readonly = is_readonly + 1"+
			" WHERE model_id = "+strconv.Itoa(modelDef.Model.ModelId)+" AND set_name = "+ToQuoted(setName))
	if err != nil {
		return err
	}

	// check if workset exist and not readonly
	var setId, nRd int
	err = TrxSelectFirst(trx,
		"SELECT set_id, is_readonly FROM workset_lst"+
			" WHERE model_id = "+strconv.Itoa(modelDef.Model.ModelId)+" AND set_name = "+ToQuoted(setName),
		func(row *sql.Row) error {
			if err := row.Scan(&setId, &nRd); err != nil {
				return err
			}
			return nil
		})
	switch {
	case err == sql.ErrNoRows:
		return errors.New("failed to update: workset not found: " + setName)
	case err != nil:
		return err
	case nRd != 1:
		return errors.New("failed to update: workset is read-only: " + setName)
	}
	sId := strconv.Itoa(setId)

	// merge parameter(s) value notes
	for k := range paramLst {

		spHid := strconv.Itoa(paramLst[k].ParamHid)

		for j := range paramLst[k].Txt {

			if lId, ok := langDef.IdByCode(paramLst[k].Txt[j].LangCode); ok { // if language code valid

				slId := strconv.Itoa(lId)

				err = TrxUpdate(trx,
					"DELETE FROM workset_parameter_txt"+
						" WHERE set_id = "+sId+
						" AND parameter_hid = "+spHid+
						" AND lang_id = "+slId)
				if err != nil {
					return err
				}
				err = TrxUpdate(trx,
					"INSERT INTO workset_parameter_txt (set_id, parameter_hid, lang_id, note)"+
						" SELECT "+
						sId+", "+" parameter_hid, "+slId+", "+toQuotedOrNullMax(paramLst[k].Txt[j].Note, noteDbMax)+
						" FROM workset_parameter"+
						" WHERE set_id = "+sId+
						" AND parameter_hid = "+spHid)
				if err != nil {
					return err
				}
			}
		}
	}

	// "unlock" workset for parameter values write: restore original value of is_readonly=0
	err = TrxUpdate(trx,
		"UPDATE workset_lst"+
			" SET is_readonly = 0,"+
			" update_dt = "+ToQuoted(helper.MakeDateTime(time.Now()))+
			" WHERE set_id = "+sId)
	if err != nil {
		return err
	}

	return nil
}
