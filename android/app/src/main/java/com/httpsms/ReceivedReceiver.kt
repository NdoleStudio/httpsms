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

        var sim = Constants.SIM1
        var owner = Settings.getSIM1PhoneNumber(context)
        if (intent.getIntExtra("simSlot", 1) > 1 && Settings.isDualSIM(context)) {
            owner = Settings.getSIM2PhoneNumber(context)
            sim = Constants.SIM2
        }

        if (!Settings.isIncomingMessageEnabled(context, sim)) {
            Timber.w("[${sim}] is not active for incoming messages")
            return
        }

        handleMessageReceived(
            context,
            smsSender,
            owner,
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
