// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
	"strconv"
)

// GetProfileList return profile names: profile_lst table rows.
//
// Profile is a named group of (key, value) options, similar to ini-file.
// Default model options has profile_name = model_name.
func GetProfileList(dbConn *sql.DB) ([]string, error) {

	var rs []string

	err := SelectRows(dbConn,
		"SELECT profile_name FROM profile_lst ORDER BY 1",
		func(rows *sql.Rows) error {
			var r string
			if err := rows.Scan(&r); err != nil {
				return err
			}
			rs = append(rs, r)
			return nil
		})
	if err != nil {
		return nil, err
	}

	return rs, nil
}

// GetProfile return profile_option table rows as (key, value) map.
//
// Profile is a named group of (key, value) options, similar to ini-file.
// Default model options has profile_name = model_name.
func GetProfile(dbConn *sql.DB, name string) (*ProfileMeta, error) {

	meta := ProfileMeta{Name: name}

	kv, err := getOpts(dbConn,
		"SELECT option_key, option_value FROM profile_option WHERE profile_name = "+ToQuoted(name))
	if err != nil {
		return nil, err
	}
	meta.Opts = kv

	return &meta, nil
}

// GetRunOptions return run_option table rows as (key, value) map.
func GetRunOptions(dbConn *sql.DB, runId int) (map[string]string, error) {

	return getOpts(dbConn,
		"SELECT option_key, option_value FROM run_option WHERE run_id = "+strconv.Itoa(runId))
}

// getOpts return option table (profile_option or run_option) rows as (key, value) map.
func getOpts(dbConn *sql.DB, query string) (map[string]string, error) {

	kv := make(map[string]string)

	err := SelectRows(dbConn, query,
		func(rows *sql.Rows) error {
			var key, val string
			if err := rows.Scan(&key, &val); err != nil {
				return err
			}
			kv[key] = val
			return nil
		})
	if err != nil {
		return nil, err
	}

	return kv, nil
}

// GetModelOptions return model run_option table rows as map of maps: map(run_id, map(key, value)).
func GetModelRunOptions(dbConn *sql.DB, modelId int) (map[int]map[string]string, error) {

	return getRunOpts(dbConn,
		"SELECT"+
			" M.run_id, M.option_key, M.option_value"+
			" FROM run_option M"+
			" INNER JOIN run_lst H ON (H.run_id = M.run_id)"+
			" WHERE H.model_id = "+strconv.Itoa(modelId)+
			" ORDER BY 1, 2")
}

// getRunOpts return run_option rows as map of maps: map(run_id, map(key, value)).
func getRunOpts(dbConn *sql.DB, query string) (map[int]map[string]string, error) {

	rkv := make(map[int]map[string]string)

	err := SelectRows(dbConn, query,
		func(rows *sql.Rows) error {
			var runId int
			var key, val string
			if err := rows.Scan(&runId, &key, &val); err != nil {
				return err
			}
			if _, ok := rkv[runId]; !ok {
				rkv[runId] = make(map[string]string)
			}
			rkv[runId][key] = val
			return nil
		})
	if err != nil {
		return nil, err
	}

	return rkv, nil
}
