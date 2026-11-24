import assert from 'node:assert/strict'
import { test } from 'node:test'
import { createPinia, setActivePinia } from 'pinia'
import { ref } from 'vue'
import { useBillingStore } from './billing'

function setupStore(
  apiFetch: (path: string, options: { method: string }) => Promise<void>,
) {
  Object.assign(globalThis, {
    ref,
    useApi: () => ({ apiFetch }),
    useNotificationsStore: () => ({ addNotification: () => {} }),
  })
  setActivePinia(createPinia())
  return useBillingStore()
}

test('subscription cancellation succeeds with an empty 204 response', async () => {
  const calls: { path: string; method: string }[] = []
  const store = setupStore(async (path, options) => {
    calls.push({ path, method: options.method })
  })

  await assert.doesNotReject(() => store.cancelSubscription())
  assert.deepEqual(calls, [
    { path: '/v1/users/subscription', method: 'DELETE' },
  ])
})

test('subscription cancellation propagates API failures', async () => {
  const error = Object.assign(new Error('Cancellation failed'), {
    statusCode: 500,
  })
  const store = setupStore(async () => {
    throw error
  })

  await assert.rejects(
    () => store.cancelSubscription(),
    (caught) => caught === error,
  )
})
