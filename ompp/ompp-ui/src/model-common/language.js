// db structures common functions: language and language list

// hard coded enum code for total enum id
export const ALL_WORD_CODE = 'all'

// return empty language-specific model words list
export const emptyWordList = () => {
  return {
    ModelName: '',
    ModelDigest: '',
    LangCode: '',
    LangWords: [],
    ModelLangCode: '',
    ModelWords: [],
    Msg: []
  }
}

// find label by code in language-specific model words list, return code if label not found
export const wordByCode = (mw, code) => {
  if (!code) return ''
  if (!mw) return code
  // search in translated strings
  if (mw.Msg && (mw.Msg.length || 0) > 0) {
    for (let k = 0; k < mw.Msg.length; k++) {
      if (mw.Msg[k].Code === code) return mw.Msg[k].Label
    }
  }
  // search in model-specific list of words
  if (mw.ModelWords && (mw.ModelWords.length || 0) > 0) {
    for (let k = 0; k < mw.ModelWords.length; k++) {
      if (mw.ModelWords[k].Code === code) return mw.ModelWords[k].Label
    }
  }
  // search in common language-specific list of words
  if (mw.LangWords && (mw.LangWords.length || 0) > 0) {
    for (let k = 0; k < mw.LangWords.length; k++) {
      if (mw.LangWords[k].Code === code) return mw.LangWords[k].Label
    }
  }
  return code // not found: return original code
}

// from language code return the code in lower case and first part of it: en_US => en or fr-CA => fr
export const splitLangCode = (langCode) => {
  const r = {
    lower: '', // language code in lower case
    first: '', // first part of language code to match fr-CA to FR model language
    isEmpty: true
  }
  if (langCode && typeof langCode === typeof 'string' && langCode !== '') {
    r.lower = langCode.toLowerCase()
    const pLst = r.lower.split(/[-_]/)
    r.first = (Array.isArray(pLst) && pLst.length > 0) ? pLst[0] : ''
    r.isEmpty = !r.lower && !r.first
  }
  return r
}

// return empty list of model languages
/*
[{
  "LangCode": "EN",
  "Name": "English"
}]
*/
export const emptyLangList = () => {
  return []
}

// return true if each list element isModel()
export const isLangList = (ml) => {
  if (!ml) return false
  if (!Array.isArray(ml)) return false
  for (const lcn of ml) {
    if (!lcn.hasOwnProperty('LangCode') || !lcn.hasOwnProperty('Name')) return false
  }
  return true
}

// find language name by code in model list of languages, return code if name not found
export const langNameByCode = (ml, code) => {
  if (!code) return ''
  if (!ml) return code
  // search in model-specific list of words
  if (isLangList(ml)) {
    for (const lcn of ml) {
      if ((lcn.LangCode || '') === code) return (lcn.Name || '') || code
    }
  }
  return code // not found: return original code
}
