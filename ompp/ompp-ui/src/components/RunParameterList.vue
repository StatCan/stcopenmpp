<template>

  <om-table-tree
    :refresh-tickle="refreshTickle"
    :refresh-tree-tickle="refreshTreeTickle"
    :tree-data="paramTreeData"
    :label-kind="treeLabelKind"
    :is-expand="false"
    :tree-store="paramTreeStore"
    :is-any-group="isAnyGroup"
    :is-any-hidden="isAnyHidden"
    :is-show-hidden="isShowHidden"
    :is-group-click="isGroupClick"
    :is-leaf-click="isParamLeafClick"
    :is-add="isAdd"
    :is-add-group="isAddGroup"
    :is-add-disabled="isAddDisabled"
    :is-remove="isRemove"
    :is-remove-group="isRemoveGroup"
    :is-remove-disabled="isRemoveDisabled"
    :is-download="isParamDownload"
    :is-download-disabled="isParamDownloadDisabled"
    :filter-placeholder="$t('Find parameter...')"
    :no-results-label="$t('No model parameters found')"
    :no-nodes-label="$t('No model parameters found or server offline')"
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
    @om-table-tree-leaf-select="onParamLeafClick"
    @om-table-tree-leaf-add="onAddClick"
    @om-table-tree-group-add="onGroupAddClick"
    @om-table-tree-leaf-remove="onRemoveClick"
    @om-table-tree-group-remove="onGroupRemoveClick"
    @om-table-tree-leaf-download="onDownloadClick"
    @om-table-tree-leaf-note="onShowParamNote"
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
  name: 'RunParameterList',
  components: { OmTableTree },

  props: {
    runDigest: { type: String, required: true },
    refreshTickle: { type: Boolean, default: false },
    refreshParamTreeTickle: { type: Boolean, default: false },
    isNoHidden: { type: Boolean, default: false },
    isGroupClick: { type: Boolean, default: false },
    isParamLeafClick: { type: Boolean, default: false },
    isAdd: { type: Boolean, default: false },
    isAddGroup: { type: Boolean, default: false },
    isAddDisabled: { type: Boolean, default: false },
    isRemove: { type: Boolean, default: false },
    isRemoveGroup: { type: Boolean, default: false },
    isRemoveDisabled: { type: Boolean, default: false },
    isParamDownload: { type: Boolean, default: false },
    isParamDownloadDisabled: { type: Boolean, default: false },
    nameFilter: { type: Array, default: () => [] }, // if not empty then use only parameters and groups included in the name list
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
      paramTreeData: [],
      paramTreeStore: {
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
      'theModelUpdated'
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
    refreshParamTreeTickle () { this.doRefresh() },
    theModelUpdated () { this.doRefresh() }
  },

  emits: [
    'run-parameter-tree-updated',
    'run-parameter-clear-in-list',
    'run-parameter-select',
    'run-parameter-add',
    'run-parameter-remove',
    'run-parameter-group-add',
    'run-parameter-group-remove',
    'run-parameter-info-show',
    'run-parameter-group-info-show'
  ],

  methods: {
    ...mapActions(useUiStateStore, ['dispatchTreeExpandedKeys']),

    // update parameters tree data and refresh tree view
    doRefresh () {
      const td = this.makeParamTreeData()
      this.paramTreeData = td.tree
      this.refreshTreeTickle = !this.refreshTreeTickle
      this.$emit('run-parameter-tree-updated', td.leafCount)
    },

    // show or hide hidden parameters and groups
    onToogleHiddenNodes (isShow) {
      const isNow = this.isShowHidden
      this.isShowHidden = isShow || this.isNoHidden
      if (this.isShowHidden !== isNow) this.doRefresh()
    },
    // show or hide filtered out parameters and groups
    onToogleInListFilter (isShow) {
      this.isShowFiltered = isShow
      this.doRefresh()
    },
    // click on clear filter: show all parameters and groups
    onClearInListFilter () {
      this.$emit('run-parameter-clear-in-list')
    },
    // click on parameter: open current run parameter values tab
    onParamLeafClick (name, parts) {
      const p = Mdf.paramTextByName(this.theModel, name)
      if (!Mdf.isParam(p?.Param)) {
        this.$q.notify({ type: 'negative', message: this.$t('Parameter not found: ') + (name || '') })
        return
      }
      // check parameter type: if parameter is enum based then type size should not exceed max size limit
      const nt = Mdf.typeEnumSizeById(this.theModel, p.Param.TypeId)
      if (this.maxTypeSize > 0 && nt > this.maxTypeSize) {
        this.$q.notify({ type: 'negative', message: this.$t('Parameter type size exceed the limit: ') + this.maxTypeSize.toString() + ' ' + (name || '') })
        return
      }

      // check all dimensions: dimension size should not exceed max size limit
      for (const d of p.ParamDimsTxt) {
        const ne = Mdf.typeEnumSizeById(this.theModel, d.Dim.TypeId)
        if (this.maxTypeSize > 0 && ne > this.maxTypeSize) {
          this.$q.notify({ type: 'negative', message: this.$t('Parameter dimension size exceed the limit: ') + this.maxTypeSize.toString() + ' ' + (name || '') + '.' + (d.Dim.Name || '') })
          return
        }
      }

      this.$emit('run-parameter-select', name, parts) // pass message to the parent
    },
    // click on add parameter: add parameter from current run
    onAddClick (name, parts) {
      this.$emit('run-parameter-add', name, parts)
    },
    // click on add group: add group from current run
    onGroupAddClick (name, parts) {
      this.$emit('run-parameter-group-add', name, parts)
    },
    // click on remove parameter: remove parameter from current run
    onRemoveClick (name, parts) {
      this.$emit('run-parameter-remove', name, parts)
    },
    // click on remove group: remove group from current run
    onGroupRemoveClick (name, parts) {
      this.$emit('run-parameter-group-remove', name, parts)
    },
    // click on show parameter notes dialog button
    onShowParamNote (name, parts) {
      this.$emit('run-parameter-info-show', name, parts)
    },
    // click on show group notes dialog button
    onShowGroupNote (name, parts) {
      this.$emit('run-parameter-group-info-show', name, parts)
    },
    // download parameter as csv
    onDownloadClick  (name, parts) {
      const p = Mdf.paramTextByName(this.theModel, name)
      if (!Mdf.isParam(p?.Param)) {
        this.$q.notify({ type: 'negative', message: this.$t('Parameter not found: ') + (name || '') })
        return
      }
      const u = this.omsUrl +
          '/api/model/' + Mdf.modelDigest(this.theModel) +
          '/run/' + encodeURIComponent(this.runDigest) +
          '/parameter/' + encodeURIComponent(name) +
          ((this.$q.platform.is.win) ? (this.idCsvDownload ? '/csv-id-bom' : '/csv-bom') : (this.idCsvDownload ? '/csv-id' : '/csv'))

      openURL(u)
    },

    // return tree of model parameters
    makeParamTreeData () {
      this.isAnyGroup = false
      this.isAnyHidden = false
      this.isAnyFiltered = false

      if (!Mdf.paramCount(this.theModel)) {
        return { tree: [], leafCount: 0 } // empty list of parameters
      }
      if (!Mdf.isParamTextList(this.theModel)) {
        this.$q.notify({ type: 'negative', message: this.$t('Model parameters list is empty or invalid') })
        return { tree: [], leafCount: 0 } // invalid list of parameters
      }
      const mDgst = Mdf.modelDigest(this.theModel)

      // if filter names not empty then display only groups and parameters from that name list
      // build groups name list and parameter names list to include into the tree
      const gLst = this.theModel.GroupTxt

      let isPflt = false
      let isGflt = false
      const fpLst = {}
      const fgLst = {}

      if (this.nameFilter && this.nameFilter.length > 0) {
        for (let k = 0; k < gLst.length; k++) {
          if (!gLst[k].Group.IsParam) continue // skip output tables group

          if (this.nameFilter.findIndex((name) => { return name === gLst[k].Group.Name }) >= 0) {
            isGflt = true
            fgLst[gLst[k].Group.Name] = true // found group name in the filter include names list
          }
        }

        for (const p of this.theModel.ParamTxt) {
          if (this.nameFilter.findIndex((name) => { return name === p.Param.Name }) >= 0) {
            isPflt = true
            fpLst[p.Param.Name] = true // found parameter name in the filter include names list
          }
        }
      }

      // make groups map: map group id to group node
      const gUse = {}

      for (let k = 0; k < gLst.length; k++) {
        if (!gLst[k].Group.IsParam) continue // skip output tables group

        const gId = gLst[k].Group.GroupId
        const isNote = Mdf.noteOfDescrNote(gLst[k]) !== ''
        this.isAnyHidden = !this.isNoHidden && (this.isAnyHidden || gLst[k].Group.IsHidden)
        this.isAnyFiltered = this.isAnyFiltered || (isGflt && !fgLst[gLst[k].Group.Name])

        gUse[gId] = {
          group: gLst[k],
          item: {
            key: 'pgr-' + gId + '-' + this.nextId++,
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
        if (!gLst[k].Group.IsParam) continue // skip output tables group
        if (!this.isNoHidden && !this.isShowHidden && gLst[k].Group.IsHidden) continue // skip hidden group
        if (isGflt && !this.isShowFiltered && !fgLst[gLst[k].Group.Name]) continue // skip filtered out group

        const iGid = gLst[k].Group.GroupId

        const isNotTop = gLst.findIndex((gt) => {
          if (!gt.Group.IsParam) return false
          if (gt.Group.GroupId === iGid) return false
          if (gt.Group.GroupPc.length <= 0) return false
          return gt.Group.GroupPc.findIndex((pc) => pc.ChildGroupId === iGid) >= 0
        }) >= 0
        if (isNotTop) continue // not a top level group

        const g = Mdf.deepClone(gUse[iGid].item)
        gTree.push(g)
        gProc.push({
          gId: iGid,
          path: [iGid],
          item: g
        })
      }
      this.isAnyGroup = gTree.length > 0

      // build groups tree
      const pUse = {}

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
              g.item.key = 'pgr-' + pc.ChildGroupId + '-' + this.nextId++
              g.path.push(g.gId)
              gProc.push(g)
              gpNow.item.children.push(g.item)
            }
          }

          // if this is a child leaf parameter
          if (pc.ChildLeafId >= 0) {
            const p = Mdf.paramTextById(this.theModel, pc.ChildLeafId)
            if (Mdf.isParam(p.Param)) {
              if (!this.isNoHidden) {
                this.isAnyHidden = this.isAnyHidden || p.Param.IsHidden
                if (!this.isShowHidden && p.Param.IsHidden) continue // skip hidden parameter
              }
              if (isPflt) {
                this.isAnyFiltered = this.isAnyFiltered || !fpLst[p.Param.Name]
                if (!this.isShowFiltered && !fpLst[p.Param.Name]) continue // skip filtered out parameter
              }

              gpNow.item.children.push({
                key: 'ptl-' + pc.ChildLeafId + '-' + this.nextId++,
                label: p.Param.Name,
                descr: Mdf.descrOfDescrNote(p),
                children: [],
                link: this.isParamLeafClick ? '/?model=' + encodeURIComponent(mDgst) + '&run=' + encodeURIComponent(this.runDigest) + '&parameter=' + encodeURIComponent(p.Param.Name) : '',
                parts: '',
                isGroup: false,
                isAbout: true,
                isAboutEmpty: false
              })
              pUse[p.Param.ParamId] = true
            }
          }
        }
      }

      // walk the tree and remove empty branches
      gTree = Tsc.removeEmptyGroups(gTree)

      // add parameters which are not included in any group
      const leafUse = {}

      for (let k = 0; k < gLst.length; k++) {
        if (!gLst[k].Group.IsParam) continue // skip output tables group

        for (const pc of gLst[k].Group.GroupPc) {
          if (pc.ChildLeafId >= 0) leafUse[pc.ChildLeafId] = true // store leaf parameter id
        }
      }

      let nLeaf = 0
      for (const p of this.theModel.ParamTxt) {
        if (!this.isNoHidden) {
          this.isAnyHidden = this.isAnyHidden || p.Param.IsHidden
          if (!this.isShowHidden && p.Param.IsHidden) continue // skip hidden parameter
        }
        if (isPflt) {
          this.isAnyFiltered = this.isAnyFiltered || !fpLst[p.Param.Name]
          if (!this.isShowFiltered && !fpLst[p.Param.Name]) continue // skip filtered out parameter
        }

        if (!leafUse[p.Param.ParamId]) { // if parameter is not a leaf of any group
          gTree.push({
            key: 'ptl-' + p.Param.ParamId + '-' + this.nextId++,
            label: p.Param.Name,
            descr: Mdf.descrOfDescrNote(p),
            children: [],
            link:  this.isParamLeafClick ? '/?model=' + encodeURIComponent(mDgst) + '&run=' + encodeURIComponent(this.runDigest) + '&parameter=' + encodeURIComponent(p.Param.Name) : '',
            parts: '',
            isGroup: false,
            isAbout: true,
            isAboutEmpty: false
          })
          pUse[p.Param.ParamId] = true
        }
        if (pUse[p.Param.ParamId]) nLeaf++
      }

      return { tree: gTree, leafCount: nLeaf }
    }
  },

  mounted () {
    this.paramTreeStore = {
      // push tree expanded keys into store for that model name and tree kind
      expandPush: (expKeys) => {
        this.dispatchTreeExpandedKeys(Mdf.modelName(this.theModel), 'run-parameter-list', expKeys)
      },
      // push tree expanded keys into store for that model name and tree kind
      expandPull: () => {
        return this.treeExpandedKeys(Mdf.modelName(this.theModel), 'run-parameter-list')
      }
    }
    this.doRefresh()
  }
}
</script>
