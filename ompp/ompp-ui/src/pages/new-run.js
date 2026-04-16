import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'
import NewRunInit from 'components/NewRunInit.vue'
import RunBar from 'components/RunBar.vue'
import WorksetBar from 'components/WorksetBar.vue'
import TableList from 'components/TableList.vue'
import EntityList from 'components/EntityList.vue'
import RunInfoDialog from 'components/RunInfoDialog.vue'
import WorksetInfoDialog from 'components/WorksetInfoDialog.vue'
import TableInfoDialog from 'components/TableInfoDialog.vue'
import GroupInfoDialog from 'components/GroupInfoDialog.vue'
import EntityInfoDialog from 'components/EntityInfoDialog.vue'
import EntityAttrInfoDialog from 'components/EntityAttrInfoDialog.vue'
import EntityGroupInfoDialog from 'components/EntityGroupInfoDialog.vue'
import MarkdownEditor from 'components/MarkdownEditor.vue'
import { openURL } from 'quasar'

const ENTITY_ATTR_WARNING_LIMIT = 16 // total number of entity attributes selected to show warning: too many attributes
const ENTITY_DIM_WARNING_LIMIT = 4 // number of selected dimension attributes in any entity to show warning: too many attributes
const USER_FILES_PREFIX = 'OM_USER_FILES/'

export default {
  name: 'NewRun',
  components: {
    NewRunInit,
    RunBar,
    WorksetBar,
    TableList,
    EntityList,
    RunInfoDialog,
    WorksetInfoDialog,
    TableInfoDialog,
    GroupInfoDialog,
    EntityInfoDialog,
    EntityAttrInfoDialog,
    EntityGroupInfoDialog,
    MarkdownEditor
  },

  props: {
    digest: { type: String, default: '' },
    refreshTickle: { type: Boolean, default: false },
    runRequest: { type: Object, default: () => { return Mdf.emptyRunRequest() } }
  },

  data () {
    return {
      isInitRun: false,
      runCurrent: Mdf.emptyRunText(), // currently selected run
      worksetCurrent: Mdf.emptyWorksetText(), // currently selected workset
      useWorkset: false,
      useBaseRun: false,
      runTemplateLst: [],
      mpiTemplateLst: [],
      presetLst: [],
      profileLst: [],
      enableIni: false,
      enableIniAnyKey: false,
      iniFileLst: [],
      runOpts: {
        runName: '',
        worksetName: '',
        baseRunDigest: '',
        runDescr: {}, // run description[language code]
        subCount: 1,
        threadCount: 1,
        progressPercent: 1,
        progressStep: 0,
        workDir: '',
        useCsvDir: false,
        csvDir: '',
        csvId: false,
        iniName: '',
        useIni: false,
        iniAnyKey: false,
        profile: '',
        sparseOutput: false,
        runTmpl: '',
        mpiNpCount: 0,
        mpiUseJobs: false,
        mpiOnRoot: false,
        mpiTmpl: ''
      },
      microOpts: Mdf.emptyRunRequestMicrodata(),
      advOptsExpanded: false,
      mpiOptsExpanded: false,
      langOptsExpanded: false,
      retainTablesGroups: [], // if not empty then names of tables and groups to retain
      tablesRetain: [],
      refreshTableTreeTickle: false,
      tableCount: 0,
      tableInfoName: '',
      tableInfoTickle: false,
      groupInfoName: '',
      groupInfoTickle: false,
      entityAttrCount: 0, // total number of attributes of all entities
      entityAttrsUse: [], // entity.attribute names included selected for model run
      refreshEntityTreeTickle: false,
      attrInfoName: '',
      entityInfoName: '',
      entityGroupInfoName: '',
      attrInfoTickle: false,
      entityInfoTickle: false,
      entityGroupInfoTickle: false,
      showOptChangeDialog: false, // run options change dialog
      optChange: {
        nSub: 1,
        nProc: 0,
        nThread: 1,
        isUseJob: false,
        isOnRoot: false,
        maxThread: 1,
        maxCpu: 1
      },
      newRunNotes: {
        type: Object,
        default: () => ({})
      },
      loadWait: false,
      loadConfig: false,
      loadIni: false,
      loadCsv: false,
      loadProfile: false,
      isDiskOver: false,
      loadDiskUse: false,
      isRunOptsShow: true,
      runInfoTickle: false,
      worksetInfoTickle: false,
      txtNewRun: [], // array for run description and notes
      csvTreeData: [],
      showCsvFilesTree: false,
      csvTreeFilter: '',
      csvTreeExpanded: [],
      iniTreeData: [],
      showIniFilesTree: false,
      iniTreeFilter: ''
    }
  },

  computed: {
    isNotEmptyRunCurrent () { return Mdf.isNotEmptyRunText(this.runCurrent) },
    isNotEmptyWorksetCurrent () { return Mdf.isNotEmptyWorksetText(this.worksetCurrent) },
    isNotEmptyLanguageList () { return Mdf.isLangList(this.langList) },
    isEmptyProfileList () { return !Mdf.isLength(this.profileLst) },
    isEmptyRunTemplateList () { return !Mdf.isLength(this.runTemplateLst) },
    isIniDefault () { return this.modelByDigest(this.digest).IsIni },
    iniDefaultName () { return this.theModel.Model.Name + '.ini' },
    // return true if current can be used for model run: if workset in read-only state
    isReadonlyWorksetCurrent () {
      return this.worksetCurrent?.Name && this.worksetCurrent?.IsReadonly
    },
    // return true if current run is completed: success, error or exit
    // if run not successfully completed then it we don't know is it possible or not to use it as base run
    isCompletedRunCurrent () {
      return this.runCurrent?.RunDigest && Mdf.isRunCompleted(this.runCurrent)
    },
    isRunDeleted () { return Mdf.isNotEmptyRunText(this.runCurrent) && Mdf.isRunDeletedStatus(this.runCurrent.Status, this.runCurrent.Name) },
    isNoTables () { return !this.tablesRetain || this.tablesRetain.length <= 0 },
    isMicrodata () { return !!this.serverConfig.AllowMicrodata && Mdf.entityCount(this.theModel) > 0 },
    isNoEntityAttrsUse () { return !this.entityAttrsUse || this.entityAttrsUse.length <= 0 },

    ...mapState(useModelStore, [
      'theModel',
      'modelLanguage',
      'groupTableLeafs',
      'groupEntityLeafs',
      'topEntityAttrs',
      'langList'
    ]),
    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl',
      serverConfig: 'config',
      diskUseState: 'diskUse'
    }),
    ...mapState(useUiStateStore, [
      'runDigestSelected',
      'worksetNameSelected',
      'uiLang'
    ])
  },

  watch: {
    digest () { this.doRefresh() },
    refreshTickle () { this.doRefresh() }
  },

  emits: [
    'run-job-select',
    'run-log-select',
    'run-list-refresh',
    'tab-mounted'
  ],

  methods: {
    ...mapActions(useModelStore, [
      'modelByDigest',
      'runTextByDigest',
      'isExistInRunTextList',
      'worksetTextByName'
    ]),
    ...mapActions(useServerStateStore, [
      'dispatchServerConfig',
      'dispatchDiskUse'
    ]),

    fileTimeStamp (t) { return Mdf.modTsToTimeStamp(t) },
    fileSizeStr (size) {
      const fs = Mdf.fileSizeParts(size)
      return fs.val + ' ' + this.$t(fs.name)
    },

    // update page view
    doRefresh () {
      // refersh server config and disk usage
      this.doConfigRefresh()
      this.doGetDiskUse()
      this.doIniRefresh()
      this.doCsvRefresh()

      // use selected run digest as base run digest or previous run digest if user resubmitting the run
      // use selected workset name or previous run workset name if user resubmitting the run
      let rDgst = this.runDigestSelected
      let wsName = this.worksetNameSelected
      if (Mdf.isNotEmptyRunRequest(this.runRequest)) {
        wsName = Mdf.getRunOption(this.runRequest.Opts, 'OpenM.SetName') || wsName
        rDgst = Mdf.getRunOption(this.runRequest.Opts, 'OpenM.BaseRunDigest') || rDgst
      }
      this.runCurrent = this.runTextByDigest({ ModelDigest: this.digest, RunDigest: rDgst })
      this.worksetCurrent = this.worksetTextByName({ ModelDigest: this.digest, Name: wsName })
      this.tableCount = Mdf.tableCount(this.theModel)

      // reset run options and state
      this.isInitRun = false

      this.runOpts.runName = ''
      this.runOpts.worksetName = ''
      this.runOpts.baseRunDigest = ''
      this.useWorkset = this.isReadonlyWorksetCurrent
      this.useBaseRun = this.isUseCurrentAsBaseRun() && !this.isRunDeleted
      this.runOpts.sparseOutput = false
      this.mpiNpCount = 0
      this.runOpts.mpiUseJobs = false // this.serverConfig.IsJobControl
      this.runOpts.mpiOnRoot = false

      // get model run template list
      // append empty '' string first to allow model run without template
      // if default run template exist then select it
      this.runTemplateLst = []
      const cfgDefaultTmpl = Mdf.configEnvValue(this.serverConfig, 'OM_CFG_DEFAULT_RUN_TMPL')

      if (Mdf.isLength(this.serverConfig.RunCatalog.RunTemplates)) {
        let isFound = false

        this.runTemplateLst.push('')
        for (const p of this.serverConfig.RunCatalog.RunTemplates) {
          this.runTemplateLst.push(p)
          if (!isFound) isFound = p === cfgDefaultTmpl
        }
        this.runOpts.runTmpl = isFound ? cfgDefaultTmpl : this.runTemplateLst[0]
      }

      // get MPI run template list and select default template
      this.runOpts.mpiTmpl = ''
      this.mpiTemplateLst = this.serverConfig.RunCatalog.MpiTemplates

      if (Mdf.isLength(this.mpiTemplateLst)) {
        let isFound = false
        const dMpiTmpl = this.serverConfig.RunCatalog.DefaultMpiTemplate

        for (let k = 0; !isFound && k < this.mpiTemplateLst.length; k++) {
          isFound = this.mpiTemplateLst[k] === cfgDefaultTmpl
        }
        if (isFound) this.runOpts.mpiTmpl = cfgDefaultTmpl

        if (!isFound && dMpiTmpl) {
          for (let k = 0; !isFound && k < this.mpiTemplateLst.length; k++) {
            isFound = this.mpiTemplateLst[k] === dMpiTmpl
          }
          if (isFound) this.runOpts.mpiTmpl = dMpiTmpl
        }
        if (!isFound) this.runOpts.mpiTmpl = this.mpiTemplateLst[0]
      }

      // csv directory usage
      this.runOpts.useCsvDir = this.serverConfig.AllowFiles && ((this.runOpts.csvDir || '') !== '')
      this.csvTreeData = []
      this.csvTreeFilter = ''
      this.showCsvFilesTree = false
      this.csvTreeExpanded = []

      // check if usage of ini-file options allowed by server
      const cfgIni = Mdf.configEnvValue(this.serverConfig, 'OM_CFG_INI_ALLOW').toLowerCase()
      this.enableIni = cfgIni === 'true' || cfgIni === '1' || cfgIni === 'yes'
      this.runOpts.iniName = ''

      if (this.enableIni) {
        if (this.serverConfig.AllowFiles) {
          for (const name of this.iniFileLst) {
            if (name === USER_FILES_PREFIX + this.iniDefaultName) this.runOpts.iniName = name
          }
        }
        if (!this.runOpts.iniName) this.runOpts.iniName = this.isIniDefault ? this.iniDefaultName : ''
      }

      const cfgAnyIni = Mdf.configEnvValue(this.serverConfig, 'OM_CFG_INI_ANY_KEY').toLowerCase()
      this.enableIniAnyKey = this.enableIni && (cfgAnyIni === 'true' || cfgAnyIni === '1' || cfgAnyIni === 'yes')

      if (!this.enableIni || (this.runOpts.iniName || '') === '') this.runOpts.useIni = false
      if (!this.enableIniAnyKey) this.runOpts.iniAnyKey = false

      this.iniTreeData = []
      this.iniTreeFilter = ''
      this.showIniFilesTree = false

      // get profile list from server
      this.runOpts.profile = ''
      this.doProfileListRefresh()

      // init retain tables list from existing base run
      this.tablesRetain = []
      if (Mdf.isNotEmptyRunText(this.runCurrent)) {
        for (const t of this.runCurrent.Table) {
          if (t?.Name) this.tablesRetain.push(t?.Name)
        }
      }

      // entity microdata for that model run
      this.entityAttrCount = 0
      this.entityAttrsUse = []
      this.microOpts = Mdf.emptyRunRequestMicrodata()

      if (this.isMicrodata) {
        for (const et of this.theModel.EntityTxt) {
          this.entityAttrCount += Mdf.entityAttrCount(et)
        }

        // init microdata list from existing base run
        if (Mdf.isNotEmptyRunText(this.runCurrent)) {
          for (const e of this.runCurrent.Entity) {
            if (e?.Name && Array.isArray(e?.Attr)) {
              for (const aName of e.Attr) {
                this.entityAttrsUse.push(e.Name + '.' + aName)
              }
            }
          }
        }
      }

      // make list of model languages, description and notes for workset editor
      this.newRunNotes = {}

      this.txtNewRun = []
      if (Mdf.isLangList(this.langList)) {
        for (const lcn of this.langList) {
          this.txtNewRun.push({
            LangCode: lcn.LangCode,
            LangName: lcn.Name,
            Descr: '',
            Note: ''
          })
        }
      } else {
        if (!this.txtNewRun.length) {
          this.txtNewRun.push({
            LangCode: this.modelLanguage.LangCode,
            LangName: this.modelLanguage.Name,
            Descr: '',
            Note: ''
          })
        }
      }

      // get run options presets as array of { name, label, descr, opts{....} }
      this.presetLst = Mdf.configRunOptsPresets(this.serverConfig, this.theModel.Model.Name, this.uiLang, this.modelLanguage.LangCode)

      // if previous run request resubmitted then apply settings from run request
      // else if first preset starts with "current-model-name." then apply preset
      if (Mdf.isNotEmptyRunRequest(this.runRequest)) {
        this.applyRunRequest(this.runRequest)
      } else {
        if (Array.isArray(this.presetLst) && this.presetLst.length > 0) {
          if (this.presetLst[0].name?.startsWith(this.theModel.Model.Name + '.')) this.doPresetSelected(this.presetLst[0])
        }
      }
    },

    // use current run as base base run if:
    //   current run is compeleted and
    //   current workset not readonly
    //   or current workset not is full and current workset not based on run
    isUseCurrentAsBaseRun () {
      return this.isCompletedRunCurrent && (!this.isReadonlyWorksetCurrent || this.isPartialWorkset())
    },
    // current workset not is full and current workset not based on run
    isPartialWorkset () {
      return (Mdf.worksetParamCount(this.worksetCurrent) !== Mdf.paramCount(this.theModel)) &&
          (!this.worksetCurrent?.BaseRunDigest || !this.isExistInRunTextList({ ModelDigest: this.digest, RunDigest: this.worksetCurrent?.BaseRunDigest }))
    },
    // if use base run un-checked then user must supply full set of input parameters
    onUseBaseRunClick () {
      if (!this.useBaseRun && !this.runOpts.csvDir && this.isUseCurrentAsBaseRun()) {
        this.$q.notify({ type: 'warning', message: this.$t('Input scenario should include all parameters otherwise model run may fail') })
      }
    },
    // if return is true then display warning to the user about too many microdata attributes selected
    isTooManyEntityAttrs () {
      if (!this.entityAttrsUse || this.entityAttrsUse.length <= 0) return false
      if (this.entityAttrsUse.length > ENTITY_ATTR_WARNING_LIMIT) return true

      // check if for any entity number of selected dimension attributes exceeds the warning limit
      const eaLst = Array.from(this.entityAttrsUse)
      eaLst.sort()
      let prevEnt = ''
      let eaCount = 0

      for (const eau of eaLst) {
        const n = eau.indexOf('.')
        if (n <= 0 || n >= eau.length - 1) continue // skip: expected entity.attribute

        const eName = eau.substring(0, n)
        const aName = eau.substring(n + 1)

        if (eName !== prevEnt) {
          eaCount = 0
          prevEnt = eName
        }

        // check if this is enum based attribute or bool: it is a dimension attribute
        const a = Mdf.entityAttrTextByName(this.theModel, eName, aName)

        if (Mdf.isNotEmptyEntityAttr(a)) {
          const tTxt = Mdf.typeTextById(this.theModel, a.Attr.TypeId)
          if (!Mdf.isBuiltIn(tTxt.Type) || Mdf.isBool(tTxt.Type)) { // enum based attribute or bool: use it as dimension
            eaCount++
            if (eaCount > ENTITY_DIM_WARNING_LIMIT) { // return true // too many dimension attributes selected for that entity
              return true
            }
          }
        }
      }
      return false
    },

    // click on MPI use job control: set number of processes at least 1
    onMpiUseJobs () {
      if (!this.serverConfig.IsJobControl) {
        this.$q.notify({ type: 'warning', message: this.$t('Model run Jobs disabled on the server') })
        return
      }
      if (this.runOpts.mpiUseJobs && this.runOpts.mpiNpCount <= 0) this.runOpts.mpiNpCount = 1
    },
    // change MPI number of processes: set job control usage flag as it is on the server
    onMpiNpCount () {
      if (this.runOpts.mpiNpCount <= 0) {
        this.runOpts.mpiUseJobs = false
      } else {
        this.runOpts.mpiUseJobs = this.serverConfig.IsJobControl
      }
    },

    // filter csv files tree nodes by name (label) or description
    doCsvTreeFilter (node, filter) {
      const flt = filter.toLowerCase()
      return (node.label && node.label.toLowerCase().indexOf(flt) > -1) ||
        ((node.descr || '') !== '' && node.descr.toLowerCase().indexOf(flt) > -1)
    },
    // clear csv files tree filter value
    resetCsvFilter () {
      this.csvTreeFilter = ''
      this.$refs.csvTreeFilterInput.focus()
    },
    // select csv directory
    onCsvDirClick (path) {
      if (!this.serverConfig.AllowFiles || !path || typeof path !== typeof 'string' || path === '') return

      this.runOpts.csvDir = (!path.startsWith(USER_FILES_PREFIX) ? USER_FILES_PREFIX : '') + path
      this.runOpts.useCsvDir = true
    },
    // return true if this is a path to selected csv directory
    isCsvDirPath (path) {
      return !!path && typeof path === typeof 'string' && this.runOpts.useCsvDir && this.runOpts.csvDir === USER_FILES_PREFIX + path
    },
    // return csv directory path without user files prefix
    csvDirLabel (path) {
      if (!path || typeof path !== typeof 'string') return ''
      return path.startsWith(USER_FILES_PREFIX) ? path.substring(USER_FILES_PREFIX.length) : path
    },
    // return count of of *.csv and *.tsv files
    // it can be a compatibility issues if server is on Linux and files extension is in upper case, e.g. *.CSV
    csvFileCount (cLst) {
      if (!cLst || !Array.isArray(cLst)) return 0

      let nc = 0
      for (const c of cLst) {
        if (!c || !c.hasOwnProperty('key')) continue
        if (!c.hasOwnProperty('isGroup') || typeof c.isGroup !== typeof true || c.isGroup) continue // skip directories
        if (!c.hasOwnProperty('Path') || typeof c.Path !== typeof 'string') continue

        if (c.Path.endsWith('.csv') || c.Path.endsWith('.tsv')) nc++
      }
      return nc
    },

    // filter ini files tree nodes by name (label) or description
    doIniTreeFilter (node, filter) {
      const flt = filter.toLowerCase()
      return (node.label && node.label.toLowerCase().indexOf(flt) > -1) ||
        ((node.descr || '') !== '' && node.descr.toLowerCase().indexOf(flt) > -1)
    },
    // clear ini files tree filter value
    resetIniFilter () {
      this.iniTreeFilter = ''
      this.$refs.iniTreeFilterInput.focus()
    },
    // select ini file
    onIniLeafClick (path) {
      if (!this.enableIni || !path || path === '') return

      this.runOpts.iniName = path
      this.runOpts.useIni = true
    },
    // return ini file path without user files prefix
    iniFileLabel (path) {
      if (!path || typeof path !== typeof 'string') return ''
      return path.startsWith(USER_FILES_PREFIX) ? path.substring(USER_FILES_PREFIX.length) : path + ' (' + this.$t('Default') + ')'
    },

    // open file download url
    onFileDownloadClick (name, link) {
      openURL('/files/' + link)
    },

    // update ini files tree
    makeIniTreeData (fLst) {
      this.iniTreeFilter = ''
      this.showIniFilesTree = false
      this.iniTreeData = []

      if (!fLst || !Array.isArray(fLst) || fLst.length <= 0) return // empty file list

      // add root folder
      const fTree = []
      const fRoot = {
        key: 'fi-root-0-ini-0',
        Path: '/',
        link: '',
        label: '*.ini',
        descr: '',
        children: [],
        isGroup: true
      }
      fTree.push(fRoot)

      // add default modelName.ini, if exists
      if (this.isIniDefault) {
        fRoot.children.push({
          key: 'fi-default-1-model-ini-1',
          path: this.iniDefaultName,
          link: '',
          label: this.iniFileLabel(this.iniDefaultName),
          descr: '',
          children: [],
          isGroup: false
        })
      }

      // make href link: for each part of the path do encodeURIComponent and keep / as is
      const pathEncode = (path) => {
        if (!path || typeof path !== typeof 'string') return ''

        const ps = path.split('/')
        for (let k = 0; k < ps.length; k++) {
          ps[k] = encodeURIComponent(ps[k])
        }
        return ps.join('/')
      }

      // add path to all ini files as children of root folder, skip any other folders
      for (const fi of fLst) {
        if (!fi.Path || fi.Path === '.' || fi.Path === '..') continue
        if (fi.Path === '/' || fi.IsDir) continue // skip all folders

        fRoot.children.push({
          key: 'fi-' + fi.Path + '-' + (fi.ModTime || 0).toString(),
          path: USER_FILES_PREFIX + fi.Path,
          link: pathEncode(fi.Path),
          label: fi.Path,
          descr: this.fileTimeStamp(fi.ModTime) + (!fi.IsDir ? ' : ' + this.fileSizeStr(fi.Size) : ''),
          children: [],
          isGroup: fi.IsDir
        })
      }

      this.iniTreeData = Object.freeze(fTree)
    },

    // update csv files tree
    makeCsvTreeData (fLst) {
      this.csvTreeFilter = ''
      this.showCsvFilesTree = false
      this.csvTreeData = []
      if (!fLst || !Array.isArray(fLst) || fLst.length <= 0) return // empty file list

      // update user files tree
      const td = Mdf.makeFileTree(fLst, 'ct')

      if (td.isRootDir) {
        this.csvTreeData = Object.freeze(td.tree)
      } else {
        const fRoot = {
          key: 'ct-root-0-csv-0',
          Path: '/',
          link: '',
          label: '*.csv *.tsv',
          descr: '',
          children: td.tree,
          isGroup: true
        }
        this.csvTreeData = Object.freeze([fRoot])
      }

      // expand top level groups
      this.csvTreeExpanded = []
      for (const f of this.csvTreeData) {
        if (f.isGroup) this.csvTreeExpanded.push(f.key)
      }
    },

    // show current run info dialog
    doShowRunNote (modelDgst, runDgst) {
      if (modelDgst !== this.digest || runDgst !== this.runCurrent?.RunDigest) {
        console.warn('invlaid model digest or run digest:', modelDgst, runDgst)
        return
      }
      this.runInfoTickle = !this.runInfoTickle
    },
    // show current workset notes dialog
    doShowWorksetNote (modelDgst, name) {
      if (modelDgst !== this.digest || name !== this.worksetCurrent?.Name) {
        console.warn('invlaid model digest or workset name:', modelDgst, name)
        return
      }
      this.worksetInfoTickle = !this.worksetInfoTickle
    },
    // show output table notes dialog
    doShowTableNote (name) {
      this.tableInfoName = name
      this.tableInfoTickle = !this.tableInfoTickle
    },
    // show group notes dialog
    doShowGroupNote (name) {
      this.groupInfoName = name
      this.groupInfoTickle = !this.groupInfoTickle
    },
    // show entity attribute notes dialog
    doShowEntityAttrNote (attrName, entName) {
      this.attrInfoName = attrName
      this.entityInfoName = entName
      this.attrInfoTickle = !this.attrInfoTickle
    },
    // show entity attribute notes dialog
    doShowEntityNote (entName) {
      this.entityInfoName = entName
      this.entityInfoTickle = !this.entityInfoTickle
    },
    // show entity attributes group notes dialog
    doShowEntityGroupNote (groupName, entName) {
      this.entityInfoName = entName
      this.entityGroupInfoName = groupName
      this.entityGroupInfoTickle = !this.entityGroupInfoTickle
    },

    // click on clear filter: retain all output tables and groups
    onRetainAllTables () {
      this.tablesRetain = []
      this.tablesRetain.length = this.theModel.TableTxt.length
      let k = 0
      for (const t of this.theModel.TableTxt) {
        this.tablesRetain[k++] = t.Table.Name
      }
      this.refreshTableTreeTickle = !this.refreshTableTreeTickle
      this.$q.notify({ type: 'info', message: this.$t('Retain all output tables') })
    },

    // add output table into the retain tables list
    onTableAdd (name) {
      if (this.tablesRetain.length >= this.tableCount) return // all tables already in the retain list
      if (!name) {
        console.warn('Unable to add table into retain list, table name is empty')
        return
      }
      // add into tables retain list if not in the list already
      let isAdded = false
      if (this.tablesRetain.indexOf(name) < 0) {
        this.tablesRetain.push(name)
        isAdded = true
      }

      if (isAdded) {
        this.$q.notify({
          type: 'info',
          message: (this.tablesRetain.length < this.tableCount) ? this.$t('Retain output table: ') + ' ' + name : this.$t('Retain all output tables')
        })
        this.refreshTableTreeTickle = !this.refreshTableTreeTickle
      }
    },

    // add group of output tables into the retain tables list
    onTableGroupAdd (groupName) {
      if (!this.tablesRetain.length >= this.tableCount) return // all tables already in the retain list
      if (!groupName) {
        console.warn('Unable to add table group into retain list, group name is empty')
        return
      }
      // add each table from the group tables retain list if not in the list already
      const gt = this.groupTableLeafs[groupName]
      let isAdded = false
      if (gt) {
        for (const tn in gt?.leafs) {
          if (this.tablesRetain.indexOf(tn) < 0) {
            this.tablesRetain.push(tn)
            isAdded = true
          }
        }
      }

      if (isAdded) {
        this.$q.notify({
          type: 'info',
          message: (this.tablesRetain.length < this.tableCount) ? this.$t('Retain group of output tables: ') + groupName : this.$t('Retain all output tables')
        })
        this.refreshTableTreeTickle = !this.refreshTableTreeTickle
      }
    },

    // remove output table from the retain tables list
    onTableRemove (name) {
      if (this.tablesRetain.length <= 0) return // retain tables list already empty
      if (!name) {
        console.warn('Unable to remove table from retain list, table name is empty')
        return
      }
      this.$q.notify({ type: 'info', message: this.$t('Suppress output table: ') + name })

      this.tablesRetain = this.tablesRetain.filter(tn => tn !== name)
      this.refreshTableTreeTickle = !this.refreshTableTreeTickle
    },

    // remove group of output tables from the retain tables list
    onTableGroupRemove (groupName) {
      if (this.tablesRetain.length <= 0) return // retain tables list already empty
      if (!groupName) {
        console.warn('Unable to remove table group from retain list, group name is empty')
        return
      }
      this.$q.notify({ type: 'info', message: this.$t('Suppress group of output tables: ') + groupName })

      // remove tables group from the list
      const gt = this.groupTableLeafs[groupName]
      if (gt) {
        this.tablesRetain = this.tablesRetain.filter(tn => !gt?.leafs[tn])
        this.refreshTableTreeTickle = !this.refreshTableTreeTickle
      }
    },

    // click on clear microdata attributes list: do not use any microdata for that model run
    onClearEntityAttrs () {
      this.entityAttrsUse = []
      this.refreshEntityTreeTickle = !this.refreshEntityTreeTickle
    },

    // add entity attribute into the microdata list
    onAttrAdd (attrName, entName, isAllowHidden) {
      if (this.entityAttrsUse.length >= this.entityAttrCount) return // all attributes already in microdata list
      if (!attrName || !entName) {
        console.warn('Unable to add microdata, attribute or entity name is empty:', attrName, entName)
        return
      }

      // add into microdata list if entity.attribute not in the list already
      const name = entName + '.' + attrName
      let isAdded = false
      if (this.entityAttrsUse.indexOf(name) < 0) {
        this.entityAttrsUse.push(name)
        isAdded = true
      }

      if (!isAdded) {
        this.$q.notify({ type: 'info', message: this.$t('Nothing new to add') })
      } else {
        this.$q.notify({
          type: 'info',
          message: this.entityAttrsUse.length < this.entityAttrCount ? this.$t('Add to microdata: ') + name : this.$t('All entity attributes included into microdata')
        })
        this.refreshEntityTreeTickle = !this.refreshEntityTreeTickle
      }
    },

    // add all entity attributes into the microdata list
    onEntityAdd (entName, isAllowHidden) {
      if (this.entityAttrsUse.length >= this.entityAttrCount) return // all attributes already in microdata list
      if (!entName) {
        console.warn('Unable to add microdata, entity name is empty:', entName)
        return
      }
      const ent = Mdf.entityTextByName(this.theModel, entName)
      if (!Mdf.isNotEmptyEntityText(ent)) {
        this.$q.notify({ type: 'error', message: this.$t('Entity microdata not found or empty: ') + entName })
        console.warn('Entity microdata not found or empty:', entName)
        return
      }

      // current list of added entity.attribute names
      const nameUse = {}
      for (const nm of this.entityAttrsUse) {
        nameUse[nm] = true
      }

      // add each attribute of the entity into microdata if not already in the list
      const allLeafs = Mdf.entityGroupLeafs(this.theModel, !isAllowHidden)
      const eLfs = allLeafs?.[ent.Entity.EntityId]
      const tops = this.topEntityAttrs?.[ent.Entity.EntityId] // all top attributes, including hidden
      let isAdded = false

      for (const ea of ent.EntityAttrTxt) {
        if (!isAllowHidden && ea.Attr.IsInternal) continue // internal attributes disabled

        let isAdd = !!tops?.attrId?.[ea.Attr.AttrId] // is it entity top attribute

        if (!isAdd && !!eLfs) { // check if this attribute is visible in any of entity groups
          for (const gn in eLfs) {
            isAdd = !!eLfs[gn].leafs?.[ea.Attr.Name]
            if (isAdd) break
          }
        }
        if (!isAdd) continue // skip: attribute is not a member of the group and not a top entity attribute

        const name = entName + '.' + ea.Attr.Name
        if (!nameUse?.[name]) {
          this.entityAttrsUse.push(name)
          nameUse[name] = true
          isAdded = true
        }
      }

      if (!isAdded) {
        this.$q.notify({ type: 'info', message: this.$t('Nothing new to add') })
      } else {
        this.$q.notify({
          type: 'info',
          message: (this.entityAttrsUse.length < this.entityAttrCount) ? this.$t('Add to microdata: ') + entName : this.$t('All entity attributes included into microdata')
        })
        this.refreshEntityTreeTickle = !this.refreshEntityTreeTickle
      }
    },
    // add group of attributes into the microdata list
    onEntityGroupAdd (groupName, isAllowHidden, parts) {
      if (this.entityAttrsUse.length >= this.entityAttrCount) return // all attributes already in microdata list

      // find the group and check if it is not hidden
      if (!groupName || !parts || !parts?.entityName) {
        console.warn('Unable to add microdata, group name or event parts is empty:', groupName, parts)
        return
      }
      const ent = Mdf.entityTextByName(this.theModel, parts.entityName)
      if (!Mdf.isNotEmptyEntityText(ent)) {
        this.$q.notify({ type: 'error', message: this.$t('Entity microdata not found or empty: ') + parts.entityName })
        console.warn('Entity microdata not found or empty:', parts.entityName, parts)
        return
      }
      const gId = parts.selfId
      const gt = Mdf.entityGroupTextById(this.theModel, parts.entityId, gId)
      if (!Mdf.isEntityGroupText(gt) || (!isAllowHidden && gt.Group.IsHidden)) {
        this.$q.notify({ type: 'error', message: this.$t('Unable to find group of attributes: ') + groupName })
        console.warn('Unable to find group of attributes:', groupName, parts.entityId, gId, gt)
        return
      }
      const eLfs = Mdf.entityGroupLeafs(this.theModel, !isAllowHidden)
      const leafs = eLfs?.[parts.entityId]?.[groupName]?.leafs // group leafs: {attrName: true,....}

      // current list of added entity.attribute names
      const nameUse = {}
      for (const nm of this.entityAttrsUse) {
        nameUse[nm] = true
      }

      // add each attribute of the entity into microdata if not already in the list
      let isAdded = false

      for (const ea of ent.EntityAttrTxt) {
        if (!isAllowHidden && ea.Attr.IsInternal) continue // internal attributes disabled
        if (!leafs?.[ea.Attr.Name]) continue // skip: attribute is not a member of the group

        const name = parts.entityName + '.' + ea.Attr.Name
        if (!nameUse?.[name]) {
          this.entityAttrsUse.push(name)
          nameUse[name] = true
          isAdded = true
        }
      }

      if (!isAdded) {
        this.$q.notify({ type: 'info', message: this.$t('Nothing new to add') })
      } else {
        this.$q.notify({
          type: 'info',
          message: (this.entityAttrsUse.length < this.entityAttrCount) ? this.$t('Add to microdata: ') + groupName : this.$t('All entity attributes included into microdata')
        })
        this.refreshEntityTreeTickle = !this.refreshEntityTreeTickle
      }
    },

    // remove entity attribute from microdata list
    onAttrRemove (attrName, entName) {
      if (this.entityAttrsUse.length <= 0) return // microdata list already empty
      if (!attrName || !entName) {
        console.warn('Unable to remove from microdata list, attribute or entity name is empty:', attrName, entName)
        return
      }
      const name = entName + '.' + attrName
      this.$q.notify({ type: 'info', message: this.$t('Exclude from microdata: ') + name })

      this.entityAttrsUse = this.entityAttrsUse.filter(ean => ean !== name)
      this.refreshEntityTreeTickle = !this.refreshEntityTreeTickle
    },
    // remove all attributes of the entity from microdata list
    onEntityRemove (entName) {
      if (this.entityAttrsUse.length <= 0) return // microdata list already empty
      if (!entName) {
        console.warn('Unable to remove entity form microdata, entity name is empty:', entName)
        return
      }
      this.$q.notify({ type: 'info', message: this.$t('Exclude from microdata: ') + entName })

      // remove each entity.attribute from microdata list
      const ent = Mdf.entityTextByName(this.theModel, entName)
      if (Mdf.isNotEmptyEntityText(ent)) {
        for (const ea of ent.EntityAttrTxt) {
          const name = entName + '.' + ea.Attr.Name
          this.entityAttrsUse = this.entityAttrsUse.filter(ean => ean !== name)
        }
      }
      this.refreshEntityTreeTickle = !this.refreshEntityTreeTickle
    },
    // remove group of attributes from microdata list
    onEntityGroupRemove (groupName, parts) {
      if (this.entityAttrsUse.length <= 0) return // microdata list already empty

      // find the group
      if (!groupName || !parts || !parts?.entityName) {
        console.warn('Unable to add microdata, group name or event parts is empty:', groupName, parts)
        return
      }
      const gId = parts.selfId
      const gt = Mdf.entityGroupTextById(this.theModel, parts.entityId, gId)
      if (!Mdf.isEntityGroupText(gt)) {
        this.$q.notify({ type: 'error', message: this.$t('Unable to find group of attributes: ') + groupName })
        console.warn('Unable to find group of attributes:', groupName, parts.entityId, gId, gt)
        return
      }
      this.$q.notify({ type: 'info', message: this.$t('Exclude from microdata: ') + groupName })

      // remove each entity.attribute from microdata list
      const leafs = this.groupEntityLeafs?.[parts.entityId]?.[groupName]?.leafs // group leafs: {attrName: true,....}

      for (const ln in leafs) {
        const name = parts.entityName + '.' + ln
        this.entityAttrsUse = this.entityAttrsUse.filter(ean => ean !== name)
      }
      this.refreshEntityTreeTickle = !this.refreshEntityTreeTickle
    },

    // set default name of new model run
    onRunNameFocus (e) {
      if (typeof this.runOpts.runName !== typeof 'string' || (this.runOpts.runName || '') === '') {
        this.runOpts.runName = this.theModel.Model.Name + '_' + (this.isReadonlyWorksetCurrent ? this.worksetCurrent?.Name + '_' : '') + Mdf.dtToUnderscoreTimeStamp(new Date())
      }
    },
    // check if run name entered and cleanup input to be compatible with file name rules
    onRunNameBlur (e) {
      const { isEntered, name } = Mdf.doFileNameClean(this.runOpts.runName)
      if (isEntered && name !== this.runOpts.runName) {
        this.$q.notify({ type: 'warning', message: this.$t('Run name should not contain any of: ') + Mdf.invalidFileNameChars })
      }
      this.runOpts.runName = isEntered ? name : ''
    },
    // cleanup run description input
    onRunDescrBlur (e) {
      for (const lcd in this.runOpts.runDescr) {
        const descr = Mdf.cleanTextInput((this.runOpts.runDescr[lcd] || ''))
        this.runOpts.runDescr[lcd] = descr
      }
    },
    // check if working directory path entered and cleanup input to be compatible with file path rules
    onWorkDirBlur (e) {
      const { isEntered, dir } = this.doDirClean(this.runOpts.workDir)
      this.runOpts.workDir = isEntered ? dir : ''
    },
    doDirClean (dirValue) {
      return (dirValue || '') ? { isEntered: true, dir: Mdf.cleanPathInput(dirValue) } : { isEntered: false, dir: '' }
    },

    // apply preset to run options
    doPresetSelected (preset) {
      if (!preset || !preset?.opts) {
        this.$q.notify({ type: 'warning', message: this.$t('Invalid run options') })
        console.warn('Invalid run options:', preset)
        return
      }
      // merge preset with run options
      const ps = preset.opts

      this.runOpts.subCount = ps.subCount ?? this.runOpts.subCount
      if (this.runOpts.subCount < 1) {
        this.runOpts.subCount = 1
      }
      this.runOpts.threadCount = ps.threadCount ?? this.runOpts.threadCount
      if (this.runOpts.threadCount < 1) {
        this.runOpts.threadCount = 1
      }
      this.runOpts.progressPercent = ps.progressPercent ?? this.runOpts.progressPercent
      if (this.runOpts.progressPercent < 1) {
        this.runOpts.progressPercent = 1
      }
      this.runOpts.progressStep = ps.progressStep ?? this.runOpts.progressStep
      if (this.runOpts.progressStep < 0) {
        this.runOpts.progressStep = 0
      }
      this.runOpts.workDir = ps.workDir ?? this.runOpts.workDir
      this.runOpts.csvDir = ps.csvDir ?? this.runOpts.csvDir
      this.runOpts.useCsvDir = this.serverConfig.AllowFiles && ((this.runOpts.csvDir || '') !== '') && (ps.useCsvDir ?? this.runOpts.useCsvDir)
      this.runOpts.csvId = this.serverConfig.AllowFiles && ((this.runOpts.csvDir || '') !== '') && (ps.csvId ?? this.runOpts.csvId)
      if (this.enableIni) {
        this.runOpts.iniName = ps.iniName ?? this.runOpts.iniName
        this.runOpts.useIni = ((this.runOpts.iniName || '') !== '') && (ps.useIni ?? this.runOpts.useIni)
        if (this.enableIniAnyKey && this.runOpts.useIni) this.runOpts.iniAnyKey = ps.iniAnyKey ?? this.runOpts.iniAnyKey
      } else {
        this.runOpts.useIni = false
        this.enableIniAnyKey = false
      }
      this.runOpts.profile = ps.profile ?? this.runOpts.profile
      this.runOpts.sparseOutput = ps.sparseOutput ?? this.runOpts.sparseOutput
      this.runOpts.runTmpl = ps.runTmpl ?? this.runOpts.runTmpl
      this.runOpts.mpiNpCount = ps.mpiNpCount ?? this.runOpts.mpiNpCount
      if (this.runOpts.mpiNpCount < 0) {
        this.runOpts.mpiNpCount = 0
      }
      this.runOpts.mpiOnRoot = ps.mpiOnRoot ?? this.runOpts.mpiOnRoot
      this.runOpts.mpiUseJobs = this.serverConfig.IsJobControl && (ps.mpiUseJobs ?? (this.serverConfig.IsJobControl && this.runOpts.mpiNpCount > 0))
      this.runOpts.mpiTmpl = ps.mpiTmpl ?? this.runOpts.mpiTmpl

      // expand sections if preset options supplied with non-default values
      this.mpiOptsExpanded = (ps.mpiNpCount || 0) !== 0 || (ps.mpiTmpl || '') !== ''

      this.advOptsExpanded = (ps.threadCount || 0) > 1 ||
        (ps.workDir || '') !== '' ||
        !!ps.useCsvDir ||
        (ps.csvDir || '') !== '' ||
        !!ps.csvId ||
        !!ps.useIni ||
        !!ps.iniAnyKey ||
        (ps.profile || '') !== '' ||
        !!ps.sparseOutput ||
        (ps.runTmpl || '') !== ''

      this.$q.notify({
        type: 'info',
        message: preset?.descr || preset?.label || (this.$t('Using Run Options: ') + preset?.name || '')
      })
    },

    // if previous run request resubmitted then apply settings from run request
    applyRunRequest (rReq) {
      if (!Mdf.isNotEmptyRunRequest(rReq)) {
        console.warn('Invalid (empty) run request', rReq)
        return
      }

      // merge preset with run options
      this.runOpts.runName = Mdf.getRunOption(rReq.Opts, 'OpenM.RunName') || this.runOpts.runName
      this.runOpts.worksetName = Mdf.getRunOption(rReq.Opts, 'OpenM.SetName') || this.runOpts.worksetName
      this.runOpts.baseRunDigest = Mdf.getRunOption(rReq.Opts, 'OpenM.BaseRunDigest') || this.runOpts.baseRunDigest

      this.runOpts.subCount = Mdf.getIntRunOption(rReq.Opts, 'OpenM.SubValues', 0) || this.runOpts.subCount
      if (this.runOpts.subCount < 1) {
        this.runOpts.subCount = 1
      }
      this.runOpts.threadCount = rReq.Threads ?? this.runOpts.threadCount
      if (this.runOpts.threadCount < 1) {
        this.runOpts.threadCount = 1
      }
      this.runOpts.progressPercent = Mdf.getIntRunOption(rReq.Opts, 'OpenM.ProgressPercent', 0) || this.runOpts.progressPercent
      if (this.runOpts.progressPercent < 1) {
        this.runOpts.progressPercent = 1
      }
      this.runOpts.progressStep = Mdf.getIntRunOption(rReq.Opts, 'OpenM.ProgressStep', 0) || this.runOpts.progressStep
      if (this.runOpts.progressStep < 0) {
        this.runOpts.progressStep = 0
      }
      this.runOpts.workDir = rReq.Dir ?? this.runOpts.workDir

      this.runOpts.useCsvDir = false
      if (this.serverConfig.AllowFiles) {
        this.runOpts.csvDir = Mdf.getRunOption(rReq.Opts, 'OpenM.ParamDir') || ''
        this.runOpts.useCsvDir = this.runOpts.csvDir !== ''
        if (this.runOpts.useCsvDir) this.runOpts.csvId = Mdf.getBoolRunOption(rReq.Opts, 'OpenM.IdCsv')
      }
      if (this.enableIni) {
        this.runOpts.iniName = Mdf.getRunOption(rReq.Opts, 'OpenM.IniFile')
        this.runOpts.useIni = this.runOpts.iniName !== ''
        if (this.enableIniAnyKey && this.runOpts.useIni) {
          this.runOpts.iniAnyKey = Mdf.getBoolRunOption(rReq.Opts, 'OpenM.IniAnyKey') || this.runOpts.iniAnyKey
        }
      }

      this.runOpts.profile = Mdf.getRunOption(rReq.Opts, 'OpenM.Profile') || this.runOpts.profile
      this.runOpts.sparseOutput = Mdf.getBoolRunOption(rReq.Opts, 'OpenM.SparseOutput') || this.runOpts.sparseOutput

      if (!rReq?.IsMpi && (rReq?.Mpi?.Np || 0) <= 0) {
        this.runOpts.mpiNpCount = 0
        this.runOpts.runTmpl = rReq.Template ?? this.runOpts.runTmpl
      } else {
        this.runOpts.mpiNpCount = rReq.Mpi.Np ?? this.runOpts.mpiNpCount
        if (this.runOpts.mpiNpCount <= 0) {
          this.runOpts.mpiNpCount = 1
        }
        this.runOpts.mpiUseJobs = this.serverConfig.IsJobControl && !(rReq.Mpi.IsNotByJob ?? false)
        this.runOpts.mpiOnRoot = !(rReq.Mpi.IsNotOnRoot ?? false)
        this.runOpts.mpiTmpl = rReq.Template ?? this.runOpts.mpiTmpl
      }

      // expand tables retained groups into leafs
      if (rReq.Tables.length > 0) {
        this.tablesRetain = [] // clear existing tables retain

        for (const name of rReq.Tables) {
          if (!name) continue

          // if this is not a group then add table name
          if (!Mdf.isGroupName(this.theModel, name)) {
            if (this.tablesRetain.indexOf(name) < 0) this.tablesRetain.push(name)
          } else {
            // expand group into leafs retained
            const gt = this.groupTableLeafs[name]
            if (gt) {
              for (const tn in gt?.leafs) {
                if (this.tablesRetain.indexOf(tn) < 0) {
                  this.tablesRetain.push(tn)
                }
              }
            }
          }
        }
      }

      // restore microdata run options
      if (this.isMicrodata && rReq?.Microdata) {
        // clear existing microdata state
        this.entityAttrsUse = []
        this.microOpts = Mdf.emptyRunRequestMicrodata()

        if (Array.isArray(rReq.Microdata?.Entity)) {
          for (const e of rReq.Microdata.Entity) {
            if (e?.Name && Array.isArray(e?.Attr)) {
              for (const a of e.Attr) {
                this.entityAttrsUse.push(e.Name + '.' + a)
              }
            }
          }
        }
      }

      // use existing run description and notes
      for (let k = 0; k < this.txtNewRun.length; k++) {
        const descr = Mdf.getRunOption(rReq.Opts, this.txtNewRun[k].LangCode + '.RunDescription')
        if (descr !== '') this.txtNewRun[k].Descr = descr

        for (let j = 0; j < rReq.RunNotes.length; j++) {
          if ((rReq.RunNotes[j]?.LangCode || '') === this.txtNewRun[k].LangCode) {
            const note = (rReq.RunNotes[j]?.Note || '')
            if (note !== '') this.txtNewRun[k].Note = note
          }
        }
      }

      // expand sections if run options supplied with non-default values
      this.mpiOptsExpanded = (this.runOpts.mpiNpCount || 0) !== 0 || !!rReq.IsMpi

      this.advOptsExpanded = (rReq.Threads || 0) > 1 ||
        (rReq.Dir || '') !== '' ||
        (this.runOpts.csvDir || '') !== '' ||
        !!this.csvId ||
        !!this.runOpts.useIni ||
        !!this.runOpts.iniAnyKey ||
        (this.runOpts.profile || '') !== '' ||
        !!this.runOpts.sparseOutput ||
        (this.runOpts.runTmpl || '') !== ''

      for (let k = 0; !this.langOptsExpanded && k < this.txtNewRun.length; k++) {
        this.langOptsExpanded = this.txtNewRun[k].Descr !== '' || this.txtNewRun[k].Note !== ''
      }
    },

    // on model run click: if workset partial and no base run and no csv directory then do not run the model
    onModelRunClick () {
      const dgst = (this.useBaseRun && this.isCompletedRunCurrent) ? this.runCurrent?.RunDigest || '' : ''
      const wsName = (this.useWorkset && this.isReadonlyWorksetCurrent) ? this.worksetCurrent?.Name || '' : ''

      if (!dgst && (!this.serverConfig.AllowFiles || !this.runOpts.useCsvDir || ((this.runOpts.csvDir || '') === ''))) {
        if (!wsName) {
          this.$q.notify({ type: 'warning', message: this.$t('Please use input scenario or base model run or CSV files to specifiy input parameters') })
          return
        }
        if (wsName && this.isPartialWorkset()) {
          this.$q.notify({ type: 'warning', message: this.$t('Input scenario should include all parameters otherwise model run may fail: ') + wsName })
          return
        }
      }

      // validate MPI run options if job control enabled on the server
      // and recommend to the user:
      //  mpiUseJobs = true
      //  mpiOnRoot = false
      //  threadCount <= MpiMaxThreads
      //  mpiNpCount = (subCount / threadCount) +1 if subCount % threadCount !== 0
      //  (mpiNpCount * threadCount) <= MaxOwnMpiRes.Cpu and MpiRes.Cpu
      //  mpiNpCount = +1 for root process
      const iProc = Mdf.cleanIntNonNegativeInput(this.runOpts.mpiNpCount, 0)
      let nSub = Mdf.cleanIntNonNegativeInput(this.runOpts.subCount, 1)
      if (nSub <= 0) nSub = 1

      if (nSub > 0 && this.serverConfig.IsJobControl && iProc > 0) {
        const isUseJob = this.runOpts.mpiUseJobs || false
        const isOnRoot = this.runOpts.mpiOnRoot || false

        const iTh = Mdf.cleanIntNonNegativeInput(this.runOpts.threadCount, 1)
        let nTh = (iTh > 0) ? iTh : 1
        let nProc = (iProc > 1) ? iProc - 1 : iProc // do not use root process for modelling

        // do not use more threads than server max threads
        let maxTh = this.serverConfig.MpiMaxThreads || 1
        if (maxTh <= 0) maxTh = 1
        if (nTh > maxTh) nTh = maxTh

        // do not use more cores or threads than sub-values
        if (nProc > nSub) nProc = nSub
        if (nTh > nSub) nTh = nSub

        let nCpu = nProc * nTh
        if (nCpu !== nSub) nCpu = nSub // try to use one core for each sub-value

        // if number of cores > server max MPI cores allowed then find smallest number of cores which can fit into max MPI cores limit
        let maxCpu = this.serverConfig.MpiRes.Cpu || 0
        const ownCpu = this.serverConfig.MaxOwnMpiRes.Cpu || 0
        if (ownCpu > 0 && maxCpu > ownCpu) maxCpu = ownCpu
        if (maxCpu <= 0) maxCpu = 1

        if (nCpu > maxCpu) {
          const n = Math.ceil(nCpu / maxCpu)
          let nc = maxCpu
          let md = nCpu - (n * nc)

          while (nc > 1 && md > 0) {
            md = nCpu - (n * (nc - 1))
            if (md <= 0) break
            nc--
          }
          nCpu = nc
          if (nCpu <= 0) nCpu = 1
        }

        // recalculate number of processes and threads based on the nuber of cores avaliable
        nProc = Math.ceil(nCpu / nTh)
        if (nProc < 1) nProc = 1

        if (nProc * nTh > nCpu) {
          while (nTh > 1 && ((nTh - 1) * nProc) >= nCpu) {
            nTh--
          }
          nCpu = nProc * nTh
        }

        // +1 extra root process, do not use root process for modelling
        nProc++

        // show recommended run options: use run jobs service, do not use root for modelling
        if (!isUseJob || isOnRoot || iTh !== nTh || iProc !== nProc) {
          this.showOptsChange(nSub, nProc, nTh, true, false, maxTh, maxCpu)
          return
        }
      }
      // else do run the model
      this.doModelRun()
    },
    // show yes/no dialog to confirm MPI options changes
    showOptsChange (nSub, nProc, nTh, isUseJob, isOnRoot, maxTh, maxCpu) {
      this.optChange.nSub = nSub
      this.optChange.nProc = nProc
      this.optChange.nThread = nTh
      this.optChange.isUseJob = isUseJob
      this.optChange.isOnRoot = isOnRoot
      this.optChange.maxThread = maxTh
      this.optChange.maxCpu = maxCpu
      this.showOptChangeDialog = true
    },
    // user answer yes and accepted MPI options changes:
    // adjust number of processes, use job contorl, do not run modelling on root
    onYesOptsChange () {
      this.runOpts.subCount = this.optChange.nSub || 1
      this.runOpts.mpiNpCount = this.optChange.nProc || 0
      this.runOpts.threadCount = this.optChange.nThread || 1
      this.runOpts.mpiUseJobs = this.serverConfig.IsJobControl && this.optChange.isUseJob
      this.runOpts.mpiOnRoot = this.serverConfig.IsJobControl && !!this.optChange.isOnRoot
      this.doModelRun() // do run the model
    },

    onNoOptsChange () {
      this.doModelRun() // do run the model
    },

    // run the model
    doModelRun () {
      // set new run options
      this.runOpts.runName = Mdf.cleanFileNameInput(this.runOpts.runName)
      this.runOpts.subCount = Mdf.cleanIntNonNegativeInput(this.runOpts.subCount, 1)
      this.runOpts.threadCount = Mdf.cleanIntNonNegativeInput(this.runOpts.threadCount, 1)
      this.runOpts.workDir = Mdf.cleanPathInput(this.runOpts.workDir)
      this.runOpts.csvDir = Mdf.cleanPathInput(this.runOpts.csvDir)
      this.runOpts.useIni = (this.enableIni && this.runOpts.useIni) || false
      this.runOpts.iniAnyKey = (this.enableIniAnyKey && this.runOpts.useIni && this.runOpts.iniAnyKey) || false
      this.runOpts.profile = Mdf.cleanTextInput(this.runOpts.profile)
      this.runOpts.sparseOutput = this.runOpts.sparseOutput || false
      this.runOpts.runTmpl = Mdf.cleanTextInput(this.runOpts.runTmpl)
      this.runOpts.mpiNpCount = Mdf.cleanIntNonNegativeInput(this.runOpts.mpiNpCount, 0)
      this.runOpts.mpiUseJobs = this.serverConfig.IsJobControl && (this.runOpts.mpiUseJobs || false)
      this.runOpts.mpiOnRoot = this.runOpts.mpiOnRoot || false
      this.runOpts.mpiTmpl = Mdf.cleanTextInput(this.runOpts.mpiTmpl)
      this.runOpts.progressPercent = Mdf.cleanIntNonNegativeInput(this.runOpts.progressPercent, 1)

      this.runOpts.progressStep = Mdf.cleanFloatInput(this.runOpts.progressStep, 0.0)
      if (this.runOpts.progressStep < 0) this.runOpts.progressStep = 0.0

      this.runOpts.worksetName = (this.useWorkset && this.isReadonlyWorksetCurrent) ? this.worksetCurrent?.Name || '' : ''
      this.runOpts.baseRunDigest = (this.useBaseRun && this.isCompletedRunCurrent) ? this.runCurrent?.RunDigest || '' : ''

      // reduce tables retain list by using table groups
      this.retainTablesGroups = [] // retain all tables

      if (this.tablesRetain.length > 0 && this.tablesRetain.length < this.tableCount) {
        let tLst = Array.from(this.tablesRetain)

        // make output tables groups list sorted by group size
        const gLst = []
        for (const gName in this.groupTableLeafs) {
          gLst.push({
            name: gName,
            size: this.groupTableLeafs[gName].size
          })
        }
        gLst.sort((left, right) => { return left.size - right.size })

        // replace table names with group name
        let isAny = false
        do {
          isAny = false

          for (const gs of gLst) {
            const gt = this.groupTableLeafs[gs.name]

            let isAll = true
            for (const tn in gt?.leafs) {
              isAll = tLst.indexOf(tn) >= 0
              if (!isAll) break
            }
            if (!isAll) continue

            tLst = tLst.filter(tn => !gt?.leafs[tn])
            tLst.push(gs.name)
            isAny = true
          }
        } while (isAny)

        this.retainTablesGroups = tLst
      }

      // create microdata entity attributes array
      if (this.isMicrodata && this.entityAttrsUse.length > 0) {
        const mdLst = []

        for (const eau of this.entityAttrsUse) {
          const n = eau.indexOf('.')
          if (n <= 0 || n >= eau.length - 1) continue // skip: expected entity.attribute

          const eName = eau.substring(0, n)
          const aName = eau.substring(n + 1)

          // check if entity attribute exist and if it is an internal attribute
          const a = Mdf.entityAttrTextByName(this.theModel, eName, aName)

          if (a.Attr.Name !== aName) continue // entity attribute not found
          if (a.Attr.IsInternal) this.microOpts.IsInternal = true // allow use of internal attributes

          let isFound = false
          for (const md of mdLst) {
            if (md.Name !== eName) continue
            isFound = true
            md.Attr.push(aName) // entity already found, append another attribute
            break
          }
          if (!isFound) { // entity not found, add new entity and attribute to the list
            mdLst.push({
              Name: eName,
              Attr: []
            })
            mdLst[mdLst.length - 1].Attr.push(aName)
          }
        }

        this.microOpts.IsToDb = mdLst.length > 0
        this.microOpts.Entity = mdLst
      }

      // collect description and notes for each language
      this.newRunNotes = {}
      for (const t of this.txtNewRun) {
        const refKey = 'new-run-note-editor-' + t.LangCode
        if (!Mdf.isLength(this.$refs[refKey]) || !this.$refs[refKey][0]) continue

        const udn = this.$refs[refKey][0].getDescrNote()
        if ((udn.descr || udn.note || '') !== '') {
          this.runOpts.runDescr[t.LangCode] = udn.descr
          this.newRunNotes[t.LangCode] = udn.note
        }
      }

      // start new model run: send request to the server
      this.isInitRun = true
      this.loadWait = true
    },

    // new model run started: response from server
    doneNewRunInit (ok, runStamp, submitStamp) {
      this.isInitRun = false
      this.loadWait = false

      if (!ok || !submitStamp) {
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or model run failed to start') })
        return
      }

      // model wait in the queue
      if (!runStamp) {
        this.$q.notify({ type: 'info', message: this.$t('Model run wait in the queue: ' + Mdf.fromUnderscoreTimeStamp(submitStamp)) })

        if (this.serverConfig.IsJobControl) {
          this.$emit('run-job-select', submitStamp) // show service state and job control page
        } else {
          this.$emit('run-log-select', submitStamp) // no job control: show run log page
        }
      } else {
        // else: model started
        this.$q.notify({ type: 'info', message: this.$t('Model run started: ' + Mdf.fromUnderscoreTimeStamp(runStamp)) })
        this.$emit('run-list-refresh')
        this.$emit('run-log-select', runStamp)
      }
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

    // receive list of ini files from the server if user files enabled
    async doIniRefresh () {
      this.loadIni = true

      // receive list of ini files from the server if user files enabled
      let fLst = []

      if (this.serverConfig.AllowFiles) {
        this.iniFileLst = []
        let isOk = false

        const u = this.omsUrl + '/api/files/file-tree/ini/path'
        try {
          const response = await this.$axios.get(u)
          fLst = response.data
          isOk = true
        } catch (e) {
          let em = ''
          try {
            if (e.response) em = e.response.data || ''
          } finally {}
          console.warn('Server offline or INI files list retrieve failed.', em)
          this.$q.notify({ type: 'negative', message: this.$t('Server offline or INI files list retrieve failed.') })
        }

        // if there are ini files in user files then update ini files path list
        // select default model ini file, if no ini files already selected
        if (isOk && Array.isArray(fLst) && Mdf.isFilePathTree(fLst)) {
          let isDefault = false
          let isCurrent = false
          const dnUf = USER_FILES_PREFIX + this.iniDefaultName
          for (const fi of fLst) {
            if (fi.IsDir || ((fi.Path || '') === '')) continue
            const p = USER_FILES_PREFIX + fi.Path
            this.iniFileLst.push(p)
            if (p === dnUf) isDefault = true
            if (p === this.runOpts.iniName) isCurrent = true
          }
          if (!isCurrent) this.runOpts.iniName = ''
          if (this.runOpts.iniName === '' && isDefault) this.runOpts.iniName = dnUf
          if (!this.runOpts.iniName) this.runOpts.iniName = this.isIniDefault ? this.iniDefaultName : ''
        }
      }
      this.makeIniTreeData(fLst)

      this.loadIni = false
    },

    // receive csv files tree from the server if user files enabled
    async doCsvRefresh () {
      this.loadCsv = true

      // receive list of csv and tsv files from the server if user files enabled
      let fLst = []

      if (this.serverConfig.AllowFiles) {
        let isOk = false

        const u = this.omsUrl + '/api/files/file-tree/' + encodeURIComponent('csv,tsv') + '/path'
        try {
          const response = await this.$axios.get(u)
          fLst = response.data
          isOk = true
        } catch (e) {
          let em = ''
          try {
            if (e.response) em = e.response.data || ''
          } finally {}
          console.warn('Server offline or CSV files tree retrieve failed.', em)
          this.$q.notify({ type: 'negative', message: this.$t('Server offline or CSV files tree retrieve failed.') })
        }

        // if there are csv files in user files then update csv files tree
        // if current csv directory not does exists then de-select current csv directory
        if (isOk && Array.isArray(fLst) && Mdf.isFilePathTree(fLst)) {
          let isCurrent = false
          for (const fi of fLst) {
            if (fi.IsDir || ((fi.Path || '') === '')) continue
            isCurrent = fi.Path === this.runOpts.csvDir
            if (isCurrent) break
          }
          if (!isCurrent) this.runOpts.csvDir = ''
        }
        if (this.runOpts.csvDir === '') this.runOpts.useCsvDir = false
      }
      this.makeCsvTreeData(fLst)

      this.loadCsv = false
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
    },

    // receive profile list by model digest
    async doProfileListRefresh () {
      let isOk = false
      this.loadProfile = true

      const u = this.omsUrl + '/api/model/' + encodeURIComponent(this.digest) + '/profile-list'
      try {
        const response = await this.$axios.get(u)

        // expected string array of profile names
        // append empty '' string first to make default selection === "no profile"
        this.profileLst = []
        if (Mdf.isLength(response.data)) {
          this.profileLst.push('')
          for (const p of response.data) {
            this.profileLst.push(p)
          }
        }
        isOk = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or profile list retrieve failed.', em)
      }
      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or profile list retrieve failed: ') + this.digest })
      }
      this.loadProfile = false
    }
  },

  mounted () {
    this.doRefresh()
    this.$emit('tab-mounted', 'new-run', { digest: this.digest })
  }
}
