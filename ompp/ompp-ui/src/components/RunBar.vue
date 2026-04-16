<!-- run info bar: show run info in flex bar -->
<template>
  <div
    class="row items-center"
    >

    <q-btn
      @click="onShowRunNote"
      :disable="!isNotEmptyRun"
      flat
      dense
      class="col-auto text-white rounded-borders q-mr-xs"
      :class="isDeleted ? 'bg-negative' : ((!isDeleted && (isSuccess || isInProgress)) ? 'bg-primary' : 'bg-warning')"
      :icon="isSuccess ? 'mdi-information' : (isInProgress ? 'mdi-run' : 'mdi-alert-circle-outline')"
      :title="(isDeleted ? $t('Deleted') : $t('About')) + ' ' + runText.Name"
      />

    <div
      class="col-auto"
      >
      <span v-if="isNotEmptyRun">{{ runText.Name }}<br />
      <span class="om-text-descr"><span class="mono q-pr-sm">{{ lastDateTimeStr }}</span>{{ descrOfRun }}</span></span>
      <span v-else disabled>{{ $t('Server offline or model run not found') }}</span>
    </div>

  </div>
</template>

<script>
import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import * as Mdf from 'src/model-common'

export default {
  name: 'RunBar',

  props: {
    modelDigest: { type: String, default: '' },
    runDigest: { type: String, default: '' },
    refreshRunTickle: { type: Boolean, default: false }
  },

  data () {
    return {
      runText: Mdf.emptyRunText()
    }
  },

  computed: {
    isNotEmptyRun () { return Mdf.isNotEmptyRunText(this.runText) },
    isSuccess () { return this.runText.Status === Mdf.RUN_SUCCESS },
    isInProgress () { return this.runText.Status === Mdf.RUN_IN_PROGRESS || this.runText.Status === Mdf.RUN_INITIAL },
    isDeleted () { return Mdf.isRunDeletedStatus(this.runText.Status, this.runText.Name) },
    lastDateTimeStr () { return Mdf.dtStr(this.runText.UpdateDateTime) },
    descrOfRun () { return Mdf.descrOfTxt(this.runText) },

    ...mapState(useModelStore, [
      'runTextListUpdated'
    ]),
    ...mapState(useServerStateStore, {
      serverConfig: 'config'
    })
  },

  watch: {
    modelDigest () { this.doRefresh() },
    runDigest () { this.doRefresh() },
    refreshRunTickle () { this.doRefresh() },
    runTextListUpdated () { this.doRefresh() }
  },

  emits: ['run-info-click'],

  methods: {
    ...mapActions(useModelStore, ['runTextByDigest']),

    doRefresh () {
      this.runText = this.runTextByDigest({ ModelDigest: this.modelDigest, RunDigest: this.runDigest })
    },
    onShowRunNote () {
      this.$emit('run-info-click', this.modelDigest, this.runDigest)
    }
  },

  mounted () {
    this.doRefresh()
  }
}
</script>

<style lang="scss" scope="local">
</style>
