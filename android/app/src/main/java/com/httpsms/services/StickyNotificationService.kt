package com.httpsms.services

import android.app.*
import android.content.Context
import android.content.Intent
import android.graphics.Color
import android.os.IBinder
import android.widget.Toast
import com.httpsms.MainActivity
import com.httpsms.R
import timber.log.Timber

class StickyNotificationService: Service() {
    override fun onBind(intent: Intent?): IBinder? {
        Timber.d("Some component want to bind with the service [${intent?.action}]")
        return null
    }

    override fun onCreate() {
        Timber.d("The service has been created")
        super.onCreate()
        val notification = createNotification()
        startForeground(1, notification)
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        Timber.d("onStartCommand executed with startId: $startId")
        // by returning this we make sure the service is restarted if the system kills the service
        return START_STICKY
    }

    override fun onDestroy() {
        super.onDestroy()
        Timber.d("The service has been destroyed")
        Toast.makeText(this, "Service destroyed", Toast.LENGTH_SHORT).show()
    }


    private fun createNotification(): Notification {
        val notificationChannelId = "sticky_notification_channel"

        // depending on the Android API that we're dealing with we will have
        // to use a specific method to create the notification
        val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        val channel = NotificationChannel(
            notificationChannelId,
            notificationChannelId,
            NotificationManager.IMPORTANCE_HIGH
        ).let {
            it.enableLights(true)
            it.enableVibration(false)
            it.lightColor = Color.RED
            it
        }
        notificationManager.createNotificationChannel(channel)

        val pendingIntent: PendingIntent = Intent(this, MainActivity::class.java).let {
                notificationIntent -> PendingIntent.getActivity(
            this,
            0,
            notificationIntent,
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT)
        }

        val builder: Notification.Builder = Notification.Builder(
            this,
            notificationChannelId
        )

        return builder
            .setContentTitle("HTTP SMS")
            .setContentText("Service running in background")
            .setContentIntent(pendingIntent)
            .setSmallIcon(R.mipmap.ic_launcher_round)
            .setTicker("Ticker text")
            .build()
    }
}
