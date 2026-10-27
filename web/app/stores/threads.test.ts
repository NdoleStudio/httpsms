import assert from 'node:assert/strict'
import { test } from 'node:test'
import { createPinia, setActivePinia } from 'pinia'
import { computed, ref } from 'vue'
import type {
  EntitiesContact,
  EntitiesMessageThread,
} from '../../shared/types/api'
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
  const phones = {
    owner: '+18005550199',
    phones: [],
    getHeartbeat: async () => [],
  }
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
          responseThreads = responseThreads.map((item) =>
            item.id === 'unread' ? { ...item, unread_count: 0 } : item,
          )
          return { data: thread('unread', 0) }
        }
        if (path === '/v1/messages') return { data: [] }
        const pending = nextResponse
        nextResponse = undefined
        if (pending) return { ...(await pending), has_more: false }
        const skip = Number(options.params?.skip ?? 0)
        const limit = Number(options.params?.limit ?? 20)
        return {
          data: responseThreads.slice(skip, skip + limit),
          has_more: responseThreads.length > skip + limit,
        }
      },
    }),
    usePhonesStore: () => phones,
    useNotificationsStore: () => ({ addNotification: () => {} }),
  })
  setActivePinia(createPinia())
  return {
    store: useThreadsStore(),
    phones,
    calls,
    respondWith: (threads: EntitiesMessageThread[]) => {
      responseThreads = threads
    },
    deferNextLoad: (response: Promise<{ data: EntitiesMessageThread[] }>) => {
      nextResponse = response
    },
  }
}

function contact(
  name: string,
  phoneNumbers = ['+18005550100'],
): EntitiesContact {
  return {
    id: 'contact-id',
    name,
    phone_numbers: phoneNumbers,
    emails: [],
    properties: {},
    user_id: 'user-id',
    created_at: '2026-10-27T10:00:00Z',
    updated_at: '2026-10-27T11:00:00Z',
  }
}

async function setupReadConversationHistory() {
  const fixture = setupStore()
  const other = { ...thread('other', 1), contact: '+18005550101' }
  fixture.respondWith([thread('unread', 1), other])
  await fixture.store.loadThreads()
  fixture.store.setThreadId('unread')
  fixture.store.unreadOnly = true
  await fixture.store.resetThreadUnreadCount('unread')
  fixture.respondWith([other])
  await fixture.store.loadThreads()
  await fixture.store.loadThreadMessages('other')
  return fixture
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

for (const scenario of [
  {
    name: 'adding a contact refreshes a retained read conversation',
    existing: undefined,
    saved: contact('New contact'),
    expectedName: 'New contact',
  },
  {
    name: 'editing a contact refreshes a retained read conversation',
    existing: contact('Old name'),
    saved: contact('Updated name'),
    expectedName: 'Updated name',
  },
  {
    name: 'removing the conversation number clears its retained contact',
    existing: contact('Old name'),
    saved: contact('Updated name', ['+18005550101']),
    expectedName: undefined,
  },
]) {
  test(scenario.name, async () => {
    const { store, respondWith } = setupStore()
    respondWith([
      { ...thread('unread', 1), contact_details: scenario.existing },
    ])
    await store.loadThreads()
    store.setThreadId('unread')
    store.unreadOnly = true
    await store.resetThreadUnreadCount('unread')
    respondWith([])
    await store.loadThreads()

    await store.loadThreads([scenario.saved])

    assert.equal(store.currentThread?.id, 'unread')
    assert.equal(
      store.currentThread.contact_details?.name,
      scenario.expectedName,
    )
    assert.deepEqual(store.visibleThreads, [])
  })
}

test('a late poll cannot restore unread counts after mark-read succeeds', async () => {
  const { store, deferNextLoad } = setupStore()
  await store.loadThreads()
  store.setThreadId('unread')
  store.unreadOnly = true
  const pending = Promise.withResolvers<{ data: EntitiesMessageThread[] }>()
  deferNextLoad(pending.promise)
  store.loadingThreads = true
  const poll = store.loadThreads()

  await store.resetThreadUnreadCount('unread')
  pending.resolve({ data: [thread('unread', 2)] })
  await poll

  assert.equal(store.currentThread?.unread_count, 0)
  assert.deepEqual(store.visibleThreads, [])
  assert.equal(store.loadingThreads, false)
})

test('refreshing an overlapping read preserves messages received after the read', async () => {
  const { store, deferNextLoad, respondWith } = setupStore()
  await store.loadThreads()
  store.setThreadId('unread')
  store.unreadOnly = true
  const pending = Promise.withResolvers<{ data: EntitiesMessageThread[] }>()
  deferNextLoad(pending.promise)
  const poll = store.loadThreads()

  await store.resetThreadUnreadCount('unread')
  respondWith([
    {
      ...thread('unread', 1),
      last_message_id: 'new-inbound-message',
      last_message_content: 'Received after mark-read',
    },
  ])
  pending.resolve({ data: [thread('unread', 2)] })
  await poll

  assert.equal(store.currentThread?.unread_count, 1)
  assert.equal(store.currentThread.last_message_id, 'new-inbound-message')
  assert.equal(store.visibleThreads.length, 1)
  assert.equal(store.loadingThreads, false)
})

test('Back navigation restores an earlier read conversation in Unread', async () => {
  const { store, calls } = await setupReadConversationHistory()

  store.setThreadId('unread')

  assert.equal(store.currentThread?.id, 'unread')
  assert.equal(store.currentThread.unread_count, 0)
  await store.loadThreadMessages('unread')
  assert.equal(calls.at(-1)?.params?.contact, '+18005550100')
  assert.deepEqual(
    store.visibleThreads.map((item) => item.id),
    ['other'],
  )
})

test('saved contacts refresh previously opened read conversations too', async () => {
  const { store } = await setupReadConversationHistory()
  await store.loadThreads([contact('Alice')])

  store.setThreadId('unread')

  assert.equal(store.currentThread?.contact_details?.name, 'Alice')
})

for (const action of ['reset', 'delete', 'archive', 'owner', 'view']) {
  test(`navigation cache does not restore conversations after ${action}`, async () => {
    const { store, phones, respondWith } = await setupReadConversationHistory()
    if (action === 'reset') store.resetState()
    if (action === 'delete') await store.deleteThread('unread')
    if (action === 'archive') {
      await store.updateThread({ threadId: 'unread', isArchived: true })
    }
    if (action === 'owner' || action === 'view') {
      if (action === 'owner') phones.owner = '+18005550198'
      if (action === 'view') store.archivedThreads = true
      respondWith([])
      await store.loadThreads()
    }

    store.setThreadId('unread')

    assert.equal(store.currentThread, null)
    assert.equal(store.hasThreadId('unread'), false)
  })
}

function manyThreads(count: number) {
  return Array.from({ length: count }, (_, index) => ({
    ...thread(`page-${index}`, 1),
    contact: `+1800555${String(index).padStart(4, '0')}`,
  }))
}

for (const count of [0, 19, 20, 21]) {
  test(`Load More is available only when more than 20 threads exist (${count})`, async () => {
    const { store, respondWith } = setupStore()
    respondWith(manyThreads(count))

    await store.loadThreads()

    assert.equal(store.threads.length, Math.min(count, 20))
    assert.equal(store.hasMoreThreads, count > 20)
  })
}

test('Load More expands the loaded list and disappears on the final page', async () => {
  const { store, respondWith, calls } = setupStore()
  respondWith(manyThreads(45))
  await store.loadThreads()

  await store.loadMoreThreads()
  assert.equal(store.threads.length, 40)
  assert.equal(store.hasMoreThreads, true)
  await store.loadMoreThreads()
  assert.equal(store.threads.length, 45)
  assert.equal(new Set(store.threads.map((item) => item.id)).size, 45)
  assert.equal(store.hasMoreThreads, false)
  assert.equal(store.loadingMoreThreads, false)
  const completedCalls = calls.length
  await store.loadMoreThreads()
  assert.equal(calls.length, completedCalls)
})

test('polling preserves loaded pages and changing filters resets them', async () => {
  const { store, respondWith } = setupStore()
  respondWith(manyThreads(45))
  await store.loadThreads()
  await store.loadMoreThreads()

  await store.loadThreads()
  assert.equal(store.threads.length, 40)

  store.unreadOnly = true
  await store.loadThreads()
  assert.equal(store.threads.length, 20)
  await store.loadMoreThreads()
  store.archivedThreads = true
  await store.loadThreads()
  assert.equal(store.threads.length, 20)
})

test('switching phones resets loaded pages', async () => {
  const { store, respondWith, phones } = setupStore()
  respondWith(manyThreads(45))
  await store.loadThreads()
  await store.loadMoreThreads()

  phones.owner = '+18005550198'
  await store.loadThreads()

  assert.equal(store.threads.length, 20)
})

test('Load More failure preserves rows and permits a retry', async () => {
  const { store, respondWith, deferNextLoad } = setupStore()
  respondWith(manyThreads(45))
  await store.loadThreads()
  const pending = Promise.withResolvers<{ data: EntitiesMessageThread[] }>()
  deferNextLoad(pending.promise)

  const loading = store.loadMoreThreads()
  pending.reject(new Error('Network failed'))
  await assert.rejects(loading, /Network failed/)

  assert.equal(store.threads.length, 20)
  assert.equal(store.hasMoreThreads, true)
  assert.equal(store.loadingMoreThreads, false)
  await store.loadMoreThreads()
  assert.equal(store.threads.length, 40)
})

test('new activity does not close an open conversation pushed beyond the loaded page', async () => {
  const { store, respondWith } = setupStore()
  const existing = manyThreads(45)
  respondWith(existing)
  await store.loadThreads()
  store.setThreadId('page-19')
  respondWith([thread('newest', 1), ...existing])

  await store.loadThreads()

  assert.equal(store.currentThread?.id, 'page-19')
  await store.loadThreadMessages('page-19')
})

test('a second Load More click is ignored while the first is running', async () => {
  const { store, respondWith, deferNextLoad } = setupStore()
  const existing = manyThreads(65)
  respondWith(existing)
  await store.loadThreads()
  const pending = Promise.withResolvers<{ data: EntitiesMessageThread[] }>()
  deferNextLoad(pending.promise)

  const loading = store.loadMoreThreads()
  assert.equal(store.loadingMoreThreads, true)
  await store.loadMoreThreads()
  pending.resolve({ data: existing.slice(0, 20) })
  await loading

  assert.equal(store.threads.length, 40)
  assert.equal(store.loadingMoreThreads, false)
})

test('changing filters cancels an older Load More result', async () => {
  const { store, respondWith, deferNextLoad } = setupStore()
  const existing = manyThreads(45)
  respondWith(existing)
  await store.loadThreads()
  const pending = Promise.withResolvers<{ data: EntitiesMessageThread[] }>()
  deferNextLoad(pending.promise)

  const loading = store.loadMoreThreads()
  store.unreadOnly = true
  await store.loadThreads()
  pending.resolve({ data: existing.slice(0, 20) })
  await loading

  assert.equal(store.threads.length, 20)
  assert.equal(store.loadingMoreThreads, false)
  await store.loadMoreThreads()
  assert.equal(store.threads.length, 40)
})
