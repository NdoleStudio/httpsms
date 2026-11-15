import assert from 'node:assert/strict'
import { test } from 'node:test'
import { createPinia, disposePinia, setActivePinia } from 'pinia'
import { computed } from 'vue'
import { getBlackFridayCheckoutUrl } from '../utils/black-friday'
import { useBlackFridayStore } from './blackFriday'

test('expires the countdown, cached offer, and checkout links together on an open page', (t) => {
  t.mock.timers.enable({
    apis: ['Date', 'setInterval'],
    now: new Date('2026-11-30T23:59:58Z'),
  })
  const pinia = createPinia()
  setActivePinia(pinia)
  t.after(() => disposePinia(pinia))
  const promotion = useBlackFridayStore()
  const offerVisible = computed(() => promotion.active)
  const links = ['pro', 'enterprise'].map((product) =>
    computed(
      () =>
        new URL(
          getBlackFridayCheckoutUrl(
            new URL(`https://store.lemonsqueezy.com/buy/${product}`),
            'free',
            promotion.now,
          ),
        ),
    ),
  )

  assert.equal(offerVisible.value, true)
  for (const link of links) {
    assert.equal(
      link.value.searchParams.get('checkout[discount_code]'),
      'BF2026',
    )
  }
  t.mock.timers.tick(1000)
  assert.equal(promotion.countdown, '0 Days 00:00:01')
  t.mock.timers.tick(1000)
  assert.equal(promotion.countdown, null)
  assert.equal(offerVisible.value, false)
  for (const link of links) {
    assert.equal(link.value.searchParams.has('checkout[discount_code]'), false)
  }
  const expiredAt = promotion.now
  t.mock.timers.tick(10_000)
  assert.equal(promotion.now, expiredAt)
})

test('stops the shared clock when the store is disposed', (t) => {
  t.mock.timers.enable({
    apis: ['Date', 'setInterval'],
    now: new Date('2026-11-30T23:59:58Z'),
  })
  const pinia = createPinia()
  setActivePinia(pinia)
  t.after(() => disposePinia(pinia))
  const promotion = useBlackFridayStore()
  const initialTime = promotion.now

  promotion.$dispose()
  t.mock.timers.tick(2000)

  assert.equal(promotion.now, initialTime)
})
