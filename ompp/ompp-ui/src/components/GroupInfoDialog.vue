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
            <td class="om-p-head-left">{{ $t('Name') }}:</td>
            <td class="om-p-cell-left mono">{{ groupName }}</td>
          </tr>
          <tr v-if="docLink">
            <td class="om-p-head-center"></td>
            <td class="om-p-cell-left">
              <a target="_blank" :href="'doc/' + docLink + '#' + groupName" class="file-link"><q-icon name="mdi-book-open" size="md" color="primary" class="q-pr-sm"/>{{ $t('Model Documentation') }}</a>
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
  name: 'GroupInfoDialog',

  props: {
    showTickle: { type: Boolean, default: false },
    groupName: { type: String, default: '' }
  },

  data () {
    return {
      showDlg: false,
      title: '',
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
      const groupText = Mdf.groupTextByName(this.theModel, this.groupName)
      this.title = Mdf.descrOfDescrNote(groupText) || this.groupName

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
      this.notes = marked.parse(DOMPurify.sanitize(Mdf.noteOfDescrNote(groupText)))

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
