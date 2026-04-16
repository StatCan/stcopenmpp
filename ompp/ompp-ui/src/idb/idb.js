// IndexedDB wrapper to store model views and other UI settings
//
export const OM_IDB_NAME = 'openmpp_ui'

export const connection = async (dbName = OM_IDB_NAME) => {
  let theDb

  // set common handlers for db open events and return open request
  const dbOpenRequest = (resolve, reject, dbVer) => {
    const openRq = (dbVer && dbVer > 1) ? indexedDB.open(dbName, dbVer) : indexedDB.open(dbName) // do open

    openRq.onsuccess = () => {
      theDb = openRq.result
      theDb.onversionchange = (e) => theDb.close() // close on version change to avoid blocking
      resolve(theDb)
    }
    openRq.onerror = () => {
      console.warn('Error at indexedDB open:', dbName, dbVer, openRq.error)
      reject(openRq.error)
    }
    openRq.onblocked = (e) => {
      console.warn('Blocked indexedDB:', dbName, dbVer, e)
    }

    return openRq
  }

  // open existing or create store and return db open request
  const createStore = async (storeName) => {
    if (theDb.objectStoreNames.contains(storeName)) {
      return theDb // store already exist
    }
    // else create new store
    const dbVer = theDb.version + 1

    return new Promise((resolve, reject) => {
      const openRq = dbOpenRequest(resolve, reject, dbVer)
      openRq.onupgradeneeded = () => {
        const db = openRq.result
        if (!db.objectStoreNames.contains(storeName)) db.createObjectStore(storeName)
      }
    })
  }

  // delete store if exist and return db open request
  const doDeleteStore = async (storeName) => {
    if (!theDb.objectStoreNames.contains(storeName)) {
      return theDb // no such store in database
    }
    // else delete the store
    const dbVer = theDb.version + 1

    return new Promise((resolve, reject) => {
      const openRq = dbOpenRequest(resolve, reject, dbVer)
      openRq.onupgradeneeded = () => {
        const db = openRq.result
        if (db.objectStoreNames.contains(storeName)) db.deleteObjectStore(storeName)
      }
    })
  }

  // open existing or create new indexed db
  const open = async () => {
    return new Promise((resolve, reject) => dbOpenRequest(resolve, reject))
  }
  await open()

  return {
    db: () => theDb,
    storeNames: () => (theDb && theDb instanceof IDBDatabase) ? Array.from(theDb.objectStoreNames) : [],

    // start read only transaction and return the store
    openReadOnly: async (storeName) => {
      await createStore(storeName)
      const theStore = theDb.transaction(storeName, 'readonly').objectStore(storeName)

      return {
        store: () => theStore,
        getByKey: async (key) => getByKey(theStore, key),
        getKey: async (key) => getKey(theStore, key),
        getAll: async () => getAll(theStore),
        getAllKeys: async () => getAllKeys(theStore)
      }
    },

    // start read-write transaction and return the store
    openReadWrite: async (storeName) => {
      await createStore(storeName)
      const theStore = theDb.transaction(storeName, 'readwrite').objectStore(storeName)

      return {
        store: () => theStore,
        getByKey: async (key) => getByKey(theStore, key),
        getKey: async (key) => getKey(theStore, key),
        getAll: async () => getAll(theStore),
        getAllKeys: async () => getAllKeys(theStore),
        put: async (key, value) => put(theStore, key, value),
        remove: async (key) => remove(theStore, key)
      }
    },

    // delete store if exist and return new instance of database
    deleteStore: async (storeName) => doDeleteStore(storeName)
  }
}

// get value by key
const getByKey = async (store, key) => {
  if (!key) return false // return empty value if key is empty

  return new Promise(resolve => {
    const rq = store.get(key)
    rq.onsuccess = () => resolve(rq.result)
  })
}

// get key from store: select if exist
const getKey = async (store, key) => {
  if (!key) return false // return empty value if key is empty

  return new Promise(resolve => {
    const rq = store.getKey(key)
    rq.onsuccess = () => resolve(rq.result)
  })
}

// return array of all values from store
const getAll = async (store) => {
  return new Promise(resolve => {
    const rq = store.getAll()
    rq.onsuccess = () => resolve(rq.result)
  })
}

// return array of all keys from store
const getAllKeys = async (store) => {
  return new Promise(resolve => {
    const rq = store.getAllKeys()
    rq.onsuccess = () => resolve(rq.result)
  })
}

// insert or update existing value by key
const put = async (store, key, val) => {
  if (!key || !val) return false // empty return if key or value is empty

  return new Promise(resolve => {
    const rq = store.put(val, key)
    rq.onsuccess = () => resolve(rq.result)
  })
}

// delete by key, return is undefined
const remove = async (store, key) => {
  if (!key) return false // empty return if key is empty

  return new Promise(resolve => {
    const rq = store.delete(key)
    rq.onsuccess = () => resolve(rq.result)
  })
}
