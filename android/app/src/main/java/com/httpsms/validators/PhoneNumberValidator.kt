package com.httpsms.validators

import com.google.i18n.phonenumbers.PhoneNumberUtil
import timber.log.Timber

class PhoneNumberValidator {
    companion object {
        private val phoneNumberUtil = PhoneNumberUtil.getInstance()
        fun isValidPhoneNumber(phoneNumber: String, countryCode: String): Boolean {
            Timber.e(countryCode)
            return try {
                if (phoneNumber.isEmpty()) {
                    return false
                }
                val number = phoneNumberUtil.parse(fixNumber(phoneNumber), countryCode)
                phoneNumberUtil.isValidNumber(number)
            } catch (e: Exception) {
                false
            }
        }
        fun formatE164(phoneNumber: String, countryCode: String): String {
            return try {
                val number = phoneNumberUtil.parse(fixNumber(phoneNumber), countryCode)
                phoneNumberUtil.format(number, PhoneNumberUtil.PhoneNumberFormat.E164)
            } catch (e: Exception) {
                phoneNumber
            }
        }

        private fun fixNumber(phoneNumber: String): String {
            if (phoneNumber.length >= 11 && !phoneNumber.startsWith("+")) {
                return "+${phoneNumber}"
            }
            return phoneNumber
        }
    }
}
