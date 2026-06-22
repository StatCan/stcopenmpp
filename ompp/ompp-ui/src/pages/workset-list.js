import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'
import WorksetParameterList from 'components/WorksetParameterList.vue'
import RunParameterList from 'components/RunParameterList.vue'
import RefreshWorkset from 'components/RefreshWorkset.vue'
import WorksetInfoDialog from 'components/WorksetInfoDialog.vue'
import ParameterInfoDialog from 'components/ParameterInfoDialog.vue'
import GroupInfoDialog from 'components/GroupInfoDialog.vue'
import NewWorkset from 'components/NewWorkset.vue'
import CreateWorkset from 'components/CreateWorkset.vue'
import DeleteWorkset from 'components/DeleteWorkset.vue'
import ConfirmDialog from 'components/ConfirmDialog.vue'
import DeleteConfirmDialog from 'components/DeleteConfirmDialog.vue'
import MarkdownEditor from 'components/MarkdownEditor.vue'

export default {
  name: 'WorksetList',
  components: {
    WorksetParameterList,
    RunParameterList,
    RefreshWorkset,
    WorksetInfoDialog,
    ParameterInfoDialog,
    GroupInfoDialog,
    NewWorkset,
    CreateWorkset,
    DeleteWorkset,
    ConfirmDialog,
    DeleteConfirmDialog,
    MarkdownEditor
  },

  props: {
    digest: { type: String, default: '' },
    refreshTickle: { type: Boolean, default: false }
  },

  data () {
    return {
      loadWait: false,
      loadWsWait: false,
      refreshWsTickle: false,
      refreshWsFromTickle: false,
      refreshParamTreeTickle: false,
      refreshParamTreeFromTickle: false,
      worksetCurrent: Mdf.emptyWorksetText(), // currently selected workset
      isTreeCollapsed: false,
      wsTreeData: [],
      wsTreeFilter: '',
      wsTreeExpanded: [],
      wsTreeTicked: [],
      isFromRunShow: false,
      isParamTreeShow: false,
      paramTreeCount: 0,
      runCurrent: Mdf.emptyRunText(), // selected run
      isRunSuccess: false,
      isFromWorksetShow: false,
      worksetNameFrom: '',
      worksetFrom: Mdf.emptyWorksetText(), // workset to sopy from
      isNotEmptyFrom: false,
      isReadonlyFrom: false,
      worksetInfoTickle: false,
      worksetInfoName: '',
      runInfoDigest: '',
      groupInfoTickle: false,
      groupInfoName: '',
      paramInfoTickle: false,
      paramInfoName: '',
      worksetNameToDelete: ',',
      showDeleteWorksetTickle: false,
      isDeleteWorksetNow: false,
      loadWorksetDelete: false,
      isNewWorksetShow: false,
      isCreateWorksetNow: false,
      loadWorksetCreate: false,
      nameOfNewWorkset: '',
      newDescrNotes: [], // new workset description and notes
      showParamFromWorksetTickle: false,
      showParamFromRunTickle: false,
      showGroupFromRunTickle: false,
      showGroupFromWorkTickle: false,
      confirmMsg: '',
      sourceParams: [],
      replaceParams: [],
      isShowNoteEditor: false,
      showDeleteParameterTickle: false,
      showDeleteGroupTickle: false,
      wsMultipleCount: 0,
      isWsDeleteAll: false,
      showDeleteMultipleDialogTickle: false,
      loadWsMultipletDelete: false,
      uploadFileSelect: false,
      uploadFile: null,
      isNoDigestCheck: false,
      noteEditorLangCode: '',
      noteCurrent: ''
    }
  },

  computed: {
    isNotEmptyWorksetCurrent () { return Mdf.isNotEmptyWorksetText(this.worksetCurrent) },
    isReadonlyWorksetCurrent () { return Mdf.isNotEmptyWorksetText(this.worksetCurrent) && this.worksetCurrent.IsReadonly },
    descrWorksetCurrent () { return Mdf.descrOfTxt(this.worksetCurrent) },
    fileSelected () { return !(this.uploadFile === null) },
    isDiskOver () { return !!this?.serverConfig?.IsDiskUse && !!this.diskUseState?.DiskUse?.IsOver },
    isReadonlyServer () { return this.serverConfig?.IsReadonly ?? true },

    ...mapState(useModelStore, [
      'theModel',
      'groupParameterLeafs',
      'worksetTextList',
      'worksetTextListUpdated'
    ]),
    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl',
      serverConfig: 'config',
      diskUseState: 'diskUse'
    }),
    ...mapState(useUiStateStore, [
      'runDigestSelected',
      'worksetNameSelected',
      'paramViewWorksetUpdatedCount',
      'idCsvDownload',
      'uiLang'
    ])
  },

  watch: {
    digest () { this.doRefresh() },
    refreshTickle () { this.doRefresh() },
    worksetTextListUpdated () { this.doRefresh() },
    worksetNameSelected () {
      this.worksetCurrent = this.worksetTextByName({ ModelDigest: this.digest, Name: this.worksetNameSelected })
      if (!Mdf.isNotEmptyWorksetText(this.worksetCurrent) || this.worksetCurrent.IsReadonly) {
        this.isNewWorksetShow = false
        this.isShowNoteEditor = false
        this.isFromRunShow = false
        this.isFromWorksetShow = false
      }
      if (this.worksetNameSelected === this.worksetNameFrom) this.worksetNameFrom = ''
    }
  },

  emits: [
    'set-select',
    'new-run-select',
    'set-update-readonly',
    'set-parameter-select',
    'set-parameter-delete',
    'set-list-refresh',
    'set-list-delete',
    'upload-select',
    'download-select',
    'tab-mounted'
  ],

  methods: {
    ...mapActions(useModelStore, [
      'runTextByDigest',
      'worksetTextByName'
    ]),

    dateTimeStr (dt) { return Mdf.dtStr(dt) },
    isEdit () { return this.isFromRunShow || this.isFromWorksetShow || this.isNewWorksetShow || this.isShowNoteEditor },

    // update page view
    doRefresh () {
      this.worksetCurrent = this.worksetTextByName({ ModelDigest: this.digest, Name: this.worksetNameSelected })

      this.wsTreeData = this.makeWorksetTreeData(this.worksetTextList)
      if (this.wsTreeData?.length > 0) {
        const wsTop = this.wsTreeData[0]
        this.wsTreeExpanded = [wsTop.key]

        const tn = []
        for (const ws of wsTop.children) {
          if (this.wsTreeTicked.findIndex((wsKey) => { return wsKey === ws.key }) >= 0) {
            tn.push(ws.key)
          }
        }
        this.wsTreeTicked = tn
      }

      this.paramTreeCount = Mdf.worksetParamCount(this.worksetCurrent)

      this.worksetFrom = this.worksetTextByName({ ModelDigest: this.digest, Name: this.worksetNameFrom })
      this.isNotEmptyFrom = Mdf.isNotEmptyWorksetText(this.worksetFrom)
      this.isReadonlyFrom = Mdf.isNotEmptyWorksetText(this.worksetFrom) && this.worksetFrom.IsReadonly

      this.runCurrent = this.runTextByDigest({ ModelDigest: this.digest, RunDigest: this.runDigestSelected })
      this.isRunSuccess = Mdf.isRunSuccess(this.runCurrent)
    },
    // current workset updated
    doneWsLoad (isSuccess, name) {
      this.loadWsWait = false
      if (isSuccess && name) {
        this.refreshParamTreeTickle = !this.refreshParamTreeTickle
        this.paramTreeCount = Mdf.worksetParamCount(this.worksetCurrent)
      }
    },
    // workset to select parameters from updated
    doneWsFromLoad (isSuccess, name) {
      this.loadWsWait = false
      if (isSuccess && name) {
        this.worksetFrom = this.worksetTextByName({ ModelDigest: this.digest, Name: this.worksetNameFrom })
        this.isNotEmptyFrom = Mdf.isNotEmptyWorksetText(this.worksetFrom)
        this.isReadonlyFrom = Mdf.isNotEmptyWorksetText(this.worksetFrom) && this.worksetFrom.IsReadonly

        this.refreshParamTreeFromTickle = !this.refreshParamTreeFromTickle
      }
    },

    // click on workset: select this workset as current workset
    onWorksetLeafClick (name) {
      this.doWorksetNameSelect(name)
    },
    // select this workset as current workset
    doWorksetNameSelect (name) {
      // disable workset change during editing
      if (this.isNewWorksetShow || this.isShowNoteEditor || this.isFromRunShow || this.isFromWorksetShow) return

      if (this.worksetNameSelected !== name) this.$emit('set-select', name)
    },
    // expand or collapse all workset tree nodes
    doToogleExpandTree () {
      if (this.isTreeCollapsed) {
        this.$refs.worksetTree.expandAll()
      } else {
        this.$refs.worksetTree.collapseAll()
      }
      this.isTreeCollapsed = !this.isTreeCollapsed
    },
    // filter workset tree nodes by name (label), update date-time or description
    doWsTreeFilter (node, filter) {
      if (node.key === 'wsl-top-node') return true // always show top node: it is tree controls
      const flt = filter.toLowerCase()
      return (node.label && node.label.toLowerCase().indexOf(flt) > -1) ||
        ((node.lastTime || '') !== '' && node.lastTime.indexOf(flt) > -1) ||
        ((node.descr || '') !== '' && node.descr.toLowerCase().indexOf(flt) > -1)
    },
    // clear workset tree filter value
    resetFilter () {
      this.wsTreeFilter = ''
      this.$refs.filterInput.focus()
    },
    // parameters tree updated and leafs counted
    onParamTreeUpdated (cnt) {
      this.paramTreeCount = cnt || 0
    },

    // show workset notes dialog
    doShowWorksetNote (name) {
      this.worksetInfoName = name
      this.worksetInfoTickle = !this.worksetInfoTickle
    },
    // show parameter notes dialog for currently selected workset
    doParamNoteWorksetCurrent (name) {
      this.worksetInfoName = this.worksetNameSelected
      this.paramInfoName = name
      this.paramInfoTickle = !this.paramInfoTickle
    },
    // show parameter notes dialog for workset copy from parameters
    doParamNoteWorksetFrom (name) {
      this.worksetInfoName = this.worksetNameFrom
      this.paramInfoName = name
      this.paramInfoTickle = !this.paramInfoTickle
    },
    // show run parameter notes dialog
    doParamNoteRun (name) {
      this.runInfoDigest = this.runDigestSelected
      this.paramInfoName = name
      this.paramInfoTickle = !this.paramInfoTickle
    },
    // show group notes dialog
    doShowGroupNote (name) {
      this.groupInfoName = name
      this.groupInfoTickle = !this.groupInfoTickle
    },

    // new model run using current workset name: open model run tab
    onNewRunClick (name) {
      this.$emit('new-run-select', (this.worksetNameSelected !== name ? name : ''))
    },
    // toggle current workset readonly status: pass event from child up to the next level
    onWorksetReadonlyToggle () {
      this.$emit('set-update-readonly', this.digest, this.worksetNameSelected, !this.worksetCurrent.IsReadonly)
    },
    // return true if any ticked workset is has unsaved parameter
    isUnsavedTicked () {
      if (!this.wsTreeTicked?.length) return false // no selection, nothing is ticked

      const wsTop = this.wsTreeData[0]
      for (const ws of wsTop.children) {
        if (!ws.label || ws.children.length > 0) continue // it is not a workset

        if (this.wsTreeTicked.findIndex((wsKey) => { return wsKey === ws.key }) >= 0) { // workset is ticked
          if (ws.isReadonly) continue // workset is readonly

          // if there are any edited and unsaved parameters for this workset
          const n = this.paramViewWorksetUpdatedCount({ digest: this.digest, worksetName: ws.label })
          if (n > 0) return true // unable to delete because workset have unsaved parameter(s)
        }
      }
      return false // all ticked worksets are unlocked for delete
    },

    // show or hide parameters tree
    onToogleShowParamTree () {
      this.isParamTreeShow = !this.isParamTreeShow
    },
    // click on parameter: open current workset parameter values tab
    onWorksetParamClick (name) {
      this.$emit('set-parameter-select', name)
    },

    // start create new workset
    onNewWorksetClick () {
      this.resetNewWorkset()
      this.isNewWorksetShow = true
      this.isParamTreeShow = false
    },
    // cancel creation of new workset
    onNewWorksetCancel () {
      this.resetNewWorkset()
      this.isNewWorksetShow = false
    },
    // send request to create new workset
    onNewWorksetSave (dgst, name, txt) {
      this.nameOfNewWorkset = name || ''
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
      this.newDescrNotes = []
    },

    // show yes/no dialog to confirm workset delete
    onDeleteWorkset (name) {
      this.worksetNameToDelete = name
      this.showDeleteWorksetTickle = !this.showDeleteWorksetTickle
    },
    // user answer yes to confirm delete model workset
    onYesDeleteWorkset (name) {
      this.worksetNameToDelete = name
      this.isDeleteWorksetNow = true
    },
    // workset delete request completed
    doneDeleteWorkset (isSuccess, dgst, name) {
      this.worksetNameToDelete = ''
      this.isDeleteWorksetNow = false
      this.loadWorksetDelete = false
      //
      // if success and if the same model then refresh workset list from the server
      if (isSuccess && dgst && name && dgst === this.digest) {
        this.$emit('set-list-refresh')
        setTimeout(() => this.$emit('set-list-delete', dgst, name), 521)
      }
    },
    // delete multiple selected worksets
    onWsMultipleDelete () {
      if (!this.wsTreeTicked?.length) return // empty selection

      this.wsMultipleCount = this.wsTreeTicked.length
      this.isWsDeleteAll = Array.isArray(this.worksetTextList) && this.wsMultipleCount >= this.worksetTextList.length
      this.showDeleteMultipleDialogTickle = !this.showDeleteMultipleDialogTickle
    },
    // user answer yes to confirm delete multiple selected model worksets
    onYesWsMultipleDelete () {
      if (!this.wsTreeData?.length || !this.wsTreeTicked?.length) {
        console.warn('Unable to delete: invalid (empty) wokset tree or ticked list', this.runTreeData?.length, this.runTreeTicked?.length)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to delete: input scenarios list is empty') })
        return
      }

      // collect list of names to delete
      const nameLst = []
      let isCur = false
      let firstName = ''
      const wsTop = this.wsTreeData[0]

      for (const ws of wsTop.children) {
        if (!ws.label || ws.children.length > 0) continue // it is not a workset

        if (this.wsTreeTicked.findIndex((wsKey) => { return wsKey === ws.key }) < 0) { // this workset is not selected
          if (firstName === '') firstName = ws.label
          continue
        }
        if (!isCur) {
          isCur = this.worksetNameSelected === ws.label
        }

        nameLst.push(ws.label)
      }

      this.doWsDeleteMultiple(nameLst) // delete worksets

      // clear selection
      // change current workset if current workset deleted
      this.wsTreeTicked = []
      if (isCur) {
        this.doWorksetNameSelect(firstName)
      }
    },

    // delete multiple worksets
    async doWsDeleteMultiple (nameLst) {
      if (!nameLst || !Array.isArray(nameLst) || !nameLst?.length) {
        console.warn('Unable to delete: invalid (or empty) list of names, length:', nameLst?.length)
        return
      }
      const nLen = nameLst.length
      this.$q.notify({ type: 'info', message: this.$t('Deleting multiple input scenarios') + ': [ ' + nLen.toString() + ' ]' })
      this.loadWsMultipletDelete = true

      // if any workset is readonly then make it read-write
      let isOk = true
      let nUpd = 0

      for (const name of nameLst) {
        if (this.worksetTextList.findIndex(wt => wt.Name === name && wt.IsReadonly) < 0) continue

        // workset is readonly: make it read-write
        isOk = false

        const u = this.omsUrl +
          '/api/model/' + encodeURIComponent(this.digest) +
          '/workset/' + encodeURIComponent(name) +
          '/readonly/false'
        try {
          await this.$axios.post(u) // ignore response on success
          isOk = true
          nUpd++
        } catch (e) {
          let em = ''
          try {
            if (e.response) em = e.response.data || ''
          } finally {}
          console.warn('Server offline or input scenario not found.', em)
          this.$q.notify({ type: 'negative', message: this.$t('Server offline or input scenario not found: ') + name })
        }
        if (!isOk) break
      }
      // exit on error and refersh workset list if required
      if (!isOk) {
        this.loadWsMultipletDelete = false
        if (nUpd > 0) this.$emit('set-list-refresh')
        return
      }

      // delete multiple worksets
      isOk = false

      const u = this.omsUrl + '/api/model/' + encodeURIComponent(this.digest) + '/delete-worksets'
      try {
        await this.$axios.post(u, nameLst) // ignore response on success
        isOk = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Error at delete worksets, length:', nLen, em)
      }
      this.loadWsMultipletDelete = false
      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to delete multiple input scenarios') + ' [ ' + nLen.toString() + ' ]' })
        return
      }

      // refresh workset list from the server
      this.$q.notify({ type: 'info', message: this.$t('Deleted: ') + ' [ ' + nLen.toString() + ' ]' })
      this.$emit('set-list-refresh')
      setTimeout(() => this.$emit('set-list-delete'), 1123)
    },

    // click on  workset download: start workset download and show download list page
    onDownloadWorkset (name) {
      // if name is empty or workset is not read-only then do not show rn download page
      if (!name) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to download input scenario, it is not a read-only') })
        return
      }
      const wt = this.worksetTextByName({ ModelDigest: this.digest, Name: name })
      if (!wt || !wt.IsReadonly) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to download input scenario, it is not a read-only') })
        return
      }

      this.startWorksetDownload(name) // start workset download and show download page on success
    },

    // start workset download
    async startWorksetDownload (name) {
      let isOk = false
      let msg = ''

      const opts = {
        Utf8BomIntoCsv: this.$q.platform.is.win,
        IdCsv: !!this.idCsvDownload
      }
      const u = this.omsUrl +
        '/api/download/model/' + encodeURIComponent(this.digest) +
        '/workset/' + encodeURIComponent((name || '')) +
        (this.uiLang !== '' ? '/lang/' + encodeURIComponent(this.uiLang) : '')
      try {
        // send download request to the server, response expected to be empty on success
        await this.$axios.post(u, opts)
        isOk = true
      } catch (e) {
        try {
          if (e.response) msg = e.response.data || ''
        } finally {}
        console.warn('Unable to download model workset', msg)
      }
      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to download input scenario') + (msg ? ('. ' + msg) : '') })
        return
      }

      this.$emit('download-select', this.digest) // download started: show download list page
      this.$q.notify({ type: 'info', message: this.$t('Input scenario download started') })
    },

    // show input scenario upload panel
    doShowFileSelect () {
      this.uploadFileSelect = true
    },
    // hides input scenario upload panel
    doCancelFileSelect () {
      this.uploadFileSelect = false
      this.uploadFile = null
    },

    // upload workset zip file
    async onUploadWorkset () {
      // check file name and notify user
      const fName = this.uploadFile?.name
      if (!fName) {
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) file name') })
        return
      }
      // check if workset with the same name already exist
      for (const wt of this.worksetTextList) {
        if (Mdf.isWorksetText(wt) && Mdf.modelName(this.theModel) + '.set.' + wt.Name + '.zip' === fName) {
          this.$q.notify({ type: 'negative', message: this.$t('Input scenario with the same name already exist: ') + wt.Name })
          return
        }
      }
      this.$q.notify({ type: 'info', message: this.$t('Uploading: ') + fName + '\u2026' })

      // make upload multipart form
      const opts = {
        NoDigestCheck: this.isNoDigestCheck
      }
      const fd = new FormData()
      fd.append('workset-upload-options', JSON.stringify(opts))
      fd.append('workset.zip', this.uploadFile, fName) // name and file name are ignored by server

      const u = this.omsUrl + '/api/upload/model/' + encodeURIComponent(this.digest) +
        '/workset' +
        (this.uiLang !== '' ? '/lang/' + encodeURIComponent(this.uiLang) : '')
      try {
        // update workset zip, drop response on success
        await this.$axios.post(u, fd)
      } catch (e) {
        let msg = ''
        try {
          if (e.response) msg = e.response.data || ''
        } finally {}
        console.warn('Unable to upload input scenario', msg, fName)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to upload input scenario: ') + fName })
        return
      }

      // notify user and close upload controls
      this.doCancelFileSelect()
      this.$q.notify({ type: 'info', message: this.$t('Uploaded: ') + fName })
      this.$q.notify({ type: 'info', message: this.$t('Import scenario: ') + fName + '\u2026' })

      // upload started: show upload list page
      this.$emit('upload-select', this.digest)
      this.$emit('set-list-refresh') // refresh workset list if upload completed fast
    },

    // delete parameter from current workset
    async doDeleteFromWorkset (wsName, paramName) {
      let isOk = false
      let msg = ''

      // validate current current workset is not read-only
      if (!wsName || !Mdf.isNotEmptyWorksetText(this.worksetCurrent) || this.worksetCurrent.IsReadonly) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to delete from input scenario, it is read-only (or undefined)') })
        return
      }
      this.$q.notify({ type: 'info', message: this.$t('Deleting: ') + paramName })
      this.loadWait = true

      const u = this.omsUrl +
        '/api/model/' + encodeURIComponent(this.digest) +
        '/workset/' + encodeURIComponent(wsName) +
        '/parameter/' + encodeURIComponent(paramName)
      try {
        // delete workset parameter, response is empty on success
        await this.$axios.delete(u)
        isOk = true
      } catch (e) {
        try {
          if (e.response) msg = e.response.data || ''
        } finally {}
        console.warn('Unable to delete parameter from input scenario', msg)
      }
      this.loadWait = false
      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to delete parameter from input scenario') + (msg ? ('. ' + msg) : '') })
        return
      }

      this.$emit('set-parameter-delete', this.digest, wsName, paramName) // parameter deleted from workset
      this.$q.notify({ type: 'info', message: this.$t('Deleted: ') + paramName })
      this.refreshWsTickle = !this.refreshWsTickle
    },

    // open description and notes editor for current workset
    onShowNoteEditor () {
      this.noteEditorLangCode = this.uiLang || this.$q.lang.getLocale() || ''
      this.noteCurrent = Mdf.noteOfTxt(this.worksetCurrent)
      this.isShowNoteEditor = true
      this.isParamTreeShow = false
    },
    // save description and notes editor content
    onSaveNoteEditor (descr, note, isUpd, lc) {
      this.doSaveNote(this.worksetNameSelected, lc, descr, note)
      this.isShowNoteEditor = false
    },
    // reset note editor to current workset values
    onCancelNoteEditor (name) {
      this.isShowNoteEditor = false
    },

    // delete parameter from current workset: ask user confirmation to delete parameter values
    onParamDelete (name) {
      this.paramInfoName = name
      this.showDeleteParameterTickle = !this.showDeleteParameterTickle
    },
    // user answer yes to delete parameter from current workset
    onYesDeleteParameter (name) {
      this.doDeleteFromWorkset(this.worksetNameSelected, name)
    },
    // delete group from current workset: ask user confirmation to delete parameter values
    onParamGroupDelete (name) {
      this.groupInfoName = name
      this.showDeleteGroupTickle = !this.showDeleteGroupTickle
    },
    // user answer yes to delete group from current workset
    onYesDeleteGroup (name) {
      this.doDeleteGroupFromWorkset(this.worksetNameSelected, name)
    },

    // show / hide copy parameter from model run section
    onFromRunShow () { this.isFromRunShow = true },
    onFromRunHide () { this.isFromRunShow = false },

    // show / hide copy parameter from other workset section
    onFromWorksetShow () { this.isFromWorksetShow = true },
    onFromWorksetHide () { this.isFromWorksetShow = false },

    // select workset to copy parameters from
    onWorksetFromClick (name) {
      this.worksetNameFrom = (this.worksetNameFrom !== name) ? name : ''
      if (this.worksetNameFrom) {
        this.refreshWsFromTickle = !this.refreshWsFromTickle
      } else {
        this.isFromWorksetShow = false
        this.isNotEmptyFrom = false
      }
    },

    // copy parameter from selected run
    onParamRunCopy (name) {
      if (!name) {
        console.warn('Unable to copy parameter from run, parameter name is empty')
        return
      }
      // if parameter already exist in workset then ask user to confirm overwrite existing parameter
      if (Mdf.isWorksetParamByName(this.worksetCurrent, name)) {
        this.paramInfoName = name
        this.showParamFromRunTickle = !this.showParamFromRunTickle
        return
      }
      // else copy parameter from run
      this.doCopyFromRun(false, this.worksetNameSelected, name, this.runDigestSelected)
    },
    // copy parameter into selected workset from copy workset
    onParamWorksetCopy (name) {
      if (!name) {
        console.warn('Unable to copy parameter from workset, parameter name is empty')
        return
      }
      // if parameter already exist in workset then ask user to confirm overwrite existing parameter
      if (Mdf.isWorksetParamByName(this.worksetCurrent, name)) {
        this.paramInfoName = name
        this.showParamFromWorksetTickle = !this.showParamFromWorksetTickle
        return
      }
      // else copy parameter from workset
      this.doCopyFromWorkset(false, this.worksetNameSelected, name, this.worksetNameFrom)
    },
    // user answer yes to replace exsiting parameter: do copy from run
    onYesParamFromRun (name) {
      this.doCopyFromRun(true, this.worksetNameSelected, name, this.runDigestSelected)
    },
    // user answer yes to replace exsiting parameter: do copy from workset
    onYesParamFromWorkset (name) {
      this.doCopyFromWorkset(true, this.worksetNameSelected, name, this.worksetNameFrom)
    },

    // copy group of parameters from selected run
    onParamGroupRunCopy (groupName) {
      if (!groupName) {
        console.warn('Unable to copy parameters group from run, group name is empty')
        return
      }
      // count run parameters from the group which are already exist in workset
      this.sourceParams = []
      this.replaceParams = []
      const gp = this.groupParameterLeafs[groupName]
      if (gp) {
        for (const pw of this.worksetCurrent.Param) {
          if (pw.Name && gp?.leafs[pw.Name]) this.replaceParams.push(pw.Name)
        }
        for (const pn in gp?.leafs) {
          this.sourceParams.push(pn)
        }
      }
      // if any of group parameters already exist in workset then ask user to confirm overwrite of existing values
      if (this.replaceParams.length > 0) {
        this.groupInfoName = groupName
        this.showGroupFromRunTickle = !this.showGroupFromRunTickle
        this.confirmMsg = this.$t('Replace {count} parameter(s) of', { count: this.replaceParams.length })
        return
      }
      // else copy group of parameters from run
      this.doCopyGroupFromRun(this.sourceParams, this.replaceParams, this.worksetNameSelected, groupName, this.runDigestSelected)
    },
    // copy group of parameters from other workset
    onParamGroupWorksetCopy (groupName) {
      if (!groupName) {
        console.warn('Unable to copy parameters group from workset, group name is empty')
        return
      }
      // count parameters from the group which are already exist in both worksets
      this.sourceParams = []
      this.replaceParams = []
      const gp = this.groupParameterLeafs[groupName]
      if (gp) {
        for (const pw of this.worksetCurrent.Param) {
          if (pw.Name && gp?.leafs[pw.Name]) {
            if (Mdf.isWorksetParamByName(this.worksetFrom, pw.Name)) this.replaceParams.push(pw.Name)
          }
        }
        for (const pw of this.worksetFrom.Param) {
          if (pw.Name && gp?.leafs[pw.Name]) this.sourceParams.push(pw.Name)
        }
      }
      // if any of group parameters already exist in workset then ask user to confirm overwrite of existing values
      if (this.replaceParams.length > 0) {
        this.groupInfoName = groupName
        this.showGroupFromWorkTickle = !this.showGroupFromWorkTickle
        this.confirmMsg = this.$t('Replace {count} parameter(s) of', { count: this.replaceParams.length })
        return
      }
      // else copy group of parameters from other workset
      this.doCopyGroupFromWorkset(this.sourceParams, this.replaceParams, this.worksetNameSelected, groupName, this.worksetNameFrom)
    },
    // user answer yes to replace exsiting parameters group: do copy group from run
    onYesGroupFromRun (groupName) {
      this.doCopyGroupFromRun(this.sourceParams, this.replaceParams, this.worksetNameSelected, groupName, this.runDigestSelected)
    },
    // user answer yes to replace exsiting parameters group: do copy group from workset
    onYesGroupFromWorkset (groupName) {
      this.doCopyGroupFromWorkset(this.sourceParams, this.replaceParams, this.worksetNameSelected, groupName, this.worksetNameFrom)
    },

    // return tree of model worksets
    makeWorksetTreeData (wLst) {
      this.wsTreeFilter = ''

      if (!Mdf.isLength(wLst)) return [] // empty workset list
      if (!Mdf.isWorksetTextList(wLst)) {
        this.$q.notify({ type: 'negative', message: this.$t('Input scenarios list is empty or invalid') })
        return [] // invalid workset list
      }

      // add worksets which are not included in any group
      const td = []
      const tdTop = {
        key: 'wsl-top-node',
        label: 'wsl-top-name',
        isReadonly: true,
        lastTime: '',
        descr: '',
        children: [],
        disabled: false
      }
      td.push(tdTop)

      for (const wt of wLst) {
        tdTop.children.push({
          key: 'wtl-' + wt.Name,
          label: wt.Name,
          isReadonly: wt.IsReadonly,
          lastTime: Mdf.dtStr(wt.UpdateDateTime),
          descr: Mdf.descrOfTxt(wt),
          children: [],
          disabled: false
        })
      }
      return td
    },

    // delete parameters group form workset
    async doDeleteGroupFromWorkset (wsName, groupName) {
      let isOk = false
      let msg = ''

      // validate current current workset is not read-only
      if (!wsName || !Mdf.isNotEmptyWorksetText(this.worksetCurrent) || this.worksetCurrent.IsReadonly) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to delete from input scenario, it is read-only (or undefined)') })
        return
      }

      // make list of workset parameters in that group
      const gp = this.groupParameterLeafs[groupName]
      if (!gp) {
        this.$q.notify({ type: 'warning', message: this.$t('Group has no parameters: ') + (groupName || '') })
        return
      }

      const pLst = []
      for (const pw of this.worksetCurrent.Param) {
        if (pw.Name && gp?.leafs[pw.Name]) pLst.push(pw.Name)
      }
      if (pLst.length <= 0) {
        this.$q.notify({ type: 'warning', message: this.$t('Input scenario group has no parameters: ') + (groupName || '') })
        return
      }

      // deleteing all parameters in that workset group
      this.$q.notify({ type: 'info', message: this.$t('Deleting group: ') + groupName })
      this.loadWait = true

      for (const pName of pLst) {
        this.$q.notify({ type: 'info', message: this.$t('Deleting: ') + pName })

        const u = this.omsUrl +
          '/api/model/' + encodeURIComponent(this.digest) +
          '/workset/' + encodeURIComponent(wsName) +
          '/parameter/' + encodeURIComponent(pName)
        try {
          // delete workset parameter, response is empty on success
          await this.$axios.delete(u)
          isOk = true
        } catch (e) {
          try {
            if (e.response) msg = e.response.data || ''
          } finally {}
          console.warn('Unable to delete parameter from input scenario', msg)
        }
        if (!isOk) {
          this.loadWait = false
          this.$q.notify({ type: 'negative', message: this.$t('Unable to delete parameter from input scenario') + (msg ? ('. ' + msg) : '') })
          return
        }
        this.$emit('set-parameter-delete', this.digest, wsName, pName) // parameter deleted from workset
      }
      this.loadWait = false

      this.$q.notify({ type: 'info', message: this.$t('Group deleted: ') + groupName })
      this.refreshWsTickle = !this.refreshWsTickle
    },

    // copy parameter from selected run
    async doCopyFromRun (isReplace, wsName, paramName, runDgst) {
      let isOk = false
      let msg = ''

      // validate current current workset is not read-only
      if (!wsName || !Mdf.isNotEmptyWorksetText(this.worksetCurrent) || this.worksetCurrent.IsReadonly) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to copy parameter into input scenario, it is read-only (or undefined)') })
        return
      }
      this.$q.notify({ type: 'info', message: this.$t('Copy: ') + paramName })
      this.loadWait = true

      const udgst = encodeURIComponent(this.digest)
      const uws = encodeURIComponent(wsName)
      const upn = encodeURIComponent(paramName)
      const urun = encodeURIComponent(runDgst)
      const u = isReplace
        ? this.omsUrl + '/api/model/' + udgst + '/workset/' + uws + '/merge/parameter/' + upn + '/from-run/' + urun
        : this.omsUrl + '/api/model/' + udgst + '/workset/' + uws + '/copy/parameter/' + upn + '/from-run/' + urun
      try {
        // copy parameter from model run, response is empty on success
        if (isReplace) {
          await this.$axios.patch(u)
        } else {
          await this.$axios.put(u)
        }
        isOk = true
      } catch (e) {
        try {
          if (e.response) msg = e.response.data || ''
        } finally {}
        console.warn('Unable to copy parameter', msg)
      }
      this.loadWait = false
      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to copy parameter') + (msg ? ('. ' + msg) : '') })
        return
      }

      this.$q.notify({ type: 'info', message: this.$t('Copy completed: ') + paramName })
      this.refreshWsTickle = !this.refreshWsTickle
    },

    // copy parameters group from selected run
    async doCopyGroupFromRun (srcNames, replaceNames, wsName, groupName, runDgst) {
      let isOk = false
      let msg = ''

      // validate current current workset is not read-only
      if (!wsName || !Mdf.isNotEmptyWorksetText(this.worksetCurrent) || this.worksetCurrent.IsReadonly) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to copy parameter into input scenario, it is read-only (or undefined)') })
        return
      }
      // check if there are any pareameters source group
      if (!srcNames || srcNames.length <= 0) {
        this.$q.notify({ type: 'warning', message: this.$t('Group has no parameters: ') + (groupName || '') })
        return
      }

      // copy all parameters from run parameters group into current workset group
      this.$q.notify({ type: 'info', message: this.$t('Copy group: ') + groupName })
      this.loadWait = true

      for (const pName of srcNames) {
        this.$q.notify({ type: 'info', message: this.$t('Copy: ') + pName })

        const isReplace = !!replaceNames && replaceNames.indexOf(pName) >= 0

        const udgst = encodeURIComponent(this.digest)
        const uws = encodeURIComponent(wsName)
        const upn = encodeURIComponent(pName)
        const urun = encodeURIComponent(runDgst)
        const u = isReplace
          ? this.omsUrl + '/api/model/' + udgst + '/workset/' + uws + '/merge/parameter/' + upn + '/from-run/' + urun
          : this.omsUrl + '/api/model/' + udgst + '/workset/' + uws + '/copy/parameter/' + upn + '/from-run/' + urun
        try {
          // copy parameter from model run, response is empty on success
          if (isReplace) {
            await this.$axios.patch(u)
          } else {
            await this.$axios.put(u)
          }
          isOk = true
        } catch (e) {
          try {
            if (e.response) msg = e.response.data || ''
          } finally {}
          console.warn('Unable to copy parameter', msg)
        }
        if (!isOk) {
          this.loadWait = false
          this.$q.notify({ type: 'negative', message: this.$t('Unable to copy parameter') + (msg ? ('. ' + msg) : '') })
          return
        }
      }
      this.loadWait = false

      this.$q.notify({ type: 'info', message: this.$t('Group copy completed: ') + groupName })
      this.refreshWsTickle = !this.refreshWsTickle
    },

    // copy parameter from other workset
    async doCopyFromWorkset (isReplace, wsName, paramName, srcWsName) {
      let isOk = false
      let msg = ''

      // validate current current workset is not read-only
      if (!wsName || !Mdf.isNotEmptyWorksetText(this.worksetCurrent) || this.worksetCurrent.IsReadonly) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to copy parameter into input scenario, it is read-only (or undefined)') })
        return
      }
      this.$q.notify({ type: 'info', message: this.$t('Copy: ') + paramName })
      this.loadWait = true

      const udgst = encodeURIComponent(this.digest)
      const uws = encodeURIComponent(wsName)
      const upn = encodeURIComponent(paramName)
      const usrc = encodeURIComponent(srcWsName)
      const u = isReplace
        ? this.omsUrl + '/api/model/' + udgst + '/workset/' + uws + '/merge/parameter/' + upn + '/from-workset/' + usrc
        : this.omsUrl + '/api/model/' + udgst + '/workset/' + uws + '/copy/parameter/' + upn + '/from-workset/' + usrc
      try {
        // copy parameter from other workset, response is empty on success
        if (isReplace) {
          await this.$axios.patch(u)
        } else {
          await this.$axios.put(u)
        }
        isOk = true
      } catch (e) {
        try {
          if (e.response) msg = e.response.data || ''
        } finally {}
        console.warn('Unable to copy parameter', msg)
      }
      this.loadWait = false
      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to copy parameter') + (msg ? ('. ' + msg) : '') })
        return
      }

      this.$q.notify({ type: 'info', message: this.$t('Copy completed: ') + paramName })
      this.refreshWsTickle = !this.refreshWsTickle
    },

    // copy group of parameters into current workset from other workset
    async doCopyGroupFromWorkset (srcNames, replaceNames, wsName, groupName, srcWsName) {
      let isOk = false
      let msg = ''

      // validate current current workset is not read-only
      if (!wsName || !Mdf.isNotEmptyWorksetText(this.worksetCurrent) || this.worksetCurrent.IsReadonly) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to copy parameter into input scenario, it is read-only (or undefined)') })
        return
      }
      // check if there are any pareameters source group
      if (!srcNames || srcNames.length <= 0) {
        this.$q.notify({ type: 'warning', message: this.$t('Group has no parameters: ') + (groupName || '') })
        return
      }

      // copy all parameters from run parameters group into current workset group
      this.$q.notify({ type: 'info', message: this.$t('Copy group: ') + groupName })
      this.loadWait = true

      for (const pName of srcNames) {
        this.$q.notify({ type: 'info', message: this.$t('Copy: ') + pName })

        const isReplace = !!replaceNames && replaceNames.indexOf(pName) >= 0

        const udgst = encodeURIComponent(this.digest)
        const uws = encodeURIComponent(wsName)
        const upn = encodeURIComponent(pName)
        const usrc = encodeURIComponent(srcWsName)
        const u = isReplace
          ? this.omsUrl + '/api/model/' + udgst + '/workset/' + uws + '/merge/parameter/' + upn + '/from-workset/' + usrc
          : this.omsUrl + '/api/model/' + udgst + '/workset/' + uws + '/copy/parameter/' + upn + '/from-workset/' + usrc
        try {
          // copy parameter from other workset, response is empty on success
          if (isReplace) {
            await this.$axios.patch(u)
          } else {
            await this.$axios.put(u)
          }
          isOk = true
        } catch (e) {
          try {
            if (e.response) msg = e.response.data || ''
          } finally {}
          console.warn('Unable to copy parameter', msg)
        }
        if (!isOk) {
          this.loadWait = false
          this.$q.notify({ type: 'negative', message: this.$t('Unable to copy parameter') + (msg ? ('. ' + msg) : '') })
          return
        }
      }
      this.loadWait = false

      this.$q.notify({ type: 'info', message: this.$t('Group copy completed: ') + groupName })
      this.refreshWsTickle = !this.refreshWsTickle
    },

    // save workset description and notes
    async doSaveNote (name, langCode, descr, note) {
      let isOk = false
      let msg = ''

      // validate current current workset is not read-only and has a language
      if (!Mdf.isNotEmptyWorksetText(this.worksetCurrent) || this.worksetCurrent.IsReadonly) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to save description and notes, input scenario is read-only (or undefined)') })
        return
      }
      if (!langCode) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to save input scenario description and notes, language is unknown') })
        return
      }
      this.loadWait = true

      const u = this.omsUrl + '/api/workset-merge'
      const wt = {
        ModelDigest: this.digest,
        Name: name,
        Txt: [{
          LangCode: langCode,
          Descr: descr || '',
          Note: note || ''
        }]
      }
      const fd = new FormData()
      fd.append('workset', JSON.stringify(wt))
      try {
        // send run description and notes, drop response on success
        await this.$axios.patch(u, fd)
        isOk = true
      } catch (e) {
        try {
          if (e.response) msg = e.response.data || ''
        } finally {}
        console.warn('Unable to save input scenario description and notes', msg)
      }
      this.loadWait = false
      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to save input scenario description and notes') + (msg ? ('. ' + msg) : '') })
        return
      }

      this.$q.notify({ type: 'info', message: this.$t('Input scenario description and notes saved') + '. ' + this.$t('Language: ') + langCode })
      this.refreshWsTickle = !this.refreshWsTickle
    }
  },

  mounted () {
    this.doRefresh()
    this.$emit('tab-mounted', 'set-list', { digest: this.digest })
  }
}
