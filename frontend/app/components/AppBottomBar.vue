<script setup lang="ts">
const route = useRoute()

const items = [
  { label: 'Inicio', icon: 'i-lucide-house', to: '/' },
  { label: 'Buscar', icon: 'i-lucide-search', to: '/search' },
  { label: 'Reportar', icon: 'i-lucide-plus', to: '/report', main: true },
  { label: 'Avisos', icon: 'i-lucide-bell', to: '/notices' },
  { label: 'Mapa', icon: 'i-lucide-map', unavailable: true }
]

const toast = useToast()

// The map does not exist yet: tell the neighbor and stay on the current screen.
function showMapSoon() {
  toast.add({ title: 'Próximamente', description: 'El mapa estará disponible pronto.' })
}

function isCurrent(to?: string) {
  if (!to) return false
  return to === '/' ? route.path === '/' : route.path.startsWith(to)
}
</script>

<template>
  <nav aria-label="Secciones" class="fixed inset-x-0 bottom-0 z-10 border-t border-default bg-default">
    <ul class="flex items-center pt-1.5 pb-2.5">
      <li v-for="item in items" :key="item.label" class="flex-1">
        <button
          v-if="item.unavailable"
          type="button"
          aria-disabled="true"
          class="flex w-full flex-col items-center gap-0.5 text-xs font-medium text-dimmed"
          @click="showMapSoon"
        >
          <UIcon :name="item.icon" class="size-[22px]" />
          {{ item.label }}
          <span class="-mt-0.5 text-[10px] leading-none">Pronto</span>
        </button>
        <NuxtLink
          v-else
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
