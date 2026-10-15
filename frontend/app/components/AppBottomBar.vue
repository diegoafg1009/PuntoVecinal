<script setup lang="ts">
const route = useRoute()

const items = [
  { label: 'Inicio', icon: 'i-lucide-house', to: '/' },
  { label: 'Buscar', icon: 'i-lucide-search', to: '/search' },
  { label: 'Reportar', icon: 'i-lucide-plus', to: '/report', main: true },
  { label: 'Avisos', icon: 'i-lucide-bell', to: '/notices' },
  { label: 'Mapa', icon: 'i-lucide-map', to: '/map' }
]

function isCurrent(to: string) {
  return to === '/' ? route.path === '/' : route.path.startsWith(to)
}
</script>

<template>
  <nav aria-label="Secciones" class="fixed inset-x-0 bottom-0 z-10 border-t border-default bg-default">
    <ul class="flex items-center pt-1.5 pb-2.5">
      <li v-for="item in items" :key="item.to" class="flex-1">
        <NuxtLink
          :to="item.to"
          :aria-current="isCurrent(item.to) ? 'page' : undefined"
          class="flex flex-col items-center gap-0.5 text-xs font-medium"
          :class="isCurrent(item.to) || item.main ? 'text-primary' : 'text-muted'"
        >
          <span v-if="item.main" class="flex rounded-md bg-primary p-1.5 text-inverted">
            <UIcon :name="item.icon" class="size-5" />
          </span>
          <UIcon v-else :name="item.icon" class="size-[22px]" />
          {{ item.label }}
        </NuxtLink>
      </li>
    </ul>
  </nav>
</template>
