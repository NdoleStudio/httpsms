package com.httpsms

import android.content.Context
import android.content.Context.RECEIVER_EXPORTED
import android.content.IntentFilter
import android.os.Build
import androidx.annotation.RequiresApi
import timber.log.Timber

object Receiver {
    private var sentReceiver: SentReceiver? = null;
    private var deliveredReceiver: DeliveredReceiver? = null;

    fun register(context: Context) {
        if(sentReceiver == null) {
            Timber.d("registering [sent] receiver for intent [${SmsManagerService.sentAction()}]")
            sentReceiver = SentReceiver()
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                context.registerReceiver(
                    sentReceiver,
                    IntentFilter(SmsManagerService.sentAction()),
                    RECEIVER_EXPORTED
                )
            } else {
                context.registerReceiver(
                    sentReceiver,
                    IntentFilter(SmsManagerService.sentAction())
                )
            }
        }

        if(deliveredReceiver == null) {
            Timber.d("registering [delivered] receiver for intent [${SmsManagerService.deliveredAction()}]")
            deliveredReceiver = DeliveredReceiver()
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                context.registerReceiver(
                    deliveredReceiver,
                    IntentFilter(SmsManagerService.deliveredAction()),
                    RECEIVER_EXPORTED
                )
            } else {
                context.registerReceiver(
                    deliveredReceiver,
                    IntentFilter(SmsManagerService.deliveredAction())
                )
            }
        }
    }

    fun isValid(context: Context, messageId: String?): Boolean {
        if (messageId == null) {
            Timber.e("cannot handle event because the message ID is null")
            return false
        }

        if (!Settings.isLoggedIn(context)) {
            Timber.w("cannot handle message with id [$messageId] because the user is not logged in")
            return false
        }

        if (!Settings.getActiveStatus(context, Constants.SIM1) && !Settings.getActiveStatus(context, Constants.SIM2)) {
            Timber.w("cannot handle message with id [$messageId] because the user is not active")
            return false
        }

        if(messageId.contains(".")) {
            Timber.d("message id [${messageId}] is for multipart segment [${messageId.split(".")[1]}]")
            return false
        }

        return true
    }
}
