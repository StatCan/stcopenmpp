<!-- delete workset by model digest and workset name -->
<script>
import { mapState } from 'pinia'
import { useServerStateStore } from '../stores/server-state'

export default {
  name: 'DeleteWorkset',

  props: {
    modelDigest: { type: String, default: '' },
    worksetName: { type: String, default: '' },
    deleteNow: { type: Boolean, default: false }
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
    deleteNow () { if (this.deleteNow) this.doDelete() }
  },

  emits: ['done', 'wait'],

  methods: {
    // delete workset by model digest and workset name
    async doDelete () {
      if (!this.modelDigest || !this.worksetName) {
        console.warn('Unable to delete input scenario: model digest or scenario name is empty')
        this.$emit('done', false, this.modelDigest, this.worksetName)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to delete input scenario: model digest or scenario name is empty') })
        return
      }

      this.loadDone = false
      this.loadWait = true
      this.$emit('wait')
      this.$q.notify({ type: 'info', message: this.$t('Deleting: ') + this.worksetName })

      const u = this.omsUrl +
        '/api/model/' + encodeURIComponent(this.modelDigest) +
        '/workset/' + encodeURIComponent((this.worksetName || ''))
      try {
        await this.$axios.delete(u) // response expected to be empty on success
        this.loadDone = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Error at delete workset', this.worksetName, em)
      }
      this.loadWait = false

      if (this.loadDone) {
        this.$q.notify({ type: 'info', message: this.$t('Deleted: ') + this.worksetName })
      } else {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to delete: ') + this.worksetName })
      }

      this.$emit('done', this.loadDone, this.modelDigest, this.worksetName)
    }
  },

  mounted () {
  }
}
</script>
