// pivot table record processing: value conversion and aggregation

import * as PcvtHlp from './pivot-cvt-helper'

// default process value: return source value as is
export const asIsPval = {
  init: () => (
    { result: void 0 }
  ),
  doNext: (val, state) => {
    state.result = val !== null ? val : void 0
    return state.result
  }
}

// as Float process value
export const asFloatPval = {
  init: () => ({ result: void 0 }),

  doNext: (val, state) => {
    const v = parseFloat(val)
    if (!isNaN(v)) state.result = v
    return state.result
  }
}

// as Int process value
export const asIntPval = {
  init: () => ({ result: void 0 }),

  doNext: (val, state) => {
    const v = parseInt(val, 10)
    if (!isNaN(v)) state.result = v
    return state.result
  }
}

// as Bool process value
export const asBoolPval = {
  init: () => ({ result: void 0 }),

  doNext: (val, state) => {
    const v = PcvtHlp.parseBool(val)
    if (typeof v === typeof true) state.result = v
    return state.result
  }
}

// sum process value aggregation
export const sumPval = {
  init: () => ({ result: void 0 }),

  doNext: (val, state) => {
    const v = parseFloat(val)
    if (!isNaN(v)) state.result = (state.result || 0) + v
    return state.result
  }
}

// average process value aggregation
export const avgPval = {
  init: () => ({
    result: void 0,
    sum: void 0,
    count: void 0
  }),

  doNext: (val, state) => {
    const v = parseFloat(val)
    if (!isNaN(v)) {
      state.sum = (state.sum || 0) + v
      state.count = (state.count || 0) + 1
      state.result = state.sum / state.count
    }
    return state.result
  }
}
