<!-- refresh status of all model runs which are not completed -->
<script>
import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import * as Mdf from 'src/model-common'

const RUN_REFRESH_TIME = 37 * 1000 // msec, run status refresh interval
const MAX_ERR_MSG = 8 // limit number of console messages

export default {
  name: 'RefreshRunStatusAll',

  props: {
    modelDigest: { type: String, default: '' },
    refreshTickle: { type: Boolean, default: false },
    disableRefresh: { type: Boolean, default: false }
  },

  render () { return null }, // no html

  data () {
    return {
      refreshInt: '',
      lastRefreshDt: 0,
      errCount: 0
    }
  },

  computed: {
    ...mapState(useModelStore, [
      'runTextList'
    ]),
    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl'
    })
  },

  watch: {
    refreshTickle () {
      this.lastRefreshDt = 0
      this.doRefresh()
    },
    disableRefresh (isDisable) { isDisable  ? this.stopRefresh() : this.startRefresh() }
  },

  methods: {
    ...mapActions(useModelStore, ['dispatchRunTextStatusUpdate']),

    // reset state and stop refresh
    stopRefresh () {
      this.lastRefreshDt = 0
      this.errCount = 0
      clearInterval(this.refreshInt)
    },
    // reset state and start refresh
    startRefresh () {
      this.stopRefresh()
      this.refreshInt = setInterval(this.doRefresh, RUN_REFRESH_TIME)
    },

    // for all not completed runs get run status
    async doRefresh () {
      if (!this.modelDigest || Mdf.runTextCount(this.runTextList) <= 0){
        return // no model or no model runs found
      }

      if (Date.now() < this.lastRefreshDt + RUN_REFRESH_TIME) return // protect from timeouts storm

      for (const r of this.runTextList) {
        if (!r?.ModelDigest || r?.ModelDigest !== this.modelDigest || !r?.RunDigest) {
          continue
        }
        const st = r?.Status ?? ''
        if (!!st && !Mdf.isRunCompletedStatus(st)) this.refreshStatus(r?.RunDigest)
      }
      this.lastRefreshDt = Date.now()
    },

    // refresh run status by run digest
    async refreshStatus (rd) {
      if (!this.modelDigest || !rd) return // no model or invalid (empty) run digest

      let rp = Mdf.emptyRunStatusProgress()
      let isOk = false

      const u = this.omsUrl +
        '/api/model/' + encodeURIComponent(this.modelDigest) +
        '/run/' + encodeURIComponent(rd) +
        '/status'

      try {
        const response = await this.$axios.get(u)
        rp = response.data
        this.errCount = 0
        isOk = true
      } catch (e) {
        let em = ''
        this.errCount++
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        if (this.errCount < MAX_ERR_MSG) console.warn('Server offline or model run progress retrieval failed', em)
      }

      // minimal validation of run status response
      if (!isOk || !rp) return
      if (!rp?.ModelDigest || !rp?.RunDigest || !rp?.Status || !rp?.UpdateDateTime) return

      this.dispatchRunTextStatusUpdate(rp) // update run status
    }
  },

  mounted () {
    this.startRefresh()
  },
  beforeUnmount () {
    this.stopRefresh()
  }
}
</script>
