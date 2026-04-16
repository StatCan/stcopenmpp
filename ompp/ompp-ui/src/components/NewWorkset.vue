<template>

<q-card-section>
  <table>
  <tbody>
    <tr class="section-title">
      <td>
        <q-btn
          @click="$emit('cancel-new-set')"
          flat
          dense
          class="bg-primary text-white rounded-borders q-mr-xs"
          icon="mdi-close-circle"
          :title="$t('Cancel')"
          />
        <q-btn
          @click="onSaveNewWorkset"
          :disable="isEmptyNewName"
          flat
          dense
          class="bg-primary text-white rounded-borders"
          icon="mdi-content-save"
          :title="$t('Save') + ' ' + (newName || '')"
          />
      </td>
      <td class="bg-primary text-white q-px-md">
        <span>{{ $t('Create new input scenario') }}</span>
      </td>
    </tr>

    <tr>
      <td class="q-pr-xs">
        <span class="text-negative text-weight-bold">* </span><span class="q-pr-sm">{{ $t('Name') }} :</span>
      </td>
      <td>
        <q-input
          debounce="500"
          v-model="newName"
          maxlength="255"
          size="80"
          required
          @focus="onNewNameFocus"
          @blur="onNewNameBlur"
          :rules="[ val => (val || '') !== '' ]"
          outlined
          dense
          clearable
          hide-bottom-space
          :placeholder="$t('Name of the new input scenario') + ' (* ' + $t('Required') + ')'"
          :title="$t('Name of the new input scenario')"
          >
        </q-input>
      </td>
    </tr>
  </tbody>
  </table>

  <markdown-editor
    v-for="t in txtNewWorkset"
    :key="t.LangCode"
    :ref="'new-ws-note-editor-' + t.LangCode"
    :the-key="t.LangCode"
    :the-descr="t.Descr"
    :descr-prompt="$t('Input scenario description') + ' (' + t.LangName + ')'"
    :the-note="t.Note"
    :note-prompt="$t('Input scenario notes') + ' (' + t.LangName + ')'"
    :description-editable="true"
    :notes-editable="true"
    :is-hide-save="true"
    :is-hide-cancel="true"
    :lang-code="t.LangCode"
    class="q-pa-sm"
  >
  </markdown-editor>

</q-card-section>

</template>

<script>
import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import * as Mdf from 'src/model-common'
import MarkdownEditor from 'components/MarkdownEditor.vue'

export default {
  name: 'NewWorkset',
  components: { MarkdownEditor },

  props: {
    digest: { type: String, default: '' },
    refreshTickle: { type: Boolean, default: false }
  },

  data () {
    return {
      newName: '',
      txtNewWorkset: [] // workset description and notes
    }
  },

  computed: {
    isNotEmptyLanguageList () { return Mdf.isLangList(this.langList) },

    // return true if name of new workset is empty after cleanup
    isEmptyNewName () { return (Mdf.cleanFileNameInput(this.newName) || '') === '' },

    ...mapState(useModelStore, [
      'langList',
      'modelLanguage'
    ])
  },

  watch: {
    digest () { this.doRefresh() },
    refreshTickle () { this.doRefresh() }
  },

  emits: ['save-new-set', 'cancel-new-set'],

  methods: {
    ...mapActions(useModelStore, ['isExistInWorksetTextList']),

    // update page view
    doRefresh () {
      // make list of model languages, description and notes for workset editor
      this.txtNewWorkset = []
      if (Mdf.isLangList(this.langList)) {
        for (const lcn of this.langList) {
          this.txtNewWorkset.push({
            LangCode: lcn.LangCode,
            LangName: lcn.Name,
            Descr: '',
            Note: ''
          })
        }
      } else {
        if (!this.txtNewWorkset.length) {
          this.txtNewWorkset.push({
            LangCode: this.modelLanguage.LangCode,
            LangName: this.modelLanguage.Name,
            Descr: '',
            Note: ''
          })
        }
      }
    },

    // set default name of new workset
    onNewNameFocus (e) {
      if (typeof this.newName !== typeof 'string' || (this.newName || '') === '') {
        this.newName = 'New_' + Mdf.dtToUnderscoreTimeStamp(new Date())
      }
    },
    // check if new workset name entered and cleanup input to be compatible with file name rules
    onNewNameBlur (e) {
      const { isEntered, name } = Mdf.doFileNameClean(this.newName)
      if (isEntered && name !== this.newName) {
        this.$q.notify({ type: 'warning', message: this.$t('Scenario name should not contain any of: ') + Mdf.invalidFileNameChars })
      }
      this.newName = isEntered ? name : ''

      if (this.isExistInWorksetTextList({ ModelDigest: this.digest, Name: this.newName })) {
        this.$q.notify({ type: 'negative', message: this.$t('Error: input scenario already exist: ') + (this.newName || '') })
      }
    },

    // validate and save new workset
    onSaveNewWorkset () {
      const name = Mdf.cleanFileNameInput(this.newName)
      if (name === '') {
        this.$q.notify({ type: 'negative', message: this.$t('Invalid (empty) input scenario name') + ((name || '') !== '' ? ' ' + (name || '') : '') })
        return
      }

      // collect description and notes for each language
      const dnLst = []

      for (const t of this.txtNewWorkset) {
        const refKey = 'new-ws-note-editor-' + t.LangCode
        if (!Mdf.isLength(this.$refs[refKey]) || !this.$refs[refKey][0]) continue

        const udn = this.$refs[refKey][0].getDescrNote()
        if ((udn.descr || udn.note || '') !== '') {
          dnLst.push({
            LangCode: t.LangCode,
            Descr: udn.descr,
            Note: udn.note
          })
        }
      }

      // send request to create new workset
      this.$emit('save-new-set', this.digest, name, dnLst)
    }
  },

  mounted () {
    this.doRefresh()
  }
}
</script>

<style lang="scss" scope="local">
  .section-title {
    line-height: 2.375rem;
  }
</style>
