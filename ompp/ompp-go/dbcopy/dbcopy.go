// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

/*
dbcopy is command line tool for import-export OpenM++ model metadata, input parameters and run results.

Dbcopy support 5 possible -dbcopy.To directions:

	"text":    copy from database to .json and .csv or .tsv files (this is default)
	"db":      copy from .json and .csv files to database
	"db2db":   copy from one database to other
	"csv":     copy from databse to .csv or .tsv files
	"csv-all": copy from databse to .csv or .tsv files

Dbcopy also can delete entire model or model run results, set of input parameters or modeling task from database (see dbcopy.Delete below).
Dbcopy also can rename model run results, set of input parameters or modeling task in database (see dbcopy.Rename below).

Arguments for dbcopy can be specified on command line or through .ini file:

	dbcopy -ini my.ini
	dbcopy -OpenM.IniFile my-dbcopy.ini

Command line arguments take precedence over ini-file options.

Only model argument does not have default value and must be specified explicitly:

	dbcopy -m modelOne
	dbcopy -dbcopy.ModelName modelOne
	dbcopy -dbcopy.ModelDigest 649f17f26d67c37b78dde94f79772445

Model digest is globally unique and you may want to use it if there are multiple versions of the model.

To produce TSV output files instead of CSV use -dbcopy.IntoTsv option.

Copy to "text": read from database and save into metadata .json and .csv or .tsv values (parameters and output tables):

	dbcopy -m modelOne
	dbcopy -m modelOne -dbcopy.IntoTsv

Copy to "db": read from metadata .json and .csv values and insert or update database:

	dbcopy -m modelOne -dbcopy.To db

Copy to "db2db": direct copy between two databases:

	dbcopy -m modelOne -dbcopy.To db2db -dbcopy.ToSqlite modelOne.sqlite

Copy to "csv": read entire model from database and save into .csv or .tsv files:

	dbcopy -m modelOne -dbcopy.To csv
	dbcopy -m modelOne -dbcopy.To csv -dbcopy.IntoTsv

Separate sub-directory created for each input set and each model run results.

Copy to "csv-all": read entire model from database and save into .csv or .tsv files:

	dbcopy -m modelOne -dbcopy.To csv-all
	dbcopy -m modelOne -dbcopy.To csv-all -dbcopy.IntoTsv

It dumps all input parameters sets into all_input_sets/parameterName.csv (or .tsv) files.
And for all model runs input parameters and output tables saved into all_model_runs/tableName.csv (or .tsv) files.

By default if output directory already exist then dbdopy delete it first to create a clean output results.
If you want to keep existing output directory then use  -dbcopy.KeepOutputDir true:

	dbcopy -m modelOne -dbcopy.KeepOutputDir true

By default entire model data is copied.
It is also possible to copy only:
model run results and input parameters, set of input parameters (workset), modeling task metadata and task run history.

To copy only one set of input parameters:

	dbcopy -m modelOne -s Default
	dbcopy -m modelOne -dbcopy.SetName Default

To copy only one model run results and input parameters:

	dbcopy -m modelOne -dbcopy.RunId 101
	dbcopy -m modelOne -dbcopy.RunDigest d722febf683992aa624ce9844a2e597d
	dbcopy -m modelOne -dbcopy.RunName "My Model Run"

Model run name is not unique and by default first model run with such name is used.
To use last model run or first model run do:

	dbcopy -m modelOne -dbcopy.RunName "My Model Run" -dbcopy.LastRun
	dbcopy -m modelOne -dbcopy.LastRun
	dbcopy -m modelOne -dbcopy.FirstRun

To copy only one modeling task metadata and run history:

	dbcopy -m modelOne -dbcopy.TaskId 1
	dbcopy -m modelOne -dbcopy.TaskName taskOne

It is convenient to pack (unpack) text files into .zip archive:

	dbcopy -m modelOne -dbcopy.Zip=true
	dbcopy -m modelOne -dbcopy.Zip
	dbcopy -m modelOne -dbcopy.SetName Default -dbcopy.Zip

By default model name is used to create output directory for text files or as input directory to import from.
It may be a problem on Linux if current directory already contains executable "modelName".

To specify output or input directory for text files:

	dbcopy -m modelOne -dbcopy.OutputDir one
	dbcopy -m modelOne -dbcopy.OutputDir one -s Default
	dbcopy -m modelOne -dbcopy.InputDir one -dbcopy.To db -dbcopy.ToSqlite oneModel.sqlite

If you are using InputDir or OutputDir result path combined with
model name, model run name or name of input parameters set to prevent path conflicts.
For example:

	dbcopy -m modelOne -dbcopy.OutputDir one -s Default

will place "Default" input set of parameters into directory one/modelOne.set.Default.

If neccesary you can specify exact directory for input parameters by using "-dbcopy.ParamDir" or "-p":

	dbcopy -m modelOne -s Custom -dbcopy.ParamDir two
	dbcopy -m modelOne -s Custom -dbcopy.ParamDir two -dbcopy.Zip
	dbcopy -m modelOne -s Custom -dbcopy.ParamDir two -dbcopy.To db
	dbcopy -m modelOne -s Custom -dbcopy.ParamDir two -dbcopy.To db -dbcopy.Zip
	dbcopy -m modelOne -s Custom -dbcopy.ParamDir two -dbcopy.OutputDir my-m1 -dbcopy.To db -dbcopy.Zip

Dbcopy create output directories (and json files) for model data by combining model name and run name or input set name.
By default names may be combined with run id (set id) to make it unique.
For example:

	json file: modelName.run.1234.MyRun.json
	directory: modelName/run.1234.MyRun

In case of output into csv by default directories and files combined with id's only if run name is not unique.
To explicitly control usage of id's in directory and file names use IdOutputNames=true or IdOutputNames=false:

	dbcopy -m modelOne -dbcopy.To csv
	dbcopy -m modelOne -dbcopy.To csv -dbcopy.IdOutputNames=true
	dbcopy -m modelOne -dbcopy.To csv -dbcopy.IdOutputNames=false

Dbcopy create csv files for model parameters, microdata output tables value(s) and accumulators.
It is often accumulators or microdata not required and you can suppress by using NoAccumulatorsCsv=true or NoMicrodata=true:

	dbcopy -m modelOne -dbcopy.NoAccumulatorsCsv
	dbcopy -m modelOne -dbcopy.NoAccumulatorsCsv=true
	dbcopy -m modelOne -dbcopy.NoAccumulatorsCsv -dbcopy.To csv
	dbcopy -m modelOne -dbcopy.NoAccumulatorsCsv -dbcopy.LastRun
	dbcopy -m modelOne -dbcopy.NoAccumulatorsCsv -dbcopy.TaskName taskOne
	dbcopy -m modelOne -dbcopy.NoMicrodata
	dbcopy -m modelOne -dbcopy.NoMicrodata=true
	dbcopy -m modelOne -dbcopy.NoAccumulatorsCsv -dbcopy.NoMicrodata

By default parameters and output results .csv files contain codes in dimension column(s), e.g.: Sex=[Male,Female].
If you want to create csv files with numeric id's Sex=[0,1] instead then use IdCsv=true option:

	dbcopy -m modelOne -dbcopy.IdCsv
	dbcopy -m modelOne -dbcopy.IdCsv -dbcopy.To csv
	dbcopy -m modelOne -dbcopy.IdCsv -s Default
	dbcopy -m modelOne -dbcopy.IdCsv -dbcopy.RunId 101
	dbcopy -m modelOne -dbcopy.IdCsv -dbcopy.RunDigest d722febf683992aa624ce9844a2e597d
	dbcopy -m modelOne -dbcopy.IdCsv -dbcopy.TaskName taskOne

Dbcopy can transfer the data between differnt versions of the same model or even between different models.
For example, it is possible create new input set of parameters just from csv file(s) with model data, nothing else is required.
On the other hand dbcopy package output data with model metadata (e.g. parameter name, model name, model digest, etc.).
If JSON metadata file(s) are supplied then dbcopy using it for validation to make sure input data match destination model.
It may be neccessary to disable model digest validation In order to transfer data between diffrent versions of the model.
You can do it by editing JSON file in text editor or by using NoDigestCheck=true:

	dbcopy -m modelOne -dbcopy.To db -dbcopy.SetName MyData  -dbcopy.NoDigestCheck
	dbcopy -m modelOne -dbcopy.To db -dbcopy.SetName MyData  -dbcopy.NoDigestCheck=true
	dbcopy -m modelOne -dbcopy.To db -dbcopy.RunName MyResut -dbcopy.NoDigestCheck

To delete from database entire model, model run results, set of input parameters or modeling task:

	dbcopy -m modelOne -dbcopy.Delete
	dbcopy -m modelOne -dbcopy.Delete -dbcopy.RunId 101
	dbcopy -m modelOne -dbcopy.Delete -dbcopy.RunName "My Model Run"
	dbcopy -m modelOne -dbcopy.Delete -dbcopy.RunDigest d722febf683992aa624ce9844a2e597d
	dbcopy -m modelOne -dbcopy.Delete -dbcopy.FirstRun
	dbcopy -m modelOne -dbcopy.Delete -dbcopy.LastRun
	dbcopy -m modelOne -dbcopy.Delete -dbcopy.SetId 2
	dbcopy -m modelOne -dbcopy.Delete -s Default
	dbcopy -m modelOne -dbcopy.Delete -dbcopy.TaskId 1
	dbcopy -m modelOne -dbcopy.Delete -dbcopy.TaskName taskOne

To rename model run results, input set of parameters or modeling task:

	dbcopy -m modelOne -dbcopy.Rename -dbcopy.RunId 101 -dbcopy.ToRunName New_Run_Name
	dbcopy -m modelOne -dbcopy.Rename -dbcopy.RunName "My Model Run" -dbcopy.ToRunName "New Run Name"
	dbcopy -m modelOne -dbcopy.Rename -dbcopy.RunDigest d722febf683992aa624ce9844a2e597d -dbcopy.ToRunName "New Run Name"
	dbcopy -m modelOne -dbcopy.Rename -dbcopy.FirstRun -dbcopy.ToRunName "New Run Name"
	dbcopy -m modelOne -dbcopy.Rename -dbcopy.LastRun  -dbcopy.ToRunName "New Run Name"
	dbcopy -m modelOne -dbcopy.Rename -s Default -dbcopy.ToSetName "New Name"
	dbcopy -m modelOne -dbcopy.Rename -dbcopy.SetName Default -dbcopy.ToSetName "New Name"
	dbcopy -m modelOne -dbcopy.Rename -dbcopy.SetId 2 -dbcopy.ToSetName "New Name"
	dbcopy -m modelOne -dbcopy.Rename -dbcopy.TaskName taskOne -dbcopy.ToTaskName "New Task Name"
	dbcopy -m modelOne -dbcopy.Rename -dbcopy.TaskId 1 -dbcopy.ToTaskName "New Task Name"

By default float and double values converted into csv text with "%.15g" format.
It is possible to specify other format for float values values:

	dbcopy -m modelOne -dbcopy.DoubleFormat "%.7G"

You can suppress zero values and / or NULL (missing) values in output tables or microdata CSV files:

	dbcopy -m modelOne -dbcopy.To csv -dbcopy.NoZeroCsv
	dbcopy -m modelOne -dbcopy.To csv -dbcopy.NoZeroCsv=true
	dbcopy -m modelOne -dbcopy.To csv -dbcopy.NoNullCsv
	dbcopy -m modelOne -dbcopy.To csv -dbcopy.NoNullCsv=true
	dbcopy -m modelOne -dbcopy.To csv -dbcopy.NoZeroCsv -dbcopy.NoNullCsv

Dbcopy do auto detect input files encoding to convert source text into utf-8.
On Windows you may want to expliciltly specify encoding name:

	dbcopy -m modelOne -dbcopy.To db -dbcopy.CodePage windows-1252

If you want to write utf-8 BOM into output csv file then:

	dbcopy -m modelOne -dbcopy.Utf8BomIntoCsv
	dbcopy -m modelOne -dbcopy.Utf8BomIntoCsv -dbcopy.To csv

By default dbcopy using SQLite database connection:

	dbcopy -m modelOne

is equivalent of:

	dbcopy -m modelOne -dbcopy.FromSqlite modelOne.sqlite
	dbcopy -m modelOne -dbcopy.Database "Database=modelOne.sqlite; Timeout=86400; OpenMode=ReadOnly;"
	dbcopy -m modelOne -dbcopy.Database "Database=modelOne.sqlite; Timeout=86400; OpenMode=ReadOnly;" -dbcopy.DatabaseDriver SQLite

Output database connection settings by default are the same as input database,
which may not be suitable because you don't want to overwrite input database.

To specify output database connection string and driver:

	dbcopy -m modelOne -dbcopy.To db -dbcopy.ToSqlite modelOne.sqlite
	dbcopy -m modelOne -dbcopy.To db -dbcopy.ToDatabase "Database=modelOne.sqlite; Timeout=86400; OpenMode=ReadWrite;"
	dbcopy -m modelOne -dbcopy.To db -dbcopy.ToDatabase "Database=modelOne.sqlite; Timeout=86400; OpenMode=ReadWrite;" -dbcopy.ToDatabaseDriver SQLite

	dbcopy -m modelOne -dbcopy.To db2db -dbcopy.FromSqlite src.sqlite -dbcopy.ToSqlite dst.sqlite

Other supported database drivers are "sqlite3" and "odbc":

	dbcopy -m modelOne -dbcopy.To db -dbcopy.ToDatabaseDriver odbc -dbcopy.ToDatabase "DSN=bigSql"
	dbcopy -m modelOne -dbcopy.To db -dbcopy.ToDatabaseDriver sqlite3 -dbcopy.ToDatabase "file:dst.sqlite?mode=rw"

ODBC dbcopy tested with MySQL (MariaDB), PostgreSQL, Microsoft SQL, Oracle and DB2.

If dbcopy used for massive database copy it may be convenient to control it from shell script by process ID:

	dbcopy -dbcopy.PidSaveTo some/dir/dbcopy.pid.txt

By default dbcopy log messages are in user OS language, user can override message language:

	dbcopy -m modelOne -OpenM.MessageLanguage FR
	dbcopy -m modelOne -OpenM.MessageLanguage fr-CA

Also dbcopy support OpenM++ standard log settings (described in openM++ wiki):

	-OpenM.LogToConsole: if true then log to standard output, default: true
	-v:                  short form of: -OpenM.LogToConsole
	-OpenM.LogToFile:    if true then log to file
	-OpenM.LogFilePath:  path to log file, default = current/dir/exeName.log
	-OpenM.LogUseTs:     if true then use time-stamp in log file name
	-OpenM.LogUsePid:    if true then use pid-stamp in log file name
	-OpenM.LogSql:       if true then log sql statements into log file
*/
package main

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jeandeaual/go-locale"
	_ "github.com/mattn/go-sqlite3"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// dbcopy config keys to get values from ini-file or command line arguments.
const (
	copyToArgKey        = "dbcopy.To"                // copy to: text=db-to-text, db=text-to-db, db2db=db-to-db, csv=db-to-csv, csv-all=db-to-csv-all-in-one
	deleteArgKey        = "dbcopy.Delete"            // delete model or workset or model run or modeling task from database
	renameArgKey        = "dbcopy.Rename"            // rename workset or model run or modeling task
	modelNameArgKey     = "dbcopy.ModelName"         // model name
	modelNameShortKey   = "m"                        // model name (short form)
	modelDigestArgKey   = "dbcopy.ModelDigest"       // model hash digest
	setNameArgKey       = "dbcopy.SetName"           // workset name
	setNameShortKey     = "s"                        // workset name (short form)
	setNewNameArgKey    = "dbcopy.ToSetName"         // new workset name, to rename workset
	setIdArgKey         = "dbcopy.SetId"             // workset id, workset is a set of model input parameters
	runNameArgKey       = "dbcopy.RunName"           // model run name
	runNewNameArgKey    = "dbcopy.ToRunName"         // new run name, to rename run
	runIdArgKey         = "dbcopy.RunId"             // model run id
	runDigestArgKey     = "dbcopy.RunDigest"         // model run hash digest
	runFirstArgKey      = "dbcopy.FirstRun"          // use first model run
	runLastArgKey       = "dbcopy.LastRun"           // use last model run
	taskNameArgKey      = "dbcopy.TaskName"          // modeling task name
	taskNewNameArgKey   = "dbcopy.ToTaskName"        // new task name, to rename task
	taskIdArgKey        = "dbcopy.TaskId"            // modeling task id
	fromSqliteArgKey    = "dbcopy.FromSqlite"        // input db is SQLite file
	dbConnStrArgKey     = "dbcopy.Database"          // db connection string
	dbDriverArgKey      = "dbcopy.DatabaseDriver"    // db driver name, ie: SQLite, odbc, sqlite3
	toSqliteArgKey      = "dbcopy.ToSqlite"          // output db is SQLite file
	toDbConnStrArgKey   = "dbcopy.ToDatabase"        // output db connection string
	toDbDriverArgKey    = "dbcopy.ToDatabaseDriver"  // output db driver name, ie: SQLite, odbc, sqlite3
	inputDirArgKey      = "dbcopy.InputDir"          // input dir to read model .json and .csv files
	outputDirArgKey     = "dbcopy.OutputDir"         // output dir to write model .json and .csv files
	keepOutputDirArgKey = "dbcopy.KeepOutputDir"     // keep output directory if it is already exist
	paramDirArgKey      = "dbcopy.ParamDir"          // path to workset parameters directory
	paramDirShortKey    = "p"                        // path to workset parameters directory (short form)
	zipArgKey           = "dbcopy.Zip"               // create output or use as input model.zip
	intoTsvArgKey       = "dbcopy.IntoTsv"           // if true then create .tsv output files instead of .csv by default
	useIdCsvArgKey      = "dbcopy.IdCsv"             // if true then create csv files with enum id's default: enum code
	useIdNamesArgKey    = "dbcopy.IdOutputNames"     // if true then always use id's in output directory and file names, false never use it
	noDigestCheckArgKey = "dbcopy.NoDigestCheck"     // if true then ignore input model digest, use model name only
	noAccCsvArgKey      = "dbcopy.NoAccumulatorsCsv" // if true then do not create accumulators .csv files
	noMicrodataArgKey   = "dbcopy.NoMicrodata"       // if true then suppress microdata output
	noZeroArgKey        = "dbcopy.NoZeroCsv"         // if true then do not write zero values into output tables or microdata csv
	noNullArgKey        = "dbcopy.NoNullCsv"         // if true then do not write NULL values into output tables or microdata csv
	doubleFormatArgKey  = "dbcopy.DoubleFormat"      // convert to string format for float and double
	encodingArgKey      = "dbcopy.CodePage"          // code page for converting source files, e.g. windows-1252
	useUtf8CsvArgKey    = "dbcopy.Utf8BomIntoCsv"    // if true then write utf-8 BOM into csv file
	msgLangArgKey       = "OpenM.MessageLanguage"    // output messages language, e.g. fr-CA
	pidFileArgKey       = "dbcopy.PidSaveTo"         // file path to save dbcopy processs ID
)

// useIdNames is type to define how to make run and set directory and file names
type useIdNames uint8

const (
	defaultUseIdNames useIdNames = iota // default: use id only to prevent name conflicts
	yesUseIdNames                       // always use run and set id in directory and file names
	noUseIdNames                        // never use run and set id in directory and file names
)

// run options
var theCfg = struct {
	isKeepOutputDir bool   // if true then keep existing output directory
	isNoDigestCheck bool   // if true then ignore input model digest, use model name only to load values from csv
	isTsv           bool   // if true then create .tsv output files instead of .csv by default
	isIdCsv         bool   // if true then create csv files with enum id's default: enum code
	isNoAccCsv      bool   // if true then do not create accumulators .csv files
	isNoMicrodata   bool   // if true then suppress microdata output
	isNoZeroCsv     bool   // if true then do not write zero values into output tables .csv files
	isNoNullCsv     bool   // if true then do not write NULL values into output tables .csv files
	doubleFmt       string // format to convert float or double value to string
	encodingName    string // code page for converting source files, e.g. windows-1252
	isWriteUtf8Bom  bool   // if true then write utf-8 BOM into csv file
}{
	doubleFmt:    "%.15g", // default format to convert float or double values to string
	encodingName: "",      // by default detect utf-8 encoding or use OS-specific default: windows-1252 on Windowds and utf-8 outside
}

func main() {
	defer exitOnPanic() // fatal error handler: log and exit

	err := mainBody(os.Args)
	if err != nil {
		omppLog.LogNoLT(err.Error())
		os.Exit(1)
	}
	omppLog.Log("Done.") // compeleted OK
}

func mainBody(args []string) error {

	// set dbcopy command line argument keys and ini-file keys
	_ = flag.String(copyToArgKey, "text", "copy to: `text`=db-to-text, db=text-to-db, db2db=db-to-db, csv=db-to-csv, csv-all=db-to-csv-all-in-one")
	_ = flag.Bool(deleteArgKey, false, "delete from database: model, set of input parameters, model run or modeling task")
	_ = flag.Bool(renameArgKey, false, "rename set of input parameters, model run or modeling task")
	_ = flag.String(modelNameArgKey, "", "model name")
	_ = flag.String(modelNameShortKey, "", "model name (short of "+modelNameArgKey+")")
	_ = flag.String(modelDigestArgKey, "", "model hash digest")
	_ = flag.String(setNameArgKey, "", "set name (name of model input parameters set), if specified then copy only this set")
	_ = flag.String(setNameShortKey, "", "set name (short of "+setNameArgKey+")")
	_ = flag.String(setNewNameArgKey, "", "rename input set of parameters to that new name")
	_ = flag.Int(setIdArgKey, 0, "set id (id of model input parameters set), if specified then copy only this set")
	_ = flag.String(runNameArgKey, "", "model run name, if specified then copy only this run data")
	_ = flag.String(runNewNameArgKey, "", "rename model run to that new name")
	_ = flag.Int(runIdArgKey, 0, "model run id, if specified then copy only this run data")
	_ = flag.String(runDigestArgKey, "", "model run hash digest, if specified then copy only this run data")
	_ = flag.Bool(runFirstArgKey, false, "if true then select first model run or first model run with specified name ")
	_ = flag.Bool(runLastArgKey, false, "if true then select last model run or last model run with specified name ")
	_ = flag.String(taskNameArgKey, "", "modeling task name, if specified then copy only this modeling task data")
	_ = flag.String(taskNewNameArgKey, "", "rename modeling task to that new name")
	_ = flag.Int(taskIdArgKey, 0, "modeling task id, if specified then copy only this run modeling task data")
	_ = flag.String(fromSqliteArgKey, "", "input database SQLite file path")
	_ = flag.String(dbConnStrArgKey, "", "input database connection string")
	_ = flag.String(dbDriverArgKey, db.SQLiteDbDriver, "input database driver name: SQLite, odbc, sqlite3")
	_ = flag.String(toSqliteArgKey, "", "output database SQLite file path")
	_ = flag.String(toDbConnStrArgKey, "", "output database connection string")
	_ = flag.String(toDbDriverArgKey, db.SQLiteDbDriver, "output database driver name: SQLite, odbc, sqlite3")
	_ = flag.String(inputDirArgKey, "", "input directory to read model .json and .csv files")
	_ = flag.String(outputDirArgKey, "", "output directory for model .json and .csv files")
	_ = flag.Bool(keepOutputDirArgKey, theCfg.isKeepOutputDir, "keep (do not delete) existing output directory")
	_ = flag.String(paramDirArgKey, "", "path to parameters directory (input parameters set directory)")
	_ = flag.String(paramDirShortKey, "", "path to parameters directory (short of "+paramDirArgKey+")")
	_ = flag.Bool(zipArgKey, false, "create output model.zip or use model.zip as input")
	_ = flag.Bool(intoTsvArgKey, theCfg.isTsv, "if true then create .tsv output files instead of .csv by default")
	_ = flag.Bool(useIdNamesArgKey, false, "if true then always use id's in output directory names, false never use. Default for csv: only if name conflict")
	_ = flag.Bool(useIdCsvArgKey, false, "if true then create csv files with enum id's default: enum code")
	_ = flag.Bool(noDigestCheckArgKey, theCfg.isNoDigestCheck, "if true then ignore input model digest, use model name only")
	_ = flag.Bool(noAccCsvArgKey, theCfg.isNoAccCsv, "if true then do not create accumulators .csv files")
	_ = flag.Bool(noMicrodataArgKey, theCfg.isNoMicrodata, "if true then suppress microdata output")
	_ = flag.Bool(noZeroArgKey, theCfg.isNoZeroCsv, "if true then do not write zero values into output tables .csv files")
	_ = flag.Bool(noNullArgKey, theCfg.isNoNullCsv, "if true then do not write NULL values into output tables .csv files")
	_ = flag.String(doubleFormatArgKey, theCfg.doubleFmt, "convert to string format for float and double")
	_ = flag.Bool(useUtf8CsvArgKey, theCfg.isWriteUtf8Bom, "if true then write utf-8 BOM into csv file")
	_ = flag.String(encodingArgKey, theCfg.encodingName, "code page to convert source file into utf-8, e.g.: windows-1252")
	_ = flag.String(msgLangArgKey, "", "output messages language, e.g.: fr-CA")
	_ = flag.String(pidFileArgKey, "", "file path to save dbcopy process ID")

	// pairs of full and short argument names to map short name to full name
	var optFs = []config.FullShort{
		{Full: modelNameArgKey, Short: modelNameShortKey},
		{Full: setNameArgKey, Short: setNameShortKey},
		{Full: paramDirArgKey, Short: paramDirShortKey},
	}

	// parse command line arguments and ini-file
	runOpts, logOpts, err := config.New(encodingArgKey, false, optFs)
	if err != nil {
		return helper.ErrorNew("invalid arguments:", err)
	}

	omppLog.New(logOpts) // adjust log options according to command line arguments or ini-values

	// path to bin directory where dbcopy.exe is located
	selfPath, err := filepath.Abs(args[0])
	if err != nil {
		return helper.ErrorNew("Error: unable to make absolute path to dbcopy:", err)
	}
	binDir := filepath.Dir(selfPath)

	// get default user language
	mLang := runOpts.String(msgLangArgKey)

	if mLang == "" { // get default user language

		mLang, err = locale.GetLocale()
		if err != nil {
			omppLog.Log("Warning: unable to get user default language")
		}
	}

	// load translated strings from dbcopy.message.ini
	if _, _, err = omppLog.LoadMessageIni("dbcopy", binDir, mLang, theCfg.encodingName); err != nil {
		omppLog.Log("Error at loading dbcopy.message.ini or go-common.message.ini:", err)
	}

	// save dbcopy process id into file
	if pidFile := runOpts.String(pidFileArgKey); pidFile != "" {
		pid := os.Getpid()
		if err = os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
			omppLog.Log("Error writing PID to file:", pidFile)
			return err
		}
	}

	// model name or model digest is required
	modelName := runOpts.String(modelNameArgKey)
	modelDigest := runOpts.String(modelDigestArgKey)

	if modelName == "" && modelDigest == "" {
		return helper.ErrorNew("invalid (empty) model name and model digest")
	}
	omppLog.Log("Model", modelName, modelDigest)

	// set run options
	theCfg.isKeepOutputDir = runOpts.Bool(keepOutputDirArgKey)
	theCfg.isNoDigestCheck = runOpts.Bool(noDigestCheckArgKey)
	theCfg.isIdCsv = runOpts.Bool(useIdCsvArgKey)
	theCfg.isNoAccCsv = runOpts.Bool(noAccCsvArgKey)
	theCfg.isNoMicrodata = runOpts.Bool(noMicrodataArgKey)
	theCfg.isNoZeroCsv = runOpts.Bool(noZeroArgKey)
	theCfg.isNoNullCsv = runOpts.Bool(noNullArgKey)
	theCfg.isTsv = runOpts.Bool(intoTsvArgKey)
	theCfg.doubleFmt = runOpts.String(doubleFormatArgKey)
	theCfg.encodingName = runOpts.String(encodingArgKey)
	theCfg.isWriteUtf8Bom = runOpts.Bool(useUtf8CsvArgKey)

	// minimal validation of run options
	//
	copyToArg := strings.ToLower(runOpts.String(copyToArgKey))
	isDel := runOpts.Bool(deleteArgKey)
	isRename := runOpts.Bool(renameArgKey)

	if (isDel || isRename) && runOpts.IsExist(copyToArgKey) {
		return helper.ErrorFmt("dbcopy invalid arguments: %s or %s cannot be used with %s", deleteArgKey, renameArgKey, copyToArgKey)
	}
	// to-database can be used only with "db" or "db2db"
	if copyToArg != "db" && copyToArg != "db2db" &&
		(runOpts.IsExist(toDbConnStrArgKey) || runOpts.IsExist(toDbDriverArgKey) || runOpts.IsExist(toSqliteArgKey)) {
		return helper.ErrorFmt("dbcopy invalid arguments: output database can be specified only if %s =db or =db2db", copyToArgKey)
	}
	// id csv is only for output
	if copyToArg != "text" && copyToArg != "csv" && copyToArg != "csv-all" && runOpts.IsExist(useIdCsvArgKey) {
		return helper.ErrorFmt("dbcopy invalid arguments: %s can be used only if %s=text or =csv or =csv-all", useIdCsvArgKey, copyToArgKey)
	}
	// no zero and no null options can be used only for csv output
	if copyToArg != "csv" && copyToArg != "csv-all" && (runOpts.IsExist(noZeroArgKey) || runOpts.IsExist(noNullArgKey)) {
		return helper.ErrorFmt("dbcopy invalid arguments: %s or %s can be used only if %s =text or =csv or =csv-all", noZeroArgKey, noNullArgKey, copyToArgKey)
	}
	// parameter directory is only for workset copy db-to-text or text-to-db
	if runOpts.IsExist(paramDirArgKey) &&
		(copyToArg != "text" && copyToArg != "db" || !runOpts.IsExist(setNameArgKey) && !runOpts.IsExist(setIdArgKey)) {
		return helper.ErrorFmt("dbcopy invalid arguments: %s can be used only with %s  or %s  and if %s =text or =db", paramDirArgKey, setNameArgKey, setIdArgKey, copyToArgKey)
	}
	// new run name can be used with run name, run id or run digest arguments
	if runOpts.IsExist(runNewNameArgKey) &&
		(!isRename ||
			!runOpts.IsExist(runNameArgKey) && !runOpts.IsExist(runIdArgKey) && !runOpts.IsExist(runDigestArgKey) &&
				!runOpts.IsExist(runFirstArgKey) && !runOpts.IsExist(runLastArgKey)) {
		return helper.ErrorFmt("dbcopy invalid arguments: %s must be used with %s and any of: %s %s %s %s %s",
			runNameArgKey, renameArgKey,
			runNameArgKey, runIdArgKey, runDigestArgKey, runFirstArgKey, runLastArgKey)
	}
	// new set name can be used with set name or set id arguments
	if runOpts.IsExist(setNewNameArgKey) &&
		(isRename ||
			!runOpts.IsExist(setNameArgKey) && !runOpts.IsExist(setIdArgKey)) {
		return helper.ErrorFmt("dbcopy invalid arguments: %s must be used with %s any of: %s %s", setNewNameArgKey, renameArgKey, setNameArgKey, setIdArgKey)
	}
	// new task name can be used with task name or task id arguments
	if runOpts.IsExist(taskNewNameArgKey) &&
		(!isRename ||
			!runOpts.IsExist(taskNameArgKey) && !runOpts.IsExist(taskIdArgKey)) {
		return helper.ErrorFmt("dbcopy invalid arguments: %s must be used with %s and any of: %s %s", taskNewNameArgKey, renameArgKey, taskNameArgKey, taskIdArgKey)
	}

	// do delete model run, workset or entire model
	// if not delete then copy: workset, model run data, modeilng task
	// by default: copy entire model
	//
	switch {

	// do delete
	case isDel:

		switch {
		case runOpts.IsExist(runNameArgKey) || runOpts.IsExist(runIdArgKey) || runOpts.IsExist(runDigestArgKey) ||
			runOpts.IsExist(runFirstArgKey) || runOpts.IsExist(runLastArgKey):
			// delete model run
			err = dbDeleteRun(modelName, modelDigest, runOpts)
		case runOpts.IsExist(setNameArgKey) || runOpts.IsExist(setIdArgKey): // delete workset
			err = dbDeleteWorkset(modelName, modelDigest, runOpts)
		case runOpts.IsExist(taskNameArgKey) || runOpts.IsExist(taskIdArgKey): // delete modeling task
			err = dbDeleteTask(modelName, modelDigest, runOpts)
		default:
			err = dbDeleteModel(modelName, modelDigest, runOpts) // delete entrire model
		}

	// do rename
	case isRename:

		switch {
		case runOpts.IsExist(runNameArgKey) || runOpts.IsExist(runIdArgKey) || runOpts.IsExist(runDigestArgKey) ||
			runOpts.IsExist(runFirstArgKey) || runOpts.IsExist(runLastArgKey):
			// rename model run
			err = dbRenameRun(modelName, modelDigest, runOpts)
		case runOpts.IsExist(setNameArgKey) || runOpts.IsExist(setIdArgKey): // rename workset
			err = dbRenameWorkset(modelName, modelDigest, runOpts)
		case runOpts.IsExist(taskNameArgKey) || runOpts.IsExist(taskIdArgKey): // rename modeling task
			err = dbRenameTask(modelName, modelDigest, runOpts)
		default:
			return helper.ErrorNew("dbcopy invalid argument(s) for", renameArgKey)
		}

	// copy model run
	case !isDel && !isRename &&
		(runOpts.IsExist(runNameArgKey) || runOpts.IsExist(runIdArgKey) || runOpts.IsExist(runDigestArgKey) || runOpts.IsExist(runFirstArgKey) || runOpts.IsExist(runLastArgKey)):

		switch copyToArg {
		case "text":
			err = dbToTextRun(modelName, modelDigest, runOpts)
		case "db":
			err = textToDbRun(modelName, modelDigest, runOpts)
		case "db2db":
			err = dbToDbRun(modelName, modelDigest, runOpts)
		default:
			return helper.ErrorNew("dbcopy invalid argument for", copyToArgKey, ":", copyToArg)
		}

	// copy workset
	case !isDel && !isRename && (runOpts.IsExist(setNameArgKey) || runOpts.IsExist(setIdArgKey)):

		switch copyToArg {
		case "text":
			err = dbToTextWorkset(modelName, modelDigest, runOpts)
		case "db":
			err = textToDbWorkset(modelName, modelDigest, runOpts)
		case "db2db":
			err = dbToDbWorkset(modelName, modelDigest, runOpts)
		default:
			return helper.ErrorNew("dbcopy invalid argument for", copyToArgKey, ":", copyToArg)
		}

	// copy modeling task
	case !isDel && !isRename && (runOpts.IsExist(taskNameArgKey) || runOpts.IsExist(taskIdArgKey)):

		switch copyToArg {
		case "text":
			err = dbToTextTask(modelName, modelDigest, runOpts)
		case "db":
			err = textToDbTask(modelName, modelDigest, runOpts)
		case "db2db":
			err = dbToDbTask(modelName, modelDigest, runOpts)
		default:
			return helper.ErrorNew("dbcopy invalid argument for", copyToArgKey, ":", copyToArg)
		}

	default: // copy entire model

		switch copyToArg {
		case "text":
			err = dbToText(modelName, modelDigest, runOpts)
		case "csv":
			err = dbToCsv(modelName, modelDigest, false, runOpts)
		case "csv-all":
			err = dbToCsv(modelName, modelDigest, true, runOpts)
		case "db":
			err = textToDb(modelName, runOpts)
		case "db2db":
			err = dbToDb(modelName, modelDigest, runOpts)
		default:
			return helper.ErrorNew("dbcopy invalid argument for", copyToArgKey, ":", copyToArg)
		}
	}

	return err // return nil
}

// exitOnPanic log error message and exit with return = 2
func exitOnPanic() {
	r := recover()
	if r == nil {
		return // not in panic
	}
	switch e := r.(type) {
	case error:
		omppLog.LogNoLT(e.Error())
	case string:
		omppLog.LogNoLT(e)
	default:
		omppLog.LogNoLT("FAILED")
	}
	os.Exit(2) // final exit
}
