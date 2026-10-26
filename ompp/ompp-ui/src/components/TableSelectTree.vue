<!-- select output tables: single tree with checkboxes, search, select all / clear and "selected only" filter -->
<template>
<div class="table-select-tree">

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
      ref="filterInput"
      v-model="filterText"
      debounce="250"
      outlined
      dense
      clearable
      class="col"
      :placeholder="$t('Find output table...')"
      >
      <template v-slot:prepend>
        <q-icon name="search" />
      </template>
    </q-input>
    <q-chip
      dense
      square
      :color="selectedCount ? 'primary' : 'negative'"
      text-color="white"
      class="col-auto"
      :title="$t('Selected output tables')"
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
    :no-results-label="$t('No output tables found')"
    :no-nodes-label="$t('No output tables found or server offline')"
    >
    <template v-slot:default-header="prop">
      <div class="row no-wrap items-center full-width tst-node">
        <div class="col">
          <span v-if="treeLabelKind !== 'descr-only' || !prop.node.descr" :class="{ 'text-weight-medium': prop.node.isGroup }">{{ prop.node.label }}</span>
          <span v-if="treeLabelKind !== 'name-only' && prop.node.descr" :class="treeLabelKind === 'descr-only' ? '' : 'q-pl-sm om-text-descr'">{{ prop.node.descr }}</span>
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
import { useUiStateStore } from '../stores/ui-state'
import * as Mdf from 'src/model-common'
import { tabStateMixin } from 'src/cache/tab-state'

export default {
  name: 'TableSelectTree',
  mixins: [tabStateMixin],

  // cache search text, filters and expanded groups
  tabState: {
    fields: ['filterText', 'isSelectedOnly', 'isShowHidden', 'expandedKeys'],
    key () { return 'table-select:' + Mdf.modelDigest(this.theModel) }
  },

  props: {
    tables: { type: Array, default: () => [] } // names of selected output tables
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
      leafs: [], // all leaf nodes of the tree: { key, name }, same table can be in multiple groups
      allNames: [], // names of all model output tables, including hidden
      hiddenNames: {} // names of hidden output tables
    }
  },

  computed: {
    selectedSet () { return new Set(this.tables || []) },
    selectedCount () { return this.allNames.reduce((n, tn) => n + (this.selectedSet.has(tn) ? 1 : 0), 0) },
    totalCount () { return this.allNames.length },
    isFiltering () { return (this.filterText || '') !== '' },
    // tree keys of selected tables: table may appear in multiple groups and all occurrences are ticked
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
    ...mapState(useUiStateStore, ['treeLabelKind'])
  },

  watch: {
    theModelUpdated () { this.doRefresh() },
    isShowHidden () { this.doRefresh() },
    filterText (t) {
      if (t) this.$nextTick(() => this.expandAll()) // show search results
    }
  },

  emits: ['update:tables', 'table-info-show', 'table-group-info-show'],

  methods: {
    // build tree of output table groups and tables
    doRefresh () {
      const m = this.theModel
      const gLst = (Array.isArray(m?.GroupTxt) ? m.GroupTxt : []).filter(g => g?.Group && !g.Group.IsParam)
      const tLst = Array.isArray(m?.TableTxt) ? m.TableTxt : []

      const gById = {}
      const childGroups = new Set()
      for (const g of gLst) {
        gById[g.Group.GroupId] = g
        for (const pc of (g.Group.GroupPc || [])) {
          if (pc.ChildGroupId >= 0) childGroups.add(pc.ChildGroupId)
        }
      }
      const tById = {}
      this.allNames = []
      this.hiddenNames = {}
      this.isAnyHidden = false
      for (const t of tLst) {
        tById[t.Table.TableId] = t
        this.allNames.push(t.Table.Name)
        if (t.Table.IsHidden) {
          this.hiddenNames[t.Table.Name] = true
          this.isAnyHidden = true
        }
      }

      let n = 0
      const leafs = []
      const inGroup = new Set()

      const makeLeaf = (t, ctx) => {
        const node = {
          key: 't' + (n++) + ':' + t.Table.Name,
          name: t.Table.Name,
          label: t.Table.Name,
          descr: t.TableDescr || '',
          isGroup: false,
          ctx: ctx, // text of parent groups, used for search: all tables of matched group are found
          text: (t.Table.Name + ' ' + (t.TableDescr || '')).toLowerCase()
        }
        leafs.push(node)
        return node
      }

      // make group node and all children, skip circular references, hidden and empty groups
      const makeGroup = (g, path, ctx) => {
        if (g.Group.IsHidden) this.isAnyHidden = true
        if (g.Group.IsHidden && !this.isShowHidden) return null

        const gDescr = g.DescrNote?.Descr || g.Descr || ''
        const gCtx = ctx + ' ' + (g.Group.Name + ' ' + gDescr).toLowerCase()
        const node = { key: 'g' + (n++) + ':' + g.Group.Name, name: g.Group.Name, label: g.Group.Name, descr: gDescr, isGroup: true, children: [], names: [] }

        for (const pc of (g.Group.GroupPc || [])) {
          if (pc.ChildGroupId >= 0 && gById[pc.ChildGroupId] && !path.includes(pc.ChildGroupId)) {
            const c = makeGroup(gById[pc.ChildGroupId], [...path, pc.ChildGroupId], gCtx)
            if (c) {
              node.children.push(c)
              for (const tn of c.names) {
                if (!node.names.includes(tn)) node.names.push(tn)
              }
            }
          }
          if (pc.ChildLeafId >= 0 && tById[pc.ChildLeafId]) {
            const t = tById[pc.ChildLeafId]
            inGroup.add(t.Table.TableId)
            if (t.Table.IsHidden && !this.isShowHidden) continue
            node.children.push(makeLeaf(t, gCtx))
            if (!node.names.includes(t.Table.Name)) node.names.push(t.Table.Name)
          }
        }
        return node.children.length > 0 ? node : null
      }

      const tree = []
      for (const g of gLst) {
        if (childGroups.has(g.Group.GroupId)) continue // not a top level group
        const node = makeGroup(g, [g.Group.GroupId], '')
        if (node) tree.push(node)
      }
      // tables which are not included in any group
      for (const t of tLst) {
        if (inGroup.has(t.Table.TableId)) continue
        if (t.Table.IsHidden && !this.isShowHidden) continue
        tree.push(makeLeaf(t, ''))
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
      const t = (this.filterText || '').toLowerCase()
      return !t || node.text.includes(t) || node.ctx.includes(t)
    },
    // return true if leaf node is visible under current search
    isLeafVisible (node) {
      const t = (this.filterText || '').toLowerCase()
      return !t || node.text.includes(t) || node.ctx.includes(t)
    },

    // tree ticked keys updated: apply added and removed keys as table names
    onTicked (keys) {
      const now = new Set(keys)
      const before = new Set(this.tickedKeys)
      const sel = new Set(this.tables || [])

      for (const lf of this.leafs) {
        if (now.has(lf.key) && !before.has(lf.key)) sel.add(lf.name)
      }
      for (const lf of this.leafs) {
        if (!now.has(lf.key) && before.has(lf.key)) sel.delete(lf.name)
      }
      this.emitTables(sel)
    },

    // select all tables or all found tables if search text is not empty
    onSelectAll () {
      const sel = new Set(this.tables || [])
      if (!this.isFiltering) {
        for (const tn of this.allNames) sel.add(tn)
      } else {
        for (const lf of this.leafs) {
          if (this.isLeafVisible(lf)) sel.add(lf.name)
        }
      }
      this.emitTables(sel)
    },
    // clear all tables or all found tables if search text is not empty
    onClearAll () {
      if (!this.isFiltering) {
        this.emitTables(new Set())
        return
      }
      const sel = new Set(this.tables || [])
      for (const lf of this.leafs) {
        if (this.isLeafVisible(lf)) sel.delete(lf.name)
      }
      this.emitTables(sel)
    },
    // emit selected table names in model order
    emitTables (sel) {
      const cur = this.selectedSet
      let isSame = sel.size === cur.size
      if (isSame) {
        for (const tn of sel) {
          if (!cur.has(tn)) {
            isSame = false
            break
          }
        }
      }
      if (isSame) return // no changes in selection
      this.$emit('update:tables', this.allNames.filter(tn => sel.has(tn)))
    },

    // number of selected tables in the group
    groupSelected (node) {
      let n = 0
      for (const tn of node.names) {
        if (this.selectedSet.has(tn)) n++
      }
      return n
    },

    onInfo (node) {
      if (node.isGroup) {
        this.$emit('table-group-info-show', node.name, '')
      } else {
        this.$emit('table-info-show', node.name, '')
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
  .tst-node {
    min-height: 1.75rem;
  }
</style>
