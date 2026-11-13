<script setup lang="ts">
import type { ReportSummary } from '~/api/types.gen'

const props = defineProps<{ report: ReportSummary }>()

const photoFailed = ref(false)
watch(() => props.report.coverPhoto.url, () => (photoFailed.value = false))

const place = computed(() => [
  props.report.distanceMeters === null ? null : formatDistance(props.report.distanceMeters),
  props.report.reference
].filter(Boolean).join(' · '))
</script>

<template>
  <NuxtLink :to="`/reports/${report.id}`" class="block rounded-lg focus-visible:outline-2 focus-visible:outline-primary">
    <UCard :ui="{ body: 'p-3 sm:p-3 space-y-2.5' }" class="transition hover:bg-elevated">
      <div class="flex items-center justify-between gap-2">
        <div class="flex flex-wrap items-center gap-1.5">
          <UBadge :label="report.category.label" color="neutral" variant="soft" size="sm" />
          <UBadge :label="statusLabel(report.status)" :color="statusColor(report.status)" variant="soft" size="sm" />
        </div>
        <span class="shrink-0 text-xs text-muted">{{ publishedAgo(report.createdAt) }}</span>
      </div>

      <div class="aspect-[7/3] w-full overflow-hidden rounded-lg bg-accented">
        <img
          v-if="!photoFailed"
          :src="report.coverPhoto.url"
          :alt="report.title"
          loading="lazy"
          class="size-full object-cover"
          @error="photoFailed = true"
        >
        <div v-else class="flex size-full items-center justify-center text-muted">
          <UIcon name="i-lucide-camera" class="size-6" />
        </div>
      </div>

      <h2 class="line-clamp-2 font-semibold text-highlighted">
        {{ report.title }}
      </h2>

      <p class="flex items-center gap-1 text-xs text-muted">
        <template v-if="place">
          <UIcon name="i-lucide-map-pin" class="size-3.5 shrink-0" />
          <span class="truncate">{{ place }}</span>
        </template>
        <span class="flex-1" />
        <UIcon name="i-lucide-users" class="size-3.5 shrink-0" />
        <span class="shrink-0">{{ report.affectedCount }}</span>
      </p>
    </UCard>
  </NuxtLink>
</template>
