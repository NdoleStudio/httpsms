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

        sendSIM1Heartbeat()
        if (Settings.isDualSIM(applicationContext)) {
            sendSIM2Heartbeat()
        }

        return Result.success()
    }

    private fun sendSIM1Heartbeat() {
        if (!Settings.getActiveStatus(applicationContext, Constants.SIM1)) {
            Timber.w("[SIM1] user is not active, stopping processing")
            return
        }

        HttpSmsApiService.create(applicationContext).storeHeartbeat(Settings.getSIM1PhoneNumber(applicationContext))
        Timber.d("[SIM1] finished sending heartbeat to server")

        Settings.setHeartbeatTimestampAsync(applicationContext, System.currentTimeMillis())
        Timber.d("[SIM1] set the heartbeat timestamp")
    }

    private fun sendSIM2Heartbeat() {
        if (!Settings.getActiveStatus(applicationContext, Constants.SIM2)) {
            Timber.w("[SIM2] user is not active, stopping processing")
            return
        }

        HttpSmsApiService.create(applicationContext).storeHeartbeat(Settings.getSIM2PhoneNumber(applicationContext))
        Timber.d("[SIM2] finished sending heartbeat to server")

        Settings.setHeartbeatTimestampAsync(applicationContext, System.currentTimeMillis())
        Timber.d("[SIM2] set the heartbeat timestamp")
    }
}
