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
            context,
            smsSender,
            Settings.getOwnerOrDefault(context),
            smsBody
        )
    }

    private fun handleMessageReceived(context: Context, from: String, to : String, content: String) {
        val timestamp = ZonedDateTime.now(ZoneOffset.UTC)

        if (!Settings.isLoggedIn(context)) {
            Timber.w("user is not logged in")
            return
        }

        if (!Settings.getActiveStatus(context)) {
            Timber.w("user is not active")
            return
        }

        Thread {
            Timber.i("forwarding received message from [${from}]")
            HttpSmsApiService.create(context).receive(from, to, content, timestamp)
        }.start()
    }
}
