<!-- reload current model by digest -->
<script>
import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'

export default {
  name: 'RefershModel',

  props: {
    digest: { type: String, default: '' },
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
    ...mapState(useModelStore, [
      'theModel'
    ]),
    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl'
    }),
    ...mapState(useUiStateStore, ['uiLang'])
  },

  watch: {
    refreshTickle () { this.doRefresh() }
  },

  emits: ['done', 'wait'],

  methods: {
    ...mapActions(useModelStore, [
      'dispatchTheModel',
      'dispatchWordList',
      'dispatchLangList'
    ]),

    // refersh current model
    async doRefresh () {
      if (!this.digest) {
        console.warn('Unable to refresh model: digest is empty')
        this.$q.notify({ type: 'negative', message: this.$t('Unable to refresh model: digest is empty') })
        return
      }

      this.loadDone = false
      this.loadWait = true
      this.$emit('wait')
      let dgst = ''
      let isOk = false

      const udgst = encodeURIComponent(this.digest)
      const ulc = encodeURIComponent(this.uiLang)

      const u = this.omsUrl + '/api/model/' + udgst + '/pack/text' + (this.uiLang !== '' ? '/lang/' + ulc : '')
      try {
        const response = await this.$axios.get(u)
        const d = response.data
        if (Mdf.isModel(d)) {
          dgst = Mdf.modelDigest(d)
          isOk = dgst === this.digest
          if (isOk) this.dispatchTheModel(d) // update current model in store
        }
        this.loadDone = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or model not found.', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or model not found: ') + this.digest })
      }

      // on error notify user and exit
      if (!isOk) {
        this.$emit('done', isOk, dgst)
        this.loadWait = false

        console.warn('Unable to refresh model by digest:', this.digest, ':', dgst, ':')
        this.$q.notify({ type: 'negative', message: this.$t('Unable to refresh model by digest: ', this.digest) })
        return
      }

      // refresh model "words" language-specific strings
      const uw = this.omsUrl + '/api/model/' + udgst + '/word-list' + (this.uiLang !== '' ? '/lang/' + ulc : '')
      try {
        const response = await this.$axios.get(uw)
        this.dispatchWordList(response.data) // update model words list in store
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or model words refresh failed', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or model words refresh failed') })
      }

      // refresh list of model languages
      const ul = this.omsUrl + '/api/model/' + udgst + '/lang-list'
      try {
        const response = await this.$axios.get(ul)
        this.dispatchLangList(response.data) // update list of model languages in store
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or languages list refresh failed', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or languages list refresh failed') })
      }
      this.$emit('done', isOk, dgst) // notify user on success
      this.loadWait = false
    }
  },

  mounted () {
    // if model already loaded then exit
    if (!!this.digest && Mdf.modelDigest(this.theModel) === this.digest) {
      this.$emit('done', true, this.digest)
      return
    }
    this.doRefresh() // else load new model
  }
}
</script>
