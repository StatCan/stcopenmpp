// db structures common functions: model and model list

import * as Dnf from './descr-note'
import * as Mlang from './language'

/*
model list:
  [
    {
      Model db.ModelDicRow // model_dic db row
      Dir   string         // model directory, relative to model root and slashed: dir/sub
      IsIni  bool          // if true the default ini file exists: models/bin/dir/sub/modelName.ini
      Extra string         // if not empty then model extra content
    }
  ]
*/
// return true if each list element isModel()
export const isModelList = (mLst) => {
  if (!mLst) return false
  if (!Array.isArray(mLst)) return false
  for (const m of mLst) {
    if (!isModel(m)) return false
  }
  return true
}

// return model list entry by digest: model_dic row and additional properties
// digest expected to be unique in models tree
export const modelByDigest = (dgst, mLst) => {
  if (!dgst || typeof dgst !== typeof 'string') return ''
  if (!mLst || !Array.isArray(mLst)) return ''
  for (const m of mLst) {
    if (isModel(m) && modelDigest(m) === dgst) return m
  }
  return emptyModel()
}

// return model folder by digest from model list
export const modelDirByDigest = (dgst, mLst) => {
  if (!dgst || typeof dgst !== typeof 'string') return ''
  if (!mLst || !Array.isArray(mLst)) return ''
  for (const m of mLst) {
    if (isModel(m) && modelDigest(m) === dgst) {
      return (!m?.Dir || m.Dir === '.' || m.Dir === '/' || m.Dir === './') ? '' : m.Dir
    }
  }
  return ''
}

// return model extra properties by digest from model list
export const modelExtraByDigest = (dgst, mLst) => {
  if (!dgst || typeof dgst !== typeof 'string') return ''
  if (!mLst || !Array.isArray(mLst)) return ''
  for (const m of mLst) {
    if (isModel(m) && modelDigest(m) === dgst) {
      if (!m.hasOwnProperty('Extra') || typeof m.Extra !== 'object' || !m.Extra) return {}
      return m.Extra
    }
  }
  return {}
}

// get link to model documentation in current language from ModelName.extra.json
export const modelDocLinkByDigest = (dgst, mLst, uiLang, modelLang) => {
  const me = modelExtraByDigest(dgst, mLst) // content of ModelName.extra.json file

  const docLst = me?.ModelDoc
  if (!Array.isArray(docLst) || docLst.length <= 0) return ''

  const ui2p = Mlang.splitLangCode(uiLang)
  let docLink = ''
  let fLink = ''

  for (let k = 0; k < docLst.length; k++) {
    const dlc = (docLst[k]?.LangCode || '')
    if (typeof dlc === typeof 'string') {
      if (dlc.toLowerCase() === ui2p.lower) {
        docLink = docLst[k]?.Link || ''
        break
      }
      if (fLink === '') fLink = (dlc.toLowerCase() === ui2p.first) ? (docLst[k]?.Link || '') : ''
    }
  }
  if (docLink === '' && fLink !== '') {
    docLink = fLink // match UI language by code: en-US => en
  }
  if (docLink !== '') return docLink

  const mlc = modelLang.LangCode.toLowerCase() // find link to model documentation in model language

  for (let k = 0; k < docLst.length; k++) {
    const dlc = (docLst[k]?.LangCode || '')
    if ((typeof dlc === typeof 'string') && dlc.toLowerCase() === mlc) {
      docLink = docLst[k]?.Link || ''
      break
    }
  }

  // if link to model documentation not found by language then use first link
  if (docLink === '') docLink = docLst[0]?.Link || ''

  return docLink
}

// return empty Model and additional properties of model list
export const emptyModel = () => {
  return {
    Model: {
      Name: '',
      Digest: '',
      CreateDateTime: '',
      DefaultLangCode: '',
      Version: ''
    },
    Dir: '',
    IsIni: false,
    Extra: ''
  }
}

// if this is a model (model_dic)
const hasModelProperties = (md) => {
  if (!md) return false
  if (!md.hasOwnProperty('Model')) return false
  return md.Model.hasOwnProperty('Name') && md.Model.hasOwnProperty('Digest') &&
    md.Model.hasOwnProperty('CreateDateTime') && md.Model.hasOwnProperty('DefaultLangCode') && md.Model.hasOwnProperty('Version')
}

// if this is not empty model
export const isModel = (md) => {
  if (!hasModelProperties(md)) return false
  return (md.Model.Name || '') !== '' && (md.Model.Digest || '') !== '' && (md.Model.CreateDateTime || '') !== ''
}

// if this is an empty model: model with empty name and digest
export const isEmptyModel = (md) => {
  if (!hasModelProperties(md)) return false
  return (md.Model.Name || '') === '' || (md.Model.Digest || '') === '' || (md.Model.CreateDateTime || '') === ''
}

// digest of the model
export const modelDigest = (md) => {
  if (!md) return ''
  if (!md.hasOwnProperty('Model')) return ''
  return (md.Model.Digest || '')
}

// name of the model
export const modelName = (md) => {
  if (!md) return ''
  if (!md.hasOwnProperty('Model')) return ''
  return (md.Model.Name || '')
}

// model name and version
export const modelNameVer = (md) => {
  return isModel(md) ? md.Model.Name + ((md.Model.Version || '') ? ': ' + md.Model.Version : '') : ''
}

// make model title
export const modelTitle = (md) => {
  if (!isModel(md)) return ''
  const descr = Dnf.descrOfDescrNote(md)
  return md.Model.Name + ((md.Model.Version || '') ? ': ' + md.Model.Version : '') + ((descr !== '') ? ': ' + descr : '')
}
