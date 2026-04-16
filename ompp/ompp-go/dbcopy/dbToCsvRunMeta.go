// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
)

// write all model run data into csv files: parameters, output expressions and accumulators
func toRunListCsv(
	dbConn *sql.DB,
	modelDef *db.ModelMeta,
	outDir string,
	fileCreated map[string]bool,
	doUseIdNames useIdNames,
	isAllInOne bool,
) (bool, error) {

	// get all successfully completed model runs
	rl, err := db.GetRunFullTextList(dbConn, modelDef.Model.ModelId, true, "")
	if err != nil {
		return false, err
	}
	if theCfg.isNoMicrodata { // microdata output disabled
		for k := range rl {
			rl[k].EntityGen = []db.EntityGenMeta{}
			rl[k].RunEntity = []db.RunEntityRow{}
		}
	}

	// use of run and set id's in directory names:
	// if explicitly required then always use id's in the names
	// by default: only if name conflict
	isUseIdNames := false
	if doUseIdNames == yesUseIdNames {
		isUseIdNames = true
	}
	if doUseIdNames == defaultUseIdNames {
		for k := range rl {
			for i := range rl {
				if isUseIdNames = i != k && rl[i].Run.Name == rl[k].Run.Name; isUseIdNames {
					break
				}
			}
			if isUseIdNames {
				break
			}
		}
	}

	// read all run parameters, output accumulators and expressions, microdata and dump it into csv files
	for k := range rl {
		err = toRunCsv(
			dbConn, modelDef, &rl[k], outDir, isUseIdNames, isAllInOne, fileCreated)
		if err != nil {
			return isUseIdNames, err
		}
	}

	// write model run rows into csv
	row := make([]string, 12)

	idx := 0
	err = toCsvFile(
		outDir,
		"run_lst.csv",
		[]string{
			"run_id", "model_id", "run_name", "sub_count",
			"sub_started", "sub_completed", "create_dt", "status",
			"update_dt", "run_digest", "value_digest", "run_stamp"},
		func() (bool, []string, error) {
			if 0 <= idx && idx < len(rl) {
				row[0] = strconv.Itoa(rl[idx].Run.RunId)
				row[1] = strconv.Itoa(rl[idx].Run.ModelId)
				row[2] = rl[idx].Run.Name
				row[3] = strconv.Itoa(rl[idx].Run.SubCount)
				row[4] = strconv.Itoa(rl[idx].Run.SubStarted)
				row[5] = strconv.Itoa(rl[idx].Run.SubCompleted)
				row[6] = rl[idx].Run.CreateDateTime
				row[7] = rl[idx].Run.Status
				row[8] = rl[idx].Run.UpdateDateTime
				row[9] = rl[idx].Run.RunDigest
				row[10] = rl[idx].Run.ValueDigest
				row[11] = rl[idx].Run.RunStamp
				idx++
				return false, row, nil
			}
			return true, row, nil // end of model run rows
		})
	if err != nil {
		return isUseIdNames, helper.ErrorNew("failed to write into", "run_lst.csv", err)
	}

	// write model run text rows into csv
	row = make([]string, 4)

	idx = 0
	j := 0
	err = toCsvFile(
		outDir,
		"run_txt.csv",
		[]string{"run_id", "lang_code", "descr", "note"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(rl) { // end of run rows
				return true, row, nil
			}

			// if end of current run texts then find next run with text rows
			if j < 0 || j >= len(rl[idx].Txt) {
				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(rl) { // end of run rows
						return true, row, nil
					}
					if len(rl[idx].Txt) > 0 {
						break
					}
				}
			}

			// make model run text []string row
			row[0] = strconv.Itoa(rl[idx].Txt[j].RunId)
			row[1] = rl[idx].Txt[j].LangCode
			row[2] = rl[idx].Txt[j].Descr

			if rl[idx].Txt[j].Note == "" { // empty "" string is NULL
				row[3] = "NULL"
			} else {
				row[3] = rl[idx].Txt[j].Note
			}
			j++
			return false, row, nil
		})
	if err != nil {
		return isUseIdNames, helper.ErrorNew("failed to write into", "run_txt.csv", err)
	}

	// convert run option map to array of (id,key,value) rows
	var kvArr [][]string
	k := 0
	for j := range rl {
		for key, val := range rl[j].Opts {
			kvArr = append(kvArr, make([]string, 3))
			kvArr[k][0] = strconv.Itoa(rl[j].Run.RunId)
			kvArr[k][1] = key
			kvArr[k][2] = val
			k++
		}
	}

	// write model run option rows into csv
	row = make([]string, 3)

	idx = 0
	err = toCsvFile(
		outDir,
		"run_option.csv",
		[]string{"run_id", "option_key", "option_value"},
		func() (bool, []string, error) {
			if 0 <= idx && idx < len(kvArr) {
				row = kvArr[idx]
				idx++
				return false, row, nil
			}
			return true, row, nil // end of run rows
		})
	if err != nil {
		return isUseIdNames, helper.ErrorNew("failed to write into", "run_option.csv", err)
	}

	// write run parameter rows into csv
	row = make([]string, 3)

	idx = 0
	j = 0
	err = toCsvFile(
		outDir,
		"run_parameter.csv",
		[]string{"run_id", "parameter_hid", "sub_count"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(rl) { // end of model run rows
				return true, row, nil
			}

			// if end of current run parameters then find next run with parameter rows
			if j < 0 || j >= len(rl[idx].Param) {
				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(rl) { // end of run rows
						return true, row, nil
					}
					if len(rl[idx].Param) > 0 {
						break
					}
				}
			}

			// make run parameter []string row
			row[0] = strconv.Itoa(rl[idx].Run.RunId)
			row[1] = strconv.Itoa(rl[idx].Param[j].ParamHid)
			row[2] = strconv.Itoa(rl[idx].Param[j].SubCount)
			j++
			return false, row, nil
		})
	if err != nil {
		return isUseIdNames, helper.ErrorNew("failed to write into", "run_parameter.csv", err)
	}

	// write parameter value notes rows into csv
	row = make([]string, 4)

	idx = 0
	pix := 0
	j = 0
	err = toCsvFile(
		outDir,
		"run_parameter_txt.csv",
		[]string{"run_id", "parameter_hid", "lang_code", "note"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(rl) { // end of model run rows
				return true, row, nil
			}

			// if end of current run parameter text then find next run with parameter text rows
			if pix < 0 || pix >= len(rl[idx].Param) || j < 0 || j >= len(rl[idx].Param[pix].Txt) {

				j = 0
				for {
					if 0 <= pix && pix < len(rl[idx].Param) {
						pix++
					}
					if pix < 0 || pix >= len(rl[idx].Param) {
						idx++
						pix = 0
					}
					if idx < 0 || idx >= len(rl) { // end of model run rows
						return true, row, nil
					}
					if pix >= len(rl[idx].Param) { // end of run parameter text rows for that run
						continue
					}
					if len(rl[idx].Param[pix].Txt) > 0 {
						break
					}
				}
			}

			// make run parameter text []string row
			row[0] = strconv.Itoa(rl[idx].Param[pix].Txt[j].RunId)
			row[1] = strconv.Itoa(rl[idx].Param[pix].Txt[j].ParamHid)
			row[2] = rl[idx].Param[pix].Txt[j].LangCode

			if rl[idx].Param[pix].Txt[j].Note == "" { // empty "" string is NULL
				row[3] = "NULL"
			} else {
				row[3] = rl[idx].Param[pix].Txt[j].Note
			}
			j++
			return false, row, nil
		})
	if err != nil {
		return isUseIdNames, helper.ErrorNew("failed to write into", "run_parameter_txt.csv", err)
	}

	// write run output tables rows into csv
	row = make([]string, 2)

	idx = 0
	j = 0
	err = toCsvFile(
		outDir,
		"run_table.csv",
		[]string{"run_id", "table_hid"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(rl) { // end of model run rows
				return true, row, nil
			}

			// if end of current run output tables then find next run with table rows
			if j < 0 || j >= len(rl[idx].Table) {
				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(rl) { // end of run rows
						return true, row, nil
					}
					if len(rl[idx].Table) > 0 {
						break
					}
				}
			}

			// make run output table []string row
			row[0] = strconv.Itoa(rl[idx].Run.RunId)
			row[1] = strconv.Itoa(rl[idx].Table[j].TableHid)
			j++
			return false, row, nil
		})
	if err != nil {
		return isUseIdNames, helper.ErrorNew("failed to write into", "run_table.csv", err)
	}

	// write run entity generation rows into csv: entity_gen join to model_entity_dic
	row = make([]string, 7)

	idx = 0
	j = 0
	err = toCsvFile(
		outDir,
		"entity_gen.csv",
		[]string{"run_id", "entity_gen_hid", "model_id", "model_entity_id", "entity_hid", "db_entity_table", "gen_digest"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(rl) { // end of model run rows
				return true, row, nil
			}

			// if end of current run entity generations then find next run with entity generation rows
			if j < 0 || j >= len(rl[idx].EntityGen) {
				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(rl) { // end of run rows
						return true, row, nil
					}
					if len(rl[idx].EntityGen) > 0 {
						break
					}
				}
			}

			// make run entity generation []string row
			row[0] = strconv.Itoa(rl[idx].Run.RunId)
			row[1] = strconv.Itoa(rl[idx].EntityGen[j].GenHid)
			row[2] = strconv.Itoa(rl[idx].EntityGen[j].ModelId)
			row[3] = strconv.Itoa(rl[idx].EntityGen[j].EntityId)
			row[4] = strconv.Itoa(rl[idx].EntityGen[j].EntityHid)
			row[5] = rl[idx].EntityGen[j].DbEntityTable
			row[6] = rl[idx].EntityGen[j].GenDigest
			j++
			return false, row, nil
		})
	if err != nil {
		return isUseIdNames, helper.ErrorNew("failed to write into", "entity_gen.csv", err)
	}

	// write run entity generation attributes rows into csv: entity_gen_attr join to entity_gen rows
	row = make([]string, 4)

	idx = 0
	eix := 0
	j = 0
	err = toCsvFile(
		outDir,
		"entity_gen_attr.csv",
		[]string{"run_id", "entity_gen_hid", "attr_id", "entity_hid"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(rl) { // end of model run rows
				return true, row, nil
			}

			// if end of current run entity generation attributes then find next run with entity generation attributes rows
			if eix < 0 || eix >= len(rl[idx].EntityGen) || j < 0 || j >= len(rl[idx].EntityGen[eix].GenAttr) {

				j = 0
				for {
					if 0 <= eix && eix < len(rl[idx].EntityGen) {
						eix++
					}
					if eix < 0 || eix >= len(rl[idx].EntityGen) {
						idx++
						eix = 0
					}
					if idx < 0 || idx >= len(rl) { // end of model run rows
						return true, row, nil
					}
					if eix >= len(rl[idx].EntityGen) { // end of run entity generation attributes rows for that run
						continue
					}
					if len(rl[idx].EntityGen[eix].GenAttr) > 0 {
						break
					}
				}
			}

			// make run entity generation attributes []string row
			row[0] = strconv.Itoa(rl[idx].Run.RunId)
			row[1] = strconv.Itoa(rl[idx].EntityGen[eix].GenAttr[j].GenHid)
			row[2] = strconv.Itoa(rl[idx].EntityGen[eix].GenAttr[j].AttrId)
			row[3] = strconv.Itoa(rl[idx].EntityGen[eix].EntityHid)
			j++
			return false, row, nil
		})
	if err != nil {
		return isUseIdNames, helper.ErrorNew("failed to write into", "entity_gen_attr.csv", err)
	}

	// write run entity rows into csv: run_entity join to entity_gen rows
	row = make([]string, 4)

	idx = 0
	j = 0
	err = toCsvFile(
		outDir,
		"run_entity.csv",
		[]string{"run_id", "entity_gen_hid", "row_count", "value_digest"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(rl) { // end of model run rows
				return true, row, nil
			}

			// if end of current run entity then find next run with run entity rows
			if j < 0 || j >= len(rl[idx].RunEntity) {
				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(rl) { // end of run rows
						return true, row, nil
					}
					if len(rl[idx].EntityGen) > 0 {
						break
					}
				}
			}

			// make run entity []string row
			row[0] = strconv.Itoa(rl[idx].Run.RunId)
			row[1] = strconv.Itoa(rl[idx].RunEntity[j].GenHid)
			row[2] = strconv.Itoa(rl[idx].RunEntity[j].RowCount)
			row[3] = rl[idx].RunEntity[j].ValueDigest
			j++
			return false, row, nil
		})
	if err != nil {
		return isUseIdNames, helper.ErrorNew("failed to write into", "run_entity.csv", err)
	}

	// write run progress rows into csv
	row = make([]string, 7)

	idx = 0
	j = 0
	err = toCsvFile(
		outDir,
		"run_progress.csv",
		[]string{"run_id", "sub_id", "create_dt", "status", "update_dt", "progress_count", "progress_value"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(rl) { // end of model run rows
				return true, row, nil
			}

			// if end of current run progress then find next run with progress rows
			if j < 0 || j >= len(rl[idx].Progress) {
				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(rl) { // end of run rows
						return true, row, nil
					}
					if len(rl[idx].Param) > 0 {
						break
					}
				}
			}

			// make run progress []string row
			row[0] = strconv.Itoa(rl[idx].Run.RunId)
			row[1] = strconv.Itoa(rl[idx].Progress[j].SubId)
			row[2] = rl[idx].Progress[j].CreateDateTime
			row[3] = rl[idx].Progress[j].Status
			row[4] = rl[idx].Progress[j].UpdateDateTime
			row[5] = strconv.Itoa(rl[idx].Progress[j].Count)
			if theCfg.doubleFmt != "" {
				row[6] = fmt.Sprintf(theCfg.doubleFmt, rl[idx].Progress[j].Value)
			} else {
				row[6] = fmt.Sprint(rl[idx].Progress[j].Value)
			}
			j++
			return false, row, nil
		})
	if err != nil {
		return isUseIdNames, helper.ErrorNew("failed to write into", "run_progress.csv", err)
	}

	return isUseIdNames, nil
}
