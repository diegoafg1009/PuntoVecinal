<script setup lang="ts">
import type { FormError } from '@nuxt/ui'

const draft = useReportDraftStore()

const state = reactive({
  get category() { return draft.category },
  get title() { return draft.title }
})

const fieldUi = { label: 'font-semibold text-highlighted' }

function validate(): FormError[] {
  const errors = draft.validateStep1()
  return Object.entries(errors).map(([name, message]) => ({ name, message }))
}

function next() {
  draft.step = 2
}
</script>

<template>
  <UForm
    :state="state"
    :validate="validate"
    :validate-on="[]"
    class="flex flex-col gap-3.5"
    @submit="next"
  >
    <UAlert
      color="error"
      variant="subtle"
      icon="i-lucide-triangle-alert"
      title="¿Es una emergencia o un tema de seguridad?"
      description="Llama a Serenazgo o al 105. Esta app no atiende urgencias."
    />

    <UFormField name="category" label="¿De qué se trata?" :ui="fieldUi">
      <div class="flex flex-wrap gap-1.5">
        <UButton
          v-for="category in categories"
          :key="category.slug"
          :label="category.label"
          :icon="draft.category === category.slug ? 'i-lucide-check' : undefined"
          :color="draft.category === category.slug ? 'primary' : 'neutral'"
          :variant="draft.category === category.slug ? 'solid' : 'outline'"
          :aria-pressed="draft.category === category.slug"
          size="sm"
          @click="draft.category = category.slug"
        />
      </div>
    </UFormField>

    <UFormField
      name="title"
      label="¿Qué pasa?"
      help="En una línea. Es el título que verán los vecinos."
      :ui="fieldUi"
    >
      <UInput v-model="draft.title" :maxlength="TITLE_MAX_LENGTH" class="w-full" />
    </UFormField>

    <UFormField name="details" label="Más detalles (opcional)" :ui="fieldUi">
      <UTextarea v-model="draft.details" :maxlength="DETAILS_MAX_LENGTH" :rows="3" class="w-full" />
    </UFormField>

    <UButton type="submit" label="Seguir" trailing-icon="i-lucide-chevron-right" block />
  </UForm>
</template>
