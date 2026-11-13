<script setup lang="ts">
const session = useSessionStore()
</script>

<template>
  <div v-if="session.isAnonymous" class="mx-auto flex w-full max-w-[560px] flex-col items-center gap-2.5 px-4 py-20 text-center">
    <UIcon name="i-lucide-bell" class="size-10 text-primary" />
    <p class="text-sm font-medium text-highlighted">
      Los avisos necesitan una cuenta
    </p>
    <p class="text-xs text-muted">
      Con una cuenta podrás recibir novedades de los reportes que sigues. La creación de cuentas estará disponible próximamente.
    </p>
  </div>
  <ComingSoon v-else />
</template>
