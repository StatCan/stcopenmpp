<!-- reload workset-text-list by digest -->
<script>
import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'

export default {
  name: 'RefreshWorksetList',

  props: {
    digest: { type: String, default: '' },
    refreshTickle: { type: Boolean, default: false },
    refreshWorksetListTickle: { type: Boolean, default: false }
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
      'worksetTextList'
    ]),
    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl'
    }),
    ...mapState(useUiStateStore, ['uiLang'])
  },

  watch: {
    refreshTickle () { this.doRefresh() },
    refreshWorksetListTickle () { this.doRefresh() }
  },

  emits: ['done', 'wait'],

  methods: {
    ...mapActions(useModelStore, ['dispatchWorksetTextList']),

    // refersh workset list
    async doRefresh () {
      if (!this.digest) {
        console.warn('Unable to refresh input scenarios: model digest is empty')
        this.$q.notify({ type: 'negative', message: this.$t('Unable to refresh input scenarios: model digest is empty') })
        return
      }

      this.loadDone = false
      this.loadWait = true
      this.$emit('wait')

      const u = this.omsUrl +
        '/api/model/' + encodeURIComponent(this.digest) +
        '/workset-list/text' + (this.uiLang !== '' ? '/lang/' + encodeURIComponent(this.uiLang) : '')
      try {
        const response = await this.$axios.get(u)
        this.dispatchWorksetTextList(response.data) // update workset list in store
        this.loadDone = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('No input scenarios published (or server offline)', em)
        this.$q.notify({ type: 'negative', message: this.$t('No input scenarios published (or server offline)') + ' ' + this.digest })
      }

      this.$emit('done', this.loadDone)
      this.loadWait = false
    }
  },

  mounted () {
    // if workset list for current model already loaded then exit
    if (!!this.digest && Mdf.isLength(this.worksetTextList)) {
      if (this.worksetTextList[0].ModelDigest === this.digest) {
        this.loadDone = true
        this.$emit('done', this.loadDone)
        return
      }
    }
    this.doRefresh() // else load workset list
  }
}
</script>
