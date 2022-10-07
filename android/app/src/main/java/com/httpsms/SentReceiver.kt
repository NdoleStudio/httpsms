package com.httpsms

import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.telephony.SmsManager
import timber.log.Timber
import java.time.ZoneOffset
import java.time.ZonedDateTime

internal class SentReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        when (resultCode) {
            Activity.RESULT_OK -> handleMessageSent(context, intent.getStringExtra(Constants.KEY_MESSAGE_ID))
            SmsManager.RESULT_ERROR_GENERIC_FAILURE -> handleMessageFailed(context, intent.getStringExtra(Constants.KEY_MESSAGE_ID), "GENERIC_FAILURE")
            SmsManager.RESULT_ERROR_NO_SERVICE -> handleMessageFailed(context, intent.getStringExtra(Constants.KEY_MESSAGE_ID), "NO_SERVICE")
            SmsManager.RESULT_ERROR_NULL_PDU -> handleMessageFailed(context, intent.getStringExtra(Constants.KEY_MESSAGE_ID), "NULL_PDU")
            SmsManager.RESULT_ERROR_RADIO_OFF -> handleMessageFailed(context, intent.getStringExtra(Constants.KEY_MESSAGE_ID), "RADIO_OFF")
            else -> handleMessageFailed(context, intent.getStringExtra(Constants.KEY_MESSAGE_ID), "UNKNOWN")
        }
    }

    private fun handleMessageSent(context: Context, messageId: String?) {
        val timestamp = ZonedDateTime.now(ZoneOffset.UTC)
        if (!Receiver.isValid(context, messageId)) {
            return
        }

        Thread {
            Timber.d("sent message with ID [${messageId}]")
            HttpSmsApiService.create(context).sendSentEvent(messageId!!,timestamp)
        }.start()
    }

    private fun handleMessageFailed(context: Context, messageId: String?, reason: String) {
        val timestamp = ZonedDateTime.now(ZoneOffset.UTC)
        if (!Receiver.isValid(context, messageId)) {
            return
        }

        Thread {
            Timber.i("message with ID [${messageId}] not sent with reason [$reason]")
            HttpSmsApiService.create(context).sendFailedEvent(messageId!!, timestamp, reason)
        }.start()
    }
}
