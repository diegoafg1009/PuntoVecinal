<script setup lang="ts">
const toast = useToast()

// The map does not exist yet: tell the neighbor and stay on the current screen.
function showMapSoon() {
  toast.add({ title: 'Próximamente', description: 'El mapa estará disponible pronto.' })
}

const items = [
  { label: 'Inicio', icon: 'i-lucide-house', to: '/' },
  { label: 'Buscar', icon: 'i-lucide-search', to: '/search' },
  { label: 'Avisos', icon: 'i-lucide-bell', to: '/notices' },
  {
    label: 'Mapa',
    icon: 'i-lucide-map',
    badge: 'Pronto',
    class: 'text-dimmed',
    'aria-disabled': 'true',
    onSelect: showMapSoon
  }
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
