import Vue from 'vue'
import { intervalToDuration, formatDuration } from 'date-fns'
import { parsePhoneNumber, isValidPhoneNumber } from 'libphonenumber-js'

export const formatPhoneNumber = (value: string) => {
  if (!isValidPhoneNumber(value)) {
    return value
  }
  const phoneNumber = parsePhoneNumber(value)
  if (phoneNumber) {
    return phoneNumber.formatInternational()
  }
  return value
}

Vue.filter('phoneNumber', (value: string): string => {
  return formatPhoneNumber(value)
})

Vue.filter('phoneCountry', (value: string): string => {
  const phoneNumber = parsePhoneNumber(value)
  if (phoneNumber && phoneNumber.country) {
    // @ts-ignore
    const regionNames = new Intl.DisplayNames(undefined, { type: 'region' })
    return regionNames.of(phoneNumber.country) ?? 'earth'
  }
  return 'Earth'
})

Vue.filter('timestamp', (value: string): string => {
  return new Date(value).toLocaleString()
})

Vue.filter('money', (value: string): string => {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(parseInt(value))
})

Vue.filter('decimal', (value: string): string => {
  return new Intl.NumberFormat('en-US', {
    style: 'decimal',
  }).format(parseInt(value))
})

Vue.filter('billingPeriod', (value: string): string => {
  const date = new Date(value)
  const options: Intl.DateTimeFormatOptions = {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  }
  return date.toLocaleDateString('en-US', options)
})

Vue.filter('billingPeriodDate', (value: string): string => {
  const date = new Date(value)
  const options: Intl.DateTimeFormatOptions = {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  }
  return date.toLocaleDateString('en-US', options)
})

Vue.filter('billingPeriodDateOrdinal', (value: string): string => {
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
})

Vue.filter('humanizeTime', (value: string): string => {
  const durations = intervalToDuration({
    start: new Date(),
    end: new Date(value),
  })
  return formatDuration(durations)
})

Vue.filter('capitalize', (value: string): string => {
  return value.charAt(0).toUpperCase() + value.slice(1)
})
