import { defineStore } from 'pinia'
import type {
  EntitiesContact,
  EntitiesMessageThread,
  EntitiesMessage,
} from '~~/shared/types/api'

const PAGE_SIZE = 20

export const useThreadsStore = defineStore('threads', () => {
  const threads = ref<EntitiesMessageThread[]>([])
  const threadId = ref<string | null>(null)
  const loadingThreads = ref(true)
  const loadingMoreThreads = ref(false)
  const hasMoreThreads = ref(false)
  let pageCount = 1
  let pageContext = ''
  let loadMoreGeneration = 0
  const archivedThreads = ref(false)
  const unreadOnly = ref(false)
  const retainedThread = ref<EntitiesMessageThread | null>(null)
  const openedThreads = ref(new Map<string, EntitiesMessageThread>())
  const fetchedThreadIds = ref(new Set<string>())
  let threadRequestId = 0
  let stateGeneration = 0
  let readMutationGeneration = 0
  const { apiFetch } = useApi()
  const notificationsStore = useNotificationsStore()

  const currentThread = computed<EntitiesMessageThread | null>(() => {
    return (
      threads.value.find((x) => x.id === threadId.value) ??
      ((unreadOnly.value ||
        loadingThreads.value ||
        hasMoreThreads.value ||
        (threadId.value !== null &&
          fetchedThreadIds.value.has(threadId.value))) &&
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
    () => currentThread.value !== null && !loadingThreads.value,
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
    if (
      openedThreads.value.has(updatedThread.id) ||
      updatedThread.id === threadId.value
    ) {
      openedThreads.value.set(updatedThread.id, replacement)
    }
  }

  async function loadThreads(savedContacts: EntitiesContact[] = []) {
    const requestId = ++threadRequestId
    const readGeneration = readMutationGeneration
    const selectedThread = currentThread.value
    if (selectedThread) {
      openedThreads.value.set(selectedThread.id, selectedThread)
    }
    for (const openedThread of openedThreads.value.values()) {
      for (const contact of savedContacts) {
        const matchesNumber = contact.phone_numbers.includes(
          openedThread.contact,
        )
        if (matchesNumber || openedThread.contact_details?.id === contact.id) {
          openedThread.contact_details = matchesNumber ? contact : undefined
        }
      }
    }
    const phonesStore = usePhonesStore()
    if (phonesStore.owner === null && phonesStore.phones.length === 0) {
      loadingThreads.value = false
      hasMoreThreads.value = false
      return
    }

    const owner = phonesStore.owner ?? phonesStore.phones[0]?.phone_number
    const archived = archivedThreads.value
    const unread = unreadOnly.value
    const context = JSON.stringify([owner, archived, unread])
    if (context !== pageContext) {
      pageContext = context
      pageCount = 1
      hasMoreThreads.value = false
      loadingMoreThreads.value = false
      loadMoreGeneration++
    }

    try {
      const responses = await Promise.all(
        Array.from({ length: pageCount }, (_, page) =>
          apiFetch<{ data: EntitiesMessageThread[]; has_more: boolean }>(
            '/v1/message-threads',
            {
              params: {
                owner,
                limit: PAGE_SIZE,
                skip: page * PAGE_SIZE,
                is_archived: archived,
                unread_only: unread,
                contacts: true,
              },
            },
          ),
        ),
      )
      if (requestId !== threadRequestId) return
      if (readGeneration !== readMutationGeneration) {
        await loadThreads()
        return
      }

      phonesStore.getHeartbeat().catch(console.error)
      retainedThread.value = currentThread.value
      const combined = new Map<string, EntitiesMessageThread>()
      let hasMore = false
      for (const response of responses) {
        hasMore = response.has_more
        for (const thread of response.data) {
          if (!combined.has(thread.id)) combined.set(thread.id, thread)
        }
      }
      threads.value = [...combined.values()]
      hasMoreThreads.value = hasMore
      for (const thread of threads.value) {
        if (openedThreads.value.has(thread.id)) {
          openedThreads.value.set(thread.id, thread)
        }
      }
    } finally {
      if (requestId === threadRequestId) {
        loadingThreads.value = false
      }
    }
  }

  async function loadMoreThreads() {
    if (
      loadingThreads.value ||
      loadingMoreThreads.value ||
      !hasMoreThreads.value
    )
      return

    const generation = ++loadMoreGeneration
    const previousPageCount = pageCount
    pageCount++
    loadingMoreThreads.value = true
    try {
      await loadThreads()
    } catch (error) {
      if (generation !== loadMoreGeneration) return
      pageCount = previousPageCount
      throw error
    } finally {
      if (generation === loadMoreGeneration) loadingMoreThreads.value = false
    }
  }

  async function fetchThread(id: string): Promise<EntitiesMessageThread> {
    const generation = stateGeneration
    const response = await apiFetch<{ data: EntitiesMessageThread }>(
      `/v1/message-threads/${id}`,
    )
    if (generation !== stateGeneration) {
      throw new DOMException('Conversation lookup was canceled', 'AbortError')
    }
    openedThreads.value.set(response.data.id, response.data)
    fetchedThreadIds.value.add(response.data.id)
    return response.data
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
    const previousThread = currentThread.value
    if (previousThread) {
      openedThreads.value.set(previousThread.id, previousThread)
    }
    const cached = id === null ? undefined : openedThreads.value.get(id)
    const phonesStore = usePhonesStore()
    const owner = phonesStore.owner ?? phonesStore.phones[0]?.phone_number
    retainedThread.value =
      id === null
        ? null
        : (threads.value.find((thread) => thread.id === id) ??
          ((unreadOnly.value ||
            loadingThreads.value ||
            hasMoreThreads.value ||
            fetchedThreadIds.value.has(id)) &&
          cached &&
          cached.owner === owner &&
          cached.is_archived === archivedThreads.value
            ? cached
            : null))
    if (retainedThread.value) {
      openedThreads.value.set(retainedThread.value.id, retainedThread.value)
    }
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
    openedThreads.value.delete(payload.threadId)
    fetchedThreadIds.value.delete(payload.threadId)
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
      readMutationGeneration++
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
    openedThreads.value.delete(id)
    fetchedThreadIds.value.delete(id)
    notificationsStore.addNotification({
      message: 'The message thread has been deleted successfully',
      type: 'success',
    })
  }

  function resetState() {
    threadRequestId++
    stateGeneration++
    readMutationGeneration = 0
    loadMoreGeneration++
    pageCount = 1
    pageContext = ''
    hasMoreThreads.value = false
    loadingMoreThreads.value = false
    threads.value = []
    threadId.value = null
    retainedThread.value = null
    openedThreads.value.clear()
    fetchedThreadIds.value.clear()
    archivedThreads.value = false
    unreadOnly.value = false
    loadingThreads.value = true
  }

  return {
    threads,
    threadId,
    loadingThreads,
    loadingMoreThreads,
    hasMoreThreads,
    archivedThreads,
    unreadOnly,
    visibleThreads,
    currentThread,
    hasThread,
    hasThreadId,
    loadThreads,
    loadMoreThreads,
    fetchThread,
    loadThreadMessages,
    setThreadId,
    toggleArchive,
    updateThread,
    resetThreadUnreadCount,
    deleteThread,
    resetState,
  }
})
