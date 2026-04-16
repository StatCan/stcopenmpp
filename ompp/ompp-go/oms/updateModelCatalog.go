// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/language"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// RefreshSqlite open db-connection to model.sqlite files in model directory and read model_dic row for each model.
// If multiple version of the same model (equal by digest) exist in different files then only one is used.
// All previously opened db connections are closed.
func (mc *ModelCatalog) refreshSqlite(modelDir, modelLogDir string) error {

	// model directory must exist
	isDir := modelDir != "" && modelDir != "."
	if isDir {
		isDir = helper.IsDirExist(modelDir)
	}
	if !isDir {
		return errors.New("Error: model directory not exist or not accesible: " + modelDir)
	}

	// model log directory is optional, if empty or not exists then model log disabled
	isLogDir := modelLogDir != "" && modelLogDir != "."
	if isLogDir {
		isLogDir = helper.IsDirExist(modelLogDir)
	}

	// get list of models/bin/*.sqlite files
	pathLst := []string{}
	err := filepath.WalkDir(modelDir, func(src string, de os.DirEntry, err error) error {
		if err != nil {
			if err != filepath.SkipDir {
				omppLog.Log("Error at refresh model catalog, path: ", src, " : ", err.Error())
			}
			return err
		}
		if strings.EqualFold(filepath.Ext(src), ".sqlite") {
			pathLst = append(pathLst, src)
		}
		return nil
	})
	if err != nil {
		omppLog.Log("Error: fail to scan model directory: ", err.Error())
		return errors.New("Error: fail to scan model directory")
	}
	sort.Strings(pathLst) // sort by path to model.sqlite: same as sort by model name in default case

	// make list of models from model.sqlite files:
	// open db connection to model.sqlite and read list of model_dic rows.
	// if model exist in multiple sqlite files then only one is used.
	dglLst := []string{}
	mLst := []modelDef{}

	for _, fp := range pathLst {

		addLst, e := modelsFromSqliteFile(fp, dglLst, modelDir, isLogDir, modelLogDir)
		if e != nil || len(addLst) <= 0 {
			continue
		}
		mLst = append(mLst, addLst...)

		for k := range addLst {
			dglLst = append(dglLst, addLst[k].meta.Model.Digest)
		}
	}

	// lock and update model catalog
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	// update model directories
	mc.modelDir = modelDir
	mc.isDirEnabled = isDir
	mc.modelLogDir = modelLogDir
	mc.isLogDirEnabled = isLogDir

	// close existing connections and store updated list of models and db connections
	for k := range mc.modelLst {
		if err := mc.modelLst[k].dbConn.Close(); err != nil {
			omppLog.Log("Error: close db connection error: " + err.Error())
		}
	}

	mc.modelLst = mLst // set new list of the models
	return nil
}

// open db file, read models metadata and append it into catalog
// return error if any model digest already exists in catalog
func (mc *ModelCatalog) loadModelDbFile(srcPath string) (int, error) {

	mbinDir, _ := theCatalog.getModelDir()
	logDir, isLog := theCatalog.getModelLogDir()

	mLst, err := modelsFromSqliteFile(srcPath, []string{}, mbinDir, isLog, logDir)
	if err != nil {
		return 0, err
	}

	// lock and update model catalog
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	// check if any models are already exist in the catalog
	for k := range mc.modelLst {
		for j := range mLst {
			if mLst[j].meta.Model.Digest == mc.modelLst[k].meta.Model.Digest {

				err = errors.New("Error: model already exist in catalog" + ": " + mLst[j].meta.Model.Name + " " + mLst[j].meta.Model.Digest)
				omppLog.Log(err.Error())

				if e := mLst[j].dbConn.Close(); e != nil {
					omppLog.Log("Error: close db connection error: " + e.Error())
				}
				return 0, err
			}
		}
	}
	mc.modelLst = append(mc.modelLst, mLst...) // append model to catalog and sort models by file paths

	slices.SortStableFunc(mc.modelLst, func(left, right modelDef) int { return strings.Compare(left.relPath, right.relPath) })

	return len(mLst), nil
}

// open SQLite db connection and retrive model or list of models, skip models which are in digest list already
func modelsFromSqliteFile(srcPath string, dgstLst []string, modelDir string, isLogDir bool, modelLogDir string) ([]modelDef, error) {

	// read common.message.ini
	cmIni := []helper.IniEntry{}
	if ea, e := config.ReadCommonMessageIni(theCfg.omsBinDir, theCfg.encodingName); e == nil {
		cmIni = ea
	}

	// open db connection and check version of openM++ database
	dbc, _, err := db.Open(db.MakeSqliteDefault(srcPath), db.SQLiteDbDriver, false)
	if err != nil {
		omppLog.Log("Error: ", srcPath, " : ", err.Error())
		return nil, err
	}
	if err := db.CheckOpenmppSchemaVersion(dbc); err != nil {
		omppLog.Log("Error: invalid database, likely not an openM++ database: ", srcPath)
		dbc.Close()
		return nil, err
	}

	dbPath, err := filepath.Abs(srcPath)
	if err != nil {
		omppLog.Log("Error: ", srcPath, " : ", err.Error())
		return nil, err
	}
	dbRel, err := filepath.Rel(modelDir, srcPath)
	if err != nil {
		omppLog.Log("Error: ", srcPath, " : ", err.Error())
		return nil, err
	}

	// read list of models: model_dic rows
	dicLst, err := db.GetModelList(dbc)
	if err != nil || len(dicLst) <= 0 {
		omppLog.Log("Error: ", srcPath, " : ", err.Error())
		dbc.Close()
		return nil, err
	}
	if len(dicLst) <= 0 {
		omppLog.Log("Warning: empty database, no models found: ", srcPath)
		dbc.Close()
		return nil, nil
	}

	langDef, err := db.GetLanguages(dbc)
	if err != nil {
		omppLog.Log("Error: ", srcPath, " : ", err.Error())
		dbc.Close()
		return nil, err
	}
	if langDef == nil {
		omppLog.Log("Warning: no languages found in database: ", srcPath)
		dbc.Close()
		return nil, nil
	}

	// append to list of models if not already exist
	mLst := []modelDef{}

	for idx := range dicLst {

		// skip model if same digest already exist in model list
		isFound := false
		for k := 0; !isFound && k < len(dgstLst); k++ {
			isFound = dicLst[idx].Digest == dgstLst[k]
		}
		for k := 0; !isFound && k < len(mLst); k++ {
			isFound = dicLst[idx].Digest == mLst[k].meta.Model.Digest
		}
		if isFound {
			omppLog.Log("Skip: model already exist in other database: ", dicLst[idx].Name, " ", dicLst[idx].Digest)
			continue
		}

		// read metadata from database
		meta, err := db.GetModelById(dbc, dicLst[idx].ModelId)
		if err != nil {
			omppLog.Log("Error at get model metadata: ", dicLst[idx].Name, " ", dicLst[idx].Digest, ": ", err.Error())
			dbc.Close()
			return nil, err
		}

		// read model_dic_txt rows from database
		txt, err := db.GetModelTextRowById(dbc, dicLst[idx].ModelId, "")
		if err != nil {
			omppLog.Log("Error at get model_dic_txt: ", dicLst[idx].Name, " ", dicLst[idx].Digest, ": ", err.Error())
			dbc.Close()
			return nil, err
		}
		// partial initialization of model text metadata: only model_dic_txt rows
		mt := &db.ModelTxtMeta{
			ModelName:   meta.Model.Name,
			ModelDigest: meta.Model.Digest,
			ModelTxt:    txt}

		// read model_word from database
		wLst, err := db.GetModelWord(dbc, dicLst[idx].ModelId, "")
		if err != nil {
			omppLog.Log("Error at get model language-specific stirngs: ", dicLst[idx].Name, " ", dicLst[idx].Digest, ": ", err.Error())
			dbc.Close()
			return nil, err
		}

		// make list of model translated strings
		// merge of model.message.ini, common.message.ini, model_word, lang_word
		msgLst := db.NewLangMsg(langDef.Lang, wLst.ModelWord)

		// update translated strings usig common.message.ini
		msgLst = db.AppendLangMsgFromIni(msgLst, cmIni)

		// update translated strings using model.message.ini
		dbDir := filepath.Dir(srcPath) // assume model.exe is located in the same directory as db.sqlite

		if mdIni, e := config.ReadMessageIni(dicLst[idx].Name, dbDir, theCfg.encodingName); e == nil {
			msgLst = db.AppendLangMsgFromIni(msgLst, mdIni)
		}

		// make model languages list, starting from default language
		ml := []string{}
		lt := []language.Tag{}

		for k := range msgLst {
			if strings.EqualFold(msgLst[k].LangCode, dicLst[idx].DefaultLangCode) {
				ml = append([]string{msgLst[k].LangCode}, ml...)
				lt = append([]language.Tag{language.Make(msgLst[k].LangCode)}, lt...)
			} else {
				ml = append(ml, msgLst[k].LangCode)
				lt = append(lt, language.Make(msgLst[k].LangCode))
			}
		}

		// read model extra content from models/bin/dir/model.extra.json
		me := ""
		if bt, err := os.ReadFile(filepath.Join(dbDir, dicLst[idx].Name+".extra.json")); err == nil {
			me = string(bt)
		}

		// append to model list
		mLst = append(mLst, modelDef{
			dbConn:        dbc,
			binDir:        dbDir,
			dbPath:        dbPath,
			relPath:       filepath.ToSlash(dbRel),
			logDir:        modelLogDir,
			isLogDir:      isLogDir,
			isIni:         helper.IsFileExist(filepath.Join(dbDir, dicLst[idx].Name+".ini")),
			meta:          meta,
			isTxtMetaFull: false,
			txtMeta:       mt,
			langCodes:     ml,
			langMeta:      langDef,
			matcher:       language.NewMatcher(lt),
			modelWord:     wLst,
			msg:           msgLst,
			extra:         me})
	}

	// close db connection if there models in that database or all models already in the model list
	if len(mLst) <= 0 {
		dbc.Close()
	}
	return mLst, nil
}

// close all db-connection to model.sqlite files and clear model list.
func (mc *ModelCatalog) closeAll() error {

	// lock and update model catalog
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	// close existing db connections
	var firstErr error
	for k := range mc.modelLst {
		if err := mc.modelLst[k].dbConn.Close(); err != nil {
			omppLog.Log("Error: close db connection error: " + err.Error())
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	// clear model list
	mc.modelLst = []modelDef{}
	return firstErr
}

// close model db-connection and remove model from models list.
func (mc *ModelCatalog) closeModel(dn string) (string, string, error) {

	if dn == "" {
		return "", "", nil
	}
	// lock and update model catalog
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	// close model db connection and remove model from the list
	isFound := false
	name := ""
	binDir := ""
	n := 0

	for k := range mc.modelLst {

		if !isFound && (mc.modelLst[k].meta.Model.Digest == dn || mc.modelLst[k].meta.Model.Name == dn) {
			if err := mc.modelLst[k].dbConn.Close(); err != nil {
				omppLog.Log("Error: close db connection error" + ": " + dn + " : " + err.Error())
				return "", "", err
			}
			isFound = true
			name = mc.modelLst[k].meta.Model.Name
			binDir = mc.modelLst[k].binDir
			continue
		}
		mc.modelLst[n] = mc.modelLst[k]
		n++
	}
	if isFound {
		mc.modelLst = mc.modelLst[:n]
	}
	return name, binDir, nil
}

// close model and delete all model files: exe, sqlite, ini,...
func (mc *ModelCatalog) deleteModel(dn string) error {

	if dn == "" {
		return nil
	}

	// close the model and remove it from model catalog
	name, modelDir, err := theCatalog.closeModel(dn)
	if err != nil {
		return err
	}
	if name == "" { // model not found
		return errors.New("Error: model not found " + dn)
	}

	// find all files in model directory where name is:
	//   ModelName.* ModelName_mpi.* ModelNameD.* ModelNameD_mpi.*
	pathLst := []string{}

	ff := func(pattern string) error {
		p, e := filepath.Glob(pattern)
		if e != nil {
			omppLog.Log("Error: fail to scan model directory: ", pattern, ": ", e.Error())
			return errors.New("Error: fail to scan model directory")
		}
		if len(p) > 0 {
			pathLst = append(pathLst, p...)
		}
		return nil
	}
	if err = ff(filepath.Join(modelDir, name) + ".*"); err != nil {
		return err
	}
	if err = ff(filepath.Join(modelDir, name) + "D.*"); err != nil {
		return err
	}
	if err = ff(filepath.Join(modelDir, name) + "_mpi.*"); err != nil {
		return err
	}
	if err = ff(filepath.Join(modelDir, name) + "D_mpi.*"); err != nil {
		return err
	}

	// delete model files
	for _, p := range pathLst {
		if ok := fileDeleteAndLog(true, p); !ok {
			return errors.New("Error: unable to delete model file(s)")
		}
	}
	return nil
}

// getNewTimeStamp return new unique timestamp and source time of it.
func (mc *ModelCatalog) getNewTimeStamp() (string, time.Time) {
	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	tNow := time.Now()
	ts := helper.MakeTimeStamp(tNow)
	if ts == mc.lastTimeStamp {
		time.Sleep(2 * time.Millisecond)
		tNow = time.Now()
		ts = helper.MakeTimeStamp(tNow)
	}
	mc.lastTimeStamp = ts
	return ts, tNow
}

// Update model text metadata in catalog:
// Set boolean flag to indicate if text metadata fully loaded from database and ModelTxtMeta itself.
// Return false if model digest not found in catalog.
func (mc *ModelCatalog) setModelTextMeta(digest string, isFull bool, txtMeta *db.ModelTxtMeta) bool {

	if txtMeta == nil {
		return false // model text is empty
	}

	mc.theLock.Lock()
	defer mc.theLock.Unlock()

	idx, ok := mc.indexByDigest(digest)
	if !ok {
		return false // model not found, empty result
	}

	mc.modelLst[idx].isTxtMetaFull = isFull
	mc.modelLst[idx].txtMeta = txtMeta
	return true
}
