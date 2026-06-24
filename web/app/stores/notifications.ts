import { defineStore } from 'pinia'

export type NotificationType = 'error' | 'success' | 'info'

export interface Notification {
  message: string
  timeout: number
  active: boolean
  type: NotificationType
}

export interface NotificationRequest {
  message: string
  type: NotificationType
}

const DEFAULT_TIMEOUT = 3000

export const useNotificationsStore = defineStore('notifications', () => {
  const notification = ref<Notification>({
    active: false,
    message: '',
    type: 'success',
    timeout: DEFAULT_TIMEOUT,
  })

  function addNotification(request: NotificationRequest) {
    notification.value = {
      active: true,
      message: request.message,
      type: request.type,
      timeout: Math.floor(Math.random() * 100) + DEFAULT_TIMEOUT,
    }
  }

  function disableNotification() {
    notification.value.active = false
  }

  return {
    notification,
    addNotification,
    disableNotification,
  }
})
