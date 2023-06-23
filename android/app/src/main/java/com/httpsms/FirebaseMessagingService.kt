package com.httpsms

import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import androidx.work.*
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import timber.log.Timber
import java.time.ZoneOffset
import java.time.ZonedDateTime

class MyFirebaseMessagingService : FirebaseMessagingService() {
    // [START receive_message]
    override fun onMessageReceived(remoteMessage: RemoteMessage) {
        initTimber()
        Timber.d(MyFirebaseMessagingService::onMessageReceived.name)

        if (remoteMessage.data.containsKey(Constants.KEY_HEARTBEAT_ID)) {
            Timber.w("received heartbeat message with ID [${remoteMessage.data[Constants.KEY_HEARTBEAT_ID]}] and priority [${remoteMessage.priority}] and original priority [${remoteMessage.originalPriority}]")
            sendHeartbeat()
            return
        }

        val messageID = remoteMessage.data[Constants.KEY_MESSAGE_ID]
        if (messageID == null)  {
            Timber.e("cannot get message id from notification data with key [${Constants.KEY_MESSAGE_ID}]")
            return
        }

        scheduleJob(messageID)
    }
    // [END receive_message]

    // [START on_new_token]
    /**
     * Called if the FCM registration token is updated. This may occur if the security of
     * the previous token had been compromised. Note that this is called when the
     * FCM registration token is initially generated so this is where you would retrieve the token.
     */
    override fun onNewToken(token: String) {
        initTimber()
        Timber.d("Refreshed token: $token")

        // If you want to send messages to this application instance or
        // manage this apps subscriptions on the server side, send the
        // FCM registration token to your app server.
        sendRegistrationToServer(token)
    }
    // [END on_new_token]

    private fun sendHeartbeat() {
        Timber.d("sending heartbeat from FCM notification")
        if (!Settings.isLoggedIn(applicationContext)) {
            Timber.w("user is not logged in, not sending heartbeat")
            return
        }
        Thread {
            try {
                HttpSmsApiService.create(applicationContext).storeHeartbeat(Settings.getSIM1PhoneNumber(applicationContext))
                Settings.setHeartbeatTimestampAsync(applicationContext, System.currentTimeMillis())
            } catch (exception: Exception) {
                Timber.e(exception)
            }
            Timber.d("finished sending pulse")
        }.start()
    }

    private fun scheduleJob(messageID: String) {
        // [START dispatch_job]
        val inputData: Data = workDataOf(Constants.KEY_MESSAGE_ID to messageID)
        val work = OneTimeWorkRequest
            .Builder(SendSmsWorker::class.java)
            .setInputData(inputData)
            .addTag(messageID)
            .build()

        WorkManager
            .getInstance(this)
            .enqueue(work)

        Timber.d("work enqueued with ID [${work.id}] for messageID [${messageID}]")
        // [END dispatch_job]
    }
    private fun sendRegistrationToServer(token: String) {
        Timber.d("sendRegistrationTokenToServer($token)")
        Settings.setFcmTokenAsync(this, token)

        if (Settings.isLoggedIn(this)) {
            Timber.d("updating SIM1 phone with new fcm token")
            HttpSmsApiService.create(this).updatePhone(Settings.getSIM1PhoneNumber(this), token, Constants.SIM1)
        }

        if(Settings.isDualSIM(this)) {
            Timber.d("updating SIM2 phone with new fcm token")
            HttpSmsApiService.create(this).updatePhone(Settings.getSIM2PhoneNumber(this), token, Constants.SIM2)
        }
    }

    private fun initTimber() {
        if (Timber.treeCount > 1) {
            Timber.d("timber is already initialized with count [${Timber.treeCount}]")
            return
        }

        if (BuildConfig.DEBUG) {
            Timber.plant(Timber.DebugTree())
            Timber.plant(LogtailTree())
        }
    }

    internal class SendSmsWorker(appContext: Context, workerParams: WorkerParameters) : Worker(appContext, workerParams) {
        override fun doWork(): Result {
            if (!Settings.isLoggedIn(applicationContext)) {
                Timber.w("user is not logged in, stopping processing")
                return Result.failure()
            }

            val messageID = this.inputData.getString(Constants.KEY_MESSAGE_ID)
            if (messageID == null) {
                Timber.e("cannot get outstanding message for work [${this.id}]")
                return Result.failure()
            }

            val message = getMessage(applicationContext, messageID) ?: return Result.failure()
            if (!Settings.getActiveStatus(applicationContext, message.sim)) {
                Timber.w("[${message.sim}] SIM is not active, stopping processing")
                handleFailed(applicationContext, messageID)
                return Result.failure()
            }

            val parts = getMessageParts(applicationContext, message)
            if (parts.size == 1) {
                return handleSingleMessage(applicationContext, message)
            }
            return handleMultipartMessage(applicationContext, message, parts)
        }

        private fun handleMultipartMessage(context: Context, message:Message, parts: ArrayList<String>): Result {
            registerReceivers(context, message.id)

            Timber.d("sending SMS for message with ID [${message.id}]")
            return try {
                val sentIntents = ArrayList<PendingIntent>()
                val deliveredIntents = ArrayList<PendingIntent>()

                for (i in 0 until parts.size) {
                    var id = "${message.id}.$i"

                    // Listen for 'delivered' and 'sent' intents only on the last part in the
                    // multipart SMS message
                    if (i == parts.size -1) {
                        id = message.id
                    }

                    sentIntents.add(createPendingIntent(id, SmsManagerService.sentAction(id)))
                    deliveredIntents.add(createPendingIntent(id, SmsManagerService.deliveredAction(id)))
                }
                SmsManagerService().sendMultipartMessage(this.applicationContext,message.contact, parts, message.sim, sentIntents, deliveredIntents)
                Timber.d("sent SMS for message with ID [${message.id}] in [${parts.size}] parts")
                Result.success()
            } catch (e: Exception) {
                Timber.e(e)
                Timber.d("could not send SMS for message with ID [${message.id}] in [${parts.size}] parts")
                Result.failure()
            }
        }


        private fun handleSingleMessage(context: Context, message:Message): Result {
            registerReceivers(context, message.id)
            sendMessage(
                message,
                createPendingIntent(message.id, SmsManagerService.sentAction(message.id)),
                createPendingIntent(message.id, SmsManagerService.deliveredAction(message.id))
            )
            return Result.success()
        }

        private fun registerReceivers(context: Context, messageID: String) {
            context.registerReceiver(
                SentReceiver(),
                IntentFilter(SmsManagerService.sentAction(messageID))
            )
            context.registerReceiver(
                DeliveredReceiver(),
                IntentFilter(SmsManagerService.deliveredAction(messageID))
            )
        }

        private fun handleFailed(context: Context, messageID: String) {
            Timber.d("sending failed event for message with ID [${messageID}]")
            HttpSmsApiService.create(context)
                .sendFailedEvent(messageID, ZonedDateTime.now(ZoneOffset.UTC), "MOBILE_APP_INACTIVE")
        }

        private fun getMessage(context: Context, messageID: String): Message? {
            Timber.d("fetching message with ID [${messageID}]")
            val message =  HttpSmsApiService.create(context).getOutstandingMessage(messageID)

            if (message != null) {
                Timber.d("fetched message with ID [${message.id}]")
                return message
            }

            Timber.e("cannot get message from API with ID [${messageID}]")
            return null
        }

        private fun sendMessage(message: Message, sentIntent: PendingIntent, deliveredIntent: PendingIntent) {
            Timber.d("sending SMS for message with ID [${message.id}]")
            try {
                SmsManagerService().sendTextMessage(this.applicationContext,message.contact, message.content, message.sim, sentIntent, deliveredIntent)
            } catch (e: Exception) {
                Timber.e(e)
                Timber.d("could not send SMS for message with ID [${message.id}]")
                return
            }
            Timber.d("sent SMS for message with ID [${message.id}]")
        }

        private fun getMessageParts(context: Context, message: Message): ArrayList<String> {
            Timber.d("getting parts for message with ID [${message.id}]")
            return try {
                val parts = SmsManagerService().messageParts(context, message.content)
                Timber.d("message with ID [${message.id}] has [${parts.size}] parts")
                parts
            } catch (e: Exception) {
                Timber.e(e)
                Timber.d("could not get parts message with ID [${message.id}] returning [1] part with entire content")
                val list = ArrayList<String>()
                list.add(message.content)
                list
            }
        }

        private fun createPendingIntent(id: String, action: String): PendingIntent {
            val intent = Intent(action)
            intent.putExtra(Constants.KEY_MESSAGE_ID, id)

            return PendingIntent.getBroadcast(
                this.applicationContext,
                0,
                intent,
                PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
            )
        }
    }
}
