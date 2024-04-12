package com.httpsms.receivers

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.telephony.TelephonyManager
import timber.log.Timber


class PhoneStateReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        Timber.d("onReceive: ${intent.action}")
        val stateStr = intent.extras!!.getString(TelephonyManager.EXTRA_STATE)
        val subscriptionId = intent.extras!!.getString(TelephonyManager.EXTRA_SUBSCRIPTION_ID)
        val number = intent.extras!!.getString(TelephonyManager.EXTRA_INCOMING_NUMBER)
        Timber.w("state = [${stateStr}] number = [${number}], subscriptionID = [${subscriptionId}]")
        val bundle = intent.extras
        if (bundle != null) {
            for (key in bundle.keySet()) {
                Timber.w(key + " : " + if (bundle[key] != null) bundle[key] else "NULL")
            }
        }
    }
}
