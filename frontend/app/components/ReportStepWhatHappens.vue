<script setup lang="ts">
import type { FormError } from '@nuxt/ui'

const draft = useReportDraftStore()
const { data: categories, isPending, isError, refetch, isRefetching } = useCategories()

const form = useTemplateRef('form')

// A stored category the API does not offer any more is dropped; the rest of the draft stays.
watch(categories, (list) => {
  if (!list) return
  draft.pruneCategory(list.map(category => category.slug))
  form.value?.clear('category') // drops the «vuelve a cargar» message once they are there
}, { immediate: true })

const state = reactive({
  get category() { return draft.category },
  get title() { return draft.title }
})

const fieldUi = { label: 'font-semibold text-highlighted' }

function validate(): FormError[] {
  const errors = draft.validateStep1()
  // A stored category only counts once the API has confirmed it is still offered.
  if (!errors.category && !categories.value?.some(category => category.slug === draft.category)) {
    errors.category = 'Vuelve a cargar las categorías para continuar.'
  }
  return Object.entries(errors).map(([name, message]) => ({ name, message }))
}

function next() {
  draft.step = 2
}
</script>

<template>
  <UForm
    ref="form"
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
      <div v-if="isPending" class="flex flex-wrap gap-1.5" aria-busy="true">
        <USkeleton v-for="width in ['w-20', 'w-24', 'w-32', 'w-28', 'w-16']" :key="width" class="h-8 rounded-md" :class="width" />
      </div>
      <UAlert
        v-else-if="isError"
        color="error"
        variant="subtle"
        icon="i-lucide-triangle-alert"
        title="No pudimos cargar las categorías."
        :actions="[{ label: 'Reintentar', color: 'error', variant: 'outline', loading: isRefetching, onClick: () => refetch() }]"
      />
      <div v-else class="flex flex-wrap gap-1.5">
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

    <UButton type="submit" label="Seguir" :disabled="isPending" trailing-icon="i-lucide-chevron-right" block />
  </UForm>
</template>
