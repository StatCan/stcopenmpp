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
            <td class="om-p-cell-left mono">{{ paramName }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Type') }}</td>
            <td class="om-p-cell-left mono">{{ typeTitle }}</td>
          </tr>
          <tr v-if="paramSize.rank > 1">
            <td class="om-p-head-left">{{ $t('Size') }}</td>
            <td class="om-p-cell-left mono">{{ paramSize.dimSize }} = {{ paramSize.dimTotal }}</td>
          </tr>
          <tr v-if="paramSize.rank == 1">
            <td class="om-p-head-left">{{ $t('Size') }}</td>
            <td class="om-p-cell-left mono">{{ paramSize.dimSize }}</td>
          </tr>
          <tr v-if="paramSize.rank < 1">
            <td class="om-p-head-left">{{ $t('Rank') }}</td>
            <td class="om-p-cell-left mono">{{ paramSize.rank }}</td>
          </tr>
          <tr v-if="(paramRunSet.SubCount || 0) > 1">
            <td class="om-p-head-left">{{ $t('Sub-values Count') }}</td>
            <td class="om-p-cell-left mono">{{ paramRunSet.SubCount || 0 }}</td>
          </tr>
          <tr v-if="(paramRunSet.SubCount || 0) > 1">
            <td class="om-p-head-left">{{ $t('Default Sub Id') }}</td>
            <td class="om-p-cell-left mono">{{ paramRunSet.DefaultSubId || 0 }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Digest') }}</td>
            <td class="om-p-cell-left mono">{{ paramText.Param.Digest }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Import Digest') }}</td>
            <td class="om-p-cell-left mono">{{ paramText.Param.ImportDigest }}</td>
          </tr>
          <tr v-if="(runDigest || '') !== ''">
            <td class="om-p-head-left">{{ $t('Value Digest') }}</td>
            <td class="om-p-cell-left mono">{{ paramRunSet.ValueDigest || $t('Empty') }}</td>
          </tr>
          <tr v-if="docLink">
            <td class="om-p-head-center"></td>
            <td class="om-p-cell-left">
              <a target="_blank" :href="'doc/' + docLink + '#' + paramName" class="file-link"><q-icon name="mdi-book-open" size="md" color="primary" class="q-pr-sm"/>{{ $t('Model Documentation') }}</a>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="paramSize.rank > 0" class="q-pt-md">
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
    </q-card-section>

    <q-card-section v-if="valueNotes" class="text-body1 q-pb-none">
      <div class="q-pt-md" v-html="valueNotes" />
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
  name: 'ParameterInfoDialog',

  props: {
    showTickle: { type: Boolean, default: false },
    paramName: { type: String, default: '' },
    runDigest: { type: String, default: '' },
    worksetName: { type: String, default: '' }
  },

  data () {
    return {
      showDlg: false,
      paramText: Mdf.emptyParamText(),
      title: '',
      notes: '',
      valueNotes: '',
      typeTitle: '',
      paramSize: Mdf.emptyParamSize(),
      dimSizeTxt: [],
      paramRunSet: Mdf.emptyParamRunSet(),
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
      // find parameter in model parameters list
      this.paramText = Mdf.paramTextByName(this.theModel, this.paramName)
      if (!Mdf.isNotEmptyParamText(this.paramText)) {
        console.warn('parameter not found by name:', this.paramName)
        this.$q.notify({ type: 'negative', message: this.$t('Parameter not found: ') + this.paramName })
        return
      }

      // find parameter sub-values info in model run or workset
      this.paramRunSet = Mdf.emptyParamRunSet()
      const mDigest = Mdf.modelDigest(this.theModel)

      if ((this.runDigest || '') !== '') {
        this.paramRunSet = Mdf.paramRunSetByName(
          this.runTextByDigest({ ModelDigest: mDigest, RunDigest: this.runDigest }),
          this.paramName)
      } else {
        if ((this.worksetName || '') !== '') {
          this.paramRunSet = Mdf.paramRunSetByName(
            this.worksetTextByName({ ModelDigest: mDigest, Name: this.worksetName }),
            this.paramName)
        }
      }

      // title: parameter description or name
      this.title = Mdf.descrOfDescrNote(this.paramText) || this.paramText.Param.Name

      // parameter note and value notes: convert from markdown to html
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

      this.notes = marked.parse(DOMPurify.sanitize(Mdf.noteOfDescrNote(this.paramText)))
      this.valueNotes = marked.parse(DOMPurify.sanitize(Mdf.noteOfTxt(this.paramRunSet)))

      // find parameter type
      const t = Mdf.typeTextById(this.theModel, (this.paramText.Param.TypeId || 0))
      this.typeTitle = Mdf.descrOfDescrNote(t)
      if ((this.typeTitle || '') === '') this.typeTitle = t.Type.Name || ''

      // find parameter size info
      this.paramSize = Mdf.paramSizeByName(this.theModel, this.paramName)

      // find dimension names, description, size, type names and description
      this.dimSizeTxt = Array(this.paramSize.rank)

      for (let k = 0; k < this.paramSize.rank; k++) {
        const dt = Mdf.typeTextById(this.theModel, (this.paramText.ParamDimsTxt[k].Dim.TypeId || 0))

        this.dimSizeTxt[k] = {
          name: this.paramText.ParamDimsTxt[k].Dim.Name,
          descr: Mdf.descrOfDescrNote(this.paramText.ParamDimsTxt[k]),
          size: this.paramSize.dimSize[k],
          typeName: dt.Type.Name || '',
          typeDescr: Mdf.descrOfDescrNote(dt)
        }
      }

      // get link to model documentation
      this.docLink = this.serverConfig.IsModelDoc ? Mdf.modelDocLinkByDigest(mDigest, this.modelList, this.uiLang, this.modelLanguage) : ''

      this.showDlg = true
    }
  },

  methods: {
    ...mapActions(useModelStore, [
      'runTextByDigest',
      'worksetTextByName'
    ])
  }
}
</script>

<style scope="local">
  @import 'highlight.js/styles/github.css'
</style>
