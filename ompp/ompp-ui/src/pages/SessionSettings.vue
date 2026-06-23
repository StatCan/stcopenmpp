<template>
<q-page class="text-body1 q-pa-sm">

  <q-expansion-item
    switch-toggle-side
    default-opened
    header-class="bg-primary text-white"
    class="q-mb-xs"
    :label="$t('Session state and settings')"
    >
    <table class="settings-table q-mb-sm">
      <tbody>

        <tr>
          <td class="om-light-border q-pa-sm">
            <q-btn
              @click="onModelListClear"
              :disable="!modelCount"
              flat
              dense
              class="bg-primary text-white rounded-borders"
              icon="cancel"
              :title="$t('Clear models list')"
              />
          </td>
          <td class="om-light-border q-pa-sm om-text-secondary">{{ $t('List of Models') }}:</td>
          <td class="om-light-border q-pa-sm">{{ modelCount ? modelCount.toLocaleString() + ' ' + $t('model(s)') : $t('Empty') }}</td>
        </tr>

        <tr>
          <td class="om-light-border q-pa-sm">
            <q-btn
              @click="onModelClear"
              :disable="!isModel"
              flat
              dense
              class="bg-primary text-white rounded-borders"
              icon="cancel"
              :title="$t('Clear model')"
              />
          </td>
          <td class="om-light-border q-pa-sm om-text-secondary">{{ $t('Current Model') }}:</td>
          <td class="om-light-border q-pa-sm">{{ modelTitle }}</td>
        </tr>

        <tr>
          <td class="om-light-border q-pa-sm">
            <q-btn
              @click="onRunTextListClear"
              :disable="!runCount"
              flat
              dense
              class="bg-primary text-white rounded-borders"
              icon="cancel"
              :title="$t('Clear list of model runs')"
              />
          </td>
          <td class="om-light-border q-pa-sm om-text-secondary">{{ $t('Model Runs') }}:</td>
          <td class="om-light-border q-pa-sm">{{ runCount ? runCount.toLocaleString() + ' ' + $t('run result(s)') : $t('Empty') }}</td>
        </tr>

        <tr>
          <td class="om-light-border q-pa-sm">
            <q-btn
              @click="onWorksetTextListClear"
              :disable="!worksetCount"
              flat
              dense
              class="bg-primary text-white rounded-borders"
              icon="cancel"
              :title="$t('Clear list of input scenarios')"
              />
          </td>
          <td class="om-light-border q-pa-sm om-text-secondary">{{ $t('Input Scenarios') }}:</td>
          <td class="om-light-border q-pa-sm">{{ worksetCount ? worksetCount.toLocaleString() + ' ' + $t('scenario(s)') : $t('Empty') }}</td>
        </tr>

        <tr>
          <td class="om-light-border q-pa-sm">
            <q-btn
              @click="onUiLanguageClear"
              :disable="!uiLang"
              flat
              dense
              class="bg-primary text-white rounded-borders"
              icon="cancel"
              :title="$t('Reset language to default value')"
              />
          </td>
          <td class="om-light-border q-pa-sm om-text-secondary">{{ $t('Language') }}:</td>
          <td class="om-light-border q-pa-sm">{{ uiLangTitle }} <span class="om-text-secondary">[{{ this.$q.lang.getLocale() }}]</span></td>
        </tr>

        <tr>
          <td colspan="2" class="om-light-border q-pa-sm om-text-secondary">{{ $t('Model Downloads') }}:</td>
          <td class="om-light-border q-pa-sm">
            <q-radio v-model="fastDownload" val="yes" :disable="!serverConfig.AllowDownload" :label="$t('Fast, only to analyze output values')" />
            <br />
            <q-radio v-model="fastDownload" val="no" :disable="!serverConfig.AllowDownload" :label="$t('Full, compatible with desktop model')" />
            <template v-if="!!serverConfig.AllowMicrodata">
              <br />
              <q-checkbox v-model="isMicroDownload" :disable="!serverConfig.AllowDownload || fastDownload === 'yes'" :label="$t('Do full downloads, including microdata')" />
            </template>
            <br />
            <q-checkbox v-model="isIdCsvDownload" :disable="!serverConfig.AllowDownload" :label="$t('Download with dimension items ID instead of code')" />
          </td>
        </tr>
        <tr>
          <td colspan="2" class="om-light-border q-pa-sm om-text-secondary">{{ $t('Tree Labels') }}:</td>
          <td class="om-light-border q-pa-sm">
            <q-radio v-model="labelKind" val="name-only"  :label="$t('Show only name')" />
            <br />
            <q-radio v-model="labelKind" val="descr-only" :label="$t('Show only description')" />
            <br />
            <q-radio v-model="labelKind" val="default" :label="$t('Name and description')" />
          </td>
        </tr>

      </tbody>
    </table>
  </q-expansion-item>

  <q-expansion-item
    switch-toggle-side
    default-opened
    header-class="bg-primary text-white"
    :label="isModel ? $t('Model views of') + ' ' + modelTitle : $t('Model views')"
    >
    <q-card>

      <q-card-section class="row items-center q-py-sm">
        <span class="col-auto">
          <q-btn
            @click="onDownloadViews"
            :disable="!isModel || (!paramNames.length && !tableNames.length && !entityNames.length)"
            flat
            dense
            no-caps
            align="left"
            class="bg-primary text-white rounded-borders"
            icon="mdi-download"
            :title="isModel ? $t('Download views of') + ' ' + modelTitle : $t('Download model views')"
            />
        </span>
        <span
          :class="{ 'om-text-secondary' : !isModel || (!paramNames.length && !tableNames.length && !entityNames.length) }"
          class="col-grow q-pl-sm"
        >{{ isModel ? $t('Download views of') + ' ' + modelTitle : $t('Download model views') }}</span>
      </q-card-section>

      <q-card-section class="row items-center q-pt-sm q-pb-md">
          <span class="col-auto">
            <q-btn
              @click="onUploadViews"
              :disable="!isModel || !isUploadFile || isReadonlyServer"
              flat
              dense
              class="bg-primary text-white rounded-borders"
              icon="mdi-upload"
              :title="$t('Upload views of') + (isModel ? (' ' + modelTitle) : '')"
              />
          </span>
          <span class="col-grow q-pl-sm">
            <q-file
              v-model="uploadFile"
              :disable="!isModel || isReadonlyServer"
              accept=".view.json"
              outlined
              dense
              clearable
              hide-bottom-space
              color="primary"
              :label="isModel ? $t('Upload views of') + ' ' + modelTitle : $t('Upload model views')"
              >
            </q-file>
          </span>
      </q-card-section>

      <q-card-section
        class="row items-center bg-primary text-white q-py-sm q-mx-md"
        :class="{ 'om-bg-inactive' : !paramNames.length }"
        >
        <span class="col-grow">{{ $t('Default views of parameters: ') + (paramNames.length ? paramNames.length.toLocaleString() : $t('None')) }}</span>
      </q-card-section>
      <q-list
        class="q-mb-xs"
        >
        <q-item v-for="pName of paramNames" :key="pName">
          <q-item-section avatar>
            <q-btn
             :disable="isReadonlyServer"
              @click="onRemoveParamView(pName)"
              flat
              dense
              class="bg-primary text-white rounded-borders"
              icon="delete"
              :title="$t('Erase default view of parameter') + ' ' + pName"
              />
          </q-item-section>
          <q-item-section>
            <q-item-label>{{ pName }}</q-item-label>
            <q-item-label caption>{{ parameterDescr(pName) }}</q-item-label>
          </q-item-section>
        </q-item>
      </q-list>

      <q-card-section
        class="row items-center bg-primary text-white q-py-sm q-mx-md"
        :class="{ 'om-bg-inactive' : !tableNames.length }"
        >
        <span class="col-grow">{{ $t('Default views of output tables: ') + (tableNames.length ? tableNames.length.toLocaleString() : $t('None')) }}</span>
      </q-card-section>
      <q-list
        class="q-mb-xs"
        >
        <q-item v-for="tName of tableNames" :key="tName">
          <q-item-section avatar>
            <q-btn
              :disable="isReadonlyServer"
              @click="onRemoveTableView(tName)"
              flat
              dense
              class="bg-primary text-white rounded-borders"
              icon="delete"
              :title="$t('Erase default view of output table') + ' ' + tName"
              />
          </q-item-section>
          <q-item-section>
            <q-item-label>{{ tName }}</q-item-label>
            <q-item-label caption>{{ tableDescr(tName) }}</q-item-label>
          </q-item-section>
        </q-item>
      </q-list>

      <template v-if="!!serverConfig.AllowMicrodata">
        <q-card-section
          class="row items-center bg-primary text-white q-py-sm q-mx-md"
          :class="{ 'om-bg-inactive' : !entityNames.length }"
          >
          <span class="col-grow">{{ $t('Default microdata views: ') + (entityNames.length ? entityNames.length.toLocaleString() : $t('None')) }}</span>
        </q-card-section>
        <q-list
          class="q-mb-xs"
          >
          <q-item v-for="eName of entityNames" :key="eName">
            <q-item-section avatar>
              <q-btn
                :disable="isReadonlyServer"
                @click="onRemoveMicrodataView(eName)"
                flat
                dense
                class="bg-primary text-white rounded-borders"
                icon="delete"
                :title="$t('Erase default microdata view') + ' ' + eName"
                />
            </q-item-section>
            <q-item-section>
              <q-item-label>{{ eName }}</q-item-label>
              <q-item-label caption>{{ entityDescr(eName) }}</q-item-label>
            </q-item-section>
          </q-item>
        </q-list>
      </template>

    </q-card>
  </q-expansion-item>

</q-page>
</template>

<script>
import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'
import * as Idb from 'src/idb/idb'
import { exportFile } from 'quasar'

export default {
  name: 'SessionSettings',

  data () {
    return {
      dbRows: [], // user views: parameter views or output table views
      paramNames: [], // parameter names from dbRows
      tableNames: [], // output table names from dbRows
      entityNames: [], // microdata entity names from dbRows
      uploadFile: null,
      fastDownload: 'yes',
      isMicroDownload: false,
      isIdCsvDownload: false,
      labelKind: 'default'
    }
  },

  computed: {
    uiLangTitle () { return this.uiLang !== '' ? this.uiLang : this.$t('Default') },
    isModel () { return Mdf.isModel(this.theModel) },
    modelName () { return Mdf.modelName(this.theModel) },
    modelTitle () { return Mdf.isModel(this.theModel) ? Mdf.modelTitle(this.theModel) : this.$t('Not selected') },
    runCount () { return Mdf.runTextCount(this.runTextList) },
    worksetCount () { return Mdf.worksetTextCount(this.worksetTextList) },
    isUploadFile () {
      return this.uploadFile instanceof File && (this.uploadFile?.name || '') !== '' && this.uploadFile?.type === 'application/json'
    },
    isReadonlyServer () { return this.serverConfig?.IsReadonly ?? true },

    ...mapState(useModelStore, [
      'theModel',
      'runTextList',
      'worksetTextList',
      'modelCount'
    ]),
    ...mapState(useServerStateStore, {
      serverConfig: 'config'
    }),
    ...mapState(useUiStateStore, [
      'uiLang',
      'treeLabelKind',
      'noAccDownload',
      'noMicrodataDownload',
      'idCsvDownload'
    ])
  },

  watch: {
    fastDownload (val) {
      this.dispatchNoAccDownload(val === 'yes')
      if (val === 'yes' || !this.serverConfig.AllowMicrodata) {
        this.isMicroDownload = false
        this.dispatchNoMicrodataDownload(!this.isMicroDownload)
      }
    },
    isMicroDownload () {
      this.dispatchNoMicrodataDownload(!this.isMicroDownload)
    },
    isIdCsvDownload (isIdCsv) {
      this.dispatchIdCsvDownload(isIdCsv)
    },
    labelKind (val) {
      this.dispatchTreeLabelKind((val === 'name-only' || val === 'descr-only') ? val : '')
    }
  },

  emits: ['user-view-updated'],

  methods: {
    ...mapActions(useModelStore, [
      'dispatchTheModel',
      'dispatchModelList',
      'dispatchRunTextList',
      'dispatchWorksetTextList'
    ]),
    ...mapActions(useUiStateStore, [
      'dispatchUiLang',
      'dispatchNoAccDownload',
      'dispatchNoMicrodataDownload',
      'dispatchIdCsvDownload',
      'dispatchTreeLabelKind',
      'dispatchViewDeleteByModel'
    ]),
    // refresh model settings: select from indexed db
    doRefresh () {
      this.clearState()
      this.fastDownload = this.noAccDownload ? 'yes' : 'no'
      this.isMicroDownload = !this.noAccDownload && !this.noMicrodataDownload && !!this.serverConfig.AllowMicrodata
      this.isIdCsvDownload = this.serverConfig.AllowDownload && this.idCsvDownload
      this.labelKind = (this.treeLabelKind === 'name-only' || this.treeLabelKind === 'descr-only') ? this.treeLabelKind : 'default'

      if (this.modelName) this.doReadViews()
    },

    onModelClear () {
      const digest = Mdf.modelDigest(this.theModel)
      this.clearState()
      this.dispatchViewDeleteByModel(digest)
      this.dispatchTheModel(Mdf.emptyModel())
    },
    onModelListClear () {
      const digest = Mdf.modelDigest(this.theModel)
      this.clearState()
      this.dispatchViewDeleteByModel(digest)
      this.dispatchModelList([])
    },
    clearState () {
      this.dbRows = []
      this.paramNames = []
      this.tableNames = []
      this.entityNames = []
      this.uploadFile = null
    },
    onRunTextListClear () { this.dispatchRunTextList([]) },
    onWorksetTextListClear () { this.dispatchWorksetTextList([]) },
    onUiLanguageClear () { this.dispatchUiLang('') },

    // return parameter description by name
    parameterDescr (name) { return Mdf.descrOfDescrNote(Mdf.paramTextByName(this.theModel, name)) },

    // return output table description by name
    tableDescr (name) {
      const tt = Mdf.tableTextByName(this.theModel, name)
      return tt?.TableDescr || ''
    },

    // return entity description by name
    entityDescr (name) {
      return this.serverConfig.AllowMicrodata ? Mdf.descrOfDescrNote(Mdf.entityTextByName(this.theModel, name)) : ''
    },

    // download user views views
    onDownloadViews () {
      const fName = this.modelName + '.view.json'

      // make parameter, output tables and microdata views json
      const pv = this.dbRows.filter(r => this.paramNames.findIndex(pName => pName === r.name) >= 0)
      const tv = this.dbRows.filter(r => this.tableNames.findIndex(tName => tName === r.name) >= 0)
      let ev = []
      if (this.serverConfig.AllowMicrodata) ev = this.dbRows.filter(r => this.entityNames.findIndex(eName => eName === r.name) >= 0)

      let vs = ''
      if (Mdf.isLength(pv) || Mdf.isLength(tv)) {
        try {
          vs = JSON.stringify({
            model: {
              name: this.modelName,
              parameterViews: pv,
              tableViews: tv,
              microdataViews: ev
            }
          })
        } catch (e) {
          vs = ''
          console.warn('Error at stringify of:', pv, tv, ev)
        }
      }
      if (!vs) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to save views: ') + this.modelName })
        return
      }

      // save as modelName.json
      const ret = exportFile(fName, vs, 'application/json')
      if (!ret) {
        console.warn('Unable to save views:', fName, ret)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to save views: ') + fName })
      }
    },

    // upload model user views
    async onUploadViews () {
      if (!this.isUploadFile) return

      // read and parse model user views json
      let vs
      try {
        const t = await this.uploadFile.text()
        if (t) vs = JSON.parse(t)
      } catch (e) {
        vs = undefined
        console.warn('Error at json parse of text from:', this.uploadFile)
      }
      if (!vs || vs?.model?.name !== this.modelName) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to restore views: ') + this.modelName })
        return
      }

      // write user views into indexed db
      let nViews = 0

      if (Array.isArray(vs.model?.parameterViews) && vs.model?.parameterViews?.length) {
        let name = ''
        try {
          const dbCon = await Idb.connection()
          const rw = await dbCon.openReadWrite(this.modelName)
          for (const v of vs.model.parameterViews) {
            if (!v?.name || !v?.view) continue
            name = v.name
            await rw.put(v.name, v.view)
            nViews++
          }
        } catch (e) {
          console.warn('Unable to save default parameter view:', name, e)
          this.$q.notify({ type: 'negative', message: this.$t('Unable to save default parameter view: ') + name })
          return
        }
      }
      if (Array.isArray(vs.model?.tableViews) && vs.model?.tableViews?.length) {
        let name = ''
        try {
          const dbCon = await Idb.connection()
          const rw = await dbCon.openReadWrite(this.modelName)
          for (const v of vs.model.tableViews) {
            if (!v?.name || !v?.view) continue
            name = v.name
            await rw.put(v.name, v.view)
            nViews++
          }
        } catch (e) {
          console.warn('Unable to save default output table view:', name, e)
          this.$q.notify({ type: 'negative', message: this.$t('Unable to save default output table view: ') + name })
          return
        }
      }
      if (this.serverConfig.AllowMicrodata && Array.isArray(vs.model?.microdataViews) && vs.model?.microdataViews?.length) {
        let name = ''
        try {
          const dbCon = await Idb.connection()
          const rw = await dbCon.openReadWrite(this.modelName)
          for (const v of vs.model.microdataViews) {
            if (!v?.name || !v?.view) continue
            name = v.name
            await rw.put(v.name, v.view)
            nViews++
          }
        } catch (e) {
          console.warn('Unable to save default microdata view:', name, e)
          this.$q.notify({ type: 'negative', message: this.$t('Unable to save default microdata view: ') + name })
          return
        }
      }

      // refresh user views: read new version of views from database
      if (nViews) {
        this.doReadViews()
        this.$q.notify({ type: 'info', message: this.$t('User views count: ') + nViews.toString() })
      } else {
        this.$q.notify({ type: 'info', message: this.$t('No user views found: ') + this.modelName })
      }

      // upload views into user home directory
      this.$emit('user-view-updated', Mdf.modelDigest(this.theModel), this.modelName)
    },

    // delete parameter default view
    onRemoveParamView (name) {
      this.doRemoveView(1, name)
    },
    // delete output table default view
    onRemoveTableView (name) {
      this.doRemoveView(2, name)
    },
    // delete entity microdata default view
    onRemoveMicrodataView (name) {
      this.doRemoveView(3, name)
    },
    async doRemoveView (kind, name) {
      if (!this.modelName) return // model not selected

      try {
        const dbCon = await Idb.connection()
        const rw = await dbCon.openReadWrite(this.modelName)
        await rw.remove(name)
      } catch (e) {
        console.warn('Unable to erase default view of', name, e)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to erase default view of: ') + name })
        return
      }
      switch (kind) {
        case 1:
          this.paramNames = this.paramNames.filter(pn => pn !== name)
          break
        case 2:
          this.tableNames = this.tableNames.filter(tn => tn !== name)
          break
        case 3:
          this.entityNames = this.entityNames.filter(en => en !== name)
          break
        default:
          console.warn('Unable to erase default view of invalid kind:', kind, name)
          this.$q.notify({ type: 'negative', message: this.$t('Unable to erase default view of: ') + name })
          return
      }

      this.$q.notify({
        type: 'info',
        message: this.$t('Default view erased: ') + name
      })

      // upload views into user home directory
      this.$emit('user-view-updated', Mdf.modelDigest(this.theModel), this.modelName)
    },

    // select all user views from indexed db
    async doReadViews () {
      this.dbRows = []
      this.paramNames = []
      this.tableNames = []
      this.entityNames = []

      // select all rows from model indexed db
      this.dbRows = []
      try {
        const dbCon = await Idb.connection()
        const rd = await dbCon.openReadOnly(this.modelName)
        const keyArr = await rd.getAllKeys()
        if (!Mdf.isLength(keyArr)) return // no rows in model db

        for (const key of keyArr) {
          const v = await rd.getByKey(key)
          if (v) {
            this.dbRows.push({ name: key, view: v })
          }
        }
      } catch (e) {
        console.warn('Unable to retrieve model settings', this.modelName, e)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to retrieve model settings: ') + this.modelName })
        return
      }
      if (!Mdf.isLength(this.dbRows)) return // no rows in model db or all rows are empty

      // refresh parameter names, table names and microdata entity list where view is exist
      for (const pt of this.theModel.ParamTxt) {
        if (this.dbRows.findIndex(r => r.name === pt.Param.Name) >= 0) this.paramNames.push(pt.Param.Name)
      }
      for (const tt of this.theModel.TableTxt) {
        if (this.dbRows.findIndex(r => r.name === tt.Table.Name) >= 0) this.tableNames.push(tt.Table.Name)
      }
      for (const et of this.theModel.EntityTxt) {
        if (this.dbRows.findIndex(r => r.name === et.Entity.Name) >= 0) this.entityNames.push(et.Entity.Name)
      }
    }
  },

  mounted () {
    this.doRefresh()
  }
}
</script>

<style lang="scss" scoped>
  .settings-table {
    border-collapse: collapse;
  }
</style>
