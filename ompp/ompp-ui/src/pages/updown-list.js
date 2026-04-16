import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'
import ModelBar from 'components/ModelBar.vue'
import RunBar from 'components/RunBar.vue'
import WorksetBar from 'components/WorksetBar.vue'
import ModelInfoDialog from 'components/ModelInfoDialog.vue'
import RunInfoDialog from 'components/RunInfoDialog.vue'
import WorksetInfoDialog from 'components/WorksetInfoDialog.vue'
import RefreshWorksetList from 'components/RefreshWorksetList.vue'
import DeleteConfirmDialog from 'components/DeleteConfirmDialog.vue'
import { openURL } from 'quasar'

/* eslint-disable no-multi-spaces */
const LOG_REFRESH_TIME = 1000               // msec, log files refresh interval
const MAX_LOG_SEND_COUNT = 4                // max request to send without response
const MAX_LOG_NO_DATA_COUNT = 5             // pause log refresh if no new data or empty response exceed this count (5 = 5 seconds)
const MAX_LOG_RECENT_COUNT = 149            // pause log refresh if "recent" progess and no new data exceed this count
const MAX_LOG_WAIT_PROGRESS_COUNT = 20 * 60 // "recent" progress threshold (20 * 60 = 20 minutes)
/* eslint-enable no-multi-spaces */

export default {
  name: 'UpDownList',
  components: { ModelBar, RunBar, WorksetBar, ModelInfoDialog, RunInfoDialog, WorksetInfoDialog, RefreshWorksetList, DeleteConfirmDialog },

  props: {
    digest: { type: String, default: '' },
    toUpDownSection: { type: String, default: 'down' },
    refreshTickle: { type: Boolean, default: false }
  },

  data () {
    return {
      upDownStatusLst: [],
      downStatusLst: [],
      upStatusLst: [],
      downloadExpand: false,
      uploadExpand: false,
      filesExpand: false,
      folderSelected: '',
      upDownSelected: '',
      totalDownCount: 0,
      readyDownCount: 0,
      progressDownCount: 0,
      errorDownCount: 0,
      totalUpCount: 0,
      readyUpCount: 0,
      progressUpCount: 0,
      errorUpCount: 0,
      loadWait: false,
      loadWsListWait: false,
      refreshWsListTickle: false,
      loadConfig: false,
      isDiskOver: false,
      loadDiskUse: false,
      isLogRefreshPaused: false,
      lastLogDt: 0,
      logRefreshInt: '',
      logRefreshCount: 0,
      logSendCount: 0,
      logNoDataCount: 0,
      logAllKey: '',
      refreshFolderTreeTickle: false,
      modelInfoTickle: false,
      modelInfoDigest: '',
      runInfoTickle: false,
      runInfoDigest: '',
      worksetInfoTickle: false,
      worksetInfoName: '',
      showUpDownDeleteDialogTickle: false,
      folderToDelete: '',
      upDownToDelete: '',
      dialogTitleToDelete: '',
      folderTreeData: [],
      isAnyFolderDir: false,
      folderTreeFilter: '',
      isFolderTreeExpanded: false,
      fastDownload: 'yes',
      isNoDigestCheck: false,
      isMicroDownload: false,
      isIdCsvDownload: false,
      wsUploadFile: null,
      runUploadFile: null,
      filesTreeData: [],
      filesTreeFilter: '',
      isFilesTreeExpanded: false,
      filesTreeExpanded: [],
      showFilesDeleteDialogTickle: false,
      nameToDelete: '',
      pathToDelete: '',
      newFolderParentName: '',
      newFolderParentPath: '',
      newFolderName: '',
      showFolderPrompt: false,
      uploadParentName: '',
      uploadParentPath: '',
      showUploadFillePrompt: false,
      uploadFile: null
    }
  },

  computed: {
    lastLogTimeStamp () {
      return this.lastLogDt ? Mdf.dtToTimeStamp(new Date(this.lastLogDt)) : ''
    },
    modelNameVer () {
      return Mdf.modelNameVer(this.theModel)
    },
    isDownloadEnabled () { return this.serverConfig.AllowDownload },
    isUploadEnabled () { return this.serverConfig.AllowUpload && this.digest },
    wsFileSelected () { return !(this.wsUploadFile === null) && (this.wsUploadFile?.name || '') !== '' },
    runFileSelected () { return !(this.runUploadFile === null) && (this.runUploadFile?.name || '') !== '' },
    uploadFileSelected () { return !(this.uploadFile === null) && (this.uploadFile?.name || '') !== '' },

    ...mapState(useModelStore, [
      'theModel',
      'runTextList',
      'worksetTextList'
    ]),
    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl',
      serverConfig: 'config',
      diskUseState: 'diskUse'
    }),
    ...mapState(useUiStateStore, [
      'noAccDownload',
      'noMicrodataDownload',
      'idCsvDownload'
    ])
  },

  watch: {
    refreshTickle () { this.initView() },
    digest () { this.initView() },
    fastDownload (val) {
      this.dispatchNoAccDownload(val === 'yes')
      if (val === 'yes' || !this.serverConfig.AllowMicrodata) {
        this.isMicroDownload = false
        this.dispatchNoMicrodataDownload(!this.isMicroDownload)
      }
    },
    isMicroDownload () {
      this.dispatchNoMicrodataDownload(!this.isMicroDownload)
    },
    isIdCsvDownload (isIdCsv) {
      this.dispatchIdCsvDownload(isIdCsv)
    }
  },

  emits: ['upload-select', 'tab-mounted'],

  methods: {
    ...mapActions(useModelStore, [
      'runTextByDigest'
    ]),
    ...mapActions(useServerStateStore, [
      'dispatchServerConfig',
      'dispatchDiskUse'
    ]),
    ...mapActions(useUiStateStore, [
      'dispatchNoAccDownload',
      'dispatchNoMicrodataDownload',
      'dispatchIdCsvDownload'
    ]),

    isReady (status) { return status === 'ready' },
    isProgress (status) { return status === 'progress' },
    isError (status) { return status === 'error' },
    isModelKind (kind) { return kind === 'model' },
    isRunKind (kind) { return kind === 'run' },
    isWorksetKind (kind) { return kind === 'workset' },
    isDeleteKind (kind) { return kind === 'delete' },
    isUploadKind (kind) { return kind === 'upload' },
    isAnyDownloadKind (kind) { return kind === 'model' || kind === 'run' || kind === 'workset' },
    isUnkownKind (kind) { return kind !== 'model' && kind !== 'run' && kind !== 'workset' && kind !== 'delete' && kind !== 'upload' },
    fileTimeStamp (t) { return Mdf.modTsToTimeStamp(t) },
    fileSizeStr (size) {
      const fs = Mdf.fileSizeParts(size)
      return fs.val + ' ' + this.$t(fs.name)
    },

    // show model notes dialog
    doShowModelNote (modelDgst) {
      this.modelInfoDigest = modelDgst
      this.modelInfoTickle = !this.modelInfoTickle
    },
    // show run notes dialog
    doShowRunNote (modelDgst, runDgst) {
      this.modelInfoDigest = modelDgst
      this.runInfoDigest = runDgst
      this.runInfoTickle = !this.runInfoTickle
    },
    // show current workset notes dialog
    doShowWorksetNote (modelDgst, name) {
      this.modelInfoDigest = modelDgst
      this.worksetInfoName = name
      this.worksetInfoTickle = !this.worksetInfoTickle
    },

    // refersh log files list
    onLogRefresh () {
      if (this.isLogRefreshPaused) return
      //
      if (this.logSendCount++ < MAX_LOG_SEND_COUNT) {
        this.doLogListRefresh()

        // refresh files list in selected folder
        if ((this.folderSelected || '') !== '' && (this.upDownSelected || '') !== '') {
          const st = this.statusByUpDownFolder(this.folderSelected, this.upDownSelected)
          if (this.isProgress(st)) this.doFolderFilesRefresh(this.upDownSelected, this.folderSelected)
        }
      }
      this.logRefreshCount++
    },
    // return status by folder
    statusByUpDownFolder (folder, upDown) {
      if (upDown === 'up') {
        const n = this.upStatusLst.findIndex((uds) => uds.Folder === folder)
        return n >= 0 ? this.upStatusLst[n].Status : ''
      }
      // else find in download
      const n = this.downStatusLst.findIndex((uds) => uds.Folder === folder)
      return n >= 0 ? this.downStatusLst[n].Status : ''
    },

    // show or hide folder tree
    onFolderTreeClick (upDown, folder) {
      if (!folder || (upDown !== 'up' && upDown !== 'down')) return

      if (folder === this.folderSelected && upDown === this.upDownSelected) { // collapse: this folder is now open
        this.folderSelected = ''
        this.upDownSelected = ''
      } else {
        this.folderSelected = folder
        this.upDownSelected = upDown
        this.doFolderFilesRefresh(upDown, this.folderSelected)
        this.isFolderTreeExpanded = false
      }
    },
    // expand or collapse all folder tree nodes
    doToogleExpandFolderTree () {
      if (this.isFolderTreeExpanded) {
        this.$refs.folderTree[0].collapseAll()
      } else {
        this.$refs.folderTree[0].expandAll()
      }
      this.isFolderTreeExpanded = !this.isFolderTreeExpanded
    },
    // filter folder tree nodes by name (label) or description
    doFolderTreeFilter (node, filter) {
      const flt = filter.toLowerCase()
      return (node.label && node.label.toLowerCase().indexOf(flt) > -1) ||
        ((node.descr || '') !== '' && node.descr.toLowerCase().indexOf(flt) > -1)
    },
    // clear folder tree filter value
    resetFolderFilter () {
      this.folderTreeFilter = ''
      this.$refs.folderTreeFilterInput[0].focus()
    },
    // open file download url
    onFolderLeafClick (name, path) {
      openURL(path)
    },

    // delete download or upload results by folder name
    onUpDownDeleteClick (upDown, folder) {
      this.folderToDelete = folder
      this.upDownToDelete = upDown
      this.dialogTitleToDelete = (upDown === 'up' ? this.$t('Delete upload files') : this.$t('Delete download files')) + '?'
      this.showUpDownDeleteDialogTickle = !this.showUpDownDeleteDialogTickle
    },
    // user answer Yes to delete download or upload results by folder name
    onYesUpDownDelete (folder, itemId, upDown) {
      this.doDeleteUpDown(upDown, folder)
      this.logRefreshPauseToggle()
      this.isLogRefreshPaused = false
    },

    // filter user files tree nodes by name (label) or description
    doFilesTreeFilter (node, filter) {
      const flt = filter.toLowerCase()
      return (node.label && node.label.toLowerCase().indexOf(flt) > -1) ||
        ((node.descr || '') !== '' && node.descr.toLowerCase().indexOf(flt) > -1)
    },
    // clear user files tree filter value
    resetFilesFilter () {
      this.filesTreeFilter = ''
      this.$refs.filesTreeFilterInput.focus()
    },
    // expand or collapse all user files tree nodes
    doToogleExpandFilesTree () {
      if (this.isFilesTreeExpanded) {
        this.$refs.filesTree.collapseAll()
      } else {
        this.$refs.filesTree.expandAll()
      }
      this.isFilesTreeExpanded = !this.isFilesTreeExpanded
    },
    // open file download url
    onFilesDownloadClick (name, link) {
      openURL('/files/' + link)
    },

    // return true if if file or folder form user files can be deleted
    isFilesDeleteEnabled (path) {
      if (!this.serverConfig.AllowFiles || !this.serverConfig.AllowUpload) return false
      return !!path && typeof path === typeof 'string' && path !== '' && path !== '.' && path !== '..' && path !== '/'
    },
    // return true if it is a special path: download or upload folder
    isUpDownPath (path) {
      return path === 'download' || path === 'upload'
    },
    // delete file or folder from user files
    onFilesDeleteClick (name, isGrp, path) {
      this.nameToDelete = name
      this.pathToDelete = path
      this.upDownToDelete = ''
      this.dialogTitleToDelete = (!isGrp ? this.$t('Delete file') : this.$t('Delete folder')) + '?'
      this.showFilesDeleteDialogTickle = !this.showFilesDeleteDialogTickle
    },
    // delete all downloads or upload files
    onAllUpDownDeleteClick (path) {
      if (path !== 'download' && path !== 'upload') {
        console.warn('Invalid path at delete all downloads or upload files:', path)
        return
      }
      if (path === 'download') {
        this.upDownToDelete = 'down'
        this.dialogTitleToDelete = this.$t('Delete all download files') + '?'
        this.nameToDelete = this.$t('Delete all files of all models in download folder')
      } else {
        this.upDownToDelete = 'up'
        this.dialogTitleToDelete = this.$t('Delete all upload files') + '?'
        this.nameToDelete = this.$t('Delete all files of all models in upload folder')
      }
      this.showFilesDeleteDialogTickle = !this.showFilesDeleteDialogTickle
    },
    // user answer Yes to delete file of folder from user files
    onYesFilesDelete (name, path, kind) {
      if (kind) {
        this.doAllDeleteUpDown(kind)
        return
      }
      this.doDeleteFiles(name, path)
    },

    // create new folder in user files directory
    onFilesCreateFolderClick (parentName, parentPath) {
      this.newFolderParentName = parentName || ''
      this.newFolderParentPath = parentPath || ''
      this.newFolderName = ''
      this.showFolderPrompt = true
    },
    // user entered new folder name
    onYesNewFolderClick () {
      this.showFolderPrompt = false

      if (!this.serverConfig.AllowFiles || !this.serverConfig.AllowUpload) return
      if ((this.newFolderName || '') === '') return

      // cleanup and validate new folder name
      let name = Mdf.cleanPathInput(this.newFolderName)

      if (this.newFolderName === '.' || this.newFolderName === '..' || this.newFolderName === '/' ||
        this.newFolderName === 'download' || this.newFolderName === 'upload' ||
        this.newFolderName !== name.trim()) {
        console.warn('Invalid name of new folder', this.newFolderName)
        this.$q.notify({ type: 'negative', message: this.$t('Invalid name of new folder: ') + this.newFolderName })
        return
      }

      // create new folder under the parent and refersh user files tree
      if (this.newFolderParentPath !== '' && this.newFolderParentPath !== '.' && this.newFolderParentPath !== '..' && this.newFolderParentPath !== '/') {
        name = this.newFolderParentPath + '/' + name
      }
      this.doCreateNewFolder(name)
    },
    // upload file into user files folder
    onUploadFileClick (parentName, parentPath) {
      this.uploadParentName = parentName || ''
      this.uploadParentPath = parentPath || ''
      this.uploadFile = null
      this.showUploadFillePrompt = true
    },
    // user selected file for upload into user files directory
    onYesUploadFileClick () {
      this.showUploadFillePrompt = false

      if (!this.serverConfig.AllowFiles || !this.serverConfig.AllowUpload) return
      if (!this.uploadFileSelected) return

      // create new folder under the parent and refersh user files tree
      const pp = (this.uploadParentPath !== '' && this.uploadParentPath !== '.' && this.uploadParentPath !== '..' && this.uploadParentPath !== '/') ? this.uploadParentPath : ''

      this.doUploadFile(pp)
    },

    // update page view
    initView () {
      // refersh server config and disk usage
      this.doConfigRefresh()
      this.doGetDiskUse()

      if (!this.serverConfig.AllowDownload && !this.serverConfig.AllowUpload) {
        this.$q.notify({ type: 'negative', message: this.$t('Downloads and uploads are not allowed') })
        this.downloadExpand = false
        this.uploadExpand = false
        return
      }
      this.uploadExpand = this.serverConfig.AllowUpload && (this.toUpDownSection === 'up' || !this.serverConfig.AllowDownload)
      this.downloadExpand = !this.uploadExpand

      this.upDownStatusLst = []
      this.downStatusLst = []
      this.upStatusLst = []
      this.folderSelected = ''
      this.upDownSelected = ''
      this.folderTreeData = []
      this.isAnyFolderDir = false
      this.folderTreeFilter = ''
      this.fastDownload = this.noAccDownload ? 'yes' : 'no'
      this.isMicroDownload = !this.noAccDownload && !this.noMicrodataDownload && !!this.serverConfig.AllowMicrodata
      this.isIdCsvDownload = this.serverConfig.AllowDownload && this.idCsvDownload
      this.filesTreeData = []
      this.filesTreeFilter = ''
      this.filesTreeExpanded = []

      this.stopLogRefresh()
      this.startLogRefresh()
      this.doUserFilesRefresh()
    },

    // pause on/off log files refresh
    logRefreshPauseToggle () {
      this.logRefreshCount = 0
      this.logSendCount = 0
      this.logNoDataCount = 0
      this.isLogRefreshPaused = !this.isLogRefreshPaused
    },
    startLogRefresh () {
      this.isLogRefreshPaused = false
      this.logRefreshCount = 0
      this.logSendCount = 0
      this.logNoDataCount = 0
      this.lastLogDt = Date.now() - (LOG_REFRESH_TIME + 2)
      this.logRefreshInt = setInterval(this.onLogRefresh, LOG_REFRESH_TIME)
    },
    stopLogRefresh () {
      this.logRefreshCount = 0
      clearInterval(this.logRefreshInt)
    },

    // retrieve list of download and upload log files by model digest
    async doLogListRefresh () {
      this.logSendCount = 0
      const now = Date.now()
      if (now - this.lastLogDt < LOG_REFRESH_TIME) return // protect from timeouts storm
      this.lastLogDt = now

      this.loadWait = true
      let isOk = false
      let dR = []
      let uR = []

      if (this.serverConfig.AllowDownload) { // retrieve download status
        const u = this.omsUrl +
          ((this.digest && this.digest !== Mdf.allModelsDownloadLog)
            ? '/api/download/log/model/' + encodeURIComponent(this.digest)
            : '/api/download/log-all')
        try {
          const response = await this.$axios.get(u)
          dR = response.data
          isOk = true
        } catch (e) {
          let em = ''
          try {
            if (e.response) em = e.response.data || ''
          } finally {}
          console.warn('Server offline or download log files retrieve failed.', em)
        }
      }
      if (this.serverConfig.AllowUpload) { // retrieve upload status
        const u = this.omsUrl +
          ((this.digest && this.digest !== Mdf.allModelsUploadLog)
            ? '/api/upload/log/model/' + encodeURIComponent(this.digest)
            : '/api/upload/log-all')
        try {
          const response = await this.$axios.get(u)
          uR = response.data
          isOk = true
        } catch (e) {
          let em = ''
          try {
            if (e.response) em = e.response.data || ''
          } finally {}
          console.warn('Server offline or upload log files retrieve failed.', em)
        }
      }
      const udLst = [].concat(dR, uR)

      this.loadWait = false

      if (!isOk || !udLst || !Array.isArray(udLst) || !Mdf.isUpDownLogList(udLst)) {
        if (this.logNoDataCount++ > MAX_LOG_NO_DATA_COUNT) this.isLogRefreshPaused = true // pause refresh on errors
        return
      }
      if (udLst.length <= 0) {
        if (this.logNoDataCount++ > MAX_LOG_NO_DATA_COUNT) this.isLogRefreshPaused = true // pause refresh if no log files exist
      }

      // check if any changes in log files and update status counts
      let logKey = ''
      let nDownReady = 0
      let nDownProgress = 0
      let nDownError = 0
      let nUpReady = 0
      let nUpProgress = 0
      let nUpError = 0
      let maxTime = 0

      for (let k = 0; k < udLst.length; k++) {
        logKey = logKey + '|' + udLst[k].LogFileName + ':' + udLst[k].LogModTime.toString() + ':' + udLst[k].FolderModTime.toString() + ':' + udLst[k].ZipModTime.toString()

        switch (udLst[k].Status) {
          case 'ready':
            if (this.isAnyDownloadKind(udLst[k].Kind)) nDownReady++
            if (this.isUploadKind(udLst[k].Kind)) nUpReady++
            break

          case 'progress':
            if (this.isAnyDownloadKind(udLst[k].Kind)) nDownProgress++
            if (this.isUploadKind(udLst[k].Kind)) nUpProgress++

            if (udLst[k].LogModTime > maxTime) maxTime = udLst[k].LogModTime
            if (udLst[k].FolderModTime > maxTime) maxTime = udLst[k].FolderModTime
            if (udLst[k].ZipModTime > maxTime) maxTime = udLst[k].ZipModTime
            break

          case 'error':
            if (this.isAnyDownloadKind(udLst[k].Kind)) nDownError++
            if (this.isUploadKind(udLst[k].Kind)) nUpError++
            break
        }
      }

      // pause refresh if no changes in log files, folders and zip files
      // wait longer if there is a "recent" progress
      const isRecent = (nDownProgress > 0 || nUpProgress > 0) && (Date.now() - maxTime < 1000 * MAX_LOG_WAIT_PROGRESS_COUNT)

      if (this.logAllKey === logKey) {
        this.isLogRefreshPaused = this.logNoDataCount++ > (!isRecent ? MAX_LOG_NO_DATA_COUNT : MAX_LOG_RECENT_COUNT)
      } else {
        this.logNoDataCount = 0 // new data found
      }

      // notify on completed downloads or uploads and copy log file show / hide status
      for (let k = 0; k < udLst.length; k++) {
        udLst[k].isShowLog = false
        const n = this.upDownStatusLst.findIndex((st) => st.Folder === udLst[k].Folder && st.Kind === udLst[k].Kind)
        if (n < 0) continue

        if (this.isProgress(this.upDownStatusLst[n].Status) && (this.isAnyDownloadKind(udLst[k].Kind))) {
          if (this.isReady(udLst[k].Status)) this.$q.notify({ type: 'info', message: this.$t('Download completed') + (udLst[k].Folder ? (' ' + udLst[k].Folder) : '') })
          if (this.isError(udLst[k].Status)) this.$q.notify({ type: 'negative', message: this.$t('Download error') + (udLst[k].Folder ? (' ' + udLst[k].Folder) : '') })
        }
        if (this.isProgress(this.upDownStatusLst[n].Status) && this.isUploadKind(udLst[k].Kind)) {
          if (this.isReady(udLst[k].Status)) this.$q.notify({ type: 'info', message: this.$t('Upload completed') + (udLst[k].Folder ? (' ' + udLst[k].Folder) : '') })
          if (this.isError(udLst[k].Status)) this.$q.notify({ type: 'negative', message: this.$t('Upload error') + (udLst[k].Folder ? (' ' + udLst[k].Folder) : '') })
        }
        udLst[k].isShowLog = this.upDownStatusLst[n].isShowLog
      }

      // refresh workset list on upload success
      if (nUpReady !== this.readyUpCount) {
        this.refreshWsListTickle = !this.refreshWsListTickle
      }

      // replace download and upload status list
      const dL = []
      const uL = []

      for (const st of udLst) {
        if (this.isAnyDownloadKind(st.Kind)) {
          dL.push(st)
        }
        if (this.isUploadKind(st.Kind)) {
          uL.push(st)
        }
      }
      this.upDownStatusLst = udLst
      this.downStatusLst = dL
      this.upStatusLst = uL

      this.logAllKey = logKey
      this.readyDownCount = nDownReady
      this.progressDownCount = nDownProgress
      this.errorDownCount = nDownError
      this.readyUpCount = nUpReady
      this.progressUpCount = nUpProgress
      this.errorUpCount = nUpError
      this.totalDownCount = this.downStatusLst.length
      this.totalUpCount = this.upStatusLst.length
    },

    // retrieve list of files in download or upload folder
    async doFolderFilesRefresh (upDown, folder) {
      if (!folder || !upDown) {
        return // exit on empty folder
      }
      this.loadWait = true
      let isOk = false
      let fLst = []

      const u = this.omsUrl + '/api/' + (upDown === 'up' ? 'upload' : 'download') + '/file-tree/' + encodeURIComponent(folder || '')
      try {
        const response = await this.$axios.get(u)
        fLst = response.data
        isOk = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or file tree retrieve failed.', em)
      }
      this.loadWait = false

      if (!isOk || !fLst || !Array.isArray(fLst) || fLst.length <= 0 || !Mdf.isFilePathTree(fLst)) {
        return
      }

      // update folder files tree
      const td = Mdf.makeFileTree(fLst, 'fi')
      this.isAnyFolderDir = td.isAnyDir
      this.folderTreeData = Object.freeze(td.tree)
    },

    // retrieve list of files in download or upload folder
    async doUserFilesRefresh () {
      if (!this.serverConfig.AllowFiles) return // user files disabled

      this.loadWait = true
      let isOk = false
      let fLst = []

      const u = this.omsUrl + '/api/files/file-tree/_/path'
      try {
        const response = await this.$axios.get(u)
        fLst = response.data
        isOk = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or file tree retrieve failed.', em)
      }

      if (!isOk || !fLst || !Array.isArray(fLst) || fLst.length <= 0 || !Mdf.isFilePathTree(fLst)) {
        this.loadWait = false
        return
      }

      // update user files tree and expand top level groups
      const td = Mdf.makeFileTree(fLst, 'fi')
      this.isAnyFolderDir = td.isAnyDir
      this.filesTreeData = Object.freeze(td.tree)

      this.filesTreeExpanded = []
      for (const f of this.filesTreeData) {
        if (f.isGroup) this.filesTreeExpanded.push(f.key)
      }
      this.loadWait = false
    },

    // upload model run zip file
    async onUploadRun () {
      // check file name and notify user
      const fName = this.runUploadFile?.name
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
      this.$q.notify({ type: 'info', message: this.$t('Uploading: ') + fName + '\u2026' })

      // make upload multipart form
      const fd = new FormData()
      fd.append('run.zip', this.runUploadFile, fName) // name and file name are ignored by server

      const u = this.omsUrl + '/api/upload/model/' + encodeURIComponent(this.digest) + '/run'
      try {
        // update model run zip, drop response on success
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

      // notify user and clean upload file name
      this.runUploadFile = null
      this.$q.notify({ type: 'info', message: this.$t('Uploaded: ') + fName })
      this.$q.notify({ type: 'info', message: this.$t('Import model run: ') + fName + '\u2026' })

      // upload started: show upload list page
      this.$emit('upload-select', this.digest)

      // refresh list of uploads
      this.logRefreshPauseToggle()
      this.isLogRefreshPaused = false
      this.uploadExpand = true
      this.downloadExpand = false
    },

    // upload workset zip file
    async onUploadWorkset () {
      // check file name and notify user
      const fName = this.wsUploadFile?.name
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
      fd.append('workset.zip', this.wsUploadFile, fName) // name and file name are ignored by server

      const u = this.omsUrl + '/api/upload/model/' + encodeURIComponent(this.digest) + '/workset'
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

      // notify user and clean upload file name
      this.wsUploadFile = null
      this.$q.notify({ type: 'info', message: this.$t('Uploaded: ') + fName })
      this.$q.notify({ type: 'info', message: this.$t('Import scenario: ') + fName + '\u2026' })

      // upload started: show upload list page
      this.$emit('upload-select', this.digest)

      // refresh list of uploads
      this.logRefreshPauseToggle()
      this.isLogRefreshPaused = false
      this.uploadExpand = true
      this.downloadExpand = false
    },

    // upload file into user files directory
    async doUploadFile (parentPath) {
      // check file name and notify user
      const fName = this.uploadFile?.name
      if (!fName) {
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) file name') })
        this.uploadFile = null
        return
      }
      const fPath = ((parentPath !== '' && parentPath !== '.' && parentPath !== '..' && parentPath !== '/') ? parentPath + '/' : '') + fName

      this.$q.notify({ type: 'info', message: this.$t('Uploading: ') + fName + '\u2026' })

      // make upload multipart form
      const fd = new FormData()
      fd.append('user-files.file', this.uploadFile, fName) // name and file name are ignored by server

      const u = this.omsUrl + '/api/files/file?path=' + encodeURIComponent(fPath)
      try {
        // update user file, drop response on success
        await this.$axios.post(u, fd)
      } catch (e) {
        let msg = ''
        try {
          if (e.response) msg = e.response.data || ''
        } finally {}
        console.warn('Unable to upload file to', msg, fPath)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to upload file: ') + fName })
        this.uploadFile = null
        return
      }

      // notify user and clean upload file name
      this.uploadFile = null
      this.$q.notify({ type: 'info', message: this.$t('Uploaded: ') + fPath })

      this.$nextTick(() => { this.doUserFilesRefresh() })
    },

    // delete download or upload by folder name
    async doDeleteUpDown (upDown, folder) {
      if (!folder || !upDown) {
        console.warn('Unable to delete: invalid (empty) folder name or upload-download direction')
        return
      }
      this.$q.notify({ type: 'info', message: this.$t('Deleting: ') + folder })

      this.loadWait = true
      let isOk = false

      const u = this.omsUrl + '/api/' + (upDown === 'up' ? 'upload' : 'download') + '/delete/' + encodeURIComponent(folder || '')
      try {
        // send delete request to the server, response expected to be empty on success
        await this.$axios.delete(u)
        isOk = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Error at delete of:', folder, em)
      }
      this.loadWait = false

      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to delete: ') + folder })
        return
      }

      // notify user and refresh files tree
      this.$q.notify({ type: 'info', message: this.$t('Deleted: ') + folder })

      this.logRefreshPauseToggle()
      this.isLogRefreshPaused = false
      this.uploadExpand = upDown === 'up'
      this.downloadExpand = upDown !== 'up'
    },

    // delete from user files folder or file
    async doDeleteFiles (name, path) {
      if (!name || !path) {
        console.warn('Unable to delete: invalid (empty) file name or path')
        return
      }
      this.$q.notify({ type: 'info', message: this.$t('Deleting: ') + name })

      this.loadWait = true
      let isOk = false

      const u = this.omsUrl + '/api/files/delete?path=' + encodeURIComponent(path)
      try {
        // send delete request to the server, response expected to be empty on success
        await this.$axios.delete(u)
        isOk = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Error at delete of:', path, em)
      }
      this.loadWait = false

      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to delete: ') + path })
        return
      }

      // notify user and refresh files tree
      this.$q.notify({ type: 'info', message: this.$t('Deleted: ') + name })
      this.doUserFilesRefresh()
    },

    // delete all download files or all upload files
    async doAllDeleteUpDown (upDown) {
      if (!upDown) {
        console.warn('Unable to delete: invalid (empty) upload-download direction')
        return
      }
      this.$q.notify({ type: 'info', message: (upDown === 'up' ? this.$t('Deleting all upload files') : this.$t('Deleting all download files')) })

      this.loadWait = true
      let isOk = false

      const u = this.omsUrl + '/api/' + (upDown === 'up' ? 'upload' : 'download') + '/delete-all'
      try {
        // send delete request to the server, response expected to be empty on success
        await this.$axios.delete(u)
        isOk = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Error at delete all files:', em)
      }
      this.loadWait = false

      if (!isOk) {
        this.$q.notify({ type: 'negative', message: (upDown === 'up' ? this.$t('Unable to delete upload files') : this.$t('Unable to delete download files')) })
        return
      }

      // notify user and refresh files tree, downloads and uploads
      this.$q.notify({ type: 'info', message: this.$t('Files deleted.') })

      this.doUserFilesRefresh()
      this.logRefreshPauseToggle()
      this.isLogRefreshPaused = false
    },

    // create new folder in user files directory
    async doCreateNewFolder (path) {
      if (!path) {
        console.warn('Unable to create folder: invalid (empty) file name or path')
        return
      }
      this.$q.notify({ type: 'info', message: this.$t('Creating: ') + path })

      this.loadWait = true
      let isOk = false

      const u = this.omsUrl + '/api/files/folder?path=' + encodeURIComponent(path)
      try {
        // send create request to the server, response expected to be empty on success
        await this.$axios.put(u)
        isOk = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Error at creating of:', path, em)
      }
      this.loadWait = false

      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to create: ') + path })
        return
      }

      // notify user and refresh files tree
      // this.$q.notify({ type: 'info', message: this.$t('Created: ') + path })
      this.doUserFilesRefresh()
    },

    // receive server configuration, including configuration of model catalog and run catalog
    async doConfigRefresh () {
      this.loadConfig = true

      const u = this.omsUrl + '/api/service/config'
      try {
        // send request to the server
        const response = await this.$axios.get(u)
        this.dispatchServerConfig(response.data) // update server config in store
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or configuration retrieve failed.', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or configuration retrieve failed.') })
      }
      this.loadConfig = false
    },

    // receive disk space usage from server
    async doGetDiskUse () {
      this.loadDiskUse = true
      let isOk = false

      const u = this.omsUrl + '/api/service/disk-use'
      try {
        // send request to the server
        const response = await this.$axios.get(u)
        this.dispatchDiskUse(response.data) // update disk space usage in store
        isOk = Mdf.isDiskUseState(response.data) // validate disk usage info
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or disk usage retrieve failed.', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or disk space usage retrieve failed.') })
      }
      this.loadDiskUse = false

      // update disk space usage to notify user
      if (isOk) {
        this.isDiskOver = this.diskUseState?.DiskUse?.IsOver
      } else {
        this.isDiskOver = true
        console.warn('Disk usage retrieve failed:', this.nDdiskUseErr)
        this.$q.notify({ type: 'negative', message: this.$t('Disk space usage retrieve failed') })
      }
    }
  },

  mounted () {
    this.initView()
    this.$emit('tab-mounted', 'updown-list', { digest: this.digest })
  },
  beforeUnmount () {
    this.stopLogRefresh()
  }
}
