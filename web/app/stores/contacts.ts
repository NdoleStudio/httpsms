import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { EntitiesContact } from '~~/shared/types/api'
import { getApiErrorMessage } from '~/utils/api-error'

export interface ContactInput {
  name: string
  emails: string[]
  phone_numbers: string[]
  properties?: Record<string, string>
}

export const useContactsStore = defineStore('contacts', () => {
  const contacts = ref<EntitiesContact[]>([])
  const loading = ref(false)
  const search = ref('')
  const { apiFetch } = useApi()
  const notificationsStore = useNotificationsStore()

  const total = computed(() => contacts.value.length)

  const filteredContacts = computed<EntitiesContact[]>(() => {
    const term = search.value.trim().toLowerCase()
    if (!term) return contacts.value

    return contacts.value.filter((contact) => {
      const name = contact.name.toLowerCase()
      const emails = contact.emails.join(' ').toLowerCase()
      const phoneNumbers = contact.phone_numbers.join(' ').toLowerCase()

      return (
        name.includes(term) ||
        emails.includes(term) ||
        phoneNumbers.includes(term)
      )
    })
  })

  async function loadContacts(force = false): Promise<void> {
    if (contacts.value.length > 0 && !force) return

    loading.value = true
    try {
      const term = search.value.trim()
      const params: Record<string, string | number> = { limit: 100 }
      if (term) {
        params.query = term
      }
      const response = await apiFetch<{ data: EntitiesContact[] }>(
        '/v1/contacts',
        { params },
      )
      contacts.value = response.data ?? []
    } catch (error: unknown) {
      notificationsStore.addNotification({
        message: getApiErrorMessage(error, 'Error while loading contacts'),
        type: 'error',
      })
      throw error
    } finally {
      loading.value = false
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
      contacts.value = contacts.value.filter((contact) => contact.id !== id)
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
    contacts.value = []
    loading.value = false
    search.value = ''
  }

  return {
    contacts,
    loading,
    search,
    total,
    filteredContacts,
    loadContacts,
    saveContacts,
    updateContact,
    deleteContact,
    uploadCsv,
    resetState,
  }
})
