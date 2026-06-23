import { intervalToDuration, formatDuration } from 'date-fns'
import { parsePhoneNumber, isValidPhoneNumber } from 'libphonenumber-js'

export function formatPhoneNumber(value: string): string {
  if (!value || typeof value !== 'string') {
    return value ?? ''
  }
  if (!isValidPhoneNumber(value)) {
    return value
  }
  const phoneNumber = parsePhoneNumber(value)
  if (phoneNumber) {
    return phoneNumber.formatInternational()
  }
  return value
}

export function phoneCountry(value: string): string {
  const phoneNumber = parsePhoneNumber(value)
  if (phoneNumber && phoneNumber.country) {
    const regionNames = new Intl.DisplayNames(undefined, { type: 'region' })
    return regionNames.of(phoneNumber.country) ?? 'Earth'
  }
  return 'Earth'
}

export function formatTimestamp(value: string): string {
  return new Date(value).toLocaleString()
}

export function formatMoney(value: string | number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(typeof value === 'string' ? parseInt(value) : value)
}

export function formatDecimal(value: string | number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'decimal',
  }).format(typeof value === 'string' ? parseInt(value) : value)
}

export function formatBillingPeriod(value: string): string {
  return new Date(value).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
  })
}

export function formatBillingPeriodDateOrdinal(value: string): string {
  const date = new Date(value)
  const day = date.getDate()
  const month = date.toLocaleDateString('en-US', { month: 'long' })
  const year = date.getFullYear()

  const suffix =
    day % 10 === 1 && day !== 11
      ? 'st'
      : day % 10 === 2 && day !== 12
        ? 'nd'
        : day % 10 === 3 && day !== 13
          ? 'rd'
          : 'th'

  return `${month} ${day}<sup>${suffix}</sup> ${year}`
}

export function humanizeTime(value: string): string {
  const durations = intervalToDuration({
    start: new Date(),
    end: new Date(value),
  })
  return formatDuration(durations)
}
