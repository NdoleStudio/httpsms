package com.httpsms

import android.Manifest
import android.annotation.SuppressLint
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.telephony.TelephonyManager
import android.widget.Toast
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import androidx.activity.viewModels
import androidx.appcompat.app.AppCompatActivity
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.core.app.ActivityCompat
import com.google.android.gms.common.ConnectionResult
import com.google.android.gms.common.GoogleApiAvailability
import com.httpsms.ui.login.LoginScreen
import com.httpsms.ui.login.LoginViewModel
import com.httpsms.ui.theme.HttpSmsTheme
import com.journeyapps.barcodescanner.ScanContract
import com.journeyapps.barcodescanner.ScanOptions
import timber.log.Timber

class LoginActivity : AppCompatActivity() {
    private val viewModel: LoginViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        redirectToMain()
        
        viewModel.initialize(this, getString(R.string.default_server_url))

        setContent {
            HttpSmsTheme {
                val uiState by viewModel.uiState.collectAsState()
                
                LaunchedEffect(uiState.loginSuccess) {
                    if (uiState.loginSuccess) {
                        redirectToMain()
                    }
                }

                LoginScreen(
                    viewModel = viewModel,
                    onQrScanClick = { startQrCodeScan() },
                    onLoginClick = {
                        val error = isGooglePlayServicesAvailable()
                        if (error != null) {
                            Toast.makeText(this@LoginActivity, error, Toast.LENGTH_SHORT).show()
                        } else {
                            viewModel.login(
                                context = this@LoginActivity,
                                countryCode = getCountryCode(),
                                onGooglePlayServicesError = {
                                    Toast.makeText(this@LoginActivity, it, Toast.LENGTH_SHORT).show()
                                },
                                onFcmTokenMissing = {
                                    Toast.makeText(
                                        this@LoginActivity,
                                        "Cannot find FCM token. Make sure you have Google Play Services installed",
                                        Toast.LENGTH_LONG
                                    ).show()
                                }
                            )
                        }
                    }
                )
            }
        }
    }

    private val barcodeLauncher = registerForActivityResult(ScanContract()) { result ->
        if (result.contents != null) {
            viewModel.onApiKeyChange(result.contents)
            Toast.makeText(this, "Scanned: ${result.contents}", Toast.LENGTH_LONG).show()
        } else {
            Toast.makeText(this, "Scan cancelled", Toast.LENGTH_SHORT).show()
        }
    }

    private fun startQrCodeScan() {
        val options = ScanOptions()
        options.setPrompt("Scan a QR code")
        options.setBeepEnabled(true)
        options.setOrientationLocked(false)
        options.setCameraId(0)
        barcodeLauncher.launch(options)
    }

    override fun onStart() {
        super.onStart()
        Timber.i("on start")
        requestPermissions()
    }

    @SuppressLint("HardwareIds")
    @Suppress("DEPRECATION")
    private fun getPhoneNumber(context: Context): String? {
        val telephonyManager = this.getSystemService(Context.TELEPHONY_SERVICE) as TelephonyManager
        if (ActivityCompat.checkSelfPermission(
                this,
                Manifest.permission.READ_SMS
            ) != PackageManager.PERMISSION_GRANTED
        ) {
            Timber.e("cannot get owner because permissions are not granted")
            return Settings.getSIM1PhoneNumber(this)
        }

        if (telephonyManager.line1Number != null && telephonyManager.line1Number  != "") {
            Settings.setSIM1PhoneNumber(context, telephonyManager.line1Number)
        }

        return telephonyManager.line1Number
    }

    private fun requestPermissions() {
        Timber.d("requesting permissions")
        val requestPermissionLauncher = registerForActivityResult(ActivityResultContracts.RequestMultiplePermissions()) { permissions ->
            permissions.entries.forEach {
                Timber.d("${it.key} = ${it.value}")
            }
        }

        var permissions = arrayOf(
            Manifest.permission.SEND_SMS,
            Manifest.permission.RECEIVE_SMS,
            Manifest.permission.READ_PHONE_STATE,
            Manifest.permission.READ_SMS,
        )

        if(Build.VERSION.SDK_INT >= 33) {
            permissions += Manifest.permission.POST_NOTIFICATIONS
        }

        requestPermissionLauncher.launch(permissions)

        Timber.d("creating permissions launcher")
    }

    private fun isGooglePlayServicesAvailable(): String? {
        val googleApiAvailability = GoogleApiAvailability.getInstance()
        val status = googleApiAvailability.isGooglePlayServicesAvailable(this)
        if (status != ConnectionResult.SUCCESS) {
            if (googleApiAvailability.isUserResolvableError(status)) {
                googleApiAvailability.getErrorDialog(this, status, 2404)?.show()
            }
            return googleApiAvailability.getErrorString(status)
        }
        return null
    }

    private fun redirectToMain() {
        if (!Settings.isLoggedIn(this)) {
            return
        }
        finish()

        val switchActivityIntent = Intent(this, MainActivity::class.java)
        startActivity(switchActivityIntent)
    }

    private fun getCountryCode() : String {
        val tm = this.getSystemService(Context.TELEPHONY_SERVICE) as TelephonyManager
        val code = tm.networkCountryIso.uppercase()
        if (code.isEmpty()) {
            return this.resources.configuration.locales.get(0).country.uppercase()
        }
        return code
    }
}
