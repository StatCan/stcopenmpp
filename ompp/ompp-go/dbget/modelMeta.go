// Copyright OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/stcopenmpp/ompp-go/ompp"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// write notes into Name.Lang.md file, ex: modelOne.FR.md or to console
func writeNote(dir, name string, langCode string, note *string) error {
	if !theCfg.isNote || note == nil || *note == "" {
		return nil
	}
	if theCfg.isConsole {
		fmt.Println(*note)
		return nil
	}

	nm := helper.CleanFileName(name)
	if langCode != "" {
		nm += "." + langCode
	}
	nm += ".md"

	err := os.WriteFile(filepath.Join(dir, nm), []byte(*note), 0644)
	if err != nil {
		return helper.ErrorNew("failed to write notes:", name, langCode, ":", err)
	}
	return nil
}

// write model metada from database into text csv, tsv or json file
func modelMeta(srcDb *sql.DB, modelId int) error {

	// get model metadata
	meta, err := db.GetModelById(srcDb, modelId)
	if err != nil {
		return helper.ErrorNew("Error at get model metadata by id:", modelId, ":", err)
	}
	if meta == nil {
		return helper.ErrorNew("Invalid (empty) model metadata")
	}

	// for json use specified file name or make default as modelName.model.json
	// for csv use specified directory or make default as modelName.model
	fp := ""
	dir := theCfg.dir
	ext := extByKind()

	if theCfg.isConsole {
		omppLog.Log("Do", theCfg.action, meta.Model.Name)
	} else {
		if theCfg.kind == asJson {

			fp = theCfg.fileName
			if fp == "" {
				fp = helper.CleanFileName(meta.Model.Name) + ".model.json"
			}
			fp = filepath.Join(theCfg.dir, fp)

			omppLog.Log("Do", theCfg.action, ":", fp)

		} else {
			if dir == "" {
				dir = meta.Model.Name + ".model"
			}
			// remove output directory if required, create output directory if not already exists
			if err := makeOutputDir(dir, theCfg.isKeepOutputDir); err != nil {
				return err
			}
			omppLog.Log("Do", theCfg.action, ":", dir)
		}
	}

	// write json output to console or file without language-specific part of model metadata
	if theCfg.isNoLang && theCfg.kind == asJson {
		return toJsonOutput(fp, ompp.CopyModelMetaToUnpack(meta))
	}
	// merge with language-specific portion of model metadata

	// read model text metadata from database and update catalog
	txt, err := db.GetModelText(srcDb, modelId, "", true)
	if err != nil {
		return helper.ErrorNew("Error at get model text metadata:", meta.Model.Name, ":", err)
	}

	me := ompp.ModelMetaEncoder{}
	err = me.New(meta, txt, theCfg.lang, meta.Model.DefaultLangCode)
	if err != nil {
		return helper.ErrorNew("Invalid (empty) model metadata, default model language:", meta.Model.DefaultLangCode, ":", err)
	}

	// write json output into file or console
	if theCfg.kind == asJson {

		var w io.Writer
		if fp == "" { // output to console
			w = os.Stdout
		} else {
			f, err := os.OpenFile(fp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
			if err != nil {
				return helper.ErrorNew("json file create error:", err)
			}
			defer f.Close()
			w = f
		}
		je := json.NewEncoder(w)
		je.SetIndent("", "  ")

		return me.DoEncode(false, je)
	}
	// else write csv or tsv output into file or console

	// get model languages map language id's to codes
	langIdCode := map[int]string{}
	langCodeId := map[string]int{}

	type lcv struct {
		langId int
		code   string
		value  string
	}
	langLst := []lcv{}

	err = db.SelectRows(srcDb,
		"SELECT L.lang_id, L.lang_code, L.lang_name FROM lang_lst L"+
			" WHERE L.lang_code = "+db.ToQuoted(theCfg.lang)+
			" ORDER BY 1",
		func(rows *sql.Rows) error {
			var r lcv
			if e := rows.Scan(&r.langId, &r.code, &r.value); e != nil {
				return e
			}
			langLst = append(langLst, r)
			langIdCode[r.langId] = r.code
			langCodeId[r.code] = r.langId
			return nil
		})
	if err != nil {
		return err
	}

	isAnyLang := !theCfg.isNoLang && len(langLst) > 0 // if any language required and found

	theLangId := 0
	sLangId := ""
	sLangCode := ""
	if len(langLst) > 0 {
		theLangId = langLst[0].langId
		sLangId = strconv.Itoa(theLangId)
		sLangCode = theCfg.lang
	}

	// make output path, return emtpy "" string to use console output
	outPath := func(name string) string {
		if theCfg.isConsole {
			return ""
		}
		return filepath.Join(dir, name+ext)
	}

	// write model_dic master row with description and notes
	sModelId := strconv.Itoa(modelId)

	meMd := me.MetaDescrNote.ModelDicDescrNote
	row := make([]string, 9)
	idx := 0
	err = toCsvOutput(
		outPath("model_dic"),
		[]string{"model_id", "model_name", "model_digest", "model_type", "model_ver", "create_dt", "default_lang_code", "lang_code", "descr"},
		func() (bool, []string, error) {
			if idx >= 1 {
				return true, row, nil // end of db rows
			}
			row[0] = sModelId
			row[1] = meMd.Model.Name
			row[2] = meMd.Model.Digest
			row[3] = strconv.Itoa(meMd.Model.Type)
			row[4] = meMd.Model.Version
			row[5] = meMd.Model.CreateDateTime
			row[6] = meMd.Model.DefaultLangCode
			if isAnyLang {
				row[7] = meMd.DescrNote.LangCode
				row[8] = meMd.DescrNote.Descr
				if e := writeNote(dir, "model_dic."+meMd.Model.Name, langIdCode[theLangId], &meMd.DescrNote.Note); e != nil {
					return false, row, e
				}
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "model_dic"+ext, err)
	}

	// write language row, language words and model words into csv
	if isAnyLang {

		row = make([]string, 3)
		idx := 0
		err = toCsvOutput(
			outPath("lang_lst"),
			[]string{"lang_id", "lang_code", "lang_name"},
			func() (bool, []string, error) {
				if idx >= len(langLst) {
					return true, row, nil // end of db rows
				}
				row[0] = sLangId
				row[1] = sLangCode
				row[2] = langLst[idx].value
				idx++
				return false, row, nil
			})
		if err != nil {
			return helper.ErrorNew("failed to write into", "lang_lst"+ext, err)
		}

		// write language word rows into csv
		lcvArr := []lcv{}
		err = db.SelectRows(srcDb,
			"SELECT lang_id, word_code, word_value FROM lang_word WHERE lang_id = "+sLangId+" ORDER BY 1, 2",
			func(rows *sql.Rows) error {
				var r lcv
				if e := rows.Scan(&r.langId, &r.code, &r.value); e != nil {
					return e
				}
				lcvArr = append(lcvArr, r)
				return nil
			})
		if err != nil {
			return err
		}

		row = make([]string, 3)
		idx = 0
		err = toCsvOutput(
			outPath("lang_word"),
			[]string{"lang_code", "word_code", "word_value"},
			func() (bool, []string, error) {
				if idx >= len(lcvArr) {
					return true, row, nil // end of db rows
				}
				row[0] = sLangCode
				row[1] = lcvArr[idx].code
				row[2] = lcvArr[idx].value
				idx++
				return false, row, nil
			})
		if err != nil {
			return helper.ErrorNew("failed to write into", "lang_word"+ext, err)
		}

		// write model words into csv
		lcvArr = lcvArr[:0]

		err = db.SelectRows(srcDb,
			"SELECT model_id, lang_id, word_code, word_value FROM model_word"+
				" WHERE model_id = "+sModelId+
				" AND lang_id = "+sLangId+
				" ORDER BY 1, 2, 3",
			func(rows *sql.Rows) error {
				var r lcv
				var mId int
				var srcVal sql.NullString
				if e := rows.Scan(&mId, &r.langId, &r.code, &srcVal); e != nil {
					return e
				}
				if srcVal.Valid {
					r.value = srcVal.String
				}
				lcvArr = append(lcvArr, r)
				return nil
			})
		if err != nil {
			return err
		}

		row = make([]string, 3)
		idx = 0
		err = toCsvOutput(
			outPath("model_word"),
			[]string{"lang_code", "word_code", "word_value"},
			func() (bool, []string, error) {
				if idx >= len(lcvArr) {
					return true, row, nil // end of db rows
				}
				row[0] = sLangCode
				row[1] = lcvArr[idx].code
				row[2] = lcvArr[idx].value
				idx++
				return false, row, nil
			})
		if err != nil {
			return helper.ErrorNew("failed to write into", "model_word"+ext, err)
		}
	}

	// write type rows with description and notes
	meTd := me.MetaDescrNote.TypeTxt
	row = make([]string, 10)
	idx = 0
	err = toCsvOutput(
		outPath("type_dic"),
		[]string{
			"model_id", "model_type_id", "type_name", "type_digest", "dic_id", "total_enum_id", "min_enum_id", "max_enum_id", "lang_code", "descr",
		},
		func() (bool, []string, error) {
			if idx >= len(meTd) {
				return true, row, nil // end of db rows
			}
			row[0] = sModelId
			row[1] = strconv.Itoa(meTd[idx].Type.TypeId)
			row[2] = meTd[idx].Type.Name
			row[3] = meTd[idx].Type.Digest
			row[4] = strconv.Itoa(meTd[idx].Type.DicId)
			row[5] = strconv.Itoa(meTd[idx].Type.TotalEnumId)
			row[6] = strconv.Itoa(meTd[idx].Type.MinEnumId)
			row[7] = strconv.Itoa(meTd[idx].Type.MaxEnumId)
			if isAnyLang {
				row[8] = *meTd[idx].DescrNote.LangCode
				row[9] = *meTd[idx].DescrNote.Descr
				if e := writeNote(dir, "type_dic."+meTd[idx].Type.Name, langIdCode[theLangId], meTd[idx].DescrNote.Note); e != nil {
					return false, row, e
				}
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "type_dic"+ext, err)
	}

	// write type_enum_lst rows with description and notes
	row = make([]string, 6)
	idx = 0
	j := 0
	err = toCsvOutput(
		outPath("type_enum_lst"),
		[]string{"model_id", "model_type_id", "enum_id", "enum_name", "lang_code", "descr"},
		func() (bool, []string, error) {

			if idx >= len(me.MetaDescrNote.TypeTxt) {
				return true, row, nil // end of db rows
			}
			// if end of current type enums then find next type with enum list or next range
			if !me.MetaDescrNote.TypeTxt[idx].Type.IsRange && j >= len(me.MetaDescrNote.TypeTxt[idx].TypeEnumTxt) ||
				me.MetaDescrNote.TypeTxt[idx].Type.IsRange && j > me.MetaDescrNote.TypeTxt[idx].Type.MaxEnumId-me.MetaDescrNote.TypeTxt[idx].Type.MinEnumId {

				j = 0
				for {
					idx++
					if idx >= len(me.MetaDescrNote.TypeTxt) { // end of type rows
						return true, row, nil
					}
					if me.MetaDescrNote.TypeTxt[idx].Type.IsRange || len(me.MetaDescrNote.TypeTxt[idx].TypeEnumTxt) > 0 {
						break
					}
				}
			}
			meTi := me.MetaDescrNote.TypeTxt[idx]

			// make type enum []string row
			row[0] = sModelId
			row[1] = strconv.Itoa(meTi.Type.TypeId)

			if !meTi.Type.IsRange {
				row[2] = strconv.Itoa(meTi.TypeEnumTxt[j].Enum.EnumId)
				row[3] = meTi.TypeEnumTxt[j].Enum.Name
				if isAnyLang {
					row[4] = *meTi.TypeEnumTxt[j].DescrNote.LangCode
					row[5] = *meTi.TypeEnumTxt[j].DescrNote.Descr
					if e := writeNote(dir, "type_enum_lst."+meTi.Type.Name+"."+meTi.TypeEnumTxt[j].Enum.Name, langIdCode[theLangId], meTi.TypeEnumTxt[j].DescrNote.Note); e != nil {
						return false, row, e
					}
				}
			} else {
				sId := strconv.Itoa(meTi.Type.MinEnumId + j) // range type: enum id is the same as enum code
				row[2] = sId
				row[3] = sId
			}
			j++

			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "type_enum_lst"+ext, err)
	}

	// write parameter rows with description and notes
	mePd := me.MetaDescrNote.ParamTxt
	row = make([]string, 13)
	idx = 0
	err = toCsvOutput(
		outPath("parameter_dic"),
		[]string{
			"model_id", "model_parameter_id", "parameter_name", "parameter_digest",
			"db_run_table", "db_set_table", "parameter_rank", "model_type_id",
			"is_hidden", "num_cumulated", "import_digest", "lang_code", "descr",
		},
		func() (bool, []string, error) {
			if idx >= len(mePd) {
				return true, row, nil // end of db rows
			}
			row[0] = sModelId
			row[1] = strconv.Itoa(mePd[idx].Param.ParamId)
			row[2] = mePd[idx].Param.Name
			row[3] = mePd[idx].Param.Digest
			row[4] = mePd[idx].Param.DbRunTable
			row[5] = mePd[idx].Param.DbSetTable
			row[6] = strconv.Itoa(mePd[idx].Param.Rank)
			row[7] = strconv.Itoa(mePd[idx].Param.TypeId)
			row[8] = strconv.FormatBool(mePd[idx].Param.IsHidden)
			row[9] = strconv.Itoa(mePd[idx].Param.NumCumulated)
			row[10] = mePd[idx].Param.ImportDigest
			if isAnyLang {
				row[11] = *mePd[idx].DescrNote.LangCode
				row[12] = *mePd[idx].DescrNote.Descr
				if e := writeNote(dir, "parameter_dic."+mePd[idx].Param.Name, langIdCode[theLangId], mePd[idx].DescrNote.Note); e != nil {
					return false, row, e
				}
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "parameter_dic"+ext, err)
	}

	// write parameter_dims rows with description and notes
	row = make([]string, 7)
	idx = 0
	j = 0
	err = toCsvOutput(
		outPath("parameter_dims"),
		[]string{"model_id", "model_parameter_id", "dim_id", "dim_name", "model_type_id", "lang_code", "descr"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(me.MetaDescrNote.ParamTxt) { // end of parameter rows
				return true, row, nil
			}

			// if end of current parameter dimensions then find next parameter with dimension list
			if j >= len(me.MetaDescrNote.ParamTxt[idx].ParamDimsTxt) {

				j = 0
				for {
					idx++
					if idx >= len(me.MetaDescrNote.ParamTxt) { // end of parameter rows
						return true, row, nil
					}
					if len(me.MetaDescrNote.ParamTxt[idx].ParamDimsTxt) > 0 {
						break
					}
				}
			}
			mePi := me.MetaDescrNote.ParamTxt[idx]

			row[0] = sModelId
			row[1] = strconv.Itoa(mePi.Param.ParamId)
			row[2] = strconv.Itoa(mePi.ParamDimsTxt[j].Dim.DimId)
			row[3] = mePi.ParamDimsTxt[j].Dim.Name
			row[4] = strconv.Itoa(mePi.ParamDimsTxt[j].Dim.TypeId)
			if isAnyLang {
				row[5] = *mePi.ParamDimsTxt[j].DescrNote.LangCode
				row[6] = *mePi.ParamDimsTxt[j].DescrNote.Descr
				if e := writeNote(dir, "parameter_dims."+mePi.Param.Name+"."+mePi.ParamDimsTxt[j].Dim.Name, langIdCode[theLangId], mePi.ParamDimsTxt[j].DescrNote.Note); e != nil {
					return false, row, e
				}
			}
			j++

			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "parameter_dims"+ext, err)
	}

	// write table_dic rows with description and notes
	meTbl := me.MetaDescrNote.TableTxt
	row = make([]string, 16)
	idx = 0
	err = toCsvOutput(
		outPath("table_dic"),
		[]string{
			"model_id", "model_table_id", "table_name", "table_digest",
			"is_user", "table_rank", "is_sparse", "db_expr_table",
			"db_acc_table", "db_acc_all_view", "expr_dim_pos", "is_hidden",
			"import_digest", "lang_code", "descr", "expr_descr",
		},
		func() (bool, []string, error) {
			if idx >= len(meTbl) {
				return true, row, nil // end of db rows
			}
			row[0] = sModelId
			row[1] = strconv.Itoa(meTbl[idx].Table.TableId)
			row[2] = meTbl[idx].Table.Name
			row[3] = meTbl[idx].Table.Digest
			row[4] = strconv.FormatBool(meTbl[idx].Table.IsUser)
			row[5] = strconv.Itoa(meTbl[idx].Table.Rank)
			row[6] = strconv.FormatBool(meTbl[idx].Table.IsSparse)
			row[7] = meTbl[idx].Table.DbExprTable
			row[8] = meTbl[idx].Table.DbAccTable
			row[9] = meTbl[idx].Table.DbAccAllView
			row[10] = strconv.Itoa(meTbl[idx].Table.ExprPos)
			row[11] = strconv.FormatBool(meTbl[idx].Table.IsHidden)
			row[12] = meTbl[idx].Table.ImportDigest
			if isAnyLang {
				row[13] = *meTbl[idx].LangCode
				row[14] = *meTbl[idx].TableDescr
				row[15] = *meTbl[idx].ExprDescr
				if e := writeNote(dir, "table_dic."+meTbl[idx].Table.Name, langIdCode[theLangId], meTbl[idx].TableNote); e != nil {
					return false, row, e
				}
				if e := writeNote(dir, "table_dic.expr."+meTbl[idx].Table.Name, langIdCode[theLangId], meTbl[idx].ExprNote); e != nil {
					return false, row, e
				}
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "table_dic"+ext, err)
	}

	// write table_dims rows with description and notes
	row = make([]string, 9)
	idx = 0
	j = 0
	err = toCsvOutput(
		outPath("table_dims"),
		[]string{"model_id", "model_table_id", "dim_id", "dim_name", "model_type_id", "is_total", "dim_size", "lang_code", "descr"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(me.MetaDescrNote.TableTxt) { // end of table rows
				return true, row, nil
			}

			// if end of current table dimensions then find next table with dimension list
			if j >= len(me.MetaDescrNote.TableTxt[idx].TableDimsTxt) {

				j = 0
				for {
					idx++
					if idx >= len(me.MetaDescrNote.TableTxt) { // end of table rows
						return true, row, nil
					}
					if len(me.MetaDescrNote.TableTxt[idx].TableDimsTxt) > 0 {
						break
					}
				}
			}
			meTbi := me.MetaDescrNote.TableTxt[idx]

			row[0] = sModelId
			row[1] = strconv.Itoa(meTbi.Table.TableId)
			row[2] = strconv.Itoa(meTbi.TableDimsTxt[j].Dim.DimId)
			row[3] = meTbi.TableDimsTxt[j].Dim.Name
			row[4] = strconv.Itoa(meTbi.TableDimsTxt[j].Dim.TypeId)
			row[5] = strconv.FormatBool(meTbi.TableDimsTxt[j].Dim.IsTotal)
			row[6] = strconv.Itoa(meTbi.TableDimsTxt[j].Dim.DimSize)
			if isAnyLang {
				row[7] = *meTbi.TableDimsTxt[j].DescrNote.LangCode
				row[8] = *meTbi.TableDimsTxt[j].DescrNote.Descr
				if e := writeNote(dir, "table_dims."+meTbi.Table.Name+"."+meTbi.TableDimsTxt[j].Dim.Name, langIdCode[theLangId], meTbi.TableDimsTxt[j].DescrNote.Note); e != nil {
					return false, row, e
				}
			}
			j++

			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "table_dims"+ext, err)
	}

	// write table_acc rows with description and notes
	row = make([]string, 8)
	idx = 0
	j = 0
	err = toCsvOutput(
		outPath("table_acc"),
		[]string{"model_id", "model_table_id", "acc_id", "acc_name", "is_derived", "acc_src", "lang_code", "descr"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(me.MetaDescrNote.TableTxt) { // end of table rows
				return true, row, nil
			}

			// if end of current table accumulators then find next table with accumulator list
			if j >= len(me.MetaDescrNote.TableTxt[idx].TableAccTxt) {

				j = 0
				for {
					idx++
					if idx >= len(me.MetaDescrNote.TableTxt) { // end of table rows
						return true, row, nil
					}
					if len(me.MetaDescrNote.TableTxt[idx].TableAccTxt) > 0 {
						break
					}
				}
			}
			meTbi := me.MetaDescrNote.TableTxt[idx]

			row[0] = sModelId
			row[1] = strconv.Itoa(meTbi.Table.TableId)
			row[2] = strconv.Itoa(meTbi.TableAccTxt[j].Acc.AccId)
			row[3] = meTbi.TableAccTxt[j].Acc.Name
			row[4] = strconv.FormatBool(meTbi.TableAccTxt[j].Acc.IsDerived)
			row[5] = meTbi.TableAccTxt[j].Acc.SrcAcc
			if isAnyLang {
				row[6] = *meTbi.TableAccTxt[j].DescrNote.LangCode
				row[7] = *meTbi.TableAccTxt[j].DescrNote.Descr
				if e := writeNote(dir, "table_acc."+meTbi.Table.Name+"."+meTbi.TableAccTxt[j].Acc.Name, langIdCode[theLangId], meTbi.TableAccTxt[j].DescrNote.Note); e != nil {
					return false, row, e
				}
			}
			j++

			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "table_acc"+ext, err)
	}

	// write table_expr rows with description and notes
	row = make([]string, 8)
	idx = 0
	j = 0
	err = toCsvOutput(
		outPath("table_expr"),
		[]string{"model_id", "model_table_id", "expr_id", "expr_name", "expr_decimals", "expr_src", "lang_code", "descr"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(me.MetaDescrNote.TableTxt) { // end of table rows
				return true, row, nil
			}

			// if end of current table expressions then find next table with expression list
			if j >= len(me.MetaDescrNote.TableTxt[idx].TableExprTxt) {

				j = 0
				for {
					idx++
					if idx >= len(me.MetaDescrNote.TableTxt) { // end of table rows
						return true, row, nil
					}
					if len(me.MetaDescrNote.TableTxt[idx].TableExprTxt) > 0 {
						break
					}
				}
			}
			meTbi := me.MetaDescrNote.TableTxt[idx]

			row[0] = sModelId
			row[1] = strconv.Itoa(meTbi.Table.TableId)
			row[2] = strconv.Itoa(meTbi.TableExprTxt[j].Expr.ExprId)
			row[3] = meTbi.TableExprTxt[j].Expr.Name
			row[4] = strconv.Itoa(meTbi.TableExprTxt[j].Expr.Decimals)
			row[5] = meTbi.TableExprTxt[j].Expr.SrcExpr
			if isAnyLang {
				row[6] = *meTbi.TableExprTxt[j].DescrNote.LangCode
				row[7] = *meTbi.TableExprTxt[j].DescrNote.Descr
				if e := writeNote(dir, "table_expr."+meTbi.Table.Name+"."+meTbi.TableExprTxt[j].Expr.Name, langIdCode[theLangId], meTbi.TableExprTxt[j].DescrNote.Note); e != nil {
					return false, row, e
				}
			}
			j++

			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "table_expr"+ext, err)
	}

	// write entity rows with description and notes
	meEnt := me.MetaDescrNote.EntityTxt
	row = make([]string, 6)
	idx = 0
	err = toCsvOutput(
		outPath("entity_dic"),
		[]string{
			"model_id", "model_entity_id", "entity_name", "entity_digest", "lang_code", "descr",
		},
		func() (bool, []string, error) {
			if idx >= len(meEnt) {
				return true, row, nil // end of db rows
			}
			row[0] = sModelId
			row[1] = strconv.Itoa(meEnt[idx].Entity.EntityId)
			row[2] = meEnt[idx].Entity.Name
			row[3] = meEnt[idx].Entity.Digest
			if isAnyLang {
				row[4] = *meEnt[idx].DescrNote.LangCode
				row[5] = *meEnt[idx].DescrNote.Descr
				if e := writeNote(dir, "entity_dic."+meEnt[idx].Entity.Name, langIdCode[theLangId], meEnt[idx].DescrNote.Note); e != nil {
					return false, row, e
				}
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "entity_dic"+ext, err)
	}

	// write entity attribute rows with description and notes
	row = make([]string, 8)
	idx = 0
	j = 0
	err = toCsvOutput(
		outPath("entity_attr"),
		[]string{"model_id", "model_entity_id", "attr_id", "attr_name", "model_type_id", "is_internal", "lang_code", "descr"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(me.MetaDescrNote.EntityTxt) { // end of entity rows
				return true, row, nil
			}

			// if end of current entity attributes then find next entity with attribute list
			if j >= len(me.MetaDescrNote.EntityTxt[idx].EntityAttrTxt) {

				j = 0
				for {
					idx++
					if idx >= len(me.MetaDescrNote.EntityTxt) { // end of entity rows
						return true, row, nil
					}
					if len(me.MetaDescrNote.EntityTxt[idx].EntityAttrTxt) > 0 {
						break
					}
				}
			}
			meEni := me.MetaDescrNote.EntityTxt[idx]

			row[0] = sModelId
			row[1] = strconv.Itoa(meEni.Entity.EntityId)
			row[2] = strconv.Itoa(meEni.EntityAttrTxt[j].Attr.AttrId)
			row[3] = meEni.EntityAttrTxt[j].Attr.Name
			row[4] = strconv.Itoa(meEni.EntityAttrTxt[j].Attr.TypeId)
			row[5] = strconv.FormatBool(meEni.EntityAttrTxt[j].Attr.IsInternal)
			if isAnyLang {
				row[6] = *meEni.EntityAttrTxt[j].DescrNote.LangCode
				row[7] = *meEni.EntityAttrTxt[j].DescrNote.Descr
				if e := writeNote(dir, "entity_attr."+meEni.Entity.Name+"."+meEni.EntityAttrTxt[j].Attr.Name, langIdCode[theLangId], meEni.EntityAttrTxt[j].DescrNote.Note); e != nil {
					return false, row, e
				}
			}
			j++

			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "entity_attr"+ext, err)
	}

	// write group rows with description and notes
	meGrp := me.MetaDescrNote.GroupTxt
	row = make([]string, 7)
	idx = 0
	err = toCsvOutput(
		outPath("group_lst"),
		[]string{
			"model_id", "group_id", "is_parameter", "group_name", "is_hidden", "lang_code", "descr",
		},
		func() (bool, []string, error) {
			if idx >= len(meGrp) {
				return true, row, nil // end of db rows
			}
			row[0] = sModelId
			row[1] = strconv.Itoa(meGrp[idx].Group.GroupId)
			row[2] = strconv.FormatBool(meGrp[idx].Group.IsParam)
			row[3] = meGrp[idx].Group.Name
			row[4] = strconv.FormatBool(meGrp[idx].Group.IsHidden)
			if isAnyLang {
				row[5] = *meGrp[idx].DescrNote.LangCode
				row[6] = *meGrp[idx].DescrNote.Descr
				if e := writeNote(dir, "group_lst."+meGrp[idx].Group.Name, langIdCode[theLangId], meGrp[idx].DescrNote.Note); e != nil {
					return false, row, e
				}
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "group_lst"+ext, err)
	}

	// write group parent-child rows into csv
	row = make([]string, 5)
	idx = 0
	j = 0
	err = toCsvOutput(
		outPath("group_pc"),
		[]string{"model_id", "group_id", "child_pos", "child_group_id", "leaf_id"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(me.MetaDescrNote.GroupTxt) { // end of group rows
				return true, row, nil
			}

			// if end of current group parent-child then find next group with children
			if j >= len(me.MetaDescrNote.GroupTxt[idx].Group.GroupPc) {

				j = 0
				for {
					idx++
					if idx >= len(me.MetaDescrNote.GroupTxt) { // end of group rows
						return true, row, nil
					}
					if len(me.MetaDescrNote.GroupTxt[idx].Group.GroupPc) > 0 {
						break
					}
				}
			}
			meGi := me.MetaDescrNote.GroupTxt[idx].Group

			row[0] = sModelId
			row[1] = strconv.Itoa(meGi.GroupId)
			row[2] = strconv.Itoa(meGi.GroupPc[j].ChildPos)

			if meGi.GroupPc[j].ChildGroupId < 0 { // negative value is NULL
				row[3] = "NULL"
			} else {
				row[3] = strconv.Itoa(meGi.GroupPc[j].ChildGroupId)
			}
			if meGi.GroupPc[j].ChildLeafId < 0 { // negative value is NULL
				row[4] = "NULL"
			} else {
				row[4] = strconv.Itoa(meGi.GroupPc[j].ChildLeafId)
			}
			j++

			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "group_pc"+ext, err)
	}

	// write entity attribute group rows with description and notes
	meAgr := me.MetaDescrNote.EntityGroupTxt
	row = make([]string, 7)
	idx = 0
	err = toCsvOutput(
		outPath("entity_group_lst"),
		[]string{
			"model_id", "model_entity_id", "group_id", "group_name", "is_hidden", "lang_code", "descr",
		},
		func() (bool, []string, error) {
			if idx >= len(meAgr) {
				return true, row, nil // end of db rows
			}
			row[0] = sModelId
			row[1] = strconv.Itoa(meAgr[idx].Group.EntityId)
			row[2] = strconv.Itoa(meAgr[idx].Group.GroupId)
			row[3] = meAgr[idx].Group.Name
			row[4] = strconv.FormatBool(meAgr[idx].Group.IsHidden)
			if isAnyLang {
				row[5] = *meAgr[idx].DescrNote.LangCode
				row[6] = *meAgr[idx].DescrNote.Descr
				if e := writeNote(dir, "entity_group_lst."+meAgr[idx].Group.Name, langIdCode[theLangId], meAgr[idx].DescrNote.Note); e != nil {
					return false, row, e
				}
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "entity_group_lst"+ext, err)
	}

	// write entity attributes group parent-child rows into csv
	row = make([]string, 6)
	idx = 0
	j = 0
	err = toCsvOutput(
		outPath("entity_group_pc"),
		[]string{"model_id", "model_entity_id", "group_id", "child_pos", "child_group_id", "attr_id"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(me.MetaDescrNote.EntityGroupTxt) { // end of group rows
				return true, row, nil
			}

			// if end of current group parent-child then find next group with children
			if j >= len(me.MetaDescrNote.EntityGroupTxt[idx].Group.GroupPc) {

				j = 0
				for {
					idx++
					if idx >= len(me.MetaDescrNote.EntityGroupTxt) { // end of group rows
						return true, row, nil
					}
					if len(me.MetaDescrNote.EntityGroupTxt[idx].Group.GroupPc) > 0 {
						break
					}
				}
			}
			meAgi := me.MetaDescrNote.EntityGroupTxt[idx].Group

			row[0] = sModelId
			row[1] = strconv.Itoa(meAgi.EntityId)
			row[2] = strconv.Itoa(meAgi.GroupId)
			row[3] = strconv.Itoa(meAgi.GroupPc[j].ChildPos)

			if meAgi.GroupPc[j].ChildGroupId < 0 { // negative value is NULL
				row[4] = "NULL"
			} else {
				row[4] = strconv.Itoa(meAgi.GroupPc[j].ChildGroupId)
			}
			if meAgi.GroupPc[j].AttrId < 0 { // negative value is NULL
				row[5] = "NULL"
			} else {
				row[5] = strconv.Itoa(meAgi.GroupPc[j].AttrId)
			}
			j++

			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "entity_group_pc"+ext, err)
	}

	return nil
}
