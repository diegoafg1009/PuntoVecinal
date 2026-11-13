const DB_NAME = 'pv'
const STORE_NAME = 'report-photos'

export interface StoredPhoto {
  id: string
  blob: Blob
}

function open(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, 1)
    request.onupgradeneeded = () => request.result.createObjectStore(STORE_NAME, { keyPath: 'id' })
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
    request.onblocked = () => reject(new Error('IndexedDB blocked'))
  })
}

/** Runs one request in a transaction; resolves with its result once the transaction completes. */
async function run<T>(mode: IDBTransactionMode, action: (store: IDBObjectStore) => IDBRequest<T>): Promise<T> {
  const db = await open()
  try {
    return await new Promise<T>((resolve, reject) => {
      const transaction = db.transaction(STORE_NAME, mode)
      const request = action(transaction.objectStore(STORE_NAME))
      transaction.oncomplete = () => resolve(request.result)
      transaction.onerror = () => reject(transaction.error)
      transaction.onabort = () => reject(transaction.error)
    })
  } finally {
    db.close()
  }
}

// Every access swallows its failure: when IndexedDB is blocked or full the photos live only
// in memory and the form keeps working, it just cannot restore them later.

export async function storePhoto(photo: StoredPhoto): Promise<void> {
  try {
    await run('readwrite', store => store.put(photo))
  } catch {
    // See above.
  }
}

export async function deleteStoredPhoto(id: string): Promise<void> {
  try {
    await run('readwrite', store => store.delete(id))
  } catch {
    // See above.
  }
}

export async function loadStoredPhotos(): Promise<StoredPhoto[]> {
  try {
    return await run('readonly', store => store.getAll() as IDBRequest<StoredPhoto[]>)
  } catch {
    return []
  }
}

export async function clearStoredPhotos(): Promise<void> {
  try {
    await run('readwrite', store => store.clear())
  } catch {
    // See above.
  }
}
