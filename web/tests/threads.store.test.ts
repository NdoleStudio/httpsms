import { strict as assert } from 'node:assert'
import { test } from 'node:test'
import { createPinia, setActivePinia } from 'pinia'
import { computed, ref } from 'vue'
import type { EntitiesMessageThread } from '../shared/types/api'
import { useThreadsStore } from '../app/stores/threads'

type Thread = EntitiesMessageThread & {
  is_read: boolean
}

type ThreadsStore = ReturnType<typeof useThreadsStore> & {
  markThreadRead(threadId: string, force?: boolean): Promise<void>
}

type TestGlobals = typeof globalThis & {
  ref: typeof ref
  computed: typeof computed
  useApi(): {
    apiFetch: <T>(url: string, options?: Record<string, unknown>) => Promise<T>
  }
  useNotificationsStore(): {
    addNotification(request: {
      message: string
      type: 'error' | 'success' | 'info'
    }): void
  }
  usePhonesStore(): {
    owner: string | null
    phones: Array<{ phone_number: string }>
    getHeartbeat(): Promise<unknown[]>
  }
}

function createThread(overrides: Partial<Thread> = {}) {
  return {
    color: 'indigo',
    contact: '+18005550100',
    created_at: '2026-07-18T00:00:00Z',
    id: 'thread-1',
    is_archived: false,
    is_read: false,
    last_message_content: 'Hello',
    last_message_id: 'message-1',
    order_timestamp: '2026-07-18T00:00:00Z',
    owner: '+18005550199',
    status: 'PENDING',
    updated_at: '2026-07-18T00:00:00Z',
    user_id: 'user-1',
    ...overrides,
  } as Thread
}

function createStoreHarness(options?: {
  threads?: Thread[]
  responseThread?: Thread
}) {
  const calls: Array<[string, Record<string, unknown> | undefined]> = []
  const notifications: Array<{
    message: string
    type: 'error' | 'success' | 'info'
  }> = []
  const responseThread =
    options?.responseThread ?? createThread({ is_read: true })

  const globals = globalThis as TestGlobals

  globals.ref = ref
  globals.computed = computed
  globals.useApi = () => ({
    apiFetch: async <T>(url: string, options?: Record<string, unknown>) => {
      calls.push([url, options])
      return { data: responseThread } as T
    },
  })
  globals.useNotificationsStore = () => ({
    addNotification: (request: {
      message: string
      type: 'error' | 'success' | 'info'
    }) => {
      notifications.push(request)
    },
  })
  globals.usePhonesStore = () => ({
    owner: null,
    phones: [],
    getHeartbeat: async () => [],
  })

  setActivePinia(createPinia())
  const store = useThreadsStore() as ThreadsStore
  store.threads = options?.threads ?? [createThread()]

  return { store, calls, notifications, responseThread }
}

test('markThreadRead persists unread threads and replaces the response thread', async () => {
  const updatedThread = createThread({
    is_read: true,
    last_message_content: 'Updated',
  })
  const { store, calls } = createStoreHarness({ responseThread: updatedThread })

  assert.equal(typeof store.markThreadRead, 'function')

  await store.markThreadRead('thread-1')

  assert.deepEqual(calls, [
    [
      '/v1/message-threads/thread-1',
      {
        method: 'PUT',
        body: { is_read: true },
      },
    ],
  ])
  assert.deepEqual(store.threads[0], updatedThread)
})

test('markThreadRead skips already read threads', async () => {
  const readThread = createThread({ is_read: true })
  const { store, calls } = createStoreHarness({ threads: [readThread] })

  assert.equal(typeof store.markThreadRead, 'function')

  await store.markThreadRead('thread-1')

  assert.deepEqual(calls, [])
  assert.deepEqual(store.threads[0], readThread)
})
