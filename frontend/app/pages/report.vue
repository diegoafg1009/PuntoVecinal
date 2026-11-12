<script setup lang="ts">
import type { JoinInput } from '~/composables/useJoinReport'

definePageMeta({ focusedOnMobile: true })

const draft = useReportDraftStore()

const { join, isPending: joining, errorMessage: joinError } = useJoinReport(
  () => draft.joinTarget?.id ?? '',
  () => draft.reset()
)

// The aporte carries the first draft photo, which the join screen also shows.
const firstPhoto = computed(() => draft.photos[0])

function joinReport(input: JoinInput) {
  join({ ...input, ...(firstPhoto.value && { photo: firstPhoto.value.blob }) })
}

// Back to the dialog with the same results.
function backToSimilar() {
  draft.joinTarget = null
  draft.similarOpen = true
}

const steps = [
  { title: 'Qué pasa', icon: 'i-lucide-message-circle' },
  { title: 'Foto y lugar', icon: 'i-lucide-camera' },
  { title: 'Revisar', icon: 'i-lucide-flag' }
]

const confirmingClose = ref(false)

// With something written, ask what to do with it; an empty form just closes.
function close() {
  if (draft.hasData) {
    confirmingClose.value = true
  } else {
    navigateTo('/')
  }
}

async function saveAndLeave() {
  confirmingClose.value = false
  await navigateTo('/')
}

async function discardAndLeave() {
  confirmingClose.value = false
  draft.reset()
  await navigateTo('/')
}
</script>

<template>
  <div v-if="draft.joinTarget" class="mx-auto flex min-h-screen w-full max-w-[560px] flex-col bg-default lg:min-h-0 lg:rounded-xl lg:border lg:border-default">
    <div class="flex h-14 items-center gap-1 border-b border-default pr-4 pl-2">
      <UButton icon="i-lucide-arrow-left" color="neutral" variant="ghost" aria-label="Atrás" @click="backToSimilar" />
      <h1 class="flex-1 px-1 font-semibold text-highlighted">
        Sumarte a este reporte
      </h1>
    </div>
    <div class="flex-1 p-4">
      <ReportJoin
        :title="draft.joinTarget.title"
        :category="draft.joinTarget.category"
        :status="draft.joinTarget.status"
        :cover-photo="draft.joinTarget.coverPhoto"
        :affected-count="draft.joinTarget.affectedCount"
        :distance-meters="draft.joinTarget.distanceMeters"
        :editable="false"
        :text="draft.contributionText"
        :photo-url="firstPhoto?.url"
        :initial-name="draft.authorName"
        :pending="joining"
        :error="joinError"
        @back="backToSimilar"
        @submit="joinReport"
      />
    </div>
  </div>

  <div v-else class="mx-auto flex min-h-screen w-full max-w-[560px] flex-col bg-default lg:min-h-0 lg:rounded-xl lg:border lg:border-default">
    <div class="flex h-14 items-center gap-1 border-b border-default pr-4 pl-2">
      <UButton
        v-if="draft.step > 1"
        icon="i-lucide-arrow-left"
        color="neutral"
        variant="ghost"
        aria-label="Atrás"
        @click="draft.step--"
      />
      <UButton
        v-else
        icon="i-lucide-x"
        color="neutral"
        variant="ghost"
        aria-label="Cerrar"
        @click="close"
      />
      <h1 class="flex-1 px-1 font-semibold text-highlighted">
        Nuevo reporte
      </h1>
      <span class="text-xs text-muted">{{ draft.step }} de {{ steps.length }}</span>
    </div>

    <div class="flex flex-1 flex-col gap-4 p-4">
      <UStepper :model-value="draft.step - 1" :items="steps" disabled size="sm" class="w-full" />

      <ReportEmergencyNotice />

      <ReportStepWhatHappens v-if="draft.step === 1" />
      <ReportStepPhotoAndPlace v-else-if="draft.step === 2" />
      <ReportStepReview v-else />
    </div>

    <ReportSimilarDialog />

    <UModal
      v-model:open="confirmingClose"
      title="¿Guardar borrador?"
      description="Puedes seguir con este reporte más tarde o empezar de cero."
    >
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton label="Descartar" color="neutral" variant="outline" @click="discardAndLeave" />
          <UButton label="Guardar borrador" @click="saveAndLeave" />
        </div>
      </template>
    </UModal>
  </div>
</template>
