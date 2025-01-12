package com.httpsms

import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.telephony.SmsManager
import androidx.work.Constraints
import androidx.work.Data
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequest
import androidx.work.WorkManager
import androidx.work.Worker
import androidx.work.WorkerParameters
import androidx.work.workDataOf
import timber.log.Timber

internal class SentReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        when (resultCode) {
            Activity.RESULT_OK -> handleMessageSent(context, intent.getStringExtra(Constants.KEY_MESSAGE_ID))
            SmsManager.RESULT_ERROR_GENERIC_FAILURE -> handleMessageFailed(context, intent.getStringExtra(Constants.KEY_MESSAGE_ID), "GENERIC_FAILURE")
            SmsManager.RESULT_ERROR_NO_SERVICE -> handleMessageFailed(context, intent.getStringExtra(Constants.KEY_MESSAGE_ID), "NO_SERVICE")
            SmsManager.RESULT_ERROR_NULL_PDU -> handleMessageFailed(context, intent.getStringExtra(Constants.KEY_MESSAGE_ID), "NULL_PDU")
            SmsManager.RESULT_ERROR_RADIO_OFF -> handleMessageFailed(context, intent.getStringExtra(Constants.KEY_MESSAGE_ID), "RADIO_OFF")
            else -> handleMessageFailed(context, intent.getStringExtra(Constants.KEY_MESSAGE_ID), "UNKNOWN")
        }
    }

    private fun handleMessageSent(context: Context, messageId: String?) {
        if (!Receiver.isValid(context, messageId)) {
            return
        }

        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.CONNECTED)
            .build()

        val inputData: Data = workDataOf(
            Constants.KEY_MESSAGE_ID to messageId,
            Constants.KEY_MESSAGE_TIMESTAMP to Settings.currentTimestamp()
        )

        val work = OneTimeWorkRequest
            .Builder(SentMessageWorker::class.java)
            .setConstraints(constraints)
            .setInputData(inputData)
            .build()

        WorkManager
            .getInstance(context)
            .enqueue(work)

        Timber.d("work enqueued with ID [${work.id}] for [SENT] message with ID [${messageId}]")
    }

    private fun handleMessageFailed(context: Context, messageId: String?, reason: String) {
        if (!Receiver.isValid(context, messageId)) {
            return
        }

        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.CONNECTED)
            .build()

        val inputData: Data = workDataOf(
            Constants.KEY_MESSAGE_ID to messageId,
            Constants.KEY_MESSAGE_REASON to reason,
            Constants.KEY_MESSAGE_TIMESTAMP to Settings.currentTimestamp()
        )

        val work = OneTimeWorkRequest
            .Builder(FailedMessageWorker::class.java)
            .setConstraints(constraints)
            .setInputData(inputData)
            .build()

        WorkManager
            .getInstance(context)
            .enqueue(work)

        Timber.d("work enqueued with ID [${work.id}] for [FAILED] message with ID [${messageId}]")
    }

    internal class SentMessageWorker(appContext: Context, workerParams: WorkerParameters) : Worker(appContext, workerParams) {
        override fun doWork(): Result {
            val messageId = this.inputData.getString(Constants.KEY_MESSAGE_ID)
            val timestamp = this.inputData.getString(Constants.KEY_MESSAGE_TIMESTAMP)

            Timber.i("[${timestamp}] sending [SENT] message event with ID [${messageId}]")

            if (HttpSmsApiService.create(applicationContext).sendSentEvent(messageId!!, timestamp!!)){
                return Result.success()
            }
            return Result.retry()
        }
    }

    internal class FailedMessageWorker(appContext: Context, workerParams: WorkerParameters) : Worker(appContext, workerParams) {
        override fun doWork(): Result {
            val messageId = this.inputData.getString(Constants.KEY_MESSAGE_ID)
            val reason = this.inputData.getString(Constants.KEY_MESSAGE_REASON)
            val timestamp = this.inputData.getString(Constants.KEY_MESSAGE_TIMESTAMP)

            Timber.i("[${timestamp}] sending [FAILED] message event with ID [${messageId}] and reason [$reason]")

            if (HttpSmsApiService.create(applicationContext).sendFailedEvent(messageId!!, timestamp!!, reason!!)){
                return Result.success()
            }
            return Result.retry()
        }
    }
}
