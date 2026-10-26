<!-- select microdata entity attributes: single tree with checkboxes, search, select all / clear and "selected only" filter -->
<template>
<div class="entity-select-tree">

  <div class="row items-center no-wrap q-gutter-x-xs q-mb-xs">
    <q-btn
      @click="onToggleExpand"
      unelevated
      dense
      color="primary"
      class="col-auto rounded-borders"
      :icon="isExpanded ? 'mdi-collapse-all-outline' : 'mdi-expand-all-outline'"
      :title="isExpanded ? $t('Collapse all') : $t('Expand all')"
      :aria-label="isExpanded ? $t('Collapse all') : $t('Expand all')"
      />
    <q-input
      v-model="filterText"
      debounce="250"
      outlined
      dense
      clearable
      class="col"
      :placeholder="$t('Find entity or attribute...')"
      >
      <template v-slot:prepend>
        <q-icon name="search" />
      </template>
    </q-input>
    <q-chip
      dense
      square
      :color="selectedCount ? 'primary' : 'grey-7'"
      text-color="white"
      class="col-auto"
      :title="$t('Selected entity attributes')"
      >
      {{ selectedCount.toLocaleString() + ' / ' + totalCount.toLocaleString() }}
    </q-chip>
  </div>

  <div class="row items-center q-gutter-x-sm q-mb-xs">
    <q-btn
      @click="onSelectAll"
      :disable="selectedCount >= totalCount"
      flat
      dense
      no-caps
      color="primary"
      icon="mdi-checkbox-multiple-marked-outline"
      :label="isFiltering ? $t('Select found') : $t('Select all')"
      />
    <q-btn
      @click="onClearAll"
      :disable="selectedCount <= 0"
      flat
      dense
      no-caps
      color="primary"
      icon="mdi-checkbox-multiple-blank-outline"
      :label="isFiltering ? $t('Clear found') : $t('Clear all')"
      />
    <q-toggle
      v-model="isSelectedOnly"
      dense
      :label="$t('Selected only')"
      />
    <q-toggle
      v-if="isAnyHidden"
      v-model="isShowHidden"
      dense
      :label="$t('Show hidden')"
      />
  </div>

  <q-tree
    ref="tree"
    :nodes="treeData"
    node-key="key"
    tick-strategy="leaf"
    :ticked="tickedKeys"
    @update:ticked="onTicked"
    v-model:expanded="expandedKeys"
    :filter="filterKey"
    :filter-method="doFilter"
    no-transition
    dense
    :no-results-label="$t('No entity attributes found')"
    :no-nodes-label="$t('No microdata entities found or server offline')"
    >
    <template v-slot:default-header="prop">
      <div class="row no-wrap items-center full-width est-node">
        <div class="col" :class="{ 'text-grey-6': prop.node.disabled }">
          <q-icon v-if="prop.node.isEntity" name="mdi-microscope" class="q-mr-xs" />
          <span v-if="treeLabelKind !== 'descr-only' || !prop.node.descr" :class="{ 'text-weight-medium': prop.node.isGroup }">{{ prop.node.label }}</span>
          <span v-if="treeLabelKind !== 'name-only' && prop.node.descr" :class="treeLabelKind === 'descr-only' ? '' : 'q-pl-sm om-text-descr'">{{ prop.node.descr }}</span>
          <q-icon
            v-if="prop.node.disabled"
            name="mdi-alert-outline"
            color="warning"
            class="q-ml-xs"
            :title="$t('Entity attribute size exceed the limit: ') + maxTypeSize.toString()"
            />
        </div>
        <q-badge
          v-if="prop.node.isGroup"
          outline
          :color="groupSelected(prop.node) ? 'primary' : 'grey-7'"
          class="col-auto q-ml-sm"
          >{{ groupSelected(prop.node) + ' / ' + prop.node.names.length }}</q-badge>
        <q-btn
          @click.stop="onInfo(prop.node)"
          flat
          round
          dense
          size="sm"
          icon="mdi-information-outline"
          class="col-auto q-ml-xs"
          :title="$t('About') + ' ' + prop.node.label"
          :aria-label="$t('About') + ' ' + prop.node.label"
          />
      </div>
    </template>
  </q-tree>

</div>
</template>

<script>
import { mapState } from 'pinia'
import { useModelStore } from '../stores/model'
import { useServerStateStore } from '../stores/server-state'
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'
import { tabStateMixin } from 'src/cache/tab-state'

export default {
  name: 'EntitySelectTree',
  mixins: [tabStateMixin],

  // cache search text, filters and expanded groups
  tabState: {
    fields: ['filterText', 'isSelectedOnly', 'isShowHidden', 'expandedKeys'],
    key () { return 'entity-select:' + Mdf.modelDigest(this.theModel) }
  },

  props: {
    attrs: { type: Array, default: () => [] } // selected microdata: array of 'Entity.attribute' names
  },

  data () {
    return {
      filterText: '',
      isSelectedOnly: false,
      isShowHidden: false,
      isAnyHidden: false,
      isExpanded: false,
      expandedKeys: [],
      treeData: [],
      leafs: [], // all leaf nodes of the tree: { key, name: 'Entity.attribute', disabled }
      allNames: [], // names of all entity attributes: 'Entity.attribute'
      tooBig: {} // attributes which cannot be used: size of attribute type exceeds the limit
    }
  },

  computed: {
    selectedSet () { return new Set(this.attrs || []) },
    selectedCount () { return this.allNames.reduce((n, an) => n + (this.selectedSet.has(an) ? 1 : 0), 0) },
    totalCount () { return this.allNames.length },
    isFiltering () { return (this.filterText || '') !== '' },
    maxTypeSize () {
      const s = Mdf.configEnvValue(this.serverConfig, 'OM_CFG_TYPE_MAX_LEN')
      return ((s || '') !== '') ? (parseInt(s) || 0) : 0
    },
    // tree keys of selected attributes: attribute may appear in multiple groups and all occurrences are ticked
    tickedKeys () {
      const keys = []
      for (const lf of this.leafs) {
        if (this.selectedSet.has(lf.name)) keys.push(lf.key)
      }
      return keys
    },
    // q-tree re-filter when any of filter conditions changed, filter must be non-empty to apply
    filterKey () {
      return (this.filterText || '') + '\u0001' + (this.isSelectedOnly ? '1' : '0') + (this.isSelectedOnly ? '\u0001' + this.selectedCount : '')
    },

    ...mapState(useModelStore, ['theModel', 'theModelUpdated']),
    ...mapState(useServerStateStore, { serverConfig: 'config' }),
    ...mapState(useUiStateStore, ['treeLabelKind'])
  },

  watch: {
    theModelUpdated () { this.doRefresh() },
    isShowHidden () { this.doRefresh() },
    maxTypeSize () { this.doRefresh() },
    filterText (t) {
      if (t) this.$nextTick(() => this.expandAll()) // show search results
    }
  },

  emits: ['update:attrs', 'entity-info-show', 'entity-group-info-show', 'entity-attr-info-show'],

  methods: {
    // build tree: entities, groups of attributes and attributes
    doRefresh () {
      const m = this.theModel
      const eLst = Array.isArray(m?.EntityTxt) ? m.EntityTxt : []
      const gAll = Array.isArray(m?.EntityGroupTxt) ? m.EntityGroupTxt : []

      this.allNames = []
      this.tooBig = {}
      this.isAnyHidden = false

      let n = 0
      const leafs = []
      const tree = []

      for (const ent of eLst) {
        if (!ent?.Entity || !Array.isArray(ent?.EntityAttrTxt)) continue
        const eId = ent.Entity.EntityId
        const eName = ent.Entity.Name
        const eDescr = Mdf.descrOfDescrNote(ent)

        // attributes of the entity
        const aById = {}
        for (const ea of ent.EntityAttrTxt) {
          if (!ea?.Attr) continue
          const name = eName + '.' + ea.Attr.Name
          aById[ea.Attr.AttrId] = ea
          this.allNames.push(name)
          if (ea.Attr.IsInternal) this.isAnyHidden = true

          if (this.maxTypeSize > 0 && Mdf.typeEnumSizeById(m, ea.Attr.TypeId) > this.maxTypeSize) this.tooBig[name] = true
        }

        // groups of attributes of the entity
        const gLst = gAll.filter(g => g?.Group && g.Group.EntityId === eId)
        const gById = {}
        const childGroups = new Set()
        for (const g of gLst) {
          gById[g.Group.GroupId] = g
          for (const pc of (g.Group.GroupPc || [])) {
            if (pc.ChildGroupId >= 0) childGroups.add(pc.ChildGroupId)
          }
        }
        const inGroup = new Set()

        const makeLeaf = (ea, ctx) => {
          const name = eName + '.' + ea.Attr.Name
          const descr = Mdf.descrOfDescrNote(ea)
          const node = {
            key: 'a' + (n++) + ':' + name,
            name: name,
            label: ea.Attr.Name,
            descr: descr,
            isGroup: false,
            entityName: eName,
            disabled: !!this.tooBig[name], // attribute size exceeds the limit: cannot be used
            ctx: ctx, // text of parent entity and groups, used for search: all attributes of matched group are found
            text: (ea.Attr.Name + ' ' + descr).toLowerCase()
          }
          leafs.push(node)
          return node
        }

        const addNames = (node, names) => {
          for (const an of names) {
            if (!node.names.includes(an)) node.names.push(an)
          }
        }

        // make group node and all children, skip circular references, hidden and empty groups
        const makeGroup = (g, path, ctx) => {
          if (g.Group.IsHidden) this.isAnyHidden = true
          if (g.Group.IsHidden && !this.isShowHidden) return null

          const gDescr = Mdf.descrOfDescrNote(g)
          const gCtx = ctx + ' ' + (g.Group.Name + ' ' + gDescr).toLowerCase()
          const node = { key: 'g' + (n++) + ':' + eName + '.' + g.Group.Name, name: g.Group.Name, label: g.Group.Name, descr: gDescr, isGroup: true, entityName: eName, children: [], names: [] }

          for (const pc of (g.Group.GroupPc || [])) {
            if (pc.ChildGroupId >= 0 && gById[pc.ChildGroupId] && !path.includes(pc.ChildGroupId)) {
              const c = makeGroup(gById[pc.ChildGroupId], [...path, pc.ChildGroupId], gCtx)
              if (c) {
                node.children.push(c)
                addNames(node, c.names)
              }
            }
            if (pc.AttrId >= 0 && aById[pc.AttrId]) {
              const ea = aById[pc.AttrId]
              inGroup.add(pc.AttrId)
              if (ea.Attr.IsInternal && !this.isShowHidden) continue
              const lf = makeLeaf(ea, gCtx)
              node.children.push(lf)
              addNames(node, [lf.name])
            }
          }
          return node.children.length > 0 ? node : null
        }

        // entity is a top level node: top level groups and attributes which are not in any group
        const eCtx = (eName + ' ' + eDescr).toLowerCase()
        const eNode = { key: 'e' + (n++) + ':' + eName, name: eName, label: eName, descr: eDescr, isGroup: true, isEntity: true, entityName: eName, children: [], names: [] }

        for (const g of gLst) {
          if (childGroups.has(g.Group.GroupId)) continue // not a top level group
          const gn = makeGroup(g, [g.Group.GroupId], eCtx)
          if (gn) {
            eNode.children.push(gn)
            addNames(eNode, gn.names)
          }
        }
        for (const ea of ent.EntityAttrTxt) {
          if (!ea?.Attr || inGroup.has(ea.Attr.AttrId)) continue
          if (ea.Attr.IsInternal && !this.isShowHidden) continue
          const lf = makeLeaf(ea, eCtx)
          eNode.children.push(lf)
          addNames(eNode, [lf.name])
        }
        if (eNode.children.length > 0) tree.push(eNode)
      }

      this.leafs = Object.freeze(leafs)
      this.treeData = Object.freeze(tree)

      // keep only expanded keys which exist in the new tree
      const keys = new Set()
      const walk = (nodes) => {
        for (const nd of nodes) {
          if (nd.isGroup) {
            keys.add(nd.key)
            walk(nd.children)
          }
        }
      }
      walk(tree)
      this.expandedKeys = this.expandedKeys.filter(k => keys.has(k))
      this.isExpanded = this.expandedKeys.length > 0
    },

    // filter tree nodes: groups are visible if any child visible
    doFilter (node, filter) {
      if (node.isGroup) return false
      if (this.isSelectedOnly && !this.selectedSet.has(node.name)) return false
      return this.isLeafFound(node)
    },
    // return true if leaf node match search text
    isLeafFound (node) {
      const t = (this.filterText || '').toLowerCase()
      return !t || node.text.includes(t) || node.ctx.includes(t)
    },

    // tree ticked keys updated: apply added and removed keys as attribute names
    onTicked (keys) {
      const now = new Set(keys)
      const before = new Set(this.tickedKeys)
      const sel = new Set(this.attrs || [])

      for (const lf of this.leafs) {
        if (now.has(lf.key) && !before.has(lf.key) && !lf.disabled) sel.add(lf.name)
      }
      for (const lf of this.leafs) {
        if (!now.has(lf.key) && before.has(lf.key)) sel.delete(lf.name)
      }
      this.emitAttrs(sel)
    },

    // select all visible attributes or all found attributes if search text is not empty
    // hidden (internal) attributes are included only if "show hidden" is on
    onSelectAll () {
      const sel = new Set(this.attrs || [])
      for (const lf of this.leafs) {
        if (!lf.disabled && this.isLeafFound(lf)) sel.add(lf.name)
      }
      this.emitAttrs(sel)
    },
    // clear all attributes or all found attributes if search text is not empty
    onClearAll () {
      if (!this.isFiltering) {
        this.emitAttrs(new Set())
        return
      }
      const sel = new Set(this.attrs || [])
      for (const lf of this.leafs) {
        if (this.isLeafFound(lf)) sel.delete(lf.name)
      }
      this.emitAttrs(sel)
    },
    // emit selected attribute names in model order, only if selection changed
    emitAttrs (sel) {
      const cur = this.selectedSet
      let isSame = sel.size === cur.size
      if (isSame) {
        for (const an of sel) {
          if (!cur.has(an)) {
            isSame = false
            break
          }
        }
      }
      if (isSame) return // no changes in selection
      this.$emit('update:attrs', this.allNames.filter(an => sel.has(an)))
    },

    // number of selected attributes in the group or entity
    groupSelected (node) {
      let n = 0
      for (const an of node.names) {
        if (this.selectedSet.has(an)) n++
      }
      return n
    },

    onInfo (node) {
      if (node.isEntity) {
        this.$emit('entity-info-show', node.entityName)
      } else if (node.isGroup) {
        this.$emit('entity-group-info-show', node.name, node.entityName)
      } else {
        this.$emit('entity-attr-info-show', node.label, node.entityName)
      }
    },

    onToggleExpand () {
      if (this.isExpanded) {
        this.$refs.tree.collapseAll()
        this.isExpanded = false
      } else {
        this.expandAll()
      }
    },
    expandAll () {
      if (this.$refs.tree) this.$refs.tree.expandAll()
      this.isExpanded = true
    }
  },

  mounted () {
    this.doRefresh()
  }
}
</script>

<style scoped>
  .est-node {
    min-height: 1.75rem;
  }
</style>
