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
            <td class="om-p-head-left">{{ $t('Entity Name') }}</td>
            <td class="om-p-cell-left mono">{{ entityName }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Attributes Count') }}</td>
            <td class="om-p-cell-left mono">{{ totalAttrCount }}</td>
          </tr>
          <tr v-if="internalAttrCount">
            <td class="om-p-head-left">{{ $t('Internal Attributes') }}</td>
            <td class="om-p-cell-left mono">{{ internalAttrCount }}</td>
          </tr>
          <template v-if="runDigest">
            <tr>
              <td class="om-p-head-left">{{ $t('Run Attributes Count') }}</td>
              <td class="om-p-cell-left mono">{{ runAttrCount }}</td>
            </tr>
            <tr v-if="internalRunAttrCount || internalAttrCount">
              <td class="om-p-head-left">{{ $t('Run Internal Attributes') }}</td>
              <td class="om-p-cell-left mono">{{ internalRunAttrCount }}</td>
            </tr>
            <tr>
              <td class="om-p-head-left">{{ $t('Microdata Count') }}</td>
              <td class="om-p-cell-left mono">{{ rowCount }}</td>
            </tr>
          </template>
          <tr v-if="docLink">
            <td class="om-p-head-center"></td>
            <td class="om-p-cell-left">
              <a target="_blank" :href="'doc/' + docLink + '#' + entityName" class="file-link"><q-icon name="mdi-book-open" size="md" color="primary" class="q-pr-sm"/>{{ $t('Model Documentation') }}</a>
            </td>
          </tr>
        </tbody>
      </table>
    </q-card-section>

    <q-card-section class="text-body1 q-pb-none">
      <table class="om-p-table">
          <thead>
            <tr>
              <th class="om-p-head-center text-weight-medium">{{ $t('Attribute') }}</th>
              <th class="om-p-head-center text-weight-medium">{{ $t('Type of') }}</th>
              <th class="om-p-head-center text-weight-medium">{{ $t('Description and Notes') }}</th>
            </tr>
          </thead>
        <tbody>

          <tr v-for="ea in attrs" :key="'ea-' + (ea.name || 'no-name')">
            <td class="om-p-cell">{{ ea.name }}</td>
            <td class="om-p-cell">
              <span class="mono">{{ ea.typeName }}</span>
              <template v-if="ea.isInternal">
                <br/>
                <span>{{ $t('Internal attribute') }}</span>
              </template>
            </td>
            <td class="om-p-cell">
              <span>{{ ea.descr }}</span>
              <template v-if="ea.notes">
                <div class="om-text-descr" v-html="ea.notes" />
              </template>
            </td>
          </tr>

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
  name: 'EntityInfoDialog',

  props: {
    showTickle: { type: Boolean, default: false },
    entityName: { type: String, default: '' },
    runDigest: { type: String, default: '' }
  },

  data () {
    return {
      showDlg: false,
      title: '',
      totalAttrCount: 0,
      internalAttrCount: 0,
      runAttrCount: 0,
      internalRunAttrCount: 0,
      rowCount: 0,
      notes: '',
      runCurrent: Mdf.emptyRunText(), // currently selected run
      attrs: [],
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
      const entText = Mdf.entityTextByName(this.theModel, this.entityName)
      this.title = Mdf.descrOfDescrNote(entText) || this.entityName

      // if run specified then include only entity.attribute from this model run
      const aUse = {}
      if (this.runDigest) {
        this.runCurrent = this.runTextByDigest({ ModelDigest: Mdf.modelDigest(this.theModel), RunDigest: this.runDigest })
      }
      const isRun = this.runDigest && Mdf.isNotEmptyRunText(this.runCurrent)

      this.rowCount = 0
      if (isRun) {
        for (const e of this.runCurrent.Entity) {
          if (e?.Name && e.Name === this.entityName && Array.isArray(e?.Attr)) {
            for (const a of e.Attr) {
              aUse[a] = true
            }
            this.rowCount = e?.RowCount || 0 // if this is microdata of model run then show microdata row count
          }
        }
      }

      // make attriburtes list: name, type, description, notes, isInternal bool flag
      // count regular attributes and internal attributes
      if (entText?.EntityAttrTxt && Array.isArray(entText?.EntityAttrTxt)) {
        this.totalAttrCount = 0
        this.internalAttrCount = 0
        this.runAttrCount = 0
        this.internalRunAttrCount = 0
        this.attrs = []

        for (const ea of entText.EntityAttrTxt) {
          this.totalAttrCount++
          const isInt = ea?.Attr?.IsInternal

          if (isInt) this.internalAttrCount++

          if (isRun && !aUse[ea?.Attr?.Name]) continue // skip: this attribute not used for the run microdata
          if (isRun) {
            this.runAttrCount++
            if (isInt) this.internalRunAttrCount++
          }

          // find attribute type
          const t = Mdf.typeTextById(this.theModel, ea.Attr.TypeId)
          this.typeName = t.Type.Name

          this.attrs.push({
            name: ea.Attr.Name,
            isInternal: isInt,
            typeName: t.Type.Name,
            descr: Mdf.descrOfDescrNote(ea),
            notes: marked.parseInline(DOMPurify.sanitize(Mdf.noteOfDescrNote(ea)))
          })
        }
      }

      // notes: convert from markdown to html
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
        // smartypants: true
      })
      this.notes = marked.parse(DOMPurify.sanitize(Mdf.noteOfDescrNote(entText)))

      // get link to model documentation
      this.docLink = this.serverConfig.IsModelDoc ? Mdf.modelDocLinkByDigest(Mdf.modelDigest(this.theModel), this.modelList, this.uiLang, this.modelLanguage) : ''

      this.showDlg = true
    }
  },

  methods: {
    ...mapActions(useModelStore, ['runTextByDigest'])
  }
}
</script>

<style scope="local">
  @import 'highlight.js/styles/github.css'
</style>
