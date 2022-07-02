package com.httpsms

import android.os.Bundle
import android.view.View
import androidx.appcompat.app.AppCompatActivity
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
    }

    private fun registerListeners() {
        loginButton().setOnClickListener { onLoginClick() }
    }

    private fun onLoginClick() {
        Timber.e("login button clicked")
        loginButton().isEnabled = false
        val progressBar = findViewById<LinearProgressIndicator>(R.id.loginProgressIndicator)
        progressBar.visibility = View.VISIBLE

        val apiKeyLayout = findViewById<TextInputLayout>(R.id.loginApiKeyTextInputLayout)
        apiKeyLayout.error = null

        val apiKey = findViewById<TextInputEditText>(R.id.loginApiKeyTextInput)
        apiKey.isEnabled = false

        val liveData = MutableLiveData<String?>()
        liveData.observe(this) { authResult ->
            run {
                progressBar.visibility = View.INVISIBLE
                if (authResult != null) {
                    apiKey.isEnabled = true
                    loginButton().isEnabled = true
                    apiKeyLayout.error = authResult
                    return@run
                }
                Timber.d("login successfully redirecting to main view")
                Settings.setApiKeyAsync(this, apiKey.text.toString())
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
