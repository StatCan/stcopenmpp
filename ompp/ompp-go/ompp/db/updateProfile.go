// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"database/sql"
)

// UpdateProfile insert new or replace existing profile in profile_lst and profile_option tables.
//
// It always replacing all existing rows by delete from profile_lst and profile_option where profile_name = profile.Name
// and insert new rows into profile_lst and profile_option tables.
func UpdateProfile(dbConn *sql.DB, profile *ProfileMeta) error {

	// source is empty: nothing to do, exit
	if profile == nil || profile.Name == "" {
		return nil
	}

	// do update in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	if err = doDeleteProfile(trx, profile.Name); err != nil {
		trx.Rollback()
		return err
	}
	if err = doInsertProfile(trx, profile); err != nil {
		trx.Rollback()
		return err
	}
	trx.Commit()
	return nil
}

// DeleteProfile delete rows from profile in profile_lst and profile_option tables.
func DeleteProfile(dbConn *sql.DB, name string) error {

	// do update in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	if err = doDeleteProfile(trx, name); err != nil {
		trx.Rollback()
		return err
	}
	trx.Commit()
	return nil
}

// doDeleteProfile delete profile rows from profile_lst and profile_option tables.
// It does update as part of transaction.
func doDeleteProfile(trx *sql.Tx, name string) error {

	// delete existing profile
	qn := ToQuoted(name)

	err := TrxUpdate(trx, "DELETE FROM profile_option WHERE profile_name = "+qn)
	if err != nil {
		return err
	}
	err = TrxUpdate(trx, "DELETE FROM profile_lst WHERE profile_name = "+qn)
	if err != nil {
		return err
	}
	return nil
}

// doInsertProfile insert new profile in profile_lst and profile_option tables.
// It does update as part of transaction.
func doInsertProfile(trx *sql.Tx, profile *ProfileMeta) error {

	// exit if profile name is empty
	if profile.Name == "" {
		return nil
	}

	// insert profile name
	qn := toQuotedMax(profile.Name, nameDbMax)

	err := TrxUpdate(trx, "INSERT INTO profile_lst (profile_name) VALUES ("+qn+")")
	if err != nil {
		return err
	}

	// insert profile options
	for key, val := range profile.Opts {

		err = TrxUpdate(trx, "INSERT INTO profile_option (profile_name, option_key, option_value)"+
			" VALUES ("+qn+", "+toQuotedMax(key, nameDbMax)+", "+toQuotedMax(val, optionDbMax)+")")
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateProfileOption insert new or replace existing profile option in profile_option table.
// It is also will insert profile name into profile_lst table if such name does not already exist.
func UpdateProfileOption(dbConn *sql.DB, name, key, val string) error {

	// source is empty: nothing to do, exit
	if name == "" || key == "" {
		return nil
	}

	// do update in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	if err = doUpdateProfileOption(trx, name, key, val); err != nil {
		trx.Rollback()
		return err
	}
	trx.Commit()
	return nil
}

// DeleteProfileOption delete row from profile_option table.
func DeleteProfileOption(dbConn *sql.DB, name, key string) error {

	// do delete in transaction scope
	trx, err := dbConn.Begin()
	if err != nil {
		return err
	}
	if err = TrxUpdate(trx,
		"DELETE FROM profile_option WHERE profile_name = "+ToQuoted(name)+" AND option_key = "+ToQuoted(key)); err != nil {
		trx.Rollback()
		return err
	}
	trx.Commit()
	return nil
}

// doUpdateProfileOption insert new or replace existing profile option in profile_option table.
// It is also will insert profile name into profile_lst table if such name does not already exist.
// It does update as part of transaction.
func doUpdateProfileOption(trx *sql.Tx, name, key, val string) error {

	// insert profile name if not already exist
	qn := toQuotedMax(name, nameDbMax)

	err := TrxUpdate(trx,
		"INSERT INTO profile_lst (profile_name)"+
			" SELECT "+qn+" FROM profile_lst S"+
			" WHERE NOT EXISTS "+
			" ("+
			" SELECT * FROM profile_lst E WHERE E.profile_name = "+qn+
			")")
	if err != nil {
		return err
	}

	// delete existing profile_option row and insert new option
	err = TrxUpdate(trx, "DELETE FROM profile_option WHERE profile_name = "+qn+" AND option_key = "+ToQuoted(key))
	if err != nil {
		return err
	}
	err = TrxUpdate(trx, "INSERT INTO profile_option (profile_name, option_key, option_value)"+
		" VALUES ("+qn+", "+toQuotedMax(key, nameDbMax)+", "+toQuotedMax(val, optionDbMax)+")")
	if err != nil {
		return err
	}
	return nil
}
