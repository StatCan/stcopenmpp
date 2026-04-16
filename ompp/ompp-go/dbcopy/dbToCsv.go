// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/stcopenmpp/ompp-go/ompp/config"
	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// lineCsvConverter return csv file row []string or isEof = true
type lineCsvConverter func() (isEof bool, row []string, err error)

// write model metadata from database into text csv files
func dbToCsv(modelName string, modelDigest string, isAllInOne bool, runOpts *config.RunOptions) error {

	// open source database connection and check is it valid
	cs, dn := db.IfEmptyMakeDefaultReadOnly(modelName, runOpts.String(fromSqliteArgKey), runOpts.String(dbConnStrArgKey), runOpts.String(dbDriverArgKey))

	srcDb, _, err := db.Open(cs, dn, false)
	if err != nil {
		return err
	}
	defer srcDb.Close()

	if err := db.CheckOpenmppSchemaVersion(srcDb); err != nil {
		return err
	}

	// get model metadata
	modelDef, err := db.GetModel(srcDb, modelName, modelDigest)
	if err != nil {
		return err
	}
	modelName = modelDef.Model.Name // set model name: it can be empty and only model digest specified

	// create new output directory, use modelName subdirectory
	outDir := filepath.Join(runOpts.String(outputDirArgKey), modelName)

	if !theCfg.isKeepOutputDir {
		if ok := dirDeleteAndLog(outDir); !ok {
			return helper.ErrorNew("Error: unable to delete:" + outDir)
		}
	}
	if err = os.MkdirAll(outDir, 0750); err != nil {
		return err
	}
	fileCreated := make(map[string]bool)

	// write model definition into csv files
	if err = toModelCsv(srcDb, modelDef, outDir); err != nil {
		return err
	}

	// write list of languages into csv file
	if err = toLanguageCsv(srcDb, outDir); err != nil {
		return err
	}

	// write model language-specific strings into csv file
	if err = toModelWordCsv(srcDb, modelDef.Model.ModelId, outDir); err != nil {
		return err
	}

	// write model text (description and notes) into csv file
	if err = toModelTextCsv(srcDb, modelDef.Model.ModelId, outDir); err != nil {
		return err
	}

	// write model profile into csv file
	if err = toModelProfileCsv(srcDb, modelName, outDir); err != nil {
		return err
	}

	// use of run and set id's in directory names:
	// if true then always use id's in the names, false never use it
	// by default: only if name conflict
	doUseIdNames := defaultUseIdNames
	if runOpts.IsExist(useIdNamesArgKey) {
		if runOpts.Bool(useIdNamesArgKey) {
			doUseIdNames = yesUseIdNames
		} else {
			doUseIdNames = noUseIdNames
		}
	}
	isIdNames := false

	// write all model run data into csv files: parameters, output expressions and accumulators
	if isIdNames, err = toRunListCsv(srcDb, modelDef, outDir, fileCreated, doUseIdNames, isAllInOne); err != nil {
		return err
	}

	// write all readonly workset data into csv files: input parameters
	if err = toWorksetListCsv(srcDb, modelDef, outDir, fileCreated, isIdNames, isAllInOne); err != nil {
		return err
	}

	// write all modeling tasks and task run history into csv files
	if err = toTaskListCsv(srcDb, modelDef.Model.ModelId, outDir); err != nil {
		return err
	}

	// pack model metadata, run results and worksets into zip
	if runOpts.Bool(zipArgKey) {
		zipPath, err := helper.PackZip(outDir, !theCfg.isKeepOutputDir, "")
		if err != nil {
			return err
		}
		omppLog.Log("Packed", zipPath)
	}

	return nil
}

// toModelCsv writes model metadata into csv files.
func toModelCsv(dbConn *sql.DB, modelDef *db.ModelMeta, outDir string) error {

	// write model master row into csv
	row := make([]string, 7)
	row[0] = strconv.Itoa(modelDef.Model.ModelId)
	row[1] = modelDef.Model.Name
	row[2] = modelDef.Model.Digest
	row[3] = strconv.Itoa(modelDef.Model.Type)
	row[4] = modelDef.Model.Version
	row[5] = modelDef.Model.CreateDateTime
	row[6] = modelDef.Model.DefaultLangCode

	idx := 0
	err := toCsvFile(
		outDir,
		"model_dic.csv",
		[]string{"model_id", "model_name", "model_digest", "model_type", "model_ver", "create_dt", "default_lang_code"},
		func() (bool, []string, error) {
			if idx == 0 { // only one model_dic row exist
				idx++
				return false, row, nil
			}
			return true, row, nil // end of model rows
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "model_dic.csv", err)
	}

	// write type rows into csv
	row = make([]string, 7)
	row[0] = strconv.Itoa(modelDef.Model.ModelId)

	idx = 0
	err = toCsvFile(
		outDir,
		"type_dic.csv",
		[]string{"model_id", "model_type_id", "type_hid", "type_name", "type_digest", "dic_id", "total_enum_id"},
		func() (bool, []string, error) {
			if 0 <= idx && idx < len(modelDef.Type) {
				row[1] = strconv.Itoa(modelDef.Type[idx].TypeId)
				row[2] = strconv.Itoa(modelDef.Type[idx].TypeHid)
				row[3] = modelDef.Type[idx].Name
				row[4] = modelDef.Type[idx].Digest
				row[5] = strconv.Itoa(modelDef.Type[idx].DicId)
				row[6] = strconv.Itoa(modelDef.Type[idx].TotalEnumId)
				idx++
				return false, row, nil
			}
			return true, row, nil // end of type rows
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "type_dic.csv", err)
	}

	// write type enum rows into csv
	row = make([]string, 4)
	row[0] = strconv.Itoa(modelDef.Model.ModelId)

	idx = 0
	j := 0
	err = toCsvFile(
		outDir,
		"type_enum_lst.csv",
		[]string{"model_id", "model_type_id", "enum_id", "enum_name"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(modelDef.Type) { // end of type rows
				return true, row, nil
			}

			// if end of current type enums then find next type with enum list
			if j < 0 ||
				!modelDef.Type[idx].IsRange && j >= len(modelDef.Type[idx].Enum) ||
				modelDef.Type[idx].IsRange && j > modelDef.Type[idx].MaxEnumId-modelDef.Type[idx].MinEnumId {

				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(modelDef.Type) { // end of type rows
						return true, row, nil
					}
					if modelDef.Type[idx].IsRange || len(modelDef.Type[idx].Enum) > 0 {
						break
					}
				}
			}

			// make type enum []string row
			if !modelDef.Type[idx].IsRange {
				row[1] = strconv.Itoa(modelDef.Type[idx].Enum[j].TypeId)
				row[2] = strconv.Itoa(modelDef.Type[idx].Enum[j].EnumId)
				row[3] = modelDef.Type[idx].Enum[j].Name
			} else {
				row[1] = strconv.Itoa(modelDef.Type[idx].TypeId)
				sId := strconv.Itoa(modelDef.Type[idx].MinEnumId + j) // range type: enum id is the same as enum code
				row[2] = sId
				row[3] = sId
			}
			j++

			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "type_enum_lst.csv", err)
	}

	// write parameter rows into csv
	row = make([]string, 12)
	row[0] = strconv.Itoa(modelDef.Model.ModelId)

	idx = 0
	err = toCsvFile(
		outDir,
		"parameter_dic.csv",
		[]string{
			"model_id", "model_parameter_id", "parameter_hid", "parameter_name",
			"parameter_digest", "db_run_table", "db_set_table", "parameter_rank",
			"model_type_id", "is_hidden", "num_cumulated", "import_digest"},
		func() (bool, []string, error) {
			if 0 <= idx && idx < len(modelDef.Param) {
				row[1] = strconv.Itoa(modelDef.Param[idx].ParamId)
				row[2] = strconv.Itoa(modelDef.Param[idx].ParamHid)
				row[3] = modelDef.Param[idx].Name
				row[4] = modelDef.Param[idx].Digest
				row[5] = modelDef.Param[idx].DbRunTable
				row[6] = modelDef.Param[idx].DbSetTable
				row[7] = strconv.Itoa(modelDef.Param[idx].Rank)
				row[8] = strconv.Itoa(modelDef.Param[idx].TypeId)
				row[9] = strconv.FormatBool(modelDef.Param[idx].IsHidden)
				row[10] = strconv.Itoa(modelDef.Param[idx].NumCumulated)
				row[11] = modelDef.Param[idx].ImportDigest
				idx++
				return false, row, nil
			}
			return true, row, nil // end of parameter rows
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "parameter_dic.csv", err)
	}

	// write parameter import rows into csv
	row = make([]string, 5)
	row[0] = strconv.Itoa(modelDef.Model.ModelId)

	idx = 0
	j = 0
	err = toCsvFile(
		outDir,
		"parameter_import.csv",
		[]string{"model_id", "model_parameter_id", "from_name", "from_model_name", "is_sample_dim"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(modelDef.Param) { // end of parameter rows
				return true, row, nil
			}

			// if end of current parameter imports then find next parameter with import list
			if j < 0 || j >= len(modelDef.Param[idx].Import) {
				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(modelDef.Param) { // end of parameter rows
						return true, row, nil
					}
					if len(modelDef.Param[idx].Import) > 0 {
						break
					}
				}
			}

			// make parameter import []string row
			row[1] = strconv.Itoa(modelDef.Param[idx].Import[j].ParamId)
			row[2] = modelDef.Param[idx].Import[j].FromName
			row[3] = modelDef.Param[idx].Import[j].FromModel
			row[4] = strconv.FormatBool(modelDef.Param[idx].Import[j].IsSampleDim)
			j++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "parameter_import.csv", err)
	}

	// write parameter dimension rows into csv
	row = make([]string, 5)
	row[0] = strconv.Itoa(modelDef.Model.ModelId)

	idx = 0
	j = 0
	err = toCsvFile(
		outDir,
		"parameter_dims.csv",
		[]string{"model_id", "model_parameter_id", "dim_id", "dim_name", "model_type_id"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(modelDef.Param) { // end of parameter rows
				return true, row, nil
			}

			// if end of current parameter dimensions then find next parameter with dimension list
			if j < 0 || j >= len(modelDef.Param[idx].Dim) {
				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(modelDef.Param) { // end of parameter rows
						return true, row, nil
					}
					if len(modelDef.Param[idx].Dim) > 0 {
						break
					}
				}
			}

			// make parameter dimension []string row
			row[1] = strconv.Itoa(modelDef.Param[idx].Dim[j].ParamId)
			row[2] = strconv.Itoa(modelDef.Param[idx].Dim[j].DimId)
			row[3] = modelDef.Param[idx].Dim[j].Name
			row[4] = strconv.Itoa(modelDef.Param[idx].Dim[j].TypeId)
			j++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "parameter_dims.csv", err)
	}

	// write output table rows into csv
	row = make([]string, 14)
	row[0] = strconv.Itoa(modelDef.Model.ModelId)

	idx = 0
	err = toCsvFile(
		outDir,
		"table_dic.csv",
		[]string{
			"model_id", "model_table_id", "table_hid", "table_name",
			"table_digest", "is_user", "table_rank", "is_sparse",
			"db_expr_table", "db_acc_table", "db_acc_all_view", "expr_dim_pos",
			"is_hidden", "import_digest"},
		func() (bool, []string, error) {
			if 0 <= idx && idx < len(modelDef.Table) {
				row[1] = strconv.Itoa(modelDef.Table[idx].TableId)
				row[2] = strconv.Itoa(modelDef.Table[idx].TableHid)
				row[3] = modelDef.Table[idx].Name
				row[4] = modelDef.Table[idx].Digest
				row[5] = strconv.FormatBool(modelDef.Table[idx].IsUser)
				row[6] = strconv.Itoa(modelDef.Table[idx].Rank)
				row[7] = strconv.FormatBool(modelDef.Table[idx].IsSparse)
				row[8] = modelDef.Table[idx].DbExprTable
				row[9] = modelDef.Table[idx].DbAccTable
				row[10] = modelDef.Table[idx].DbAccAllView
				row[11] = strconv.Itoa(modelDef.Table[idx].ExprPos)
				row[12] = strconv.FormatBool(modelDef.Table[idx].IsHidden)
				row[13] = modelDef.Table[idx].ImportDigest
				idx++
				return false, row, nil
			}
			return true, row, nil // end of output table rows
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "table_dic.csv", err)
	}

	// write output tables dimension rows into csv
	row = make([]string, 7)
	row[0] = strconv.Itoa(modelDef.Model.ModelId)

	idx = 0
	j = 0
	err = toCsvFile(
		outDir,
		"table_dims.csv",
		[]string{"model_id", "model_table_id", "dim_id", "dim_name", "model_type_id", "is_total", "dim_size"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(modelDef.Table) { // end of output tables rows
				return true, row, nil
			}

			// if end of current output tables dimensions then find next output table with dimension list
			if j < 0 || j >= len(modelDef.Table[idx].Dim) {
				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(modelDef.Table) { // end of output tables rows
						return true, row, nil
					}
					if len(modelDef.Table[idx].Dim) > 0 {
						break
					}
				}
			}

			// make output table dimension []string row
			row[1] = strconv.Itoa(modelDef.Table[idx].Dim[j].TableId)
			row[2] = strconv.Itoa(modelDef.Table[idx].Dim[j].DimId)
			row[3] = modelDef.Table[idx].Dim[j].Name
			row[4] = strconv.Itoa(modelDef.Table[idx].Dim[j].TypeId)
			row[5] = strconv.FormatBool(modelDef.Table[idx].Dim[j].IsTotal)
			row[6] = strconv.Itoa(modelDef.Table[idx].Dim[j].DimSize)
			j++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "table_dims.csv", err)
	}

	// write output tables accumulator rows into csv
	row = make([]string, 6)
	row[0] = strconv.Itoa(modelDef.Model.ModelId)

	idx = 0
	j = 0
	err = toCsvFile(
		outDir,
		"table_acc.csv",
		[]string{"model_id", "model_table_id", "acc_id", "acc_name", "is_derived", "acc_src"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(modelDef.Table) { // end of output table rows
				return true, row, nil
			}

			// if end of current output tables accumulators then find next output table with accumulator list
			if j < 0 || j >= len(modelDef.Table[idx].Acc) {
				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(modelDef.Table) { // end of output table rows
						return true, row, nil
					}
					if len(modelDef.Table[idx].Acc) > 0 {
						break
					}
				}
			}

			// make output table accumulator []string row
			row[1] = strconv.Itoa(modelDef.Table[idx].Acc[j].TableId)
			row[2] = strconv.Itoa(modelDef.Table[idx].Acc[j].AccId)
			row[3] = modelDef.Table[idx].Acc[j].Name
			row[4] = strconv.FormatBool(modelDef.Table[idx].Acc[j].IsDerived)
			row[5] = modelDef.Table[idx].Acc[j].SrcAcc
			j++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "table_acc.csv", err)
	}

	// write output tables expression rows into csv
	row = make([]string, 6)
	row[0] = strconv.Itoa(modelDef.Model.ModelId)

	idx = 0
	j = 0
	err = toCsvFile(
		outDir,
		"table_expr.csv",
		[]string{"model_id", "model_table_id", "expr_id", "expr_name", "expr_decimals", "expr_src"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(modelDef.Table) { // end of output table rows
				return true, row, nil
			}

			// if end of current output tables expressions then find next output table with expression list
			if j < 0 || j >= len(modelDef.Table[idx].Expr) {
				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(modelDef.Table) { // end of output table rows
						return true, row, nil
					}
					if len(modelDef.Table[idx].Expr) > 0 {
						break
					}
				}
			}

			// make output table expression []string row
			row[1] = strconv.Itoa(modelDef.Table[idx].Expr[j].TableId)
			row[2] = strconv.Itoa(modelDef.Table[idx].Expr[j].ExprId)
			row[3] = modelDef.Table[idx].Expr[j].Name
			row[4] = strconv.Itoa(modelDef.Table[idx].Expr[j].Decimals)
			row[5] = modelDef.Table[idx].Expr[j].SrcExpr
			j++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "table_expr.csv", err)
	}

	// write model entity rows into csv
	row = make([]string, 5)
	row[0] = strconv.Itoa(modelDef.Model.ModelId)

	idx = 0
	err = toCsvFile(
		outDir,
		"entity_dic.csv",
		[]string{
			"model_id", "model_entity_id", "entity_hid", "entity_name", "entity_digest"},
		func() (bool, []string, error) {
			if 0 <= idx && idx < len(modelDef.Entity) {
				row[1] = strconv.Itoa(modelDef.Entity[idx].EntityId)
				row[2] = strconv.Itoa(modelDef.Entity[idx].EntityHid)
				row[3] = modelDef.Entity[idx].Name
				row[4] = modelDef.Entity[idx].Digest
				idx++
				return false, row, nil
			}
			return true, row, nil // end of model entity rows
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "entity_dic.csv", err)
	}

	// write entity attribute rows into csv
	row = make([]string, 6)
	row[0] = strconv.Itoa(modelDef.Model.ModelId)

	idx = 0
	j = 0
	err = toCsvFile(
		outDir,
		"entity_attr.csv",
		[]string{"model_id", "model_entity_id", "attr_id", "attr_name", "model_type_id", "is_internal"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(modelDef.Entity) { // end of entity rows
				return true, row, nil
			}

			// if end of current entity attributes then find next entity with attribute list
			if j < 0 || j >= len(modelDef.Entity[idx].Attr) {
				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(modelDef.Entity) { // end of entity rows
						return true, row, nil
					}
					if len(modelDef.Entity[idx].Attr) > 0 {
						break
					}
				}
			}

			// make entity attribute []string row
			row[1] = strconv.Itoa(modelDef.Entity[idx].Attr[j].EntityId)
			row[2] = strconv.Itoa(modelDef.Entity[idx].Attr[j].AttrId)
			row[3] = modelDef.Entity[idx].Attr[j].Name
			row[4] = strconv.Itoa(modelDef.Entity[idx].Attr[j].TypeId)
			row[5] = strconv.FormatBool(modelDef.Entity[idx].Attr[j].IsInternal)
			j++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "entity_attr.csv", err)
	}

	// write model group rows into csv
	row = make([]string, 5)
	row[0] = strconv.Itoa(modelDef.Model.ModelId)

	idx = 0
	err = toCsvFile(
		outDir,
		"group_lst.csv",
		[]string{"model_id", "group_id", "is_parameter", "group_name", "is_hidden"},
		func() (bool, []string, error) {
			if 0 <= idx && idx < len(modelDef.Group) {
				row[1] = strconv.Itoa(modelDef.Group[idx].GroupId)
				row[2] = strconv.FormatBool(modelDef.Group[idx].IsParam)
				row[3] = modelDef.Group[idx].Name
				row[4] = strconv.FormatBool(modelDef.Group[idx].IsHidden)
				idx++
				return false, row, nil
			}
			return true, row, nil // end of model group rows
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "group_lst.csv", err)
	}

	// write group parent-child rows into csv
	row = make([]string, 5)
	row[0] = strconv.Itoa(modelDef.Model.ModelId)

	idx = 0
	j = 0
	err = toCsvFile(
		outDir,
		"group_pc.csv",
		[]string{"model_id", "group_id", "child_pos", "child_group_id", "leaf_id"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(modelDef.Group) { // end of groups rows
				return true, row, nil
			}

			// if end of current group children rows then find next group with parent-child list
			if j < 0 || j >= len(modelDef.Group[idx].GroupPc) {
				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(modelDef.Group) { // end of groups rows
						return true, row, nil
					}
					if len(modelDef.Group[idx].GroupPc) > 0 {
						break
					}
				}
			}

			// make group parnet-child []string row
			row[1] = strconv.Itoa(modelDef.Group[idx].GroupPc[j].GroupId)
			row[2] = strconv.Itoa(modelDef.Group[idx].GroupPc[j].ChildPos)

			if modelDef.Group[idx].GroupPc[j].ChildGroupId < 0 { // negative value is NULL
				row[3] = "NULL"
			} else {
				row[3] = strconv.Itoa(modelDef.Group[idx].GroupPc[j].ChildGroupId)
			}
			if modelDef.Group[idx].GroupPc[j].ChildLeafId < 0 { // negative value is NULL
				row[4] = "NULL"
			} else {
				row[4] = strconv.Itoa(modelDef.Group[idx].GroupPc[j].ChildLeafId)
			}
			j++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "group_pc.csv", err)
	}

	// write model entity group rows into csv
	row = make([]string, 5)
	row[0] = strconv.Itoa(modelDef.Model.ModelId)

	idx = 0
	err = toCsvFile(
		outDir,
		"entity_group_lst.csv",
		[]string{"model_id", "model_entity_id", "group_id", "group_name", "is_hidden"},
		func() (bool, []string, error) {
			if 0 <= idx && idx < len(modelDef.EntityGroup) {
				row[1] = strconv.Itoa(modelDef.EntityGroup[idx].EntityId)
				row[2] = strconv.Itoa(modelDef.EntityGroup[idx].GroupId)
				row[3] = modelDef.EntityGroup[idx].Name
				row[4] = strconv.FormatBool(modelDef.EntityGroup[idx].IsHidden)
				idx++
				return false, row, nil
			}
			return true, row, nil // end of model group rows
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "entity_group_lst.csv", err)
	}

	// write entity group parent-child rows into csv
	row = make([]string, 6)
	row[0] = strconv.Itoa(modelDef.Model.ModelId)

	idx = 0
	j = 0
	err = toCsvFile(
		outDir,
		"entity_group_pc.csv",
		[]string{"model_id", "model_entity_id", "group_id", "child_pos", "child_group_id", "attr_id"},
		func() (bool, []string, error) {

			if idx < 0 || idx >= len(modelDef.EntityGroup) { // end of entity groups rows
				return true, row, nil
			}

			// if end of current entity group children rows then find next group with parent-child list
			if j < 0 || j >= len(modelDef.EntityGroup[idx].GroupPc) {
				j = 0
				for {
					idx++
					if idx < 0 || idx >= len(modelDef.EntityGroup) { // end of entity groups rows
						return true, row, nil
					}
					if len(modelDef.EntityGroup[idx].GroupPc) > 0 {
						break
					}
				}
			}

			// make entity group parnet-child []string row
			row[1] = strconv.Itoa(modelDef.EntityGroup[idx].GroupPc[j].EntityId)
			row[2] = strconv.Itoa(modelDef.EntityGroup[idx].GroupPc[j].GroupId)
			row[3] = strconv.Itoa(modelDef.EntityGroup[idx].GroupPc[j].ChildPos)

			if modelDef.EntityGroup[idx].GroupPc[j].ChildGroupId < 0 { // negative value is NULL
				row[4] = "NULL"
			} else {
				row[4] = strconv.Itoa(modelDef.EntityGroup[idx].GroupPc[j].ChildGroupId)
			}
			if modelDef.EntityGroup[idx].GroupPc[j].AttrId < 0 { // negative value is NULL
				row[5] = "NULL"
			} else {
				row[5] = strconv.Itoa(modelDef.EntityGroup[idx].GroupPc[j].AttrId)
			}
			j++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "entity_group_pc.csv", err)
	}

	return nil
}

// toCsvFile write into csvDir/fileName.csv or into csvDir/fileName.tsv file.
// if isTsv is true then output into TSV else into CSV
func toCsvFile(
	csvDir string, fileName string, columnNames []string, lineCvt lineCsvConverter) error {

	// create csv file
	if theCfg.isTsv && strings.HasSuffix(fileName, ".csv") {
		fileName = fileName[:len(fileName)-4] + ".tsv"
	}
	f, err := os.OpenFile(filepath.Join(csvDir, fileName), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if theCfg.isWriteUtf8Bom { // if required then write utf-8 bom
		if _, err = f.Write(helper.Utf8bom); err != nil {
			return err
		}
	}

	wr := csv.NewWriter(f)
	if theCfg.isTsv {
		wr.Comma = '\t'
	}

	// write header line: column names, if provided
	if len(columnNames) > 0 {
		if err = wr.Write(columnNames); err != nil {
			return err
		}
	}

	// write csv lines until eof
	for {
		isEof, row, err := lineCvt()
		if err != nil {
			return err
		}
		if isEof {
			break
		}
		if err = wr.Write(row); err != nil {
			return err
		}
	}

	// flush and return error, if any
	wr.Flush()
	return wr.Error()
}
