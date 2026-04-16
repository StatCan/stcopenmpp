<template>
<q-page class="text-body1">

  <div class="row items-center full-width q-pt-sm q-px-sm">

    <q-btn
      v-if="isAnyModelGroup"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs om-tree-control-button"
      :icon="!isTreeExpanded ? 'keyboard_arrow_down' :'keyboard_arrow_up'"
      :title="!isTreeExpanded ? $t('Expand all') : $t('Collapse all')"
      @click="doToogleExpandTree"
      />
    <q-btn
      v-if="!!isSortModelTree && !!treeData && treeData?.length > 0"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs om-tree-control-button"
      :icon="isDescModelTree ? 'mdi-sort-alphabetical-descending' :'mdi-sort-alphabetical-ascending'"
      :title="isDescModelTree ? $t('Sort Descending') : $t('Sort Ascending')"
      @click="doToogleTreeSort"
      />
    <span class="col-grow">
      <q-input
        ref="filterInput"
        debounce="500"
        v-model="treeFilter"
        outlined
        dense
        :placeholder="$t('Find model...')"
        >
        <template v-slot:append>
          <q-icon v-if="treeFilter !== ''" name="cancel" class="cursor-pointer" @click="resetFilter" />
          <q-icon v-else name="search" />
        </template>
      </q-input>
    </span>

  </div>

  <div class="q-pa-sm">
    <q-tree
      ref="theTree"
      :nodes="treeData"
      node-key="key"
      default-expand-all
      v-model:expanded="expandedKeys"
      :filter="treeFilter"
      :filter-method="doTreeFilter"
      no-transition
      :no-results-label="$t('No models found')"
      :no-nodes-label="$t('Server offline or no models published')"
      >
      <template v-slot:default-header="prop">
        <div
          v-if="!prop.node.digest"
          class="row no-wrap items-center"
          :class="{'om-tree-found-node': treeWalk.keysFound[prop.node.key]}"
          >
          <div class="col">
            <template v-if="treeLabelKind == 'name-only'"><span>{{ prop.node.label }}</span></template>
            <template v-if="treeLabelKind == 'descr-only'"><span>{{ prop.node.descr || prop.node.label }}</span></template>
            <template v-if="treeLabelKind !== 'name-only' && treeLabelKind !== 'descr-only'">
              <span>{{ prop.node.label }}<br />
              <span class="om-text-descr">{{ prop.node.descr }}</span></span>
            </template>
          </div>
        </div>
        <div v-else
          class="row no-wrap items-center full-width om-tree-leaf"
          :class="{'om-tree-found-node': treeWalk.keysFound[prop.node.key]}"
          >
          <q-btn
            @click.stop="doShowModelNote(prop.node.digest)"
            :flat="prop.node.digest !== theModelDigest"
            :outline="prop.node.digest === theModelDigest"
            round
            dense
            padding="xs"
            color="primary"
            class="col-auto"
            :icon="prop.node.digest === theModelDigest ? 'mdi-information' : 'mdi-information-outline'"
            :title="$t('About') + ' ' + prop.node.label"
            />
          <q-btn
            @click.stop="doModelDocLink(prop.node.digest)"
            :disable="!isModelDocLink(prop.node.digest)"
            flat
            round
            dense
            padding="xs"
            :color="isModelDocLink(prop.node.digest) ? 'primary' : 'secondary'"
            class="col-auto"
            icon="mdi-book-open-outline"
            :title="$t('Model Documentation') + ' ' + prop.node.label"
            />
          <q-btn
            v-if="serverConfig.AllowDownload"
            @click.stop="onModelDownload(prop.node.digest, prop.node.label)"
            flat
            round
            dense
            padding="xs"
            color="primary"
            class="col-auto"
            icon="mdi-download-circle-outline"
            :title="$t('Download') + ' ' + prop.node.label"
            />
          <router-link
            :to="'/model/' + prop.node.digest"
            :title="prop.node.label"
            class="col om-tree-leaf-link q-ml-xs"
            :class="{ 'text-bold text-primary' : prop.node.digest === theModelDigest }"
            >
            <template v-if="treeLabelKind == 'name-only'"><span>{{ prop.node.label }}</span></template>
            <template v-if="treeLabelKind == 'descr-only'"><span>{{ prop.node.descr || prop.node.label }}</span></template>
            <template v-if="treeLabelKind !== 'name-only' && treeLabelKind !== 'descr-only'">
              <span>{{ prop.node.label }}<br />
              <span class="om-text-descr">{{ prop.node.descr }}</span></span>
            </template>
          </router-link>
        </div>
      </template>
    </q-tree>
  </div>

  <model-info-dialog :show-tickle="modelInfoTickle" :digest="modelInfoDigest"></model-info-dialog>

  <confirm-dialog
    @confirm-yes="onYesDownload"
    :show-tickle="showDownloadConfirmTickle"
    :item-id="downloadDigest"
    :item-name="downloadLabel"
    :dialog-title="$t('Download model data' + '?')"
    :body-text="$t('Download')"
    :icon-name="'mdi-download-circle'"
    >
  </confirm-dialog>

  <q-inner-loading :showing="loadWait">
    <q-spinner-gears size="lg" color="primary" />
  </q-inner-loading>

</q-page>
</template>

<script>
import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'
import ModelInfoDialog from 'components/ModelInfoDialog.vue'
import ConfirmDialog from 'components/ConfirmDialog.vue'
import { openURL } from 'quasar'

export default {
  name: 'ModelList',
  components: { ModelInfoDialog, ConfirmDialog },

  props: {
    refreshTickle: { type: Boolean, default: false }
  },

  data () {
    return {
      treeFilter: '',
      isTreeExpanded: false,
      isAnyModelGroup: false,
      nextId: 100,
      treeData: [],
      expandedKeys: [],
      treeWalk: {
        isAnyFound: false,
        keysFound: {} // if node match filter then map keysFound[node.key] = true
      },
      modelInfoTickle: false,
      modelInfoDigest: '',
      showDownloadConfirmTickle: false,
      downloadDigest: '',
      downloadLabel: '',
      loadDone: false,
      loadWait: false
    }
  },

  computed: {
    theModelDigest () { return Mdf.modelDigest(this.theModel) },

    ...mapState(useModelStore, [
      'theModel',
      'modelList',
      'modelCount',
      'modelLanguage'
    ]),
    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl',
      serverConfig: 'config'
    }),
    ...mapState(useUiStateStore, [
      'isSortModelTree',
      'isDescModelTree',
      'treeExpandedKeys',
      'noAccDownload',
      'noMicrodataDownload',
      'idCsvDownload',
      'treeLabelKind',
      'uiLang'
    ])
  },

  watch: {
    refreshTickle () { this.doRefresh() },
    treeFilter () { this.updateTreeWalk() },
    isDescModelTree () { this.sortTree(this.treeData) }
  },

  emits: ['download-select'],

  methods: {
    ...mapActions(useModelStore, [
      'dispatchModelList'
    ]),
    ...mapActions(useUiStateStore, [
      'dispatchSortModelTree',
      'dispatchTreeExpandedKeys'
    ]),

    // expand or collapse all tree nodes
    doToogleExpandTree () {
      if (this.isTreeExpanded) {
        this.$refs.theTree.collapseAll()
      } else {
        this.$refs.theTree.expandAll()
      }
      this.isTreeExpanded = !this.isTreeExpanded

      // remove duplicates if expand is result of search
      this.expandedKeys = this.expandedKeys.filter((key, idx, arr) => arr.indexOf(key) === idx)
    },
    // toggle model tree sort order
    doToogleTreeSort () {
      this.dispatchSortModelTree({ isSort: true, isDesc: !this.isDescModelTree })
    },
    // filter by model name (label) or model description
    doTreeFilter (node, filter) {
      return this.treeWalk.isAnyFound && !!this.treeWalk.keysFound[node.key]
    },
    // update filtered nodes key list, include all children if group match the filter
    updateTreeWalk () {
      this.treeWalk.isAnyFound = false
      this.treeWalk.keysFound = {}

      if (!this.treeFilter) return // filter is empty

      const flt = this.treeFilter.toLowerCase()

      // walk the tree and check every node by filter match
      const td = []
      for (const g of this.treeData) {
        td.push(g)
      }
      while (td.length > 0) {
        const t = td.pop()

        let isFound = (t.label && t.label.toLowerCase().indexOf(flt) > -1) ||
          ((t.descr || '') !== '' && t.descr.toLowerCase().indexOf(flt) > -1)

        if (!isFound) isFound = this.treeWalk.keysFound[t.key] === true
        if (isFound && !this.treeWalk.keysFound[t.key]) this.treeWalk.keysFound[t.key] = true

        // if current node match filter then add all children to matched keys list
        for (const c of t.children) {
          td.push(c)
          if (isFound) this.treeWalk.keysFound[c.key] = true
        }
        if (!this.treeWalk.isAnyFound) this.treeWalk.isAnyFound = isFound
      }

      // if any node match the filter then expand the tree
      if (this.treeWalk.isAnyFound && !this.isTreeExpanded) {
        this.$nextTick(() => { this.doToogleExpandTree() })
      }
    },
    // clear filter value
    resetFilter () {
      this.treeFilter = ''
      this.treeWalk.isAnyFound = false
      this.treeWalk.keysFound = {}
      this.$refs.filterInput.focus()
    },

    // show model notes dialog
    doShowModelNote (digest) {
      this.modelInfoDigest = digest
      this.modelInfoTickle = !this.modelInfoTickle
    },
    isModelDocLink (digest) {
      if (!this.serverConfig.IsModelDoc) return ''
      const u = Mdf.modelDocLinkByDigest(digest, this.modelList, this.uiLang, this.modelLanguage)
      return u && u !== ''
    },
    // show model notes dialog
    doModelDocLink (digest) {
      if (!this.serverConfig.IsModelDoc) return
      const u = Mdf.modelDocLinkByDigest(digest, this.modelList, this.uiLang, this.modelLanguage)
      if (u) openURL('doc/' + u)
    },

    // show yes/no dialog to confirm model download
    onModelDownload (digest, label) {
      this.downloadDigest = digest
      this.downloadLabel = label
      this.showDownloadConfirmTickle = !this.showDownloadConfirmTickle
    },
    // user answer yes to confirm model download: start model download and show download list page
    onYesDownload (label, digest, kind) {
      if (!digest) {
        console.warn('Unable to download model:', digest, label)
        this.$q.notify({ type: 'negative', message: this.$t('Unable to download model') + ' ' + (label || '') })
        return
      }
      this.startModelDownload(digest) // start model download and show download page on success
    },

    // return tree of models
    makeTreeData (mLst, ekArr) {
      this.isAnyModelGroup = false
      this.treeFilter = ''
      this.expandedKeys = []
      const expKeys = (!!ekArr && Array.isArray(ekArr)) ? ekArr : []

      if (!Mdf.isLength(mLst)) return [] // empty model list
      if (!Mdf.isModelList(mLst)) {
        this.$q.notify({ type: 'negative', message: this.$t('Model list is empty or invalid') })
        return [] // invalid model list
      }

      // make folders tree
      const fm = {}
      const td = []

      for (const md of mLst) {
        if (!md?.Dir || md.Dir === '.' || md.Dir === '/' || md.Dir === './') continue // empty or top-level directory

        // add each folder/sub-folder into the tree
        const fa = md.Dir.split('/')
        let p = '', pp = ''

        for (const fn of fa) {
          if (!fn || fn === '.') continue // empty or top-level folder

          p = pp ? pp + '/' + fn : fn

          if (fm?.[p]) {
            pp = p
            continue // path already exist
          }

          const f = {
            key: 'mf-' + p + '-' + this.nextId++,
            digest: '',
            label: fn,
            descr: '',
            dir: pp,
            children: [],
            disabled: false
          }
          fm[p] = f
          if (fm?.[pp]) fm[pp].children.push(f)
          if (!pp) {
            td.push(f) // add new top-level folder
          }
          if (expKeys.indexOf(f.key) >= 0 && this.expandedKeys.indexOf(f.key) < 0) {
            this.expandedKeys.push(f.key)
          }
          pp = p
        }

        if (fm?.[p]) { // add model to the current folder, if it is not empty or top-level directory
          fm[p].children.push({
            key: 'md-' + md.Model.Digest + '-' + this.nextId++,
            digest: md.Model.Digest,
            label: md.Model.Name,
            descr: Mdf.descrOfDescrNote(md),
            dir: p,
            children: [],
            disabled: false
          })
        }
      }

      this.isAnyModelGroup = td.length > 0

      // add models which are not included in any group
      for (const md of mLst) {
        if (!md?.Dir || md.Dir === '.' || md.Dir === '/' || md.Dir === './') { // empty or top-level directory
          td.push({
            key: 'md-' + md.Model.Digest + '-' + this.nextId++,
            digest: md.Model.Digest,
            label: md.Model.Name,
            descr: Mdf.descrOfDescrNote(md),
            dir: '',
            children: [],
            disabled: false
          })
        }
      }

      this.sortTree(td)
      return td
    },

    // sort each tree level by labels custom ascending or descending order, keep folders before models
    sortTree (tData) {
      if (!this.isSortModelTree) return

      // sort tree in alphabetical or in reverse order
      // folders always before models
      // use case-neutral sort,
      // if case-neutral strings are equal then do addtional locale case sensintive comparison
      // example:
      //   Model moDel model model2 one other path RISK Risk riSk RiskA RiskP RiskP two
      let ln = this.uiLang
      if (!ln) ln = this.$q.lang.getLocale()

      const cmpNode = (left, right) => {
        // folder name always before model name
        if (!left.digest && !!right.digest) return -1
        if (!!left.digest && !right.digest) return 1

        // sort by name in custom ascending or descending order
        if (left.label === right.label) return 0

        const lLc = left.label.toLocaleLowerCase(ln)
        const rLc = right.label.toLocaleLowerCase(ln)

        if (lLc < rLc) return !this.isDescModelTree ? -1 : 1
        if (lLc > rLc) return !this.isDescModelTree ? 1 : -1

        const i = left.label.localeCompare(right.label, ln)

        if (i < 0) return !this.isDescModelTree ? 1 : -1
        if (i > 0) return !this.isDescModelTree ? -1 : 1
        return 0
      }

      tData.sort(cmpNode) // sort tree top level

      // walk the tree and sort node children
      const tnc = []
      for (const g of tData) {
        tnc.push(g)
      }
      while (tnc.length > 0) {
        const t = tnc.pop()
        if (Array.isArray(t?.children)) {
          t.children.sort(cmpNode) // sort that node children
          // add all children nodes to the walk list
          for (const c of t.children) {
            tnc.push(c)
          }
        }
      }
    },

    // refersh model list
    async doRefresh () {
      this.loadDone = false
      this.loadWait = true

      const u = this.omsUrl + '/api/model-list/text' + (this.uiLang !== '' ? '/lang/' + encodeURIComponent(this.uiLang) : '')
      try {
        const response = await this.$axios.get(u)
        this.dispatchModelList(response.data) // update model list in store
        this.treeData = this.makeTreeData(response.data, []) // update model list tree
        this.loadDone = true
      } catch (e) {
        let em = ''
        try {
          if (e.response) em = e.response.data || ''
        } finally {}
        console.warn('Server offline or no models published', em)
        this.$q.notify({ type: 'negative', message: this.$t('Server offline or no models published') })
      }
      this.loadWait = false

      // expand after refresh
      this.$nextTick(() => {
        this.isTreeExpanded = false // toogle to default-expand-all
        this.doToogleExpandTree()
      })

    },

    // start model download
    async startModelDownload (dgst) {
      let isOk = false
      let msg = ''

      const opts = {
        NoAccumulatorsCsv: this.noAccDownload,
        NoMicrodata: this.noMicrodataDownload,
        Utf8BomIntoCsv: this.$q.platform.is.win,
        IdCsv: !!this.idCsvDownload
      }
      const u = this.omsUrl +
      '/api/download/model/' + encodeURIComponent((dgst || '')) +
      (this.uiLang !== '' ? '/lang/' + encodeURIComponent(this.uiLang) : '')
      try {
        // send download request to the server, response expected to be empty on success
        await this.$axios.post(u, opts)
        isOk = true
      } catch (e) {
        try {
          if (e.response) msg = e.response.data || ''
        } finally {}
        console.warn('Unable to download model', msg)
      }
      if (!isOk) {
        this.$q.notify({ type: 'negative', message: this.$t('Unable to download model') + (msg ? ('. ' + msg) : '') })
        return
      }

      this.$emit('download-select', dgst) // download started: show download list page
      this.$q.notify({ type: 'info', message: this.$t('Model download started') })
    }
  },

  // route leave guard: on leaving save model tree expanded state
  beforeRouteLeave (to, from) {
    this.dispatchTreeExpandedKeys('all-models', 'model-list', this.expandedKeys)
    return true
  },

  mounted () {
    // if model list already loaded then exit
    if (this.modelCount > 0) {
      this.loadDone = true
      const expKeys = this.treeExpandedKeys('all-models', 'model-list')
      this.treeData = this.makeTreeData(this.modelList, expKeys) // update model list tree
      return
    }
    this.doRefresh() // load new model list
  }
}
</script>

<style lang="scss" scope="local">
</style>
