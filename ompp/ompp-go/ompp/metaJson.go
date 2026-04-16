// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package ompp

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/stcopenmpp/ompp-go/ompp/db"
	"github.com/stcopenmpp/ompp-go/ompp/helper"
)

// model type metadata, "unpacked" during marshaled to json (range enums restored)
type TypeMetaUnpack struct {
	TypeDicRow *db.TypeDicRow   // model type rows: type_dic join to model_type_dic
	Enum       []db.TypeEnumRow // type enum rows: type_enum_lst join to model_type_dic
}

// copy of ModelMeta, using alias for TypeMeta to do a special range type marshaling
type ModelMetaUnpack struct {
	Model       *db.ModelDicRow      // model_dic table row
	Type        []TypeMetaUnpack     // types metadata: type name and enums
	Param       []db.ParamMeta       // parameters metadata: parameter name, type, dimensions
	Table       []db.TableMeta       // output tables metadata: table name, dimensions, accumulators, expressions
	Entity      []db.EntityMeta      // model entities and attributes
	Group       []db.GroupMeta       // groups of parameters or output tables
	EntityGroup []db.EntityGroupMeta // groups of entity attributes
}

func CopyModelMetaToUnpack(meta *db.ModelMeta) *ModelMetaUnpack {
	if meta == nil {
		return nil
	}

	mcp := ModelMetaUnpack{
		Model:       &meta.Model,
		Type:        make([]TypeMetaUnpack, len(meta.Type)),
		Param:       meta.Param,
		Table:       meta.Table,
		Entity:      meta.Entity,
		Group:       meta.Group,
		EntityGroup: meta.EntityGroup,
	}
	for k := range meta.Type {
		mcp.Type[k].TypeDicRow = &meta.Type[k].TypeDicRow
		mcp.Type[k].Enum = meta.Type[k].Enum
	}
	return &mcp
}

// marshal type row and type enums[] to json, "unpack" range enums which may be not loaded from database
func (src *TypeMetaUnpack) MarshalJSON() ([]byte, error) {

	tm := struct {
		*db.TypeDicRow
		Enum []db.TypeEnumRow
	}{
		TypeDicRow: src.TypeDicRow,
		Enum:       src.Enum,
	}

	// if it is a range and enums not loaded from database then create enums
	if tm.IsRange && len(tm.Enum) <= 0 {

		n := 1 + tm.MaxEnumId - tm.MinEnumId
		tm.Enum = make([]db.TypeEnumRow, n)

		for k := 0; k < n; k++ {

			nId := tm.MinEnumId + k
			tm.Enum[k] = db.TypeEnumRow{
				ModelId: tm.ModelId,
				TypeId:  tm.TypeId,
				EnumId:  nId,
				Name:    strconv.Itoa(nId),
			}
		}
	}

	return json.Marshal(tm)
}

// model metadata encoder into json.
//
// model metadata db rows with language-specific description and notes.
// It is sliced by one single language, but it can be different single language for each row.
// It is either user preferred language, model default language, first of the row or empty "" language.
type ModelMetaEncoder struct {
	MetaDescrNote    modelMetaDescrNote // model metadata, including description and notes
	preferedLangCode string             // prefered language code, e.g.: fr-CA
	defaultLangCode  string             // model default language code, e.g.: EN
}

// retrun true if ModelMetaEncoder initialized
func (me *ModelMetaEncoder) IsInit() bool {
	return me.MetaDescrNote.Model.Name != "" &&
		me.MetaDescrNote.Model.Digest != "" &&
		me.MetaDescrNote.Model.CreateDateTime != "" &&
		(me.preferedLangCode != "" || me.defaultLangCode != "")
}

// encode model metadata into json.
// If isPack is true then range types are packed.
// if isIndent is true then json output indented.
func (me *ModelMetaEncoder) DoEncode(isPack bool, je *json.Encoder) error {
	if !me.IsInit() {
		return helper.ErrorNew("Invalid (empty) model metadata")
	}

	if isPack {
		return je.Encode(me.MetaDescrNote) // encode metadata packed
	}
	// else unpack range types and encode unpacked

	// copy of modelMetaDescrNote, using alias for TypeMeta to do a special range type marshaling
	type modelMetaUnpackDescrNote struct {
		*db.ModelDicDescrNote                        // model text rows: model_dic_txt
		TypeTxt               []typeUnpackDescrNote  // model type text rows: type_dic_txt join to model_type_dic
		ParamTxt              []ParamDescrNote       // model parameter text rows: parameter_dic, model_parameter_dic, parameter_dic_txt, parameter_dims_txt
		TableTxt              []TableDescrNote       // model output table text rows: table_dic, model_table_dic, table_dic_txt, table_dims_txt, table_acc_txt, table_expr_txt
		EntityTxt             []EntityDescrNote      // model entity text rows: join of entity_dic, model_entity_dic, entity_dic_txt, entity_attr_txt
		GroupTxt              []GroupDescrNote       // model group text rows: group_txt join to group_lst
		EntityGroupTxt        []EntityGroupDescrNote // model entity group text rows: entity_group_txt join to entity_group_lst
	}
	mcp := modelMetaUnpackDescrNote{
		ModelDicDescrNote: &me.MetaDescrNote.ModelDicDescrNote,
		TypeTxt:           make([]typeUnpackDescrNote, len(me.MetaDescrNote.TypeTxt)),
		ParamTxt:          me.MetaDescrNote.ParamTxt,
		TableTxt:          me.MetaDescrNote.TableTxt,
		EntityTxt:         me.MetaDescrNote.EntityTxt,
		GroupTxt:          me.MetaDescrNote.GroupTxt,
		EntityGroupTxt:    me.MetaDescrNote.EntityGroupTxt,
	}

	for k := range me.MetaDescrNote.TypeTxt {
		mcp.TypeTxt[k].Type = me.MetaDescrNote.TypeTxt[k].Type
		mcp.TypeTxt[k].DescrNote = &me.MetaDescrNote.TypeTxt[k].DescrNote
		mcp.TypeTxt[k].TypeEnumTxt = me.MetaDescrNote.TypeTxt[k].TypeEnumTxt

		mcp.TypeTxt[k].langCode = *mcp.TypeTxt[k].DescrNote.LangCode
		if mcp.TypeTxt[k].langCode == "" {
			mcp.TypeTxt[k].langCode = me.preferedLangCode
		}
		if mcp.TypeTxt[k].langCode == "" {
			mcp.TypeTxt[k].langCode = me.defaultLangCode
		}
	}

	return je.Encode(mcp)
}

// model metadata db rows with language-specific description and notes.
// It is sliced by one single language, but it can be different single language for each row.
// It is either user preferred language, model default language, first of the row or empty "" language.
type modelMetaDescrNote struct {
	db.ModelDicDescrNote                        // model text rows: model_dic_txt
	TypeTxt              []TypeDescrNote        // model type text rows: type_dic_txt join to model_type_dic
	ParamTxt             []ParamDescrNote       // model parameter text rows: parameter_dic, model_parameter_dic, parameter_dic_txt, parameter_dims_txt
	TableTxt             []TableDescrNote       // model output table text rows: table_dic, model_table_dic, table_dic_txt, table_dims_txt, table_acc_txt, table_expr_txt
	EntityTxt            []EntityDescrNote      // model entity text rows: join of entity_dic, model_entity_dic, entity_dic_txt, entity_attr_txt
	GroupTxt             []GroupDescrNote       // model group text rows: group_txt join to group_lst
	EntityGroupTxt       []EntityGroupDescrNote // model entity group text rows: entity_group_txt join to entity_group_lst
}

// TypeDescrNote is join of type_dic_txt, model_type_dic, type_dic_txt
type TypeDescrNote struct {
	Type        *db.TypeDicRow      // model type row: type_dic join to model_type_dic
	DescrNote   aDescrNote          // from type_dic_txt
	TypeEnumTxt []typeEnumDescrNote // type enum text rows: type_enum_txt join to model_type_dic
}

// ParamDescrNote is join of parameter_dic, model_parameter_dic, parameter_dic_txt, parameter_dims_txt
type ParamDescrNote struct {
	Param        *db.ParamDicRow      // parameter row: parameter_dic join to model_parameter_dic table
	DescrNote    aDescrNote           // from parameter_dic_txt
	ParamDimsTxt []ParamDimsDescrNote // parameter dimension text rows: parameter_dims_txt join to model_parameter_dic
}

// ParamDimsDescrNote is join of parameter_dims, model_parameter_dic, parameter_dims_txt
type ParamDimsDescrNote struct {
	Dim       *db.ParamDimsRow // parameter dimension row: parameter_dims join to model_parameter_dic table
	DescrNote aDescrNote       // from parameter_dims_txt
}

// TableDescrNote is join of table_dic, model_table_dic, table_dic_txt, table_dims_txt, table_acc_txt, table_expr_txt
type TableDescrNote struct {
	Table        *db.TableDicRow      // output table row: table_dic join to model_table_dic
	LangCode     *string              // table_dic_txt.lang_code
	TableDescr   *string              // table_dic_txt.descr
	TableNote    *string              // table_dic_txt.note
	ExprDescr    *string              // table_dic_txt.expr_descr
	ExprNote     *string              // table_dic_txt.expr_note
	TableDimsTxt []TableDimsDescrNote // output table dimension text rows: table_dims_txt join to model_table_dic
	TableAccTxt  []TableAccDescrNote  // output table accumulator text rows: table_acc_txt join to model_table_dic
	TableExprTxt []TableExprDescrNote // output table expression text rows: table_expr_txt join to model_table_dic
}

// TableDimsDescrNote is join of table_dims, model_table_dic, table_dims_txt
type TableDimsDescrNote struct {
	Dim       *db.TableDimsRow // parameter dimension row: table_dims join to model_table_dic table
	DescrNote aDescrNote       // from table_dims_txt
}

// TableAccDescrNote is join of table_acc, model_table_dic, table_acc_txt
type TableAccDescrNote struct {
	Acc       *db.TableAccRow // output table accumulator row: table_acc join to model_table_dic
	DescrNote aDescrNote      // from table_acc_txt
}

// TableExprDescrNote is join of table_expr, model_table_dic, table_expr_txt
type TableExprDescrNote struct {
	Expr      *db.TableExprRow // output table expression row: table_expr join to model_table_dic
	DescrNote aDescrNote       // from table_expr_txt
}

// EntityDescrNote is join of entity_dic, model_entity_dic, entity_dic_txt, entity_attr_txt
type EntityDescrNote struct {
	Entity        *db.EntityDicRow      // entity row: entity_dic join to model_entity_dic
	DescrNote     aDescrNote            // from entity_dic_txt
	EntityAttrTxt []EntityAttrDescrNote // entity attribute text rows: entity_attr, model_entity_dic, entity_attr_txt
}

// EntityAttrDescrNote is join of entity_attr, model_entity_dic, entity_attr_txt
type EntityAttrDescrNote struct {
	Attr      *db.EntityAttrRow // entity attribute row: entity_attr join to model_entity_dic table
	DescrNote aDescrNote        // from entity_attr_txt
}

// GroupDescrNote is join of group_lst, group_pc and group_txt
type GroupDescrNote struct {
	Group     *db.GroupMeta // parameters or output tables group rows: group_lst join to group_pc
	DescrNote aDescrNote    // from group_txt
}

// EntityGroupDescrNote is join of entity_group_lst, entity_group_pc and entity_group_txt
type EntityGroupDescrNote struct {
	Group     *db.EntityGroupMeta // parameters or output tables group rows: entity_group_lst join to entity_group_pc
	DescrNote aDescrNote          // from entity_group_txt
}

// return language-specific model metadata by model digest or name language.
// It can be `lc` prefered language or `lcd` default model language or empty if no model text db rows exist.
func (me *ModelMetaEncoder) New(meta *db.ModelMeta, txtMeta *db.ModelTxtMeta, lc string, lcd string) error {

	if meta == nil || txtMeta == nil {
		errors.New("Error: invalid (empty) model metadata")
	}
	me.preferedLangCode = lc
	me.defaultLangCode = lcd

	mt := modelMetaDescrNote{
		ModelDicDescrNote: db.ModelDicDescrNote{Model: meta.Model},
		TypeTxt:           make([]TypeDescrNote, len(meta.Type)),
		ParamTxt:          make([]ParamDescrNote, len(meta.Param)),
		TableTxt:          make([]TableDescrNote, len(meta.Table)),
		EntityTxt:         make([]EntityDescrNote, len(meta.Entity)),
		GroupTxt:          make([]GroupDescrNote, len(meta.Group)),
		EntityGroupTxt:    make([]EntityGroupDescrNote, len(meta.EntityGroup)),
	}
	emptyStr := ""

	// model types
	for k := range mt.TypeTxt {
		mt.TypeTxt[k].Type = &meta.Type[k].TypeDicRow
		mt.TypeTxt[k].TypeEnumTxt = make([]typeEnumDescrNote, len(meta.Type[k].Enum))
		mt.TypeTxt[k].DescrNote.LangCode = &emptyStr
		mt.TypeTxt[k].DescrNote.Descr = &emptyStr
		mt.TypeTxt[k].DescrNote.Note = &emptyStr

		for j := range mt.TypeTxt[k].TypeEnumTxt {
			mt.TypeTxt[k].TypeEnumTxt[j].Enum = &meta.Type[k].Enum[j]
		}
	}

	// model parameters
	for k := range mt.ParamTxt {
		mt.ParamTxt[k].Param = &meta.Param[k].ParamDicRow
		mt.ParamTxt[k].ParamDimsTxt = make([]ParamDimsDescrNote, len(meta.Param[k].Dim))
		mt.ParamTxt[k].DescrNote.LangCode = &emptyStr
		mt.ParamTxt[k].DescrNote.Descr = &emptyStr
		mt.ParamTxt[k].DescrNote.Note = &emptyStr

		for j := range mt.ParamTxt[k].ParamDimsTxt {
			mt.ParamTxt[k].ParamDimsTxt[j].Dim = &meta.Param[k].Dim[j]
			mt.ParamTxt[k].ParamDimsTxt[j].DescrNote.LangCode = &emptyStr
			mt.ParamTxt[k].ParamDimsTxt[j].DescrNote.Descr = &emptyStr
			mt.ParamTxt[k].ParamDimsTxt[j].DescrNote.Note = &emptyStr
		}
	}

	// model output tables, remove sql for accumulators and expressions
	for k := range mt.TableTxt {
		mt.TableTxt[k].Table = &meta.Table[k].TableDicRow
		mt.TableTxt[k].TableDimsTxt = make([]TableDimsDescrNote, len(meta.Table[k].Dim))
		mt.TableTxt[k].TableAccTxt = make([]TableAccDescrNote, len(meta.Table[k].Acc))
		mt.TableTxt[k].TableExprTxt = make([]TableExprDescrNote, len(meta.Table[k].Expr))
		mt.TableTxt[k].LangCode = &emptyStr
		mt.TableTxt[k].TableDescr = &emptyStr
		mt.TableTxt[k].TableNote = &emptyStr
		mt.TableTxt[k].ExprDescr = &emptyStr
		mt.TableTxt[k].ExprNote = &emptyStr

		for j := range mt.TableTxt[k].TableDimsTxt {
			mt.TableTxt[k].TableDimsTxt[j].Dim = &meta.Table[k].Dim[j]
			mt.TableTxt[k].TableDimsTxt[j].DescrNote.LangCode = &emptyStr
			mt.TableTxt[k].TableDimsTxt[j].DescrNote.Descr = &emptyStr
			mt.TableTxt[k].TableDimsTxt[j].DescrNote.Note = &emptyStr
		}
		for j := range mt.TableTxt[k].TableAccTxt {
			mt.TableTxt[k].TableAccTxt[j].Acc = &meta.Table[k].Acc[j]
			// mt.TableTxt[k].TableAccTxt[j].Acc.AccSql = "" // remove sql of accumulator
			mt.TableTxt[k].TableAccTxt[j].DescrNote.LangCode = &emptyStr
			mt.TableTxt[k].TableAccTxt[j].DescrNote.Descr = &emptyStr
			mt.TableTxt[k].TableAccTxt[j].DescrNote.Note = &emptyStr
		}
		for j := range mt.TableTxt[k].TableExprTxt {
			mt.TableTxt[k].TableExprTxt[j].Expr = &meta.Table[k].Expr[j]
			// mt.TableTxt[k].TableExprTxt[j].Expr.ExprSql = "" // remove sql of expression
			mt.TableTxt[k].TableExprTxt[j].DescrNote.LangCode = &emptyStr
			mt.TableTxt[k].TableExprTxt[j].DescrNote.Descr = &emptyStr
			mt.TableTxt[k].TableExprTxt[j].DescrNote.Note = &emptyStr
		}
	}

	// model entities
	for k := range mt.EntityTxt {
		mt.EntityTxt[k].Entity = &meta.Entity[k].EntityDicRow
		mt.EntityTxt[k].EntityAttrTxt = make([]EntityAttrDescrNote, len(meta.Entity[k].Attr))
		mt.EntityTxt[k].DescrNote.LangCode = &emptyStr
		mt.EntityTxt[k].DescrNote.Descr = &emptyStr
		mt.EntityTxt[k].DescrNote.Note = &emptyStr

		for j := range mt.EntityTxt[k].EntityAttrTxt {
			mt.EntityTxt[k].EntityAttrTxt[j].Attr = &meta.Entity[k].Attr[j]
			mt.EntityTxt[k].EntityAttrTxt[j].DescrNote.LangCode = &emptyStr
			mt.EntityTxt[k].EntityAttrTxt[j].DescrNote.Descr = &emptyStr
			mt.EntityTxt[k].EntityAttrTxt[j].DescrNote.Note = &emptyStr
		}
	}

	// model groups
	for k := range mt.GroupTxt {
		mt.GroupTxt[k].DescrNote.LangCode = &emptyStr
		mt.GroupTxt[k].DescrNote.Descr = &emptyStr
		mt.GroupTxt[k].DescrNote.Note = &emptyStr

		mt.GroupTxt[k].Group = &db.GroupMeta{
			GroupLstRow: meta.Group[k].GroupLstRow,
			GroupPc:     meta.Group[k].GroupPc,
		}
	}

	// model entity attribute groups
	for k := range mt.EntityGroupTxt {
		mt.EntityGroupTxt[k].DescrNote.LangCode = &emptyStr
		mt.EntityGroupTxt[k].DescrNote.Descr = &emptyStr
		mt.EntityGroupTxt[k].DescrNote.Note = &emptyStr

		mt.EntityGroupTxt[k].Group = &db.EntityGroupMeta{
			EntityGroupLstRow: meta.EntityGroup[k].EntityGroupLstRow,
			GroupPc:           meta.EntityGroup[k].GroupPc,
		}
	}

	//
	// set language-specific rows by matched language or by default language or by zero index language
	//

	// set model description and notes
	if len(txtMeta.ModelTxt) > 0 {

		var nf, i int
		for ; i < len(txtMeta.ModelTxt); i++ {
			if txtMeta.ModelTxt[i].LangCode == lc {
				break // language match
			}
			if txtMeta.ModelTxt[i].LangCode == lcd {
				nf = i // index of default language
			}
		}
		if i >= len(txtMeta.ModelTxt) {
			i = nf // use default language or zero index row
		}
		mt.DescrNote = db.DescrNote{
			LangCode: txtMeta.ModelTxt[i].LangCode,
			Descr:    txtMeta.ModelTxt[i].Descr,
			Note:     txtMeta.ModelTxt[i].Note}
	}

	// set model types description and notes
	if len(mt.TypeTxt) > 0 && len(txtMeta.TypeTxt) > 0 {

		var isKey, isFound, isMatch bool
		var nf, ni, si, di int

		for ; si < len(txtMeta.TypeTxt); si++ {

			// destination rows must be defined by [di] index
			if di >= len(mt.TypeTxt) {
				break // done with all destination text
			}

			// check if source and destination keys equal
			mId := mt.TypeTxt[di].Type.ModelId
			tId := mt.TypeTxt[di].Type.TypeId

			isKey = txtMeta.TypeTxt[si].ModelId == mId &&
				txtMeta.TypeTxt[si].TypeId == tId

			// start of next key: set value
			if !isKey && isFound {

				if !isMatch { // if no match then use default
					ni = nf
				}
				mt.TypeTxt[di].DescrNote = aDescrNote{
					LangCode: &txtMeta.TypeTxt[ni].LangCode,
					Descr:    &txtMeta.TypeTxt[ni].Descr,
					Note:     &txtMeta.TypeTxt[ni].Note}

				// reset to start next search
				isFound = false
				isMatch = false
				di++ // move to next type
				si-- // repeat current source row
				continue
			}

			// inside of key
			if isKey {

				if !isFound {
					isFound = true // first key found
					nf = si
				}
				// match the language
				isMatch = txtMeta.TypeTxt[si].LangCode == lc
				if isMatch {
					ni = si // perefred language match
				}
				if txtMeta.TypeTxt[si].LangCode == lcd {
					nf = si // index of default language
				}
			}

			// if keys not equal and destination key behind source
			// then move to next destination row and repeat current source row
			if !isKey &&
				(txtMeta.TypeTxt[si].ModelId > mId ||
					txtMeta.TypeTxt[si].ModelId == mId &&
						txtMeta.TypeTxt[si].TypeId > tId) {

				di++ // move to next type
				si-- // repeat current source row
				continue
			}
		} // for

		// last row
		if isFound && di < len(mt.TypeTxt) {

			if !isMatch { // if no match then use default
				ni = nf
			}
			if ni < len(txtMeta.TypeTxt) {
				mt.TypeTxt[di].DescrNote = aDescrNote{
					LangCode: &txtMeta.TypeTxt[ni].LangCode,
					Descr:    &txtMeta.TypeTxt[ni].Descr,
					Note:     &txtMeta.TypeTxt[ni].Note}
			}
		}
	}

	// set model enums description and notes
	if len(mt.TypeTxt) > 0 && len(txtMeta.TypeEnumTxt) > 0 {

		var isKey, isFound, isMatch bool
		var nf, ni, si, pi, ci int

		for ; si < len(txtMeta.TypeEnumTxt); si++ {

			// destination rows: parent and child must be defined by valid (pi, ci) indexes
			if pi >= len(mt.TypeTxt) {
				break // done with all destination text
			}
			if pi < len(mt.TypeTxt) &&
				ci >= len(mt.TypeTxt[pi].TypeEnumTxt) {

				if pi++; pi >= len(mt.TypeTxt) {
					break // done with all destination text
				}

				ci = 0 // move to next type
				si--   // repeat current source row
				continue
			}

			// check if source and destination keys equal
			mId := mt.TypeTxt[pi].Type.ModelId
			tId := mt.TypeTxt[pi].Type.TypeId
			eId := mt.TypeTxt[pi].TypeEnumTxt[ci].Enum.EnumId

			isKey = txtMeta.TypeEnumTxt[si].ModelId == mId &&
				txtMeta.TypeEnumTxt[si].TypeId == tId &&
				txtMeta.TypeEnumTxt[si].EnumId == eId

			// start of next key: set value
			if !isKey && isFound {

				if !isMatch { // if no match then use default
					ni = nf
				}
				if txtMeta.TypeEnumTxt[ni].Descr != "" || txtMeta.TypeEnumTxt[ni].Note != "" {
					mt.TypeTxt[pi].TypeEnumTxt[ci].DescrNote.LangCode = &txtMeta.TypeEnumTxt[ni].LangCode
					mt.TypeTxt[pi].TypeEnumTxt[ci].DescrNote.Descr = &txtMeta.TypeEnumTxt[ni].Descr
					mt.TypeTxt[pi].TypeEnumTxt[ci].DescrNote.Note = &txtMeta.TypeEnumTxt[ni].Note
				}

				// reset to start next search
				isFound = false
				isMatch = false
				ci++ // move to next type
				si-- // repeat current source row
				continue
			}

			// inside of key
			if isKey {

				if !isFound {
					isFound = true // first key found
					nf = si
				}
				// match the language
				isMatch = txtMeta.TypeEnumTxt[si].LangCode == lc
				if isMatch {
					ni = si // perefred language match
				}
				if txtMeta.TypeEnumTxt[si].LangCode == lcd {
					nf = si // index of default language
				}
			}

			// if keys not equal and destination key behind source
			// then move to next destination row and repeat current source row
			if !isKey &&
				(txtMeta.TypeEnumTxt[si].ModelId > mId ||
					txtMeta.TypeEnumTxt[si].ModelId == mId &&
						txtMeta.TypeEnumTxt[si].TypeId > tId ||
					txtMeta.TypeEnumTxt[si].ModelId == mId &&
						txtMeta.TypeEnumTxt[si].TypeId == tId &&
						txtMeta.TypeEnumTxt[si].EnumId > eId) {

				ci++ // move to next type
				si-- // repeat current source row
				continue
			}
		} // for

		// last row
		if isFound && pi < len(mt.TypeTxt) && ci < len(mt.TypeTxt[pi].TypeEnumTxt) {

			if !isMatch { // if no match then use default
				ni = nf
			}
			if ni < len(txtMeta.TypeEnumTxt) {
				if txtMeta.TypeEnumTxt[ni].Descr != "" || txtMeta.TypeEnumTxt[ni].Note != "" {
					mt.TypeTxt[pi].TypeEnumTxt[ci].DescrNote.LangCode = &txtMeta.TypeEnumTxt[ni].LangCode
					mt.TypeTxt[pi].TypeEnumTxt[ci].DescrNote.Descr = &txtMeta.TypeEnumTxt[ni].Descr
					mt.TypeTxt[pi].TypeEnumTxt[ci].DescrNote.Note = &txtMeta.TypeEnumTxt[ni].Note
				}
			}
		}
	}

	// set parameter description and notes
	if len(mt.ParamTxt) > 0 && len(txtMeta.ParamTxt) > 0 {

		var isKey, isFound, isMatch bool
		var nf, ni, si, di int

		for ; si < len(txtMeta.ParamTxt); si++ {

			// destination rows must be defined by [di] index
			if di >= len(mt.ParamTxt) {
				break // done with all destination text
			}

			// check if source and destination keys equal
			mId := mt.ParamTxt[di].Param.ModelId
			tId := mt.ParamTxt[di].Param.ParamId

			isKey = txtMeta.ParamTxt[si].ModelId == mId &&
				txtMeta.ParamTxt[si].ParamId == tId

			// start of next key: set value
			if !isKey && isFound {

				if !isMatch { // if no match then use default
					ni = nf
				}
				mt.ParamTxt[di].DescrNote = aDescrNote{
					LangCode: &txtMeta.ParamTxt[ni].LangCode,
					Descr:    &txtMeta.ParamTxt[ni].Descr,
					Note:     &txtMeta.ParamTxt[ni].Note}

				// reset to start next search
				isFound = false
				isMatch = false
				di++ // move to next parameter
				si-- // repeat current source row
				continue
			}

			// inside of key
			if isKey {

				if !isFound {
					isFound = true // first key found
					nf = si
				}
				// match the language
				isMatch = txtMeta.ParamTxt[si].LangCode == lc
				if isMatch {
					ni = si // perefred language match
				}
				if txtMeta.ParamTxt[si].LangCode == lcd {
					nf = si // index of default language
				}
			}

			// if keys not equal and destination key behind source
			// then move to next destination row and repeat current source row
			if !isKey &&
				(txtMeta.ParamTxt[si].ModelId > mId ||
					txtMeta.ParamTxt[si].ModelId == mId &&
						txtMeta.ParamTxt[si].ParamId > tId) {

				di++ // move to next parameter
				si-- // repeat current source row
				continue
			}
		} // for

		// last row
		if isFound && di < len(mt.ParamTxt) {

			if !isMatch { // if no match then use default
				ni = nf
			}
			if ni < len(txtMeta.ParamTxt) {
				mt.ParamTxt[di].DescrNote = aDescrNote{
					LangCode: &txtMeta.ParamTxt[ni].LangCode,
					Descr:    &txtMeta.ParamTxt[ni].Descr,
					Note:     &txtMeta.ParamTxt[ni].Note}
			}
		}
	}

	// set parameter dimensions description and notes
	if len(mt.ParamTxt) > 0 && len(txtMeta.ParamDimsTxt) > 0 {

		var isKey, isFound, isMatch bool
		var nf, ni, si, pi, ci int

		for ; si < len(txtMeta.ParamDimsTxt); si++ {

			// destination rows: parent and child must be defined by valid (pi, ci) indexes
			if pi >= len(mt.ParamTxt) {
				break // done with all destination text
			}
			if pi < len(mt.ParamTxt) &&
				ci >= len(mt.ParamTxt[pi].ParamDimsTxt) {

				if pi++; pi >= len(mt.ParamTxt) {
					break // done with all destination text
				}

				ci = 0 // move to next type
				si--   // repeat current source row
				continue
			}

			// check if source and destination keys equal
			mId := mt.ParamTxt[pi].Param.ModelId
			pId := mt.ParamTxt[pi].Param.ParamId
			dimId := mt.ParamTxt[pi].ParamDimsTxt[ci].Dim.DimId

			isKey = txtMeta.ParamDimsTxt[si].ModelId == mId &&
				txtMeta.ParamDimsTxt[si].ParamId == pId &&
				txtMeta.ParamDimsTxt[si].DimId == dimId

			// start of next key: set value
			if !isKey && isFound {

				if !isMatch { // if no match then use default
					ni = nf
				}
				mt.ParamTxt[pi].ParamDimsTxt[ci].DescrNote = aDescrNote{
					LangCode: &txtMeta.ParamDimsTxt[ni].LangCode,
					Descr:    &txtMeta.ParamDimsTxt[ni].Descr,
					Note:     &txtMeta.ParamDimsTxt[ni].Note}

				// reset to start next search
				isFound = false
				isMatch = false
				ci++ // move to next type
				si-- // repeat current source row
				continue
			}

			// inside of key
			if isKey {

				if !isFound {
					isFound = true // first key found
					nf = si
				}
				// match the language
				isMatch = txtMeta.ParamDimsTxt[si].LangCode == lc
				if isMatch {
					ni = si // perefred language match
				}
				if txtMeta.ParamDimsTxt[si].LangCode == lcd {
					nf = si // index of default language
				}
			}

			// if keys not equal and destination key behind source
			// then move to next destination row and repeat current source row
			if !isKey &&
				(txtMeta.ParamDimsTxt[si].ModelId > mId ||
					txtMeta.ParamDimsTxt[si].ModelId == mId &&
						txtMeta.ParamDimsTxt[si].ParamId > pId ||
					txtMeta.ParamDimsTxt[si].ModelId == mId &&
						txtMeta.ParamDimsTxt[si].ParamId == pId &&
						txtMeta.ParamDimsTxt[si].DimId > dimId) {

				ci++ // move to next type
				si-- // repeat current source row
				continue
			}
		} // for

		// last row
		if isFound && pi < len(mt.ParamTxt) && ci < len(mt.ParamTxt[pi].ParamDimsTxt) {

			if !isMatch { // if no match then use default
				ni = nf
			}
			if ni < len(txtMeta.ParamDimsTxt) {
				mt.ParamTxt[pi].ParamDimsTxt[ci].DescrNote = aDescrNote{
					LangCode: &txtMeta.ParamDimsTxt[ni].LangCode,
					Descr:    &txtMeta.ParamDimsTxt[ni].Descr,
					Note:     &txtMeta.ParamDimsTxt[ni].Note}
			}
		}
	}

	// set output table description and notes
	if len(mt.TableTxt) > 0 && len(txtMeta.TableTxt) > 0 {

		var isKey, isFound, isMatch bool
		var nf, ni, si, di int

		for ; si < len(txtMeta.TableTxt); si++ {

			// destination rows must be defined by [di] index
			if di >= len(mt.TableTxt) {
				break // done with all destination text
			}

			// check if source and destination keys equal
			mId := mt.TableTxt[di].Table.ModelId
			tId := mt.TableTxt[di].Table.TableId

			isKey = txtMeta.TableTxt[si].ModelId == mId &&
				txtMeta.TableTxt[si].TableId == tId

			// start of next key: set value
			if !isKey && isFound {

				if !isMatch { // if no match then use default
					ni = nf
				}
				mt.TableTxt[di].LangCode = &txtMeta.TableTxt[ni].LangCode
				mt.TableTxt[di].TableDescr = &txtMeta.TableTxt[ni].Descr
				mt.TableTxt[di].TableNote = &txtMeta.TableTxt[ni].Note
				mt.TableTxt[di].ExprDescr = &txtMeta.TableTxt[ni].ExprDescr
				mt.TableTxt[di].ExprNote = &txtMeta.TableTxt[ni].ExprNote

				// reset to start next search
				isFound = false
				isMatch = false
				di++ // move to next output table
				si-- // repeat current source row
				continue
			}

			// inside of key
			if isKey {

				if !isFound {
					isFound = true // first key found
					nf = si
				}
				// match the language
				isMatch = txtMeta.TableTxt[si].LangCode == lc
				if isMatch {
					ni = si // perefred language match
				}
				if txtMeta.TableTxt[si].LangCode == lcd {
					nf = si // index of default language
				}
			}

			// if keys not equal and destination key behind source
			// then move to next destination row and repeat current source row
			if !isKey &&
				(txtMeta.TableTxt[si].ModelId > mId ||
					txtMeta.TableTxt[si].ModelId == mId &&
						txtMeta.TableTxt[si].TableId > tId) {

				di++ // move to next output table
				si-- // repeat current source row
				continue
			}
		} // for

		// last row
		if isFound && di < len(mt.TableTxt) {

			if !isMatch { // if no match then use default
				ni = nf
			}
			if ni < len(txtMeta.TableTxt) {
				mt.TableTxt[di].LangCode = &txtMeta.TableTxt[ni].LangCode
				mt.TableTxt[di].TableDescr = &txtMeta.TableTxt[ni].Descr
				mt.TableTxt[di].TableNote = &txtMeta.TableTxt[ni].Note
				mt.TableTxt[di].ExprDescr = &txtMeta.TableTxt[ni].ExprDescr
				mt.TableTxt[di].ExprNote = &txtMeta.TableTxt[ni].ExprNote
			}
		}
	}

	// set output table dimensions description and notes
	if len(mt.TableTxt) > 0 && len(txtMeta.TableDimsTxt) > 0 {

		var isKey, isFound, isMatch bool
		var nf, ni, si, pi, ci int

		for ; si < len(txtMeta.TableDimsTxt); si++ {

			// destination rows: parent and child must be defined by valid (pi, ci) indexes
			if pi >= len(mt.TableTxt) {
				break // done with all destination text
			}
			if pi < len(mt.TableTxt) &&
				ci >= len(mt.TableTxt[pi].TableDimsTxt) {

				if pi++; pi >= len(mt.TableTxt) {
					break // done with all destination text
				}

				ci = 0 // move to next type
				si--   // repeat current source row
				continue
			}

			// check if source and destination keys equal
			mId := mt.TableTxt[pi].Table.ModelId
			tId := mt.TableTxt[pi].Table.TableId
			dimId := mt.TableTxt[pi].TableDimsTxt[ci].Dim.DimId

			isKey = txtMeta.TableDimsTxt[si].ModelId == mId &&
				txtMeta.TableDimsTxt[si].TableId == tId &&
				txtMeta.TableDimsTxt[si].DimId == dimId

			// start of next key: set value
			if !isKey && isFound {

				if !isMatch { // if no match then use default
					ni = nf
				}
				mt.TableTxt[pi].TableDimsTxt[ci].DescrNote = aDescrNote{
					LangCode: &txtMeta.TableDimsTxt[ni].LangCode,
					Descr:    &txtMeta.TableDimsTxt[ni].Descr,
					Note:     &txtMeta.TableDimsTxt[ni].Note}

				// reset to start next search
				isFound = false
				isMatch = false
				ci++ // move to next type
				si-- // repeat current source row
				continue
			}

			// inside of key
			if isKey {

				if !isFound {
					isFound = true // first key found
					nf = si
				}
				// match the language
				isMatch = txtMeta.TableDimsTxt[si].LangCode == lc
				if isMatch {
					ni = si // perefred language match
				}
				if txtMeta.TableDimsTxt[si].LangCode == lcd {
					nf = si // index of default language
				}
			}

			// if keys not equal and destination key behind source
			// then move to next destination row and repeat current source row
			if !isKey &&
				(txtMeta.TableDimsTxt[si].ModelId > mId ||
					txtMeta.TableDimsTxt[si].ModelId == mId &&
						txtMeta.TableDimsTxt[si].TableId > tId ||
					txtMeta.TableDimsTxt[si].ModelId == mId &&
						txtMeta.TableDimsTxt[si].TableId == tId &&
						txtMeta.TableDimsTxt[si].DimId > dimId) {

				ci++ // move to next type
				si-- // repeat current source row
				continue
			}
		} // for

		// last row
		if isFound && pi < len(mt.TableTxt) && ci < len(mt.TableTxt[pi].TableDimsTxt) {

			if !isMatch { // if no match then use default
				ni = nf
			}
			if ni < len(txtMeta.TableDimsTxt) {
				mt.TableTxt[pi].TableDimsTxt[ci].DescrNote = aDescrNote{
					LangCode: &txtMeta.TableDimsTxt[ni].LangCode,
					Descr:    &txtMeta.TableDimsTxt[ni].Descr,
					Note:     &txtMeta.TableDimsTxt[ni].Note}
			}
		}
	}

	// set output table accumulators description and notes
	if len(mt.TableTxt) > 0 && len(txtMeta.TableAccTxt) > 0 {

		var isKey, isFound, isMatch bool
		var nf, ni, si, pi, ci int

		for ; si < len(txtMeta.TableAccTxt); si++ {

			// destination rows: parent and child must be defined by valid (pi, ci) indexes
			if pi >= len(mt.TableTxt) {
				break // done with all destination text
			}
			if pi < len(mt.TableTxt) &&
				ci >= len(mt.TableTxt[pi].TableAccTxt) {

				if pi++; pi >= len(mt.TableTxt) {
					break // done with all destination text
				}

				ci = 0 // move to next type
				si--   // repeat current source row
				continue
			}

			// check if source and destination keys equal
			mId := mt.TableTxt[pi].Table.ModelId
			tId := mt.TableTxt[pi].Table.TableId
			accId := mt.TableTxt[pi].TableAccTxt[ci].Acc.AccId

			isKey = txtMeta.TableAccTxt[si].ModelId == mId &&
				txtMeta.TableAccTxt[si].TableId == tId &&
				txtMeta.TableAccTxt[si].AccId == accId

			// start of next key: set value
			if !isKey && isFound {

				if !isMatch { // if no match then use default
					ni = nf
				}
				mt.TableTxt[pi].TableAccTxt[ci].DescrNote = aDescrNote{
					LangCode: &txtMeta.TableAccTxt[ni].LangCode,
					Descr:    &txtMeta.TableAccTxt[ni].Descr,
					Note:     &txtMeta.TableAccTxt[ni].Note}

				// reset to start next search
				isFound = false
				isMatch = false
				ci++ // move to next type
				si-- // repeat current source row
				continue
			}

			// inside of key
			if isKey {

				if !isFound {
					isFound = true // first key found
					nf = si
				}
				// match the language
				isMatch = txtMeta.TableAccTxt[si].LangCode == lc
				if isMatch {
					ni = si // perefred language match
				}
				if txtMeta.TableAccTxt[si].LangCode == lcd {
					nf = si // index of default language
				}
			}

			// if keys not equal and destination key behind source
			// then move to next destination row and repeat current source row
			if !isKey &&
				(txtMeta.TableAccTxt[si].ModelId > mId ||
					txtMeta.TableAccTxt[si].ModelId == mId &&
						txtMeta.TableAccTxt[si].TableId > tId ||
					txtMeta.TableAccTxt[si].ModelId == mId &&
						txtMeta.TableAccTxt[si].TableId == tId &&
						txtMeta.TableAccTxt[si].AccId > accId) {

				ci++ // move to next type
				si-- // repeat current source row
				continue
			}
		} // for

		// last row
		if isFound && pi < len(mt.TableTxt) && ci < len(mt.TableTxt[pi].TableAccTxt) {

			if !isMatch { // if no match then use default
				ni = nf
			}
			if ni < len(txtMeta.TableAccTxt) {
				mt.TableTxt[pi].TableAccTxt[ci].DescrNote = aDescrNote{
					LangCode: &txtMeta.TableAccTxt[ni].LangCode,
					Descr:    &txtMeta.TableAccTxt[ni].Descr,
					Note:     &txtMeta.TableAccTxt[ni].Note}
			}
		}
	}

	// set output table expressions description and notes
	if len(mt.TableTxt) > 0 && len(txtMeta.TableExprTxt) > 0 {

		var isKey, isFound, isMatch bool
		var nf, ni, si, pi, ci int

		for ; si < len(txtMeta.TableExprTxt); si++ {

			// destination rows: parent and child must be defined by valid (pi, ci) indexes
			if pi >= len(mt.TableTxt) {
				break // done with all destination text
			}
			if pi < len(mt.TableTxt) &&
				ci >= len(mt.TableTxt[pi].TableExprTxt) {

				if pi++; pi >= len(mt.TableTxt) {
					break // done with all destination text
				}

				ci = 0 // move to next type
				si--   // repeat current source row
				continue
			}

			// check if source and destination keys equal
			mId := mt.TableTxt[pi].Table.ModelId
			tId := mt.TableTxt[pi].Table.TableId
			exprId := mt.TableTxt[pi].TableExprTxt[ci].Expr.ExprId

			isKey = txtMeta.TableExprTxt[si].ModelId == mId &&
				txtMeta.TableExprTxt[si].TableId == tId &&
				txtMeta.TableExprTxt[si].ExprId == exprId

			// start of next key: set value
			if !isKey && isFound {

				if !isMatch { // if no match then use default
					ni = nf
				}
				mt.TableTxt[pi].TableExprTxt[ci].DescrNote = aDescrNote{
					LangCode: &txtMeta.TableExprTxt[ni].LangCode,
					Descr:    &txtMeta.TableExprTxt[ni].Descr,
					Note:     &txtMeta.TableExprTxt[ni].Note}

				// reset to start next search
				isFound = false
				isMatch = false
				ci++ // move to next type
				si-- // repeat current source row
				continue
			}

			// inside of key
			if isKey {

				if !isFound {
					isFound = true // first key found
					nf = si
				}
				// match the language
				isMatch = txtMeta.TableExprTxt[si].LangCode == lc
				if isMatch {
					ni = si // perefred language match
				}
				if txtMeta.TableExprTxt[si].LangCode == lcd {
					nf = si // index of default language
				}
			}

			// if keys not equal and destination key behind source
			// then move to next destination row and repeat current source row
			if !isKey &&
				(txtMeta.TableExprTxt[si].ModelId > mId ||
					txtMeta.TableExprTxt[si].ModelId == mId &&
						txtMeta.TableExprTxt[si].TableId > tId ||
					txtMeta.TableExprTxt[si].ModelId == mId &&
						txtMeta.TableExprTxt[si].TableId == tId &&
						txtMeta.TableExprTxt[si].ExprId > exprId) {

				ci++ // move to next type
				si-- // repeat current source row
				continue
			}
		} // for

		// last row
		if isFound && pi < len(mt.TableTxt) && ci < len(mt.TableTxt[pi].TableExprTxt) {

			if !isMatch { // if no match then use default
				ni = nf
			}
			if ni < len(txtMeta.TableExprTxt) {
				mt.TableTxt[pi].TableExprTxt[ci].DescrNote = aDescrNote{
					LangCode: &txtMeta.TableExprTxt[ni].LangCode,
					Descr:    &txtMeta.TableExprTxt[ni].Descr,
					Note:     &txtMeta.TableExprTxt[ni].Note}
			}
		}
	}

	// set entity description and notes
	if len(mt.EntityTxt) > 0 && len(txtMeta.EntityTxt) > 0 {

		var isKey, isFound, isMatch bool
		var nf, ni, si, di int

		for ; si < len(txtMeta.EntityTxt); si++ {

			// destination rows must be defined by [di] index
			if di >= len(mt.EntityTxt) {
				break // done with all destination text
			}

			// check if source and destination keys equal
			mId := mt.EntityTxt[di].Entity.ModelId
			tId := mt.EntityTxt[di].Entity.EntityId

			isKey = txtMeta.EntityTxt[si].ModelId == mId &&
				txtMeta.EntityTxt[si].EntityId == tId

			// start of next key: set value
			if !isKey && isFound {

				if !isMatch { // if no match then use default
					ni = nf
				}
				mt.EntityTxt[di].DescrNote = aDescrNote{
					LangCode: &txtMeta.EntityTxt[ni].LangCode,
					Descr:    &txtMeta.EntityTxt[ni].Descr,
					Note:     &txtMeta.EntityTxt[ni].Note}

				// reset to start next search
				isFound = false
				isMatch = false
				di++ // move to next entity
				si-- // repeat current source row
				continue
			}

			// inside of key
			if isKey {

				if !isFound {
					isFound = true // first key found
					nf = si
				}
				// match the language
				isMatch = txtMeta.EntityTxt[si].LangCode == lc
				if isMatch {
					ni = si // perefred language match
				}
				if txtMeta.EntityTxt[si].LangCode == lcd {
					nf = si // index of default language
				}
			}

			// if keys not equal and destination key behind source
			// then move to next destination row and repeat current source row
			if !isKey &&
				(txtMeta.EntityTxt[si].ModelId > mId ||
					txtMeta.EntityTxt[si].ModelId == mId &&
						txtMeta.EntityTxt[si].EntityId > tId) {

				di++ // move to next entity
				si-- // repeat current source row
				continue
			}
		} // for

		// last row
		if isFound && di < len(mt.EntityTxt) {

			if !isMatch { // if no match then use default
				ni = nf
			}
			if ni < len(txtMeta.EntityTxt) {
				mt.EntityTxt[di].DescrNote = aDescrNote{
					LangCode: &txtMeta.EntityTxt[ni].LangCode,
					Descr:    &txtMeta.EntityTxt[ni].Descr,
					Note:     &txtMeta.EntityTxt[ni].Note}
			}
		}
	}

	// set entity attributes description and notes
	if len(mt.EntityTxt) > 0 && len(txtMeta.EntityAttrTxt) > 0 {

		var isKey, isFound, isMatch bool
		var nf, ni, si, pi, ci int

		for ; si < len(txtMeta.EntityAttrTxt); si++ {

			// destination rows: parent and child must be defined by valid (pi, ci) indexes
			if pi >= len(mt.EntityTxt) {
				break // done with all destination text
			}
			if pi < len(mt.EntityTxt) &&
				ci >= len(mt.EntityTxt[pi].EntityAttrTxt) {

				if pi++; pi >= len(mt.EntityTxt) {
					break // done with all destination text
				}

				ci = 0 // move to next type
				si--   // repeat current source row
				continue
			}

			// check if source and destination keys equal
			mId := mt.EntityTxt[pi].Entity.ModelId
			eId := mt.EntityTxt[pi].Entity.EntityId
			aId := mt.EntityTxt[pi].EntityAttrTxt[ci].Attr.AttrId

			isKey = txtMeta.EntityAttrTxt[si].ModelId == mId &&
				txtMeta.EntityAttrTxt[si].EntityId == eId &&
				txtMeta.EntityAttrTxt[si].AttrId == aId

			// start of next key: set value
			if !isKey && isFound {

				if !isMatch { // if no match then use default
					ni = nf
				}
				mt.EntityTxt[pi].EntityAttrTxt[ci].DescrNote = aDescrNote{
					LangCode: &txtMeta.EntityAttrTxt[ni].LangCode,
					Descr:    &txtMeta.EntityAttrTxt[ni].Descr,
					Note:     &txtMeta.EntityAttrTxt[ni].Note}

				// reset to start next search
				isFound = false
				isMatch = false
				ci++ // move to next type
				si-- // repeat current source row
				continue
			}

			// inside of key
			if isKey {

				if !isFound {
					isFound = true // first key found
					nf = si
				}
				// match the language
				isMatch = txtMeta.EntityAttrTxt[si].LangCode == lc
				if isMatch {
					ni = si // perefred language match
				}
				if txtMeta.EntityAttrTxt[si].LangCode == lcd {
					nf = si // index of default language
				}
			}

			// if keys not equal and destination key behind source
			// then move to next destination row and repeat current source row
			if !isKey &&
				(txtMeta.EntityAttrTxt[si].ModelId > mId ||
					txtMeta.EntityAttrTxt[si].ModelId == mId &&
						txtMeta.EntityAttrTxt[si].EntityId > eId ||
					txtMeta.EntityAttrTxt[si].ModelId == mId &&
						txtMeta.EntityAttrTxt[si].EntityId == eId &&
						txtMeta.EntityAttrTxt[si].AttrId > aId) {

				ci++ // move to next type
				si-- // repeat current source row
				continue
			}
		} // for

		// last row
		if isFound && pi < len(mt.EntityTxt) && ci < len(mt.EntityTxt[pi].EntityAttrTxt) {

			if !isMatch { // if no match then use default
				ni = nf
			}
			if ni < len(txtMeta.EntityAttrTxt) {
				mt.EntityTxt[pi].EntityAttrTxt[ci].DescrNote = aDescrNote{
					LangCode: &txtMeta.EntityAttrTxt[ni].LangCode,
					Descr:    &txtMeta.EntityAttrTxt[ni].Descr,
					Note:     &txtMeta.EntityAttrTxt[ni].Note}
			}
		}
	}

	// set group description and notes
	if len(mt.GroupTxt) > 0 && len(txtMeta.GroupTxt) > 0 {

		var isKey, isFound, isMatch bool
		var nf, ni, si, di int

		for ; si < len(txtMeta.GroupTxt); si++ {

			// destination rows must be defined by [di] index
			if di >= len(mt.GroupTxt) {
				break // done with all destination text
			}

			// check if source and destination keys equal
			mId := mt.GroupTxt[di].Group.ModelId
			gId := mt.GroupTxt[di].Group.GroupId

			isKey = txtMeta.GroupTxt[si].ModelId == mId &&
				txtMeta.GroupTxt[si].GroupId == gId

			// start of next key: set value
			if !isKey && isFound {

				if !isMatch { // if no match then use default
					ni = nf
				}
				mt.GroupTxt[di].DescrNote = aDescrNote{
					LangCode: &txtMeta.GroupTxt[ni].LangCode,
					Descr:    &txtMeta.GroupTxt[ni].Descr,
					Note:     &txtMeta.GroupTxt[ni].Note}

				// reset to start next search
				isFound = false
				isMatch = false
				di++ // move to next group
				si-- // repeat current source row
				continue
			}

			// inside of key
			if isKey {

				if !isFound {
					isFound = true // first key found
					nf = si
				}
				// match the language
				isMatch = txtMeta.GroupTxt[si].LangCode == lc
				if isMatch {
					ni = si // perefred language match
				}
				if txtMeta.GroupTxt[si].LangCode == lcd {
					nf = si // index of default language
				}
			}

			// if keys not equal and destination key behind source
			// then move to next destination row and repeat current source row
			if !isKey &&
				(txtMeta.GroupTxt[si].ModelId > mId ||
					txtMeta.GroupTxt[si].ModelId == mId &&
						txtMeta.GroupTxt[si].GroupId > gId) {

				di++ // move to next group
				si-- // repeat current source row
				continue
			}
		} // for

		// last row
		if isFound && di < len(mt.GroupTxt) {

			if !isMatch { // if no match then use default
				ni = nf
			}
			if ni < len(txtMeta.GroupTxt) {
				mt.GroupTxt[di].DescrNote = aDescrNote{
					LangCode: &txtMeta.GroupTxt[ni].LangCode,
					Descr:    &txtMeta.GroupTxt[ni].Descr,
					Note:     &txtMeta.GroupTxt[ni].Note}
			}
		}
	}

	// set entity group description and notes
	if len(mt.EntityGroupTxt) > 0 && len(txtMeta.EntityGroupTxt) > 0 {

		var isKey, isFound, isMatch bool
		var nf, ni, si, di int

		for ; si < len(txtMeta.EntityGroupTxt); si++ {

			// destination rows must be defined by [di] index
			if di >= len(mt.EntityGroupTxt) {
				break // done with all destination text
			}

			// check if source and destination keys equal
			mId := mt.EntityGroupTxt[di].Group.ModelId
			eId := mt.EntityGroupTxt[di].Group.EntityId
			gId := mt.EntityGroupTxt[di].Group.GroupId

			isKey = txtMeta.EntityGroupTxt[si].ModelId == mId &&
				txtMeta.EntityGroupTxt[si].EntityId == eId &&
				txtMeta.EntityGroupTxt[si].GroupId == gId

			// start of next key: set value
			if !isKey && isFound {

				if !isMatch { // if no match then use default
					ni = nf
				}
				mt.EntityGroupTxt[di].DescrNote = aDescrNote{
					LangCode: &txtMeta.EntityGroupTxt[ni].LangCode,
					Descr:    &txtMeta.EntityGroupTxt[ni].Descr,
					Note:     &txtMeta.EntityGroupTxt[ni].Note}

				// reset to start next search
				isFound = false
				isMatch = false
				di++ // move to next group
				si-- // repeat current source row
				continue
			}

			// inside of key
			if isKey {

				if !isFound {
					isFound = true // first key found
					nf = si
				}
				// match the language
				isMatch = txtMeta.EntityGroupTxt[si].LangCode == lc
				if isMatch {
					ni = si // perefred language match
				}
				if txtMeta.EntityGroupTxt[si].LangCode == lcd {
					nf = si // index of default language
				}
			}

			// if keys not equal and destination key behind source
			// then move to next destination row and repeat current source row
			if !isKey &&
				(txtMeta.EntityGroupTxt[si].ModelId > mId ||
					txtMeta.EntityGroupTxt[si].ModelId == mId && txtMeta.EntityGroupTxt[si].EntityId > eId ||
					txtMeta.EntityGroupTxt[si].ModelId == mId && txtMeta.EntityGroupTxt[si].EntityId == eId && txtMeta.EntityGroupTxt[si].GroupId > gId) {

				di++ // move to next group
				si-- // repeat current source row
				continue
			}
		} // for

		// last row
		if isFound && di < len(mt.EntityGroupTxt) {

			if !isMatch { // if no match then use default
				ni = nf
			}
			if ni < len(txtMeta.EntityGroupTxt) {
				mt.EntityGroupTxt[di].DescrNote = aDescrNote{
					LangCode: &txtMeta.EntityGroupTxt[ni].LangCode,
					Descr:    &txtMeta.EntityGroupTxt[ni].Descr,
					Note:     &txtMeta.EntityGroupTxt[ni].Note}
			}
		}
	}

	me.MetaDescrNote = mt
	return nil
}

type aDescrNote struct {
	LangCode *string // lang_code VARCHAR(32)  NOT NULL
	Descr    *string // descr     VARCHAR(255) NOT NULL
	Note     *string // note      VARCHAR(32000)
}

// typeEnumDescrNote is join of type_enum_lst, model_type_dic, type_enum_txt
type typeEnumDescrNote struct {
	Enum      *db.TypeEnumRow // type enum row: type_enum_lst join to model_type_dic
	DescrNote aDescrNote      // from type_enum_txt
}

// TypeDescrNote is join of type_dic_txt, model_type_dic, type_dic_txt
type typeUnpackDescrNote struct {
	Type        *db.TypeDicRow      // model type row: type_dic join to model_type_dic
	DescrNote   *aDescrNote         // from type_dic_txt
	TypeEnumTxt []typeEnumDescrNote // type enum text rows: type_enum_txt join to model_type_dic
	langCode    string              // language for description and notes
}

// marshal type text metadata to json, "unpack" range enums which may be not loaded from database
func (src *typeUnpackDescrNote) MarshalJSON() ([]byte, error) {

	tm := struct {
		Type        *db.TypeDicRow
		DescrNote   *aDescrNote
		TypeEnumTxt []typeEnumDescrNote // type enum text rows: type_enum_txt join to model_type_dic
	}{
		Type:        src.Type,
		DescrNote:   src.DescrNote,
		TypeEnumTxt: src.TypeEnumTxt,
	}
	// if type not a range or enums loaded from database then use standard json marshal
	if !tm.Type.IsRange {
		return json.Marshal(tm)
	}
	if len(tm.TypeEnumTxt) > 0 {
		return json.Marshal(tm) // all range enums are loaded from database
	}
	// else it is a range type and there no enums: marshal array of [min, max] enum Id, Name, Descr

	n := 1 + (tm.Type.MaxEnumId - tm.Type.MinEnumId)
	tm.TypeEnumTxt = make([]typeEnumDescrNote, n)
	emptyNote := ""

	for k := 0; k < n; k++ {

		nId := k + tm.Type.MinEnumId
		et := typeEnumDescrNote{
			Enum: &db.TypeEnumRow{
				ModelId: tm.Type.ModelId,
				TypeId:  tm.Type.TypeId,
				EnumId:  nId,
				Name:    strconv.Itoa(nId),
			},
			DescrNote: aDescrNote{
				LangCode: &src.langCode,
				Note:     &emptyNote,
			},
		}
		et.DescrNote.Descr = &et.Enum.Name // for range type enum code same as description and same as enum id

		tm.TypeEnumTxt[k] = et
	}

	return json.Marshal(tm)
}

// marshal enum text to json, if not description or notes not epty, otherwise return "DescrNote": null
func (src typeEnumDescrNote) MarshalJSON() ([]byte, error) {

	tm := struct {
		Enum      *db.TypeEnumRow
		DescrNote *aDescrNote
	}{
		Enum: src.Enum,
	}
	if src.DescrNote.Descr != nil && *src.DescrNote.Descr != "" || src.DescrNote.Note != nil && *src.DescrNote.Note != "" {
		tm.DescrNote = &src.DescrNote
	}

	return json.Marshal(tm)
}
