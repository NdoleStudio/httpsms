package com.httpsms

import android.app.PendingIntent
import android.content.Context
import android.os.Build
import android.telephony.SmsManager

class SmsManagerService {
    companion object {
        private const val ACTION_SMS_SENT = "SMS_SENT"
        private const val ACTION_SMS_DELIVERED = "SMS_DELIVERED"

        fun sentAction(messageID: String): String {
            return "$ACTION_SMS_SENT.$messageID"
        }

        fun deliveredAction(messageID: String): String {
            return "$ACTION_SMS_DELIVERED.$messageID"
        }
    }

    fun sendMessage(context: Context, message: Message, sentIntent:PendingIntent, deliveryIntent: PendingIntent) {
        getSmsManager(context)
            .sendTextMessage(message.contact, null, message.content, sentIntent, deliveryIntent)
    }

    @Suppress("DEPRECATION")
    private fun getSmsManager(context: Context): SmsManager {
        return if (Build.VERSION.SDK_INT < 31) {
            SmsManager.getDefault()
        } else {
            context.getSystemService(SmsManager::class.java)
        }
    }
}
