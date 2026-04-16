// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"errors"
	"strconv"
)

// GetRun return model run row by id: run_lst table row.
func GetRun(dbConn *sql.DB, runId int) (*RunRow, error) {
	return getRunRow(dbConn,
		"SELECT"+
			" H.run_id, H.model_id, H.run_name, H.sub_count,"+
			" H.sub_started, H.sub_completed, H.create_dt, H.status,"+
			" H.update_dt, H.run_digest, H.value_digest, H.run_stamp"+
			" FROM run_lst H"+
			" WHERE H.run_id = "+strconv.Itoa(runId))
}

// GetFirstRun return first run of the model: run_lst table row.
func GetFirstRun(dbConn *sql.DB, modelId int) (*RunRow, error) {
	return getRunRow(dbConn,
		"SELECT"+
			" H.run_id, H.model_id, H.run_name, H.sub_count,"+
			" H.sub_started, H.sub_completed, H.create_dt, H.status,"+
			" H.update_dt, H.run_digest, H.value_digest, H.run_stamp"+
			" FROM run_lst H"+
			" WHERE H.run_id ="+
			" (SELECT MIN(M.run_id) FROM run_lst M WHERE M.model_id = "+strconv.Itoa(modelId)+")")
}

// GetLastRun return last run of the model: run_lst table row.
func GetLastRun(dbConn *sql.DB, modelId int) (*RunRow, error) {
	return getRunRow(dbConn,
		"SELECT"+
			" H.run_id, H.model_id, H.run_name, H.sub_count,"+
			" H.sub_started, H.sub_completed, H.create_dt, H.status,"+
			" H.update_dt, H.run_digest, H.value_digest, H.run_stamp"+
			" FROM run_lst H"+
			" WHERE H.run_id ="+
			" (SELECT MAX(M.run_id) FROM run_lst M WHERE M.model_id = "+strconv.Itoa(modelId)+")")
}

// GetLastCompletedRun return last completed run of the model: run_lst table row.
//
// Run completed if run status one of: s=success, x=exit, e=error
func GetLastCompletedRun(dbConn *sql.DB, modelId int) (*RunRow, error) {
	return getRunRow(dbConn,
		"SELECT"+
			" H.run_id, H.model_id, H.run_name, H.sub_count,"+
			" H.sub_started, H.sub_completed, H.create_dt, H.status,"+
			" H.update_dt, H.run_digest, H.value_digest, H.run_stamp"+
			" FROM run_lst H"+
			" WHERE H.run_id ="+
			" ("+
			" SELECT MAX(M.run_id) FROM run_lst M"+
			" WHERE M.model_id = "+strconv.Itoa(modelId)+
			" AND M.status IN ("+ToQuoted(DoneRunStatus)+", "+ToQuoted(ErrorRunStatus)+", "+ToQuoted(ExitRunStatus)+")"+
			" )")
}

// GetRunByDigest return model run row by digest: run_lst table row.
func GetRunByDigest(dbConn *sql.DB, digest string) (*RunRow, error) {
	return getRunRow(dbConn,
		"SELECT"+
			" H.run_id, H.model_id, H.run_name, H.sub_count,"+
			" H.sub_started, H.sub_completed, H.create_dt, H.status,"+
			" H.update_dt, H.run_digest, H.value_digest, H.run_stamp"+
			" FROM run_lst H"+
			" WHERE H.run_digest = "+ToQuoted(digest))
}

// GetRunByStamp return model run row by run stamp: run_lst table row.
//
// If there is multiple runs with this stamp then run with min(run_id) returned
func GetRunByStamp(dbConn *sql.DB, modelId int, stamp string) (*RunRow, error) {
	return getRunRow(dbConn,
		"SELECT"+
			" H.run_id, H.model_id, H.run_name, H.sub_count,"+
			" H.sub_started, H.sub_completed, H.create_dt, H.status,"+
			" H.update_dt, H.run_digest, H.value_digest, H.run_stamp"+
			" FROM run_lst H"+
			" WHERE H.run_id = "+
			" ("+
			" SELECT MIN(M.run_id) FROM run_lst M"+
			" WHERE M.model_id = "+strconv.Itoa(modelId)+
			" AND M.run_stamp = "+ToQuoted(stamp)+
			")")
}

// GetRunByName return model run row by run name: run_lst table row.
//
// If there is multiple runs with this name then run with min(run_id) returned
func GetRunByName(dbConn *sql.DB, modelId int, name string) (*RunRow, error) {
	return getRunRow(dbConn,
		"SELECT"+
			" H.run_id, H.model_id, H.run_name, H.sub_count,"+
			" H.sub_started, H.sub_completed, H.create_dt, H.status,"+
			" H.update_dt, H.run_digest, H.value_digest, H.run_stamp"+
			" FROM run_lst H"+
			" WHERE H.run_id = "+
			" ("+
			" SELECT MIN(M.run_id) FROM run_lst M"+
			" WHERE M.model_id = "+strconv.Itoa(modelId)+
			" AND M.run_name = "+ToQuoted(name)+
			")")
}

// GetLastRunByName return model run row by run name: run_lst table row.
//
// If there is multiple runs with this name then run with max(run_id) returned
func GetLastRunByName(dbConn *sql.DB, modelId int, name string) (*RunRow, error) {
	return getRunRow(dbConn,
		"SELECT"+
			" H.run_id, H.model_id, H.run_name, H.sub_count,"+
			" H.sub_started, H.sub_completed, H.create_dt, H.status,"+
			" H.update_dt, H.run_digest, H.value_digest, H.run_stamp"+
			" FROM run_lst H"+
			" WHERE H.run_id = "+
			" ("+
			" SELECT MAX(M.run_id) FROM run_lst M"+
			" WHERE M.model_id = "+strconv.Itoa(modelId)+
			" AND M.run_name = "+ToQuoted(name)+
			")")
}

// GetRunByDigestStampName return model run row by run digest or run stamp or run name: run_lst table row.
//
// It does select run row by digest, if not found then by model id and stamp, if not found by model id and run name.
// If there is multiple runs with this stamp or name then run with min(run_id) returned
func GetRunByDigestStampName(dbConn *sql.DB, modelId int, rdsn string) (*RunRow, error) {

	r, err := GetRunByDigest(dbConn, rdsn)
	if err == nil && r == nil {
		r, err = GetRunByStamp(dbConn, modelId, rdsn)
	}
	if err == nil && r == nil {
		r, err = GetRunByName(dbConn, modelId, rdsn)
	}
	return r, err
}

// GetRunListByDigestStampName return list of model run rows by run digest or run stamp or run name: run_lst table rows.
//
// It does select run row by digest, if not found then by model id and stamp, if not found by model id and run name.
// If there is multiple runs with this stamp or name then multiple rows returned
func GetRunListByDigestStampName(dbConn *sql.DB, modelId int, rdsn string) ([]RunRow, error) {

	sql := "SELECT" +
		" H.run_id, H.model_id, H.run_name, H.sub_count," +
		" H.sub_started, H.sub_completed, H.create_dt, H.status," +
		" H.update_dt, H.run_digest, H.value_digest, H.run_stamp" +
		" FROM run_lst H"

	rLst, err := getRunLst(dbConn,
		sql+" WHERE H.model_id = "+strconv.Itoa(modelId)+
			" AND H.run_digest = "+ToQuoted(rdsn)+
			" ORDER BY 1")

	if err == nil && len(rLst) <= 0 {
		rLst, err = getRunLst(dbConn,
			sql+" WHERE H.model_id = "+strconv.Itoa(modelId)+
				" AND H.run_stamp = "+ToQuoted(rdsn)+
				" ORDER BY 1")
	}
	if err == nil && len(rLst) <= 0 {
		rLst, err = getRunLst(dbConn,
			sql+" WHERE H.model_id = "+strconv.Itoa(modelId)+
				" AND H.run_name = "+ToQuoted(rdsn)+
				" ORDER BY 1")
	}
	return rLst, err
}

// getRunRow return run_lst table row.
func getRunRow(dbConn *sql.DB, query string) (*RunRow, error) {

	var r RunRow

	err := SelectFirst(dbConn, query,
		func(row *sql.Row) error {
			var svd sql.NullString
			if err := row.Scan(
				&r.RunId, &r.ModelId, &r.Name, &r.SubCount,
				&r.SubStarted, &r.SubCompleted, &r.CreateDateTime, &r.Status,
				&r.UpdateDateTime, &r.RunDigest, &svd, &r.RunStamp); err != nil {
				return err
			}
			if svd.Valid {
				r.ValueDigest = svd.String
			}
			return nil
		})
	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, err
	}

	return &r, nil
}

// getRunLst return run_lst table rows.
func getRunLst(dbConn *sql.DB, query string) ([]RunRow, error) {

	var runRs []RunRow

	err := SelectRows(dbConn, query,
		func(rows *sql.Rows) error {
			var r RunRow
			var svd sql.NullString
			if err := rows.Scan(
				&r.RunId, &r.ModelId, &r.Name, &r.SubCount,
				&r.SubStarted, &r.SubCompleted, &r.CreateDateTime, &r.Status,
				&r.UpdateDateTime, &r.RunDigest, &svd, &r.RunStamp); err != nil {
				return err
			}
			if svd.Valid {
				r.ValueDigest = svd.String
			}
			runRs = append(runRs, r)
			return nil
		})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return runRs, nil
}

// GetRunList return list of model runs by model_id: run_lst rows.
func GetRunList(dbConn *sql.DB, modelId int) ([]RunRow, error) {

	// model not found: model id must be positive
	if modelId <= 0 {
		return nil, nil
	}

	// get list of runs for that model id
	q := "SELECT" +
		" H.run_id, H.model_id, H.run_name, H.sub_count," +
		" H.sub_started, H.sub_completed, H.create_dt, H.status," +
		" H.update_dt, H.run_digest, H.value_digest, H.run_stamp" +
		" FROM run_lst H" +
		" WHERE H.model_id = " + strconv.Itoa(modelId) +
		" ORDER BY 1"

	runRs, err := getRunLst(dbConn, q)
	if err != nil {
		return nil, err
	}
	if len(runRs) <= 0 { // no model runs
		return nil, nil
	}

	return runRs, nil
}

// GetRunListText return list of model runs with description and notes: run_lst and run_txt rows.
//
// If langCode not empty then only specified language selected else all languages
func GetRunListText(dbConn *sql.DB, modelId int, langCode string) ([]RunRow, []RunTxtRow, error) {

	// model not found: model id must be positive
	if modelId <= 0 {
		return nil, nil, nil
	}

	// get list of runs for that model id
	q := "SELECT" +
		" H.run_id, H.model_id, H.run_name, H.sub_count," +
		" H.sub_started, H.sub_completed, H.create_dt, H.status," +
		" H.update_dt, H.run_digest, H.value_digest, H.run_stamp" +
		" FROM run_lst H" +
		" WHERE H.model_id = " + strconv.Itoa(modelId) +
		" ORDER BY 1"

	runRs, err := getRunLst(dbConn, q)
	if err != nil {
		return nil, nil, err
	}
	if len(runRs) <= 0 { // no model runs
		return nil, nil, nil
	}

	// get run description and notes by model id and language
	q = "SELECT M.run_id, M.lang_id, L.lang_code, M.descr, M.note" +
		" FROM run_txt M" +
		" INNER JOIN run_lst H ON (H.run_id = M.run_id)" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE H.model_id = " + strconv.Itoa(modelId)
	if langCode != "" {
		q += " AND L.lang_code = " + ToQuoted(langCode)
	}
	q += " ORDER BY 1, 2"

	txtRs, err := getRunText(dbConn, q)
	if err != nil {
		return nil, nil, err
	}
	return runRs, txtRs, nil
}

// GetRunText return model run description and notes: run_txt table rows.
//
// If langCode not empty then only specified language selected else all languages
func GetRunText(dbConn *sql.DB, runId int, langCode string) ([]RunTxtRow, error) {

	q := "SELECT M.run_id, M.lang_id, L.lang_code, M.descr, M.note" +
		" FROM run_txt M" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE M.run_id = " + strconv.Itoa(runId)
	if langCode != "" {
		q += " AND L.lang_code = " + ToQuoted(langCode)
	}
	q += " ORDER BY 1, 2"

	return getRunText(dbConn, q)
}

// getRunText return model run description and notes: run_txt table rows.
func getRunText(dbConn *sql.DB, query string) ([]RunTxtRow, error) {

	// select db rows from run_txt
	var txtLst []RunTxtRow

	err := SelectRows(dbConn, query,
		func(rows *sql.Rows) error {
			var r RunTxtRow
			var lId int
			var note sql.NullString
			if err := rows.Scan(
				&r.RunId, &lId, &r.LangCode, &r.Descr, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			txtLst = append(txtLst, r)
			return nil
		})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return txtLst, nil
}

// GetRunParamText return run parameter value notes: run_parameter_txt table rows.
//
// If langCode not empty then only specified language selected else all languages
func GetRunParamText(dbConn *sql.DB, runId int, paramHid int, langCode string) ([]RunParamTxtRow, error) {

	q := "SELECT M.run_id, M.parameter_hid, M.lang_id, L.lang_code, M.note" +
		" FROM run_parameter_txt M" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE M.run_id = " + strconv.Itoa(runId) +
		" AND M.parameter_hid = " + strconv.Itoa(paramHid)
	if langCode != "" {
		q += " AND L.lang_code = " + ToQuoted(langCode)
	}
	q += " ORDER BY 1, 2, 3"

	// do select and set parameter id in output results
	return getRunParamText(dbConn, q)
}

// GetRunAllParamText return all run parameters value notes: run_parameter_txt table rows.
//
// If langCode not empty then only specified language selected else all languages
func GetRunAllParamText(dbConn *sql.DB, runId int, langCode string) ([]RunParamTxtRow, error) {

	// make select using Hid
	q := "SELECT M.run_id, M.parameter_hid, M.lang_id, L.lang_code, M.note" +
		" FROM run_parameter_txt M" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE M.run_id = " + strconv.Itoa(runId)
	if langCode != "" {
		q += " AND L.lang_code = " + ToQuoted(langCode)
	}
	q += " ORDER BY 1, 2, 3"

	// do select and set parameter id in output results
	return getRunParamText(dbConn, q)
}

// getRunParamText return run parameter value notes: run_parameter_txt table rows.
func getRunParamText(dbConn *sql.DB, query string) ([]RunParamTxtRow, error) {

	var txtLst []RunParamTxtRow

	err := SelectRows(dbConn, query,
		func(rows *sql.Rows) error {
			var r RunParamTxtRow
			var lId int
			var note sql.NullString
			if err := rows.Scan(
				&r.RunId, &r.ParamHid, &lId, &r.LangCode, &note); err != nil {
				return err
			}
			if note.Valid {
				r.Note = note.String
			}
			txtLst = append(txtLst, r)
			return nil
		})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return txtLst, nil
}

// GetRunProgress return sub-values run progress for specified run id: run_progress table rows.
func GetRunProgress(dbConn *sql.DB, runId int) ([]RunProgress, error) {

	rs, err := getRunProgress(
		dbConn,
		"SELECT"+
			" RP.run_id, RP.sub_id, RP.create_dt, RP.status, RP.update_dt, RP.progress_count, RP.progress_value"+
			" FROM run_progress RP"+
			" WHERE RP.run_id = "+strconv.Itoa(runId)+
			" ORDER BY 1, 2")
	if err != nil {
		return nil, err
	}

	rpLst := make([]RunProgress, len(rs))
	for k := range rpLst {
		rpLst[k] = rs[k].Progress
	}
	return rpLst, err
}

// getRunProgress return sub-values run progress: run_progress table rows.
func getRunProgress(dbConn *sql.DB, query string) ([]runProgressRow, error) {

	rpLst := []runProgressRow{}

	err := SelectRows(dbConn, query,
		func(rows *sql.Rows) error {
			var r runProgressRow
			if err := rows.Scan(
				&r.RunId, &r.Progress.SubId, &r.Progress.CreateDateTime, &r.Progress.Status,
				&r.Progress.UpdateDateTime, &r.Progress.Count, &r.Progress.Value); err != nil {
				return err
			}
			rpLst = append(rpLst, r)
			return nil
		})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return rpLst, nil
}

// GetEntityGenList return entity generation rows by run id: entity_gen and entity_gen_attr table rows.
func GetEntityGenList(dbConn *sql.DB, runId int) ([]EntityGenMeta, error) {

	genLst := []EntityGenMeta{}

	// append entity_gen rows: generation Hid, db table name and generation digest
	// where this entity generation has microdata for that run id
	q := "SELECT H.run_id, EG.entity_gen_hid, H.model_id, ME.model_entity_id, EG.entity_hid, EG.db_entity_table, EG.gen_digest" +
		" FROM run_lst H" +
		" INNER JOIN run_entity RE ON (RE.run_id = H.run_id)" +
		" INNER JOIN entity_gen EG ON (EG.entity_gen_hid = RE.entity_gen_hid)" +
		" INNER JOIN model_entity_dic ME ON (ME.model_id = H.model_id AND ME.entity_hid = EG.entity_hid)" +
		" WHERE H.run_id = " + strconv.Itoa(runId) +
		" ORDER BY 1, 2"

	err := SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var r entityGenRow
			var nId int
			if err := rows.Scan(&nId, &r.GenHid, &r.ModelId, &r.EntityId, &r.EntityHid, &r.DbEntityTable, &r.GenDigest); err != nil {
				return err
			}
			genLst = append(genLst, EntityGenMeta{entityGenRow: r, GenAttr: []entityGenAttrRow{}})
			return nil
		})
	switch {
	case err == sql.ErrNoRows: // no entity microdata
		return nil, nil
	case err != nil:
		return nil, err
	}

	// append entity_gen_attr rows: generation Hid, and attribute id
	// where this entity generation has microdata for that run id
	q = "SELECT H.run_id, EA.entity_gen_hid, EA.attr_id" +
		" FROM run_lst H" +
		" INNER JOIN run_entity RE ON (RE.run_id = H.run_id)" +
		" INNER JOIN entity_gen_attr EA ON (EA.entity_gen_hid = RE.entity_gen_hid)" +
		" WHERE H.run_id = " + strconv.Itoa(runId) +
		" ORDER BY 1, 2, 3"

	err = SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var r entityGenAttrRow
			var nId int
			if err := rows.Scan(&nId, &r.GenHid, &r.AttrId); err != nil {
				return err
			}

			// find existing entity_gen row
			for k := range genLst {
				if genLst[k].GenHid == r.GenHid {
					genLst[k].GenAttr = append(genLst[k].GenAttr, r)
					return nil
				}
			}
			return errors.New("Error at entity generation attribute select, not found entity generation Hid: " + strconv.Itoa(r.GenHid))
		})
	if err != nil {
		return nil, err
	}

	return genLst, nil
}

// getRunEntity return run entity rows: run_entity table rows.
func getRunEntity(dbConn *sql.DB, runId int) ([]RunEntityRow, error) {

	reLst := []RunEntityRow{}

	// append run_entity rows: generation Hid and value digest
	q := "SELECT run_id, entity_gen_hid, row_count, value_digest" +
		" FROM run_entity" +
		" WHERE run_id = " + strconv.Itoa(runId) +
		" ORDER BY 1, 2"

	err := SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var r RunEntityRow
			var svd sql.NullString
			if err := rows.Scan(&r.RunId, &r.GenHid, &r.RowCount, &svd); err != nil {
				return err
			}
			if svd.Valid {
				r.ValueDigest = svd.String
			}
			reLst = append(reLst, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	return reLst, nil
}

// GetRunEntityGenList return run entity rows by model id: run_entity table rows.
func GetRunEntityGenByModel(dbConn *sql.DB, modelId int) ([]RunEntityRow, error) {

	reLst := []RunEntityRow{}

	// append run_entity rows: generation Hid and value digest
	q := "SELECT RE.run_id, RE.entity_gen_hid, RE.row_count, RE.value_digest" +
		" FROM run_lst H" +
		" INNER JOIN run_entity RE ON (RE.run_id = H.run_id)" +
		" WHERE H.model_id = " + strconv.Itoa(modelId) +
		" ORDER BY 1, 2"

	err := SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var r RunEntityRow
			var svd sql.NullString
			if err := rows.Scan(&r.RunId, &r.GenHid, &r.RowCount, &svd); err != nil {
				return err
			}
			if svd.Valid {
				r.ValueDigest = svd.String
			}
			reLst = append(reLst, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	return reLst, nil
}

// GetRunFull return full metadata for model run: run_lst, run_option, run_parameter, run_table, run_progress, and entity generation rows.
func GetRunFull(dbConn *sql.DB, runRow *RunRow) (*RunMeta, error) {

	// validate parameters
	if runRow == nil {
		return nil, errors.New("invalid (empty) model run row, it may be model run not found")
	}
	sRunId := strconv.Itoa(runRow.RunId)

	// run meta header: run_lst row, model name and digest
	meta := &RunMeta{Run: *runRow, Txt: []RunTxtRow{}, EntityGen: []EntityGenMeta{}, RunEntity: []RunEntityRow{}}

	// get run options by run id
	q := "SELECT" +
		" M.run_id, M.option_key, M.option_value" +
		" FROM run_option M" +
		" INNER JOIN run_lst H ON (H.run_id = M.run_id)" +
		" WHERE H.run_id = " + sRunId +
		" ORDER BY 1, 2"

	optRs, err := getRunOpts(dbConn, q)
	if err != nil {
		return nil, err
	}
	meta.Opts = optRs[runRow.RunId]

	// append run_parameter rows: Hid and sub-value count
	q = "SELECT M.run_id, M.parameter_hid, M.sub_count, M.value_digest" +
		" FROM run_parameter M" +
		" INNER JOIN run_lst H ON (H.run_id = M.run_id)" +
		" WHERE H.run_id = " + sRunId +
		" ORDER BY 1, 2"

	err = SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var r runParam
			var nId int
			var svd sql.NullString
			if err := rows.Scan(&nId, &r.ParamHid, &r.SubCount, &svd); err != nil {
				return err
			}
			if svd.Valid {
				r.ValueDigest = svd.String
			}
			r.Txt = []RunParamTxtRow{}
			meta.Param = append(meta.Param, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// append run_table rows: table Hid
	q = "SELECT M.run_id, M.table_hid, M.value_digest" +
		" FROM run_table M" +
		" INNER JOIN run_lst H ON (H.run_id = M.run_id)" +
		" WHERE H.run_id = " + sRunId +
		" ORDER BY 1, 2"

	err = SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var r runTable
			var nId int
			var svd sql.NullString
			if err := rows.Scan(&nId, &r.TableHid, &svd); err != nil {
				return err
			}
			if svd.Valid {
				r.ValueDigest = svd.String
			}
			meta.Table = append(meta.Table, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// get run sub-values progress for that run id
	rpRs, err := GetRunProgress(dbConn, runRow.RunId)
	if err != nil {
		return nil, err
	}
	meta.Progress = rpRs

	// get entity generation and run entity for that run id
	egLst, err := GetEntityGenList(dbConn, runRow.RunId)
	if err != nil {
		return nil, err
	}
	meta.EntityGen = egLst

	reLst, err := getRunEntity(dbConn, runRow.RunId)
	if err != nil {
		return nil, err
	}
	meta.RunEntity = reLst

	return meta, nil
}

// GetRunFullText return full metadata, including text, for completed model run:
// run_lst, run_txt, run_option, run_parameter, run_parameter_txt, run_table, run_progress, and entity generation rows.
//
// If isSuccess true then return only successfully completed runs
// else return all runs: success, error, exit, progress.
// If langCode not empty then only specified language selected else all languages
func GetRunFullText(dbConn *sql.DB, runRow *RunRow, isSuccess bool, langCode string) (*RunMeta, error) {

	// validate parameters
	if runRow == nil {
		return nil, errors.New("invalid (empty) model run row, it may be model run not found")
	}
	sRunId := strconv.Itoa(runRow.RunId)

	// where filters
	runWhere := " WHERE H.run_id = " + sRunId
	if isSuccess {
		runWhere += " AND H.status = " + ToQuoted(DoneRunStatus)
	} else {
		runWhere += " AND H.status IN (" +
			ToQuoted(DoneRunStatus) + ", " + ToQuoted(ErrorRunStatus) + ", " + ToQuoted(ExitRunStatus) + ", " + ToQuoted(ProgressRunStatus) + ")"
	}

	var langFilter string
	if langCode != "" {
		langFilter = " AND L.lang_code = " + ToQuoted(langCode)
	}

	// run meta header: run_lst row, model name and digest
	meta := &RunMeta{Run: *runRow, Txt: []RunTxtRow{}, EntityGen: []EntityGenMeta{}, RunEntity: []RunEntityRow{}}

	// get run description and notes by run id and language
	q := "SELECT M.run_id, M.lang_id, L.lang_code, M.descr, M.note" +
		" FROM run_txt M" +
		" INNER JOIN run_lst H ON (H.run_id = M.run_id)" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		runWhere +
		langFilter +
		" ORDER BY 1, 2"

	runTxtRs, err := getRunText(dbConn, q)
	if err != nil {
		return nil, err
	}
	meta.Txt = runTxtRs

	// get run options by run id
	q = "SELECT" +
		" M.run_id, M.option_key, M.option_value" +
		" FROM run_option M" +
		" INNER JOIN run_lst H ON (H.run_id = M.run_id)" +
		runWhere +
		" ORDER BY 1, 2"

	optRs, err := getRunOpts(dbConn, q)
	if err != nil {
		return nil, err
	}
	meta.Opts = optRs[runRow.RunId]

	// append run_parameter rows: Hid and sub-value count
	q = "SELECT M.run_id, M.parameter_hid, M.sub_count, M.value_digest" +
		" FROM run_parameter M" +
		" INNER JOIN run_lst H ON (H.run_id = M.run_id)" +
		runWhere +
		" ORDER BY 1, 2"

	hi := make(map[int]int) // map (parameter Hid) => index in parameter array

	err = SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var r runParam
			var nId int
			var svd sql.NullString
			if err := rows.Scan(&nId, &r.ParamHid, &r.SubCount, &svd); err != nil {
				return err
			}
			if svd.Valid {
				r.ValueDigest = svd.String
			}
			i := len(meta.Param)
			meta.Param = append(meta.Param, r)
			hi[r.ParamHid] = i
			return nil
		})
	if err != nil {
		return nil, err
	}

	// run_parameter_txt: select using Hid
	q = "SELECT M.run_id, M.parameter_hid, M.lang_id, L.lang_code, M.note" +
		" FROM run_parameter_txt M" +
		" INNER JOIN run_lst H ON (H.run_id = M.run_id)" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		runWhere +
		langFilter +
		" ORDER BY 1, 2, 3"

	paramTxtRs, err := getRunParamText(dbConn, q)
	if err != nil {
		return nil, err
	}

	// append parameter value notes to corresponding Hid of parameter
	for k := range paramTxtRs {
		i, ok := hi[paramTxtRs[k].ParamHid]
		if !ok {
			return nil, errors.New("model run: " + sRunId + " " + runRow.Name + ", parameter " + strconv.Itoa(paramTxtRs[k].ParamHid) + " not found")
		}
		meta.Param[i].Txt = append(meta.Param[i].Txt, paramTxtRs[k])
	}

	// append run_table rows: table Hid
	q = "SELECT M.run_id, M.table_hid, M.value_digest" +
		" FROM run_table M" +
		" INNER JOIN run_lst H ON (H.run_id = M.run_id)" +
		runWhere +
		" ORDER BY 1, 2"

	err = SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var r runTable
			var nId int
			var svd sql.NullString
			if err := rows.Scan(&nId, &r.TableHid, &svd); err != nil {
				return err
			}
			if svd.Valid {
				r.ValueDigest = svd.String
			}
			meta.Table = append(meta.Table, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// get run sub-values progress for that run id
	rpRs, err := GetRunProgress(dbConn, runRow.RunId)
	if err != nil {
		return nil, err
	}
	meta.Progress = rpRs

	// get entity generation and run entity for that run id
	egLst, err := GetEntityGenList(dbConn, runRow.RunId)
	if err != nil {
		return nil, err
	}
	meta.EntityGen = egLst

	reLst, err := getRunEntity(dbConn, runRow.RunId)
	if err != nil {
		return nil, err
	}
	meta.RunEntity = reLst

	return meta, nil
}

// GetRunFullTextList return list of full metadata, including text, for model runs:
// run_lst, run_txt, run_option, run_parameter, run_parameter_txt, run_table, run_progress and entity generation rows.
//
// If isSuccess true then return only successfully completed runs
// else return all runs: success, error, exit, progress.
// If langCode not empty then only specified language selected else all languages
func GetRunFullTextList(dbConn *sql.DB, modelId int, isSuccess bool, langCode string) ([]RunMeta, error) {

	// where filters
	var statusFilter string
	if isSuccess {
		statusFilter = " AND H.status = " + ToQuoted(DoneRunStatus)
	} else {
		statusFilter = " AND H.status IN (" +
			ToQuoted(DoneRunStatus) + ", " + ToQuoted(ErrorRunStatus) + ", " + ToQuoted(ExitRunStatus) + ", " + ToQuoted(ProgressRunStatus) + ")"
	}

	var langFilter string
	if langCode != "" {
		langFilter = " AND L.lang_code = " + ToQuoted(langCode)
	}

	// get list of runs for that model id
	smId := strconv.Itoa(modelId)

	q := "SELECT" +
		" H.run_id, H.model_id, H.run_name, H.sub_count," +
		" H.sub_started, H.sub_completed, H.create_dt, H.status," +
		" H.update_dt, H.run_digest, H.value_digest, H.run_stamp" +
		" FROM run_lst H" +
		" WHERE H.model_id = " + smId +
		statusFilter +
		" ORDER BY 1"

	runRs, err := getRunLst(dbConn, q)
	if err != nil {
		return nil, err
	}

	// get run description and notes by model id and language
	q = "SELECT M.run_id, M.lang_id, L.lang_code, M.descr, M.note" +
		" FROM run_txt M" +
		" INNER JOIN run_lst H ON (H.run_id = M.run_id)" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE H.model_id = " + smId +
		statusFilter +
		langFilter +
		" ORDER BY 1, 2"

	runTxtRs, err := getRunText(dbConn, q)
	if err != nil {
		return nil, err
	}

	// get run options by model id
	q = "SELECT" +
		" M.run_id, M.option_key, M.option_value" +
		" FROM run_option M" +
		" INNER JOIN run_lst H ON (H.run_id = M.run_id)" +
		" WHERE H.model_id = " + smId +
		statusFilter +
		" ORDER BY 1, 2"

	optRs, err := getRunOpts(dbConn, q)
	if err != nil {
		return nil, err
	}

	// run_parameter_txt: select using Hid
	q = "SELECT M.run_id, M.parameter_hid, M.lang_id, L.lang_code, M.note" +
		" FROM run_parameter_txt M" +
		" INNER JOIN run_lst H ON (H.run_id = M.run_id)" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE H.model_id = " + smId +
		statusFilter +
		langFilter +
		" ORDER BY 1, 2, 3"

	paramTxtRs, err := getRunParamText(dbConn, q)
	if err != nil {
		return nil, err
	}

	// get sub-values run progress by model id
	q = "SELECT" +
		" RP.run_id, RP.sub_id, RP.create_dt, RP.status, RP.update_dt, RP.progress_count, RP.progress_value" +
		" FROM run_lst H" +
		" INNER JOIN run_progress RP ON (RP.run_id = H.run_id)" +
		" WHERE H.model_id = " + smId +
		statusFilter +
		" ORDER BY 1, 2"

	rpRs, err := getRunProgress(dbConn, q)
	if err != nil {
		return nil, err
	}

	// convert to output result: join run pieces in struct by run id
	rl := make([]RunMeta, len(runRs))
	m := make(map[int]int) // map[run id] => index of run_lst row

	for k := range runRs {
		runId := runRs[k].RunId
		rl[k].Run = runRs[k]
		rl[k].Opts = optRs[runId]
		rl[k].EntityGen = []EntityGenMeta{}
		rl[k].RunEntity = []RunEntityRow{}
		rl[k].Progress = []RunProgress{}
		m[runId] = k
	}
	for k := range runTxtRs {
		if i, ok := m[runTxtRs[k].RunId]; ok {
			rl[i].Txt = append(rl[i].Txt, runTxtRs[k])
		}
	}
	for k := range rpRs {
		if i, ok := m[rpRs[k].RunId]; ok {
			rl[i].Progress = append(rl[i].Progress, rpRs[k].Progress)
		}
	}

	// append run_parameter rows: parameter Hid and sub-value count
	q = "SELECT M.run_id, M.parameter_hid, M.sub_count, M.value_digest" +
		" FROM run_parameter M" +
		" INNER JOIN run_lst H ON (H.run_id = M.run_id)" +
		" WHERE H.model_id = " + smId +
		statusFilter +
		" ORDER BY 1, 2"

	hi := make(map[int]map[int]int) // map[run id] => map[parameter Hid] => index in parameter array

	err = SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var r runParam
			var nId int
			var svd sql.NullString
			if err := rows.Scan(&nId, &r.ParamHid, &r.SubCount, &svd); err != nil {
				return err
			}
			if svd.Valid {
				r.ValueDigest = svd.String
			}

			idx, ok := m[nId] // find run id index
			if !ok {
				return nil // skip run if not in previous run list
			}

			i := len(rl[idx].Param)
			rl[idx].Param = append(rl[idx].Param, r) // append run_parameter row

			if _, ok = hi[nId]; !ok {
				hi[nId] = make(map[int]int)
			}
			hi[nId][r.ParamHid] = i // update map[run id] => map[hId] => parameter index

			return nil
		})
	if err != nil {
		return nil, err
	}

	// for each run_parameter_txt row
	for k := range paramTxtRs {

		i, ok := m[paramTxtRs[k].RunId]
		if !ok {
			continue // run id not found: run list updated between selects
		}
		mh, ok := hi[paramTxtRs[k].RunId]
		if !ok {
			continue // run id not found: run list updated between selects
		}
		// append parameter value notes to that parameter Hid
		if j, ok := mh[paramTxtRs[k].ParamHid]; ok {
			rl[i].Param[j].Txt = append(rl[i].Param[j].Txt, paramTxtRs[k])
		}
	}

	// append run_table rows: table Hid
	q = "SELECT M.run_id, M.table_hid, M.value_digest" +
		" FROM run_table M" +
		" INNER JOIN run_lst H ON (H.run_id = M.run_id)" +
		" WHERE H.model_id = " + smId +
		statusFilter +
		" ORDER BY 1, 2"

	err = SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var r runTable
			var nId int
			var svd sql.NullString
			if err := rows.Scan(&nId, &r.TableHid, &svd); err != nil {
				return err
			}
			if svd.Valid {
				r.ValueDigest = svd.String
			}

			idx, ok := m[nId] // find run id index
			if !ok {
				return nil // skip run if not in previous run list
			}
			rl[idx].Table = append(rl[idx].Table, r) // append run_table row

			return nil
		})
	if err != nil {
		return nil, err
	}

	// append entity_gen rows: generation Hid, db table name and generation digest
	q = "SELECT H.run_id, EG.entity_gen_hid, H.model_id, ME.model_entity_id, EG.entity_hid, EG.db_entity_table, EG.gen_digest" +
		" FROM run_lst H" +
		" INNER JOIN run_entity RE ON (RE.run_id = H.run_id)" +
		" INNER JOIN entity_gen EG ON (EG.entity_gen_hid = RE.entity_gen_hid)" +
		" INNER JOIN model_entity_dic ME ON (ME.model_id = H.model_id AND ME.entity_hid = EG.entity_hid)" +
		" WHERE H.model_id = " + smId +
		statusFilter +
		" ORDER BY 1, 2"

	ei := make(map[int]map[int]int) // map[run id] => map[generation Hid] => index in generation array

	err = SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var r entityGenRow
			var nId int
			if err := rows.Scan(&nId, &r.GenHid, &r.ModelId, &r.EntityId, &r.EntityHid, &r.DbEntityTable, &r.GenDigest); err != nil {
				return err
			}

			idx, ok := m[nId] // find run id index
			if !ok {
				return nil // skip run if not in previous run list
			}

			i := len(rl[idx].EntityGen)
			rl[idx].EntityGen = append(rl[idx].EntityGen, EntityGenMeta{entityGenRow: r, GenAttr: []entityGenAttrRow{}}) // append entity_gen row

			if _, ok = ei[nId]; !ok {
				ei[nId] = make(map[int]int)
			}
			ei[nId][r.GenHid] = i // update map[run id] => map[generation Hid] => index in generation array

			return nil
		})
	if err != nil {
		return nil, err
	}

	// append entity_gen_attr rows: generation Hid, and attribute id
	q = "SELECT H.run_id, EA.entity_gen_hid, EA.attr_id" +
		" FROM run_lst H" +
		" INNER JOIN run_entity RE ON (RE.run_id = H.run_id)" +
		" INNER JOIN entity_gen_attr EA ON (EA.entity_gen_hid = RE.entity_gen_hid)" +
		" WHERE H.model_id = " + smId +
		statusFilter +
		" ORDER BY 1, 2, 3"

	err = SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var r entityGenAttrRow
			var nId int
			if err := rows.Scan(&nId, &r.GenHid, &r.AttrId); err != nil {
				return err
			}

			i, ok := m[nId]
			if !ok {
				return nil // run id not found: run list updated between selects
			}
			mh, ok := ei[nId]
			if !ok {
				return nil // run id not found: run list updated between selects
			}
			// append generation attribute to that generation Hid
			if j, ok := mh[r.GenHid]; ok {
				rl[i].EntityGen[j].GenAttr = append(rl[i].EntityGen[j].GenAttr, r)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}

	// append run_entity rows: generation Hid and value digest
	q = "SELECT H.run_id, RE.entity_gen_hid, RE.row_count, RE.value_digest" +
		" FROM run_lst H" +
		" INNER JOIN run_entity RE ON (RE.run_id = H.run_id)" +
		" WHERE H.model_id = " + smId +
		statusFilter +
		" ORDER BY 1, 2"

	err = SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var r RunEntityRow
			var svd sql.NullString
			if err := rows.Scan(&r.RunId, &r.GenHid, &r.RowCount, &svd); err != nil {
				return err
			}
			if svd.Valid {
				r.ValueDigest = svd.String
			}

			idx, ok := m[r.RunId] // find run id index
			if !ok {
				return nil // skip run if not in previous run list
			}

			rl[idx].RunEntity = append(rl[idx].RunEntity, r) // append run_entity row
			return nil
		})
	if err != nil {
		return nil, err
	}

	return rl, nil
}
