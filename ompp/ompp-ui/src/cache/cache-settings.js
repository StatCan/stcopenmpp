// UI cache settings, stored in browser localStorage so they are available synchronously at startup.

const SETTINGS_KEY = 'openmpp-ui-cache-settings'

export const DEFAULT_MAX_MB = 256 // default size limit of cached server responses

const defaultSettings = () => ({
  isStateCache: true, // remember UI state: selected model and runs, open tabs, views, filters, form inputs
  isHttpCache: true, // cache server responses: model metadata, completed run data, offline fallback
  maxMb: DEFAULT_MAX_MB
})

let current = null

// return current cache settings
export const getCacheSettings = () => {
  if (current) return current

  current = defaultSettings()
  try {
    const s = JSON.parse(localStorage.getItem(SETTINGS_KEY) || '{}')
    if (typeof s?.isStateCache === typeof true) current.isStateCache = s.isStateCache
    if (typeof s?.isHttpCache === typeof true) current.isHttpCache = s.isHttpCache
    if (typeof s?.maxMb === typeof 1 && s.maxMb >= 16) current.maxMb = s.maxMb
  } catch (e) {
    // localStorage not available or invalid content: use defaults
  }
  return current
}

// update cache settings, only properties present in (s) are updated
export const setCacheSettings = (s) => {
  const c = getCacheSettings()
  if (typeof s?.isStateCache === typeof true) c.isStateCache = s.isStateCache
  if (typeof s?.isHttpCache === typeof true) c.isHttpCache = s.isHttpCache
  if (typeof s?.maxMb === typeof 1 && s.maxMb >= 16) c.maxMb = s.maxMb
  try {
    localStorage.setItem(SETTINGS_KEY, JSON.stringify(c))
  } catch (e) {
    // localStorage not available
  }
  return c
}
