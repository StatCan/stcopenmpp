import { mapState } from 'pinia'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import DeleteConfirmDialog from 'components/DeleteConfirmDialog.vue'
import * as Mdf from 'src/model-common'
import { copyToClipboard } from 'quasar'

/* eslint-disable no-multi-spaces */
const ADMIN_STATE_REFRESH_TIME = 1973 // msec, admin state refresh interval
const MAX_STATE_SEND_COUNT = 5        // max request to send without response
const MAX_NO_STATE_COUNT = 4          // max invalid response count
/* eslint-enable no-multi-spaces */

export default {
  name: 'AdminState',
  components: { DeleteConfirmDialog },

  props: {
    refreshTickle: { type: Boolean, default: false }
  },

  data () {
    return {
      adminState: Mdf.emptyAdminState(),
      isShowServers: false,
      isShowOmsActive: false,
      isShowActiveRuns: true,
      isShowQueueRuns: false,
      locale: '',
      activeRunState: { // active job control state: key is 'Oms-SubmitStamp', value is RunJob
      },
      isShowActiveFilter: false,
      omsActiveFilter: [],
      nameActiveFilter: [],
      digestActiveFilter: [],
      omsActiveOpts: [],
      nameActiveOpts: [],
      digestActiveOpts: [],
      activeRunLog: { // active job log selected: key, file name and log file content
        user: '',
        submit: '',
        fileName: '',
        lines: []
      },
      queueRunState: { // queue job control state: key is 'Oms-SubmitStamp', value is RunJob
      },
      pastRunState: { // past job control state: key is 'Year_Month-Oms-SubmitStamp', value is PastRunJob
      },
      isShowQueueFilter: false,
      omsQueueFilter: [],
      nameQueueFilter: [],
      digestQueueFilter: [],
      omsQueueOpts: [],
      nameQueueOptsr: [],
      digestQueueOpts: [],
      pastRunLog: { // past job log selected: key, file name and log file content
        yearMonth: '',
        user: '',
        submit: '',
        fileName: '',
        lines: []
      },
      stopRunTitle: '',
      stopSubmitStamp: '',
      stopOmsName: '',
      showStopRunTickle: false,
      isShowPastRuns: false,
      pastRefreshTs: '',
      isPastPulling: false,
      pastRuns: [],
      isShowPastFilter: false,
      ymPastFilter: [],
      omsPastFilter: [],
      namePastFilter: [],
      digestPastFilter: [],
      statusPastFilter: [],
      ymPastOpts: [],
      omsPastOpts: [],
      namePastOpts: [],
      digestPastOpts: [],
      statusPastOpts: Object.freeze(['success', 'kill', 'error']),
      isRefreshPaused: false,
      isRefreshDisabled: false,
      stateRefreshTickle: 0,
      stateSendCount: 0,
      stateNoDataCount: 0,
      stateRefreshInt: ''
    }
  },

  computed: {
    isConfigAdminAll () { return !!this.serverConfig.IsAdminAll },

    // return true if there are any active runs filters
    isAnyActiveFilter () {
      return this.omsActiveFilter.length + this.nameActiveFilter.length + this.digestActiveFilter.length > 0
    },
    // return true if there are any queue runs filters
    isAnyQueueFilter () {
      return this.omsQueueFilter.length + this.nameQueueFilter.length + this.digestQueueFilter.length > 0
    },
    // return true if there are any past runs filters
    isAnyPastFilter () {
      return this.ymPastFilter.length + this.omsPastFilter.length + this.namePastFilter.length + this.digestPastFilter.length + this.statusPastFilter.length > 0
    },
    // count of past job folders
    pastFolderCount () {
      let n = 0
      if (Array.isArray(this.pastRuns)) {
        for (const p of this.pastRuns) {
          if (typeof p?.IsDir === typeof true && p?.IsDir) n++
        }
      }
      return n
    },
    // count of past job files
    pastFileCount () {
      let n = 0
        for (const p of this.pastRuns) {
          if (typeof p?.IsDir === typeof true && !p?.IsDir) n++
      }
      return n
    },
    // count of past job files which are matching filters
    pastInFilterCount() {
      let n = 0
        for (const p of this.pastRuns) {
          if ((typeof p?.IsDir === typeof true && !p?.IsDir) && this.isInPastFilter(p)) n++
      }
      return n
    },

    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl',
      serverConfig: 'config'
    }),
    ...mapState(useUiStateStore, [
      'uiLang'
    ])
  },

  watch: {
    refreshTickle () { this.initView() },
    isConfigAdminAll () { this.initView() }
  },

  methods: {
    isUnderscoreTs (ts) { return Mdf.isUnderscoreTimeStamp(ts) },
    fromUnderscoreTs (ts) { return Mdf.isUnderscoreTimeStamp(ts) ? Mdf.fromUnderscoreTimeStamp(ts) : ts },

    isSuccess (status) { return status === 'success' },
    isInProgress (status) { return status === 'progress' || status === 'init' || status === 'wait' },
    runStatusDescr (status) { return Mdf.statusText(status) },

    // convert last used milliseconds date-time to timestamp string
    // return empty '' string if it is before 2022-08-17 23:45:59
    lastUsedDt (ts) {
      return (!!ts && ts > Date.UTC(2022, 7, 17, 23, 45, 59)) ? Mdf.dtToTimeStamp(new Date(ts)) : ''
    },

    toHourMinSec (totalSec) { return ((totalSec || '') ? Mdf.toHourMinSec(totalSec) : '') }, // moel run thime as hours:mm:ss

    // return count of oms users where disk usage is over the limit
    omsDiskOverCount () {
      if (!this.adminState.IsDiskUse || !Array.isArray(this.adminState?.OmsActive)) return 0

      let n = 0
      for (const om of this.adminState.OmsActive) {
        if (om.IsDiskOver) n++
      }
      return n
    },

    // convert size in MBytes to string
    sizeMbStr (sizeMb) {
      if (!sizeMb || typeof sizeMb !== typeof 1 || sizeMb < 0) return '0'
      if (this.locale) {
        return sizeMb.toLocaleString(this.locale)
      }
      return sizeMb.toFixed(0)
    },

    // update page view
    initView () {
      this.adminState = Mdf.emptyAdminState()
      this.activeRunState = {}
      this.clearActiveRunLog()
      this.clearActiveFilter()
      this.queueRunState = {}
      this.clearQueueFilter()
      this.pastRunState = {}
      this.clearPastRunLog()
      this.clearPastFilter()
      this.isPastPulling = false

      // locale for number formatting
      let lc = this.uiLang || this.$q.lang.getLocale() || ''
      if (lc) {
        try {
          const cla = Intl.getCanonicalLocales(lc)
          lc = cla?.[0] || ''
        } catch (e) {
          lc = ''
          console.warn('Error: undefined canonical locale:', e)
        }
      }
      this.locale = (typeof lc === typeof 'string') ? lc : ''

      // restart config refresh
      this.stopRefresh()
      if (this.isConfigAdminAll) {
        this.startRefresh()
      }
    },

    // refersh admin state
    onStateRefresh () {
      if (this.isRefreshPaused) {
        return
      }
      this.stateRefreshTickle = !this.stateRefreshTickle
      if (this.stateSendCount < MAX_STATE_SEND_COUNT) {
        this.doStateRefresh()
      }
    },
    refreshPauseToggle () {
      this.stateSendCount = 0
      this.stateNoDataCount = 0
      this.isRefreshPaused = !this.isRefreshPaused
    },
    startRefresh () {
      this.isRefreshPaused = false
      this.isRefreshDisabled = false
      this.stateSendCount = 0
      this.stateNoDataCount = 0
      this.stateRefreshInt = setInterval(this.onStateRefresh, ADMIN_STATE_REFRESH_TIME)
    },
    stopRefresh () {
      this.isRefreshDisabled = true
      clearInterval(this.stateRefreshInt)
    },

    // return array of servers from RunCompUsage by oms name and sumit stamp
    runCompUse (oms, stamp) {
      const rcu = []
      for (const r of this.adminState.RunCompUsage) {
        if (r.Oms === oms && r.SubmitStamp === stamp) rcu.push(r)
      }
      return rcu
    },

    // clear all active runs filters
    clearActiveFilter () {
      this.isShowActiveFilter = false
      this.omsActiveFilter = []
      this.nameActiveFilter = []
      this.digestActiveFilter = []
    },

    // return true if active run is in the active filter
    isInActiveFilter (ar) {
      if (!ar || !this.isAnyActiveFilter) {
        return true // ignore empty filter
      }
      return this.isOmsInActiveFilter(ar.Oms) && this.isNameInActiveFilter(ar.ModelName) && this.isDigestInActiveFilter(ar.ModelDigest)
    },
    // return true if oms of active run is in filter: show active runs for this oms
    isOmsInActiveFilter (oms) {
      if (!oms || !this.omsActiveFilter.length) { // ignore empty filter
        return true
      }
      return this.omsActiveFilter.findIndex(v => v === oms) >= 0
    },
    // return true if model name of active run is in filter: show active runs for this model name
    isNameInActiveFilter (name) {
      if (!name || !this.nameActiveFilter.length) { // ignore empty filter
        return true
      }
      return this.nameActiveFilter.findIndex(v => v === name) >= 0
    },
    // return true if this model digest of active run is in filter: show active runs for this model digest
    isDigestInActiveFilter (dgst) {
      if (!dgst || !this.digestActiveFilter.length) { // ignore empty filter
        return true
      }
      return this.digestActiveFilter.findIndex(v => v === dgst) >= 0
    },

    // do show or hide active job control state
    onActiveRunState (oms, stamp) {
      if (this.isActiveRunState(oms, stamp)) {
        this.clearActiveRunState(oms, stamp)
      } else {
        this.getActiveRunState(oms, stamp)
      }
    },
    // return true if active job control state is avalibale for that oms and submit stamp
    isActiveRunState (oms, stamp) {
      return !!this.findActiveRunState(oms, stamp)
    },
    // return true if active job control state is avalibale for that oms and submit stamp
    findActiveRunState (oms, stamp) {
      if (!oms || !stamp) return ''
      const key = oms + '-' + stamp
      if (!this.activeRunState?.[key]) return ''
      const rj = this.activeRunState?.[key]
      return Mdf.isRunJob(rj) ? rj : ''
    },
    // delete job active control state: set value of ['oms-stamp'] to empty '' string
    clearActiveRunState (oms, stamp) {
      if (!oms || !stamp) return
      const key = oms + '-' + stamp
      if (this.activeRunState?.[key]) this.activeRunState[key] = ''
    },

    // return active job control state as string
    viewActiveRunState (oms, stamp) {
      const rj = this.findActiveRunState(oms, stamp)
      if (!rj) return ''

      try {
        return JSON.stringify(rj, null, 2)
      } catch (e) {
        console.warn('Unable to convert model run info:', oms, stamp, e)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to convert model run info') })
        this.clearActiveRunState(oms, stamp) // clean active job control state
      }
      return '' // retrun empty on error
    },

    // copy job active control state as string to clipboard
    async toClipboardActiveRunState (oms, stamp) {
      const txt = this.viewActiveRunState(oms, stamp)
      if (!txt) {
        console.warn('empty job control state:', txt)
        this.$q.notify({ type: 'warning', message: this.$t('Unable to copy model run info to clipboard') })
        return
      }
      try {
        await copyToClipboard(txt)
        this.$q.notify({ type: 'info', message: this.$t('Copy model run info to clipboard:') + txt.length + ' ' + this.$t('characters') })
      } catch (e) {
        console.warn('Unable copy to job control state to clipboard:', e)
        this.$q.notify({ type: 'warning', message: this.$t('Unable to copy model run info to clipboard') })
      }
    },

    // clean active run log selection
    clearActiveRunLog () {
      this.activeRunLog.user = ''
      this.activeRunLog.submit = ''
      this.activeRunLog.fileName = ''
      this.activeRunLog.lines = []
    },

    // return true if run log is avalibale for that oms and submit stamp
    isActiveRunLog (oms, stamp) {
      if (!oms || !stamp) return false
      return ((this.activeRunLog.user || '') === oms && (this.activeRunLog.submit || '') === stamp)
    },

    // do show or hide active run log
    onActiveRunLog (oms, stamp) {
      if (!oms || !stamp) return

      // if this is an active run log
      const isLog = ((this.activeRunLog.user || '') === oms && (this.activeRunLog.submit || '') === stamp)
      this.clearActiveRunLog() // clear any selected run log
      if (isLog) {
        return
      }
      // else there is no currently selected run log: select new
      this.activeRunLog.user = oms
      this.activeRunLog.submit = stamp
    },

    // copy active run log string to clipboard
    async toClipboardActiveRunLog (oms, stamp) {
      if (!this.isActiveRunLog(oms, stamp)) return

      const txt = this.activeRunLog.lines.join('\n')
      if (!txt) {
        console.warn('empty active run log:', txt)
        this.$q.notify({ type: 'warning', message: this.$t('Unable to copy model run log to clipboard') })
        return
      }
      try {
        await copyToClipboard(txt)
        this.$q.notify({ type: 'info', message: this.$t('Copy model run log to clipboard:') + txt.length + ' ' + this.$t('characters') })
      } catch (e) {
        console.warn('Unable to copy model run log:', e)
        this.$q.notify({ type: 'warning', message: this.$t('Unable to copy model run log to clipboard') })
      }
    },

    // clear all queue runs filters
    clearQueueFilter () {
      this.isShowQueueFilter = false
      this.omsQueueFilter = []
      this.nameQueueFilter = []
      this.digestQueueFilter = []
    },
    // return true if active run is in the active filter
    isInQueueFilter (qr) {
      if (!qr || !this.isAnyQueueFilter) {
        return true // ignore empty filter
      }
      return this.isOmsInQueueFilter(qr.Oms) && this.isNameInQueueFilter(qr.ModelName) && this.isDigestInQueueFilter(qr.ModelDigest)
    },
    // return true if oms of queue run is in filter: show queue runs for this oms
    isOmsInQueueFilter (oms) {
      if (!oms || !this.omsQueueFilter.length) { // ignore empty filter
        return true
      }
      return this.omsQueueFilter.findIndex(v => v === oms) >= 0
    },
    // return true if model name of queue run is in filter: show queue runs for this model name
    isNameInQueueFilter (name) {
      if (!name || !this.nameQueueFilter.length) { // ignore empty filter
        return true
      }
      return this.nameQueueFilter.findIndex(v => v === name) >= 0
    },
    // return true if this model digest of queue run is in filter: show queue runs for this model digest
    isDigestInQueueFilter (dgst) {
      if (!dgst || !this.digestQueueFilter.length) { // ignore empty filter
        return true
      }
      return this.digestQueueFilter.findIndex(v => v === dgst) >= 0
    },

    // do show or hide queue job control state
    onQueueRunState (oms, stamp) {
      if (this.isQueueRunState(oms, stamp)) {
        this.clearQueueRunState(oms, stamp)
      } else {
        this.getQueueRunState(oms, stamp)
      }
    },
    // return true if queue job control state is avalibale for that oms and submit stamp
    isQueueRunState (oms, stamp) {
      return !!this.findQueueRunState(oms, stamp)
    },
    // return true if queue job control state is avalibale for that oms and submit stamp
    findQueueRunState (oms, stamp) {
      if (!oms || !stamp) return ''
      const key = oms + '-' + stamp
      if (!this.queueRunState?.[key]) return ''
      const rj = this.queueRunState?.[key]
      return Mdf.isRunJob(rj) ? rj : ''
    },
    // delete queue job control state: set value of ['oms-stamp'] to empty '' string
    clearQueueRunState (oms, stamp) {
      if (!oms || !stamp) return
      const key = oms + '-' + stamp
      if (this.queueRunState?.[key]) this.queueRunState[key] = ''
    },

    // return queue job control state as string
    viewQueueRunState (oms, stamp) {
      const rj = this.findQueueRunState(oms, stamp)
      if (!rj) return ''

      try {
        return JSON.stringify(rj, null, 2)
      } catch (e) {
        console.warn('Unable to convert model run info:', oms, stamp, e)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to convert model run info') })
        this.clearQueueRunState(oms, stamp) // clean queue job control state
      }
      return '' // retrun empty on error
    },

    // copy queue job control state as string to clipboard
    async toClipboardQueueRunState (oms, stamp) {
      const txt = this.viewQueueRunState(oms, stamp)
      if (!txt) {
        console.warn('empty job control state:', txt)
        this.$q.notify({ type: 'warning', message: this.$t('Unable to copy model run info to clipboard') })
        return
      }
      try {
        await copyToClipboard(txt)
        this.$q.notify({ type: 'info', message: this.$t('Copy model run info to clipboard:') + txt.length + ' ' + this.$t('characters') })
      } catch (e) {
        console.warn('Unable copy to job control state to clipboard:', e)
        this.$q.notify({ type: 'warning', message: this.$t('Unable to copy model run info to clipboard') })
      }
    },

    // delete from the queue: ask user confirmation to delete model run
    onStopRunConfirm (oms, stamp) {
      if (this.adminState.QueueRuns.findIndex(qr => qr.Oms === (oms || '') && qr.SubmitStamp === (stamp || '')) < 0) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to find model run') + (oms || '') + ' ' + (stamp || '') })
        return
      }
      this.stopRunTitle = oms + ' ' + this.fromUnderscoreTs(stamp)
      this.stopSubmitStamp = stamp
      this.stopOmsName = oms
      this.showStopRunTickle = !this.showStopRunTickle
    },

    // user answer is Yes to delete model run from the queue
    async onYesStopRun (title, stamp, oms) {
      if (!oms || !stamp) {
        console.warn('Unable to stop: user (oms) or submit stamp is empty', oms, stamp)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to stop: user (oms) or submit stamp is empty') })
        return
      }

      const u = this.omsUrl +
        '/api/admin-all/run/stop/queue/user/' + encodeURIComponent(oms) +
        /stamp/ + encodeURIComponent(stamp)
      try {
        await this.$axios.put(u) // ignore response on success
      } catch (e) {
        console.warn('Unable to stop model run', e)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to stop model run: ') + title })
        return // exit on error
      }

      // notify user on success, even run may not exist
      this.$q.notify({ type: 'info', message: this.$t('Stopping model run: ') + title })
    },

    // clear all past runs filters
    clearPastFilter () {
      this.isShowPastFilter = false
      this.ymPastFilter = []
      this.omsPastFilter = []
      this.namePastFilter = []
      this.digestPastFilter = []
      this.statusPastFilter = []
    },

    // return true if past run is in the filter
    isInPastFilter (pr) {
      if (!pr || !this.isAnyPastFilter) return true // ignore empty filter

      if (pr.IsDir) return this.isYearMonthInPastFilter(pr.YearMonth)

      return this.isYearMonthInPastFilter(pr.YearMonth) &&
             this.isOmsInPastFilter(pr.Oms) &&
             this.isNameInPastFilter(pr.ModelName) &&
             this.isDigestInPastFilter(pr.ModelDigest) &&
             this.isStatusInPastFilter(pr.Status)
    },

    // return true if year_month of past run is in filter: show past runs for this year_month
    isYearMonthInPastFilter (ym) {
      if (!ym || !this.ymPastFilter.length) { // ignore empty filter
        return true
      }
      return this.ymPastFilter.findIndex(v => v === ym) >= 0
    },
    // return true if oms of past run is in filter: show past runs for this oms
    isOmsInPastFilter (oms) {
      if (!oms || !this.omsPastFilter.length) { // ignore empty filter
        return true
      }
      return this.omsPastFilter.findIndex(v => v === oms) >= 0
    },
    // return true if model name of past run is in filter: show past runs for this model name
    isNameInPastFilter (name) {
      if (!name || !this.namePastFilter.length) { // ignore empty filter
        return true
      }
      return this.namePastFilter.findIndex(v => v === name) >= 0
    },
    // return true if model digest of past run is in filter: show past runs for this model digest
    isDigestInPastFilter (dgst) {
      if (!dgst || !this.digestPastFilter.length) { // ignore empty filter
        return true
      }
      return this.digestPastFilter.findIndex(v => v === dgst) >= 0
    },
    // return true if run status of past run is in filter: show past runs for this run status
    isStatusInPastFilter (st) {
      if (!st || !this.statusPastFilter.length) { // ignore empty filter
        return true
      }
      return this.statusPastFilter.findIndex(v => v === st) >= 0
    },

    // do show or hide past job control state
    onPastRunState (ym, oms, stamp) {
      if (this.isPastRunState(ym, oms, stamp)) {
        this.clearPastRunState(ym, oms, stamp)
      } else {
        this.getPastRunState(ym, oms, stamp)
      }
    },
    // return true if past job control state is avalibale for that year_month, oms and submit stamp
    isPastRunState (ym, oms, stamp) {
      return !!this.findPastRunState(ym, oms, stamp)
    },
    // return true if past job control state is avalibale for that year_month, oms and submit stamp
    findPastRunState (ym, oms, stamp) {
      if (!ym || !oms || !stamp) return ''
      const key = ym + '-' + oms + '-' + stamp
      if (!this.pastRunState?.[key]) return ''
      const rj = this.pastRunState?.[key]
      return Mdf.isRunJob(rj) ? rj : ''
    },
    // delete job past control state: set value of ['year_month-oms-stamp'] to empty '' string
    clearPastRunState (ym, oms, stamp) {
      if (!ym || !oms || !stamp) return
      const key = ym + '-' + oms + '-' + stamp
      if (this.pastRunState?.[key]) this.pastRunState[key] = ''
    },

    // return past job control state as string
    viewPastRunState (ym, oms, stamp) {
      const rj = this.findPastRunState(ym, oms, stamp)
      if (!rj) return ''

      try {
        return JSON.stringify(rj, null, 2)
      } catch (e) {
        console.warn('Unable to convert model run info:', ym, oms, stamp, e)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to convert model run info') })
        this.clearPastRunState(oms, stamp) // clean past job control state
      }
      return '' // retrun empty on error
    },

    // copy job past control state as string to clipboard
    async toClipboardPastRunState (ym, oms, stamp) {
      const txt = this.viewPastRunState(ym, oms, stamp)
      if (!txt) {
        console.warn('empty job control state:', txt)
        this.$q.notify({ type: 'warning', message: this.$t('Unable to copy model run info to clipboard') })
        return
      }
      try {
        await copyToClipboard(txt)
        this.$q.notify({ type: 'info', message: this.$t('Copy model run info to clipboard:') + txt.length + ' ' + this.$t('characters') })
      } catch (e) {
        console.warn('Unable copy to job control state to clipboard:', e)
        this.$q.notify({ type: 'warning', message: this.$t('Unable to copy model run info to clipboard') })
      }
    },

    // receive global admin server state
    async doStateRefresh () {
      this.stateSendCount++

      let u = this.omsUrl + '/api/admin-all/state'
      try {
        // send request to the server
        const response = await this.$axios.get(u)
        const std = response?.data
        if (Mdf.isAdminState(std)) { // if response is a admin state
          this.stateSendCount = 0
          this.stateNoDataCount = 0
          this.adminState = std
        } else {
          this.stateNoDataCount++
        }
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        if (this.stateNoDataCount++ <= MAX_NO_STATE_COUNT) {
          console.warn('Server offline or state retrieval failed.', em)
        }
      }
      if (this.stateNoDataCount > MAX_NO_STATE_COUNT) {
        this.isRefreshPaused = true
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or state retrieval failed.') })
        return
      }

      // update active filter options
      let omsOpts = {}
      let nameOpts = {}
      let digestOpts = {}

      for (const ar of this.adminState.ActiveRuns) {
        omsOpts[ar.Oms] = true
        nameOpts[ar.ModelName] = true
        digestOpts[ar.ModelDigest] = true
      }
      this.omsActiveOpts = Object.freeze(Object.keys(omsOpts).sort())
      this.nameActiveOpts = Object.freeze(Object.keys(nameOpts).sort())
      this.digestActiveOpts = Object.freeze(Object.keys(digestOpts).sort())

      // update queue filter options
      omsOpts = {}
      nameOpts = {}
      digestOpts = {}

      for (const qr of this.adminState.QueueRuns) {
        omsOpts[qr.Oms] = true
        nameOpts[qr.ModelName] = true
        digestOpts[qr.ModelDigest] = true
      }
      this.omsQueueOpts = Object.freeze(Object.keys(omsOpts).sort())
      this.nameQueueOpts = Object.freeze(Object.keys(nameOpts).sort())
      this.digestQueueOpts = Object.freeze(Object.keys(digestOpts).sort())

      // clear active run state and queue run state
      const ak = Object.keys(this.activeRunState)
      for (const key of ak) {
        if (this.adminState.ActiveRuns.findIndex(ar => ar.Oms + '-' + ar.SubmitStamp === key) < 0) {
          this.activeRunState[key] = ''
        }
      }
      const qk = Object.keys(this.queueRunState)
      for (const key of qk) {
        if (this.adminState.QueueRuns.findIndex(qr => qr.Oms + '-' + qr.SubmitStamp === key) < 0) {
          this.queueRunState[key] = ''
        }
      }

      // check if active run log selected to view and it still active
      if (!(this.activeRunLog.user || '') || !(this.activeRunLog.submit || '')) {
        return
      }
      if (this.adminState.ActiveRuns.findIndex(ar => ar.Oms === this.activeRunLog.user && ar.SubmitStamp === this.activeRunLog.submit) < 0) {
        this.clearActiveRunLog()
        return
      }

      // update active run log
      let isOk = false
      let ml = []
      const oms = this.activeRunLog.user || ''
      const stamp = this.activeRunLog.submit || ''
      u = this.omsUrl +
        '/api/admin-all/job/active/user/' + encodeURIComponent(oms) + /stamp/ + encodeURIComponent(stamp) + '/log'
      try {
        // send request to the server
        const response = await this.$axios.get(u)
        ml = response.data
        isOk = Array.isArray(ml) // if response is run log file name and lines
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Unable to get model run log:', oms, stamp, em)
      }

      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to get model run log:') + ' ' + oms + ' ' + stamp })
        this.clearActiveRunLog()
        return
      }

      // else: update model run log file name and lines
      this.activeRunLog.fileName = (ml.length > 0 && typeof ml[0] === typeof 'string') ? ml[0] : ''
      this.activeRunLog.lines = (ml.length > 1) ? ml.slice(1) : []
    },

    // pause or resume all model runs queue
    async doQueuePauseResume (isPause) {
      const u = this.omsUrl + '/api/admin-all/jobs-pause/' + (isPause ? 'true' : 'false')
      try {
        await this.$axios.post(u) // ignore response on success
      } catch (e) {
        console.warn('Unable to pause or resume all model run queues', isPause, e)
        this.$q.notify({ type: 'negative', message: this.$t(isPause ? 'Unable to pause all model run queues' : 'Unable to resume all model run queues') })
        return // exit on error
      }

      // notify user on success, actual service state update dealyed
      this.$q.notify({ type: 'info', message: this.$t(isPause ? 'Pausing all model run queues...' : 'Resuming all model run queues...') })
    },

    // get active job control state
    async getActiveRunState (oms, stamp) {
      if (!oms || !stamp) {
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) oms user  or submit stamp:') + ' ' + (oms || '') + ' ' + (stamp || '') })
        return
      }

      // update active run state
      let isOk = false
      const u = this.omsUrl +
        '/api/admin-all/job/active/user/' + encodeURIComponent(oms) + /stamp/ + encodeURIComponent(stamp) + '/state'
      try {
        // send request to the server
        const response = await this.$axios.get(u)
        const rj = response.data
        isOk = Mdf.isNotEmptyRunJob(rj) // if response is job control state
        this.activeRunState[oms + '-' + stamp] = rj
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Unable to retrieve model run info:', oms, stamp, em)
      }

      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to retrieve model run info:') + ' ' + oms + ' ' + stamp })
      }
    },

    // get queue job control state
    async getQueueRunState (oms, stamp) {
      if (!oms || !stamp) {
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) oms user  or submit stamp:') + ' ' + (oms || '') + ' ' + (stamp || '') })
        return
      }

      // update queue run state
      let isOk = false
      const u = this.omsUrl +
        '/api/admin-all/job/queue/user/' + encodeURIComponent(oms) + /stamp/ + encodeURIComponent(stamp) + '/state'
      try {
        // send request to the server
        const response = await this.$axios.get(u)
        const rj = response.data
        isOk = Mdf.isNotEmptyRunJob(rj) // if response is job control state
        this.queueRunState[oms + '-' + stamp] = rj
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Unable to retrieve model run info:', oms, stamp, em)
      }

      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to retrieve model run info:') + ' ' + oms + ' ' + stamp })
      }
    },

    // update shadow past history
    async onRereshPast () {
      let isOk = false
      this.isPastPulling = true
      let pd = []
      this.pastRuns = []

      const u = this.omsUrl + '/api/admin-all/job/past/file-tree'
      try {
        // send request to the server
        const response = await this.$axios.get(u)
        isOk = Mdf.isPastHistory(response?.data) // if response is past shadow history array
        if (isOk) pd = response.data
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Unable to retrieve model runs history', em)
      }

      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to retrieve model runs history') })
      } else {
        // sort files list in reverse order of year-month and submit stamp, normal ascending order of oms user names
        pd.sort((left, right) => {
          // directory  and file names names in year-month reverse order
          switch (true) {
            case (left.YearMonth < right.YearMonth): return 1
            case (left.YearMonth > right.YearMonth): return -1
          }
          // year-month directory before file names
          switch (true) {
            case (left.IsDir && !right.IsDir): return -1
            case (!left.IsDir && right.IsDir): return 1
          }
          // sort submit stamp in  reverse order
          switch (true) {
            case (left.SubmitStamp < right.SubmitStamp): return 1
            case (left.SubmitStamp > right.SubmitStamp): return -1
          }
          // sort oms in "normal" ascending order
          switch (true) {
            case (left.Oms < right.Oms): return -1
            case (left.Oms > right.Oms): return 1
          }
          return 0
        })

        // update filter options
        const ymOpts = {}
        const omsOpts = {}
        const nameOpts = {}
        const digestOpts = {}

        for (const pr of pd) {
          ymOpts[pr.YearMonth] = true
          omsOpts[pr.Oms] = true
          nameOpts[pr.ModelName] = true
          digestOpts[pr.ModelDigest] = true
        }
        this.ymPastOpts = Object.freeze(Object.keys(ymOpts).sort())
        this.omsPastOpts = Object.freeze(Object.keys(omsOpts).sort())
        this.namePastOpts = Object.freeze(Object.keys(nameOpts).sort())
        this.digestPastOpts = Object.freeze(Object.keys(digestOpts).sort())

        this.pastRuns = Object.freeze(pd)
        this.isShowPastRuns = true // show results
      }
      this.isPastPulling = false
      this.pastRefreshTs = Mdf.dtToTimeStamp(new Date())
    },

    // get past job control state
    async getPastRunState (ym, oms, stamp) {
      if (!ym || !oms || !stamp) {
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) year-month, oms user  or submit stamp:') + ' ' + (ym || '') + ' ' + (oms || '') + ' ' + (stamp || '') })
        return
      }

      // update past run state
      let isOk = false
      const u = this.omsUrl +
        '/api/admin-all/job/past' +
        '/folder/' + encodeURIComponent(ym) +
        '/user/' + encodeURIComponent(oms) +
        '/stamp/' + encodeURIComponent(stamp) + '/state'
      try {
        // send request to the server
        const response = await this.$axios.get(u)
        const rj = response.data
        isOk = Mdf.isNotEmptyRunJob(rj) // if response is job control state
        this.pastRunState[ym + '-' + oms + '-' + stamp] = rj
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Unable to retrieve model run info:', ym, oms, stamp, em)
      }

      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to retrieve model run info:') + ym + ' ' + ' ' + oms + ' ' + stamp })
      }
    },

    // clean past run log selection
    clearPastRunLog () {
      this.pastRunLog.yearMonth = ''
      this.pastRunLog.user = ''
      this.pastRunLog.submit = ''
      this.pastRunLog.fileName = ''
      this.pastRunLog.lines = []
    },

    // return true if run log is avalibale for that oms and submit stamp
    isPastRunLog (ym, oms, stamp) {
      if (!ym || !oms || !stamp) return false
      return ((this.pastRunLog.yearMonth || '') === ym && (this.pastRunLog.user || '') === oms && (this.pastRunLog.submit || '') === stamp)
    },

    // copy past run log string to clipboard
    async toClipboardPastRunLog (ym, oms, stamp) {
      if (!this.isPastRunLog(ym, oms, stamp)) return

      const txt = this.pastRunLog.lines.join('\n')
      if (!txt) {
        console.warn('empty past run log:', txt)
        this.$q.notify({ type: 'warning', message: this.$t('Unable to copy model run log to clipboard') })
        return
      }
      try {
        await copyToClipboard(txt)
        this.$q.notify({ type: 'info', message: this.$t('Copy model run log to clipboard:') + txt.length + ' ' + this.$t('characters') })
      } catch (e) {
        console.warn('Unable to copy model run log:', e)
        this.$q.notify({ type: 'warning', message: this.$t('Unable to copy model run log to clipboard') })
      }
    },

    // do show or hide past run log
    onPastRunLog (ym, oms, stamp) {
      if (!ym || !oms || !stamp) return

      // if this is an past run log
      if ((this.pastRunLog.yearMonth || '') === ym && (this.pastRunLog.user || '') === oms && (this.pastRunLog.submit || '') === stamp) {
        this.clearPastRunLog() // clear any selected run log
        return
      }
      // else there is no currently selected run log:
      // retrive log file content
      this.getPastRunLog(ym, oms, stamp)
    },

    // get past job log file content
    async getPastRunLog (ym, oms, stamp) {
      if (!ym || !oms || !stamp) {
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) year-month, oms user  or submit stamp:') + ' ' + (ym || '') + ' ' + (oms || '') + ' ' + (stamp || '') })
        return
      }

      // get past run log
      let isOk = false
      let ml = []
      const u = this.omsUrl +
        '/api/admin-all/job/past' +
        '/folder/' + encodeURIComponent(ym) +
        '/user/' + encodeURIComponent(oms) +
        '/stamp/' + encodeURIComponent(stamp) + '/log'
      try {
        // send request to the server
        const response = await this.$axios.get(u)
        ml = response.data
        isOk = Array.isArray(ml) // if response is run log file name and lines
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Unable to get model run log:', ym, oms, stamp, em)
      }

      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to get model run log:') + ym + ' ' + ' ' + oms + ' ' + stamp })
        return
      }
      // else: update model run log file name and lines
      this.pastRunLog.yearMonth = ym
      this.pastRunLog.user = oms
      this.pastRunLog.submit = stamp

      this.pastRunLog.fileName = (ml.length > 0 && typeof ml[0] === typeof 'string') ? ml[0] : ''
      this.pastRunLog.lines = (ml.length > 1) ? ml.slice(1) : []
    }
  },

  mounted () {
    this.initView()
  },
  beforeUnmount () {
    this.stopRefresh()
  }
}
