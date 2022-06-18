import { ActionContext } from 'vuex'
import axios from '~/plugins/axios'
import { MessageThread } from '~/models/message-thread'
import { Message } from '~/models/message'

type State = {
  owner: string
  threads: Array<MessageThread>
  threadId: string | null
  threadMessages: Array<Message>
}

export const state = (): State => ({
  threads: [],
  threadId: null,
  threadMessages: [],
  owner: '+37259139660',
})

export const getters = {
  getThreads(state: State): Array<MessageThread> {
    return state.threads
  },
  getOwner(state: State): string {
    return state.owner
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
}
