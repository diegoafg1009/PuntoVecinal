<script setup lang="ts">
import { useInfiniteQuery } from '@tanstack/vue-query'
import { listReportContributions } from '~/api'
import { listReportContributionsQueryKey } from '~/api/@tanstack/vue-query.gen'
import type { ContributionPage } from '~/api/types.gen'

const props = defineProps<{ reportId: string, count: number }>()

const PAGE_SIZE = 20

const query = useInfiniteQuery(computed(() => ({
  queryKey: listReportContributionsQueryKey({ path: { id: props.reportId } }),
  queryFn: async ({ pageParam, signal }: { pageParam: string | undefined, signal: AbortSignal }) => {
    const { data } = await listReportContributions({
      path: { id: props.reportId },
      query: { limit: PAGE_SIZE, ...(pageParam && { cursor: pageParam }) },
      signal
    })
    if (!data) throw new Error('Could not load the contributions')
    return data as ContributionPage
  },
  initialPageParam: undefined as string | undefined,
  getNextPageParam: (last: ContributionPage) => last.nextCursor ?? undefined,
  // Same bounds as the report: cached data must not outlive the signed photo links.
  staleTime: 5 * 60_000,
  gcTime: 30 * 60_000
})))

const items = computed(() => query.data.value?.pages.flatMap(page => page.items) ?? [])

const failedPhotos = ref(new Set<string>())
function markFailed(id: string) {
  failedPhotos.value = new Set(failedPhotos.value).add(id)
}
</script>

<template>
  <section class="space-y-3">
    <h2 class="font-semibold text-highlighted">
      Aportes y comentarios · {{ count }}
    </h2>

    <div v-if="query.isPending.value" class="space-y-3" aria-busy="true">
      <USkeleton class="h-16 w-full" />
      <USkeleton class="h-16 w-full" />
    </div>

    <div v-else-if="query.isError.value" class="space-y-2 text-center">
      <p class="text-sm text-muted">
        No pudimos cargar los aportes.
      </p>
      <UButton label="Reintentar" color="neutral" variant="outline" size="sm" @click="query.refetch()" />
    </div>

    <p v-else-if="!items.length" class="text-sm text-muted">
      Todavía no hay aportes. Sé el primero en aportar.
    </p>

    <ul v-else class="space-y-4">
      <li v-for="item in items" :key="item.id" class="flex gap-2.5">
        <UAvatar :alt="item.authorName ?? 'Vecino anónimo'" size="sm" class="shrink-0" />
        <div class="min-w-0 flex-1 space-y-1.5">
          <p class="text-xs text-muted">
            {{ item.authorName ?? 'Vecino anónimo' }} · {{ publishedAgo(item.createdAt) }}
          </p>
          <p class="text-sm whitespace-pre-line">
            {{ item.text }}
          </p>
          <template v-if="item.photoUrl">
            <img
              v-if="!failedPhotos.has(item.id)"
              :src="item.photoUrl"
              alt="Foto del aporte"
              loading="lazy"
              class="max-h-60 rounded-lg object-cover"
              @error="markFailed(item.id)"
            >
            <div v-else class="flex size-24 items-center justify-center rounded-lg bg-accented text-muted">
              <UIcon name="i-lucide-camera" class="size-5" />
            </div>
          </template>
        </div>
      </li>
    </ul>

    <UButton
      v-if="query.hasNextPage.value"
      label="Ver más"
      color="neutral"
      variant="outline"
      block
      :loading="query.isFetchingNextPage.value"
      @click="query.fetchNextPage()"
    />
  </section>
</template>
