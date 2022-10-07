package com.httpsms.worker

import android.content.Context
import androidx.work.Worker
import androidx.work.WorkerParameters
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

        if (!Settings.getActiveStatus(applicationContext)) {
            Timber.w("user is not active, stopping processing")
            return Result.failure()
        }

        HttpSmsApiService.create(applicationContext)
            .storeHeartbeat(Settings.getOwnerOrDefault(applicationContext))
        Timber.d("finished sending heartbeat to server")

        Settings.setHeartbeatTimestampAsync(applicationContext, System.currentTimeMillis())
        Timber.d("set the heartbeat timestamp")

        return Result.success()
    }




}
