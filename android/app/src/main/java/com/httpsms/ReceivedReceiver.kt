package com.httpsms

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.provider.Telephony
import androidx.work.BackoffPolicy
import androidx.work.Constraints
import androidx.work.Data
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequest
import androidx.work.WorkManager
import androidx.work.Worker
import androidx.work.WorkerParameters
import androidx.work.workDataOf
import timber.log.Timber
import java.time.ZoneOffset
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter
import java.util.concurrent.TimeUnit

class ReceivedReceiver: BroadcastReceiver()
{
    override fun onReceive(context: Context,intent: Intent) {
        if (intent.action != Telephony.Sms.Intents.SMS_RECEIVED_ACTION) {
            Timber.e("received invalid intent with action [${intent.action}]")
            return
        }

        var smsSender = ""
        var smsBody = ""

        for (smsMessage in Telephony.Sms.Intents.getMessagesFromIntent(intent)) {
            smsSender = smsMessage.displayOriginatingAddress
            smsBody += smsMessage.messageBody
        }

        var sim = Constants.SIM1
        var owner = Settings.getSIM1PhoneNumber(context)
        if (intent.getIntExtra("android.telephony.extra.SLOT_INDEX", 0) > 0 && Settings.isDualSIM(context)) {
            owner = Settings.getSIM2PhoneNumber(context)
            sim = Constants.SIM2
        }

        if (!Settings.isIncomingMessageEnabled(context, sim)) {
            Timber.w("[${sim}] is not active for incoming messages")
            return
        }

        handleMessageReceived(
            context,
            sim,
            smsSender,
            owner,
            smsBody
        )
    }

    private fun handleMessageReceived(context: Context, sim: String, from: String, to : String, content: String) {
        val timestamp = ZonedDateTime.now(ZoneOffset.UTC)

        if (!Settings.isLoggedIn(context)) {
            Timber.w("[${sim}] user is not logged in")
            return
        }

        if (!Settings.getActiveStatus(context, sim)) {
            Timber.w("[${sim}] user is not active")
            return
        }

        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.CONNECTED)
            .build()

        val inputData: Data = workDataOf(
            Constants.KEY_MESSAGE_FROM to from,
            Constants.KEY_MESSAGE_TO to to,
            Constants.KEY_MESSAGE_SIM to sim,
            Constants.KEY_MESSAGE_CONTENT to content,
            Constants.KEY_MESSAGE_TIMESTAMP to DateTimeFormatter.ofPattern(Constants.TIMESTAMP_PATTERN).format(timestamp).replace("+", "Z")
        )

        val work = OneTimeWorkRequest
            .Builder(ReceivedSmsWorker::class.java)
            .setConstraints(constraints)
            .setInputData(inputData)
            .build()

        WorkManager
            .getInstance(context)
            .enqueue(work)

        Timber.d("work enqueued with ID [${work.id}] for received message from [${from}] to [${to}]")
    }

    internal class ReceivedSmsWorker(appContext: Context, workerParams: WorkerParameters) : Worker(appContext, workerParams) {
        override fun doWork(): Result {
            Timber.i("[${this.inputData.getString(Constants.KEY_MESSAGE_SIM)}] forwarding received message from [${this.inputData.getString(Constants.KEY_MESSAGE_FROM)}] to [${this.inputData.getString(Constants.KEY_MESSAGE_TO)}]")

            if (HttpSmsApiService.create(applicationContext).receive(
                this.inputData.getString(Constants.KEY_MESSAGE_SIM)!!,
                this.inputData.getString(Constants.KEY_MESSAGE_FROM)!!,
                this.inputData.getString(Constants.KEY_MESSAGE_TO)!!,
                this.inputData.getString(Constants.KEY_MESSAGE_CONTENT)!!,
                this.inputData.getString(Constants.KEY_MESSAGE_TIMESTAMP)!!,
            )) {
                return Result.success()
            }

            return Result.retry()
        }
    }
}
