package com.httpsms.receivers

import android.annotation.SuppressLint
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.os.Build
import android.provider.CallLog
import android.telephony.SubscriptionManager
import android.telephony.TelephonyManager
import androidx.work.Constraints
import androidx.work.Data
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequest
import androidx.work.WorkManager
import androidx.work.Worker
import androidx.work.WorkerParameters
import androidx.work.workDataOf
import com.httpsms.Constants
import com.httpsms.HttpSmsApiService
import com.httpsms.Settings
import timber.log.Timber
import java.time.Instant
import java.time.ZoneOffset
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter


class PhoneStateReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        Timber.d("onReceive: ${intent.action}")
        val stateStr = intent.extras!!.getString(TelephonyManager.EXTRA_STATE)

        @Suppress("DEPRECATION")
        val number = intent.extras!!.getString(TelephonyManager.EXTRA_INCOMING_NUMBER)
        if (stateStr != "IDLE" || number == null) {
            Timber.d("event is not a missed call or permission is not granted state = [${stateStr}]")
            return
        }

        // Sleep so that the call gets added into the call log
        Thread.sleep(200)

        val lastCall = getCallLog(context, number)
        if (lastCall == null) {
            Timber.d("The call from [${number}] was not a missed call.")
            return
        }

        handleMissedCallEvent(context, number, lastCall)
    }

    private fun handleMissedCallEvent(context: Context, contact: String, callLog: Pair<ZonedDateTime, String>) {
        val (timestamp, sim) = callLog
        val owner = Settings.getPhoneNumber(context, sim)

        if (!Settings.isLoggedIn(context)) {
            Timber.w("[${sim}] user is not logged in")
            return
        }

        if (!Settings.isIncomingCallEventsEnabled(context, callLog.second)) {
            Timber.w("[${sim}] incoming call events is not enabled")
            return
        }

        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.CONNECTED)
            .build()

        val inputData: Data = workDataOf(
            Constants.KEY_MESSAGE_FROM to contact,
            Constants.KEY_MESSAGE_SIM to sim,
            Constants.KEY_MESSAGE_TO to owner,
            Constants.KEY_MESSAGE_TIMESTAMP to DateTimeFormatter.ofPattern(Constants.TIMESTAMP_PATTERN).format(timestamp).replace("+", "Z")
        )

        val work = OneTimeWorkRequest
            .Builder(MissedCallWorker::class.java)
            .setConstraints(constraints)
            .setInputData(inputData)
            .build()

        WorkManager
            .getInstance(context)
            .enqueue(work)

        Timber.d("work enqueued with ID [${work.id}] for missed phone call from [${contact}] to [${owner}] in  [${sim}]")
    }

    @SuppressLint("MissingPermission")
    private fun getSlotIndexFromSubscriptionId(context: Context, subscriptionId: Int): String {
        val localSubscriptionManager: SubscriptionManager = if (Build.VERSION.SDK_INT < 31) {
            @Suppress("DEPRECATION")
            SubscriptionManager.from(context)
        } else {
            context.getSystemService(SubscriptionManager::class.java)
        }

        var sim = Constants.SIM1
        localSubscriptionManager.activeSubscriptionInfoList.forEach {
            if (it.subscriptionId == subscriptionId) {
               if (it.simSlotIndex > 0){
                   sim = Constants.SIM2
               }
            }
        }
        return sim
    }

    private fun getCallLog(context: Context, phoneNumber: String): Pair<ZonedDateTime, String>? {
        // Specify the columns you want to retrieve from the call log
        val projection = arrayOf(CallLog.Calls.NUMBER, CallLog.Calls.DATE, CallLog.Calls.TYPE, CallLog.Calls.PHONE_ACCOUNT_ID)

        // Query the call log content provider
        val cursor = context.contentResolver.query(
            CallLog.Calls.CONTENT_URI,
            projection,
            null,
            null,
            CallLog.Calls.DATE + " DESC" // Order by date in descending order
        )

        // Check if the cursor is not null and contains at least one entry
        if (cursor != null && cursor.moveToFirst()) {
            val number = cursor.getString(cursor.getColumnIndexOrThrow(CallLog.Calls.NUMBER))
            if (number != phoneNumber) {
                Timber.w("last phone call has phone number [${number}] but the expected phone number was [${phoneNumber}]")
                return null
            }

            if (cursor.getInt(cursor.getColumnIndexOrThrow(CallLog.Calls.TYPE)) != CallLog.Calls.MISSED_TYPE) {
                Timber.w("last phone call from phone number was [${phoneNumber}] was not a missed call")
                return null
            }

            val date = cursor.getLong(cursor.getColumnIndexOrThrow(CallLog.Calls.DATE))
            val sim = getSlotIndexFromSubscriptionId(context, cursor.getInt(cursor.getColumnIndexOrThrow(CallLog.Calls.PHONE_ACCOUNT_ID)))

            // Convert date to a readable format (optional)
            val dateString = java.text.DateFormat.getDateTimeInstance().format(date)
            Timber.d("missed call date is [${dateString}], SIM = [${sim}]")

            // Close the cursor to free up resources
            cursor.close()

            // Construct a string representing the last call
            return Pair(ZonedDateTime.ofInstant(Instant.ofEpochMilli(date), ZoneOffset.UTC), sim)
        }

        // Close the cursor if it's not null even if it doesn't contain any data
        cursor?.close()

        // Return null if no calls are found
        return null
    }

    internal class MissedCallWorker(appContext: Context, workerParams: WorkerParameters) : Worker(appContext, workerParams) {
        override fun doWork(): Result {
            Timber.i("[${this.inputData.getString(Constants.KEY_MESSAGE_SIM)}] forwarding missed call from [${this.inputData.getString(Constants.KEY_MESSAGE_FROM)}] to [${this.inputData.getString(Constants.KEY_MESSAGE_TO)}]")

            if (HttpSmsApiService.create(applicationContext).sendMissedCallEvent(
                    this.inputData.getString(Constants.KEY_MESSAGE_SIM)!!,
                    this.inputData.getString(Constants.KEY_MESSAGE_FROM)!!,
                    this.inputData.getString(Constants.KEY_MESSAGE_TO)!!,
                    this.inputData.getString(Constants.KEY_MESSAGE_TIMESTAMP)!!,
                )) {
                return Result.success()
            }

            return Result.retry()
        }
    }
}
