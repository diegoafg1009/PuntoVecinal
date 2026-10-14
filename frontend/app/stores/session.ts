import type { Session } from '@supabase/supabase-js'

export const useSessionStore = defineStore('session', () => {
  const session = ref<Session | null>(null)

  const isAuthenticated = computed(() => session.value !== null)
  const userId = computed(() => session.value?.user.id ?? null)
  const accessToken = computed(() => session.value?.access_token ?? null)

  let initialization: Promise<void> | undefined

  /**
   * Restores the session stored on this device (supabase-js refreshes its
   * token on its own) and keeps the store in sync with later changes.
   */
  function initialize(): Promise<void> {
    initialization ??= (async () => {
      const supabase = useSupabase()
      const { data } = await supabase.auth.getSession()
      session.value = data.session
      supabase.auth.onAuthStateChange((_event, next) => {
        session.value = next
      })
    })()
    return initialization
  }

  /** Creates an anonymous session on this device, without asking for data. */
  async function signInAsGuest(): Promise<void> {
    const { data, error } = await useSupabase().auth.signInAnonymously()
    if (error) throw error
    session.value = data.session
  }

  return { session, isAuthenticated, userId, accessToken, initialize, signInAsGuest }
})
