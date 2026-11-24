import assert from 'node:assert/strict'
import { test } from 'node:test'
import { createPinia, setActivePinia } from 'pinia'
import { ref } from 'vue'
import type { User as FirebaseUser } from 'firebase/auth'
import { useApi, setAuthHeader, setApiKey } from '../composables/useApi'
import { useAuthStore } from './auth'

interface UserResponse {
  data: { id: string; api_key: string; subscription_status?: string }
}

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((done) => {
    resolve = done
  })
  return { promise, resolve }
}

function firebaseUser(id: string, token = Promise.resolve(`token-${id}`)) {
  return {
    uid: id,
    email: null,
    displayName: null,
    getIdToken: () => token,
  } as FirebaseUser
}

function setupStore() {
  const calls: Headers[] = []
  let response = Promise.resolve<UserResponse>({
    data: { id: 'account-a', api_key: 'key-a' },
  })
  Object.assign(globalThis, {
    ref,
    useApi,
    useRuntimeConfig: () => ({
      public: { apiBaseUrl: 'https://api.example.test' },
    }),
    $fetch: {
      create:
        (config: {
          headers: HeadersInit
          onRequest: (context: { options: { headers: Headers } }) => void
        }) =>
        async () => {
          const options = { headers: new Headers(config.headers) }
          config.onRequest({ options })
          calls.push(options.headers)
          return response
        },
    },
    usePhonesStore: () => ({
      activePhone: { id: 'phone-a' },
      loadPhones: async () => {},
    }),
  })
  setAuthHeader(null)
  setApiKey(null)
  setActivePinia(createPinia())
  const store = useAuthStore()
  return {
    store,
    calls,
    respondWith: (next: Promise<UserResponse>) => {
      response = next
    },
  }
}

test('loading a user sends its API key alongside bearer auth on shared requests', async () => {
  const { store, calls, respondWith } = setupStore()
  const client = useApi().apiFetch
  await store.onAuthStateChanged(firebaseUser('account-a'))
  await store.loadUser()
  await client('/v1/messages')

  assert.equal(calls.at(-1)?.get('Authorization'), 'Bearer token-account-a')
  assert.equal(calls.at(-1)?.get('x-api-key'), 'key-a')

  respondWith(
    Promise.resolve({ data: { id: 'account-a', api_key: 'new-key-a' } }),
  )
  await store.loadUser()
  await client('/v1/messages')
  assert.equal(calls.at(-1)?.get('x-api-key'), 'new-key-a')
})

test('an older account refresh cannot overwrite newer subscription state or API keys', async () => {
  const { store, calls, respondWith } = setupStore()
  const stale = deferred<UserResponse>()
  respondWith(stale.promise)
  const oldLoad = store.loadUser()
  respondWith(
    Promise.resolve({
      data: {
        id: 'account-a',
        api_key: 'new-key',
        subscription_status: 'cancelled',
      },
    }),
  )
  await store.loadUser()
  stale.resolve({
    data: {
      id: 'account-a',
      api_key: 'old-key',
      subscription_status: 'active',
    },
  })
  await oldLoad
  await useApi().apiFetch('/v1/messages')

  assert.equal(store.user?.subscription_status, 'cancelled')
  assert.equal(calls.at(-1)?.get('x-api-key'), 'new-key')
})

test('an aborted account refresh cannot publish its response even if the request completes', async () => {
  const { store, calls, respondWith } = setupStore()
  const controller = new AbortController()
  const stale = deferred<UserResponse>()
  respondWith(stale.promise)
  const oldLoad = store.loadUser(controller.signal)
  controller.abort()
  stale.resolve({ data: { id: 'account-a', api_key: 'old-key' } })
  await oldLoad
  await useApi().apiFetch('/v1/messages')

  assert.equal(store.user, null)
  assert.equal(calls.at(-1)?.has('x-api-key'), false)
})

for (const replacement of ['aborted', 'failed'] as const) {
  test(`a successful account load still publishes when its newer replacement is ${replacement}`, async () => {
    const { store, calls, respondWith } = setupStore()
    const original = deferred<UserResponse>()
    respondWith(original.promise)
    const originalLoad = store.loadUser()

    if (replacement === 'aborted') {
      const controller = new AbortController()
      const newer = deferred<UserResponse>()
      respondWith(newer.promise)
      const newerLoad = store.loadUser(controller.signal)
      controller.abort()
      newer.resolve({ data: { id: 'account-a', api_key: 'aborted-key' } })
      await newerLoad
    } else {
      const error = new Error('Account fetch failed')
      respondWith(Promise.reject(error))
      await assert.rejects(
        () => store.loadUser(),
        (caught) => caught === error,
      )
    }

    original.resolve({ data: { id: 'account-a', api_key: 'original-key' } })
    await originalLoad
    respondWith(
      Promise.resolve({ data: { id: 'account-a', api_key: 'original-key' } }),
    )
    await useApi().apiFetch('/v1/messages')

    assert.equal(store.user?.id, 'account-a')
    assert.equal(calls.at(-1)?.get('x-api-key'), 'original-key')
  })
}

for (const mutation of ['update', 'rotate'] as const) {
  test(`an earlier account refresh cannot restore an old API key after ${mutation}`, async () => {
    const { store, calls, respondWith } = setupStore()
    const stale = deferred<UserResponse>()
    respondWith(stale.promise)
    const refresh = store.loadUser()
    respondWith(
      Promise.resolve({ data: { id: 'account-a', api_key: 'new-key' } }),
    )

    if (mutation === 'update') {
      await store.updateUser({ timezone: 'UTC' })
    } else {
      await store.rotateApiKey('account-a')
    }
    stale.resolve({ data: { id: 'account-a', api_key: 'old-key' } })
    await refresh
    await useApi().apiFetch('/v1/messages')

    assert.equal(store.user?.api_key, 'new-key')
    assert.equal(calls.at(-1)?.get('x-api-key'), 'new-key')
  })
}

for (const logout of ['reset', 'auth callback', 'token callback'] as const) {
  test(`a pending user load cannot restore credentials after ${logout}`, async () => {
    const { store, calls, respondWith } = setupStore()
    await store.onAuthStateChanged(firebaseUser('account-a'))
    await store.loadUser()
    const pending = deferred<UserResponse>()
    respondWith(pending.promise)
    const load = store.loadUser()

    if (logout === 'reset') store.resetState()
    if (logout === 'auth callback') await store.onAuthStateChanged(null)
    if (logout === 'token callback') await store.onIdTokenChanged(null)
    pending.resolve({ data: { id: 'account-a', api_key: 'old-key-a' } })
    await load
    await useApi().apiFetch('/v1/messages')

    assert.equal(store.user, null)
    assert.equal(store.authUser, null)
    assert.equal(calls.at(-1)?.has('Authorization'), false)
    assert.equal(calls.at(-1)?.has('x-api-key'), false)
  })
}

for (const switchAccount of ['firebase', 'action'] as const) {
  test(`switching accounts through ${switchAccount} clears old keys and discards pending loads`, async () => {
    const { store, calls, respondWith } = setupStore()
    await store.onAuthStateChanged(firebaseUser('account-a'))
    await store.loadUser()
    const pending = deferred<UserResponse>()
    respondWith(pending.promise)
    const oldLoad = store.loadUser()
    respondWith(
      Promise.resolve({ data: { id: 'account-b', api_key: 'key-b' } }),
    )

    if (switchAccount === 'firebase') {
      await store.onAuthStateChanged(firebaseUser('account-b'))
      await store.loadUser()
    } else {
      await store.setAuthUserAction({
        id: 'account-b',
        email: null,
        displayName: null,
      })
    }
    assert.equal(calls.at(-1)?.has('x-api-key'), false)
    pending.resolve({ data: { id: 'account-a', api_key: 'old-key-a' } })
    await oldLoad
    await useApi().apiFetch('/v1/messages')

    assert.equal(store.user?.id, 'account-b')
    assert.equal(calls.at(-1)?.get('x-api-key'), 'key-b')
  })
}

test('logging back into the same account does not revive an earlier session response', async () => {
  const { store, calls, respondWith } = setupStore()
  await store.onAuthStateChanged(firebaseUser('account-a'))
  const pending = deferred<UserResponse>()
  respondWith(pending.promise)
  const oldLoad = store.loadUser()
  store.resetState()
  await store.onAuthStateChanged(firebaseUser('account-a'))
  respondWith(
    Promise.resolve({ data: { id: 'account-a', api_key: 'new-key-a' } }),
  )
  await store.loadUser()
  pending.resolve({ data: { id: 'account-a', api_key: 'old-key-a' } })
  await oldLoad
  await useApi().apiFetch('/v1/messages')

  assert.equal(calls.at(-1)?.get('x-api-key'), 'new-key-a')
})

for (const operation of ['update', 'rotate'] as const) {
  test(`a pending ${operation} cannot restore credentials after logout`, async () => {
    const { store, calls, respondWith } = setupStore()
    await store.onAuthStateChanged(firebaseUser('account-a'))
    const pending = deferred<UserResponse>()
    respondWith(pending.promise)
    const request =
      operation === 'update'
        ? store.updateUser({ timezone: 'UTC' })
        : store.rotateApiKey('account-a')
    store.resetState()
    pending.resolve({ data: { id: 'account-a', api_key: 'old-key-a' } })
    await request
    await useApi().apiFetch('/v1/messages')

    assert.equal(store.user, null)
    assert.equal(calls.at(-1)?.has('x-api-key'), false)
  })
}

test('a pending Firebase token cannot restore credentials after logout', async () => {
  const { store, calls } = setupStore()
  const token = deferred<string>()
  const login = store.onAuthStateChanged(
    firebaseUser('account-a', token.promise),
  )
  store.resetState()
  token.resolve('old-token-a')
  await login
  await useApi().apiFetch('/v1/messages')

  assert.equal(store.authUser, null)
  assert.equal(calls.at(-1)?.has('Authorization'), false)
})
