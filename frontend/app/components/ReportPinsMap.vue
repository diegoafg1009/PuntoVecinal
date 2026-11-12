<script setup lang="ts">
import type { LayerGroup, Map as LeafletMap } from 'leaflet'
import type { ReportPin } from '~/api/types.gen'
import 'leaflet/dist/leaflet.css'

const FLY_ZOOM = 16

const props = defineProps<{
  pins: ReportPin[]
  /** Where to center the map; the pilot zone when empty. */
  center: { lat: number, lng: number } | null
}>()

const emit = defineEmits<{
  select: [id: string]
  clear: []
}>()

const container = useTemplateRef('container')
let map: LeafletMap | undefined
let layer: LayerGroup | undefined
let leaflet: typeof import('leaflet') | undefined

// Lucide map-pin outline, stroked with the status color. Leaflet's default PNG icons break under bundlers.
function pinHtml(color: string) {
  return `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="white" style="stroke: ${color}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 10c0 4.993-5.539 10.193-7.399 11.799a1 1 0 0 1-1.202 0C9.539 20.193 4 14.993 4 10a8 8 0 0 1 16 0"/><circle cx="12" cy="10" r="3"/></svg>`
}

function drawPins() {
  if (!leaflet || !layer) return
  const L = leaflet
  layer.clearLayers()
  for (const pin of props.pins) {
    L.marker([pin.location.latitude, pin.location.longitude], {
      keyboard: false,
      title: pin.title,
      icon: L.divIcon({
        html: pinHtml(statusPinColor(pin.status)),
        className: '',
        iconSize: [32, 32],
        iconAnchor: [16, 30]
      })
    })
      .on('click', (event) => {
        L.DomEvent.stopPropagation(event)
        emit('select', pin.id)
      })
      .addTo(layer)
  }
}

onMounted(async () => {
  const L = await import('leaflet')
  if (!container.value) return
  leaflet = L

  map = L.map(container.value, { zoomControl: false })
    .setView(props.center ?? PILOT_CENTER, props.center ? FLY_ZOOM : PILOT_ZOOM)
  L.control.zoom({ position: 'bottomleft', zoomInTitle: 'Acercar', zoomOutTitle: 'Alejar' }).addTo(map)
  L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
  }).addTo(map)
  layer = L.layerGroup().addTo(map)
  map.on('click', () => emit('clear'))
  drawPins()
})

watch(() => props.pins, drawPins)

// The neighbor shares their location after the map is open: fly to it.
watch(() => props.center, (position) => {
  if (position && map) map.flyTo(position, FLY_ZOOM)
})

onBeforeUnmount(() => {
  map?.remove()
  map = undefined
})
</script>

<template>
  <div ref="container" class="size-full bg-accented" />
</template>
