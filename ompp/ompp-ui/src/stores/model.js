// current model and list of the models
import { defineStore, acceptHMRUpdate } from 'pinia'

import * as Mdf from '../model-common'
import { useUiStateStore } from './ui-state'

// assign new value to current model
const commitModel = (state, model) => {
  const digest = Mdf.modelDigest(model)
  const storeDigest = Mdf.modelDigest(state.theModel)

  if (Mdf.isModel(model) || Mdf.isEmptyModel(model)) {
    state.theModel = Object.freeze(model)
    state.theModelDir = Mdf.modelDirByDigest(digest, state.modelList)
    if (digest !== storeDigest) {
      state.groupParameterLeafs = Object.freeze(Mdf.groupLeafs(model, true))
      state.groupTableLeafs = Object.freeze(Mdf.groupLeafs(model, false))
      state.groupEntityLeafs = Object.freeze(Mdf.entityGroupLeafs(model))
      state.topEntityAttrs = Object.freeze(Mdf.entityTopAttrs(model))
    }
    state.theModelUpdated++
  }
}

export const useModelStore = defineStore('model', {
  state: () => ({
    modelList: [],
    modelListUpdated: 0,
    theModel: Mdf.emptyModel(),
    theModelDir: '',
    theModelUpdated: 0,
    groupParameterLeafs: {},
    groupTableLeafs: {},
    groupEntityLeafs: {},
    topEntityAttrs: {},
    runTextList: [],
    runTextListUpdated: 0,
    worksetTextList: [],
    worksetTextListUpdated: 0,
    wordList: Mdf.emptyWordList(),
    langList: Mdf.emptyLangList()
  }),

  getters: {
    // return model count or zero if model list empty or invalid
    modelCount (state) {
      return Mdf.isModelList(state.modelList) ? Mdf.lengthOf(state.modelList) : 0
    },

    // return current model language code and name, example: { LangCode: 'EN', Name: 'English' }
    modelLanguage (state) {
      let code = ''
      if (!!state.wordList && state.wordList.hasOwnProperty('ModelLangCode')) {
        code = state.wordList.ModelLangCode || ''
      }
      if (!code && !!state.theModel && state.theModel.hasOwnProperty('Model') && state.theModel.Model.hasOwnProperty('DefaultLangCode')) {
        code = state.theModel.Model.DefaultLangCode || ''
      }
      return {
        LangCode: code,
        Name: Mdf.langNameByCode(state.langList, code)
      }
    }
  },

  actions: {
    //
    // getters
    //

    // find model by digest in model list
    modelByDigest (digest) {
      if (!digest || typeof digest !== typeof 'string') return Mdf.emptyModel()
      for (const m of this.modelList) {
        if (m.Model.Digest === digest) return m
      }
      return Mdf.emptyModel()
    },

    // return true if run list contain the run: find by model digest and run digest
    isExistInRunTextList (rt) {
      if (!rt || !rt.hasOwnProperty('ModelDigest') || !rt.hasOwnProperty('RunDigest')) return false
      if (!Mdf.isLength(this.runTextList)) return false
      return this.runTextList.findIndex((r) => rt.ModelDigest === r.ModelDigest && rt.RunDigest === r.RunDigest) >= 0
    },

    // find run by model digest and run digest, return empty run if not found
    runTextByDigest (rt) {
      if (!rt || !rt.hasOwnProperty('ModelDigest') || !rt.hasOwnProperty('RunDigest')) return Mdf.emptyRunText()
      if (!Mdf.isLength(this.runTextList)) return Mdf.emptyRunText()
      return this.runTextList.find((r) => rt.ModelDigest === r.ModelDigest && rt.RunDigest === r.RunDigest) || Mdf.emptyRunText()
    },

    // return true if workset list contain the workset: find by model digest and workset name
    isExistInWorksetTextList (wt) {
      if (!wt || !wt.hasOwnProperty('ModelDigest') || !wt.hasOwnProperty('Name')) return false
      if (!Mdf.isLength(this.worksetTextList)) return false
      return this.worksetTextList.findIndex((w) => wt.ModelDigest === w.ModelDigest && wt.Name === w.Name) >= 0
    },

    // find workset by model digest and workset name, return empty workset if not found
    worksetTextByName (wt) {
      if (!wt || !wt.hasOwnProperty('ModelDigest') || !wt.hasOwnProperty('Name')) return Mdf.emptyWorksetText()
      if (!Mdf.isLength(this.worksetTextList)) return Mdf.emptyWorksetText()
      return this.worksetTextList.find((w) => wt.ModelDigest === w.ModelDigest && wt.Name === w.Name) || Mdf.emptyWorksetText()
    },

    //
    // actions
    //

    // assign new value to model language-specific strings (model words)
    dispatchWordList (mw) {
      this.wordList = Mdf.emptyWordList()
      if (!mw) return
      this.wordList.ModelName = (mw.ModelName || '')
      this.wordList.ModelDigest = (mw.ModelDigest || '')
      this.wordList.LangCode = (mw.LangCode || '')
      this.wordList.ModelLangCode = (mw.ModelLangCode || '')
      if (mw.hasOwnProperty('LangWords')) {
        if (mw.LangWords && (mw.LangWords.length || 0) > 0) this.wordList.LangWords = mw.LangWords
      }
      if (mw.hasOwnProperty('ModelWords')) {
        if (mw.ModelWords && (mw.ModelWords.length || 0) > 0) this.wordList.ModelWords = mw.ModelWords
      }
      if (mw.hasOwnProperty('Msg')) {
        if (mw.Msg && (mw.Msg.length || 0) > 0) this.wordList.Msg = mw.Msg
      }
    },

    // assign new value to list of model languages
    dispatchLangList (ml) {
      this.langList = Mdf.isLangList(ml) ? ml : Mdf.emptyLangList()
    },

    // set new value to current model, clear run list and workset list
    dispatchTheModel (model) {
      const digest = Mdf.modelDigest(model)
      const storeDigest = Mdf.modelDigest(this.theModel)

      commitModel(this, model)

      // clear model words list if new model digest not same as word list model digest
      if ((digest || '') !== this.wordList.ModelDigest) this.wordList = Mdf.emptyWordList()

      // clear list of model languages if model changed or if it is empty
      if ((digest === '' || (digest !== storeDigest))) this.langList = Mdf.emptyLangList()

      // clear run list if model digest not same as run list model digest
      let dg = ''
      if (this.runTextList.length > 0) {
        if (Mdf.isRunText(this.runTextList[0])) dg = this.runTextList[0].ModelDigest || ''
      }
      if ((digest || '') !== dg) {
        this.runTextList = []
        this.runTextListUpdated++
      }

      // clear workset list if model digest not same as workset list model digest
      dg = ''
      if (this.worksetTextList.length > 0) {
        if (Mdf.isWorksetText(this.worksetTextList[0])) dg = this.worksetTextList[0].ModelDigest || ''
      }
      if ((digest || '') !== dg) {
        this.worksetTextList = []
        this.worksetTextListUpdated++
      }

      // clear parameters view state if new model selected
      if (digest !== storeDigest) {
        // keep view state on model switch
        // dispatch('uiState/viewDeleteByModel', storeDigest, { root: true })
      }
      // clear selected run if new model selected
      const uiStateStore = useUiStateStore()

      if (digest !== storeDigest || this.runTextList[0]?.ModelDigest !== digest) {
        uiStateStore.dispatchRunDigestSelected('')
      }
      // clear selected workset if new model selected
      if (digest !== storeDigest || this.worksetTextList[0]?.ModelDigest !== digest) {
        uiStateStore.dispatchWorksetNameSelected('')
      }
    },

    // set new value to model list and clear current model
    dispatchModelList (ml) {
      // assign new value to model list, if (ml) is a model list
      if (Mdf.isModelList(ml)) {
        let isKeep = false
        const storeDigest = Mdf.modelDigest(this.theModel)

        for (const m of ml) {
          // check if current model exist in the new model list
          if (!isKeep) isKeep = storeDigest === Mdf.modelDigest(m)

          // parse model extra properties
          let me = {}
          try {
            const ms = m?.Extra || ''
            if (ms && typeof ms === typeof 'string') me = JSON.parse(ms)
          } catch {
            me = {}
          }
          m.Extra = me
        }
        // if current model does not exist in model list then clear current model
        if (!isKeep) {
          commitModel(this, Mdf.emptyModel())
        }
        this.modelList = ml
        this.modelListUpdated++
      }
    },

    // update run text
    dispatchRunText (rt) {
      if (!Mdf.isRunText(rt)) return
      if (!Mdf.isNotEmptyRunText(rt)) return

      const k = this.runTextList.findIndex((r) => rt.ModelDigest === r.ModelDigest && rt.RunDigest === r.RunDigest)
      if (k >= 0) {
        this.runTextList[k] = rt
        this.runTextListUpdated++
      } else {
        if (this.theModel.Model.Digest === rt.ModelDigest) {
          this.runTextList.unshift(rt) // run text stored in reverse chronological order
          this.runTextListUpdated++
        }
      }
    },

    // update run status and update data-time, also update selected run
    dispatchRunTextStatusUpdate (rp) {
      if (!rp || !Mdf.isLength(this.runTextList)) return
      if (!rp.hasOwnProperty('ModelDigest') || !rp.hasOwnProperty('RunDigest')) return

      const k = this.runTextList.findIndex((r) => r.ModelDigest === rp.ModelDigest && r.RunDigest === rp.RunDigest)
      if (k < 0) return

      if (rp.hasOwnProperty('Status')) {
        if ((rp.Status || '') !== '') {
          this.runTextList[k].Status = rp.Status
          this.runTextListUpdated++
        }
      }
      if (rp.hasOwnProperty('UpdateDateTime')) {
        if ((rp.UpdateDateTime || '') !== '') {
          this.runTextList[k].UpdateDateTime = rp.UpdateDateTime
          this.runTextListUpdated++
        }
      }
    },

    // set new value to run list
    dispatchRunTextList (rtl) {
      // clear selected run if new model selected
      let isKeep = false
      if (Mdf.isLength(rtl) && Mdf.isLength(this.runTextList)) {
        let dgNew = ''
        if (Mdf.isNotEmptyRunText(rtl[0])) dgNew = (rtl[0].ModelDigest || '')
        let dgOld = ''
        if (Mdf.isNotEmptyRunText(this.runTextList[0])) dgOld = (this.runTextList[0].ModelDigest || '')
        isKeep = dgNew !== '' && dgNew === dgOld
      }
      if (!isKeep) {
        const uiStateStore = useUiStateStore()
        uiStateStore.dispatchRunDigestSelected('')
      }

      // assign new value to run text list: if (rtl) is a run text list the assign reverse it and commit to store
      if (!Mdf.isRunTextList(rtl)) return // reject invalid run text list

      rtl.reverse() // sort in reverse chronological order

      // if parameter or table or entity list was not empty then copy it into new run text list
      for (const rt of rtl) {
        const k = this.runTextList.findIndex((r) => rt.ModelDigest === r.ModelDigest && rt.RunDigest === r.RunDigest)
        if (k >= 0) {
          if (Mdf.lengthOf(rt.Param) <= 0 && Mdf.lengthOf(this.runTextList[k].Param) > 0) rt.Param = Mdf.deepClone(this.runTextList[k].Param)
          if (Mdf.lengthOf(rt.Table) <= 0 && Mdf.lengthOf(this.runTextList[k].Table) > 0) rt.Table = Mdf.deepClone(this.runTextList[k].Table)
          if (Mdf.lengthOf(rt.Entity) <= 0 && Mdf.lengthOf(this.runTextList[k].Entity) > 0) rt.Entity = Mdf.deepClone(this.runTextList[k].Entity)
        }
      }
      this.runTextList = rtl
      this.runTextListUpdated++
    },

    // update workset text
    dispatchWorksetText (wt) {
      if (!Mdf.isWorksetText(wt)) return
      if (!Mdf.isNotEmptyWorksetText(wt) || !Mdf.isLength(this.worksetTextList)) return

      const k = this.worksetTextList.findIndex((w) => w.ModelDigest === wt.ModelDigest && w.Name === wt.Name)
      if (k >= 0) {
        this.worksetTextList[k] = Mdf.deepClone(wt)
        this.worksetTextListUpdated++
      } else {
        if (this.theModel.Model.Digest === wt.ModelDigest) {
          this.worksetTextList.push(Mdf.deepClone(wt))
          this.worksetTextListUpdated++
        }
      }
    },

    // set current workset status: set readonly status and update date-time
    dispatchWorksetStatus (ws) {
      if (!Mdf.isWorksetStatus(ws) || !Mdf.isLength(this.worksetTextList)) return

      const k = this.worksetTextList.findIndex((w) => w.ModelDigest === ws.ModelDigest && w.Name === ws.Name)
      if (k >= 0) {
        this.worksetTextList[k].IsReadonly = !!ws.IsReadonly
        this.worksetTextList[k].UpdateDateTime = (ws.UpdateDateTime || '')
        this.worksetTextListUpdated++
      }
    },

    // set new value to workset list
    dispatchWorksetTextList (wtl) {
      // clear selected workset if new model selected
      let isKeep = false
      if (Mdf.isLength(wtl) && Mdf.isLength(this.worksetTextList)) {
        let dgNew = ''
        if (Mdf.isNotEmptyWorksetText(wtl[0])) dgNew = (wtl[0].ModelDigest || '')
        let dgOld = ''
        if (Mdf.isNotEmptyWorksetText(this.worksetTextList[0])) dgOld = (this.worksetTextList[0].ModelDigest || '')
        isKeep = dgNew !== '' && dgNew === dgOld
      }
      if (!isKeep) {
        const uiStateStore = useUiStateStore()
        uiStateStore.dispatchWorksetNameSelected('')
      }

      // update workset list
      if (!Mdf.isWorksetTextList(wtl)) return // reject invalid workset text list

      // if parameter text was not empty then copy it into new workset text list
      for (const wt of wtl) {
        const k = this.worksetTextList.findIndex((w) => w.ModelDigest === wt.ModelDigest && w.Name === wt.Name)
        if (k >= 0) {
          if (Mdf.lengthOf(wt.Param) <= 0 && Mdf.lengthOf(this.worksetTextList[k].Param) > 0) wt.Param = Mdf.deepClone(this.worksetTextList[k].Param)
        }
      }
      this.worksetTextList = wtl
      this.worksetTextListUpdated++
    }
  }
})


if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useModelStore, import.meta.hot))
}
