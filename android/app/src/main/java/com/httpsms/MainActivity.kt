package com.httpsms

import android.Manifest
import android.Manifest.permission.READ_PHONE_NUMBERS
import android.annotation.SuppressLint
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Bundle
import android.telephony.PhoneNumberUtils
import android.telephony.TelephonyManager
import android.widget.TextView
import android.widget.Toast
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import com.google.android.material.switchmaterial.SwitchMaterial
import timber.log.Timber
import java.util.*


class MainActivity : AppCompatActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        initTimber()

        redirectToLogin()

        setContentView(R.layout.activity_main)

        createChannel()

        requestPermission(this, Manifest.permission.SEND_SMS)
        requestPermission(this, Manifest.permission.RECEIVE_SMS)
        requestPermission(this, READ_PHONE_NUMBERS)
        requestPermission(this, Manifest.permission.READ_PHONE_STATE)
        requestPermission(this, Manifest.permission.RECEIVE_SMS)

        setOwner(getPhoneNumber(this))
        setActiveStatus(this)
    }

    override fun onResume() {
        super.onResume()
        Timber.d( "on activity resume")
        redirectToLogin()
    }

    private fun initTimber() {
        if (BuildConfig.DEBUG) {
            Timber.plant(Timber.DebugTree())
        }
    }

    private fun redirectToLogin() {
        if (Settings.isLoggedIn(this)) {
            return
        }
        val switchActivityIntent = Intent(this, LoginActivity::class.java)
        startActivity(switchActivityIntent)
    }

    private fun setActiveStatus(context: Context) {
        val switch = findViewById<SwitchMaterial>(R.id.cardSwitch)
        switch.isChecked = Settings.getActiveStatus(context)
        switch.setOnCheckedChangeListener{
            _, isChecked -> Settings.setActiveStatusAsync(context, isChecked)
        }
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

    @SuppressLint("HardwareIds")
    private fun getPhoneNumber(context: Context): String {
        val telephonyManager = this.getSystemService(Context.TELEPHONY_SERVICE) as TelephonyManager
        if (ActivityCompat.checkSelfPermission(
                this,
                Manifest.permission.READ_SMS
            ) != PackageManager.PERMISSION_GRANTED && ActivityCompat.checkSelfPermission(
                this,
                READ_PHONE_NUMBERS
            ) != PackageManager.PERMISSION_GRANTED && ActivityCompat.checkSelfPermission(
                this,
                Manifest.permission.READ_PHONE_STATE
            ) != PackageManager.PERMISSION_GRANTED
        ) {
            return "NO_PHONE_NUMBER"
        }

        if (telephonyManager.line1Number != null) {
            Settings.setOwnerAsync(context, telephonyManager.line1Number)
        }

        return telephonyManager.line1Number ?: "NO_PHONE_NUMBER"
    }

    private fun requestPermission(context: Context, permission: String) {
        // Register the permissions callback, which handles the user's response to the
        // system permissions dialog. Save the return value, an instance of
        // ActivityResultLauncher. You can use either a val, as shown in this snippet,
        // or a late init var in your onAttach() or onCreate() method.
        val requestPermissionLauncher =
            registerForActivityResult(
                ActivityResultContracts.RequestPermission()
            ) { isGranted: Boolean ->
                if (isGranted) {
                    val toast = Toast.makeText(context, "Granted", Toast.LENGTH_SHORT)
                    toast.show()
                } else {
                    val toast = Toast.makeText(context, "NOT Granted", Toast.LENGTH_LONG)
                    toast.show()
                }
            }
        if (ActivityCompat.checkSelfPermission(
                context,
                permission
            ) != PackageManager.PERMISSION_GRANTED
        ) {
            // You can directly ask for the permission.
            // The registered ActivityResultCallback gets the result of this request.
            requestPermissionLauncher.launch(permission)
        }
    }
}