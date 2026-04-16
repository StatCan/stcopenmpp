<template>
  <q-dialog v-model="showDlg">
    <q-card class=" text-body1">

      <q-card-section class="text-h6 bg-primary text-white">{{ dialogTitle }}</q-card-section>

      <q-card-section class="q-pt-none">
        <div>{{ $t('Discard all changes?') }}</div>
      </q-card-section>

      <q-card-actions align="right">
        <q-btn
          :label="$t('No')"
          autofocus
          flat
          v-close-popup
          color="primary"
          />
        <q-btn
          :label="$t('Yes')"
          @click="onYesDiscardChanges"
          flat
          v-close-popup
          color="primary"
          />
      </q-card-actions>

    </q-card>
  </q-dialog>
</template>

<script>
export default {
  name: 'EditDiscardDialog',

  props: {
    showTickle: { type: Boolean, default: false, required: true },
    dialogTitle: { type: String, required: true }
  },

  data () {
    return {
      showDlg: false
    }
  },

  watch: {
    showTickle () { this.showDlg = true }
  },

  emits: ['discard-changes-yes'],

  methods: {
    onYesDiscardChanges () {
      this.$emit('discard-changes-yes')
    }
  }
}
</script>
