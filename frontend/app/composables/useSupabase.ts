import { createClient, type SupabaseClient } from '@supabase/supabase-js'

let client: SupabaseClient | undefined

/** Shared Supabase client. The frontend uses it only for authentication. */
export function useSupabase(): SupabaseClient {
  if (!client) {
    const { supabaseUrl, supabaseAnonKey } = useRuntimeConfig().public
    client = createClient(supabaseUrl, supabaseAnonKey)
  }
  return client
}
