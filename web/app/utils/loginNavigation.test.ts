import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
  createMemoryHistory,
  createRouter,
  type NavigationGuard,
} from 'vue-router'

test('returning home from login bypasses the dashboard preference for that visit only', async () => {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: ['/login', '/threads', '/blog'].map((path) => ({
      path,
      name: path.slice(1),
      component: {},
    })),
  })
  Object.assign(globalThis, {
    defineNuxtRouteMiddleware: (guard: NavigationGuard) =>
      router.addRoute({ path: '/', component: {}, beforeEnter: guard }),
    localStorage: {
      getItem: (key: string) =>
        key === 'httpsms_redirect_to_threads' ? 'true' : null,
    },
    navigateTo: (path: string) => path,
  })
  await import('../middleware/redirectToThreads')

  await router.push('/login?to=/threads/')
  await router.push('/')
  assert.equal(router.currentRoute.value.path, '/')

  await router.push('/blog')
  await router.push('/')
  assert.equal(router.currentRoute.value.path, '/threads')
})
