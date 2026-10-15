<script setup lang="ts">
import type { Map as LeafletMap, Marker } from 'leaflet'
import 'leaflet/dist/leaflet.css'

// Assumption: the Cooperativa Primavera area in Comas, to be adjusted with the real coordinates.
const DEFAULT_CENTER = { lat: -11.936, lng: -77.05 }
const DEFAULT_ZOOM = 16
const PLACED_ZOOM = 17

const location = defineModel<DraftLocation | null>('location', { required: true })

const container = useTemplateRef('container')
let map: LeafletMap | undefined
let marker: Marker | undefined

// Lucide map-pin, drawn in the primary color. Leaflet's default PNG icons break under bundlers.
const PIN_HTML = '<svg xmlns="http://www.w3.org/2000/svg" width="36" height="36" viewBox="0 0 24 24" fill="white" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 10c0 4.993-5.539 10.193-7.399 11.799a1 1 0 0 1-1.202 0C9.539 20.193 4 14.993 4 10a8 8 0 0 1 16 0"/><circle cx="12" cy="10" r="3"/></svg>'

function currentCenter() {
  return location.value ?? DEFAULT_CENTER
}

function choose(position: { lat: number, lng: number }) {
  location.value = { lat: position.lat, lng: position.lng }
}

onMounted(async () => {
  const L = await import('leaflet')
  if (!container.value) return

  map = L.map(container.value, { zoomControl: false }).setView(currentCenter(), location.value ? PLACED_ZOOM : DEFAULT_ZOOM)
  L.control.zoom({ position: 'bottomleft', zoomInTitle: 'Acercar', zoomOutTitle: 'Alejar' }).addTo(map)
  L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
  }).addTo(map)

  marker = L.marker(currentCenter(), {
    draggable: true,
    keyboard: false,
    icon: L.divIcon({
      html: PIN_HTML,
      className: 'text-primary',
      iconSize: [36, 36],
      iconAnchor: [18, 34]
    })
  }).addTo(map)

  marker.on('dragend', () => choose(marker!.getLatLng()))
  // Tapping is easier than dragging on a phone.
  map.on('click', event => choose(event.latlng))
})

// The position also changes from outside (the «Usar mi ubicación» button, a restored draft).
watch(location, (position) => {
  if (!position || !map || !marker) return
  const current = marker.getLatLng()
  if (current.lat === position.lat && current.lng === position.lng) return
  marker.setLatLng(position)
  map.setView(position, Math.max(map.getZoom(), PLACED_ZOOM))
})

onBeforeUnmount(() => {
  map?.remove()
  map = undefined
})
</script>

<template>
  <div class="relative h-[220px] overflow-hidden rounded-lg bg-accented">
    <div ref="container" class="size-full" />
    <UBadge
      label="Arrastra el pin"
      color="neutral"
      variant="solid"
      size="sm"
      class="pointer-events-none absolute top-3 left-3 z-[1000]"
    />
    <div class="absolute top-3 right-3 z-[1000]">
      <slot />
    </div>
  </div>
</template>
