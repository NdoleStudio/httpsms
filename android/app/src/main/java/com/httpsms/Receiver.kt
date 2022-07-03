package com.httpsms

import android.content.Context
import android.content.IntentFilter
import timber.log.Timber


object Receiver {
    fun isValid(context: Context, messageId: String?): Boolean {
        if (messageId == null) {
            Timber.e("cannot handle event because the message ID is null")
            return false
        }

        if (!Settings.isLoggedIn(context)) {
            Timber.w("cannot handle message with id [$messageId] because the user is not logged in")
            return false
        }

        if (!Settings.getActiveStatus(context)) {
            Timber.w("cannot handle message with id [$messageId] because the user is not active")
            return false
        }
        return true
    }

    fun registerReceivers(context: Context) {
        try {
            context.unregisterReceiver(SentReceiver())
        } catch (error: IllegalArgumentException ) {
            Timber.e(error)
        }

        context.registerReceiver(
            SentReceiver(),
            IntentFilter(SmsManagerService.ACTION_SMS_SENT)
        )

        try {
            context.unregisterReceiver(DeliveredReceiver())
        } catch (error: IllegalArgumentException ) {
            Timber.e(error)
        }

        context.registerReceiver(
            DeliveredReceiver(),
            IntentFilter(SmsManagerService.ACTION_SMS_DELIVERED)
        )
    }
}
