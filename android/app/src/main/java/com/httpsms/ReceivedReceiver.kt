package com.httpsms

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.provider.Telephony
import timber.log.Timber
import java.time.ZoneOffset
import java.time.ZonedDateTime

class ReceivedReceiver: BroadcastReceiver()
{
    override fun onReceive(context: Context,intent: Intent) {
        if (intent.action != Telephony.Sms.Intents.SMS_RECEIVED_ACTION) {
            Timber.e("received invalid intent with action [${intent.action}]")
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
            Timber.i("forwarding received message from [${from}]")
            HttpSmsApiService().receive(from, to, content, timestamp)
        }.start()
    }
}