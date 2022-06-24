package com.httpsms

import android.app.PendingIntent
import android.content.Context
import android.telephony.SmsManager

class SmsManagerService {
    fun SendMessage(context: Context, message: Message, sentIntent:PendingIntent, deliveryIntent: PendingIntent) {
        val smsManager: SmsManager = context.getSystemService(SmsManager::class.java)
        smsManager.sendTextMessage(message.contact, message.owner, message.content, sentIntent, deliveryIntent)
    }
}
