<!-- monitor progress of model run: receive run state and run log from the server -->
<script>
import { mapState } from 'pinia'
import { useServerStateStore } from '../stores/server-state'
import * as Mdf from 'src/model-common'

export default {
  name: 'RefreshRunLog',

  props: {
    modelDigest: { type: String, default: '' },
    stamp: { type: String, default: '' },
    refreshTickle: { type: Boolean, default: false },
    start: { type: Number, default: 0 },
    count: { type: Number, default: 0 }
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
    stamp () { this.doRefresh() },
    refreshTickle () { this.doRefresh() }
  },

  emits: ['done', 'wait'],

  methods: {
    // receive run state and log page from the server
    async doRefresh () {
      if (!this.modelDigest || !this.stamp) {
        console.warn('Unable to refresh model run log: digest or run-or-submit stamp is empty')
        this.$q.notify({ type: 'negative', message: this.$t('Unable to refresh model run log: digest or stamp is empty') })
        return
      }

      this.loadDone = false
      this.loadWait = true
      this.$emit('wait')

      // set new run parameters
      let rslp = Mdf.emptyRunStateLog()
      const nStart = (this.start || 0) > 0 ? (this.start || 0) : 0
      const nCount = (this.count || 0) > 0 ? (this.count || 0) : 0

      const u = this.omsUrl +
        '/api/run/log/model/' + encodeURIComponent(this.modelDigest) +
        '/stamp/' + encodeURIComponent(this.stamp) +
        '/start/' + nStart.toString() +
        '/count/' + nCount.toString()
      try {
        const response = await this.$axios.get(u)
        rslp = response.data
        this.loadDone = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or model run log retrieval failed', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or model run log retrieval failed: ') + this.stamp })
      }

      // return state of run log progress
      if (!Mdf.isRunStateLog(rslp)) rslp = Mdf.emptyRunStateLog()

      this.$emit('done', this.loadDone, rslp)
      this.loadWait = false
    }
  },

  mounted () {
  }
}
</script>
