// db structures common functions: helpers
import cloneDeep from 'lodash/cloneDeep'

export const deepClone = cloneDeep

// return true if argument hasOwnProperty length
const hasLength = (a) => { return a && a.hasOwnProperty('length') }

// return true if argument has length > 0
export const isLength = (a) => { return hasLength(a) && (a.length || 0) > 0 }

// return argument.length or zero if no length
export const lengthOf = (a) => { return hasLength(a) ? (a.length || 0) : 0 }

// return route path for parameter or output table page
// for example: /model/:digest/run/:runDigest/parameter/:parameterName
export const parameterRunPath = (model, runDigest, paramName) => {
  return '/model/' + encodeURIComponent(model || '-') +
    '/run/' + encodeURIComponent(runDigest || '-') +
    '/parameter/' + encodeURIComponent(paramName || '-')
}
export const parameterWorksetPath = (model, wsName, paramName) => {
  return '/model/' + encodeURIComponent(model || '-') +
    '/set/' + encodeURIComponent(wsName || '') +
    '/parameter/' + encodeURIComponent(paramName || '-')
}
export const tablePath = (model, runDigest, tableName) => {
  return '/model/' + encodeURIComponent(model || '-') +
    '/run/' + encodeURIComponent(runDigest || '-') +
    '/table/' + encodeURIComponent(tableName || '-')
}
export const microdataPath = (model, runDigest, entityName) => {
  return '/model/' + encodeURIComponent(model || '-') +
    '/run/' + encodeURIComponent(runDigest || '-') +
    '/entity/' + encodeURIComponent(entityName || '-')
}

// length of timestap string
export const TIME_STAMP_LEN = '1234_67_90_23_56_89_123'.length
export const TIME_STAMP_SEC_LEN = '1234_67_90_23_56_89'.length

// return date-time string: truncate milliseconds from timestamp
export const dtStr = (ts) => { return ((typeof ts === typeof 'string') ? (ts || '').substr(0, 19) : '') }

// format date-time string to timestamp string: 2021-07-16 13:40:53.882 into 2021_07_16_13_40_53_882
export const toUnderscoreTimeStamp = (ts) => {
  if (!ts || typeof ts !== typeof 'string') return ''
  return ts.replace(/[:\s-.]/g, '_')
}

// format timestamp string to date-time string: 2021_07_16_13_40_53_882 into 2021-07-16 13:40:53.882
export const fromUnderscoreTimeStamp = (ts) => {
  if (!ts || typeof ts !== typeof 'string' || (ts.length !== TIME_STAMP_LEN && ts.length !== TIME_STAMP_SEC_LEN)) return ''

  const a = Array.from(ts, (c, i) => {
    if (i === 4 && c === '_') return '-'
    if (i === 7 && c === '_') return '-'
    if (i === 10 && c === '_') return ' '
    if (i === 13 && c === '_') return ':'
    if (i === 16 && c === '_') return ':'
    if (i === 19 && c === '_') return '.'
    return c
  })
  return a.join('')
}

// return true if argument look like as date-time string: 1111_00_00_99_99_99_999
export const isUnderscoreTimeStamp = (ts) => {
  if (!ts || typeof ts !== typeof 'string' || (ts.length !== TIME_STAMP_LEN && ts.length !== TIME_STAMP_SEC_LEN)) return false

  for (let i = 0; i < ts.length; i++) {
    if (i === 4 || i === 7 || i === 10 || i === 13 || i === 16 || i === 19) {
      if (ts[i] !== '_') return false
    } else {
      if (ts[i] < '0' || ts[i] > '9') return false
    }
  }
  return true
}

// convert file modification time tp timestamp string: YYYY-MM-DD hh:mm:ss.SSS
export const modTsToTimeStamp = (t) => {
  if (!t || t <= 0) return ''
  const dt = new Date()
  dt.setTime(t)
  return dtToTimeStamp(dt)
}

// format date-time to timestamp string: YYYY-MM-DD hh:mm:ss.SSS
export const dtToTimeStamp = (dt) => {
  if (!dt || !(dt instanceof Date)) return ''
  const month = dt.getMonth() + 1
  const day = dt.getDate()
  const hour = dt.getHours()
  const minute = dt.getMinutes()
  const sec = dt.getSeconds()
  const ms = dt.getMilliseconds()
  //
  return dt.getFullYear().toString() + '-' +
    (month < 10 ? '0' + month.toString() : month.toString()) + '-' +
    (day < 10 ? '0' + day.toString() : day.toString()) + ' ' +
    (hour < 10 ? '0' + hour.toString() : hour.toString()) + ':' +
    (minute < 10 ? '0' + minute.toString() : minute.toString()) + ':' +
    (sec < 10 ? '0' + sec.toString() : sec.toString()) + '.' +
    ('000' + ms.toString()).slice(-3)
}

// format date-time to timestamp string: YYYY_MM_DD_hh_mm_ss_SSS
export const dtToUnderscoreTimeStamp = (dt) => {
  if (!dt || !(dt instanceof Date)) return ''
  const month = dt.getMonth() + 1
  const day = dt.getDate()
  const hour = dt.getHours()
  const minute = dt.getMinutes()
  const sec = dt.getSeconds()
  const ms = dt.getMilliseconds()
  //
  return dt.getFullYear().toString() + '_' +
    (month < 10 ? '0' + month.toString() : month.toString()) + '_' +
    (day < 10 ? '0' + day.toString() : day.toString()) + '_' +
    (hour < 10 ? '0' + hour.toString() : hour.toString()) + '_' +
    (minute < 10 ? '0' + minute.toString() : minute.toString()) + '_' +
    (sec < 10 ? '0' + sec.toString() : sec.toString()) + '_' +
    ('000' + ms.toString()).slice(-3)
}

// time to complete: hours, minutes, seconds
export const toIntervalStr = (startTs, stopTs) => {
  if ((startTs || '') === '' || (stopTs || '') === '') return ''

  const start = Date.parse(startTs)
  const stop = Date.parse(stopTs)
  if (start && stop) {
    const s = Math.floor((stop - start) / 1000) % 60
    const m = Math.floor((stop - start) / (60 * 1000)) % 60
    const h = Math.floor((stop - start) / (60 * 60 * 1000))

    return ((h > 0) ? h.toString() + ':' : '') +
      ('00' + m.toString()).slice(-2) + ':' +
      ('00' + s.toString()).slice(-2)
  }
  return ''
}

// comvert seconds to hours, minutes, seconds string
export const toHourMinSec = (totalSec) => {
  if (!totalSec) return '00:00'

  const s = totalSec % 60
  const m = Math.floor(totalSec / 60) % 60
  const h = Math.floor(totalSec / (60 * 60))

  return ((h > 0) ? h.toString() + ':' : '') +
    ('00' + m.toString()).slice(-2) + ':' +
    ('00' + s.toString()).slice(-2)
}

// convert size in bytes to KB, MB,... rounded value and unit name
export const fileSizeParts = (size) => {
  if (!size || typeof size !== typeof 1 || size < 0) return { val: '0', name: '' }

  if (size < 1024) {
    return { val: size.toFixed(0), name: 'Bytes' }
  }
  size /= 1024.0
  if (size < 1024) {
    return { val: size.toFixed(0), name: 'KB' }
  }
  size /= 1024.0
  if (size < 1024) {
    return { val: size.toFixed(0), name: 'MB' }
  }
  size /= 1024.0
  if (size < 1024) {
    return { val: size.toFixed(0), name: 'GB' }
  }
  size /= 1024.0
  return { val: size.toFixed(2), name: 'TB' }
}

// clean string input: replace special characters "'`$}{@\ with space and trim
export const cleanTextInput = (sValue) => {
  if (typeof sValue !== typeof 'string' || sValue === '' || sValue === void 0) return ''
  let s = sValue.replace(/[`$}{@\\]/g, '\xa0')
  s = s.replace(/\s+/g, '\xa0').trim()
  return s || ''
}

// check if string enterd and clean it to make compatible with file name input rules:
// replace special characters "'`:*?><|$}{@&^;/\ with underscore _ and trim
export const doFileNameClean = (fileName) => {
  return (fileName || '') ? { isEntered: true, name: cleanFileNameInput(fileName) } : { isEntered: false, name: '' }
}

// invalid characters for file name, URL or dangerous for js
export const invalidFileNameChars = '"\'`:*?><|$}{@&^;/\\'

// clean file name input: replace special characters "'`:*?><|$}{@&^;/\ with underscore _ and trim
export const cleanFileNameInput = (fileName) => {
  if (typeof fileName !== typeof 'string' || fileName === '' || fileName === void 0) return ''
  let s = fileName.replace(/["'`:*?><|$}{@&^;/\\]/g, '_')
  s = s.replace(/__+/g, '_')
  s = s.replace(/\s+/g, '\xa0').trim()
  return s || ''
}

// clean path input: remove special characters "'`:*?><|$}{@&^; and force it to be relative path and use / separator
export const cleanPathInput = (path) => {
  if (typeof path !== typeof 'string' || path === '' || path === void 0) return ''

  // remove special characters and replace all \ with /
  let s = path.replace(/["'`:*?><|$}{@&^;]/g, '').replace(/\\/g, '/').trim()

  // replace repeated // with single / and remove all ..
  let n = s.length
  let nPrev = n
  do {
    nPrev = n
    s = s.replace('//', '/').replace(/\.\./g, '')
    n = s.length
  } while (n > 0 && nPrev !== n)

  // remove leading /
  s = s.replace(/^\//, '')
  return s || ''
}

// clean column value input from sql dangerous characters: "'`:?|$}{@&;%\ and -- replace it with underscore _ and trim
export const cleanColumnValueInput = (src) => {
  if (typeof src !== typeof 'string' || src === '' || src === void 0) return ''
  let s = src.replace(/["'`:?|$}{@&;%\\]/g, '_')
  s = s.replace(/__+/g, '_')
  s = s.replace(/\s+/g, '\xa0').trim()
  return s || ''
}

// clean integer as non-negative input, ignore + or - sign
export const cleanIntNonNegativeInput = (sValue, nDefault = 0) => {
  if (sValue === '' || sValue === void 0) return nDefault
  const n = parseInt(sValue)
  return (!isNaN(n) && Number.isInteger(n)) ? n : nDefault
}

// clean float number input
export const cleanFloatInput = (sValue, fDefault = 0.0) => {
  if (sValue === '' || sValue === void 0) return fDefault
  const f = parseFloat(sValue)
  return !isNaN(f) ? f : fDefault
}

// split csv values line into string array, trim and unquote each value, quotes can be "double" or 'single'
export const splitCsv = (srcLine) => {
  if (!srcLine || typeof srcLine !== typeof 'string') return []

  // clean string input: replace special characters `$}{@\ with space and trim
  const clean = (sValue) => {
    if (typeof sValue !== typeof 'string' || sValue === '' || sValue === void 0) return ''
    const s = sValue.replace(/[`$}{@\\]/g, '\xa0').trim()
    return s || ''
  }

  const str = clean(srcLine).trim()
  if (str.length <= 0) return []

  // trim and unquote value, quotes can be "double" or 'single'
  const trimUnquote = (s) => {
    s = s.trim()
    if (s.length > 1 && (s[0] === '"' || s[0] === "'") && s[s.length - 1] === s[0]) s = s.substring(1, s.length - 1)
    return s
  }

  const csv = []
  let isQ = false
  let quote = ''
  let val = ''

  for (let k = 0; k < str.length; k++) {
    const c = str[k]

    // if this end of value then it push to result
    if (!isQ && c === ',') {
      csv.push(trimUnquote(val))
      val = ''
      continue
    }

    // if open quote found then append it to the value
    if (!isQ && (c === '"' || c === "'")) {
      val += c
      isQ = true
      quote = c
      continue
    }
    // if closing quote found then check if it is duplicate quote and append one quote for each duplicate
    if (isQ && c === quote) {
      if (k >= str.length - 1 || str[k + 1] !== c) val += c
      isQ = false
      quote = ''
      continue
    }

    val += c
  }
  // push last value to result if source string is not empty
  if (str.length > 0) {
    csv.push(trimUnquote(val))
  }

  return csv
}

export const CALCULATED_ID_OFFSET = 12000 // calculated id offset, for example for Attr34 calculated attribute id is 12034

// return calculation function or comparison expression, for example: AVG => OM_AVG or return empty '' string on error
export const toCalcFnc = (fncSrc, eSrc) => {
  switch (fncSrc) {
    case 'AVG':
      return 'OM_AVG(' + eSrc + ')'
    case 'COUNT':
      return 'OM_COUNT(' + eSrc + ')'
    case 'SUM':
      return 'OM_SUM(' + eSrc + ')'
    case 'MAX':
      return 'OM_MAX(' + eSrc + ')'
    case 'MIN':
      return 'OM_MIN(' + eSrc + ')'
    case 'VAR':
      return 'OM_VAR(' + eSrc + ')'
    case 'SD':
      return 'OM_SD(' + eSrc + ')'
    case 'SE':
      return 'OM_SE(' + eSrc + ')'
    case 'CV':
      return 'OM_CV(' + eSrc + ')'
  }
  return ''
}

// return comparison expression, for example: DIFF => expr0[variant] - expr0[base] or return empty '' string on error
export const toCompareFnc = (fncSrc, eSrc) => {
  switch (fncSrc) {
    case 'DIFF':
      return eSrc + '[variant] - ' + eSrc + '[base]'
    case 'RATIO':
      return eSrc + '[variant] / ' + eSrc + '[base]'
    case 'PERCENT':
      return '100.0 * (' + eSrc + '[variant] - ' + eSrc + '[base]) / ' + eSrc + '[base]'
  }
  return ''
}

// return csv calculation name by source function name, ex: AVG => avg or return empty '' string on error
export const toCsvFnc = (src) => {
  switch (src) {
    case 'AVG':
      return 'avg'
    case 'COUNT':
      return 'count'
    case 'SUM':
      return 'sum'
    case 'MAX':
      return 'max'
    case 'MIN':
      return 'min'
    case 'VAR':
      return 'var'
    case 'SD':
      return 'sd'
    case 'SE':
      return 'se'
    case 'CV':
      return 'cv'
    case 'DIFF':
      return 'diff'
    case 'RATIO':
      return 'ratio'
    case 'PERCENT':
      return 'percent'
  }
  return ''
}

// return aggregated comparison expression, for example: AVG, DIFF => OM_AVG(Income[variant] - Income[base])
// if comparison cmpFnc is empty then return aggregation expression: AVG => OM_AVG(Income)
// or return empty '' string on error
export const toAggregateCompareFnc = (aggrFnc, cmpFnc, eSrc) => {
  if (!cmpFnc) {
    return toCalcFnc(aggrFnc, eSrc)
  }
  switch (cmpFnc) {
    case 'DIFF':
      return toCalcFnc(aggrFnc, eSrc + '[variant] - ' + eSrc + '[base]')
    case 'RATIO':
      return toCalcFnc(aggrFnc, eSrc + '[variant] / ' + eSrc + '[base]')
    case 'PERCENT':
      return toCalcFnc(aggrFnc, '100.0 * (' + eSrc + '[variant] - ' + eSrc + '[base]) / ' + eSrc + '[base]')
  }
  return ''
}

// value filter comparisons
export const filterOpList = [{
  code: '=',
  label: '='
}, {
  code: '!=',
  label: '!='
}, {
  code: '<',
  label: '<'
}, {
  code: '<=',
  label: '<='
}, {
  code: '>',
  label: '>'
}, {
  code: '>=',
  label: '>='
}, {
  code: 'BETWEEN',
  label: 'Between min and max'
}, {
  code: 'IN',
  label: 'In the list'
}]
