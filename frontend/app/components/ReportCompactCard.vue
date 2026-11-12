<script setup lang="ts">
import type { Category, ReportStatus } from '~/api/types.gen'

const props = defineProps<{
  title: string
  category: Category
  status: ReportStatus
  coverPhoto: string | null
  affectedCount: number
  /** Meters from the neighbor; left out when the position is unknown. */
  distanceMeters?: number | null
}>()

const photoFailed = ref(false)
watch(() => props.coverPhoto, () => (photoFailed.value = false))
</script>

<template>
  <UCard :ui="{ body: 'p-3 sm:p-3' }">
    <div class="flex items-center gap-3">
      <div class="size-16 shrink-0 overflow-hidden rounded-lg bg-accented">
        <img
          v-if="coverPhoto && !photoFailed"
          :src="coverPhoto"
          :alt="title"
          class="size-full object-cover"
          @error="photoFailed = true"
        >
        <div v-else class="flex size-full items-center justify-center text-muted">
          <UIcon name="i-lucide-camera" class="size-5" />
        </div>
      </div>
      <div class="min-w-0 flex-1 space-y-1">
        <div class="flex flex-wrap items-center gap-1.5">
          <UBadge :label="category.label" color="neutral" variant="soft" size="sm" />
          <UBadge :label="statusLabel(status)" :color="statusColor(status)" variant="soft" size="sm" />
        </div>
        <p class="line-clamp-2 text-[15px] font-semibold text-highlighted">
          {{ title }}
        </p>
        <p class="flex items-center gap-1 text-xs text-muted">
          <template v-if="distanceMeters != null">
            <UIcon name="i-lucide-map-pin" class="size-3.5 shrink-0" />
            <span class="flex-1">A {{ formatDistance(distanceMeters) }}</span>
          </template>
          <span v-else class="flex-1" />
          <UIcon name="i-lucide-users" class="size-3.5 shrink-0" />
          <span class="shrink-0">a {{ affectedCount }} les afecta</span>
        </p>
      </div>
    </div>
  </UCard>
</template>
