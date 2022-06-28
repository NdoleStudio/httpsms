package com.httpsms

import android.content.Context
import android.util.Log
import androidx.preference.PreferenceManager

object Settings {
    const val DEFAULT_PHONE_NUMBER = "NO_PHONE_NUMBER"

    private const val SETTINGS_OWNER = "SETTINGS_OWNER"
    private const val SETTINGS_ACTIVE = "SETTINGS_ACTIVE_STATUS"
    private const val SETTINGS_API_KEY = "SETTINGS_API_KEY"
    private const val SETTINGS_FCM_TOKEN = "SETTINGS_FCM_TOKEN"

    fun getOwner(context: Context): String? {
        Log.d(TAG, Settings::getOwner.name)

        val owner = PreferenceManager
            .getDefaultSharedPreferences(context)
            .getString(this.SETTINGS_OWNER, null)

        if (owner == null) {
            Log.e(TAG, "cannot get owner from preference [${this.SETTINGS_OWNER}]")
            return null
        }

        Log.d(TAG, "owner: [$owner]")
        return owner
    }

    fun getPhoneOrDefault(context: Context): String {
        return getOwner(context) ?: return DEFAULT_PHONE_NUMBER
    }

    fun setOwnerAsync(context: Context, owner: String) {
        Log.d(TAG, Settings::getOwner.name)

        PreferenceManager.getDefaultSharedPreferences(context)
            .edit()
            .putString(this.SETTINGS_OWNER, owner)
            .apply()
    }

    fun getActiveStatus(context: Context): Boolean {
        Log.d(TAG, Settings::getActiveStatus.name)

        val activeStatus = PreferenceManager
            .getDefaultSharedPreferences(context)
            .getBoolean(this.SETTINGS_ACTIVE,false)

        Log.d(TAG, "active status: [$activeStatus]")
        return activeStatus
    }

    fun setActiveStatusAsync(context: Context, status: Boolean) {
        Log.d(TAG, Settings::setActiveStatusAsync.name)

        PreferenceManager.getDefaultSharedPreferences(context)
            .edit()
            .putBoolean(this.SETTINGS_ACTIVE, status)
            .apply()
    }

    fun isLoggedIn(context: Context): Boolean {
       return getApiKey(context) != null
    }

    fun getApiKey(context: Context): String?{
        Log.d(TAG, Settings::getApiKey.name)

        val activeStatus = PreferenceManager
            .getDefaultSharedPreferences(context)
            .getString(this.SETTINGS_API_KEY,null)

        Log.d(TAG, "API_KEY: [$activeStatus]")
        return activeStatus
    }

    fun setApiKeyAsync(context: Context, apiKey: String) {
        Log.d(TAG, Settings::setApiKeyAsync.name)

        PreferenceManager.getDefaultSharedPreferences(context)
            .edit()
            .putString(this.SETTINGS_API_KEY, apiKey)
            .apply()
    }

    fun getFcmToken(context: Context): String?{
        Log.d(TAG, Settings::getFcmToken.name)

        val activeStatus = PreferenceManager
            .getDefaultSharedPreferences(context)
            .getString(this.SETTINGS_FCM_TOKEN,null)

        Log.d(TAG, "API_KEY: [$activeStatus]")
        return activeStatus
    }

    fun setFcmToken(context: Context, apiKey: String) {
        Log.d(TAG, Settings::setApiKeyAsync.name)

        PreferenceManager.getDefaultSharedPreferences(context)
            .edit()
            .putString(this.SETTINGS_FCM_TOKEN, apiKey)
            .apply()
    }

    private val TAG = Settings::class.simpleName
}
