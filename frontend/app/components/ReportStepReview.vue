<script setup lang="ts">
import type { FormError } from '@nuxt/ui'
import { useMutation } from '@tanstack/vue-query'
import { createReport } from '~/api'
import type { CreateReportData } from '~/api/types.gen'

const TOO_MANY_REPORTS = 429

// Assumption: a short list of basic rules, to be reviewed by the team before release.
const RULES = [
  'Reporta solo problemas reales del barrio.',
  'Sé respetuoso: sin insultos, amenazas ni datos personales de nadie.',
  'Evita que se vean caras o placas en las fotos.',
  'No uses esta app para emergencias o temas de seguridad: llama a Serenazgo o al 105.',
  'Si no hay un nombre elegido, tu reporte aparece como «Vecino anónimo».'
]

const draft = useReportDraftStore()
const session = useSessionStore()
const toast = useToast()
const { data: categories } = useCategories()

const form = useTemplateRef('form')

// Neighbors without an account can choose a name; a registered user's name arrives with accounts.
const canChooseName = computed(() => session.session?.user.is_anonymous === true)

const acceptedRules = ref(false)
const showName = ref(canChooseName.value && draft.authorName.trim() !== '')
const rulesOpen = ref(false)
const publishError = ref('')

const state = reactive({
  get rules() { return acceptedRules.value },
  get authorName() { return draft.authorName }
})

const fieldUi = { label: 'font-semibold text-highlighted' }

const categoryLabel = computed(() =>
  categories.value?.find(category => category.slug === draft.category)?.label ?? draft.category)

const appearsAs = computed(() =>
  showName.value && draft.authorName.trim() ? draft.authorName.trim() : 'Vecino anónimo')

watch(showName, (on) => {
  // Turning the switch off forgets the name so a reload does not turn it back on.
  if (!on) draft.authorName = ''
  form.value?.clear('authorName')
})

watch(acceptedRules, (accepted) => {
  if (accepted) form.value?.clear('rules')
})

watch(() => draft.authorName, (name) => {
  if (name.trim()) form.value?.clear('authorName')
})

function validate(): FormError[] {
  const errors: FormError[] = []
  if (!acceptedRules.value) {
    errors.push({ name: 'rules', message: 'Acepta las reglas de convivencia para publicar.' })
  }
  if (showName.value && !draft.authorName.trim()) {
    errors.push({ name: 'authorName', message: 'Escribe tu nombre o apodo, o apaga la opción.' })
  }
  return errors
}

class PublishError extends Error {
  constructor(readonly status: number) {
    super(`Publishing failed with status ${status}`)
  }
}

const { mutate: publish, isPending } = useMutation({
  mutationFn: async (body: CreateReportData['body']) => {
    const { data, error, response } = await createReport({ body })
    if (error !== undefined || !data) throw new PublishError(response?.status ?? 0)
    return data
  },
  onSuccess: async (created) => {
    draft.reset()
    await navigateTo(`/reports/${created.id}`)
    toast.add({ title: 'Reporte publicado', icon: 'i-lucide-circle-check', color: 'success' })
  },
  onError: (error) => {
    publishError.value = error instanceof PublishError && error.status === TOO_MANY_REPORTS
      ? 'Publicaste varios reportes seguidos. Espera unos minutos antes de publicar otro.'
      : 'No pudimos publicar tu reporte. Inténtalo de nuevo.'
  }
})

function submit() {
  const body = draft.publishInput({ showName: showName.value })
  if (!body) {
    // Steps 1 and 2 are required; send the neighbor back to the first one that is incomplete.
    publishError.value = 'Faltan datos del reporte. Revisa los pasos anteriores.'
    draft.step = Object.keys(draft.validateStep1()).length ? 1 : 2
    return
  }
  publishError.value = ''
  publish(body)
}
</script>

<template>
  <UForm
    ref="form"
    :state="state"
    :validate="validate"
    :validate-on="[]"
    class="flex flex-col gap-4"
    @submit="submit"
  >
    <section class="flex flex-col gap-1.5">
      <div class="flex items-center justify-between">
        <h2 class="font-semibold text-highlighted">
          Qué pasa
        </h2>
        <UButton label="Editar" color="neutral" variant="link" size="xs" @click="draft.step = 1" />
      </div>
      <UBadge :label="categoryLabel ?? ''" color="neutral" variant="subtle" class="self-start" />
      <p class="text-highlighted">
        {{ draft.title }}
      </p>
      <p v-if="draft.details.trim()" class="text-sm whitespace-pre-line text-muted">
        {{ draft.details }}
      </p>
    </section>

    <section class="flex flex-col gap-1.5">
      <div class="flex items-center justify-between">
        <h2 class="font-semibold text-highlighted">
          Foto y lugar
        </h2>
        <UButton label="Editar" color="neutral" variant="link" size="xs" @click="draft.step = 2" />
      </div>
      <div class="flex flex-wrap gap-2">
        <img
          v-for="photo in draft.photos"
          :key="photo.id"
          :src="photo.url"
          alt="Foto del reporte"
          class="size-24 rounded-lg object-cover"
        >
      </div>
      <p class="flex items-center gap-1.5 text-sm text-muted">
        <UIcon name="i-lucide-map-pin" class="size-4 shrink-0" />
        <span>{{ draft.reference.trim() || 'Lugar marcado en el mapa' }}</span>
      </p>
    </section>

    <USeparator />

    <div v-if="canChooseName" class="flex flex-col gap-2">
      <USwitch v-model="showName" label="Mostrar mi nombre o apodo" />
      <UFormField v-if="showName" name="authorName" label="Nombre o apodo" :ui="fieldUi">
        <UInput v-model="draft.authorName" :maxlength="AUTHOR_NAME_MAX_LENGTH" class="w-full" />
      </UFormField>
    </div>
    <p class="text-sm text-muted">
      Tu reporte aparecerá como <strong class="text-highlighted">{{ appearsAs }}</strong>.
    </p>

    <UFormField name="rules">
      <UCheckbox v-model="acceptedRules">
        <template #label>
          <span>
            Acepto las
            <button type="button" class="cursor-pointer text-primary underline" @click.prevent="rulesOpen = true">
              reglas de convivencia
            </button>
          </span>
        </template>
      </UCheckbox>
    </UFormField>

    <UAlert
      v-if="publishError"
      color="error"
      variant="subtle"
      icon="i-lucide-triangle-alert"
      :title="publishError"
      role="alert"
    />

    <UButton type="submit" label="Publicar" :loading="isPending" :disabled="isPending" block />

    <UModal v-model:open="rulesOpen" title="Reglas de convivencia">
      <template #body>
        <ul class="list-disc space-y-2 pl-5 text-sm">
          <li v-for="rule in RULES" :key="rule">
            {{ rule }}
          </li>
        </ul>
      </template>
      <template #footer>
        <UButton label="Entendido" block @click="rulesOpen = false" />
      </template>
    </UModal>
  </UForm>
</template>
