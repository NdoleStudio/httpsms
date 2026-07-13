import { defineStore } from 'pinia'
import { ref } from 'vue'

const STORAGE_KEY = 'httpsms_redirect_to_threads'

function readFlag(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === 'true'
  } catch (error) {
    console.error(error)
    return false
  }
}

export const useRedirectPreferenceStore = defineStore(
  'redirectPreference',
  () => {
    const enabled = ref(readFlag())
    const dismissedThisSession = ref(false)

    function enable() {
      enabled.value = true
      try {
        localStorage.setItem(STORAGE_KEY, 'true')
      } catch (error) {
        console.error(error)
      }
      navigateTo('/threads', { replace: true })
    }

    function dismiss() {
      dismissedThisSession.value = true
    }

    function resetState() {
      enabled.value = false
      dismissedThisSession.value = false
      try {
        localStorage.removeItem(STORAGE_KEY)
      } catch (error) {
        console.error(error)
      }
    }

    return { enabled, dismissedThisSession, enable, dismiss, resetState }
  },
)
