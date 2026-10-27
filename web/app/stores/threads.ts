import { defineStore } from 'pinia'
import type {
  EntitiesContact,
  EntitiesMessageThread,
  EntitiesMessage,
} from '~~/shared/types/api'

export const useThreadsStore = defineStore('threads', () => {
  const threads = ref<EntitiesMessageThread[]>([])
  const threadId = ref<string | null>(null)
  const loadingThreads = ref(true)
  const archivedThreads = ref(false)
  const unreadOnly = ref(false)
  const retainedThread = ref<EntitiesMessageThread | null>(null)
  let threadRequestId = 0
  const { apiFetch } = useApi()
  const notificationsStore = useNotificationsStore()

  const currentThread = computed<EntitiesMessageThread | null>(() => {
    return (
      threads.value.find((x) => x.id === threadId.value) ??
      ((unreadOnly.value || loadingThreads.value) &&
      retainedThread.value?.id === threadId.value
        ? retainedThread.value
        : null)
    )
  })

  const visibleThreads = computed(() =>
    unreadOnly.value
      ? threads.value.filter((thread) => thread.unread_count > 0)
      : threads.value,
  )

  const hasThread = computed(
    () => threadId.value != null && !loadingThreads.value,
  )

  function hasThreadId(id: string): boolean {
    return (
      threads.value.some((thread) => thread.id === id) ||
      currentThread.value?.id === id
    )
  }

  function replaceThread(updatedThread: EntitiesMessageThread) {
    const index = threads.value.findIndex(
      (thread) => thread.id === updatedThread.id,
    )
    const existingThread =
      index !== -1 ? threads.value[index] : retainedThread.value
    const replacement = {
      ...updatedThread,
      // Mark-read PUT responses are not contact-enriched.
      contact_details:
        updatedThread.contact_details ??
        (updatedThread.contact === existingThread?.contact
          ? existingThread.contact_details
          : undefined),
    }
    if (index !== -1) {
      threads.value[index] = replacement
    }
    if (updatedThread.id === threadId.value) {
      retainedThread.value = replacement
    }
  }

  async function loadThreads(savedContacts: EntitiesContact[] = []) {
    const requestId = ++threadRequestId
    const selectedThread = currentThread.value
    if (selectedThread) {
      for (const contact of savedContacts) {
        const matchesNumber = contact.phone_numbers.includes(
          selectedThread.contact,
        )
        if (
          matchesNumber ||
          selectedThread.contact_details?.id === contact.id
        ) {
          selectedThread.contact_details = matchesNumber ? contact : undefined
        }
      }
    }
    const phonesStore = usePhonesStore()
    if (phonesStore.owner === null && phonesStore.phones.length === 0) {
      loadingThreads.value = false
      return
    }

    try {
      const response = await apiFetch<{ data: EntitiesMessageThread[] }>(
        '/v1/message-threads',
        {
          params: {
            owner: phonesStore.owner ?? phonesStore.phones[0]?.phone_number,
            limit: 20,
            is_archived: archivedThreads.value,
            unread_only: unreadOnly.value,
            contacts: true,
          },
        },
      )
      if (requestId !== threadRequestId) return

      phonesStore.getHeartbeat().catch(console.error)
      retainedThread.value = currentThread.value
      threads.value = [...response.data]
    } finally {
      if (requestId === threadRequestId) {
        loadingThreads.value = false
      }
    }
  }

  async function loadThreadMessages(
    id: string | null,
  ): Promise<EntitiesMessage[]> {
    setThreadId(id)
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
    retainedThread.value =
      id === null
        ? null
        : (threads.value.find((thread) => thread.id === id) ??
          (retainedThread.value?.id === id ? retainedThread.value : null))
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
    setThreadId(null)
    notificationsStore.addNotification({
      message: payload.isArchived ? 'Archived' : 'Unarchived',
      type: 'success',
    })
  }

  async function resetThreadUnreadCount(threadId: string, force = false) {
    const thread =
      threads.value.find((item) => item.id === threadId) ??
      (retainedThread.value?.id === threadId ? retainedThread.value : null)
    if (!thread) throw new Error(`Cannot find thread with id ${threadId}`)
    if (!force && thread.unread_count === 0) return

    try {
      const response = await apiFetch<{ data: EntitiesMessageThread }>(
        `/v1/message-threads/${threadId}`,
        {
          method: 'PUT',
          body: { unread_count: 0 },
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
    setThreadId(null)
    notificationsStore.addNotification({
      message: 'The message thread has been deleted successfully',
      type: 'success',
    })
  }

  function resetState() {
    threadRequestId++
    threads.value = []
    threadId.value = null
    retainedThread.value = null
    archivedThreads.value = false
    unreadOnly.value = false
    loadingThreads.value = true
  }

  return {
    threads,
    threadId,
    loadingThreads,
    archivedThreads,
    unreadOnly,
    visibleThreads,
    currentThread,
    hasThread,
    hasThreadId,
    loadThreads,
    loadThreadMessages,
    setThreadId,
    toggleArchive,
    updateThread,
    resetThreadUnreadCount,
    deleteThread,
    resetState,
  }
})
