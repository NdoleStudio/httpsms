package com.httpsms

import android.Manifest
import android.Manifest.permission.READ_PHONE_NUMBERS
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.telephony.PhoneNumberUtils
import android.view.View
import android.widget.TextView
import android.widget.Toast
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.lifecycle.MutableLiveData
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import com.google.android.material.button.MaterialButton
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.google.android.material.progressindicator.LinearProgressIndicator
import com.google.android.material.switchmaterial.SwitchMaterial
import com.httpsms.services.StickyNotificationService
import com.httpsms.worker.HeartbeatWorker
import okhttp3.internal.format
import okhttp3.internal.notify
import timber.log.Timber
import java.time.Instant
import java.time.ZoneId
import java.time.ZoneOffset
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter
import java.util.*
import java.util.concurrent.TimeUnit


class MainActivity : AppCompatActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        initTimber()

        redirectToLogin()

        setContentView(R.layout.activity_main)

        createChannel()

        requestPermissions(this)

        setOwner(getPhoneNumber(this))
        setActiveStatus(this)
        registerListeners()
        refreshToken(this)

        startStickyNotification(this)
        scheduleHeartbeatWorker(this)
        setLastHeartbeatTimestamp(this)
        setVersion()
        setHeartbeatListener(this)
    }

    override fun onResume() {
        super.onResume()
        Timber.d( "on activity resume")
        redirectToLogin()
        refreshToken(this)
        setLastHeartbeatTimestamp(this)
    }

    private fun setLastHeartbeatTimestamp(context: Context) {
        val refreshTimestampView = findViewById<TextView>(R.id.cardRefreshTime)
        val timestamp = Settings.getHeartbeatTimestamp(context)

        if (timestamp == 0.toLong()) {
            Timber.d("not heartbeat timestamp has been set")
            refreshTimestampView.text = "--"
            return
        }

        val timestampZdt = ZonedDateTime.ofInstant(Instant.ofEpochMilli(timestamp), ZoneOffset.UTC)
        val localTime = timestampZdt.withZoneSameInstant(ZoneId.systemDefault())
        Timber.d("heartbeat timestamp in UTC is [${timestampZdt}] and local is [$localTime]")

        refreshTimestampView.text = localTime.format(DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss"))
    }

    private fun setVersion() {
        val appVersionView = findViewById<TextView>(R.id.mainAppVersion)
        appVersionView.text = format(getString(R.string.app_version), BuildConfig.VERSION_NAME)
    }

    private fun scheduleHeartbeatWorker(context: Context) {
        val tag = "TAG_HEARTBEAT_WORKER"

        val heartbeatWorker =
            PeriodicWorkRequestBuilder<HeartbeatWorker>(15, TimeUnit.MINUTES)
                .addTag(tag)
                .build()

        WorkManager
            .getInstance(context)
            .enqueueUniquePeriodicWork(tag, ExistingPeriodicWorkPolicy.KEEP, heartbeatWorker)

        Timber.d("finished scheduling heartbeat worker with ID [${heartbeatWorker.id}]")
    }

    private fun startStickyNotification(context: Context) {
        Timber.d("starting foreground service")
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

        Thread {
            val updated = HttpSmsApiService.create(context).updatePhone(Settings.getOwnerOrDefault(context), Settings.getFcmToken(context) ?: "")
            if (updated) {
                Settings.setFcmTokenLastUpdateTimestampAsync(context, currentTimeStamp)
                Timber.i("fcm token uploaded successfully")
                return@Thread
            }
            Timber.e("could not update fcm token")
        }.start()
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

    private fun registerListeners() {
        findViewById<MaterialButton>(R.id.mainLogoutButton).setOnClickListener { onLogoutClick() }
    }

    private fun onLogoutClick() {
        Timber.d("logout button clicked")
        MaterialAlertDialogBuilder(this)
            .setTitle("Confirm")
            .setMessage("Are you sure you want to logout of the Http SMS App?")
            .setNeutralButton("Cancel"){ _, _ -> Timber.d("logout dialog canceled") }
            .setPositiveButton("Logout"){_, _ ->
                Timber.d("logging out user")
                Settings.setApiKeyAsync(this, null)
                Settings.setOwnerAsync(this, null)
                Settings.setFcmTokenLastUpdateTimestampAsync(this, 0)
                redirectToLogin()
            }
            .show()
    }

    private fun redirectToLogin():Boolean {
        if (Settings.isLoggedIn(this)) {
            return false
        }
        val switchActivityIntent = Intent(this, LoginActivity::class.java)
        startActivity(switchActivityIntent)
        return true
    }

    private fun setActiveStatus(context: Context) {
        val switch = findViewById<SwitchMaterial>(R.id.cardSwitch)
        switch.isChecked = Settings.getActiveStatus(context)
        switch.setOnCheckedChangeListener{
            _, isChecked ->
            run {
                if (isChecked && !hasAllPermissions(context)) {
                    Toast.makeText(context, "PERMISSIONS_NOT_GRANTED", Toast.LENGTH_SHORT).show()
                } else {
                    Settings.setActiveStatusAsync(context, isChecked)
                }
            }
        }
    }

    private fun hasAllPermissions(context: Context): Boolean {
        if (ActivityCompat.checkSelfPermission(
                context,
                Manifest.permission.SEND_SMS
            ) == PackageManager.PERMISSION_GRANTED && ActivityCompat.checkSelfPermission(
                context,
                READ_PHONE_NUMBERS
            ) == PackageManager.PERMISSION_GRANTED && ActivityCompat.checkSelfPermission(
                context,
                Manifest.permission.RECEIVE_SMS
            ) == PackageManager.PERMISSION_GRANTED && ActivityCompat.checkSelfPermission(
                context,
                Manifest.permission.READ_PHONE_STATE
            ) == PackageManager.PERMISSION_GRANTED && (Build.VERSION.SDK_INT < 33 || ActivityCompat.checkSelfPermission(
                context,
                Manifest.permission.POST_NOTIFICATIONS
            ) == PackageManager.PERMISSION_GRANTED)
        ) {
            return true
        }
        return false
    }

    private fun setOwner(phoneNumber: String) {
        val titleText = findViewById<TextView>(R.id.cardPhoneNumber)
        titleText.text = PhoneNumberUtils.formatNumber(phoneNumber, Locale.getDefault().country)
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

    private fun getPhoneNumber(context: Context): String {
        return Settings.getOwnerOrDefault(context)
    }

    private fun requestPermissions(context:Context) {
        if(!Settings.isLoggedIn(context)) {
            return
        }

        Timber.d("requesting permissions")
        val requestPermissionLauncher = registerForActivityResult(ActivityResultContracts.RequestMultiplePermissions()) { permissions ->
            permissions.entries.forEach {
                Timber.d("${it.key} = ${it.value}")
                setOwner(getPhoneNumber(context))
            }
        }

        var permissions = arrayOf(
            Manifest.permission.SEND_SMS,
            Manifest.permission.RECEIVE_SMS,
            READ_PHONE_NUMBERS,
            Manifest.permission.READ_SMS,
            Manifest.permission.READ_PHONE_STATE
        )

        if(Build.VERSION.SDK_INT >= 33) {
            permissions += Manifest.permission.POST_NOTIFICATIONS
        }

        requestPermissionLauncher.launch(permissions)

        Timber.d("creating permissions launcher")
    }

    private fun setHeartbeatListener(context: Context) {
        findViewById<MaterialButton>(R.id.mainHeartbeatButton).setOnClickListener{onHeartbeatClick(context)}
    }

    private fun onHeartbeatClick(context: Context) {
        Timber.d("heartbeat button clicked")
        val heartbeatButton = findViewById<MaterialButton>(R.id.mainHeartbeatButton)
        heartbeatButton.isEnabled = false

        val progressBar = findViewById<LinearProgressIndicator>(R.id.mainProgressIndicator)
        progressBar.visibility = View.VISIBLE


        val liveData = MutableLiveData<String?>()
        liveData.observe(this) { exception ->
            run {
                progressBar.visibility = View.INVISIBLE
                heartbeatButton.isEnabled = true

                if (exception != null) {
                    Timber.w("heartbeat sending failed with [$exception]")
                    Toast.makeText(context, exception, Toast.LENGTH_SHORT).show()
                    return@run
                }
                Toast.makeText(context, "Heartbeat Sent", Toast.LENGTH_SHORT).show()

                setLastHeartbeatTimestamp(this)
            }
        }

        Thread {
            var error: String? = null
            try {
                HttpSmsApiService.create(context).storeHeartbeat(Settings.getOwnerOrDefault(context))
                Settings.setHeartbeatTimestampAsync(applicationContext, System.currentTimeMillis())
            } catch (exception: Exception) {
                Timber.e(exception)
                error = exception.javaClass.simpleName
            }
            liveData.postValue(error)
            Timber.d("finished sending pulse")
        }.start()
    }
}
