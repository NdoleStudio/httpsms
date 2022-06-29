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
            Activity.RESULT_OK -> handleMessageDelivered(intent.getStringExtra(Constants.KEY_MESSAGE_ID))
            else -> handleMessageFailed(intent.getStringExtra(Constants.KEY_MESSAGE_ID))
        }
    }

    private fun handleMessageDelivered(messageId: String?) {
        val timestamp = ZonedDateTime.now(ZoneOffset.UTC)
        Thread {
            Timber.i("delivered message with ID [${messageId}]")
            if (messageId == null) {
                Timber.e("cannot handle event because the message ID is null")
                return@Thread
            }
            HttpSmsApiService().sendDeliveredEvent(messageId, timestamp)
        }.start()
    }

    private fun handleMessageFailed(messageId: String?) {
        val timestamp = ZonedDateTime.now(ZoneOffset.UTC)
        Thread {
            Timber.i("message with ID [${messageId}] not delivered")
            if (messageId == null) {
                Timber.e("cannot handle event because the message ID is null")
                return@Thread
            }
            HttpSmsApiService().sendFailedEvent(messageId,timestamp, "NOT_DELIVERED")
        }.start()
    }
}
