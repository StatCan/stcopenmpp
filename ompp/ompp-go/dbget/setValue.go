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

// write workset parameters into csv or tsv files
func setValue(srcDb *sql.DB, modelId int, runOpts *config.RunOptions) error {

	// get model metadata
	meta, err := db.GetModelById(srcDb, modelId)
	if err != nil {
		return helper.ErrorNew("Error at get model metadata by id:", modelId, ":", err)
	}

	// find workset, it must be readonly
	wsRow, err := findWs(srcDb, modelId, runOpts)
	if err != nil {
		return err
	}

	// create output directory
	// if output directory name not explicitly specified then use set.SetName by default
	wsDir := theCfg.dir

	if theCfg.isConsole {
		omppLog.Log("Do", theCfg.action, wsRow.Name)
	} else {

		if wsDir == "" {
			wsDir = "set." + helper.CleanFileName(wsRow.Name)
		}
		if err = makeOutputDir(wsDir, theCfg.isKeepOutputDir); err != nil {
			return err
		}
		omppLog.Log("Do", theCfg.action, ":", wsDir)
	}

	return setValueOut(srcDb, meta, wsRow, wsDir)
}

// write workset parameters into csv or tsv files
func setValueOut(srcDb *sql.DB, meta *db.ModelMeta, wsRow *db.WorksetRow, paramCsvDir string) error {

	// get workset parameters list
	hIds, _, _, err := db.GetWorksetParamList(srcDb, wsRow.SetId)
	if err != nil {
		return helper.ErrorNew("Error: unable to get workset parameters list:", wsRow.Name, ":", err)
	}

	// write all parameters into csv file
	nP := len(hIds)

	omppLog.Log("  Parameters:", nP)
	logT := time.Now().Unix()

	for j := 0; j < nP; j++ {

		idx, ok := meta.ParamByHid(hIds[j])
		if !ok {
			return helper.ErrorFmt("missing workset parameter Hid: %d workset: %s", hIds[j], wsRow.Name)
		}

		logT = omppLog.LogIfTime(logT, logPeriod, helper.Fmt("    %d of %d: %s", j, nP, meta.Param[idx].Name))

		fp := ""
		if !theCfg.isConsole {
			fp = filepath.Join(paramCsvDir, meta.Param[idx].Name+extByKind())
		}
		e := parameterValue(srcDb, meta, meta.Param[idx].Name, wsRow.SetId, true, fp, false, nil)
		if e != nil {
			return e
		}
	}

	return nil
}

// write all model worksets parameters into csv or tsv files
func setAllValue(srcDb *sql.DB, modelId int, runOpts *config.RunOptions) error {

	// get model metadata and list of readonly worksets
	meta, err := db.GetModelById(srcDb, modelId)
	if err != nil {
		return helper.ErrorNew("Error at get model metadata by id:", modelId, ":", err)
	}

	wsLst, err := db.GetWorksetList(srcDb, modelId)
	if err != nil {
		return helper.ErrorNew("Error at get workset list by model id:", modelId, ":", err)
	}
	wsLst = slices.DeleteFunc(wsLst, func(w db.WorksetRow) bool { return !w.IsReadonly })

	if len(wsLst) <= 0 {
		omppLog.Log("Do %s: there are no readonly worksets", theCfg.action)
		return nil
	}

	// create output directory
	// if output directory name not explicitly specified then use ModelName by default
	csvTop := theCfg.dir

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

	// for each workset write parameters into csv or tsv files
	for _, ws := range wsLst {

		if !ws.IsReadonly {
			continue // unexpected change of workset readonly status
		}
		omppLog.Log("Workset", ws.SetId, ws.Name)

		// workset output directory: set.Name
		wsDir := ""
		if !theCfg.isConsole {
			wsDir = filepath.Join(csvTop, "set."+helper.CleanFileName(ws.Name))

			if err = makeOutputDir(wsDir, theCfg.isKeepOutputDir); err != nil {
				return err
			}
		}

		err = setValueOut(srcDb, meta, &ws, wsDir)
		if err != nil {
			return err
		}
	}

	return nil
}

// write workset list from database into text csv, tsv or json file
func setList(srcDb *sql.DB, modelId int, runOpts *config.RunOptions) error {

	// get model metadata
	meta, err := db.GetModelById(srcDb, modelId)
	if err != nil {
		return helper.ErrorNew("Error at get model metadata by id:", modelId, ":", err)
	}

	// get model run list and run_txt if user language defined
	wl := []db.WorksetRow{}
	wt := []db.WorksetTxtRow{}

	if !theCfg.isNoLang && theCfg.lang != "" {
		wl, wt, err = db.GetWorksetListText(srcDb, modelId, theCfg.lang)
	} else {
		wl, err = db.GetWorksetList(srcDb, modelId)
	}
	if err != nil {
		return helper.ErrorNew("Error at get model workset list:", err)
	}

	if len(wl) <= 0 {
		omppLog.LogFmt("Do %s: there are no input sets (no input scenarios)", theCfg.action)
		return nil
	}

	// for each workset_lst find workset_txt row if exist and convert to "public" workset format
	wpl := make([]db.WorksetPub, len(wl))

	nt := 0
	for ni := range wl {

		// find text row for current master row by set id
		isFound := false
		for ; nt < len(wt); nt++ {
			isFound = wt[nt].SetId == wl[ni].SetId
			if wt[nt].SetId >= wl[ni].SetId {
				break // text found or text missing: text set id ahead of master set id
			}
		}

		// convert to "public" format
		var p *db.WorksetPub
		var err error

		if isFound && nt < len(wt) {
			p, err = (&db.WorksetMeta{Set: wl[ni], Txt: []db.WorksetTxtRow{wt[nt]}}).ToPublic(srcDb, meta)
		} else {
			p, err = (&db.WorksetMeta{Set: wl[ni]}).ToPublic(srcDb, meta)
		}
		if err != nil {
			return helper.ErrorNew("Error at workset conversion:", err)
		}
		if p != nil {
			wpl[ni] = *p
		}
	}

	// use specified file name or make default as modelName.set-list.json or .csv or .tsv
	fp := ""
	ext := extByKind()

	if theCfg.isConsole {
		omppLog.Log("Do", theCfg.action, meta.Model.Name)
	} else {
		fp = theCfg.fileName
		if fp == "" {
			fp = helper.CleanFileName(meta.Model.Name) + ".set-list" + ext
		}
		fp = filepath.Join(theCfg.dir, fp)

		omppLog.Log("Do", theCfg.action, ":", fp)
	}

	// write json output into file or console
	if theCfg.kind == asJson {
		return toJsonOutput(fp, wpl) // save results
	}
	// else write csv or tsv output into file or console

	// write model workset rows into csv, including description
	row := make([]string, 6)

	idx := 0
	err = toCsvOutput(
		fp,
		[]string{"set_name", "base_run_digest", "is_readonly", "update_dt", "lang_code", "descr"},
		func() (bool, []string, error) {
			if 0 <= idx && idx < len(wpl) {
				row[0] = wpl[idx].Name
				row[1] = wpl[idx].BaseRunDigest
				row[2] = strconv.FormatBool(wpl[idx].IsReadonly)
				row[3] = wpl[idx].UpdateDateTime
				row[4] = ""
				row[5] = ""

				// language, description and notes if any exist
				if !theCfg.isNoLang && len(wpl[idx].Txt) > 0 {

					row[4] = wpl[idx].Txt[0].LangCode
					row[5] = wpl[idx].Txt[0].Descr

					if e := writeNote(
						theCfg.dir, wpl[idx].Name, wpl[idx].Txt[0].LangCode, &wpl[idx].Txt[0].Note,
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
		return helper.ErrorNew("failed to write workset list into csv", err)
	}

	return nil
}
