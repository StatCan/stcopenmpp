<!-- reload run-text-list by digest -->
<script>
import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'

export default {
  name: 'RefreshRunList',

  props: {
    digest: { type: String, default: '' },
    refreshTickle: { type: Boolean, default: false },
    refreshRunListTickle: { type: Boolean, default: false }
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
      'runTextList'
    ]),
    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl'
    }),
    ...mapState(useUiStateStore, ['uiLang'])
  },

  watch: {
    refreshTickle () { this.doRefresh() },
    refreshRunListTickle () { this.doRefresh() }
  },

  emits: ['done', 'wait'],

  methods: {
    ...mapActions(useModelStore, ['dispatchRunTextList']),

    // refersh run list
    async doRefresh () {
      if (!this.digest) {
        console.warn('Unable to refresh model runs: digest is empty')
        this.$q.notify({ type: 'negative', message: this.$t('Unable to refresh model runs: digest is empty') })
        return
      }

      this.loadDone = false
      this.loadWait = true
      this.$emit('wait')

      const u = this.omsUrl +
        '/api/model/' + encodeURIComponent(this.digest) +
        '/run-list/text' + (this.uiLang !== '' ? '/lang/' + encodeURIComponent(this.uiLang) : '')
      try {
        const response = await this.$axios.get(u)
        this.dispatchRunTextList(response.data) // update run list in store
        this.loadDone = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('No model runs published (or server offline)', em)
        this.$q.notify({ type: 'negative', message: this.$t('No model runs published (or server offline): ') + this.digest })
      }

      this.$emit('done', this.loadDone)
      this.loadWait = false
    }
  },

  mounted () {
    /*
    // if run list for current model already loaded then exit
    if (!!this.digest && Mdf.isLength(this.runTextList)) {
      if (this.runTextList[0].ModelDigest === this.digest) {
        this.loadDone = true
        this.$emit('done', this.loadDone)
        return
      }
    }
    */
    this.doRefresh() // else load run list
  }
}
</script>
