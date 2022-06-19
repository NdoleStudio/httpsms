import Vue from 'vue'
import { parsePhoneNumber } from 'libphonenumber-js'

Vue.filter('phoneNumber', (value: string): string => {
  const phoneNumber = parsePhoneNumber(value)
  if (phoneNumber) {
    return phoneNumber.formatInternational()
  }
  return value
})

Vue.filter('phoneCountry', (value: string): string => {
  const phoneNumber = parsePhoneNumber(value)
  if (phoneNumber && phoneNumber.country) {
    const regionNames = new Intl.DisplayNames(undefined, { type: 'region' })
    return regionNames.of(phoneNumber.country) ?? 'earth'
  }
  return 'Earth'
})

Vue.filter('timestamp', (value: string): string => {
  return new Date(value).toLocaleString()
})
