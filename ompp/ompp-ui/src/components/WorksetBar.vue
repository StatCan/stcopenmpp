<!-- workset info bar: show workset (a.k.a. input secenario) info in flex bar -->
<template>
  <div
    class="row items-center"
    >

    <q-btn
      v-if="isShowMenu"
      outline
      dense
      class="col-auto text-primary rounded-borders q-mr-xs"
      icon="menu"
      :title="$t('Menu')"
      :aria-label="$t('Menu')"
      >
      <q-menu auto-close>
        <q-list>

          <q-item
            @click="onShowWorksetNote"
            :disable="!isNotEmptyWorkset"
            clickable
            >
            <q-item-section avatar>
              <q-icon
                color="primary"
                name="mdi-information-outline"
                />
            </q-item-section>
            <q-item-section>{{ $t('About') + ' ' + worksetName }}</q-item-section>
          </q-item>
          <q-separator />

          <q-item
            v-if="isNewRunButton"
            @click="onNewRunClick"
            :disable="!isNotEmptyWorkset || !isReadonlyWorkset"
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" name="mdi-run" />
            </q-item-section>
            <q-item-section>{{ $t('Run the Model') }}</q-item-section>
          </q-item>
          <q-item
            v-if="isReadonlyButton"
            :disable="!isNotEmptyWorkset || isReadonlyDisabled"
            @click="onWorksetReadonlyToggle"
            clickable
            >
            <q-item-section avatar>
              <q-icon color="primary" :name="(!isNotEmptyWorkset || isReadonlyWorkset) ? 'mdi-lock' : 'mdi-lock-open-variant'" />
            </q-item-section>
            <q-item-section>{{ ((!isNotEmptyWorkset || isReadonlyWorkset) ? $t('Open to edit scenario') : $t('Close to run scenario')) + ' ' + worksetName }}</q-item-section>
          </q-item>

        </q-list>
      </q-menu>
    </q-btn>

    <q-btn
      @click="onShowWorksetNote"
      :disable="!isNotEmptyWorkset"
      flat
      dense
      class="col-auto text-white rounded-borders bg-primary"
      icon="mdi-information"
      :title="$t('About') + ' ' + worksetName"
      />
    <q-separator v-if="isShowMenu" vertical inset spaced="sm" color="secondary" />

    <q-btn
      v-if="isNewRunButton"
      @click="onNewRunClick"
      :disable="!isNotEmptyWorkset || !isReadonlyWorkset"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders"
      icon="mdi-run"
      :title="$t('Run the Model')"
      />
    <q-btn
      v-if="isReadonlyButton"
      :disable="!isNotEmptyWorkset || isReadonlyDisabled"
      @click="onWorksetReadonlyToggle"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-ml-xs"
      :icon="(!isNotEmptyWorkset || isReadonlyWorkset) ? 'mdi-lock' : 'mdi-lock-open-variant'"
      :title="((!isNotEmptyWorkset || isReadonlyWorkset) ? $t('Open to edit scenario') : $t('Close to run scenario')) + ' ' + worksetName"
      />

    <div
      class="col-auto q-ml-xs"
      >
      <span>{{ worksetName }}<br />
      <span class="om-text-descr"><span class="mono q-pr-sm">{{ lastDateTimeStr }}</span>{{ descrOfWorkset }}</span></span>
    </div>

  </div>
</template>

<script>
import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import * as Mdf from 'src/model-common'

export default {
  name: 'WorksetBar',

  props: {
    modelDigest: { type: String, default: '' },
    worksetName: { type: String, default: '' },
    refreshworksetTickle: { type: Boolean, default: false },
    isNewRunButton: { type: Boolean, default: false },
    isReadonlyButton: { type: Boolean, default: false },
    isReadonlyDisabled: { type: Boolean, default: false },
    isShowMenu: { type: Boolean, default: false }
  },

  data () {
    return {
      worksetText: Mdf.emptyWorksetText()
    }
  },

  computed: {
    isNotEmptyWorkset () { return Mdf.isNotEmptyWorksetText(this.worksetText) },
    lastDateTimeStr () { return Mdf.dtStr(this.worksetText.UpdateDateTime) },
    descrOfWorkset () { return Mdf.descrOfTxt(this.worksetText) },

    // if true then workset is read-only and model run enabled
    isReadonlyWorkset () {
      return Mdf.isNotEmptyWorksetText(this.worksetText) && this.worksetText.IsReadonly
    },

    ...mapState(useModelStore, [
      'worksetTextListUpdated'
    ]),
    ...mapState(useServerStateStore, {
      serverConfig: 'config'
    })
  },

  watch: {
    modelDigest () { this.doRefresh() },
    worksetName () { this.doRefresh() },
    refreshworksetTickle () { this.doRefresh() },
    worksetTextListUpdated () { this.doRefresh() }
  },

  emits: ['set-info-click', 'new-run-select', 'set-update-readonly'],

  methods: {
    ...mapActions(useModelStore, ['worksetTextByName']),

    doRefresh () {
      this.worksetText = this.worksetTextByName({ ModelDigest: this.modelDigest, Name: this.worksetName })
    },
    onShowWorksetNote () {
      this.$emit('set-info-click', this.modelDigest, this.worksetName)
    },
    // new model run using current workset name: open model run tab
    onNewRunClick () {
      this.$emit('new-run-select', this.worksetName)
    },
    // toggle current workset readonly status: pass event from child up to the next level
    onWorksetReadonlyToggle () {
      this.$emit('set-update-readonly', this.modelDigest, this.worksetName, !this.worksetText.IsReadonly)
    }
  },

  mounted () {
    this.doRefresh()
  }
}
</script>

<style lang="scss" scope="local">
</style>
