// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"path/filepath"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// copy model from text json and csv files into database
func textToDb(modelName string, runOpts *config.RunOptions) error {

	// validate parameters
	if modelName == "" {
		return helper.ErrorNew("invalid (empty) model name")
	}

	// open source database connection and check is it valid
	dn := runOpts.String(toDbDriverArgKey)
	if dn == "" && runOpts.IsExist(dbDriverArgKey) {
		dn = runOpts.String(dbDriverArgKey)
	}
	cs, dn := db.IfEmptyMakeDefault(modelName, runOpts.String(toSqliteArgKey), runOpts.String(toDbConnStrArgKey), dn)

	dstDb, dbFacet, err := db.Open(cs, dn, true)
	if err != nil {
		return err
	}
	defer dstDb.Close()

	if err := db.CheckOpenmppSchemaVersion(dstDb); err != nil {
		return err
	}

	// use modelName as subdirectory inside of input and output directories or as name of model.zip file
	inpDir := runOpts.String(inputDirArgKey)

	if !runOpts.Bool(zipArgKey) {
		inpDir = filepath.Join(inpDir, modelName) // json and csv files located in modelName subdir
	} else {
		omppLog.Log("Unpack", modelName+".zip")

		outDir := runOpts.String(outputDirArgKey)
		if outDir == "" {
			outDir = filepath.Join(inpDir, modelName)
		}
		if err = helper.UnpackZip(filepath.Join(inpDir, modelName+".zip"), !theCfg.isKeepOutputDir, outDir); err != nil {
			return err
		}
		inpDir = filepath.Join(outDir, modelName)
	}

	// insert model metadata from json file into database
	modelDef, err := fromModelJsonToDb(dstDb, dbFacet, inpDir, modelName)
	if err != nil {
		return err
	}

	// insert languages and model text metadata from json file into database
	langDef, err := fromLangTextJsonToDb(dstDb, modelDef, inpDir)
	if err != nil {
		return err
	}

	// insert model runs data from csv into database:
	// parameters, output expressions and accumulators
	if err = fromRunTextListToDb(dstDb, dbFacet, modelDef, langDef, inpDir); err != nil {
		return err
	}

	// insert model workset data from csv into database: input parameters
	if err = fromWorksetTextListToDb(dstDb, modelDef, langDef, inpDir); err != nil {
		return err
	}

	// insert modeling tasks and tasks run history from json file into database
	if err = fromTaskListJsonToDb(dstDb, modelDef, langDef, inpDir); err != nil {
		return err
	}
	return nil
}

// fromModelJsonToDb reads model metadata from json file and insert it into database.
func fromModelJsonToDb(dbConn *sql.DB, dbFacet db.Facet, inpDir string, modelName string) (*db.ModelMeta, error) {

	// restore  model metadta from json
	js, err := helper.FileToUtf8(filepath.Join(inpDir, modelName+".model.json"), "")
	if err != nil {
		return nil, err
	}
	modelDef := &db.ModelMeta{}

	isExist, err := modelDef.FromJson([]byte(js))
	if err != nil {
		return nil, err
	}
	if !isExist {
		return nil, helper.ErrorNew("model not found:", modelName)
	}
	if modelDef.Model.Name != modelName {
		return nil, helper.ErrorFmt("model name: %s not found in .json file", modelName)
	}

	// insert model metadata into destination database if not exists
	if _, err = db.UpdateModel(dbConn, dbFacet, modelDef); err != nil {
		return nil, err
	}

	// insert, update or delete model default profile
	var modelProfile db.ProfileMeta
	isExist, err = helper.FromJsonFile(filepath.Join(inpDir, modelName+".profile.json"), &modelProfile)
	if err != nil {
		return nil, err
	}
	if isExist && modelProfile.Name == modelName { // if this is profile default model profile then do update
		if err = db.UpdateProfile(dbConn, &modelProfile); err != nil {
			return nil, err
		}
	}

	return modelDef, nil
}

// fromLangTextJsonToDb reads languages and model text from json file and insert it into database.
func fromLangTextJsonToDb(dbConn *sql.DB, modelDef *db.ModelMeta, inpDir string) (*db.LangMeta, error) {

	// restore language list from json and if exist then update db tables
	js, err := helper.FileToUtf8(filepath.Join(inpDir, modelDef.Model.Name+".lang.json"), "")
	if err != nil {
		return nil, err
	}
	langDef := &db.LangMeta{}

	isExist, err := langDef.FromJson([]byte(js))
	if err != nil {
		return nil, err
	}
	if isExist {
		if err = db.UpdateLanguage(dbConn, langDef); err != nil {
			return nil, err
		}
	}

	// get full list of languages
	langDef, err = db.GetLanguages(dbConn)
	if err != nil {
		return nil, err
	}

	// restore text data from json and if exist then update db tables
	var modelTxt db.ModelTxtMeta
	isExist, err = helper.FromJsonFile(filepath.Join(inpDir, modelDef.Model.Name+".text.json"), &modelTxt)
	if err != nil {
		return nil, err
	}
	if isExist {
		if err = db.UpdateModelText(dbConn, modelDef, langDef, &modelTxt); err != nil {
			return nil, err
		}
	}

	// restore model language-specific strings from json and if exist then update db table
	var mwDef db.ModelWordMeta
	isExist, err = helper.FromJsonFile(filepath.Join(inpDir, modelDef.Model.Name+".word.json"), &mwDef)
	if err != nil {
		return nil, err
	}
	if isExist {
		if err = db.UpdateModelWord(dbConn, modelDef, langDef, &mwDef); err != nil {
			return nil, err
		}
	}

	return langDef, nil
}
