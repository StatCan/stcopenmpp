<template>
<div class="text-body1">

  <q-card
    class="q-ma-sm"
    >
    <q-card-section>

      <div
        class="row reverse-wrap items-center"
        >

        <q-btn
          v-if="!isRefreshCompleted"
          @click="refreshPauseToggle"
          flat
          dense
          class="col-auto bg-primary text-white rounded-borders q-mr-xs"
          :icon="!isRefreshPaused ? (((refreshCount % 2) === 1) ? 'mdi-autorenew' : 'mdi-sync') : 'mdi-play-circle-outline'"
          :title="!isRefreshPaused ? $t('Pause auto refresh') : $t('Auto refresh model run log')"
          />
        <q-btn
          v-else
          disable
          flat
          dense
          class="col-auto bg-primary text-white rounded-borders q-mr-xs"
          icon="mdi-autorenew"
          :title="$t('Refresh')"
          />

        <q-btn
          v-if="!runState.IsFinal && (runState.RunStamp || '')"
          @click="showStopRunTickle = !showStopRunTickle"
          flat
          dense
          class="col-auto bg-primary text-white rounded-borders q-mr-xs"
          icon="mdi-alert-octagon-outline"
          :title="$t('Stop model run')"
          />
        <q-btn
          v-else
          disable
          flat
          dense
          class="col-auto bg-primary text-white rounded-borders q-mr-xs"
          icon="mdi-alert-octagon-outline"
          :title="$t('Stop model run')"
          />

        <div
          class="col-auto"
          >
          <span>{{ $t('Progress of') }}: <span :title="$t('Run Stamp')" class="mono om-text-secondary">{{ stamp }}</span></span><br />
          <span class="mono om-text-descr">{{ lastProgressTimeStamp }}</span>
        </div>

      </div>

    </q-card-section>
  </q-card>

  <q-card
    v-for="rsp in runStatusProgress" :key="rsp.RunDigest + '-' + (rsp.Status || 'st')"
    class="q-ma-sm"
    >
    <q-card-section>

    <run-bar
      :model-digest="digest"
      :run-digest="rsp.RunDigest"
      @run-info-click="doShowRunNote"
      class="q-mb-sm"
      >
    </run-bar>

    <table v-if="rsp.UpdateDateTime" class="om-p-table">
      <thead>
        <tr>
          <th class="om-p-head-center text-weight-medium">{{ rsp.SubCompleted > 0 ? $t('Completed') : $t('Sub-values') }}</th>
          <th class="om-p-head-center text-weight-medium">{{ $t('Status') }}</th>
          <th class="om-p-head-center text-weight-medium">{{ $t('Updated') }}</th>
          <th class="om-p-head-center text-weight-medium">{{ $t('Value Digest') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td class="om-p-cell-right"><span v-if="rsp.SubCompleted > 0">{{ rsp.SubCompleted }} / </span>{{ rsp.SubCount }}</td>
          <td class="om-p-cell-left">{{ $t(runStatusDescr(rsp.Status)) }}</td>
          <td class="om-p-cell-left">{{ rsp.UpdateDateTime }}</td>
          <td class="om-p-cell-left">{{ rsp.ValueDigest }}</td>
        </tr>
        <tr v-if="rsp.Progress">
          <td class="om-p-head-center text-weight-medium">{{ $t('Sub-value') }}</td>
          <td class="om-p-head-center text-weight-medium">{{ $t('Status') }}</td>
          <td class="om-p-head-center text-weight-medium">{{ $t('Updated') }}</td>
          <td class="om-p-head-center text-weight-medium">{{ $t('Progress') }}</td>
        </tr>
        <tr v-for="pi in rsp.Progress" :key="(pi.Status || 'st') + '-' + (pi.SubId || 'sub') + '-' + (pi.UpdateDateTime || 'upt')">
          <td class="om-p-cell-right">{{ pi.SubId }}</td>
          <td class="om-p-cell-left">{{ $t(runStatusDescr(pi.Status)) }}</td>
          <td class="om-p-cell-left">{{ pi.UpdateDateTime }}</td>
          <td class="om-p-cell-right">{{ pi.Count }}% ({{ pi.Value }})</td>
        </tr>
      </tbody>
    </table>

    </q-card-section>
  </q-card>

  <q-card
    v-if="runState.UpdateDateTime"
    class="q-ma-sm"
    >
    <q-card-section>
      <span class="mono">{{runState.UpdateDateTime}}</span>
      <span v-if="runState.LogFileName" class="mono"><i>[{{runState.LogFileName}}]</i></span>
      <div>
        <pre>{{logLines.join('\n')}}</pre>
      </div>
    </q-card-section>
  </q-card>

  <refresh-run-log
    :model-digest="digest"
    :stamp="stamp"
    :refresh-tickle="isLogRefresh"
    :start="logStart"
    :count="logCount"
    @done="doneRunLogRefresh"
    @wait="()=>{}"
    >
  </refresh-run-log>
  <refresh-run-progress
    v-if="(runState.RunStamp || stamp) !== ''"
    :model-digest="digest"
    :run-stamp="(runState.RunStamp || stamp)"
    :refresh-tickle="isProgressRefresh"
    @done="doneRunProgressRefresh"
    @wait="()=>{}"
    >
  </refresh-run-progress>

  <run-info-dialog :show-tickle="runInfoTickle" :model-digest="digest" :run-digest="runInfoDigest"></run-info-dialog>

  <confirm-dialog
    @confirm-yes="onYesStopRun"
    :show-tickle="showStopRunTickle"
    :item-name="runState.TaskRunName || runState.RunName || runState.RunStamp"
    :dialog-title="$t('Stop model run?')"
    :body-text="$t('Stop')"
    :icon-name="'mdi-alert-octagon'"
    >
  </confirm-dialog>

</div>
</template>

<script>
import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import * as Mdf from 'src/model-common'
import RefreshRunLog from 'components/RefreshRunLog.vue'
import RefreshRunProgress from 'components/RefreshRunProgress.vue'
import RunBar from 'components/RunBar.vue'
import RunInfoDialog from 'components/RunInfoDialog.vue'
import ConfirmDialog from 'components/ConfirmDialog.vue'

/* eslint-disable no-multi-spaces */
const MAX_EMPTY_LOG_COUNT = 5          // pause log refresh if empty response exceed this count (5 = 5 seconds)
const MAX_EMPTY_PROGRESS_COUNT = 5     // pause progress refresh if empty response exceed this count (5 = 5 seconds)
const MAX_SEND_COUNT = 4               // max request to send without response
const RUN_PROGRESS_REFRESH_TIME = 1000 // msec, run progress refresh time
const RUN_PROGRESS_SUB_RATIO = 4       // multipler for refresh time to get sub values progress
const MIN_LOG_COUNT = 200              // min size of page log read request
/* eslint-enable no-multi-spaces */

export default {
  name: 'RunLog',
  components: { RefreshRunLog, RefreshRunProgress, RunBar, RunInfoDialog, ConfirmDialog },

  props: {
    digest: { type: String, default: '' },
    stamp: { type: String, default: '' },
    refreshTickle: { type: Boolean, default: false }
  },

  data () {
    return {
      isRefreshCompleted: false,
      isLogCompleted: false,
      isRefreshPaused: false,
      isLogRefresh: false,
      isProgressRefresh: false,
      refreshInt: '',
      refreshCount: 0,
      lastLogDt: 0,
      lastProgressDt: 0,
      sendLogCount: 0,
      sendProgressCount: 0,
      runState: Mdf.emptyRunState(),
      runStatusProgress: [],
      emptyLogCount: 0,
      emptyProgressCount: 0,
      logStart: 0,
      logCount: MIN_LOG_COUNT,
      logLines: [],
      runInfoTickle: false,
      runInfoDigest: '',
      showStopRunTickle: false
    }
  },

  computed: {
    lastProgressTimeStamp () {
      return this.lastProgressDt ? Mdf.dtToTimeStamp(new Date(this.lastProgressDt)) : ''
    },

    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl'
    })
  },

  watch: {
    refreshTickle () { this.initView() },
    stamp () { this.initView() }
  },

  emits: ['run-list-refresh', 'run-completed-list', 'tab-mounted'],

  methods: {
    ...mapActions(useModelStore, [
      'runTextByDigest',
      //
      'dispatchRunTextStatusUpdate'
    ]),

    isSuccess (status) { return status === Mdf.RUN_SUCCESS },
    isInProgress (status) { return status === Mdf.RUN_IN_PROGRESS || status === Mdf.RUN_INITIAL },
    runStatusDescr (status) { return Mdf.statusText(status) },
    descrOfRun (rd) { return Mdf.descrOfTxt(this.runTextByDigest({ ModelDigest: this.digest, RunDigest: rd })) },

    // show run notes dialog
    doShowRunNote (modelDgst, runDgst) {
      this.runInfoDigest = runDgst
      this.runInfoTickle = !this.runInfoTickle
    },

    // update page view
    initView () {
      this.isRefreshCompleted = false
      this.isLogCompleted = false
      this.runState = Mdf.emptyRunState()
      this.runStatusProgress = []
      this.emptyLogCount = 0
      this.emptyProgressCount = 0
      this.logStart = 0
      this.logLines = []
      this.stopRefreshProgress()
      this.startRefreshProgress()
    },

    // refersh model run progress
    refreshRunProgress () {
      if (this.isRefreshPaused) return
      //
      if (!this.isLogCompleted && this.sendLogCount++ < MAX_SEND_COUNT) this.isLogRefresh = !this.isLogRefresh

      this.refreshCount++
      if (this.isLogCompleted || this.refreshCount < RUN_PROGRESS_SUB_RATIO || (this.refreshCount % RUN_PROGRESS_SUB_RATIO) === 1) {
        if (this.sendProgressCount++ < MAX_SEND_COUNT) this.isProgressRefresh = !this.isProgressRefresh
      }
    },
    // pause on/off run progress refresh
    refreshPauseToggle () {
      this.refreshCount = 0
      this.sendLogCount = 0
      this.sendProgressCount = 0
      this.emptyLogCount = 0
      this.emptyProgressCount = 0
      this.isRefreshPaused = !this.isRefreshPaused
    },
    startRefreshProgress () {
      this.isRefreshPaused = false
      this.lastLogDt = 0
      this.lastProgressDt = 0
      this.refreshCount = 0
      this.sendLogCount = 0
      this.sendProgressCount = 0
      this.emptyLogCount = 0
      this.emptyProgressCount = 0
      this.refreshInt = setInterval(this.refreshRunProgress, RUN_PROGRESS_REFRESH_TIME)
    },
    stopRefreshProgress () {
      this.refreshCount = 0
      clearInterval(this.refreshInt)
    },

    // user answer is Yes to stop model run
    async onYesStopRun () {
      if (!this.digest || (this.runState.RunStamp || '') === '') {
        console.warn('Unable to stop: model digest or run stamp is empty', this.digest, (this.runState.RunStamp || ''))
        this.$q.notify({ type: 'negative', message: this.$t('Unable to stop: model digest or run stamp is empty') })
        return
      }

      const u = this.omsUrl +
        '/api/run/stop/model/' + encodeURIComponent(this.digest) +
        '/stamp/' + encodeURIComponent(this.runState.RunStamp || '')
      try {
        await this.$axios.put(u) // ignore response on success
      } catch (e) {
        console.warn('Unable to stop model run', e)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to stop model run: ') + (this.runState.RunStamp || '') })
        return // exit on error
      }

      // notify user on success, even run may not exist
      this.$q.notify({ type: 'info', message: this.$t('Stopping model run: ') + (this.runState.RunStamp || '') })
    },

    // model current run log status and page: response from server
    doneRunLogRefresh (ok, rslp) {
      this.sendLogCount = 0
      const now = Date.now()
      if (this.isLogCompleted || now - this.lastLogDt < RUN_PROGRESS_REFRESH_TIME) return // protect from timeouts storm
      this.lastLogDt = now

      if (!ok) return

      if (!Mdf.isNotEmptyRunStateLog(rslp)) {
        if (this.emptyLogCount++ > MAX_EMPTY_LOG_COUNT) this.isRefreshPaused = true // pause refresh if run state and log not available
        return
      }
      this.runState = Mdf.toRunStateFromLog(rslp) // new run log page
      this.emptyLogCount = 0

      // update log lines
      const nStart = this.logLines.length - rslp.Offset
      if (nStart >= 0) {
        const nLen = (rslp.Offset + rslp.Size) - this.logLines.length
        for (let k = nStart; k < nLen; k++) {
          this.logLines.push(rslp.Lines[k] || '')
        }
      }

      // check is it final update: model run completed
      const isDone = (this.runState.IsFinal && rslp.Offset + rslp.Size >= rslp.TotalSize)
      if (!isDone) {
        // not last log page: continue
        this.logStart = this.logLines.length
        this.logCount = rslp.TotalSize - this.logStart
        if (this.logCount < MIN_LOG_COUNT) this.logCount = MIN_LOG_COUNT
      } else {
        this.isLogCompleted = true // run state final and last log page received
      }
    },

    // model run status progress: response from server
    doneRunProgressRefresh (ok, rpLst) {
      this.sendProgressCount = 0
      const now = Date.now()
      if (!this.isLogCompleted && now - this.lastProgressDt < RUN_PROGRESS_REFRESH_TIME) return // protect from timeouts storm
      this.lastProgressDt = now

      if (!ok) return

      if (!Mdf.isLength(rpLst)) {
        if (this.emptyProgressCount++ > MAX_EMPTY_PROGRESS_COUNT) this.isRefreshPaused = true // pause refresh if run progress not available
        return
      }
      this.emptyProgressCount = 0 // new progress response

      // update run list progress, check if there is any new runs, status changes or completed runs
      let isNew = false
      let isAllCompleted = this.isLogCompleted
      const rca = []

      for (const rp of rpLst) {
        if (!Mdf.isNotEmptyRunStatusProgress(rp)) continue

        const k = this.runStatusProgress.findIndex(r => r.RunDigest === rp.RunDigest)
        const st = k >= 0 ? this.runStatusProgress[k].Status : ''
        isNew = isNew || k < 0 || rp.Status !== st

        const isCompleted = Mdf.isRunCompletedStatus(rp.Status)
        if (isCompleted && rp.Status !== st) rca.push(rp.RunDigest)

        isAllCompleted = isAllCompleted && isCompleted
        this.dispatchRunTextStatusUpdate(rp)
      }

      if (isNew) {
        this.$emit('run-list-refresh') // if any new runs or status change then refresh run list
      }
      if (rca.length > 0) {
        this.$nextTick(() => {
          this.$emit('run-completed-list', rca) // for completed runs list of parameters must be available
        })
      }
      this.runStatusProgress = rpLst // new run progress array
      //
      // if log completed and all run status completed
      //
      if (isAllCompleted) {
        this.isRefreshCompleted = true
        this.stopRefreshProgress()
      }
    }
  },

  mounted () {
    this.initView()
    this.$emit('tab-mounted', 'run-log', { digest: this.digest, runStamp: this.stamp })
  },
  beforeUnmount () {
    this.stopRefreshProgress()
  }
}
</script>
