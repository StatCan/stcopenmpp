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

// DeleteWorkset delete workset metadata and workset parameter values from database.
func DeleteWorkset(dbConn *sql.DB, setId int) error {

	// validate parameters
	if setId <= 0 {
		return errors.New("invalid workset id: " + strconv.Itoa(setId))
	}

	// delete inside of transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	if err := dbDeleteWorkset(trx, setId); err != nil {
		trx.Rollback()
		return err
	}
	trx.Commit()
	return nil
}

// DeleteWorksetAllParameters delete all parameters metadata and values from workset.
//
// Workset must be read-write in order to delete all parameters.
// If workset does not exist then nothing deleted and no errors returned, it is empty operation.
func DeleteWorksetAllParameters(dbConn *sql.DB, setId int) error {

	// validate parameters
	if setId <= 0 {
		return errors.New("invalid workset id: " + strconv.Itoa(setId))
	}

	// delete inside of transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	if err := dbDeleteWorksetAllParameters(trx, setId); err != nil {
		trx.Rollback()
		return err
	}
	trx.Commit()
	return nil
}

// doDeleteWorkset delete workset metadata and workset parameter values from database.
// It does update as part of transaction
func dbDeleteWorkset(trx *sql.Tx, setId int) error {

	// update workset master record to prevent workset use
	sId := strconv.Itoa(setId)
	err := TrxUpdate(trx, "UPDATE workset_lst SET set_name = 'deleted' WHERE set_id = "+sId)
	if err != nil {
		return err
	}

	// delete all parameters data and metadats from workset
	err = dbDeleteWorksetAllParameters(trx, setId)
	if err != nil {
		return err
	}

	// delete workset from modeling tasks
	err = TrxUpdate(trx, "DELETE FROM task_set WHERE set_id = "+sId)
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM workset_txt WHERE set_id = "+sId)
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM workset_lst WHERE set_id = "+sId)
	if err != nil {
		return err
	}

	return nil
}

// dbDeleteWorksetAllParameters delete all parameters metadata and values from workset.
// It does update as part of transaction
// Workset must be read-write in order to delete all parameters.
func dbDeleteWorksetAllParameters(trx *sql.Tx, setId int) error {

	// "lock" workset to prevent update or use by the model
	sId := strconv.Itoa(setId)

	err := TrxUpdate(trx, "UPDATE workset_lst SET is_readonly = is_readonly + 1 WHERE set_id = "+sId)
	if err != nil {
		return err
	}

	// check if workset exist and not readonly
	nRd := 0
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
		return nil // workset not found: nothing to do
	case err != nil:
		return err
	case nRd != 1:
		return errors.New("failed to delete: workset is read-only: " + sId)
	}

	// build a list of workset parameters db-tables
	var tblArr []string
	err = TrxSelectRows(trx,
		"SELECT P.db_set_table"+
			" FROM workset_parameter W"+
			" INNER JOIN parameter_dic P ON (P.parameter_hid = W.parameter_hid)"+
			" WHERE W.set_id = "+sId,
		func(rows *sql.Rows) error {
			tn := ""
			if err := rows.Scan(&tn); err != nil {
				return err
			}
			tblArr = append(tblArr, tn)
			return nil
		})
	if err != nil {
		return err
	}

	// delete workset parameter values
	for k := range tblArr {
		err = TrxUpdate(trx, "DELETE FROM "+tblArr[k]+" WHERE set_id = "+sId)
		if err != nil {
			return err
		}
	}

	// delete workset parameters metadata
	err = TrxUpdate(trx, "DELETE FROM workset_parameter_txt WHERE set_id = "+sId)
	if err != nil {
		return err
	}
	err = TrxUpdate(trx, "DELETE FROM workset_parameter WHERE set_id = "+sId)
	if err != nil {
		return err
	}

	// "unlock" workset before commit: restore original value of is_readonly=0
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

// DeleteWorksetParameter do delete parameter metadata and values from workset.
//
// If parameter not exist in workset then nothing deleted.
// Workset must be read-write in order to delete parameter.
// It is return parameter Hid = 0 if nothing deleted.
func DeleteWorksetParameter(dbConn *sql.DB, modelId int, setName, paramName string) (int, error) {

	// validate parameters
	if modelId <= 0 {
		return 0, errors.New("invalid model id: " + strconv.Itoa(modelId))
	}
	if setName == "" {
		return 0, errors.New("invalid (empty) workset name")
	}
	if paramName == "" {
		return 0, errors.New("invalid (empty) parameter name")
	}

	// delete inside of transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return 0, err
	}
	paramHid, err := dbDeleteWorksetParameter(trx, modelId, setName, paramName)
	if err != nil {
		trx.Rollback()
		return 0, err
	}
	trx.Commit()
	return paramHid, nil
}

// dbDeleteWorksetParameter delete workset parameter metadata and values from database.
// It does update as part of transaction.
func dbDeleteWorksetParameter(trx *sql.Tx, modelId int, setName, paramName string) (int, error) {

	// "lock" workset to prevent update or use by the model
	err := TrxUpdate(trx,
		"UPDATE workset_lst"+
			" SET is_readonly = is_readonly + 1"+
			" WHERE model_id = "+strconv.Itoa(modelId)+" AND set_name = "+ToQuoted(setName))
	if err != nil {
		return 0, err
	}

	// check if workset exist and not readonly
	setId := 0
	nRd := 0
	err = TrxSelectFirst(trx,
		"SELECT set_id, is_readonly FROM workset_lst"+
			" WHERE model_id = "+strconv.Itoa(modelId)+" AND set_name = "+ToQuoted(setName),
		func(row *sql.Row) error {
			if err := row.Scan(&setId, &nRd); err != nil {
				return err
			}
			return nil
		})
	switch {
	case err == sql.ErrNoRows:
		return 0, nil // workset not found: nothing to do
	case err != nil:
		return 0, err
	case nRd != 1:
		return 0, errors.New("failed to update: workset is read-only: " + setName)
	}
	sId := strconv.Itoa(setId)

	// build a list of workset parameters db-tables
	var paramHid int
	var tblName string
	err = TrxSelectFirst(trx,
		"SELECT P.parameter_hid, P.db_set_table"+
			" FROM workset_parameter W"+
			" INNER JOIN parameter_dic P ON (P.parameter_hid = W.parameter_hid)"+
			" WHERE W.set_id = "+sId+
			" AND P.parameter_name = "+ToQuoted(paramName),
		func(row *sql.Row) error {
			if err := row.Scan(&paramHid, &tblName); err != nil {
				return err
			}
			return nil
		})
	switch {
	case err == sql.ErrNoRows: // parameter not exist in workset: restore original value of is_readonly=0 and exit
		err = TrxUpdate(trx,
			"UPDATE workset_lst SET is_readonly = 0 WHERE set_id ="+strconv.Itoa(setId))
		if err != nil {
			return 0, err
		}
		return 0, nil // parameter not found: nothing to do
	case err != nil:
		return 0, err
	}
	spHid := strconv.Itoa(paramHid)

	// delete workset parameter values
	err = TrxUpdate(trx, "DELETE FROM "+tblName+" WHERE set_id = "+sId)
	if err != nil {
		return 0, err
	}

	// delete model workset metadata
	err = TrxUpdate(trx, "DELETE FROM workset_parameter_txt WHERE set_id = "+sId+" AND parameter_hid = "+spHid)
	if err != nil {
		return 0, err
	}

	err = TrxUpdate(trx, "DELETE FROM workset_parameter WHERE set_id = "+sId+" AND parameter_hid = "+spHid)
	if err != nil {
		return 0, err
	}

	// "unlock" workset before commit: restore original value of is_readonly=0
	err = TrxUpdate(trx,
		"UPDATE workset_lst"+
			" SET is_readonly = 0,"+
			" update_dt = "+ToQuoted(helper.MakeDateTime(time.Now()))+
			" WHERE set_id = "+strconv.Itoa(setId))
	if err != nil {
		return 0, err
	}

	return paramHid, nil
}
