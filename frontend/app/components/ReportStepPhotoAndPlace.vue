<script setup lang="ts">
import type { FormError } from '@nuxt/ui'

const draft = useReportDraftStore()

const form = useTemplateRef('form')
const fileInput = useTemplateRef('fileInput')

const state = reactive({
  get photos() { return draft.photos },
  get location() { return draft.location }
})

const fieldUi = { label: 'font-semibold text-highlighted' }

const adding = ref(false)
const photoNotice = ref('')
const locating = ref(false)
const locationFailed = ref(false)

function validate(): FormError[] {
  return Object.entries(draft.validateStep2()).map(([name, message]) => ({ name, message }))
}

async function onFilesChosen(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  input.value = '' // lets the same file be chosen again after removing it
  if (!files.length) return

  adding.value = true
  photoNotice.value = ''
  const { failed, overLimit } = await draft.addPhotos(files)
  adding.value = false

  if (overLimit) photoNotice.value = `Un reporte lleva como máximo ${PHOTO_MAX_COUNT} fotos.`
  else if (failed) photoNotice.value = 'No pudimos usar esa foto.'
  if (draft.photos.length) form.value?.clear('photos')
}

function removePhoto(id: string) {
  photoNotice.value = ''
  draft.removePhoto(id)
}

// The browser asks for permission only here, when the neighbor taps the button.
function useMyLocation() {
  locationFailed.value = false
  if (!('geolocation' in navigator)) {
    locationFailed.value = true
    return
  }
  locating.value = true
  navigator.geolocation.getCurrentPosition(
    (position) => {
      locating.value = false
      draft.setLocation({ lat: position.coords.latitude, lng: position.coords.longitude })
    },
    () => {
      locating.value = false
      locationFailed.value = true
    },
    { enableHighAccuracy: false, timeout: 10_000 }
  )
}

watch(() => draft.location, (position) => {
  if (!position) return
  locationFailed.value = false
  form.value?.clear('location')
})

function next() {
  draft.step = 3
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
    <UFormField name="photos" :ui="fieldUi">
      <template #label>
        <span class="flex items-center gap-2">
          Fotos
          <UBadge label="obligatoria, de 1 a 3" color="neutral" variant="outline" size="sm" />
        </span>
      </template>

      <div class="flex flex-wrap gap-2" :aria-busy="!draft.ready || adding">
        <div
          v-for="photo in draft.photos"
          :key="photo.id"
          class="relative size-24 overflow-hidden rounded-lg bg-accented"
        >
          <img :src="photo.url" alt="Foto del reporte" class="size-full object-cover">
          <UButton
            icon="i-lucide-x"
            color="neutral"
            variant="solid"
            size="xs"
            aria-label="Quitar foto"
            class="absolute top-1 right-1 rounded-full"
            @click="removePhoto(photo.id)"
          />
        </div>

        <USkeleton v-if="adding || !draft.ready" class="size-24 rounded-lg" />

        <button
          v-if="draft.photos.length < PHOTO_MAX_COUNT"
          type="button"
          :disabled="adding || !draft.ready"
          class="flex size-24 cursor-pointer flex-col items-center justify-center gap-1 rounded-lg border border-dashed border-accented text-xs font-medium text-muted hover:bg-elevated disabled:cursor-not-allowed disabled:opacity-75"
          @click="fileInput?.click()"
        >
          <UIcon name="i-lucide-image-plus" class="size-[22px] text-highlighted" />
          Añadir
        </button>

        <input
          ref="fileInput"
          type="file"
          accept="image/*"
          multiple
          class="hidden"
          @change="onFilesChosen"
        >
      </div>
    </UFormField>

    <p class="-mt-1.5 text-xs text-muted">
      Evita que se vean caras o placas.
    </p>

    <UAlert
      v-if="photoNotice"
      color="warning"
      variant="subtle"
      icon="i-lucide-triangle-alert"
      :title="photoNotice"
      role="alert"
    />

    <UFormField
      name="location"
      label="¿Dónde es?"
      help="Mueve el pin al lugar exacto."
      :ui="fieldUi"
    >
      <ClientOnly>
        <ReportLocationMap :location="draft.location" @update:location="draft.setLocation($event!)">
          <UButton
            label="Usar mi ubicación"
            icon="i-lucide-locate-fixed"
            color="neutral"
            variant="outline"
            size="xs"
            class="bg-default"
            :loading="locating"
            @click="useMyLocation"
          />
        </ReportLocationMap>
        <template #fallback>
          <USkeleton class="h-[220px] w-full rounded-lg" />
        </template>
      </ClientOnly>
    </UFormField>

    <UAlert
      v-if="locationFailed"
      color="warning"
      variant="subtle"
      icon="i-lucide-map-pin-off"
      title="No pudimos obtener tu ubicación. Márcala en el mapa."
      role="alert"
    />

    <UFormField name="reference" label="Referencia" :ui="fieldUi">
      <UInput v-model="draft.reference" :maxlength="REFERENCE_MAX_LENGTH" class="w-full" />
    </UFormField>

    <UButton
      type="submit"
      label="Seguir"
      :disabled="adding || !draft.ready"
      trailing-icon="i-lucide-chevron-right"
      block
    />
  </UForm>
</template>
