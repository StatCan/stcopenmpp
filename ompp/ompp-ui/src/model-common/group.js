// db structures common functions: parameters groups and output tables groups

import * as Mdl from './model'
import * as Prm from './parameter'
import * as Tbl from './output-table'

// is model has group text list and each element is Group
export const isGroupTextList = (md) => {
  if (!Mdl.isModel(md)) return false
  if (!md.hasOwnProperty('GroupTxt')) return false
  if (!Array.isArray(md.GroupTxt)) return false
  for (let k = 0; k < md.GroupTxt.length; k++) {
    if (!isGroupText(md.GroupTxt[k])) return false
  }
  return true
}

// return true if this is non empty Group
export const isGroup = (g) => {
  if (!g) return false
  if (!g.hasOwnProperty('GroupId') || !g.hasOwnProperty('Name') || !g.hasOwnProperty('IsParam') || !g.hasOwnProperty('IsHidden')) return false
  if (!g.hasOwnProperty('GroupPc') || !Array.isArray(g.GroupPc)) return false
  return (g.Name || '') !== ''
}

// return true if this is non empty Group
export const isGroupText = (gt) => {
  if (!gt) return false
  if (!gt.hasOwnProperty('Group') || !gt.hasOwnProperty('DescrNote')) return false
  return isGroup(gt.Group)
}

// return empty GroupTxt
export const emptyGroupText = () => {
  return {
    Group: {
      GroupId: -1,
      IsParam: false,
      Name: '',
      IsHidden: false,
      GroupPc: []
    },
    DescrNote: {
      LangCode: '',
      Descr: '',
      Note: ''
    }
  }
}

// find GroupTxt by group id
export const groupTextById = (md, nId) => {
  if (!Mdl.isModel(md) || nId < 0) { // model empty or parameter id invalid: return empty result
    return emptyGroupText()
  }
  for (let k = 0; k < md.GroupTxt.length; k++) {
    if (!isGroup(md.GroupTxt[k].Group)) continue
    if (md.GroupTxt[k].Group.GroupId === nId) return md.GroupTxt[k]
  }
  return emptyGroupText() // not found
}

// find GroupTxt by name
export const groupTextByName = (md, name) => {
  if (!Mdl.isModel(md) || (name || '') === '') { // model empty or name empty: return empty result
    return emptyGroupText()
  }
  for (let k = 0; k < md.GroupTxt.length; k++) {
    if (!isGroup(md.GroupTxt[k].Group)) continue
    if (md.GroupTxt[k].Group.Name === name) return md.GroupTxt[k]
  }
  return emptyGroupText() // not found
}

// return true if name is a group name
export const isGroupName = (md, name) => {
  if (!Mdl.isModel(md) || (name || '') === '') return false // model empty or name empty

  for (let k = 0; k < md.GroupTxt.length; k++) {
    if (!isGroup(md.GroupTxt[k].Group)) continue
    if (md.GroupTxt[k].Group.Name === name) return true
  }
  return false // not found
}

// return groups map to leafs:
//  {aGroupName: {
//    groupId: 123,
//    isHidden: true,
//    size: 2,
//    leafs: {leafName: true, otherLeafName: true}
//    },
//  nextGroupName: {....}
export const groupLeafs = (md, isParam) => {
  if (!Mdl.isModel(md)) { // model is empty: return empty result
    return {}
  }

  // initial list of groups: no leafs
  const gm = {}
  const gUse = {}
  const gLst = md.GroupTxt
  let nGrp = 0

  for (let k = 0; k < gLst.length; k++) {
    if (!isGroup(gLst[k].Group)) continue
    if ((isParam && !gLst[k].Group.IsParam) || (!isParam && gLst[k].Group.IsParam)) continue

    const gId = gLst[k].Group.GroupId
    gm[gLst[k].Group.Name] = {
      groupId: gId,
      isHidden: gLst[k].Group.IsHidden,
      size: 0,
      leafs: {}
    }
    gUse[gId] = {
      idx: k,
      out: [gId],
      in: [],
      leafs: []
    }
    nGrp++

    // add first leafs and child groups
    for (const pc of gLst[k].Group.GroupPc) {
      if (pc.ChildGroupId >= 0) gUse[gId].in.push(pc.ChildGroupId)
      if (pc.ChildLeafId >= 0) gUse[gId].leafs.push(pc.ChildLeafId)
    }
  }
  if (nGrp <= 0) return {} // no groups

  if ((isParam && !Prm.paramCount(md)) || (!isParam && !Tbl.tableCount(md))) return {} // no leafs

  // expand each group by replacing child groups by leafs
  for (const gId in gUse) {
    const g = gUse[gId]

    while (g.in.length > 0) {
      const childId = g.in.pop()
      if (g.out.indexOf(childId) >= 0) continue // skip: group already in child list of that parent

      g.out.push(childId) // done with that group

      const gChild = gUse[childId]
      if (!gChild) continue // skip: group does not exist

      // add all children leafs and all children groups into current group
      for (const pc of gLst[gChild.idx].Group.GroupPc) {
        if (pc.ChildGroupId >= 0) g.in.push(pc.ChildGroupId)
        if (pc.ChildLeafId >= 0) g.leafs.push(pc.ChildLeafId)
      }
    }
  }

  // for each leaf find leaf name (parameter name or table name) and include into group map
  for (const gName in gm) {
    const g = gm[gName]

    for (const leafId of gUse[g.groupId].leafs) {
      if (isParam) {
        const pt = Prm.paramTextById(md, leafId)
        if (pt.Param.Name) {
          g.leafs[pt.Param.Name] = true
          g.size++
        }
      } else {
        const tbt = Tbl.tableTextById(md, leafId)
        if (tbt.Table.Name) {
          g.leafs[tbt.Table.Name] = true
          g.size++
        }
      }
    }
  }

  return gm
}
