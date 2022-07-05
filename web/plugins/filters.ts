import Vue from 'vue'
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
