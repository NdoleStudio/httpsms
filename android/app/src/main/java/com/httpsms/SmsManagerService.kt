package com.httpsms

import android.annotation.SuppressLint
import android.app.PendingIntent
import android.content.Context
import android.os.Build
import android.telephony.SmsManager
import android.telephony.SubscriptionManager


class SmsManagerService {
    companion object {
        private const val ACTION_SMS_SENT = "SMS_SENT"
        private const val ACTION_SMS_DELIVERED = "SMS_DELIVERED"

        fun sentAction(messageID: String): String {
            return "$ACTION_SMS_SENT.$messageID"
        }

        fun deliveredAction(messageID: String): String {
            return "$ACTION_SMS_DELIVERED.$messageID"
        }

        @SuppressLint("MissingPermission")
        fun isDualSIM(context: Context) : Boolean {
            val localSubscriptionManager: SubscriptionManager = if (Build.VERSION.SDK_INT < 31) {
                SubscriptionManager.from(context)
            } else {
                context.getSystemService(SubscriptionManager::class.java)
            }
            return localSubscriptionManager.activeSubscriptionInfoList.size > 1
        }
    }

    fun messageParts(context: Context, content: String): ArrayList<String> {
        return getSmsManager(context).divideMessage(content)
    }

    fun sendMultipartMessage(context: Context, contact: String, parts: ArrayList<String>, sim: String, sendIntents: ArrayList<PendingIntent>, deliveryIntents: ArrayList<PendingIntent>) {
        getSmsManager(context, sim).sendMultipartTextMessage(contact, null, parts, sendIntents, deliveryIntents)
    }

    fun sendTextMessage(context: Context, contact: String, content: String, sim: String, sentIntent:PendingIntent, deliveryIntent: PendingIntent) {
        getSmsManager(context, sim).sendTextMessage(contact, null, content, sentIntent, deliveryIntent)
    }

    @Suppress("DEPRECATION")
    @SuppressLint("MissingPermission")
    private fun getSmsManager(context: Context, sim: String = "DEFAULT"): SmsManager {
        val localSubscriptionManager: SubscriptionManager = if (Build.VERSION.SDK_INT < 31) {
            SubscriptionManager.from(context)
        } else {
            context.getSystemService(SubscriptionManager::class.java)
        }

        val subscriptionId = if (sim == "SIM1" && localSubscriptionManager.activeSubscriptionInfoList.size > 0) {
            localSubscriptionManager.activeSubscriptionInfoList[0].subscriptionId
        } else if (sim == "SIM2" && localSubscriptionManager.activeSubscriptionInfoList.size > 1) {
            localSubscriptionManager.activeSubscriptionInfoList[1].subscriptionId
        } else{
            SubscriptionManager.getDefaultSmsSubscriptionId()
        }

        return if (Build.VERSION.SDK_INT < 31) {
            SmsManager.getSmsManagerForSubscriptionId(subscriptionId)
        } else {
            context.getSystemService(SmsManager::class.java).createForSubscriptionId(subscriptionId)
        }
    }
}
