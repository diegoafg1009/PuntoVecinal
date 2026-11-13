import type { ReportStatus } from '~/api/types.gen'

export interface ReportFilterValues {
  categories: string[]
  status: ReportStatus | null
}

/** Feed and map filters, shared by Inicio and Mapa. Kept in memory only: a reload clears them. */
export const useReportFiltersStore = defineStore('reportFilters', () => {
  const categories = ref<string[]>([])
  const status = ref<ReportStatus | null>(null)

  const activeCount = computed(() => categories.value.length + (status.value ? 1 : 0))

  /** Query parameters for the API, leaving out empty values so the query key stays stable. */
  const queryParams = computed(() => ({
    ...(categories.value.length > 0 && { category: [...categories.value] }),
    ...(status.value && { status: status.value })
  }))

  function apply(values: ReportFilterValues) {
    categories.value = [...values.categories]
    status.value = values.status
  }

  function toggleCategory(slug: string) {
    categories.value = categories.value.includes(slug)
      ? categories.value.filter(item => item !== slug)
      : [...categories.value, slug]
  }

  function setStatus(value: ReportStatus | null) {
    status.value = value
  }

  function clear() {
    categories.value = []
    status.value = null
  }

  return { categories, status, activeCount, queryParams, apply, toggleCategory, setStatus, clear }
})
