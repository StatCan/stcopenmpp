<!-- model info bar: show model info in flex bar -->
<template>
  <div
    class="row items-center"
    >

    <q-btn
      @click="onShowModelNote"
      :disable="isNoModel"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs"
      icon="mdi-information"
      :title="$t('About') + ' ' + model.Model.Name"
      />

    <div
      class="col-auto"
      >
      <span>{{ model.Model.Name }}<br />
      <span class="om-text-descr"><span class="mono q-pr-sm">{{ createDateTimeStr }}</span>{{ descrOfModel }}</span></span>
    </div>

  </div>
</template>

<script>
import { mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import * as Mdf from 'src/model-common'

export default {
  name: 'ModelBar',

  props: {
    modelDigest: { type: String, default: '' }
  },

  data () {
    return {
      model: Mdf.emptyModel()
    }
  },

  computed: {
    isNoModel () { return !Mdf.isModel(this.model) || Mdf.isEmptyModel(this.model) },
    createDateTimeStr () { return Mdf.dtStr(this.model.Model.CreateDateTime) },
    descrOfModel () { return (this.model?.Model?.Version ? this.model?.Model?.Version + ' ' : '') + Mdf.modelTitle(this.model) }
  },

  watch: {
    modelDigest () { this.doRefresh() }
  },

  emits: ['model-info-click'],

  methods: {
    ...mapActions(useModelStore, ['modelByDigest']),

    doRefresh () {
      this.model = this.modelByDigest(this.modelDigest)
    },
    onShowModelNote () {
      this.$emit('model-info-click', this.modelDigest)
    }
  },

  mounted () {
    this.doRefresh()
  }
}
</script>

<style lang="scss" scope="local">
</style>
