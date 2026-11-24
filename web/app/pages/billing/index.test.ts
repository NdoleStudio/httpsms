import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { setImmediate } from 'node:timers/promises'
import { runInNewContext } from 'node:vm'
import ts from 'typescript'
import { parse } from 'vue/compiler-sfc'
import { getApiErrorMessage } from '../../utils/api-error'

const { descriptor } = parse(
  readFileSync(new URL('./index.vue', import.meta.url), 'utf8'),
)
const script = ts.createSourceFile(
  'billing.ts',
  descriptor.scriptSetup!.content,
  ts.ScriptTarget.Latest,
  true,
  ts.ScriptKind.TS,
)
const cancelPlan = script.statements.find(
  (statement) =>
    ts.isFunctionDeclaration(statement) &&
    statement.name?.text === 'cancelPlan',
)!
const { outputText } = ts.transpileModule(cancelPlan.getText(script), {
  compilerOptions: { target: ts.ScriptTarget.ESNext },
})
const unmountHook = script.statements.find(
  (statement) =>
    ts.isExpressionStatement(statement) &&
    ts.isCallExpression(statement.expression) &&
    statement.expression.expression.getText(script) === 'onBeforeUnmount',
)
const unmountCode = unmountHook?.getText(script) ?? ''

function setupCancellation(
  cancelSubscription: () => Promise<void>,
  refreshError?: unknown,
  statuses = ['cancelled'],
) {
  let reloads = 0
  let refreshes = 0
  let elapsed = 0
  let deadlineCallback: (() => void) | undefined
  let unmount = () => {}
  let refreshSignal: AbortSignal | undefined
  const waitingStates: { dialog: boolean; notifications: number }[] = []
  const navigations: string[] = []
  const notifications: { message: string; type: string }[] = []
  const loading = { value: false }
  const dialog = { value: true }
  const cancellationPending = { value: false }
  const billingUser: {
    value: { id: string; subscription_status: string; api_key: string } | null
  } = { value: null }
  const authStore = {
    user: {
      id: 'account-a',
      subscription_status: 'active',
      api_key: 'original-key',
    },
  }
  const api = {
    apiFetch: async (_path: string, options: { signal?: AbortSignal }) => {
      refreshSignal = options.signal
      if (refreshError) throw refreshError
      return {
        data: {
          id: 'account-a',
          subscription_status:
            statuses[Math.min(refreshes++, statuses.length - 1)]!,
          api_key: 'response-key',
        },
      }
    },
  }
  const cancel = runInNewContext(`${unmountCode}\n${outputText}\ncancelPlan`, {
    loading,
    dialog,
    cancellationPending,
    cancellingSubscription: { value: false },
    billingPageActive: true,
    stopCancellationWait: undefined,
    cancellationRefresh: undefined,
    AbortController,
    onBeforeUnmount: (callback: () => void) => {
      unmount = callback
    },
    authStore,
    billingUser,
    apiFetch: (path: string, options: { signal?: AbortSignal }) =>
      api.apiFetch(path, options),
    billingStore: { cancelSubscription },
    notificationsStore: {
      addNotification: (notification: { message: string; type: string }) => {
        notifications.push(notification)
      },
    },
    window: { location: { reload: () => reloads++ } },
    navigateTo: (path: string) => navigations.push(path),
    getApiErrorMessage,
    Date: { now: () => elapsed },
    setTimeout: (callback: () => void, delay: number) => {
      if (delay === 10000) {
        deadlineCallback = callback
      } else {
        waitingStates.push({
          dialog: dialog.value,
          notifications: notifications.length,
        })
        elapsed += delay
        callback()
        if (elapsed >= 10000) deadlineCallback?.()
      }
      return 1
    },
    clearTimeout: () => {
      deadlineCallback = undefined
    },
  }) as () => Promise<void>

  return {
    cancel,
    loading,
    dialog,
    cancellationPending,
    authStore,
    billingUser,
    api,
    notifications,
    navigations,
    reloads: () => reloads,
    refreshes: () => refreshes,
    refreshSignal: () => refreshSignal,
    elapsed: () => elapsed,
    expireDeadline: () => {
      elapsed = 10000
      deadlineCallback?.()
    },
    waitingStates,
    unmount: () => unmount(),
  }
}

test('successful cancellation updates billing status in place without a browser reload', async () => {
  const pending = setupCancellation(async () => {})

  await pending.cancel()

  assert.equal(pending.reloads(), 0)
  assert.deepEqual(pending.navigations, [])
  assert.equal(pending.billingUser.value?.subscription_status, 'cancelled')
  assert.equal(pending.authStore.user.subscription_status, 'active')
  assert.equal(pending.authStore.user.api_key, 'original-key')
  assert.equal(pending.dialog.value, false)
  assert.equal(pending.loading.value, false)
  assert.equal(pending.cancellationPending.value, false)
  assert.equal(pending.notifications.at(-1)?.type, 'success')
  assert.equal(pending.notifications.length, 1)
})

test('failed cancellation stays on the page without refreshing and displays an error', async () => {
  const pending = setupCancellation(async () => {
    throw new Error('Cancellation failed')
  })

  await pending.cancel()

  assert.equal(pending.reloads(), 0)
  assert.deepEqual(pending.navigations, [])
  assert.equal(pending.notifications.at(-1)?.type, 'error')
  assert.equal(pending.loading.value, false)
  assert.equal(pending.authStore.user.subscription_status, 'active')
  assert.equal(pending.dialog.value, false)
})

test('a refresh failure after cancellation reports the refresh error rather than a cancellation failure', async () => {
  const pending = setupCancellation(async () => {}, {
    data: { message: 'Account fetch failed' },
  })

  await pending.cancel()

  assert.equal(pending.reloads(), 0)
  assert.deepEqual(pending.navigations, [])
  assert.equal(pending.dialog.value, false)
  assert.equal(pending.loading.value, false)
  assert.equal(pending.notifications.at(-1)?.type, 'error')
  assert.match(pending.notifications.at(-1)!.message, /cancelled.*refresh/)
  assert.match(pending.notifications.at(-1)!.message, /Account fetch failed/)
  assert.equal(pending.notifications.length, 1)
})

test('cancellation waits for the webhook when the first account response is stale', async () => {
  const pending = setupCancellation(async () => {}, undefined, [
    'active',
    'cancelled',
  ])

  await pending.cancel()

  assert.equal(pending.billingUser.value?.subscription_status, 'cancelled')
  assert.equal(pending.cancellationPending.value, false)
  assert.ok(pending.refreshes() > 1)
  assert.equal(pending.loading.value, false)
  assert.equal(pending.reloads(), 0)
  assert.deepEqual(pending.waitingStates, [{ dialog: true, notifications: 0 }])
  assert.equal(pending.dialog.value, false)
  assert.equal(pending.notifications.length, 1)
})

test('a delayed webhook leaves a visible pending state after bounded polling', async () => {
  const pending = setupCancellation(async () => {}, undefined, ['active'])

  await pending.cancel()

  assert.equal(pending.cancellationPending.value, true)
  assert.ok(pending.refreshes() > 1 && pending.refreshes() <= 20)
  assert.equal(pending.loading.value, false)
  assert.equal(pending.notifications.at(-1)?.type, 'info')
  assert.match(pending.notifications.at(-1)!.message, /cancellation.*progress/i)
  assert.equal(pending.elapsed(), 10000)
  assert.ok(
    pending.waitingStates.every(
      (state) => state.dialog && state.notifications === 0,
    ),
  )
  assert.equal(pending.dialog.value, false)
  assert.equal(pending.notifications.length, 1)
})

test('the dialog closes at ten seconds even if the account refresh is still in flight', async () => {
  const pending = setupCancellation(async () => {})
  let signal: AbortSignal | undefined
  pending.api.apiFetch = (_path, options) => {
    signal = options.signal
    return new Promise(() => {})
  }
  const cancellation = pending.cancel()
  await setImmediate()

  assert.equal(pending.dialog.value, true)
  assert.equal(pending.notifications.length, 0)
  pending.expireDeadline()
  await cancellation

  assert.equal(pending.dialog.value, false)
  assert.equal(pending.loading.value, false)
  assert.equal(pending.notifications.length, 1)
  assert.equal(pending.notifications[0]?.type, 'info')
  assert.equal(signal?.aborted, true)
})

test('leaving billing stops polling and prevents a notification on the next page', async () => {
  const pending = setupCancellation(async () => {})
  let resolveRefresh!: () => void
  let refreshes = 0
  pending.api.apiFetch = async () => {
    refreshes++
    await new Promise<void>((resolve) => {
      resolveRefresh = resolve
    })

    return {
      data: {
        id: 'account-a',
        subscription_status: 'cancelled',
        api_key: 'response-key',
      },
    }
  }
  const cancellation = pending.cancel()
  await setImmediate()

  pending.unmount()
  pending.expireDeadline()
  await cancellation
  resolveRefresh()
  await setImmediate()

  assert.equal(pending.notifications.length, 0)
  assert.equal(refreshes, 1)
  assert.equal(pending.billingUser.value, null)
  assert.equal(pending.authStore.user.api_key, 'original-key')
})

test('the total deadline includes a cancellation request that is still in progress', async () => {
  let resolveCancellation!: () => void
  const pending = setupCancellation(
    () =>
      new Promise<void>((resolve) => {
        resolveCancellation = resolve
      }),
  )
  const cancellation = pending.cancel()
  await setImmediate()

  assert.equal(pending.dialog.value, true)
  assert.equal(pending.loading.value, true)
  assert.equal(pending.notifications.length, 0)
  pending.expireDeadline()
  await cancellation

  assert.equal(pending.dialog.value, false)
  assert.equal(pending.loading.value, false)
  assert.equal(pending.notifications.length, 1)
  assert.match(pending.notifications[0]!.message, /cancellation.*progress/i)
  resolveCancellation()
  await setImmediate()
  assert.equal(pending.billingUser.value, null)
  assert.equal(pending.refreshes(), 0)
  assert.equal(pending.notifications.length, 1)
})

test('an old cancellation response cannot update billing or notify after the account changes', async () => {
  const pending = setupCancellation(async () => {})
  let resolveRefresh!: () => void
  pending.api.apiFetch = async () => {
    await new Promise<void>((resolve) => {
      resolveRefresh = resolve
    })
    return {
      data: {
        id: 'account-a',
        subscription_status: 'cancelled',
        api_key: 'response-key',
      },
    }
  }
  const cancellation = pending.cancel()
  await setImmediate()
  pending.authStore.user.id = 'account-b'
  pending.expireDeadline()
  await cancellation
  resolveRefresh()
  await setImmediate()

  assert.equal(pending.billingUser.value, null)
  assert.equal(pending.notifications.length, 0)
  assert.equal(pending.authStore.user.id, 'account-b')
  assert.equal(pending.authStore.user.api_key, 'original-key')
})
