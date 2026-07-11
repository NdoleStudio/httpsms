package com.httpsms

import android.Manifest
import android.annotation.SuppressLint
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.provider.Settings as ProviderSettings
import android.widget.Toast
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import androidx.activity.viewModels
import androidx.appcompat.app.AppCompatActivity
import androidx.work.Constraints
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.NetworkType
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import com.httpsms.services.StickyNotificationService
import com.httpsms.ui.main.MainScreen
import com.httpsms.ui.main.MainViewModel
import com.httpsms.ui.theme.HttpSmsTheme
import com.httpsms.worker.HeartbeatWorker
import timber.log.Timber
import java.util.concurrent.TimeUnit


class MainActivity : AppCompatActivity() {
    private val viewModel: MainViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        initTimber()

        redirectToLogin()

        viewModel.initialize(this, getString(R.string.app_version, BuildConfig.VERSION_NAME))

        setContent {
            HttpSmsTheme {
                MainScreen(
                    viewModel = viewModel,
                    onSettingsClick = { onSettingsClick() },
                    onSmsPermissionClick = {
                        val intent = Intent(Intent.ACTION_VIEW, Uri.parse("https://httpsms.com/blog/grant-send-and-read-sms-permissions-on-android"))
                        startActivity(intent)
                    },
                    onBatteryOptimizationClick = {
                        val intent = Intent()
                        intent.action = ProviderSettings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS
                        intent.data = Uri.parse("package:$packageName")
                        startActivity(intent)
                    },
                    onHeartbeatClick = {
                        viewModel.sendHeartbeat(this) { error ->
                            if (error != null) {
                                Timber.w("heartbeat sending failed with [$error]")
                                Toast.makeText(this, error, Toast.LENGTH_LONG).show()
                            } else {
                                Toast.makeText(this, "Heartbeat sent successfully", Toast.LENGTH_SHORT).show()
                            }
                        }
                    }
                )
            }
        }

        createChannel()
        refreshToken(this)

        startStickyNotification(this)
        scheduleHeartbeatWorker(this)
    }

    override fun onStart() {
        super.onStart()
        requestPermissions(this)
    }

    override fun onResume() {
        super.onResume()
        Timber.d( "on activity resume")
        redirectToLogin()
        refreshToken(this)
        viewModel.updateState(this, getString(R.string.app_version, BuildConfig.VERSION_NAME))
    }

    private fun requestPermissions(context:Context) {
        Timber.d("requesting permissions")
        val requestPermissionLauncher = registerForActivityResult(ActivityResultContracts.RequestMultiplePermissions()) { permissions ->
            permissions.entries.forEach {
                Timber.d("${it.key} = ${it.value}")
                if (it.key == Manifest.permission.READ_CALL_LOG && !it.value) {
                    Timber.w("disabling incoming call events since for SIM1 and SIM2")
                    Settings.setIncomingCallEventsEnabled(context, Constants.SIM1, false)
                    Settings.setIncomingCallEventsEnabled(context, Constants.SIM2, false)
                }
            }
            viewModel.updateState(context, getString(R.string.app_version, BuildConfig.VERSION_NAME))
        }

        var permissions = arrayOf(
            Manifest.permission.SEND_SMS,
            Manifest.permission.RECEIVE_SMS,
            Manifest.permission.READ_SMS
        )

        if(Build.VERSION.SDK_INT >= 33) {
            permissions += Manifest.permission.POST_NOTIFICATIONS
        }

        if(Settings.isIncomingCallEventsEnabled(context,Constants.SIM1) || Settings.isIncomingCallEventsEnabled(context,Constants.SIM2) ) {
            permissions += Manifest.permission.READ_CALL_LOG
            permissions += Manifest.permission.READ_PHONE_STATE
        }

        requestPermissionLauncher.launch(permissions)

        Timber.d("creating permissions launcher")
    }

    private fun scheduleHeartbeatWorker(context: Context) {
        val tag = "TAG_HEARTBEAT_WORKER"

        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.CONNECTED)
            .build()

        val heartbeatWorker =
            PeriodicWorkRequestBuilder<HeartbeatWorker>(15, TimeUnit.MINUTES)
                .setConstraints(constraints)
                .addTag(tag)
                .build()

        WorkManager
            .getInstance(context)
            .enqueueUniquePeriodicWork(tag, ExistingPeriodicWorkPolicy.KEEP, heartbeatWorker)

        Timber.d("finished scheduling heartbeat worker with ID [${heartbeatWorker.id}]")
    }

    private fun startStickyNotification(context: Context) {
        Timber.d("starting foreground service")
        if(!Settings.getActiveStatus(context, Constants.SIM1) && !Settings.getActiveStatus(context, Constants.SIM2)) {
            Timber.d("active status is false, not starting foreground service")
            return
        }
        val notificationIntent = Intent(context, StickyNotificationService::class.java)
        val service = context.startForegroundService(notificationIntent)
        Timber.d("foreground service started [${service?.className}]")
    }

    private fun refreshToken(context: Context) {
        if(!Settings.isLoggedIn(context)) {
            Timber.w("cannot refresh token because owner is not logged in")
            return
        }

        if(!Settings.hasOwner(context)) {
            Timber.w("cannot refresh token because owner does not exist")
            return
        }

        if (Settings.getFcmToken(context) == null) {
            Timber.w("cannot refresh token because token does not exist")
            return
        }

        val updateTimestamp = Settings.getFcmTokenLastUpdateTimestamp(context)
        Timber.d("FCM_TOKEN_UPDATE_TIMESTAMP: $updateTimestamp")

        val interval = 24 * 60 * 60 * 1000 // 1 day
        val currentTimeStamp = System.currentTimeMillis()

        if (currentTimeStamp - updateTimestamp < interval) {
            Timber.i("update interval [${currentTimeStamp - updateTimestamp}] < 24 hours [$interval]")
            return
        }

        sendFCMToken(currentTimeStamp, context, Settings.getSIM1PhoneNumber(context), Constants.SIM1)
        if (Settings.isDualSIM(context)) {
            sendFCMToken(currentTimeStamp, context, Settings.getSIM2PhoneNumber(context), Constants.SIM2)
        }
    }

    private fun sendFCMToken(timestamp: Long, context:Context, phoneNumber: String, sim: String) {
        Thread {
            val response = HttpSmsApiService.create(context).updateFcmToken(phoneNumber, sim,Settings.getFcmToken(context) ?: "")
            if (response.first != null) {
                Settings.setUserID(context, response.first!!.userID)
                Settings.setFcmTokenLastUpdateTimestampAsync(context, timestamp)
                Timber.i("[${sim}] FCM token uploaded successfully")
                return@Thread
            } else {
                Timber.e("[${sim}] could not update FCM token")
            }
        }.start()
    }

    private fun initTimber() {
        if (Timber.treeCount > 1) {
            Timber.d("timber is already initialized with count [${Timber.treeCount}]")
            return
        }

        if(Settings.isDebugLogEnabled(this)) {
            Timber.plant(Timber.DebugTree())
            Timber.plant(LogzTree(this.applicationContext))
        }
    }

    private fun onSettingsClick() {
        Timber.d("settings button clicked")
        val switchActivityIntent = Intent(this, SettingsActivity::class.java)
        startActivity(switchActivityIntent)
    }

    private fun redirectToLogin():Boolean {
        if (Settings.isLoggedIn(this)) {
            return false
        }
        val switchActivityIntent = Intent(this, LoginActivity::class.java)
        startActivity(switchActivityIntent)
        return true
    }

    private fun createChannel() {
        // Create the NotificationChannel
        val name = getString(R.string.notification_channel_default)
        val descriptionText = getString(R.string.notification_channel_default)
        val importance = NotificationManager.IMPORTANCE_DEFAULT
        val mChannel = NotificationChannel(name, name, importance)
        mChannel.description = descriptionText
        // Register the channel with the system; you can't change the importance
        // or other notification behaviors after this
        val notificationManager = getSystemService(NOTIFICATION_SERVICE) as NotificationManager
        notificationManager.createNotificationChannel(mChannel)
    }
}
