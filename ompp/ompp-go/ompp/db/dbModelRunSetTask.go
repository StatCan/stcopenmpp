// Copyright (c) 2016 OpenM++
// This code is licensed under the MIT license (see LICENSE.txt for details)

package db

// Model run part of database: run, workset, task, profile.
//
// That portion of model database updated during model run and should not be cached.

// Model run status (run_lst table) and modeling task run status (task_run_lst table):
//   if task status = w (wait) then
//      model wait and NOT completed until other process set status to one of finals: s,x,e
//      model check if any new sets inserted into task_set and run it as they arrive
const (
	InitRunStatus     = "i" // i = initial status
	ProgressRunStatus = "p" // p = run in progress
	WaitRunStatus     = "w" // w = wait: run in progress, under external supervision
	DoneRunStatus     = "s" // s = completed successfully
	ExitRunStatus     = "x" // x = exit and not completed
	ErrorRunStatus    = "e" // e = error failure
	DeleteRunStatus   = "d" // d = deleted model run
)

// RunMeta is metadata of model run: name, status, run options, description, notes.
type RunMeta struct {
	Run       RunRow            // model run master row: run_lst
	Txt       []RunTxtRow       // run text rows: run_txt
	Opts      map[string]string // options used to run the model: run_option
	Param     []runParam        // run parameters: parameter_hid, sub-value count, run_parameter_txt table rows
	Table     []runTable        // run tables: table_hid fom run_table rows
	EntityGen []EntityGenMeta   // run entity generation: entity_gen, entity_gen_attr table rows
	RunEntity []RunEntityRow    // run microdata entities: run_entity table rows
	Progress  []RunProgress     // run progress by sub-values: run_progress table rows
}

// RunPub is "public" model run metadata for json import-export
type RunPub struct {
	ModelName           string            // model name for that run
	ModelDigest         string            // model digest for that run
	ModelVersion        string            // model_ver     VARCHAR(32)  NOT NULL
	ModelCreateDateTime string            // create_dt     VARCHAR(32)  NOT NULL
	Name                string            // run_name      VARCHAR(255) NOT NULL
	SubCount            int               // sub_count     INT          NOT NULL, -- subvalue count
	SubStarted          int               // sub_started   INT          NOT NULL, -- number of subvalues started
	SubCompleted        int               // sub_completed INT          NOT NULL, -- number of subvalues completed
	CreateDateTime      string            // create_dt     VARCHAR(32)  NOT NULL, -- start date-time
	Status              string            // status        VARCHAR(1)   NOT NULL, -- run status: i=init p=progress s=success x=exit e=error(failed)
	UpdateDateTime      string            // update_dt     VARCHAR(32)  NOT NULL, -- last update date-time
	RunId               int               // used only as output value, input value ignored
	RunDigest           string            // run_digest    VARCHAR(32)  NULL,     -- digest of the run metadata: model digest, run name, sub count, created date-time, run stamp
	ValueDigest         string            // value_digest  VARCHAR(32),           -- if not NULL then digest of the run values: all parameters and output tables
	RunStamp            string            // run_stamp     VARCHAR(32)  NOT NULL, -- process run stamp, by default is log time stamp
	Txt                 []DescrNote       // run text: description and notes by language
	Opts                map[string]string // options used to run the model: run_option
	Param               []ParamRunSetPub  // run parameters: name, sub-value count and value notes by language
	Table               []TableRunPub     // run tables: name for tables included in run_table
	Entity              []EntityRunPub    // run entities: entity generation and attributes
	Progress            []RunProgress     // run progress by sub-values: run_progress table rows
}

// ParamRunSetTxtPub is "public" run or workset parameter metadata for json import-export: name, description and notes
type ParamRunSetTxtPub struct {
	Name string     // parameter name
	Txt  []LangNote // parameter value notes by language
}

// ParamRunSetPub is "public" run or workset parameter value metadata for json import-export
type ParamRunSetPub struct {
	ParamRunSetTxtPub
	SubCount     int    // number of parameter sub-values
	DefaultSubId int    // default sub-value id for that parameter workset
	ValueDigest  string // value digest, not empty only as result of select from run_parameter; input from "public" value digest is ignored
}

// ParamValuePub is "public" run or workset parameter metadata and values for json import-export.
type ParamValuePub struct {
	ParamRunSetPub                 // parameter metadata
	Kind           string          // where to get parameter from: "value", "run", "set" or empty "" which is "value" by default
	From           string          // run digest (or name or stamp) or workset name to copy parameter from
	Value          []CellCodeParam // parameter value(s)
}

// TableRunPub is "public" metadata for output tables included in model run results
type TableRunPub struct {
	Name        string // parameter name
	ValueDigest string // value digest, not empty only as result of select from table_parameter; input from "public" value digest is ignored
}

// EntityRunPub is "public" metadata of entity run generation and attributes for json import-export
type EntityRunPub struct {
	Name        string   // entity name
	GenDigest   string   // digest of entity generation, not empty only as result of select from run_entity; input from "public" digest is ignored
	ValueDigest string   // value digest, not empty only as result of select from run_entity; input from "public" value digest is ignored
	RowCount    int      // if not zero then entity microdata row count
	Attr        []string // names of entity generation attributes
}

// RunRow is model run row: run_lst table row.
//
// Run status: i=init p=progress s=success x=exit e=error(failed).
// Run id must be different from working set id (use id_lst to get it)
type RunRow struct {
	RunId          int    // run_id        INT          NOT NULL, -- unique run id
	ModelId        int    // model_id      INT          NOT NULL
	Name           string // run_name      VARCHAR(255) NOT NULL, -- model run name
	SubCount       int    // sub_count     INT          NOT NULL, -- subvalue count
	SubStarted     int    // sub_started   INT          NOT NULL, -- number of subvalues started
	SubCompleted   int    // sub_completed INT          NOT NULL, -- number of subvalues completed
	CreateDateTime string // create_dt     VARCHAR(32)  NOT NULL, -- start date-time
	Status         string // status        VARCHAR(1)   NOT NULL, -- run status: i=init p=progress s=success x=exit e=error(failed)
	UpdateDateTime string // update_dt     VARCHAR(32)  NOT NULL, -- last update date-time
	RunDigest      string // run_digest    VARCHAR(32)  NULL,     -- digest of the run metadata: model digest, run name, sub count, created date-time, run stamp
	ValueDigest    string // value_digest  VARCHAR(32),           -- if not NULL then digest of the run values: all parameters and output tables
	RunStamp       string // run_stamp     VARCHAR(32)  NOT NULL, -- process run stamp, by default is log time stamp
}

// RunTxtRow is db row of run_txt
type RunTxtRow struct {
	RunId    int    // run_id    INT          NOT NULL
	LangCode string // lang_code VARCHAR(32)  NOT NULL
	Descr    string // descr     VARCHAR(255) NOT NULL
	Note     string // note      VARCHAR(32000)
}

// runParam is a holder for run parameter Hid, subvalue count and run_parameter_txt rows
type runParam struct {
	ParamHid    int              // parameter_hid INT NOT NULL
	SubCount    int              // number of parameter sub-values
	ValueDigest string           // value_digest  VARCHAR(32), -- if not NULL then digest of parameter value for the run
	Txt         []RunParamTxtRow // run_parameter_txt table rows
}

// RunParamTxtRow is db row of run_parameter_txt
type RunParamTxtRow struct {
	RunId    int    // run_id        INT         NOT NULL
	ParamHid int    // parameter_hid INT         NOT NULL
	LangCode string // lang_code     VARCHAR(32) NOT NULL
	Note     string // note          VARCHAR(32000)
}

// runTable is a holder for run table Hid where row exist in run_table
type runTable struct {
	TableHid    int    // table_hid INT NOT NULL
	ValueDigest string // value_digest  VARCHAR(32), -- if not NULL then digest of table value for the run
}

// RunProgress is a "public" sub-value run_progress db row
type RunProgress struct {
	SubId          int     // sub_id         INT         NOT NULL, -- sub-value id (zero based index)
	CreateDateTime string  // create_dt      VARCHAR(32) NOT NULL, -- start date-time
	Status         string  // status         VARCHAR(1)  NOT NULL, -- run status: i=init p=progress s=success x=exit e=error(failed)
	UpdateDateTime string  // update_dt      VARCHAR(32) NOT NULL, -- last update date-time
	Count          int     // progress_count INT         NOT NULL, -- progress count: percent completed
	Value          float64 // progress_value FLOAT       NOT NULL, -- progress value: number of cases (case based) or time (time based)
}

// runProgressRow is db row of run_progress
type runProgressRow struct {
	RunId    int         // run_id         INT         NOT NULL
	Progress RunProgress // sub-value run progress
}

// EntityGenMeta is a holder of entity generation db rows from entity_gen, model_entity_dic, entity_gen_attr and run_entity tables
// Entity generation is a model entity with set of attributes included in particular model run(s).
// Model run typically include less attributes in microdata output than entity has in model metadata.
type EntityGenMeta struct {
	entityGenRow                    // entity generation: entity_gen join to model_entity_dic table
	GenAttr      []entityGenAttrRow // entity generation attributes: entity_gen_attr join to entity_gen
}

// entityGen is db row of entity_gen join to model_entity_dic table where row exist in run_entity.
// Entity generation is a model entity with set of attributes included in particular model run(s).
// Model run typically include less attributes in microdata output than entity has in model metadata.
type entityGenRow struct {
	GenHid        int    // entity_gen_hid  INT         NOT NULL, -- entity generation unique id
	ModelId       int    // model_id        INT         NOT NULL
	EntityId      int    // model_entity_id INT         NOT NULL
	EntityHid     int    // entity_hid      INT         NOT NULL, -- unique entity id
	DbEntityTable string // db_entity_table VARCHAR(64) NOT NULL, -- db table name: Person_g87abcdef
	GenDigest     string // gen_digest      VARCHAR(32) NOT NULL, -- digest of entity generation
}

// entityGenAttrRow is db row of entity_gen_attr join to entity_gen table where row exist in run_entity
type entityGenAttrRow struct {
	GenHid int // entity_gen_hid  INT NOT NULL, -- entity generation unique id
	AttrId int // attr_id         INT NOT NULL
}

// RunEntityRow is db row of run_entity join to entity_gen table
type RunEntityRow struct {
	RunId       int    // run_id         INT         NOT NULL
	GenHid      int    // entity_gen_hid INT NOT NULL
	RowCount    int    // row_count    INT NOT NULL, -- if not zero then entity microdata row count
	ValueDigest string // value_digest VARCHAR(32),  -- if not NULL then digest of table value for the run
}

// WorksetMeta is a model workset metadata: name, parameters, decription, notes.
//
// Workset (working set of model input parameters):
// it can be a full set, which include all model parameters
// or subset and include only some parameters.
//
// Each model must have "default" workset.
// Default workset must include ALL model parameters (it is a full set).
// Default workset is a first workset of the model: set_id = min(set_id).
// If workset is a subset (does not include all model parameters)
// then it can be based on model run results, specified by run_id (not NULL).
//
// Workset can be editable or read-only.
// If workset is editable then you can modify input parameters or workset description, notes, etc.
// If workset is read-only then you can run the model using that workset as input.
//
// Important: working set_id must be different from run_id (use id_lst to get it)
// Important: always update parameter values inside of transaction scope
// Important: before parameter update do is_readonly = is_readonly + 1 to "lock" workset
//
// WorksetMeta is workset metadata db rows: workset_lst, workset_txt, workset_parameter, workset_parameter_txt
type WorksetMeta struct {
	Set   WorksetRow      // workset master row: workset_lst
	Txt   []WorksetTxtRow // workset text rows: workset_txt
	Param []worksetParam  // workset parameter: parameter_hid, sub-value count and workset_parameter_txt rows
}

// WorksetHdrPub is "public" workset metadata for json import-export
type WorksetHdrPub struct {
	ModelName           string      // model name for that workset
	ModelDigest         string      // model digest for that workset
	ModelVersion        string      // model_ver     VARCHAR(32)  NOT NULL
	ModelCreateDateTime string      // create_dt     VARCHAR(32)  NOT NULL
	Name                string      // workset name: set_name VARCHAR(255) NOT NULL
	BaseRunDigest       string      // if not empty then digest of the base run
	IsReadonly          bool        // readonly flag
	UpdateDateTime      string      // last update date-time
	IsCleanBaseRun      bool        // if true then update set base run digest to NULL
	Txt                 []DescrNote // workset text: description and notes by language
}

// WorksetPub is "public" workset metadata and parameter metadata for json import-export
type WorksetPub struct {
	WorksetHdrPub
	Param []ParamRunSetPub // workset parameters: name and text (value notes by language)
}

// WorksetCreatePub is "public" workset metadata and parameters list for json import-export.
// Each parameter must have metadata: name, subvalues count, optinal notes and value(s).
// Values can be either literal cell values or copy direction, for example run digest to copy from.
type WorksetCreatePub struct {
	WorksetHdrPub
	Param []ParamValuePub // workset parameters: name, text (value notes by language) and value
}

// WorksetRow is workset_lst table row.
type WorksetRow struct {
	SetId          int    // unique working set id
	BaseRunId      int    // if not NULL and positive then base run id (source run id)
	ModelId        int    // model_id     INT          NOT NULL
	Name           string // set_name     VARCHAR(255) NOT NULL
	IsReadonly     bool   // is_readonly  SMALLINT     NOT NULL
	UpdateDateTime string // update_dt    VARCHAR(32)  NOT NULL, -- last update date-time
	isNullBaseRun  bool   // if true then update set base run digest to NULL
}

// WorksetParam is a holder for workset parameter Hid, sub-value count and workset_parameter_txt rows
type worksetParam struct {
	ParamHid     int                  // parameter_hid INT NOT NULL
	SubCount     int                  // number of parameter sub-values
	DefaultSubId int                  // default sub-value id for that parameter workset
	Txt          []WorksetParamTxtRow // workset_parameter_txt table rows
}

// WorksetTxtRow is db row of workset_txt
type WorksetTxtRow struct {
	SetId    int    // set_id    INT          NOT NULL
	LangCode string // lang_code VARCHAR(32)  NOT NULL
	Descr    string // descr     VARCHAR(255) NOT NULL
	Note     string // note      VARCHAR(32000)
}

// WorksetParamTxtRow is workset_parameter_txt table row.
type WorksetParamTxtRow struct {
	SetId    int    // set_id        INT NOT NULL
	ParamHid int    // parameter_hid INT NOT NULL
	LangCode string // lang_code VARCHAR(32)  NOT NULL
	Note     string // note          VARCHAR(32000), -- parameter value note
}

// TaskMeta is metadata for modeling task: name, status, description, notes, task run history.
//
// Modeling task is a named set of input model inputs (of workset ids) to run the model.
// Typical use case: create multiple input sets by varying some model parameters,
// combine it under named "task" and run the model with that task name.
// As result multiple model "runs" created ("run" is input and output data of model run).
// Such run of model called "task run" and allow to study dependencies between model input and output.
//
// Task can be edited by user: new input workset ids added or some workset id(s) excluded.
// As result current task body (workset ids of the task) may be different
// from older version of it: task_set set_id's  may not be same as task_run_set set_id's.
// TaskRun and TaskRunSet is a history and result of that task run,
// but there is no guarantee of any workset in task history still exist
// or contain same input parameter values as it was at the time of task run.
// To find actual input for any particular model run and/or task run we must use run_id.
type TaskMeta struct {
	TaskDef               // task definition: metadata and input worksets
	TaskRun []taskRunItem // task run history: task_run_lst and task_run_set rows
}

// TaskDef is modeling task definition: metadata and input worksets
type TaskDef struct {
	Task TaskRow      // modeling task row: task_lst
	Txt  []TaskTxtRow // task text rows: task_txt
	Set  []int        // task body (current list of workset id's): task_set
}

// taskRunItem is master task_run_lst row (task run status, date-time...)
// and details as list of (run id, ste id) pairs
type taskRunItem struct {
	TaskRunRow                 // task run history row: task_run_lst
	TaskRunSet []TaskRunSetRow // task run history body: task_run_set
}

// TaskRow is db row of task_lst.
type TaskRow struct {
	TaskId  int    // task_id      INT          NOT NULL, -- unique task id
	ModelId int    // model_id     INT          NOT NULL
	Name    string // task_name    VARCHAR(255) NOT NULL
}

// TaskTxtRow is db row of task_txt
type TaskTxtRow struct {
	TaskId   int    // task_id  INT           NOT NULL
	LangCode string // lang_code VARCHAR(32)  NOT NULL
	Descr    string // descr     VARCHAR(255) NOT NULL
	Note     string // note      VARCHAR(32000)
}

// TaskRunRow is db row of task_run_lst.
// This table contains task run history and status.
//
// Task status: i=init p=progress w=wait s=success x=exit e=error(failed)
//   if task status = w (wait) then
//      model wait and NOT completed until other process set status to one of finals: s,x,e
//      model check if any new sets inserted into task_set and run it as they arrive
type TaskRunRow struct {
	TaskRunId      int    // task_run_id INT          NOT NULL, -- unique task run id
	TaskId         int    // task_id     INT          NOT NULL
	Name           string // run_name    VARCHAR(255) NOT NULL, -- task run name
	SubCount       int    // sub_count   INT          NOT NULL, -- subvalue count of task run
	CreateDateTime string // create_dt   VARCHAR(32)  NOT NULL, -- start date-time
	Status         string // status      VARCHAR(1)   NOT NULL, -- task status: i=init p=progress w=wait s=success x=exit e=error(failed)
	UpdateDateTime string // update_dt   VARCHAR(32)  NOT NULL, -- last update date-time
	RunStamp       string // run_stamp   VARCHAR(32)  NOT NULL, -- process run stamp, by default is log time stamp
}

// TaskRunSetRow is db row of task_run_set.
// This table contains task run input (working set id) and output (model run id)
type TaskRunSetRow struct {
	TaskRunId int // task_run_id INT NOT NULL
	RunId     int // run_id      INT NOT NULL, -- if > 0 then result run id
	SetId     int // set_id      INT NOT NULL, -- if > 0 then input working set id
	TaskId    int // task_id     INT NOT NULL
}

// TaskPub is "public" modeling task metadata, task input worksets and task run history for json import-export
type TaskPub struct {
	TaskDefPub              // task definition: metadata and input worksets
	TaskRun    []taskRunPub // task run history: task_run_lst
}

// TaskDefPub is "public" modeling task metadata and task input worksets for json import-export
type TaskDefPub struct {
	ModelName           string      // model name for that task list
	ModelDigest         string      // model digest for that task list
	ModelVersion        string      // model_ver     VARCHAR(32)  NOT NULL
	ModelCreateDateTime string      // create_dt     VARCHAR(32)  NOT NULL
	Name                string      // task_name    VARCHAR(255) NOT NULL
	Txt                 []DescrNote // task text: description and notes by language
	Set                 []string    // task body: list of workset names
}

// taskRunPub is "public" metadata of task run history.
type taskRunPub struct {
	Name           string          // run_name   VARCHAR(255) NOT NULL, -- task run name
	SubCount       int             // sub_count  INT          NOT NULL, -- subvalue count of task run
	CreateDateTime string          // create_dt  VARCHAR(32)  NOT NULL, -- start date-time
	Status         string          // status     VARCHAR(1)   NOT NULL, -- task status: i=init p=progress w=wait s=success x=exit e=error(failed)
	UpdateDateTime string          // update_dt  VARCHAR(32)  NOT NULL, -- last update date-time
	RunStamp       string          // run_stamp  VARCHAR(32)  NOT NULL, -- process run stamp, by default is log time stamp
	TaskRunSet     []taskRunSetPub // task run history body: run and set pairs
}

// taskRunSetPub is "public" metadata of task run history body: run and set pairs.
// To find workset name is used, it is unique by model.
// To find model run use run digest.
type taskRunSetPub struct {
	Run struct { // "public" link to model run
		Name           string // run_name      VARCHAR(255) NOT NULL
		SubCompleted   int    // sub_completed INT          NOT NULL, -- number of subvalues completed
		CreateDateTime string // create_dt     VARCHAR(32)  NOT NULL, -- start date-time
		Status         string // status        VARCHAR(1)   NOT NULL, -- run status: i=init p=progress s=success x=exit e=error(failed)
		RunDigest      string // run_digest    VARCHAR(32)  NULL,     -- digest of the run metadata: model digest, run name, sub count, created date-time, run stamp
		ValueDigest    string // value_digest  VARCHAR(32),           -- if not NULL then digest of the run values: all parameters and output tables
		RunStamp       string // run_stamp     VARCHAR(32)  NOT NULL, -- process run stamp, by default is log time stamp
	}
	SetName string // name of input workset which used for that model run
}

// TaskRunSetTxt is additional task text: description and notes by language for all input worksets and model runs of the task.
// Run identified by run digest or, if digest is null then by run name
type TaskRunSetTxt struct {
	SetTxt map[string][]DescrNote // map workset name to description and notes by language
	RunTxt map[string][]DescrNote // map run digest-or-name to description and notes by language
}
