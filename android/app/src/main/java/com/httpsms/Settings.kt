package com.httpsms

import android.content.Context
import android.util.Log
import androidx.preference.PreferenceManager

object Settings {
    const val DEFAULT_PHONE_NUMBER = "NO_PHONE_NUMBER"

    private const val SETTINGS_OWNER = "SETTINGS_OWNER"
    private const val SETTINGS_ACTIVE = "SETTINGS_ACTIVE_STATUS"

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

    private val TAG = Settings::class.simpleName
}
