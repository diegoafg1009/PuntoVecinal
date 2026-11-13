<script setup lang="ts">
import { useInfiniteQuery } from '@tanstack/vue-query'
import { useIntersectionObserver } from '@vueuse/core'
import type { ReportPage } from '~/api/types.gen'
import { searchReportsInfiniteOptions } from '~/api/@tanstack/vue-query.gen'

const FIVE_MINUTES = 5 * 60_000
const PAGE_SIZE = 20

const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const { location, locating, share, restore } = useNeighborLocation()

// The submitted search lives in the address: /search?q=poste&near=1
const submittedQ = computed(() => (typeof route.query.q === 'string' ? route.query.q.trim() : ''))
const near = computed(() => route.query.near === '1')

const text = ref(typeof route.query.q === 'string' ? route.query.q : '')
watch(() => route.query.q, (q) => {
  text.value = typeof q === 'string' ? q : ''
})

function setQuery(next: { q?: string, near?: boolean }) {
  const q = next.q ?? submittedQ.value
  const isNear = next.near ?? near.value
  return router.replace({
    query: {
      ...(q ? { q } : {}),
      ...(isNear ? { near: '1' } : {})
    }
  })
}

onMounted(async () => {
  await restore()
  // Opened with «Cerca de mí» but no location is available: drop it.
  if (near.value && !location.value) await setQuery({ near: false })
})

function submit() {
  return setQuery({ q: text.value.trim() })
}

async function toggleNear(on: boolean) {
  if (!on) {
    await setQuery({ near: false })
    return
  }
  if (await share()) await setQuery({ near: true })
}

const searchEnabled = computed(() => session.isAuthenticated && (submittedQ.value !== '' || (near.value && location.value !== null)))

// q, near and the location are part of the key, so changing any restarts from the first page.
const query = useInfiniteQuery(computed(() => ({
  ...searchReportsInfiniteOptions({
    query: {
      limit: PAGE_SIZE,
      ...(submittedQ.value ? { q: submittedQ.value } : {}),
      ...(near.value ? { sort: 'distance' as const } : {}),
      ...(location.value ? { latitude: location.value.latitude, longitude: location.value.longitude } : {})
    }
  }),
  initialPageParam: undefined as never,
  getNextPageParam: (lastPage: ReportPage) => lastPage.nextCursor ?? undefined,
  staleTime: FIVE_MINUTES,
  // Cached pages must not outlive their signed photo links (1 hour).
  gcTime: 30 * 60_000,
  enabled: searchEnabled.value
})))

const results = computed(() => query.data.value?.pages.flatMap(page => page.items) ?? [])

const sentinel = useTemplateRef('sentinel')
useIntersectionObserver(sentinel, ([entry]) => {
  if (entry?.isIntersecting && query.hasNextPage.value && !query.isFetchingNextPage.value && !query.isFetchNextPageError.value) {
    query.fetchNextPage()
  }
}, { rootMargin: '200px' })
</script>

<template>
  <div class="mx-auto w-full max-w-[560px] bg-default lg:rounded-xl lg:border lg:border-default">
    <h1 class="hidden border-b border-default px-4 py-3 font-semibold text-highlighted lg:block">
      Buscar
    </h1>
    <div class="space-y-3 p-4">
      <form role="search" @submit.prevent="submit">
        <UInput
          v-model="text"
          type="search"
          icon="i-lucide-search"
          placeholder="Busca por calle o problema"
          aria-label="Buscar reportes"
          color="primary"
          highlight
          :maxlength="100"
          enterkeyhint="search"
          class="w-full"
        />
      </form>

      <div class="flex">
        <UButton
          label="Cerca de mí"
          size="xs"
          :color="near ? 'primary' : 'neutral'"
          :variant="near ? 'soft' : 'outline'"
          :loading="locating"
          :aria-pressed="near"
          @click="toggleNear(!near)"
        />
      </div>

      <template v-if="searchEnabled">
        <div v-if="query.isPending.value" class="space-y-3" aria-busy="true">
          <USkeleton v-for="n in 3" :key="n" class="h-64 w-full" />
        </div>

        <div v-else-if="query.isError.value && !results.length" class="space-y-3 py-10 text-center">
          <p class="text-muted">
            No pudimos hacer la búsqueda.
          </p>
          <UButton label="Reintentar" color="neutral" variant="outline" @click="query.refetch()" />
        </div>

        <p v-else-if="!results.length" class="py-10 text-center text-muted">
          {{ submittedQ ? `No encontramos reportes para «${submittedQ}».` : 'No hay reportes cerca de ti.' }}
        </p>

        <template v-else>
          <ReportCard v-for="report in results" :key="report.id" :report="report" />

          <div v-if="query.isFetchNextPageError.value" class="space-y-2 py-4 text-center">
            <p class="text-sm text-muted">
              No pudimos cargar más reportes.
            </p>
            <UButton label="Reintentar" color="neutral" variant="outline" size="sm" @click="query.fetchNextPage()" />
          </div>
          <USkeleton v-else-if="query.isFetchingNextPage.value" class="h-64 w-full" />
        </template>
      </template>

      <div ref="sentinel" class="h-px" aria-hidden="true" />
    </div>
  </div>
</template>
