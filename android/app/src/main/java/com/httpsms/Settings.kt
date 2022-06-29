package com.httpsms

import android.content.Context
import androidx.preference.PreferenceManager
import timber.log.Timber

object Settings {
    private const val DEFAULT_PHONE_NUMBER = "66836863" // NOT_FOUND :)

    private const val SETTINGS_OWNER = "SETTINGS_OWNER"
    private const val SETTINGS_ACTIVE = "SETTINGS_ACTIVE_STATUS"
    private const val SETTINGS_API_KEY = "SETTINGS_API_KEY"
    private const val SETTINGS_FCM_TOKEN = "SETTINGS_FCM_TOKEN"

    fun getOwner(context: Context): String? {
        Timber.d(Settings::getOwner.name)

        val owner = PreferenceManager
            .getDefaultSharedPreferences(context)
            .getString(this.SETTINGS_OWNER, null)

        if (owner == null) {
            Timber.e("cannot get owner from preference [${this.SETTINGS_OWNER}]")
            return null
        }

        Timber.d("owner: [$owner]")
        return owner
    }

    fun getOwnerOrDefault(context: Context): String {
        return getOwner(context) ?: return DEFAULT_PHONE_NUMBER
    }

    fun setOwnerAsync(context: Context, owner: String) {
        Timber.d(Settings::getOwner.name)

        PreferenceManager.getDefaultSharedPreferences(context)
            .edit()
            .putString(this.SETTINGS_OWNER, owner)
            .apply()
    }

    fun getActiveStatus(context: Context): Boolean {
        Timber.d(Settings::getActiveStatus.name)

        val activeStatus = PreferenceManager
            .getDefaultSharedPreferences(context)
            .getBoolean(this.SETTINGS_ACTIVE,false)

        Timber.d("active status: [$activeStatus]")
        return activeStatus
    }

    fun setActiveStatusAsync(context: Context, status: Boolean) {
        Timber.d(Settings::setActiveStatusAsync.name)

        PreferenceManager.getDefaultSharedPreferences(context)
            .edit()
            .putBoolean(this.SETTINGS_ACTIVE, status)
            .apply()
    }

    fun isLoggedIn(context: Context): Boolean {
       return getApiKey(context) != null
    }

    private fun getApiKey(context: Context): String?{
        Timber.d(Settings::getApiKey.name)

        val apiKey = PreferenceManager
            .getDefaultSharedPreferences(context)
            .getString(this.SETTINGS_API_KEY,null)

        Timber.d("API_KEY: [$apiKey]")
        return apiKey
    }

    fun getApiKeyOrDefault(context:Context): String {
        return getApiKey(context) ?: ""
    }

    fun setApiKeyAsync(context: Context, apiKey: String) {
        Timber.d(Settings::setApiKeyAsync.name)

        PreferenceManager.getDefaultSharedPreferences(context)
            .edit()
            .putString(this.SETTINGS_API_KEY, apiKey)
            .apply()
    }

    fun getFcmToken(context: Context): String?{
        Timber.d(Settings::getFcmToken.name)

        val activeStatus = PreferenceManager
            .getDefaultSharedPreferences(context)
            .getString(this.SETTINGS_FCM_TOKEN,null)

        Timber.d("API_KEY: [$activeStatus]")
        return activeStatus
    }

    fun setFcmTokenAsync(context: Context, apiKey: String) {
        Timber.d(Settings::setApiKeyAsync.name)

        PreferenceManager.getDefaultSharedPreferences(context)
            .edit()
            .putString(this.SETTINGS_FCM_TOKEN, apiKey)
            .apply()
    }
}
