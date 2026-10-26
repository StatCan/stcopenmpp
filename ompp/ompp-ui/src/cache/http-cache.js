// Cache of oms web-service responses, implemented as axios adapter.
//
// Three kinds of requests are cached:
//
//   1. model metadata: GET model text, word list and language list.
//      Model digest is a hash of model metadata, so content is immutable: served from cache first.
//
//   2. data of completed model runs: POST read of parameter, output table and microdata pages from a run.
//      Data of completed run never change: served from cache first,
//      but only if run (and all runs to compare) are completed: success, error or exit status.
//
//   3. lists and texts: GET model list, run list, workset list, run text, workset text, server config, user views.
//      These can change at any time: always requested from the server,
//      cached copy is used only if server is not reachable (offline fallback).
//      Input scenario parameter pages (editable) also use offline fallback only.
//
// Cached run data is deleted when run is deleted, model cached data deleted when model deleted.
// Total size of cache is limited, least recently used responses are evicted first.

import axios, { AxiosHeaders } from 'axios'
import * as Mdf from 'src/model-common'
import { useModelStore } from 'src/stores/model'
import * as Cdb from './cache-db'
import { getCacheSettings } from './cache-settings'

const DAY_MS = 24 * 60 * 60 * 1000
const IMMUTABLE_TTL_MS = 30 * DAY_MS // time to keep model metadata and completed run data
const FALLBACK_TTL_MS = 30 * DAY_MS // time to keep offline fallback copy of lists and texts
const MAX_ENTRY_CHARS = 32 * 1024 * 1024 // do not cache responses larger than that
const OFFLINE_NOTIFY_MS = 30 * 1000 // minimal interval between "offline: cached data" notifications

export const CACHE_HEADER = 'x-ompp-cache' // response header to mark response from cache: 'hit' or 'offline'

const KIND_IMMUTABLE = 1 // cache first: model metadata
const KIND_RUN_DATA = 2 // cache first if all runs completed: run parameter, table, microdata
const KIND_FALLBACK = 3 // network first, cache used only if server offline

// rules to select cacheable requests by method and url path
const rules = [
  // model metadata: text, words, languages
  { kind: KIND_IMMUTABLE, method: 'get', rx: /\/api\/model\/([^/]+)\/(?:pack\/text|text|text\/all|word-list|lang-list)(?:\/lang\/[^/]+)?$/ },

  // run data pages: parameter, output table and microdata
  { kind: KIND_RUN_DATA, method: 'post', rx: /\/api\/model\/([^/]+)\/run\/([^/]+)\/(?:parameter\/value-id|table\/(?:value-id|calc-id|compare-id)|microdata\/(?:value-id|calc-id|compare-id))$/ },

  // lists and texts: offline fallback
  { kind: KIND_FALLBACK, method: 'get', rx: /\/api\/model-list\/text(?:\/lang\/[^/]+)?$/ },
  { kind: KIND_FALLBACK, method: 'get', rx: /\/api\/model\/([^/]+)\/(?:run-list|workset-list)\/text(?:\/lang\/[^/]+)?$/ },
  { kind: KIND_FALLBACK, method: 'get', rx: /\/api\/model\/([^/]+)\/run\/([^/]+)\/text(?:\/lang\/[^/]+)?$/ },
  { kind: KIND_FALLBACK, method: 'get', rx: /\/api\/model\/([^/]+)\/workset\/([^/]+)\/text(?:\/lang\/[^/]+)?$/ },
  { kind: KIND_FALLBACK, method: 'get', rx: /\/api\/service\/config$/ },
  { kind: KIND_FALLBACK, method: 'get', rx: /\/api\/user\/view\/model\/[^/]+$/ },
  { kind: KIND_FALLBACK, method: 'post', rx: /\/api\/model\/([^/]+)\/workset\/([^/]+)\/parameter\/value-id$/ }
]

let baseAdapter = null
let metaMap = null // map of cached response key => { size, ts, used, tags }
let metaPromise = null
let lastOfflineNotify = 0
let stats = { hit: 0, offline: 0, miss: 0 }

// install caching adapter into global axios instance and (optional) other axios instances
export const installHttpCache = (...instances) => {
  baseAdapter = axios.getAdapter(axios.defaults.adapter)
  axios.defaults.adapter = cacheAdapter
  for (const ai of instances) {
    if (ai?.defaults) ai.defaults.adapter = cacheAdapter
  }
}

// return url object for request config
const urlOf = (config) => {
  try {
    const base = (typeof window !== 'undefined' && window.location) ? window.location.href : 'http://localhost/'
    const u = (config?.baseURL && !/^[a-z]+:\/\//i.test(config.url || '')) ? (config.baseURL.replace(/\/+$/, '') + '/' + (config.url || '').replace(/^\/+/, '')) : (config.url || '')
    return new URL(u, base)
  } catch (e) {
    return null
  }
}

// find cache rule for request: return { rule, match } or null
const findRule = (method, path) => {
  for (const r of rules) {
    if (r.method !== method) continue
    const m = r.rx.exec(path)
    if (m) return { rule: r, match: m }
  }
  return null
}

// body of request as string
const bodyOf = (config) => {
  const d = config?.data
  if (d === undefined || d === null) return ''
  if (typeof d === typeof 'string') return d
  try {
    return JSON.stringify(d)
  } catch (e) {
    return ''
  }
}

// decode url path segment
const decodePart = (s) => {
  try {
    return decodeURIComponent(s || '')
  } catch (e) {
    return s || ''
  }
}

// return list of run digests from request: run from url path and run digests from Runs array of request body
const runsOfRequest = (match, body) => {
  const runs = [decodePart(match[2])]
  try {
    const b = body ? JSON.parse(body) : null
    if (Array.isArray(b?.Runs)) {
      for (const r of b.Runs) {
        if (typeof r !== typeof 'string') return null // unknown kind of run reference: not cacheable
        if (r && !runs.includes(r)) runs.push(r)
      }
    }
  } catch (e) {
    return null
  }
  return runs
}

// return true if all runs are completed: success, error or exit
const isAllRunsCompleted = (modelDigest, runs) => {
  if (!Array.isArray(runs) || runs.length <= 0) return false
  try {
    const ms = useModelStore()
    for (const rd of runs) {
      const rt = ms.runTextByDigest({ ModelDigest: modelDigest, RunDigest: rd })
      if (!Mdf.isNotEmptyRunText(rt) || !Mdf.isRunCompleted(rt)) return false
    }
    return true
  } catch (e) {
    return false
  }
}

// make cache tags for invalidation: model digest and run digests
const tagsOf = (modelDigest, runs) => {
  const t = []
  if (modelDigest) t.push('m:' + modelDigest)
  if (Array.isArray(runs)) {
    for (const r of runs) {
      if (r) t.push('r:' + modelDigest + ':' + r)
    }
  }
  return t
}

// load metadata of cached responses
const loadMeta = async () => {
  if (metaMap) return metaMap
  if (!metaPromise) {
    metaPromise = (async () => {
      const m = new Map()
      const all = await Cdb.getAllEntries(Cdb.META_STORE)
      for (const [k, v] of all) {
        if (v && typeof v === 'object') m.set(k, v)
      }
      metaMap = m
      return m
    })()
  }
  return metaPromise
}

// evict least recently used responses if cache size limit exceeded
const evictIfNeeded = async () => {
  const m = await loadMeta()
  const maxChars = (getCacheSettings().maxMb || 256) * 1024 * 1024 / 2 // two bytes per char estimate

  let total = 0
  for (const v of m.values()) total += (v.size || 0)
  if (total <= maxChars) return

  const target = maxChars * 0.8
  const lst = Array.from(m.entries()).sort((a, b) => (a[1].used || 0) - (b[1].used || 0))
  const del = []
  for (const [k, v] of lst) {
    if (total <= target) break
    total -= (v.size || 0)
    del.push(k)
    m.delete(k)
  }
  await Cdb.deleteHttpEntries(del)
}

// store response in cache
const storeResponse = async (key, kind, tags, response) => {
  let data = response?.data
  if (data === undefined || data === null) return
  if (typeof data !== typeof 'string') {
    try {
      data = JSON.stringify(data)
    } catch (e) {
      return
    }
  }
  if (data.length > MAX_ENTRY_CHARS) return

  let ct = ''
  try {
    ct = AxiosHeaders.from(response.headers || {}).get('content-type') || ''
  } catch (e) {}

  const now = Date.now()
  const meta = { size: data.length, ts: now, used: now, kind, tags }
  const m = await loadMeta()
  m.set(key, meta)
  await Cdb.putHttpEntry(key, { data, status: response.status || 200, contentType: ct }, meta)
  await evictIfNeeded()
}

// read response from cache, return undefined if not found or expired
const readResponse = async (key, ttlMs) => {
  const m = await loadMeta()
  const meta = m.get(key)
  if (!meta) return undefined
  if (ttlMs && Date.now() - (meta.ts || 0) > ttlMs) return undefined

  const e = await Cdb.getValue(Cdb.HTTP_STORE, key)
  if (!e || typeof e.data !== typeof 'string') {
    m.delete(key)
    return undefined
  }
  // update last used time, do not wait for completion
  meta.used = Date.now()
  Cdb.putValue(Cdb.META_STORE, key, meta)
  return e
}

// make axios response from cached entry
const cachedResponse = (config, e, how) => {
  const h = new AxiosHeaders()
  if (e.contentType) h.set('content-type', e.contentType)
  h.set(CACHE_HEADER, how)
  return {
    data: e.data,
    status: e.status || 200,
    statusText: 'OK',
    headers: h,
    config,
    request: null
  }
}

// return true if error is a network error: server not reachable
const isNetworkError = (err) => {
  if (!err || err.response) return false
  if (axios.isCancel(err) || err.code === 'ERR_CANCELED') return false
  return true
}

// notify user what server is offline and cached data displayed
const notifyOffline = () => {
  const now = Date.now()
  if (now - lastOfflineNotify < OFFLINE_NOTIFY_MS) return
  lastOfflineNotify = now
  try {
    window.dispatchEvent(new CustomEvent(OFFLINE_EVENT))
  } catch (e) {}
}

export const OFFLINE_EVENT = 'ompp-cache-offline' // window event: server offline, cached response used

// delete cached responses where any of tags match
export const invalidateByTags = async (tagLst) => {
  if (!Array.isArray(tagLst) || tagLst.length <= 0) return
  const m = await loadMeta()
  const del = []
  for (const [k, v] of m.entries()) {
    if (Array.isArray(v?.tags) && v.tags.some(t => tagLst.includes(t))) del.push(k)
  }
  for (const k of del) m.delete(k)
  await Cdb.deleteHttpEntries(del)
}

// after successful update request delete related cached responses: deleted runs or deleted model
const invalidateOnUpdate = async (method, path, body) => {
  let m = /\/api\/admin\/model\/([^/]+)\/delete$/.exec(path)
  if (m) {
    await invalidateByTags(['m:' + decodePart(m[1])])
    return
  }
  m = /\/api\/model\/([^/]+)\/delete-runs$/.exec(path)
  if (m) {
    const md = decodePart(m[1])
    let lst = []
    try {
      lst = JSON.parse(body || '[]')
    } catch (e) {}
    if (Array.isArray(lst)) await invalidateByTags(lst.filter(r => typeof r === typeof 'string').map(r => 'r:' + md + ':' + r))
    return
  }
  m = /\/api\/model\/([^/]+)\/run\/([^/]+)(?:\/|$)/.exec(path)
  if (m && method === 'delete') {
    await invalidateByTags(['r:' + decodePart(m[1]) + ':' + decodePart(m[2])])
  }
}

// axios adapter with cache
const cacheAdapter = async (config) => {
  const method = (config?.method || 'get').toLowerCase()
  const cfg = getCacheSettings()

  if (!cfg.isHttpCache || !baseAdapter) return baseAdapter(config)

  const url = urlOf(config)
  if (!url) return baseAdapter(config)

  const path = url.pathname
  const body = bodyOf(config)

  // not a cacheable request: if this is update request then delete related cached responses
  const rm = findRule(method, path)
  if (!rm) {
    if (method === 'get' || method === 'head') return baseAdapter(config)

    const rsp = await baseAdapter(config)
    try {
      await invalidateOnUpdate(method, path, body)
    } catch (e) {
      console.warn('UI cache: invalidate error', e)
    }
    return rsp
  }

  const key = method + ' ' + url.host + path + url.search + (body ? ' ' + body : '')
  const modelDigest = decodePart(rm.match[1])

  // cache first: model metadata and data of completed runs
  if (rm.rule.kind === KIND_IMMUTABLE || rm.rule.kind === KIND_RUN_DATA) {
    let runs = []
    if (rm.rule.kind === KIND_RUN_DATA) {
      runs = runsOfRequest(rm.match, body)
      if (!runs || !isAllRunsCompleted(modelDigest, runs)) {
        stats.miss++
        return baseAdapter(config) // run in progress or unknown: do not cache
      }
    }
    try {
      const e = await readResponse(key, IMMUTABLE_TTL_MS)
      if (e) {
        stats.hit++
        return cachedResponse(config, e, 'hit')
      }
    } catch (e) {
      console.warn('UI cache: read error', e)
    }
    stats.miss++
    const rsp = await baseAdapter(config)
    if (rsp && rsp.status >= 200 && rsp.status < 300) {
      storeResponse(key, rm.rule.kind, tagsOf(modelDigest, runs), rsp).catch((e) => console.warn('UI cache: write error', e))
    }
    return rsp
  }

  // network first, use cache if server offline
  let rsp
  try {
    rsp = await baseAdapter(config)
  } catch (err) {
    if (!isNetworkError(err)) throw err

    let e
    try {
      e = await readResponse(key, FALLBACK_TTL_MS)
    } catch (ex) {}
    if (!e) throw err

    stats.offline++
    notifyOffline()
    return cachedResponse(config, e, 'offline')
  }
  if (rsp && rsp.status >= 200 && rsp.status < 300) {
    const runs = rm.match[2] && /\/run\//.test(path) ? [decodePart(rm.match[2])] : []
    storeResponse(key, KIND_FALLBACK, tagsOf(modelDigest, runs), rsp).catch((e) => console.warn('UI cache: write error', e))
  }
  return rsp
}

// return cache statistics: number and size of cached responses, number of hits in current session
export const httpCacheStats = async () => {
  const m = await loadMeta()
  let size = 0
  for (const v of m.values()) size += (v.size || 0)
  return { count: m.size, size: size * 2, hit: stats.hit, offline: stats.offline, miss: stats.miss }
}

// delete all cached responses
export const clearHttpCache = async () => {
  const m = await loadMeta()
  m.clear()
  stats = { hit: 0, offline: 0, miss: 0 }
  await Cdb.clearHttp()
}
