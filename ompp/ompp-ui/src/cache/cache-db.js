// IndexedDB storage for UI cache: persisted UI state and cached oms server responses.
//
// Uses its own database (separate from 'openmpp_ui' user views database) with a fixed schema,
// so it never competes with the dynamic version upgrades of the user views database.
//
// Every function here is best effort: if IndexedDB is not available (private browsing, blocked storage)
// then reads return undefined and writes are ignored, so the UI keeps working without a cache.

const DB_NAME = 'openmpp_ui_cache'
const DB_VERSION = 1

export const STATE_STORE = 'state' // persisted UI state: key => value
export const HTTP_STORE = 'http' // cached server responses: key => { data, status, headers }
export const META_STORE = 'http-meta' // small metadata of cached responses: key => { size, ts, used, tags }

let dbPromise = null

// open database (once) and return promise of IDBDatabase or null if indexedDB not available
export const openDb = () => {
  if (dbPromise) return dbPromise

  dbPromise = new Promise((resolve) => {
    try {
      if (typeof indexedDB === 'undefined' || !indexedDB) {
        resolve(null)
        return
      }
      const rq = indexedDB.open(DB_NAME, DB_VERSION)

      rq.onupgradeneeded = () => {
        const db = rq.result
        for (const s of [STATE_STORE, HTTP_STORE, META_STORE]) {
          if (!db.objectStoreNames.contains(s)) db.createObjectStore(s)
        }
      }
      rq.onsuccess = () => {
        const db = rq.result
        db.onversionchange = () => {
          db.close()
          dbPromise = null
        }
        resolve(db)
      }
      rq.onerror = () => {
        console.warn('UI cache: unable to open indexedDB', rq.error)
        resolve(null)
      }
      rq.onblocked = () => {
        console.warn('UI cache: indexedDB open blocked')
      }
    } catch (e) {
      console.warn('UI cache: indexedDB not available', e)
      resolve(null)
    }
  })
  return dbPromise
}

// run request inside of transaction and return promise of request result, undefined on error
const doRequest = async (storeNames, mode, fn) => {
  const db = await openDb()
  if (!db) return undefined

  return new Promise((resolve) => {
    try {
      const tx = db.transaction(storeNames, mode)
      let result
      const stores = Array.isArray(storeNames) ? storeNames.map(s => tx.objectStore(s)) : [tx.objectStore(storeNames)]
      const rq = fn(...stores)
      if (rq) rq.onsuccess = () => { result = rq.result }

      tx.oncomplete = () => resolve(result)
      tx.onerror = () => {
        console.warn('UI cache: indexedDB transaction error', tx.error)
        resolve(undefined)
      }
      tx.onabort = () => {
        console.warn('UI cache: indexedDB transaction aborted', tx.error)
        resolve(undefined)
      }
    } catch (e) {
      console.warn('UI cache: indexedDB error', e)
      resolve(undefined)
    }
  })
}

// get value by key from the store
export const getValue = async (storeName, key) => doRequest(storeName, 'readonly', (st) => st.get(key))

// insert or replace value by key
export const putValue = async (storeName, key, value) => doRequest(storeName, 'readwrite', (st) => st.put(value, key))

// delete value by key
export const deleteValue = async (storeName, key) => doRequest(storeName, 'readwrite', (st) => st.delete(key))

// delete all values from the store
export const clearStore = async (storeName) => doRequest(storeName, 'readwrite', (st) => st.clear())

// return all keys and values of the store as array of [key, value]
export const getAllEntries = async (storeName) => {
  const db = await openDb()
  if (!db) return []

  return new Promise((resolve) => {
    const res = []
    try {
      const tx = db.transaction(storeName, 'readonly')
      const rq = tx.objectStore(storeName).openCursor()
      rq.onsuccess = () => {
        const c = rq.result
        if (c) {
          res.push([c.key, c.value])
          c.continue()
        }
      }
      tx.oncomplete = () => resolve(res)
      tx.onerror = () => resolve(res)
      tx.onabort = () => resolve(res)
    } catch (e) {
      console.warn('UI cache: indexedDB error', e)
      resolve(res)
    }
  })
}

// put cached response and its metadata in single transaction
export const putHttpEntry = async (key, entry, meta) => doRequest([HTTP_STORE, META_STORE], 'readwrite', (hs, ms) => {
  hs.put(entry, key)
  return ms.put(meta, key)
})

// delete cached responses and metadata by array of keys
export const deleteHttpEntries = async (keys) => {
  if (!Array.isArray(keys) || keys.length <= 0) return
  return doRequest([HTTP_STORE, META_STORE], 'readwrite', (hs, ms) => {
    for (const k of keys) {
      hs.delete(k)
      ms.delete(k)
    }
    return null
  })
}

// delete all cached responses
export const clearHttp = async () => doRequest([HTTP_STORE, META_STORE], 'readwrite', (hs, ms) => {
  hs.clear()
  return ms.clear()
})
