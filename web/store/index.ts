import { ActionContext } from 'vuex'
import { MessageThread } from '~/models/message-thread'
import { Message } from '~/models/message'
import { Heartbeat } from '~/models/heartbeat'
import axios from '~/plugins/axios'
import { Phone } from '~/models/phone'

const defaultNotificationTimeout = 3000

type NotificationType = 'error' | 'success' | 'info'

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

export type User = {
  id: string
}

export type State = {
  owner: string
  user: User | null
  phones: Array<Phone>
  threads: Array<MessageThread>
  threadId: string | null
  heartbeat: null | Heartbeat
  pooling: boolean
  notification: Notification
  threadMessages: Array<Message>
}

export const state = (): State => ({
  threads: [],
  threadId: null,
  heartbeat: null,
  pooling: false,
  threadMessages: [],
  phones: [],
  owner: '+37259139660',
  user: null,
  notification: {
    active: false,
    message: '',
    type: 'success',
    timeout: defaultNotificationTimeout,
  },
})

export type AppData = {
  url: string
  name: string
  documentationUrl: string
  githubUrl: string
}

export const getters = {
  getThreads(state: State): Array<MessageThread> {
    return state.threads
  },

  getAppData(): AppData {
    let url = process.env.APP_URL as string
    if (url.length > 0 && url[url.length - 1] === '/') {
      url = url.substring(0, url.length - 1)
    }
    return {
      url,
      documentationUrl: process.env.APP_DOCUMENTATION_URL as string,
      githubUrl: process.env.APP_GITHUB_URL as string,
      name: process.env.APP_NAME as string,
    }
  },

  getUser(state: State): User | null {
    return state.user
  },

  getOwner(state: State): string {
    return state.owner
  },

  getPhones(state: State): Array<Phone> {
    return state.phones
  },

  hasThread(state: State): boolean {
    return state.threadId != null
  },

  getThreadMessages(state: State): Array<Message> {
    return state.threadMessages
  },

  getThread(state: State): MessageThread {
    const thread = state.threads.find((x) => x.id === state.threadId)
    if (thread === undefined) {
      throw new Error(`cannot find thread with id ${state.threadId}`)
    }
    return thread
  },

  getHeartbeat(state: State): Heartbeat | null {
    return state.heartbeat
  },

  getPolling(state: State): boolean {
    return state.pooling
  },

  getNotification(state: State): Notification {
    return state.notification
  },
}

export const mutations = {
  setThreads(state: State, payload: Array<MessageThread>) {
    state.threads = [...payload]
  },
  setThreadId(state: State, payload: string | null) {
    state.threadId = payload
  },
  setThreadMessages(state: State, payload: Array<Message>) {
    state.threadMessages = payload
  },
  setHeartbeat(state: State, payload: Heartbeat | null) {
    state.heartbeat = payload
  },
  setPooling(state: State, payload: boolean) {
    state.pooling = payload
  },
  setUser(state: State, payload: User | null) {
    state.user = payload
  },
  setNotification(state: State, notification: NotificationRequest) {
    state.notification = {
      ...state.notification,
      active: true,
      message: notification.message,
      type: notification.type,
      timeout: Math.floor(Math.random() * 100) + defaultNotificationTimeout, // Reset the timeout
    }
  },
  disableNotification(state: State) {
    state.notification.active = false
  },

  setPhones(state: State, payload: Array<Phone>) {
    state.phones = payload

    const owner = payload.find((x) => x.phone_number === state.owner)
    if (!owner && state.phones.length > 0) {
      state.owner = state.phones[0].phone_number
    }
  },
}

export type SendMessageRequest = {
  from: string
  to: string
  content: string
}

export const actions = {
  async loadThreads(context: ActionContext<State, State>) {
    const response = await axios.get('/v1/message-threads', {
      params: {
        owner: context.getters.getOwner,
      },
    })
    context.commit('setThreads', response.data.data)
  },

  async loadPhones(context: ActionContext<State, State>, force: boolean) {
    if (context.getters.getPhones.length > 0 && !force) {
      return
    }
    const response = await axios.get('/v1/phones')
    context.commit('setPhones', response.data.data)
  },

  async getHeartbeat(context: ActionContext<State, State>) {
    const response = await axios.get('/v1/heartbeats', {
      params: {
        limit: 1,
        owner: context.getters.getOwner,
      },
    })

    if (response.data.data.length > 0) {
      context.commit('setHeartbeat', response.data.data[0])
      return
    }

    context.commit('setHeartbeat', null)
  },

  setPolling(context: ActionContext<State, State>, status: boolean) {
    context.commit('setPooling', status)
  },

  async sendMessage(
    context: ActionContext<State, State>,
    request: SendMessageRequest
  ) {
    await axios.post('/v1/messages/send', request)
    await Promise.all([
      context.dispatch('loadThreadMessages', context.getters.getThread.id),
      context.dispatch('loadThreads'),
    ])
  },

  setThreadId(context: ActionContext<State, State>, threadId: string | null) {
    context.commit('setThreadId', threadId)
  },

  addNotification(
    context: ActionContext<State, State>,
    request: NotificationRequest
  ) {
    context.commit('setNotification', request)
  },

  disableNotification(context: ActionContext<State, State>) {
    context.commit('disableNotification')
  },

  async loadThreadMessages(
    context: ActionContext<State, State>,
    threadId: string | null
  ) {
    await context.commit('setThreadId', threadId)
    const response = await axios.get('/v1/messages', {
      params: {
        contact: context.getters.getThread.contact,
        owner: context.getters.getThread.owner,
      },
    })
    context.commit('setThreadMessages', response.data.data)
  },

  setUser(context: ActionContext<State, State>, user: User | null) {
    context.commit('setUser', user)
  },
}
