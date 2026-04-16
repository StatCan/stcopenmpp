// Copyright OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"strconv"

	"golang.org/x/text/language"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
)

// match user language to the list of model languages, if no match then return empty "" model language code
func matchUserLang(srcDb *sql.DB, mdRow db.ModelDicRow) (string, error) {

	// get language list from database
	ls, err := db.GetLanguages(srcDb)
	if err != nil {
		return "", err
	}
	if ls == nil {
		return "", nil // no languages in database
	}

	// make model languages list, starting from default language
	ml := []string{}
	lt := []language.Tag{}

	for k := range ls.Lang {
		if ls.Lang[k].LangCode == mdRow.DefaultLangCode {
			ml = append([]string{ls.Lang[k].LangCode}, ml...)
			lt = append([]language.Tag{language.Make(ls.Lang[k].LangCode)}, lt...)
		} else {
			ml = append(ml, ls.Lang[k].LangCode)
			lt = append(lt, language.Make(ls.Lang[k].LangCode))
		}
	}
	matcher := language.NewMatcher(lt)

	// match user language to the list of database languages
	_, np, _ := matcher.Match(language.Make(theCfg.userLang))

	if np >= 0 && np < len(ml) {
		return ml[np], nil
	}
	return "", nil
}

// find model run row by digest, stamp or name, if rdsn is not "" empty, or by run id, if id > 0, or by first or last bool flag
func findRun(srcDb *sql.DB, modelId int, rdsn string, runId int, isFirst, isLast bool) (string, *db.RunRow, error) {

	if rdsn == "" && runId <= 0 && !isFirst && !isLast {
		return "", nil, nil
	}
	if rdsn != "" {
		r, e := db.GetRunByDigestStampName(srcDb, modelId, rdsn)
		return rdsn, r, e
	}
	if runId > 0 {
		r, e := db.GetRun(srcDb, runId)

		if e == nil && r != nil && r.ModelId != modelId {
			return "", nil, helper.ErrorNew("Error: model run not found by id:", runId)
		}
		return strconv.Itoa(runId), r, e
	}
	if isFirst {
		r, e := db.GetFirstRun(srcDb, modelId)
		return helper.LT("first model run"), r, e
	}
	// else: must be last model run
	r, e := db.GetLastRun(srcDb, modelId)
	return helper.LT("last model run"), r, e
}

// find workset by name or by id and check if it is readonly workset
func findWs(srcDb *sql.DB, modelId int, runOpts *config.RunOptions) (*db.WorksetRow, error) {

	wsName := runOpts.String(wsArgKey)
	var ws *db.WorksetRow
	var err error

	if wsName != "" {

		ws, err = db.GetWorksetByName(srcDb, modelId, wsName)
		if err != nil {
			return nil, helper.ErrorNew("Error at get workset:", wsName, err)
		}
		if ws == nil {
			return nil, helper.ErrorNew("Error: workset not found:", wsName)
		}
	} else {

		nId := runOpts.Int(wsIdArgKey, -1)
		if nId < 0 {
			return nil, helper.ErrorNew("Error: invalid (empty) input scenario name and id")
		}
		ws, err = db.GetWorkset(srcDb, nId)
		if err != nil {
			return nil, helper.ErrorNew("Error at get workset by id:", nId, err)
		}
		if ws == nil {
			return nil, helper.ErrorNew("Error: workset not found by id:", nId)
		}
		wsName = ws.Name
	}

	if !ws.IsReadonly {
		return nil, helper.ErrorNew("Error: workset must be read-only:", wsName)
	}

	return ws, nil
}
