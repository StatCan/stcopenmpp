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
            <td class="om-p-cell-left mono">{{ modelName }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Version') }}</td>
            <td class="om-p-cell-left mono">{{ version }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Created') }}</td>
            <td class="om-p-cell-left mono">{{ createDateTime }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Digest') }}</td>
            <td class="om-p-cell-left mono">{{ digest }}</td>
          </tr>
          <tr v-if="dir">
            <td class="om-p-head-left">{{ $t('Folder') }}</td>
            <td class="om-p-cell-left mono">{{ dir }}</td>
          </tr>
          <tr v-if="docLink">
            <td class="om-p-head-center"></td>
            <td class="om-p-cell-left">
              <a target="_blank" :href="'doc/' + docLink" class="file-link"><q-icon name="mdi-book-open" size="md" color="primary" class="q-pr-sm"/>{{ $t('Model Documentation') }}</a>
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
  name: 'ModelInfoDialog',

  props: {
    showTickle: { type: Boolean, default: false },
    digest: { type: String, default: '' }
  },

  data () {
    return {
      showDlg: false,
      title: '',
      notes: '',
      modelName: '',
      createDateTime: '',
      version: '',
      dir: '',
      docLink: ''
    }
  },

  computed: {
    ...mapState(useModelStore, [
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
      // find model in model list by digest
      const md = this.modelByDigest(this.digest)
      if (!Mdf.isModel(md)) {
        console.warn('model not found by digest:', this.digest)
        this.$q.notify({ type: 'negative', message: this.$t('Model not found') })
        return
      }

      // set basic model info
      this.title = Mdf.modelTitle(md)
      this.modelName = Mdf.modelName(md)
      this.createDateTime = Mdf.dtStr(md.Model.CreateDateTime)
      this.version = md.Model.Version || ''
      this.dir = Mdf.modelDirByDigest(this.digest, this.modelList)

      // model notes: convert from markdown to html
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
      this.notes = marked.parse(DOMPurify.sanitize(Mdf.noteOfDescrNote(md)))

      // get link to model documentation
      this.docLink = this.serverConfig.IsModelDoc ? Mdf.modelDocLinkByDigest(this.digest, this.modelList, this.uiLang, this.modelLanguage) : ''

      this.showDlg = true
    }
  },

  methods: {
    ...mapActions(useModelStore, ['modelByDigest'])
  }
}
</script>

<style lang="scss" scope="local">
  .file-link {
    text-decoration: none;
  }
</style>

<style scope="local">
  @import 'highlight.js/styles/github.css'
</style>
