<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { getReportMapOptions, getReportOptions } from '~/api/@tanstack/vue-query.gen'

const FIVE_MINUTES = 5 * 60_000

const session = useSessionStore()
const filters = useReportFiltersStore()
const { location, locating, share, restore } = useNeighborLocation()

onMounted(restore)

const center = computed(() => location.value ? { lat: location.value.latitude, lng: location.value.longitude } : null)

const query = useQuery(computed(() => ({
  ...getReportMapOptions({ query: filters.queryParams }),
  staleTime: FIVE_MINUTES,
  enabled: session.isAuthenticated
})))

const pins = computed(() => query.data.value?.items ?? [])

const selectedId = ref<string | null>(null)
const selected = computed(() => pins.value.find(pin => pin.id === selectedId.value) ?? null)
// A filter change can remove the selected pin from the map.
watch(selected, (pin) => {
  if (!pin) selectedId.value = null
})

// The photo is only fetched for the tapped pin. Its signed link lasts 1 hour, so the cache is shorter.
const detail = useQuery(computed(() => ({
  ...getReportOptions({ path: { id: selectedId.value ?? '' } }),
  staleTime: FIVE_MINUTES,
  gcTime: 30 * 60_000,
  enabled: session.isAuthenticated && selectedId.value !== null
})))

const photoUrl = computed(() => detail.data.value?.photos[0]?.url ?? null)
const photoFailed = ref(false)
watch(photoUrl, () => (photoFailed.value = false))

const LEGEND = [
  { label: 'Publicado', status: 'publicado' },
  { label: 'Derivado / Recibido por la entidad', status: 'derivado' },
  { label: 'En atención', status: 'en_atencion' },
  { label: 'Atendido según la entidad', status: 'atendido' },
  { label: 'Cerrado', status: 'cerrado' }
] as const

const legendOpen = ref(false)
const noReports = computed(() => query.isSuccess.value && !pins.value.length)
</script>

<template>
  <!-- On desktop the map sits in a centered panel titled «Mapa», as in Figma. -->
  <div class="mx-auto w-full max-w-[960px] bg-default lg:rounded-xl lg:border lg:border-default">
    <h1 class="hidden border-b border-default px-4 py-3 font-semibold text-highlighted lg:block">
      Mapa
    </h1>

    <div class="relative h-[calc(100dvh-8.5rem)] overflow-hidden lg:h-[640px] lg:rounded-b-xl">
      <ClientOnly>
        <ReportPinsMap
          :pins="pins"
          :center="center"
          @select="selectedId = $event"
          @clear="selectedId = null"
        />
      </ClientOnly>

      <!-- Floating controls sit above Leaflet's panes (z-index up to 1000). -->
      <div class="pointer-events-none absolute inset-x-0 top-3 z-[1000] px-4">
        <div class="pointer-events-auto">
          <ReportFilterChips variant="solid" />
        </div>
      </div>

      <div class="absolute top-14 left-4 z-[1000]">
        <UCard :ui="{ body: 'p-2 sm:p-2' }">
          <button
            type="button"
            class="flex items-center gap-1.5 text-xs font-medium text-muted"
            :aria-expanded="legendOpen"
            @click="legendOpen = !legendOpen"
          >
            <UIcon name="i-lucide-info" class="size-3.5" />
            Colores
            <UIcon :name="legendOpen ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'" class="size-3.5" />
          </button>
          <ul v-if="legendOpen" class="mt-2 space-y-1.5">
            <li v-for="item in LEGEND" :key="item.status" class="flex items-center gap-2 text-xs">
              <span class="size-3 rounded-full border-2" :style="{ borderColor: statusPinColor(item.status) }" />
              {{ item.label }}
            </li>
          </ul>
        </UCard>
      </div>

      <UButton
        icon="i-lucide-locate-fixed"
        color="neutral"
        variant="outline"
        class="absolute right-4 bottom-28 z-[1000] bg-default"
        aria-label="Usar mi ubicación"
        :loading="locating"
        @click="share('No pudimos obtener tu ubicación. El mapa sigue centrado en la zona.')"
      />

      <div v-if="query.isPending.value && session.isAuthenticated" class="absolute inset-x-0 top-24 z-[1000] flex justify-center" aria-busy="true">
        <UBadge label="Cargando reportes…" color="neutral" variant="solid" icon="i-lucide-loader-circle" :ui="{ leadingIcon: 'animate-spin' }" />
      </div>

      <div v-else-if="query.isError.value" class="absolute inset-x-4 top-24 z-[1000] mx-auto max-w-sm space-y-3 rounded-lg border border-default bg-default p-4 text-center shadow-lg">
        <p class="text-muted">
          No pudimos cargar los reportes.
        </p>
        <UButton label="Reintentar" color="neutral" variant="outline" @click="query.refetch()" />
      </div>

      <div v-else-if="noReports && filters.activeCount > 0" class="absolute inset-x-4 top-24 z-[1000] mx-auto max-w-sm space-y-3 rounded-lg border border-default bg-default p-4 text-center shadow-lg">
        <p class="text-muted">
          Ningún reporte coincide con los filtros.
        </p>
        <UButton label="Quitar filtros" color="neutral" variant="outline" @click="filters.clear()" />
      </div>

      <div v-else-if="noReports" class="absolute inset-x-4 top-24 z-[1000] mx-auto max-w-sm space-y-3 rounded-lg border border-default bg-default p-4 text-center shadow-lg">
        <p class="text-muted">
          Aún no hay reportes
        </p>
        <UButton label="Reportar" to="/report" />
      </div>

      <NuxtLink
        v-if="selected"
        :to="`/reports/${selected.id}`"
        class="absolute inset-x-4 bottom-4 z-[1000] block rounded-lg focus-visible:outline-2 focus-visible:outline-primary lg:inset-x-auto lg:left-4 lg:w-[400px]"
      >
        <UCard :ui="{ body: 'p-3 sm:p-3' }" class="shadow-lg">
          <div class="flex items-center gap-3">
            <div class="flex size-14 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-accented text-muted">
              <USkeleton v-if="detail.isPending.value" class="size-full" />
              <img v-else-if="photoUrl && !photoFailed" :src="photoUrl" alt="" class="size-full object-cover" @error="photoFailed = true">
              <UIcon v-else name="i-lucide-camera" class="size-6" />
            </div>
            <div class="min-w-0 flex-1 space-y-1">
              <p class="line-clamp-2 text-sm font-medium text-highlighted">
                {{ selected.title }}
              </p>
              <div class="flex flex-wrap items-center gap-1.5">
                <UBadge :label="selected.category.label" color="neutral" variant="soft" size="sm" />
                <UBadge :label="statusLabel(selected.status)" :color="statusColor(selected.status)" variant="soft" size="sm" />
              </div>
            </div>
            <UIcon name="i-lucide-chevron-right" class="size-[18px] shrink-0 text-muted" />
          </div>
        </UCard>
      </NuxtLink>
    </div>
  </div>
</template>
