const STORAGE_KEY = 'pv:report-draft'

export const TITLE_MAX_LENGTH = 100
export const DETAILS_MAX_LENGTH = 1000
export const REFERENCE_MAX_LENGTH = 150
export const PHOTO_MAX_COUNT = 3
export const STEP_COUNT = 3

export interface Step1Errors {
  category?: string
  title?: string
}

export interface Step2Errors {
  photos?: string
  location?: string
}

export interface DraftLocation {
  lat: number
  lng: number
}

export interface DraftPhoto {
  id: string
  blob: Blob
  /** Object URL for the thumbnail; revoked when the photo is removed. */
  url: string
}

export interface AddPhotosResult {
  /** At least one file could not be read as an image. */
  failed: boolean
  /** Some files were left out because the report already has the maximum of photos. */
  overLimit: boolean
}

interface StoredDraft {
  category: string | null
  title: string
  details: string
  step: number
  location: DraftLocation | null
  reference: string
  photoIds: string[]
}

function isCoordinate(value: unknown, limit: number): value is number {
  return typeof value === 'number' && Number.isFinite(value) && Math.abs(value) <= limit
}

function roundCoordinate(value: number): number {
  return Math.round(value * 1e6) / 1e6
}

/** Reads the stored draft, ignoring anything unknown or malformed. */
function readStored(): Partial<StoredDraft> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const data: unknown = JSON.parse(raw)
    if (typeof data !== 'object' || data === null) return {}
    const { category, title, details, step, location, reference, photoIds } = data as Record<string, unknown>
    const place = typeof location === 'object' && location !== null ? location as Record<string, unknown> : null
    return {
      ...(typeof category === 'string' && { category }),
      ...(typeof title === 'string' && { title: title.slice(0, TITLE_MAX_LENGTH) }),
      ...(typeof details === 'string' && { details: details.slice(0, DETAILS_MAX_LENGTH) }),
      ...(Number.isInteger(step) && (step as number) >= 1 && (step as number) <= STEP_COUNT && { step: step as number }),
      ...(place && isCoordinate(place.lat, 90) && isCoordinate(place.lng, 180) && { location: { lat: place.lat, lng: place.lng } }),
      ...(typeof reference === 'string' && { reference: reference.slice(0, REFERENCE_MAX_LENGTH) }),
      ...(Array.isArray(photoIds) && { photoIds: photoIds.filter((id): id is string => typeof id === 'string').slice(0, PHOTO_MAX_COUNT) })
    }
  } catch {
    return {}
  }
}

function writeStored(draft: StoredDraft) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(draft))
  } catch {
    // Storage is blocked or full: the form keeps working, it just cannot be restored later.
  }
}

export const useReportDraftStore = defineStore('reportDraft', () => {
  const stored = readStored()

  const category = ref<string | null>(stored.category ?? null)
  const title = ref(stored.title ?? '')
  const details = ref(stored.details ?? '')
  const step = ref(stored.step ?? 1)
  const location = ref<DraftLocation | null>(stored.location ?? null)
  const reference = ref(stored.reference ?? '')
  const photos = ref<DraftPhoto[]>([])
  /** False until the stored photos have been read back from IndexedDB. */
  const ready = ref(false)

  const storedPhotoIds = stored.photoIds ?? []

  // Until the photos are back, keep the stored ids so an early write does not forget them.
  watch([category, title, details, step, location, reference, photos, ready], () => {
    writeStored({
      category: category.value,
      title: title.value,
      details: details.value,
      step: step.value,
      location: location.value,
      reference: reference.value,
      photoIds: ready.value ? photos.value.map(photo => photo.id) : storedPhotoIds
    })
  }, { deep: true })

  function toDraftPhoto(id: string, blob: Blob): DraftPhoto {
    return { id, blob, url: URL.createObjectURL(blob) }
  }

  async function restorePhotos() {
    const available = new Map((await loadStoredPhotos()).map(photo => [photo.id, photo.blob]))
    // Anything not in the stored order is an orphan from an earlier session.
    photos.value = storedPhotoIds
      .filter(id => available.has(id))
      .map(id => toDraftPhoto(id, available.get(id)!))
    ready.value = true
  }

  if (import.meta.client) restorePhotos()
  else ready.value = true

  /** Processes the files one at a time and keeps those that fit, up to PHOTO_MAX_COUNT. */
  async function addPhotos(files: File[]): Promise<AddPhotosResult> {
    const result: AddPhotosResult = { failed: false, overLimit: false }
    for (const file of files) {
      if (photos.value.length >= PHOTO_MAX_COUNT) {
        result.overLimit = true
        break
      }
      try {
        const blob = await processPhoto(file)
        const photo = toDraftPhoto(crypto.randomUUID(), blob)
        photos.value.push(photo)
        await storePhoto({ id: photo.id, blob })
      } catch {
        result.failed = true
      }
    }
    return result
  }

  async function removePhoto(id: string) {
    const photo = photos.value.find(item => item.id === id)
    if (!photo) return
    photos.value = photos.value.filter(item => item.id !== id)
    URL.revokeObjectURL(photo.url)
    await deleteStoredPhoto(id)
  }

  function setLocation(position: DraftLocation) {
    location.value = { lat: roundCoordinate(position.lat), lng: roundCoordinate(position.lng) }
  }

  function validateStep1(): Step1Errors {
    const errors: Step1Errors = {}
    if (!category.value) errors.category = 'Elige una categoría.'
    if (!title.value.trim()) errors.title = 'Cuéntanos qué pasa.'
    return errors
  }

  function validateStep2(): Step2Errors {
    const errors: Step2Errors = {}
    if (photos.value.length === 0) errors.photos = 'Agrega al menos una foto.'
    if (!location.value) errors.location = 'Marca el lugar en el mapa.'
    return errors
  }

  /** Clears a category the API no longer offers, keeping the rest of the draft. */
  function pruneCategory(validSlugs: string[]) {
    if (category.value !== null && !validSlugs.includes(category.value)) category.value = null
  }

  /** True when the neighbor wrote or chose something worth keeping or discarding. */
  const hasData = computed(() =>
    category.value !== null
    || title.value.trim() !== ''
    || details.value.trim() !== ''
    || photos.value.length > 0
    || location.value !== null
    || reference.value.trim() !== ''
  )

  /** Empties the draft; the watcher above overwrites the stored copy with the empty one. */
  function reset() {
    category.value = null
    title.value = ''
    details.value = ''
    step.value = 1
    location.value = null
    reference.value = ''
    photos.value.forEach(photo => URL.revokeObjectURL(photo.url))
    photos.value = []
    clearStoredPhotos()
  }

  return {
    category, title, details, step, location, reference, photos, ready,
    hasData, validateStep1, validateStep2, pruneCategory, addPhotos, removePhoto, setLocation, reset
  }
})
