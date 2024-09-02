package com.httpsms

import android.Manifest
import android.annotation.SuppressLint
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.telephony.PhoneNumberUtils
import android.telephony.TelephonyManager
import android.view.View
import android.webkit.URLUtil
import android.widget.LinearLayout
import android.widget.Toast
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.lifecycle.MutableLiveData
import com.google.android.gms.common.ConnectionResult
import com.google.android.gms.common.GoogleApiAvailability
import com.google.android.material.button.MaterialButton
import com.google.android.material.progressindicator.LinearProgressIndicator
import com.google.android.material.textfield.TextInputEditText
import com.google.android.material.textfield.TextInputLayout
import com.journeyapps.barcodescanner.ScanContract
import com.journeyapps.barcodescanner.ScanOptions
import timber.log.Timber
import java.net.URI


class LoginActivity : AppCompatActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        redirectToMain()
        setContentView(R.layout.activity_login)
        registerListeners()
        setPhoneNumber()
        disableSim2()
        setServerURL()
        setupApiKeyInput()
    }

    private fun setupApiKeyInput() {
        val apiKeyInputLayout = findViewById<TextInputLayout>(R.id.loginApiKeyTextInputLayout)
        val apiKeyInput = findViewById<TextInputEditText>(R.id.loginApiKeyTextInput)

        // 设置点击监听器启动扫描
        apiKeyInput.setOnClickListener {
            startQrCodeScan() // 触发 QR Code 扫描
        }

        // 设置 endIcon 的点击事件监听器
        apiKeyInputLayout.setEndIconOnClickListener {
            Toast.makeText(this, "End icon clicked", Toast.LENGTH_SHORT).show()
            // 在这里处理 endIcon 的点击事件，例如启动相机扫描
            startQrCodeScan()
        }
    }

    private val barcodeLauncher = registerForActivityResult(ScanContract()) { result ->
        if (result.contents != null) {
            val apiKeyInput = findViewById<TextInputEditText>(R.id.loginApiKeyTextInput)
            apiKeyInput.setText(result.contents)
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

    override fun onResume() {
        super.onResume()
        setPhoneNumber()
        disableSim2()
    }

    private fun registerListeners() {
        loginButton().setOnClickListener { onLoginClick() }
    }

    private fun disableSim2() {
        if (SmsManagerService.isDualSIM(this)) {
            Timber.d("dual sim detected")
            val sim2Layout = findViewById<LinearLayout>(R.id.loginPhoneNumberLayoutSIM2)
            sim2Layout.visibility = LinearLayout.VISIBLE
            return
        }
        Timber.d("single sim detected")
        val sim2Layout = findViewById<LinearLayout>(R.id.loginPhoneNumberLayoutSIM2)
        sim2Layout.visibility = View.GONE
    }

    private fun setPhoneNumber() {
        val phoneNumber = getPhoneNumber(this)
        if(phoneNumber == null) {
            Timber.d("cannot get phone due to no permissions")
            return
        }

        val phoneInput = findViewById<TextInputEditText>(R.id.loginPhoneNumberInputSIM1)
        phoneInput.setText(phoneNumber)
        Timber.d("[SIM1] phone number [$phoneNumber] set successfully")
    }

    private fun setServerURL() {
        val serverUrlInput = findViewById<TextInputEditText>(R.id.loginServerUrlInput)
        serverUrlInput.setText(getString(R.string.default_server_url))
        Timber.d("default server url [${serverUrlInput.text.toString()}] set successfully")
    }

    @SuppressLint("HardwareIds")
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


    private fun onLoginClick() {
        Timber.d("login button clicked")

        val error = isGooglePlayServicesAvailable()
        if (error != null) {
            Timber.d("google play services not installed [${error}]")
            Toast.makeText(this, error, Toast.LENGTH_SHORT).show()
            return
        }

        loginButton().isEnabled = false
        val progressBar = findViewById<LinearProgressIndicator>(R.id.loginProgressIndicator)
        progressBar.visibility = View.VISIBLE

        val apiKeyLayout = findViewById<TextInputLayout>(R.id.loginApiKeyTextInputLayout)
        apiKeyLayout.error = null

        val apiKey = findViewById<TextInputEditText>(R.id.loginApiKeyTextInput)
        apiKey.isEnabled = false

        val serverUrlLayout = findViewById<TextInputLayout>(R.id.loginServerUrlLayout)
        serverUrlLayout.error = null

        val serverUrl = findViewById<TextInputEditText>(R.id.loginServerUrlInput)
        serverUrl.isEnabled = false

        val phoneNumberLayout = findViewById<TextInputLayout>(R.id.loginPhoneNumberLayoutSIM1)
        phoneNumberLayout.error = null

        val phoneNumber = findViewById<TextInputEditText>(R.id.loginPhoneNumberInputSIM1)
        phoneNumber.isEnabled = false

        val phoneNumberLayoutSIM2 = findViewById<TextInputLayout>(R.id.loginPhoneNumberLayoutSIM2)
        phoneNumberLayoutSIM2.error = null

        val phoneNumberSIM2 = findViewById<TextInputEditText>(R.id.loginPhoneNumberInputSIM2)
        phoneNumberSIM2.isEnabled = false

        val resetView = fun () {
            apiKey.isEnabled = true
            serverUrl.isEnabled = true
            progressBar.visibility = View.INVISIBLE
            phoneNumber.isEnabled = true
            phoneNumberSIM2.isEnabled = true
            loginButton().isEnabled = true
        }

        if (
            !PhoneNumberUtils.isWellFormedSmsAddress(phoneNumber.text.toString().trim()) ||
            !PhoneNumberUtils.isGlobalPhoneNumber(phoneNumber.text.toString().trim())
        ) {
            Timber.e("[SIM1] phone number [${phoneNumber.text.toString()}] is not valid")
            resetView()
            phoneNumberLayout.error = "Invalid E.164 phone number"
            return
        }

        if (
            SmsManagerService.isDualSIM(this) && (
                    !PhoneNumberUtils.isWellFormedSmsAddress(phoneNumberSIM2.text.toString().trim()) ||
                    !PhoneNumberUtils.isGlobalPhoneNumber(phoneNumberSIM2.text.toString().trim())
            )
        ) {
            Timber.e("[SIM2] phone number [${phoneNumberSIM2.text.toString()}] is not valid")
            resetView()
            phoneNumberLayoutSIM2.error = "Invalid E.164 phone number"
            return
        }

        if(!URLUtil.isValidUrl(serverUrl.text.toString().trim())) {
            Timber.e("url number [${serverUrl.text.toString()}] is not a valid URL")
            resetView()
            serverUrlLayout.error = "Server URL [${serverUrl.text.toString()}] is invalid"
            return
        }

        if (!URLUtil.isHttpsUrl(serverUrl.text.toString().trim())) {
            Timber.e("url number [${serverUrl.text.toString()}] is not an https URL")
            resetView()
            serverUrlLayout.error = "Server URL [${serverUrl.text.toString()}] must be HTTPS"
            return
        }

        val liveData = MutableLiveData<Pair<String?, String?>>()
        liveData.observe(this) { authResult ->
            run {
                progressBar.visibility = View.INVISIBLE
                if (authResult.first != null) {
                    resetView()
                    apiKeyLayout.error = authResult.first
                    return@run
                }

                if (authResult.second != null) {
                    resetView()
                    serverUrlLayout.error = authResult.second
                    return@run
                }

                Settings.setApiKeyAsync(this, apiKey.text.toString())
                Settings.setServerUrlAsync(this, serverUrl.text.toString().trim())

                val e164PhoneNumber = formatE164(phoneNumber.text.toString().trim())
                Settings.setSIM1PhoneNumber(this, e164PhoneNumber)

                if(SmsManagerService.isDualSIM(this)) {
                    val sim2PhoneNumber = formatE164(phoneNumberSIM2.text.toString().trim())
                    Settings.setSIM2PhoneNumber(this, sim2PhoneNumber)
                }

                Timber.d("login successfully redirecting to main view")
                redirectToMain()
            }
        }

        Thread {
            val error = HttpSmsApiService(apiKey.text.toString(), URI(serverUrl.text.toString().trim())).validateApiKey()
            liveData.postValue(error)
            Timber.d("finished validating api URL")
        }.start()
    }

    private fun formatE164(number: String): String {
        var phoneNumber = number.trim()
        if (!number.startsWith("+")) {
            phoneNumber = "+$number"
        }

        Timber.d("formatting phone number [${phoneNumber}] into e164")

        val formattedNumber = PhoneNumberUtils.formatNumberToE164(
            phoneNumber,
            this.resources.configuration.locales.get(0).country
        )

        if (formattedNumber !== null) {
            return formattedNumber
        }

        return phoneNumber;
    }

    private fun addPlus(number: String): String {
        if (number.startsWith("+")) {
            return number
        }
        return "+$number"
    }

    private fun redirectToMain() {
        if (!Settings.isLoggedIn(this)) {
            return
        }
       finish()

        val switchActivityIntent = Intent(this, MainActivity::class.java)
        startActivity(switchActivityIntent)
    }

    private fun loginButton(): MaterialButton {
        return findViewById(R.id.loginButton)
    }
}
