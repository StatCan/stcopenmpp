<!-- reload run-text by model digest and run digest -->
<script>
import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'

export default {
  name: 'RefreshRun',

  props: {
    modelDigest: { type: String, default: '' },
    runDigest: { type: String, default: '' },
    refreshTickle: { type: Boolean, default: false },
    refreshRunTickle: { type: Boolean, default: false }
  },

  render () { return null }, // no html

  data () {
    return {
      loadDone: false,
      loadWait: false
    }
  },

  computed: {
    ...mapState(useModelStore, [
      'theModel'
    ]),
    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl'
    }),
    ...mapState(useUiStateStore, [
      'runDigestSelected',
      'uiLang'
    ])
  },

  watch: {
    refreshTickle () { this.doRefresh() },
    refreshRunTickle () { this.doRefresh() }
  },

  emits: ['done', 'wait'],

  methods: {
    ...mapActions(useModelStore, [ 'dispatchRunText' ]),

    // refersh run text
    async doRefresh () {
      if (!this.runDigest) return // exit on empty run digest

      if (!this.modelDigest) {
        console.warn('Unable to refresh model run: model digest or run digest is empty')
        this.$q.notify({ type: 'negative', message: this.$t('Unable to refresh model run: digest is empty') })
        return
      }

      this.loadDone = false
      this.loadWait = true
      this.$emit('wait')
      let rt = Mdf.emptyRunText()

      const u = this.omsUrl +
        '/api/model/' + encodeURIComponent(this.modelDigest) +
        '/run/' + encodeURIComponent(this.runDigest) +
        '/text' + (this.uiLang !== '' ? '/lang/' + encodeURIComponent(this.uiLang) : '')
      try {
        const response = await this.$axios.get(u)
        rt = response.data
        this.loadDone = Mdf.isRunText(rt)
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or model run not found', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or model run not found: ') + this.runDigest })
      }
      this.loadWait = false

      if (this.loadDone) {
        this.dispatchRunText(rt) // on success update run in store
      } else {
        console.warn('Model run not found', this.runDigest)
        this.$q.notify({ type: 'negative', message: this.$t('Model run not found: ') + this.runDigest })

      }
      this.$emit('done', this.loadDone, (rt?.RunDigest ||''))
    }
  },

  mounted () {
    if (!!this.modelDigest && !!this.runDigest) this.doRefresh()
  }
}
</script>
