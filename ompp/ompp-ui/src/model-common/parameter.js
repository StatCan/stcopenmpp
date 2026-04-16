// db structures common functions: parameter and parameter list

import * as Mdl from './model'
import * as Tdf from './type'

// number of model parameters
export const paramCount = (md) => {
  if (!Mdl.isModel(md)) return 0
  return md?.ParamTxt?.length || 0
}

// is model has parameter text list and each element is Param
export const isParamTextList = (md) => {
  if (!Mdl.isModel(md)) return false
  if (!Array.isArray(md?.ParamTxt) || (md?.ParamTxt?.length || 0) === 0) return false
  for (let k = 0; k < md.ParamTxt.length; k++) {
    if (!isParam(md.ParamTxt[k].Param)) return false
  }
  return true
}

// return true if this is non empty Param
export const isParam = (p) => {
  if (!p) return false
  if (!p.hasOwnProperty('ParamId') || !p.hasOwnProperty('Name') || !p.hasOwnProperty('Digest')) return false

  return p.ParamId >= 0 && (p?.Name || '') !== '' && (p?.Digest || '') !== ''
}

// if this is not empty ParamTxt: parameter id, parameter name, parameter digest
export const isNotEmptyParamText = (pt) => {
  if (!pt) return false
  if (!pt?.Param || !pt.hasOwnProperty('DescrNote')) return false
  return isParam(pt.Param)
}

// return empty ParamTxt
export const emptyParamText = () => {
  return {
    Param: {
      ParamId: 0,
      Name: '',
      Digest: '',
      ImportDigest: ''
    },
    DescrNote: {
      LangCode: '',
      Descr: '',
      Note: ''
    }
  }
}

// find ParamTxt by parameter id
export const paramTextById = (md, nId) => {
  if (!Mdl.isModel(md) || nId < 0) { // model empty or parameter id invalid: return empty result
    return emptyParamText()
  }
  for (let k = 0; k < md.ParamTxt.length; k++) {
    if (!isParam(md.ParamTxt[k].Param)) continue
    if (md.ParamTxt[k].Param.ParamId === nId) return md.ParamTxt[k]
  }
  return emptyParamText() // not found
}

// find ParamTxt by name
export const paramTextByName = (md, name) => {
  if (!Mdl.isModel(md) || (name || '') === '') { // model empty or name empty: return empty result
    return emptyParamText()
  }
  for (let k = 0; k < md.ParamTxt.length; k++) {
    if (!isParam(md.ParamTxt[k].Param)) continue
    if (md.ParamTxt[k].Param.Name === name) return md.ParamTxt[k]
  }
  return emptyParamText() // not found
}

// return empty parameter size
export const emptyParamSize = () => {
  return {
    rank: 0,
    dimTotal: 0,
    dimSize: []
  }
}

// find parameter size by name: number of parameter rows
export const paramSizeByName = (md, name) => {
  // if this is not a parameter then size =empty value else rowCount at least =1
  let ret = emptyParamSize()
  const p = paramTextByName(md, name)
  if (!isParam(p.Param)) return ret

  // parameter rank must be same as dimension count
  if (p.hasOwnProperty('ParamDimsTxt')) {
    if ((p.Param.Rank || 0) !== (p.ParamDimsTxt.length || 0)) return ret
    //
    ret.rank = (p.Param.Rank || 0)
  }

  // multiply all dimension sizes, assume =1 for built-in types
  ret.dimTotal = 1
  for (let k = 0; k < p.ParamDimsTxt.length; k++) {
    if (!p.ParamDimsTxt[k].hasOwnProperty('Dim')) {
      ret = emptyParamSize()
      return ret
    }
    const n = Tdf.typeEnumSizeById(md, p.ParamDimsTxt[k].Dim.TypeId)
    ret.dimSize.push(n || 0)
    if (ret.dimSize[k] > 0) ret.dimTotal *= ret.dimSize[k]
  }
  return ret
}

// return true if this is non empty ParamRunSet with Txt[]
export const isNotEmptyParamRunSet = (prs) => {
  if (!prs) return false
  return (prs?.Name || '') !== '' &&
    (prs?.SubCount || -1) >= 0 && Array.isArray(prs?.Txt) &&
    prs.hasOwnProperty('ValueDigest') && typeof prs.ValueDigest === typeof 'string'
}

// return empty ParamRunSetPub, which is Param[i] of run text or workset text
export const emptyParamRunSet = () => {
  return {
    Name: '',
    SubCount: 0,
    DefaultSubId: 0, // exist only in workset
    ValueDigest: '', // not empty only for run parameters
    Txt: []
  }
}

// find ParamRunSet by parameter name in Param[] array of run text or workset text
// return empty value if not found
export const paramRunSetByName = (runOrSet, name) => {
  if (!runOrSet || !name) return emptyParamRunSet()
  if (!Array.isArray(runOrSet?.Param) || (runOrSet?.Param?.length || 0) === 0) return emptyParamRunSet()
  for (let k = 0; k < runOrSet.Param.length; k++) {
    if (isNotEmptyParamRunSet(runOrSet.Param[k]) && runOrSet.Param[k].Name === name) {
      return runOrSet.Param[k]
    }
  }
  return emptyParamRunSet() // not found
}
