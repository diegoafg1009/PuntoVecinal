<script setup lang="ts">
import type { FormError } from '@nuxt/ui'
import type { Category, ReportStatus } from '~/api/types.gen'

const CONTRIBUTION_MAX_LENGTH = 1200

const props = defineProps<{
  title: string
  category: Category
  status: ReportStatus
  coverPhoto: string | null
  affectedCount: number
  distanceMeters?: number | null
  /** False when the aporte is already written (Reportar): it is shown, not edited. */
  editable: boolean
  /** The given aporte text and photo, when not editable. */
  text?: string
  photoUrl?: string | null
  /** Name suggested at the start, e.g. the one chosen in Reportar. */
  initialName?: string
  pending?: boolean
  error?: string
}>()

const emit = defineEmits<{
  back: []
  submit: [input: { text: string, authorName?: string, photo?: Blob }]
}>()

const session = useSessionStore()
const canChooseName = computed(() => session.session?.user.is_anonymous === true)

const form = useTemplateRef('form')
const fileInput = useTemplateRef('fileInput')

const showName = ref(canChooseName.value && (props.initialName ?? '').trim() !== '')
const name = ref(props.initialName ?? '')
const draftText = ref('')
const photo = ref<{ blob: Blob, url: string } | null>(null)
const adding = ref(false)
const photoNotice = ref('')

const state = reactive({
  get text() { return draftText.value },
  get authorName() { return name.value }
})

const fieldUi = { label: 'font-semibold text-highlighted' }

const shownText = computed(() => props.editable ? draftText.value : (props.text ?? ''))
const shownPhoto = computed(() => props.editable ? photo.value?.url ?? null : props.photoUrl ?? null)

watch(showName, (on) => {
  if (!on) name.value = ''
  form.value?.clear('authorName')
})
watch(name, (value) => {
  if (value.trim()) form.value?.clear('authorName')
})
watch(draftText, (value) => {
  if (value.trim()) form.value?.clear('text')
})

onBeforeUnmount(() => {
  if (photo.value) URL.revokeObjectURL(photo.value.url)
})

function validate(): FormError[] {
  const errors: FormError[] = []
  if (props.editable && !draftText.value.trim()) {
    errors.push({ name: 'text', message: 'Cuéntanos tu aporte.' })
  }
  if (showName.value && !name.value.trim()) {
    errors.push({ name: 'authorName', message: 'Escribe tu nombre o apodo, o apaga la opción.' })
  }
  return errors
}

function removePhoto() {
  if (photo.value) URL.revokeObjectURL(photo.value.url)
  photo.value = null
}

async function onFileChosen(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = '' // lets the same file be chosen again after removing it
  if (!file) return

  adding.value = true
  photoNotice.value = ''
  try {
    const blob = await processPhoto(file)
    removePhoto()
    photo.value = { blob, url: URL.createObjectURL(blob) }
  } catch {
    photoNotice.value = 'No pudimos usar esa foto.'
  } finally {
    adding.value = false
  }
}

// When not editable the parent already holds the photo and adds it to the request.
function submit() {
  if (props.pending) return
  const authorName = showName.value ? name.value.trim().slice(0, AUTHOR_NAME_MAX_LENGTH) : ''
  emit('submit', {
    text: shownText.value.trim(),
    ...(authorName && { authorName }),
    ...(props.editable && photo.value && { photo: photo.value.blob })
  })
}
</script>

<template>
  <UForm
    ref="form"
    :state="state"
    :validate="validate"
    :validate-on="[]"
    class="flex flex-col gap-3.5"
    @submit="submit"
  >
    <ReportCompactCard
      :title="title"
      :category="category"
      :status="status"
      :cover-photo="coverPhoto"
      :affected-count="affectedCount"
      :distance-meters="distanceMeters"
    />

    <p class="text-sm">
      Tu apoyo se suma a este caso; no se crea uno nuevo.
    </p>

    <ReportAuthorNameField v-model:show-name="showName" v-model:name="name" />

    <section class="flex flex-col gap-2">
      <h2 class="text-sm font-semibold text-highlighted">
        Tu aporte
      </h2>

      <template v-if="editable">
        <UFormField name="text" :ui="fieldUi">
          <UTextarea
            v-model="draftText"
            :maxlength="CONTRIBUTION_MAX_LENGTH"
            :rows="4"
            placeholder="Cuenta qué viste o qué más se debe saber"
            class="w-full"
          />
        </UFormField>

        <div class="flex flex-wrap gap-2" :aria-busy="adding">
          <div v-if="photo" class="relative size-24 overflow-hidden rounded-lg bg-accented">
            <img :src="photo.url" alt="Foto del aporte" class="size-full object-cover">
            <UButton
              icon="i-lucide-x"
              color="neutral"
              variant="solid"
              size="xs"
              aria-label="Quitar foto"
              class="absolute top-1 right-1 rounded-full"
              @click="removePhoto"
            />
          </div>
          <USkeleton v-if="adding" class="size-24 rounded-lg" />
          <button
            type="button"
            :disabled="adding"
            class="flex size-24 cursor-pointer flex-col items-center justify-center gap-1 rounded-lg border border-dashed border-accented text-xs font-medium text-muted hover:bg-elevated disabled:cursor-not-allowed disabled:opacity-75"
            @click="fileInput?.click()"
          >
            <UIcon name="i-lucide-image-plus" class="size-[22px] text-highlighted" />
            {{ photo ? 'Cambiar' : 'Añadir' }}
          </button>
          <input ref="fileInput" type="file" accept="image/*" class="hidden" @change="onFileChosen">
        </div>
        <UAlert
          v-if="photoNotice"
          color="warning"
          variant="subtle"
          icon="i-lucide-triangle-alert"
          :title="photoNotice"
          role="alert"
        />
      </template>

      <template v-else>
        <p class="text-sm whitespace-pre-line">
          {{ shownText }}
        </p>
        <img
          v-if="shownPhoto"
          :src="shownPhoto"
          alt="Foto de tu aporte"
          class="size-14 rounded-lg object-cover"
        >
      </template>
    </section>

    <UAlert
      v-if="error"
      color="error"
      variant="subtle"
      icon="i-lucide-triangle-alert"
      :title="error"
      role="alert"
    />

    <UButton
      type="submit"
      label="Sumarme a este reporte"
      icon="i-lucide-users"
      :loading="pending"
      :disabled="pending || adding"
      block
    />
    <UButton
      type="button"
      label="No es el mismo, volver"
      color="neutral"
      variant="link"
      block
      @click="emit('back')"
    />
  </UForm>
</template>
