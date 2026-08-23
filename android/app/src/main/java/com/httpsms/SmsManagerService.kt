package com.httpsms

import android.Manifest
import android.annotation.SuppressLint
import android.app.PendingIntent
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import android.telephony.SmsManager
import android.telephony.SubscriptionManager
import androidx.core.app.ActivityCompat
import timber.log.Timber


class SmsManagerService {
    companion object {
        private const val ACTION_SMS_SENT = "SMS_SENT"
        private const val ACTION_SMS_DELIVERED = "SMS_DELIVERED"

        fun sentAction(): String {
            return "${BuildConfig.APPLICATION_ID}.$ACTION_SMS_SENT"
        }

        fun deliveredAction(): String {
            return "${BuildConfig.APPLICATION_ID}.$ACTION_SMS_DELIVERED"
        }

        fun isDualSIM(context: Context) : Boolean {
            if (ActivityCompat.checkSelfPermission(context, Manifest.permission.READ_PHONE_STATE) != PackageManager.PERMISSION_GRANTED
            ) {
                Timber.w("cannot check if dual sim, no permission")
                return false
            }
            val localSubscriptionManager: SubscriptionManager = if (Build.VERSION.SDK_INT < 31) {
                SubscriptionManager.from(context)
            } else {
                context.getSystemService(SubscriptionManager::class.java)
            }
            return (localSubscriptionManager.activeSubscriptionInfoList?.size ?: 0) > 1
        }
    }

    fun messageParts(context: Context, content: String): ArrayList<String> {
        return getSmsManager(context).divideMessage(content)
    }

    fun sendMultipartMessage(context: Context, contact: String, parts: ArrayList<String>, sim: String, sendIntents: ArrayList<PendingIntent>, deliveryIntents: ArrayList<PendingIntent>) {
        try {
            getSmsManager(context, sim).sendMultipartTextMessage(contact, null, parts, sendIntents, deliveryIntents)
        } catch (e: NullPointerException) {
            if (e.message?.contains("EmergencyNumber.getNumber()") == true) {
                Timber.w(e, "Caught EmergencyNumber NPE, falling back to default SmsManager")
                getDefaultSmsManager(context).sendMultipartTextMessage(contact, null, parts, sendIntents, deliveryIntents)
            } else {
                throw e
            }
        }
    }

    fun sendTextMessage(context: Context, contact: String, content: String, sim: String, sentIntent:PendingIntent, deliveryIntent: PendingIntent) {
        try {
            getSmsManager(context, sim).sendTextMessage(contact, null, content, sentIntent, deliveryIntent)
        } catch (e: NullPointerException) {
            if (e.message?.contains("EmergencyNumber.getNumber()") == true) {
                Timber.w(e, "Caught EmergencyNumber NPE, falling back to default SmsManager")
                getDefaultSmsManager(context).sendTextMessage(contact, null, content, sentIntent, deliveryIntent)
            } else {
                throw e
            }
        }
    }

    @Suppress("DEPRECATION")
    private fun getDefaultSmsManager(context: Context): SmsManager {
        return if (Build.VERSION.SDK_INT >= 31) {
            context.getSystemService(SmsManager::class.java)
        } else {
            SmsManager.getDefault()
        }
    }

    @Suppress("DEPRECATION")
    @SuppressLint("MissingPermission")
    private fun getSmsManager(context: Context, sim: String = Constants.SIM1): SmsManager {
        val localSubscriptionManager: SubscriptionManager = if (Build.VERSION.SDK_INT < 31) {
            SubscriptionManager.from(context)
        } else {
            context.getSystemService(SubscriptionManager::class.java)
        }

        val infoList = localSubscriptionManager.activeSubscriptionInfoList
        Timber.d("active subscription info size: [${infoList?.size ?: 0}]")

        val subscriptionId = if (sim == Constants.SIM1 && !infoList.isNullOrEmpty()) {
            infoList[0].subscriptionId
        } else if (sim == Constants.SIM2 && (infoList?.size ?: 0) > 1) {
            infoList!![1].subscriptionId
        } else{
            SubscriptionManager.getDefaultSmsSubscriptionId()
        }

        if (subscriptionId == SubscriptionManager.INVALID_SUBSCRIPTION_ID) {
            return getDefaultSmsManager(context)
        }

        return if (Build.VERSION.SDK_INT < 31) {
            SmsManager.getSmsManagerForSubscriptionId(subscriptionId)
        } else {
            context.getSystemService(SmsManager::class.java).createForSubscriptionId(subscriptionId)
        }
    }

    // Wrapper for the smsManager's sendMultimediaMessage
    fun sendMultimediaMessage(context: Context, pduUri: android.net.Uri, sim: String, sentIntent: PendingIntent) {
        val smsManager = getSmsManager(context, sim)
        smsManager.sendMultimediaMessage(context, pduUri, null, null, sentIntent)
    }
}
