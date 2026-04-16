// Copyright (c) 2021 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

import (
	"container/list"
	"encoding/csv"
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

func TestCompareOutputTable(t *testing.T) {

	DoTestCompareTable("CompareOutputTable", t)
	DoTestCompareTable("CompareScalarTable", t)
}

func DoTestCompareTable(section string, t *testing.T) {

	// load ini-file and parse test run options
	opts, err := config.FromIni("testdata/test.ompp.db.compare.ini", "")
	if err != nil {
		t.Fatal(err)
	}

	modelName := opts.String(section + ".ModelName")
	modelDigest := opts.String(section + ".ModelDigest")
	modelSqliteDbPath := opts.String(section + ".DbPath")
	tableName := opts.String(section + ".TableName")

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

	for k := 0; k < 100; k++ {

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

		if cLst := opts.String(section + ".Calculate_" + strconv.Itoa(k+1)); cLst != "" {
			appendToCalc(cLst, false, CALCULATED_ID_OFFSET)
		}
		if cLst := opts.String(section + ".CalculateAggr_" + strconv.Itoa(k+1)); cLst != "" {
			appendToCalc(cLst, true, 2*CALCULATED_ID_OFFSET)
		}
		if len(calcLt) <= 0 {
			continue
		}

		var baseRunId int = 0
		if sVal := opts.String(section + ".BaseRunId_" + strconv.Itoa(k+1)); sVal != "" {
			if baseRunId, err = strconv.Atoi(sVal); err != nil {
				t.Fatal(err)
			}
		}
		t.Log("base run:", baseRunId)

		runIds := []int{}
		if sVal := opts.String(section + ".RunIds_" + strconv.Itoa(k+1)); sVal != "" {

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
			t.Fatal("ERROR: empty run list at "+section+".RunIds", k+1)
		}
		t.Log("run id's:", runIds)

		tableLt := &ReadCalculteTableLayout{
			ReadLayout: ReadLayout{
				Name:   tableName,
				FromId: baseRunId,
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
		csvDir := filepath.Join(opts.String(section+".CsvOutDir"), "Test"+section+"-"+helper.MakeTimeStamp(time.Now()))
		err = os.MkdirAll(csvDir, 0750)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Output to:", csvDir)

		err = writeTestToCsvIdFile(csvDir, modelDef, tableName, csvCvt, cLst)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// create or truncate csv file and write cell list, using id's, not codes
func writeTestToCsvIdFile(
	csvDir string,
	modelDef *ModelMeta,
	name string,
	csvCvt CsvConverter,
	cellLst *list.List) error {

	// converter from db cell to csv id row []string
	var cvt func(interface{}, []string) (bool, error)
	var err error

	if csvCvt.IsUseEnumId() {
		cvt, err = csvCvt.ToCsvIdRow()
	} else {
		cvt, err = csvCvt.ToCsvRow()
	}
	if err != nil {
		return err
	}

	// create csv file
	fn, err := csvCvt.CsvFileName()
	if err != nil {
		return err
	}

	flag := os.O_CREATE | os.O_TRUNC | os.O_WRONLY

	f, err := os.OpenFile(filepath.Join(csvDir, fn), flag, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	wr := csv.NewWriter(f)

	// write header line: column names
	cs, err := csvCvt.CsvHeader()
	if err != nil {
		return err
	}
	if err = wr.Write(cs); err != nil {
		return err
	}

	for c := cellLst.Front(); c != nil; c = c.Next() {

		// write cell line: run id, dimension(s) and value
		if _, err := cvt(c.Value, cs); err != nil {
			return err
		}
		if err := wr.Write(cs); err != nil {
			return err
		}
	}

	// flush and return error, if any
	wr.Flush()
	return wr.Error()
}
