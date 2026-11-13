export default defineNuxtRouteMiddleware(async (to) => {
  const session = useSessionStore()
  await session.initialize()

  if (!session.isAuthenticated && to.path !== '/welcome') {
    return navigateTo('/welcome')
  }
  if (session.isAuthenticated && to.path === '/welcome') {
    return navigateTo('/')
  }
})
