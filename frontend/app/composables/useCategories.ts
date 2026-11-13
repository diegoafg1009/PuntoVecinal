import { useQuery } from '@tanstack/vue-query'
import { listCategoriesOptions } from '~/api/@tanstack/vue-query.gen'

// The categories are fixed, so one answer is good for the whole visit.
const STALE_TIME = Infinity

/** Asks the API for the report categories, in the order it gives them. */
export function useCategories() {
  const session = useSessionStore()

  return useQuery(computed(() => ({
    ...listCategoriesOptions(),
    staleTime: STALE_TIME,
    enabled: session.isAuthenticated
  })))
}
