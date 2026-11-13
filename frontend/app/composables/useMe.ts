import { useQuery } from '@tanstack/vue-query'
import { getMe } from '~/api'
import { getMeQueryKey } from '~/api/@tanstack/vue-query.gen'

/** Asks the API who the current session belongs to. */
export function useMe() {
  const session = useSessionStore()

  return useQuery(computed(() => ({
    // A new session (for example after signing in) must not reuse the old answer.
    queryKey: [...getMeQueryKey(), session.userId] as const,
    queryFn: async ({ signal }: { signal: AbortSignal }) => {
      const { data } = await getMe({ signal, throwOnError: true })
      return data
    },
    enabled: session.isAuthenticated
  })))
}
