package com.httpsms

import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import androidx.work.OneTimeWorkRequest
import androidx.work.WorkManager
import androidx.work.Worker
import androidx.work.WorkerParameters
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import timber.log.Timber


class MyFirebaseMessagingService : FirebaseMessagingService() {

    // [START receive_message]
    override fun onMessageReceived(remoteMessage: RemoteMessage) {
        Timber.d(MyFirebaseMessagingService::onMessageReceived.name)
        scheduleJob()
    }
    // [END receive_message]

    // [START on_new_token]
    /**
     * Called if the FCM registration token is updated. This may occur if the security of
     * the previous token had been compromised. Note that this is called when the
     * FCM registration token is initially generated so this is where you would retrieve the token.
     */
    override fun onNewToken(token: String) {
        Timber.d("Refreshed token: $token")

        // If you want to send messages to this application instance or
        // manage this apps subscriptions on the server side, send the
        // FCM registration token to your app server.
        sendRegistrationToServer(token)
    }
    // [END on_new_token]

    private fun scheduleJob() {
        // [START dispatch_job]
        val work = OneTimeWorkRequest
            .Builder(SendSmsWorker::class.java)
            .build()

        WorkManager
            .getInstance(this)
            .enqueue(work)
        // [END dispatch_job]
    }
    private fun sendRegistrationToServer(token: String) {
        Timber.d("sendRegistrationTokenToServer($token)")
        Settings.setFcmTokenAsync(this, token)

        if (Settings.isLoggedIn(this)) {
            Timber.d("updating phone with new fcm token")
            HttpSmsApiService(Settings.getApiKeyOrDefault(this)).updatePhone(Settings.getOwnerOrDefault(this), token)
        }

    }

    internal class SendSmsWorker(appContext: Context, workerParams: WorkerParameters) : Worker(appContext, workerParams) {
        override fun doWork(): Result {
            if (!Settings.isLoggedIn(applicationContext)) {
                Timber.w("user is not logged in, stopping processing")
                return Result.failure()
            }

            val owner = Settings.getOwner(applicationContext) ?: return Result.failure()
            val message = getMessage(applicationContext, owner) ?: return Result.failure()

            registerReceivers()

            sendMessage(
                message,
                createPendingIntent(message, SmsManagerService.ACTION_SMS_SENT),
                createPendingIntent(message, SmsManagerService.ACTION_SMS_DELIVERED)
            )

            return Result.success()
        }

        private fun getMessage(context: Context, owner: String): Message? {
            Timber.d("fetching message")
            val messages = HttpSmsApiService(Settings.getApiKeyOrDefault(context)).getOutstandingMessages(owner)

            if (messages.isNotEmpty()) {
                Timber.d("fetched message with ID [${messages.first().id}]")
                return messages.first()
            }

            Timber.e("cannot get message from API")
            return null
        }

        private fun sendMessage(message: Message, sentIntent: PendingIntent, deliveredIntent: PendingIntent) {
            Timber.d("sending SMS for message with ID [${message.id}]")
            SmsManagerService().sendMessage(this.applicationContext, message, sentIntent, deliveredIntent)
            Timber.d("sent SMS for message with ID [${message.id}]")
        }


        private fun registerReceivers() {
            this.applicationContext.registerReceiver(
                SentReceiver(),
                IntentFilter(SmsManagerService.ACTION_SMS_SENT)
            )

            this.applicationContext.registerReceiver(
                DeliveredReceiver(),
                IntentFilter(SmsManagerService.ACTION_SMS_DELIVERED)
            )
        }

        private fun createPendingIntent(message: Message, action: String): PendingIntent {
            val intent = Intent(action)
            intent.putExtra(Constants.KEY_MESSAGE_ID, message.id)

            return PendingIntent.getBroadcast(
                this.applicationContext,
                0,
                intent,
                PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
            )
        }
    }
}
