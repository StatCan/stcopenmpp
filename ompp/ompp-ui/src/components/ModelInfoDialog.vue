<template>
  <model-info-base-dialog
    :show-tickle="showTickle"
    :model="model"
    :doc-link="docLink"
    >
  </model-info-base-dialog>
</template>

<script>
import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'
import ModelInfoBaseDialog from 'components/ModelInfoBaseDialog.vue'

export default {
  name: 'ModelInfoDialog',
  components: { ModelInfoBaseDialog },

  props: {
    showTickle: { type: Boolean, default: false },
    digest: { type: String, default: '' }
  },

  data () {
    return {
      showDlg: false,
      model: Mdf.emptyModel(),
      docLink: ''
    }
  },

  computed: {
    ...mapState(useModelStore, [
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
      // get link to model documentation
      this.docLink = ''
      if (this.serverConfig.IsModelDoc) {
        const u = Mdf.modelDocLink(Mdf.modelExtra(md), this.uiLang, this.modelLanguage)
        this.docLink = u || ''
      }

      this.model = md
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
