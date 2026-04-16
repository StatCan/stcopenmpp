<template>

  <om-table-tree
    :refresh-tickle="refreshTickle"
    :refresh-tree-tickle="refreshTreeTickle"
    :tree-data="tableTreeData"
    :label-kind="treeLabelKind"
    :is-expand="false"
    :tree-store="tableTreeStore"
    :is-any-group="isAnyGroup"
    :is-any-hidden="isAnyHidden"
    :is-show-hidden="isShowHidden"
    :is-group-click="isGroupClick"
    :is-leaf-click="isTableLeafClick"
    :is-add="isAdd"
    :is-add-group="isAddGroup"
    :is-add-disabled="isAddDisabled"
    :is-remove="isRemove"
    :is-remove-group="isRemoveGroup"
    :is-remove-disabled="isRemoveDisabled"
    :is-download="isTableDownload"
    :is-download-disabled="isTableDownloadDisabled"
    :filter-placeholder="$t('Find output table...')"
    :no-results-label="$t('No output tables found')"
    :no-nodes-label="$t('No output tables found or server offline')"
    :is-any-in-list="isAnyFiltered"
    :is-on-off-not-in-list="isOnOffNotInList"
    :is-show-not-in-list="isShowFiltered"
    :in-list-on-label="inListOnLabel"
    :in-list-off-label="inListOffLabel"
    :in-list-icon="inListIcon"
    :is-in-list-clear="isInListClear"
    :in-list-clear-label="inListClearLabel"
    :in-list-clear-icon="inListClearIcon"
    @om-table-tree-show-hidden="onToogleHiddenNodes"
    @om-table-tree-show-not-in-list="onToogleInListFilter"
    @om-table-tree-clear-in-list="onClearInListFilter"
    @om-table-tree-leaf-select="onTableLeafClick"
    @om-table-tree-leaf-add="onAddClick"
    @om-table-tree-group-add="onGroupAddClick"
    @om-table-tree-leaf-remove="onRemoveClick"
    @om-table-tree-group-remove="onGroupRemoveClick"
    @om-table-tree-leaf-download="onDownloadClick"
    @om-table-tree-leaf-note="onShowTableNote"
    @om-table-tree-group-note="onShowGroupNote"
    >
  </om-table-tree>

</template>

<script>
import { mapState, mapActions } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'
import OmTableTree from 'components/OmTableTree.vue'
import * as Tsc from 'components/tree-common.js'
import { openURL } from 'quasar'

export default {
  name: 'TableList',
  components: { OmTableTree },

  props: {
    runDigest: { type: String, required: true },
    refreshTickle: { type: Boolean, default: false },
    refreshTableTreeTickle: { type: Boolean, default: false },
    isNoHidden: { type: Boolean, default: false },
    isGroupClick: { type: Boolean, default: false },
    isTableLeafClick: { type: Boolean, default: false },
    isAdd: { type: Boolean, default: false },
    isAddGroup: { type: Boolean, default: false },
    isAddDisabled: { type: Boolean, default: false },
    isRemove: { type: Boolean, default: false },
    isRemoveGroup: { type: Boolean, default: false },
    isRemoveDisabled: { type: Boolean, default: false },
    isTableDownload: { type: Boolean, default: false },
    isTableDownloadDisabled: { type: Boolean, default: false },
    nameFilter: { type: Array, default: () => [] }, // if not empty then use only tables and groups included in the name list
    isOnOffNotInList: { type: Boolean, default: false },
    inListOnLabel: { type: String, default: '' },
    inListOffLabel: { type: String, default: '' },
    inListIcon: { type: String, default: '' },
    isInListClear: { type: Boolean, default: false },
    inListClearLabel: { type: String, default: '' },
    inListClearIcon: { type: String, default: '' }
  },

  data () {
    return {
      refreshTreeTickle: false,
      isAnyGroup: false,
      isAnyHidden: false,
      isShowHidden: false,
      isAnyFiltered: false,
      isShowFiltered: false,
      runCurrent: Mdf.emptyRunText(), // currently selected run
      tableTreeData: [],
      tableTreeStore: {
        expandPush: void 0, // store action to dispatch tree expanded keys
        expandPull: void 0 // store getter for tree expanded keys
      },
      nextId: 100
    }
  },

  computed: {
    maxTypeSize () {
      const s = Mdf.configEnvValue(this.serverConfig, 'OM_CFG_TYPE_MAX_LEN')
      return ((s || '') !== '') ? parseInt(s) : 0
    },

    ...mapState(useModelStore, [
      'theModel',
      'theModelUpdated',
      'runTextListUpdated'
    ]),
    ...mapState(useServerStateStore, {
      omsUrl: 'omsUrl',
      serverConfig: 'config'
    }),
    ...mapState(useUiStateStore, [
      'treeLabelKind',
      'treeExpandedKeys',
      'idCsvDownload'
    ])
  },

  watch: {
    runDigest () { this.doRefresh() },
    refreshTickle () { this.doRefresh() },
    refreshTableTreeTickle () { this.doRefresh() },
    theModelUpdated () { this.doRefresh() },
    runTextListUpdated () { this.doRefresh() }
  },

  emits: [
    'table-tree-updated',
    'table-clear-in-list',
    'table-select',
    'table-add',
    'table-remove',
    'table-group-add',
    'table-group-remove',
    'table-info-show',
    'table-group-info-show'
  ],

  methods: {
    ...mapActions(useModelStore, ['runTextByDigest']),
    ...mapActions(useUiStateStore, ['dispatchTreeExpandedKeys']),

    // update output tables tree data and refresh tree view
    doRefresh () {
      if (this.runDigest) {
        this.runCurrent = this.runTextByDigest({ ModelDigest: Mdf.modelDigest(this.theModel), RunDigest: this.runDigest })
      }
      const td = this.makeTableTreeData()
      this.tableTreeData = td.tree
      this.refreshTreeTickle = !this.refreshTreeTickle
      this.$emit('table-tree-updated', td.leafCount)
    },

    // show or hide hidden output tables and groups
    onToogleHiddenNodes (isShow) {
      const isNow = this.isShowHidden
      this.isShowHidden = isShow || this.isNoHidden
      if (this.isShowHidden !== isNow) this.doRefresh()
    },
    // show or hide filtered out tables and groups
    onToogleInListFilter (isShow) {
      this.isShowFiltered = isShow
      this.doRefresh()
    },
    // click on clear filter: show all output tables and groups
    onClearInListFilter () {
      this.$emit('table-clear-in-list')
    },
    // click on output table: open current run output table values tab
    onTableLeafClick (name, parts) {
      // check all dimensions: dimension size should not exceed max size limit
      const t = Mdf.tableTextByName(this.theModel, name)
      if (!Mdf.isTable(t?.Table)) {
        this.$q.notify({ type: 'negative', message: this.$t('Output table not found: ') + (name || '') })
        return
      }
      for (const d of t.TableDimsTxt) {
        const ne = Mdf.typeEnumSizeById(this.theModel, d.Dim.TypeId)

        if (this.maxTypeSize > 0 && ne > this.maxTypeSize) {
          this.$q.notify({ type: 'negative', message: this.$t('Output table dimension size exceed the limit: ') + this.maxTypeSize.toString() + ' ' + (name || '') + '.' + (d.Dim.Name || '') })
          return
        }
      }
      this.$emit('table-select', name, parts) // pass message to the parent
    },
    // click on add output table: add output table from current run
    onAddClick (name, parts) {
      this.$emit('table-add', name, parts)
    },
    // click on add group: add output tables group from current run
    onGroupAddClick (name, parts) {
      this.$emit('table-group-add', name, parts)
    },
    // click on remove output table: remove output table from current run
    onRemoveClick (name, parts) {
      this.$emit('table-remove', name, parts)
    },
    // click on remove group: remove output tables group from current run
    onGroupRemoveClick (name, parts) {
      this.$emit('table-group-remove', name, parts)
    },
    // click on show output table notes dialog button
    onShowTableNote (name, parts) {
      this.$emit('table-info-show', name, parts)
    },
    // click on show group notes dialog button
    onShowGroupNote (name, parts) {
      this.$emit('table-group-info-show', name, parts)
    },
    // download output table as csv
    onDownloadClick  (name, parts) {
      if (!name || !Mdf.isRunTextHasTable(this.runCurrent, name)) {
        this.$q.notify({ type: 'negative', message: this.$t('Output table not found: ') + (name || '') })
        return
      }
      const u = this.omsUrl +
        '/api/model/' + encodeURIComponent(Mdf.modelDigest(this.theModel)) +
        '/run/' + encodeURIComponent(this.runDigest) +
        '/table/' + encodeURIComponent(name) +
        '/expr' +
        ((this.$q.platform.is.win) ? (this.idCsvDownload ? '/csv-id-bom' : '/csv-bom') : (this.idCsvDownload ? '/csv-id' : '/csv'))

      openURL(u)
    },

    // return tree of output tables
    makeTableTreeData () {
      this.isAnyGroup = false
      this.isAnyHidden = false
      this.isAnyFiltered = false

      if (!Mdf.tableCount(this.theModel)) {
        return { tree: [], leafCount: 0 } // empty list of output tables
      }
      if (!Mdf.isTableTextList(this.theModel)) {
        this.$q.notify({ type: 'negative', message: this.$t('Model output tables list is empty or invalid') })
        return { tree: [], leafCount: 0 } // invalid list of output tables
      }
      const mDgst = Mdf.modelDigest(this.theModel)

      // if filter names not empty then display only groups and tables from that name list
      // build groups name list and tables names list to include into the tree
      const gLst = this.theModel.GroupTxt

      let isTflt = false
      let isGflt = false
      const ftLst = {}
      const fgLst = {}

      if (this.nameFilter && this.nameFilter.length > 0) {
        for (let k = 0; k < gLst.length; k++) {
          if (gLst[k].Group.IsParam) continue // skip parameters group

          if (this.nameFilter.findIndex((name) => { return name === gLst[k].Group.Name }) >= 0) {
            isGflt = true
            fgLst[gLst[k].Group.Name] = true // found group name in the filter include names list
          }
        }

        for (const t of this.theModel.TableTxt) {
          if (this.runDigest && !Mdf.isRunTextHasTable(this.runCurrent, t.Table.Name)) continue // skip suppressed table

          if (this.nameFilter.findIndex((name) => { return name === t.Table.Name }) >= 0) {
            isTflt = true
            ftLst[t.Table.Name] = true // found table name in the filter include names list
          }
        }
      }

      // map tables which are included in model run (not suppressed)
      // skip hidden tables if show hidden tables disabled
      const tUse = {}
      for (const t of this.theModel.TableTxt) {
        if (this.runDigest && !Mdf.isRunTextHasTable(this.runCurrent, t.Table.Name)) continue // skip suppressed table

        this.isAnyFiltered = this.isAnyFiltered || (isTflt && !ftLst[t.Table.Name])
        this.isAnyHidden = !this.isNoHidden && (this.isAnyHidden || t.Table.IsHidden)
        if (!this.isNoHidden && !this.isShowHidden && t.Table.IsHidden) continue // skip hidden table if reqired

        tUse[t.Table.TableId] = { isLeaf: false }
      }

      // make groups map: map group id to group node
      const gUse = {}

      for (let k = 0; k < gLst.length; k++) {
        if (gLst[k].Group.IsParam) continue // skip parameters group

        const gId = gLst[k].Group.GroupId
        const isNote = Mdf.noteOfDescrNote(gLst[k]) !== ''
        this.isAnyHidden = !this.isNoHidden && (this.isAnyHidden || gLst[k].Group.IsHidden)
        this.isAnyFiltered = this.isAnyFiltered || (isGflt && !fgLst[gLst[k].Group.Name])

        gUse[gId] = {
          group: gLst[k],
          item: {
            key: 'tgr-' + gId + '-' + this.nextId++,
            label: gLst[k].Group.Name,
            descr: Mdf.descrOfDescrNote(gLst[k]),
            children: [],
            parts: '',
            isGroup: true,
            isAbout: isNote,
            isAboutEmpty: !isNote
          }
        }
      }

      // add top level groups as starting point into groups tree
      let gTree = []
      const gProc = []

      for (let k = 0; k < gLst.length; k++) {
        if (gLst[k].Group.IsParam) continue // skip parameters group
        if (!this.isNoHidden && !this.isShowHidden && gLst[k].Group.IsHidden) continue // skip hidden group
        if (isGflt && !this.isShowFiltered && !fgLst[gLst[k].Group.Name]) continue // skip filtered out group

        const iGId = gLst[k].Group.GroupId

        if (!gUse[iGId]) continue // skip this group: it is empty group

        const isNotTop = gLst.findIndex((gt) => {
          if (gt.Group.IsParam) return false
          if (gt.Group.GroupId === iGId) return false
          if (gt.Group.GroupPc.length <= 0) return false
          return gt.Group.GroupPc.findIndex((pc) => pc.ChildGroupId === iGId) >= 0
        }) >= 0
        if (isNotTop) continue // not a top level group

        const g = Mdf.deepClone(gUse[iGId].item)
        gTree.push(g)
        gProc.push({
          gId: iGId,
          path: [iGId],
          item: g
        })
      }
      this.isAnyGroup = gTree.length > 0

      // build groups tree
      while (gProc.length > 0) {
        const gpNow = gProc.pop()
        const gTxt = gUse[gpNow.gId].group

        // make all children of current item
        for (const pc of gTxt.Group.GroupPc) {
          // if this is a child group
          if (pc.ChildGroupId >= 0) {
            const gChildUse = gUse[pc.ChildGroupId]
            if (gChildUse) {
              if (!this.isNoHidden && !this.isShowHidden && gChildUse.group.Group.IsHidden) continue // skip hidden group
              if (isGflt && !this.isShowFiltered && !fgLst[gChildUse.group.Group.Name]) continue // skip filtered out group

              // check for circular reference
              if (gpNow.path.indexOf(pc.ChildGroupId) >= 0) {
                console.warn('Error: circular refernece to group:', pc.ChildGroupId, 'path:', gpNow.path)
                continue // skip this group
              }

              const g = {
                gId: pc.ChildGroupId,
                path: Mdf.deepClone(gpNow.path),
                item: Mdf.deepClone(gChildUse.item)
              }
              g.item.key = 'tgr-' + pc.ChildGroupId + '-' + this.nextId++
              g.path.push(g.gId)
              gProc.push(g)
              gpNow.item.children.push(g.item)
            }
          }

          // if this is a child leaf output table
          if (pc.ChildLeafId >= 0) {
            const t = Mdf.tableTextById(this.theModel, pc.ChildLeafId)
            if (!Mdf.isTable(t.Table)) continue
            if (!tUse[t.Table.TableId]) continue // skip: table suppressed or hidden
            if (isTflt && !this.isShowFiltered && !ftLst[t.Table.Name]) continue // skip filtered out table

            gpNow.item.children.push({
              key: 'ttl-' + pc.ChildLeafId + '-' + this.nextId++,
              label: t.Table.Name,
              descr: (t.TableDescr || ''),
              children: [],
              link: this.isTableLeafClick ? '/?model=' + encodeURIComponent(mDgst) + '&run=' + encodeURIComponent(this.runDigest) + '&table=' + encodeURIComponent(t.Table.Name) : '',
              parts: '',
              isGroup: false,
              isAbout: true,
              isAboutEmpty: false
            })
            tUse[t.Table.TableId].isLeaf = true
          }
        }
      }

      // walk the tree and remove empty branches
      gTree = Tsc.removeEmptyGroups(gTree)

      // add output tables which are not included in any group (not a leaf nodes)
      const leafUse = {}

      for (let k = 0; k < gLst.length; k++) {
        if (gLst[k].Group.IsParam) continue // skip parameters group

        for (const pc of gLst[k].Group.GroupPc) {
          if (pc.ChildLeafId >= 0) leafUse[pc.ChildLeafId] = true // store leaf output table id
        }
      }

      let nLeaf = 0
      for (const t of this.theModel.TableTxt) {
        if (!tUse[t.Table.TableId]) continue // skip: table suppressed or hidden
        if (isTflt && !this.isShowFiltered && !ftLst[t.Table.Name]) continue // skip filtered out table

        if (!leafUse[t.Table.TableId]) { // if output table is not a leaf of any group
          gTree.push({
            key: 'ttl-' + t.Table.TableId + '-' + this.nextId++,
            label: t.Table.Name,
            descr: (t.TableDescr || ''),
            link: this.isTableLeafClick ? '/?model=' + encodeURIComponent(mDgst) + '&run=' + encodeURIComponent(this.runDigest) + '&table=' + encodeURIComponent(t.Table.Name) : '',
            children: [],
            parts: '',
            isGroup: false,
            isAbout: true,
            isAboutEmpty: false
          })
          tUse[t.Table.TableId].isLeaf = true
        }
        if (tUse[t.Table.TableId].isLeaf) nLeaf++
      }

      return { tree: gTree, leafCount: nLeaf }
    }
  },

  mounted () {
    this.tableTreeStore = {
      // push tree expanded keys into store for that model name and tree kind
      expandPush: (expKeys) => {
        this.dispatchTreeExpandedKeys(Mdf.modelName(this.theModel), 'table-list', expKeys)
      },
      // push tree expanded keys into store for that model name and tree kind
      expandPull: () => {
        return this.treeExpandedKeys(Mdf.modelName(this.theModel), 'table-list')
      }
    }
    this.doRefresh()
  }
}
</script>
