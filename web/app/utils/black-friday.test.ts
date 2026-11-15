import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
  getBlackFridayCheckoutUrl,
  getBlackFridayCountdown,
  isBlackFridayEligible,
} from './black-friday'

test('counts down to the UTC deadline regardless of the input timezone', () => {
  for (const now of [
    '2026-11-29T22:57:56Z',
    '2026-11-30T00:57:56+02:00',
    '2026-11-29T17:57:56-05:00',
  ]) {
    assert.equal(getBlackFridayCountdown(Date.parse(now)), '1 Day 01:02:04')
  }
})

test('spells out plural days in the countdown', () => {
  assert.equal(
    getBlackFridayCountdown(Date.parse('2026-11-15T12:00:00Z')),
    '15 Days 12:00:00',
  )
})

test('keeps the final partial second visible and expires exactly at the deadline', () => {
  assert.equal(
    getBlackFridayCountdown(Date.parse('2026-11-30T23:59:59.500Z')),
    '0 Days 00:00:01',
  )
  assert.equal(
    getBlackFridayCountdown(Date.parse('2026-12-01T00:00:00Z')),
    null,
  )
  assert.equal(
    getBlackFridayCountdown(Date.parse('2026-12-02T00:00:00Z')),
    null,
  )
})

test('shows the landing promotion only after confirming the visitor is signed out', () => {
  assert.equal(isBlackFridayEligible('index', true, false), true)
  assert.equal(isBlackFridayEligible('index', false, false), false)
  assert.equal(isBlackFridayEligible('index', true, true, 'free'), false)
  assert.equal(isBlackFridayEligible('blog', true, false), false)
})

test('shows the promotion across app pages only for confirmed signed-in free users', () => {
  for (const page of [
    'billing',
    'threads',
    'threads-id',
    'settings',
    'bulk-messages',
    'search-messages',
    'phone-api-keys',
    'heartbeats-id',
  ]) {
    assert.equal(isBlackFridayEligible(page, true, true, 'free'), true)
    assert.equal(isBlackFridayEligible(page, true, true), false)
    assert.equal(isBlackFridayEligible(page, true, false, 'free'), false)
    assert.equal(isBlackFridayEligible(page, false, true, 'free'), false)
    for (const plan of [
      'pro-monthly',
      'pro-yearly',
      'pro-lifetime',
      'unknown',
    ]) {
      assert.equal(isBlackFridayEligible(page, true, true, plan), false)
    }
  }
})

test('prefills the active promotion at checkout without losing customer details', () => {
  const url = new URL('https://store.lemonsqueezy.com/buy/pro?embed=1')
  url.searchParams.set('checkout[custom][user_id]', 'test-user')
  url.searchParams.set('checkout[email]', 'buyer@example.test')
  const checkout = new URL(
    getBlackFridayCheckoutUrl(url, 'free', Date.parse('2026-11-30T23:59:59Z')),
  )

  assert.equal(checkout.searchParams.get('checkout[discount_code]'), 'BF2026')
  assert.equal(
    checkout.searchParams.get('checkout[custom][user_id]'),
    'test-user',
  )
  assert.equal(
    checkout.searchParams.get('checkout[email]'),
    'buyer@example.test',
  )
  assert.equal(checkout.searchParams.get('embed'), '1')
})

test('does not prefill the promotion for paid or unconfirmed users', () => {
  for (const plan of [undefined, 'pro-yearly', 'pro-lifetime', 'unknown']) {
    const url = new URL('https://store.lemonsqueezy.com/buy/pro')
    assert.equal(
      getBlackFridayCheckoutUrl(url, plan, Date.parse('2026-11-15T12:00:00Z')),
      'https://store.lemonsqueezy.com/buy/pro',
    )
  }
})

test('stops prefilling the discount at the UTC promotion deadline', () => {
  const url = new URL('https://store.lemonsqueezy.com/buy/pro')
  for (const now of ['2026-12-01T00:00:00Z', '2026-12-02T00:00:00Z']) {
    assert.equal(
      getBlackFridayCheckoutUrl(url, 'free', Date.parse(now)),
      'https://store.lemonsqueezy.com/buy/pro',
    )
  }
})
