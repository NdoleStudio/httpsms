package com.httpsms

import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.telephony.SmsManager
import android.util.Log
import java.time.ZoneOffset
import java.time.ZonedDateTime

internal class SentReceiver : BroadcastReceiver() {
    companion object {
        private val TAG = SentReceiver::class.simpleName
    }

    override fun onReceive(context: Context, intent: Intent) {
        when (resultCode) {
            Activity.RESULT_OK -> handleMessageSent(intent.getStringExtra(Constants.KEY_MESSAGE_ID))
            SmsManager.RESULT_ERROR_GENERIC_FAILURE -> handleMessageFailed(intent.getStringExtra(Constants.KEY_MESSAGE_ID), "GENERIC_FAILURE")
            SmsManager.RESULT_ERROR_NO_SERVICE -> handleMessageFailed(intent.getStringExtra(Constants.KEY_MESSAGE_ID), "NO_SERVICE")
            SmsManager.RESULT_ERROR_NULL_PDU -> handleMessageFailed(intent.getStringExtra(Constants.KEY_MESSAGE_ID), "NULL_PDU")
            SmsManager.RESULT_ERROR_RADIO_OFF -> handleMessageFailed(intent.getStringExtra(Constants.KEY_MESSAGE_ID), "RADIO_OFF")
            else -> handleMessageFailed(intent.getStringExtra(Constants.KEY_MESSAGE_ID), "UNKNOWN")
        }
    }

    private fun handleMessageSent(messageId: String?) {
        val timestamp = ZonedDateTime.now(ZoneOffset.UTC)
        Thread {
            Log.i(TAG, "sent message with ID [${messageId}]")
            if (messageId == null) {
                Log.e(TAG, "cannot handle event because the message ID is null")
                return@Thread
            }
            HttpSmsApiService().sendSentEvent(messageId,timestamp)
        }.start()
    }

    private fun handleMessageFailed(messageId: String?, reason: String) {
        val timestamp = ZonedDateTime.now(ZoneOffset.UTC)
        Thread {
            Log.i(TAG, "message with ID [${messageId}] not sent with reason [$reason]")
            if (messageId == null) {
                Log.e(TAG, "cannot handle event because the message ID is null")
                return@Thread
            }
            HttpSmsApiService().sendFailedEvent(messageId, timestamp, reason)
        }.start()
    }
}
