// UI session state
import { defineStore, acceptHMRUpdate } from 'pinia'

import * as Mdf from '../model-common'
import { useModelStore } from './model'

export const useUiStateStore = defineStore('ui-state', {
  state: () => ({
    uiLang: '',
    defaultTitle: 'OpenM++',
    runDigestSelected: '',
    worksetNameSelected: '',
    taskNameSelected: '',
    noAccDownload: true,
    noMicrodataDownload: true,
    idCsvDownload: false,
    treeLabelKind: '',
    uiTreeExpandedKeys: {},
    isSortModelTree: false,
    isDescModelTree: false,
    paramViews: {},
    tableViews: {},
    mdViews: {},
    modelView: {}
  }),

  getters: {
  },

  actions: {
    //
    // getters
    //

    // return selection part of model view by model digest
    modelViewSelected (modelDigest) {
      return (typeof modelDigest !== typeof 'string' || !this.modelView[modelDigest])
        ? Mdf.emptyModelView()
        : {
            runDigest: this.modelView[modelDigest]?.runDigest || '',
            digestCompareList: Array.isArray(this.modelView[modelDigest]?.digestCompareList) ? this.modelView[modelDigest]?.digestCompareList : [],
            worksetName: this.modelView[modelDigest]?.worksetName || '',
            taskName: this.modelView[modelDigest]?.taskName || ''
          }
    },

    // return copy tab items by model digest
    tabsView (modelDigest) {
      return (typeof modelDigest !== typeof 'string' || !this.modelView[modelDigest] || !Array.isArray(this?.modelView[modelDigest]?.tabs))
        ? []
        : Mdf.deepClone(this.modelView[modelDigest].tabs)
    },

    // return copy of parameter view by key
    paramView (key) {
      return this.paramViews?.[key]?.view ? Mdf.deepClone(this.paramViews[key].view) : undefined
    },

    // count parameters view by model digest
    paramViewCount (modelDigest) {
      const m = (typeof modelDigest === typeof 'string') ? modelDigest : '-'
      let n = 0
      for (const key in this.paramViews) {
        if (this.paramViews?.[key]?.digest === m) n++
      }
      return n
    },

    // count updated parameters by model digest
    paramViewUpdatedCount (modelDigest) {
      const m = (typeof modelDigest === typeof 'string') ? modelDigest : '-'
      let n = 0
      for (const key in this.paramViews) {
        if (!this.paramViews?.[key]?.digest === m) continue
        const pv = this.paramViews?.[key]?.view
        if (!!pv && pv.edit?.isUpdated && !!pv.edit.isUpdated) n++
      }
      return n
    },

    // count updated parameters by model digest and workset name
    paramViewWorksetUpdatedCount (mw) {
      if (!mw || !mw?.digest || !mw?.worksetName) return 0
      if (typeof mw.digest !== typeof 'string' || mw.digest === '') return 0
      if (typeof mw.worksetName !== typeof 'string' || mw.worksetName === '') return 0

      let n = 0
      for (const key in this.paramViews) {
        if (this.paramViews?.[key]?.digest === mw.digest && this.paramViews?.[key]?.worksetName === mw.worksetName) {
          const pv = this.paramViews?.[key]?.view
          if (!!pv && pv.edit?.isUpdated && !!pv.edit.isUpdated) n++
        }
      }
      return n
    },

    // return copy of table view by key
    tableView (key) {
      return this.tableViews?.[key]?.view ? Mdf.deepClone(this.tableViews[key].view) : undefined
    },

    // return copy of microdata view by key
    microdataView (key) {
      return this.mdViews?.[key]?.view ? Mdf.deepClone(this.mdViews[key].view) : undefined
    },

    // retrun tree expanded keys by model name and tree kind,  for example: table-tree
    treeExpandedKeys (modelName, treeKind) {
      if (!modelName && !treeKind) return []
      if (typeof modelName !== typeof 'string' || typeof treeKind !== typeof 'string') return []

      const expKeys = this?.uiTreeExpandedKeys?.[modelName + '*:*' + treeKind]
      if (!!expKeys && Array.isArray(expKeys)) {
        return Array.from(expKeys)
      }
      return []
    },

    //
    // actions
    //

    // set new language to select model text metadata
    dispatchUiLang (lang) {
      const modelStore = useModelStore()

      this.uiLang = (lang || '')
      modelStore.dispatchWordList(Mdf.emptyWordList()) // clear model word list
    },

    // store default document title
    dispatchDefaultTitle (title) { this.defaultTitle = (typeof title === typeof 'string' && !!title) ? title : 'OpenM++' },

    // set fast or full download: use accumulators or not
    dispatchNoAccDownload (noAcc) { this.noAccDownload = !!noAcc },

    // set fast or full download: use microdata or not
    dispatchNoMicrodataDownload (noMd) { this.noMicrodataDownload = !!noMd },

    // set download with id csv flag, by default download dimension items with enum codes
    dispatchIdCsvDownload (isIdCsv) { this.idCsvDownload = isIdCsv },

    // set tree label kind (parameter and table tree): name only, description only or both by default
    dispatchTreeLabelKind (labelKind) {
      this.treeLabelKind = (labelKind === 'name-only' || labelKind === 'descr-only') ? labelKind : ''
    },

    // save model list tree descending or ascending sort order
    dispatchSortModelTree (sort) {
      if (!sort || !sort?.isSort) {
        this.isSortModelTree = false
        this.isDescModelTree = false
      } else {
        this.isSortModelTree = !!sort?.isSort
        this.isDescModelTree = !!sort?.isDesc
      }
    },

    // save expanded state of tree by model name and tree kind,  for example: table-tree
    dispatchTreeExpandedKeys (modelName, treeKind, expandedKeys) {
      if (!modelName && !treeKind) return
      if (typeof modelName !== typeof 'string' || typeof treeKind !== typeof 'string') return

      const expKeys = (!!expandedKeys && Array.isArray(expandedKeys)) ? Array.from(expandedKeys) : []
      this.uiTreeExpandedKeys[modelName + '*:*' + treeKind] = expKeys
    },

    // update or clear selected run digest
    dispatchRunDigestSelected (modelView) {
      const mDgst = (typeof modelView?.digest === typeof 'string') ? modelView.digest : ''
      if (!mDgst) return

      if (typeof modelView?.runDigest === typeof 'string') {
        this.runDigestSelected = modelView.runDigest
        if (!this.modelView[mDgst]) this.modelView[mDgst] = Mdf.emptyModelView() // create model view state if not exist
        this.modelView[mDgst].runDigest = modelView.runDigest
        // this.modelView[mDgst].digestCompareList = []
      }
    },

    // add digest to the list of runs to compare
    dispatchAddRunCompareDigest (modelRunDigest) {
      const mDgst = (typeof modelRunDigest?.digest === typeof 'string') ? modelRunDigest.digest : ''
      const rDgst = (typeof modelRunDigest?.runDigest === typeof 'string') ? modelRunDigest.runDigest : ''
      if (!mDgst || !rDgst) return

      if (!this.modelView[mDgst]) this.modelView[mDgst] = Mdf.emptyModelView() // create model view state if not exist

      if (!this.modelView[mDgst].digestCompareList.includes(rDgst)) this.modelView[mDgst].digestCompareList.push(rDgst)
    },

    // remove digest from the list of runs to compare
    dispatchDeleteRunCompareDigest (modelRunDigest) {
      const mDgst = (typeof modelRunDigest?.digest === typeof 'string') ? modelRunDigest.digest : ''
      const rDgst = (typeof modelRunDigest?.runDigest === typeof 'string') ? modelRunDigest.runDigest : ''
      if (!mDgst || !rDgst) return

      if (!this.modelView[mDgst]) this.modelView[mDgst] = Mdf.emptyModelView() // create model view state if not exist

      const n = this.modelView[mDgst].digestCompareList.indexOf(rDgst)
      if (n >= 0) {
        this.modelView[mDgst].digestCompareList.splice(n, 1)
      }
    },

    // update or clear list of runs digest to compare
    dispatchRunCompareDigestList (modelView) {
      const mDgst = (typeof modelView?.digest === typeof 'string') ? modelView.digest : ''
      if (!mDgst) return

      if (Array.isArray(modelView?.digestCompareList)) {
        if (!this.modelView[mDgst]) this.modelView[mDgst] = Mdf.emptyModelView() // create model view state if not exist
        this.modelView[mDgst].digestCompareList = modelView.digestCompareList
      }
    },

    // update or clear selected workset name
    dispatchWorksetNameSelected (modelView) {
      const mDgst = (typeof modelView?.digest === typeof 'string') ? modelView.digest : ''
      if (!mDgst) return

      if (typeof modelView?.worksetName === typeof 'string') {
        this.worksetNameSelected = modelView.worksetName
        if (!this.modelView[mDgst]) this.modelView[mDgst] = Mdf.emptyModelView() // create model view state if not exist
        this.modelView[mDgst].worksetName = modelView.worksetName
      }
    },

    // replace tab items with new value
    dispatchTabsView (modelView) {
      const mDgst = (typeof modelView?.digest === typeof 'string') ? modelView.digest : ''
      if (!mDgst || !Array.isArray(modelView?.tabs)) return

      if (!this.modelView[mDgst]) this.modelView[mDgst] = Mdf.emptyModelView() // create model view state if not exist

      this.modelView[mDgst].tabs = []
      for (const t of modelView.tabs) {
        if (typeof t?.kind === typeof 'string' && t?.routeParts?.digest === mDgst) {
          this.modelView[mDgst].tabs.push({ kind: t.kind, routeParts: t.routeParts })
        }
      }
    },

    // delete parameter views, table views and model tab list by model digest
    dispatchViewDeleteByModel (modelDigest) {
      this.paramViewDeleteByModel(modelDigest)
      this.tableViewDeleteByModel(modelDigest)

      // clear model view
      const mDgst = (typeof modelDigest === typeof 'string') ? modelDigest : ''
      if (!mDgst) return

      if (this.modelView[mDgst]) {
        if ((this.runDigestSelected || '') !== '' && (this.modelView[mDgst]?.runDigest || '') === this.runDigestSelected) {
          this.runDigestSelected = ''
          this.modelView[mDgst].digestCompareList = []
        }
        if ((this.worksetNameSelected || '') !== '' && (this.modelView[mDgst]?.worksetName || '') === this.worksetNameSelected) {
          this.worksetNameSelected = ''
        }
        this.modelView[mDgst] = Mdf.emptyModelView()
      }
    },

    // restore restore model view selection: run digest, workset name and task name by model digest
    dispatchViewSelectedRestore (modelDigest) {
      const mDgst = (typeof modelDigest === typeof 'string') ? modelDigest : ''
      if (!mDgst || !this.modelView[mDgst]) return

      this.runDigestSelected = this.modelView[mDgst]?.runDigest || ''
      this.worksetNameSelected = this.modelView[mDgst]?.worksetName || ''
      this.taskNameSelected = this.modelView[mDgst]?.taskName || ''
    },

    // insert or update parameter view by route key
    dispatchParamView (pv) {
      if (!pv || !pv?.key) return
      if (typeof pv.key !== typeof 'string' || pv.key === '') return

      // insert new or replace existing parameter view
      if (pv?.view) {
        this.paramViews[pv.key] = Mdf.deepClone({
          view: pv.view,
          digest: pv?.digest || '',
          modelName: pv?.modelName || '',
          runDigest: pv?.runDigest || '',
          worksetName: pv?.worksetName || '',
          parameterName: pv?.parameterName || ''
        })
        return
      }
      // else: update existing parameter view
      if (!this.paramViews?.[pv.key]?.view) return // parameter view not found

      if (Array.isArray(pv?.rows)) {
        this.paramViews[pv.key].view.rows = Mdf.deepClone(pv.rows)
      }
      if (Array.isArray(pv?.cols)) {
        this.paramViews[pv.key].view.cols = Mdf.deepClone(pv.cols)
      }
      if (Array.isArray(pv?.others)) {
        this.paramViews[pv.key].view.others = Mdf.deepClone(pv.others)
      }
      if (typeof pv?.isRowColControls === typeof true) {
        this.paramViews[pv.key].view.isRowColControls = pv.isRowColControls
      }
      if (typeof pv?.rowColMode === typeof 1) {
        this.paramViews[pv.key].view.rowColMode = pv.rowColMode
      }
      if (pv?.edit) {
        this.paramViews[pv.key].view.edit = Mdf.deepClone(pv.edit)
      }
      if (typeof pv?.pageStart === typeof 1) {
        this.paramViews[pv.key].view.pageStart = pv.pageStart
      }
      if (typeof pv?.pageSize === typeof 1) {
        this.paramViews[pv.key].view.pageSize = pv.pageSize
      }
    },

    // delete parameter view by route key, if exist
    dispatchParamViewDelete (key) {
      if (typeof key === typeof 'string' && this.paramViews?.[key]) delete this.paramViews[key]
    },

    // delete all parameters view by model digest
    dispatchParamViewDeleteByModel (modelDigest) {
      const m = (typeof modelDigest === typeof 'string') ? modelDigest : '-'
      for (const key in this.paramViews) {
        if (this.paramViews?.[key]?.digest === m) delete this.paramViews[key]
      }
    },

    // insert or replace table view by route key
    dispatchTableView (tv) {
      if (!tv || !tv?.key) return
      if (typeof tv.key !== typeof 'string' || tv.key === '') return

      // insert new or replace existing table view
      if (tv?.view) {
        this.tableViews[tv.key] = Mdf.deepClone({
          view: tv.view,
          digest: tv?.digest || '',
          modelName: tv?.modelName || '',
          runDigest: tv?.runDigest || '',
          tableName: tv?.tableName || ''
        })
        return
      }
      // else: update existing output table view
      if (!this.tableViews?.[tv.key]?.view) return // output table view not found

      if (Array.isArray(tv?.rows)) {
        this.tableViews[tv.key].view.rows = Mdf.deepClone(tv.rows)
      }
      if (Array.isArray(tv?.cols)) {
        this.tableViews[tv.key].view.cols = Mdf.deepClone(tv.cols)
      }
      if (Array.isArray(tv?.others)) {
        this.tableViews[tv.key].view.others = Mdf.deepClone(tv.others)
      }
      if (typeof tv?.isRowColControls === typeof true) {
        this.tableViews[tv.key].view.isRowColControls = tv.isRowColControls
      }
      if (typeof tv?.rowColMode === typeof 1) {
        this.tableViews[tv.key].view.rowColMode = tv.rowColMode
      }
      if (typeof tv?.kind === typeof 1) {
        this.tableViews[tv.key].view.kind = tv.kind % 5 || 0 // table has only 5 possible view kinds
      }
      if (typeof tv?.calc === typeof 'string') {
        this.tableViews[tv.key].view.calc = tv.calc
      }
      if (typeof tv?.pageStart === typeof 1) {
        this.tableViews[tv.key].view.pageStart = tv.pageStart
      }
      if (typeof tv?.pageSize === typeof 1) {
        this.tableViews[tv.key].view.pageSize = tv.pageSize
      }
      if (Array.isArray(tv?.valueFilter)) {
        this.tableViews[tv.key].view.valueFilter = Mdf.deepClone(tv.valueFilter)
      }
      if (typeof tv?.scaleId === typeof 1) {
        this.tableViews[tv.key].view.scaleId = tv.scaleId
      }
      if (typeof tv?.scaleCalc === typeof 1) {
        this.tableViews[tv.key].view.scaleCalc = tv.scaleCalc
      }
      if (typeof tv?.isChart === typeof true) {
        this.tableViews[tv.key].view.isChart = tv.isChart
      }
      if (typeof tv?.chartType === typeof 'string') {
        this.tableViews[tv.key].view.chartType = tv.chartType
      }
      if (typeof tv?.prctChartSize === typeof 1) {
        this.tableViews[tv.key].view.prctChartSize = tv.prctChartSize
      }
      if (typeof tv?.isChartLabels === typeof true) {
        this.tableViews[tv.key].view.isChartLabels = tv.isChartLabels
      }
    },

    // delete table view by route key, if exist (key must be a string)
    dispatchTableViewDelete (key) {
      if (typeof key === typeof 'string' && this.tableViews?.[key]) delete this.tableViews[key]
    },

    // delete table view by model digest
    dispatchTableViewDeleteByModel (modelDigest) {
      const m = (typeof modelDigest === typeof 'string') ? modelDigest : '-'
      for (const key in this.tableViews) {
        if (this.tableViews?.[key]?.digest === m) delete this.tableViews[key]
      }
    },

    // insert or replace microdata view by route key
    dispatchMicrodataView (mv) {
      if (!mv || !mv?.key) return
      if (typeof mv.key !== typeof 'string' || mv.key === '') return

      // insert new or replace existing microdata view
      if (mv?.view) {
        this.mdViews[mv.key] = Mdf.deepClone({
          view: mv.view,
          digest: mv?.digest || '',
          modelName: mv?.modelName || '',
          runDigest: mv?.runDigest || '',
          entityName: mv?.entityName || ''
        })
        return
      }
      // else: update existing microdata view
      if (!this.mdViews?.[mv.key]?.view) return // microdata view not found

      if (Array.isArray(mv?.rows)) {
        this.mdViews[mv.key].view.rows = Mdf.deepClone(mv.rows)
      }
      if (Array.isArray(mv?.cols)) {
        this.mdViews[mv.key].view.cols = Mdf.deepClone(mv.cols)
      }
      if (Array.isArray(mv?.others)) {
        this.mdViews[mv.key].view.others = Mdf.deepClone(mv.others)
      }
      if (typeof mv?.isRowColControls === typeof true) {
        this.mdViews[mv.key].view.isRowColControls = mv.isRowColControls
      }
      if (typeof mv?.rowColMode === typeof 1) {
        this.mdViews[mv.key].view.rowColMode = mv.rowColMode
      }
      if (typeof mv?.pageStart === typeof 1) {
        this.mdViews[mv.key].view.pageStart = mv.pageStart
      }
      if (typeof mv?.pageSize === typeof 1) {
        this.mdViews[mv.key].view.pageSize = mv.pageSize
      }
      if (typeof mv?.aggrCalc === typeof 'string') {
        this.mdViews[mv.key].view.aggrCalc = mv.aggrCalc
      }
      if (typeof mv?.cmpCalc === typeof 'string') {
        this.mdViews[mv.key].view.cmpCalc = mv.cmpCalc
      }
      if (Array.isArray(mv?.groupBy)) {
        this.mdViews[mv.key].view.groupBy = Mdf.deepClone(mv.groupBy)
      }
      if (Array.isArray(mv?.calcEnums)) {
        this.mdViews[mv.key].view.calcEnums = Mdf.deepClone(mv.calcEnums)
      }
      if (Array.isArray(mv?.valueFilter)) {
        this.mdViews[mv.key].view.valueFilter = Mdf.deepClone(mv.valueFilter)
      }
    },

    // delete microdata view by route key, if exist
    dispatchMicrodataViewDelete (key) {
      if (typeof key === typeof 'string' && this.mdViews?.[key]) delete this.mdViews[key]
    },

    // delete microdata's view by model digest
    dispatchMicrodataViewDeleteByModel (modelDigest) {
      const m = (typeof modelDigest === typeof 'string') ? modelDigest : '-'
      for (const key in this.mdViews) {
        if (this.mdViews?.[key]?.digest === m) delete this.mdViews[key]
      }
    }
  }
})


if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useUiStateStore, import.meta.hot))
}
