// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"container/list"
	"database/sql"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
)

// copy model from source database to destination database
func dbToDb(modelName string, modelDigest string, runOpts *config.RunOptions) error {

	// validate source and destination
	csInp, dnInp := db.IfEmptyMakeDefaultReadOnly(modelName, runOpts.String(fromSqliteArgKey), runOpts.String(dbConnStrArgKey), runOpts.String(dbDriverArgKey))
	csOut, dnOut := db.IfEmptyMakeDefault(modelName, runOpts.String(toSqliteArgKey), runOpts.String(toDbConnStrArgKey), runOpts.String(toDbDriverArgKey))

	if csInp == csOut && dnInp == dnOut {
		return helper.ErrorNew("source same as destination: cannot overwrite model in database")
	}

	// open source database connection and check is it valid
	srcDb, _, err := db.Open(csInp, dnInp, false)
	if err != nil {
		return err
	}
	defer srcDb.Close()

	if err := db.CheckOpenmppSchemaVersion(srcDb); err != nil {
		return err
	}

	// open destination database and check is it valid
	dstDb, dbFacet, err := db.Open(csOut, dnOut, true)
	if err != nil {
		return err
	}
	defer dstDb.Close()

	if err := db.CheckOpenmppSchemaVersion(dstDb); err != nil {
		return err
	}

	// get source model metadata and languages, make a deep copy to use for destination database writing
	err = copyDbToDb(srcDb, dstDb, dbFacet, modelName, modelDigest)
	if err != nil {
		return err
	}
	return nil
}

// copyDbToDb select from source database and insert or update existing
// model metadata in all languages, model runs, workset(s), modeling tasks and task run history.
//
// Model id's and hId's updated with destination database id's.
// For example, in source db model id can be 11 and in destination it will be 200,
// same for all other id's: type Hid, parameter Hid, table Hid, run id, set id, task id, etc.
func copyDbToDb(
	srcDb *sql.DB, dstDb *sql.DB, dbFacet db.Facet, modelName string, modelDigest string) error {

	// source: get model metadata
	srcModel, err := db.GetModel(srcDb, modelName, modelDigest)
	if err != nil {
		return err
	}
	modelName = srcModel.Model.Name // set model name: it can be empty and only model digest specified

	// source: get list of languages
	srcLang, err := db.GetLanguages(srcDb)
	if err != nil {
		return err
	}

	// source: get model text (description and notes) in all languages
	modelTxt, err := db.GetModelText(srcDb, srcModel.Model.ModelId, "", true)
	if err != nil {
		return err
	}

	// source: get model laguage-specific strings in all languages
	mwDef, err := db.GetModelWord(srcDb, srcModel.Model.ModelId, "")
	if err != nil {
		return err
	}

	// source: get model profile: default model profile is profile where name = model name
	modelProfile, err := db.GetProfile(srcDb, modelName)
	if err != nil {
		return err
	}

	// deep copy of model metadata and languages is required
	// because during db writing metadata structs updated with destination database id's,
	// for example, in source db model id can be 11 and in destination it will be 200,
	// same for all other id's: type Hid, parameter Hid, table Hid, entity Hid. run id, set id, task id, etc.
	dstModel, err := srcModel.Clone()
	if err != nil {
		return err
	}
	dstLang, err := srcLang.Clone()
	if err != nil {
		return err
	}

	// destination: insert model metadata into destination database if not exists
	if _, err = db.UpdateModel(dstDb, dbFacet, dstModel); err != nil {
		return err
	}

	// destination: insert or update language list
	if err = db.UpdateLanguage(dstDb, dstLang); err != nil {
		return err
	}

	// destination: get full list of languages in destination database
	dstLang, err = db.GetLanguages(dstDb)
	if err != nil {
		return err
	}

	// destination: insert, update or delete model default profile
	if err = db.UpdateProfile(dstDb, modelProfile); err != nil {
		return err
	}

	// destination: insert or update model text data (description and notes)
	if err = db.UpdateModelText(dstDb, dstModel, dstLang, modelTxt); err != nil {
		return err
	}

	// destination: insert or update model language-specific strings
	if err = db.UpdateModelWord(dstDb, dstModel, dstLang, mwDef); err != nil {
		return err
	}

	// source to destination: copy model runs: parameters, output expressions and accumulators
	err = copyRunListDbToDb(srcDb, dstDb, dbFacet, srcModel, dstModel, dstLang)
	if err != nil {
		return err
	}

	// source to destination: copy all readonly worksets parameters
	err = copyWorksetListDbToDb(srcDb, dstDb, srcModel, dstModel, dstLang)
	if err != nil {
		return err
	}

	// source to destination: copy all modeling tasks
	err = copyTaskListDbToDb(srcDb, dstDb, srcModel, dstModel, dstLang)
	if err != nil {
		return err
	}

	return nil
}

// return closure to iterate over list until the last element
func makeFromList(srcLst *list.List) func() (interface{}, error) {

	c := srcLst.Front()

	from := func() (interface{}, error) {
		if c == nil {
			return nil, nil // end of data
		}

		cell := c.Value
		c = c.Next()
		return cell, nil
	}
	return from
}
