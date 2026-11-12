<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import type { ReportStatus } from '~/api/types.gen'

const STATUSES: ReportStatus[] = ['publicado', 'derivado', 'recibido', 'en_atencion', 'atendido', 'cerrado']

const props = withDefaults(defineProps<{
  /** Inicio uses soft active chips with «Estado» second; Mapa uses solid chips with «Estado» last. */
  variant?: 'soft' | 'solid'
}>(), { variant: 'soft' })

const filters = useReportFiltersStore()
const { data: categories } = useCategories()

const activeVariant = computed(() => props.variant)

const statusItems = computed<DropdownMenuItem[]>(() => [
  {
    label: 'Todos',
    type: 'checkbox',
    checked: filters.status === null,
    onUpdateChecked: () => filters.setStatus(null)
  },
  ...STATUSES.map<DropdownMenuItem>(status => ({
    label: statusLabel(status),
    type: 'checkbox',
    checked: filters.status === status,
    onUpdateChecked: () => filters.setStatus(status)
  }))
])

const statusChip = computed(() => filters.status ? statusLabel(filters.status) : 'Estado')
</script>

<template>
  <div class="flex items-center gap-1.5 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden" role="group" aria-label="Filtros">
    <UButton
      label="Todo"
      size="xs"
      class="shrink-0"
      :color="filters.activeCount === 0 ? 'primary' : 'neutral'"
      :variant="filters.activeCount === 0 ? activeVariant : 'outline'"
      :aria-pressed="filters.activeCount === 0"
      @click="filters.clear()"
    />

    <UDropdownMenu v-if="variant === 'soft'" :items="statusItems" :content="{ align: 'start' }">
      <UButton
        :label="statusChip"
        trailing-icon="i-lucide-chevron-down"
        size="xs"
        class="shrink-0"
        :color="filters.status ? 'primary' : 'neutral'"
        :variant="filters.status ? activeVariant : 'outline'"
        :aria-pressed="filters.status !== null"
      />
    </UDropdownMenu>

    <UButton
      v-for="category in categories ?? []"
      :key="category.slug"
      :label="category.label"
      size="xs"
      class="shrink-0"
      :color="filters.categories.includes(category.slug) ? 'primary' : 'neutral'"
      :variant="filters.categories.includes(category.slug) ? activeVariant : 'outline'"
      :aria-pressed="filters.categories.includes(category.slug)"
      @click="filters.toggleCategory(category.slug)"
    />

    <UDropdownMenu v-if="variant === 'solid'" :items="statusItems" :content="{ align: 'end' }">
      <UButton
        :label="statusChip"
        trailing-icon="i-lucide-chevron-down"
        size="xs"
        class="shrink-0"
        :color="filters.status ? 'primary' : 'neutral'"
        :variant="filters.status ? activeVariant : 'outline'"
        :aria-pressed="filters.status !== null"
      />
    </UDropdownMenu>
  </div>
</template>
