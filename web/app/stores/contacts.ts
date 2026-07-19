import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { EntitiesContact } from '~~/shared/types/api'
import { getApiErrorMessage } from '~/utils/api-error'

export interface ContactInput {
  name: string
  emails: string[]
  phone_numbers: string[]
  properties?: Record<string, string>
}

export interface LoadContactsOptions {
  force?: boolean
  skip?: number
  limit?: number
}

// DEFAULT_LIMIT mirrors the contacts page's initial items-per-page. It is only
// used when a caller (e.g. a mutation refresh) does not specify its own limit.
const DEFAULT_LIMIT = 10

export const useContactsStore = defineStore('contacts', () => {
  const contacts = ref<EntitiesContact[]>([])
  const total = ref(0)
  const loading = ref(false)
  const search = ref('')
  const { apiFetch } = useApi()
  const notificationsStore = useNotificationsStore()
  let loadContactsGeneration = 0

  // The pagination window last requested by the page. Mutation-triggered
  // refreshes reuse it so the user stays on the page they were viewing.
  let currentSkip = 0
  let currentLimit = DEFAULT_LIMIT

  function normalizeOptions(
    options: LoadContactsOptions | boolean,
  ): LoadContactsOptions {
    if (typeof options === 'boolean') {
      return { force: options }
    }
    return options
  }

  async function loadContacts(
    options: LoadContactsOptions | boolean = {},
  ): Promise<void> {
    const { force = false, skip, limit } = normalizeOptions(options)

    if (skip !== undefined) {
      currentSkip = skip
    }
    if (limit !== undefined) {
      currentLimit = limit
    }

    if (contacts.value.length > 0 && !force) return

    const generation = ++loadContactsGeneration
    loading.value = true
    try {
      const term = search.value.trim()
      const params: Record<string, string | number> = {
        skip: currentSkip,
        limit: currentLimit,
      }
      if (term) {
        params.query = term
      }
      const response = await apiFetch<{
        data: EntitiesContact[]
        total?: number
      }>('/v1/contacts', { params })
      if (generation === loadContactsGeneration) {
        contacts.value = response.data ?? []
        total.value = response.total ?? contacts.value.length
      }
    } catch (error: unknown) {
      if (generation !== loadContactsGeneration) {
        return
      }
      notificationsStore.addNotification({
        message: getApiErrorMessage(error, 'Error while loading contacts'),
        type: 'error',
      })
      throw error
    } finally {
      if (generation === loadContactsGeneration) {
        loading.value = false
      }
    }
  }

  async function saveContacts(
    items: ContactInput | ContactInput[],
  ): Promise<void> {
    const contactsToSave = Array.isArray(items) ? items : [items]

    loading.value = true
    try {
      try {
        await apiFetch('/v1/contacts', {
          method: 'POST',
          body: contactsToSave,
        })
      } catch (error: unknown) {
        notificationsStore.addNotification({
          message: getApiErrorMessage(error, 'Error while saving contacts'),
          type: 'error',
        })
        throw error
      }

      notificationsStore.addNotification({
        message:
          contactsToSave.length > 1 ? 'Contacts created' : 'Contact created',
        type: 'success',
      })
      await loadContacts(true)
    } finally {
      loading.value = false
    }
  }

  async function updateContact(
    id: string,
    payload: ContactInput,
  ): Promise<void> {
    loading.value = true
    try {
      try {
        await apiFetch(`/v1/contacts/${id}`, {
          method: 'PUT',
          body: payload,
        })
      } catch (error: unknown) {
        notificationsStore.addNotification({
          message: getApiErrorMessage(error, 'Error while updating contact'),
          type: 'error',
        })
        throw error
      }

      notificationsStore.addNotification({
        message: 'Contact updated',
        type: 'success',
      })
      await loadContacts(true)
    } finally {
      loading.value = false
    }
  }

  async function deleteContact(id: string): Promise<void> {
    loading.value = true
    try {
      await apiFetch(`/v1/contacts/${id}`, { method: 'DELETE' })
      // Invalidate any in-flight load so a stale response cannot resurrect the
      // just-deleted contact. Bumping the generation makes loadContacts skip
      // its assignment (and its finally toggling loading) for the older request.
      loadContactsGeneration++
      contacts.value = contacts.value.filter((contact) => contact.id !== id)
      total.value = Math.max(0, total.value - 1)
      notificationsStore.addNotification({
        message: 'Contact deleted',
        type: 'success',
      })
    } catch (error: unknown) {
      notificationsStore.addNotification({
        message: getApiErrorMessage(error, 'Error while deleting contact'),
        type: 'error',
      })
      throw error
    } finally {
      loading.value = false
    }
  }

  async function uploadCsv(file: File): Promise<void> {
    loading.value = true
    try {
      try {
        const formData = new FormData()
        formData.append('document', file)

        await apiFetch('/v1/contacts/upload', {
          method: 'POST',
          body: formData,
        })
      } catch (error: unknown) {
        notificationsStore.addNotification({
          message: getApiErrorMessage(error, 'Error while importing contacts'),
          type: 'error',
        })
        throw error
      }

      notificationsStore.addNotification({
        message: 'Contacts imported successfully',
        type: 'success',
      })
      await loadContacts(true)
    } finally {
      loading.value = false
    }
  }

  function resetState() {
    loadContactsGeneration++
    contacts.value = []
    total.value = 0
    loading.value = false
    search.value = ''
    currentSkip = 0
    currentLimit = DEFAULT_LIMIT
  }

  return {
    contacts,
    total,
    loading,
    search,
    loadContacts,
    saveContacts,
    updateContact,
    deleteContact,
    uploadCsv,
    resetState,
  }
})
