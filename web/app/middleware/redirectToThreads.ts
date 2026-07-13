export default defineNuxtRouteMiddleware(() => {
  try {
    if (localStorage.getItem('httpsms_redirect_to_threads') === 'true') {
      return navigateTo('/threads', { replace: true })
    }
  } catch (error) {
    console.error(error)
  }
})
