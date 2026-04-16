// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"sync"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"golang.org/x/text/language"
)

// ModelCatalog is a list of the models and database connections.
// If model directory specified then model catalog include model.sqlite files from model directory.
type ModelCatalog struct {
	theLock         sync.Mutex // mutex to lock for model list operations
	isDirEnabled    bool       // if true then use sqlite files from model directory
	modelDir        string     // models bin directory, it is a root dir under which model.sqlite and model.exe expected to be located
	modelLogDir     string     // default model log directory
	isLogDirEnabled bool       // if true then default log directory exist
	lastTimeStamp   string     // most recent timestamp
	modelLst        []modelDef // list of model metadata and associated database connections
}

// list of models and database connections
var theCatalog ModelCatalog

// ModelCatalogConfig is "public" state of model catalog for json import-export
type ModelCatalogConfig struct {
	ModelDir        string // model bin directory
	ModelLogDir     string // default model log directory
	IsLogDirEnabled bool   // if true then default log directory exist
	LastTimeStamp   string // most recent timestamp
}

// modelDef is database connection and model metadata database rows
type modelDef struct {
	dbConn        *sql.DB           // database connection
	binDir        string            // database and .exe directory: directory part of models/bin/dir/sub/model.sqlite
	dbPath        string            // absolute path to sqlite database file: /root/models/bin/dir/sub/model.sqlite
	relPath       string            // relative path to sqlite database file: relative to model root and slashed: dir/sub/model.sqlite
	logDir        string            // model log directory
	isLogDir      bool              // if true then use model log directory for model run logs
	isIni         bool              // if true the default ini file exists: models/bin/dir/sub/modelName.ini
	meta          *db.ModelMeta     // model metadata, language-neutral part, should not be nil
	isTxtMetaFull bool              // if true then ModelTxtMeta fully loaded else only []ModelTxtRow
	txtMeta       *db.ModelTxtMeta  // if not nil then language-specific model metadata
	langCodes     []string          // language codes, first is default language
	matcher       language.Matcher  // matcher to search text by language
	langMeta      *db.LangMeta      // list of languages: one list per db connection, order of languages NOT the same as language codes
	modelWord     *db.ModelWordMeta // if not nil then list of model words, order of languages NOT the same as language codes
	msg           []db.LangMsg      // list of languages and translated strings: merge of model.message.ini, common.message.ini, model_word, lang_word
	extra         string            // if not empty then model extra content from models/bin/dir/model.extra.json
}

// modelBasic is basic model info: name, digest, files location
type modelBasic struct {
	model    db.ModelDicRow // model_dic db row
	binDir   string         // database and .exe directory: directory part of models/bin/dir/sub/model.sqlite
	dbPath   string         // absolute path to sqlite database file: models/bin/dir/sub/model.sqlite
	relPath  string         // relative path to sqlite database file: relative to model root and slashed: dir/sub/model.sqlite
	logDir   string         // model log directory
	isLogDir bool           // if true then use model log directory for model run logs
	isIni    bool           // if true the default ini file exists: models/bin/dir/sub/modelName.ini
	extra    string         // if not empty then model extra content from models/bin/dir/model.extra.json
}

// ModelWordLabel is (code, label) rows from lang_word and model_word language-specific db tables.
// It is either in user preferred language or model default language.
type ModelWordLabel struct {
	ModelName     string      // model name for text metadata
	ModelDigest   string      // model digest for text metadata
	LangCode      string      // language code selected for lang_word table rows
	LangWords     []codeLabel // lang_word db table rows as (code, value)
	ModelLangCode string      // language code selected for model_word table rows
	ModelWords    []codeLabel // model_word db table rows as (code, value)
	MsgLangCode   string      // language code selected for translated strings
	Msg           []codeLabel // translated strings: merge of model.message.ini, common.message.ini, model_word, lang_word
}

// codeLabel is code + label pair, for example, language-specific "words" db table row.
type codeLabel struct {
	Code  string
	Label string
}
