import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'
import RunParameterList from 'components/RunParameterList.vue'
import TableList from 'components/TableList.vue'
import EntityList from 'components/EntityList.vue'
import RefreshRun from 'components/RefreshRun.vue'
import RefreshRunArray from 'components/RefreshRunArray.vue'
import RunInfoDialog from 'components/RunInfoDialog.vue'
import ParameterInfoDialog from 'components/ParameterInfoDialog.vue'
import TableInfoDialog from 'components/TableInfoDialog.vue'
import GroupInfoDialog from 'components/GroupInfoDialog.vue'
import EntityInfoDialog from 'components/EntityInfoDialog.vue'
import EntityAttrInfoDialog from 'components/EntityAttrInfoDialog.vue'
import EntityGroupInfoDialog from 'components/EntityGroupInfoDialog.vue'
import DeleteConfirmDialog from 'components/DeleteConfirmDialog.vue'
import NewWorkset from 'components/NewWorkset.vue'
import CreateWorkset from 'components/CreateWorkset.vue'
import MarkdownEditor from 'components/MarkdownEditor.vue'

export default {
  name: 'RunList',
  components: {
    RunParameterList,
    TableList,
    EntityList,
    RefreshRun,
    RefreshRunArray,
    RunInfoDialog,
    ParameterInfoDialog,
    TableInfoDialog,
    GroupInfoDialog,
    EntityInfoDialog,
    EntityAttrInfoDialog,
    EntityGroupInfoDialog,
    DeleteConfirmDialog,
    NewWorkset,
    CreateWorkset,
    MarkdownEditor
  },

  props: {
    digest: { type: String, default: '' },
    refreshTickle: { type: Boolean, default: false }
  },

  data () {
    return {
      loadWait: false,
      loadRunWait: false,
      loadRunArrayWait: false,
      runCurrent: Mdf.emptyRunText(), // currently selected run
      isRunTreeCollapsed: false,
      runTreeData: [],
      runFilter: '',
      runTreeExpanded: [],
      runTreeTicked: [],
      firstCompareName: '', // run name of first run to compare
      compareDigestArray: [], // array of run digests to compare
      paramDiff: [], // name list of different parameters
      tableDiff: [], // name list of different tables
      entityDiff: [], // name list of different microdata entities
      entityAttrsUse: [], // entity.attribute names included in current run, filtered by run compare
      isNewWorksetShow: false,
      isCreateWorksetNow: false,
      loadWorksetCreate: false,
      nameOfNewWorkset: '',
      copyDigestNewWorkset: '',
      copyParamNewWorkset: [],
      newDescrNotes: [], // new workset description and notes
      isParamTreeShow: false,
      paramTreeCount: 0,
      paramVisibleCount: 0,
      refreshRunTickle: false,
      runDigestRefresh: '',
      refreshRunArrayTickle: false,
      refreshParamTreeTickle: false,
      isTableTreeShow: false,
      tableTreeCount: 0,
      tableVisibleCount: 0,
      refreshTableTreeTickle: false,
      isEntityTreeShow: false,
      entityTreeCount: 0, // total number of entities in current run
      entityVisibleCount: 0, // number of entities visible after run compare
      refreshEntityTreeTickle: false,
      runInfoTickle: false,
      runInfoDigest: '',
      groupInfoTickle: false,
      groupInfoName: '',
      paramInfoTickle: false,
      paramInfoName: '',
      tableInfoTickle: false,
      tableInfoName: '',
      attrInfoName: '',
      entityInfoName: '',
      entityGroupInfoName: '',
      attrInfoTickle: false,
      entityInfoTickle: false,
      entityGroupInfoTickle: false,
      runNameToDelete: '',
      runDigestToDelete: '',
      runStatusToDelete: '',
      showDeleteDialogTickle: false,
      runMultipleCount: 0,
      showDeleteMultipleDialogTickle: false,
      uploadFileSelect: false,
      uploadFile: null,
      isShowNoteEditor: false,
      noteEditorLangCode: ''
    }
  },

  computed: {
    isNotEmptyRunCurrent () { return Mdf.isNotEmptyRunText(this.runCurrent) },
    descrRunCurrent () { return Mdf.descrOfTxt(this.runCurrent) },
    isCompare () { return Mdf.lengthOf(this.compareDigestArray) > 0 },
    fileSelected () { return !(this.uploadFile === null) },
    isMicrodata () { return !!this.serverConfig.AllowMicrodata && Mdf.entityCount(this.theModel) > 0 },
    isDiskOver () { return !!this?.serverConfig?.IsDiskUse && !!this.diskUseState?.DiskUse?.IsOver },

    ...mapState(useModelStore, [
      'theModel',
      'runTextList',
      'runTextListUpdated'
    ]),
    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl',
      serverConfig: 'config',
      diskUseState: 'diskUse'
    }),
    ...mapState(useUiStateStore, [
      'uiLang',
      'runDigestSelected',
      'noAccDownload',
      'noMicrodataDownload',
      'idCsvDownload'
    ])
  },

  watch: {
    digest () { this.doRefresh() },
    refreshTickle () { this.doRefresh() },
    runTextListUpdated () { this.onRunTextListUpdated() },
    runDigestSelected () {
      this.runCurrent = this.runTextByDigest({ ModelDigest: this.digest, RunDigest: this.runDigestSelected })
      this.refreshRunCompare()
    }
  },

  emits: [
    'run-select',
    'run-list-refresh',
    'run-list-delete',
    'run-log-select',
    'set-select',
    'run-parameter-select',
    'table-select',
    'entity-select',
    'download-select',
    'upload-select',
    'tab-mounted'
  ],

  methods: {
    ...mapActions(useModelStore, [
      'runTextByDigest'
    ]),
    ...mapActions(useUiStateStore, [
      'modelViewSelected',
      //
      'dispatchAddRunCompareDigest',
      'dispatchDeleteRunCompareDigest',
      'dispatchRunCompareDigestList'
    ]),

    isSuccess (status) { return status === Mdf.RUN_SUCCESS },
    isInProgress (status) { return status === Mdf.RUN_IN_PROGRESS || status === Mdf.RUN_INITIAL },
    isRunDeleted (status, name) { return Mdf.isRunDeletedStatus(status, name) },
    dateTimeStr (dt) { return Mdf.dtStr(dt) },
    runCurrentDescr () { return Mdf.descrOfTxt(this.runCurrent) },
    runCurrentNote () { return Mdf.noteOfTxt(this.runCurrent) },
    isDigestCompare (dgst) { return this.compareDigestArray.includes(dgst) },

    // update page view
    doRefresh () {
      this.doRunTextListUpdated()
      this.paramTreeCount = Mdf.paramCount(this.theModel)
      this.paramVisibleCount = this.paramTreeCount
      this.tableTreeCount = Mdf.tableCount(this.theModel)
      this.tableVisibleCount = this.tableTreeCount
      this.entityTreeCount = this.runCurrent.Entity.length
      this.entityVisibleCount = this.entityTreeCount
      this.entityAttrsUse = []
      this.refreshRunCompare()
    },
    // run list updated: update run list tree and current run
    onRunTextListUpdated () { this.doRunTextListUpdated() },
    doRunTextListUpdated () {
      this.runTreeData = this.makeRunTreeData(this.runTextList)
      if (this.runTreeData?.length > 0) {
        const rtTop = this.runTreeData[0]
        this.runTreeExpanded = [rtTop.key]

        const tn = []
        for (const rt of rtTop.children) {
          if (this.runTreeTicked.findIndex((rtKey) => { return rtKey === rt.key }) >= 0) {
            tn.push(rt.key)
          }
        }
        this.runTreeTicked = tn
      }
      if (this.runDigestSelected) this.runCurrent = this.runTextByDigest({ ModelDigest: this.digest, RunDigest: this.runDigestSelected })
    },
    // update run comparison view
    refreshRunCompare () {
      const mv = this.modelViewSelected(this.digest)
      if (!!mv && Array.isArray(mv?.digestCompareList)) {
        this.compareDigestArray = mv.digestCompareList
        this.refreshRunArrayTickle = !this.refreshRunArrayTickle
      }
    },

    // click on run: select this run as current run
    onRunLeafClick (dgst) {
      this.doRunDigestSelect(dgst)
    },
    // select a new run digest
    doRunDigestSelect (dgst) {
      if (!dgst || this.runDigestSelected === dgst) return // there is no new run digest
      this.clearRunCompare()
      this.$emit('run-select', dgst)
    },
    // expand or collapse all run tree nodes
    doToogleExpandRunTree () {
      if (this.isRunTreeCollapsed) {
        this.$refs.runTree.expandAll()
      } else {
        this.$refs.runTree.collapseAll()
      }
      this.isRunTreeCollapsed = !this.isRunTreeCollapsed
    },
    // filter run tree nodes by name (label), update date-time or description
    doRunTreeFilter (node, filter) {
      if (node.key === 'rtl-top-node') return true // always show top node: it is tree controls
      const flt = filter.toLowerCase()
      return (node.label && node.label.toLowerCase().indexOf(flt) > -1) ||
        ((node.lastTime || '') !== '' && node.lastTime.indexOf(flt) > -1) ||
        ((node.descr || '') !== '' && node.descr.toLowerCase().indexOf(flt) > -1)
    },
    // clear run tree filter value
    resetRunFilter () {
      this.runFilter = ''
      this.$refs.runFilterInput.focus()
    },
    // show run notes dialog
    doShowRunNote (dgst) {
      this.runInfoDigest = dgst
      this.runInfoTickle = !this.runInfoTickle
    },
    // click on run log: show run log page
    doRunLogClick (stamp) {
      // if run stamp is empty then log is not available
      if (!stamp) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to show run log: run stamp is empty') })
        return
      }
      this.$emit('run-log-select', stamp)
    },

    // show run description and notes dialog
    onEditRunNote (dgst) {
      this.isShowNoteEditor = true
      this.noteEditorLangCode = this.uiLang || this.$q.lang.getLocale() || ''
      this.isParamTreeShow = false
      this.isTableTreeShow = false
    },
    // save run notes editor content
    onSaveRunNote (descr, note, isUpd, lc) {
      this.doSaveRunNote(this.runDigestSelected, lc, descr, note)
      this.isShowNoteEditor = false
    },
    onCancelRunNote (dgst) {
      this.isShowNoteEditor = false
    },

    // array of runs to compare loaded from the server
    doneRunArrayLoad (isSuccess, count) {
      this.loadRunArrayWait = false
      if (!isSuccess || !Mdf.isRunTextList(this.runTextList)) return

      // check if all runs to compare exists in run list
      const isCmp = this.compareDigestArray.length > 0
      const dArr = []
      for (const dg of this.compareDigestArray) {
        if (this.runTextList.findIndex((rt) => { return rt.RunDigest === dg }) >= 0) {
          dArr.push(dg)
        }
      }

      // update run comparison state
      if (dArr.length <= 0) {
        if (isCmp) this.clearRunCompare()
      } else {
        this.compareDigestArray = dArr
        this.updateRunCompare()
        this.dispatchRunCompareDigestList({ digest: this.digest, digestCompareList: dArr })
      }
    },
    // update run comparison: list of different parameters, different tables and suppressed tables
    updateRunCompare () {
      if (!Mdf.isLength(this.compareDigestArray)) return

      const rc = Mdf.runCompare(this.runCurrent, this.compareDigestArray, Mdf.tableCount(this.theModel), this.runTextList)

      // notify user about results
      if (rc.paramDiff.length) {
        this.paramVisibleCount = Mdf.paramCount(this.theModel)
        this.$q.notify({ type: 'info', message: this.$t('Number of different parameters: ') + rc.paramDiff.length })
      } else {
        this.$q.notify({ type: 'info', message: this.$t('All parameters values identical') })
      }
      if (rc.tableDiff.length) {
        this.tableVisibleCount = Mdf.tableCount(this.theModel)
        this.$q.notify({ type: 'info', message: this.$t('Number of different output tables: ') + rc.tableDiff.length })
      } else {
        if (!rc.tableSupp.length) this.$q.notify({ type: 'info', message: this.$t('All output tables values identical') })
      }
      if (rc.tableSupp.length) {
        this.$q.notify({ type: 'info', message: this.$t('Number of output tables which are not found: ') + rc.tableSupp.length })
      }
      if (this.isMicrodata && !!this.runCurrent.Entity.length) {
        // update list of visible microdata attributes:
        // include all entity.attribute from current run if entity is missing in variant run(s)
        this.entityAttrsUse = []

        for (const e of this.runCurrent.Entity) {
          if (rc.entityMiss.findIndex((em) => { return em === e.Name }) >= 0) continue // skip missing entity
          if (e?.Name && Array.isArray(e?.Attr)) {
            for (const a of e.Attr) {
              this.entityAttrsUse.push(e.Name + '.' + a)
            }
          }
        }

        if (rc.entityDiff.length) {
          this.entityVisibleCount = this.runCurrent.Entity.length
          this.$q.notify({ type: 'info', message: this.$t('Number of different microdata: ') + rc.entityDiff.length })
        } else {
          if (!rc.entityMiss.length) this.$q.notify({ type: 'info', message: this.$t('All microdata values identical') })
        }
        if (rc.entityMiss.length) {
          this.$q.notify({ type: 'info', message: this.$t('Number of microdata entities which are not found: ') + rc.entityMiss.length })
        }
      }

      this.firstCompareName = rc.firstRunName
      this.paramDiff = Object.freeze(rc.paramDiff)
      this.tableDiff = Object.freeze(rc.tableDiff)
      this.entityDiff = Object.freeze(rc.entityDiff)
      this.refreshParamTreeTickle = !this.refreshParamTreeTickle
      this.refreshTableTreeTickle = !this.refreshTableTreeTickle
      this.refreshEntityTreeTickle = !this.refreshEntityTreeTickle
    },

    // run comparison click: set or clear run comparison
    onRunCompareClick (dgst) {
      // if clear run comparison filter
      if (this.removeRunCompareDigest(dgst)) {
        if (this.compareDigestArray.length > 0) {
          this.updateRunCompare()
        } else {
          this.clearRunCompare()
        }
        return
      }
      // esle: start run comparison by refresh run
      this.runDigestRefresh = dgst
      this.refreshRunTickle = !this.refreshRunTickle
      this.isNewWorksetShow = false // clear new workset create panel
      this.resetNewWorkset()
    },
    // remove run digest from run compare list
    removeRunCompareDigest (dgst) {
      const nPos = this.compareDigestArray.indexOf(dgst)
      if (nPos < 0) return false

      this.compareDigestArray.splice(nPos, 1)
      this.dispatchDeleteRunCompareDigest({ digest: this.digest, runDigest: dgst })
      return true
    },

    // run to compare loaded from the server
    doneRunLoad (isSuccess, dgst) {
      this.loadRunWait = false
      if (!isSuccess) return

      const rt = this.runTextByDigest({ ModelDigest: this.digest, RunDigest: dgst })
      if (!Mdf.isNotEmptyRunText(rt)) {
        console.warn('Unable to compare run:', dgst, this.runDigestRefresh)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to compare model run') + ' ' + this.runDigestRefresh })
        this.runDigestRefresh = ''
        return
      }
      this.runDigestRefresh = ''

      // update run compare names
      if (this.compareDigestArray.indexOf(dgst) < 0) {
        this.compareDigestArray.push(dgst)
        this.updateRunCompare()
      }
      this.dispatchAddRunCompareDigest({ digest: this.digest, runDigest: dgst })
    },
    // clear run comparison
    clearRunCompare () {
      this.runDigestRefresh = ''
      this.compareDigestArray = []
      this.firstCompareName = ''
      this.paramDiff = []
      this.tableDiff = []
      this.entityDiff = []
      this.entityAttrsUse = []
      this.paramVisibleCount = this.paramTreeCount
      this.tableVisibleCount = this.tableTreeCount
      this.entityVisibleCount = this.entityTreeCount
      this.refreshParamTreeTickle = !this.refreshParamTreeTickle
      this.refreshTableTreeTickle = !this.refreshTableTreeTickle
      this.refreshEntityTreeTickle = !this.refreshEntityTreeTickle
      this.dispatchRunCompareDigestList({ digest: this.digest, digestCompareList: [] })
      // clear new workset create panel
      this.isNewWorksetShow = false
      this.resetNewWorkset()
    },

    // create new workset from parameters different between two runs
    //
    // return true to disable new workset create button click
    isNewWorksetDisabled () {
      return !this.isCompare ||
        this.compareDigestArray.length !== 1 || // allow to create new workset only from single run
        !Array.isArray(this.paramDiff) ||
        this.paramDiff.length <= 0 ||
        this.isNewWorksetShow ||
        this.isShowNoteEditor ||
        Mdf.isRunDeletedStatus(this.runCurrent.Status, this.runCurrent.Name)
    },
    // start create new workset
    onNewWorksetClick () {
      this.resetNewWorkset()
      this.isNewWorksetShow = true
      this.isParamTreeShow = false
      this.isTableTreeShow = false
    },
    // cancel creation of new workset
    onNewWorksetCancel () {
      this.resetNewWorkset()
      this.isNewWorksetShow = false
    },
    // send request to create new workset
    onNewWorksetSave (dgst, name, txt) {
      this.nameOfNewWorkset = name || ''
      this.copyDigestNewWorkset = ''
      this.copyParamNewWorkset = []

      // parameters to copy into the new workset from run compare
      if (this.compareDigestArray.length === 1) {
        this.copyDigestNewWorkset = this.compareDigestArray[0]

        const pLst = []
        for (const pn of this.paramDiff) {
          pLst.push({
            name: pn,
            isGroup: false
          })
        }
        this.copyParamNewWorkset = Object.freeze(pLst)
      }

      this.newDescrNotes = txt || []
      this.isCreateWorksetNow = true
    },
    // request to create workset completed
    doneWorksetCreate  (isSuccess, dgst, name) {
      this.loadWorksetCreate = false
      this.resetNewWorkset()

      if (!isSuccess) return // workset not created: keep user input

      this.isNewWorksetShow = false // workset created: close section

      if (dgst && name && dgst === this.digest) { // if the same model then refresh workset from server
        this.$emit('set-select', name)
      }
    },
    // clean new workset data
    resetNewWorkset () {
      this.isCreateWorksetNow = false
      this.nameOfNewWorkset = ''
      this.copyDigestNewWorkset = ''
      this.copyParamNewWorkset = []
      this.newDescrNotes = []
    },
    // end of create new workset
    //

    // show yes/no dialog to confirm run delete
    onRunDelete (runName, dgst, status) {
      this.runNameToDelete = runName
      this.runDigestToDelete = dgst
      this.runStatusToDelete = status
      this.showDeleteDialogTickle = !this.showDeleteDialogTickle
    },
    // user answer yes to confirm delete model run
    onYesRunDelete (runName, dgst) {
      let isClear = this.runDigestSelected === dgst
      let isCmp = false
      if (!isClear) {
        isCmp = this.removeRunCompareDigest(dgst)
        isClear = this.compareDigestArray.length <= 0
      }
      if (isClear) {
        this.clearRunCompare()
      } else {
        if (isCmp) this.updateRunCompare()
      }

      this.doRunDeleteStart(dgst, runName)
    },
    // delete multiple selected runs
    onRunMultipleDelete () {
      if (!this.runTreeTicked?.length) return // no runs selected

      this.runMultipleCount = this.runTreeTicked.length
      this.showDeleteMultipleDialogTickle = !this.showDeleteMultipleDialogTickle
    },
    // user answer yes to confirm delete multiple selected model runs
    onYesRunMultipleDelete () {
      if (!this.runTreeData?.length || !this.runTreeTicked?.length) {
        console.warn('Unable to delete: invalid (empty) run tree or ticked runs', this.runTreeData?.length, this.runTreeTicked?.length)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to delete: model runs list is empty') })
        return
      }

      // collect list of run digests to delete
      const rmLst = []
      let isCmp = false
      let isCur = false
      let firstDgst = ''
      const rtTop = this.runTreeData[0]

      for (const rt of rtTop.children) {
        if (!rt.digest || rt.children.length > 0) continue // it is not a model run

        if (this.runTreeTicked.findIndex((rtKey) => { return rtKey === rt.key }) < 0) { // this model run is not selected
          if (firstDgst === '') firstDgst = rt.digest
          continue
        }
        if (!isCur) {
          isCur = this.runDigestSelected === rt.digest
        }
        if (!isCur) {
          isCmp = this.removeRunCompareDigest(rt.digest)
        }

        rmLst.push(rt.digest)
      }

      this.doRunDeleteMultipleStart(rmLst) // start delete

      // clear selection
      // clear or update run comparison if current run deleted or any comparison runs deleted
      this.runTreeTicked = []

      if (isCur || this.compareDigestArray.length <= 0) {
        this.clearRunCompare()
      } else {
        if (isCmp) this.updateRunCompare()
      }
      if (isCur) {
        this.doRunDigestSelect(firstDgst)
      }
    },

    // click on run download: start run download and show download list page
    onDownloadRun (dgst) {
      // if run digest is empty or run not completed successfully then do not show download page
      if (!dgst || !Mdf.isRunSuccess(
        this.runTextByDigest({ ModelDigest: this.digest, RunDigest: dgst })
      )) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to download model run, it is not completed successfully') })
        return
      }

      this.startRunDownload(dgst) // start run download and show download page on success
    },

    // show or hide parameters tree
    onToogleShowParamTree () {
      this.isParamTreeShow = !this.isParamTreeShow
      this.isTableTreeShow = false
      this.isEntityTreeShow = false
    },
    // click on parameter: open current run parameter values tab
    onRunParamClick (name) {
      this.$emit('run-parameter-select', name)
    },
    // show run parameter notes dialog
    // parameters tree updated and leafs counted
    onParamTreeUpdated  (cnt) {
      this.paramTreeCount = cnt || 0
      this.paramVisibleCount = !this.isCompare ? this.paramTreeCount : Mdf.paramCount(this.theModel)
    },
    doShowParamNote (name) {
      this.paramInfoName = name
      this.paramInfoTickle = !this.paramInfoTickle
    },
    // show group notes dialog
    doShowGroupNote (name) {
      this.groupInfoName = name
      this.groupInfoTickle = !this.groupInfoTickle
    },

    // show or hide output tables tree
    onToogleShowTableTree () {
      this.isTableTreeShow = !this.isTableTreeShow
      this.isParamTreeShow = false
      this.isEntityTreeShow = false
    },
    // click on output table: open current run output table values tab
    onTableLeafClick (name) {
      this.$emit('table-select', name)
    },
    // tables tree updated and leafs counted
    onTableTreeUpdated (cnt) {
      this.tableTreeCount = cnt || 0
      this.tableVisibleCount = !this.isCompare ? this.tableTreeCount : Mdf.tableCount(this.theModel)
    },
    // show run output table notes dialog
    doShowTableNote (name) {
      this.tableInfoName = name
      this.tableInfoTickle = !this.tableInfoTickle
    },

    // entity selected: show entity data page if this entity row count is non zero for current model run
    onEntityClick (name, parts) {
      let isEmpty = true
      for (const e of this.runCurrent.Entity) {
        if (e?.Name && e.Name === name && Array.isArray(e?.Attr)) {
          isEmpty = (e?.RowCount || 0) <= 0 // microdata row count in that model run
        }
      }
      if (isEmpty) {
        this.$q.notify({ type: 'negative', message: this.$t('Entity microdata is empty') })
        return
      }
      this.$emit('entity-select', name)
    },
    // show or hide output microdata entity tree
    onToogleShowEntityTree () {
      this.isEntityTreeShow = !this.isEntityTreeShow
      this.isParamTreeShow = false
      this.isTableTreeShow = false
    },
    // entity tree updated and leafs counted
    onEntityTreeUpdated  (entCount, leafCount) {
      this.entityTreeCount = entCount || 0
      this.entityVisibleCount = !this.isCompare ? this.entityTreeCount : this.runCurrent.Entity.length
    },
    // show run entity info dialog: entity and attributes
    doShowEntityRunNote (entName) {
      this.entityInfoName = entName
      this.entityInfoTickle = !this.entityInfoTickle
    },
    // show entity attribute notes dialog
    doShowEntityAttrNote (attrName, entName) {
      this.attrInfoName = attrName
      this.entityInfoName = entName
      this.attrInfoTickle = !this.attrInfoTickle
    },
    // show entity attributes group notes dialog
    doShowEntityGroupNote (groupName, entName) {
      this.entityInfoName = entName
      this.entityGroupInfoName = groupName
      this.entityGroupInfoTickle = !this.entityGroupInfoTickle
    },

    // return tree of model runs
    makeRunTreeData (rLst) {
      this.runFilter = ''

      if (!Mdf.isLength(rLst)) return [] // empty run list
      if (!Mdf.isRunTextList(rLst)) {
        this.$q.notify({ type: 'negative', message: this.$t('Model run list is empty or invalid') })
        return [] // invalid run list
      }

      // add runs which are not included in any group
      const td = []
      const tdTop = {
        key: 'rtl-top-node',
        digest: 'rtl-top-digest',
        label: '',
        stamp: '',
        status: '',
        lastTime: '',
        descr: '',
        children: [],
        disabled: false
      }
      td.push(tdTop)

      for (const r of rLst) {
        tdTop.children.push({
          key: 'rtl-' + r.RunDigest,
          digest: r.RunDigest,
          label: r.Name,
          stamp: r.RunStamp,
          status: r.Status,
          lastTime: Mdf.dtStr(r.UpdateDateTime),
          descr: Mdf.descrOfTxt(r),
          children: [],
          disabled: false
        })
      }
      return td
    },

    // start run delete
    async doRunDeleteStart (dgst, runName) {
      if (!dgst) {
        console.warn('Unable to delete: invalid (empty) run digest')
        return
      }
      this.$q.notify({ type: 'info', message: this.$t('Deleting: ') + dgst + ' ' + (runName || '') })
      this.loadWait = true

      let isOk = false
      const u = this.omsUrl +
        '/api/model/' + encodeURIComponent(this.digest) +
        '/run/' + encodeURIComponent((dgst || ''))
      try {
        await this.$axios.delete(u) // response expected to be empty on success
        isOk = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Error at delete model run', dgst, runName, em)
      }
      this.loadWait = false
      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to delete: ') + dgst + ' ' + (runName || '') })
        return
      }

      // refresh run list from the server
      // this.$q.notify({ type: 'info', message: this.$t('Start deleting: ') + dgst + ' ' + (runName || '') })
      setTimeout(
        () => {
          this.$emit('run-list-refresh')
          setTimeout(() => this.$emit('run-list-delete'), 521)
        },
        521)
    },

    // start to delete multiple model runs
    async doRunDeleteMultipleStart (dgLst) {
      if (!dgLst || !Array.isArray(dgLst) || !dgLst?.length) {
        console.warn('Unable to delete: invalid (or empty) list of runs, length:', dgLst?.length)
        return
      }
      const nLen = dgLst.length
      this.$q.notify({ type: 'info', message: this.$t('Deleting multiple model runs') + ': [ ' + nLen.toString() + ' ]' })
      this.loadWait = true

      let isOk = false
      const u = this.omsUrl + '/api/model/' + encodeURIComponent(this.digest) + '/delete-runs'
      try {
        await this.$axios.post(u, dgLst) // ignore response on success
        isOk = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Error at delete model runs, length:', nLen, em)
      }
      this.loadWait = false
      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to delete multiple model runs') + ' [ ' + nLen.toString() + ' ]' })
        return
      }

      // refresh run list from the server
      // this.$q.notify({ type: 'info', message: this.$t('Start deleting: ') + dgst + ' ' + (runName || '') })
      setTimeout(
        () => {
          this.$emit('run-list-refresh')
          setTimeout(() => this.$emit('run-list-delete'), 521)
        },
        2011)
    },

    // start run download
    async startRunDownload (dgst) {
      let isOk = false
      let msg = ''

      const opts = {
        NoAccumulatorsCsv: this.noAccDownload,
        NoMicrodata: this.noMicrodataDownload,
        Utf8BomIntoCsv: this.$q.platform.is.win,
        IdCsv: !!this.idCsvDownload
      }
      const u = this.omsUrl +
        '/api/download/model/' + encodeURIComponent(this.digest) +
        '/run/' + encodeURIComponent((dgst || '')) +
        (this.uiLang !== '' ? '/lang/' + encodeURIComponent(this.uiLang) : '')
      try {
        // send download request to the server, response expected to be empty on success
        await this.$axios.post(u, opts)
        isOk = true
      } catch (e) {
        try {
          if (e.response) msg = e.response.data || ''
        } finally {}
        console.warn('Unable to download model run', msg)
      }
      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to download model run: ') + (msg || '') })
        return
      }

      this.$emit('download-select', this.digest) // download started: show download list page
      this.$q.notify({ type: 'info', message: this.$t('Model run download started') })
    },

    // show model run upload panel
    doShowFileSelect () {
      this.uploadFileSelect = true
    },
    // hides model upload panel
    doCancelFileSelect () {
      this.uploadFileSelect = false
      this.uploadFile = null
    },

    // upload model run zip file
    async onUploadRun () {
      // check file name and notify user
      const fName = this.uploadFile?.name
      if (!fName) {
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) file name') })
        return
      }
      // check if model run with the same name already exist
      for (const rt of this.runTextList) {
        if (Mdf.isRunText(rt) && Mdf.modelName(this.theModel) + '.run.' + rt.Name + '.zip' === fName) {
          this.$q.notify({ type: 'negative', message: this.$t('Model run with the same name already exist: ') + rt.Name })
          return
        }
      }
      this.$q.notify({ type: 'info', message: this.$t('Uploading:') + fName + '\u2026' })

      // make upload multipart form
      const fd = new FormData()
      fd.append('run.zip', this.uploadFile, fName) // name and file name are ignored by server

      const u = this.omsUrl +
      '/api/upload/model/' +
      encodeURIComponent(this.digest) + '/run' +
      (this.uiLang !== '' ? '/lang/' + encodeURIComponent(this.uiLang) : '')
      try {
        // update run zip, drop response on success
        await this.$axios.post(u, fd)
      } catch (e) {
        let msg = ''
        try {
          if (e.response) msg = e.response.data || ''
        } finally {}
        console.warn('Unable to upload model run', msg, fName)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to upload model run: ') + fName })
        return
      }

      // notify user and close upload controls
      this.doCancelFileSelect()
      this.$q.notify({ type: 'info', message: this.$t('Uploaded: ') + fName })
      this.$q.notify({ type: 'info', message: this.$t('Import model run: ') + fName + '\u2026' })

      // upload started: show upload list page
      this.$emit('upload-select', this.digest)
      this.$emit('run-list-refresh') // refresh model run list if upload completed fast
    },

    // save run notes
    async doSaveRunNote (dgst, langCode, descr, note) {
      let isOk = false
      let msg = ''

      // validate current run is not empty and has a language
      if (!Mdf.isNotEmptyRunText(this.runCurrent)) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to save model run description and notes, current model run is undefined') })
        return
      }
      if (!langCode) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to save model run description and notes, language is unknown') })
        return
      }
      this.loadWait = true

      const u = this.omsUrl + '/api/run/text'
      const rt = {
        ModelDigest: this.digest,
        RunDigest: dgst,
        Txt: [{
          LangCode: langCode,
          Descr: descr || '',
          Note: note || ''
        }]
      }
      try {
        // send run description and notes, response expected to be empty on success
        await this.$axios.patch(u, rt)
        isOk = true
      } catch (e) {
        try {
          if (e.response) msg = e.response.data || ''
        } finally {}
        console.warn('Unable to save model run description and notes', msg)
      }
      this.loadWait = false
      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to save model run description and notes') + (msg ? ('. ' + msg) : '') })
        return
      }

      this.$emit('run-select', dgst)
      this.$q.notify({ type: 'info', message: this.$t('Model run description and notes saved') + '. ' + this.$t('Language: ') + langCode })
    }
  },

  mounted () {
    this.doRefresh()
    this.$emit('tab-mounted', 'run-list', { digest: this.digest })
  }
}
