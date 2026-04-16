<template>
<q-dialog v-model="showDlg">
  <q-card class="text-body1">

    <q-card-section v-if="dialogTitle" class="text-h6 bg-primary text-white">{{ dialogTitle }}</q-card-section>

    <q-card-section class="row items-center">
      <q-avatar v-if="iconName" :icon="iconName" color="primary" text-color="white" />
      <span class="q-ml-sm">{{ bodyText }} {{ itemName }}?</span>
    </q-card-section>

    <q-card-section v-if="bodyNote" class="row">
      {{ bodyNote }}
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
        @click="onYesClick"
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
  name: 'ConfirmDialog',

  props: {
    showTickle: { type: Boolean, default: false, required: true },
    itemName: { type: String, default: '', required: true },
    itemId: { type: String, default: '' },
    kind: { type: String, default: '' },
    dialogTitle: { type: String, default: '' },
    bodyText: { type: String, default: '' },
    bodyNote: { type: String, default: '' },
    iconName: { type: String, default: '' }
  },

  data () {
    return {
      showDlg: false
    }
  },

  watch: {
    showTickle () { this.showDlg = true }
  },

  emits: ['confirm-yes'],

  methods: {
    onYesClick () {
      this.$emit('confirm-yes', this.itemName, this.itemId, this.kind)
    }
  }
}
</script>
