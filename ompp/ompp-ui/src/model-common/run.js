// db structures common functions: model run and run list

import * as Hlpr from './helper'

// run count: number of run text entries in the run text list
export const runTextCount = (rtl) => {
  return isRunTextList(rtl) ? rtl.length : 0
}

// return true if each list element isRunText()
export const isRunTextList = (rtl) => {
  if (!rtl) return false
  if (!Array.isArray(rtl)) return false
  if (rtl.length > 0 && !isRunText(rtl[0])) return false
  return true
}

// if this is run text
export const isRunText = (rt) => {
  if (!rt) return false
  if (!rt.hasOwnProperty('ModelName') || !rt.hasOwnProperty('ModelDigest')) return false
  if (!rt.hasOwnProperty('Name') || !rt.hasOwnProperty('SubCount')) return false
  if (!rt.hasOwnProperty('CreateDateTime') || !rt.hasOwnProperty('Status') || !rt.hasOwnProperty('UpdateDateTime')) return false
  if (!rt.hasOwnProperty('RunId') || !rt.hasOwnProperty('RunDigest') || !rt.hasOwnProperty('ValueDigest') || !rt.hasOwnProperty('RunStamp')) return false
  if (!Array.isArray(rt.Param) || !Array.isArray(rt.Table) || !Array.isArray(rt.Entity) || !Array.isArray(rt.Txt) || !rt.hasOwnProperty('Opts')) return false
  return true
}

// if this is not empty run text: model name, model digest, run name, run digest, run stamp, status, sub-count, create date-time
export const isNotEmptyRunText = (rt) => {
  if (!isRunText(rt)) return false
  return (rt.ModelName || '') !== '' && (rt.ModelDigest || '') !== '' &&
    (rt.Name || '') !== '' && (rt.RunDigest || '') !== '' && (rt.RunStamp || '') !== '' &&
    (rt.Status || '') !== '' && (rt.SubCount || 0) !== 0 && (rt.CreateDateTime || '') !== ''
}

// return empty run text
export const emptyRunText = () => {
  return {
    ModelName: '',
    ModelDigest: '',
    Name: '',
    SubCount: 0,
    CreateDateTime: '',
    Status: '',
    UpdateDateTime: '',
    RunId: 0,
    RunDigest: '',
    ValueDigest: '',
    RunStamp: '',
    Opts: {},
    Param: [],
    Table: [],
    Entity: [],
    Txt: []
  }
}

// return true if this is not the same run text or it is updated: different run status or update time.
export const isRunTextUpdated = (rt, rtOther) => {
  if (!isNotEmptyRunText(rt) || !isNotEmptyRunText(rtOther)) return false

  return rt.ModelDigest !== rtOther.ModelDigest || rt.RunDigest !== rtOther.RunDigest ||
    rt.Status !== rtOther.Status || rt.UpdateDateTime !== rtOther.UpdateDateTime
}

// find last model run: last successful or last completed or last in the model run list
export const lastRunDigest = (rtl) => {
  if (!rtl) return ''
  if (!Array.isArray(rtl)) return ''

  let lastDg = ''
  let lastCompleted = ''
  for (let k = 0; k < rtl.length; k++) {
    if (isRunSuccess(rtl[k])) return rtl[k].RunDigest
    if (!lastCompleted && isRunCompleted(rtl[k])) lastCompleted = rtl[k].RunDigest
    if (!lastDg) lastDg = rtl[k].RunDigest
  }
  return lastCompleted || lastDg
}

// return true if table included in run text: find table name in run tables array
export const isRunTextHasTable = (rt, name) => {
  if (!name || !isNotEmptyRunText(rt)) return false

  for (let k = 0; k < rt.Table.length; k++) {
    if ((rt.Table[k]?.Name || '') === name) return true
  }
  return false
}

// return empty run table
export const emptyRunTable = () => {
  return {
    Name: '',
    ValueDigest: ''
  }
}

// if this is not empty run table: table Name is not empty and ValueDigest is a string
export const isNotEmptyRunTable = (rtbl) => {
  return rtbl &&
    rtbl?.Name && typeof rtbl.Name === typeof 'string' && rtbl.Name &&
    rtbl.hasOwnProperty('ValueDigest') && typeof rtbl.ValueDigest === typeof 'string'
}

// find run Table by table name in Table[] array of run text
// return empty value if not found
export const runTableByName = (rt, name) => {
  if (!name || !isNotEmptyRunText(rt)) return emptyRunTable()

  for (let k = 0; k < rt.Table.length; k++) {
    if ((rt.Table[k]?.Name || '') === name) return rt.Table[k]
  }
  return emptyRunTable() // not found
}

// compare runs and return: names of different parameters, different tables, suppressed tables and name of the first run
export const runCompare = (runCurrent, compareDigestLst, tableCount, runTextLst) => {
  if (!isNotEmptyRunText(runCurrent) || !Hlpr.isLength(compareDigestLst) || !tableCount || !Hlpr.isLength(runTextLst) || !isRunTextList(runTextLst)) {
    return { firstRunName: '', paramDiff: [], tableDiff: [], tableSupp: [], entityDiff: [], entityMiss: [] }
  }

  const pn = []
  let tn = []
  const sn = []
  let en = []
  const ems = []

  let firstName = ''
  const isAllTbl = runCurrent.Table.length === tableCount

  for (const dg of compareDigestLst) {
    const rt = runTextLst.find(r => r.RunDigest === dg)
    if (!rt) continue

    // compare parameters by value digest: both Param[] arrays must contain all parameters
    for (let k = 0; k < runCurrent.Param.length; k++) {
      if (runCurrent.Param[k].ValueDigest !== rt.Param[k].ValueDigest) {
        if (!pn.includes(runCurrent.Param[k].Name)) pn.push(runCurrent.Param[k].Name)
      }
    }

    // compare output tables by value digest:
    // Table[] arrays can contain diffrent tables or both can have full table list
    if (isAllTbl && rt.Table.length === tableCount) { // both runs contain all output tables
      for (let k = 0; k < runCurrent.Table.length; k++) {
        if (runCurrent.Table[k].ValueDigest !== rt.Table[k].ValueDigest) {
          if (!tn.includes(runCurrent.Table[k].Name)) tn.push(runCurrent.Table[k].Name)
        }
      }
    } else {
      // at least one of the Table[] arrays does not contain all model tables
      for (let k = 0; k < runCurrent.Table.length; k++) {
        const j = rt.Table.findIndex((t) => { return t.Name === runCurrent.Table[k].Name })
        if (j >= 0) {
          if (rt.Table[j].ValueDigest !== runCurrent.Table[k].ValueDigest) {
            if (!tn.includes(runCurrent.Table[k].Name)) tn.push(runCurrent.Table[k].Name)
          }
        } else {
          if (!sn.includes(runCurrent.Table[k].Name)) sn.push(runCurrent.Table[k].Name)
        }
      }
    }

    // compare microdata run entity by value digest
    for (let k = 0; k < runCurrent.Entity.length; k++) {
      const j = rt.Entity.findIndex((e) => { return e.Name === runCurrent.Entity[k].Name })
      if (j >= 0) {
        if (rt.Entity[j].ValueDigest !== runCurrent.Entity[k].ValueDigest) {
          if (!en.includes(runCurrent.Entity[k].Name)) en.push(runCurrent.Entity[k].Name)
        }
      } else {
        if (!ems.includes(runCurrent.Entity[k].Name)) ems.push(runCurrent.Entity[k].Name)
      }
    }

    if (!firstName) firstName = rt.Name // first run name to compare
  }

  tn = tn.filter(name => !sn.includes(name)) // remove missing tables name from the list of different tables
  en = en.filter(name => !ems.includes(name)) // remove missing entities name from the list of different microdata entities

  return {
    firstRunName: firstName,
    paramDiff: pn,
    tableDiff: tn,
    tableSupp: sn,
    entityDiff: en,
    entityMiss: ems
  }
}

/* eslint-disable no-multi-spaces */
export const RUN_SUCCESS = 's'      // run completed successfuly
export const RUN_IN_PROGRESS = 'p'  // run in progress
export const RUN_INITIAL = 'i'      // run not started yet
export const RUN_WAIT = 'w'         // task run paused
export const RUN_FAILED = 'e'       // run falied (comleted with error)
export const RUN_EXIT = 'x'         // run exit and not completed
export const RUN_DELETE = 'd'       // run delete in progress
/* eslint-enable no-multi-spaces */

// return true if run completed successfuly
export const isRunSuccess = (rt) => {
  return isRunText(rt) && rt.Status === RUN_SUCCESS
}

// return true if run in progress
export const isRunInProgress = (rt) => {
  return isRunText(rt) && (rt.Status === RUN_IN_PROGRESS || rt.Status === RUN_INITIAL)
}

// return true if run completed, status is one of: s=success, x=exit, e=error
export const isRunCompleted = (rt) => {
  return isRunText(rt) && isRunCompletedStatus(rt.Status)
}

// return true if run completed, status is one of: s=success, x=exit, e=error
export const isRunCompletedStatus = (status) => {
  return status === RUN_SUCCESS || status === RUN_EXIT || status === RUN_FAILED
}

// return true if run delete in progress: status is: d=delete or name starts with 'deleted:'
export const isRunDeletedStatus = (status, name) => {
  return status === RUN_DELETE || (typeof name === typeof 'string' && name.startsWith('deleted:'))
}

// return run status description, e.g: i=init p=progress s=success x=exit e=error(failed)
export const statusText = (status) => {
  switch (status || '') {
    case RUN_SUCCESS: return 'success'
    case RUN_IN_PROGRESS: return 'in progress'
    case RUN_INITIAL: return 'initial'
    case RUN_WAIT: return 'waiting'
    case RUN_FAILED: return 'failed'
    case RUN_EXIT: return 'exit (not completed)'
    case RUN_DELETE: return 'deleted'
    case 'success': return 'success'
    case 'in progress': return 'in progress'
    case 'progress': return 'in progress'
    case 'init': return 'initial'
    case 'initial': return 'initial'
    case 'wait': return 'waiting'
    case 'waiting': return 'waiting'
    case 'failed': return 'failed'
    case 'error': return 'failed'
    case 'kill': return 'canceled'
    case 'exit': return 'exit (not completed)'
    case 'exit (not completed)': return 'exit (not completed)'
    case 'deleted': return 'deleted'
  }
  return 'unknown'
}

// RunState: model run state from oms run catalog
//
// if this is model run state
export const isRunState = (rst) => {
  if (!rst) return false
  return rst.hasOwnProperty('ModelName') && rst.hasOwnProperty('ModelDigest') &&
  rst.hasOwnProperty('RunStamp') && rst.hasOwnProperty('SubmitStamp') && rst.hasOwnProperty('IsFinal') &&
    rst.hasOwnProperty('RunName') && rst.hasOwnProperty('UpdateDateTime') && rst.hasOwnProperty('TaskRunName') &&
    rst.hasOwnProperty('IsLog') && rst.hasOwnProperty('LogFileName')
}

// if this is not empty model run state
export const isNotEmptyRunState = (rst) => {
  if (!isRunState(rst)) return false
  return (rst.ModelDigest || '') !== '' && (rst.UpdateDateTime || '') !== '' && ((rst.RunStamp || '') !== '' || (rst.SubmitStamp || '') !== '')
}

// return model run state
export const emptyRunState = () => {
  return {
    ModelName: '',
    ModelDigest: '',
    RunStamp: '',
    SubmitStamp: '',
    IsFinal: false,
    UpdateDateTime: '',
    RunName: '',
    TaskRunName: '',
    IsLog: false,
    LogFileName: ''
  }
}

// RunStateLogPage: RunState and log file page
//
// if this is RunStateLogPage: model run state and run log page
export const isRunStateLog = (rlp) => {
  if (!rlp) return false
  if (!isRunState(rlp)) return false
  return rlp.hasOwnProperty('Offset') && rlp.hasOwnProperty('Size') &&
    rlp.hasOwnProperty('TotalSize') && Array.isArray(rlp.Lines)
}

// if this is not empty RunStateLogPage: model run state and run log page
export const isNotEmptyRunStateLog = (rlp) => {
  return isNotEmptyRunState(rlp)
}

// return empty RunStateLogPage: model run state and run log page
export const emptyRunStateLog = () => {
  return {
    ModelName: '',
    ModelDigest: '',
    RunStamp: '',
    SubmitStamp: '',
    IsFinal: false,
    UpdateDateTime: '',
    RunName: '',
    TaskRunName: '',
    IsLog: false,
    LogFileName: '',
    Offset: 0,
    Size: 0,
    TotalSize: 0,
    Lines: []
  }
}

// return RunState part of RunStateLogPage
export const toRunStateFromLog = (rlp) => {
  if (!rlp) return emptyRunState()
  if (!isRunState(rlp)) return emptyRunState()
  return {
    ModelName: rlp.ModelName,
    ModelDigest: rlp.ModelDigest,
    RunStamp: rlp.RunStamp || '',
    SubmitStamp: rlp.SubmitStamp || '',
    IsFinal: !!rlp.IsFinal,
    RunName: rlp.RunName || '',
    TaskRunName: rlp.TaskRunName || '',
    UpdateDateTime: rlp.UpdateDateTime || '',
    IsLog: !!rlp.IsLog,
    LogFileName: rlp.LogFileName || ''
  }
}

// Run progerss: RunPub with run_lst row array of run_progress for each sub-value
//
// return true if each list element isRunStatusProgress()
export const isRunStatusProgressList = (rpl) => {
  if (!rpl) return false
  if (!Array.isArray(rpl)) return false
  if (rpl.length > 0 && !isRunStatusProgress(rpl[0])) return false
  return true
}

// return empty run status and progress
export const emptyRunStatusProgress = () => {
  return {
    ModelName: '',
    ModelDigest: '',
    Name: '',
    RunDigest: '',
    ValueDigest: '',
    RunStamp: '',
    SubCount: 0,
    SubCompleted: 0,
    Status: '',
    CreateDateTime: '',
    UpdateDateTime: '',
    Progress: []
  }
}

// if this is run status and progress
export const isRunStatusProgress = (rp) => {
  if (!rp) return false
  if (!rp.hasOwnProperty('ModelName') || !rp.hasOwnProperty('ModelDigest')) return false
  if (!rp.hasOwnProperty('Name') || !rp.hasOwnProperty('RunDigest') || !rp.hasOwnProperty('ValueDigest') || !rp.hasOwnProperty('RunStamp')) return false
  if (!rp.hasOwnProperty('SubCount') || !rp.hasOwnProperty('SubCompleted')) return false
  if (!rp.hasOwnProperty('Status') || !rp.hasOwnProperty('CreateDateTime') || !rp.hasOwnProperty('UpdateDateTime')) return false
  if (!Array.isArray(rp.Progress)) return false
  return true
}

// if this is not empty run progress: model name, model digest, run name, run stamp, sub-count, status, create date-time
export const isNotEmptyRunStatusProgress = (rp) => {
  if (!isRunStatusProgress(rp)) return false
  return (rp.ModelName || '') !== '' && (rp.ModelDigest || '') !== '' &&
    (rp.Name || '') !== '' && (rp.RunDigest || '') !== '' && (rp.RunStamp || '') !== '' &&
    (rp.SubCount || 0) !== 0 && (rp.Status || '') !== '' && (rp.CreateDateTime || '') !== ''
}

// return empty run progress item
export const emptyRunProgress = () => {
  return {
    SubId: 0,
    Status: '',
    CreateDateTime: '',
    UpdateDateTime: '',
    Count: 0,
    Value: 0
  }
}

// if this is run progress item
export const isRunProgress = (rpi) => {
  if (!rpi) return false
  if (!rpi.hasOwnProperty('SubId') || !rpi.hasOwnProperty('Status')) return false
  if (!rpi.hasOwnProperty('CreateDateTime') || !rpi.hasOwnProperty('UpdateDateTime')) return false
  if (!rpi.hasOwnProperty('Count') || !rpi.hasOwnProperty('Value')) return false
  return true
}

// if this is not empty run progress item: status, create date-time, update date-time
export const isNotEmptyRunProgress = (rpi) => {
  if (!isRunProgress(rpi)) return false
  return (rpi.Status || '') !== '' && (rpi.CreateDateTime || '') !== '' && (rpi.UpdateDateTime || '') !== ''
}

// job control item
// for queue jobs RunStatus, LogFileName and log Lines are always empty
/*
{
  SubmitStamp: "2022_07_04_20_08_21_158",
  JobStatus: "",
  ModelName: "RiskPaths",
  ModelDigest: "d90e1e9a49a06d972ecf1d50e684c62b",
  RunStamp: "2022_07_04_20_08_21_159",
  Opts: {},
  Threads: 1,
  IsMpi: false,
  Mpi: {
    Np: 0,
    IsNotOnRoot: false,
    IsNotByJob: false
  },
  Template: "",
  Res: {
    Cpu: 4,
    Mem: 4,
    ProcessCount: 2,
    ThreadCount: 2,
    ProcessMemMb: 32,
    ThreadMemMb: 512
  },
  Tables: [],
  Microdata: {
    IsToDb:     true,
    IsInternal: false,
    Entity: [
      {
        Name: "Person",
        Attr: ["Age", "Income"]
      }
    ]
  },
  RunNotes: [],
  LogFileName: "RiskPaths.2022_07_05_19_55_38_627.console.log",
  RunStatus: [],
  Lines: []
}
*/

// return empty run job control
export const emptyRunJob = (stamp) => {
  return {
    SubmitStamp: (!!stamp && typeof stamp === typeof 'string' && stamp !== '') ? stamp : 'none',
    ModelName: '',
    ModelDigest: '',
    RunStamp: '',
    Opts: {},
    Threads: 1,
    IsMpi: false,
    Mpi: {
      Np: 0,
      IsNotOnRoot: false
    },
    Template: '',
    Tables: [],
    Microdata: {
      IsToDb: false,
      IsInternal: false,
      Entity: []
    },
    RunNotes: [],
    Res: {
      Cpu: 1,
      Mem: 0,
      ProcessCount: 1,
      ThreadCount: 1,
      ProcessMemMb: 0,
      ThreadMemMb: 0
    },
    IsOverLimit: false,
    QueuePos: 0,
    LogFileName: ''
  }
}

// return empty job control item
export const emptyJobItem = (stamp) => {
  return Object.assign(
    {},
    emptyRunJob(stamp),
    {
      JobStatus: '',
      RunStatus: [],
      Lines: []
    }
  )
}

// return true if this is run job control state
export const isRunJob = (jc) => {
  if (!jc) return false
  if (!jc.hasOwnProperty('SubmitStamp') || typeof jc.SubmitStamp !== typeof 'string') {
    return false
  }
  if (!isRunRequest(jc)) return false

  if (!jc.hasOwnProperty('Mpi')) return false
  if (!jc.Mpi.hasOwnProperty('IsNotByJob') || typeof jc.Mpi.IsNotByJob !== typeof true) return false
  if (!jc.hasOwnProperty('IsOverLimit') || typeof jc.IsOverLimit !== typeof true) return false

  if (!jc.hasOwnProperty('Res') ||
    !jc.Res.hasOwnProperty('Cpu') || typeof jc.Res.Cpu !== typeof 1 ||
    !jc.Res.hasOwnProperty('Mem') || typeof jc.Res.Mem !== typeof 1 ||
    !jc.Res.hasOwnProperty('ProcessCount') || typeof jc.Res.ProcessCount !== typeof 1 ||
    !jc.Res.hasOwnProperty('ThreadCount') || typeof jc.Res.ThreadCount !== typeof 1 ||
    !jc.Res.hasOwnProperty('ProcessMemMb') || typeof jc.Res.ProcessMemMb !== typeof 1 ||
    !jc.Res.hasOwnProperty('ThreadMemMb') || typeof jc.Res.ThreadMemMb !== typeof 1) {
    return false
  }
  if (!jc.hasOwnProperty('QueuePos') || typeof jc.QueuePos !== typeof 1) return false
  if (!jc.hasOwnProperty('LogFileName')) return false
  return true
}

// return true if this is not empty job control state
export const isNotEmptyRunJob = (jc) => {
  return isRunJob(jc) && typeof jc.ModelDigest === typeof 'string' && jc.ModelDigest !== ''
}

// return true if this is job control state
export const isJobItem = (jc) => {
  if (!jc) return false
  if (!isRunJob(jc)) return false
  if (!jc.hasOwnProperty('JobStatus') || !jc.hasOwnProperty('Lines')) return false
  return Array.isArray(jc.RunStatus) && Array.isArray(jc.Lines)
}

// return true if this is not empty job control state
export const isNotEmptyJobItem = (jc) => {
  return isJobItem(jc) && typeof jc.ModelDigest === typeof 'string' && jc.ModelDigest !== ''
}

// model run request, if request submitted from UI then RunStamp expected to be empty '' string
/*
{
  ModelName: "RiskPaths",
  ModelDigest: "d90e1e9a49a06d972ecf1d50e684c62b",
  RunStamp: "",
  Dir: "",
  Opts: {},
  Threads: 1,
  IsMpi: false,
  Mpi: {
    Np: 0,
    IsNotOnRoot: false,
    IsNotByJob:  false
  },
  Template: "",
  Tables: [],
  Microdata: {
    IsToDb:     true,
    IsInternal: false,
    Entity: [
      {
        Name: "Person",
        Attr: ["Age", "Income"]
      }
    ]
  },
  RunNotes: []
}
*/

// return empty run request
export const emptyRunRequest = () => {
  return {
    ModelName: '',
    ModelDigest: '',
    RunStamp: '',
    Opts: {},
    Threads: 1,
    IsMpi: false,
    Mpi: {
      Np: 0,
      IsNotOnRoot: false
    },
    Template: '',
    Tables: [],
    Microdata: {
      IsToDb: false,
      IsInternal: false,
      Entity: []
    },
    RunNotes: []
  }
}

// return empty microdata part of run request
export const emptyRunRequestMicrodata = () => {
  return {
    IsToDb: false,
    IsInternal: false,
    Entity: []
  }
}

// return true if this is Microdata part of run request
export const isRunRequestMicrodata = (rqMd) => {
  if (!rqMd) return false
  if (!rqMd.hasOwnProperty('IsToDb') || typeof rqMd.IsToDb !== typeof true) return false
  if (!rqMd.hasOwnProperty('IsInternal') || typeof rqMd.IsInternal !== typeof true) return false
  if (!rqMd.hasOwnProperty('Entity')) return false

  return Array.isArray(rqMd.Entity)
}

// return true if this is model run request
export const isRunRequest = (rReq) => {
  if (!rReq) return false
  if (!rReq.hasOwnProperty('ModelName') || !rReq.hasOwnProperty('ModelDigest') || !rReq.hasOwnProperty('RunStamp') ||
    !rReq.hasOwnProperty('Opts') || !rReq.hasOwnProperty('Template') || !rReq.hasOwnProperty('Mpi') ||
    !rReq.hasOwnProperty('RunNotes') || !rReq.hasOwnProperty('Tables')) {
    return false
  }
  if (!rReq.hasOwnProperty('Threads') || typeof rReq.Threads !== typeof 1) return false
  if (!rReq.hasOwnProperty('IsMpi') || typeof rReq.IsMpi !== typeof true) return false
  if (!rReq.Mpi.hasOwnProperty('Np') || typeof rReq.Mpi.Np !== typeof 1) return false
  if (!rReq.Mpi.hasOwnProperty('IsNotOnRoot') || typeof rReq.Mpi.IsNotOnRoot !== typeof true) return false

  if (!rReq.hasOwnProperty('Microdata') || !isRunRequestMicrodata(rReq.Microdata)) return false

  return Array.isArray(rReq.RunNotes) && Array.isArray(rReq.Tables)
}

// return true if this is not empty run request
export const isNotEmptyRunRequest = (rReq) => {
  return isRunRequest(rReq) && typeof rReq.ModelDigest === typeof 'string' && rReq.ModelDigest !== ''
}

// return true run request part of the job
export const runRequestFromJob = (jc) => {
  const rReq = emptyRunRequest()

  if (!isNotEmptyJobItem(jc)) return rReq // job item is empty: return empty run request

  rReq.ModelName = jc.ModelName || ''
  rReq.ModelDigest = jc.ModelDigest || ''
  rReq.RunStamp = jc.RunStamp || ''
  rReq.Opts = {}
  rReq.Threads = jc.Threads || 1
  rReq.IsMpi = jc.IsMpi
  rReq.Mpi.Np = jc.Mpi.Np
  rReq.Mpi.IsNotOnRoot = jc.Mpi.IsNotOnRoot
  rReq.Mpi.IsNotByJob = jc.Mpi.IsNotByJob
  rReq.Template = jc.Template || ''
  rReq.Tables = Array.from(jc.Tables)
  rReq.Microdata = (jc?.Microdata && isRunRequestMicrodata(jc.Microdata)) ? jc.Microdata : emptyRunRequestMicrodata()
  rReq.RunNotes = Array.from(jc.RunNotes)

  for (const key in jc.Opts) {
    if (!!key && (jc.Opts[key] || '') !== '') rReq.Opts[key] = jc.Opts[key]
  }

  return rReq
}

// return model run name or workset name from job run options: from Opts['OpenM.RunName']
export const getJobRunTitle = (jc) => {
  if (!jc) return ''
  if ((jc?.SubmitStamp || '') === '') return ''
  if (!jc.hasOwnProperty('Opts') || typeof jc.Opts !== 'object') return ''

  const runName = getRunOption(jc.Opts, 'OpenM.RunName')
  if (runName !== '') return runName

  return getRunOption(jc.Opts, 'OpenM.SetName')
}

// return model run option value by name or empty '' string if not found
export const getRunOption = (opts, name) => {
  if (!opts || typeof opts !== 'object') return ''

  const src = name.toLowerCase()

  for (const key in opts) {
    if (!!key && (opts[key] || '') !== '' && key.toLowerCase().endsWith(src)) return opts[key]
  }
  return '' // option not found
}

// return integer model run option value by name or default value if not found
export const getIntRunOption = (opts, name, defaultValue) => {
  if (!opts || typeof opts !== 'object') return defaultValue

  const src = name.toLowerCase()

  for (const key in opts) {
    if (!!key && (opts[key] || '') !== '' && key.toLowerCase().endsWith(src)) {
      const v = parseInt(opts[key], 10)
      return !isNaN(v) ? v : defaultValue
    }
  }
  return defaultValue // option not found
}

// return true if boolean model run option value is 'true', return false if not found
export const getBoolRunOption = (opts, name) => {
  if (!opts || typeof opts !== 'object') return false

  const src = name.toLowerCase()

  for (const key in opts) {
    if (!!key && (opts[key] || '') !== '' && key.toLowerCase().endsWith(src)) {
      return opts[key] === 'true'
    }
  }
  return false // option not found
}
