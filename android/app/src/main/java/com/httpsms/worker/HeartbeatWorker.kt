package com.httpsms.worker

import android.content.Context
import androidx.work.Worker
import androidx.work.WorkerParameters
import com.httpsms.Constants
import com.httpsms.HttpSmsApiService
import com.httpsms.Settings
import timber.log.Timber

class HeartbeatWorker(appContext: Context, workerParams: WorkerParameters) : Worker(appContext, workerParams) {
    override fun doWork(): Result {
        Timber.d("executing heartbeat worker")
        if (!Settings.isLoggedIn(applicationContext)) {
            Timber.w("user is not logged in, stopping processing")
            return Result.failure()
        }

        val phoneNumbers = mutableListOf<String>()
        if (Settings.getActiveStatus(applicationContext, Constants.SIM1)) {
            phoneNumbers.add(Settings.getSIM1PhoneNumber(applicationContext))
        }
        if (Settings.getActiveStatus(applicationContext, Constants.SIM2)) {
            phoneNumbers.add(Settings.getSIM2PhoneNumber(applicationContext))
        }

        if (phoneNumbers.isEmpty()) {
            Timber.w("both [SIM1] and [SIM2] are inactive stopping processing.")
            return Result.success()
        }

        HttpSmsApiService.create(applicationContext).storeHeartbeat(phoneNumbers.toTypedArray(), Settings.isCharging(applicationContext))
        Timber.d("finished sending heartbeats to server")

        Settings.setHeartbeatTimestampAsync(applicationContext, System.currentTimeMillis())
        Timber.d("Set the heartbeat timestamp")

        return Result.success()
    }
}
