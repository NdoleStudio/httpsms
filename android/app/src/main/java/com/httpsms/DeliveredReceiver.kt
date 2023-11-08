package com.httpsms

import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import androidx.work.Constraints
import androidx.work.Data
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequest
import androidx.work.WorkManager
import androidx.work.Worker
import androidx.work.WorkerParameters
import androidx.work.workDataOf
import timber.log.Timber


internal class DeliveredReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        when (resultCode) {
            Activity.RESULT_OK -> handleMessageDelivered(context, intent.getStringExtra(Constants.KEY_MESSAGE_ID))
            else -> handleMessageFailed(context, intent.getStringExtra(Constants.KEY_MESSAGE_ID))
        }
    }

    private fun handleMessageDelivered(context: Context, messageId: String?) {
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
            .Builder(DeliveredMessageWorker::class.java)
            .setConstraints(constraints)
            .setInputData(inputData)
            .build()

        WorkManager
            .getInstance(context)
            .enqueue(work)

        Timber.d("work enqueued with ID [${work.id}] for [DELIVERED] message with ID [${messageId}]")
    }

    private fun handleMessageFailed(context: Context, messageId: String?) {
        if (!Receiver.isValid(context, messageId)) {
            return
        }

        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.CONNECTED)
            .build()

        val inputData: Data = workDataOf(
            Constants.KEY_MESSAGE_ID to messageId,
            Constants.KEY_MESSAGE_REASON to "CANNOT BE DELIVERED",
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


    internal class DeliveredMessageWorker(appContext: Context, workerParams: WorkerParameters) : Worker(appContext, workerParams) {
        override fun doWork(): Result {
            val messageId = this.inputData.getString(Constants.KEY_MESSAGE_ID)
            val timestamp = this.inputData.getString(Constants.KEY_MESSAGE_TIMESTAMP)

            Timber.i("[${timestamp}] sending [DELIVERED] message event with ID [${messageId}]")

            if (HttpSmsApiService.create(applicationContext).sendDeliveredEvent(messageId!!, timestamp!!)){
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
