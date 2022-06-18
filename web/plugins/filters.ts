import Vue from 'vue'
import { parsePhoneNumber } from 'libphonenumber-js'

Vue.filter('phoneNumber', (value: string): string => {
  const phoneNumber = parsePhoneNumber(value)
  if (phoneNumber) {
    return phoneNumber.formatInternational()
  }
  return value
})
