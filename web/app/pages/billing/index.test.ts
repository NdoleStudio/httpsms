import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { runInNewContext } from 'node:vm'
import ts from 'typescript'
import { parse } from 'vue/compiler-sfc'

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

function setupCancellation(cancelSubscription: () => Promise<void>) {
  let reloads = 0
  const navigations: string[] = []
  const notifications: { message: string; type: string }[] = []
  const loading = { value: false }
  const cancel = runInNewContext(`${outputText}\ncancelPlan`, {
    loading,
    billingStore: { cancelSubscription },
    notificationsStore: {
      addNotification: (notification: { message: string; type: string }) => {
        notifications.push(notification)
      },
    },
    window: { location: { reload: () => reloads++ } },
    navigateTo: (path: string) => navigations.push(path),
    getApiErrorMessage: (_error: unknown, fallback: string) => fallback,
  }) as () => Promise<void>

  return { cancel, loading, notifications, navigations, reloads: () => reloads }
}

test('successful cancellation refreshes the billing page without navigating away', async () => {
  const pending = setupCancellation(async () => {})

  await pending.cancel()

  assert.equal(pending.reloads(), 1)
  assert.deepEqual(pending.navigations, [])
  assert.equal(pending.notifications.at(-1)?.type, 'success')
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
})
