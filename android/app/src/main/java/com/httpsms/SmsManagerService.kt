package com.httpsms

import android.app.PendingIntent
import android.content.Context
import android.os.Build
import android.telephony.SmsManager

class SmsManagerService {
    companion object {
        const val ACTION_SMS_SENT = "SMS_SENT"
        const val ACTION_SMS_DELIVERED = "SMS_DELIVERED"
    }

    fun sendMessage(context: Context, message: Message, sentIntent:PendingIntent, deliveryIntent: PendingIntent) {
        getSmsManager(context)
            .sendTextMessage(message.contact, message.owner, message.content, sentIntent, deliveryIntent)
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
