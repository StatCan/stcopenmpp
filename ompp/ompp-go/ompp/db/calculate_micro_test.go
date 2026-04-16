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

func TestTranslateMicroCalcToSql(t *testing.T) {

	// load ini-file and parse test run options
	opts, err := config.FromIni("testdata/test.ompp.db.micro-aggregate.ini", "")
	if err != nil {
		t.Fatal(err)
	}

	modelName := opts.String("TranslateMicroCalcToSql.ModelName")
	modelDigest := opts.String("TranslateMicroCalcToSql.ModelDigest")
	modelSqliteDbPath := opts.String("TranslateMicroCalcToSql.DbPath")
	entityName := opts.String("TranslateMicroCalcToSql.EntityName")

	baseRunId := 0
	if sVal := opts.String("TranslateMicroCalcToSql.BaseRunId"); sVal != "" {
		baseRunId, err = strconv.Atoi(sVal)
		if err != nil {
			t.Fatal(err)
		}
	}

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

	// find entity generation by entity name and run id
	entity := &EntityMeta{}
	entityGen := &EntityGenMeta{}

	// find model entity by entity name
	eIdx, ok := modelDef.EntityByName(entityName)
	if !ok {
		t.Fatal("entity not found:", entityName)
	}
	entity = &modelDef.Entity[eIdx]

	// get list of entity generations for that model run
	egLst, err := GetEntityGenList(srcDb, baseRunId)
	if err != nil {
		t.Fatal("Error at get run entities: ", entityName, ": ", baseRunId, ": ", err.Error())
	}

	// find entity generation by entity name
	gIdx := -1
	for k := range egLst {

		if egLst[k].EntityId == entity.EntityId {
			gIdx = k
			break
		}
	}
	if gIdx < 0 {
		t.Fatal("Error: model run entity generation not found: ", entityName, ": ", baseRunId)
	}

	entityGen = &egLst[gIdx]

	t.Log("Check microdata aggregation SQL")

	for k := 0; k < 400; k++ {
		srcCalc := opts.String("TranslateMicroCalcToSql.Src_" + strconv.Itoa(k+1))
		if srcCalc == "" {
			continue
		}
		t.Log(srcCalc)

		cteValid := opts.String("TranslateMicroCalcToSql.Cte_" + strconv.Itoa(k+1))
		mainValid := opts.String("TranslateMicroCalcToSql.Main_" + strconv.Itoa(k+1))

		sGroupBy := opts.String("TranslateMicroCalcToSql.GroupBy_" + strconv.Itoa(k+1))
		groupBy := helper.ParseCsvLine(sGroupBy, ',')

		t.Log("Group by: ", groupBy)

		runIds := []int{}
		if sVal := opts.String("TranslateMicroCalcToSql.RunIds_" + strconv.Itoa(k+1)); sVal != "" {

			sArr := helper.ParseCsvLine(sVal, ',')
			for j := range sArr {
				if id, err := strconv.Atoi(sArr[j]); err != nil {
					t.Fatal(err)
				} else {
					runIds = append(runIds, id)
				}
			}
		}
		t.Log("run id's: ", runIds)

		// create list of microdata columns
		aggrCols, e := makeMicroAggrCols(entity, entityGen, groupBy)
		if e != nil {
			t.Fatal("Fail to makeMicroAggrCols:", entityName, ":", groupBy)
		}

		// parameter columns
		paramCols := makeParamCols(modelDef.Param)

		// Translate microdata aggregation into main sql query.
		mainSql, isCompare, e := translateMicroCalcToSql(entity, entityGen, aggrCols, paramCols, 2*CALCULATED_ID_OFFSET, srcCalc)
		if e != nil {
			t.Fatal(e)
		}
		t.Log("isCompare:", isCompare)

		// Build CTE part of aggregation sql from the list of aggregated attributes.
		cteSql, e := makeMicroCteAggrSql(entity, entityGen, aggrCols, baseRunId, runIds)
		if e != nil {
			t.Fatal(e)
		}
		pCteSql, e := makeParamCteSql(paramCols, baseRunId, runIds)
		if e != nil {
			t.Fatal(e)
		}
		if pCteSql != "" {
			cteSql += ", " + pCteSql
		}

		cteSql = "WITH " + cteSql

		if cteSql != cteValid {
			t.Error("Expected:", cteValid)
			t.Error("****FAIL:", cteSql)
		}
		if mainSql != mainValid {
			t.Error("Expected:", mainValid)
			t.Error("****FAIL:", mainSql)
		}

		if mainSql == mainValid && cteSql == cteValid {
			t.Log("=>", cteSql)
			t.Log("=>", mainSql)
		}
	}
}

func TestCalculateMicrodata(t *testing.T) {

	// load ini-file and parse test run options
	opts, err := config.FromIni("testdata/test.ompp.db.micro-aggregate.ini", "")
	if err != nil {
		t.Fatal(err)
	}

	modelName := opts.String("CalculateMicrodata.ModelName")
	modelDigest := opts.String("CalculateMicrodata.ModelDigest")
	modelSqliteDbPath := opts.String("CalculateMicrodata.DbPath")
	entityName := opts.String("CalculateMicrodata.EntityName")

	isIdCsv, err := strconv.ParseBool(opts.String("CalculateMicrodata.IdCsv"))
	if err != nil {
		t.Fatal(err)
	}

	baseRunId := 0
	if sVal := opts.String("CalculateMicrodata.BaseRunId"); sVal != "" {
		baseRunId, err = strconv.Atoi(sVal)
		if err != nil {
			t.Fatal(err)
		}
	}

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

	// find entity generation by entity name and run id
	entity := &EntityMeta{}
	entityGen := &EntityGenMeta{}

	// find model entity by entity name
	eIdx, ok := modelDef.EntityByName(entityName)
	if !ok {
		t.Fatal("entity not found:", entityName)
	}
	entity = &modelDef.Entity[eIdx]

	// get list of entity generations for that model run
	egLst, err := GetEntityGenList(srcDb, baseRunId)
	if err != nil {
		t.Fatal("Error at get run entities: ", entityName, ": ", baseRunId, ": ", err.Error())
	}

	// find entity generation by entity name
	gIdx := -1
	for k := range egLst {

		if egLst[k].EntityId == entity.EntityId {
			gIdx = k
			break
		}
	}
	if gIdx < 0 {
		t.Fatal("Error: model run entity generation not found: ", entityName, ": ", baseRunId)
	}

	entityGen = &egLst[gIdx]

	// test only: include all model runs all model runs
	rLst, err := GetRunList(srcDb, modelDef.Model.ModelId)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Check microdata aggregation SQL")

	for k := 0; k < 400; k++ {
		srcCalc := opts.String("CalculateMicrodata.Calculate_" + strconv.Itoa(k+1))
		if srcCalc == "" {
			continue
		}
		t.Log(srcCalc)

		sGroupBy := opts.String("CalculateMicrodata.GroupBy_" + strconv.Itoa(k+1))
		groupBy := helper.ParseCsvLine(sGroupBy, ',')

		t.Log("Group by: ", groupBy)

		runIds := []int{}
		if sVal := opts.String("CalculateMicrodata.RunIds_" + strconv.Itoa(k+1)); sVal != "" {

			sArr := helper.ParseCsvLine(sVal, ',')
			for j := range sArr {
				if id, err := strconv.Atoi(sArr[j]); err != nil {
					t.Fatal(err)
				} else {
					runIds = append(runIds, id)
				}
			}
		}
		t.Log("run id's: ", runIds)

		// create csv converter
		csvCvt := &CellMicroCalcConverter{
			CellEntityConverter: CellEntityConverter{
				ModelDef:  modelDef,
				Name:      entityName,
				EntityGen: entityGen,
				IsIdCsv:   isIdCsv,
			},
			GroupBy:  groupBy,
			CalcMaps: EmptyCalcMaps(),
		}
		for _, r := range rLst {
			if r.RunId == baseRunId {
				csvCvt.RunIdToLabel[r.RunId] = r.RunDigest
				break
			}
		}
		for _, rId := range runIds {
			for _, r := range rLst {
				if r.RunId == rId {
					csvCvt.RunIdToLabel[r.RunId] = r.RunDigest
					break
				}
			}
		}

		calcLt := []CalculateLayout{}

		appendToCalc := func(src string, idOffset int) {

			ce := helper.ParseCsvLine(src, ',')
			for j := range ce {

				c := strings.TrimSpace(ce[j])
				if c[0] == '"' && c[len(c)-1] == '"' {
					c = c[1 : len(c)-1]
				}

				if c != "" {
					cId := idOffset + j
					cName := "micro_" + strconv.Itoa(j)

					calcLt = append(calcLt, CalculateLayout{
						Calculate: c,
						CalcId:    cId,
						Name:      cName,
					})
					t.Log(calcLt[len(calcLt)-1].CalcId, "Calculate:", c)

					csvCvt.CalcIdToName[cId] = cName
				}
			}
		}

		if cLst := opts.String("CalculateMicrodata.Calculate_" + strconv.Itoa(k+1)); cLst != "" {
			appendToCalc(cLst, CALCULATED_ID_OFFSET)
		}
		if len(calcLt) <= 0 {
			continue // skip empty calculation
		}

		microLt := &ReadCalculteMicroLayout{
			ReadLayout: ReadLayout{
				Name:   entityName,
				FromId: baseRunId,
			},
			CalculateMicroLayout: CalculateMicroLayout{
				Calculation: calcLt,
				GroupBy:     groupBy,
			},
		}

		// aggregate microdata
		cLst, rdLt, err := CalculateMicrodata(srcDb, modelDef, microLt, runIds)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Row count:", cLst.Len())
		t.Log("Read layout Offset Size IsFullPage IsLastPage:", rdLt.Offset, rdLt.Size, rdLt.IsFullPage, rdLt.IsLastPage)

		// create new output directory and csv file
		csvDir := filepath.Join(opts.String("CalculateMicrodata.CsvOutDir"), "TestCalculateMicrodata-"+helper.MakeTimeStamp(time.Now()))
		err = os.MkdirAll(csvDir, 0750)
		if err != nil {
			t.Fatal(err)
		}

		err = writeTestToCsvIdFile(csvDir, modelDef, entityName, csvCvt, cLst)
		if err != nil {
			t.Fatal(err)
		}

		// read valid csv input and compare
		// valid := opts.String("CalculateMicrodata.Valid_"+strconv.Itoa(k+1)]
	}
}
