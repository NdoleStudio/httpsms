import { defineStore } from 'pinia'
import type {
  EntitiesMessageThread,
  EntitiesMessage,
} from '~~/shared/types/api'

export const useThreadsStore = defineStore('threads', () => {
  const threads = ref<EntitiesMessageThread[]>([])
  const threadId = ref<string | null>(null)
  const loadingThreads = ref(true)
  const archivedThreads = ref(false)
  const { apiFetch } = useApi()
  const notificationsStore = useNotificationsStore()

  const currentThread = computed<EntitiesMessageThread | null>(() => {
    return threads.value.find((x) => x.id === threadId.value) ?? null
  })

  const hasThread = computed(
    () => threadId.value != null && !loadingThreads.value,
  )

  function hasThreadId(id: string): boolean {
    return threads.value.find((x) => x.id === id) !== undefined
  }

  function replaceThread(updatedThread: EntitiesMessageThread) {
    const index = threads.value.findIndex(
      (thread) => thread.id === updatedThread.id,
    )
    if (index !== -1) threads.value[index] = updatedThread
  }

  async function loadThreads() {
    const phonesStore = usePhonesStore()
    if (phonesStore.owner === null && phonesStore.phones.length === 0) {
      loadingThreads.value = false
      return
    }

    const response = await apiFetch<{ data: EntitiesMessageThread[] }>(
      '/v1/message-threads',
      {
        params: {
          owner: phonesStore.owner ?? phonesStore.phones[0]?.phone_number,
          limit: 100,
          is_archived: archivedThreads.value,
        },
      },
    )

    phonesStore.getHeartbeat().catch(console.error)
    threads.value = [...response.data]
    loadingThreads.value = false
  }

  async function loadThreadMessages(
    id: string | null,
  ): Promise<EntitiesMessage[]> {
    threadId.value = id
    const thread = currentThread.value
    if (!thread) throw new Error(`Cannot find thread with id ${id}`)

    const response = await apiFetch<{ data: EntitiesMessage[] }>(
      '/v1/messages',
      {
        params: {
          contact: thread.contact,
          owner: thread.owner,
          limit: 50,
        },
      },
    )
    return response.data
  }

  function setThreadId(id: string | null) {
    threadId.value = id
  }

  function toggleArchive() {
    archivedThreads.value = !archivedThreads.value
  }

  async function updateThread(payload: {
    threadId: string
    isArchived: boolean
  }) {
    await apiFetch(`/v1/message-threads/${payload.threadId}`, {
      method: 'PUT',
      body: { is_archived: payload.isArchived },
    })
    threads.value = threads.value.filter(
      (thread) => thread.id !== payload.threadId,
    )
    threadId.value = null
    notificationsStore.addNotification({
      message: payload.isArchived ? 'Archived' : 'Unarchived',
      type: 'success',
    })
  }

  async function markThreadRead(threadId: string, force = false) {
    const thread = threads.value.find((item) => item.id === threadId)
    if (!thread) throw new Error(`Cannot find thread with id ${threadId}`)
    if (!force && thread.is_read) return

    try {
      const response = await apiFetch<{ data: EntitiesMessageThread }>(
        `/v1/message-threads/${threadId}`,
        {
          method: 'PUT',
          body: { is_read: true },
        },
      )
      replaceThread(response.data)
    } catch (error) {
      notificationsStore.addNotification({
        message: 'The message thread could not be marked as read',
        type: 'error',
      })
      try {
        await loadThreads()
      } catch (reloadError) {
        throw new AggregateError(
          [error, reloadError],
          'Could not mark the message thread as read or reload threads',
          { cause: reloadError },
        )
      }
      throw error
    }
  }

  async function deleteThread(id: string) {
    await apiFetch(`/v1/message-threads/${id}`, { method: 'DELETE' })
    threadId.value = null
    notificationsStore.addNotification({
      message: 'The message thread has been deleted successfully',
      type: 'success',
    })
  }

  function resetState() {
    threads.value = []
    threadId.value = null
    archivedThreads.value = false
    loadingThreads.value = true
  }

  return {
    threads,
    threadId,
    loadingThreads,
    archivedThreads,
    currentThread,
    hasThread,
    hasThreadId,
    loadThreads,
    loadThreadMessages,
    setThreadId,
    toggleArchive,
    updateThread,
    markThreadRead,
    deleteThread,
    resetState,
  }
})
