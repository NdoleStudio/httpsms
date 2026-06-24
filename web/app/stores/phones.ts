import { defineStore } from 'pinia'
import type { EntitiesPhone, EntitiesHeartbeat } from '~~/shared/types/api'
import { getApiErrorMessage } from '~/utils/api-error'

export const usePhonesStore = defineStore('phones', () => {
  const phones = ref<EntitiesPhone[]>([])
  const owner = ref<string | null>(null)
  const heartbeat = ref<EntitiesHeartbeat | null>(null)
  const { apiFetch } = useApi()
  const notificationsStore = useNotificationsStore()

  const activePhone = computed<EntitiesPhone | null>(() => {
    return phones.value.find((x) => x.phone_number === owner.value) ?? null
  })

  function setOwner(value: string) {
    owner.value = value
  }

  async function loadPhones(force: boolean = false) {
    if (phones.value.length > 0 && !force) return

    const response = await apiFetch<{ data: EntitiesPhone[] }>('/v1/phones', {
      params: { limit: 100 },
    })
    phones.value = response.data

    const authStore = useAuthStore()
    if (authStore.user?.active_phone_id) {
      const phone = response.data.find(
        (x) => x.id === authStore.user?.active_phone_id,
      )
      if (phone) {
        owner.value = phone.phone_number
      }
    }

    if (!owner.value && phones.value.length > 0) {
      owner.value = phones.value[0]!.phone_number
    }
  }

  async function deletePhone(phoneID: string) {
    await apiFetch(`/v1/phones/${phoneID}`, { method: 'DELETE' })
    await loadPhones(true)
  }

  async function updatePhone(phone: EntitiesPhone) {
    try {
      const response = await apiFetch<{ message: string }>('/v1/phones', {
        method: 'PUT',
        body: {
          fcm_token: phone.fcm_token,
          sim: phone.sim,
          phone_number: phone.phone_number,
          message_expiration_seconds: parseInt(
            phone.message_expiration_seconds.toString(),
          ),
          missed_call_auto_reply: phone.missed_call_auto_reply,
          max_send_attempts: parseInt(phone.max_send_attempts.toString()),
          messages_per_minute: parseInt(phone.messages_per_minute.toString()),
          message_send_schedule_id: phone.message_send_schedule_id ?? null,
        },
      })
      notificationsStore.addNotification({
        message: response.message,
        type: 'success',
      })
      await loadPhones(true)
    } catch (error: unknown) {
      notificationsStore.addNotification({
        message: getApiErrorMessage(error, 'Error while updating phone'),
        type: 'error',
      })
    }
  }

  async function getHeartbeat(limit = 1): Promise<EntitiesHeartbeat[]> {
    const response = await apiFetch<{ data: EntitiesHeartbeat[] }>(
      '/v1/heartbeats',
      {
        query: { limit, owner: owner.value },
      },
    )
    if (response.data.length > 0) {
      heartbeat.value = response.data[0]!
    } else {
      heartbeat.value = null
    }
    return response.data
  }

  function resetState() {
    phones.value = []
    owner.value = null
    heartbeat.value = null
  }

  return {
    phones,
    owner,
    heartbeat,
    activePhone,
    setOwner,
    loadPhones,
    deletePhone,
    updatePhone,
    getHeartbeat,
    resetState,
  }
})
