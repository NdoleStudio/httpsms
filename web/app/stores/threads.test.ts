import assert from 'node:assert/strict'
import { test } from 'node:test'
import { createPinia, setActivePinia } from 'pinia'
import { computed, ref } from 'vue'
import type { EntitiesMessageThread } from '../../shared/types/api'
import { useThreadsStore } from './threads'

function thread(id: string, unreadCount: number): EntitiesMessageThread {
  return {
    id,
    unread_count: unreadCount,
    owner: '+18005550199',
    contact: '+18005550100',
    color: 'indigo',
    is_archived: false,
    last_message_content: 'Hello',
    last_message_id: 'message-id',
    status: 'received',
    created_at: '2026-10-27T10:00:00Z',
    updated_at: '2026-10-27T10:00:00Z',
    order_timestamp: '2026-10-27T10:00:00Z',
    user_id: 'user-id',
  }
}

function setupStore() {
  const calls: { path: string; params?: Record<string, unknown> }[] = []
  let responseThreads = [thread('unread', 2), thread('read', 0)]
  let nextResponse: Promise<{ data: EntitiesMessageThread[] }> | undefined
  Object.assign(globalThis, {
    ref,
    computed,
    useApi: () => ({
      apiFetch: async (
        path: string,
        options: {
          params?: Record<string, unknown>
          method?: string
        },
      ) => {
        calls.push({ path, params: options.params })
        if (options.method === 'PUT') {
          return { data: thread('unread', 0) }
        }
        if (path === '/v1/messages') return { data: [] }
        const pending = nextResponse
        nextResponse = undefined
        return pending ?? { data: responseThreads }
      },
    }),
    usePhonesStore: () => ({
      owner: '+18005550199',
      phones: [],
      getHeartbeat: async () => [],
    }),
    useNotificationsStore: () => ({ addNotification: () => {} }),
  })
  setActivePinia(createPinia())
  return {
    store: useThreadsStore(),
    calls,
    respondWith: (threads: EntitiesMessageThread[]) => {
      responseThreads = threads
    },
    deferNextLoad: (response: Promise<{ data: EntitiesMessageThread[] }>) => {
      nextResponse = response
    },
  }
}

test('Unread requests server-side filtering within the API limit', async () => {
  const { store, calls } = setupStore()
  store.unreadOnly = true
  store.archivedThreads = true

  await store.loadThreads()

  assert.equal(calls[0]?.params?.unread_only, true)
  assert.equal(calls[0]?.params?.is_archived, true)
  assert.equal(calls[0]?.params?.owner, '+18005550199')
  assert.equal(calls[0]?.params?.limit, 20)
})

test('marking a thread read hides it from Unread without closing it', async () => {
  const { store, calls, respondWith } = setupStore()
  await store.loadThreads()
  store.setThreadId('unread')
  store.unreadOnly = true

  await store.resetThreadUnreadCount('unread')

  assert.ok(store.visibleThreads)
  assert.deepEqual(store.visibleThreads, [])
  assert.equal(store.currentThread?.id, 'unread')
  assert.equal(store.currentThread.unread_count, 0)

  respondWith([])
  await store.loadThreads()
  assert.equal(store.currentThread?.id, 'unread')
  await store.loadThreadMessages('unread')
  assert.equal(calls.at(-1)?.path, '/v1/messages')
})

test('switching back to All restores read conversations', async () => {
  const { store } = setupStore()
  await store.loadThreads()
  store.unreadOnly = true

  assert.ok(store.visibleThreads)
  assert.deepEqual(
    store.visibleThreads.map((item) => item.id),
    ['unread'],
  )

  store.unreadOnly = false
  await store.loadThreads()
  assert.deepEqual(
    store.visibleThreads.map((item) => item.id),
    ['unread', 'read'],
  )
})

test('resetting the store clears the filter and retained conversation', async () => {
  const { store, respondWith } = setupStore()
  await store.loadThreads()
  store.setThreadId('unread')
  store.unreadOnly = true
  respondWith([])
  await store.loadThreads()

  store.resetState()

  assert.equal(store.unreadOnly, false)
  assert.equal(store.currentThread, null)
  assert.deepEqual(store.threads, [])
})

test('an older polling response cannot overwrite a newer unread-filter load', async () => {
  const { store, respondWith, deferNextLoad } = setupStore()
  const pending = Promise.withResolvers<{ data: EntitiesMessageThread[] }>()
  deferNextLoad(pending.promise)
  const oldLoad = store.loadThreads()
  store.unreadOnly = true
  respondWith([thread('unread', 2)])
  await store.loadThreads()

  pending.resolve({ data: [thread('read', 0)] })
  await oldLoad

  assert.deepEqual(
    store.threads.map((item) => item.id),
    ['unread'],
  )
})

test('All does not retain a conversation removed from the server list', async () => {
  const { store, respondWith } = setupStore()
  await store.loadThreads()
  store.setThreadId('unread')
  respondWith([])

  await store.loadThreads()

  assert.equal(store.currentThread, null)
})
