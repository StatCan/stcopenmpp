/**
 * Code below is a slightly modified version of:
 *
 * Copyright (c) 2017 - 2018 - Yev Vlasenko
 * MIT License
 * https://github.com/euvl/v-clipboard
 */
const cssText = 'position:fixed;pointer-events:none;z-index:-9999;opacity:0;'
const copyErrorMessage = 'Failed to copy value to clipboard. Unknown type.'

export const toClipboard = input => {
  if (input === void 0 || input === null) return true // nothing to copy

  let value
  if (typeof input === typeof 'string') {
    value = input
  } else {
    try {
      value = JSON.stringify(input)
    } catch (e) {
      console.warn(copyErrorMessage)
      return false
    }
  }

  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.setAttribute('readonly', '')
  textarea.style.cssText = cssText

  document.body.appendChild(textarea)

  if (navigator.userAgent.match(/ipad|ipod|iphone/i)) {
    textarea.contentEditable = true
    textarea.readOnly = true

    const range = document.createRange()

    range.selectNodeContents(textarea)

    const selection = window.getSelection()

    selection.removeAllRanges()
    selection.addRange(range)
    textarea.setSelectionRange(0, 999999)
  } else {
    textarea.select()
  }

  let success = false
  try {
    success = document.execCommand('copy')
  } catch (err) {
    console.warn(err)
  }

  document.body.removeChild(textarea)
  return success
}
