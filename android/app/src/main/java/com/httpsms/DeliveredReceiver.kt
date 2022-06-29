package com.httpsms

import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import timber.log.Timber
import java.time.ZoneOffset
import java.time.ZonedDateTime


internal class DeliveredReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        when (resultCode) {
            Activity.RESULT_OK -> handleMessageDelivered(context, intent.getStringExtra(Constants.KEY_MESSAGE_ID))
            else -> handleMessageFailed(context, intent.getStringExtra(Constants.KEY_MESSAGE_ID))
        }
    }

    private fun handleMessageDelivered(context: Context, messageId: String?) {
        val timestamp = ZonedDateTime.now(ZoneOffset.UTC)
        if (!Receiver.isValid(context, messageId)) {
            return
        }
        Thread {
            Timber.i("delivered message with ID [${messageId}]")
            HttpSmsApiService(Settings.getApiKeyOrDefault(context)).sendDeliveredEvent(messageId!!, timestamp)
        }.start()
    }

    private fun handleMessageFailed(context: Context, messageId: String?) {
        val timestamp = ZonedDateTime.now(ZoneOffset.UTC)
        if (!Receiver.isValid(context, messageId)) {
            return
        }

        Thread {
            Timber.i("message with ID [${messageId}] not delivered")
            HttpSmsApiService(Settings.getApiKeyOrDefault(context)).sendFailedEvent(messageId!!,timestamp, "NOT_DELIVERED")
        }.start()
    }
}
