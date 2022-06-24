package com.httpsms

import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.util.Log
import androidx.work.OneTimeWorkRequest
import androidx.work.WorkManager
import androidx.work.Worker
import androidx.work.WorkerParameters
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage


class MyFirebaseMessagingService : FirebaseMessagingService() {

    // [START receive_message]
    override fun onMessageReceived(remoteMessage: RemoteMessage) {
        // TODO(developer): Handle FCM messages here.
        // Not getting messages here? See why this may be: https://goo.gl/39bRNJ
        Log.d(TAG, "From: ${remoteMessage.from}")

        scheduleJob()

        // Check if message contains a notification payload.
        remoteMessage.notification?.let {
            Log.d(TAG, "Message Notification Body: ${it.body}")
        }

        // Also if you intend on generating your own notifications as a result of a received FCM
        // message, here is where that should be initiated. See sendNotification method below.
    }
    // [END receive_message]

    // [START on_new_token]
    /**
     * Called if the FCM registration token is updated. This may occur if the security of
     * the previous token had been compromised. Note that this is called when the
     * FCM registration token is initially generated so this is where you would retrieve the token.
     */
    override fun onNewToken(token: String) {
        Log.d(TAG, "Refreshed token: $token")

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
    private fun sendRegistrationToServer(token: String?) {
        // TODO: Implement this method to send token to your app server.
        Log.d(TAG, "sendRegistrationTokenToServer($token)")
    }

    companion object {
        private val TAG = MyFirebaseMessagingService::class.simpleName
    }

    internal class SendSmsWorker(appContext: Context, workerParams: WorkerParameters) : Worker(appContext, workerParams) {
        override fun doWork(): Result {
            val message = getMessage() ?: return Result.failure()

            registerReceivers()

            sendMessage(
                message,
                createPendingIntent(message, SmsManagerService.ACTION_SMS_SENT),
                createPendingIntent(message, SmsManagerService.ACTION_SMS_DELIVERED)
            )

            return Result.success()
        }

        private fun getMessage(): Message? {
            Log.i(TAG, "fetching message")
            val messages = HttpSmsApiService().getOutstandingMessages()

            if (messages.isNotEmpty()) {
                Log.i(TAG, "fetched message with ID [${messages.first().id}]")
                return messages.first()
            }

            Log.e(TAG, "cannot get message from API")
            return null
        }

        private fun sendMessage(message: Message, sentIntent: PendingIntent, deliveredIntent: PendingIntent) {
            Log.i(TAG, "sending SMS for message with ID [${message.id}]")
            SmsManagerService().sendMessage(this.applicationContext, message, sentIntent, deliveredIntent)
            Log.i(TAG, "sent SMS for message with ID [${message.id}]")
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

            Log.w(TAG, "message id = [${intent.getStringExtra(Constants.KEY_MESSAGE_ID)}]")

            return PendingIntent.getBroadcast(
                this.applicationContext,
                0,
                intent,
                PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
            )
        }
    }
}
