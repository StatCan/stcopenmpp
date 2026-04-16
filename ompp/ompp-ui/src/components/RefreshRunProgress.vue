<!-- monitor progress of model run: receive run progress from the server -->
<script>
import { mapState } from 'pinia'
import { useServerStateStore } from '../stores/server-state'
import * as Mdf from 'src/model-common'

export default {
  name: 'RefreshRunProgress',

  props: {
    modelDigest: { type: String, default: '' },
    runStamp: { type: String, default: '' },
    refreshTickle: { type: Boolean, default: false }
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
    })
  },

  watch: {
    runStamp () { this.doRefresh() },
    refreshTickle () { this.doRefresh() }
  },

  emits: ['done', 'wait'],

  methods: {
    // receive run status and progress from the server
    async doRefresh () {
      if (!this.modelDigest || !this.runStamp) {
        console.warn('Unable to refresh model run progress: digest or run stamp is empty')
        this.$q.notify({ type: 'negative', message: this.$t('Unable to refresh model run progress: digest or run stamp is empty') })
        return
      }

      this.loadDone = false
      this.loadWait = true
      this.$emit('wait')

      let rpl = [] // set run progress list to empty array initially

      const u = this.omsUrl +
        '/api/model/' + encodeURIComponent(this.modelDigest) +
        '/run/' + encodeURIComponent(this.runStamp) +
        '/status/list'

      try {
        const response = await this.$axios.get(u)
        rpl = response.data
        this.loadDone = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or model run progress retrieval failed', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or model run progress retrieval failed: ') + this.runStamp })
      }

      // return array of run status and progress elements or empty on error
      if (!Mdf.isRunStatusProgressList(rpl)) rpl = []

      this.$emit('done', this.loadDone, rpl)
      this.loadWait = false
    }
  },

  mounted () {
  }
}
</script>
