// Copyright OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// write model run parameters, output tables and microdata into csv or tsv files
func runValue(srcDb *sql.DB, modelId int, runOpts *config.RunOptions) error {

	// find model run
	msg, run, err := findRun(srcDb, modelId, runOpts.String(runArgKey), runOpts.Int(runIdArgKey, 0), runOpts.Bool(runFirstArgKey), runOpts.Bool(runLastArgKey))
	if err != nil {
		return helper.ErrorNew("Error at get model run:", msg, err)
	}
	if run == nil {
		return helper.ErrorNew("Error: model run not found")
	}
	if run.Status != db.DoneRunStatus {
		return helper.ErrorNew("Error: model run not completed successfully:", run.Name)
	}
	runMeta, err := db.GetRunFull(srcDb, run)
	if err != nil {
		return helper.ErrorNew("Error at get model run:", run.Name, err)
	}

	// get model metadata
	meta, err := db.GetModelById(srcDb, modelId)
	if err != nil {
		return helper.ErrorNew("Error at get model metadata by id:", modelId, ":", err)
	}

	// create output directory
	// if output directory name not explicitly specified then use run.RunName by default
	runTop := theCfg.dir
	isDefaultTop := theCfg.dir == ""

	if theCfg.isConsole {
		omppLog.Log("Do", theCfg.action, runMeta.Run.Name)
	} else {

		if runTop == "" {
			switch {
			case runOpts.Bool(runFirstArgKey):
				runTop = helper.CleanFileName(meta.Model.Name) + ".first-run"
			case runOpts.Bool(runLastArgKey):
				runTop = helper.CleanFileName(meta.Model.Name) + ".last-run"
			default:
				runTop = "run." + helper.CleanFileName(runMeta.Run.Name)
			}
			if err = makeOutputDir(runTop, theCfg.isKeepOutputDir); err != nil {
				return err
			}
		}
		omppLog.Log("Do", theCfg.action, ":", runTop)
	}

	return runValueOut(srcDb, meta, runMeta, runTop, isDefaultTop, runOpts)
}

// write model run parameters, output tables and microdata into csv or tsv files
func runValueOut(srcDb *sql.DB, meta *db.ModelMeta, runMeta *db.RunMeta, runTop string, isDefaultTop bool, runOpts *config.RunOptions) error {

	// create sub directories for parameters, output tables and microdata
	paramCsvDir := ""
	tableCsvDir := ""
	microCsvDir := ""
	nMd := len(runMeta.EntityGen)

	if !theCfg.isConsole {

		dirSuffix := "" // if output directory not specified then add .no-zero and .no-null suffix
		if isDefaultTop {
			if runOpts.Bool(noZeroArgKey) {
				dirSuffix = dirSuffix + ".no-zero"
			}
			if runOpts.Bool(noNullArgKey) {
				dirSuffix = dirSuffix + ".no-null"
			}
		}
		paramCsvDir = filepath.Join(runTop, "parameters")
		tableCsvDir = filepath.Join(runTop, "output-tables"+dirSuffix)
		microCsvDir = filepath.Join(runTop, "microdata"+dirSuffix)

		if e := makeOutputDir(paramCsvDir, theCfg.isKeepOutputDir); e != nil {
			return e
		}
		if e := makeOutputDir(tableCsvDir, theCfg.isKeepOutputDir); e != nil {
			return e
		}
		if nMd > 0 {
			if e := makeOutputDir(microCsvDir, theCfg.isKeepOutputDir); e != nil {
				return e
			}
		}
	}

	// write all parameters into csv file
	nP := len(meta.Param)
	omppLog.Log("  Parameters:", nP)
	logT := time.Now().Unix()

	for j := 0; j < nP; j++ {

		logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s", j, nP, meta.Param[j].Name))

		fp := ""
		if !theCfg.isConsole {
			fp = filepath.Join(paramCsvDir, meta.Param[j].Name+extByKind())
		}
		e := parameterValue(srcDb, meta, meta.Param[j].Name, runMeta.Run.RunId, false, fp, false, nil)
		if e != nil {
			return e
		}
	}

	// write output tables into csv file, if the table included in run results
	nT := len(runMeta.Table)
	omppLog.Log("  Tables:", nT)

	for j := 0; j < nT; j++ {

		// check if table exist in model run results
		name := ""
		for k := range meta.Table {
			if meta.Table[k].TableHid == runMeta.Table[j].TableHid {
				name = meta.Table[k].Name
				break
			}
		}
		if name == "" {
			continue // skip table: it is suppressed and not in run results
		}
		logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s", j, nT, name))

		fp := ""
		if !theCfg.isConsole {
			fp = filepath.Join(tableCsvDir, name+extByKind())
		}
		e := tableRunValue(srcDb, meta, name, runMeta.Run.RunId, runOpts, fp, false, nil)
		if e != nil {
			return e
		}
	}

	// write microdata into csv file, if there is any microdata for that model run
	if nMd > 0 {
		omppLog.Log("  Microdata:", nMd)

		for j := 0; j < nMd; j++ {

			// check if microdata exist in model run results
			eId := runMeta.EntityGen[j].EntityId
			eIdx, isFound := meta.EntityByKey(eId)
			if !isFound {
				return helper.ErrorNew("error: entity not found by Id:", eId, runMeta.EntityGen[j].GenDigest)
			}
			logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s", j, nMd, meta.Entity[eIdx].Name))

			fp := ""
			if !theCfg.isConsole {
				fp = filepath.Join(microCsvDir, meta.Entity[eIdx].Name+extByKind())
			}

			e := microdataRunValue(srcDb, meta, meta.Entity[eIdx].Name, &runMeta.Run, runOpts, fp)
			if e != nil {
				return e
			}

		}
	}
	return nil
}

// write all model runs parameters and output tables into csv or tsv files
func runAllValue(srcDb *sql.DB, modelId int, runOpts *config.RunOptions) error {

	// get model metadata and run list
	// run list includes all runs, use only sucessfully completed
	meta, err := db.GetModelById(srcDb, modelId)
	if err != nil {
		return helper.ErrorNew("Error at get model metadata by id:", modelId, ":", err)
	}
	rl, err := db.GetRunList(srcDb, modelId)
	if err != nil {
		return helper.ErrorNew("Error at get model runs list:", err)
	}
	rl = slices.DeleteFunc(rl, func(r db.RunRow) bool { return r.Status != db.DoneRunStatus })

	if len(rl) <= 0 {
		omppLog.LogFmt("Do %s: there are no completed model runs", theCfg.action)
		return nil
	}

	// check if any run name is not unique then use run id's in directory names
	isUseIdNames := false
	for k := range rl {
		for i := range rl {
			if isUseIdNames = i != k && rl[i].Name == rl[k].Name; isUseIdNames {
				break
			}
		}
		if isUseIdNames {
			break
		}
	}

	// create output directory
	// if output directory name not explicitly specified then use ModelName by default
	csvTop := theCfg.dir
	isDefaultTop := !isUseIdNames && theCfg.dir == ""

	if theCfg.isConsole {
		omppLog.Log("Do", theCfg.action, meta.Model.Name)
	} else {
		if csvTop == "" {
			csvTop = filepath.Join(helper.CleanFileName(meta.Model.Name))
			if err = makeOutputDir(csvTop, theCfg.isKeepOutputDir); err != nil {
				return err
			}
		}
		omppLog.Log("Do", theCfg.action, ":", csvTop)
	}

	// for each run write parameters, output tables and microdata into csv or tsv files
	for _, rm := range rl {

		runMeta, err := db.GetRunFull(srcDb, &rm)
		if err != nil {
			return helper.ErrorNew("Error at get model run:", rm.Name, err)
		}
		if runMeta.Run.Status != db.DoneRunStatus {
			continue // unexpected change of model run status
		}
		omppLog.Log("Model run", rm.RunId, rm.Name)

		// run output directory is: run.Name_Of_the_Run or run.ID.Name_Of_the_Run
		runTop := ""
		if !theCfg.isConsole {
			if !isUseIdNames {
				runTop = filepath.Join(csvTop, "run."+helper.CleanFileName(rm.Name))
			} else {
				runTop = filepath.Join(csvTop, "run."+strconv.Itoa(rm.RunId)+"."+helper.CleanFileName(rm.Name))
			}
			if err = makeOutputDir(runTop, theCfg.isKeepOutputDir); err != nil {
				return err
			}
		}

		err = runValueOut(srcDb, meta, runMeta, runTop, isDefaultTop, runOpts)
		if err != nil {
			return err
		}
	}

	return nil
}

// write run list from database into text csv, tsv or json file
func runList(srcDb *sql.DB, modelId int, runOpts *config.RunOptions) error {

	// get model metadata
	meta, err := db.GetModelById(srcDb, modelId)
	if err != nil {
		return helper.ErrorNew("Error at get model metadata by id:", modelId, ":", err)
	}

	// get model run list and run_txt if user language defined
	rl := []db.RunRow{}
	rt := []db.RunTxtRow{}

	if !theCfg.isNoLang && theCfg.lang != "" {
		rl, rt, err = db.GetRunListText(srcDb, modelId, theCfg.lang)
	} else {
		rl, err = db.GetRunList(srcDb, modelId)
	}
	if err != nil {
		return helper.ErrorNew("Error at get model runs list:", err)
	}

	// for each run_lst find run_txt row if exist and convert to "public" run format
	rpl := make([]db.RunPub, len(rl))

	nt := 0
	for ni := range rl {

		// skip if run is not completed successfuly
		if rl[ni].Status != db.DoneRunStatus {
			continue
		}

		// find text row for current master row by run id
		isFound := false
		for ; nt < len(rt); nt++ {
			isFound = rt[nt].RunId == rl[ni].RunId
			if rt[nt].RunId >= rl[ni].RunId {
				break // text found or text missing: text run id ahead of master run id
			}
		}

		// convert to "public" format
		var p *db.RunPub

		if isFound && nt < len(rt) {
			p, err = (&db.RunMeta{Run: rl[ni], Txt: []db.RunTxtRow{rt[nt]}}).ToPublic(meta)
		} else {
			p, err = (&db.RunMeta{Run: rl[ni]}).ToPublic(meta)
		}
		if err != nil {
			return helper.ErrorNew("Error at run conversion:", err)
		}
		if p != nil {
			rpl[ni] = *p
		}
	}

	if len(rpl) <= 0 {
		omppLog.LogFmt("Do %s: there are no completed model runs", theCfg.action)
		return nil
	}

	// use specified file name or make default as modelName.run-list.json or .csv or .tsv
	fp := ""
	ext := extByKind()

	if theCfg.isConsole {
		omppLog.Log("Do", theCfg.action, meta.Model.Name)
	} else {
		fp = theCfg.fileName
		if fp == "" {
			fp = helper.CleanFileName(meta.Model.Name) + ".run-list" + ext
		}
		fp = filepath.Join(theCfg.dir, fp)

		omppLog.Log("Do", theCfg.action, ":", fp)
	}

	// write json output into file or console
	if theCfg.kind == asJson {
		return toJsonOutput(fp, rpl) // save results
	}
	// else write csv or tsv output into file or console

	// use of model id in notes .md file name if model name duplicates
	isUseIdNames := false
	for k := range rpl {
		for i := k + 1; i < len(rpl); i++ {
			if isUseIdNames = rpl[i].Name == rpl[k].Name; isUseIdNames {
				break
			}
		}
		if isUseIdNames {
			break
		}
	}

	// write model run rows into csv, including description
	row := make([]string, 13)

	idx := 0
	err = toCsvOutput(
		fp,
		[]string{
			"run_id", "run_name", "sub_count",
			"sub_started", "sub_completed", "create_dt", "status",
			"update_dt", "run_digest", "value_digest", "run_stamp", "lang_code", "descr"},
		func() (bool, []string, error) {
			if 0 <= idx && idx < len(rpl) {
				row[0] = strconv.Itoa(rpl[idx].RunId)
				row[1] = rpl[idx].Name
				row[2] = strconv.Itoa(rpl[idx].SubCount)
				row[3] = strconv.Itoa(rpl[idx].SubStarted)
				row[4] = strconv.Itoa(rpl[idx].SubCompleted)
				row[5] = rpl[idx].CreateDateTime
				row[6] = rpl[idx].Status
				row[7] = rpl[idx].UpdateDateTime
				row[8] = rpl[idx].RunDigest
				row[9] = rpl[idx].ValueDigest
				row[10] = rpl[idx].RunStamp
				row[11] = ""
				row[12] = ""

				// language, description and notes if any exist
				if !theCfg.isNoLang && len(rpl[idx].Txt) > 0 {

					row[11] = rpl[idx].Txt[0].LangCode
					row[12] = rpl[idx].Txt[0].Descr

					nm := rpl[idx].Name
					if isUseIdNames {
						nm = "run." + strconv.Itoa(rpl[idx].RunId) + "." + nm
					}
					if e := writeNote(
						theCfg.dir, nm, rpl[idx].Txt[0].LangCode, &rpl[idx].Txt[0].Note,
					); e != nil {
						return true, row, err
					}
				}

				idx++
				return false, row, nil
			}
			return true, row, nil // end of run_lst rows
		})
	if err != nil {
		return helper.ErrorNew("failed to write run list into csv", err)
	}

	return nil
}
