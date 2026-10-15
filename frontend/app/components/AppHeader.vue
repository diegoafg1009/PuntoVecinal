<script setup lang="ts">
const items = [
  { label: 'Inicio', icon: 'i-lucide-house', to: '/' },
  { label: 'Buscar', icon: 'i-lucide-search', to: '/buscar' },
  { label: 'Avisos', icon: 'i-lucide-bell', to: '/avisos' },
  { label: 'Mapa', icon: 'i-lucide-map', to: '/mapa' }
]
</script>

<template>
  <header class="h-16 items-center gap-4 border-b border-default bg-default px-6">
    <AppBrand class="mr-auto" />
    <UNavigationMenu :items="items" variant="link" />
    <UButton to="/reportar" label="Reportar" icon="i-lucide-plus" />
    <AppAvatar size="md" />
  </header>
</template>
