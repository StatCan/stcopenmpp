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
            <td class="om-p-head-left">{{ $t('Entity') }}</td>
            <td class="om-p-cell-left mono">{{ entityName }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Attribute') }}:</td>
            <td class="om-p-cell-left mono">{{ attrName }}</td>
          </tr>
          <template v-if="!!typeDescr && typeName !== typeDescr">
            <tr>
              <td rowspan="2" class="om-p-head-left">{{ $t('Type of') }}:</td>
              <td class="om-p-cell mono">{{ typeName }}</td>
            </tr>
            <tr>
              <td class="om-p-cell">{{ typeDescr }}</td>
            </tr>
          </template>
          <template v-else>
            <tr>
              <td class="om-p-head-left">{{ $t('Type of') }}</td>
              <td class="om-p-cell mono">{{ typeName }}</td>
            </tr>
          </template>
          <tr v-if="isInternal">
            <td class="om-p-head-left"></td>
            <td class="om-p-cell-left mono">{{ $t('Internal attribute') }}</td>
          </tr>
          <tr v-if="docLink">
            <td class="om-p-head-center"></td>
            <td class="om-p-cell-left">
              <a target="_blank" :href="'doc/' + docLink + '#' + attrName" class="file-link"><q-icon name="mdi-book-open" size="md" color="primary" class="q-pr-sm"/>{{ $t('Model Documentation') }}</a>
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
import { mapState } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'
import { marked } from 'marked'
import hljs from 'highlight.js'
import DOMPurify from 'dompurify'

export default {
  name: 'EntityAttrInfoDialog',

  props: {
    showTickle: { type: Boolean, default: false },
    entityName: { type: String, default: '' },
    attrName: { type: String, default: '' }
  },

  data () {
    return {
      showDlg: false,
      title: '',
      typeName: '',
      typeDescr: '',
      isInternal: false,
      notes: '',
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
      const ea = Mdf.entityAttrTextByName(this.theModel, this.entityName, this.attrName)

      this.title = Mdf.descrOfDescrNote(ea) || this.attrName

      if (Mdf.isNotEmptyEntityAttr(ea)) {
        this.isInternal = ea.Attr.IsInternal

        // find attribute type
        const t = Mdf.typeTextById(this.theModel, ea.Attr.TypeId)
        this.typeName = t.Type.Name
        this.typeDescr = Mdf.descrOfDescrNote(t)

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
        })
        this.notes = marked.parse(DOMPurify.sanitize(Mdf.noteOfDescrNote(ea)))
      }

      // get link to model documentation
      this.docLink = this.serverConfig.IsModelDoc ? Mdf.modelDocLinkByDigest(Mdf.modelDigest(this.theModel), this.modelList, this.uiLang, this.modelLanguage) : ''

      this.showDlg = true
    }
  }
}
</script>

<style scope="local">
  @import 'highlight.js/styles/github.css'
</style>
