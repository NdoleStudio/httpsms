package com.httpsms

import android.Manifest
import android.Manifest.permission.READ_PHONE_NUMBERS
import android.annotation.SuppressLint
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.icu.lang.UCharacter.GraphemeClusterBreak.T
import android.nfc.Tag
import android.os.Bundle
import android.telephony.PhoneNumberUtils
import android.telephony.TelephonyManager
import android.util.Log
import android.view.Menu
import android.view.MenuItem
import android.widget.TextView
import android.widget.Toast
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import java.util.*


class MainActivity : AppCompatActivity() {
    companion object {
        private val TAG = MyFirebaseMessagingService::class.simpleName
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        val intent = Intent("testing")
        intent.putExtra(Constants.KEY_MESSAGE_ID, "123")

        Log.w(TAG, "message id = [${intent.getStringExtra(Constants.KEY_MESSAGE_ID)}]")

        val phoneNumber = getPhoneNumber(this)

        val titleText = findViewById<TextView>(R.id.cardPhoneNumber)
        titleText.text = PhoneNumberUtils.formatNumber(phoneNumber, Locale.getDefault().country)

        requestPermission(this, Manifest.permission.SEND_SMS)
        requestPermission(this, Manifest.permission.RECEIVE_SMS)
        requestPermission(this, Manifest.permission.READ_PHONE_NUMBERS)
        requestPermission(this, Manifest.permission.READ_PHONE_STATE)
        requestPermission(this, Manifest.permission.RECEIVE_SMS)

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
            return "NO_PHONE_NUMBER";
        }
        return telephonyManager.line1Number  ?: "NO_PHONE_NUMBER"
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
        if (ActivityCompat.checkSelfPermission(context, permission) != PackageManager.PERMISSION_GRANTED) {
            // You can directly ask for the permission.
            // The registered ActivityResultCallback gets the result of this request.
            requestPermissionLauncher.launch(permission)
        }
    }
}
