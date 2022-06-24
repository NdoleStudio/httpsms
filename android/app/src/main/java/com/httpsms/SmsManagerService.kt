package com.httpsms

import android.app.PendingIntent
import android.content.Context
import android.telephony.SmsManager

class SmsManagerService {
    companion object {
        const val ACTION_SMS_SENT = "SMS_SENT"
        const val ACTION_SMS_RECEIVED = "SMS_RECEIVED"
        const val ACTION_SMS_DELIVERED = "SMS_DELIVERED"
    }
    fun sendMessage(context: Context, message: Message, sentIntent:PendingIntent, deliveryIntent: PendingIntent) {
        val smsManager: SmsManager = context.getSystemService(SmsManager::class.java)
        smsManager.sendTextMessage(message.contact, message.owner, message.content, sentIntent, deliveryIntent)
    }
}
