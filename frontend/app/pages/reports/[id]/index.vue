<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { getReport } from '~/api'
import { getReportQueryKey } from '~/api/@tanstack/vue-query.gen'

definePageMeta({ ownTopBarOnMobile: true })

const route = useRoute()
const router = useRouter()
const session = useSessionStore()

const id = computed(() => String(route.params.id))

class ReportNotFoundError extends Error {}

const query = useQuery(computed(() => ({
  queryKey: getReportQueryKey({ path: { id: id.value } }),
  queryFn: async ({ signal }: { signal: AbortSignal }) => {
    const { data, response } = await getReport({ path: { id: id.value }, signal })
    if (response?.status === 404) throw new ReportNotFoundError()
    if (!data) throw new Error('Could not load the report')
    return data
  },
  // Same bounds as the feed: cached data must not outlive the signed photo links.
  staleTime: 5 * 60_000,
  gcTime: 30 * 60_000,
  retry: (count: number, error: Error) => !(error instanceof ReportNotFoundError) && count < 2,
  enabled: session.isAuthenticated
})))

const report = computed(() => query.data.value)
const notFound = computed(() => query.error.value instanceof ReportNotFoundError)

// Vue Router keeps the previous in-app location in history.state.back.
function goBack() {
  if (window.history.state?.back) router.back()
  else navigateTo('/')
}

const failedPhotos = ref(new Set<number>())
const photoIndex = ref(0)
watch(id, () => {
  failedPhotos.value = new Set()
  photoIndex.value = 0
})

function markFailed(index: number) {
  failedPhotos.value = new Set(failedPhotos.value).add(index)
}

const meta = computed(() => report.value
  ? [report.value.authorName ?? 'Vecino anónimo', publishedAgo(report.value.createdAt), report.value.reference]
      .filter(Boolean)
      .join(' · ')
  : '')

const location = computed({
  get: () => report.value ? { lat: report.value.location.latitude, lng: report.value.location.longitude } : null,
  set: () => {}
})
</script>

<template>
  <div class="mx-auto w-full max-w-[560px] bg-default lg:rounded-xl lg:border lg:border-default">
    <div class="sticky top-0 z-10 flex h-14 items-center gap-1 border-b border-default bg-default pr-4 pl-2 lg:static lg:rounded-t-xl">
      <UButton
        icon="i-lucide-arrow-left"
        color="neutral"
        variant="ghost"
        aria-label="Volver"
        @click="goBack"
      />
      <span class="font-semibold">Reporte</span>
    </div>

    <div v-if="query.isPending.value" class="space-y-3 p-4" aria-busy="true">
      <USkeleton class="h-[200px] w-full lg:h-[280px]" />
      <USkeleton class="h-5 w-40" />
      <USkeleton class="h-7 w-full" />
      <USkeleton class="h-4 w-2/3" />
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

    <article v-else class="space-y-3.5 p-4">
      <div class="relative">
        <UCarousel
          v-slot="{ item, index }"
          :items="report.photos"
          :arrows="report.photos.length > 1"
          :ui="{ item: 'basis-full ps-0', container: 'ms-0', prev: 'start-2 sm:start-2', next: 'end-2 sm:end-2' }"
          class="w-full"
          @select="photoIndex = $event"
        >
          <div class="h-[200px] w-full overflow-hidden rounded-lg bg-accented lg:h-[280px]">
            <img
              v-if="!failedPhotos.has(index)"
              :src="item.url"
              :alt="`Foto ${index + 1} de ${report.photos.length}: ${report.title}`"
              class="size-full object-cover"
              @error="markFailed(index)"
            >
            <div v-else class="flex size-full items-center justify-center text-muted">
              <UIcon name="i-lucide-camera" class="size-6" />
            </div>
          </div>
        </UCarousel>
        <UBadge
          v-if="report.photos.length > 1"
          :label="`${photoIndex + 1} / ${report.photos.length}`"
          color="neutral"
          variant="solid"
          size="sm"
          class="pointer-events-none absolute top-2 right-2"
        />
      </div>

      <div class="flex flex-wrap items-center gap-1.5">
        <UBadge :label="report.category.label" color="neutral" variant="soft" size="sm" />
        <UBadge :label="statusLabel(report.status)" :color="statusColor(report.status)" variant="soft" size="sm" />
      </div>

      <h1 class="text-lg font-semibold text-highlighted">
        {{ report.title }}
      </h1>
      <p class="text-xs text-muted">
        {{ meta }}
      </p>

      <p v-if="report.details" class="text-sm whitespace-pre-line">
        {{ report.details }}
      </p>

      <div class="flex flex-wrap items-center gap-2">
        <ReportSupportButton :report="report" />
        <UButton
          label="Aportar"
          icon="i-lucide-plus"
          color="neutral"
          variant="outline"
          :to="`/reports/${report.id}/join`"
        />
      </div>

      <ReportLocationMap v-model:location="location" readonly />

      <USeparator />

      <ReportContributions :report-id="report.id" :count="report.contributionCount" />
    </article>
  </div>
</template>
