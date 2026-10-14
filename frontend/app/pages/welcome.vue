<script setup lang="ts">
definePageMeta({ layout: false })

const session = useSessionStore()
const toast = useToast()

const loading = ref(false)
const failed = ref(false)

function showLoginSoon() {
  toast.add({ title: 'Próximamente', description: 'El inicio de sesión aún no está disponible.', icon: 'i-lucide-clock' })
}

async function continueAsGuest() {
  loading.value = true
  failed.value = false
  try {
    await session.signInAsGuest()
    await navigateTo('/')
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen flex-col md:bg-elevated">
    <header class="hidden h-16 items-center gap-2 border-b border-default bg-default px-6 md:flex">
      <UIcon name="i-lucide-map-pin" class="size-[22px] text-primary" />
      <span class="font-semibold text-highlighted">Punto Vecinal</span>
    </header>

    <main class="flex flex-1 items-center justify-center md:items-start md:pt-6 md:pb-10">
      <div class="flex w-full max-w-[560px] flex-col items-center gap-2.5 p-4 md:rounded-xl md:border md:border-default md:bg-default">
        <UIcon name="i-lucide-map-pin" class="size-12 text-primary" />
        <h1 class="text-2xl font-medium text-highlighted">
          Punto Vecinal
        </h1>
        <p class="text-center text-xs text-muted">
          Reporta problemas de tu barrio y sigue su solución.
        </p>

        <UAlert
          v-if="failed"
          class="w-full"
          color="error"
          variant="subtle"
          icon="i-lucide-circle-alert"
          title="No pudimos iniciar tu sesión"
          description="Revisa tu conexión e inténtalo de nuevo."
        />

        <UButton
          :label="failed ? 'Intentar de nuevo' : 'Comenzar'"
          class="rounded-full"
          block
          :loading="loading"
          @click="continueAsGuest"
        />
        <UButton
          label="Ya tengo una cuenta"
          class="rounded-full"
          color="neutral"
          variant="outline"
          block
          @click="showLoginSoon"
        />

        <p class="text-center text-[11px] text-muted">
          Puedes empezar sin registrarte.
        </p>
      </div>
    </main>
  </div>
</template>
