import { defineStore } from 'pinia'
import type { User as FirebaseUser } from 'firebase/auth'
import { setAuthHeader, setApiKey } from '../composables/useApi'
import type { EntitiesUser } from '~~/shared/types/api'

export interface AuthUser {
  email: string | null
  displayName: string | null
  id: string
}

export const useAuthStore = defineStore('auth', () => {
  const authStateChanged = ref(false)
  const authUser = ref<AuthUser | null>(null)
  const user = ref<EntitiesUser | null>(null)
  const { apiFetch } = useApi()
  let sessionVersion = 0
  let userRequestVersion = 0
  let publishedUserVersion = 0
  let publishedMutationVersion = 0

  function publishLoadedUser(
    loadedUser: EntitiesUser,
    version: number,
    mutationVersion: number,
  ) {
    if (version < publishedUserVersion) return
    publishedUserVersion = version
    const nextUser =
      mutationVersion !== publishedMutationVersion && user.value
        ? { ...loadedUser, api_key: user.value.api_key }
        : loadedUser
    setApiKey(nextUser.api_key)
    user.value = nextUser
  }

  function publishMutatedUser(updatedUser: EntitiesUser, version: number) {
    if (version < publishedMutationVersion) return
    publishedMutationVersion = version
    // Preserve later billing reads while applying the mutation's API key.
    const nextUser =
      version < publishedUserVersion && user.value
        ? { ...user.value, api_key: updatedUser.api_key }
        : updatedUser
    publishedUserVersion = Math.max(publishedUserVersion, version)
    setApiKey(nextUser.api_key)
    user.value = nextUser
  }

  function clearCredentials() {
    sessionVersion++
    user.value = null
    setAuthHeader(null)
    setApiKey(null)
  }

  async function setAuthUserAction(newUser: AuthUser | null | undefined) {
    const userChanged = newUser?.id !== authUser.value?.id
    if (userChanged) clearCredentials()
    authUser.value = newUser ?? null
    authStateChanged.value = true

    if (userChanged && newUser) {
      await Promise.all([loadUser(), loadPhones()])
    }
  }

  async function onAuthStateChanged(firebaseUser: FirebaseUser | null) {
    if (firebaseUser == null) {
      resetState()
      return
    }
    if (authUser.value?.id !== firebaseUser.uid) clearCredentials()
    const session = sessionVersion
    const token = await firebaseUser.getIdToken()
    if (session !== sessionVersion) return
    setAuthHeader(token)
    const { uid, email, displayName } = firebaseUser
    authUser.value = { id: uid, email, displayName }
    authStateChanged.value = true
  }

  async function onIdTokenChanged(firebaseUser: FirebaseUser | null) {
    await onAuthStateChanged(firebaseUser)
  }

  async function loadUser(signal?: AbortSignal) {
    const session = sessionVersion
    const version = ++userRequestVersion
    const mutationVersion = publishedMutationVersion
    const response = await apiFetch<{ data: EntitiesUser }>('/v1/users/me', {
      signal,
    })
    if (signal?.aborted || session !== sessionVersion) {
      return
    }
    publishLoadedUser(response.data, version, mutationVersion)
  }

  async function updateUser(payload: { owner?: string; timezone?: string }) {
    const session = sessionVersion
    const phonesStore = usePhonesStore()
    if (payload.owner) {
      phonesStore.setOwner(payload.owner)
    }

    const activePhone = phonesStore.activePhone
    if (!activePhone) return

    const version = ++userRequestVersion
    const response = await apiFetch<{ data: EntitiesUser }>('/v1/users/me', {
      method: 'PUT',
      body: {
        active_phone_id: activePhone.id,
        timezone: payload.timezone ?? user.value?.timezone,
      },
    })

    if (session !== sessionVersion) return
    publishMutatedUser(response.data, version)
  }

  async function deleteUserAccount(): Promise<string> {
    await apiFetch<{ message: string }>('/v1/users/me', {
      method: 'DELETE',
    })
    return 'Your account has been deleted successfully'
  }

  async function rotateApiKey(userId: string): Promise<EntitiesUser> {
    const session = sessionVersion
    const version = ++userRequestVersion
    const response = await apiFetch<{ data: EntitiesUser }>(
      `/v1/users/${userId}/api-keys`,
      {
        method: 'DELETE',
      },
    )
    if (session === sessionVersion) {
      publishMutatedUser(response.data, version)
    }
    return response.data
  }

  function resetState() {
    clearCredentials()
    authUser.value = null
    authStateChanged.value = true
  }

  function loadPhones() {
    const phonesStore = usePhonesStore()
    return phonesStore.loadPhones(false)
  }

  return {
    authStateChanged,
    authUser,
    user,
    setAuthUserAction,
    onAuthStateChanged,
    onIdTokenChanged,
    loadUser,
    updateUser,
    deleteUserAccount,
    rotateApiKey,
    resetState,
  }
})
