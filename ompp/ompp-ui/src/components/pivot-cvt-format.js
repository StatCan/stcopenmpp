// pivot table value processing: formatters

import * as PcvtHlp from './pivot-cvt-helper'

export const maxDecimalDefault = 4 // by default use 4 decimals max to format expressions and accumulators

// more-less options to control decimals for float number format or show "source" value without any formatting
/* eslint-disable no-multi-spaces */
export const moreLessDefault = () => ({
  isRawUse: false,      // if true then show "raw value" button
  isRawValue: false,    // if true then display "raw" value, do not apply format(),
  isFloat: false,       // if true then format decimals and use more and less buttons
  isAllDecimal: false,  // if true then show all decimals, do not limit it by max decimals
  isDoMore: false,      // if true then "more decimals" button enabled
  isDoLess: false       // if true then "less decimals" button enabled
})
/* eslint-enable no-multi-spaces */

// default format: do not convert value, only validate if empty value allowed (if parameter nullable)
export const formatDefault = (options) => {
  const opts = Object.assign({}, moreLessDefault(), { isNullable: false, isRawValue: true, isByKey: false }, options)
  return {
    format: (val) => val, // disable format() value by default
    parse: (s) => s, // disable parse() value by default
    isValid: (s) => true, // disable validation by default
    options: () => Object.assign({}, opts), // immutable default options: return a copy
    resetOptions: () => {},
    doRawValue: () => {},
    doMore: () => {},
    doLess: () => {},
    byKey: (isByKey) => {}
  }
}

// format number as float
// options is a merge of formatNumber options (see below) and more-less options to control decimals
export const formatFloat = (options) => {
  const opts = Object.assign({},
    moreLessDefault(),
    formatNumber.makeOpts(options),
    { isRawUse: true, isFloat: true })

  // adjust format options and more-less options
  if (opts.maxDecimal < 0) opts.maxDecimal = 0
  if (opts.nDecimal < 0) opts.nDecimal = 0
  if (opts.nDecimal > opts.maxDecimal) opts.nDecimal = opts.maxDecimal

  opts.isDoMore = !opts.isRawValue && (opts.nDecimal < opts.maxDecimal || !opts.isAllDecimal)
  opts.isDoLess = !opts.isRawValue && (opts.nDecimal > 0 || opts.isAllDecimal)

  // save default format options
  const defaultOpts = {
    nDecimal: opts.nDecimal,
    maxDecimal: opts.maxDecimal,
    isRawValue: opts.isRawValue,
    isAllDecimal: opts.isAllDecimal,
    isDoMore: opts.isDoMore,
    isDoLess: opts.isDoLess
  }

  return {
    format: (val) => {
      if (opts.isRawValue) return val // return source value
      return formatNumber.format(val, opts)
    },

    parse: (s) => {
      if (s === '' || s === void 0) return void 0
      const v = parseFloat(s)
      return !isNaN(v) ? v : void 0
    },

    isValid: (s) => {
      if (s === '' || s === void 0) return opts.isNullable
      if (typeof s === typeof 1) return isFinite(s)
      if (typeof s === typeof 'string') return /^[-+]?[0-9]+(\.[0-9]+)?([eE][-+]?[0-9]+)?$/.test(s.trim())
      return false
    },

    options: () => opts,

    resetOptions: () => {
      Object.assign(opts, defaultOpts)
    },

    doRawValue: () => {
      opts.isRawValue = !opts.isRawValue
      opts.isDoMore = !opts.isRawValue && (opts.nDecimal < opts.maxDecimal || !opts.isAllDecimal)
      opts.isDoLess = !opts.isRawValue && (opts.nDecimal > 0 || opts.isAllDecimal)
    },

    doMore: () => {
      if (!opts.isDoMore) return

      if (opts.nDecimal < opts.maxDecimal) {
        opts.nDecimal++
      } else {
        opts.isAllDecimal = true
      }

      opts.isDoMore = !opts.isRawValue && (opts.nDecimal < opts.maxDecimal || !opts.isAllDecimal)
      opts.isDoLess = !opts.isRawValue && (opts.nDecimal > 0 || opts.isAllDecimal)
    },

    doLess: () => {
      if (!opts.isDoLess) return

      if (opts.isAllDecimal) {
        opts.isAllDecimal = false
      } else {
        if (opts.nDecimal > 0) opts.nDecimal--
      }

      opts.isDoMore = !opts.isRawValue && (opts.nDecimal < opts.maxDecimal || !opts.isAllDecimal)
      opts.isDoLess = !opts.isRawValue && (opts.nDecimal > 0 || opts.isAllDecimal)
    }
  }
}

// format number as integer
export const formatInt = (options) => {
  const opts = Object.assign({},
    moreLessDefault(),
    formatNumber.makeOpts(options),
    { nDecimal: 0, maxDecimal: 0, isRawUse: true }
  )
  return {
    format: (val) => {
      if (opts.isRawValue) return val // return source value
      return formatNumber.format(val, opts)
    },
    parse: (s) => {
      if (s === '' || s === void 0) return void 0
      const v = parseInt(s, 10)
      return !isNaN(v) ? v : void 0
    },
    isValid: (s) => {
      if (s === '' || s === void 0) return opts.isNullable
      if (typeof s === typeof 1) return Number.isInteger(s)
      if (typeof s === typeof 'string') return /^[-+]?[0-9]+$/.test(s.trim())
      return false
    },
    options: () => opts,
    resetOptions: () => {
      opts.nDecimal = 0
      opts.maxDecimal = 0
      opts.isRawValue = false
    },
    doRawValue: () => { opts.isRawValue = !opts.isRawValue }
  }
}

// format enum value: return enum label by enum id
export const formatEnum = (options) => {
  const opts = Object.assign({},
    moreLessDefault(),
    { enums: [], isRawUse: true, isNullable: false },
    options)

  const labels = {} // enums as map of id to label
  for (const e of opts.enums) {
    labels[e.value] = e.label
  }

  return {
    getEnums: () => opts.enums, // return enum [value, label] for select dropdown
    // return enum id by enum label or void 0 if not found
    enumIdByLabel: (label) => {
      if (label === void 0 || label === '' || typeof label !== typeof 'string') return void 0 // invalid or empty label
      for (const e of opts.enums) {
        if (e.label === label) return e.value
      }
      return void 0 // not found
    },
    format: (val) => {
      if (opts.isRawValue) return val // return source value
      return (val !== void 0 && val !== null) ? labels[val] || val : ''
    },
    parse: (s) => {
      if (s === '' || s === void 0) return void 0
      const v = parseInt(s, 10)
      return !isNaN(v) ? v : void 0
    },
    isValid: (s) => {
      if (s === '' || s === void 0) return opts.isNullable
      if (typeof s === typeof 1) {
        return labels.hasOwnProperty(s)
      }
      if (typeof s === typeof 'string' && /^[-+]?[0-9]+$/.test(s)) {
        const v = parseInt(s, 10)
        return !isNaN(v) ? labels.hasOwnProperty(v) : false
      }
      return false
    },
    options: () => opts,
    resetOptions: () => { opts.isRawValue = false },
    doRawValue: () => { opts.isRawValue = !opts.isRawValue }
  }
}

// format boolean value
export const formatBool = (options) => {
  const opts = Object.assign({},
    moreLessDefault(),
    { isRawUse: true, isNullable: false },
    options)
  return {
    format: (val) => {
      if (opts.isRawValue) return val // return source value
      return (val !== void 0 && val !== null) ? (val ? '\u2713' : 'x') || val : ''
    },
    parse: (s) => {
      return PcvtHlp.parseBool(s)
    },
    isValid: (s) => {
      if (s === '' || s === void 0) return opts.isNullable
      return (typeof PcvtHlp.parseBool(s)) === typeof true
    },
    options: () => opts,
    resetOptions: () => { opts.isRawValue = false },
    doRawValue: () => { opts.isRawValue = !opts.isRawValue }
  }
}

// format number, shared portion
/* eslint-disable no-multi-spaces */
const formatNumber = {
  makeOpts: (options) => {
    const opts = Object.assign({},
      {
        isNullable: false,            // if true then allow empty (NULL) value
        isRawValue: false,            // if true then display "raw" value, do not apply format()
        isAllDecimal: false,          // if true then show all decimals, do not limit it by max decimals
        locale: '',                   // locale name, if defined then return toLocaleString()
        nDecimal: maxDecimalDefault,  // number of decimals, ex: 2 => 1234.56
        maxDecimal: maxDecimalDefault // max decimals to display, using toFixed(nDecimal) and nDecimal <= maxDecimal
      },
      options)
    return opts
  },

  format: (val, opts) => {
    if (typeof val !== typeof 1) {
      return val !== void 0 ? val : '' // invalid or undefined value
    }
    if (!opts) return val // no options: default output

    // if locale defined then use built-in locale conversion
    if (opts.locale) {
      return opts.isRawValue
        ? val.toString()
        : (opts.isAllDecimal
            ? val.toLocaleString(opts.locale, { maximumFractionDigits: 20 })
            : val.toLocaleString(opts.locale, { minimumFractionDigits: opts.nDecimal, maximumFractionDigits: opts.nDecimal })
          )
    }
    // else use default numeric formating, example: 1,234.5678
    const groupSep = ','   // thousands group separator
    const groupLen = 3     // grouping size
    const decimalSep = '.' // decimals separator

    // convert to numeric string, optionally with fixed decimals
    const src = (!opts.isRawValue && opts.nDecimal >= 0) ? val.toFixed(opts.nDecimal) : val.toString()

    if (src.length <= 1) return src // empty value as string or single digit number

    // spit integer and decimals parts, optionally replace decimals separator
    const nD = src.search(/(\.|e|E)/)
    const left = nD > 0 ? src.substr(0, nD) : src
    const rest = nD >= 0
      ? (decimalSep !== '.')
          ? src.substr(nD).replace('.', decimalSep)
          : src.substr(nD)
      : ''

    // if only decimals
    // or no integer part digit grouping required then return numeric string
    let nL = left.length
    if (nL <= 0 || !groupSep) return left + rest

    const isSign = left[0] === '-' || left[0] === '+'
    let isLast = false
    let sL = ''
    do {
      const n = nL - groupLen
      isLast = n <= 0 || (isSign && n === 1)
      sL = (!isLast ? groupSep + left.substr(n, groupLen) : left.substr(0, nL)) + sL
      nL = n
    } while (!isLast)

    return sL + rest
  }
}

// format using multiple formats defined by key: format int, float or bool
// options.formatter[key] must be defined for all keys and return a specific formatter object.
// isByKey option is always true and cannot be changed using byKey() method
export const formatByKey = (options) => {
  const opts = Object.assign({}, formatDefault(options).options(), { isByKey: true })

  // make float options
  const floatOpts = Object.assign({}, formatFloat().options())

  opts.isFloat = opts?.isFloat || false
  let m = 0
  for (const fmtKey in opts.formatter) {
    if (!(opts.formatter[fmtKey] instanceof Object)) continue

    const fo = opts.formatter[fmtKey]?.options?.()
    if (!fo?.isFloat) continue
    opts.isFloat = true

    fo.isAllDecimal = fo?.isAllDecimal || false
    if (!fo?.maxDecimal || fo.maxDecimal < 0) fo.maxDecimal = 0
    if (!fo?.nDecimal || fo.nDecimal < 0) fo.nDecimal = 0
    if (fo.nDecimal > fo.maxDecimal) fo.nDecimal = fo.maxDecimal

    if (m < fo.maxDecimal) m = fo.maxDecimal
  }
  if (opts.isFloat) floatOpts.maxDecimal = m

  // update all decimals, more, less options based on each float item format
  const updateAllMoreLess = () => {
    if (!opts.isFloat) {
      opts.isDoMore = false
      opts.isDoLess = false
      return
    }
    let isAll = true
    let isAny = false

    for (const fmtKey in opts.formatter) {
      const fo = opts.formatter[fmtKey]?.options()
      if (!fo?.isFloat) continue

      isAll = isAll && fo.isAllDecimal
      isAny = isAny || fo.isAllDecimal || fo.nDecimal > 0
    }
    floatOpts.isAllDecimal = isAll

    opts.isDoMore = !opts.isRawValue && (floatOpts.nDecimal < floatOpts.maxDecimal || !floatOpts.isAllDecimal)
    opts.isDoLess = !opts.isRawValue && (floatOpts.nDecimal > 0 || floatOpts.isAllDecimal || isAny)
  }
  updateAllMoreLess()

  // save default format options
  const defaultFloatOpts = {
    nDecimal: floatOpts.nDecimal,
    maxDecimal: floatOpts.maxDecimal,
    isAllDecimal: floatOpts.isAllDecimal,
    isFloat: true
  }
  const defaultOpts = {
    isRawUse: opts.isRawUse,
    isRawValue: opts.isRawValue,
    isFloat: opts.isFloat,
    isAllDecimal: opts.isAllDecimal,
    isDoMore: opts.isDoMore,
    isDoLess: opts.isDoLess,
    isByKey: opts.isByKey
  }

  const defaultiItemsFormat = {}
  for (const fmtKey in opts.formatter) {
    defaultiItemsFormat[fmtKey] = Object.assign({}, opts.formatter[fmtKey].options())
  }

  return {
    byKey: (isByKey) => {}, // by key always true

    format: (val, fmtKey) => {
      if (opts.isRawValue || typeof fmtKey === typeof void 0 || !opts.isByKey || typeof opts?.formatter[fmtKey] === typeof void 0) {
        return val
      }
      return opts.formatter[fmtKey].format(val)
    },

    parse: (s, fmtKey) => {
      if (typeof fmtKey === typeof void 0 || !opts.isByKey || typeof opts?.formatter[fmtKey] === typeof void 0) {
        return s
      }
      return opts.formatter[fmtKey].parse(s)
    },

    isValid: (s, fmtKey) => {
      if (typeof fmtKey === typeof void 0 || !opts.isByKey || typeof opts?.formatter[fmtKey] === typeof void 0) {
        return false
      }
      return opts.formatter[fmtKey].isValid(s)
    },

    options: () => opts,

    resetOptions: () => {
      Object.assign(opts, defaultOpts)
      Object.assign(floatOpts, defaultFloatOpts)

      for (const fmtKey in opts.formatter) {
        const fo = opts.formatter[fmtKey].options()
        Object.assign(fo, defaultiItemsFormat[fmtKey])
      }
    },

    doRawValue: () => {
      opts.isRawValue = !opts.isRawValue
      updateAllMoreLess()
    },

    doMore: () => {
      if (!opts.isDoMore) return

      if (floatOpts.nDecimal < floatOpts.maxDecimal) {
        floatOpts.nDecimal++
      } else {
        floatOpts.isAllDecimal = true
      }

      for (const fmtKey in opts.formatter) {
        const fo = opts.formatter[fmtKey].options()
        if (!fo || !fo?.isFloat) continue

        if (fo.nDecimal < floatOpts.nDecimal) {
          // use the same decimals for all values to avod more less clicks without table body changes
          fo.nDecimal = floatOpts.nDecimal
        } else {
          fo.isAllDecimal = true // that item format is at all decimals now
        }
      }
      updateAllMoreLess()
    },

    doLess: () => {
      if (!opts.isDoLess) return

      if (!floatOpts.isAllDecimal && floatOpts.nDecimal > 0) {
        floatOpts.nDecimal--
      }
      floatOpts.isAllDecimal = false

      for (const fmtKey in opts.formatter) {
        const fo = opts.formatter[fmtKey].options()
        if (!fo || !fo?.isFloat) continue

        if (fo.nDecimal > floatOpts.nDecimal) {
          // use the same decimals for all values to avod more less clicks without table body changes
          fo.nDecimal = floatOpts.nDecimal
        }
        fo.isAllDecimal = false
      }
      updateAllMoreLess()
    },

    // get shared float options
    floatOptions: () => { return Object.assign({}, floatOpts) },

    // set number of decimals and all decimals flag (nDecimal and isAllDecimal) for all float formatters
    setDecimals: (n, isAll) => {
      if (n < 0)  n = 0
      floatOpts.isAllDecimal = isAll || (n > floatOpts.maxDecimal)
      floatOpts.nDecimal = !floatOpts.isAllDecimal ? n : floatOpts.maxDecimal

      for (const fmtKey in opts.formatter) {
        const fo = opts.formatter[fmtKey].options()
        if (!fo || !fo?.isFloat) continue

        fo.isAllDecimal = floatOpts.isAllDecimal || (floatOpts.nDecimal > fo.maxDecimal)
        fo.nDecimal = !fo.isAllDecimal ? floatOpts.nDecimal : fo.maxDecimal
      }
      updateAllMoreLess()
    }
  }
}

// format number as float using multiple formats defined by key, e.g. different number decimals
// if isByKey option is true and itemsFormat is not empty object then caller can use format.format(val, fmtKey)
// if itemsFormat[fmtKey] exist then it must have isAllDecimal, nDecimal, maxDecimal properties
// otherwise formatFloat is used
export const formatFloatByKey = (options) => {
  const floatFmt = formatFloat(options)
  const opts = floatFmt.options()

  // check if formats by key enabled
  opts.isByKey = opts?.isByKey || false
  opts.itemsFormat = opts?.itemsFormat || {}
  if (!(opts?.itemsFormat instanceof Object)) opts.itemsFormat = {}

  let n = 0
  let m = 0
  for (const fmtKey in opts.itemsFormat) {
    if (opts.itemsFormat[fmtKey] instanceof Object) {
      n++
      const fmt = opts.itemsFormat[fmtKey]

      fmt.isAllDecimal = fmt?.isAllDecimal || false
      if (!fmt?.maxDecimal || fmt.maxDecimal < 0) fmt.maxDecimal = 0
      if (!fmt?.nDecimal || fmt.nDecimal < 0) fmt.nDecimal = 0
      if (fmt.nDecimal > fmt.maxDecimal) fmt.nDecimal = fmt.maxDecimal

      if (m < fmt.maxDecimal) m = fmt.maxDecimal
    }
  }
  if (n <= 0) opts.isByKey = false // there are no format items
  if (opts.isByKey) opts.maxDecimal = m

  // update all decimals, more, less options based on each item format
  const updateAllMoreLess = () => {
    let isAll = true
    let isAny = false

    for (const fmtKey in opts.itemsFormat) {
      isAll = isAll && opts.itemsFormat[fmtKey].isAllDecimal
      isAny = isAny || opts.itemsFormat[fmtKey].isAllDecimal || opts.itemsFormat[fmtKey].nDecimal > 0
    }
    if (opts.isByKey) opts.isAllDecimal = isAll

    opts.isDoMore = !opts.isRawValue && (opts.nDecimal < opts.maxDecimal || !opts.isAllDecimal)
    opts.isDoLess = !opts.isRawValue && (opts.nDecimal > 0 || opts.isAllDecimal || isAny)
  }
  updateAllMoreLess()

  // save default format options
  const defaultOpts = {
    nDecimal: opts.nDecimal,
    maxDecimal: opts.maxDecimal,
    isRawValue: opts.isRawValue,
    isAllDecimal: opts.isAllDecimal,
    isDoMore: opts.isDoMore,
    isDoLess: opts.isDoLess,
    isByKey: opts.isByKey,
    itemsFormat: {}
  }
  for (const fmtKey in opts.itemsFormat) {
    defaultOpts.itemsFormat[fmtKey] = Object.assign({}, opts.itemsFormat[fmtKey])
  }

  return {
    byKey: (isByKey) => { opts.isByKey = isByKey }, // set or clear format by key option

    format: (val, fmtKey) => {
      if (typeof fmtKey === typeof void 0 || !opts.isByKey || typeof opts?.itemsFormat[fmtKey] === typeof void 0) {
        return floatFmt.format(val)
      }
      const fo = Object.assign({}, opts, opts?.itemsFormat[fmtKey])

      return formatNumber.format(val, fo)
    },

    parse: (s) => {
      return floatFmt.parse(s)
    },

    isValid: (s, fmtKey) => {
      if (typeof fmtKey === typeof void 0 || !opts.isByKey || typeof opts?.itemsFormat[fmtKey] === typeof void 0) {
        return floatFmt.isValid(s)
      }
      if (s === '' || s === void 0) {
        const isNable = opts?.itemsFormat[fmtKey]?.isNullable
        return (typeof isNable !== typeof void 0) ? isNable : opts.isNullable
      }
      return floatFmt.isValid(s)
    },

    options: () => opts,

    resetOptions: () => {
      Object.assign(opts, defaultOpts)
      opts.itemsFormat = {}
      for (const fmtKey in defaultOpts.itemsFormat) {
        opts.itemsFormat[fmtKey] = Object.assign({}, defaultOpts.itemsFormat[fmtKey])
      }
    },

    doRawValue: () => {
      opts.isRawValue = !opts.isRawValue
      updateAllMoreLess()
    },

    doMore: () => {
      if (!opts.isDoMore) return

      if (opts.nDecimal < opts.maxDecimal) {
        opts.nDecimal++
      } else {
        opts.isAllDecimal = true
      }

      for (const fmtKey in opts.itemsFormat) {
        const fo = opts.itemsFormat[fmtKey]
        if (fo.nDecimal < opts.nDecimal) {
          // use the same decimals for all values to avod more less clicks without table body changes
          fo.nDecimal = opts.nDecimal
        } else {
          fo.isAllDecimal = true // that item format is at all decimals now
        }
      }
      updateAllMoreLess()
    },

    doLess: () => {
      if (!opts.isDoLess) return

      if (!opts.isAllDecimal && opts.nDecimal > 0) {
        opts.nDecimal--
      }
      opts.isAllDecimal = false

      for (const fmtKey in opts.itemsFormat) {
        const fo = opts.itemsFormat[fmtKey]

        if (fo.nDecimal > opts.nDecimal) {
          // use the same decimals for all values to avod more less clicks without table body changes
          fo.nDecimal = opts.nDecimal
        }
        fo.isAllDecimal = false
      }
      updateAllMoreLess()
    }
  }
}
/* eslint-enable no-multi-spaces */
