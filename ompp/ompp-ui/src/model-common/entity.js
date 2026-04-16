// db structures common functions: entity and entities list

import * as Mdl from './model'

// number of model entities
export const entityCount = (md) => {
  if (!Mdl.isModel(md)) return 0
  return md?.EntityTxt?.length || 0
}

// is model has entity text list and each element is Entity
export const isEntityTextList = (md) => {
  if (!Mdl.isModel(md)) return false
  if (!Array.isArray(md?.EntityTxt) || (md?.EntityTxt?.length || 0) === 0) return false
  for (const et of md.EntityTxt) {
    if (!isEntity(et.Entity)) return false
  }
  return true
}

// return true if this is non empty Entity
export const isEntity = (e) => {
  if (!e) return false
  if (!e.hasOwnProperty('EntityId') || !e.hasOwnProperty('Name') || !e.hasOwnProperty('Digest')) return false

  return e.EntityId >= 0 && (e?.Name || '') !== '' && (e?.Digest || '') !== ''
}

// if this is not empty EntityTxt: entity id, entity name, entity digest and not empty array of EntityAttrTxt
export const isNotEmptyEntityText = (et) => {
  if (!et) return false
  if (!et?.Entity || !et?.EntityAttrTxt) return false
  return isEntity(et.Entity) && Array.isArray(et.EntityAttrTxt) && et.EntityAttrTxt.length > 0
}

// return empty EntityTxt
export const emptyEntityText = () => {
  return {
    Entity: {
      EntityId: 0,
      Name: '',
      Digest: ''
    },
    EntityAttrTxt: [],
    DescrNote: {
      LangCode: '',
      Descr: '',
      Note: ''
    }
  }
}

// return true if this is non empty entity Attr
export const isEntityAttr = (ea) => {
  if (!ea) return false
  if (!ea.hasOwnProperty('EntityId') || !ea.hasOwnProperty('AttrId') ||
    !ea.hasOwnProperty('Name') || !ea.hasOwnProperty('TypeId') || !ea.hasOwnProperty('IsInternal')) return false

  if (typeof ea.EntityId !== typeof 1 || typeof ea.AttrId !== typeof 1 || typeof ea.TypeId !== typeof 1) return false
  if (typeof ea.IsInternal !== typeof true) return false

  return ea.EntityId >= 0 && ea.AttrId >= 0 && (ea?.Name || '') !== '' && ea.TypeId >= 0
}

// return true if this is non empty entity EntityAttrTxt
export const isNotEmptyEntityAttr = (eat) => {
  if (!eat) return false
  if (!eat?.Attr) return false
  return isEntityAttr(eat.Attr)
}

// return empty EntityAttrTxt
export const emptyEntityAttrTxt = () => {
  return {
    Attr: {
      EntityId: 0,
      AttrId: 0,
      Name: '',
      TypeId: 0,
      IsInternal: false
    },
    DescrNote: {
      LangCode: '',
      Descr: '',
      Note: ''
    }
  }
}

// find EntityTxt by entity id
export const entityTextById = (md, eId) => {
  if (!Mdl.isModel(md) || typeof eId !== typeof 1 || eId < 0) { // model empty or entity id invalid: return empty result
    return emptyEntityText()
  }
  for (let k = 0; k < md.EntityTxt.length; k++) {
    if (!isEntity(md.EntityTxt[k].Entity)) continue
    if (md.EntityTxt[k].Entity.EntityId === eId) return md.EntityTxt[k]
  }
  return emptyEntityText() // not found
}

// find EntityTxt by name
export const entityTextByName = (md, name) => {
  if (!Mdl.isModel(md) || (name || '') === '') { // model empty or name empty: return empty result
    return emptyEntityText()
  }
  for (let k = 0; k < md.EntityTxt.length; k++) {
    if (!isEntity(md.EntityTxt[k].Entity)) continue
    if (md.EntityTxt[k].Entity.Name === name) return md.EntityTxt[k]
  }
  return emptyEntityText() // not found
}

// number of entity attributes: length of EntityAttrTxt
export const entityAttrCount = (et) => {
  return isNotEmptyEntityText(et) ? et.EntityAttrTxt.length : 0
}

// find entity attribute EntityAttrTxt by entity id and attribute id
export const entityAttrTextById = (md, eId, aId) => {
  if (!Mdl.isModel(md) || typeof eId !== typeof 1 || eId < 0 || typeof aId !== typeof 1 || aId < 0) { // empty model or invalid id: return empty result
    return emptyEntityAttrTxt()
  }
  for (const et of md.EntityTxt) {
    if (!isNotEmptyEntityText(et)) continue
    if (et.Entity.EntityId !== eId) continue

    for (const eat of et.EntityAttrTxt) {
      if (!isNotEmptyEntityAttr(eat)) continue
      if (eat?.Attr?.AttrId === aId) return eat
    }
  }
  return emptyEntityAttrTxt() // not found
}

// finder to return entity name and attribute by entity id and attribute id
// finder.byId return { ok: true, name: entityName, a: EntityAttrTxt }
export const entityAttrFinder = (md) => {
  // if entity list is empty then return empty finder
  const emptyFinder = {
    byId: (eId, aId) => { return { ok: false, name: '', a: {} } }
  }
  if (!isEntityTextList(md)) return emptyFinder

  // map entity id to the map of attributes id to attribute
  let n = 0
  const eaById = {}

  for (const et of md.EntityTxt) {
    if (!isNotEmptyEntityText(et)) continue

    const aLst = {}
    eaById[et.Entity.EntityId] = {
      name: et.Entity.Name,
      attrs: aLst
    }

    for (const eat of et.EntityAttrTxt) {
      if (isNotEmptyEntityAttr(eat)) {
        aLst[eat.Attr.AttrId] = eat
        n++
      }
    }
  }
  if (n <= 0) return emptyFinder // return empty finder if there are no attributes

  // finder of attributes by entity id and attribute id
  return {
    byId: (eId, aId) => {
      if (eId < 0 || aId < 0) {
        return { ok: false, name: '', a: {} } // not found: invalid id's
      }
      const e = eaById[eId]
      if (!e) {
        return { ok: false, name: '', a: {} } // entity id not found
      }
      const ea = e.attrs[aId]
      if (!ea) {
        return { ok: false, name: e.name, a: {} } // attribute id not found
      }
      return {
        ok: true,
        name: e.name,
        a: ea
      }
    }
  }
}

// find entity attribute EntityAttrTxt by entity name and attribute name
export const entityAttrTextByName = (md, entName, attrName) => {
  if (!Mdl.isModel(md) || (entName || '') === '' || (attrName || '') === '') { // empty model or entity name or attribute: return empty result
    return emptyEntityAttrTxt()
  }
  for (const et of md.EntityTxt) {
    if (!isNotEmptyEntityText(et)) continue
    if (et.Entity.Name !== entName) continue

    for (const eat of et.EntityAttrTxt) {
      if (eat?.Attr?.Name === attrName) return eat
    }
  }
  return emptyEntityAttrTxt() // not found
}

/* model run or run text microdata:
    "Entity": [{
            "Name": "Person",
            "GenDigest": "76be220e9a1814557f3a2af2b32388e4",
            "ValueDigest": "1e39ee49c28ec786cb0309df668d3d1e",
            "RowCount": 131056,
            "Attr": ["Age", "AgeGroup", "Sex", "Income", "Salary", "SalaryGroup", "FullTime", "IsOldAge", "Pension"]
        }, {
            "Name": "Other",
            "GenDigest": "a3fddbd390e4c60101a28d892dc4597c",
            "ValueDigest": "60e2c86ffdca9d0de346a533934d31bd",
            "RowCount": 131056,
            "Attr": ["Age", "AgeGroup", "Income"]
        }
    ]
*/

// return empty run entity
export const emptyRunEntity = () => {
  return {
    Name: '',
    GenDigest: '',
    ValueDigest: '',
    RowCount: 0,
    Attr: []
  }
}

// return true if it is model run Entity and it is not empty: Name, GenDigest, RowCount > 0 and Attr[] array not empty
export const isNotEmptyRunEntity = (re) => {
  if (!re || !re.hasOwnProperty('Name') || !re.hasOwnProperty('GenDigest') || !re.hasOwnProperty('ValueDigest') || !re.hasOwnProperty('RowCount')) return false
  if (typeof re.Name !== typeof 'string' || typeof re.GenDigest !== typeof 'string' || typeof re.ValueDigest !== typeof 'string') return false
  if (typeof re.RowCount !== typeof 1) return false
  if (!Array.isArray(re?.Attr)) return false
  return re.Name !== '' && re.GenDigest !== '' && re.Attr.length > 0 && re.RowCount > 0
}

// find non-empty run entity by name in model run Entity[] array
export const runEntityByName = (r, name) => {
  if (!r || !r?.Entity || !Array.isArray(r.Entity)) return emptyRunEntity()
  if (!name || typeof name !== typeof 'string') return emptyRunEntity()

  for (const re of r.Entity) {
    if (isNotEmptyRunEntity(re) && re.Name === name) return re // entity name found
  }
  return emptyRunEntity()
}
