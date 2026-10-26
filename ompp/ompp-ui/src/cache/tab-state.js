// Cache local state of tab page: filters, form inputs, expanded sections.
//
// Tab pages (run list, input scenarios list, new run, downloads) are re-created each time user switches tabs,
// so all local data are lost. This mixin saves selected data properties in ui-state store
// (which is persisted in indexedDB) and restores them when tab page is mounted again or after page reload.
//
// Usage in component:
//
//   mixins: [tabStateMixin],
//   tabState: {
//     fields: ['runFilter', 'isParamTreeShow'], // data properties to cache
//     key () { return this.$route.path },      // optional: cache key, default is route path
//     auto: true,                              // optional: restore automatically after mounted, default true
//     restore (saved) { ... },                 // optional: custom restore, (saved) is object with cached fields
//     afterRestore () { ... }                  // optional: called after fields restored
//   }
//
// If auto is false then component must call this.tabStateRestore() when it is ready.

import { toRaw } from 'vue'
import { useUiStateStore } from 'src/stores/ui-state'
import { getCacheSettings } from './cache-settings'
import { flushState } from './persist-state'

const SAVE_DELAY_MS = 300

// return deep copy of data property value
const copyOf = (v) => {
  if (v === undefined) return undefined
  try {
    return JSON.parse(JSON.stringify(toRaw(v)))
  } catch (e) {
    return undefined
  }
}

export const tabStateMixin = {
  data () {
    return {
      tabStateIsRestored: false
    }
  },

  methods: {
    // return cache key of this tab page
    tabStateKey () {
      const opts = this.$options.tabState
      if (typeof opts?.key === 'function') return opts.key.call(this) || ''
      return this.$route?.path || ''
    },

    // return snapshot of cached fields
    tabStateSnapshot () {
      const opts = this.$options.tabState
      const st = {}
      for (const f of (opts?.fields || [])) {
        const v = copyOf(this[f])
        if (v !== undefined) st[f] = v
      }
      return st
    },

    // save tab state into store
    tabStateSave () {
      if (!this.tabStateIsRestored || !this._tabStateKey) return
      if (!getCacheSettings().isStateCache) return

      clearTimeout(this._tabStateTimer)
      this._tabStateTimer = 0
      try {
        const st = this.tabStateSnapshot()
        if (typeof this.$options.tabState?.extra === 'function') st.extra = this.$options.tabState.extra.call(this)
        useUiStateStore().dispatchTabState(this._tabStateKey, st)
      } catch (e) {
        console.warn('Unable to save tab state:', this._tabStateKey, e) // cache must never break the page
      }
    },

    // delete cached state of this tab page
    tabStateClear () {
      if (this._tabStateKey) useUiStateStore().dispatchTabStateDelete(this._tabStateKey)
    },

    // restore cached fields from store and start watching for changes
    tabStateRestore () {
      const opts = this.$options.tabState
      if (!opts || this.$.isUnmounted) return

      this._tabStateKey = this.tabStateKey()

      if (getCacheSettings().isStateCache && this._tabStateKey) {
        const saved = useUiStateStore().tabState(this._tabStateKey)

        if (saved && typeof saved === 'object') {
          try {
            if (typeof opts.restore === 'function') {
              opts.restore.call(this, saved)
            } else {
              for (const f of (opts.fields || [])) {
                if (Object.prototype.hasOwnProperty.call(saved, f) && typeof saved[f] === typeof this[f]) this[f] = saved[f]
              }
            }
            if (typeof opts.afterRestore === 'function') opts.afterRestore.call(this, saved)
          } catch (e) {
            console.warn('Unable to restore tab state:', this._tabStateKey, e)
          }
        }
      }
      this.tabStateIsRestored = true

      // watch cached fields and save on any change
      if (!this._tabStateUnwatch) {
        this._tabStateUnwatch = this.$watch(
          () => this.tabStateSnapshot(),
          () => {
            clearTimeout(this._tabStateTimer)
            this._tabStateTimer = setTimeout(() => this.tabStateSave(), SAVE_DELAY_MS)
          },
          { deep: true }
        )
        // same component re-used for another route, for example another model: save state and restore for new route
        this._tabStateKeyUnwatch = this.$watch(
          () => this.tabStateKey(),
          (newKey, oldKey) => {
            if (newKey === oldKey || this.$.isUnmounted) return
            this.tabStateSave()
            this.tabStateIsRestored = false
            this.$nextTick(() => this.tabStateRestore())
          }
        )
      }
    }
  },

  mounted () {
    if (!this.$options.tabState) return

    // on page reload or close save tab state immediately
    this._tabStateOnHide = () => {
      if (!this.tabStateIsRestored) return
      this.tabStateSave()
      flushState()
    }
    window.addEventListener('pagehide', this._tabStateOnHide)

    // restore after component own mounted() hook is completed
    if (this.$options.tabState.auto !== false) {
      this.$nextTick(() => this.tabStateRestore())
    }
  },

  beforeUnmount () {
    if (!this.$options.tabState) return
    window.removeEventListener('pagehide', this._tabStateOnHide)
    this.tabStateSave()
    if (this._tabStateUnwatch) this._tabStateUnwatch()
    if (this._tabStateKeyUnwatch) this._tabStateKeyUnwatch()
    this._tabStateUnwatch = null
    this._tabStateKeyUnwatch = null
    clearTimeout(this._tabStateTimer)
  }
}
