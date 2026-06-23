import { defineStore } from 'pinia'
import type { EntitiesMessage } from '~~/shared/types/api'
import type { SearchMessagesRequest } from '~~/shared/types/message'
import type { BulkMessageOrder } from '~~/shared/types/bulk-message'
import { getApiErrorMessage } from '~/utils/api-error'

export type SIM = 'SIM1' | 'SIM2' | 'DEFAULT'

export interface SendMessageRequest {
  from: string
  to: string
  content: string
  sim: SIM
  request_id?: string
}

export const useMessagesStore = defineStore('messages', () => {
  const { apiFetch } = useApi()
  const notificationsStore = useNotificationsStore()

  async function sendMessage(request: SendMessageRequest) {
    try {
      const response = await apiFetch<{ message: string }>(
        '/v1/messages/send',
        {
          method: 'POST',
          body: request,
        },
      )
      notificationsStore.addNotification({
        message: response.message,
        type: 'success',
      })
    } catch (e: unknown) {
      notificationsStore.addNotification({
        message: getApiErrorMessage(e, 'Error while sending message'),
        type: 'error',
      })
    }
    const threadsStore = useThreadsStore()
    await threadsStore.loadThreads()
  }

  async function deleteMessage(messageId: string) {
    await apiFetch(`/v1/messages/${messageId}`, { method: 'DELETE' })
    notificationsStore.addNotification({
      message: 'The message has been deleted successfully',
      type: 'success',
    })
  }

  async function searchMessages(
    payload: SearchMessagesRequest,
  ): Promise<EntitiesMessage[]> {
    const token = payload.token
    const params = { ...payload }
    delete params.token

    const response = await apiFetch<{ data: EntitiesMessage[] }>(
      '/v1/messages/search',
      {
        params,
        headers: token ? { token } : undefined,
      },
    )
    return response.data
  }

  async function sendBulkMessages(document: File): Promise<void> {
    const formData = new FormData()
    formData.append('document', document)
    const response = await apiFetch<{ message?: string }>('/v1/bulk-messages', {
      method: 'POST',
      body: formData,
    })
    notificationsStore.addNotification({
      message: response.message ?? 'Bulk messages sent successfully',
      type: 'success',
    })
  }

  async function fetchBulkMessageOrders(): Promise<BulkMessageOrder[]> {
    const response = await apiFetch<{ data: BulkMessageOrder[] }>(
      '/v1/bulk-messages',
    )
    return response.data ?? []
  }

  return {
    sendMessage,
    deleteMessage,
    searchMessages,
    sendBulkMessages,
    fetchBulkMessageOrders,
  }
})
