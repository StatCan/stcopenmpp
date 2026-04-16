<template>
<q-card>

  <template v-if="!isNotEmptyJob()">
    <q-card-section>{{ $t('None') }}</q-card-section>
  </template>
  <template v-else>

    <q-card-section>
      <table class="om-p-table">
        <tbody>
          <tr v-if="jobInfo.ModelVersion">
            <td class="om-p-head-left text-weight-medium">{{ $t('Model Version') }}</td>
            <td class="om-p-cell-left">{{ jobInfo.ModelVersion }}</td>
          </tr>
          <tr v-if="jobInfo.ModelCreateDateTime">
            <td class="om-p-head-left text-weight-medium">{{ $t('Model Created') }}</td>
            <td class="om-p-cell-left">{{ jobInfo.ModelCreateDateTime }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left text-weight-medium">{{ $t('Model Digest') }}</td>
            <td class="om-p-cell-left">{{ jobItem.ModelDigest }}</td>
          </tr>
          <tr v-if="jobInfo.OptRunName">
            <td class="om-p-head-left text-weight-medium">{{ $t('Run Name') }}</td>
            <td class="om-p-cell-left">{{ jobInfo.OptRunName }}</td>
          </tr>
          <tr v-if="jobItem.RunStamp">
            <td class="om-p-head-left text-weight-medium">{{ $t('Run Stamp') }}</td>
            <td class="om-p-cell-left">{{ jobItem.RunStamp }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left text-weight-medium">{{ $t('Submit Stamp') }}</td>
            <td class="om-p-cell-left">{{ jobItem.SubmitStamp }}</td>
          </tr>
          <tr v-if="jobInfo.OptSubValues && jobInfo.OptSubValues !== '0'">
            <td class="om-p-head-left text-weight-medium">{{ $t('Sub-values Count') }}</td>
            <td class="om-p-cell-left">{{ jobInfo.OptSubValues }}</td>
          </tr>
          <tr v-if="jobInfo.OptSetName">
            <td class="om-p-head-left text-weight-medium">{{ $t('Input Scenario') }}</td>
            <td class="om-p-cell-left">{{ jobInfo.OptSetName }}</td>
          </tr>
          <tr v-if="jobInfo.OptBaseRunDigest">
            <td class="om-p-head-left text-weight-medium">{{ $t('Base Run Digest') }}</td>
            <td class="om-p-cell-left">{{ jobInfo.OptBaseRunDigest }}</td>
          </tr>
          <tr v-if="jobItem.Tables.length > 0">
            <td class="om-p-head-left text-weight-medium">{{ $t('Output Tables') }}</td>
            <td class="om-p-cell-left">{{ jobItem.Tables.join(', ') }}</td>
          </tr>
          <tr v-if="jobInfo.nProc > 1 || jobItem.Res.ThreadCount > 1">
            <td class="om-p-head-left text-weight-medium">{{ $t('Processes / Threads') }}</td>
            <td class="om-p-cell-left">{{ jobInfo.nProc }} / {{ jobItem.Res.ThreadCount }}</td>
          </tr>
          <tr v-if="!!jobItem.IsMpi || jobItem.Res.Cpu >= 1">
            <td class="om-p-head-left text-weight-medium">{{ $t('CPU Cores') }}</td>
            <td class="om-p-cell-left">{{ jobItem.Res.Cpu }} {{ jobItem.IsMpi ? 'MPI' : '' }}</td>
          </tr>
          <tr v-if="jobItem.Res.Mem !== 0">
            <td class="om-p-head-left text-weight-medium">{{ $t('Memory') }}</td>
            <td class="om-p-cell-left">{{ jobItem.Res.Mem }} {{ $t('GByte(s)') }}</td>
          </tr>
          <tr v-if="jobItem.Template">
            <td class="om-p-head-left text-weight-medium">{{ $t('Model Run Template') }}</td>
            <td class="om-p-cell-left">{{ jobItem.Template }}</td>
          </tr>
          <tr v-if="jobItem.LogFileName">
            <td class="om-p-head-left text-weight-medium">{{ $t('Run Log') }}</td>
            <td class="om-p-cell-left">{{ jobItem.LogFileName }}</td>
          </tr>
          <tr v-if="jobInfo.runDescr.length > 0">
            <td class="om-p-head-left text-weight-medium">{{ $t('Run Description') }}</td>
            <td class="om-p-cell-left">
              <div v-for="(rd, ird) of jobInfo.runDescr" :key="'h-' + rd.key + '-' + ird.toString()">
                {{ rd.descr }}
                <q-separator v-if="ird < jobInfo.runDescr.length - 1"/>
              </div>
            </td>
          </tr>
          <template v-if="isAnyMicrodata()">
            <tr>
              <td  colspan="2" class="om-p-head-center text-weight-medium">{{ $t('Microdata') }}</td>
            </tr>
            <tr v-for="(ent, ien) of jobItem.Microdata.Entity" :key="'md-ent-' + ent.Name + '-' + ien.toString()">
              <td class="om-p-head-left text-weight-medium">{{ ent.Name }}</td>
              <td class="om-p-cell-left">{{ ent.Attr.join(', ') }}</td>
            </tr>
          </template>
        </tbody>
      </table>
    </q-card-section>

    <q-card-section
      v-for="rsp in jobItem.RunStatus" :key="rsp.RunDigest + '-' + (rsp.Status || 'st')"
      >
      <table v-if="rsp.UpdateDateTime" class="om-p-table">
        <tbody>
          <tr>
            <td colspan="4" class="om-p-head-center text-weight-medium">{{ rsp.Name }}</td>
          </tr>
          <tr>
            <td colspan="2" class="om-p-head-left text-weight-medium">{{ $t('Run Started') }}</td>
            <td colspan="2" class="om-p-cell-left">{{ dtRoundStr(rsp.CreateDateTime) }}</td>
          </tr>
          <tr>
            <td colspan="2" class="om-p-head-left text-weight-medium">{{ $t('Duration') }}</td>
            <td colspan="2" class="om-p-cell-left">{{ durationStr(rsp.CreateDateTime, rsp.UpdateDateTime) }}</td>
          </tr>
          <tr v-if="rsp.RunDigest">
            <td colspan="2" class="om-p-head-left text-weight-medium">{{ $t('Run Digest') }}</td>
            <td colspan="2" class="om-p-cell-left">{{ rsp.RunDigest }}</td>
          </tr>
          <tr>
            <td class="om-p-head-center text-weight-medium">{{ rsp.SubCompleted > 0 ? $t('Completed') : $t('Sub-values') }}</td>
            <td class="om-p-head-center text-weight-medium">{{ $t('Status') }}</td>
            <td class="om-p-head-center text-weight-medium">{{ $t('Updated') }}</td>
            <td class="om-p-head-center text-weight-medium">{{ $t('Value Digest') }}</td>
          </tr>
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

    <q-card-section
      v-if="jobItem.LogFileName"
      >
      <span class="mono"><i>{{ jobItem.LogFileName }}:</i></span>
      <div v-if="jobItem.Lines.length <= 0">
        <span class="mono q-pt-md q-pl-md">{{ $t('Log file not found or empty') }}</span>
      </div>
      <div v-else>
        <pre>{{jobItem.Lines.join('\n')}}</pre>
      </div>
    </q-card-section>

  </template>

</q-card>
</template>

<script>
import * as Mdf from 'src/model-common'

export default {
  name: 'JobInfoCard',

  props: {
    isMicrodata: { type: Boolean, default: false },
    jobItem: {
      type: Object,
      default: Mdf.emptyJobItem()
    }
  },

  data () {
    return {
    }
  },

  computed: {

    // return run job info and model info from job control item and first run status
    jobInfo () {
      const rji = {
        ModelVersion: '',
        ModelCreateDateTime: '',
        opts: [],
        runDescr: [],
        OptRunName: '',
        OptSubValues: '',
        OptSetName: '',
        OptBaseRunDigest: '',
        nProc: 1,
        RunNotes: []
      }
      if (!Mdf.isNotEmptyJobItem(this.jobItem)) return rji

      if (this.jobItem.RunStatus.length > 0) {
        rji.ModelVersion = this.jobItem.RunStatus[0]?.ModelVersion || ''
        rji.ModelCreateDateTime = this.jobItem.RunStatus[0]?.ModelCreateDateTime || ''
      }

      for (const iKey in this.jobItem.Opts) {
        if (!iKey || (this.jobItem.Opts[iKey] || '') === '') continue

        const v = this.jobItem.Opts[iKey]
        rji.opts.push({ key: iKey, val: v })

        const klc = iKey.toLowerCase()

        if (klc.endsWith('.RunDescription'.toLowerCase())) {
          rji.runDescr.push({ key: 'rd-' + iKey, descr: v })
        }
        if (klc.endsWith('OpenM.RunName'.toLowerCase())) rji.OptRunName = v
        if (klc.endsWith('OpenM.SubValues'.toLowerCase())) rji.OptSubValues = v
        if (klc.endsWith('OpenM.SetName'.toLowerCase())) rji.OptSetName = v
        if (klc.endsWith('OpenM.BaseRunDigest'.toLowerCase())) rji.OptBaseRunDigest = v
      }

      rji.nProc = this.jobItem.Res.ProcessCount
      if (this.jobItem.IsMpi && this.jobItem.Mpi.IsNotOnRoot && this.jobItem.Res.ProcessCount > 0) rji.nProc = this.jobItem.Res.ProcessCount + 1

      // run notes: sanitize and convert from markdown to html
      // rji.RunNotes = jc.RunNotes || []

      return rji
    }
  },

  methods: {
    isNotEmptyJob () { return Mdf.isNotEmptyJobItem(this.jobItem) },
    isAnyMicrodata () {
      return this.isMicrodata && this.jobItem?.Microdata?.Entity && this.jobItem?.Microdata?.Entity?.length > 0
    },
    dtRoundStr (dt) { return Mdf.dtStr(dt) },
    durationStr (startDt, lastDt) { return Mdf.toIntervalStr(Mdf.dtStr(startDt), Mdf.dtStr(lastDt)) },
    runStatusDescr (status) { return Mdf.statusText(status) }
  },

  mounted () {
  }
}
</script>
