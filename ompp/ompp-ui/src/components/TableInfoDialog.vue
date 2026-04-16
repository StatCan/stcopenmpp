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
            <td class="om-p-cell-left mono">{{ tableName }}</td>
          </tr>
          <tr v-if="tableSize.rank > 1">
            <td class="om-p-head-left">{{ $t('Size') }}</td>
            <td class="om-p-cell-left mono">{{ tableSize.dimSize }} = {{ tableSize.dimTotal }}</td>
          </tr>
          <tr v-if="tableSize.rank == 1">
            <td class="om-p-head-left">{{ $t('Size') }}</td>
            <td class="om-p-cell-left mono">{{ tableSize.dimSize }}</td>
          </tr>
          <tr v-if="tableSize.rank < 1">
            <td class="om-p-head-left">{{ $t('Rank') }}</td>
            <td class="om-p-cell-left mono">{{ tableSize.rank }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Expressions') }}</td>
            <td class="om-p-cell-left mono">{{ tableSize.exprCount || 0 }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Accumulators') }}</td>
            <td class="om-p-cell-left mono">{{ tableSize.accCount || 0 }}</td>
          </tr>
          <tr v-if="(runText.SubCount || 0) > 1">
            <td class="om-p-head-left">{{ $t('Sub-values Count') }}</td>
            <td class="om-p-cell-left mono">{{ runText.SubCount || 0 }}</td>
          </tr>
          <tr v-if="isRunHasTable">
            <td class="om-p-head-left">{{ $t('Digest') }}</td>
            <td class="om-p-cell-left mono">{{ tableText.Table.Digest }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Import Digest') }}</td>
            <td class="om-p-cell-left mono">{{ tableText.Table.ImportDigest }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Value Digest') }}</td>
            <td class="om-p-cell-left mono">{{ valueDigest }}</td>
          </tr>
          <tr v-if="docLink">
            <td class="om-p-head-center"></td>
            <td class="om-p-cell-left">
              <a target="_blank" :href="'doc/' + docLink + '#' + tableName" class="file-link"><q-icon name="mdi-book-open" size="md" color="primary" class="q-pr-sm"/>{{ $t('Model Documentation') }}</a>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="tableSize.rank > 0" class="q-pt-md">
        <table class="om-p-table">
          <thead>
            <tr>
              <th colspan="2" class="om-p-head-center text-weight-medium">{{ $t('Dimension') }}</th>
              <th class="om-p-head-center text-weight-medium">{{ $t('Size') }}</th>
              <th colspan="2" class="om-p-head-center text-weight-medium">{{ $t('Type of') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="d of dimSizeTxt" :key="d.name">
              <template v-if="!!d.descr && d.name !== d.descr">
                <td class="om-p-cell mono">{{ d.name }}</td>
                <td class="om-p-cell">{{ d.descr }}</td>
              </template>
              <template v-else>
                <td colspan="2" class="om-p-cell mono">{{ d.name }}</td>
              </template>
              <td class="om-p-cell-right">{{ d.size }}</td>
              <template v-if="!!d.typeDescr && d.typeName !== d.typeDescr">
                <td class="om-p-cell mono">{{ d.typeName }}</td>
                <td class="om-p-cell">{{ d.typeDescr }}</td>
              </template>
              <template v-else>
                <td colspan="2" class="om-p-cell mono">{{ d.typeName }}</td>
              </template>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="tableSize.exprCount > 0" class="q-pt-md">
        <table class="om-p-table">
          <thead>
            <tr>
              <th colspan="2" class="om-p-head-center text-weight-medium">{{ $t('Measure') }}</th>
              <th v-if="isAnyExprNote" class="om-p-head-center text-weight-medium">{{ $t('Notes') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="ex of exprTxt" :key="ex.name">
              <template v-if="!!ex.descr && ex.name !== ex.descr">
                <td class="om-p-cell mono">{{ ex.name }}</td>
                <td class="om-p-cell">{{ ex.descr }}</td>
              </template>
              <template v-else>
                <td colspan="2" class="om-p-cell mono">{{ ex.name }}</td>
              </template>
              <td v-if="isAnyExprNote" class="om-p-cell q-pa-sm">
                <span v-if="!!ex.note" v-html="ex.note"></span>
                <span v-else>&nbsp;</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="runDigest && !isRunHasTable" class="q-pt-md">{{ $t('This table is excluded from model run results') }}</div>

    </q-card-section>

    <q-card-section v-if="exprNotes" class="text-body1 q-pb-none">
      <div v-html="exprNotes" />
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
  name: 'TableInfoDialog',

  props: {
    showTickle: { type: Boolean, default: false },
    tableName: { type: String, default: '' },
    runDigest: { type: String, default: '' }
  },

  data () {
    return {
      showDlg: false,
      tableText: Mdf.emptyTableText(),
      title: '',
      notes: '',
      exprNotes: '',
      tableSize: Mdf.emptyTableSize(),
      dimSizeTxt: [],
      exprTxt: [],
      isAnyExprNote: false,
      isRunHasTable: false,
      valueDigest: '',
      runText: Mdf.emptyRunText(),
      docLink: ''
    }
  },

  computed: {
    ...mapState(useModelStore, [
      'theModel',
      'modelList',
      'modelLanguage'
    ]),
    ...mapState(useServerStateStore, {
      serverConfig: 'config'
    }),
    ...mapState(useUiStateStore, ['uiLang'])
  },

  watch: {
    showTickle () {
      // find output tables in model tables list
      this.tableText = Mdf.tableTextByName(this.theModel, this.tableName)
      if (!Mdf.isNotEmptyTableText(this.tableText)) {
        console.warn('output table not found by name:', this.tableName)
        this.$q.notify({ type: 'negative', message: this.$t('Output table not found: ') + this.tableName })
        return
      }

      // find current model run
      const mDigest = Mdf.modelDigest(this.theModel)
      if (this.runDigest) {
        this.runText = this.runTextByDigest({ ModelDigest: mDigest, RunDigest: this.runDigest })
      }

      // title: table description or name
      this.title = this.tableText.TableDescr || this.tableText.Name

      // table note and expression notes: convert from markdown to html
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
      this.notes = marked.parse(DOMPurify.sanitize(this.tableText.TableNote))
      this.exprNotes = marked.parse(DOMPurify.sanitize(this.tableText.ExprNote))

      // find table size info and check is this table included into the run
      this.tableSize = Mdf.tableSizeByName(this.theModel, this.tableName)

      // find dimension names, description, size, type names and description
      this.dimSizeTxt = Array(this.tableSize.rank)

      for (let k = 0; k < this.tableSize.rank; k++) {
        const dt = Mdf.typeTextById(this.theModel, (this.tableText.TableDimsTxt[k].Dim.TypeId || 0))

        this.dimSizeTxt[k] = {
          name: this.tableText.TableDimsTxt[k].Dim.Name,
          descr: Mdf.descrOfDescrNote(this.tableText.TableDimsTxt[k]),
          size: this.tableSize.dimSize[k],
          typeName: dt.Type.Name || '',
          typeDescr: Mdf.descrOfDescrNote(dt)
        }
      }

      this.exprTxt = Array(this.tableSize.exprCount)
      this.isAnyExprNote = false

      for (let k = 0; k < this.tableSize.exprCount; k++) {
        this.exprTxt[k] = {
          name: this.tableText.TableExprTxt[k].Expr.Name,
          descr: Mdf.descrOfDescrNote(this.tableText.TableExprTxt[k]),
          note: marked.parse(DOMPurify.sanitize(Mdf.noteOfDescrNote(this.tableText.TableExprTxt[k])))
        }
        this.isAnyExprNote = this.isAnyExprNote || ((this.exprTxt[k].note || '') !== '')
      }

      if (this.runDigest) {
        const rTbl = Mdf.runTableByName(this.runText, this.tableName)
        this.isRunHasTable = Mdf.isNotEmptyRunTable(rTbl)
        if (this.isRunHasTable) {
          this.valueDigest = rTbl?.ValueDigest || this.$t('Empty')
        }
      }

      // get link to model documentation
      this.docLink = this.serverConfig.IsModelDoc ? Mdf.modelDocLinkByDigest(mDigest, this.modelList, this.uiLang, this.modelLanguage) : ''

      this.showDlg = true
    }
  },

  methods: {
    ...mapActions(useModelStore, [
      'runTextByDigest'
    ])
  }
}
</script>

<style scope="local">
  @import 'highlight.js/styles/github.css'
</style>
