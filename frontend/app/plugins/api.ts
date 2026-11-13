import { client } from '~/api/client.gen'

/** Points the generated API client at our backend and signs its requests. */
export default defineNuxtPlugin(() => {
  const { apiBaseUrl } = useRuntimeConfig().public
  const session = useSessionStore()

  client.setConfig({ baseUrl: apiBaseUrl })
  client.interceptors.request.use((request) => {
    if (session.accessToken) {
      request.headers.set('Authorization', `Bearer ${session.accessToken}`)
    }
    return request
  })
})
