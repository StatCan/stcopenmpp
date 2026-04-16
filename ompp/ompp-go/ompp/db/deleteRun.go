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

// Delete model run: delete metadata, parameters run values and output tables run values from database.
func DeleteRun(dbConn *sql.DB, runId int) error {

	// validate parameters
	if runId <= 0 {
		return errors.New("invalid run id: " + strconv.Itoa(runId))
	}

	// delete inside of transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	if err := doDeleteRun(trx, runId); err != nil {
		trx.Rollback()
		return err
	}
	trx.Commit()
	return nil
}

// delete model run metadata and run values (parameter, output tables, microdata) run values from database.
// if run values used by any other run as a base run then base run id updated to the next minimal run id.
// It does update as part of transaction
func doDeleteRun(trx *sql.Tx, runId int) error {

	// update model run master record to prevent run use
	sId := strconv.Itoa(runId)
	delTs := helper.MakeTimeStamp(time.Now())
	err := TrxUpdate(trx,
		"UPDATE run_lst"+
			" SET status = 'd', run_name = "+ToQuoted("deleted: "+delTs)+
			" WHERE run_id = "+sId)
	if err != nil {
		return err
	}

	// update to NULL base run id for all worksets where base run id = target run id
	err = TrxUpdate(trx, "UPDATE workset_lst SET base_run_id = NULL WHERE base_run_id = "+sId)
	if err != nil {
		return err
	}

	// replace value digests for parameter, output tables and microdata to avoid using values as a base run values
	delDgst := ("x-" + sId + "-" + delTs)

	err = TrxUpdate(trx,
		"UPDATE run_parameter SET value_digest = "+toQuotedMax(delDgst, codeDbMax)+
			" WHERE run_id = "+sId+
			" AND EXISTS (SELECT * FROM run_lst RL WHERE RL.run_id = run_parameter.run_id AND RL.run_id = "+sId+")")
	if err != nil {
		return err
	}
	err = TrxUpdate(trx,
		"UPDATE run_table SET value_digest = "+toQuotedMax(delDgst, codeDbMax)+
			" WHERE run_id = "+sId+
			" AND EXISTS (SELECT * FROM run_lst RL WHERE RL.run_id = run_table.run_id AND RL.run_id = "+sId+")")
	if err != nil {
		return err
	}
	err = TrxUpdate(trx,
		"UPDATE run_entity SET value_digest = "+toQuotedMax(delDgst, codeDbMax)+
			" WHERE run_id = "+sId+
			" AND EXISTS (SELECT * FROM run_lst RL WHERE RL.run_id = run_entity.run_id AND RL.run_id = "+sId+")")
	if err != nil {
		return err
	}

	// build a list of run parameter values db-tables
	var pTbls []string
	pHidTbl := map[int]string{}

	err = TrxSelectRows(trx,
		"SELECT P.parameter_hid, P.db_run_table"+
			" FROM run_parameter RP"+
			" INNER JOIN parameter_dic P ON (P.parameter_hid = RP.parameter_hid)"+
			" WHERE RP.run_id = "+sId+
			" ORDER BY 1",
		func(rows *sql.Rows) error {
			var hId int
			tn := ""
			if err := rows.Scan(&hId, &tn); err != nil {
				return err
			}
			pTbls = append(pTbls, tn)
			pHidTbl[hId] = tn
			return nil
		})
	if err != nil {
		return err
	}

	// build a list of run expression and accumulator values db-tables
	var eTbls []string
	var aTbls []string
	eHidTbl := map[int]string{}
	aHidTbl := map[int]string{}

	err = TrxSelectRows(trx,
		"SELECT T.table_hid, T.db_expr_table, T.db_acc_table"+
			" FROM run_table RT"+
			" INNER JOIN table_dic T ON (T.table_hid = RT.table_hid)"+
			" WHERE RT.run_id = "+sId+
			" ORDER BY 1",
		func(rows *sql.Rows) error {
			var hId int
			etn := ""
			atn := ""
			if err := rows.Scan(&hId, &etn, &atn); err != nil {
				return err
			}
			eTbls = append(eTbls, etn)
			aTbls = append(aTbls, atn)
			eHidTbl[hId] = etn
			aHidTbl[hId] = atn
			return nil
		})
	if err != nil {
		return err
	}

	// build a list of run microdata values db-tables
	var mTbls []string
	mHidTbl := map[int]string{}

	err = TrxSelectRows(trx,
		"SELECT EG.entity_gen_hid, EG.db_entity_table"+
			" FROM entity_gen EG"+
			" INNER JOIN run_entity RE ON (RE.entity_gen_hid = EG.entity_gen_hid)"+
			" WHERE RE.run_id = "+sId+
			" ORDER BY 1",
		func(rows *sql.Rows) error {
			var hId int
			tn := ""
			if err := rows.Scan(&hId, &tn); err != nil {
				return err
			}
			mTbls = append(mTbls, tn)
			mHidTbl[hId] = tn
			return nil
		})
	if err != nil {
		return err
	}

	// delete model runs:
	// find new base run id for all model parameters where parameter run value shared between runs

	type hr struct {
		hId   int // parameter or output table Hid
		runId int // run id in run_parameter or run_table
	}
	type hrMap map[hr]bool // map of [hid, run_id] to keep list of shared parameters, output tables or microdata

	// list of run parameters where base run id is the run id to be deleted
	hrm := hrMap{}
	hLst := []int{}
	hMap := map[int]int{} // parameter or output table Hid to new base run id
	pId := -1

	err = TrxSelectRows(trx,
		"SELECT parameter_hid, run_id"+
			" FROM run_parameter"+
			" WHERE run_id <> "+sId+
			" AND base_run_id = "+sId+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r hr
			if err := rows.Scan(&r.hId, &r.runId); err != nil {
				return err
			}
			hrm[r] = true
			if pId <= 0 || pId != r.hId {
				hLst = append(hLst, r.hId)
				pId = r.hId
				hMap[pId] = 0
			}
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
			" WHEN"+
			" NOT ("+
			" SELECT MIN(NR.run_id)"+
			" FROM run_parameter NR"+
			" INNER JOIN run_lst RL ON (RL.run_id = NR.run_id)"+
			" WHERE NR.parameter_hid = run_parameter.parameter_hid"+
			" AND NR.run_id <> "+sId+
			" AND NR.value_digest = run_parameter.value_digest"+
			" AND RL.status IN ("+okLst+")"+
			" ) IS NULL"+
			" THEN"+
			" ("+
			" SELECT MIN(NR.run_id)"+
			" FROM run_parameter NR"+
			" INNER JOIN run_lst RL ON (RL.run_id = NR.run_id)"+
			" WHERE NR.parameter_hid = run_parameter.parameter_hid "+
			" AND NR.run_id <> "+sId+
			" AND NR.value_digest = run_parameter.value_digest"+
			" AND RL.status IN ("+okLst+")"+
			" )"+
			" WHEN"+
			" NOT ("+
			" SELECT MIN(NR.run_id)"+
			" FROM run_parameter NR"+
			" INNER JOIN run_lst RL ON (RL.run_id = NR.run_id)"+
			" WHERE NR.parameter_hid = run_parameter.parameter_hid"+
			" AND NR.run_id <> "+sId+
			" AND NR.value_digest = run_parameter.value_digest"+
			" ) IS NULL"+
			" THEN"+
			" ("+
			" SELECT MIN(NR.run_id)"+
			" FROM run_parameter NR"+
			" INNER JOIN run_lst RL ON (RL.run_id = NR.run_id)"+
			" WHERE NR.parameter_hid = run_parameter.parameter_hid"+
			" AND NR.run_id <> "+sId+
			" AND NR.value_digest = run_parameter.value_digest"+
			" )"+
			" ELSE NULL"+ // new base run not found
			" END"+
			" WHERE run_id <> "+sId+
			" AND base_run_id = "+sId)
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
			if _, ok := hrm[r]; !ok {
				return nil // skip: parameter_hid, run_id not found in the list shared parameters
			}
			if n <= 0 {
				return errors.New("Unable to delete model run parameter:" + " " + strconv.Itoa(r.hId) + " run_id: " + strconv.Itoa(r.runId) + " " + "error: no new base run id exist")
			}
			hMap[r.hId] = n // new base run id

			return nil
		})
	if err != nil {
		return err
	}

	// update parameter value run id with new base run id
	for _, hId := range hLst {

		n := hMap[hId]
		if n <= 0 {
			return errors.New("Unable to delete model run parameter:" + " " + strconv.Itoa(hId) + " " + "error: new base run id not found")
		}
		err = TrxUpdate(trx,
			"UPDATE "+pHidTbl[hId]+" SET run_id = "+strconv.Itoa(n)+" WHERE run_id = "+sId)
		if err != nil {
			return err
		}
	}

	// delete model runs:
	// find new base run id for all model output tables where table values shared between runs

	// list of run output tables where base run id is the run id to be deleted
	hrm = hrMap{}
	hLst = []int{}
	clear(hMap)
	pId = -1

	err = TrxSelectRows(trx,
		"SELECT table_hid, run_id"+
			" FROM run_table"+
			" WHERE run_id <> "+sId+
			" AND base_run_id = "+sId+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r hr
			if err := rows.Scan(&r.hId, &r.runId); err != nil {
				return err
			}
			hrm[r] = true
			if pId <= 0 || pId != r.hId {
				hLst = append(hLst, r.hId)
				pId = r.hId
				hMap[pId] = 0
			}
			return nil
		})
	if err != nil {
		return err
	}

	// set new base run id in run_table table
	err = TrxUpdate(trx,
		"UPDATE run_table SET base_run_id ="+
			" CASE"+
			" WHEN"+
			" NOT ("+
			" SELECT MIN(NR.run_id)"+
			" FROM run_table NR"+
			" INNER JOIN run_lst RL ON (RL.run_id = NR.run_id)"+
			" WHERE NR.table_hid = run_table.table_hid"+
			" AND NR.run_id <> "+sId+
			" AND NR.value_digest = run_table.value_digest"+
			" AND RL.status IN ("+okLst+")"+
			" ) IS NULL"+
			" THEN"+
			" ("+
			" SELECT MIN(NR.run_id)"+
			" FROM run_table NR"+
			" INNER JOIN run_lst RL ON (RL.run_id = NR.run_id)"+
			" WHERE NR.table_hid = run_table.table_hid "+
			" AND NR.run_id <> "+sId+
			" AND NR.value_digest = run_table.value_digest"+
			" AND RL.status IN ("+okLst+")"+
			" )"+
			" WHEN"+
			" NOT ("+
			" SELECT MIN(NR.run_id)"+
			" FROM run_table NR"+
			" INNER JOIN run_lst RL ON (RL.run_id = NR.run_id)"+
			" WHERE NR.table_hid = run_table.table_hid"+
			" AND NR.run_id <> "+sId+
			" AND NR.value_digest = run_table.value_digest"+
			" ) IS NULL"+
			" THEN"+
			" ("+
			" SELECT MIN(NR.run_id)"+
			" FROM run_table NR"+
			" INNER JOIN run_lst RL ON (RL.run_id = NR.run_id)"+
			" WHERE NR.table_hid = run_table.table_hid"+
			" AND NR.run_id <> "+sId+
			" AND NR.value_digest = run_table.value_digest"+
			" )"+
			" ELSE NULL"+ // new base run not found
			" END"+
			" WHERE run_id <> "+sId+
			" AND base_run_id = "+sId)
	if err != nil {
		return err
	}

	// collect new base run id for run output tables
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
			if _, ok := hrm[r]; !ok {
				return nil // skip: table_hid, run_id not found in the list shared output tables
			}
			if n <= 0 {
				return errors.New("Unable to delete model run output table:" + " " + strconv.Itoa(r.hId) + " " + " run_id: " + strconv.Itoa(r.runId) + " " + "error: no new base run id exist")
			}
			hMap[r.hId] = n // new base run id

			return nil
		})
	if err != nil {
		return err
	}

	// update expression values db tables and accumulator values db tables run id with new base run id
	for _, hId := range hLst {

		n := hMap[hId]
		if n <= 0 {
			return errors.New("Unable to delete model run output table:" + " " + strconv.Itoa(hId) + " " + "error: new base run id not found")
		}

		err = TrxUpdate(trx,
			"UPDATE "+eHidTbl[hId]+" SET run_id = "+strconv.Itoa(n)+" WHERE run_id = "+sId)
		if err != nil {
			return err
		}

		err = TrxUpdate(trx,
			"UPDATE "+aHidTbl[hId]+" SET run_id = "+strconv.Itoa(n)+" WHERE run_id = "+sId)
		if err != nil {
			return err
		}
	}

	// delete model runs:
	// find new base run id all entity generations where microdata run value shared between runs

	// list of entity generations where base run id is the run id to be deleted
	hrm = hrMap{}
	hLst = []int{}
	clear(hMap)
	pId = -1

	err = TrxSelectRows(trx,
		"SELECT entity_gen_hid, run_id"+
			" FROM run_entity"+
			" WHERE run_id <> "+sId+
			" AND base_run_id = "+sId+
			" ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var r hr
			if err := rows.Scan(&r.hId, &r.runId); err != nil {
				return err
			}
			hrm[r] = true
			if pId <= 0 || pId != r.hId {
				hLst = append(hLst, r.hId)
				pId = r.hId
				hMap[pId] = 0
			}
			return nil
		})
	if err != nil {
		return err
	}

	// set new base run id in run_entity table
	err = TrxUpdate(trx,
		"UPDATE run_entity SET base_run_id ="+
			" CASE"+
			" WHEN"+
			" NOT ("+
			" SELECT MIN(NR.run_id)"+
			" FROM run_entity NR"+
			" INNER JOIN run_lst RL ON (RL.run_id = NR.run_id)"+
			" WHERE NR.entity_gen_hid = run_entity.entity_gen_hid"+
			" AND NR.run_id <> "+sId+
			" AND NR.value_digest = run_entity.value_digest"+
			" AND RL.status = "+ToQuoted(DoneRunStatus)+
			" ) IS NULL"+
			" THEN"+
			" ("+
			" SELECT MIN(NR.run_id)"+
			" FROM run_entity NR"+
			" INNER JOIN run_lst RL ON (RL.run_id = NR.run_id)"+
			" WHERE NR.entity_gen_hid = run_entity.entity_gen_hid "+
			" AND NR.run_id <> "+sId+
			" AND NR.value_digest = run_entity.value_digest"+
			" AND RL.status = "+ToQuoted(DoneRunStatus)+
			" )"+
			" WHEN"+
			" NOT ("+
			" SELECT MIN(NR.run_id)"+
			" FROM run_entity NR"+
			" INNER JOIN run_lst RL ON (RL.run_id = NR.run_id)"+
			" WHERE NR.entity_gen_hid = run_entity.entity_gen_hid"+
			" AND NR.run_id <> "+sId+
			" AND NR.value_digest = run_entity.value_digest"+
			" ) IS NULL"+
			" THEN"+
			" ("+
			" SELECT MIN(NR.run_id)"+
			" FROM run_entity NR"+
			" INNER JOIN run_lst RL ON (RL.run_id = NR.run_id)"+
			" WHERE NR.entity_gen_hid = run_entity.entity_gen_hid"+
			" AND NR.run_id <> "+sId+
			" AND NR.value_digest = run_entity.value_digest"+
			" )"+
			" ELSE NULL"+ // new base run not found
			" END"+
			" WHERE run_id <> "+sId+
			" AND base_run_id = "+sId)
	if err != nil {
		return err
	}

	// collect new base run id for run microdata
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
			if _, ok := hrm[r]; !ok {
				return nil // skip: entity_gen_hid, run_id not found in the list shared microdata
			}
			if n <= 0 {
				return errors.New("Unable to delete model run microdata:" + " " + strconv.Itoa(r.hId) + " run_id: " + strconv.Itoa(r.runId) + " " + "error: no new base run id exist")
			}
			hMap[r.hId] = n // new base run id

			return nil
		})
	if err != nil {
		return err
	}

	// update microdata values db tables run id with new base run id
	for _, hId := range hLst {

		n := hMap[hId]
		if n <= 0 {
			return errors.New("Unable to delete model run microdata:" + " " + strconv.Itoa(hId) + " " + "error: new base run id not found")
		}
		err = TrxUpdate(trx,
			"UPDATE "+mHidTbl[hId]+" SET run_id = "+strconv.Itoa(n)+" WHERE run_id = "+sId)
		if err != nil {
			return err
		}
	}

	// delete model runs: delete run metadata
	err = TrxUpdate(trx, "DELETE FROM run_progress WHERE run_id = "+sId)
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM run_entity WHERE run_id = "+sId)
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM run_table WHERE run_id = "+sId)
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM run_parameter_import WHERE run_id = "+sId)
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM run_parameter_txt WHERE run_id = "+sId)
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM run_parameter WHERE run_id = "+sId)
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM run_option WHERE run_id = "+sId)
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM run_txt WHERE run_id = "+sId)
	if err != nil {
		return err
	}

	err = TrxUpdate(trx, "DELETE FROM run_lst WHERE run_id = "+sId)
	if err != nil {
		return err
	}

	// delete rows from run microdata value tables
	// or drop microdata value tables where no run_entity exists
	for k := range mTbls {

		// delete entity generation and attributes where no run_entity exists
		dTbl := "--" + string([]rune(mTbls[k])[2:])

		err = TrxUpdate(trx,
			"UPDATE entity_gen SET db_entity_table = "+ToQuoted(dTbl)+
				" WHERE db_entity_table = "+ToQuoted(mTbls[k])+
				" AND NOT EXISTS (SELECT * FROM run_entity RE WHERE RE.entity_gen_hid = entity_gen.entity_gen_hid)",
		)
		if err != nil {
			return err
		}

		var n int = 0
		err = TrxSelectFirst(trx,
			"SELECT COUNT(*) FROM entity_gen WHERE db_entity_table = "+ToQuoted(dTbl),
			func(row *sql.Row) error {
				if err := row.Scan(&n); err != nil {
					return err
				}
				return nil
			})
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		err = TrxUpdate(trx,
			"UPDATE entity_gen SET db_entity_table = "+ToQuoted(mTbls[k])+
				" WHERE db_entity_table = "+ToQuoted(dTbl)+
				" AND NOT EXISTS (SELECT * FROM run_entity RE WHERE RE.entity_gen_hid = entity_gen.entity_gen_hid)",
		)
		if err != nil {
			return err
		}

		// delete rows from microdata table
		// or drop microdata table if there are no run_entity rows exist for that table
		if n <= 0 {

			err = TrxUpdate(trx, "DELETE FROM "+mTbls[k]+" WHERE run_id = "+sId)
			if err != nil {
				return err
			}

		} else { // no run_entity rows exist for that db table: drop db table

			err = TrxUpdate(trx,
				"DELETE FROM entity_gen_attr"+
					" WHERE EXISTS"+
					" ("+
					" SELECT * FROM entity_gen EG"+
					" WHERE EG.entity_gen_hid = entity_gen_attr.entity_gen_hid"+
					" AND EG.db_entity_table = "+ToQuoted(mTbls[k])+
					" )"+
					" AND NOT EXISTS (SELECT * FROM run_entity RE WHERE RE.entity_gen_hid = entity_gen_attr.entity_gen_hid)",
			)
			if err != nil {
				return err
			}
			err = TrxUpdate(trx,
				"DELETE FROM entity_gen"+
					" WHERE db_entity_table = "+ToQuoted(mTbls[k])+
					" AND NOT EXISTS (SELECT * FROM run_entity RE WHERE RE.entity_gen_hid = entity_gen.entity_gen_hid)",
			)
			if err != nil {
				return err
			}

			err = TrxUpdate(trx, "DROP TABLE "+mTbls[k])
			if err != nil {
				return err
			}
		}
	}

	// delete rows from run parameter value tables
	for k := range pTbls {
		err = TrxUpdate(trx, "DELETE FROM "+pTbls[k]+" WHERE run_id = "+sId)
		if err != nil {
			return err
		}
	}

	// delete rows from run expression value and accumulator value tables
	for k := range eTbls {
		err = TrxUpdate(trx, "DELETE FROM "+eTbls[k]+" WHERE run_id = "+sId)
		if err != nil {
			return err
		}
	}
	for k := range aTbls {
		err = TrxUpdate(trx, "DELETE FROM "+aTbls[k]+" WHERE run_id = "+sId)
		if err != nil {
			return err
		}
	}

	return nil
}
