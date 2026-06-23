import { defineStore } from 'pinia'
import type { BillingUsage } from '~~/shared/types/billing'
import type { User } from '~~/shared/types/user'
import type {
  EntitiesWebhook,
  EntitiesDiscord,
  EntitiesMessageSendSchedule,
  EntitiesPhoneAPIKey,
  RequestsWebhookStore,
  RequestsWebhookUpdate,
  RequestsDiscordStore,
  RequestsDiscordUpdate,
  RequestsMessageSendScheduleStore,
  RequestsUserNotificationUpdate,
  RequestsUserPaymentInvoice,
  ResponsesUserSubscriptionPaymentsResponse,
} from '~~/shared/types/api'

export const useBillingStore = defineStore('billing', () => {
  const billingUsage = ref<BillingUsage | null>(null)
  const billingUsageHistory = ref<BillingUsage[]>([])
  const { apiFetch } = useApi()
  const notificationsStore = useNotificationsStore()

  async function loadBillingUsage() {
    const response = await apiFetch<{ data: BillingUsage }>('/v1/billing/usage')
    billingUsage.value = response.data
  }

  async function loadBillingUsageHistory() {
    const response = await apiFetch<{ data: BillingUsage[] }>(
      '/v1/billing/usage-history',
    )
    billingUsageHistory.value = response.data
  }

  async function getSubscriptionUpdateLink(): Promise<string> {
    const response = await apiFetch<{ data: string }>(
      '/v1/users/subscription-update-url',
    )
    return response.data
  }

  async function cancelSubscription(): Promise<string> {
    const response = await apiFetch<{ message: string }>(
      '/v1/users/subscription',
      {
        method: 'DELETE',
      },
    )
    return response.message
  }

  async function indexSubscriptionPayments(): Promise<ResponsesUserSubscriptionPaymentsResponse> {
    const response = await apiFetch<ResponsesUserSubscriptionPaymentsResponse>(
      '/v1/users/subscription/payments',
      { params: { limit: 100 } },
    )
    return response
  }

  async function generateSubscriptionPaymentInvoice(
    subscriptionInvoiceId: string,
    request: RequestsUserPaymentInvoice,
  ): Promise<void> {
    const response = await apiFetch(
      `/v1/users/subscription/invoices/${subscriptionInvoiceId}`,
      {
        method: 'POST',
        body: request,
        responseType: 'blob',
      },
    )

    const pdfBlob = new Blob([response as Blob], { type: 'application/pdf' })
    const url = window.URL.createObjectURL(pdfBlob)
    const tempLink = document.createElement('a')
    tempLink.href = url
    tempLink.setAttribute('download', 'Invoice.pdf')
    document.body.appendChild(tempLink)
    tempLink.click()
    document.body.removeChild(tempLink)
    window.URL.revokeObjectURL(url)
  }

  // Webhooks
  async function createWebhook(
    payload: RequestsWebhookStore,
  ): Promise<EntitiesWebhook> {
    const response = await apiFetch<{ data: EntitiesWebhook }>('/v1/webhooks', {
      method: 'POST',
      body: payload,
    })
    return response.data
  }

  async function getWebhooks(): Promise<EntitiesWebhook[]> {
    const response = await apiFetch<{ data: EntitiesWebhook[] }>(
      '/v1/webhooks',
      {
        params: { limit: 100 },
      },
    )
    return response.data
  }

  async function updateWebhook(
    payload: RequestsWebhookUpdate & { id: string },
  ): Promise<EntitiesWebhook> {
    const response = await apiFetch<{ data: EntitiesWebhook }>(
      `/v1/webhooks/${payload.id}`,
      {
        method: 'PUT',
        body: payload,
      },
    )
    return response.data
  }

  async function deleteWebhook(id: string): Promise<void> {
    await apiFetch(`/v1/webhooks/${id}`, { method: 'DELETE' })
  }

  // Discord
  async function createDiscord(
    payload: RequestsDiscordStore,
  ): Promise<EntitiesDiscord> {
    const response = await apiFetch<{ data: EntitiesDiscord }>(
      '/v1/discord-integrations',
      {
        method: 'POST',
        body: payload,
      },
    )
    return response.data
  }

  async function getDiscordIntegrations(): Promise<EntitiesDiscord[]> {
    const response = await apiFetch<{ data: EntitiesDiscord[] }>(
      '/v1/discord-integrations',
      {
        params: { limit: 100 },
      },
    )
    return response.data
  }

  async function updateDiscordIntegration(
    payload: RequestsDiscordUpdate & { id: string },
  ): Promise<EntitiesDiscord> {
    const response = await apiFetch<{ data: EntitiesDiscord }>(
      `/v1/discord-integrations/${payload.id}`,
      {
        method: 'PUT',
        body: payload,
      },
    )
    return response.data
  }

  async function deleteDiscordIntegration(id: string): Promise<void> {
    await apiFetch(`/v1/discord-integrations/${id}`, { method: 'DELETE' })
  }

  // Send Schedules
  async function getSendSchedules(): Promise<EntitiesMessageSendSchedule[]> {
    const response = await apiFetch<{ data: EntitiesMessageSendSchedule[] }>(
      '/v1/send-schedules',
    )
    return response.data
  }

  async function createSendSchedule(
    payload: RequestsMessageSendScheduleStore,
  ): Promise<EntitiesMessageSendSchedule> {
    const response = await apiFetch<{ data: EntitiesMessageSendSchedule }>(
      '/v1/send-schedules',
      {
        method: 'POST',
        body: payload,
      },
    )
    return response.data
  }

  async function updateSendSchedule(
    payload: RequestsMessageSendScheduleStore & { id: string },
  ): Promise<EntitiesMessageSendSchedule> {
    const response = await apiFetch<{ data: EntitiesMessageSendSchedule }>(
      `/v1/send-schedules/${payload.id}`,
      {
        method: 'PUT',
        body: payload,
      },
    )
    return response.data
  }

  async function deleteSendSchedule(id: string): Promise<void> {
    await apiFetch(`/v1/send-schedules/${id}`, { method: 'DELETE' })
  }

  // Phone API Keys
  async function storePhoneApiKey(name: string): Promise<EntitiesPhoneAPIKey> {
    const response = await apiFetch<{
      data: EntitiesPhoneAPIKey
      message: string
    }>('/v1/phone-api-keys', {
      method: 'POST',
      body: { name },
    })
    notificationsStore.addNotification({
      message: response.message,
      type: 'success',
    })
    return response.data
  }

  async function indexPhoneApiKeys(): Promise<EntitiesPhoneAPIKey[]> {
    const response = await apiFetch<{ data: EntitiesPhoneAPIKey[] }>(
      '/v1/phone-api-keys',
      {
        params: { limit: 100 },
      },
    )
    return response.data
  }

  async function deletePhoneApiKey(id: string): Promise<void> {
    const response = await apiFetch<{ message: string }>(
      `/v1/phone-api-keys/${id}`,
      { method: 'DELETE' },
    )
    notificationsStore.addNotification({
      message: response.message,
      type: 'success',
    })
  }

  async function deletePhoneFromPhoneApiKey(
    phoneApiKeyId: string,
    phoneId: string,
  ): Promise<void> {
    const response = await apiFetch<{ message: string }>(
      `/v1/phone-api-keys/${phoneApiKeyId}/phones/${phoneId}`,
      { method: 'DELETE' },
    )
    notificationsStore.addNotification({
      message: response.message,
      type: 'success',
    })
  }

  // Email notifications
  async function saveEmailNotifications(
    userId: string,
    payload: RequestsUserNotificationUpdate,
  ): Promise<void> {
    const authStore = useAuthStore()
    const response = await apiFetch<{ data: User }>(
      `/v1/users/${userId}/notifications`,
      {
        method: 'PUT',
        body: payload,
      },
    )
    authStore.user = response.data
  }

  return {
    billingUsage,
    billingUsageHistory,
    loadBillingUsage,
    loadBillingUsageHistory,
    getSubscriptionUpdateLink,
    cancelSubscription,
    indexSubscriptionPayments,
    generateSubscriptionPaymentInvoice,
    createWebhook,
    getWebhooks,
    updateWebhook,
    deleteWebhook,
    createDiscord,
    getDiscordIntegrations,
    updateDiscordIntegration,
    deleteDiscordIntegration,
    getSendSchedules,
    createSendSchedule,
    updateSendSchedule,
    deleteSendSchedule,
    storePhoneApiKey,
    indexPhoneApiKeys,
    deletePhoneApiKey,
    deletePhoneFromPhoneApiKey,
    saveEmailNotifications,
  }
})
