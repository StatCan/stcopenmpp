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

// UpdateWorksetReadonly update workset readonly status.
func UpdateWorksetReadonly(dbConn *sql.DB, setId int, isReadonly bool) error {

	return Update(dbConn,
		"UPDATE workset_lst"+
			" SET is_readonly = "+toBoolSqlConst(isReadonly)+", "+" update_dt = "+ToQuoted(helper.MakeDateTime(time.Now()))+
			" WHERE set_id ="+strconv.Itoa(setId))
}

// UpdateWorksetReadonlyByName update workset readonly status by workset name.
func UpdateWorksetReadonlyByName(dbConn *sql.DB, modelId int, name string, isReadonly bool) error {

	return Update(dbConn,
		"UPDATE workset_lst"+
			" SET is_readonly = "+toBoolSqlConst(isReadonly)+", "+" update_dt = "+ToQuoted(helper.MakeDateTime(time.Now()))+
			" WHERE set_id = ("+
			" SELECT MIN(W.set_id) FROM workset_lst W"+
			" WHERE W.model_id = "+strconv.Itoa(modelId)+
			" AND set_name = "+ToQuoted(name)+
			")")
}

// RenameWorkset do rename workset if new name is not empty "" string.
//
// If isAnyReadonly parameter is true then do rename regarless of workset read-only status
// else return error if workset is not read-write.
func RenameWorkset(dbConn *sql.DB, setId int, newSetName string, isAnyReadonly bool) (bool, error) {

	if newSetName == "" {
		return false, nil // exit if new name is empty: nothing to do
	}

	// validate parameters
	if setId <= 0 {
		return false, errors.New("invalid workset id: " + strconv.Itoa(setId))
	}

	// do update in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return false, err
	}
	isFound, err := doRenameWorkset(trx, setId, newSetName, isAnyReadonly)

	if err != nil {
		trx.Rollback()
		return false, err
	}
	trx.Commit()

	return isFound, nil
}

// doRenameWorkset rename workset if new name is not empty "" string.
// if isAnyReadonly parameter is true then ignore workset read-only status
// else return error if workset is not read-write
func doRenameWorkset(trx *sql.Tx, setId int, newSetName string, isAnyReadonly bool) (bool, error) {

	if newSetName == "" {
		return false, nil // exit if new name is empty: nothing to do
	}

	// "lock" workset to prevent update or use by the model
	sId := strconv.Itoa(setId)

	err := TrxUpdate(trx, "UPDATE workset_lst SET is_readonly = is_readonly + 1 WHERE set_id = "+sId)
	if err != nil {
		return false, err
	}

	// check if workset exist and not readonly
	var nRd int = 0
	err = TrxSelectFirst(trx,
		"SELECT is_readonly FROM workset_lst WHERE set_id = "+sId,
		func(row *sql.Row) error {
			if err := row.Scan(&nRd); err != nil {
				return err
			}
			return nil
		})
	switch {
	case err == sql.ErrNoRows:
		return false, nil // workset not found: nothing to do
	case err != nil:
		return false, err
	case !isAnyReadonly && nRd != 1: // return error only if read-only validation required
		return false, errors.New("failed to update: workset is read-only: " + sId)
	}

	// check if new name is unique
	err = TrxSelectFirst(trx,
		"SELECT COUNT(*) FROM workset_lst WHERE set_id <> "+sId+" AND set_name = "+ToQuoted(newSetName),
		func(row *sql.Row) error {
			nCnt := 0
			if err := row.Scan(&nCnt); err != nil {
				return err
			}
			if nCnt != 0 {
				return errors.New("failed to update: workset name must be unique: " + newSetName)
			}
			return nil
		})
	if err != nil {
		return false, err
	}

	// rename workset and
	// "unlock" workset before commit: restore original value of is_readonly
	err = TrxUpdate(trx,
		"UPDATE workset_lst"+
			" SET set_name = "+toQuotedMax(newSetName, nameDbMax)+", "+
			" is_readonly = is_readonly - 1,"+
			" update_dt = "+ToQuoted(helper.MakeDateTime(time.Now()))+
			" WHERE set_id ="+sId)
	if err != nil {
		return false, err
	}

	return true, nil
}

// UpdateWorkset create new workset metadata, replace or merge existing workset metadata in database.
//
// Set name is used to find workset and set id updated with actual database value.
// Workset must be read-write for replace or merge.
//
// Only metadata updated: list of workset parameters not changed.
// To add or remove parameters form workset use UpdateWorksetParameterFrom() method.
// It is an error if incoming list parameters include any parameter which are not already in workset_parameter table.
//
// If workset does not exist then empty workset created, without parameters.
// Parameter list in workset metadata must be empty if workset does not exist.
//
// Replace is replace of existing metadata or create empty new workset.
// If workset exist then workset metadata replaced and parameters text metadata replaced.
//
// Merge does merge of text metadata with existing workset or create empty new workset.
// If workset exist then text is updated if such language already exist or inserted if no text in that language.
func (meta *WorksetMeta) UpdateWorkset(dbConn *sql.DB, modelDef *ModelMeta, isReplace bool, langDef *LangMeta) error {

	// validate parameters
	if modelDef == nil {
		return errors.New("invalid (empty) model metadata")
	}
	if langDef == nil {
		return errors.New("invalid (empty) language list")
	}
	if meta.Set.Name == "" {
		return errors.New("invalid (empty) workset name")
	}
	if meta.Set.ModelId != modelDef.Model.ModelId {
		return errors.New("workset: " + meta.Set.Name + " invalid model id " + strconv.Itoa(meta.Set.ModelId) + " expected: " + strconv.Itoa(modelDef.Model.ModelId))
	}

	// do update in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	err = doUpdateWorkset(trx, modelDef, meta, isReplace, langDef)
	if err != nil {
		trx.Rollback()
		return err
	}
	trx.Commit()

	return nil
}

// doUpdateWorkset insert new or update existing workset metadata in database.
// It does update as part of transaction
// Set name is used to find workset and set id updated with actual database value
// Workset must be read-write for replace or merge.
func doUpdateWorkset(trx *sql.Tx, modelDef *ModelMeta, meta *WorksetMeta, isReplace bool, langDef *LangMeta) error {

	smId := strconv.Itoa(modelDef.Model.ModelId)

	// workset name cannot be empty
	if meta.Set.Name == "" {
		return errors.New("invalid (empty) workset name, id: " + strconv.Itoa(meta.Set.SetId))
	}
	meta.Set.ModelId = modelDef.Model.ModelId // update model id

	// get new set id if workset not exist
	//
	// UPDATE id_lst SET id_value =
	//   CASE
	//     WHEN 0 = (SELECT COUNT(*) FROM workset_lst WHERE model_id = 1 AND set_name = 'set 22')
	//       THEN id_value + 1
	//     ELSE id_value
	//   END
	// WHERE id_key = 'run_id_set_id'
	err := TrxUpdate(trx,
		"UPDATE id_lst SET id_value ="+
			" CASE"+
			" WHEN 0 ="+
			" (SELECT COUNT(*) FROM workset_lst"+
			" WHERE model_id = "+smId+" AND set_name = "+ToQuoted(meta.Set.Name)+
			" )"+
			" THEN id_value + 1"+
			" ELSE id_value"+
			" END"+
			" WHERE id_key = 'run_id_set_id'")
	if err != nil {
		return err
	}

	// check if this workset already exist and get readonly status
	setId := 0
	isReadonly := false
	err = TrxSelectFirst(trx,
		"SELECT set_id, is_readonly FROM workset_lst WHERE model_id = "+smId+" AND set_name = "+ToQuoted(meta.Set.Name),
		func(row *sql.Row) error {
			nReadonly := 0
			if err := row.Scan(&setId, &nReadonly); err != nil {
				return err
			}
			isReadonly = nReadonly != 0 // oracle: smallint is float64
			return nil

		})
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if isReadonly {
		return errors.New("failed to update: workset already exists and it is read-only: " + strconv.Itoa(meta.Set.SetId) + ": " + meta.Set.Name)
	}

	// if workset not exist then create new empty workset
	if setId <= 0 {

		// get new set id
		err = TrxSelectFirst(trx,
			"SELECT id_value FROM id_lst WHERE id_key = 'run_id_set_id'",
			func(row *sql.Row) error {
				return row.Scan(&setId)
			})
		switch {
		case err == sql.ErrNoRows:
			return errors.New("invalid destination database, likely not an openM++ database")
		case err != nil:
			return err
		}

		meta.Set.SetId = setId // update set id with actual value

		// insert new workset with empty parameters list
		return doInsertWorkset(trx, modelDef, meta, langDef)
	}
	// else: update existing workset
	meta.Set.SetId = setId // workset exist, id may be different

	// get Hid's of current workset parameters list
	var hs []int
	err = TrxSelectRows(trx,
		"SELECT parameter_hid FROM workset_parameter WHERE set_id ="+strconv.Itoa(setId),
		func(rows *sql.Rows) error {
			var i int
			err := rows.Scan(&i)
			if err != nil {
				return err
			}
			hs = append(hs, i)
			return err
		})
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// check: all parameters in the new incoming parameters list must be already in database
pLoop:
	for j := range meta.Param {
		for _, h := range hs {
			if meta.Param[j].ParamHid == h {
				continue pLoop // id found in the new parameters list
			}
		}
		// parameter exist in the new incoming parameters list and not in database
		return errors.New("failed to update: workset parameter not exist in database, hId: " + strconv.Itoa(meta.Param[j].ParamHid) + " : " + meta.Set.Name)
	}

	// do replace of metadata or merge
	if isReplace {
		return doReplaceWorkset(trx, modelDef, meta, langDef)
	}
	// else
	return doMergeWorkset(trx, modelDef, meta, langDef)
}

// doInsertWorkset insert new workset metadata in database.
// It does update as part of transaction.
// Workset parameters list must be empty.
func doInsertWorkset(trx *sql.Tx, modelDef *ModelMeta, meta *WorksetMeta, langDef *LangMeta) error {

	// workset parameters list must be empty in order to create new workset
	if len(meta.Param) > 0 {
		return errors.New("error: cannot create new workset with non-empty list of parameters " + meta.Set.Name)
	}

	// if workset based on existing run then base run id must be positive
	sbId := ""
	if meta.Set.BaseRunId > 0 {
		sbId = strconv.Itoa(meta.Set.BaseRunId)
	} else {
		sbId = "NULL"
	}

	// set update time if not defined
	if meta.Set.UpdateDateTime == "" || !helper.IsTimeStamp(meta.Set.UpdateDateTime) {
		meta.Set.UpdateDateTime = helper.MakeDateTime(time.Now())
	}

	// INSERT INTO workset_lst (set_id, base_run_id, model_id, set_name, is_readonly, update_dt)
	// VALUES (22, NULL, 1, 'set 22', 0, '2012-08-17 16:05:59.123')
	err := TrxUpdate(trx,
		"INSERT INTO workset_lst (set_id, base_run_id, model_id, set_name, is_readonly, update_dt)"+
			" VALUES ("+
			strconv.Itoa(meta.Set.SetId)+", "+
			sbId+", "+
			strconv.Itoa(modelDef.Model.ModelId)+", "+
			toQuotedMax(meta.Set.Name, nameDbMax)+", "+
			toBoolSqlConst(meta.Set.IsReadonly)+", "+
			ToQuoted(meta.Set.UpdateDateTime)+")")
	if err != nil {
		return err
	}

	// insert new rows into workset_txt
	// parameters list must be empty: workset_parameter_txt not inserted
	if err = doInsertWorksetBody(trx, modelDef, meta, langDef); err != nil {
		return err
	}
	return err
}

// doReplaceWorkset replace workset metadata in database.
// It does update as part of transaction.
// It does delete existing parameter values which are not in the list of workset parameters.
func doReplaceWorkset(trx *sql.Tx, modelDef *ModelMeta, meta *WorksetMeta, langDef *LangMeta) error {

	// if workset based on existing run then base run id must be positive
	sbId := ""
	if meta.Set.BaseRunId > 0 {
		sbId = strconv.Itoa(meta.Set.BaseRunId)
	} else {
		sbId = "NULL"
	}

	sId := strconv.Itoa(meta.Set.SetId)

	// UPDATE workset_lst
	// SET is_readonly = 0, base_run_id = 1234, update_dt = '2012-08-17 16:05:59.123'
	// WHERE set_id = 22
	//
	if meta.Set.UpdateDateTime == "" {
		meta.Set.UpdateDateTime = helper.MakeDateTime(time.Now())
	}
	err := TrxUpdate(trx,
		"UPDATE workset_lst"+
			" SET is_readonly = "+toBoolSqlConst(meta.Set.IsReadonly)+", "+
			" base_run_id = "+sbId+", "+
			" update_dt = "+ToQuoted(meta.Set.UpdateDateTime)+
			" WHERE set_id ="+sId)
	if err != nil {
		return err
	}

	// delete existing workset_parameter_txt, workset_txt
	err = TrxUpdate(trx, "DELETE FROM workset_parameter_txt WHERE set_id = "+sId)
	if err != nil {
		return err
	}
	err = TrxUpdate(trx, "DELETE FROM workset_txt WHERE set_id = "+sId)
	if err != nil {
		return err
	}

	// insert new rows into workset body tables: workset_txt, workset_parameter_txt
	if err = doInsertWorksetBody(trx, modelDef, meta, langDef); err != nil {
		return err
	}
	return nil
}

// doInsertWorksetBody insert into workset metadata tables: workset_txt, workset_parameter_txt
// It does update as part of transaction.
func doInsertWorksetBody(trx *sql.Tx, modelDef *ModelMeta, meta *WorksetMeta, langDef *LangMeta) error {

	sId := strconv.Itoa(meta.Set.SetId)

	// insert workset text (description and notes)
	for j := range meta.Txt {

		meta.Txt[j].SetId = meta.Set.SetId // update set id

		// if language code valid then insert into workset_txt
		if lId, ok := langDef.IdByCode(meta.Txt[j].LangCode); ok {

			err := TrxUpdate(trx,
				"INSERT INTO workset_txt (set_id, lang_id, descr, note) VALUES ("+
					sId+", "+
					strconv.Itoa(lId)+", "+
					toQuotedMax(meta.Txt[j].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(meta.Txt[j].Note, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	// insert new workset parameter text: parameter value notes
	for k := range meta.Param {

		// insert new workset parameter text: parameter value notes
		for j := range meta.Param[k].Txt {

			meta.Param[k].Txt[j].SetId = meta.Set.SetId // update set id

			// if language code valid then insert into workset_parameter_txt
			if lId, ok := langDef.IdByCode(meta.Param[k].Txt[j].LangCode); ok {

				err := TrxUpdate(trx,
					"INSERT INTO workset_parameter_txt (set_id, parameter_hid, lang_id, note) VALUES ("+
						sId+", "+
						strconv.Itoa(meta.Param[k].ParamHid)+", "+
						strconv.Itoa(lId)+", "+
						toQuotedOrNullMax(meta.Param[k].Txt[j].Note, noteDbMax)+")")
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// doMergeWorkset merge workset metadata in database.
// It does update as part of transaction.
// Workset master row updated with non-empty values:
// read-only status if new read-only value is true.
// Only parameter text is merged, not sub-value count.
func doMergeWorkset(trx *sql.Tx, modelDef *ModelMeta, meta *WorksetMeta, langDef *LangMeta) error {

	// UPDATE workset_lst
	// SET is_readonly = 1, base_run_id = 1234, update_dt = '2012-08-17 16:05:59.123'
	// WHERE set_id = 22
	//
	sId := strconv.Itoa(meta.Set.SetId)
	sl := ""
	if meta.Set.IsReadonly {
		sl += " is_readonly = 1, "
	}
	if meta.Set.BaseRunId > 0 {
		sl += " base_run_id = " + strconv.Itoa(meta.Set.BaseRunId) + ", "
	} else {
		if meta.Set.isNullBaseRun {
			sl += " base_run_id = NULL, "
		}
	}
	meta.Set.UpdateDateTime = helper.MakeDateTime(time.Now())
	sl += " update_dt = " + ToQuoted(meta.Set.UpdateDateTime)

	err := TrxUpdate(trx, "UPDATE workset_lst SET "+sl+" WHERE set_id ="+sId)
	if err != nil {
		return err
	}

	// merge by delete and insert into workset text (description and notes)
	for j := range meta.Txt {

		meta.Txt[j].SetId = meta.Set.SetId // update set id

		if lId, ok := langDef.IdByCode(meta.Txt[j].LangCode); ok { // if language code valid

			slId := strconv.Itoa(lId)
			err = TrxUpdate(trx,
				"DELETE FROM workset_txt WHERE set_id = "+sId+" AND lang_id = "+slId)
			if err != nil {
				return err
			}
			err = TrxUpdate(trx,
				"INSERT INTO workset_txt (set_id, lang_id, descr, note) VALUES ("+
					sId+", "+
					slId+", "+
					toQuotedMax(meta.Txt[j].Descr, descrDbMax)+", "+
					toQuotedOrNullMax(meta.Txt[j].Note, noteDbMax)+")")
			if err != nil {
				return err
			}
		}
	}

	// merge by delete and insert into workset parameter text: parameter value notes
	for k := range meta.Param {
		for j := range meta.Param[k].Txt {

			meta.Param[k].Txt[j].SetId = meta.Set.SetId // update set id

			if lId, ok := langDef.IdByCode(meta.Param[k].Txt[j].LangCode); ok { // if language code valid

				spHid := strconv.Itoa(meta.Param[k].ParamHid)
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
						sId+", "+" parameter_hid, "+slId+", "+toQuotedOrNullMax(meta.Param[k].Txt[j].Note, noteDbMax)+
						" FROM workset_parameter"+
						" WHERE set_id = "+sId+
						" AND parameter_hid = "+spHid)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
