// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"errors"
	"strconv"
)

// Delete existing model metadata and drop model data tables from database.
func DeleteModel(dbConn *sql.DB, modelId int) error {

	// validate parameters
	if modelId <= 0 {
		return errors.New("invalid model id: " + strconv.Itoa(modelId))
	}

	// delete inside of transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	if err := doDeleteModel(trx, modelId); err != nil {
		trx.Rollback()
		return err
	}
	trx.Commit()
	return nil
}

// delete existing model metadata and drop model data tables from database.
// It does update as part of transaction
func doDeleteModel(trx *sql.Tx, modelId int) error {

	// update model master record to prevent model use
	smId := strconv.Itoa(modelId)
	err := TrxUpdate(trx, "UPDATE model_dic SET model_name = 'deleted' WHERE model_id = "+smId)
	if err != nil {
		return err
	}

	// delete modeling tasks and task run history
	err = TrxUpdate(trx,
		"DELETE FROM task_run_set WHERE EXISTS"+
			" (SELECT task_id FROM task_lst M WHERE M.task_id = task_run_set.task_id AND M.model_id = "+smId+")")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM task_run_lst WHERE EXISTS"+
			" (SELECT task_id FROM task_lst M WHERE M.task_id = task_run_lst.task_id AND M.model_id = "+smId+")")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM task_set WHERE EXISTS"+
			" (SELECT task_id FROM task_lst M WHERE M.task_id = task_set.task_id AND M.model_id = "+smId+")")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM task_txt WHERE EXISTS"+
			" (SELECT task_id FROM task_lst M WHERE M.task_id = task_txt.task_id AND M.model_id = "+smId+")")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM task_lst WHERE model_id = "+smId)
	if err != nil {
		return err
	}

	// delete model worksets metadata
	err = TrxUpdate(trx,
		"DELETE FROM workset_parameter_txt WHERE EXISTS"+
			" (SELECT set_id FROM workset_lst M WHERE M.set_id = workset_parameter_txt.set_id AND M.model_id = "+smId+")")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM workset_parameter WHERE EXISTS"+
			" (SELECT set_id FROM workset_lst M WHERE M.set_id = workset_parameter.set_id AND M.model_id = "+smId+")")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM workset_txt WHERE EXISTS"+
			" (SELECT set_id FROM workset_lst M WHERE M.set_id = workset_txt.set_id AND M.model_id = "+smId+")")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM workset_lst WHERE model_id = "+smId)
	if err != nil {
		return err
	}

	// delete model runs:
	// update to NULL base run id for all worksets where base run id's belong to model
	// it is expected to be empty result unless workset based on model run of other model (reserved for the future)
	err = TrxUpdate(trx,
		"UPDATE workset_lst SET base_run_id = NULL"+
			" WHERE EXISTS"+
			" ("+
			" SELECT RL.run_id FROM run_lst RL"+
			" WHERE RL.run_id = workset_lst.base_run_id"+
			" AND RL.model_id = "+smId+
			" )")
	if err != nil {
		return err
	}

	// list of model parameters
	pHids := []int{}
	pHidTbl := map[int]string{} // map parameter Hid to run parameter db table name

	err = TrxSelectRows(trx,
		"SELECT P.parameter_hid, P.db_run_table"+
			" FROM model_parameter_dic M"+
			" INNER JOIN parameter_dic P ON (P.parameter_hid = M.parameter_hid)"+
			" WHERE M.model_id = "+smId+
			" ORDER BY 1",
		func(rows *sql.Rows) error {
			var hId int
			rt := ""
			if err := rows.Scan(&hId, &rt); err != nil {
				return err
			}
			pHids = append(pHids, hId)
			pHidTbl[hId] = rt

			return nil
		})
	if err != nil {
		return err
	}

	// list of output tables
	tHids := []int{}
	tHidTbl := map[int]struct {
		eTbl string
		aTbl string
	}{}

	err = TrxSelectRows(trx,
		"SELECT T.table_hid, T.db_expr_table, T.db_acc_table"+
			" FROM model_table_dic M"+
			" INNER JOIN table_dic T ON (T.table_hid = M.table_hid)"+
			" WHERE M.model_id = "+smId+
			" ORDER BY 1",
		func(rows *sql.Rows) error {
			var hId int
			et := ""
			at := ""
			if err := rows.Scan(&hId, &et, &at); err != nil {
				return err
			}
			tHids = append(tHids, hId)
			tHidTbl[hId] = struct {
				eTbl string
				aTbl string
			}{eTbl: et, aTbl: at}

			return nil
		})
	if err != nil {
		return err
	}

	// build a list of run microdata values db-tables
	mHids := []int{}
	mHidTbl := map[int]string{}

	err = TrxSelectRows(trx,
		"SELECT EG.entity_gen_hid, EG.db_entity_table"+
			" FROM entity_gen EG"+
			" INNER JOIN model_entity_dic M ON (M.entity_hid = EG.entity_hid)"+
			" WHERE M.model_id = "+smId+
			" ORDER BY 1",
		func(rows *sql.Rows) error {
			var hId int
			tn := ""
			if err := rows.Scan(&hId, &tn); err != nil {
				return err
			}
			mHids = append(mHids, hId)
			mHidTbl[hId] = tn
			return nil
		})
	if err != nil {
		return err
	}

	// delete model runs:
	// find new base run id for all run parameters
	// where parameter shared between models and parameter run value shared between runs
	type hr struct {
		hId   int // parameter or output table Hid
		runId int // run id in run_parameter or run_table
	}

	hrLst := []hr{}        // list of [Hid, run id] where base run is shared
	hrBase := map[hr]int{} // map [Hid, run id] to new base run id

	err = TrxSelectRows(trx,
		"SELECT RP.parameter_hid, RP.run_id"+
			" FROM run_parameter RP"+
			" INNER JOIN run_lst RL ON (RL.run_id = RP.run_id)"+
			" WHERE RL.model_id <> "+smId+
			" AND EXISTS"+
			" ("+
			" SELECT E.run_id"+
			" FROM run_parameter E"+
			" INNER JOIN run_lst ERL ON (ERL.run_id = E.run_id)"+
			" WHERE ERL.model_id = "+smId+
			" AND E.parameter_hid = RP.parameter_hid"+
			" AND E.run_id = RP.base_run_id"+
			" )"+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r hr
			if err := rows.Scan(&r.hId, &r.runId); err != nil {
				return err
			}
			hrLst = append(hrLst, r)
			hrBase[r] = 0

			return nil
		})
	if err != nil {
		return err
	}

	// set new base run id in run_parameter table
	okLst := ToQuoted(DoneRunStatus) + ", " + ToQuoted(ProgressRunStatus) + ", " + ToQuoted(ExitRunStatus) // run status p=progress s=success x=exit

	err = TrxUpdate(trx,
		"UPDATE run_parameter SET base_run_id ="+
			" CASE"+
			" WHEN NOT"+
			" ("+
			" SELECT MIN(N.run_id)"+
			" FROM run_parameter N"+
			" INNER JOIN run_lst NRL ON (NRL.run_id = N.run_id)"+
			" WHERE N.parameter_hid = run_parameter.parameter_hid"+
			" AND N.value_digest = run_parameter.value_digest"+
			" AND NRL.model_id <> "+smId+
			" AND NRL.status IN ("+okLst+")"+
			" ) IS NULL"+
			" THEN"+
			" ("+
			" SELECT MIN(N.run_id)"+
			" FROM run_parameter N"+
			" INNER JOIN run_lst NRL ON (NRL.run_id = N.run_id)"+
			" WHERE N.parameter_hid = run_parameter.parameter_hid"+
			" AND N.value_digest = run_parameter.value_digest"+
			" AND NRL.model_id <> "+smId+
			" AND NRL.status IN ("+okLst+")"+
			" )"+
			" WHEN NOT"+
			" ("+
			" SELECT MIN(N.run_id)"+
			" FROM run_parameter N"+
			" INNER JOIN run_lst NRL ON (NRL.run_id = N.run_id)"+
			" WHERE N.parameter_hid = run_parameter.parameter_hid"+
			" AND N.value_digest = run_parameter.value_digest"+
			" AND NRL.model_id <> "+smId+
			" ) IS NULL"+
			" THEN"+
			" ("+
			" SELECT MIN(N.run_id)"+
			" FROM run_parameter N"+
			" INNER JOIN run_lst NRL ON (NRL.run_id = N.run_id)"+
			" WHERE N.parameter_hid = run_parameter.parameter_hid"+
			" AND N.value_digest = run_parameter.value_digest"+
			" AND NRL.model_id <> "+smId+
			" )"+
			" ELSE NULL"+
			" END"+
			" WHERE NOT EXISTS"+
			" ("+
			" SELECT RL.run_id FROM run_lst RL WHERE RL.run_id = run_parameter.run_id AND RL.model_id = "+smId+
			" )"+
			" AND EXISTS"+
			" ("+
			" SELECT ERP.run_id"+
			" FROM run_parameter ERP"+
			" INNER JOIN run_lst ERL ON (ERL.run_id = ERP.run_id)"+
			" WHERE ERL.model_id = "+smId+
			" AND ERP.parameter_hid = run_parameter.parameter_hid"+
			" AND ERP.run_id = run_parameter.base_run_id"+
			" )")
	if err != nil {
		return err
	}

	// collect new base run id for run parameters
	err = TrxSelectRows(trx,
		"SELECT parameter_hid, run_id, base_run_id"+
			" FROM run_parameter"+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r hr
			var nSql sql.NullInt64
			var n int
			if err := rows.Scan(&r.hId, &r.runId, &nSql); err != nil {
				return err
			}
			if nSql.Valid {
				n = int(nSql.Int64)
			} else {
				n = 0 // new base run undefined
			}
			if _, ok := hrBase[r]; !ok {
				return nil // skip: parameter_hid, run_id not found in the list shared parameters
			}
			if n <= 0 {
				return errors.New("Unable to delete model run parameter:" + " " + strconv.Itoa(r.hId) + " run_id: " + strconv.Itoa(r.runId) + " " + "error: no new base run id exist")
			}
			hrBase[r] = n // new base run id

			return nil
		})
	if err != nil {
		return err
	}

	// update run parameter db tables with new base run id
	for _, r := range hrLst {

		n := hrBase[r]
		if n <= 0 {
			return errors.New("Unable to delete model run parameter:" + " " + strconv.Itoa(r.hId) + " " + "error: new base run id not found, old run id:" + " " + strconv.Itoa(r.runId))
		}
		err = TrxUpdate(trx,
			"UPDATE "+pHidTbl[r.hId]+" SET run_id = "+strconv.Itoa(n)+" WHERE run_id = "+strconv.Itoa(r.runId))
		if err != nil {
			return err
		}
	}

	// delete model runs:
	// find new base run id for all output tables
	// where output table shared between models and output table value shared between runs

	// list of output tables where base run id is the run id to be deleted
	hrLst = []hr{}
	clear(hrBase)

	err = TrxSelectRows(trx,
		"SELECT RT.table_hid, RT.run_id"+
			" FROM run_table RT"+
			" INNER JOIN run_lst RL ON (RL.run_id = RT.run_id)"+
			" WHERE RL.model_id <> "+smId+
			" AND EXISTS"+
			" ("+
			" SELECT E.run_id"+
			" FROM run_table E"+
			" INNER JOIN run_lst ERL ON (ERL.run_id = E.run_id)"+
			" WHERE ERL.model_id = "+smId+
			" AND E.table_hid = RT.table_hid"+
			" AND E.run_id = RT.base_run_id"+
			" )"+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r hr
			if err := rows.Scan(&r.hId, &r.runId); err != nil {
				return err
			}
			hrLst = append(hrLst, r)
			hrBase[r] = 0

			return nil
		})
	if err != nil {
		return err
	}

	// set new base run id in run_table
	err = TrxUpdate(trx,
		"UPDATE run_table SET base_run_id ="+
			" CASE"+
			" WHEN NOT"+
			" ("+
			" SELECT MIN(N.run_id)"+
			" FROM run_table N"+
			" INNER JOIN run_lst NRL ON (NRL.run_id = N.run_id)"+
			" WHERE N.table_hid = run_table.table_hid"+
			" AND N.value_digest = run_table.value_digest"+
			" AND NRL.model_id <> "+smId+
			" AND NRL.status IN ("+okLst+")"+
			" ) IS NULL"+
			" THEN"+
			" ("+
			" SELECT MIN(N.run_id)"+
			" FROM run_table N"+
			" INNER JOIN run_lst NRL ON (NRL.run_id = N.run_id)"+
			" WHERE N.table_hid = run_table.table_hid"+
			" AND N.value_digest = run_table.value_digest"+
			" AND NRL.model_id <> "+smId+
			" AND NRL.status IN ("+okLst+")"+
			" )"+
			" WHEN NOT"+
			" ("+
			" SELECT MIN(N.run_id)"+
			" FROM run_table N"+
			" INNER JOIN run_lst NRL ON (NRL.run_id = N.run_id)"+
			" WHERE N.table_hid = run_table.table_hid"+
			" AND N.value_digest = run_table.value_digest"+
			" AND NRL.model_id <> "+smId+
			" ) IS NULL"+
			" THEN"+
			" ("+
			" SELECT MIN(N.run_id)"+
			" FROM run_table N"+
			" INNER JOIN run_lst NRL ON (NRL.run_id = N.run_id)"+
			" WHERE N.table_hid = run_table.table_hid"+
			" AND N.value_digest = run_table.value_digest"+
			" AND NRL.model_id <> "+smId+
			" )"+
			" ELSE NULL"+
			" END"+
			" WHERE NOT EXISTS"+
			" ("+
			" SELECT RL.run_id FROM run_lst RL WHERE RL.run_id = run_table.run_id AND RL.model_id = "+smId+
			" )"+
			" AND EXISTS"+
			" ("+
			" SELECT E.run_id"+
			" FROM run_table E"+
			" INNER JOIN run_lst ERL ON (ERL.run_id = E.run_id)"+
			" WHERE ERL.model_id = "+smId+
			" AND E.table_hid = run_table.table_hid"+
			" AND E.run_id = run_table.base_run_id"+
			" )")
	if err != nil {
		return err
	}

	// collect new base run id for output tables
	err = TrxSelectRows(trx,
		"SELECT table_hid, run_id, base_run_id"+
			" FROM run_table"+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r hr
			var nSql sql.NullInt64
			var n int
			if err := rows.Scan(&r.hId, &r.runId, &nSql); err != nil {
				return err
			}
			if nSql.Valid {
				n = int(nSql.Int64)
			} else {
				n = 0 // new base run undefined
			}
			if _, ok := hrBase[r]; !ok {
				return nil // skip: table_hid, run_id not found in the list shared output tables
			}
			if n <= 0 {
				return errors.New("Unable to delete model run output table:" + " " + strconv.Itoa(r.hId) + " run_id: " + strconv.Itoa(r.runId) + " " + "error: no new base run id exist")
			}
			hrBase[r] = n // new base run id

			return nil
		})
	if err != nil {
		return err
	}

	// update accumulators and expressions db tables with new base run id
	for _, r := range hrLst {

		n := hrBase[r]
		if n <= 0 {
			return errors.New("Unable to delete model run from output table:" + " " + strconv.Itoa(r.hId) + " " + "error: new base run id not found, old run id:" + " " + strconv.Itoa(r.runId))
		}

		err = TrxUpdate(trx,
			"UPDATE "+tHidTbl[r.hId].aTbl+" SET run_id = "+strconv.Itoa(n)+" WHERE run_id = "+strconv.Itoa(r.runId))
		if err != nil {
			return err
		}

		err = TrxUpdate(trx,
			"UPDATE "+tHidTbl[r.hId].eTbl+" SET run_id = "+strconv.Itoa(n)+" WHERE run_id = "+strconv.Itoa(r.runId))
		if err != nil {
			return err
		}
	}

	// delete model runs:
	// find new base run id for all microdata
	// where entity generation shared between models and microdata value shared between runs

	// list of entity generations where base run id is the run id to be deleted
	hrLst = []hr{}
	clear(hrBase)

	err = TrxSelectRows(trx,
		"SELECT EG.entity_gen_hid, EG.run_id"+
			" FROM run_entity EG"+
			" INNER JOIN run_lst RL ON (RL.run_id = EG.run_id)"+
			" WHERE RL.model_id <> "+smId+
			" AND EXISTS"+
			" ("+
			" SELECT E.run_id"+
			" FROM run_entity E"+
			" INNER JOIN run_lst ERL ON (ERL.run_id = E.run_id)"+
			" WHERE ERL.model_id = "+smId+
			" AND E.entity_gen_hid = EG.entity_gen_hid"+
			" AND E.run_id = EG.base_run_id"+
			" )"+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r hr
			if err := rows.Scan(&r.hId, &r.runId); err != nil {
				return err
			}
			hrLst = append(hrLst, r)
			hrBase[r] = 0

			return nil
		})
	if err != nil {
		return err
	}

	// set new base run id in run_entity
	err = TrxUpdate(trx,
		"UPDATE run_entity SET base_run_id ="+
			" CASE"+
			" WHEN NOT"+
			" ("+
			" SELECT MIN(N.run_id)"+
			" FROM run_entity N"+
			" INNER JOIN run_lst NRL ON (NRL.run_id = N.run_id)"+
			" WHERE N.entity_gen_hid = run_entity.entity_gen_hid"+
			" AND N.value_digest = run_entity.value_digest"+
			" AND NRL.model_id <> "+smId+
			" AND NRL.status = "+ToQuoted(DoneRunStatus)+
			" ) IS NULL"+
			" THEN"+
			" ("+
			" SELECT MIN(N.run_id)"+
			" FROM run_entity N"+
			" INNER JOIN run_lst NRL ON (NRL.run_id = N.run_id)"+
			" WHERE N.entity_gen_hid = run_entity.entity_gen_hid"+
			" AND N.value_digest = run_entity.value_digest"+
			" AND NRL.model_id <> "+smId+
			" AND NRL.status = "+ToQuoted(DoneRunStatus)+
			" )"+
			" WHEN NOT"+
			" ("+
			" SELECT MIN(N.run_id)"+
			" FROM run_entity N"+
			" INNER JOIN run_lst NRL ON (NRL.run_id = N.run_id)"+
			" WHERE N.entity_gen_hid = run_entity.entity_gen_hid"+
			" AND N.value_digest = run_entity.value_digest"+
			" AND NRL.model_id <> "+smId+
			" ) IS NULL"+
			" THEN"+
			" ("+
			" SELECT MIN(N.run_id)"+
			" FROM run_entity N"+
			" INNER JOIN run_lst NRL ON (NRL.run_id = N.run_id)"+
			" WHERE N.entity_gen_hid = run_entity.entity_gen_hid"+
			" AND N.value_digest = run_entity.value_digest"+
			" AND NRL.model_id <> "+smId+
			" )"+
			" ELSE NULL"+
			" END"+
			" WHERE NOT EXISTS"+
			" ("+
			" SELECT RL.run_id FROM run_lst RL WHERE RL.run_id = run_entity.run_id AND RL.model_id = "+smId+
			" )"+
			" AND EXISTS"+
			" ("+
			" SELECT E.run_id"+
			" FROM run_entity E"+
			" INNER JOIN run_lst ERL ON (ERL.run_id = E.run_id)"+
			" WHERE ERL.model_id = "+smId+
			" AND E.entity_gen_hid = run_entity.entity_gen_hid"+
			" AND E.run_id = run_entity.base_run_id"+
			" )")
	if err != nil {
		return err
	}

	// collect new base run id for microdata
	err = TrxSelectRows(trx,
		"SELECT entity_gen_hid, run_id, base_run_id"+
			" FROM run_entity"+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r hr
			var nSql sql.NullInt64
			var n int
			if err := rows.Scan(&r.hId, &r.runId, &nSql); err != nil {
				return err
			}
			if nSql.Valid {
				n = int(nSql.Int64)
			} else {
				n = 0 // new base run undefined
			}
			if _, ok := hrBase[r]; !ok {
				return nil // skip: entity_gen_hid, run_id not found in the list microdata
			}
			if n <= 0 {
				return errors.New("Unable to delete model run microdata:" + " " + strconv.Itoa(r.hId) + " run_id: " + strconv.Itoa(r.runId) + " " + "error: no new base run id exist")
			}
			hrBase[r] = n // new base run id

			return nil
		})
	if err != nil {
		return err
	}

	// update microdata db tables with new base run id
	for _, r := range hrLst {

		n := hrBase[r]
		if n <= 0 {
			return errors.New("Unable to delete model run microdata:" + " " + strconv.Itoa(r.hId) + " " + "error: new base run id not found, old run id:" + " " + strconv.Itoa(r.runId))
		}
		err = TrxUpdate(trx,
			"UPDATE "+mHidTbl[r.hId]+" SET run_id = "+strconv.Itoa(n)+" WHERE run_id = "+strconv.Itoa(r.runId))
		if err != nil {
			return err
		}
	}

	// delete model runs:
	// delete run metadata
	err = TrxUpdate(trx,
		"DELETE FROM run_progress WHERE EXISTS"+
			" (SELECT run_id FROM run_lst M WHERE M.run_id = run_progress.run_id AND M.model_id = "+smId+")")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM run_entity WHERE EXISTS"+
			" (SELECT run_id FROM run_lst M WHERE M.run_id = run_entity.run_id AND M.model_id = "+smId+")")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM run_table WHERE EXISTS"+
			" (SELECT run_id FROM run_lst M WHERE M.run_id = run_table.run_id AND M.model_id = "+smId+")")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM run_parameter_import WHERE EXISTS"+
			" (SELECT run_id FROM run_lst M WHERE M.run_id = run_parameter_import.run_id AND M.model_id = "+smId+")")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM run_parameter_txt WHERE EXISTS"+
			" (SELECT run_id FROM run_lst M WHERE M.run_id = run_parameter_txt.run_id AND M.model_id = "+smId+")")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM run_parameter WHERE EXISTS"+
			" (SELECT run_id FROM run_lst M WHERE M.run_id = run_parameter.run_id AND M.model_id = "+smId+")")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM run_option WHERE EXISTS"+
			" (SELECT run_id FROM run_lst M WHERE M.run_id = run_option.run_id AND M.model_id = "+smId+")")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM run_txt WHERE EXISTS"+
			" (SELECT run_id FROM run_lst M WHERE M.run_id = run_txt.run_id AND M.model_id = "+smId+")")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM run_lst WHERE model_id = "+smId)
	if err != nil {
		return err
	}

	// skip model default profile: profile_lst and profile_option
	// there is no explicit link between profile and model

	// delete model groups
	err = TrxUpdate(trx,
		"DELETE FROM group_pc WHERE EXISTS"+
			" (SELECT group_id FROM group_lst M WHERE M.group_id = group_pc.group_id AND M.model_id = "+smId+")")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM group_txt WHERE EXISTS"+
			" (SELECT group_id FROM group_lst M WHERE M.group_id = group_txt.group_id AND M.model_id = "+smId+")")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM group_lst WHERE model_id = "+smId)
	if err != nil {
		return err
	}

	// delete model entities:
	// build list of model entities where entity not shared between models
	type mdItem struct {
		eHId int    // entity Hid
		gHId int    // entity generation Hid
		tbl  string // entity microdata db table
	}
	var mdArr []mdItem

	err = TrxSelectRows(trx,
		"SELECT"+
			" EG.entity_hid, EG.entity_gen_hid, EG.db_entity_table"+
			" FROM entity_gen EG"+
			" WHERE EXISTS"+
			" ("+
			" SELECT entity_hid"+
			" FROM model_entity_dic M"+
			" WHERE M.entity_hid = EG.entity_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT entity_hid"+
			" FROM model_entity_dic NE"+
			" WHERE NE.entity_hid = EG.entity_hid AND NE.model_id <> "+smId+
			" )"+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r mdItem
			if err := rows.Scan(&r.eHId, &r.gHId, &r.tbl); err != nil {
				return err
			}
			mdArr = append(mdArr, r)
			return nil
		})
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM entity_gen_attr"+
			" WHERE EXISTS"+
			" ("+
			" SELECT M.entity_hid"+
			" FROM model_entity_dic M"+
			" WHERE M.entity_hid = entity_gen_attr.entity_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT NE.entity_hid"+
			" FROM model_entity_dic NE"+
			" WHERE NE.entity_hid = entity_gen_attr.entity_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM entity_gen"+
			" WHERE EXISTS"+
			" ("+
			" SELECT M.entity_hid"+
			" FROM model_entity_dic M"+
			" WHERE M.entity_hid = entity_gen.entity_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT NE.entity_hid"+
			" FROM model_entity_dic NE"+
			" WHERE NE.entity_hid = entity_gen.entity_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM entity_attr_txt"+
			" WHERE EXISTS"+
			" ("+
			" SELECT M.entity_hid"+
			" FROM model_entity_dic M"+
			" WHERE M.entity_hid = entity_attr_txt.entity_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT NE.entity_hid"+
			" FROM model_entity_dic NE"+
			" WHERE NE.entity_hid = entity_attr_txt.entity_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM entity_attr"+
			" WHERE EXISTS"+
			" ("+
			" SELECT M.entity_hid"+
			" FROM model_entity_dic M"+
			" WHERE M.entity_hid = entity_attr.entity_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT NE.entity_hid"+
			" FROM model_entity_dic NE"+
			" WHERE NE.entity_hid = entity_attr.entity_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM entity_dic_txt"+
			" WHERE EXISTS"+
			" ("+
			" SELECT M.entity_hid"+
			" FROM model_entity_dic M"+
			" WHERE M.entity_hid = entity_dic_txt.entity_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT NE.entity_hid"+
			" FROM model_entity_dic NE"+
			" WHERE NE.entity_hid = entity_dic_txt.entity_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	// delete model entities:
	// delete model entity master rows
	err = TrxUpdate(trx, "DELETE FROM model_entity_dic WHERE model_id = "+smId)
	if err != nil {
		return err
	}

	// delete model entities:
	// delete entity master rows where entity does not belong to any model
	err = TrxUpdate(trx,
		"DELETE FROM entity_dic"+
			" WHERE NOT EXISTS"+
			" (SELECT NE.entity_hid FROM model_entity_dic NE WHERE NE.entity_hid = entity_dic.entity_hid)")
	if err != nil {
		return err
	}

	// delete model output tables:
	// build list of model output tables where table not shared between models
	type outTbl struct {
		hId    int    // table Hid
		expr   string // expressions db table
		acc    string // accumulators db table
		accAll string // all accumulators db view
	}
	var tblArr []outTbl

	err = TrxSelectRows(trx,
		"SELECT"+
			" D.table_hid, D.db_expr_table, D.db_acc_table, D.db_acc_all_view"+
			" FROM table_dic D"+
			" WHERE EXISTS"+
			" ("+
			" SELECT table_hid"+
			" FROM model_table_dic M"+
			" WHERE M.table_hid = D.table_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT table_hid"+
			" FROM model_table_dic NE"+
			" WHERE NE.table_hid = D.table_hid AND NE.model_id <> "+smId+
			" )",
		func(rows *sql.Rows) error {
			var r outTbl
			if err := rows.Scan(&r.hId, &r.expr, &r.acc, &r.accAll); err != nil {
				return err
			}
			tblArr = append(tblArr, r)
			return nil
		})
	if err != nil {
		return err
	}

	// delete model output tables:
	// delete output tables metadata where table not shared between models
	err = TrxUpdate(trx,
		"DELETE FROM table_expr_txt"+
			" WHERE EXISTS"+
			" ("+
			" SELECT table_hid"+
			" FROM model_table_dic M"+
			" WHERE M.table_hid = table_expr_txt.table_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT table_hid"+
			" FROM model_table_dic NE"+
			" WHERE NE.table_hid = table_expr_txt.table_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM table_expr"+
			" WHERE EXISTS"+
			" ("+
			" SELECT table_hid"+
			" FROM model_table_dic M"+
			" WHERE M.table_hid = table_expr.table_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT table_hid"+
			" FROM model_table_dic NE"+
			" WHERE NE.table_hid = table_expr.table_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM table_acc_txt"+
			" WHERE EXISTS"+
			" ("+
			" SELECT table_hid"+
			" FROM model_table_dic M"+
			" WHERE M.table_hid = table_acc_txt.table_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT table_hid"+
			" FROM model_table_dic NE"+
			" WHERE NE.table_hid = table_acc_txt.table_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM table_acc"+
			" WHERE EXISTS"+
			" ("+
			" SELECT table_hid"+
			" FROM model_table_dic M"+
			" WHERE M.table_hid = table_acc.table_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT table_hid"+
			" FROM model_table_dic NE"+
			" WHERE NE.table_hid = table_acc.table_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM table_dims_txt"+
			" WHERE EXISTS"+
			" ("+
			" SELECT table_hid"+
			" FROM model_table_dic M"+
			" WHERE M.table_hid = table_dims_txt.table_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT table_hid"+
			" FROM model_table_dic NE"+
			" WHERE NE.table_hid = table_dims_txt.table_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM table_dims"+
			" WHERE EXISTS"+
			" ("+
			" SELECT table_hid"+
			" FROM model_table_dic M"+
			" WHERE M.table_hid = table_dims.table_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT table_hid"+
			" FROM model_table_dic NE"+
			" WHERE NE.table_hid = table_dims.table_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM table_dic_txt"+
			" WHERE EXISTS"+
			" ("+
			" SELECT table_hid"+
			" FROM model_table_dic M"+
			" WHERE M.table_hid = table_dic_txt.table_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT table_hid"+
			" FROM model_table_dic NE"+
			" WHERE NE.table_hid = table_dic_txt.table_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	// delete model output tables:
	// delete model output table master rows
	err = TrxUpdate(trx, "DELETE FROM model_table_dic WHERE model_id = "+smId)
	if err != nil {
		return err
	}

	// delete model output tables:
	// delete output table master rows where output table does not belong to any model
	err = TrxUpdate(trx,
		"DELETE FROM table_dic"+
			" WHERE NOT EXISTS"+
			" (SELECT table_hid FROM model_table_dic NE WHERE NE.table_hid = table_dic.table_hid)")
	if err != nil {
		return err
	}

	// delete model parameters:
	// build list of model parameters where parameter not shared between models
	type paramItem struct {
		hId int    // parameter Hid
		ws  string // workset parameter values db-table name
		run string // run parameter values db-table name
	}
	var paramArr []paramItem

	err = TrxSelectRows(trx,
		"SELECT"+
			" D.parameter_hid, D.db_set_table, D.db_run_table"+
			" FROM parameter_dic D"+
			" WHERE EXISTS"+
			" ("+
			" SELECT parameter_hid"+
			" FROM model_parameter_dic M"+
			" WHERE M.parameter_hid = D.parameter_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT parameter_hid"+
			" FROM model_parameter_dic NE"+
			" WHERE NE.parameter_hid = D.parameter_hid AND NE.model_id <> "+smId+
			" )",
		func(rows *sql.Rows) error {
			var r paramItem
			if err := rows.Scan(&r.hId, &r.ws, &r.run); err != nil {
				return err
			}
			paramArr = append(paramArr, r)
			return nil
		})
	if err != nil {
		return err
	}

	// delete model parameters:
	// delete parameters metadata where parameter not shared between models
	err = TrxUpdate(trx,
		"DELETE FROM parameter_dims_txt"+
			" WHERE EXISTS"+
			" ("+
			" SELECT parameter_hid"+
			" FROM model_parameter_dic M"+
			" WHERE M.parameter_hid = parameter_dims_txt.parameter_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT parameter_hid"+
			" FROM model_parameter_dic NE"+
			" WHERE NE.parameter_hid = parameter_dims_txt.parameter_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM parameter_dims"+
			" WHERE EXISTS"+
			" ("+
			" SELECT parameter_hid"+
			" FROM model_parameter_dic M"+
			" WHERE M.parameter_hid = parameter_dims.parameter_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT parameter_hid"+
			" FROM model_parameter_dic NE"+
			" WHERE NE.parameter_hid = parameter_dims.parameter_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM parameter_dic_txt"+
			" WHERE EXISTS"+
			" ("+
			" SELECT parameter_hid"+
			" FROM model_parameter_dic M"+
			" WHERE M.parameter_hid = parameter_dic_txt.parameter_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT parameter_hid"+
			" FROM model_parameter_dic NE"+
			" WHERE NE.parameter_hid = parameter_dic_txt.parameter_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM model_parameter_import WHERE model_id = "+smId)
	if err != nil {
		return err
	}

	// delete model parameters:
	// delete model parameter master rows
	err = TrxUpdate(trx, "DELETE FROM model_parameter_dic WHERE model_id = "+smId)
	if err != nil {
		return err
	}

	// delete model parameters:
	// delete parameter master row where parameter does not belong to any model
	err = TrxUpdate(trx,
		"DELETE FROM parameter_dic"+
			" WHERE NOT EXISTS"+
			" (SELECT parameter_hid FROM model_parameter_dic NE WHERE NE.parameter_hid = parameter_dic.parameter_hid)")
	if err != nil {
		return err
	}

	// delete model types:
	// delete types metadata where type is not built-in and not shared between models
	err = TrxUpdate(trx,
		"DELETE FROM type_enum_txt"+
			" WHERE type_hid > "+strconv.Itoa(maxBuiltInTypeId)+
			" AND EXISTS"+
			" ("+
			" SELECT type_hid"+
			" FROM model_type_dic M"+
			" WHERE M.type_hid = type_enum_txt.type_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT type_hid"+
			" FROM model_type_dic NE"+
			" WHERE NE.type_hid = type_enum_txt.type_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM type_enum_lst"+
			" WHERE type_hid > "+strconv.Itoa(maxBuiltInTypeId)+
			" AND EXISTS"+
			" ("+
			" SELECT type_hid"+
			" FROM model_type_dic M"+
			" WHERE M.type_hid = type_enum_lst.type_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT type_hid"+
			" FROM model_type_dic NE"+
			" WHERE NE.type_hid = type_enum_lst.type_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	err = TrxUpdate(trx,
		"DELETE FROM type_dic_txt"+
			" WHERE type_hid > "+strconv.Itoa(maxBuiltInTypeId)+
			" AND EXISTS"+
			" ("+
			" SELECT type_hid"+
			" FROM model_type_dic M"+
			" WHERE M.type_hid = type_dic_txt.type_hid AND M.model_id = "+smId+
			" )"+
			" AND NOT EXISTS"+
			" ("+
			" SELECT type_hid"+
			" FROM model_type_dic NE"+
			" WHERE NE.type_hid = type_dic_txt.type_hid AND NE.model_id <> "+smId+
			" )")
	if err != nil {
		return err
	}

	// delete model types:
	// delete model type master rows
	err = TrxUpdate(trx, "DELETE FROM model_type_dic WHERE model_id = "+smId)
	if err != nil {
		return err
	}

	// delete model types:
	// delete type master rows where type is not built-in and not belong to any model
	err = TrxUpdate(trx,
		"DELETE FROM type_dic"+
			" WHERE type_hid > "+strconv.Itoa(maxBuiltInTypeId)+
			" AND NOT EXISTS (SELECT type_hid FROM model_type_dic MT WHERE MT.type_hid = type_dic.type_hid)")
	if err != nil {
		return err
	}

	// delete model language-specific message rows
	err = TrxUpdate(trx, "DELETE FROM model_word WHERE model_id = "+smId)
	if err != nil {
		return err
	}

	// delete model text rows: model description and notes
	err = TrxUpdate(trx, "DELETE FROM model_dic_txt WHERE model_id = "+smId)
	if err != nil {
		return err
	}

	// delete model master row
	err = TrxUpdate(trx, "DELETE FROM model_dic WHERE model_id = "+smId)
	if err != nil {
		return err
	}

	// drop db-tables for entities microdata values
	// where entity not shared between models
	for k := range mdArr {

		err = TrxUpdate(trx, "DROP TABLE "+mdArr[k].tbl)
		if err != nil {
			return err
		}
	}

	// drop db tables and views for accumulators and expressions
	// where output table not shared between models
	for k := range tblArr {

		err = TrxUpdate(trx, "DROP VIEW "+tblArr[k].accAll)
		if err != nil {
			return err
		}

		err = TrxUpdate(trx, "DROP TABLE "+tblArr[k].expr)
		if err != nil {
			return err
		}

		err = TrxUpdate(trx, "DROP TABLE "+tblArr[k].acc)
		if err != nil {
			return err
		}
	}

	// drop db-tables for parameter workset values and run values
	// where parameter not shared between models
	for k := range paramArr {

		err = TrxUpdate(trx, "DROP TABLE "+paramArr[k].ws)
		if err != nil {
			return err
		}

		err = TrxUpdate(trx, "DROP TABLE "+paramArr[k].run)
		if err != nil {
			return err
		}
	}

	return nil
}
