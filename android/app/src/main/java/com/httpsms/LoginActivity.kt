package com.httpsms

import android.Manifest
import android.annotation.SuppressLint
import android.content.Context
import android.content.pm.PackageManager
import android.os.Bundle
import android.telephony.PhoneNumberUtils
import android.telephony.TelephonyManager
import android.view.View
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.lifecycle.MutableLiveData
import com.google.android.material.button.MaterialButton
import com.google.android.material.progressindicator.LinearProgressIndicator
import com.google.android.material.textfield.TextInputEditText
import com.google.android.material.textfield.TextInputLayout
import timber.log.Timber

class LoginActivity : AppCompatActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        redirectToMain()
        setContentView(R.layout.activity_login)
        registerListeners()
        setPhoneNumber()
    }

    private fun registerListeners() {
        loginButton().setOnClickListener { onLoginClick() }
    }

    private fun setPhoneNumber() {
        val phoneNumber = getPhoneNumber(this)
        if(phoneNumber == null) {
            Timber.d("cannot get phone due to no permissions")
            return
        }

        val phoneInput = findViewById<TextInputEditText>(R.id.loginPhoneNumberInput)
        phoneInput.setText(phoneNumber)
        Timber.d("phone number [$phoneNumber] set successfully")
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

        val phoneNumberLayout = findViewById<TextInputLayout>(R.id.loginPhoneNumberLayout)
        phoneNumberLayout.error = null

        val phoneNumber = findViewById<TextInputEditText>(R.id.loginPhoneNumberInput)
        phoneNumber.isEnabled = false

        val resetView = fun () {
            apiKey.isEnabled = true
            progressBar.visibility = View.INVISIBLE
            phoneNumber.isEnabled = true
            loginButton().isEnabled = true
        }

        if (
            !PhoneNumberUtils.isWellFormedSmsAddress(phoneNumber.text.toString()) ||
            !PhoneNumberUtils.isGlobalPhoneNumber(phoneNumber.text.toString())
        ) {
            Timber.e("phone number [$phoneNumber] is not valid")
            resetView()
            phoneNumberLayout.error = "Invalid E.164 phone number"
            return
        }

        val liveData = MutableLiveData<String?>()
        liveData.observe(this) { authResult ->
            run {
                progressBar.visibility = View.INVISIBLE
                if (authResult != null) {
                    resetView()
                    apiKeyLayout.error = authResult
                    return@run
                }

                Settings.setApiKeyAsync(this, apiKey.text.toString())

                val e164PhoneNumber = PhoneNumberUtils.formatNumberToE164(
                    phoneNumber.text.toString(),
                    this.resources.configuration.locales.get(0).country
                )
                Settings.setOwnerAsync(this, e164PhoneNumber)

                Timber.d("login successfully redirecting to main view")
                redirectToMain()
            }
        }

        Thread {
            val error = HttpSmsApiService(apiKey.text.toString()).validateApiKey()
            liveData.postValue(error)
            Timber.i("login successful")
        }.start()
    }


    private fun redirectToMain() {
        if (!Settings.isLoggedIn(this)) {
            return
        }
       finish()
    }

    private fun loginButton(): MaterialButton {
        return findViewById(R.id.loginButton)
    }
}
