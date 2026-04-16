<!-- Tree of input pamateres or output tables -->
<!--
Tree nodes (tree items) are single selection and readonly.
It can be "folder" node with children or "leaf" node (no children).
If you want to have "folder" without children then set item.isGroup = true.
Each node must have v-for key and at least 'label' to display
It is also can have 'descr' text to display as second line (description).
If isAbout is true then show about icon button node.
If isAboutEmpty is true then about icon button is disabled and non-clickable.
About click event: @click('om-table-tree-about', item.label).

Expected array of tree items as:
{
  key:           'item-key',
  label:         'item name to display',
  descr:         'item description',
  link:          'link to open as _blank target'
  children:      [array of child items],  // folder children
  parts:         '',                      // any payload
  isGroup:       false,                   // if true then item is a folder even there are no children
  isAbout:       false,                   // if true then show about button to item
  isAboutEmpty:  false,                   // if true then disable about button
  isClick:       false,                   // if true then node is click-able
  isDownload:    false                    // if true then node is download-able
}
-->
<template>
<div class="text-body1">

  <div
    class="row no-wrap items-center full-width"
    >
    <q-btn
      v-if="isAnyGroup"
      flat
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs om-tree-control-button"
      :icon="isTreeExpand ? 'keyboard_arrow_up' : 'keyboard_arrow_down'"
      :title="isTreeExpand ? $t('Collapse all') : $t('Expand all')"
      @click="doToogleExpandTree"
      />
    <q-btn
      v-if="isAnyHidden"
      @click="$emit('om-table-tree-show-hidden', !isShowHidden)"
      dense
      :unelevated="!isShowHidden"
      :outline="isShowHidden"
      color="primary"
      class="col-auto q-mr-xs om-tree-control-button"
      :icon="!isShowHidden ? 'mdi-eye-off-outline' : 'mdi-eye-outline'"
      :title="!isShowHidden ? $t('Do not show hidden items') : $t('Show all hidden items')"
      />
    <q-btn
      v-if="isOnOffNotInList && isAnyInList"
      @click="$emit('om-table-tree-show-not-in-list', !isShowNotInList)"
      dense
      :unelevated="!isShowNotInList"
      :outline="isShowNotInList"
      color="primary"
      class="col-auto q-mr-xs om-tree-control-button"
      :icon="inListIcon || 'mdi-filter-outline'"
      :title="isShowNotInList ? $t(inListOffLabel || 'Show filtered out items') : $t(inListOnLabel || 'Do not show filtered out items')"
      />
    <q-btn
      v-if="isInListClear"
      :disable="!isAnyInList"
      @click="$emit('om-table-tree-clear-in-list')"
      dense
      class="col-auto bg-primary text-white rounded-borders q-mr-xs om-tree-control-button"
      :icon="inListClearIcon || 'mdi-close-circle'"
      :title="$t(inListClearLabel || 'Remove items filter')"
      />
    <span class="col-grow">
      <q-input
        ref="filterInput"
        debounce="500"
        v-model="treeFilter"
        outlined
        dense
        :placeholder="filterPlaceholder"
        >
        <template v-slot:append>
          <q-icon v-if="treeFilter !== ''" name="cancel" class="cursor-pointer" @click="resetFilter" />
          <q-icon v-else name="search" />
        </template>
      </q-input>
    </span>
  </div>

  <div class="q-px-sm">
    <q-tree
      ref="tableTree"
      :nodes="treeData"
      node-key="key"
      v-model:expanded="expandedKeys"
      :filter="treeFilter"
      :filter-method="doTreeFilter"
      no-transition
      :no-results-label="noResultsLabel"
      :no-nodes-label="noNodesLabel"
      >
      <template v-slot:default-header="prop">

        <div
          v-if="prop.node.isGroup || (prop.node.children && prop.node.children.length)"
          :class="{'om-tree-found-node': treeWalk.keysFound[prop.node.key]}"
          class="row no-wrap items-center full-width"
          >
          <q-btn
            v-if="prop.node.isAbout"
            @click.stop="$emit('om-table-tree-group-note', prop.node.label, prop.node.parts)"
            flat
            round
            dense
            :disable="prop.node.isAboutEmpty"
            color="primary"
            padding="xs"
            class="col-auto"
            icon="mdi-information-outline"
            :title="$t('About') + ' ' + prop.node.label"
            />
          <q-btn
            v-if="!!prop.node.link"
            @click.stop="onLink(prop.node.link, prop.node.label)"
            flat
            round
            dense
            color="primary"
            padding="xs"
            class="col-auto"
            icon="open_in_new"
            :title="$t('Open') + ' ' + prop.node.label"
            />
          <q-btn
            v-if="isAddGroup"
            @click.stop="$emit('om-table-tree-group-add', prop.node.label, prop.node.parts)"
            :disable="isAddDisabled"
            flat
            round
            dense
            :color="!isAddDisabled ? 'primary' : 'secondary'"
            padding="xs"
            class="col-auto"
            icon="mdi-plus-circle-outline"
            :title="$t('Add') + ' ' + prop.node.label"
            />
          <q-btn
            v-if="isRemoveGroup"
            @click.stop="$emit('om-table-tree-group-remove', prop.node.label, prop.node.parts)"
            :disable="isRemoveDisabled"
            flat
            round
            dense
            :color="!isRemoveDisabled ? 'primary' : 'secondary'"
            padding="xs"
            class="col-auto"
            icon="mdi-minus-circle-outline"
            :title="$t('Remove') + ' ' + prop.node.label"
            />
          <q-btn
            v-if="isDownloadGroup || !!prop.node?.isDownload"
            @click.stop="$emit('om-table-tree-group-download', prop.node.label, prop.node.parts)"
            :disable="isDownloadDisabled"
            flat
            round
            dense
            :color="!isDownloadDisabled ? 'primary' : 'secondary'"
            padding="xs"
            class="col-auto"
            icon="mdi-download-circle-outline"
            :title="$t('Download') + ' ' + prop.node.label"
            />
          <div
            v-if="isGroupClick || !!prop.node?.isClick"
            @click.stop="$emit('om-table-tree-group-select', prop.node.label, prop.node.parts)"
            class="label-click col q-ml-xs q-pl-xs"
            >
            <template v-if="labelKind === 'name-only'">{{ prop.node.label }}</template>
            <template v-if="labelKind === 'descr-only'">{{ prop.node.descr }}</template>
            <template v-if="labelKind !== 'name-only' && labelKind !== 'descr-only'">
              <span>{{ prop.node.label }}<br />
              <span class="om-text-descr">{{ prop.node.descr }}</span></span>
            </template>
          </div>
          <div
            v-else
            class="col q-ml-xs q-pl-xs"
            >
            <template v-if="labelKind === 'name-only'">{{ prop.node.label }}</template>
            <template v-if="labelKind === 'descr-only'">{{ prop.node.descr }}</template>
            <template v-if="labelKind !== 'name-only' && labelKind !== 'descr-only'">
              <span>{{ prop.node.label }}<br />
              <span class="om-text-descr">{{ prop.node.descr }}</span></span>
            </template>
          </div>
        </div>

        <div v-else
          :class="{'om-tree-found-node': treeWalk.keysFound[prop.node.key]}"
          class="row no-wrap items-center full-width om-tree-leaf"
          >
          <q-btn
            v-if="prop.node.isAbout"
            @click.stop="$emit('om-table-tree-leaf-note', prop.node.label, prop.node.parts)"
            flat
            round
            dense
            :disable="prop.node.isAboutEmpty"
            color="primary"
            padding="xs"
            class="col-auto"
            icon="mdi-information-outline"
            :title="$t('About') + ' ' + prop.node.label"
            />
          <q-btn
            v-if="!!prop.node.link"
            @click.stop="onLink(prop.node.link, prop.node.label)"
            flat
            round
            dense
            color="primary"
            padding="xs"
            class="col-auto"
            icon="open_in_new"
            :title="$t('Open') + ' ' + prop.node.label"
            />
          <q-btn
            v-if="isAdd"
            @click.stop="$emit('om-table-tree-leaf-add', prop.node.label, prop.node.parts)"
            :disable="isAddDisabled"
            flat
            round
            dense
            :color="!isAddDisabled ? 'primary' : 'secondary'"
            padding="xs"
            class="col-auto"
            icon="mdi-plus-circle-outline"
            :title="$t('Add') + ' ' + prop.node.label"
            />
          <q-btn
            v-if="isRemove"
            @click.stop="$emit('om-table-tree-leaf-remove', prop.node.label, prop.node.parts)"
            :disable="isRemoveDisabled"
            flat
            round
            dense
            :color="!isRemoveDisabled ? 'primary' : 'secondary'"
            padding="xs"
            class="col-auto"
            icon="mdi-minus-circle-outline"
            :title="$t('Remove') + ' ' + prop.node.label"
            />
          <q-btn
            v-if="(isDownload || !!prop.node?.isDownload)"
            @click.stop="$emit('om-table-tree-leaf-download', prop.node.label, prop.node.parts)"
            :disable="isDownloadDisabled"
            flat
            round
            dense
            :color="!isDownloadDisabled ? 'primary' : 'secondary'"
            padding="xs"
            class="col-auto"
            icon="mdi-download-circle-outline"
            :title="$t('Download') + ' ' + prop.node.label"
            />
          <div
            @click="$emit('om-table-tree-leaf-select', prop.node.label, prop.node.parts)"
            :class="{'label-click': (isLeafClick || !!prop.node?.isClick)}"
            class="col q-ml-xs q-pl-xs"
            >
            <template v-if="labelKind === 'name-only'">{{ prop.node.label }}</template>
            <template v-if="labelKind === 'descr-only'">{{ prop.node.descr }}</template>
            <template v-if="labelKind !== 'name-only' && labelKind !== 'descr-only'">
              <span>{{ prop.node.label }}<br />
              <span class="om-text-descr">{{ prop.node.descr }}</span></span>
            </template>
          </div>
        </div>

      </template>
    </q-tree>
  </div>

</div>
</template>

<style lang="scss" scoped>
  .label-click {
    &:hover {
      background: rgba(0, 0, 0, 0.05);
      cursor: pointer;
    }
  }
</style>

<script>
import { openURL } from 'quasar'

export default {
  name: 'OmTableTree',

  props: {
    refreshTickle: { type: Boolean, default: false },
    refreshTreeTickle: { type: Boolean, default: false },
    treeData: { type: Array, default: () => [] },
    labelKind: { type: String, default: '' },
    isExpand: { type: Boolean, default: false },
    isAnyGroup: { type: Boolean, default: false },
    isAnyHidden: { type: Boolean, default: false },
    isShowHidden: { type: Boolean, default: false },
    isGroupClick: { type: Boolean, default: false },
    isLeafClick: { type: Boolean, default: false },
    isAdd: { type: Boolean, default: false },
    isAddGroup: { type: Boolean, default: false },
    isAddDisabled: { type: Boolean, default: false },
    isRemove: { type: Boolean, default: false },
    isRemoveGroup: { type: Boolean, default: false },
    isRemoveDisabled: { type: Boolean, default: false },
    isDownload: { type: Boolean, default: false },
    isDownloadGroup: { type: Boolean, default: false },
    isDownloadDisabled: { type: Boolean, default: false },
    filterPlaceholder: { type: String, default: '' },
    noResultsLabel: { type: String, default: '' },
    noNodesLabel: { type: String, default: '' },
    isAnyInList: { type: Boolean, default: false },
    isOnOffNotInList: { type: Boolean, default: false },
    isShowNotInList: { type: Boolean, default: false },
    inListOnLabel: { type: String, default: '' },
    inListOffLabel: { type: String, default: '' },
    inListIcon: { type: String, default: '' },
    isInListClear: { type: Boolean, default: false },
    inListClearLabel: { type: String, default: '' },
    inListClearIcon: { type: String, default: '' },
    treeStore: {
      type: Object,
      default: () => ({
        expandPush: void 0, // store action to dispatch tree expanded keys
        expandPull: void 0 // store getter for tree expanded keys
      })
    }
  },

  data () {
    return {
      treeFilter: '',
      isTreeExpand: this.isExpand,
      expandedKeys: [],
      treeWalk: {
        isAnyFound: false,
        keysFound: {} // if node match filter then map keysFound[node.key] = true
      }
    }
  },

  watch: {
    refreshTreeTickle () {
      this.isTreeExpand = this.isExpand
      this.doRefresh()
    },
    refreshTickle () {
      this.isTreeExpand = this.isExpand
      this.doRefresh()
    },
    treeFilter () { this.updateTreeWalk() }
  },

  emits: [
    'om-table-tree-show-hidden',
    'om-table-tree-show-not-in-list',
    'om-table-tree-clear-in-list',
    'om-table-tree-group-note',
    'om-table-tree-group-add',
    'om-table-tree-group-remove',
    'om-table-tree-group-download',
    'om-table-tree-group-select',
    'om-table-tree-leaf-note',
    'om-table-tree-leaf-add',
    'om-table-tree-leaf-remove',
    'om-table-tree-leaf-download',
    'om-table-tree-leaf-select'
  ],

  methods: {
    onLink (link, name) { openURL(link) },

    // update view on tree data change
    doRefresh () {
      this.treeFilter = ''
      this.treeWalk.isAnyFound = false
      this.treeWalk.keysFound = {}

      // restore expanded state
      if (this?.treeStore?.expandPull) {
        this.expandedKeys = this.treeStore.expandPull()
        this.isTreeExpand = this.expandedKeys.length > 0
      }
    },

    // expand or collapse all tree nodes
    doToogleExpandTree () {
      if (this.isTreeExpand) {
        this.$refs.tableTree.collapseAll()
      } else {
        this.$refs.tableTree.expandAll()
      }
      this.isTreeExpand = !this.isTreeExpand
    },

    // filter tree nodes by name (label) or description
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

      // if any node match the filter then exapnd the tree
      if (this.treeWalk.isAnyFound && !this.isTreeExpand) {
        this.$nextTick(() => { this.doToogleExpandTree() })
      }
    },
    // clear tree filter value
    resetFilter () {
      this.treeFilter = ''
      this.treeWalk.isAnyFound = false
      this.treeWalk.keysFound = {}
      this.$refs.filterInput.focus()
    }
  },

  mounted () {
    this.doRefresh()
  },

  // save tree expanded state
  beforeUnmount () {
    if (this?.treeStore?.expandPush) {
      const expKeys = Array.isArray(this.expandedKeys) ? this.expandedKeys : []
      this.treeStore.expandPush(expKeys)
    }
  }
}
</script>
