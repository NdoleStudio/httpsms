package com.httpsms

import android.app.Activity
import android.content.Context
import android.content.Intent
import android.content.BroadcastReceiver
import android.util.Log
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
            Log.i(TAG, "delivered message with ID [${messageId}]")
            if (messageId == null) {
                Log.e(TAG, "cannot handle event because the message ID is null")
                return@Thread
            }
            HttpSmsApiService().sendDeliveredEvent(messageId, timestamp)
        }.start()
    }

    private fun handleMessageFailed(messageId: String?) {
        val timestamp = ZonedDateTime.now(ZoneOffset.UTC)
        Thread {
            Log.i(TAG, "message with ID [${messageId}] not delivered")
            if (messageId == null) {
                Log.e(TAG, "cannot handle event because the message ID is null")
                return@Thread
            }
            HttpSmsApiService().sendFailedEvent(messageId,timestamp, "NOT_DELIVERED")
        }.start()
    }

    companion object {
        private val TAG = DeliveredReceiver::class.simpleName
    }
}
