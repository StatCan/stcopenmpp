<template>

  <om-table-tree
    :refresh-tickle="refreshTickle"
    :refresh-tree-tickle="refreshTreeTickle"
    :tree-data="entityTreeData"
    :label-kind="treeLabelKind"
    :is-expand="isTreeExpand"
    :tree-store="entityTreeStore"
    :is-any-group="isAnyEntity"
    :is-any-hidden="isAnyHidden"
    :is-show-hidden="isShowHidden"
    :is-group-click="false"
    :is-leaf-click="isAttrClick"
    :is-add="isAddEntityAttr"
    :is-add-group="isAddEntity"
    :is-add-disabled="isAddDisabled"
    :is-remove="isRemoveEntityAttr"
    :is-remove-group="isRemoveEntity"
    :is-remove-disabled="isRemoveDisabled"
    :is-download-group="false"
    :is-download-group-disabled="isEntityDownloadDisabled"
    :filter-placeholder="$t('Find entity or attribute...')"
    :no-results-label="$t('No entity attributes found')"
    :no-nodes-label="$t('No model entities found or server offline')"
    :is-any-in-list="isAnyEntity"
    :in-list-on-label="inListOnLabel"
    :in-list-off-label="inListOffLabel"
    :in-list-icon="inListIcon"
    :is-in-list-clear="isInListClear"
    :in-list-clear-label="inListClearLabel"
    :in-list-clear-icon="inListClearIcon"
    @om-table-tree-show-hidden="onToogleHiddenNodes"
    @om-table-tree-clear-in-list="onClearInListFilter"
    @om-table-tree-group-select="onEntityClick"
    @om-table-tree-leaf-add="onAttrAddClick"
    @om-table-tree-group-add="onGroupAddClick"
    @om-table-tree-leaf-remove="onAttrRemoveClick"
    @om-table-tree-group-remove="onGroupRemoveClick"
    @om-table-tree-group-download="onDownloadClick"
    @om-table-tree-leaf-note="onShowAttrNote"
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
import { openURL } from 'quasar'

export default {
  name: 'EntityList',
  components: { OmTableTree },

  props: {
    runDigest: { type: String, default: '' },
    refreshTickle: { type: Boolean, default: false },
    refreshEntityTreeTickle: { type: Boolean, default: false },
    isTreeExpand: { type: Boolean, default: false },
    isEntityClick: { type: Boolean, default: false },
    isAttrClick: { type: Boolean, default: false },
    isAddEntityAttr: { type: Boolean, default: false },
    isAddEntity: { type: Boolean, default: false },
    isAddDisabled: { type: Boolean, default: false },
    isRemoveEntityAttr: { type: Boolean, default: false },
    isRemoveEntity: { type: Boolean, default: false },
    isRemoveDisabled: { type: Boolean, default: false },
    isEntityDownload: { type: Boolean, default: false },
    isEntityDownloadDisabled: { type: Boolean, default: false },
    nameFilter: { type: Array, default: () => [] }, // if not empty then use only entity.attribute included in this list
    isInListEnable: { type: Boolean, default: false },
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
      isAnyEntity: false,
      isAnyHidden: false,
      isShowHidden: false,
      isAnyFiltered: false,
      runCurrent: Mdf.emptyRunText(), // currently selected run
      entityTreeData: [],
      entityTreeStore: {
        expandPush: void 0, // store action to dispatch tree expanded keys
        expandPull: void 0 // store getter for tree expanded keys
      },
      groupLeafs: {}, // group leafs for all entities
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
      'runTextListUpdated',
      'groupEntityLeafs',
      'topEntityAttrs'
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
    refreshEntityTreeTickle () { this.doRefresh() },
    theModelUpdated () { this.doRefresh() },
    runTextListUpdated () { if (this.runDigest) this.doRefresh() }
  },

  emits: [
    'entity-tree-updated',
    'entity-clear-in-list',
    'entity-select',
    'entity-attr-add',
    'entity-add',
    'entity-group-add',
    'entity-attr-remove',
    'entity-remove',
    'entity-group-remove',
    'entity-attr-info-show',
    'entity-info-show',
    'entity-group-info-show'
  ],

  methods: {
    ...mapActions(useModelStore, ['runTextByDigest']),
    ...mapActions(useUiStateStore, ['dispatchTreeExpandedKeys']),

    // update entity attributes tree data and refresh tree view
    doRefresh () {
      if (this.runDigest) {
        this.runCurrent = this.runTextByDigest({ ModelDigest: Mdf.modelDigest(this.theModel), RunDigest: this.runDigest })
      }
      this.groupLeafs = Mdf.entityGroupLeafs(this.theModel, !this.isShowHidden)
      const td = this.makeEntityTreeData()
      this.entityTreeData = td.tree
      this.refreshTreeTickle = !this.refreshTreeTickle
      this.$emit('entity-tree-updated', td.entityCount, td.leafCount)
    },

    // show or hide internal entity attributes
    onToogleHiddenNodes (isShow) {
      this.isShowHidden = isShow
      this.doRefresh()
    },
    // click on clear filter: show all entity attributes
    onClearInListFilter () {
      this.$emit('entity-clear-in-list')
    },
    // click on entity: forward to parent
    onEntityClick (name, parts) {
      if (!this.checkAllAttrSize(name, true)) {
        return
      }
      this.$emit('entity-select', name, (parts?.entityName || '')) // there no attributes where size exceeds the limit: pass message to the parent
    },
    // click on add attribute: add attribute into microdata list
    onAttrAddClick (name, parts) {
      // check  attribute size: it should not exceed max size limit
      const eName = parts?.entityName || ''
      const ent = Mdf.entityTextByName(this.theModel, eName)
      if (!Mdf.isEntity(ent?.Entity)) {
        this.$q.notify({ type: 'negative', message: this.$t('Model entity not found: ') + eName })
        return
      }

      let isFound = false
      for (const ea of ent.EntityAttrTxt) {
        isFound = ea.Attr.Name === name

        if (isFound) {
          const ne = Mdf.typeEnumSizeById(this.theModel, ea.Attr.TypeId)

          if (this.maxTypeSize > 0 && ne > this.maxTypeSize) {
            this.$q.notify({ type: 'negative', message: this.$t('Entity attribute size exceed the limit: ') + this.maxTypeSize.toString() + ' ' + eName + '.' + (name || '') })
            return
          }
          break // attribute found and it size does not exceed the limit
        }
      }
      if (!isFound) {
        this.$q.notify({ type: 'negative', message: this.$t('Model entity attribute not found: ') + eName + '.' + (name || '') })
        return
      }

      this.$emit('entity-attr-add', name, eName, this.isShowHidden) // attribute size does not exceed the limit: pass message to the parent
    },
    // click on add entity or group of attributes: add all entity attributes into microdata list
    onGroupAddClick (name, parts) {
      if (!parts) return
      if (!this.checkAllAttrSize(name, parts.isEntity, parts.entityName)) {
        return // there is an attribute where size exceeds the limit: block event
      }
      if (parts?.isEntity) {
        this.$emit('entity-add', name, this.isShowHidden)
      } else {
        this.$emit('entity-group-add', name, this.isShowHidden, parts)
      }
    },

    // check all attributes of entity group of attributes: attribute size should not exceed max size limit
    checkAllAttrSize (name, isEntity, entName = '') {
      const eName = isEntity ? name : entName

      const ent = Mdf.entityTextByName(this.theModel, eName)
      if (!Mdf.isEntity(ent?.Entity)) {
        this.$q.notify({ type: 'negative', message: this.$t('Model entity not found: ') + (eName || '') })
        return
      }
      const eId = ent.Entity.EntityId
      const tops = this.topEntityAttrs?.[eId] // all top attributes, including hidden
      const eLfs = this.groupLeafs?.[eId]

      for (const ea of ent.EntityAttrTxt) {
        if (!this.isShowHidden && ea.Attr.IsHidden) continue // skip: ignore hidden attribute

        let isCheck = isEntity && !!tops?.attrId?.[ea.Attr.AttrId] // is it entity top attribute

        if (!isCheck) {
          if (!isEntity) { // check if this attribute is visible in the group
            isCheck = !!eLfs[name].leafs?.[ea.Attr.Name]
          } else { // check if this attribute is visible in any of entity groups
            for (const gn in eLfs) {
              isCheck = !!eLfs[gn].leafs?.[ea.Attr.Name]
              if (isCheck) break
            }
          }
        }
        if (!isCheck) continue // skip: attribute is not a member of the group and not a top entity attribute

        const ne = Mdf.typeEnumSizeById(this.theModel, ea.Attr.TypeId)

        if (this.maxTypeSize > 0 && ne > this.maxTypeSize) {
          this.$q.notify({ type: 'negative', message: this.$t('Entity attribute size exceed the limit: ') + this.maxTypeSize.toString() + ' ' + (eName || '') + '.' + (ea.Attr.Name || '') })
          return false
        }
      }
      return true // there no attributes where size exceeds the limit
    },

    // click on remove attribute: remove attribute from microdata list
    onAttrRemoveClick (name, parts) {
      this.$emit('entity-attr-remove', name, (parts?.entityName || ''))
    },
    // click on remove entity or group of attributes: remove all entity attributes from microdata list
    onGroupRemoveClick (name, parts) {
      if (!parts) return
      if (parts?.isEntity) {
        this.$emit('entity-remove', name)
      } else {
        this.$emit('entity-group-remove', name, parts)
      }
    },
    // click on show attribute notes dialog button
    onShowAttrNote (name, parts) {
      this.$emit('entity-attr-info-show', name, (parts?.entityName || ''))
    },
    // click on show group notes buttun: show entity notes or attributes group notes
    onShowGroupNote (name, parts) {
      if (!parts) return
      if (parts?.isEntity) {
        this.$emit('entity-info-show', name)
      } else {
        this.$emit('entity-group-info-show', name, (parts?.entityName || ''))
      }
    },
    // download entity microdata as csv
    onDownloadClick  (name, parts) {
      if (!name || !Mdf.isNotEmptyRunEntity(Mdf.runEntityByName(this.runCurrent, name))) {
        this.$q.notify({ type: 'negative', message: this.$t('Entity microdata not found or empty: ') + (name || '') + ' ' + this.runDigest })
        return
      }
      const u = this.omsUrl +
        '/api/model/' + encodeURIComponent(Mdf.modelDigest(this.theModel)) +
        '/run/' + encodeURIComponent(this.runDigest) +
        '/microdata/' + encodeURIComponent(name) +
        ((this.$q.platform.is.win) ? (this.idCsvDownload ? '/csv-id-bom' : '/csv-bom') : (this.idCsvDownload ? '/csv-id' : '/csv'))

      openURL(u)
    },

    // return tree of entity attributes: entities, groups of attributes and attributes are leafs
    makeEntityTreeData () {
      this.isAnyEntity = false
      this.isAnyHidden = false
      this.isAnyFiltered = false

      if (!Mdf.entityCount(this.theModel)) {
        return { tree: [], leafCount: 0 } // empty list of entities
      }
      if (!Mdf.isEntityTextList(this.theModel)) {
        this.$q.notify({ type: 'negative', message: this.$t('Model entity attributes list is empty or invalid') })
        return { tree: [], leafCount: 0 } // invalid list of entity attributes
      }
      const mDgst = Mdf.modelDigest(this.theModel)

      // if run specified then include only entity.attribute from this model run
      const eRun = {}
      const aRun = {}
      const isRun = this.runDigest && Mdf.isNotEmptyRunText(this.runCurrent)

      if (isRun) {
        for (const e of this.runCurrent.Entity) {
          if (e?.Name && Array.isArray(e?.Attr)) {
            let na = 0
            for (const a of e.Attr) {
              const name = e.Name + '.' + a
              aRun[name] = true
              na++
            }
            if (na > 0) {
              eRun[e.Name] = true
            }
          }
        }
      }

      // if filter names not empty then display only attributes from that name list
      const fltUse = {}
      for (const name of this.nameFilter) {
        fltUse[name] = true
      }

      // add entities as a top level groups into groups tree, use attributes as leafs
      const gTree = []
      const egIdx = {} // map entity id to the index of entity top group in the top level of the tree
      const aUse = {} // include only entity.attribute filtered by current run, names filter, hidden status
      const eIdUse = {} // include only entity id used as entity top group
      const eIdName = {} // map entity id to entity name

      for (const ent of this.theModel.EntityTxt) {
        eIdName[ent.Entity.EntityId] = ent.Entity.Name

        if (isRun && !eRun[ent.Entity.Name]) continue // skip: this entity does not exist in that run

        // entity top group: check if any attributes allowed for that entity
        let isAny = false

        const g = {
          key: 'etgr-' + ent.Entity.EntityId,
          label: ent.Entity.Name,
          descr: Mdf.descrOfDescrNote(ent),
          children: [],
          link: this.isEntityClick ? '/?model=' + encodeURIComponent(mDgst) + '&run=' + encodeURIComponent(this.runDigest) + '&entity=' + encodeURIComponent(ent.Entity.Name) : '',
          parts: {
            isEntity: true,
            entityId: ent.Entity.EntityId,
            entityName: ent.Entity.Name,
            selfId: ent.Entity.EntityId
          },
          isGroup: true,
          isAbout: true,
          isAboutEmpty: false,
          isClick: this.isEntityClick,
          isDownload: this.isEntityDownload
        }

        // if there are any attribute in that entity top group
        for (const ea of ent.EntityAttrTxt) {
          const name = ent.Entity.Name + '.' + ea.Attr.Name
          if (isRun && !aRun[name]) continue // skip: this entity.attribute does not exist in that run

          // if in-list filter enabled then use only entity.attribute from the list
          if (this.isInListEnable) {
            if (!fltUse?.[name]) {
              this.isAnyFiltered = true
              continue // skip filtered out entity attribute
            }
          }

          this.isAnyHidden = this.isAnyHidden || ea.Attr.IsInternal
          if (!this.isShowHidden && ea.Attr.IsInternal) continue // skip hidden internal attribute if reqired

          aUse[name] = true
          isAny = true
        }

        if (isAny) { // append entity top group to the tree
          egIdx[ent.Entity.EntityId] = gTree.length
          gTree.push(g)
          eIdUse[ent.Entity.EntityId] = true
          this.isAnyEntity = true
        }
      }
      if (!this.isAnyEntity) return { tree: [], leafCount: 0 } // empty list of entities

      // make list of the groups:
      // remove groups where entity is not included or no attributes or hidden groups
      const makeGkey = (eId, gId) => 'egr-' + eId + '-' + gId
      const gUse = {}

      for (const eGrp of this.theModel.EntityGroupTxt) {
        const eId = eGrp.Group.EntityId
        const gId = eGrp.Group.GroupId
        const eName = eIdName[eId]

        if (!eIdUse[eId]) continue // skip: this entity is not included in top entity groups

        let isAny = false
        const lg = this.groupEntityLeafs?.[eId]?.[eGrp.Group.Name]
        if (lg) {
          for (const ln in lg.leafs) {
            isAny = aUse?.[eName + '.' + ln]
            if (isAny) break
          }
        }
        if (!isAny) continue // skip: there are no attributes in that group

        this.isAnyHidden = this.isAnyHidden || eGrp.Group.IsHidden
        if (!this.isShowHidden && eGrp.Group.IsHidden) continue // skip hidden group

        const isNote = Mdf.noteOfDescrNote(eGrp) !== ''
        const egKey = makeGkey(eId, gId)
        gUse[egKey] = {
          group: eGrp,
          item: {
            key: egKey,
            label: eGrp.Group.Name,
            descr: Mdf.descrOfDescrNote(eGrp),
            children: [],
            parts: {
              isEntity: false,
              entityId: eId,
              entityName: eName,
              selfId: gId
            },
            isGroup: true,
            isAbout: isNote,
            isAboutEmpty: !isNote
          }
        }
      }

      // add first level groups as children of entity top groups
      const gProc = []

      for (const eGrp of this.theModel.EntityGroupTxt) {
        const gId = eGrp.Group.GroupId
        const eId = eGrp.Group.EntityId
        const egKey = makeGkey(eId, gId)

        if (!gUse[egKey]) continue // skip this group: it is not included into the tree

        const isNotTop = this.theModel.EntityGroupTxt.findIndex((egt) => {
          if (egt.Group.EntityId === eId && egt.Group.GroupId === gId) return false
          if (egt.Group.GroupPc.length <= 0) return false
          return egt.Group.GroupPc.findIndex((pc) => pc.EntityId === eId && pc.ChildGroupId === gId) >= 0
        }) >= 0
        if (isNotTop) continue // not a top level group

        // add first level group to the entity top group
        const g = Mdf.deepClone(gUse[egKey].item)
        const i = egIdx?.[eId]
        if (i >= 0 && i < gTree.length) gTree[i].children.push(g)

        gProc.push({
          key: egKey,
          eId: eGrp.Group.EntityId,
          gId: eGrp.Group.GroupId,
          path: [eGrp.Group.GroupId],
          item: g
        })
      }

      // build groups tree
      while (gProc.length > 0) {
        const gpNow = gProc.pop()
        const gTxt = gUse[gpNow.key].group

        // make all children of current item
        for (const pc of gTxt.Group.GroupPc) {
          // if this is a child group
          if (pc.ChildGroupId >= 0) {
            const egKey = makeGkey(gpNow.eId, pc.ChildGroupId)
            const gChildUse = gUse?.[egKey]
            if (gChildUse) {
              if (!this.isShowHidden && gChildUse.group.Group.IsHidden) continue // skip hidden group

              // check for circular reference
              if (gpNow.path.indexOf(pc.ChildGroupId) >= 0) {
                console.warn('Error: circular refernece to group:', pc.ChildGroupId, 'path:', gpNow.path)
                continue // skip this group
              }

              const g = {
                key: egKey,
                eId: gpNow.eId,
                gId: pc.ChildGroupId,
                path: Mdf.deepClone(gpNow.path),
                item: Mdf.deepClone(gChildUse.item)
              }
              g.item.key = egKey + '-' + pc.ChildPos
              g.path.push(g.gId)
              gProc.push(g)
              gpNow.item.children.push(g.item)
            }
          }

          // if this is a child leaf attribute
          if (pc.AttrId >= 0) {
            const ea = Mdf.entityAttrTextById(this.theModel, gpNow.eId, pc.AttrId)
            if (!Mdf.isNotEmptyEntityAttr(ea)) continue
            const eName = eIdName?.[gpNow.eId]
            if (!eName || !aUse[eName + '.' + ea.Attr.Name]) continue // skip: attribute: it is not in the filter list

            gpNow.item.children.push({
              key: 'etl-' + ea.Attr.EntityId + '-' + ea.Attr.AttrId + '-' + pc.ChildPos,
              label: ea.Attr.Name,
              descr: Mdf.descrOfDescrNote(ea),
              children: [],
              parts: {
                isEntity: false,
                entityId: gpNow.eId,
                entityName: eName,
                selfId: pc.AttrId
              },
              isGroup: false,
              isAbout: true,
              isAboutEmpty: false
            })
          }
        }
      }

      // add attributes which are not included in any group (not a leaf nodes)
      const ta = this.topEntityAttrs
      let nLeaf = 0

      for (const ent of this.theModel.EntityTxt) {
        const eId = ent.Entity.EntityId

        for (const ea of ent.EntityAttrTxt) {
          const name = ent.Entity.Name + '.' + ea.Attr.Name

          if (!aUse[name]) continue // skip: this entity.attribute not in use
          if (!ta?.[eId]?.attrId?.[ea.Attr.AttrId]) {
            nLeaf++
            continue // attribute included into the group
          }

          const i = egIdx?.[eId]
          if (i >= 0) {
            gTree[i].children.push({
              key: 'al-' + eId + '-' + ea.Attr.AttrId,
              label: ea.Attr.Name,
              descr: Mdf.descrOfDescrNote(ea),
              children: [],
              parts: {
                isEntity: false,
                entityId: eId,
                entityName: ent.Entity.Name,
                selfId: ea.Attr.AttrId
              },
              isGroup: false,
              isAbout: true,
              isAboutEmpty: false
            })
            nLeaf++
          }
        }
      }

      return { tree: gTree, entityCount: gTree.length, leafCount: nLeaf }
    }
  },

  mounted () {
    this.entityTreeStore = {
      // push tree expanded keys into store for that model name and tree kind
      expandPush: (expKeys) => {
        this.dispatchTreeExpandedKeys(Mdf.modelName(this.theModel), 'entity-list', expKeys)
      },
      // push tree expanded keys into store for that model name and tree kind
      expandPull: () => {
        return this.treeExpandedKeys(Mdf.modelName(this.theModel), 'entity-list')
      }
    }
    this.doRefresh()
  }
}
</script>
