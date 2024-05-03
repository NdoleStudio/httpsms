import Vue from 'vue'
import * as fns from 'date-fns'
import { parsePhoneNumber, isValidPhoneNumber } from 'libphonenumber-js'

Vue.filter('phoneNumber', (value: string): string => {
  if (!isValidPhoneNumber(value)) {
    return value
  }
  const phoneNumber = parsePhoneNumber(value)
  if (phoneNumber) {
    return phoneNumber.formatInternational()
  }
  return value
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
  const options = {
    year: 'numeric',
    month: 'long',
  }
  // @ts-ignore
  return new Date(value).toLocaleDateString('en-US', options)
})

Vue.filter('humanizeTime', (value: string): string => {
  const durations = fns.intervalToDuration({
    start: new Date(),
    end: new Date(value),
  })
  return fns.formatDuration(durations)
})
