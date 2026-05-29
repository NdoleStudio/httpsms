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
  const startDate = new Date(value)
  const options: Intl.DateTimeFormatOptions = {
    month: 'short',
    day: 'numeric',
  }
  const optionsWithYear: Intl.DateTimeFormatOptions = {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  }
  const start = startDate.toLocaleDateString('en-US', options)
  const endDate = new Date(startDate)
  endDate.setMonth(endDate.getMonth() + 1)
  endDate.setDate(endDate.getDate() - 1)
  const end = endDate.toLocaleDateString('en-US', optionsWithYear)
  return `${start} – ${end}`
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
