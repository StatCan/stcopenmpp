<!-- reload array of run-text by model digest and array of run digests -->
<script>
import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'

export default {
  name: 'RefreshRunArray',

  props: {
    modelDigest: { type: String, default: '' },
    runDigestArray: { type: Array, default: () => [] },
    refreshTickle: { type: Boolean, default: false },
    refreshAllTickle: { type: Boolean, default: false }
  },

  render () { return null }, // no html

  data () {
    return {
      loadDone: false,
      loadWait: false
    }
  },

  computed: {
    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl'
    }),
    ...mapState(useUiStateStore, ['uiLang'])
  },

  watch: {
    refreshTickle () { this.doRefresh() },
    refreshAllTickle () { this.doRefresh() }
  },

  emits: ['done', 'wait'],

  methods: {
    ...mapActions(useModelStore, ['dispatchRunText']),

    // refersh run-text array by array of runs digests
    async doRefresh () {
      if (!Array.isArray(this.runDigestArray) || (this.runDigestArray.length || 0) === 0) return // exit if array of runs is empty: nothing to do

      if (!this.modelDigest) {
        console.warn('Unable to refresh model run: model digest or run digest is empty')
        this.$q.notify({ type: 'negative', message: this.$t('Unable to refresh model run: digest is empty') })
        return
      }

      this.loadDone = false
      this.loadWait = true
      let count = 0
      this.$emit('wait')

      for (const rd of this.runDigestArray) {
        if (!rd || typeof rd !== typeof 'string' || rd === '') {
          console.warn('Unable to refresh model run: run digest is empty')
          continue // skip empty run digest
        }

        const u = this.omsUrl +
          '/api/model/' + encodeURIComponent(this.modelDigest) +
          '/run/' + encodeURIComponent(rd) +
          '/text' + (this.uiLang !== '' ? '/lang/' + encodeURIComponent(this.uiLang) : '')
        try {
          const response = await this.$axios.get(u)
          this.dispatchRunText(response.data) // update run in store
          count++
        } catch (e) {
          let em = ''
          try {
            if (e.response) em = e.response.data || ''
          } finally {}
          console.warn('Server offline or model run not found', em)
          this.$q.notify({ type: 'negative', message: this.$t('Server offline or model run not found: ') + rd })
        }
      }

      this.loadDone = true
      this.loadWait = false
      this.$emit('done', this.loadDone, count)
    }
  },

  mounted () {
    this.doRefresh()
  }
}
</script>
