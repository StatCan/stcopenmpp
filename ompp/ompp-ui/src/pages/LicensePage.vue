<!-- display openM++ license -->
<template>
<q-page class="text-body1">

  <div id="license-page" class="mono q-pa-md">
    {{ msg }}
  </div>

</q-page>
</template>

<script>
export default {
  name: 'LicensePage',

  props: {
    refreshTickle: { type: Boolean, default: false }
  },

  watch: {
    // refresh button handler
    refreshTickle () { this.doRefresh() }
  },

  data () {
    return {
      loadDone: false,
      loadWait: false,
      msg: ''
    }
  },

  methods: {
    // refersh page content
    async doRefresh () {
      this.loadDone = false
      this.loadWait = true
      this.msg = this.$t('Loading...')
      try {
        const response = await this.$axios.get('LICENSE.txt')
        this.msg = response.data
        this.loadDone = true
      } catch (e) {
        let em = ''
        this.msg = this.$t('Server offline or LICENSE.txt not found.')
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or LICENSE.txt not found.', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or LICENSE.txt not found.') })
      }
      this.loadWait = false
    }
  },

  mounted () {
    this.doRefresh()
  }
}
</script>

<style lang="scss" scoped>
  #license-page {
    min-width: 100px;
    min-height: 100px;
    white-space: pre;
  }
</style>
