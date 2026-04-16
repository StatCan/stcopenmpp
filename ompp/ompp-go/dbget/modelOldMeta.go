// Copyright OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package main

import (
	"database/sql"
	"path/filepath"
	"strconv"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
	"github.com/stcopenmpp/ompp-go/ompp/omppLog"
)

// LanguageDic compatibility view
type oldLanguageDic struct {
	LanguageID   int
	LanguageCode string
	LanguageName string
	All          string
	Min          string
	Max          string
}

// ModelDic compatibility view
type oldModelDic struct {
	Name        string
	Description string
	Note        string
	ModelType   int
	Version     string
	LanguageID  int
}

// ModelInfoDic compatibility view
type oldModelInfoDic struct {
	Time             string
	Directory        string
	CommandLine      string
	CompletionStatus string
	Subsamples       int
	CV               int
	SE               int
	ModelType        int
	FullReport       int
	Cases            int // option Parameter.Cases
	CasesRequested   int // option Parameter.Cases
	LanguageID       int
}

type oldSimulationInfoDic oldModelInfoDic // SimulationInfoDic compatibility view

// ScenarioDic
type oldScenarioDic struct {
	Name              string
	Description       string
	Note              string
	Subsamples        int
	Cases             int // option Parameter.Cases
	Seed              int // option Parameter.Seed
	PopulationScaling int // option Parameter.PopulationScaling
	PopulationSize    int // option Parameter.Population
	CopyParameters    int
	LanguageID        int
}

// TypeDic compatibility view
type oldTypeDic struct {
	TypeID int
	DicID  int
}

// SimpleTypeDic compatibility view
type oldSimpleTypeDic struct {
	TypeID int
	Name   string
}

// LogicalDic compatibility view
type oldLogicalDic struct {
	TypeID           int
	Name             string
	Value            int
	ValueName        string
	ValueDescription string
	LanguageID       int
}

// ClassificationDic compatibility view
type oldClassificationDic struct {
	TypeID         int
	Name           string
	Description    string
	Note           string
	NumberOfValues int
	LanguageID     int
}

// ClassificationValueDic compatibility view
type oldClassificationValueDic struct {
	TypeID      int
	EnumValue   int
	Name        string
	Description string
	Note        string
	LanguageID  int
}

// RangeDic compatibility view
type oldRangeDic struct {
	TypeID      int
	Name        string
	Description string
	Note        string
	Min         int // MIN(EN.enum_name)
	Max         int // MAX(EX.enum_name)
	LanguageID  int
}

// RangeValueDic compatibility view
type oldRangeValueDic struct {
	TypeID int
	Value  int // enum_name
}

// PartitionDic compatibility view
type oldPartitionDic struct {
	TypeID         int
	Name           string
	Description    string
	Note           string
	NumberOfValues int
	LanguageID     int
}

// PartitionValueDic compatibility view
type oldPartitionValueDic struct {
	TypeID      int
	Position    int
	Value       int // enum_name
	StringValue string
}

// PartitionIntervalDic compatibility view
type oldPartitionIntervalDic struct {
	TypeID      int
	Position    int
	Description string
	LanguageID  int
}

// ParameterDic compatibility view
type oldParameterDic struct {
	ParameterID                 int
	Name                        string
	Description                 string
	Note                        string
	ValueNote                   string
	TypeID                      int
	Rank                        int
	NumberOfCumulatedDimensions int
	ModelGenerated              int
	Hidden                      bool
	LanguageID                  int
}

// ParameterDimensionDic compatibility view
type oldParameterDimensionDic struct {
	ParameterID     int
	DisplayPosition int
	TypeID          int
	Position        int
}

// ParameterGroupDic compatibility view
type oldParameterGroupDic struct {
	ParameterGroupID int
	Name             string
	Description      string
	Note             string
	ModelGenerated   int
	Hidden           bool
	LanguageID       int
}

// ParameterGroupMemberDic compatibility view
type oldParameterGroupMemberDic struct {
	ParameterGroupID int
	Position         int
	ParameterID      *int64
	MemberGroupID    *int64
}

// TableDic compatibility view
type oldTableDic struct {
	TableID                      int
	Name                         string
	Description                  string
	Note                         string
	Rank                         int
	AnalysisDimensionPosition    int
	AnalysisDimensionName        string // table_name.DimA, must be TableName.Dim + "Rank-1"
	AnalysisDimensionDescription string
	AnalysisDimensionNote        string
	Sparse                       bool
	Hidden                       bool
	LanguageID                   int
}

type oldUserTableDic oldTableDic // UserTableDic

// TableClassDic compatibility view
type oldTableClassDic struct {
	TableID     int
	Position    int
	Name        string // table_name.dim0, must be TableName.Dim + Position
	Description string
	Note        string
	TypeID      int
	Totals      bool
	LanguageID  int
}

// TableExpressionDic compatibility view
type oldTableExpressionDic struct {
	TableID      int
	ExpressionID int
	Name         string
	Description  string
	Note         string
	Decimals     int
	LanguageID   int
}

// TableGroupDic compatibility view
type oldTableGroupDic struct {
	TableGroupID int
	Name         string
	Description  string
	Note         string
	Hidden       bool
	LanguageID   int
}

// TableGroupMemberDic compatibility view
type oldTableGroupMemberDic struct {
	TableGroupID  int
	Position      int
	TableID       *int64
	MemberGroupID *int64
}

// write old compatibilty model metada from database into text csv, tsv or json file
func modelOldMeta(srcDb *sql.DB, modelId int) error {

	// get model row, it should exists if model id still valid
	mdRow, err := db.GetModelRow(srcDb, modelId)
	if err != nil {
		return helper.ErrorNew("Error at get model metadata by id:", modelId, ":", err)
	}
	if mdRow == nil {
		return helper.ErrorNew("Error at get model row by id:", modelId)
	}

	// for json use specified file name or make default as modelName.old-model.json
	// for csv use specified directory or make default as modelName.old-model
	fp := ""
	dir := theCfg.dir
	ext := extByKind()

	if theCfg.isConsole {
		omppLog.Log("Do", theCfg.action, mdRow.Name)
	} else {
		if theCfg.kind == asJson {

			fp = theCfg.fileName
			if fp == "" {
				fp = helper.CleanFileName(mdRow.Name) + ".old-model.json"
			}
			fp = filepath.Join(theCfg.dir, fp)

			omppLog.Log("Do", theCfg.action, ":", fp)

		} else {
			if dir == "" {
				dir = mdRow.Name + ".old-model"
			}
			// remove output directory if required, create output directory if not already exists
			if err := makeOutputDir(dir, theCfg.isKeepOutputDir); err != nil {
				return err
			}
			omppLog.Log("Do", theCfg.action, ":", dir)
		}
	}

	// get model metadata from compatibility views
	mcv := struct {
		LanguageDic             []oldLanguageDic
		ModelDic                []oldModelDic
		ModelInfoDic            []oldModelInfoDic
		SimulationInfoDic       []oldSimulationInfoDic
		ScenarioDic             []oldScenarioDic
		TypeDic                 []oldTypeDic
		SimpleTypeDic           []oldSimpleTypeDic
		LogicalDic              []oldLogicalDic
		ClassificationDic       []oldClassificationDic
		ClassificationValueDic  []oldClassificationValueDic
		RangeDic                []oldRangeDic
		RangeValueDic           []oldRangeValueDic
		PartitionDic            []oldPartitionDic
		PartitionValueDic       []oldPartitionValueDic
		PartitionIntervalDic    []oldPartitionIntervalDic
		ParameterDic            []oldParameterDic
		ParameterDimensionDic   []oldParameterDimensionDic
		ParameterGroupDic       []oldParameterGroupDic
		ParameterGroupMemberDic []oldParameterGroupMemberDic
		TableDic                []oldTableDic
		UserTableDic            []oldUserTableDic
		TableClassDic           []oldTableClassDic
		TableExpressionDic      []oldTableExpressionDic
		TableGroupDic           []oldTableGroupDic
		TableGroupMemberDic     []oldTableGroupMemberDic
	}{
		LanguageDic:             []oldLanguageDic{},
		ModelDic:                []oldModelDic{},
		ModelInfoDic:            []oldModelInfoDic{},
		SimulationInfoDic:       []oldSimulationInfoDic{},
		ScenarioDic:             []oldScenarioDic{},
		TypeDic:                 []oldTypeDic{},
		SimpleTypeDic:           []oldSimpleTypeDic{},
		LogicalDic:              []oldLogicalDic{},
		ClassificationDic:       []oldClassificationDic{},
		ClassificationValueDic:  []oldClassificationValueDic{},
		RangeDic:                []oldRangeDic{},
		RangeValueDic:           []oldRangeValueDic{},
		PartitionDic:            []oldPartitionDic{},
		PartitionValueDic:       []oldPartitionValueDic{},
		PartitionIntervalDic:    []oldPartitionIntervalDic{},
		ParameterDic:            []oldParameterDic{},
		ParameterDimensionDic:   []oldParameterDimensionDic{},
		ParameterGroupDic:       []oldParameterGroupDic{},
		ParameterGroupMemberDic: []oldParameterGroupMemberDic{},
		TableDic:                []oldTableDic{},
		UserTableDic:            []oldUserTableDic{},
		TableClassDic:           []oldTableClassDic{},
		TableExpressionDic:      []oldTableExpressionDic{},
		TableGroupDic:           []oldTableGroupDic{},
		TableGroupMemberDic:     []oldTableGroupMemberDic{},
	}

	// select language or all languages and create language filter by user language
	// do not use view becasue All, Min, Max are sql reserved keywords
	langIdCode := map[int]string{}

	q := "SELECT" +
		" L.lang_id," +
		" L.lang_code," +
		" L.lang_name," +
		" (SELECT LWA.word_value FROM lang_word LWA WHERE LWA.lang_id = L.lang_id AND LWA.word_code = 'all')," +
		" (SELECT LWN.word_value FROM lang_word LWN WHERE LWN.lang_id = L.lang_id AND LWN.word_code = 'min')," +
		" (SELECT LWX.word_value FROM lang_word LWX WHERE LWX.lang_id = L.lang_id AND LWX.word_code = 'max')" +
		" FROM lang_lst L"
	if !theCfg.isNoLang {
		q += " WHERE L.lang_code = " + db.ToQuoted(theCfg.lang)
	}
	q += " ORDER BY 1"

	err = db.SelectRows(srcDb,
		q,
		func(rows *sql.Rows) error {
			var all, min, max sql.NullString
			var r oldLanguageDic
			if e := rows.Scan(&r.LanguageID, &r.LanguageCode, &r.LanguageName, &all, &min, &max); e != nil {
				return e
			}
			langIdCode[r.LanguageID] = r.LanguageCode // collect languages to use it for notes
			r.All = ""
			if all.Valid {
				r.All = all.String
			}
			r.Min = ""
			if min.Valid {
				r.Min = min.String
			}
			r.Max = ""
			if max.Valid {
				r.Max = max.String
			}
			mcv.LanguageDic = append(mcv.LanguageDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	langFlt := ""
	if !theCfg.isNoLang {

		if len(mcv.LanguageDic) <= 0 {
			return helper.ErrorNew("Error at get model language:", theCfg.lang)
		}
		langFlt = "M.LanguageID = " + strconv.Itoa(mcv.LanguageDic[0].LanguageID)
	}

	// at least one ModelDic row must exists
	q = "SELECT" +
		" M.Name, M.Description, M.Note, M.ModelType, M.Version, M.LanguageID" +
		" FROM ModelDic M"
	if langFlt != "" {
		q += " WHERE " + langFlt
	}
	q += " ORDER BY M.Name, M.LanguageID"

	err = db.SelectRows(srcDb,
		q,
		func(rows *sql.Rows) error {
			var note sql.NullString
			var r oldModelDic
			if e := rows.Scan(&r.Name, &r.Description, &note, &r.ModelType, &r.Version, &r.LanguageID); e != nil {
				return e
			}
			r.Note = ""
			if note.Valid {
				r.Note = note.String
			}
			mcv.ModelDic = append(mcv.ModelDic, r)
			return nil
		})
	if err != nil {
		return err
	}
	if len(mcv.ModelDic) <= 0 {
		return helper.ErrorFmt("ModelDic rows not found, model id: %d language: %s", modelId, theCfg.lang)
	}
	// ModelInfoDic and SimulationInfoDic: convert Cases from run_option string to int
	q = "SELECT" +
		" M.Time,       M.Directory, M.CommandLine,    M.CompletionStatus," +
		" M.Subsamples, M.CV,        M.SE,             M.ModelType," +
		" M.FullReport, M.Cases,     M.CasesRequested, M.LanguageID" +
		" FROM ModelInfoDic M"
	if langFlt != "" {
		q += " WHERE " + langFlt
	}
	q += " ORDER BY M.Time, M.LanguageID"

	err = db.SelectRows(srcDb,
		q,
		func(rows *sql.Rows) error {
			var c, cr sql.NullString
			var r oldModelInfoDic
			if e := rows.Scan(
				&r.Time, &r.Directory, &r.CommandLine, &r.CompletionStatus, &r.Subsamples, &r.CV, &r.SE, &r.ModelType, &r.FullReport, &c, &cr, &r.LanguageID); e != nil {
				return e
			}
			r.Cases = 0
			if c.Valid {
				if n, e := strconv.Atoi(c.String); e == nil {
					r.Cases = n
				}
			}
			r.CasesRequested = 0
			if cr.Valid {
				if n, e := strconv.Atoi(cr.String); e == nil {
					r.CasesRequested = n
				}
			}
			mcv.ModelInfoDic = append(mcv.ModelInfoDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	q = "SELECT" +
		" M.Time,       M.Directory, M.CommandLine,    M.CompletionStatus," +
		" M.Subsamples, M.CV,        M.SE,             M.ModelType," +
		" M.FullReport, M.Cases,     M.CasesRequested, M.LanguageID" +
		" FROM SimulationInfoDic M"
	if langFlt != "" {
		q += " WHERE " + langFlt
	}
	q += " ORDER BY M.Time, M.LanguageID"

	err = db.SelectRows(srcDb,
		q,
		func(rows *sql.Rows) error {
			var c, cr sql.NullString
			var r oldSimulationInfoDic
			if e := rows.Scan(
				&r.Time, &r.Directory, &r.CommandLine, &r.CompletionStatus, &r.Subsamples, &r.CV, &r.SE, &r.ModelType, &r.FullReport, &c, &cr, &r.LanguageID); e != nil {
				return e
			}
			r.Cases = 0
			if c.Valid {
				if n, e := strconv.Atoi(c.String); e == nil {
					r.Cases = n
				}
			}
			r.CasesRequested = 0
			if cr.Valid {
				if n, e := strconv.Atoi(cr.String); e == nil {
					r.CasesRequested = n
				}
			}
			mcv.SimulationInfoDic = append(mcv.SimulationInfoDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	// ScenarioDic: convert Cases, Seed, Population scaling and size from run_option string to int
	q = "SELECT" +
		" M.Name,           M.Description, M.Note,              M.Subsamples," +
		" M.Cases,          M.Seed,        M.PopulationScaling, M.PopulationSize," +
		" M.CopyParameters, M.LanguageID" +
		" FROM ScenarioDic M"
	if langFlt != "" {
		q += " WHERE " + langFlt
	}
	q += " ORDER BY M.Name, M.LanguageID"

	err = db.SelectRows(srcDb,
		q,
		func(rows *sql.Rows) error {
			var note, c, seed, sc, p sql.NullString
			var r oldScenarioDic
			if e := rows.Scan(
				&r.Name, &r.Description, &note, &r.Subsamples, &c, &seed, &sc, &p, &r.CopyParameters, &r.LanguageID); e != nil {
				return e
			}
			r.Note = ""
			if note.Valid {
				r.Note = note.String
			}
			r.Cases = 0
			if c.Valid {
				if n, e := strconv.Atoi(c.String); e == nil {
					r.Cases = n
				}
			}
			r.Seed = 0
			if seed.Valid {
				if n, e := strconv.Atoi(seed.String); e == nil {
					r.Seed = n
				}
			}
			r.PopulationScaling = 0
			if sc.Valid {
				if n, e := strconv.Atoi(sc.String); e == nil {
					r.PopulationScaling = n
				}
			}
			r.PopulationSize = 0
			if p.Valid {
				if n, e := strconv.Atoi(p.String); e == nil {
					r.PopulationSize = n
				}
			}
			mcv.ScenarioDic = append(mcv.ScenarioDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	// TypeDic compatibility views: types and enums
	err = db.SelectRows(srcDb,
		"SELECT M.TypeID, M.DicID FROM TypeDic M ORDER BY 1",
		func(rows *sql.Rows) error {
			var r oldTypeDic
			if e := rows.Scan(&r.TypeID, &r.DicID); e != nil {
				return e
			}
			mcv.TypeDic = append(mcv.TypeDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	err = db.SelectRows(srcDb,
		"SELECT M.TypeID, M.Name FROM SimpleTypeDic M ORDER BY 1",
		func(rows *sql.Rows) error {
			var r oldSimpleTypeDic
			if e := rows.Scan(&r.TypeID, &r.Name); e != nil {
				return e
			}
			mcv.SimpleTypeDic = append(mcv.SimpleTypeDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	q = "SELECT" +
		" M.TypeID, M.Name, M.Value, M.ValueName, M.ValueDescription, M.LanguageID" +
		" FROM LogicalDic M"
	if langFlt != "" {
		q += " WHERE " + langFlt
	}
	q += " ORDER BY M.TypeID, M.LanguageID"

	err = db.SelectRows(srcDb,
		q,
		func(rows *sql.Rows) error {
			var r oldLogicalDic
			if e := rows.Scan(
				&r.TypeID, &r.Name, &r.Value, &r.ValueName, &r.ValueDescription, &r.LanguageID); e != nil {
				return e
			}
			mcv.LogicalDic = append(mcv.LogicalDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	q = "SELECT" +
		" M.TypeID, M.Name, M.Description, M.Note, M.NumberOfValues, M.LanguageID" +
		" FROM ClassificationDic M"
	if langFlt != "" {
		q += " WHERE " + langFlt
	}
	q += " ORDER BY M.TypeID, M.LanguageID"

	err = db.SelectRows(srcDb,
		q,
		func(rows *sql.Rows) error {
			var note sql.NullString
			var r oldClassificationDic
			if e := rows.Scan(
				&r.TypeID, &r.Name, &r.Description, &note, &r.NumberOfValues, &r.LanguageID); e != nil {
				return e
			}
			r.Note = ""
			if note.Valid {
				r.Note = note.String
			}
			mcv.ClassificationDic = append(mcv.ClassificationDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	q = "SELECT" +
		" M.TypeID, M.EnumValue, M.Name, M.Description, M.Note, M.LanguageID" +
		" FROM ClassificationValueDic M"
	if langFlt != "" {
		q += " WHERE " + langFlt
	}
	q += " ORDER BY M.TypeID, M.EnumValue, M.LanguageID"

	err = db.SelectRows(srcDb,
		q,
		func(rows *sql.Rows) error {
			var note sql.NullString
			var r oldClassificationValueDic
			if e := rows.Scan(
				&r.TypeID, &r.EnumValue, &r.Name, &r.Description, &note, &r.LanguageID); e != nil {
				return e
			}
			r.Note = ""
			if note.Valid {
				r.Note = note.String
			}
			mcv.ClassificationValueDic = append(mcv.ClassificationValueDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	// RangeDic: convert Min and Max from enum_name string to int
	q = "SELECT" +
		" M.TypeID, M.Name, M.Description, M.Note, M.Min, M.Max, M.LanguageID" +
		" FROM RangeDic M"
	if langFlt != "" {
		q += " WHERE " + langFlt
	}
	q += " ORDER BY M.TypeID, M.LanguageID"

	err = db.SelectRows(srcDb,
		q,
		func(rows *sql.Rows) error {
			var note, min, max sql.NullString
			var r oldRangeDic
			if e := rows.Scan(
				&r.TypeID, &r.Name, &r.Description, &note, &min, &max, &r.LanguageID); e != nil {
				return e
			}
			r.Note = ""
			if note.Valid {
				r.Note = note.String
			}
			r.Min = 0
			if min.Valid {
				if n, e := strconv.Atoi(min.String); e == nil {
					r.Min = n
				}
			}
			r.Max = 0
			if max.Valid {
				if n, e := strconv.Atoi(max.String); e == nil {
					r.Max = n
				}
			}
			mcv.RangeDic = append(mcv.RangeDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	// RangeValueDic: convert Value from enum_name string to int
	err = db.SelectRows(srcDb,
		"SELECT M.TypeID, M.Value FROM RangeValueDic M ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var val string
			var r oldRangeValueDic
			if e := rows.Scan(
				&r.TypeID, &val); e != nil {
				return e
			}
			r.Value = 0
			if n, e := strconv.Atoi(val); e == nil {
				r.Value = n
			}
			mcv.RangeValueDic = append(mcv.RangeValueDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	// PartitionDic compatibility views
	q = "SELECT" +
		" M.TypeID, M.Name, M.Description, M.Note, M.NumberOfValues, M.LanguageID" +
		" FROM PartitionDic M"
	if langFlt != "" {
		q += " WHERE " + langFlt
	}
	q += " ORDER BY M.TypeID, M.LanguageID"

	err = db.SelectRows(srcDb,
		q,
		func(rows *sql.Rows) error {
			var note sql.NullString
			var r oldPartitionDic
			if e := rows.Scan(
				&r.TypeID, &r.Name, &r.Description, &note, &r.NumberOfValues, &r.LanguageID); e != nil {
				return e
			}
			r.Note = ""
			if note.Valid {
				r.Note = note.String
			}
			mcv.PartitionDic = append(mcv.PartitionDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	// PartitionValueDic: convert Value from enum_name string to int
	err = db.SelectRows(srcDb,
		"SELECT M.TypeID, M.Position, M.Value, M.StringValue FROM PartitionValueDic M ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var val string
			var r oldPartitionValueDic
			if e := rows.Scan(
				&r.TypeID, &r.Position, &val, &r.StringValue); e != nil {
				return e
			}
			r.Value = 0
			if n, e := strconv.Atoi(val); e == nil {
				r.Value = n
			}
			mcv.PartitionValueDic = append(mcv.PartitionValueDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	err = db.SelectRows(srcDb,
		"SELECT M.TypeID, M.Position, M.Description, M.LanguageID FROM PartitionIntervalDic M ORDER BY 1, 2, 4",
		func(rows *sql.Rows) error {
			var r oldPartitionIntervalDic
			if e := rows.Scan(
				&r.TypeID, &r.Position, &r.Description, &r.LanguageID); e != nil {
				return e
			}
			mcv.PartitionIntervalDic = append(mcv.PartitionIntervalDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	// ParameterDic compatibility views
	q = "SELECT" +
		" M.ParameterID,    M.Name,   M.Description, M.Note," +
		" M.ValueNote,      M.TypeID, M.Rank,        M.NumberOfCumulatedDimensions," +
		" M.ModelGenerated, M.Hidden, M.LanguageID" +
		" FROM ParameterDic M"
	if langFlt != "" {
		q += " WHERE " + langFlt
	}
	q += " ORDER BY M.ParameterID, M.LanguageID"

	err = db.SelectRows(srcDb,
		q,
		func(rows *sql.Rows) error {
			var note, vn sql.NullString
			var nh int
			var r oldParameterDic
			if e := rows.Scan(
				&r.ParameterID, &r.Name, &r.Description, &note, &vn, &r.TypeID, &r.Rank, &r.NumberOfCumulatedDimensions, &r.ModelGenerated, &nh, &r.LanguageID); e != nil {
				return e
			}
			r.Hidden = nh != 0
			r.Note = ""
			if note.Valid {
				r.Note = note.String
			}
			r.ValueNote = ""
			if vn.Valid {
				r.ValueNote = note.String
			}
			mcv.ParameterDic = append(mcv.ParameterDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	err = db.SelectRows(srcDb,
		"SELECT M.ParameterID, M.DisplayPosition, M.TypeID, M.Position FROM ParameterDimensionDic M ORDER BY M.ParameterID, M.Position",
		func(rows *sql.Rows) error {
			var r oldParameterDimensionDic
			if e := rows.Scan(
				&r.ParameterID, &r.DisplayPosition, &r.TypeID, &r.Position); e != nil {
				return e
			}
			mcv.ParameterDimensionDic = append(mcv.ParameterDimensionDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	// parameter groups compatibility views
	q = "SELECT" +
		" M.ParameterGroupID, M.Name, M.Description, M.Note, M.ModelGenerated, M.Hidden, M.LanguageID" +
		" FROM ParameterGroupDic M"
	if langFlt != "" {
		q += " WHERE " + langFlt
	}
	q += " ORDER BY M.ParameterGroupID, M.LanguageID"

	err = db.SelectRows(srcDb,
		q,
		func(rows *sql.Rows) error {
			var note sql.NullString
			var nh int
			var r oldParameterGroupDic
			if e := rows.Scan(
				&r.ParameterGroupID, &r.Name, &r.Description, &note, &r.ModelGenerated, &nh, &r.LanguageID); e != nil {
				return e
			}
			r.Hidden = nh != 0
			r.Note = ""
			if note.Valid {
				r.Note = note.String
			}
			mcv.ParameterGroupDic = append(mcv.ParameterGroupDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	err = db.SelectRows(srcDb,
		"SELECT M.ParameterGroupID, M.Position, M.ParameterID, M.MemberGroupID FROM ParameterGroupMemberDic M ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var leaf, grp sql.NullInt64
			var r oldParameterGroupMemberDic
			if e := rows.Scan(
				&r.ParameterGroupID, &r.Position, &leaf, &grp); e != nil {
				return e
			}
			r.ParameterID = nil
			if leaf.Valid {
				r.ParameterID = &leaf.Int64
			}
			r.MemberGroupID = nil
			if grp.Valid {
				r.MemberGroupID = &grp.Int64
			}
			mcv.ParameterGroupMemberDic = append(mcv.ParameterGroupMemberDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	// TableDic and UserTableDic: make AnalysisDimensionName as TableName.Dim + "Rank - 1"
	tblIdName := map[int]string{}
	q = "SELECT" +
		" M.TableID,               M.Name,                      M.Description,           M.Note," +
		" M.Rank,                  M.AnalysisDimensionPosition, M.AnalysisDimensionName, M.AnalysisDimensionDescription," +
		" M.AnalysisDimensionNote, M.Sparse,                    M.Hidden,                M.LanguageID" +
		" FROM TableDic M"
	if langFlt != "" {
		q += " WHERE " + langFlt
	}
	q += " ORDER BY M.TableID, M.LanguageID"

	err = db.SelectRows(srcDb,
		q,
		func(rows *sql.Rows) error {
			var note, vn sql.NullString
			var nh, ns int
			var r oldTableDic
			if e := rows.Scan(
				&r.TableID, &r.Name, &r.Description, &note,
				&r.Rank, &r.AnalysisDimensionPosition, &r.AnalysisDimensionName,
				&r.AnalysisDimensionDescription, &vn, &ns, &nh, &r.LanguageID); e != nil {
				return e
			}
			r.Hidden = nh != 0
			r.Sparse = ns != 0
			r.Note = ""
			if note.Valid {
				r.Note = note.String
			}
			r.AnalysisDimensionNote = ""
			if vn.Valid {
				r.AnalysisDimensionNote = vn.String
			}
			r.AnalysisDimensionName = r.Name + ".Dim" + strconv.Itoa(r.Rank-1)
			mcv.TableDic = append(mcv.TableDic, r)
			tblIdName[r.TableID] = r.Name
			return nil
		})
	if err != nil {
		return err
	}

	q = "SELECT" +
		" M.TableID,               M.Name,                      M.Description,           M.Note," +
		" M.Rank,                  M.AnalysisDimensionPosition, M.AnalysisDimensionName, M.AnalysisDimensionDescription," +
		" M.AnalysisDimensionNote, M.Sparse,                    M.Hidden,                M.LanguageID" +
		" FROM UserTableDic M"
	if langFlt != "" {
		q += " WHERE " + langFlt
	}
	q += " ORDER BY M.TableID, M.LanguageID"

	err = db.SelectRows(srcDb,
		q,
		func(rows *sql.Rows) error {
			var note, vn sql.NullString
			var nh, ns int
			var r oldUserTableDic
			if e := rows.Scan(
				&r.TableID, &r.Name, &r.Description, &note,
				&r.Rank, &r.AnalysisDimensionPosition, &r.AnalysisDimensionName,
				&r.AnalysisDimensionDescription, &vn, &ns, &nh, &r.LanguageID); e != nil {
				return e
			}
			r.Hidden = nh != 0
			r.Sparse = ns != 0
			r.Note = ""
			if note.Valid {
				r.Note = note.String
			}
			r.AnalysisDimensionNote = ""
			if vn.Valid {
				r.AnalysisDimensionNote = vn.String
			}
			r.AnalysisDimensionName = r.Name + ".Dim" + strconv.Itoa(r.Rank-1)
			mcv.UserTableDic = append(mcv.UserTableDic, r)
			tblIdName[r.TableID] = r.Name
			return nil
		})
	if err != nil {
		return err
	}

	q = "SELECT" +
		" M.TableID, M.Position, M.Name, M.Description, M.Note, M.TypeID, M.Totals, M.LanguageID" +
		" FROM TableClassDic M"
	if langFlt != "" {
		q += " WHERE " + langFlt
	}
	q += " ORDER BY M.TableID, M.Position, M.LanguageID"

	err = db.SelectRows(srcDb,
		q,
		func(rows *sql.Rows) error {
			var note sql.NullString
			var nt int
			var r oldTableClassDic
			if e := rows.Scan(
				&r.TableID, &r.Position, &r.Name, &r.Description, &note, &r.TypeID, &nt, &r.LanguageID); e != nil {
				return e
			}
			r.Name = tblIdName[r.TableID] + ".Dim" + strconv.Itoa(r.Position)
			r.Totals = nt != 0
			r.Note = ""
			if note.Valid {
				r.Note = note.String
			}
			mcv.TableClassDic = append(mcv.TableClassDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	q = "SELECT" +
		" M.TableID, M.ExpressionID, M.Name, M.Description, M.Note, M.Decimals, M.LanguageID" +
		" FROM TableExpressionDic M"
	if langFlt != "" {
		q += " WHERE " + langFlt
	}
	q += " ORDER BY M.TableID, M.ExpressionID, M.LanguageID"

	err = db.SelectRows(srcDb,
		q,
		func(rows *sql.Rows) error {
			var note sql.NullString
			var r oldTableExpressionDic
			if e := rows.Scan(
				&r.TableID, &r.ExpressionID, &r.Name, &r.Description, &note, &r.Decimals, &r.LanguageID); e != nil {
				return e
			}
			r.Note = ""
			if note.Valid {
				r.Note = note.String
			}
			mcv.TableExpressionDic = append(mcv.TableExpressionDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	// table groups compatibility views
	q = "SELECT" +
		" M.TableGroupID, M.Name, M.Description, M.Note, M.Hidden, M.LanguageID" +
		" FROM TableGroupDic M"
	if langFlt != "" {
		q += " WHERE " + langFlt
	}
	q += " ORDER BY M.TableGroupID, M.LanguageID"

	err = db.SelectRows(srcDb,
		q,
		func(rows *sql.Rows) error {
			var note sql.NullString
			var nh int
			var r oldTableGroupDic
			if e := rows.Scan(
				&r.TableGroupID, &r.Name, &r.Description, &note, &nh, &r.LanguageID); e != nil {
				return e
			}
			r.Hidden = nh != 0
			r.Note = ""
			if note.Valid {
				r.Note = note.String
			}
			mcv.TableGroupDic = append(mcv.TableGroupDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	err = db.SelectRows(srcDb,
		"SELECT M.TableGroupID, M.Position, M.TableID, M.MemberGroupID FROM TableGroupMemberDic M ORDER BY 1, 2",
		func(rows *sql.Rows) error {
			var leaf, grp sql.NullInt64
			var r oldTableGroupMemberDic
			if e := rows.Scan(
				&r.TableGroupID, &r.Position, &leaf, &grp); e != nil {
				return e
			}
			r.TableID = nil
			if leaf.Valid {
				r.TableID = &leaf.Int64
			}
			r.MemberGroupID = nil
			if grp.Valid {
				r.MemberGroupID = &grp.Int64
			}
			mcv.TableGroupMemberDic = append(mcv.TableGroupMemberDic, r)
			return nil
		})
	if err != nil {
		return err
	}

	// write json output into file or console
	if theCfg.kind == asJson {
		return toJsonOutput(fp, mcv) // save results
	}
	// else write csv or tsv output into file or console

	// make output path, return emtpy "" string to use console output
	outPath := func(name string) string {
		if theCfg.isConsole {
			return ""
		}
		return filepath.Join(dir, name+ext)
	}

	row := make([]string, 6)
	idx := 0
	err = toCsvOutput(
		outPath("LanguageDic"),
		[]string{"LanguageID", "LanguageCode", "LanguageName", "All", "Min", "Max"},
		func() (bool, []string, error) {
			if idx >= len(mcv.LanguageDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.LanguageDic[idx].LanguageID)
			row[1] = mcv.LanguageDic[idx].LanguageCode
			row[2] = mcv.LanguageDic[idx].LanguageName
			row[3] = mcv.LanguageDic[idx].All
			row[4] = mcv.LanguageDic[idx].Min
			row[5] = mcv.LanguageDic[idx].Max
			idx++
			return false, row, nil
		})
	if err != nil {
		helper.ErrorNew("failed to write into", "LanguageDic"+ext, err)
	}

	row = make([]string, 6)
	idx = 0
	err = toCsvOutput(
		outPath("ModelDic"),
		[]string{"Name", "Description", "Note", "ModelType", "Version", "LanguageID"},
		func() (bool, []string, error) {
			if idx >= len(mcv.ModelDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = mcv.ModelDic[idx].Name
			row[1] = mcv.ModelDic[idx].Description
			row[2] = "" // write notes into .md file
			row[3] = strconv.Itoa(mcv.ModelDic[idx].ModelType)
			row[4] = mcv.ModelDic[idx].Version
			row[5] = strconv.Itoa(mcv.ModelDic[idx].LanguageID)

			if e := writeNote(
				dir, "ModelDic."+mcv.ModelDic[idx].Name, langIdCode[mcv.ModelDic[idx].LanguageID], &mcv.ModelDic[idx].Note,
			); e != nil {
				return false, row, e
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "ModelDic"+ext, err)
	}

	row = make([]string, 12)
	idx = 0
	err = toCsvOutput(
		outPath("ModelInfoDic"),
		[]string{
			"Time", "Directory", "CommandLine", "CompletionStatus", "Subsamples", "CV", "SE", "ModelType", "FullReport", "Cases", "CasesRequested", "LanguageID",
		},
		func() (bool, []string, error) {
			if idx >= len(mcv.ModelInfoDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = mcv.ModelInfoDic[idx].Time
			row[1] = mcv.ModelInfoDic[idx].Directory
			row[2] = mcv.ModelInfoDic[idx].CommandLine
			row[3] = mcv.ModelInfoDic[idx].CompletionStatus
			row[4] = strconv.Itoa(mcv.ModelInfoDic[idx].Subsamples)
			row[5] = strconv.Itoa(mcv.ModelInfoDic[idx].CV)
			row[6] = strconv.Itoa(mcv.ModelInfoDic[idx].SE)
			row[7] = strconv.Itoa(mcv.ModelInfoDic[idx].ModelType)
			row[8] = strconv.Itoa(mcv.ModelInfoDic[idx].FullReport)
			row[9] = strconv.Itoa(mcv.ModelInfoDic[idx].Cases)
			row[10] = strconv.Itoa(mcv.ModelInfoDic[idx].CasesRequested)
			row[11] = strconv.Itoa(mcv.ModelInfoDic[idx].LanguageID)
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "ModelInfoDic"+ext, err)
	}

	row = make([]string, 12)
	idx = 0
	err = toCsvOutput(
		outPath("SimulationInfoDic"),
		[]string{
			"Time", "Directory", "CommandLine", "CompletionStatus", "Subsamples", "CV", "SE", "ModelType", "FullReport", "Cases", "CasesRequested", "LanguageID",
		},
		func() (bool, []string, error) {
			if idx >= len(mcv.SimulationInfoDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = mcv.SimulationInfoDic[idx].Time
			row[1] = mcv.SimulationInfoDic[idx].Directory
			row[2] = mcv.SimulationInfoDic[idx].CommandLine
			row[3] = mcv.SimulationInfoDic[idx].CompletionStatus
			row[4] = strconv.Itoa(mcv.SimulationInfoDic[idx].Subsamples)
			row[5] = strconv.Itoa(mcv.SimulationInfoDic[idx].CV)
			row[6] = strconv.Itoa(mcv.SimulationInfoDic[idx].SE)
			row[7] = strconv.Itoa(mcv.SimulationInfoDic[idx].ModelType)
			row[8] = strconv.Itoa(mcv.SimulationInfoDic[idx].FullReport)
			row[9] = strconv.Itoa(mcv.SimulationInfoDic[idx].Cases)
			row[10] = strconv.Itoa(mcv.SimulationInfoDic[idx].CasesRequested)
			row[11] = strconv.Itoa(mcv.SimulationInfoDic[idx].LanguageID)
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "SimulationInfoDic"+ext, err)
	}

	row = make([]string, 10)
	idx = 0
	err = toCsvOutput(
		outPath("ScenarioDic"),
		[]string{
			"Name", "Description", "Note", "Subsamples", "Cases", "Seed", "PopulationScaling", "PopulationSize", "CopyParameters", "LanguageID",
		},
		func() (bool, []string, error) {
			if idx >= len(mcv.ScenarioDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = mcv.ScenarioDic[idx].Name
			row[1] = mcv.ScenarioDic[idx].Description
			row[2] = ""
			row[3] = strconv.Itoa(mcv.ScenarioDic[idx].Subsamples)
			row[4] = strconv.Itoa(mcv.ScenarioDic[idx].Cases)
			row[5] = strconv.Itoa(mcv.ScenarioDic[idx].Seed)
			row[6] = strconv.Itoa(mcv.ScenarioDic[idx].PopulationScaling)
			row[7] = strconv.Itoa(mcv.ScenarioDic[idx].PopulationSize)
			row[8] = strconv.Itoa(mcv.ScenarioDic[idx].CopyParameters)
			row[9] = strconv.Itoa(mcv.ScenarioDic[idx].LanguageID)

			if e := writeNote(
				dir, "ScenarioDic."+mcv.ScenarioDic[idx].Name, langIdCode[mcv.ScenarioDic[idx].LanguageID], &mcv.ScenarioDic[idx].Note,
			); e != nil {
				return false, row, e
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "ScenarioDic"+ext, err)
	}

	row = make([]string, 2)
	idx = 0
	err = toCsvOutput(
		outPath("TypeDic"),
		[]string{"TypeID", "DicID"},
		func() (bool, []string, error) {
			if idx >= len(mcv.TypeDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.TypeDic[idx].TypeID)
			row[1] = strconv.Itoa(mcv.TypeDic[idx].DicID)
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "TypeDic"+ext, err)
	}

	row = make([]string, 2)
	idx = 0
	err = toCsvOutput(
		outPath("SimpleTypeDic"),
		[]string{"TypeID", "DicID"},
		func() (bool, []string, error) {
			if idx >= len(mcv.SimpleTypeDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.SimpleTypeDic[idx].TypeID)
			row[1] = mcv.SimpleTypeDic[idx].Name
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "SimpleTypeDic"+ext, err)
	}

	row = make([]string, 6)
	idx = 0
	err = toCsvOutput(
		outPath("LogicalDic"),
		[]string{"TypeID", "Name", "Value", "ValueName", "ValueDescription", "LanguageID"},
		func() (bool, []string, error) {
			if idx >= len(mcv.LogicalDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.LogicalDic[idx].TypeID)
			row[1] = mcv.LogicalDic[idx].Name
			row[2] = strconv.Itoa(mcv.LogicalDic[idx].Value)
			row[3] = mcv.LogicalDic[idx].ValueName
			row[4] = mcv.LogicalDic[idx].ValueDescription
			row[5] = strconv.Itoa(mcv.LogicalDic[idx].LanguageID)
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "LogicalDic"+ext, err)
	}

	row = make([]string, 6)
	idx = 0
	err = toCsvOutput(
		outPath("ClassificationDic"),
		[]string{"TypeID", "Name", "Description", "Note", "NumberOfValues", "LanguageID"},
		func() (bool, []string, error) {
			if idx >= len(mcv.ClassificationDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.ClassificationDic[idx].TypeID)
			row[1] = mcv.ClassificationDic[idx].Name
			row[2] = mcv.ClassificationDic[idx].Description
			row[3] = ""
			row[4] = strconv.Itoa(mcv.ClassificationDic[idx].NumberOfValues)
			row[5] = strconv.Itoa(mcv.ClassificationDic[idx].LanguageID)

			if e := writeNote(
				dir, "ClassificationDic."+mcv.ClassificationDic[idx].Name, langIdCode[mcv.ClassificationDic[idx].LanguageID], &mcv.ClassificationDic[idx].Note,
			); e != nil {
				return false, row, e
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "ClassificationDic"+ext, err)
	}

	row = make([]string, 6)
	idx = 0
	err = toCsvOutput(
		outPath("ClassificationValueDic"),
		[]string{"TypeID", "EnumValue", "Name", "Description", "Note", "LanguageID"},
		func() (bool, []string, error) {
			if idx >= len(mcv.ClassificationValueDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.ClassificationValueDic[idx].TypeID)
			row[1] = strconv.Itoa(mcv.ClassificationValueDic[idx].EnumValue)
			row[2] = mcv.ClassificationValueDic[idx].Name
			row[3] = mcv.ClassificationValueDic[idx].Description
			row[4] = ""
			row[5] = strconv.Itoa(mcv.ClassificationValueDic[idx].LanguageID)

			if e := writeNote(
				dir, "ClassificationValueDic."+mcv.ClassificationValueDic[idx].Name, langIdCode[mcv.ClassificationValueDic[idx].LanguageID], &mcv.ClassificationValueDic[idx].Note,
			); e != nil {
				return false, row, e
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "ClassificationValueDic"+ext, err)
	}

	row = make([]string, 7)
	idx = 0
	err = toCsvOutput(
		outPath("RangeDic"),
		[]string{"TypeID", "Name", "Description", "Note", "Min", "Max", "LanguageID"},
		func() (bool, []string, error) {
			if idx >= len(mcv.RangeDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.RangeDic[idx].TypeID)
			row[1] = mcv.RangeDic[idx].Name
			row[2] = mcv.RangeDic[idx].Description
			row[3] = ""
			row[4] = strconv.Itoa(mcv.RangeDic[idx].Min)
			row[5] = strconv.Itoa(mcv.RangeDic[idx].Max)
			row[6] = strconv.Itoa(mcv.RangeDic[idx].LanguageID)

			if e := writeNote(
				dir, "RangeDic."+mcv.RangeDic[idx].Name, langIdCode[mcv.RangeDic[idx].LanguageID], &mcv.RangeDic[idx].Note,
			); e != nil {
				return false, row, e
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "RangeDic"+ext, err)
	}

	row = make([]string, 2)
	idx = 0
	err = toCsvOutput(
		outPath("RangeValueDic"),
		[]string{"TypeID", "Value"},
		func() (bool, []string, error) {
			if idx >= len(mcv.RangeValueDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.RangeValueDic[idx].TypeID)
			row[1] = strconv.Itoa(mcv.RangeValueDic[idx].Value)
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "RangeValueDic"+ext, err)
	}

	row = make([]string, 6)
	idx = 0
	err = toCsvOutput(
		outPath("PartitionDic"),
		[]string{"TypeID", "Name", "Description", "Note", "NumberOfValues", "LanguageID"},
		func() (bool, []string, error) {
			if idx >= len(mcv.PartitionDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.PartitionDic[idx].TypeID)
			row[1] = mcv.PartitionDic[idx].Name
			row[2] = mcv.PartitionDic[idx].Description
			row[3] = ""
			row[4] = strconv.Itoa(mcv.PartitionDic[idx].NumberOfValues)
			row[5] = strconv.Itoa(mcv.PartitionDic[idx].LanguageID)

			if e := writeNote(
				dir, "PartitionDic."+mcv.PartitionDic[idx].Name, langIdCode[mcv.PartitionDic[idx].LanguageID], &mcv.PartitionDic[idx].Note,
			); e != nil {
				return false, row, e
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "PartitionDic"+ext, err)
	}

	row = make([]string, 4)
	idx = 0
	err = toCsvOutput(
		outPath("PartitionValueDic"),
		[]string{"TypeID", "Position", "Value", "StringValue"},
		func() (bool, []string, error) {
			if idx >= len(mcv.PartitionValueDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.PartitionValueDic[idx].TypeID)
			row[1] = strconv.Itoa(mcv.PartitionValueDic[idx].Position)
			row[2] = strconv.Itoa(mcv.PartitionValueDic[idx].Value)
			row[3] = mcv.PartitionValueDic[idx].StringValue
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "PartitionValueDic"+ext, err)
	}

	row = make([]string, 4)
	idx = 0
	err = toCsvOutput(
		outPath("PartitionIntervalDic"),
		[]string{"TypeID", "Position", "Description", "LanguageID"},
		func() (bool, []string, error) {
			if idx >= len(mcv.PartitionIntervalDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.PartitionIntervalDic[idx].TypeID)
			row[1] = strconv.Itoa(mcv.PartitionIntervalDic[idx].Position)
			row[2] = mcv.PartitionIntervalDic[idx].Description
			row[3] = strconv.Itoa(mcv.PartitionIntervalDic[idx].LanguageID)
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "PartitionIntervalDic"+ext, err)
	}

	row = make([]string, 11)
	idx = 0
	err = toCsvOutput(
		outPath("ParameterDic"),
		[]string{
			"ParameterID", "Name", "Description", "Note", "ValueNote", "TypeID", "Rank", "NumberOfCumulatedDimensions", "ModelGenerated", "Hidden", "LanguageID",
		},
		func() (bool, []string, error) {
			if idx >= len(mcv.ParameterDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.ParameterDic[idx].ParameterID)
			row[1] = mcv.ParameterDic[idx].Name
			row[2] = mcv.ParameterDic[idx].Description
			row[3] = ""
			row[4] = ""
			row[5] = strconv.Itoa(mcv.ParameterDic[idx].TypeID)
			row[6] = strconv.Itoa(mcv.ParameterDic[idx].Rank)
			row[7] = strconv.Itoa(mcv.ParameterDic[idx].NumberOfCumulatedDimensions)
			row[8] = strconv.Itoa(mcv.ParameterDic[idx].ModelGenerated)
			row[9] = strconv.FormatBool(mcv.ParameterDic[idx].Hidden)
			row[10] = strconv.Itoa(mcv.ParameterDic[idx].LanguageID)

			if e := writeNote(
				dir, "ParameterDic."+mcv.ParameterDic[idx].Name, langIdCode[mcv.ParameterDic[idx].LanguageID], &mcv.ParameterDic[idx].Note,
			); e != nil {
				return false, row, e
			}
			if e := writeNote(
				dir, "ParameterDic.ValueNote."+mcv.ParameterDic[idx].Name, langIdCode[mcv.ParameterDic[idx].LanguageID], &mcv.ParameterDic[idx].ValueNote,
			); e != nil {
				return false, row, e
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "ParameterDic"+ext, err)
	}

	row = make([]string, 4)
	idx = 0
	err = toCsvOutput(
		outPath("ParameterDimensionDic"),
		[]string{"ParameterID", "DisplayPosition", "TypeID", "Position"},
		func() (bool, []string, error) {
			if idx >= len(mcv.ParameterDimensionDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.ParameterDimensionDic[idx].ParameterID)
			row[1] = strconv.Itoa(mcv.ParameterDimensionDic[idx].DisplayPosition)
			row[2] = strconv.Itoa(mcv.ParameterDimensionDic[idx].TypeID)
			row[3] = strconv.Itoa(mcv.ParameterDimensionDic[idx].Position)
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "ParameterDimensionDic"+ext, err)
	}

	row = make([]string, 7)
	idx = 0
	err = toCsvOutput(
		outPath("ParameterGroupDic"),
		[]string{"ParameterGroupID", "Name", "Description", "Note", "ModelGenerated", "Hidden", "LanguageID"},
		func() (bool, []string, error) {
			if idx >= len(mcv.ParameterGroupDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.ParameterGroupDic[idx].ParameterGroupID)
			row[1] = mcv.ParameterGroupDic[idx].Name
			row[2] = mcv.ParameterGroupDic[idx].Description
			row[3] = ""
			row[4] = strconv.Itoa(mcv.ParameterGroupDic[idx].ModelGenerated)
			row[5] = strconv.FormatBool(mcv.ParameterGroupDic[idx].Hidden)
			row[6] = strconv.Itoa(mcv.ParameterGroupDic[idx].LanguageID)

			if e := writeNote(
				dir, "ParameterGroupDic."+mcv.ParameterGroupDic[idx].Name, langIdCode[mcv.ParameterGroupDic[idx].LanguageID], &mcv.ParameterGroupDic[idx].Note,
			); e != nil {
				return false, row, e
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "ParameterGroupDic"+ext, err)
	}

	row = make([]string, 4)
	idx = 0
	err = toCsvOutput(
		outPath("ParameterGroupMemberDic"),
		[]string{"ParameterGroupID", "Position", "ParameterID", "MemberGroupID"},
		func() (bool, []string, error) {
			if idx >= len(mcv.ParameterGroupMemberDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.ParameterGroupMemberDic[idx].ParameterGroupID)
			row[1] = strconv.Itoa(mcv.ParameterGroupMemberDic[idx].Position)
			row[2] = ""
			if mcv.ParameterGroupMemberDic[idx].ParameterID != nil {
				row[2] = strconv.FormatInt(*mcv.ParameterGroupMemberDic[idx].ParameterID, 10)
			}
			row[3] = ""
			if mcv.ParameterGroupMemberDic[idx].MemberGroupID != nil {
				row[3] = strconv.FormatInt(*mcv.ParameterGroupMemberDic[idx].MemberGroupID, 10)
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "ParameterGroupMemberDic"+ext, err)
	}

	row = make([]string, 12)
	idx = 0
	err = toCsvOutput(
		outPath("TableDic"),
		[]string{
			"TableID", "Name", "Description", "Note", "Rank", "AnalysisDimensionPosition", "AnalysisDimensionName", "AnalysisDimensionDescription",
			"AnalysisDimensionNote", "Sparse", "Hidden", "LanguageID",
		},
		func() (bool, []string, error) {
			if idx >= len(mcv.TableDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.TableDic[idx].TableID)
			row[1] = mcv.TableDic[idx].Name
			row[2] = mcv.TableDic[idx].Description
			row[3] = ""
			row[4] = strconv.Itoa(mcv.TableDic[idx].Rank)
			row[5] = strconv.Itoa(mcv.TableDic[idx].AnalysisDimensionPosition)
			row[6] = mcv.TableDic[idx].AnalysisDimensionName
			row[7] = mcv.TableDic[idx].AnalysisDimensionDescription
			row[8] = ""
			row[9] = strconv.FormatBool(mcv.TableDic[idx].Sparse)
			row[10] = strconv.FormatBool(mcv.TableDic[idx].Hidden)
			row[11] = strconv.Itoa(mcv.TableDic[idx].LanguageID)

			if e := writeNote(
				dir, "TableDic."+mcv.TableDic[idx].Name, langIdCode[mcv.TableDic[idx].LanguageID], &mcv.TableDic[idx].Note,
			); e != nil {
				return false, row, e
			}
			if e := writeNote(
				dir, "TableDic.AnalysisDimensionNote."+mcv.TableDic[idx].Name, langIdCode[mcv.TableDic[idx].LanguageID], &mcv.TableDic[idx].AnalysisDimensionNote,
			); e != nil {
				return false, row, e
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "TableDic"+ext, err)
	}

	row = make([]string, 12)
	idx = 0
	err = toCsvOutput(
		outPath("UserTableDic"),
		[]string{
			"TableID", "Name", "Description", "Note", "Rank", "AnalysisDimensionPosition", "AnalysisDimensionName", "AnalysisDimensionDescription",
			"AnalysisDimensionNote", "Sparse", "Hidden", "LanguageID",
		},
		func() (bool, []string, error) {
			if idx >= len(mcv.UserTableDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.UserTableDic[idx].TableID)
			row[1] = mcv.UserTableDic[idx].Name
			row[2] = mcv.UserTableDic[idx].Description
			row[3] = ""
			row[4] = strconv.Itoa(mcv.UserTableDic[idx].Rank)
			row[5] = strconv.Itoa(mcv.UserTableDic[idx].AnalysisDimensionPosition)
			row[6] = mcv.UserTableDic[idx].AnalysisDimensionName
			row[7] = mcv.UserTableDic[idx].AnalysisDimensionDescription
			row[8] = ""
			row[9] = strconv.FormatBool(mcv.UserTableDic[idx].Sparse)
			row[10] = strconv.FormatBool(mcv.UserTableDic[idx].Hidden)
			row[11] = strconv.Itoa(mcv.UserTableDic[idx].LanguageID)

			if e := writeNote(
				dir, "UserTableDic."+mcv.UserTableDic[idx].Name, langIdCode[mcv.UserTableDic[idx].LanguageID], &mcv.UserTableDic[idx].Note,
			); e != nil {
				return false, row, e
			}
			if e := writeNote(
				dir, "UserTableDic.AnalysisDimensionNote."+mcv.UserTableDic[idx].Name, langIdCode[mcv.UserTableDic[idx].LanguageID], &mcv.UserTableDic[idx].AnalysisDimensionNote,
			); e != nil {
				return false, row, e
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "UserTableDic"+ext, err)
	}

	row = make([]string, 8)
	idx = 0
	err = toCsvOutput(
		outPath("TableClassDic"),
		[]string{"TableID", "Position", "Name", "Description", "Note", "TypeID", "Totals", "LanguageID"},
		func() (bool, []string, error) {
			if idx >= len(mcv.TableClassDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.TableClassDic[idx].TableID)
			row[1] = strconv.Itoa(mcv.TableClassDic[idx].Position)
			row[2] = mcv.TableClassDic[idx].Name
			row[3] = mcv.TableClassDic[idx].Description
			row[4] = ""
			row[5] = strconv.Itoa(mcv.TableClassDic[idx].TypeID)
			row[6] = strconv.FormatBool(mcv.TableClassDic[idx].Totals)
			row[7] = strconv.Itoa(mcv.TableClassDic[idx].LanguageID)

			if e := writeNote(
				dir, "TableClassDic."+mcv.TableClassDic[idx].Name, langIdCode[mcv.TableClassDic[idx].LanguageID], &mcv.TableClassDic[idx].Note,
			); e != nil {
				return false, row, e
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "TableClassDic"+ext, err)
	}

	row = make([]string, 7)
	idx = 0
	err = toCsvOutput(
		outPath("TableExpressionDic"),
		[]string{"TableID", "ExpressionID", "Name", "Description", "Note", "Decimals", "LanguageID"},
		func() (bool, []string, error) {
			if idx >= len(mcv.TableExpressionDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.TableExpressionDic[idx].TableID)
			row[1] = strconv.Itoa(mcv.TableExpressionDic[idx].ExpressionID)
			row[2] = mcv.TableExpressionDic[idx].Name
			row[3] = mcv.TableExpressionDic[idx].Description
			row[4] = ""
			row[5] = strconv.Itoa(mcv.TableExpressionDic[idx].Decimals)
			row[6] = strconv.Itoa(mcv.TableExpressionDic[idx].LanguageID)

			if e := writeNote(
				dir, "TableExpressionDic."+mcv.TableExpressionDic[idx].Name, langIdCode[mcv.TableExpressionDic[idx].LanguageID], &mcv.TableExpressionDic[idx].Note,
			); e != nil {
				return false, row, e
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "TableExpressionDic"+ext, err)
	}

	row = make([]string, 6)
	idx = 0
	err = toCsvOutput(
		outPath("TableGroupDic"),
		[]string{"TableGroupID", "Name", "Description", "Note", "Hidden", "LanguageID"},
		func() (bool, []string, error) {
			if idx >= len(mcv.TableGroupDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.TableGroupDic[idx].TableGroupID)
			row[1] = mcv.TableGroupDic[idx].Name
			row[2] = mcv.TableGroupDic[idx].Description
			row[3] = ""
			row[4] = strconv.FormatBool(mcv.TableGroupDic[idx].Hidden)
			row[5] = strconv.Itoa(mcv.TableGroupDic[idx].LanguageID)

			if e := writeNote(
				dir, "TableGroupDic."+mcv.TableGroupDic[idx].Name, langIdCode[mcv.TableGroupDic[idx].LanguageID], &mcv.TableGroupDic[idx].Note,
			); e != nil {
				return false, row, e
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "TableGroupDic"+ext, err)
	}

	row = make([]string, 4)
	idx = 0
	err = toCsvOutput(
		outPath("TableGroupMemberDic"),
		[]string{"TableGroupID", "Position", "TableID", "MemberGroupID"},
		func() (bool, []string, error) {
			if idx >= len(mcv.TableGroupMemberDic) {
				return true, row, nil // end of model_dic rows
			}
			row[0] = strconv.Itoa(mcv.TableGroupMemberDic[idx].TableGroupID)
			row[1] = strconv.Itoa(mcv.TableGroupMemberDic[idx].Position)
			row[2] = ""
			if mcv.TableGroupMemberDic[idx].TableID != nil {
				row[2] = strconv.FormatInt(*mcv.TableGroupMemberDic[idx].TableID, 10)
			}
			row[3] = ""
			if mcv.TableGroupMemberDic[idx].MemberGroupID != nil {
				row[3] = strconv.FormatInt(*mcv.TableGroupMemberDic[idx].MemberGroupID, 10)
			}
			idx++
			return false, row, nil
		})
	if err != nil {
		return helper.ErrorNew("failed to write into", "TableGroupMemberDic"+ext, err)
	}

	return nil
}
