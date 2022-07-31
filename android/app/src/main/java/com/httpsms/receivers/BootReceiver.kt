package com.httpsms.receivers

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import com.httpsms.services.StickyNotificationService
import timber.log.Timber


class BootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action == Intent.ACTION_BOOT_COMPLETED) {
           startStickyNotification(context)
        } else {
            Timber.e("invalid intent [${intent.action}]")
        }
    }
    private fun startStickyNotification(context: Context) {
        Timber.d("starting foreground service")
        val notificationIntent = Intent(context, StickyNotificationService::class.java)
        val service = context.startForegroundService(notificationIntent)
        Timber.d("foreground service started [${service?.className}]")
    }
}
