<script setup lang="ts">
import type { SimilarReport } from '~/api/types.gen'

const draft = useReportDraftStore()

const failedPhotos = ref(new Set<string>())

function markFailed(id: string) {
  failedPhotos.value = new Set(failedPhotos.value).add(id)
}

function join(report: SimilarReport) {
  draft.joinTarget = report
  draft.similarOpen = false
}

function keepReporting() {
  draft.similarOpen = false
  draft.step = 3
}
</script>

<template>
  <UModal
    v-model:open="draft.similarOpen"
    title="¿Ya lo reportó alguien?"
    description="Encontramos reportes parecidos a menos de 300 m del lugar que marcaste. Si es el mismo problema, súmate al que ya existe en vez de crear uno nuevo."
    :ui="{ content: 'max-sm:top-auto max-sm:bottom-0 max-sm:translate-y-0 max-sm:max-w-full max-sm:rounded-b-none max-sm:rounded-t-2xl' }"
  >
    <template #body>
      <ul class="flex max-h-[40vh] flex-col gap-3 overflow-y-auto rounded-lg border border-default bg-elevated p-3">
        <li v-for="report in draft.similar" :key="report.id">
          <button
            type="button"
            class="flex w-full cursor-pointer items-center gap-3 rounded-lg border border-default bg-default p-3 text-left hover:bg-elevated focus-visible:outline-2 focus-visible:outline-primary"
            @click="join(report)"
          >
            <div class="size-20 shrink-0 overflow-hidden rounded-lg bg-accented">
              <img
                v-if="!failedPhotos.has(report.id)"
                :src="report.coverPhoto"
                :alt="report.title"
                class="size-full object-cover"
                @error="markFailed(report.id)"
              >
              <div v-else class="flex size-full items-center justify-center text-muted">
                <UIcon name="i-lucide-camera" class="size-5" />
              </div>
            </div>
            <div class="min-w-0 flex-1 space-y-1">
              <p class="line-clamp-2 text-[15px] font-semibold text-highlighted">
                {{ report.title }}
              </p>
              <UBadge :label="report.category.label" color="neutral" variant="soft" size="sm" />
              <p class="flex items-center gap-1.5 text-xs text-muted">
                <UIcon name="i-lucide-map-pin" class="size-3.5 shrink-0" />
                <span>{{ formatDistance(report.distanceMeters) }}</span>
                <UIcon name="i-lucide-users" class="size-3.5 shrink-0" />
                <span>{{ report.affectedCount }}</span>
                <span class="truncate">· {{ publishedAgo(report.createdAt) }}</span>
              </p>
            </div>
          </button>
        </li>
      </ul>
    </template>
    <template #footer>
      <UButton label="No es ninguno, seguir" trailing-icon="i-lucide-chevron-right" block @click="keepReporting" />
    </template>
  </UModal>
</template>
