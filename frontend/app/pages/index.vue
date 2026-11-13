<script setup lang="ts">
import { useInfiniteQuery } from '@tanstack/vue-query'
import { useIntersectionObserver } from '@vueuse/core'
import type { ReportPage } from '~/api/types.gen'
import { listReportsInfiniteOptions } from '~/api/@tanstack/vue-query.gen'

const FIVE_MINUTES = 5 * 60_000
const PAGE_SIZE = 20

const session = useSessionStore()
const filters = useReportFiltersStore()
const { location, locating, share, restore } = useNeighborLocation()

onMounted(restore)

const hasLocation = computed(() => location.value !== null)

// The location and the filters are part of the key, so changing them reloads the list from the first page.
const query = useInfiniteQuery(computed(() => ({
  ...listReportsInfiniteOptions({
    query: {
      limit: PAGE_SIZE,
      ...filters.queryParams,
      ...(location.value ? { latitude: location.value.latitude, longitude: location.value.longitude } : {})
    }
  }),
  initialPageParam: undefined as never,
  getNextPageParam: (lastPage: ReportPage) => lastPage.nextCursor ?? undefined,
  staleTime: FIVE_MINUTES,
  // Cached pages must not outlive their signed photo links (1 hour).
  gcTime: 30 * 60_000,
  enabled: session.isAuthenticated
})))

const reports = computed(() => query.data.value?.pages.flatMap(page => page.items) ?? [])

const sentinel = useTemplateRef('sentinel')
useIntersectionObserver(sentinel, ([entry]) => {
  if (entry?.isIntersecting && query.hasNextPage.value && !query.isFetchingNextPage.value && !query.isFetchNextPageError.value) {
    query.fetchNextPage()
  }
}, { rootMargin: '200px' })
</script>

<template>
  <!-- On desktop the feed sits in a centered panel titled «Inicio», as in Figma. -->
  <div class="mx-auto w-full max-w-[560px] bg-default lg:rounded-xl lg:border lg:border-default">
    <h1 class="hidden border-b border-default px-4 py-3 font-semibold text-highlighted lg:block">
      Inicio
    </h1>
    <div class="space-y-3 p-4">
      <div class="flex items-center justify-between gap-2">
        <h2 class="text-sm font-medium text-muted">
          {{ hasLocation ? 'Cerca de ti · más recientes' : 'Más recientes' }}
        </h2>
        <UButton
          v-if="!hasLocation"
          label="Usar mi ubicación"
          icon="i-lucide-locate-fixed"
          color="neutral"
          variant="ghost"
          size="sm"
          :loading="locating"
          @click="share()"
        />
      </div>

      <ReportFilterChips variant="soft" />

      <div v-if="query.isPending.value && session.isAuthenticated" class="space-y-3" aria-busy="true">
        <USkeleton v-for="n in 3" :key="n" class="h-64 w-full" />
      </div>

      <div v-else-if="query.isError.value && !reports.length" class="space-y-3 py-10 text-center">
        <p class="text-muted">
          No pudimos cargar los reportes.
        </p>
        <UButton label="Reintentar" color="neutral" variant="outline" @click="query.refetch()" />
      </div>

      <div v-else-if="!reports.length && query.isSuccess.value && filters.activeCount > 0" class="space-y-3 py-10 text-center">
        <p class="text-muted">
          Ningún reporte coincide con los filtros.
        </p>
        <UButton label="Quitar filtros" color="neutral" variant="outline" @click="filters.clear()" />
      </div>

      <div v-else-if="!reports.length && query.isSuccess.value" class="space-y-3 py-10 text-center">
        <p class="text-muted">
          Aún no hay reportes
        </p>
        <UButton label="Reportar" to="/report" />
      </div>

      <template v-else>
        <ReportCard v-for="report in reports" :key="report.id" :report="report" />

        <div v-if="query.isFetchNextPageError.value" class="space-y-2 py-4 text-center">
          <p class="text-sm text-muted">
            No pudimos cargar más reportes.
          </p>
          <UButton label="Reintentar" color="neutral" variant="outline" size="sm" @click="query.fetchNextPage()" />
        </div>
        <USkeleton v-else-if="query.isFetchingNextPage.value" class="h-64 w-full" />
      </template>

      <div ref="sentinel" class="h-px" aria-hidden="true" />
    </div>
  </div>
</template>
