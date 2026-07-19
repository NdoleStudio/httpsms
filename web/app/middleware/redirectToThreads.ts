import { STORAGE_KEY } from '~/stores/redirectPreference'

export default defineNuxtRouteMiddleware(() => {
  try {
    if (localStorage.getItem(STORAGE_KEY) === 'true') {
      return navigateTo('/threads', { replace: true })
    }
  } catch (error) {
    console.error(error)
  }
})
