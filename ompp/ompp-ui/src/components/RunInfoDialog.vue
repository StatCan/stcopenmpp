<template>
<q-dialog full-width v-model="showDlg">
  <q-card>

    <q-card-section class="row text-h6 bg-primary text-white">
      <div>{{ title }}</div><q-space /><q-btn icon="mdi-close" flat dense round v-close-popup />
    </q-card-section>

    <q-card-section class="text-body1 q-pb-none">
      <table class="om-p-table">
        <tbody>
          <tr>
            <td class="om-p-head-left">{{ $t('Name') }}</td>
            <td class="om-p-cell-left mono">{{ runText.Name }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Status') }}</td>
            <td class="om-p-cell-left mono" :class="isDeleted ? 'text-white bg-negative' : ''">{{ statusDescr }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Sub-values Count') }}</td>
            <td class="om-p-cell-left mono">{{ runText.SubCount || 0 }}</td>
          </tr>
          <tr  v-if="!isSucess">
            <td class="om-p-head-left">{{ $t('Sub-values completed') }}</td>
            <td class="om-p-cell-left mono">{{ runText.SubCompleted || 0 }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Started') }}</td>
            <td class="om-p-cell-left mono">{{ createDateTime }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ isSucess ? $t('Completed') : $t('Last Updated on') }}</td>
            <td class="om-p-cell-left mono">{{ lastDateTime }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Duration') }}</td>
            <td class="om-p-cell-left mono">{{ duration }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Run Stamp') }}</td>
            <td class="om-p-cell-left mono">{{ runText.RunStamp }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Run Digest') }}</td>
            <td class="om-p-cell-left mono">{{ runText.RunDigest }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Value Digest') }}</td>
            <td class="om-p-cell-left mono">{{ runText.ValueDigest }}</td>
          </tr>
          <tr v-if="baseSetName">
            <td class="om-p-head-left">{{ $t('Input Scenario') }}</td>
            <td class="om-p-cell-left mono">{{ baseSetName }}</td>
          </tr>
          <tr v-if="baseRunName || baseRunDigest">
            <td class="om-p-head-left">{{ $t('Base Run') }}</td>
            <td class="om-p-cell-left mono">{{ baseRunName }}<br v-if="baseRunName && baseRunDigest" />{{ baseRunDigest }}</td>
          </tr>
        </tbody>
      </table>
    </q-card-section>

    <q-card-section v-if="isCompare" class="text-body1 q-pb-none">
      <table class="om-p-table">
        <tbody>
          <tr v-if="!!compareRuns && compareRuns.length">
            <th class="om-p-head-left" colspan="2">{{ $t('Model runs to compare') }}</th>
          </tr>
          <tr v-for="crt of compareRuns" :key="'cr-' + crt.name">
            <td class="om-p-cell-left">{{ crt.name }}</td>
            <td class="om-p-cell-left om-text-descr">{{ crt.descr }}</td>
          </tr>
          <tr v-if="!!diffParam && diffParam.length">
            <th class="om-p-head-left" colspan="2">{{ $t('Different parameters') }}</th>
          </tr>
          <tr v-else>
            <th class="om-p-head-left" colspan="2">{{ $t('All parameters values identical') }}</th>
          </tr>
          <tr v-for="par of diffParam" :key="'dp-' + par.name">
            <td class="om-p-cell-left">{{ par.name }}</td>
            <td class="om-p-cell-left om-text-descr">{{ par.descr }}</td>
          </tr>
          <tr v-if="!!diffTable && diffTable.length">
            <th class="om-p-head-left" colspan="2">{{ $t('Different output tables') }}</th>
          </tr>
          <tr v-if="(!diffTable || !diffTable.length) && (!suppTable || !suppTable.length)">
            <th class="om-p-head-left" colspan="2">{{ $t('All output tables values identical') }}</th>
          </tr>
          <tr v-for="tbl of diffTable" :key="'dt-' + tbl.name">
            <td class="om-p-cell-left">{{ tbl.name }}</td>
            <td class="om-p-cell-left om-text-descr">{{ tbl.descr }}</td>
          </tr>
          <tr v-if="!!suppTable && suppTable.length">
            <th class="om-p-head-left" colspan="2">{{ $t('Suppressed output tables') }}</th>
          </tr>
          <tr v-for="tbl of suppTable" :key="'ds-' + tbl.name">
            <td class="om-p-cell-left">{{ tbl.name }}</td>
            <td class="om-p-cell-left om-text-descr">{{ tbl.descr }}</td>
          </tr>
          <template v-if="isMicrodata && !!runText.Entity.length">
            <tr v-if="!!diffEntity && diffEntity.length">
              <th class="om-p-head-left" colspan="2">{{ $t('Different microdata') }}</th>
            </tr>
            <tr v-if="(!diffEntity || !diffEntity.length) && (!missEntity || !missEntity.length)">
              <th class="om-p-head-left" colspan="2">{{ $t('All microdata values identical') }}</th>
            </tr>
            <tr v-for="ent of diffEntity" :key="'de-' + ent.name">
              <td class="om-p-cell-left">{{ ent.name }}</td>
              <td class="om-p-cell-left om-text-descr">{{ ent.descr }}</td>
            </tr>
            <tr v-if="!!missEntity && missEntity.length">
              <th class="om-p-head-left" colspan="2">{{ $t('Microdata not found') }}</th>
            </tr>
            <tr v-for="ent of missEntity" :key="'em-' + ent.name">
              <td class="om-p-cell-left">{{ ent.name }}</td>
              <td class="om-p-cell-left om-text-descr">{{ ent.descr }}</td>
            </tr>
          </template>
        </tbody>
      </table>
    </q-card-section>

    <q-card-section v-if="notes" class="text-body1 q-pb-none">
      <div v-html="notes" />
    </q-card-section>

    <q-card-actions align="right">
      <q-btn flat :label="$t('OK')" color="primary" v-close-popup />
    </q-card-actions>

  </q-card>
</q-dialog>
</template>

<script>
import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'
import { marked } from 'marked'
import hljs from 'highlight.js'
import DOMPurify from 'dompurify'

export default {
  name: 'RunInfoDialog',

  props: {
    showTickle: { type: Boolean, default: false },
    modelDigest: { type: String, default: '' },
    runDigest: { type: String, default: '' }
  },

  data () {
    return {
      showDlg: false,
      runText: Mdf.emptyRunText(),
      title: '',
      notes: '',
      isSucess: false,
      statusDescr: '',
      createDateTime: '',
      lastDateTime: '',
      duration: '',
      isDeleted: false,
      isCompare: false,
      baseRunDigest: '',
      baseRunName: '',
      baseSetName: '',
      compareRuns: [],
      diffParam: [],
      diffTable: [],
      suppTable: [],
      diffEntity: [],
      missEntity: []
    }
  },

  computed: {
    isMicrodata () { return !!this.serverConfig.AllowMicrodata && Mdf.entityCount(this.theModel) > 0 },

    ...mapState(useModelStore, [
      'theModel',
      'runTextList'
    ]),
    ...mapState(useServerStateStore, {
      serverConfig: 'config'
    }),
    ...mapState(useUiStateStore, ['runDigestSelected'])
  },

  watch: {
    showTickle () {
      this.runText = this.runTextByDigest({ ModelDigest: this.modelDigest, RunDigest: this.runDigest })
      if (!Mdf.isNotEmptyRunText(this.runText)) {
        console.warn('model run not found by digest:', this.runDigest)
        this.$q.notify({ type: 'negative', message: this.$t('Model run not found: ') + this.runDigest })
        return
      }

      // set basic run info
      this.title = Mdf.descrOfTxt(this.runText) || this.runText.Name
      this.isSucess = this.runText.Status === Mdf.RUN_SUCCESS
      this.statusDescr = this.$t(Mdf.statusText(this.runText.Status))
      this.createDateTime = Mdf.dtStr(this.runText.CreateDateTime)
      this.lastDateTime = Mdf.dtStr(this.runText.UpdateDateTime)
      this.duration = Mdf.toIntervalStr(this.createDateTime, this.lastDateTime)
      this.isDeleted = Mdf.isRunDeletedStatus(this.runText.Status, this.runText.Name)
      this.baseRunDigest = ''
      this.baseRunName = ''
      this.baseSetName = ''

      // if it is a base run of run comparison (if there is not empty list of digests to compare)
      // then make run compare info
      this.isCompare = false
      this.compareRuns = []
      this.diffParam = []
      this.diffTable = []
      this.suppTable = []
      this.diffEntity = []
      this.missEntity = []

      if (this.runDigest === this.runDigestSelected) {
        this.baseRunDigest = Mdf.getRunOption(this.runText.Opts, 'OpenM.BaseRunDigest')
        this.baseRunName = Mdf.getRunOption(this.runText.Opts, 'OpenM.BaseRunName')
        this.baseSetName = Mdf.getRunOption(this.runText.Opts, 'OpenM.SetName')

        const mv = this.modelViewSelected(this.modelDigest)
        this.isCompare = !!mv && Array.isArray(mv?.digestCompareList) && mv.digestCompareList.length > 0

        if (this.isCompare) {
          const rc = Mdf.runCompare(this.runText, mv.digestCompareList, Mdf.tableCount(this.theModel), this.runTextList)

          for (const dg of mv.digestCompareList) {
            const rt = this.runTextList.find(r => r.RunDigest === dg)
            if (rt) {
              this.compareRuns.push({ name: rt.Name, descr: Mdf.descrOfTxt(rt) })
            }
          }
          for (const name of rc.paramDiff) {
            const pt = this.theModel.ParamTxt.find(p => p.Param.Name === name)
            if (pt) {
              this.diffParam.push({ name: pt.Param.Name, descr: Mdf.descrOfDescrNote(pt) })
            }
          }
          for (const name of rc.tableDiff) {
            const tt = this.theModel.TableTxt.find(t => t.Table.Name === name)
            if (tt) {
              this.diffTable.push({ name: tt.Table.Name, descr: (tt.TableDescr || '') })
            }
          }
          for (const name of rc.tableSupp) {
            const tt = this.theModel.TableTxt.find(t => t.Table.Name === name)
            if (tt) {
              this.suppTable.push({ name: tt.Table.Name, descr: (tt.TableDescr || '') })
            }
          }
          if (this.isMicrodata) {
            for (const name of rc.entityDiff) {
              const et = this.theModel.EntityTxt.find(t => t.Entity.Name === name)
              if (et) {
                this.diffEntity.push({ name: et.Entity.Name, descr: Mdf.descrOfDescrNote(et) })
              }
            }
            for (const name of rc.entityMiss) {
              const et = this.theModel.EntityTxt.find(t => t.Entity.Name === name)
              if (et) {
                this.missEntity.push({ name: et.Entity.Name, descr: Mdf.descrOfDescrNote(et) })
              }
            }
          }
        }
      }

      // run notes: convert from markdown to html
      marked.setOptions({
        renderer: new marked.Renderer(),
        highlight: (code, lang) => {
          const language = hljs.getLanguage(lang) ? lang : 'plaintext'
          return hljs.highlight(code, { language }).value
        },
        pedantic: false,
        gfm: true,
        breaks: false,
        smartLists: true
      })
      this.notes = marked.parse(DOMPurify.sanitize(Mdf.noteOfTxt(this.runText)))

      this.showDlg = true
    }
  },

  methods: {
    ...mapActions(useModelStore, ['runTextByDigest']),
    ...mapActions(useUiStateStore, ['modelViewSelected'])
  }
}
</script>

<style scope="local">
  @import 'highlight.js/styles/github.css'
</style>
