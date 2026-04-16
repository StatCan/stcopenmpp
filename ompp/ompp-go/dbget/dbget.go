// Copyright OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

/*
dbget is a command line tool to export OpenM++ model metadata, input parameters and run results.
It is reading from model database and produce CSV, TSV or JSON output.

Most generic format to specify source data is to use connection string and driver name:

	dbget
	  -dbget.Do model-list
	  -dbget.Database "Database=model.sqlite; Timeout=86400; OpenMode=ReadOnly;"
	  -dbget.DatabaseDriver SQLite

Dget can read model data from SQLite, MySQL, PostgreSQL, MS SQL, Oracle and DB2.

By default openM++ is using SQLite database and it is enough to specife path to model.sqlite file:

	dbget -do model-list -db some/dir/model.sqlite
	dbget -do model-list -dbget.Sqlite some/dir/model.sqlite

If SQLite database file name is the same as model name
and located in current directory then it is enough to specify model name only:

	dbget -m modelOne -do run-list

As result of above command dbget will open modelOne.sqlite database file in current directory
and do "run-list" output list of model runs into CSV file.

Most often used options of dbget do have a short form to reduce typing on command line.
For example: -db is a short version of: -dbget.Sqlite option and -do is a short of -dbget.Do.
Longer version of options can be used on command line and ini files.

For example, if there is my.ini file:

	[dbget]
	Do     = model-list             ; dbget action: 'model-iist' = get list of the models
	Sqlite = some/dir/model.sqlite  ; path to model SQLite database file

then commands below are equal:

	dbget -ini           my.ini
	dbget -OpenM.IniFile my.ini
	dbget -do       model-list -db           some/dir/model.sqlite
	dbget -dbget.Do model-list -dbget.Sqlite some/dir/model.sqlite

By default dbget produce .csv output file(s), e.g. commands above will create model-list.csv file.
It is also possible to produce .tsv output and, for some commands, .json output:

	dbget -db modelOne.sqlite -do model-list
	dbget -db modelOne.sqlite -do model-list -csv
	dbget -db modelOne.sqlite -do model-list -tsv
	dbget -db modelOne.sqlite -do model-list -json
	dbget -db modelOne.sqlite -do model-list -dbget.As csv
	dbget -db modelOne.sqlite -do model-list -dbget.As tsv
	dbget -db modelOne.sqlite -do model-list -dbget.As json

By default dbget write results into the file and user can redirect it to console:

	dbget -db modelOne.sqlite -do model-list -dbget.ToConsole
	dbget -db modelOne.sqlite -do model-list -pipe

It is convenient to use -pipe as a short form of: -dbget.ToConsole -OpenM.LogToConsole=false
to produce output suitable for command pipes.

**Important:**
By using -pipe you are suppressing any console error message output and therefore you must check dbget exit code
or enable additonal log output to file by using -OpenM.LogToFile option.

By default dbget produces language specific output based on match of user OS language to model languages.
For example, if user OS language is fr-CA then output will be created from model FR language, if it is exists in the model database.
If there are no laguage matched then output created in default model language.

	dbget -m modelOne -do all-runs

Above -do all-runs option produce output of all modelOne model runs input parameters and output tables data into .csv files.
Dimension labels in those .csv files are language specific, for example it can be Männlich, Weiblich for Deutsche OS version.

User can override default OS language:

	dbget -m modelOne -do all-runs -lang FR
	dbget -m modelOne -do all-runs -lang fr-CA
	dbget -m modelOne -do all-runs -lang isl
	dbget -m modelOne -do all-runs -dbget.Language EN
	dbget -m modelOne -do all-runs -dbget.Language en-CA
	dbget -m modelOne -do all-runs -dbget.Language isl

If isl = Icelandic language not found in model database then closest language will be used, for example: DA,
or, if no match found in database then it is a default model language.

If user do not want language specific labels in the output then -dbget.NoLanguage option can be used.
In that case dimension items will be M, F codes instead of Male, Female lables.

	dbget -m modelOne -do all-runs -dbget.NoLanguage

If user want language neutral output with dimension items id's: 0, 1 instead codes: M, F then -dbget.IdCsv option can be used.
In that case dimension items will be M, F codes instead of Male, Female lables.

	dbget -m modelOne -do all-runs -dbget.IdCsv

**dbget commands (actions)**

	model-list       list of the models in database
	model            model metadata
	run-list         list of model runs
	set-list         list of model input scenarios (a.k.a. "input set" or workset)
	run              model run results: all parameters, output tables and microdata
	all-runs         all model runs, all parameters, output tables and microdata
	set              input scenario parameters
	all-sets         all input scenarios, all parameter values
	parameter        model run parameter values
	parameter-set    input scenario parameter values
	table            output table values (expressions)
	sub-table        output table sub-values (a.k.a. sub-samples or accumulators)
	sub-table-all    output table sub-values, including derived
	micro            microdata values from model run results
	micro-compare    compare or aggregate microdata between model runs
	old-model        model metadata in Modgen compatible form
	old-run          first model run results in Modgen compatible form
	old-parameter    parameter values in Modgen compatible form
	old-table        output table values in Modgen compatible form

Get list of the models from database:

	dbget -db modelOne.sqlite -do model-list

	dbget -db modelOne.sqlite -do model-list -dbget.As csv
	dbget -db modelOne.sqlite -do model-list -dbget.As tsv
	dbget -db modelOne.sqlite -do model-list -dbget.As json

	dbget -db modelOne.sqlite -do model-list -csv  -dbget.ToConsole
	dbget -db modelOne.sqlite -do model-list -tsv  -dbget.ToConsole
	dbget -db modelOne.sqlite -do model-list -json -dbget.ToConsole
	dbget -db modelOne.sqlite -do model-list -tsv  -pipe

	dbget -db modelOne.sqlite -do model-list -dbget.Language EN
	dbget -db modelOne.sqlite -do model-list -lang fr-CA
	dbget -db modelOne.sqlite -do model-list -lang isl

	dbget -db modelOne.sqlite -do model-list -dbget.Notes -lang en-CA
	dbget -db modelOne.sqlite -do model-list -dbget.Notes -lang fr-CA
	dbget -db modelOne.sqlite -do model-list -dbget.Notes -lang isl
	dbget -db modelOne.sqlite -do model-list -dbget.NoLanguage

	dbget -dbget.Sqlite my/dir/modelOne.sqlite -dbget.Do model-list

	dbget
	  -dbget.Do model-list
	  -dbget.Database "Database=model.sqlite; Timeout=86400; OpenMode=ReadOnly;"
	  -dbget.DatabaseDriver SQLite

Get model metadata from database:

	dbget -m modelOne -do model
	dbget -m modelOne -do model -csv
	dbget -m modelOne -do model -tsv
	dbget -m modelOne -do model -json
	dbget -m modelOne -do model -pipe
	dbget -m modelOne -do model -lang en-CA
	dbget -m modelOne -do model -lang fr-CA
	dbget -m modelOne -do model -lang isl
	dbget -m modelOne -do model -lang fr-CA -dbget.Notes
	dbget -m modelOne -do model -dbget.NoLanguage
	dbget -m modelOne -do model -dir my/output/dir
	dbget -m modelOne -do model -f my-model.csv

	dbget -dbget.ModelName modelOne -dbget.Do model -dbget.As csv -dbget.ToConsole -dbget.Language FR

Get list of model runs:

	dbget -m modelOne -do run-list
	dbget -m modelOne -do run-list -csv
	dbget -m modelOne -do run-list -tsv
	dbget -m modelOne -do run-list -json
	dbget -m modelOne -do run-list -lang fr-CA
	dbget -m modelOne -do run-list -dbget.NoLanguage
	dbget -m modelOne -do run-list -dir my/output/dir
	dbget -m modelOne -do run-list -f my-runs.csv
	dbget -m modelOne -do run-list -pipe
	dbget -m modelOne -do run-list -lang fr-CA -dbget.Notes

	dbget -db my/dir/modelOne.sqlite -dbget.ModelName modelOne -dbget.Do run-list

Get all model runs parameters and output table values:

	dbget -m modelOne -do all-runs
	dbget -m modelOne -do all-runs -lang fr-CA
	dbget -m modelOne -do all-runs -dbget.NoLanguage
	dbget -m modelOne -do all-runs -dbget.IdCsv
	dbget -m modelOne -do all-runs -tsv
	dbget -m modelOne -do all-runs -dir my/output/dir
	dbget -m modelOne -do all-runs -pipe
	dbget -m modelOne -do all-runs -dbget.NoZeroCsv
	dbget -m modelOne -do all-runs -dbget.NoNullCsv
	dbget -m modelOne -do all-runs -dbget.NoZeroCsv -dbget.NoNullCsv

	dbget -dbget.ModelName modelOne -dbget.Do all-runs

Get model run parameters and output table values:

	dbget -m modelOne -do run -dbget.FirstRun
	dbget -m modelOne -do run -dbget.LastRun
	dbget -m modelOne -do run -r Default-4
	dbget -m modelOne -do run -r Default-4 -lang fr-CA
	dbget -m modelOne -do run -r Default-4 -dbget.NoLanguage
	dbget -m modelOne -do run -r Default-4 -dbget.IdCsv
	dbget -m modelOne -do run -r Default-4 -tsv
	dbget -m modelOne -do run -r Default-4 -pipe
	dbget -m modelOne -do run -r Default-4 -dbget.NoZeroCsv
	dbget -m modelOne -do run -r Default-4 -dbget.NoNullCsv
	dbget -m modelOne -do run -r Default-4 -dbget.NoZeroCsv -dbget.NoNullCsv

	dbget -dbget.ModelName modelOne -dbget.Do run -dbget.Run Default

Get parameter run values:

	dbget -m modelOne -r Default -parameter ageSex
	dbget -m modelOne -r Default -parameter ageSex -lang fr-CA
	dbget -m modelOne -r Default -parameter ageSex -dbget.NoLanguage
	dbget -m modelOne -r Default -parameter ageSex -dbget.IdCsv
	dbget -m modelOne -r Default -parameter ageSex -tsv
	dbget -m modelOne -r Default -parameter ageSex -pipe

	dbget -m modelOne -dbget.FirstRun -parameter ageSex
	dbget -m modelOne -dbget.LastRun  -parameter ageSex

	dbget -dbget.ModelName modelOne -dbget.Do parameter -dbget.Run Default -dbget.Parameter ageSex

Get output table values:

	dbget -m modelOne -r Default -table ageSexIncome
	dbget -m modelOne -r Default -table ageSexIncome -lang fr-CA
	dbget -m modelOne -r Default -table ageSexIncome -dbget.NoLanguage
	dbget -m modelOne -r Default -table ageSexIncome -dbget.IdCsv
	dbget -m modelOne -r Default -table ageSexIncome -tsv
	dbget -m modelOne -r Default -table ageSexIncome -pipe
	dbget -m modelOne -r Default -table ageSexIncome -dbget.NoZeroCsv
	dbget -m modelOne -r Default -table ageSexIncome -dbget.NoNullCsv

	dbget -m modelOne -dbget.FirstRun -table ageSexIncome
	dbget -m modelOne -dbget.LastRun  -table ageSexIncome

	dbget -dbget.ModelName modelOne -dbget.Do table -dbget.Run Default -dbget.Table ageSexIncome

Get output table sub-values (get accumulators):

	dbget -m modelOne -r Default -sub-table ageSexIncome
	dbget -m modelOne -r Default -sub-table ageSexIncome -lang fr-CA
	dbget -m modelOne -r Default -sub-table ageSexIncome -dbget.NoLanguage
	dbget -m modelOne -r Default -sub-table ageSexIncome -dbget.IdCsv
	dbget -m modelOne -r Default -sub-table ageSexIncome -tsv
	dbget -m modelOne -r Default -sub-table ageSexIncome -pipe
	dbget -m modelOne -r Default -sub-table ageSexIncome -dbget.NoZeroCsv
	dbget -m modelOne -r Default -sub-table ageSexIncome -dbget.NoNullCsv

	dbget -m modelOne -dbget.FirstRun -sub-table ageSexIncome
	dbget -m modelOne -dbget.LastRun  -sub-table ageSexIncome

	dbget -dbget.ModelName modelOne -dbget.Do sub-table -dbget.Run Default -dbget.Table ageSexIncome

Get output table all sub-values, including derived (get all accumulators):

	dbget -m modelOne -r Default -sub-table-all ageSexIncome
	dbget -m modelOne -r Default -sub-table-all ageSexIncome -lang fr-CA
	dbget -m modelOne -r Default -sub-table-all ageSexIncome -dbget.NoLanguage
	dbget -m modelOne -r Default -sub-table-all ageSexIncome -dbget.IdCsv
	dbget -m modelOne -r Default -sub-table-all ageSexIncome -tsv
	dbget -m modelOne -r Default -sub-table-all ageSexIncome -pipe
	dbget -m modelOne -r Default -sub-table-all ageSexIncome -dbget.NoZeroCsv
	dbget -m modelOne -r Default -sub-table-all ageSexIncome -dbget.NoNullCsv

	dbget -m modelOne -dbget.FirstRun -sub-table-all ageSexIncome
	dbget -m modelOne -dbget.LastRun  -sub-table-all ageSexIncome -tsv -pipe
	dbget -m modelOne -dbget.LastRun  -sub-table-all ageSexIncome -tsv -pipe -dbget.NoZeroCsv -dbget.NoNullCsv

	dbget -dbget.ModelName modelOne -dbget.Do sub-table-all -dbget.Run Default -dbget.Table ageSexIncome

Get list of input parameters sets (list of input scenarios, list of worksets):

	dbget -m modelOne -do set-list
	dbget -m modelOne -do set-list -csv
	dbget -m modelOne -do set-list -tsv
	dbget -m modelOne -do set-list -json
	dbget -m modelOne -do set-list -lang fr-CA
	dbget -m modelOne -do set-list -dbget.NoLanguage
	dbget -m modelOne -do set-list -dir my/output/dir
	dbget -m modelOne -do set-list -f my-scenarios.csv
	dbget -m modelOne -do set-list -pipe
	dbget -m modelOne -do set-list -lang fr-CA -dbget.Notes

	dbget -db my/dir/modelOne.sqlite -dbget.ModelName modelOne -dbget.Do set-list

Get all parameters from all input sets (a.k.a. input scenarios or worksets):

	dbget -m modelOne -do all-sets
	dbget -m modelOne -do all-sets -lang fr-CA
	dbget -m modelOne -do all-sets -dbget.NoLanguage
	dbget -m modelOne -do all-sets -dbget.IdCsv
	dbget -m modelOne -do all-sets -tsv
	dbget -m modelOne -do all-sets -pipe

	dbget -dbget.ModelName modelOne -dbget.Do all-sets

Get parameter input set (a.k.a. input scenario or workset) values:

	dbget -m modelOne -s Default -parameter-set ageSex
	dbget -m modelOne -s Default -parameter-set ageSex -lang fr-CA
	dbget -m modelOne -s Default -parameter-set ageSex -dbget.NoLanguage
	dbget -m modelOne -s Default -parameter-set ageSex -dbget.IdCsv
	dbget -m modelOne -s Default -parameter-set ageSex -tsv
	dbget -m modelOne -s Default -parameter-set ageSex -pipe

	dbget -dbget.ModelName modelOne -dbget.Do parameter-set -dbget.Set Default -dbget.Parameter ageSex

Get all parameters from input set (a.k.a. input scenario or workset):

	dbget -m modelOne -s Default -do set
	dbget -m modelOne -s Default -do set -lang fr-CA
	dbget -m modelOne -s Default -do set -dbget.NoLanguage
	dbget -m modelOne -s Default -do set -dbget.IdCsv
	dbget -m modelOne -s Default -do set -tsv
	dbget -m modelOne -s Default -do set -pipe

	dbget -dbget.ModelName modelOne -dbget.Do set -dbget.Set Default

Get entity microdata:

	dbget -m modelOne -r "Microdata in database" -micro Person
	dbget -m modelOne -r "Microdata in database" -micro Person -lang fr-CA
	dbget -m modelOne -r "Microdata in database" -micro Person -dbget.NoLanguage
	dbget -m modelOne -r "Microdata in database" -micro Person -dbget.IdCsv
	dbget -m modelOne -r "Microdata in database" -micro Person -tsv
	dbget -m modelOne -r "Microdata in database" -micro Person -pipe
	dbget -m modelOne -r "Microdata in database" -micro Person -dbget.NoZeroCsv
	dbget -m modelOne -r "Microdata in database" -micro Person -dbget.NoNullCsv

	dbget -dbget.ModelName modelOne -dbget.Do micro -dbget.Run "Microdata in database" -dbget.Entity Person

**Compare or aggregate values for model run output tables**

Compare first and last RiskPaths model runs: calculate differnce of T04_FertilityRatesByAgeGroup.Expr0 values

	dbget -m RiskPaths -do table-compare
	  -dbget.FirstRun
	  -dbget.WithLastRun
	  -dbget.Table     T04_FertilityRatesByAgeGroup
	  -dbget.Calculate Expr0[variant]-Expr0[base]

Or:

	dbget -m RiskPaths -do table-compare
	  -dbget.FirstRun
	  -dbget.WithLastRun
	  -dbget.Table  T04_FertilityRatesByAgeGroup
	  -calc         Expr0[variant]-Expr0[base]

Aggregate sub-values: calculate variance of T04_FertilityRatesByAgeGroup.acc0, using RiskPaths model last run:

	dbget -m RiskPaths -do table-compare
	  -dbget.LastRun
	  -dbget.Table     T04_FertilityRatesByAgeGroup
	  -dbget.Aggregate OM_VAR(acc0)

Or:

	dbget -m RiskPaths -do table-compare
	  -dbget.LastRun
	  -dbget.Table T04_FertilityRatesByAgeGroup
	  -aggr        OM_VAR(acc0)

Compare and aggregate Riskpaths output table T04_FertilityRatesByAgeGroup:
- output Expr0 measure values as-is, without any transformation
- output the differnce between Expr0 variant and base run values (between last and first model runs)
- output standard deviation of acc0 and acc1

	dbget -m RiskPaths -do table-compare
	  -dbget.FirstRun
	  -dbget.WithLastRun
	  -dbget.Table  T04_FertilityRatesByAgeGroup
	  -calc         "Expr0       , Expr0[variant] - Expr0[base]"
	  -aggr         "OM_SD(acc0) , OM_SD(acc1)"

Default output lables for comparison and aggreagtion expessions are generated automatically,
use -dbget.CalcName or -dbget.AggrName to specify desired labels:

	dbget -m RiskPaths -do table-compare
	  -dbget.FirstRun
	  -dbget.WithLastRun
	  -dbget.Table    T04_FertilityRatesByAgeGroup
	  -calc           "Expr0       , Expr0[variant] - Expr0[base]"
	  -dbget.CalcName "Expr0       , Diffrence of Expr0 in last and first run"
	  -aggr           "OM_SD(acc0) , OM_SD(acc1)"
	  -dbget.AggrName "SD of Acc0  , SD of Acc1"

Model run can be specfied by run id or by name, run stamp or run digest:

	dbget -m RiskPaths -do table-compare
	  -dbget.Run        RiskPaths_Default
	  -dbget.WithRunIds 108,209,310
	  -dbget.Table      T04_FertilityRatesByAgeGroup
	  -calc             "Expr0       , Expr0[variant] - Expr0[base]"
	  -aggr             "OM_SD(acc0) , OM_SD(acc1)"

Compare or aggregate microdata run values.

Aggregate: average AgeGroup Income of entity Person in model run with id 219:

	dbget -m modelOne -do micro-compare
	  -dbget.RunId     219
	  -dbget.Entity    Person
	  -dbget.GroupBy   AgeGroup
	  -dbget.Aggregate OM_AVG(Income)

Model run can be specfied by run id or by name, run stamp, run digest:

	dbget -m modelOne -do micro-compare
	  -pipe
	  -tsv
	  -dbget.Run     "Microdata in database"
	  -dbget.Entity  Person
	  -dbget.GroupBy AgeGroup
	  -aggr          OM_AVG(Income)

Compare microdata first and last model run microdata
by calculating for each Person.AgeGroup average of: Income[base] - Income[variant]:

	dbget -m MyModel -do micro-compare
	  -dbget.FirstRun
	  -dbget.WithLastRun
	  -dbget.Entity  Person
	  -dbget.GroupBy AgeGroup
	  -aggr          OM_AVG(Income[base]-Income[variant])

For each Person.AgeGroup calculate:
- average Income model runs with id 219, 221, 222
- average between Income[base] run id 219 and Income[variant] runs id: 221, 222
- ratio of average Income[variant] / Income[base] model run

	dbget -m modelOne -do micro-compare
	  -tsv
	  -dbget.RunId      219
	  -dbget.WithRunIds 221,222
	  -dbget.Entity     Person
	  -dbget.GroupBy    AgeGroup
	  -aggr  "OM_AVG(Income), OM_AVG(Income[base] - Income[variant]), OM_AVG(Income[variant]) / OM_AVG(Income[base])"

Default lables for aggreagtion expessions are generated automatically,
use -dbget.AggrName to specify desired labels:

	dbget -m modelOne
	  -do micro-compare
	  -r "Microdata in database"
	  -dbget.Entity   Person
	  -dbget.GroupBy  AgeGroup,Sex
	  -aggr          "OM_AVG(Income), OM_VAR(Income)"
	  -dbget.AggrName "Average Income, Income Variance"

Backward compatibility (Modgen).

Get model metadata from compatibility (Modgen) views:

	dbget -m modelOne -do old-model
	dbget -m modelOne -do old-model -csv
	dbget -m modelOne -do old-model -tsv
	dbget -m modelOne -do old-model -json
	dbget -m modelOne -do old-model -pipe

	dbget -dbget.ModelName modelOne -dbget.Do old-model -dbget.As csv -dbget.ToConsole -dbget.Language FR

Get model run parameters and output tables values from compatibility (Modgen) views:

	dbget -m modelOne -do old-run
	dbget -m modelOne -do old-run -csv
	dbget -m modelOne -do old-run -tsv
	dbget -m modelOne -do old-run -lang fr-CA
	dbget -m modelOne -do old-run -dbget.NoLanguage
	dbget -m modelOne -do old-run -dbget.IdCsv
	dbget -m modelOne -do old-run -pipe
	dbget -m modelOne -do old-run -dir my/dir
	dbget -m modelOne -do old-run -dbget.NoZeroCsv
	dbget -m modelOne -do old-run -dbget.NoNullCsv

	dbget -dbget.ModelName modelOne -dbget.Do old-run -dbget.As csv -dbget.ToConsole -dbget.Language FR

Get parameter run values from compatibility (Modgen) views:

	dbget -m modelOne -do old-parameter -dbget.Parameter ageSex
	dbget -m modelOne -do old-parameter -dbget.Parameter ageSex -csv
	dbget -m modelOne -do old-parameter -dbget.Parameter ageSex -tsv
	dbget -m modelOne -do old-parameter -dbget.Parameter ageSex -lang fr-CA
	dbget -m modelOne -do old-parameter -dbget.Parameter ageSex -dbget.NoLanguage
	dbget -m modelOne -do old-parameter -dbget.Parameter ageSex -dbget.IdCsv
	dbget -m modelOne -do old-parameter -dbget.Parameter ageSex -pipe

	dbget -dbget.ModelName modelOne -dbget.Do old-parameter -dbget.Parameter ageSex -dbget.As csv -dbget.ToConsole -dbget.Language FR

Get output table values from compatibility (Modgen) views:

	dbget -m modelOne -do old-table -dbget.Table salarySex
	dbget -m modelOne -do old-table -dbget.Table salarySex -csv
	dbget -m modelOne -do old-table -dbget.Table salarySex -tsv
	dbget -m modelOne -do old-table -dbget.Table salarySex -lang fr-CA
	dbget -m modelOne -do old-table -dbget.Table salarySex -dbget.NoLanguage
	dbget -m modelOne -do old-table -dbget.Table salarySex -dbget.IdCsv
	dbget -m modelOne -do old-table -dbget.Table salarySex -pipe
	dbget -m modelOne -do old-table -dbget.Table salarySex -dbget.NoZeroCsv
	dbget -m modelOne -do old-table -dbget.Table salarySex -dbget.NoNullCsv

	dbget -dbget.ModelName modelOne -dbget.Do old-table -dbget.Table ageSexIncome -dbget.As csv -dbget.ToConsole -dbget.Language FR

Also dbget support OpenM++ standard log settings (described in openM++ wiki):

	-OpenM.LogToConsole: if true then log to standard output, default: true
	-v:                  short form of: -OpenM.LogToConsole
	-OpenM.LogToFile:    if true then log to file
	-OpenM.LogFilePath:  path to log file, default = current/dir/exeName.log
	-OpenM.LogUseTs:     if true then use time-stamp in log file name
	-OpenM.LogUsePid:    if true then use pid-stamp in log file name
	-OpenM.LogSql:       if true then log sql statements into log file

If dbget used for massive database operation it may be convenient to control it from shell script by process ID:

	dbget -dbget.PidSaveTo some/dir/dbget.pid.txt
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

// dbget config keys to get values from ini-file or command line arguments.
const (
	cmdArgKey           = "dbget.Do"             // action, what to do, for example: model-list
	cmdShortKey         = "do"                   // action, what to do (short form)
	asArgKey            = "dbget.As"             // output as csv, tsv or json, default: .csv
	csvArgKey           = "csv"                  // short form of: dbget.As csv
	tsvArgKey           = "tsv"                  // short form of: dbget.As tsv
	jsonArgKey          = "json"                 // short form of: dbget.As json
	outputFileArgKey    = "dbget.File"           // output file name, default: action-name.csv, e.g.: model-list.csv
	outputFileShortKey  = "f"                    // output file name (short form)
	outputDirArgKey     = "dbget.Dir"            // output directory to write .csv or .tsv files
	outputDirShortKey   = "dir"                  // output directory (short form)
	keepOutputDirArgKey = "dbget.KeepOutputDir"  // keep output directory if it is already exist
	consoleArgKey       = "dbget.ToConsole"      // if true then use stdout and do not create file(s)
	consoleShortKey     = "pipe"                 // short form of: -dbget.ToConsole -OpenM.LogToConsole=false
	langArgKey          = "dbget.Language"       // prefered output language: fr-CA
	langShortKey        = "lang"                 // prefered output language (short form)
	noLangArgKey        = "dbget.NoLanguage"     // if true then do language-neutral output: enum codes and "C" formats
	idCsvArgKey         = "dbget.IdCsv"          // if true then do language-neutral output: enum Ids and "C" formats
	encodingArgKey      = "dbget.CodePage"       // code page for converting source files, e.g. windows-1252
	useUtf8ArgKey       = "dbget.Utf8Bom"        // if true then write utf-8 BOM into output
	noZeroArgKey        = "dbget.NoZeroCsv"      // if true then do not write zero values into output tables or microdata csv
	noNullArgKey        = "dbget.NoNullCsv"      // if true then do not write NULL values into output tables or microdata csv
	doubleFormatArgKey  = "dbget.DoubleFormat"   // convert to string format for float and double
	noteArgKey          = "dbget.Notes"          // if true then output notes into .md files
	sqliteArgKey        = "dbget.Sqlite"         // input db SQLite path
	sqliteShortKey      = "db"                   // input db SQLite path (short form)
	dbConnStrArgKey     = "dbget.Database"       // db connection string
	dbDriverArgKey      = "dbget.DatabaseDriver" // db driver name, ie: SQLite, odbc, sqlite3
	modelNameArgKey     = "dbget.ModelName"      // model name
	modelNameShortKey   = "m"                    // model name (short form)
	modelDigestArgKey   = "dbget.ModelDigest"    // model hash digest
	runArgKey           = "dbget.Run"            // model run digest, stamp or name
	runShortKey         = "r"                    // model run digest, stamp or name (short form)
	runIdArgKey         = "dbget.RunId"          // model run id
	runFirstArgKey      = "dbget.FirstRun"       // use first model run
	runLastArgKey       = "dbget.LastRun"        // use last model run
	withRunsArgKey      = "dbget.WithRuns"       // with model run digests, stamps or names (variant runs)
	withRunIdsArgKey    = "dbget.WithRunIds"     // with list model run id's (variant runs)
	withRunFirstArgKey  = "dbget.WithFirstRun"   // with first model run (with first run as variant)
	withRunLastArgKey   = "dbget.WithLastRun"    // with last model run (with last run as variant)
	wsArgKey            = "dbget.Set"            // model workset name
	wsShortKey          = "s"                    // model workset name (short form)
	wsIdArgKey          = "dbget.SetId"          // model workset id
	paramArgKey         = "dbget.Parameter"      // parameter name
	paramShortKey       = "parameter"            // short form of: -dbget.Do parameter -dbget.Parameter Name
	paramWsShortKey     = "parameter-set"        // short form of: -dbget.Do parameter-set -dbget.Parameter Name
	tableArgKey         = "dbget.Table"          // output table name
	tableShortKey       = "table"                // short form of: -dbget.Do table -dbget.Table Name
	subTableShortKey    = "sub-table"            // short form of: -dbget.Do sub-table -dbget.Table Name
	subTableAllShortKey = "sub-table-all"        // short form of: -dbget.Do sub-table-all -dbget.Table Name
	entityArgKey        = "dbget.Entity"         // microdata entity name
	groupByArgKey       = "dbget.GroupBy"        // microdata group by attributes
	aggrArgKey          = "dbget.Aggregate"      // outout table or microdata aggregation expression(s)
	aggrShortKey        = "aggr"                 // short form of: -dbget.Aggregate
	calcArgKey          = "dbget.Calculate"      // calculation expression(s) to compare or aggregate
	calcShortKey        = "calc"                 // short form of: -dbget.Calculate
	aggrNameArgKey      = "dbget.AggrName"       // names of aggregation expression(s)
	calcNameArgKey      = "dbget.CalcName"       // names of calculation expression(s)
	microdataShortKey   = "micro"                // short form of: -dbget.Do micro -dbget.Entity Name
	pidFileArgKey       = "dbget.PidSaveTo"      // file path to save dbget processs ID
)

// output format: csv by default, or tsv or json
type outputAs int

const (
	asCsv outputAs = iota
	asTsv
	asJson
)

// run options
var theCfg = struct {
	action          string   // action name (what to do)
	kind            outputAs // output as csv, tsv or json
	fileName        string   // output file name, default depends on action
	dir             string   // output directory
	binDir          string   // path to bin directory where dbget.exe is located
	isKeepOutputDir bool     // if true then keep existing output directory
	isConsole       bool     // if true then write into stdout
	modelName       string   // model name
	modelDigest     string   // model digest
	doubleFmt       string   // format to convert float or double value to string
	userLang        string   // prefered output language: fr-CA
	lang            string   // model language matched to user language
	isNoLang        bool     // if true then do language-neutral output: enum codes and "C" formats
	isIdCsv         bool     // if true then do language-neutral output: enum id's and "C" formats
	encodingName    string   // "code page" to convert source file into utf-8, for example: windows-1252
	isWriteUtf8Bom  bool     // if true then write utf-8 BOM into csv file
	isNote          bool     // if true then output notes into .md files
}{
	kind:           asCsv,   // by default output as as .csv
	encodingName:   "",      // by default detect utf-8 encoding or use OS-specific default: windows-1252 on Windowds and utf-8 outside
	isWriteUtf8Bom: false,   // do not write BOM by default
	doubleFmt:      "%.15g", // default format to convert float or double values to string
}

const logPeriod = 5 // seconds, log periodically if output takes a long time

// main entry point: wrapper to handle errors
func main() {
	defer exitOnPanic() // fatal error handler: log and exit

	err := mainBody(os.Args)
	if err != nil {
		omppLog.LogNoLT(err)
		os.Exit(1)
	}
	omppLog.Log("Done.") // compeleted OK
}

// actual main body
func mainBody(args []string) error {

	isPipe := false
	doParamName := ""
	doParamWsName := ""
	doTableName := ""
	doAccTableName := ""
	doAllAccTableName := ""
	doEntityName := ""
	_ = flag.String(cmdArgKey, "", "action, what to do, for example: model-list")
	_ = flag.String(cmdShortKey, "", "action, what to do (short of "+cmdArgKey+")")
	_ = flag.String(asArgKey, "", "output as .csv, .tsv or .json, default: .csv")
	_ = flag.Bool(csvArgKey, true, "output as .csv (short of "+asArgKey+" csv)")
	_ = flag.Bool(tsvArgKey, false, "output as .tsv (short of "+asArgKey+" tsv)")
	_ = flag.Bool(jsonArgKey, false, "output as .json (short of "+asArgKey+" json)")
	_ = flag.String(outputFileArgKey, theCfg.fileName, "output file name, default depends on action")
	_ = flag.String(outputFileShortKey, theCfg.fileName, "output file name (short of "+outputFileArgKey+")")
	_ = flag.String(outputDirArgKey, theCfg.dir, "output directory for model .csv or .tsv files")
	_ = flag.String(outputDirShortKey, theCfg.dir, "output directory (short of "+outputDirArgKey+")")
	_ = flag.Bool(keepOutputDirArgKey, theCfg.isKeepOutputDir, "keep (do not delete) existing output directory")
	_ = flag.Bool(consoleArgKey, theCfg.isConsole, "if true then write into standard output instead of file(s)")
	flag.BoolVar(&isPipe, consoleShortKey, theCfg.isConsole, "short form of: -"+consoleArgKey+" -"+config.LogToConsoleArgKey+"=false")
	_ = flag.String(langArgKey, theCfg.userLang, "prefered output language")
	_ = flag.String(langShortKey, theCfg.userLang, "prefered output language (short of "+langArgKey+")")
	_ = flag.Bool(noLangArgKey, theCfg.isNoLang, "if true then do language-neutral output: enum codes and 'C' formats")
	_ = flag.Bool(idCsvArgKey, theCfg.isIdCsv, "if true then do language-neutral output: enum id's and 'C' formats")
	_ = flag.String(encodingArgKey, theCfg.encodingName, "code page to convert source file into utf-8, e.g.: windows-1252")
	_ = flag.Bool(useUtf8ArgKey, theCfg.isWriteUtf8Bom, "if true then write utf-8 BOM into output")
	_ = flag.Bool(noteArgKey, theCfg.isNote, "if true then write notes into .md files")
	_ = flag.String(doubleFormatArgKey, theCfg.doubleFmt, "convert to string format for float and double")
	_ = flag.Bool(noZeroArgKey, false, "if true then do not write zero values into output tables .csv files")
	_ = flag.Bool(noNullArgKey, false, "if true then do not write NULL values into output tables .csv files")
	_ = flag.String(sqliteArgKey, "", "input database SQLite file path")
	_ = flag.String(sqliteShortKey, "", "model name (short of "+sqliteArgKey+")")
	_ = flag.String(dbConnStrArgKey, "", "input database connection string")
	_ = flag.String(dbDriverArgKey, db.SQLiteDbDriver, "input database driver name: SQLite, odbc, sqlite3")
	_ = flag.String(modelNameArgKey, "", "model name")
	_ = flag.String(modelNameShortKey, "", "model name (short of "+modelNameArgKey+")")
	_ = flag.String(modelDigestArgKey, "", "model hash digest")
	_ = flag.String(runArgKey, "", "model run digest, run stamp or run name")
	_ = flag.String(runShortKey, "", "model run digest, run stamp or run name (short of "+runArgKey+")")
	_ = flag.Int(runIdArgKey, 0, "model run id")
	_ = flag.Bool(runFirstArgKey, false, "if true then use first model run")
	_ = flag.Bool(runLastArgKey, false, "if true then use last model run")
	_ = flag.String(withRunsArgKey, "", "with model run digests, stamps or names (variant runs)")
	_ = flag.String(withRunIdsArgKey, "", "with list model run id's (variant runs)")
	_ = flag.Bool(withRunFirstArgKey, false, "if true then use first model run (use as variant run)")
	_ = flag.Bool(withRunLastArgKey, false, "if true then use last model run (use as variant run)")
	_ = flag.String(wsArgKey, "", "input scenario (workset) name")
	_ = flag.String(wsShortKey, "", "input scenario (workset) name (short of "+wsArgKey+")")
	_ = flag.Int(wsIdArgKey, 0, "input scenario (workset) id")
	_ = flag.String(paramArgKey, "", "parameter name")
	flag.StringVar(&doParamName, paramShortKey, "", "short form of: -"+cmdArgKey+" parameter -"+paramArgKey+" Name")
	flag.StringVar(&doParamWsName, paramWsShortKey, "", "short form of: -"+cmdArgKey+" parameter-set -"+paramArgKey+" Name")
	_ = flag.String(tableArgKey, "", "output table name")
	flag.StringVar(&doTableName, tableShortKey, "", "short form of: -"+cmdArgKey+" table -"+tableArgKey+" Name")
	flag.StringVar(&doAccTableName, subTableShortKey, "", "short form of: -"+cmdArgKey+" sub-table -"+tableArgKey+" Name")
	flag.StringVar(&doAllAccTableName, subTableAllShortKey, "", "short form of: -"+cmdArgKey+" sub-table-all -"+tableArgKey+" Name")
	flag.StringVar(&doEntityName, microdataShortKey, "", "short form of: -"+cmdArgKey+" micro -"+entityArgKey+" Name")
	_ = flag.String(entityArgKey, "", "microdata entity name")
	_ = flag.String(groupByArgKey, "", "list of microdata group by attributes")
	_ = flag.String(aggrArgKey, "", "aggregation expression(s) to aggregate output table or microdata")
	_ = flag.String(aggrShortKey, "", "aggregation expression(s) (short of "+aggrArgKey+")")
	_ = flag.String(calcArgKey, "", "calculaton expression(s) to compare or caluculate output table measures")
	_ = flag.String(calcShortKey, "", "calculaton expression(s) (short of "+calcArgKey+")")
	_ = flag.String(aggrNameArgKey, "", "name list of aggregation expressions")
	_ = flag.String(calcNameArgKey, "", "name list of calculation expressions")
	_ = flag.String(pidFileArgKey, "", "file path to save dbget process ID")

	// pairs of full and short argument names to map short name to full name
	var optFs = []config.FullShort{
		{Full: cmdArgKey, Short: cmdShortKey},
		{Full: sqliteArgKey, Short: sqliteShortKey},
		{Full: modelNameArgKey, Short: modelNameShortKey},
		{Full: runArgKey, Short: runShortKey},
		{Full: wsArgKey, Short: wsShortKey},
		{Full: outputFileArgKey, Short: outputFileShortKey},
		{Full: outputDirArgKey, Short: outputDirShortKey},
		{Full: consoleArgKey, Short: consoleShortKey},
		{Full: langArgKey, Short: langShortKey},
		{Full: paramArgKey, Short: paramShortKey},
		{Full: paramArgKey, Short: paramWsShortKey},
		{Full: tableArgKey, Short: tableShortKey},
		{Full: tableArgKey, Short: subTableShortKey},
		{Full: tableArgKey, Short: subTableAllShortKey},
		{Full: entityArgKey, Short: microdataShortKey},
		{Full: aggrArgKey, Short: aggrShortKey},
		{Full: calcArgKey, Short: calcShortKey},
	}

	// parse command line arguments and ini-file
	runOpts, logOpts, err := config.New(encodingArgKey, false, optFs)
	if err != nil {
		return helper.ErrorNew("invalid arguments:", err)
	}
	if isPipe {
		logOpts.IsConsole = false // suppress log console output if -pipe required
	}
	omppLog.New(logOpts) // adjust log options according to command line arguments or ini-values

	// path to bin directory where dbget.exe is located
	selfPath, err := filepath.Abs(args[0])
	if err != nil {
		return helper.ErrorNew("Error: unable to make absolute path to dbget:", err)
	}
	theCfg.binDir = filepath.Dir(selfPath)

	// get common run options
	theCfg.action = runOpts.String(cmdArgKey)
	theCfg.fileName = helper.CleanFileName(runOpts.String(outputFileArgKey))
	theCfg.dir = helper.CleanFilePath(runOpts.String(outputDirArgKey))
	theCfg.isKeepOutputDir = runOpts.Bool(keepOutputDirArgKey)
	theCfg.isConsole = runOpts.Bool(consoleArgKey)
	theCfg.userLang = runOpts.String(langArgKey)
	theCfg.isNoLang = runOpts.Bool(noLangArgKey)
	theCfg.isIdCsv = runOpts.Bool(idCsvArgKey)
	theCfg.encodingName = runOpts.String(encodingArgKey)
	theCfg.isWriteUtf8Bom = runOpts.Bool(useUtf8ArgKey)
	theCfg.isNote = runOpts.Bool(noteArgKey)
	theCfg.doubleFmt = runOpts.String(doubleFormatArgKey)

	// get default user language
	if theCfg.userLang == "" {
		if ln, e := locale.GetLocale(); e == nil {
			theCfg.userLang = ln
		} else {
			omppLog.Log("Warning: unable to get user default language")
		}
	}
	// load translated strings from dbget.message.ini
	mLang := theCfg.userLang
	if theCfg.isNoLang {
		mLang = ""
	}
	if _, _, err = omppLog.LoadMessageIni("dbget", theCfg.binDir, mLang, theCfg.encodingName); err != nil {
		omppLog.Log("Error at loading dbget.message.ini or go-common.message.ini:", err)
	}

	// save dbget process id into file
	if pidFile := runOpts.String(pidFileArgKey); pidFile != "" {
		pid := os.Getpid()
		if err = os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
			omppLog.Log("Error writing PID to file:", err)
			return err
		}
	}

	// validate language options: user specified language cannot be combined with NoLanguage or IdCsv option
	if runOpts.String(langArgKey) != "" && (theCfg.isNoLang || theCfg.isIdCsv) {
		return helper.ErrorFmt("invalid arguments: %s cannot be combined with %s or %s", langArgKey, noLangArgKey, idCsvArgKey)
	}

	// get output format: csv, tsv or json
	if f := runOpts.String(asArgKey); f != "" {

		if runOpts.IsExist(csvArgKey) || runOpts.IsExist(tsvArgKey) || runOpts.IsExist(jsonArgKey) {
			return helper.ErrorFmt("invalid arguments: %s or %s or %s", csvArgKey, tsvArgKey, jsonArgKey)
		}
		switch strings.ToLower(f) {
		case "csv":
			theCfg.kind = asCsv
		case "tsv":
			theCfg.kind = asTsv
		case "json":
			theCfg.kind = asJson
		default:
			return helper.ErrorNew("invalid arguments:", asArgKey, f)
		}
	} else {
		if runOpts.IsExist(csvArgKey) && (runOpts.IsExist(tsvArgKey) || runOpts.IsExist(jsonArgKey)) ||
			runOpts.IsExist(tsvArgKey) && (runOpts.IsExist(csvArgKey) || runOpts.IsExist(jsonArgKey)) ||
			runOpts.IsExist(jsonArgKey) && (runOpts.IsExist(csvArgKey) || runOpts.IsExist(tsvArgKey)) {
			return helper.ErrorFmt("invalid arguments: %s or %s or %s", csvArgKey, tsvArgKey, jsonArgKey)
		}
		switch {
		case runOpts.IsExist(csvArgKey) && runOpts.Bool(csvArgKey):
			theCfg.kind = asCsv
		case runOpts.IsExist(tsvArgKey) && runOpts.Bool(tsvArgKey):
			theCfg.kind = asTsv
		case runOpts.IsExist(jsonArgKey) && runOpts.Bool(jsonArgKey):
			theCfg.kind = asJson
		// if there is no dbget.As argument and there is no dbget.csv, dbget.tsv, dbget.json
		// then use output file name extension to detect kind of output
		case !runOpts.IsExist(csvArgKey) && !runOpts.IsExist(tsvArgKey) && !runOpts.IsExist(jsonArgKey):
			// if file name is empty or extension is unknown then result is csv by default
			theCfg.kind = kindByExt(theCfg.fileName)
		default:
			return helper.ErrorFmt("invalid arguments: %s or %s or %s", csvArgKey, tsvArgKey, jsonArgKey)
		}
	}

	// output to json supported only for model metadata
	if theCfg.kind == asJson {
		if theCfg.action != "model-list" &&
			theCfg.action != "model" && theCfg.action != "old-model" &&
			theCfg.action != "run-list" && theCfg.action != "set-list" {
			return helper.ErrorNew("JSON output not allowed for:", theCfg.action)
		}
	}

	// open source database connection and check is it valid
	cs, dn := db.IfEmptyMakeDefaultReadOnly(runOpts.String(modelNameArgKey), runOpts.String(sqliteArgKey), runOpts.String(dbConnStrArgKey), runOpts.String(dbDriverArgKey))

	srcDb, _, err := db.Open(cs, dn, false)
	if err != nil {
		return err
	}
	defer srcDb.Close()

	if err := db.CheckOpenmppSchemaVersion(srcDb); err != nil {
		srcDb.Close()
		return err
	}

	// if it is not a model-list then
	//   find by model name or digest
	//   match model language to user language
	modelId := 0
	if theCfg.action != "model-list" {

		theCfg.modelName = runOpts.String(modelNameArgKey)
		theCfg.modelDigest = runOpts.String(modelDigestArgKey)

		if theCfg.modelName == "" && theCfg.modelDigest == "" {
			return helper.ErrorNew("invalid (empty) model name and model digest")
		}
		omppLog.Log("Model", theCfg.modelName, theCfg.modelDigest)

		// check if model exists in database
		ok := false
		if ok, modelId, err = db.GetModelId(srcDb, theCfg.modelName, theCfg.modelDigest); err != nil {
			return err
		}
		if !ok {
			return helper.ErrorFmt("model %s %s not found", theCfg.modelName, theCfg.modelDigest)
		}
		mdRow, err := db.GetModelRow(srcDb, modelId)
		if err != nil {
			return err
		}
		if mdRow == nil {
			return helper.ErrorNew("model not found by Id:", modelId)
		}

		// match user language to model language, use default model language if there are no match
		if !theCfg.isNoLang && !theCfg.isIdCsv {
			if theCfg.userLang != "" {
				theCfg.lang, err = matchUserLang(srcDb, *mdRow)
				if err != nil {
					return err
				}
				if theCfg.lang == "" {
					omppLog.Log("Warning: unable to match user language:", theCfg.userLang)
				}
			}
			if theCfg.lang != "" {
				omppLog.Log("Using model language:", theCfg.lang)
			} else {
				theCfg.lang = mdRow.DefaultLangCode
				omppLog.Log("Using default model language:", theCfg.lang)
			}
		}
	}

	// remove output directory if required, create output directory if not already exists
	if err := makeOutputDir(theCfg.dir, theCfg.isKeepOutputDir); err != nil {
		return err
	}

	if doParamName != "" {
		if runOpts.IsExist(cmdArgKey) && theCfg.action != "parameter" {
			return helper.ErrorNew("invalid action argument:", theCfg.action)
		}
		theCfg.action = "parameter"
	}
	if doParamWsName != "" {
		if runOpts.IsExist(cmdArgKey) && theCfg.action != "parameter-set" {
			return helper.ErrorNew("invalid action argument:", theCfg.action)
		}
		theCfg.action = "parameter-set"
	}
	if doTableName != "" {
		if runOpts.IsExist(cmdArgKey) && theCfg.action != "table" {
			return helper.ErrorNew("invalid action argument:", theCfg.action)
		}
		theCfg.action = "table"
	}
	if doAccTableName != "" {
		if runOpts.IsExist(cmdArgKey) && theCfg.action != "sub-table" {
			return helper.ErrorNew("invalid action argument:", theCfg.action)
		}
		theCfg.action = "sub-table"
	}
	if doAllAccTableName != "" {
		if runOpts.IsExist(cmdArgKey) && theCfg.action != "sub-table-all" {
			return helper.ErrorNew("invalid action argument:", theCfg.action)
		}
		theCfg.action = "sub-table-all"
	}
	if doEntityName != "" {
		if runOpts.IsExist(cmdArgKey) && theCfg.action != "micro" {
			return helper.ErrorNew("invalid action argument:", theCfg.action)
		}
		theCfg.action = "micro"
	}

	// dispatch the command
	switch theCfg.action {
	case "model-list":
		return modelList(srcDb)
	case "run-list":
		return runList(srcDb, modelId, runOpts)
	case "set-list":
		return setList(srcDb, modelId, runOpts)
	case "model":
		return modelMeta(srcDb, modelId)
	case "run":
		return runValue(srcDb, modelId, runOpts)
	case "all-runs":
		return runAllValue(srcDb, modelId, runOpts)
	case "all-sets":
		return setAllValue(srcDb, modelId, runOpts)
	case "set":
		return setValue(srcDb, modelId, runOpts)
	case "parameter":
		return parameterRunValue(srcDb, modelId, runOpts)
	case "parameter-set":
		return parameterWsValue(srcDb, modelId, runOpts)
	case "table":
		return tableValue(srcDb, modelId, runOpts)
	case "table-compare":
		return tableCompare(srcDb, modelId, runOpts)
	case "sub-table":
		return tableAcc(srcDb, modelId, runOpts)
	case "sub-table-all":
		return tableAllAcc(srcDb, modelId, runOpts)
	case "micro":
		return microdataValue(srcDb, modelId, runOpts)
	case "micro-compare":
		return microdataCompare(srcDb, modelId, runOpts)
	case "old-model":
		return modelOldMeta(srcDb, modelId)
	case "old-run":
		return runOldValue(srcDb, modelId, runOpts)
	case "old-parameter":
		return parameterOldValue(srcDb, modelId, runOpts)
	case "old-table":
		return tableOldValue(srcDb, modelId, runOpts)
	}
	return helper.ErrorNew("invalid action argument:", theCfg.action)
}

// exitOnPanic log error message and exit with return = 2
func exitOnPanic() {
	r := recover()
	if r == nil {
		return // not in panic
	}
	switch e := r.(type) {
	case error:
		omppLog.LogNoLT(e)
	case string:
		omppLog.LogNoLT(e)
	default:
		omppLog.LogNoLT("FAILED")
	}
	os.Exit(2) // final exit
}
