// pivot table record processing: readers

// output table read expression rows: one row for each expression
export const exprOutReader = (exprDimName, rank, dimProp) => {
  const scaleRd = scaleReader('ExprId') // scale values of measure: max, min, sum count avg
  let prevDef = scaleRd.rangeDef() // current scale range info

  return {
    isScale: true,
    scaleReader: scaleRd,

    rowReader: (src) => {
      // no data to read: if source rows are empty or invalid return undefined reader
      if (!src || (src?.length || 0) <= 0) return void 0

      const srcLen = src.length
      let nSrc = 0

      const rd = { // reader to return
        readRow: () => {
          if (nSrc >= srcLen) {
            scaleRd.finish()
            if (prevDef.scaleId >= 0 && prevDef.count > 1 && prevDef.calc !== NONE_SCALE) {
              scaleRd.setRange(prevDef.scaleId, prevDef.calc, prevDef.count)
            }
            return void 0
          }
          if (nSrc === 0) {
            prevDef = scaleRd.rangeDef()
            scaleRd.reset()
          }
          scaleRd.nextRow(src[nSrc])
          return src[nSrc++] // expression row: one row for each expression
        },
        readDim: {},
        readValue: (r) => (!r.IsNull ? r.Value : void 0) // expression value
      }

      // read dimension item value: enum id, expression id
      for (let n = 0; n < rank; n++) {
        rd.readDim[dimProp[n].name] = (r) => (r.DimIds.length > n ? r.DimIds[n] : void 0)
      }

      // read measure dimension: expression id
      rd.readDim[exprDimName] = (r) => (r.ExprId)

      return rd
    }
  }
}

// output table read calculated measure rows: one row for each calculation
export const calcOutBaseReader = (isCmp, runDimName, calcDimName, rank, dimProp) => {
  const scaleRd = scaleReader('CalcId') // scale values of measure: max, min, sum count avg
  let prevDef = scaleRd.rangeDef() // current scale range info

  return {
    scRd: scaleRd,

    rowRd: (src) => {
      // no data to read: if source rows are empty or invalid return undefined reader
      if (!src || (src?.length || 0) <= 0) return void 0

      const srcLen = src.length
      let nSrc = 0

      const rd = { // reader to return
        readRow: () => {
          if (nSrc >= srcLen) {
            scaleRd.finish()
            if (prevDef.scaleId >= 0 && prevDef.count > 1 && prevDef.calc !== NONE_SCALE) {
              scaleRd.setRange(prevDef.scaleId, prevDef.calc, prevDef.count)
            }
            return void 0
          }
          if (nSrc === 0) {
            prevDef = scaleRd.rangeDef()
            scaleRd.reset()
          }
          scaleRd.nextRow(src[nSrc])
          return src[nSrc++] // calculation row: one row for each calculation or comparison
        },
        readDim: {},
        readValue: (r) => (!r.IsNull ? r.Value : void 0) // expression value
      }

      // read dimension item value: enum id, expression id
      for (let n = 0; n < rank; n++) {
        rd.readDim[dimProp[n].name] = (r) => (r.DimIds.length > n ? r.DimIds[n] : void 0)
      }

      // read calculation dimension: calculation id
      rd.readDim[calcDimName] = (r) => (r.CalcId)

      if (isCmp) {
        rd.readDim[runDimName] = (r) => (r.RunId) // read run dimension: run id
      }

      return rd
    }
  }
}

// output table read calculated measure rows: one row for each calculation
export const calcOutReader = (calcDimName, rank, dimProp) => {
  const baseRd = calcOutBaseReader(false, '', calcDimName, rank, dimProp)
  return {
    rowReader: baseRd.rowRd,
    isScale: true,
    scaleReader: baseRd.scRd
  }
}

// output table read run compare rows: one row for each comparison calculation
// row is the same as calculated row and we need to process RunId
export const cmpOutReader = (runDimName, cmpDimName, rank, dimProp) => {
  const baseRd = calcOutBaseReader(true, runDimName, cmpDimName, rank, dimProp)
  return {
    rowReader: baseRd.rowRd,
    isScale: true,
    scaleReader: baseRd.scRd
  }
}

// output table read native accumulators rows: one row for each accumilator
export const accOutReader = (accDimName, subIdDim, rank, dimProp) => {
  return {
    rowReader: (src) => {
      // no data to read: if source rows are empty or invalid return undefined reader
      if (!src || (src?.length || 0) <= 0) return void 0

      const srcLen = src.length
      let nSrc = 0

      const rd = { // reader to return
        readRow: () => {
          return (nSrc < srcLen) ? src[nSrc++] : void 0 // accumilator row: one row for each accumilator
        },
        readDim: {},
        readValue: (r) => (!r.IsNull ? r.Value : void 0) // accumilator value
      }

      // read dimension item value: enum id, sub-value id, accumulator id
      for (let n = 0; n < rank; n++) {
        rd.readDim[dimProp[n].name] = (r) => (r.DimIds.length > n ? r.DimIds[n] : void 0)
      }

      // read measure dimensions: accumulator id and sub-value id
      rd.readDim[accDimName] = (r) => (r.AccId)
      rd.readDim[subIdDim] = (r) => (r.SubId)

      return rd
    },

    isScale: false
  }
}

// output table read all accumulators rows: all accumulators are in one row, including derived accumulators
export const allAccOutReader = (allAccDimName, subIdDim, allAccCount, rank, dimProp) => {
  return {
    rowReader: (src) => {
      // no data to read: if source rows are empty or invalid return undefined reader
      if (!src || (src?.length || 0) <= 0) return void 0

      // accumulator id's: measure dimension items, all accumulators dimension is at [rank + 2]
      const accIds = []
      for (const e of dimProp[rank + 2].enums) {
        accIds.push(e.value)
      }

      const srcLen = src.length
      let nSrc = 0
      let nAcc = -1 // after first read row must be nAcc = 0

      const rd = { // reader to return
        readRow: () => {
          nAcc++
          if (nAcc >= allAccCount) {
            nAcc = 0
            nSrc++
          }
          return (nSrc < srcLen) ? src[nSrc] : void 0 // accumilator row: all accumulators in one row
        },
        readDim: {},
        readValue: (r) => {
          const v = !r.IsNull[nAcc] ? r.Value[nAcc] : void 0 // accumilator value
          return v
        }
      }

      // read dimension item value: enum id, sub-value id, accumulator id
      for (let n = 0; n < rank; n++) {
        rd.readDim[dimProp[n].name] = (r) => (r.DimIds.length > n ? r.DimIds[n] : void 0)
      }

      // read measure dimensions: accumulator id and sub-value id
      rd.readDim[allAccDimName] = (r) => (accIds[nAcc])
      rd.readDim[subIdDim] = (r) => (r.SubId)

      return rd
    },

    isScale: false
  }
}

/* eslint-disable no-multi-spaces */

// scale calculation:
//   distance from base: zero, min, max, average or middle = (max+min)/2
//   distance scaled into 10 ranges
export const NONE_SCALE = 0     // undefined scale calculation
export const ZERO_SCALE = 1     // value              range: min > 0 ? max : max > 0 ? (max - min) : min
export const AVG_SCALE = 2      // value - avg        range: max - min
export const MIDDLE_SCALE = 3   // value - middle     range: max - min

export const MIX_RANGE = 0
export const HOT_RANGE = 1
export const COLD_RANGE = 2

/* eslint-enable no-multi-spaces */

// empty scale value: min, max, sum, count, avg value for expression enum id
const emptyScale = () => {
  return {
    enumId: -1,
    max: void 0,
    min: void 0,
    sum: 0,
    count: 0,
    avg: void 0,
    color: MIX_RANGE,
    range: [] // ranges lower bound, excluding first range, if value < first bound then it is in the first range, index 0 range
  }
}

// empty range definition: no scale expression id, calculation is none, zero ranges count
export const emptyRangeDef = () => {
  return {
    scaleId: -1,
    count: 0,
    calc: NONE_SCALE,
    color: MIX_RANGE
  }
}

// find min, max, sum, count, avg value for measure dimesion expressions
const scaleReader = (scaleDimName) => {
  let scaleLst = {}

  let rangeScaleId = -1
  let rangeCount = 0
  let rangeCalc = NONE_SCALE
  let rangeColor = MIX_RANGE

  // add new expresson scale values
  const addScale = (eId) => {
    if (typeof eId !== typeof 1) return
    const v = emptyScale()
    v.enumId = eId
    scaleLst[eId] = v
    return scaleLst[eId]
  }

  // do final adjustment after last row: calculate avg
  const finishOne = (eId) => {
    const v = scaleLst?.[eId]
    if (!v) return
    if (isFinite(v.sum) && isFinite(v.count) && v.count > 0) v.avg = v.sum / v.count

    // check if values are positive, negative or mix
    v.color = MIX_RANGE
    if (v.count > 0 && (isFinite(v.min) || isFinite(v.max))) {
      let low = isFinite(v.min) ? v.min : 0
      let high = isFinite(v.max) ? v.max : 0
      if (high < low) [high, low] = [low, high]

      if (low >= 0 && high > 0) v.color = HOT_RANGE
      if (low < 0 && high <= 0) v.olor = COLD_RANGE
    }
  }

  const isRangeDef = () => (rangeScaleId >= 0 && rangeCount > 1 && rangeCalc !== NONE_SCALE) // if true then range is defined

  // clear existing range info
  const clearRangeDef = () => {
    rangeScaleId = -1
    rangeCount = 0
    rangeCalc = NONE_SCALE
    rangeColor = MIX_RANGE
  }

  return {
    // reset scale values
    reset: () => {
      rangeScaleId = -1
      scaleLst = {}
    },

    // return measure scale values: max, min, sum, count, avg
    getScale: () => Object.assign({}, scaleLst),

    // process next data row
    nextRow: (row) => {
      if (!row) return
      if (row.IsNull || !isFinite(row.Value)) return // skip: it not a scale expression or value is not a finite number

      let v = scaleLst?.[row[scaleDimName]]
      if (!v) v = addScale(row[scaleDimName])
      if (!v) return

      if (typeof v.max !== typeof 1 || v.max < row.Value) {
        v.max = row.Value
      }
      if (typeof v.min !== typeof 1 || v.min > row.Value) {
        v.min = row.Value
      }
      v.count++
      v.sum += row.Value
    },

    // do final adjustment after last row: calculate avg
    finish: () => {
      for (const eId in scaleLst) {
        finishOne(eId)
      }
    },

    isRange: () => isRangeDef(), // if true then range is defined
    clearRange: () => clearRangeDef(), // clear existing range info

    // get range info: expression id, range count nad calculation
    rangeDef: () => {
      const r = {
        scaleId: rangeScaleId,
        count: rangeCount,
        calc: rangeCalc,
        color: rangeColor
      }
      return r
    },

    // calculate [0, nRange] ranges for measure values.
    // range defined by lower bound and first range excluded
    // if value < first bound then it is in the first range, index 0 range
    // for example: if [min, max] = [20, 30] and range count = 2 then bounds are: [25]
    setRange: (eId, calc, nRange) => {
      clearRangeDef() // clear existing range

      if (!calc || calc === NONE_SCALE) return
      if (typeof nRange !== typeof 1 || nRange <= 1) return // all values belong to default range, no range boundaries defined

      const v = scaleLst?.[eId]
      if (!v) {
        return
      }
      v.range = [] // clear existing range

      if (!v.count) return // empty source measure values: no data

      // distance from the middle of source array
      if (calc === MIDDLE_SCALE) {
        rangeColor = MIX_RANGE

        const mdl = (v.max + v.min) / 2
        const step = (v.max - v.min) / nRange

        for (let k = 0; k < nRange - 1; k++) {
          const b = v.min + (k + 1) * step
          v.range.push(b)
        }
        if (!v.range.length) v.range.push(mdl)
      }

      // distance from the average value
      if (calc === AVG_SCALE) {
        rangeColor = MIX_RANGE

        const dh = v.max - v.avg
        const dl = v.avg - v.min
        if (!isFinite(v.avg) || (!isFinite(dh) && !isFinite(dl))) return // no average value or no max and no min values

        const d = (dh > dl) ? dh : dl // range size = 2 * distance from average
        const step = 2 * d / nRange
        const low = v.avg - d
        // const high = v.avg + d

        for (let k = 0; k < nRange - 1; k++) {
          const b = low + (k + 1) * step
          v.range.push(b)
        }
        if (!v.range.length) v.range.push(v.avg)
      }

      // distance from value to zero
      if (calc === ZERO_SCALE) {
        rangeColor = MIX_RANGE

        if (!isFinite(v.max) && !isFinite(v.min)) return // no no max and no min values

        let low = isFinite(v.min) ? v.min : 0
        let high = isFinite(v.max) ? v.max : 0
        if (high < low) [high, low] = [low, high]

        const d = Math.abs(high) > Math.abs(low) ? Math.abs(high) : Math.abs(low)

        low = low < 0 ? -d : 0
        high = high > 0 ? d : 0

        const step = (high - low) / nRange

        for (let k = 0; k < nRange - 1; k++) {
          const b = low + (k + 1) * step
          v.range.push(b)
        }
        if (!v.range.length) {
          v.range.push(low)
          if (high > low) v.range.push(high)
        }

        if (low >= 0 && high > 0) rangeColor = HOT_RANGE
        if (low < 0 && high <= 0) rangeColor = COLD_RANGE
      }

      // if range not empty then keep range definition
      if (v.range.length > 0) {
        rangeScaleId = eId
        rangeCount = nRange
        rangeCalc = calc
      }
    },

    // scale measure value and return range index
    // if range not found or no ranges defined then return -1: any value belongs to default range
    findRange: (val) => {
      if (typeof val !== typeof 1 || !isRangeDef()) return -1 // value undefined or range not set

      const v = scaleLst?.[rangeScaleId]
      if (!v) return -1 // scale not defined for that measure

      // if no source data or only one range defined then any value belong to undefined range: default range
      if (!v.count || v.count <= 1 || v.range.length <= 0) {
        return -1
      }
      // at least two ranges defined: range.length > 0

      // find index of range where left <= val < right round
      const rLen = v.range.length
      if (val < v.range[0]) return 0
      if (val >= v.range[rLen - 1]) return rLen

      let left = 1
      let right = rLen - 1
      let n = 0

      while (left <= right) {
        n = Math.floor((left + right) / 2)

        if (n === 0 && val < v.range[n]) return 0 // value < first range upper bound
        if (n === rLen - 1 && v.range[n] <= val) return rLen // last range lower bound <= value
        if (v.range[n] <= val && val < v.range[n + 1]) return n + 1

        if (val > v.range[n]) {
          left = n + 1
        } else {
          right = n - 1
        }
      }
      return n
    }
  }
}
