package com.httpsms.receivers

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import com.httpsms.services.StickyNotificationService
import timber.log.Timber


class BootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action == Intent.ACTION_BOOT_COMPLETED) {
            Intent(context, StickyNotificationService::class.java).also {
                context.startForegroundService(it)
            }
        } else {
            Timber.e("invalid intent [${intent.action}]")
        }
    }
}
