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
    @om-table-tree-show-hidden="onToogleHiddenNodes"
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
  name: 'WorksetParameterList',
  components: { OmTableTree },

  props: {
    worksetName: { type: String, required: true },
    refreshTickle: { type: Boolean, default: false },
    refreshParamTreeTickle: { type: Boolean, default: false },
    isGroupClick: { type: Boolean, default: false },
    isParamLeafClick: { type: Boolean, default: false },
    isAdd: { type: Boolean, default: false },
    isAddGroup: { type: Boolean, default: false },
    isAddDisabled: { type: Boolean, default: false },
    isRemove: { type: Boolean, default: false },
    isRemoveGroup: { type: Boolean, default: false },
    isRemoveDisabled: { type: Boolean, default: false },
    isParamDownload: { type: Boolean, default: false },
    isParamDownloadDisabled: { type: Boolean, default: false }
  },

  data () {
    return {
      worksetCurrent: Mdf.emptyWorksetText(), // currently selected workset
      refreshTreeTickle: false,
      isAnyGroup: false,
      isAnyHidden: false,
      isShowHidden: false,
      paramTreeData: [],
      paramTreeStore: {
        expandPush: void 0, // store action to dispatch tree expanded keys
        expandPull: void 0 // store getter for tree expanded keys
      },
      nextId: 100
    }
  },

  computed: {
    isWsReadonly () { return !!this.worksetCurrent?.IsReadonly },
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
    worksetName () { this.doRefresh() },
    refreshTickle  () { this.doRefresh() },
    refreshParamTreeTickle () { this.doRefresh() },
    theModelUpdated () { this.doRefresh() }
  },

  emits: [
    'set-parameter-tree-updated',
    'set-parameter-select',
    'set-parameter-add',
    'set-parameter-group-add',
    'set-parameter-remove',
    'set-parameter-group-remove',
    'set-parameter-info-show',
    'set-parameter-group-info-show'
  ],

  methods: {
    ...mapActions(useModelStore, ['worksetTextByName']),
    ...mapActions(useUiStateStore, ['dispatchTreeExpandedKeys']),

    // update parameters tree data and refresh tree view
    doRefresh () {
      this.worksetCurrent = this.worksetTextByName({ ModelDigest: Mdf.modelDigest(this.theModel), Name: this.worksetName })
      const td = this.makeParamTreeData()
      this.paramTreeData = td.tree
      this.refreshTreeTickle = !this.refreshTreeTickle
      this.$emit('set-parameter-tree-updated', td.leafCount)
    },

    // show or hide hidden parameters and groups
    onToogleHiddenNodes (isShow) {
      this.isShowHidden = isShow
      this.doRefresh()
    },
    // click on parameter: open current workset parameter values tab
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

      this.$emit('set-parameter-select', name, parts) // pass message to the parent
    },
    // click on add parameter: add current workset parameter
    onAddClick (name, parts) {
      this.$emit('set-parameter-add', name, parts)
    },
    // click on add group: add group from current workset
    onGroupAddClick (name, parts) {
      this.$emit('set-parameter-group-add', name, parts)
    },
    // click on remove parameter: remove current workset parameter
    onRemoveClick (name, parts) {
      this.$emit('set-parameter-remove', name, parts)
    },
    // click on remove group: remove group from current workset
    onGroupRemoveClick (name, parts) {
      this.$emit('set-parameter-group-remove', name, parts)
    },
    // click on show parameter notes dialog button
    onShowParamNote (name, parts) {
      this.$emit('set-parameter-info-show', name, parts)
    },
    // click on show group notes dialog button
    onShowGroupNote (name, parts) {
      this.$emit('set-parameter-group-info-show', name, parts)
    },
    // download parameter as csv
    onDownloadClick  (name, parts) {
      if (!name || !Mdf.isWorksetParamByName(this.worksetCurrent, name)) {
        this.$q.notify({ type: 'negative', message: this.$t('Parameter not found: ') + (name || '') })
        return
      }
      const u = this.omsUrl +
          '/api/model/' + Mdf.modelDigest(this.theModel) +
          '/workset/' + encodeURIComponent(this.worksetName) +
          '/parameter/' + encodeURIComponent(name) +
          // ((this.$q.platform.is.win) ? '/csv-bom' : '/csv')
          ((this.$q.platform.is.win) ? (this.idCsvDownload ? '/csv-id-bom' : '/csv-bom') : (this.idCsvDownload ? '/csv-id' : '/csv'))

      openURL(u)
    },

    // return tree of model parameters
    makeParamTreeData () {
      this.isAnyGroup = false
      this.isAnyHidden = false

      if (!Mdf.paramCount(this.theModel) || !Mdf.isLength(this.worksetCurrent.Param)) {
        return { tree: [], leafCount: 0 } // empty list of parameters
      }
      if (!Mdf.isParamTextList(this.theModel)) {
        this.$q.notify({ type: 'negative', message: this.$t('Model parameters list is empty or invalid') })
        return { tree: [], leafCount: 0 } // invalid list of parameters
      }
      const mDgst = Mdf.modelDigest(this.theModel)

      // make parameters map: map parameter id to parameter node
      const pUse = {}
      const wpLst = this.worksetCurrent.Param

      for (const p of this.theModel.ParamTxt) {
        // skip parameter if it is not included in workset
        if (wpLst.findIndex((pw) => { return pw.Name === p.Param.Name }) < 0) continue

        // skip hidden parameter, if required
        this.isAnyHidden = this.isAnyHidden || p.Param.IsHidden
        if (!this.isShowHidden && p.Param.IsHidden) continue

        pUse[p.Param.ParamId] = {
          param: p,
          isLeaf: false,
          item: {
            key: 'ptl-' + p.Param.ParamId + '-' + this.nextId++,
            label: p.Param.Name,
            descr: Mdf.descrOfDescrNote(p),
            link: this.isParamLeafClick ? '/?model=' + encodeURIComponent(mDgst) + '&set=' + encodeURIComponent(this.worksetName) + '&parameter=' + encodeURIComponent(p.Param.Name) : '',
            children: [],
            parts: '',
            isGroup: false,
            isAbout: true,
            isAboutEmpty: false
          }
        }
      }

      // make group map: map group id to group node
      // if group is not hidden and include any workset parameter or other workset group
      const gUse = {}

      let isAny = false
      do {
        isAny = false

        for (const g of this.theModel.GroupTxt) {
          if (!g.Group.IsParam) continue // skip output tables group

          if (gUse[g.Group.GroupId]) continue // group is already processed

          // check is this group has any workset parameters or other workset groups
          let isOk = g.Group.GroupPc.findIndex((pc) => { return !!pUse[pc.ChildLeafId] }) >= 0
          if (!isOk) isOk = g.Group.GroupPc.findIndex((pc) => { return !!gUse[pc.ChildGroupId] }) >= 0

          // hide group if required
          if (isOk) {
            isOk = this.isShowHidden || !g.Group.IsHidden
            this.isAnyHidden = this.isAnyHidden || g.Group.IsHidden
          }
          if (!isOk) continue // skip this group

          // include group in result list
          isAny = true

          const gId = g.Group.GroupId
          const isNote = Mdf.noteOfDescrNote(g) !== ''

          gUse[gId] = {
            group: g,
            item: {
              key: 'pgr-' + gId + '-' + this.nextId++,
              label: g.Group.Name,
              descr: Mdf.descrOfDescrNote(g),
              children: [],
              parts: '',
              isGroup: true,
              isAbout: isNote,
              isAboutEmpty: !isNote
            }
          }
        }
      }
      while (isAny)

      // add top level groups as starting point into groups tree
      let gTree = []
      const gProc = []

      for (const g of this.theModel.GroupTxt) {
        const iGid = g.Group.GroupId
        if (!gUse[iGid]) continue // skip: this is not a workset group

        const isNotTop = this.theModel.GroupTxt.findIndex((gt) => {
          if (!gt.Group.IsParam) return false
          if (gt.Group.GroupId === iGid) return false
          if (gt.Group.GroupPc.length <= 0) return false
          return gt.Group.GroupPc.findIndex((pc) => pc.ChildGroupId === iGid) >= 0
        }) >= 0
        if (isNotTop) continue // not a top level group

        const cg = Mdf.deepClone(gUse[iGid].item)
        gTree.push(cg)
        gProc.push({
          gId: iGid,
          path: [iGid],
          item: cg
        })
      }
      this.isAnyGroup = gTree.length > 0

      // build groups tree
      while (gProc.length > 0) {
        const gpNow = gProc.pop()
        if (!gUse[gpNow.gId]) continue // skip: this is not a workset group

        // make all children of current group
        const gTxt = gUse[gpNow.gId].group

        for (const pc of gTxt.Group.GroupPc) {
          // if this is a child group
          if (pc.ChildGroupId >= 0) {
            const gChildUse = gUse[pc.ChildGroupId]
            if (gChildUse) {
              if (!this.isShowHidden && gChildUse.group.Group.IsHidden) continue // skip hidden group

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
            const p = pUse[pc.ChildLeafId]
            if (p) {
              const pn = Mdf.deepClone(p.item)
              pn.key = 'ptl-' + pc.ChildLeafId + '-' + this.nextId++
              gpNow.item.children.push(pn)
              p.isLeaf = true
            }
          }
        }
      }

      // walk the tree and remove empty branches
      gTree = Tsc.removeEmptyGroups(gTree)

      // add parameters which are not included in any group (not a leaf)
      const leafUse = {}
      for (const g of this.theModel.GroupTxt) {
        if (!g.Group.IsParam) continue // skip output tables group

        for (const pc of g.Group.GroupPc) {
          if (pc.ChildLeafId >= 0) leafUse[pc.ChildLeafId] = true // store leaf parameter id
        }
      }

      let nLeaf = 0
      for (const p of this.theModel.ParamTxt) {
        const pId = p.Param.ParamId
        const pw = pUse[pId]
        if (!pw) continue // skip: this is not a workset parameter

        if (!leafUse[pId]) { // if parameter is not a leaf
          gTree.push(pw.item)
          pw.isLeaf = true
        }
        if (pw.isLeaf) nLeaf++
      }

      return { tree: gTree, leafCount: nLeaf }
    }
  },

  mounted () {
    this.worksetCurrent = this.worksetTextByName({ ModelDigest: Mdf.modelDigest(this.theModel), Name: this.worksetName })
    this.paramTreeStore = {
      // push tree expanded keys into store for that model name and tree kind
      expandPush: (expKeys) => {
        this.dispatchTreeExpandedKeys(Mdf.modelName(this.theModel), 'ws-parameter-list', expKeys)
      },
      // push tree expanded keys into store for that model name and tree kind
      expandPull: () => {
        return this.treeExpandedKeys(Mdf.modelName(this.theModel), 'ws-parameter-list')
      }
    }
    this.doRefresh()
  }
}
</script>
