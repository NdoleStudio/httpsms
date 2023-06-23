package com.httpsms

import android.content.Context
import androidx.preference.PreferenceManager
import timber.log.Timber
import java.net.URI

object Settings {
    private const val SETTINGS_SIM1_PHONE_NUMBER = "SETTINGS_SIM1_PHONE_NUMBER"
    private const val SETTINGS_SIM2_PHONE_NUMBER = "SETTINGS_SIM2_PHONE_NUMBER"
    private const val SETTINGS_ACTIVE = "SETTINGS_ACTIVE_STATUS"
    private const val SETTINGS_SIM1_ACTIVE = "SETTINGS_SIM1_ACTIVE_STATUS"
    private const val SETTINGS_SIM2_ACTIVE = "SETTINGS_SIM2_ACTIVE_STATUS"
    private const val SETTINGS_SIM1_INCOMING_ACTIVE = "SETTINGS_SIM1_INCOMING_ACTIVE"
    private const val SETTINGS_SIM2_INCOMING_ACTIVE = "SETTINGS_SIM1_INCOMING_ACTIVE"
    private const val SETTINGS_API_KEY = "SETTINGS_API_KEY"
    private const val SETTINGS_SERVER_URL = "SETTINGS_SERVER_URL"
    private const val SETTINGS_FCM_TOKEN = "SETTINGS_FCM_TOKEN"
    private const val SETTINGS_FCM_TOKEN_UPDATE_TIMESTAMP = "SETTINGS_FCM_TOKEN_UPDATE_TIMESTAMP"
    private const val SETTINGS_HEARTBEAT_TIMESTAMP = "SETTINGS_HEARTBEAT_TIMESTAMP"

    fun getSIM1PhoneNumber(context: Context): String {
        Timber.d(Settings::getSIM1PhoneNumber.name)

        val owner = PreferenceManager
            .getDefaultSharedPreferences(context)
            .getString(this.SETTINGS_SIM1_PHONE_NUMBER, null)

        if (owner == null) {
            Timber.e("cannot get owner from preference [${this.SETTINGS_SIM1_PHONE_NUMBER}]")
            return ""
        }

        Timber.d("SETTINGS_SIM1_PHONE_NUMBER: [$owner]")
        return owner
    }

    fun getSIM2PhoneNumber(context: Context): String {
        Timber.d(Settings::getSIM2PhoneNumber.name)

        val owner = PreferenceManager
            .getDefaultSharedPreferences(context)
            .getString(this.SETTINGS_SIM2_PHONE_NUMBER, null)

        if (owner == null) {
            Timber.e("cannot get owner from preference [${this.SETTINGS_SIM2_PHONE_NUMBER}]")
            return ""
        }

        Timber.d("SETTINGS_SIM2_PHONE_NUMBER: [$owner]")
        return owner
    }

    fun hasOwner(context: Context): Boolean {
        return getSIM1PhoneNumber(context) != ""
    }

    fun getFcmTokenLastUpdateTimestamp(context: Context): Long {
        Timber.d(Settings::getFcmTokenLastUpdateTimestamp.name)

        val timestamp = PreferenceManager
            .getDefaultSharedPreferences(context)
            .getLong(this.SETTINGS_FCM_TOKEN_UPDATE_TIMESTAMP,0)

        Timber.d("SETTINGS_FCM_TOKEN_UPDATE_TIMESTAMP: [$timestamp]")
        return timestamp
    }


    fun setFcmTokenLastUpdateTimestampAsync(context: Context, timestamp: Long) {
        Timber.d(Settings::setFcmTokenLastUpdateTimestampAsync.name)

        PreferenceManager.getDefaultSharedPreferences(context)
            .edit()
            .putLong(this.SETTINGS_FCM_TOKEN_UPDATE_TIMESTAMP, timestamp)
            .apply()
    }

    fun setSIM1PhoneNumber(context: Context, owner: String?) {
        Timber.d(Settings::setSIM1PhoneNumber.name)

        PreferenceManager.getDefaultSharedPreferences(context)
            .edit()
            .putString(this.SETTINGS_SIM1_PHONE_NUMBER, owner)
            .apply()
    }

    fun setSIM2PhoneNumber(context: Context, owner: String?) {
        Timber.d(Settings::setSIM2PhoneNumber.name)

        PreferenceManager.getDefaultSharedPreferences(context)
            .edit()
            .putString(this.SETTINGS_SIM2_PHONE_NUMBER, owner)
            .apply()
    }

    fun isIncomingMessageEnabled(context: Context, sim: String): Boolean {
        var setting = this.SETTINGS_SIM1_INCOMING_ACTIVE
        if (sim == Constants.SIM2) {
            setting = this.SETTINGS_SIM2_INCOMING_ACTIVE
        }
        val activeStatus = PreferenceManager
            .getDefaultSharedPreferences(context)
            .getBoolean(setting,true)

        Timber.d("SETTINGS_${sim}_INCOMING_ACTIVE: [$activeStatus]")
        return activeStatus
    }

    fun setIncomingActiveSIM1(context: Context, status: Boolean) {
        Timber.d(Settings::setIncomingActiveSIM1.name)

        PreferenceManager.getDefaultSharedPreferences(context)
            .edit()
            .putBoolean(this.SETTINGS_SIM1_INCOMING_ACTIVE, status)
            .apply()
    }

    fun setIncomingActiveSIM2(context: Context, status: Boolean) {
        Timber.d(Settings::setIncomingActiveSIM2.name)

        PreferenceManager.getDefaultSharedPreferences(context)
            .edit()
            .putBoolean(this.SETTINGS_SIM2_INCOMING_ACTIVE, status)
            .apply()
    }


    fun getActiveStatus(context: Context, sim: String): Boolean {
        var setting = this.SETTINGS_SIM1_ACTIVE
        if (sim == Constants.SIM2) {
            setting = this.SETTINGS_SIM2_ACTIVE
        }
        val activeStatus = PreferenceManager
            .getDefaultSharedPreferences(context)
            .getBoolean(setting,true)

        Timber.d("SETTINGS_${sim}_ACTIVE: [$activeStatus]")
        return activeStatus
    }

    fun setActiveStatusAsync(context: Context, status: Boolean, sim: String) {
        Timber.d(Settings::setActiveStatusAsync.name)

        var setting = this.SETTINGS_SIM1_ACTIVE
        if (sim == Constants.SIM2) {
            setting = this.SETTINGS_SIM2_ACTIVE
        }

        PreferenceManager.getDefaultSharedPreferences(context)
            .edit()
            .putBoolean(setting, status)
            .apply()
    }

    fun isLoggedIn(context: Context): Boolean {
       return getApiKey(context) != null
    }

    fun isDualSIM(context: Context): Boolean {
        return getSIM1PhoneNumber(context) != "" && getSIM2PhoneNumber(context) != ""
    }

    private fun getApiKey(context: Context): String?{
        Timber.d(Settings::getApiKey.name)

        val apiKey = PreferenceManager
            .getDefaultSharedPreferences(context)
            .getString(this.SETTINGS_API_KEY,null)

        Timber.d("SETTINGS_API_KEY: [$apiKey]")
        return apiKey
    }

    fun getApiKeyOrDefault(context:Context): String {
        return getApiKey(context) ?: ""
    }

    fun getServerUrlOrDefault(context:Context): URI {
        val urlString = getServerUrl(context) ?: "https://api.httpsms.com"
        return URI(urlString)
    }

    private fun getServerUrl(context: Context): String? {
        Timber.d(Settings::getServerUrl.name)

        val serverUrl = PreferenceManager
            .getDefaultSharedPreferences(context)
            .getString(this.SETTINGS_SERVER_URL,null)

        Timber.d("SETTINGS_SERVER_URL: [$serverUrl]")
        return serverUrl
    }

    fun setServerUrlAsync(context: Context, serverURL: String?) {
        Timber.d(Settings::SETTINGS_SERVER_URL.name)

        PreferenceManager.getDefaultSharedPreferences(context)
            .edit()
            .putString(this.SETTINGS_SERVER_URL, serverURL)
            .apply()
    }

    fun setApiKeyAsync(context: Context, apiKey: String?) {
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

        Timber.d("SETTINGS_FCM_TOKEN: [$activeStatus]")
        return activeStatus
    }

    fun setFcmTokenAsync(context: Context, apiKey: String) {
        Timber.d(Settings::setApiKeyAsync.name)

        PreferenceManager.getDefaultSharedPreferences(context)
            .edit()
            .putString(this.SETTINGS_FCM_TOKEN, apiKey)
            .apply()
    }

    fun getHeartbeatTimestamp(context: Context): Long {
        Timber.d(Settings::getHeartbeatTimestamp.name)

        val timestamp = PreferenceManager
            .getDefaultSharedPreferences(context)
            .getLong(this.SETTINGS_HEARTBEAT_TIMESTAMP,0)

        Timber.d("SETTINGS_HEARTBEAT_TIMESTAMP: [$timestamp]")
        return timestamp
    }


    fun setHeartbeatTimestampAsync(context: Context, timestamp: Long) {
        Timber.d(Settings::setHeartbeatTimestampAsync.name)

        PreferenceManager.getDefaultSharedPreferences(context)
            .edit()
            .putLong(this.SETTINGS_HEARTBEAT_TIMESTAMP, timestamp)
            .apply()
    }
}
