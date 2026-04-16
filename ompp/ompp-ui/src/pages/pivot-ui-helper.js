// pivot table UI helper functions

import * as Pcvt from 'components/pivot-cvt'

/* eslint-disable no-multi-spaces */

export const SUB_ID_DIM = 'SubId' // sub-value id dminesion name

// kind of output table view
export const tkind = {
  EXPR: 0,  // output table expression(s)
  ACC: 1,   // output table accumulator(s)
  ALL: 2,   // output table all-accumultors view
  CALC: 3,  // output table calculated measure view
  CMP: 4    // output table run comparison
}
// kind of entity microdata view
export const ekind = {
  MICRO: 0, // entity microdata
  CALC: 1,  // aggregated (calculated) entity measure attributes
  CMP: 2    // entity run comparison
}

/* eslint-enable no-multi-spaces */

// make filter state: selection in other dimensions
export const makeFilterState = (dims) => {
  const fs = {}

  for (const d of dims) {
    fs[d.name] = []
    for (const e of d.selection) {
      fs[d.name].push(e.value)
    }
  }
  return fs
}

// compare filter state and return true if it is the same
// do not filter by skipDims dimensions, skipDims can be single name of [array of names]
export const equalFilterState = (fs, dims, skipDims) => {
  // check if all previous filter dimensions in the new list
  for (const p in fs) {
    // skip dimension filter: do not filter by sub-value id or by measure dimension
    if (skipDims) {
      if (typeof skipDims === typeof 'string' && p === skipDims) continue
      if (Array.isArray(skipDims) && skipDims.indexOf(p) >= 0) continue
    }

    let isFound = false
    for (let k = 0; !isFound && k < dims.length; k++) {
      isFound = dims[k].name === p
    }
    if (!isFound) return false
  }

  // check if all new dimensions in previous list and have same items selected
  for (const d of dims) {
    // skip dimension filter: do not filter by sub-value id or by measure dimension
    if (skipDims) {
      if (typeof skipDims === typeof 'string' && d.name === skipDims) continue
      if (Array.isArray(skipDims) && skipDims.indexOf(d.name) >= 0) continue
    }

    if (!fs.hasOwnProperty(d.name)) return false
    if (fs[d.name].length !== d.selection.length) return false
    for (const e of d.selection) {
      if (!fs[d.name].includes(e.value)) return false
    }
  }
  return true // identical filters
}

// return page layout to read parameter data
// filter by other dimension(s) selected values
// do not filter by skipDims dimensions, skipDims can be single name of [array of names]
export const makeSelectLayout = (name, otherFields, skipDims, valueFilter) => {
  const layout = {
    Name: name,
    Offset: 0,
    Size: 0,
    FilterById: [],
    Filter: []
  }

  // make filters for other dimensions to include selected value
  for (const f of otherFields) {
    // skip dimension filter: do not filter by sub-value id or by measure dimension
    if (skipDims) {
      if (typeof skipDims === typeof 'string' && f.name === skipDims) continue
      if (Array.isArray(skipDims) && skipDims.indexOf(f.name) >= 0) continue
    }

    const flt = {
      Name: f.name,
      Op: 'IN_AUTO',
      EnumIds: []
    }
    for (const e of f.selection) {
      flt.EnumIds.push(e.value)
    }
    layout.FilterById.push(flt)
  }

  // add filters by value
  if (!valueFilter || !Array.isArray(valueFilter)) return layout // exit: no filters by value

  for (const vf of valueFilter) {
    if (!vf || (vf?.name || '') === '' || (vf?.op || '') === '' || !Array.isArray(vf?.value)) continue
    if (vf.value.length <= 0) continue

    const flt = {
      Name: vf.name,
      Op: vf.op,
      Values: []
    }
    for (const v of vf.value) {
      const s = (typeof v !== typeof 'string') ? v.toString() : v
      if ((s || '') !== '') flt.Values.push(v)
    }
    layout.Filter.push(flt)
  }
  return layout
}

// prepare page of parameter data for save
// all dimension items are packed into cell key ordered by dimension name
export const makePageForSave = (dimProp, keyPos, rank, subIdName, defaultSubId, isNullable, updated) => {
  // sub-id value is zero by default
  // if parameter has multiple sub-values
  // then sub-id also can be a single value from filter or packed into cell key
  const subIdField = {
    isConst: true,
    srcPos: 0,
    value: defaultSubId || 0
  }

  // rows and columns: cell key contain items ordered by dimension names
  // for each dimension find source and destination position
  const keyDims = []
  for (let k = 0; k < keyPos.length; k++) {
    // sub-id dimension
    if (keyPos[k].name === subIdName) {
      subIdField.isConst = false
      subIdField.srcPos = keyPos[k].pos
      continue
    }
    // regular dimensions
    for (let j = 0; j < dimProp.length; j++) {
      if (keyPos[k].name === dimProp[j].name) {
        keyDims.push({
          name: keyPos[k].name,
          srcPos: keyPos[k].pos,
          dstPos: j
        })
        break
      }
    }
  }

  // for each updated cell find dimension items from cell key or from filter value
  const pv = []
  for (const bkey in updated) {
    // get dimension items from filter values and split cell key into dimension items
    const items = Pcvt.keyToItems(bkey)
    const di = Array(rank)

    for (const dk of keyDims) {
      di[dk.dstPos] = parseInt(items[dk.srcPos], 10)
    }

    // get sub-value id from cell key, from filter value or use default zero value
    // get cell value, for enum-based parameters expected to be enum id
    const nSub = !subIdField.isConst ? parseInt(items[subIdField.srcPos], 10) : subIdField.value
    const v = updated[bkey]

    pv.push({
      DimIds: di,
      IsNull: isNullable && ((v || '') === ''),
      Value: v,
      SubId: nSub
    })
  }
  return pv
}

// filter handler: update options list on user input
export const makeFilter = (f) => (val, update, abort) => {
  update(
    () => {
      if (!val) {
        f.options = f.enums
      } else {
        const vlc = val.toLowerCase()
        f.options = f.enums.filter(v => v.label.toLowerCase().indexOf(vlc) >= 0)
      }
    }
  )
}

/* json body response to read page POST must be:
{
  Page: [
    ....
  ],
  Layout: {
    Offset: 0,
    Size: 1,
    IsLastPage: true,
    IsFullPage: false
  }
}
*/
// check if response has Page and Layout
export const isPageLayoutRsp = (rsp) => {
  if (!rsp) return false
  if (!rsp.hasOwnProperty('Page') || !Array.isArray(rsp.Page)) return false
  if (!rsp.hasOwnProperty('Layout')) return false
  if (!rsp.Layout.hasOwnProperty('Offset') || typeof rsp.Layout.Offset !== typeof 1) return false
  if (!rsp.Layout.hasOwnProperty('Size') || typeof rsp.Layout.Size !== typeof 1) return false
  if (!rsp.Layout.hasOwnProperty('IsLastPage') || typeof rsp.Layout.IsLastPage !== typeof true) return false
  if (!rsp.Layout.hasOwnProperty('IsFullPage') || typeof rsp.Layout.IsFullPage !== typeof true) return false

  return true
}

// get error message if page reader falied and json resoponse is incorrect
export const errorFromPageLayoutRsp = (rsp) => {
  if (!rsp || typeof rsp !== typeof 'string') return ''

  const p = '{"Page":['
  return (rsp.startsWith(p)) ? rsp.substring(p.length, p.length + 255) : ''
}
