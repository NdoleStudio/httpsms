package com.httpsms

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.provider.Telephony
import android.util.Log
import java.time.ZoneOffset
import java.time.ZonedDateTime

class ReceivedReceiver: BroadcastReceiver()
{
    companion object {
        private val TAG = ReceivedReceiver::class.simpleName
    }

    override fun onReceive(context: Context,intent: Intent) {
        if (intent.action != Telephony.Sms.Intents.SMS_RECEIVED_ACTION) {
            Log.e(TAG, "received invalid intent with action [${intent.action}]")
            return
        }

        var smsSender = ""
        var smsBody = ""

        for (smsMessage in Telephony.Sms.Intents.getMessagesFromIntent(intent)) {
            smsSender = smsMessage.displayOriginatingAddress
            smsBody += smsMessage.messageBody
        }

        handleMessageReceived(
            smsSender,
            Settings.getOwner(context) ?: Settings.DEFAULT_PHONE_NUMBER,
            smsBody
        )
    }

    private fun handleMessageReceived(from: String, to : String, content: String) {
        val timestamp = ZonedDateTime.now(ZoneOffset.UTC)
        Thread {
            Log.i(TAG, "forwarding received message from [${from}]")
            HttpSmsApiService().receive(from, to, content, timestamp)
        }.start()
    }
}
