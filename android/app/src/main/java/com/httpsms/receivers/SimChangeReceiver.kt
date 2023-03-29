package com.httpsms.receivers

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import com.httpsms.HttpSmsApiService
import com.httpsms.Settings
import com.httpsms.SmsManagerService
import timber.log.Timber


class SimChangeReceiver : BroadcastReceiver() {
    private var lastDualSIMState: Boolean = false
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action == "android.intent.action.SIM_STATE_CHANGED") {
            updateDualSimState(context)
        } else {
            Timber.e("invalid intent [${intent.action}]")
        }
    }

    private fun updateDualSimState(context: Context) {
        Thread {
            val currentTimeStamp = System.currentTimeMillis()
            val isDualSIM = SmsManagerService.isDualSIM(context)
            if (isDualSIM == lastDualSIMState) {
                return@Thread
            }
            val updated = HttpSmsApiService.create(context).updatePhone(
                Settings.getOwnerOrDefault(context),
                Settings.getFcmToken(context) ?: "",
                isDualSIM
            )
            Timber.d("SIM state change pushed to server")

            if (updated) {
                lastDualSIMState = isDualSIM
                Settings.setFcmTokenLastUpdateTimestampAsync(context, currentTimeStamp)
                Timber.i("fcm token uploaded successfully")
                return@Thread
            }
            Timber.e("could not update fcm token")
        }.start()
    }
}
