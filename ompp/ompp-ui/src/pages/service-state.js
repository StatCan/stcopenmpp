import { mapState } from 'pinia'
import { useServerStateStore } from '../stores/server-state'
import * as Mdf from 'src/model-common'
import JobInfoCard from 'components/JobInfoCard.vue'
import DeleteConfirmDialog from 'components/DeleteConfirmDialog.vue'

/* eslint-disable no-multi-spaces */
const STATE_REFRESH_TIME = 1601       // msec, service state refresh interval
const MAX_STATE_SEND_COUNT = 5        // max request to send without response
const MAX_NO_STATE_COUNT = 4          // max invalid response count
/* eslint-enable no-multi-spaces */

export default {
  name: 'ServiceState',
  components: { JobInfoCard, DeleteConfirmDialog },

  props: {
    refreshTickle: { type: Boolean, default: false }
  },

  data () {
    return {
      srvState: Mdf.emptyServiceState(),
      activeJobs: {},
      queueJobs: {},
      historyJobs: {},
      doneHistory: [], // successfully completed jobs from history
      otherHistory: [], // jobs from history where run status is not 'success'
      isActiveShow: false,
      isQueueShow: false,
      isDoneHistoryShow: false,
      isOtherHistoryShow: false,
      isShowServers: false,
      isRefreshPaused: false,
      isRefreshDisabled: false,
      stateRefreshTickle: 0,
      stateSendCount: 0,
      stateNoDataCount: 0,
      stateRefreshInt: '',
      showStopRunTickle: false,
      stopRunTitle: '',
      stopSubmitStamp: '',
      stopModelDigest: '',
      showDeleteHistoryTickle: false,
      deleteHistoryTitle: '',
      deleteSubmitStamp: '',
      showDeleteAllHistoryTickle: false,
      deleteAllHistoryKind: '',
      deleteAllHistoryTitle: ''
    }
  },

  computed: {
    isJobControl () { return !!this.serverConfig.IsJobControl },
    isMicrodata () { return !!this.serverConfig.AllowMicrodata },

    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl',
      serverConfig: 'config'
    })
  },

  watch: {
    refreshTickle () { this.initView() },
    isJobControl () { this.initView() }
  },

  methods: {
    isUnderscoreTs (ts) { return Mdf.isUnderscoreTimeStamp(ts) },
    fromUnderscoreTs (ts) { return Mdf.isUnderscoreTimeStamp(ts) ? Mdf.fromUnderscoreTimeStamp(ts) : ts },
    isSuccess (status) { return status === 'success' },
    isInProgress (status) { return status === 'progress' || status === 'init' || status === 'wait' },
    runStatusDescr (status) { return Mdf.statusText(status) },
    isActiveJob (stamp) { return !!stamp && Mdf.isNotEmptyJobItem(this.activeJobs[stamp]) },
    isQueueJob (stamp) { return !!stamp && Mdf.isNotEmptyJobItem(this.queueJobs[stamp]) },
    isHistoryJob (stamp) { return !!stamp && Mdf.isNotEmptyJobItem(this.historyJobs[stamp]) },
    getRunTitle (jobItem) { return Mdf.getJobRunTitle(jobItem) },
    getHistoryTitle (hJob) { return hJob?.RunTitle || '' },

    // return true if job is first in the queue
    isTopQueue (stamp) {
      return this.srvState.Queue.findIndex((jc) => jc.SubmitStamp === stamp) <= 0
    },
    // return true if job is last in the queue
    isBottomQueue (stamp) {
      return this.srvState.Queue.findIndex((jc) => jc.SubmitStamp === stamp) >= this.srvState.Queue.length - 1
    },
    // convert last used milliseconds date-time to timestamp string
    // return empty '' string if it is before 2022-08-17 23:45:59
    lastUsedDt (ts) {
      return (!!ts && ts > Date.UTC(2022, 7, 17, 23, 45, 59)) ? Mdf.dtToTimeStamp(new Date(ts)) : ''
    },

    // update page view
    initView () {
      this.srvState = Mdf.emptyServiceState()
      this.stopRefresh()
      if (this.isJobControl) {
        this.startRefresh()
      }
      this.isActiveShow = true
      this.isQueueShow = true
      this.isPauseQueue = !this.srvState.IsQueuePaused
      this.isDoneHistoryShow = false
      this.isOtherHistoryShow = true
      this.activeJobs = {}
      this.queueJobs = {}
      this.historyJobs = {}
      this.doneHistory = []
      this.otherHistory = []
    },

    // refersh service state
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
      this.stateRefreshInt = setInterval(this.onStateRefresh, STATE_REFRESH_TIME)
    },
    stopRefresh () {
      this.isRefreshDisabled = true
      clearInterval(this.stateRefreshInt)
    },

    // show or hide active job item
    onActiveShow (stamp) {
      if (stamp) this.getJobState('active', stamp, false, '')
    },
    onActiveHide (stamp) {
      if (stamp) this.activeJobs[stamp] = Mdf.emptyJobItem(stamp)
    },
    // show or hide queue job item
    onQueueShow (stamp) {
      this.getJobState('queue', stamp, false, '')
    },
    onQueueHide (stamp) {
      if (stamp) this.queueJobs[stamp] = Mdf.emptyJobItem(stamp)
    },
    // show or hide job history item
    onHistoryShow (stamp) {
      this.getJobState('history', stamp, false, '')
    },
    onHistoryHide (stamp) {
      if (stamp) this.historyJobs[stamp] = Mdf.emptyJobItem(stamp)
    },

    // add new job control items into active, queue and history jobs
    // remove state of a job which is no longer exist in active or queue or history
    updateJobsState () {
      // remove state of a job which is no longer exist in active or queue or history
      for (const stamp in this.activeJobs) {
        if (this.srvState.Active.findIndex((jc) => jc.SubmitStamp === stamp) < 0) this.activeJobs[stamp] = ''
      }
      for (const stamp in this.queueJobs) {
        if (this.srvState.Queue.findIndex((jc) => jc.SubmitStamp === stamp) < 0) this.queueJobs[stamp] = ''
      }
      for (const stamp in this.historyJobs) {
        if (this.srvState.History.findIndex((jc) => jc.SubmitStamp === stamp) < 0) this.historyJobs[stamp] = ''
      }

      // add new job control items into active, queue and history jobs
      for (const aj of this.srvState.Active) {
        if ((this.activeJobs[aj.SubmitStamp] || '') === '') this.activeJobs[aj.SubmitStamp] = Mdf.isJobItem(aj) ? aj : Mdf.emptyJobItem(aj.SubmitStamp)
      }
      for (const qj of this.srvState.Queue) {
        if ((this.queueJobs[qj.SubmitStamp] || '') === '') this.queueJobs[qj.SubmitStamp] = Mdf.isJobItem(qj) ? qj : Mdf.emptyJobItem(qj.SubmitStamp)
      }
      for (const hj of this.srvState.History) {
        if ((this.historyJobs[hj.SubmitStamp] || '') === '') this.historyJobs[hj.SubmitStamp] = Mdf.isJobItem(hj) ? hj : Mdf.emptyJobItem(hj.SubmitStamp)
      }

      // split history jobs into success and other status
      const dh = []
      const th = []
      for (const hj of this.srvState.History) {
        if (this.isSuccess(hj.JobStatus)) {
          dh.push(hj)
        } else {
          th.push(hj)
        }
      }
      this.doneHistory = dh
      this.otherHistory = th
    },

    // receive server configuration, including configuration of model catalog and run catalog
    async doStateRefresh () {
      this.stateSendCount++

      const u = this.omsUrl + '/api/service/state'
      try {
        // send request to the server
        const response = await this.$axios.get(u)
        const std = response?.data
        if (Mdf.isServiceState(std)) { // if response is a server state
          this.stateSendCount = 0
          this.stateNoDataCount = 0
          //
          std.History.reverse() // display history in reverse order: latest runs first
          this.srvState = std
          this.updateJobsState()
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

      // refresh active jobs state
      if (this.isActiveShow) {
        for (const stamp in this.activeJobs) {
          if (Mdf.isNotEmptyJobItem(this.activeJobs[stamp])) this.getJobState('active', stamp, false, '')
        }
      }
    },

    // resubmit model run
    onRunAgain (stamp, mName, title) {
      const mrt = (mName || '') + (title ? ': ' + title : '')
      this.$q.notify({ type: 'info', message: this.$t('Run again') + ' ' + mrt })

      this.getJobState('history', stamp, true, mrt)
    },

    // get active or queue or history job item by submission stamp
    async getJobState (kind, stamp, isReRun, mRunTitle) {
      if (!kind || typeof kind !== typeof 'string' || (kind !== 'active' && kind !== 'queue' && kind !== 'history')) {
        console.warn('Invalid argument, it must be: active, queue or history:', kind)
        this.$q.notify({ type: 'negative', message: this.$t('Invalid argument, it must be: active, queue or history') })
        return
      }
      if (!stamp || typeof stamp !== typeof 'string' || (stamp || '') === '') {
        console.warn('Invalid (empty) submission stamp:', stamp)
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) submission stamp') })
        return
      }

      let isOk = false
      let jc = {}
      const u = this.omsUrl + '/api/service/job/' + kind + '/' + encodeURIComponent(stamp)
      try {
        // send request to the server
        const response = await this.$axios.get(u)
        jc = response.data
        isOk = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Unable to get job control state:', kind, stamp, em)
      }

      if (isOk) {
        switch (kind) {
          case 'active':
            this.activeJobs[stamp] = Mdf.isJobItem(jc) ? jc : Mdf.emptyJobItem(stamp)
            break
          case 'queue':
            this.queueJobs[stamp] = Mdf.isJobItem(jc) ? jc : Mdf.emptyJobItem(stamp)
            break
          case 'history':
            // show model run history job details or re-submit model run
            if (!isReRun) {
              this.historyJobs[stamp] = Mdf.isJobItem(jc) ? jc : Mdf.emptyJobItem(stamp)
            } else {
              if (!Mdf.isJobItem(jc)) {
                this.$q.notify({ type: 'negative', message: this.$t('Unable to find model run' + ' ' + mRunTitle) })
                return
              }
              // re-submit model run request, clear run stamp
              const rReq = Mdf.runRequestFromJob(jc)
              rReq.RunStamp = ''
              this.$router.push({ name: 'new-model-run', params: { digest: jc.ModelDigest, runRequest: rReq } })
            }
            break
          default:
            isOk = false
        }
      }
      if (!isOk) {
        console.warn('Unable to set job control state:', kind, stamp)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to retrieve model run state: ') + kind + ' ' + stamp })
      }
    },

    // stop model run: ask user confirmation to kill model run
    onStopJobConfirm (stamp, mDigest, mName) {
      if (!mDigest || !stamp) {
        const s = (mName || mDigest || '') + ' ' + (stamp || '') + ' '
        this.$q.notify({ type: 'negative', message: this.$t('Unable to find active model run: ') + s })
        return
      }
      this.stopRunTitle = mName + ' ' + this.fromUnderscoreTs(stamp)
      this.stopSubmitStamp = stamp
      this.stopModelDigest = mDigest
      this.showStopRunTickle = !this.showStopRunTickle
    },

    // user answer is Yes to stop model run
    async onYesStopRun (title, stamp, mDgst) {
      if (!mDgst || !stamp) {
        console.warn('Unable to stop: model digest or submit stamp is empty', mDgst, stamp)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to stop: model digest or submit stamp is empty') })
        return
      }

      const u = this.omsUrl +
        '/api/run/stop/model/' + encodeURIComponent(mDgst) +
        '/stamp/' + encodeURIComponent(stamp)
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

    // move job queue to specified postion
    async onJobMove (stamp, pos, mName, title) {
      const mt = (mName || '') + ' ' + (title || this.fromUnderscoreTs(stamp) || '')
      if (!stamp || typeof pos !== typeof 1) {
        console.warn('Unable to move model run', pos, stamp)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to move model run: ') + mt })
        return
      }

      const u = this.omsUrl + '/api/service/job/move/' + (pos.toString()) + '/' + encodeURIComponent(stamp)
      try {
        await this.$axios.put(u) // ignore response on success
      } catch (e) {
        console.warn('Unable to move model run', e)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to move model run: ') + mt })
        return // exit on error
      }

      // notify user on success, even run may not exist
      this.$q.notify({ type: 'info', message: this.$t('Moving model run: ') + title })
    },

    // pause or resume model runs queue
    async doQueuePauseResume (isPause) {
      const u = this.omsUrl + '/api/admin/jobs-pause/' + (isPause ? 'true' : 'false')
      try {
        await this.$axios.post(u) // ignore response on success
      } catch (e) {
        console.warn('Unable to pause or resume model run queue', isPause, e)
        this.$q.notify({ type: 'negative', message: this.$t(isPause ? 'Unable to pause model run queue' : 'Unable to resume model run queue') })
        return // exit on error
      }

      // notify user on success, actual service state update dealyed
      this.$q.notify({ type: 'info', message: this.$t(isPause ? 'Pausing model run queue...' : 'Resuming model run queue...') })
    },

    // delete all jobs history: ask user confirmation
    onDeleteAllHistoryConfirm (isSuccess) {
      this.deleteAllHistoryTitle = this.$t('Delete all history') + ' [' + (isSuccess ? this.doneHistory.length.toLocaleString() : this.otherHistory.length.toLocaleString()) + '] ?'
      this.deleteAllHistoryKind = isSuccess ? 'success' : 'not-success'
      this.showDeleteAllHistoryTickle = !this.showDeleteAllHistoryTickle
    },

    // user answer is Yes to delete all job history
    async onYesDeleteAllJobHistory (itemName, itemId, kind) {
      if (!kind || (kind !== 'success' && kind !== 'not-success')) {
        console.warn('Unable to delete all jobs history, invalid kind:', kind)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to delete all jobs history (invalid kind)') })
        return
      }
      const isSuccess = kind === 'success'
      const nLen = isSuccess ? this.doneHistory.length : this.otherHistory.length

      const u = this.omsUrl +
        '/api/service/job/delete/history-all/' + (isSuccess ? 'true' : 'false')
      try {
        await this.$axios.delete(u) // ignore response on success
      } catch (e) {
        console.warn('Unable to delete all jobs history', e)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to delete all jobs history') })
        return // exit on error
      }

      // notify user on success
      this.$q.notify({ type: 'info', message: this.$t('Deleting all jobs history: ') + nLen.toLocaleString() })
    },

    // delete job history: ask user confirmation
    onDeleteJobHistoryConfirm (stamp, mName, title) {
      this.deleteHistoryTitle = mName + ' ' + (title || this.fromUnderscoreTs(stamp))
      this.deleteSubmitStamp = stamp
      this.showDeleteHistoryTickle = !this.showDeleteHistoryTickle
    },

    // user answer is Yes to delete job history
    async onYesDeleteJobHistory (title, stamp) {
      if (!stamp) {
        console.warn('Unable to delete job history: submit stamp is empty', stamp)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to delete job history: submit stamp is empty') })
        return
      }

      const u = this.omsUrl +
        '/api/service/job/delete/history/' + encodeURIComponent(stamp)
      try {
        await this.$axios.delete(u) // ignore response on success
      } catch (e) {
        console.warn('Unable to delete job history', e)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to delete job history: ') + title })
        return // exit on error
      }

      // remove from the page view
      // it may briefly show up again because it is not actually deleted from server History[] until next files refresh
      this.historyJobs[stamp] = ''
      this.doneHistory = this.doneHistory.filter(hj => hj?.SubmitStamp !== stamp)
      this.otherHistory = this.otherHistory.filter(hj => hj?.SubmitStamp !== stamp)

      // notify user on success
      this.$q.notify({ type: 'info', message: this.$t('Deleting job history: ') + title })
    }
  },

  mounted () {
    this.initView()
  },
  beforeUnmount () {
    this.stopRefresh()
  }
}
