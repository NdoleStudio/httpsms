import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { setImmediate } from 'node:timers/promises'
import { runInNewContext } from 'node:vm'
import ts from 'typescript'
import * as vue from 'vue'
import type { Component } from 'vue'
import { compileScript, parse } from 'vue/compiler-sfc'
import * as icons from '@mdi/js'
import * as countries from '../../utils/countries'
import * as apiError from '../../utils/api-error'
import * as blackFriday from '../../utils/black-friday'

const { descriptor } = parse(
  readFileSync(new URL('./index.vue', import.meta.url), 'utf8'),
)
const compiled = compileScript(descriptor, {
  id: 'billing-component-test',
  inlineTemplate: true,
})
const { outputText } = ts.transpileModule(compiled.content, {
  compilerOptions: {
    target: ts.ScriptTarget.ESNext,
    module: ts.ModuleKind.CommonJS,
  },
})

interface RenderNode {
  type: string
  text: string
  props: Record<string, unknown>
  children: RenderNode[]
  parent: RenderNode | null
}

function node(type: string, text = ''): RenderNode {
  return { type, text, props: {}, children: [], parent: null }
}

function remove(child: RenderNode) {
  if (!child.parent) return
  child.parent.children.splice(child.parent.children.indexOf(child), 1)
  child.parent = null
}

const renderer = vue.createRenderer<RenderNode, RenderNode>({
  createElement: (type) => node(type),
  createText: (text) => node('#text', text),
  createComment: (text) => node('#comment', text),
  setText: (element, text) => {
    element.text = text
  },
  setElementText: (element, text) => {
    element.text = text
    element.children = []
  },
  patchProp: (element, key, _previous, value) => {
    element.props[key] = value
  },
  insert: (child, parent, anchor) => {
    remove(child)
    const index = anchor ? parent.children.indexOf(anchor) : -1
    parent.children.splice(index < 0 ? parent.children.length : index, 0, child)
    child.parent = parent
  },
  remove,
  parentNode: (element) => element.parent,
  nextSibling: (element) => {
    const siblings = element.parent?.children ?? []
    return siblings[siblings.indexOf(element) + 1] ?? null
  },
})

function text(element: RenderNode): string {
  return element.text + element.children.map(text).join('')
}

function find(
  element: RenderNode,
  predicate: (element: RenderNode) => boolean,
): RenderNode[] {
  return [
    ...(predicate(element) ? [element] : []),
    ...element.children.flatMap((child) => find(child, predicate)),
  ]
}

function mountBilling(holdInitialLoad = false) {
  const notifications: { message: string; type: string }[] = []
  let responseStatus = 'active'
  const authStore = vue.reactive({
    user: {
      id: 'account-a',
      subscription_name: 'pro-monthly',
      subscription_status: 'active',
      api_key: 'original-key',
      subscription_renews_at: '2027-01-01T00:00:00Z',
      subscription_ends_at: '2027-01-01T00:00:00Z',
    },
    authUser: null,
    loadUser: async () => {},
  })
  const billingStore = {
    billingUsage: null,
    billingUsageHistory: [],
    loadBillingUsage: async () => {
      if (holdInitialLoad) await new Promise<void>(() => {})
    },
    loadBillingUsageHistory: async () => {},
    cancelSubscription: async () => {},
  }
  const imports: Record<string, unknown> = {
    vue,
    '@mdi/js': icons,
    '~/utils/countries': countries,
    '~/utils/api-error': apiError,
    '~/utils/black-friday': blackFriday,
  }
  const component = runInNewContext(`${outputText}\nexports.default`, {
    exports: {},
    require: (name: string) => {
      assert.ok(name in imports, `Unexpected import: ${name}`)
      return imports[name]
    },
    ref: vue.ref,
    computed: vue.computed,
    watch: vue.watch,
    onMounted: vue.onMounted,
    onBeforeUnmount: vue.onBeforeUnmount,
    definePageMeta: () => {},
    useHead: () => {},
    useRuntimeConfig: () => ({
      public: {
        checkoutUrl: 'https://checkout.example.test',
        enterpriseCheckoutUrl: 'https://checkout.example.test/enterprise',
      },
    }),
    useVDisplay: () => ({ lgAndUp: false }),
    useAuthStore: () => authStore,
    useBillingStore: () => billingStore,
    useApi: () => ({
      apiFetch: async () => ({
        data: {
          ...authStore.user,
          subscription_status: responseStatus,
          api_key: 'response-key',
        },
      }),
    }),
    useBlackFridayStore: () => ({ active: false, now: 0 }),
    useNotificationsStore: () => ({
      addNotification: (notification: { message: string; type: string }) =>
        notifications.push(notification),
    }),
    useFilters: () => ({
      formatDecimal: String,
      formatTimestamp: String,
    }),
    Date,
    setTimeout,
    clearTimeout,
    AbortController,
    URL,
  }) as Component
  const app = renderer.createApp(component)
  const names = new Set(
    [...descriptor.template!.content.matchAll(/<([A-Z]\w*)/g)].map(
      (match) => match[1]!,
    ),
  )
  names.delete('VDialog')
  names.add('v-code')
  for (const name of names) {
    app.component(
      name,
      vue.defineComponent({
        inheritAttrs: false,
        setup:
          (_props, { attrs, slots }) =>
          () =>
            vue.h(name, attrs, slots.default?.()),
      }),
    )
  }
  app.component(
    'VDialog',
    vue.defineComponent({
      props: { modelValue: Boolean, persistent: Boolean },
      emits: ['update:modelValue'],
      setup:
        (props, { slots, emit }) =>
        () =>
          vue.h('VDialog', { ...props }, [
            slots.activator?.({
              props: { onClick: () => emit('update:modelValue', true) },
            }),
            props.modelValue ? slots.default?.() : null,
          ]),
    }),
  )
  const root = node('root')
  app.mount(root)
  return {
    app,
    root,
    authStore,
    notifications,
    confirmCancellation: () => {
      responseStatus = 'cancelled'
    },
  }
}

function buttons(root: RenderNode, label: string) {
  return find(
    root,
    (element) => element.type === 'VBtn' && text(element).trim() === label,
  )
}

async function click(button: RenderNode) {
  const handler = button.props.onClick
  assert.equal(typeof handler, 'function')
  if (typeof handler === 'function') return handler()
}

test('the mounted billing page keeps the dialog persistent, then renders pending and confirmed plan cards', async (context) => {
  context.mock.timers.enable({ apis: ['Date', 'setTimeout'], now: 0 })
  const { app, root, authStore, notifications } = mountBilling()
  try {
    await setImmediate()
    await click(buttons(root, 'Cancel Plan')[0]!)
    await vue.nextTick()
    const cancellation = click(buttons(root, 'Cancel Plan').at(-1)!)
    await setImmediate()

    const dialog = find(
      root,
      (element) =>
        element.type === 'VDialog' && element.props.modelValue === true,
    )[0]!
    assert.equal(dialog.props.persistent, true)
    assert.equal(buttons(root, 'Keep Subscription')[0]?.props.disabled, true)
    assert.equal(notifications.length, 0)

    context.mock.timers.tick(10000)
    await cancellation
    await vue.nextTick()

    assert.match(text(root), /Cancellation in progress.*still updating/)
    assert.equal(buttons(root, 'Cancel Plan').length, 0)
    assert.equal(buttons(root, 'Update Plan').length, 0)
    assert.equal(notifications.length, 1)

    authStore.user.subscription_status = 'cancelled'
    await vue.nextTick()
    assert.doesNotMatch(text(root), /still updating/)
    assert.match(text(root), /You will be downgraded/)
    assert.ok(buttons(root, 'Upgrade Plan').length > 0)

    authStore.user.subscription_name = 'free'
    authStore.user.subscription_status = 'expired'
    await vue.nextTick()
    assert.doesNotMatch(text(root), /still updating|Your next bill/)
    assert.equal(buttons(root, 'Cancel Plan').length, 0)
    assert.ok(buttons(root, 'Upgrade Plan').length > 0)
  } finally {
    app.unmount()
    context.mock.timers.reset()
  }
})

test('unrelated billing loading does not lock the cancellation dialog', async () => {
  const { app, root } = mountBilling(true)
  try {
    await setImmediate()
    await click(buttons(root, 'Cancel Plan')[0]!)
    await vue.nextTick()

    const dialog = find(
      root,
      (element) =>
        element.type === 'VDialog' && element.props.modelValue === true,
    )[0]!
    assert.equal(dialog.props.persistent, false)
    assert.equal(buttons(root, 'Keep Subscription')[0]?.props.disabled, false)
  } finally {
    app.unmount()
  }
})

test('confirmed cancellation updates the mounted billing card without modifying auth state', async () => {
  const { app, root, authStore, notifications, confirmCancellation } =
    mountBilling()
  try {
    await setImmediate()
    await click(buttons(root, 'Cancel Plan')[0]!)
    await vue.nextTick()
    confirmCancellation()
    await click(buttons(root, 'Cancel Plan').at(-1)!)
    await vue.nextTick()

    assert.match(text(root), /You will be downgraded/)
    assert.equal(buttons(root, 'Cancel Plan').length, 0)
    assert.equal(authStore.user.subscription_status, 'active')
    assert.equal(authStore.user.api_key, 'original-key')
    assert.equal(notifications.length, 1)
    assert.equal(notifications[0]?.type, 'success')

    authStore.user.id = 'account-b'
    await vue.nextTick()
    assert.doesNotMatch(text(root), /You will be downgraded/)
    assert.ok(buttons(root, 'Cancel Plan').length > 0)
  } finally {
    app.unmount()
  }
})
