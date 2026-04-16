<template>
  <q-card-section>

    <div class="row items-center q-pb-xs">

      <q-btn
        v-if="!isHideCancel"
        @click="onCancelNote"
        flat
        dense
        class="col-auto text-white bg-primary rounded-borders q-mr-xs"
        icon="mdi-close-circle"
        :title="$t('Discard changes and stop editing')"
      />
      <q-btn
        v-if="!isHideSave"
        @click="onSaveNote"
        flat
        dense
        class="col-auto text-white bg-primary rounded-borders q-mr-xs"
        icon="mdi-content-save-edit"
        :title="$t('Save description and notes')"
      />
      <q-separator v-if="!isHideCancel || !isHideSave" vertical spaced="sm" color="secondary" />

      <span
        v-if="descriptionEditable"
        class="col-auto q-pr-xs"
        > {{ $t('Description') }}:</span>
      <q-input
        v-if="descriptionEditable"
        v-model="descrEdit"
        maxlength="255"
        size="80"
        @blur="onDescrBlur"
        outlined
        dense
        clearable
        hide-bottom-space
        class="col"
        :placeholder="descrPrompt ? descrPrompt : (theName ? $t('Description of') + ' ' + theName : $t('Description'))"
        :title="descrPrompt ? descrPrompt : (theName ? $t('Description of') + ' ' + theName : $t('Description'))"
      >
      </q-input>
      <div
        v-if="!descriptionEditable"
        class="col om-text-descr-title"
        >{{ theDescr ? theDescr : theName }}</div>

    </div>

    <MdEditor
      v-model="noteEdit"
      :placeholder="notePrompt ? notePrompt : (theName ? $t('Notes for') + ' ' + theName : $t('Notes'))"
      preview-theme="github"
      :language="langCode"
      :no-upload-img="true"
      :code-foldable="false"
      :toolbars-exclude="['image', 'save', 'github']"
      />

    <edit-discard-dialog
      @discard-changes-yes="onYesDiscardChanges"
      :show-tickle="showEditDiscardTickle"
      :dialog-title="$t('Cancel Editing') + '?'"
      >
    </edit-discard-dialog>
  </q-card-section>
</template>

<script>
import EditDiscardDialog from 'components/EditDiscardDialog.vue'
import * as Mdf from 'src/model-common'
import { MdEditor } from 'md-editor-v3'
import DOMPurify from 'dompurify'
import 'md-editor-v3/lib/style.css'

export default {
  name: 'MarkdownEditor',
  components: { MdEditor, EditDiscardDialog },

  props: {
    theKey: { type: String, default: '' },
    theName: { type: String, default: '' },
    theDescr: { type: String, default: '' },
    descrPrompt: { type: String, default: '' },
    descriptionEditable: { type: Boolean, default: false },
    theNote: { type: String, default: '' },
    notePrompt: { type: String, default: '' },
    notesEditable: { type: Boolean, default: false },
    isHideSave: { type: Boolean, default: false },
    isHideCancel: { type: Boolean, default: false },
    langCode: { type: String, default: 'en-US' }
  },

  data () {
    return {
      descrEdit: '',
      noteEdit: '',
      showEditDiscardTickle: false
    }
  },

  emits: ['save-note', 'cancel-note'],

  methods: {
    // cleanup description input
    onDescrBlur (e) {
      this.descrEdit = Mdf.cleanTextInput(this.descrEdit || '')
    },

    // send description and notes to the parent
    onSaveNote () {
      if (this.notesEditable) {
        this.noteEdit = DOMPurify.sanitize(this.noteEdit || '') // remove unsafe html tags
      }
      this.$emit(
        'save-note',
        this.descriptionEditable ? this.descrEdit : this.theDescr,
        this.notesEditable ? this.noteEdit : this.theNote,
        this.isUpdated(),
        this.theKey
      )
    },

    // return true if description or notes updated (edited)
    isUpdated () {
      return (this.descriptionEditable && this.theDescr !== this.descrEdit) ||
        (this.notesEditable && this.theNote !== this.noteEdit)
    },

    // return description and notes
    getDescrNote () {
      if (this.notesEditable) {
        this.noteEdit = DOMPurify.sanitize(this.noteEdit || '') // remove unsafe html tags
      }
      return {
        descr: this.descriptionEditable ? this.descrEdit : this.theDescr,
        note: this.notesEditable ? this.noteEdit : this.theNote,
        isUpdated: this.isUpdated(),
        key: this.theKey
      }
    },

    // cancel editing description and notes
    onCancelNote () {
      if (this.isUpdated()) {
        this.showEditDiscardTickle = !this.showEditDiscardTickle
      } else {
        this.onYesDiscardChanges()
      }
    },
    // notify parent: user answer is "Yes" to "Cancel Editing" pop-up alert
    onYesDiscardChanges () {
      this.$emit('cancel-note')
    }
  },

  // create description and notes editor
  mounted () {
    if (this.descriptionEditable) this.descrEdit = this.theDescr
    if (this.notesEditable) this.noteEdit = this.theNote
  }
}
</script>

<style scoped>
</style>
