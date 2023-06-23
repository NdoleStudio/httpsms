package com.httpsms

import android.Manifest
import android.annotation.SuppressLint
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Bundle
import android.telephony.PhoneNumberUtils
import android.telephony.TelephonyManager
import android.view.View
import android.webkit.URLUtil
import android.widget.LinearLayout
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.lifecycle.MutableLiveData
import com.google.android.material.button.MaterialButton
import com.google.android.material.progressindicator.LinearProgressIndicator
import com.google.android.material.textfield.TextInputEditText
import com.google.android.material.textfield.TextInputLayout
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
    }

    private fun registerListeners() {
        loginButton().setOnClickListener { onLoginClick() }
    }

    private fun disableSim2() {
        if (SmsManagerService.isDualSIM(this)) {
            Timber.d("dual sim detected")
            return
        }
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
        Timber.d("phone number [$phoneNumber] set successfully")
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
            ) != PackageManager.PERMISSION_GRANTED || ActivityCompat.checkSelfPermission(
                this,
                Manifest.permission.READ_PHONE_NUMBERS
            ) != PackageManager.PERMISSION_GRANTED || ActivityCompat.checkSelfPermission(
                this,
                Manifest.permission.READ_PHONE_STATE
            ) != PackageManager.PERMISSION_GRANTED
        ) {
            Timber.e("cannot get owner because permissions are not granted")
            return Settings.getOwner(this)
        }

        if (telephonyManager.line1Number != null && telephonyManager.line1Number  != "") {
            Settings.setOwnerAsync(context, telephonyManager.line1Number)
        }

        return telephonyManager.line1Number
    }


    private fun onLoginClick() {
        Timber.d("login button clicked")
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
            !PhoneNumberUtils.isWellFormedSmsAddress(phoneNumber.text.toString()) ||
            !PhoneNumberUtils.isGlobalPhoneNumber(phoneNumber.text.toString())
        ) {
            Timber.e("phone number [${phoneNumber.text.toString()}] is not valid")
            resetView()
            phoneNumberLayout.error = "Invalid E.164 phone number"
            return
        }

        if (
            SmsManagerService.isDualSIM(this) && (
                    !PhoneNumberUtils.isWellFormedSmsAddress(phoneNumberSIM2.text.toString()) ||
                    !PhoneNumberUtils.isGlobalPhoneNumber(phoneNumberSIM2.text.toString())
            )
        ) {
            Timber.e("phone number [${phoneNumberSIM2.text.toString()}] is not valid")
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

                val e164PhoneNumber = PhoneNumberUtils.formatNumberToE164(
                    phoneNumber.text.toString(),
                    this.resources.configuration.locales.get(0).country
                )
                Settings.setSIM1PhoneNumber(this, e164PhoneNumber)

                if(SmsManagerService.isDualSIM(this)) {
                    val sim2PhoneNumber = PhoneNumberUtils.formatNumberToE164(
                        phoneNumberSIM2.text.toString(),
                        this.resources.configuration.locales.get(0).country
                    )
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
