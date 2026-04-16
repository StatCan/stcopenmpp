<!-- reload workset-text by model digest and workset name -->
<script>
import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'

export default {
  name: 'RefreshWorkset',

  props: {
    modelDigest: { type: String, default: '' },
    worksetName: { type: String, default: '' },
    refreshTickle: { type: Boolean, default: false },
    refreshWorksetTickle: { type: Boolean, default: false },
    isNewRun: { type: Boolean, default: false }
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
    refreshWorksetTickle () { this.doRefresh() }
  },

  emits: ['done', 'wait'],

  methods: {
    ...mapActions(useModelStore, ['dispatchWorksetText']),

    // refersh workset-text by workset name
    async doRefresh () {
      if (!this.worksetName) return // exit on empty workset name

      if (!this.modelDigest) {
        console.warn('Unable to refresh input scenario: model digest or name is empty')
        this.$q.notify({ type: 'negative', message: this.$t('Unable to refresh input scenario: model digest or name is empty') })
        return
      }

      this.loadDone = false
      this.loadWait = true
      const isRun = this.isNewRun
      this.$emit('wait')

      const u = this.omsUrl +
        '/api/model/' + encodeURIComponent(this.modelDigest) +
        '/workset/' + encodeURIComponent(this.worksetName) +
        '/text' + (this.uiLang !== '' ? '/lang/' + encodeURIComponent(this.uiLang) : '')
      try {
        const response = await this.$axios.get(u)
        this.dispatchWorksetText(response.data) // update workset in store
        this.loadDone = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or input scenario not found.', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or input scenario not found: ') + this.worksetName })
      }

      this.$emit('done', this.loadDone, this.worksetName, isRun)
      this.loadWait = false
    }
  },

  mounted () {
    if (!!this.modelDigest && !!this.worksetName) this.doRefresh()
  }
}
</script>
