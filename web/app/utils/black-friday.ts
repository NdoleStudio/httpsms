export const BLACK_FRIDAY_END = Date.parse('2026-12-01T00:00:00Z')

export function getBlackFridayCountdown(now: number): string | null {
  const seconds = Math.ceil((BLACK_FRIDAY_END - now) / 1000)
  if (seconds <= 0) return null

  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const time = [hours, minutes, seconds % 60]
    .map((value) => String(value).padStart(2, '0'))
    .join(':')
  return `${days} ${days === 1 ? 'Day' : 'Days'} ${time}`
}

export function isBlackFridayEligible(
  page: string,
  authReady: boolean,
  signedIn: boolean,
  subscriptionName?: string,
): boolean {
  if (!authReady) return false
  if (page === 'index') return !signedIn
  return signedIn && subscriptionName === 'free'
}

export function getBlackFridayCheckoutUrl(
  url: URL,
  subscriptionName?: string,
  now = Date.now(),
): string {
  if (subscriptionName === 'free' && now < BLACK_FRIDAY_END) {
    url.searchParams.set('checkout[discount_code]', 'BF2026')
  }
  return url.toString()
}
