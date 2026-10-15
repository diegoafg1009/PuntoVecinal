<script setup lang="ts">
const items = [
  { label: 'Inicio', icon: 'i-lucide-house', to: '/' },
  { label: 'Buscar', icon: 'i-lucide-search', to: '/search' },
  { label: 'Avisos', icon: 'i-lucide-bell', to: '/notices' },
  { label: 'Mapa', icon: 'i-lucide-map', to: '/map' }
]
</script>

<template>
  <header class="h-16 items-center gap-4 border-b border-default bg-default px-6">
    <AppBrand class="mr-auto" />
    <UNavigationMenu :items="items" variant="link" />
    <UButton to="/report" label="Reportar" icon="i-lucide-plus" />
    <AppAvatar size="md" />
  </header>
</template>
