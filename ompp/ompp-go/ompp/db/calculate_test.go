// Copyright (c) 2021 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
)

func TestTransalteAccAggrToSql(t *testing.T) {

	// load ini-file and parse test run options
	opts, err := config.FromIni("testdata/test.ompp.db.calculate.ini", "")
	if err != nil {
		t.Fatal(err)
	}

	modelName := opts.String("TransalteAccAggrToSql.ModelName")
	modelDigest := opts.String("TransalteAccAggrToSql.ModelDigest")
	modelSqliteDbPath := opts.String("TransalteAccAggrToSql.DbPath")
	tableName := opts.String("TransalteAccAggrToSql.TableName")

	// open source database connection and check is it valid
	cs := MakeSqliteDefaultReadOnly(modelSqliteDbPath)
	t.Log(cs)

	srcDb, _, err := Open(cs, SQLiteDbDriver, false)
	if err != nil {
		t.Fatal(err)
	}
	defer srcDb.Close()

	if err := CheckOpenmppSchemaVersion(srcDb); err != nil {
		t.Fatal(err)
	}

	// get model metadata
	modelDef, err := GetModel(srcDb, modelName, modelDigest)
	if err != nil {
		t.Fatal(err)
	}
	if modelDef == nil {
		t.Errorf("model not found: %s :%s:", modelName, modelDigest)
	}
	t.Log("Model:", modelDef.Model.Name, " ", modelDef.Model.Digest)

	// find output table id by name
	var table *TableMeta
	if k, ok := modelDef.OutTableByName(tableName); ok {
		table = &modelDef.Table[k]
	} else {
		t.Errorf("output table not found: " + tableName)
	}

	validLst := []struct {
		src   string
		valid string
	}{}
	for k := 0; k < 400; k++ {
		s := opts.String("TransalteAccAggrToSql.Src_" + strconv.Itoa(k+1))
		if s == "" {
			continue
		}
		validLst = append(validLst,
			struct {
				src   string
				valid string
			}{
				src:   s,
				valid: opts.String("TransalteAccAggrToSql.Valid_" + strconv.Itoa(k+1)),
			})
	}

	t.Log("Check aggregation SQL")
	for _, v := range validLst {

		t.Log(v.src)

		paramCols := makeParamCols(modelDef.Param)

		cteSql, mainSql, e := transalteAccAggrToSql(table, paramCols, 0, v.src)
		if e != nil {
			t.Fatal(e)
		}

		sql := ""
		if cteSql != "" {
			sql += "WITH " + cteSql + " "
		}
		sql += mainSql

		if err != nil {
			t.Fatal(err)
		}

		if sql != v.valid {
			t.Error("Expected:", v.valid)
			t.Error("****FAIL:", sql)
		} else {
			t.Log("=>", sql)
		}
	}
}

func TestTranslateTableCalcToSql(t *testing.T) {

	// load ini-file and parse test run options
	opts, err := config.FromIni("testdata/test.ompp.db.calculate.ini", "")
	if err != nil {
		t.Fatal(err)
	}

	modelName := opts.String("TranslateTableCalcToSql.ModelName")
	modelDigest := opts.String("TranslateTableCalcToSql.ModelDigest")
	modelSqliteDbPath := opts.String("TranslateTableCalcToSql.DbPath")
	tableName := opts.String("TranslateTableCalcToSql.TableName")

	// open source database connection and check is it valid
	cs := MakeSqliteDefaultReadOnly(modelSqliteDbPath)
	t.Log(cs)

	srcDb, _, err := Open(cs, SQLiteDbDriver, false)
	if err != nil {
		t.Fatal(err)
	}
	defer srcDb.Close()

	if err := CheckOpenmppSchemaVersion(srcDb); err != nil {
		t.Fatal(err)
	}

	// get model metadata
	modelDef, err := GetModel(srcDb, modelName, modelDigest)
	if err != nil {
		t.Fatal(err)
	}
	if modelDef == nil {
		t.Errorf("model not found: %s :%s:", modelName, modelDigest)
	}
	t.Log("Model:", modelDef.Model.Name, " ", modelDef.Model.Digest)

	// find output table id by name
	var table *TableMeta
	if k, ok := modelDef.OutTableByName(tableName); ok {
		table = &modelDef.Table[k]
	} else {
		t.Errorf("output table not found: " + tableName)
	}

	t.Log("Check calculation SQL")
	for k := 0; k < 400; k++ {

		calcLt := []CalculateTableLayout{}

		appendToCalc := func(src string, isAggr bool, idOffset int) {

			ce := helper.ParseCsvLine(src, ',')
			for j := range ce {

				c := strings.TrimSpace(ce[j])
				if c[0] == '"' && c[len(c)-1] == '"' {
					c = c[1 : len(c)-1]
				}

				if c != "" {

					calcLt = append(calcLt, CalculateTableLayout{
						CalculateLayout: CalculateLayout{
							Calculate: c,
							CalcId:    idOffset + j,
						},
						IsAggr: isAggr,
					})
					t.Log("Calculate:", c)
					t.Log(tableName, " Is aggregation:", isAggr)
				}
			}
		}

		if cLst := opts.String("TranslateTableCalcToSql.Calculate_" + strconv.Itoa(k+1)); cLst != "" {
			appendToCalc(cLst, false, CALCULATED_ID_OFFSET)
		}
		if cLst := opts.String("TranslateTableCalcToSql.CalculateAggr_" + strconv.Itoa(k+1)); cLst != "" {
			appendToCalc(cLst, true, 2*CALCULATED_ID_OFFSET)
		}
		if len(calcLt) <= 0 {
			continue
		}

		baseRunId := 0
		if sVal := opts.String("TranslateTableCalcToSql.BaseRunId_" + strconv.Itoa(k+1)); sVal != "" {
			baseRunId, err = strconv.Atoi(sVal)
			if err != nil {
				t.Fatal(err)
			}
		}

		runIds := []int{}
		if sVal := opts.String("TranslateTableCalcToSql.RunIds_" + strconv.Itoa(k+1)); sVal != "" {

			sArr := helper.ParseCsvLine(sVal, ',')
			for j := range sArr {
				if id, err := strconv.Atoi(sArr[j]); err != nil {
					t.Fatal(err)
				} else {
					runIds = append(runIds, id)
				}
			}
		}
		if len(runIds) <= 0 {
			t.Fatal("ERROR: empty run list at TranslateTableCalcToSql.RunIds", k+1)
		}
		t.Log("run id's:", runIds)

		tableLt := &ReadTableLayout{
			ReadLayout: ReadLayout{
				Name: tableName,
			},
		}
		if baseRunId > 0 {
			tableLt.FromId = baseRunId
		}

		sql, e := translateTableCalcToSql(modelDef, table, &tableLt.ReadLayout, calcLt, runIds)
		if e != nil {
			t.Fatal(e)
		}

		// read valid sql and compare
		valid := opts.String("TranslateTableCalcToSql.Valid_" + strconv.Itoa(k+1))

		if sql != valid {
			t.Error("Expected:", valid)
			t.Error("****FAIL:", sql)
		} else {
			t.Log("=>", sql)
		}
	}
}

func TestCalculateOutputTable(t *testing.T) {

	// load ini-file and parse test run options
	opts, err := config.FromIni("testdata/test.ompp.db.calculate.ini", "")
	if err != nil {
		t.Fatal(err)
	}

	modelName := opts.String("CalculateOutputTable.ModelName")
	modelDigest := opts.String("CalculateOutputTable.ModelDigest")
	modelSqliteDbPath := opts.String("CalculateOutputTable.DbPath")
	tableName := opts.String("CalculateOutputTable.TableName")

	// open source database connection and check is it valid
	cs := MakeSqliteDefaultReadOnly(modelSqliteDbPath)
	t.Log(cs)

	srcDb, _, err := Open(cs, SQLiteDbDriver, false)
	if err != nil {
		t.Fatal(err)
	}
	defer srcDb.Close()

	if err := CheckOpenmppSchemaVersion(srcDb); err != nil {
		t.Fatal(err)
	}

	// get model metadata
	modelDef, err := GetModel(srcDb, modelName, modelDigest)
	if err != nil {
		t.Fatal(err)
	}
	if modelDef == nil {
		t.Errorf("model not found: %s :%s:", modelName, modelDigest)
	}
	t.Log("Model:", modelDef.Model.Name, " ", modelDef.Model.Digest)

	// create csv converter by including all model runs (test only)
	rLst, err := GetRunList(srcDb, modelDef.Model.ModelId)
	if err != nil {
		t.Fatal(err)
	}

	csvCvt := &CellTableCalcConverter{
		CellTableConverter: CellTableConverter{
			ModelDef: modelDef,
			Name:     tableName,
			IsIdCsv:  true,
		},
		CalcMaps: EmptyCalcMaps(),
	}
	for _, r := range rLst {
		csvCvt.RunIdToLabel[r.RunId] = r.RunDigest
	}

	for k := 0; k < 400; k++ {

		calcLt := []CalculateTableLayout{}

		appendToCalc := func(src string, isAggr bool, idOffset int) {

			ce := helper.ParseCsvLine(src, ',')
			for j := range ce {

				c := strings.TrimSpace(ce[j])
				if c[0] == '"' && c[len(c)-1] == '"' {
					c = c[1 : len(c)-1]
				}

				if c != "" {

					calcLt = append(calcLt, CalculateTableLayout{
						CalculateLayout: CalculateLayout{
							Calculate: c,
							CalcId:    idOffset + j,
						},
						IsAggr: isAggr,
					})
					t.Log(calcLt[len(calcLt)-1].CalcId, "Calculate:", c)
					t.Log(tableName, " Is aggregation:", isAggr)
				}
			}
		}

		if cLst := opts.String("CalculateOutputTable.Calculate_" + strconv.Itoa(k+1)); cLst != "" {
			appendToCalc(cLst, false, CALCULATED_ID_OFFSET)
		}
		if cLst := opts.String("CalculateOutputTable.CalculateAggr_" + strconv.Itoa(k+1)); cLst != "" {
			appendToCalc(cLst, true, 2*CALCULATED_ID_OFFSET)
		}
		if len(calcLt) <= 0 {
			continue
		}

		runIds := []int{}
		if sVal := opts.String("CalculateOutputTable.RunIds_" + strconv.Itoa(k+1)); sVal != "" {

			sArr := helper.ParseCsvLine(sVal, ',')
			for j := range sArr {
				if id, err := strconv.Atoi(sArr[j]); err != nil {
					t.Fatal(err)
				} else {
					runIds = append(runIds, id)
				}
			}
		}
		if len(runIds) <= 0 {
			t.Fatal("ERROR: empty run list at CalculateOutputTable.RunIds", k+1)
		}
		t.Log("run id's:", runIds)

		tableLt := &ReadCalculteTableLayout{
			ReadLayout: ReadLayout{
				Name:   tableName,
				FromId: runIds[0],
			},
			Calculation: calcLt,
		}

		// read table
		cLst, rdLt, err := CalculateOutputTable(srcDb, modelDef, tableLt, runIds)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Row count:", cLst.Len())
		t.Log("Read layout Offset Size IsFullPage IsLastPage:", rdLt.Offset, rdLt.Size, rdLt.IsFullPage, rdLt.IsLastPage)

		// create new output directory and csv file
		csvDir := filepath.Join(opts.String("CalculateOutputTable.CsvOutDir"), "TestCalculateOutputTable-"+helper.MakeTimeStamp(time.Now()))
		err = os.MkdirAll(csvDir, 0750)
		if err != nil {
			t.Fatal(err)
		}

		err = writeTestToCsvIdFile(csvDir, modelDef, tableName, csvCvt, cLst)
		if err != nil {
			t.Fatal(err)
		}

		// read valid csv input and compare
		// valid := opts.String("CalculateOutputTable.Valid_"+strconv.Itoa(k+1)]
	}
}
