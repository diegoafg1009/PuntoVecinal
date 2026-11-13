const SHARED_FLAG = 'pv:location-shared'

export interface NeighborLocation {
  latitude: number
  longitude: number
}

// Kept only in memory and shared by every user of the composable.
const location = ref<NeighborLocation | null>(null)
const locating = ref(false)
let restoreTried = false

// 4 decimals are about 11 m: enough for «120 m» and less precise than the device position.
function round(value: number) {
  return Math.round(value * 10_000) / 10_000
}

function readFlag() {
  try {
    return localStorage.getItem(SHARED_FLAG) === '1'
  } catch {
    return false
  }
}

function writeFlag(on: boolean) {
  try {
    if (on) localStorage.setItem(SHARED_FLAG, '1')
    else localStorage.removeItem(SHARED_FLAG)
  } catch {
    // Without storage the location is simply asked again on the next visit.
  }
}

function getPosition() {
  return new Promise<NeighborLocation>((resolve, reject) => {
    if (!('geolocation' in navigator)) {
      reject(new Error('Geolocation unavailable'))
      return
    }
    navigator.geolocation.getCurrentPosition(
      position => resolve({ latitude: round(position.coords.latitude), longitude: round(position.coords.longitude) }),
      reject,
      { enableHighAccuracy: false, timeout: 10_000, maximumAge: 60_000 }
    )
  })
}

/** The neighbor's location for distances. The device is asked only from `share()`, a user action. */
export function useNeighborLocation() {
  const toast = useToast()

  async function acquire() {
    locating.value = true
    try {
      location.value = await getPosition()
      writeFlag(true)
      return true
    } catch {
      location.value = null
      writeFlag(false)
      return false
    } finally {
      locating.value = false
    }
  }

  /**
   * Called from a button. Returns whether the location was obtained; if not, a toast says so,
   * with `failureMessage` when the screen needs its own wording.
   */
  async function share(failureMessage = 'No pudimos obtener tu ubicación. Los reportes se muestran sin distancia.') {
    if (await acquire()) return true
    toast.add({ title: failureMessage, color: 'error' })
    return false
  }

  /** On a later visit: reuse the location silently, only if the browser already granted it. */
  async function restore() {
    if (restoreTried || !import.meta.client) return
    restoreTried = true
    if (!readFlag()) return
    try {
      const status = await navigator.permissions.query({ name: 'geolocation' })
      if (status.state === 'granted') await acquire()
      else writeFlag(false)
    } catch {
      // No Permissions API: the neighbor taps the button again.
    }
  }

  return { location: readonly(location), locating: readonly(locating), share, restore }
}
