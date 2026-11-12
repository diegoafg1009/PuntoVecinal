<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { getReport } from '~/api'
import { getReportQueryKey } from '~/api/@tanstack/vue-query.gen'
import type { JoinInput } from '~/composables/useJoinReport'

definePageMeta({ ownTopBarOnMobile: true })

const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const neighborLocation = useNeighborLocation()

const id = computed(() => String(route.params.id))

class ReportNotFoundError extends Error {}

// Same query as the report page, so the card data comes from its cache when it is there.
const query = useQuery(computed(() => ({
  queryKey: getReportQueryKey({ path: { id: id.value } }),
  queryFn: async ({ signal }: { signal: AbortSignal }) => {
    const { data, response } = await getReport({ path: { id: id.value }, signal })
    if (response?.status === 404) throw new ReportNotFoundError()
    if (!data) throw new Error('Could not load the report')
    return data
  },
  staleTime: 5 * 60_000,
  gcTime: 30 * 60_000,
  retry: (count: number, error: Error) => !(error instanceof ReportNotFoundError) && count < 2,
  enabled: session.isAuthenticated
})))

const report = computed(() => query.data.value)
const notFound = computed(() => query.error.value instanceof ReportNotFoundError)

// The page never asks for the location: the distance shows only if it was already shared.
onMounted(() => neighborLocation.restore())

const distanceMeters = computed(() => {
  const here = neighborLocation.location.value
  if (!here || !report.value) return null
  return distanceBetween(here.latitude, here.longitude, report.value.location.latitude, report.value.location.longitude)
})

const { join, isPending, errorMessage } = useJoinReport(() => id.value)

function goBack() {
  if (window.history.state?.back) router.back()
  else navigateTo(`/reports/${id.value}`)
}

function submit(input: JoinInput) {
  join(input)
}
</script>

<template>
  <div class="mx-auto w-full max-w-[560px] bg-default lg:rounded-xl lg:border lg:border-default">
    <div class="sticky top-0 z-10 flex h-14 items-center gap-1 border-b border-default bg-default pr-4 pl-2 lg:static lg:rounded-t-xl">
      <UButton icon="i-lucide-arrow-left" color="neutral" variant="ghost" aria-label="Volver" @click="goBack" />
      <span class="font-semibold">Sumarte a este reporte</span>
    </div>

    <div v-if="query.isPending.value" class="space-y-3 p-4" aria-busy="true">
      <USkeleton class="h-24 w-full" />
      <USkeleton class="h-5 w-2/3" />
    </div>

    <div v-else-if="notFound" class="space-y-3 p-8 text-center">
      <p class="text-muted">
        No encontramos este reporte.
      </p>
      <UButton label="Ir a Inicio" to="/" />
    </div>

    <div v-else-if="!report" class="space-y-3 p-8 text-center">
      <p class="text-muted">
        No pudimos cargar el reporte.
      </p>
      <UButton label="Reintentar" color="neutral" variant="outline" @click="query.refetch()" />
    </div>

    <div v-else class="p-4">
      <ReportJoin
        :title="report.title"
        :category="report.category"
        :status="report.status"
        :cover-photo="report.photos[0]?.url ?? null"
        :affected-count="report.affectedCount"
        :distance-meters="distanceMeters"
        editable
        :pending="isPending"
        :error="errorMessage"
        @back="goBack"
        @submit="submit"
      />
    </div>
  </div>
</template>
