// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"errors"
	"strconv"
)

// GetWorkset return working set by id: workset_lst table row.
func GetWorkset(dbConn *sql.DB, setId int) (*WorksetRow, error) {
	return getWsRow(dbConn,
		"SELECT"+
			" W.set_id, W.base_run_id, W.model_id, W.set_name, W.is_readonly, W.update_dt"+
			" FROM workset_lst W"+
			" WHERE W.set_id = "+strconv.Itoa(setId))
}

// GetDefaultWorkset return default working set for the model: workset_lst table row.
//
// Default workset is a first workset for the model, each model must have default workset.
func GetDefaultWorkset(dbConn *sql.DB, modelId int) (*WorksetRow, error) {
	return getWsRow(dbConn,
		"SELECT"+
			" W.set_id, W.base_run_id, W.model_id, W.set_name, W.is_readonly, W.update_dt"+
			" FROM workset_lst W"+
			" WHERE W.set_id ="+
			" (SELECT MIN(M.set_id) FROM workset_lst M WHERE M.model_id = "+strconv.Itoa(modelId)+")")
}

// GetWorksetByName return working set by name.
//
// If model has multiple worksets with that name then return first set.
func GetWorksetByName(dbConn *sql.DB, modelId int, name string) (*WorksetRow, error) {
	return getWsRow(dbConn,
		"SELECT"+
			" W.set_id, W.base_run_id, W.model_id, W.set_name, W.is_readonly, W.update_dt"+
			" FROM workset_lst W"+
			" WHERE W.set_id ="+
			" ("+
			" SELECT MIN(M.set_id) FROM workset_lst M"+
			" WHERE M.model_id = "+strconv.Itoa(modelId)+
			" AND M.set_name = "+ToQuoted(name)+
			" )")
}

// GetWorksetList return list of model worksets: workset_lst rows.
func GetWorksetList(dbConn *sql.DB, modelId int) ([]WorksetRow, error) {

	// model not found: model id must be positive
	if modelId <= 0 {
		return nil, nil
	}

	// select worksets by model id
	q := "SELECT" +
		" W.set_id, W.base_run_id, W.model_id, W.set_name, W.is_readonly, W.update_dt" +
		" FROM workset_lst W" +
		" WHERE W.model_id = " + strconv.Itoa(modelId) +
		" ORDER BY 1"

	setRs, err := getWsLst(dbConn, q)
	if err != nil {
		return nil, err
	}
	if len(setRs) <= 0 { // no worksets found
		return nil, err
	}

	return setRs, nil
}

// GetWorksetListText return list of model worksets, description and notes: workset_lst and workset_txt rows.
//
// If langCode not empty then only specified language selected else all languages
func GetWorksetListText(dbConn *sql.DB, modelId int, langCode string) ([]WorksetRow, []WorksetTxtRow, error) {

	// model not found: model id must be positive
	if modelId <= 0 {
		return nil, nil, nil
	}

	// select worksets by model id
	q := "SELECT" +
		" W.set_id, W.base_run_id, W.model_id, W.set_name, W.is_readonly, W.update_dt" +
		" FROM workset_lst W" +
		" WHERE W.model_id = " + strconv.Itoa(modelId) +
		" ORDER BY 1"

	setRs, err := getWsLst(dbConn, q)
	if err != nil {
		return nil, nil, err
	}
	if len(setRs) <= 0 { // no worksets found
		return nil, nil, err
	}

	// select worksets description and notes by model id and language
	q = "SELECT M.set_id, M.lang_id, L.lang_code, M.descr, M.note" +
		" FROM workset_txt M" +
		" INNER JOIN workset_lst H ON (H.set_id = M.set_id)" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE H.model_id = " + strconv.Itoa(modelId)
	if langCode != "" {
		q += " AND L.lang_code = " + ToQuoted(langCode)
	}
	q += " ORDER BY 1, 2"

	txtRs, err := getWsText(dbConn, q)
	if err != nil {
		return nil, nil, err
	}
	return setRs, txtRs, nil
}

// getWsRow return workset_lst table row.
func getWsRow(dbConn *sql.DB, query string) (*WorksetRow, error) {

	var setRow WorksetRow

	err := SelectFirst(dbConn, query,
		func(row *sql.Row) error {
			var rId sql.NullInt64
			nReadonly := 0
			if err := row.Scan(
				&setRow.SetId, &rId, &setRow.ModelId, &setRow.Name, &nReadonly, &setRow.UpdateDateTime); err != nil {
				return err
			}
			setRow.IsReadonly = nReadonly != 0 // oracle: smallint is float64
			if rId.Valid {
				setRow.BaseRunId = int(rId.Int64)
			}
			return nil
		})
	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, err
	}

	return &setRow, nil
}

// getWsLst return workset_lst table rows.
func getWsLst(dbConn *sql.DB, query string) ([]WorksetRow, error) {

	// get list of workset for that model id
	var setRs []WorksetRow

	err := SelectRows(dbConn, query,
		func(rows *sql.Rows) error {
			var r WorksetRow
			var rId sql.NullInt64
			nReadonly := 0
			if err := rows.Scan(
				&r.SetId, &rId, &r.ModelId, &r.Name, &nReadonly, &r.UpdateDateTime); err != nil {
				return err
			}
			r.IsReadonly = nReadonly != 0 // oracle: smallint is float64

			if rId.Valid {
				r.BaseRunId = int(rId.Int64)
			}
			setRs = append(setRs, r)
			return nil
		})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return setRs, nil
}

// GetWorksetText return workset description and notes: workset_txt table rows.
//
// If langCode not empty then only specified language selected else all languages
func GetWorksetText(dbConn *sql.DB, setId int, langCode string) ([]WorksetTxtRow, error) {

	q := "SELECT M.set_id, M.lang_id, L.lang_code, M.descr, M.note" +
		" FROM workset_txt M" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE M.set_id = " + strconv.Itoa(setId)
	if langCode != "" {
		q += " AND L.lang_code = " + ToQuoted(langCode)
	}
	q += " ORDER BY 1, 2"

	return getWsText(dbConn, q)
}

// getWsText return workset description and notes: workset_txt table rows.
func getWsText(dbConn *sql.DB, query string) ([]WorksetTxtRow, error) {

	var txtLst []WorksetTxtRow

	err := SelectRows(dbConn, query,
		func(rows *sql.Rows) error {
			var r WorksetTxtRow
			var lId int
			var note sql.NullString
			if err := rows.Scan(
				&r.SetId, &lId, &r.LangCode, &r.Descr, &note); err != nil {
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

// GetWorksetRunIds return ids of model run results (run_id) where input parameters are from specified working set.
//
// Only successfully completed run ids returned, not failed, not "in progress".
// This method is "local" to database and if data transfered between databases it very likely return wrong results.
// This method is not recommended, use modeling task to establish relationship between input set and model run.
func GetWorksetRunIds(dbConn *sql.DB, setId int) ([]int, error) {

	var idRs []int

	err := SelectRows(dbConn,
		"SELECT RL.run_id"+
			" FROM run_lst RL"+
			" INNER JOIN run_option RO ON (RO.run_id = RL.run_id)"+
			" WHERE RL.status = "+ToQuoted(DoneRunStatus)+
			" AND RO.option_key = 'OpenM.SetId'"+
			" AND RO.option_value = "+ToQuoted(strconv.Itoa(setId))+
			" ORDER BY 1",
		func(rows *sql.Rows) error {
			var rId int
			if err := rows.Scan(&rId); err != nil {
				return err
			}
			idRs = append(idRs, rId)
			return nil
		})
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return idRs, nil
}

// GetWorksetParam return sub-values count and default sub-value id for workset parameter:
// SELECT sub_count, default_sub_id FROM workset_parameter
func GetWorksetParam(dbConn *sql.DB, setId int, paramHid int) (int, int, error) {

	var nSub, defId int

	err := SelectFirst(dbConn,
		"SELECT sub_count, default_sub_id FROM workset_parameter WHERE set_id = "+strconv.Itoa(setId)+" AND parameter_hid = "+strconv.Itoa(paramHid),
		func(row *sql.Row) error {
			if err := row.Scan(&nSub, &defId); err != nil {
				return err
			}
			return nil
		})
	if err != nil && err != sql.ErrNoRows {
		return 0, 0, err
	}
	return nSub, defId, nil
}

// GetWorksetParamList return Hid, sub-values count and default sub-value id for all parameters included in that workset:
// SELECT parameter_hid, sub_count, default_sub_id FROM workset_parameter
func GetWorksetParamList(dbConn *sql.DB, setId int) ([]int, []int, []int, error) {

	var hidRs, countRs, defIdRs []int

	err := SelectRows(dbConn,
		"SELECT parameter_hid, sub_count, default_sub_id FROM workset_parameter WHERE set_id = "+strconv.Itoa(setId)+" ORDER BY 1",
		func(rows *sql.Rows) error {
			var h, n, i int
			if err := rows.Scan(&h, &n, &i); err != nil {
				return err
			}
			hidRs = append(hidRs, h)
			countRs = append(countRs, n)
			defIdRs = append(defIdRs, i)
			return nil
		})
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, nil, err
	}
	return hidRs, countRs, defIdRs, nil
}

// GetWorksetParamText return parameter value notes: workset_parameter_txt table rows.
//
// If langCode not empty then only specified language selected else all languages
func GetWorksetParamText(dbConn *sql.DB, setId int, paramHid int, langCode string) ([]WorksetParamTxtRow, error) {

	q := "SELECT M.set_id, M.parameter_hid, M.lang_id, L.lang_code, M.note" +
		" FROM workset_parameter_txt M" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE M.set_id = " + strconv.Itoa(setId) +
		" AND M.parameter_hid = " + strconv.Itoa(paramHid)
	if langCode != "" {
		q += " AND L.lang_code = " + ToQuoted(langCode)
	}
	q += " ORDER BY 1, 2, 3"

	return getWsParamText(dbConn, q)
}

// GetWorksetAllParamText return all workset parameters value notes: workset_parameter_txt table rows.
//
// If langCode not empty then only specified language selected else all languages
func GetWorksetAllParamText(dbConn *sql.DB, setId int, langCode string) ([]WorksetParamTxtRow, error) {

	q := "SELECT M.set_id, M.parameter_hid, M.lang_id, L.lang_code, M.note" +
		" FROM workset_parameter_txt M" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE M.set_id = " + strconv.Itoa(setId)
	if langCode != "" {
		q += " AND L.lang_code = " + ToQuoted(langCode)
	}
	q += " ORDER BY 1, 2, 3"

	return getWsParamText(dbConn, q)
}

// getWsParamText return parameter value notes: workset_parameter_txt table rows.
func getWsParamText(dbConn *sql.DB, query string) ([]WorksetParamTxtRow, error) {

	var txtLst []WorksetParamTxtRow

	err := SelectRows(dbConn, query,
		func(rows *sql.Rows) error {
			var r WorksetParamTxtRow
			var lId int
			var note sql.NullString
			if err := rows.Scan(
				&r.SetId, &r.ParamHid, &lId, &r.LangCode, &note); err != nil {
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

// GetWorksetFull return full workset metadata:
// workset_lst, workset_txt, workset_parameter, workset_parameter_txt table rows.
//
// If langCode not empty then only specified language selected else all languages
func GetWorksetFull(dbConn *sql.DB, setRow *WorksetRow, langCode string) (*WorksetMeta, error) {

	// validate parameters
	if setRow == nil {
		return nil, errors.New("invalid (empty) workset row, it may be workset not found")
	}

	// where filters
	var langFilter string
	if langCode != "" {
		langFilter = " AND L.lang_code = " + ToQuoted(langCode)
	}

	// workset header: workset_lst row, model name and digest
	ws := &WorksetMeta{Set: *setRow}

	// workset_txt rows
	q := "SELECT M.set_id, M.lang_id, L.lang_code, M.descr, M.note" +
		" FROM workset_txt M" +
		" INNER JOIN workset_lst H ON (H.set_id = M.set_id)" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE H.set_id = " + strconv.Itoa(setRow.SetId) +
		langFilter +
		" ORDER BY 1, 2"

	setTxtRs, err := getWsText(dbConn, q)
	if err != nil {
		return nil, err
	}
	ws.Txt = setTxtRs

	// workset_parameter: select list of parameters Hid
	q = "SELECT M.parameter_hid, sub_count, default_sub_id" +
		" FROM workset_parameter M" +
		" INNER JOIN workset_lst H ON (H.set_id = M.set_id)" +
		" WHERE H.set_id = " + strconv.Itoa(setRow.SetId) +
		" ORDER BY 1"
	hi := make(map[int]int) // map (parameter Hid) => index in parameter array

	err = SelectRows(dbConn, q,
		func(rows *sql.Rows) error {

			var hId, nSub, defId int
			err := rows.Scan(&hId, &nSub, &defId)
			if err != nil {
				return err
			}
			r := worksetParam{ParamHid: hId, SubCount: nSub, DefaultSubId: defId}

			hi[hId] = len(ws.Param) // index of parameter Hid in parameter list
			ws.Param = append(ws.Param, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	// workset_parameter_txt: select rows and join to parameter by Hid
	q = "SELECT M.set_id, M.parameter_hid, M.lang_id, L.lang_code, M.note" +
		" FROM workset_parameter_txt M" +
		" INNER JOIN workset_lst H ON (H.set_id = M.set_id)" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE H.set_id = " + strconv.Itoa(setRow.SetId) +
		langFilter +
		" ORDER BY 1, 2, 3"

	paramTxtRs, err := getWsParamText(dbConn, q)
	if err != nil {
		return nil, err
	}
	for k := range paramTxtRs {
		if i, ok := hi[paramTxtRs[k].ParamHid]; ok {
			ws.Param[i].Txt = append(ws.Param[i].Txt, paramTxtRs[k])
		}
	}

	return ws, nil
}

// GetWorksetFullList return list of full workset metadata:
// workset_lst, workset_txt, workset_parameter, workset_parameter_txt table rows.
//
// If isReadonly true then return only readonly worksets else all worksets.
// If langCode not empty then only specified language selected else all languages
func GetWorksetFullList(dbConn *sql.DB, modelId int, isReadonly bool, langCode string) ([]WorksetMeta, error) {

	// where filters
	var roFilter string
	if isReadonly {
		roFilter = " AND H.is_readonly <> 0"
	}

	var langFilter string
	if langCode != "" {
		langFilter = " AND L.lang_code = " + ToQuoted(langCode)
	}

	// workset_lst rows
	smId := strconv.Itoa(modelId)

	q := "SELECT" +
		" H.set_id, H.base_run_id, H.model_id, H.set_name, H.is_readonly, H.update_dt" +
		" FROM workset_lst H" +
		" WHERE H.model_id = " + smId +
		roFilter +
		" ORDER BY 1"

	setRs, err := getWsLst(dbConn, q)
	if err != nil {
		return nil, err
	}

	// workset_txt rows
	q = "SELECT M.set_id, M.lang_id, L.lang_code, M.descr, M.note" +
		" FROM workset_txt M" +
		" INNER JOIN workset_lst H ON (H.set_id = M.set_id)" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE H.model_id = " + smId +
		roFilter +
		langFilter +
		" ORDER BY 1, 2"

	setTxtRs, err := getWsText(dbConn, q)
	if err != nil {
		return nil, err
	}

	// workset_parameter: select using Hid
	q = "SELECT H.set_id, M.parameter_hid, M.sub_count, M.default_sub_id" +
		" FROM workset_parameter M" +
		" INNER JOIN workset_lst H ON (H.set_id = M.set_id)" +
		" WHERE H.model_id = " + smId +
		roFilter +
		" ORDER BY 1, 2"

	var ps [][4]int // array of (set id, parameter hId, sub-value count, default_sub_id)

	err = SelectRows(dbConn, q,
		func(rows *sql.Rows) error {
			var setId, hId, nSub, defId int
			if err := rows.Scan(&setId, &hId, &nSub, &defId); err != nil {
				return err
			}
			ps = append(ps, [4]int{setId, hId, nSub, defId})
			return nil
		})
	if err != nil {
		return nil, err
	}

	// workset_parameter_txt: select using Hid
	q = "SELECT M.set_id, M.parameter_hid, M.lang_id, L.lang_code, M.note" +
		" FROM workset_parameter_txt M" +
		" INNER JOIN workset_lst H ON (H.set_id = M.set_id)" +
		" INNER JOIN lang_lst L ON (L.lang_id = M.lang_id)" +
		" WHERE H.model_id = " + smId +
		roFilter +
		langFilter +
		" ORDER BY 1, 2, 3"

	paramTxtRs, err := getWsParamText(dbConn, q)
	if err != nil {
		return nil, err
	}

	// convert to output result: join workset pieces in struct by set id
	wl := make([]WorksetMeta, len(setRs))
	m := make(map[int]int) // map [set id] => index of workset_lst row

	// workset header: workset_lst row and model name, digest
	for k := range setRs {
		setId := setRs[k].SetId
		wl[k].Set = setRs[k]
		m[setId] = k
	}

	// workset text (description and notes): append to coresponding workset
	for k := range setTxtRs {
		if i, ok := m[setTxtRs[k].SetId]; ok {
			wl[i].Txt = append(wl[i].Txt, setTxtRs[k])
		}
	}

	// workset parameters: append parameters to coresponding workset
	for k := range ps {
		if i, ok := m[ps[k][0]]; ok {
			wl[i].Param = append(
				wl[i].Param,
				worksetParam{ParamHid: ps[k][1], SubCount: ps[k][2], DefaultSubId: ps[k][3]})
		}
	}

	// workset parameters text (parameter value notes):
	// find parameter by Hid in coresponding workset and append value notes
	for k := range paramTxtRs {
		if i, ok := m[paramTxtRs[k].SetId]; ok {
			for j := range wl[i].Param {
				if wl[i].Param[j].ParamHid == paramTxtRs[k].ParamHid {
					wl[i].Param[j].Txt = append(wl[i].Param[j].Txt, paramTxtRs[k])
				}
			}
		}
	}

	return wl, nil
}
