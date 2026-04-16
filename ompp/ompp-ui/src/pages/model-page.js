import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'
import RefreshModel from 'components/RefreshModel.vue'
import RefreshRun from 'components/RefreshRun.vue'
import RefreshRunList from 'components/RefreshRunList.vue'
import RefreshRunArray from 'components/RefreshRunArray.vue'
import RefreshWorkset from 'components/RefreshWorkset.vue'
import RefreshWorksetList from 'components/RefreshWorksetList.vue'
import RefreshWorksetArray from 'components/RefreshWorksetArray.vue'
import RefreshUserViews from 'components/RefreshUserViews.vue'
import RunBar from 'components/RunBar.vue'
import RunInfoDialog from 'components/RunInfoDialog.vue'
import WorksetBar from 'components/WorksetBar.vue'
import WorksetInfoDialog from 'components/WorksetInfoDialog.vue'

/* eslint-disable no-multi-spaces */
const RUN_LST_TAB_POS = 1       // model runs list tab position
const WS_LST_TAB_POS = 4        // worksets list tab position
const NEW_RUN_TAB_POS = 8       // new model run tab position
const UP_DOWN_TAB_POS = 12      // downloads and uploads list tab position
const FREE_TAB_POS = 20         // first unassigned tab position
/* eslint-enable no-multi-spaces */

export default {
  name: 'ModelPage',
  components: {
    RefreshModel,
    RefreshRun,
    RefreshRunList,
    RefreshRunArray,
    RefreshWorkset,
    RefreshWorksetList,
    RefreshWorksetArray,
    RefreshUserViews,
    RunBar,
    RunInfoDialog,
    WorksetBar,
    WorksetInfoDialog
  },

  props: {
    digest: { type: String, default: '' },
    refreshTickle: { type: Boolean, default: false }
  },

  /* eslint-disable no-multi-spaces */
  data () {
    return {
      loadModelDone: false,
      loadRunDone: false,
      loadRunListDone: false,
      loadRunViewsDone: false,
      loadWsDone: false,
      loadWsListDone: false,
      loadWsViewsDone: false,
      loadUserViewsDone: false,
      refreshRunListTickle: false,
      refreshRunTickle: false,
      refreshWsListTickle: false,
      refreshWsTickle: false,
      refreshWsToRun: false,
      refreshRunViewsTickle: false,
      refreshWsViewsTickle: false,
      uploadViewsTickle: false,
      modelName: '',
      runDnsCurrent: '',        // run digest selected (run name, run stamp)
      wsNameCurrent: '',        // workset name selected
      runViewsArray: [],        // digests of runs to refresh to view existing tabs
      wsViewsArray: [],         // names of worksets to refresh to view existing tabs
      activeTabKey: '',         // active tab path
      tabItems: [],             // tab list
      runInfoTickle: false,
      worksetInfoTickle: false,
      toUpDownSection: 'down',    // downloads and uploads page active section
      paramEditCount: 0,          // number of edited and unsaved parameters
      pathToRouteLeave: '',       // router component leave guard: path-to leave if user confirm changes discard
      isYesToLeave: false,        // if true then do router push to leave the page
      showAllDiscardDlg: false
    }
  },
  /* eslint-enable no-multi-spaces */

  computed: {
    isEmptyTabList () { return !Mdf.isLength(this.tabItems) },
    runTextCount () { return Mdf.runTextCount(this.runTextList) },
    worksetTextCount () { return Mdf.worksetTextCount(this.worksetTextList) },

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
      'runDigestSelected',
      'worksetNameSelected'
    ])
  },

  watch: {
    digest () { this.initalView() },
    refreshTickle () { this.initalView() }
  },

  emits: [
    'set-update-readonly',
    'user-view-updated'
  ],

  methods: {
    ...mapActions(useModelStore, [
      'runTextByDigest',
      'isExistInWorksetTextList'
    ]),
    ...mapActions(useUiStateStore, [
      'tabsView',
      'paramViewUpdatedCount',
      'paramViewWorksetUpdatedCount',
      //
      'dispatchRunDigestSelected',
      'dispatchWorksetNameSelected',
      'dispatchViewSelectedRestore',
      'dispatchParamViewDeleteByModel',
      'dispatchTabsView'
    ]),

    // update page view
    initalView () {
      this.doTabAdd('run-list', { digest: this.digest })
      this.doTabAdd('set-list', { digest: this.digest })
      this.doTabAdd('new-run', { digest: this.digest })
    },
    doRefresh () {
      if (!this.tabItems || !this.tabItems.length) this.initalView() // add standard tabs

      this.dispatchViewSelectedRestore(this.digest) // restore selected run digest and workset name
      const tv = this.tabsView(this.digest) // list of additional tabs: parameters or tables tabs

      // restore additional tabs
      this.runViewsArray = []
      this.wsViewsArray = []

      for (const t of tv) {
        if (t?.kind && t?.routeParts?.digest) {
          this.doTabAdd(t.kind, t.routeParts, true)

          const rd = t.routeParts?.runDigest || ''
          if (rd && !this.runViewsArray.includes(rd)) this.runViewsArray.push(rd)

          const wsn = t.routeParts?.worksetName || ''
          if (wsn && !this.wsViewsArray.includes(wsn)) this.wsViewsArray.push(wsn)
        }
      }

      // reload run text for additional tabs
      this.loadRunViewsDone = this.runViewsArray.length <= 0
      if (!this.loadRunViewsDone) this.refreshRunViewsTickle = !this.refreshRunViewsTickle

      this.loadWsViewsDone = this.wsViewsArray.length <= 0
      if (!this.loadWsViewsDone) this.refreshWsViewsTickle = !this.refreshWsViewsTickle

      // if current path is not a one of tabs then route to the first tab
      for (const t of this.tabItems) {
        if (t.path === this.$route.path) {
          this.activeTabKey = t.path
          break
        }
      }
      if (!this.activeTabKey) {
        this.$router.push(this.tabItems[0].path)
      }
    },

    doneModelLoad (isSuccess, dgst) {
      this.modelName = Mdf.modelName(this.theModel)
      this.loadModelDone = true
      document.title = this.modelName

      // on successs update run list and workset list if empty or from other model
      if (isSuccess && (dgst || '') === this.digest) {
        if (!Mdf.isLength(this.runTextList) || (this.runTextListrunTextList?.[0]?.ModelDigest || '') !== dgst) {
          this.refreshRunListTickle = !this.refreshRunListTickle
        }
        if (!Mdf.isLength(this.worksetTextList) || (this.worksetTextList?.[0]?.ModelDigest || '') !== dgst) {
          this.refreshWsListTickle = !this.refreshWsListTickle
        }
      }
      this.doRefresh()
    },
    doneRunListLoad (isSuccess) {
      this.loadRunListDone = true
      //
      if (!isSuccess || !Mdf.isLength(this.runTextList)) { // do not refresh run: run list empty
        this.loadRunDone = true
        this.runDnsCurrent = ''
        this.dispatchRunDigestSelected({ digest: this.digest, runDigest: '' })
        return
      } // else: if run not selected then use first run: first successful or first completed or first
      if (!this.runDigestSelected) {
        this.dispatchRunDigestSelected({ digest: this.digest, runDigest: '' })
        this.runDnsCurrent = Mdf.lastRunDigest(this.runTextList)
        this.refreshRunTickle = !this.refreshRunTickle
        return
      }
      // else: if run already selected then make sure it still exist
      const rt = this.runTextByDigest({ ModelDigest: this.digest, RunDigest: this.runDigestSelected })
      if (!Mdf.isNotEmptyRunText(rt)) {
        this.dispatchRunDigestSelected({ digest: this.digest, runDigest: '' })
        this.runDnsCurrent = Mdf.lastRunDigest(this.runTextList)
        this.refreshRunTickle = !this.refreshRunTickle
        return
      }
      // else: if run completed and run parameters list loaded then exit
      if (Mdf.isRunCompletedStatus(rt?.Status) && Array.isArray(rt?.Param) && (rt?.Param?.length || 0) === Mdf.paramCount(this.theModel)) {
        this.loadRunDone = true
        return
      }
      // else: refresh run parameters list and run status
      this.runDnsCurrent = this.runDigestSelected
      this.dispatchRunDigestSelected({ digest: this.digest, runDigest: '' })
      this.refreshRunTickle = !this.refreshRunTickle
    },
    doneRunLoad (isSuccess, dgst) {
      this.loadRunDone = true
      if (isSuccess && (dgst || '') !== '') this.dispatchRunDigestSelected({ digest: this.digest, runDigest: dgst })
    },
    doneRunViewsLoad (isSuccess, count) {
      this.loadRunViewsDone = true
    },
    doneWsListLoad (isSuccess) {
      this.loadWsListDone = true
      //
      if (!isSuccess || !Mdf.isLength(this.worksetTextList)) { // do not refresh workset: workset list empty
        this.loadWsDone = true
        this.wsNameCurrent = ''
        this.dispatchWorksetNameSelected({ digest: this.digest, worksetName: '' })
        return
      } // else: if workset already selected then make sure it still exist, if not exist then use first workset
      if (!!this.worksetNameSelected && this.isExistInWorksetTextList({ ModelDigest: this.digest, Name: this.worksetNameSelected })) {
        this.wsNameCurrent = this.worksetNameSelected
      } else {
        this.wsNameCurrent = this.worksetTextList[0].Name
        this.dispatchWorksetNameSelected({ digest: this.digest, worksetName: '' })
      }
      this.refreshWsTickle = !this.refreshWsTickle
    },
    doneWsLoad (isSuccess, name, isNewRun) {
      this.loadWsDone = true
      this.refreshWsToRun = false
      //
      if (isSuccess && (name || '') !== '') {
        this.dispatchWorksetNameSelected({ digest: this.digest, worksetName: name })
        if (isNewRun) this.doNewRunSelect()
      }
    },
    doneWsViewsLoad (isSuccess, count) {
      this.loadWsViewsDone = true
    },
    doneUserViewsLoad (isSuccess, nViews) {
      this.loadUserViewsDone = true
      if (nViews > 0) {
        this.$q.notify({ type: 'info', message: this.$t('User views count: ') + nViews.toString() })
      }
    },
    // deleting run or multiple runs
    onRunListDelete () {
      this.doTabFilter(
        (t) => {
          if (t?.routeParts?.digest !== this.digest) return true
          const rd = t?.routeParts?.runDigest || ''
          if (rd) {
            return this.runTextList.findIndex(r => r?.RunDigest === rd) < 0
          }
          const rs = t?.routeParts?.runStamp || ''
          if (t.kind === 'run-log' && rs !== '') {
            return this.runTextList.findIndex(r => r?.RunStamp === rs || r?.RunDigest === rs || r?.Name === rs) < 0
          }
          return false
        }
      )
    },
    // deleting workset or multiple worksets
    onWorksetListDelete () {
      this.doTabFilter(
        (t) => {
          if (t?.routeParts?.digest !== this.digest) return true
          const nm = t?.routeParts?.worksetName || ''
          if (nm) {
            return this.worksetTextList.findIndex(w => w?.Name === nm) < 0
          }
          return false
        }
      )
    },
    // deleting parameter from workset
    onWorksetParamDelete (dgst, wsName, name) {
      if (!dgst || !wsName || !name) return

      // clean tabs from deleted workset parameter
      this.doTabFilter(
        (t) => {
          return t?.kind === 'set-parameter' &&
            t?.routeParts?.digest === dgst &&
            t?.routeParts?.worksetName === wsName &&
            t?.routeParts?.parameterName === name
        }
      )
    },

    // run(s) completed: refresh run text for selected run
    onRunCompletedList (rcArr) {
      if (!Array.isArray(rcArr) || rcArr.length === 0) {
        console.warn('Invalid (empty) list of completed runs')
        return
      }

      const idx = ((this.runDigestSelected || '') !== '') ? rcArr.indexOf(this.runDigestSelected) : 0

      if (idx >= 0) {
        this.refreshRunTickle = !this.refreshRunTickle
        this.runDnsCurrent = rcArr[idx]
      }
    },
    // run selected from the list: update current run
    onRunSelect (dgst) {
      if ((dgst || '') === '') {
        console.warn('selected run digest is empty')
        return
      }
      this.runDnsCurrent = dgst
      this.refreshRunTickle = !this.refreshRunTickle
    },
    // run started: refresh run list
    onRunListRefresh () {
      this.refreshRunListTickle = !this.refreshRunListTickle
    },
    // workset selected from the list: update current workset
    onWorksetSelect (name) {
      if ((name || '') === '') {
        console.warn('selected workset name is empty')
        return
      }
      this.wsNameCurrent = name
      this.refreshWsTickle = !this.refreshWsTickle
    },
    // refresh workset list, for example after delete
    onWorksetListRefresh () {
      this.refreshWsListTickle = !this.refreshWsListTickle
    },
    // run parameter selected from parameters list: go to run parameter page
    onRunParamSelect (name) {
      const p = this.doTabAdd('run-parameter', { digest: this.digest, runDigest: this.runDigestSelected, parameterName: name })
      if (p) this.$router.push(p)
    },
    // workset parameter selected from parameters list: go to workset parameter page
    onSetParamSelect (name) {
      const p = this.doTabAdd('set-parameter', { digest: this.digest, worksetName: this.worksetNameSelected, parameterName: name })
      if (p) this.$router.push(p)
    },
    // output table selected from tables list: go to output table page
    onTableSelect (name) {
      const p = this.doTabAdd('table', { digest: this.digest, runDigest: this.runDigestSelected, tableName: name })
      if (p) this.$router.push(p)
    },
    // run entity microdata selected from tables list: go to microdata page
    onEntitySelect (name) {
      const p = this.doTabAdd('entity', { digest: this.digest, runDigest: this.runDigestSelected, entityName: name })
      if (p) this.$router.push(p)
    },

    // update workset readonly status
    onWorksetReadonlyUpdate (dgst, name, isReadonly) {
      if (dgst !== this.digest || !name) return

      if (isReadonly) {
        // if there are any edited and unsaved parameters for this workset
        const n = this.paramViewWorksetUpdatedCount({ digest: this.digest, worksetName: name })
        if (n > 0) {
          console.warn('Unable to save input scenario: unsaved parameters count:', name, n)
          this.$q.notify({
            type: 'negative',
            message: this.$t('Unable to save input scenario {setName} because you have {count} unsaved parameter(s)', { setName: name, count: n })
          })
          return
        }
      }
      // else: update workset status
      this.$emit('set-update-readonly', dgst, name, isReadonly)
    },
    // on parameter updated (data dirty flag) change
    onEditUpdated (isUpdated, tabPath) {
      const nPos = this.tabItems.findIndex(t => t.path === tabPath)
      if (nPos >= 0) this.tabItems[nPos].updated = isUpdated
    },
    // upload views into user home directory
    onUserViewUpdated (dgst, mName) {
      this.$emit('user-view-updated', dgst, mName)
    },
    // show run notes dialog
    doShowRunNote (modelDgst, runDgst) {
      this.runInfoTickle = !this.runInfoTickle
    },
    // show current workset notes dialog
    doShowWorksetNote (modelDgst, name) {
      this.worksetInfoTickle = !this.worksetInfoTickle
    },
    // view run log: add tab with open run log page
    onRunLogSelect (stamp) {
      const p = this.doTabAdd('run-log', { digest: this.digest, runStamp: stamp })
      if (p) this.$router.push(p)
    },
    // view service state and jobs control page
    onRunJobSelect (stamp) {
      this.$router.push('/service-state')
    },
    // view downloads: add tab with open downloads page section
    onDownloadSelect () {
      const p = this.doTabAdd('updown-list', { digest: this.digest })
      if (p) {
        this.toUpDownSection = 'down'
        if (this.$route.path === p) {
          this.activeTabKey = p
        } else {
          this.$router.push(p)
        }
      }
    },
    // view uploads: add tab with open uploads page section
    onUploadSelect () {
      const p = this.doTabAdd('updown-list', { digest: this.digest })
      if (p) {
        this.toUpDownSection = 'up'
        if (this.$route.path === p) {
          this.activeTabKey = p
        } else {
          this.$router.push(p)
        }
      }
    },
    // new model run selected by user:
    // if workset name supplied then select workset first and open run tab after
    // if no workset name then open new run tab with currently selected workset
    onNewRunSelect (name) {
      if ((name || '') !== '') {
        this.wsNameCurrent = name
        this.refreshWsTickle = !this.refreshWsTickle
        this.refreshWsToRun = true // wait until workset select completed
        return
      }
      // else: workset already selected, add new run tab
      this.doNewRunSelect()
    },
    // add tab to run the model
    doNewRunSelect () {
      const p = this.doTabAdd('new-run', { digest: this.digest })
      if (p) this.$router.push(p)
    },

    // on click tab close button: close tab and route to the next tab
    onTabCloseClick (tabPath, evt) {
      const nPos = this.tabItems.findIndex(t => t.path === tabPath)

      if (nPos < 0) {
        console.warn('onTabCloseClick error: not found tab key:', tabPath)
      } else {
        const kind = this.tabItems[nPos].kind
        this.tabItems = this.tabItems.filter((ti, idx) => idx !== nPos)

        // if tab was active then focus on the next tab
        if (tabPath === this.activeTabKey || tabPath === this.$route.path) {
          const len = Mdf.lengthOf(this.tabItems)
          const n = (nPos < len) ? nPos : nPos - 1
          this.activeTabKey = (n >= 0) ? this.tabItems[n].path : ''
          if (this.activeTabKey !== '') {
            this.$router.push(this.activeTabKey)
          }
        }

        // if parameter or table tab closed then save list of tab item in state store
        if (kind === 'run-parameter' || kind === 'set-parameter' || kind === 'table' || kind === 'entity') this.storeTabItems()
      }

      // cancel event to avoid bubbling up to the tab router link
      evt.stopPropagation()
      evt.preventDefault()
    },

    // close tabs by filter condition, for example all tabs for deleted run
    doTabFilter (tabFilter) {
      let isActive = false
      let activePos = 0
      let isStore = false

      for (let k = this.tabItems.length - 1; k >= 0; k--) {
        if (!tabFilter(this.tabItems[k])) continue // skip: this tab not in remove filter

        if (this.tabItems[k].path === this.activeTabKey || this.tabItems[k].path === this.$route.path) {
          isActive = true
          activePos = k
        }
        if (this.tabItems[k].kind === 'run-parameter' || this.tabItems[k].kind === 'set-parameter' || this.tabItems[k].kind === 'table' || this.tabItems[k].kind === 'entity') {
          isStore = true
        }

        this.tabItems.splice(k, 1)
      }

      if (isActive) {
        const len = Mdf.lengthOf(this.tabItems)
        const n = (activePos < len) ? activePos : activePos - 1
        this.activeTabKey = (n >= 0) ? this.tabItems[n].path : ''
        if (this.activeTabKey !== '' && this.activeTabKey !== this.$route.path) this.$router.push(this.activeTabKey)
      }

      // if parameter or table tab closed then save list of tab item in state store
      if (isStore) this.storeTabItems()
    },

    // save list of parameters or tables tabs in store state
    storeTabItems () {
      const tv = []
      for (const t of this.tabItems) {
        if (t.kind === 'run-parameter' || t.kind === 'set-parameter' || t.kind === 'table' || t.kind === 'entity') {
          tv.push({ kind: t.kind, routeParts: t.routeParts })
        }
      }
      this.dispatchTabsView({ digest: this.digest, tabs: tv })
    },

    // tab mounted: add to the tab list and if this is current router path then make this tab active
    onTabMounted (kind, routeParts) {
      const p = this.doTabAdd(kind, routeParts)
      if (p === this.$route.path) this.activeTabKey = p
    },

    // if tab not exist then add new tab
    doTabAdd (kind, routeParts, isRestore = false) {
      const ti = this.makeTabInfo(kind, routeParts)
      if ((ti.path || '') === '') {
        console.warn('tab kind or route part(s) invalid or empty:', kind, routeParts)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to navigate: route part(s) invalid or empty') })
        return ''
      }

      // exit if tab already exist
      if (this.tabItems.findIndex((t) => t.path === ti.path) >= 0) return (ti.path || '')

      // insert predefined tab into tab list or append to tab list
      let nPos = -1
      if (ti.pos < FREE_TAB_POS) {
        nPos = 0
        while (nPos < this.tabItems.length) {
          if (this.tabItems[nPos].pos > ti.pos) break
          nPos++
        }
      } else {
        if (!isRestore) {
          nPos = 0
          while (nPos < this.tabItems.length) {
            if (this.tabItems[nPos].pos >= FREE_TAB_POS) break
            nPos++
          }
        }
      }
      if (nPos >= 0 && nPos < this.tabItems.length) {
        this.tabItems.splice(nPos, 0, ti)
      } else {
        this.tabItems.push(ti)
      }
      // if parameter, table or microdata tab added then save list of tab item in state store
      if (kind === 'run-parameter' || kind === 'set-parameter' || kind === 'table' || kind === 'entity') this.storeTabItems()

      return (ti.path || '')
    },

    makeTabInfo (tabKind, parts) {
      // empty tab info: invalid default value
      const emptyTabInfo = () => {
        return { kind: (tabKind || ''), path: '', routeParts: { digest: '' }, pos: 0, updated: false }
      }

      // tab kind must be defined and model digest same as current model
      if ((tabKind || '') === '' || !parts || (parts.digest || '') !== this.digest) return emptyTabInfo()

      const udgst = encodeURIComponent(this.digest)

      switch (tabKind) {
        case 'run-list':
          return {
            kind: tabKind,
            path: '/model/' + udgst + '/run-list',
            routeParts: parts,
            pos: RUN_LST_TAB_POS,
            updated: false
          }

        case 'set-list':
          return {
            kind: tabKind,
            path: '/model/' + udgst + '/set-list',
            routeParts: parts,
            pos: WS_LST_TAB_POS,
            updated: false
          }

        // path: '/model/' + this.digest + '/run/' + parts.runDigest + '/parameter/' + parts.parameterName,
        case 'run-parameter': {
          if ((parts.runDigest || '') === '' || (parts.parameterName || '') === '') {
            console.warn('Invalid (empty) run digest or parameter name:', parts.runDigest, parts.parameterName)
            return emptyTabInfo()
          }
          return {
            kind: tabKind,
            path: Mdf.parameterRunPath(this.digest, parts.runDigest, parts.parameterName),
            routeParts: parts,
            pos: FREE_TAB_POS,
            updated: false
          }
        }

        // path: '/model/' + this.digest + '/set/' + parts.worksetName + '/parameter/' + parts.parameterName,
        case 'set-parameter': {
          if ((parts.worksetName || '') === '' || (parts.parameterName || '') === '') {
            console.warn('Invalid (empty) workset name or parameter name:', parts.worksetName, parts.parameterName)
            return emptyTabInfo()
          }
          return {
            kind: tabKind,
            path: Mdf.parameterWorksetPath(this.digest, parts.worksetName, parts.parameterName),
            routeParts: parts,
            pos: FREE_TAB_POS,
            updated: false
          }
        }

        // path: '/model/' + this.digest + '/run/' + parts.runDigest + '/table/' + parts.tableName,
        case 'table': {
          if ((parts.runDigest || '') === '' || (parts.tableName || '') === '') {
            console.warn('Invalid (empty) run digest or output table name:', parts.runDigest, parts.tableName)
            return emptyTabInfo()
          }
          return {
            kind: tabKind,
            path: Mdf.tablePath(this.digest, parts.runDigest, parts.tableName),
            routeParts: parts,
            pos: FREE_TAB_POS,
            updated: false
          }
        }

        // path: '/model/' + this.digest + '/run/' + parts.runDigest + '/entity/' + parts.entityName,
        case 'entity': {
          if ((parts.runDigest || '') === '' || (parts.entityName || '') === '') {
            console.warn('Invalid (empty) run digest or entity name:', parts.runDigest, parts.entityName)
            return emptyTabInfo()
          }
          return {
            kind: tabKind,
            path: Mdf.microdataPath(this.digest, parts.runDigest, parts.entityName),
            routeParts: parts,
            pos: FREE_TAB_POS,
            updated: false
          }
        }

        case 'new-run':
          return {
            kind: tabKind,
            path: '/model/' + udgst + '/new-run',
            routeParts: parts,
            pos: NEW_RUN_TAB_POS,
            updated: false
          }

        case 'run-log': {
          if ((parts.runStamp || '') === '') {
            console.warn('Invalid (empty) run stamp:', parts.runStamp)
            return emptyTabInfo()
          }
          return {
            kind: tabKind,
            path: '/model/' + udgst + '/run-log/' + encodeURIComponent(parts.runStamp),
            routeParts: parts,
            pos: FREE_TAB_POS,
            updated: false
          }
        }

        case 'updown-list':
          return {
            kind: tabKind,
            path: '/model/' + udgst + '/updown-list',
            routeParts: parts,
            pos: UP_DOWN_TAB_POS,
            updated: false
          }
      }
      // default
      console.warn('tab kind invalid:', tabKind)
      return emptyTabInfo()
    },

    // make tab title: pre-defined name or description of parameter, table or microdata entity
    tabTitle (tabKind, parts) {
      switch (tabKind) {
        case 'run-list':
          return this.$t('Model Runs')

        case 'set-list':
          return this.$t('Input Scenarios')

        case 'run-parameter': {
          if ((parts.runDigest || '') === '' || (parts.parameterName || '') === '') {
            console.warn('Invalid (empty) run digest or parameter name:', parts.runDigest, parts.parameterName)
            return ''
          }
          const pds = Mdf.descrOfDescrNote(Mdf.paramTextByName(this.theModel, parts.parameterName))
          return (pds !== '') ? pds : parts.parameterName
        }

        case 'set-parameter': {
          if ((parts.worksetName || '') === '' || (parts.parameterName || '') === '') {
            console.warn('Invalid (empty) workset name or parameter name:', parts.worksetName, parts.parameterName)
            return ''
          }
          const pds = Mdf.descrOfDescrNote(Mdf.paramTextByName(this.theModel, parts.parameterName))
          return (pds !== '') ? pds : parts.parameterName
        }

        // path: '/model/' + this.digest + '/run/' + parts.runDigest + '/table/' + parts.tableName,
        case 'table': {
          if ((parts.runDigest || '') === '' || (parts.tableName || '') === '') {
            console.warn('Invalid (empty) run digest or output table name:', parts.runDigest, parts.tableName)
            return ''
          }
          const tds = Mdf.tableTextByName(this.theModel, parts.tableName)?.TableDescr || ''
          return (tds !== '') ? tds : parts.tableName
        }

        // path: '/model/' + this.digest + '/run/' + parts.runDigest + '/entity/' + parts.entityName,
        case 'entity': {
          if ((parts.runDigest || '') === '' || (parts.entityName || '') === '') {
            console.warn('Invalid (empty) run digest or entity name:', parts.runDigest, parts.entityName)
            return ''
          }
          const eds = Mdf.descrOfDescrNote(Mdf.entityTextByName(this.theModel, parts.entityName))
          return (eds !== '') ? eds : parts.entityName
        }

        case 'new-run':
          return this.$t('Run the Model')

        case 'run-log': {
          if ((parts.runStamp || '') === '') {
            console.warn('Invalid (empty) run stamp:', parts.runStamp)
            return ''
          }
          let rn = parts.runStamp
          for (const rt of this.runTextList) {
            if (rt.ModelDigest === parts.digest && rt.RunStamp === parts.runStamp) {
              rn = rt.Name
              break
            }
          }
          return rn
        }

        case 'updown-list':
          return this.$t('Downloads and Uploads')
      }
      // default
      console.warn('tab kind invalid:', tabKind)
      return ''
    },

    // before route to other page question: "Discard all changes?", user answer: "yes"
    onYesDiscardChanges () {
      this.dispatchParamViewDeleteByModel(this.digest) // discard parameter view state and edit changes
      this.isYesToLeave = true
      this.$router.push(this.pathToRouteLeave)
    }
  },

  // change active tab on route update or redirect from default model path to current page
  beforeRouteUpdate (to, from) {
    // if to.path is one of the tabs then change active tab
    let isFrom = false
    for (const t of this.tabItems) {
      if (t.path === to.path) {
        this.activeTabKey = t.path // set active tab and do navigation
        return true
      }
      if (!isFrom) isFrom = t.path === from.path
    }
    // else:
    // if to.path is default model/digest and one of the tabs is from.path then cancel navigation
    if (isFrom && to.path === '/model/' + encodeURIComponent(this.digest)) {
      if (this.activeTabKey !== from.path) this.activeTabKey = from.path
      return false
    }
    return true // else default
  },

  // route leave guard: on leaving model page check
  // if any parameters edited and not changes saved then ask user to confirm "discard all changes?"
  beforeRouteLeave (to, from) {
    // if user already confimed "yes" to leave the page
    if (this.isYesToLeave) {
      return true // leave model page and route to next page
    }
    // else:
    //  if there any edited and unsaved parameters for current model
    this.paramEditCount = this.paramViewUpdatedCount(this.digest)
    if (this.paramEditCount <= 0) {
      return true // no unsaved changes: leave model page and route to next page
    }
    // else:
    //  redirect to dialog to confirm "discard all changes?"
    this.showAllDiscardDlg = true
    this.pathToRouteLeave = to.path || '' // store path-to leave if user confirm "yes" to "discard changes?"
    return false
  },

  mounted () {
    this.initalView()
  }
}
