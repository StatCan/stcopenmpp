// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

// maxBuiltInTypeId is max type id and Hid for openM++ built-in types, ie: int, double, logical
const maxBuiltInTypeId = 100

// TotalEnumCode is predefined enum code for "total" enum
const TotalEnumCode = "all"

const codeDbMax = 32      // max database length for codes: language code, digests, date-time string
const nameDbMax = 255     // max database length for names: parameter name, table name, etc.
const descrDbMax = 255    // max database length for description: parameter description, table description, etc.
const wordDbMax = 255     // max database length for word: word_code, word_value
const optionDbMax = 32000 // max database length for option value: profile_option, run_option
const noteDbMax = 32000   // max database notes length: notes varchar (clob, text)
const stringDbMax = 32000 // max database string length: parameter varchar (clob, text)
const rangeDicId = 3      // range type

// ModelMeta is model metadata db rows, language-neutral portion of it.
//
// Types, parameters and output tables can be shared between different models and even between different databases.
// Use digest hash to find same type (parameter, table or model) in other database.
// As it is today language-specific part of model metadata (labels, description, notes, etc.)
// does not considered for "equality" comparison and not included in digest.
//
// For example, logical type consists of 2 enum (code, value) pairs: [(0, "false") (1, "true")] and
// even it has different labels in different databases, i.e. (1, "Truth") vs (1, "OK")
// such type(s) considered the same and should have identical digest(s).
//
// Inside of database *_hid (type_hid, parameter_hid, table_hid) is a unique id of corresponding object (primary key).
// Those _hid's are database-unique and should be used to find same type (parameter, output table) in other database.
// Also each type, parameter, output table have model-unique *_id (type_id, parameter_id, table_id)
// assigned by compiler and it is possible to find type, parameter or table by combination of (model_id, type_id).
//
// Unless otherwise specified each array is ordered by model-specific id's and binary search can be used.
// For example type array is ordered by (model_id, type_id) and type enum array by (model_id, type_id, enum_id).
//
type ModelMeta struct {
	Model       ModelDicRow       // model_dic table row
	Type        []TypeMeta        // types metadata: type name and enums
	Param       []ParamMeta       // parameters metadata: parameter name, type, dimensions
	Table       []TableMeta       // output tables metadata: table name, dimensions, accumulators, expressions
	Entity      []EntityMeta      // model entities and attributes
	Group       []GroupMeta       // groups of parameters or output tables
	EntityGroup []EntityGroupMeta // entity groups of attributes
}

// ModelTxtMeta is language-specific portion of model metadata db rows.
type ModelTxtMeta struct {
	ModelName      string              // model name for text metadata
	ModelDigest    string              // model digest for text metadata
	ModelTxt       []ModelTxtRow       // model text rows: model_dic_txt
	TypeTxt        []TypeTxtRow        // model type text rows: type_dic_txt join to model_type_dic
	TypeEnumTxt    []TypeEnumTxtRow    // type enum text rows: type_enum_txt join to model_type_dic
	ParamTxt       []ParamTxtRow       // model parameter text rows: parameter_dic_txt join to model_parameter_dic
	ParamDimsTxt   []ParamDimsTxtRow   // parameter dimension text rows: parameter_dims_txt join to model_parameter_dic
	TableTxt       []TableTxtRow       // model output table text rows: table_dic_txt join to model_table_dic
	TableDimsTxt   []TableDimsTxtRow   // output table dimension text rows: table_dims_txt join to model_table_dic
	TableAccTxt    []TableAccTxtRow    // output table accumulator text rows: table_acc_txt join to model_table_dic
	TableExprTxt   []TableExprTxtRow   // output table expression text rows: table_expr_txt join to model_table_dic
	EntityTxt      []EntityTxtRow      // model entities text rows: entity_dic_txt join to model_entity_dic table
	EntityAttrTxt  []EntityAttrTxtRow  // entity attributes: entity_attr_txt join to model_entity_dic table
	GroupTxt       []GroupTxtRow       // group text rows: group_txt
	EntityGroupTxt []EntityGroupTxtRow // entity group text rows: entity_group_txt
}

// TypeMeta is type metadata: type name and enums
type TypeMeta struct {
	TypeDicRow               // model type rows: type_dic join to model_type_dic
	Enum       []TypeEnumRow // type enum rows: type_enum_lst join to model_type_dic
}

// ParamMeta is parameter metadata: parameter name, type, dimensions
type ParamMeta struct {
	ParamDicRow                  // model parameter row: parameter_dic join to model_parameter_dic table
	Dim         []ParamDimsRow   // parameter dimension rows: parameter_dims join to model_parameter_dic table
	Import      []ParamImportRow // parameter import from upstream model
	typeOf      *TypeMeta        // type of parameter
	sizeOf      int              // size of parameter: db row count calculated as dimension(s) size product
}

// TableMeta is output table metadata: table name, dimensions, accumulators, expressions
type TableMeta struct {
	TableDicRow                // model output table row: table_dic join to model_table_dic
	Dim         []TableDimsRow // output table dimension rows: table_dims join to model_table_dic
	Acc         []TableAccRow  // output table accumulator rows: table_acc join to model_table_dic
	Expr        []TableExprRow // output table expression rows: table_expr join to model_table_dic
	sizeOf      int            // db row count calculated as dimension(s) size product
}

// ModelWordMeta is language-specific model_word db rows.
type ModelWordMeta struct {
	ModelName   string          // model name for text metadata
	ModelDigest string          // model digest for text metadata
	ModelWord   []ModelLangWord // language and db rows of model_word in that language
}

// ProfileMeta is rows from profile_option table.
//
// Profile is a named group of (key, value) options, similar to ini-file.
// Default model options has profile_name = model_name.
type ProfileMeta struct {
	Name string            // profile name
	Opts map[string]string // profile (key, value) options
}

// LangMeta is languages and words for each language
type LangMeta struct {
	Lang      []LangWord     // language lang_lst row and lang_word rows in that language
	idIndex   map[int]int    // language id index
	codeIndex map[string]int // language code index
}

// LangLstRow is db row of lang_lst table.
//
// langId is db-unique id of the language.
// LangCode is unique language code: EN, FR.
type LangLstRow struct {
	langId   int    // lang_id   INT          NOT NULL
	LangCode string // lang_code VARCHAR(32)  NOT NULL
	Name     string // lang_name VARCHAR(255) NOT NULL
}

// language lang_lst row and lang_word rows in that language
type LangWord struct {
	LangLstRow                   // lang_lst db-table row
	Words      map[string]string // lang_word db-table rows as (code, value) map
}

// language and db rows of model_word in that language
type ModelLangWord struct {
	LangCode string            // lang_code    VARCHAR(32)  NOT NULL
	Words    map[string]string // model_word db-table rows as (code, value) map
}

// language and translated strings in that language:
// merge of model.message.ini, common.message.ini, model_word, lang_word
// it can have extra Lang code from ini files, which does not exists in lang_lst
type LangMsg struct {
	LangCode string            // language code e.g.: lang_code VARCHAR(32)  NOT NULL
	Msg      map[string]string // translated strings map (code, label)
}

// DescrNote is a holder for language code, descripriton and notes
type DescrNote struct {
	LangCode string // lang_code VARCHAR(32)  NOT NULL
	Descr    string // descr     VARCHAR(255) NOT NULL
	Note     string // note      VARCHAR(32000)
}

// LangNote is a holder for language code and notes
type LangNote struct {
	LangCode string // lang_code VARCHAR(32)  NOT NULL
	Note     string // note      VARCHAR(32000)
}

// ModelDicRow is db row of model_dic table.
//
// ModelId (model_dic.model_id) is db-unique id of the model, use digest to find same model in other db.
type ModelDicRow struct {
	ModelId         int    // model_id         INT          NOT NULL
	Name            string // model_name       VARCHAR(255) NOT NULL
	Digest          string // model_digest     VARCHAR(32)  NOT NULL
	Type            int    // model_type       INT          NOT NULL
	Version         string // model_ver        VARCHAR(32)  NOT NULL
	CreateDateTime  string // create_dt        VARCHAR(32)  NOT NULL
	DefaultLangCode string // model default language code
}

// ModelTxtRow is db row of model_dic_txt join to model_dic
type ModelTxtRow struct {
	ModelId  int    // model_id     INT          NOT NULL
	LangCode string // lang_code    VARCHAR(32)  NOT NULL
	Descr    string // descr        VARCHAR(255) NOT NULL
	Note     string // note         VARCHAR(32000)
}

// ModelDicDescrNote is join of model_dic db row and model_dic_txt row
type ModelDicDescrNote struct {
	Model     ModelDicRow // model_dic db row
	DescrNote DescrNote   // from model_dic_txt
}

// TypeDicRow is db row of type_dic join to model_type_dic table and min, max, count of enum id's.
//
// TypeHid (type_dic.type_hid) is db-unique id of the type, use digest to find same type in other db.
// TypeId (model_type_dic.model_type_id) is model-unique type id, assigned by model compiler.
type TypeDicRow struct {
	ModelId     int    // model_id      INT          NOT NULL
	TypeId      int    // model_type_id INT          NOT NULL
	TypeHid     int    // type_hid      INT          NOT NULL, -- unique type id
	Name        string // type_name     VARCHAR(255) NOT NULL, -- type name: int, double, etc.
	Digest      string // type_digest   VARCHAR(32)  NOT NULL
	DicId       int    // dic_id        INT NOT NULL, -- dictionary id: 0=simple 1=logical 2=classification 3=range 4=partition 5=link
	TotalEnumId int    // total_enum_id INT NOT NULL, -- if total enabled this is enum_value of total item =max+1
	IsRange     bool   // if true then it is range type and enums calculated
	MinEnumId   int    // min enum id
	MaxEnumId   int    // max enum id
	sizeOf      int    // number of enums
}

// TypeTxtRow is db row of type_dic_txt join to model_type_dic table
type TypeTxtRow struct {
	ModelId  int    // model_id      INT          NOT NULL
	TypeId   int    // model_type_id INT          NOT NULL
	LangCode string // lang_code     VARCHAR(32)  NOT NULL
	Descr    string // descr         VARCHAR(255) NOT NULL
	Note     string // note          VARCHAR(32000)
}

// TypeEnumRow is db row of type_enum_lst join to model_type_dic table
type TypeEnumRow struct {
	ModelId int    // model_id      INT NOT NULL
	TypeId  int    // model_type_id INT NOT NULL
	EnumId  int    // enum_id       INT NOT NULL
	Name    string // enum_name     VARCHAR(255) NOT NULL
}

// TypeEnumTxtRow is db row of type_enum_txt join to model_type_dic table
type TypeEnumTxtRow struct {
	ModelId  int    // model_id      INT          NOT NULL
	TypeId   int    // model_type_id INT          NOT NULL
	EnumId   int    // enum_id       INT          NOT NULL
	LangCode string // lang_code     VARCHAR(32)  NOT NULL
	Descr    string // descr         VARCHAR(255) NOT NULL
	Note     string // note          VARCHAR(32000)
}

// ParamDicRow is db row of parameter_dic join to model_parameter_dic table
//
// ParamHid (parameter_dic.parameter_hid) is db-unique id of the parameter, use digest to find same parameter in other db.
// ParamId (model_parameter_dic.model_parameter_id) is model-unique parameter id, assigned by model compiler.
type ParamDicRow struct {
	ModelId      int    // model_id           INT          NOT NULL
	ParamId      int    // model_parameter_id INT          NOT NULL
	ParamHid     int    // parameter_hid      INT          NOT NULL, -- unique parameter id
	Name         string // parameter_name     VARCHAR(255) NOT NULL
	Digest       string // parameter_digest   VARCHAR(32)  NOT NULL
	Rank         int    // parameter_rank     INT          NOT NULL
	TypeId       int    // model_type_id      INT          NOT NULL
	IsExtendable bool   // is_extendable      SMALLINT     NOT NULL
	IsHidden     bool   // is_hidden          SMALLINT     NOT NULL
	NumCumulated int    // num_cumulated      INT          NOT NULL
	DbRunTable   string // db_run_table       VARCHAR(64)  NOT NULL
	DbSetTable   string // db_set_table       VARCHAR(64)  NOT NULL
	ImportDigest string // import_digest      VARCHAR(32)  NOT NULL
}

// ParamImportRow is db row of model_parameter_import table
type ParamImportRow struct {
	ModelId     int    // model_id           INT          NOT NULL
	ParamId     int    // model_parameter_id INT          NOT NULL
	FromName    string // from_name          VARCHAR(255) NOT NULL
	FromModel   string // from_model_name    VARCHAR(255) NOT NULL
	IsSampleDim bool   // is_sample_dim      SMALLINT     NOT NULL
}

// ParamTxtRow is db row of parameter_dic_txt join to model_parameter_dic table
type ParamTxtRow struct {
	ModelId  int    // model_id           INT          NOT NULL
	ParamId  int    // model_parameter_id INT          NOT NULL
	LangCode string // lang_code          VARCHAR(32)  NOT NULL
	Descr    string // descr              VARCHAR(255) NOT NULL
	Note     string // note               VARCHAR(32000)
}

// ParamDimsRow is db row of parameter_dims join to model_parameter_dic table
type ParamDimsRow struct {
	ModelId int       // model_id           INT          NOT NULL
	ParamId int       // model_parameter_id INT          NOT NULL
	DimId   int       // dim_id             INT          NOT NULL
	Name    string    // dim_name           VARCHAR(255) NOT NULL
	TypeId  int       // model_type_id      INT          NOT NULL
	typeOf  *TypeMeta // type of dimension
	sizeOf  int       // dimension size as enum count, zero if type is simple
	colName string    // db column name: dim1
}

// ParamDimsTxtRow is db row of parameter_dims_txt join to model_parameter_dic table
type ParamDimsTxtRow struct {
	ModelId  int    // model_id           INT          NOT NULL
	ParamId  int    // model_parameter_id INT          NOT NULL
	DimId    int    // dim_id             INT          NOT NULL
	LangCode string // lang_code          VARCHAR(32)  NOT NULL
	Descr    string // descr              VARCHAR(255) NOT NULL
	Note     string // note               VARCHAR(32000)
}

// TableDicRow is db row of table_dic join to model_table_dic table.
//
// TableHid (table_dic.table_hid) is db-unique id of the output table, use digest to find same table in other db.
// TableId (model_table_dic.model_table_id) is model-unique output table id, assigned by model compiler.
type TableDicRow struct {
	ModelId      int    // model_id        INT          NOT NULL
	TableId      int    // model_table_id  INT          NOT NULL
	TableHid     int    // table_hid       INT          NOT NULL, -- unique table id
	Name         string // table_name      VARCHAR(255) NOT NULL
	Digest       string // table_digest    VARCHAR(32)  NOT NULL
	IsUser       bool   // is_user         SMALLINT     NOT NULL
	Rank         int    // table_rank      INT          NOT NULL
	IsSparse     bool   // is_sparse       SMALLINT     NOT NULL
	DbExprTable  string // db_expr_table   VARCHAR(64)  NOT NULL
	DbAccTable   string // db_acc_table    VARCHAR(64)  NOT NULL
	DbAccAllView string // db_acc_all_view VARCHAR(64)  NOT NULL
	ExprPos      int    // expr_dim_pos    INT          NOT NULL
	IsHidden     bool   // is_hidden       SMALLINT     NOT NULL
	ImportDigest string // import_digest   VARCHAR(32)  NOT NULL
}

// TableTxtRow is db row of table_dic_txt join to model_table_dic table
type TableTxtRow struct {
	ModelId   int    // model_id       INT          NOT NULL
	TableId   int    // model_table_id INT          NOT NULL
	LangCode  string // lang_code      VARCHAR(32)  NOT NULL
	Descr     string // descr          VARCHAR(255) NOT NULL
	Note      string // note           VARCHAR(32000)
	ExprDescr string // expr_descr     VARCHAR(255) NOT NULL
	ExprNote  string // expr_note      VARCHAR(32000)
}

// TableDimsRow is db row of table_dims join to model_table_dic table
type TableDimsRow struct {
	ModelId int       // model_id       INT          NOT NULL
	TableId int       // model_table_id INT          NOT NULL
	DimId   int       // dim_id         INT          NOT NULL
	Name    string    // dim_name       VARCHAR(255) NOT NULL
	TypeId  int       // model_type_id  INT          NOT NULL
	IsTotal bool      // is_total       SMALLINT     NOT NULL
	DimSize int       // dim_size       INT          NOT NULL
	typeOf  *TypeMeta // type of dimension
	colName string    // db column name: dim1
}

// TableDimsTxtRow is db row of table_dims_txt join to model_table_dic table
type TableDimsTxtRow struct {
	ModelId  int    // model_id       INT          NOT NULL
	TableId  int    // model_table_id INT          NOT NULL
	DimId    int    // dim_id         INT          NOT NULL
	LangCode string // lang_code      VARCHAR(32)  NOT NULL
	Descr    string // descr          VARCHAR(255) NOT NULL
	Note     string // note           VARCHAR(32000)
}

// TableAccRow is db row of table_acc join to model_table_dic table
type TableAccRow struct {
	ModelId   int    // model_id       INT          NOT NULL
	TableId   int    // model_table_id INT           NOT NULL
	AccId     int    // acc_id         INT           NOT NULL
	Name      string // acc_name       VARCHAR(255)  NOT NULL
	IsDerived bool   // is_derived     SMALLINT      NOT NULL
	SrcAcc    string // acc_src        VARCHAR(255)  NOT NULL
	AccSql    string // acc_sql        VARCHAR(2048) NOT NULL
	colName   string // internal db column name: expr1
}

// TableAccTxtRow is db row of table_acc_txt join to model_table_dic table
type TableAccTxtRow struct {
	ModelId  int    // model_id       INT          NOT NULL
	TableId  int    // model_table_id INT          NOT NULL
	AccId    int    // acc_id         INT          NOT NULL
	LangCode string // lang_code      VARCHAR(32)  NOT NULL
	Descr    string // descr          VARCHAR(255) NOT NULL
	Note     string // note           VARCHAR(32000)
}

// TableExprRow is db row of table_expr join to model_table_dic table
type TableExprRow struct {
	ModelId  int    // model_id       INT           NOT NULL
	TableId  int    // model_table_id INT           NOT NULL
	ExprId   int    // expr_id        INT           NOT NULL
	Name     string // expr_name      VARCHAR(255)  NOT NULL
	Decimals int    // expr_decimals  INT           NOT NULL
	SrcExpr  string // expr_src       VARCHAR(255)  NOT NULL
	ExprSql  string // expr_sql       VARCHAR(2048) NOT NULL
	colName  string // internal db column name: expr1
}

// TableExprTxtRow is db row of table_expr_txt join to model_table_dic table
type TableExprTxtRow struct {
	ModelId  int    // model_id       INT          NOT NULL
	TableId  int    // model_table_id INT          NOT NULL
	ExprId   int    // expr_id        INT           NOT NULL
	LangCode string // lang_code      VARCHAR(32)  NOT NULL
	Descr    string // descr          VARCHAR(255) NOT NULL
	Note     string // note           VARCHAR(32000)
}

// EntityMeta is entity metadata: entity name, digest, attributes
type EntityMeta struct {
	EntityDicRow                 // model entity row: entity_dic join to model_entity_dic table
	Attr         []EntityAttrRow // entity attribute rows: entity_attr join to model_entity_dic table
}

// EntityDicRow is db row of entity_dic join to model_entity_dic table.
//
// EntityHid (entity_dic.entity_hid) is db-unique id of the entity, use digest to find same table in other db.
// EntityId (model_entity_dic.model_entity_id) is model-unique entity id, assigned by model compiler.
type EntityDicRow struct {
	ModelId   int    // model_id         INT          NOT NULL
	EntityId  int    // model_entity_id  INT          NOT NULL
	EntityHid int    // entity_hid       INT          NOT NULL, -- unique entity id
	Name      string // entity_name      VARCHAR(255) NOT NULL
	Digest    string // entity_digest    VARCHAR(32)  NOT NULL
}

// EntityTxtRow is db row of entity_dic_txt join to model_entity_dic table
type EntityTxtRow struct {
	ModelId  int    // model_id         INT          NOT NULL
	EntityId int    // model_entity_id  INT          NOT NULL
	LangCode string // lang_code        VARCHAR(32)  NOT NULL
	Descr    string // descr            VARCHAR(255) NOT NULL
	Note     string // note             VARCHAR(32000)
}

// EntityAttrRow is db row of entity_attr join to model_entity_dic table
type EntityAttrRow struct {
	ModelId    int       // model_id        INT          NOT NULL
	EntityId   int       // model_entity_id INT          NOT NULL
	AttrId     int       // attr_id         INT          NOT NULL
	Name       string    // attr_name       VARCHAR(255) NOT NULL
	TypeId     int       // model_type_id   INT          NOT NULL
	IsInternal bool      // is_internal     SMALLINT     NOT NULL
	typeOf     *TypeMeta // type of attribute
	colName    string    // db column name: attr1
}

// EntityAttrTxtRow is db row of entity_attr_txt join to model_entity_dic table
type EntityAttrTxtRow struct {
	ModelId  int    // model_id        INT          NOT NULL
	EntityId int    // model_entity_id INT          NOT NULL
	AttrId   int    // attr_id         INT          NOT NULL
	LangCode string // lang_code       VARCHAR(32)  NOT NULL
	Descr    string // descr           VARCHAR(255) NOT NULL
	Note     string // note            VARCHAR(32000)
}

// GroupMeta is db rows to describe parent-child group of parameters or output tables,
// it is join of group_lst to group_pc
type GroupMeta struct {
	GroupLstRow              // parameters or output tables group rows: group_lst
	GroupPc     []GroupPcRow // group parent-child relationship rows: group_pc
}

// GroupLstRow is db row of group_lst table
type GroupLstRow struct {
	ModelId  int    // model_id     INT          NOT NULL
	GroupId  int    // group_id     INT          NOT NULL
	IsParam  bool   // is_parameter SMALLINT     NOT NULL, -- if <> 0 then parameter group else output table group
	Name     string // group_name   VARCHAR(255) NOT NULL
	IsHidden bool   // is_hidden    SMALLINT     NOT NULL
}

// GroupPcRow is db row of group_pc table
type GroupPcRow struct {
	ModelId      int // model_id       INT NOT NULL
	GroupId      int // group_id       INT NOT NULL
	ChildPos     int // child_pos      INT NOT NULL
	ChildGroupId int // child_group_id INT NULL, -- if not NULL then id of child group
	ChildLeafId  int // leaf_id        INT NULL, -- if not NULL then id of parameter or output table
}

// GroupTxtRow is db row of group_txt table
type GroupTxtRow struct {
	ModelId   int // model_id  INT          NOT NULL
	GroupId   int // group_id  INT          NOT NULL
	DescrNote     // language, description, notes
}

// EntityGroupMeta is db rows to describe parent-child group of entity attributes,
// it is join of entity_group_lst to entity_group_pc
type EntityGroupMeta struct {
	EntityGroupLstRow                    // entity attribute group rows: entity_group_lst
	GroupPc           []EntityGroupPcRow // group parent-child relationship rows: entity_group_pc
}

// EntityGroupLstRow is db row of entity_group_lst table
type EntityGroupLstRow struct {
	ModelId  int    // model_id        INT          NOT NULL
	EntityId int    // model_entity_id INT          NOT NULL
	GroupId  int    // group_id        INT          NOT NULL
	Name     string // group_name      VARCHAR(255) NOT NULL
	IsHidden bool   // is_hidden       SMALLINT     NOT NULL
}

// EntityGroupPcRow is db row of entity_group_pc table
type EntityGroupPcRow struct {
	ModelId      int // model_id        INT NOT NULL
	EntityId     int // model_entity_id INT NOT NULL
	GroupId      int // group_id        INT NOT NULL
	ChildPos     int // child_pos       INT NOT NULL
	ChildGroupId int // child_group_id  INT, -- if not IS NULL then child group
	AttrId       int // attr_id         INT, -- if not IS NULL then entity attribute index
}

// EntityGroupTxtRow is db row of entity_group_txt table
type EntityGroupTxtRow struct {
	ModelId   int // model_id        INT NOT NULL
	EntityId  int // model_entity_id INT NOT NULL
	GroupId   int // group_id        INT NOT NULL
	DescrNote     // language, description, notes
}
