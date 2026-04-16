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
            <td class="om-p-cell-left mono">{{ worksetText.Name }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left">{{ $t('Updated') }}</td>
            <td class="om-p-cell-left mono">{{ lastDateTime }}</td>
          </tr>
          <tr v-if="worksetText.BaseRunDigest">
            <td class="om-p-head-left">{{ $t('Based on run') }}</td>
            <td class="om-p-cell-left mono">{{ worksetText.BaseRunDigest }}</td>
          </tr>
          <tr v-if="paramCount > 0">
            <td class="om-p-head-left">{{ $t('Parameters') }}</td>
            <td class="om-p-cell-left mono">{{ paramCount }}</td>
          </tr>
          <tr>
            <td class="om-p-head-left"></td>
            <td class="om-p-cell-left mono">{{ worksetText.IsReadonly ? $t('Read only') : $t('Read and write') }}</td>
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

  <refresh-workset v-if="(modelDigest || '') !== '' && (wsName || '') !== ''"
    :model-digest="modelDigest"
    :workset-name="wsName"
    :refresh-workset-tickle="refreshWsTickle"
    @done="doneWsLoad"
    @wait="loadWsWait = true"
    >
  </refresh-workset>

</q-dialog>
</template>

<script>
import { mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import * as Mdf from 'src/model-common'
import RefreshWorkset from 'components/RefreshWorkset.vue'
import { marked } from 'marked'
import hljs from 'highlight.js'
import DOMPurify from 'dompurify'

export default {
  name: 'WorksetInfoDialog',
  components: { RefreshWorkset },

  props: {
    showTickle: { type: Boolean, default: false },
    modelDigest: { type: String, default: '' },
    worksetName: { type: String, default: '' }
  },

  data () {
    return {
      showDlg: false,
      worksetText: Mdf.emptyWorksetText(),
      title: '',
      notes: '',
      lastDateTime: '',
      paramCount: 0,
      refreshWsTickle: false,
      loadWsWait: false,
      wsName: ''
    }
  },

  watch: {
    showTickle () {
      this.worksetText = this.worksetTextByName({ ModelDigest: this.modelDigest, Name: this.worksetName })
      if (!Mdf.isNotEmptyWorksetText(this.worksetText)) {
        console.warn('workset not found by name:', this.worksetName)
        this.$q.notify({ type: 'negative', message: this.$t('Input scenario not found: ') + this.worksetName })
        return
      }
      this.wsName = '' // clear refresh workset

      // set basic workset info
      this.title = Mdf.descrOfTxt(this.worksetText) || this.worksetText.Name
      this.lastDateTime = Mdf.dtStr(this.worksetText.UpdateDateTime)

      this.paramCount = Mdf.lengthOf(this.worksetText.Param)
      if (this.paramCount <= 0) this.wsName = this.worksetName // start refresh workset

      // workset notes: convert from markdown to html
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
      this.notes = marked.parse(DOMPurify.sanitize(Mdf.noteOfTxt(this.worksetText)))

      this.showDlg = true
    }
  },

  methods: {
    ...mapActions(useModelStore, ['worksetTextByName']),

    // update workset info on refresh workset completed
    doneWsLoad (isSuccess, name) {
      this.loadWsWait = false
      this.wsName = ''

      if (isSuccess && (name || '') === this.worksetName) {
        const wsText = this.worksetTextByName({ ModelDigest: this.modelDigest, Name: this.worksetName })

        if (Mdf.isNotEmptyWorksetText(wsText)) {
          this.paramCount = Mdf.lengthOf(wsText.Param)
        } else {
          console.warn('workset not found by name:', name)
        }
      }
    }
  }
}
</script>

<style scope="local">
  @import 'highlight.js/styles/github.css'
</style>
