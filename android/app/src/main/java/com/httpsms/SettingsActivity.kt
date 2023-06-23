package com.httpsms

import android.content.Intent
import android.os.Bundle
import androidx.appcompat.app.AppCompatActivity
import androidx.appcompat.widget.Toolbar
import com.google.android.material.appbar.MaterialToolbar
import com.google.android.material.button.MaterialButton
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import timber.log.Timber

class SettingsActivity : AppCompatActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_settings)
        registerListeners()
    }

    private fun registerListeners() {
        appToolbar().setOnClickListener { onBackClicked() }
        findViewById<MaterialButton>(R.id.settingsLogoutButton).setOnClickListener { onLogoutClick() }
    }

    private fun onBackClicked() {
        Timber.e("back button clicked")
        redirectToMain()
    }


    private fun redirectToMain() {
        finish()
        val switchActivityIntent = Intent(this, MainActivity::class.java)
        startActivity(switchActivityIntent)
    }

    private fun appToolbar(): MaterialToolbar {
        return findViewById(R.id.settings_toolbar)
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
                Settings.setSIM1PhoneNumber(this, null)
                Settings.setSIM2PhoneNumber(this, null)
                Settings.setActiveStatusAsync(this, true)
                Settings.setIncomingActiveSIM1(this, true)
                Settings.setIncomingActiveSIM2(this, true)
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
}
