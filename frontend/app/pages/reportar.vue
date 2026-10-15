<script setup lang="ts">
definePageMeta({ focusedOnMobile: true })

const draft = useReportDraftStore()

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
  <div class="mx-auto flex min-h-screen w-full max-w-[560px] flex-col bg-default lg:min-h-0 lg:rounded-xl lg:border lg:border-default">
    <div class="flex h-14 items-center gap-1 border-b border-default pr-4 pl-2">
      <UButton
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

      <ReportStepWhatHappens v-if="draft.step === 1" />
      <div v-else class="flex flex-col items-center gap-4 py-10 text-center text-muted">
        <p>Próximamente</p>
        <UButton label="‹ Volver" color="neutral" variant="outline" @click="draft.step = 1" />
      </div>
    </div>

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
