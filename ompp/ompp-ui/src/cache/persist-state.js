// Persist UI state in browser indexedDB and restore it on page load.
//
// Persisted state:
//   ui-state store: language, selected run and workset for each model, open tabs, active tab,
//                   parameter, output table and microdata views, tree expanded state, tab page inputs, scroll positions
//   model store:    model list, current model metadata, run list, workset list, model words and languages
//
// After restore model store lists are displayed immediately and refreshed from the server in background:
// see isRestoredStale().

import { toRaw, watch } from 'vue'
import * as Mdf from 'src/model-common'
import { useUiStateStore } from 'src/stores/ui-state'
import { useModelStore } from 'src/stores/model'
import * as Cdb from './cache-db'
import { getCacheSettings } from './cache-settings'

const STATE_VERSION = 1 // increase if format of persisted state changed
const UI_KEY = 'ui-state'
const MODEL_KEY = 'model-current'
const LISTS_KEY = 'model-lists'
const SAVE_DELAY_MS = 500

// ui-state store properties to persist
const UI_FIELDS = [
  'uiLang',
  'runDigestSelected',
  'worksetNameSelected',
  'taskNameSelected',
  'noAccDownload',
  'noMicrodataDownload',
  'idCsvDownload',
  'treeLabelKind',
  'uiTreeExpandedKeys',
  'paramViews',
  'tableViews',
  'mdViews',
  'modelView',
  'tabStates',
  'scrollPos'
]

const staleSet = new Set() // restored lists which must be refreshed from the server: 'run-list:digest', 'set-list:digest'
let isStarted = false
let uiTimer = 0
let modelTimer = 0
let lastModelUpdated = -1
let lastModelDigest = ''

// deep copy of reactive value into plain object
const plain = (v) => {
  if (v === undefined) return undefined
  return JSON.parse(JSON.stringify(toRaw(v)))
}

// return true once if restored list (run list or workset list) of the model must be refreshed from the server
export const isRestoredStale = (kind, digest) => {
  const k = kind + ':' + (digest || '')
  if (!staleSet.has(k)) return false
  staleSet.delete(k)
  return true
}

// restore stores from indexedDB, must be called before app mounted
export const restoreState = async (pinia) => {
  if (!getCacheSettings().isStateCache) return

  const uiStore = useUiStateStore(pinia)
  const ms = useModelStore(pinia)

  try {
    const [ui, cur, lists] = await Promise.all([
      Cdb.getValue(Cdb.STATE_STORE, UI_KEY),
      Cdb.getValue(Cdb.STATE_STORE, MODEL_KEY),
      Cdb.getValue(Cdb.STATE_STORE, LISTS_KEY)
    ])

    // restore ui state
    if (ui?.v === STATE_VERSION && ui?.state && typeof ui.state === 'object') {
      uiStore.$patch((s) => {
        for (const f of UI_FIELDS) {
          if (!Object.prototype.hasOwnProperty.call(ui.state, f)) continue
          const v = ui.state[f]
          if (v === undefined || v === null) continue
          if (typeof v !== typeof s[f] && s[f] !== undefined) continue // type mismatch: ignore
          s[f] = v
        }
      })
    }

    // restore model list, current model and model run and workset lists
    if (lists?.v === STATE_VERSION && Mdf.isModelList(lists?.modelList)) {
      ms.dispatchModelList(lists.modelList)
      staleSet.add('model-list:')
    }
    if (cur?.v === STATE_VERSION && Mdf.isModel(cur?.theModel)) {
      const dgst = Mdf.modelDigest(cur.theModel)
      const isInList = !Mdf.isLength(ms.modelList) || ms.modelList.findIndex(m => Mdf.modelDigest(m) === dgst) >= 0

      if (dgst && isInList && (cur?.uiLang || '') === (uiStore.uiLang || '')) {
        ms.dispatchTheModel(cur.theModel)
        lastModelDigest = dgst
        lastModelUpdated = ms.theModelUpdated

        if (lists?.v === STATE_VERSION) {
          if (Mdf.isRunTextList(lists?.runTextList) && lists.runTextList.length > 0 && lists.runTextList.every(r => r?.ModelDigest === dgst)) {
            ms.runTextList = lists.runTextList
            ms.runTextListUpdated++
            staleSet.add('run-list:' + dgst)
          }
          if (Mdf.isWorksetTextList(lists?.worksetTextList) && lists.worksetTextList.length > 0 && lists.worksetTextList.every(w => w?.ModelDigest === dgst)) {
            ms.worksetTextList = lists.worksetTextList
            ms.worksetTextListUpdated++
            staleSet.add('set-list:' + dgst)
          }
          if (lists?.wordList?.ModelDigest === dgst) ms.dispatchWordList(lists.wordList)
          if (Mdf.isLangList(lists?.langList)) ms.dispatchLangList(lists.langList)
        }
        // restore selected run and workset of current model
        uiStore.dispatchViewSelectedRestore(dgst)
      }
    }
  } catch (e) {
    console.warn('UI cache: unable to restore UI state', e)
  }
}

// save ui-state store
const saveUiState = async () => {
  const uiStore = useUiStateStore()
  const st = {}
  for (const f of UI_FIELDS) st[f] = plain(uiStore[f])
  await Cdb.putValue(Cdb.STATE_STORE, UI_KEY, { v: STATE_VERSION, state: st })
}

// save model store: current model only if it is changed, lists always
const saveModelState = async () => {
  const ms = useModelStore()
  const uiStore = useUiStateStore()
  const dgst = Mdf.modelDigest(ms.theModel)

  if (dgst !== lastModelDigest || ms.theModelUpdated !== lastModelUpdated) {
    lastModelDigest = dgst
    lastModelUpdated = ms.theModelUpdated
    if (dgst) {
      // model metadata is frozen (not reactive) plain object and can be stored as is
      await Cdb.putValue(Cdb.STATE_STORE, MODEL_KEY, { v: STATE_VERSION, uiLang: uiStore.uiLang || '', theModel: toRaw(ms.theModel) })
    } else {
      await Cdb.deleteValue(Cdb.STATE_STORE, MODEL_KEY)
    }
  }

  await Cdb.putValue(Cdb.STATE_STORE, LISTS_KEY, {
    v: STATE_VERSION,
    modelList: plain(ms.modelList),
    runTextList: plain(ms.runTextList),
    worksetTextList: plain(ms.worksetTextList),
    wordList: plain(ms.wordList),
    langList: plain(ms.langList)
  })
}

const scheduleUiSave = () => {
  clearTimeout(uiTimer)
  uiTimer = setTimeout(() => {
    uiTimer = 0
    saveUiState().catch((e) => console.warn('UI cache: unable to save UI state', e))
  }, SAVE_DELAY_MS)
}
const scheduleModelSave = () => {
  clearTimeout(modelTimer)
  modelTimer = setTimeout(() => {
    modelTimer = 0
    saveModelState().catch((e) => console.warn('UI cache: unable to save model state', e))
  }, SAVE_DELAY_MS * 2)
}

// save pending changes immediately, for example before page unload
export const flushState = () => {
  if (!isStarted || !getCacheSettings().isStateCache) return

  // always save ui state: store subscription may not be called yet for the most recent changes
  clearTimeout(uiTimer)
  uiTimer = 0
  saveUiState().catch(() => {})

  if (modelTimer) {
    clearTimeout(modelTimer)
    modelTimer = 0
    saveModelState().catch(() => {})
  }
}

// start saving store changes into indexedDB
export const startStateSave = (pinia) => {
  if (isStarted) return
  isStarted = true

  const uiStore = useUiStateStore(pinia)
  const ms = useModelStore(pinia)

  uiStore.$subscribe(() => {
    if (getCacheSettings().isStateCache) scheduleUiSave()
  }, { detached: true })

  // model store: watch update counters instead of deep subscription, model metadata can be very large
  watch(
    () => [ms.modelListUpdated, ms.theModelUpdated, ms.runTextListUpdated, ms.worksetTextListUpdated, ms.wordList, ms.langList],
    () => {
      if (getCacheSettings().isStateCache) scheduleModelSave()
    }
  )

  try {
    window.addEventListener('pagehide', flushState)
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'hidden') flushState()
    })
  } catch (e) {}
}

// delete persisted UI state
export const clearPersistedState = async () => {
  clearTimeout(uiTimer)
  clearTimeout(modelTimer)
  uiTimer = 0
  modelTimer = 0
  lastModelDigest = ''
  lastModelUpdated = -1
  await Cdb.clearStore(Cdb.STATE_STORE)
}
