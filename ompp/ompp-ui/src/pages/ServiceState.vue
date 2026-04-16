<template>
<q-page class="text-body1 q-pa-sm">

  <div class="row items-center full-width">

    <span class="col-auto q-pa-xs q-mr-xs">
      <q-btn
        v-if="!isRefreshDisabled && serverConfig.IsJobControl"
        @click="refreshPauseToggle"
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders"
        :icon="!isRefreshPaused ? (stateRefreshTickle ? 'mdi-autorenew' : 'mdi-sync') : 'mdi-play-circle-outline'"
        :title="!isRefreshPaused ? $t('Pause auto refresh') : $t('Auto refresh service state')"
        />
      <q-btn
        v-else
        disable
        flat
        dense
        class="col-auto bg-primary text-white rounded-borders"
        icon="mdi-autorenew"
        :title="$t('Refresh')"
        />
        <span v-if="srvState.JobUpdateDateTime" class="mono om-text-secondary q-ml-xs">{{ srvState.JobUpdateDateTime }}</span>
    </span>

    <div class="col-grow res-title q-pl-sm q-py-sm bg-primary text-white">
      <q-btn
        v-if="serverConfig.IsJobControl && srvState.ComputeState.length"
        @click="isShowServers = !isShowServers"
        flat
        dense
        class="bg-primary text-white rounded-borders q-mr-sm"
        :icon="isShowServers ? 'keyboard_arrow_up' : 'keyboard_arrow_down'"
        :title="isShowServers ? $t('Hide state of computational servers') : $t('Show state of computational servers')"
        />
      <template v-if="srvState.MpiRes.Cpu">
        <span>{{ $t('MPI CPU Cores') }}: {{ srvState.MpiRes.Cpu }}</span>
        <span class="q-pl-md">{{ $t('Used') }}: {{ srvState.ActiveTotalRes.Cpu }}</span>
        <span v-show="srvState.MpiErrorRes.Cpu" class="q-pl-md">{{ $t('Failed') }}: {{ srvState.MpiErrorRes.Cpu }}</span>
      </template>
      <span v-if="srvState.MpiRes.Cpu > 0 && srvState.LocalRes.Cpu > 0" class="q-mx-md">&#124;</span>
      <template v-if="srvState.LocalRes.Cpu">
        <span>{{ $t('Local CPU Cores') }}: {{ srvState.LocalRes.Cpu }}</span>
        <span class="q-pl-md">{{ $t('Used') }}: {{ srvState.LocalActiveRes.Cpu }}</span>
      </template>&nbsp;
    </div>

  </div>

  <div
    v-if="serverConfig.IsJobControl && srvState.ComputeState.length > 0"
    v-show="isShowServers"
    class="q-py-sm q-px-xs">

    <table class="om-p-table">
      <thead>
        <tr>
          <th class="om-p-head-center text-weight-medium">{{ $t('Server') }}</th>
          <td class="om-p-head-center text-weight-medium">{{ $t('Status') }}</td>
          <th class="om-p-head-center text-weight-medium">{{ $t('CPU Cores') }}</th>
          <th class="om-p-head-center text-weight-medium">{{ $t('Cores Used') }}</th>
          <th class="om-p-head-center text-weight-medium">{{ $t('Used by You') }}</th>
          <th class="om-p-head-center text-weight-medium">{{ $t('Errors') }}</th>
          <th class="om-p-head-center text-weight-medium">{{ $t('Last Activity Time') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="cs of srvState.ComputeState" :key="cs.Name + '-' + cs.Status + '-' + cs.LastUsedTs.toString()">
          <td class="om-p-cell-left">{{ cs.Name }}</td>
          <td v-if="cs.State === 'off'" class="bg-secondary text-white om-p-cell-center">{{ $t(cs.State) }}</td>
          <td v-if="cs.State === 'ready'" class="bg-positive text-white om-p-cell-center">{{ $t(cs.State) }}</td>
          <td v-if="cs.State === 'start'" class="bg-info text-white om-p-cell-center">&uarr; {{ $t(cs.State) }}</td>
          <td v-if="cs.State === 'stop'" class="bg-info text-white om-p-cell-center">&darr; {{ $t(cs.State) }}</td>
          <td v-if="cs.State === 'error'" class="bg-negative text-white om-p-cell-center">{{ $t(cs.State) }}</td>
          <td v-if="cs.State !== 'off' && cs.State !== 'ready' && cs.State !== 'start' && cs.State !== 'stop' && cs.State !== 'error'" class="om-p-cell-center">? {{ cs.State }} ?</td>
          <td class="om-p-cell-right">{{ cs.TotalRes.Cpu }}</td>
          <td class="om-p-cell-right">{{ cs.UsedRes.Cpu || '' }}</td>
          <td class="om-p-cell-right">{{ cs.OwnRes.Cpu || '' }}</td>
          <td class="om-p-cell-right">{{ cs.ErrorCount || '' }}</td>
          <td class="om-p-cell-right">{{ lastUsedDt(cs.LastUsedTs) }}</td>
        </tr>
      </tbody>
    </table>

  </div>

  <q-expansion-item
    :disable="!serverConfig.IsJobControl"
    v-model="isActiveShow"
    switch-toggle-side
    expand-separator
    class="q-my-sm"
    header-class="bg-primary text-white"
    >
      <template v-slot:header>
        <q-item-section>
          <q-item-label>
            <span>{{ $t('Active Model Runs') }}: {{ srvState.Active.length || $t('None') }}</span>
            <span v-if="srvState.ActiveTotalRes.Cpu > 0" class="q-mx-md">&#124;</span>
            <span v-if="srvState.ActiveTotalRes.Cpu">{{ $t('MPI CPU Cores') }}: {{ srvState.ActiveTotalRes.Cpu }}</span>
            <span v-if="srvState.LocalActiveRes.Cpu > 0" class="q-mx-md">&#124;</span>
            <span v-if="srvState.LocalActiveRes.Cpu" class="q-pr-sm">{{ $t('Local CPU Cores') }}: {{ srvState.LocalActiveRes.Cpu }}</span>
          </q-item-label>
        </q-item-section>
      </template>
    <q-list bordered>

      <q-expansion-item
        v-for="aj of srvState.Active" :key="aj.SubmitStamp"
        :disable="!aj.ModelDigest || !aj.SubmitStamp"
        @after-show="onActiveShow(aj.SubmitStamp)"
        @after-hide="onActiveHide(aj.SubmitStamp)"
        expand-icon-toggle
        switch-toggle-side
        expand-icon-class="job-expand-btn bg-primary text-white rounded-borders q-mr-sm"
        :title="$t('About') + ' ' + aj.ModelName + ' ' + aj.SubmitStamp"
        >
        <template v-slot:header>
          <q-item-section avatar class="job-hdr-action-bar q-pr-xs">
            <div class="row items-center">
              <q-btn
                :to="'/model/' + encodeURIComponent(aj.ModelDigest) + '/run-log/' + encodeURIComponent(aj.RunStamp)"
                flat
                dense
                class="col-auto bg-primary text-white rounded-borders q-mr-xs"
                icon="mdi-text-long"
                :title="$t('Run Log: ') + aj.ModelName + ' ' + aj.RunStamp"
                />
              <q-btn
                @click="onStopJobConfirm(aj.SubmitStamp, aj.ModelDigest, aj.ModelName)"
                flat
                dense
                class="col-auto bg-primary text-white rounded-borders q-mr-xs"
                icon="mdi-alert-octagon-outline"
                :title="$t('Stop model run') + ' ' + aj.SubmitStamp"
                />
            </div>
          </q-item-section>
          <q-item-section>
            <q-item-label>{{ aj.ModelName }}<span class="om-text-descr">: {{ getRunTitle(aj) }}</span></q-item-label>
            <q-item-label class="om-text-descr">
              {{ $t('Submitted:') }} <span class="mono">{{ fromUnderscoreTs(aj.SubmitStamp) }}</span>
              <span class="q-ml-md">{{ $t('Run Stamp:') }} <span class="mono">{{ fromUnderscoreTs(aj.RunStamp) }}</span></span>
            </q-item-label>
          </q-item-section>
        </template>

        <job-info-card
          :is-microdata="isMicrodata"
          :job-item="activeJobs[aj.SubmitStamp]"
          class="job-card q-mx-sm q-mb-md"
        >
        </job-info-card>

      </q-expansion-item>

    </q-list>
  </q-expansion-item>

  <q-expansion-item
    :disable="!serverConfig.IsJobControl"
    v-model="isQueueShow"
    switch-toggle-side
    expand-separator
    header-class="bg-primary text-white"
    class="q-my-sm"
    >
      <template v-slot:header>
        <q-item-section>
          <div class="row no-wrap items-center full-width">
            <div class="col-auto">
              <q-btn
                @click.stop="doQueuePauseResume(!srvState.IsQueuePaused)"
                round
                no-caps
                :icon="!srvState.IsQueuePaused ? 'mdi-pause' : 'mdi-play'"
                :title="!srvState.IsQueuePaused ? $t('Pause the queue') : $t('Resume the queue')"
                class="col-auto bg-white text-primary"
                />
            </div>
            <div class="col-grow q-pl-sm">
              <span>{{ $t('Model Run Queue') }}</span><span v-if="srvState.IsQueuePaused" class="q-pl-sm">({{ $t('paused') }})</span>: <span>{{ srvState.Queue.length || $t('None') }}</span>
              <span v-if="srvState.QueueOwnRes.Cpu > 0" class="q-mx-md">&#124;</span>
              <span v-if="srvState.QueueOwnRes.Cpu">{{ $t('MPI CPU Cores') }}: {{ srvState.QueueOwnRes.Cpu }}</span>
              <span v-if="srvState.LocalQueueRes.Cpu > 0" class="q-mx-md">&#124;</span>
              <span v-if="srvState.LocalQueueRes.Cpu" class="q-pr-sm">{{ $t('Local CPU Cores') }}: {{ srvState.LocalQueueRes.Cpu }}</span>
            </div>
          </div>
        </q-item-section>
      </template>
    <q-list bordered>

      <q-expansion-item
        v-for="(qj, qPos) of srvState.Queue" :key="qj.SubmitStamp"
        :disable="!qj.ModelDigest || !qj.SubmitStamp"
        @after-show="onQueueShow(qj.SubmitStamp)"
        @after-hide="onQueueHide(qj.SubmitStamp)"
        expand-icon-toggle
        switch-toggle-side
        expand-icon-class="job-expand-btn bg-primary text-white rounded-borders q-mr-sm"
        :title="$t('About') + ' ' + qj.ModelName + ' ' + qj.SubmitStamp"
        >
        <template v-slot:header>
          <q-item-section avatar class="q-pr-xs">
            <div class="row items-center">
              <q-badge outline color="primary" class="col-auto q-mr-xs">{{ qj.QueuePos > 0 ? qj.QueuePos : '&#8211;' }}</q-badge>
              <q-btn
                :disable="!qj.IsMpi || isTopQueue(qj.SubmitStamp)"
                @click="onJobMove(qj.SubmitStamp, 0, qj.ModelName, getRunTitle(qj))"
                flat
                dense
                class="col-auto bg-primary text-white rounded-borders q-mr-xs"
                icon="mdi-arrow-collapse-up"
                :title="$t('Move to the top')"
                />
              <q-btn
                :disable="!qj.IsMpi || isTopQueue(qj.SubmitStamp)"
                @click="onJobMove(qj.SubmitStamp, qPos - 1, qj.ModelName, getRunTitle(qj))"
                flat
                dense
                class="col-auto bg-primary text-white rounded-borders q-mr-xs"
                icon="mdi-arrow-up"
                :title="$t('Move up')"
                />
              <q-btn
                :disable="!qj.IsMpi || isBottomQueue(qj.SubmitStamp)"
                @click="onJobMove(qj.SubmitStamp, qPos + 1, qj.ModelName, getRunTitle(qj))"
                flat
                dense
                class="col-auto bg-primary text-white rounded-borders q-mr-xs"
                icon="mdi-arrow-down"
                :title="$t('Move down')"
                />
              <q-btn
                :disable="!qj.IsMpi || isBottomQueue(qj.SubmitStamp)"
                @click="onJobMove(qj.SubmitStamp, srvState.Queue.length + 1, qj.ModelName, getRunTitle(qj))"
                flat
                dense
                class="col-auto bg-primary text-white rounded-borders q-mr-xs"
                icon="mdi-arrow-collapse-down"
                :title="$t('Move to the bottom')"
                />
              <q-btn
                @click="onStopJobConfirm(qj.SubmitStamp, qj.ModelDigest, qj.ModelName, getRunTitle(qj))"
                flat
                dense
                class="col-auto bg-primary text-white rounded-borders q-mr-xs"
                icon="mdi-delete"
                :title="$t('Delete from the queue') + qj.SubmitStamp"
                />
            </div>
          </q-item-section>
          <q-item-section>
            <q-item-label>{{ qj.ModelName }}<span class="om-text-descr">: {{ getRunTitle(qj) }}</span></q-item-label>
            <q-item-label class="om-text-descr">
              {{ $t('Submitted:') }} <span class="mono">{{ fromUnderscoreTs(qj.SubmitStamp) }}</span> <span v-if="qj.IsOverLimit" class="text-negative q-ml-md">{{ $t('Exceed resource limit') }}</span>
            </q-item-label>
          </q-item-section>
        </template>

        <job-info-card
          :is-microdata="isMicrodata"
          :job-item="queueJobs[qj.SubmitStamp]"
          class="job-card q-mx-sm q-mb-md"
        >
        </job-info-card>

      </q-expansion-item>

    </q-list>
  </q-expansion-item>

  <q-expansion-item
    :disable="!serverConfig.IsJobControl"
    v-model="isOtherHistoryShow"
    switch-toggle-side
    expand-separator
    header-class="bg-primary text-white"
    class="q-my-sm"
    >
    <template v-slot:header>
      <q-item-section>
        <div class="row no-wrap items-center full-width">
          <div class="col-auto">
            <q-btn
              :disable="!otherHistory.length"
              @click.stop="onDeleteAllHistoryConfirm(false)"
              :round="!otherHistory.length"
              :outline="!otherHistory.length"
              :rounded="!!otherHistory.length"
              no-caps
              :icon="!!otherHistory.length ? 'mdi-delete' : 'mdi-delete-outline'"
              :label="!!otherHistory.length ? '[ ' + otherHistory.length.toLocaleString() + ' ]' : ''"
              :title="$t('Delete all') + (!!otherHistory.length ? ' [ ' + otherHistory.length.toLocaleString() + ' ]' : '\u2026')"
              class="col-auto bg-white text-primary"
              />
          </div>
          <span class="col-grow q-pl-sm">{{ $t('Failed Model Runs: ') + (otherHistory.length || $t('None')) }}</span>
        </div>
      </q-item-section>
    </template>
    <q-list bordered>

      <q-expansion-item
        v-for="hj of otherHistory" :key="hj.SubmitStamp"
        :disable="!hj.ModelDigest || !hj.SubmitStamp"
        @after-show="onHistoryShow(hj.SubmitStamp)"
        @after-hide="onHistoryHide(hj.SubmitStamp)"
        expand-icon-toggle
        switch-toggle-side
        expand-icon-class="job-expand-btn bg-primary text-white rounded-borders q-mr-sm"
        :title="$t('About') + ' ' + hj.ModelName + ' '+ hj.SubmitStamp"
        >
        <template v-slot:header>
          <q-item-section avatar class="job-hdr-action-bar q-pr-xs">
            <div class="row items-center">
              <q-btn
                @click="onRunAgain(hj.SubmitStamp, hj.ModelName, getHistoryTitle(hj))"
                flat
                dense
                class="col-auto bg-primary text-white rounded-borders q-mr-xs"
                icon="mdi-run"
                :title="$t('Run again') + ' ' + hj.SubmitStamp"
                />
              <q-btn
                :to="'/model/' + encodeURIComponent(hj.ModelDigest) + '/run-log/' + encodeURIComponent(hj.RunStamp)"
                flat
                dense
                class="col-auto bg-primary text-white rounded-borders q-mr-xs"
                icon="mdi-text-long"
                :title="$t('Run Log: ') + hj.ModelName + ' ' + hj.RunStamp"
                />
              <q-btn
                @click="onDeleteJobHistoryConfirm(hj.SubmitStamp, hj.ModelName, getHistoryTitle(hj))"
                flat
                dense
                class="col-auto bg-primary text-white rounded-borders q-mr-xs"
                icon="mdi-delete-outline"
                :title="$t('Delete') + ' ' + hj.SubmitStamp"
                />
            </div>
          </q-item-section>
          <q-item-section>
            <q-item-label>
              <span class="om-text-descr text-negative q-pr-sm">{{ $t(runStatusDescr(hj.JobStatus)) }}</span>
              <span> {{ hj.ModelName }}<span class="om-text-descr" v-if="getHistoryTitle(hj)">: {{ getHistoryTitle(hj) }}</span></span>
            </q-item-label>
            <q-item-label class="om-text-descr">
              {{ $t('Submitted:') }} <span class="mono">{{ fromUnderscoreTs(hj.SubmitStamp) }}</span>
              <span class="q-ml-md">{{ $t('Run Stamp:') }} <span class="mono">{{ fromUnderscoreTs(hj.RunStamp) }}</span></span>
            </q-item-label>
          </q-item-section>
        </template>

        <job-info-card
          :is-microdata="isMicrodata"
          :job-item="historyJobs[hj.SubmitStamp]"
          class="job-card q-mx-sm q-mb-md"
        >
        </job-info-card>

      </q-expansion-item>

    </q-list>
  </q-expansion-item>

  <q-expansion-item
    :disable="!serverConfig.IsJobControl"
    v-model="isDoneHistoryShow"
    switch-toggle-side
    expand-separator
    header-class="bg-primary text-white"
    class="q-my-sm"
    >
    <template v-slot:header>
      <q-item-section>
        <div class="row no-wrap items-center full-width">
          <div class="col-auto">
            <q-btn
              :disable="!doneHistory.length"
              @click.stop="onDeleteAllHistoryConfirm(true)"
              :round="!doneHistory.length"
              :outline="!doneHistory.length"
              :rounded="!!doneHistory.length"
              no-caps
              :icon="!!doneHistory.length ? 'mdi-delete' : 'mdi-delete-outline'"
              :label="!!doneHistory.length ? '[ ' + doneHistory.length.toLocaleString() + ' ]' : ''"
              :title="$t('Delete all') + (!!doneHistory.length ? ' [ ' + doneHistory.length.toLocaleString() + ' ]' : '\u2026')"
              class="col-auto bg-white text-primary"
              />
          </div>
          <span class="col-grow q-pl-sm">{{ $t('Completed Model Runs: ') + (doneHistory.length || $t('None')) }}</span>
        </div>
      </q-item-section>
    </template>
    <q-list bordered>

      <q-expansion-item
        v-for="hj of doneHistory" :key="hj.SubmitStamp"
        :disable="!hj.ModelDigest || !hj.SubmitStamp"
        @after-show="onHistoryShow(hj.SubmitStamp)"
        @after-hide="onHistoryHide(hj.SubmitStamp)"
        expand-icon-toggle
        switch-toggle-side
        expand-icon-class="job-expand-btn bg-primary text-white rounded-borders q-mr-sm"
        :title="$t('About') + ' ' + hj.ModelName + ' '+ hj.SubmitStamp"
        >
        <template v-slot:header>
          <q-item-section avatar class="job-hdr-action-bar q-pr-xs">
            <div class="row items-center">
              <q-btn
                :to="'/model/' + encodeURIComponent(hj.ModelDigest) + '/run-log/' + encodeURIComponent(hj.RunStamp)"
                flat
                dense
                class="col-auto bg-primary text-white rounded-borders q-mr-xs"
                icon="mdi-text-long"
                :title="$t('Run Log: ') + hj.ModelName + ' ' + hj.RunStamp"
                />
              <q-btn
                @click="onDeleteJobHistoryConfirm(hj.SubmitStamp, hj.ModelName, getHistoryTitle(hj))"
                flat
                dense
                class="col-auto bg-primary text-white rounded-borders q-mr-xs"
                icon="mdi-delete-outline"
                :title="$t('Delete') + ' ' + hj.SubmitStamp"
                />
            </div>
          </q-item-section>
          <q-item-section>
            <q-item-label>
              <span class="om-text-descr text-primary q-pr-sm">{{ $t(runStatusDescr(hj.JobStatus)) }}</span>
              <span>{{ hj.ModelName }}<span class="om-text-descr" v-if="getHistoryTitle(hj)">: {{ getHistoryTitle(hj) }}</span></span>
            </q-item-label>
            <q-item-label class="om-text-descr">
              {{ $t('Submitted:') }} <span class="mono">{{ fromUnderscoreTs(hj.SubmitStamp) }}</span>
              <span class="q-ml-md">{{ $t('Run Stamp:') }} <span class="mono">{{ fromUnderscoreTs(hj.RunStamp) }}</span></span>
            </q-item-label>
          </q-item-section>
        </template>

        <job-info-card
          :is-microdata="isMicrodata"
          :job-item="historyJobs[hj.SubmitStamp]"
          class="job-card q-mx-sm q-mb-md"
        >
        </job-info-card>

      </q-expansion-item>

    </q-list>
  </q-expansion-item>

  <delete-confirm-dialog
    @delete-yes="onYesStopRun"
    :show-tickle="showStopRunTickle"
    :item-name="stopRunTitle"
    :item-id="stopSubmitStamp"
    :kind="stopModelDigest"
    :dialog-title="$t('Stop model run?')"
    :icon-name="'mdi-alert-octagon'"
    >
  </delete-confirm-dialog>
  <delete-confirm-dialog
    @delete-yes="onYesDeleteJobHistory"
    :show-tickle="showDeleteHistoryTickle"
    :item-name="deleteHistoryTitle"
    :item-id="deleteSubmitStamp"
    :dialog-title="$t('Delete model run history?')"
    >
  </delete-confirm-dialog>
  <delete-confirm-dialog
    @delete-yes="onYesDeleteAllJobHistory"
    :show-tickle="showDeleteAllHistoryTickle"
    :item-name="deleteAllHistoryTitle"
    :kind="deleteAllHistoryKind"
    :dialog-title="$t('Delete all history')"
    :body-text="$t('Only model run history records deleted. Actual model run results not affected.')"
    >
  </delete-confirm-dialog>

</q-page>
</template>

<script src="./service-state.js"></script>

<style lang="scss" scope="local">
  // override card shadow inside of expansion item
  .q-expansion-item__content > div.job-card {
    box-shadow: $shadow-1;
  }
  .job-expand-btn {
    padding-right: 0;
    align-content: center;
  }
  .job-hdr-action-bar {
    min-width: 2rem;
  }
  .res-title {
    min-height: 1.5rem;
  }
</style>
