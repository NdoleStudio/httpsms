package com.httpsms.ui.settings

import android.content.Context
import androidx.lifecycle.ViewModel
import com.httpsms.Constants
import com.httpsms.Settings
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow

data class SettingsUiState(
    val isDebugLogEnabled: Boolean = false,
    val phoneNumberSIM1: String = "",
    val isIncomingSIM1Enabled: Boolean = false,
    val isActiveSIM1: Boolean = false,
    val isIncomingCallEventsSIM1Enabled: Boolean = false,
    val isDualSim: Boolean = false,
    val phoneNumberSIM2: String = "",
    val isIncomingSIM2Enabled: Boolean = false,
    val isActiveSIM2: Boolean = false,
    val isIncomingCallEventsSIM2Enabled: Boolean = false,
    val encryptionKey: String = "",
    val isEncryptReceivedMessagesEnabled: Boolean = false
)

class SettingsViewModel : ViewModel() {
    private val _uiState = MutableStateFlow(SettingsUiState())
    val uiState = _uiState.asStateFlow()

    fun initialize(context: Context) {
        _uiState.value = SettingsUiState(
            isDebugLogEnabled = Settings.isDebugLogEnabled(context),
            phoneNumberSIM1 = Settings.getSIM1PhoneNumber(context) ?: "",
            isIncomingSIM1Enabled = Settings.isIncomingMessageEnabled(context, Constants.SIM1),
            isActiveSIM1 = Settings.getActiveStatus(context, Constants.SIM1),
            isIncomingCallEventsSIM1Enabled = Settings.isIncomingCallEventsEnabled(context, Constants.SIM1),
            isDualSim = Settings.isDualSIM(context),
            phoneNumberSIM2 = Settings.getSIM2PhoneNumber(context) ?: "",
            isIncomingSIM2Enabled = Settings.isIncomingMessageEnabled(context, Constants.SIM2),
            isActiveSIM2 = Settings.getActiveStatus(context, Constants.SIM2),
            isIncomingCallEventsSIM2Enabled = Settings.isIncomingCallEventsEnabled(context, Constants.SIM2),
            encryptionKey = Settings.getEncryptionKey(context) ?: "",
            isEncryptReceivedMessagesEnabled = Settings.encryptReceivedMessages(context)
        )
    }

    fun setDebugLogEnabled(context: Context, enabled: Boolean) {
        Settings.setDebugLogEnabled(context, enabled)
        _uiState.value = _uiState.value.copy(isDebugLogEnabled = enabled)
    }

    fun setIncomingSIM1Enabled(context: Context, enabled: Boolean) {
        Settings.setIncomingActiveSIM1(context, enabled)
        _uiState.value = _uiState.value.copy(isIncomingSIM1Enabled = enabled)
    }

    fun setActiveSIM1(context: Context, enabled: Boolean) {
        Settings.setActiveStatusAsync(context, enabled, Constants.SIM1)
        _uiState.value = _uiState.value.copy(isActiveSIM1 = enabled)
    }

    fun setIncomingCallEventsSIM1Enabled(context: Context, enabled: Boolean) {
        Settings.setIncomingCallEventsEnabled(context, Constants.SIM1, enabled)
        _uiState.value = _uiState.value.copy(isIncomingCallEventsSIM1Enabled = enabled)
    }

    fun setIncomingSIM2Enabled(context: Context, enabled: Boolean) {
        Settings.setIncomingActiveSIM2(context, enabled)
        _uiState.value = _uiState.value.copy(isIncomingSIM2Enabled = enabled)
    }

    fun setActiveSIM2(context: Context, enabled: Boolean) {
        Settings.setActiveStatusAsync(context, enabled, Constants.SIM2)
        _uiState.value = _uiState.value.copy(isActiveSIM2 = enabled)
    }

    fun setIncomingCallEventsSIM2Enabled(context: Context, enabled: Boolean) {
        Settings.setIncomingCallEventsEnabled(context, Constants.SIM2, enabled)
        _uiState.value = _uiState.value.copy(isIncomingCallEventsSIM2Enabled = enabled)
    }

    fun setEncryptionKey(context: Context, key: String) {
        val trimmedKey = key.trim()
        if (trimmedKey.isEmpty()) {
            Settings.setEncryptionKey(context, null)
            Settings.setEncryptReceivedMessages(context, false)
            _uiState.value = _uiState.value.copy(encryptionKey = "", isEncryptReceivedMessagesEnabled = false)
        } else {
            Settings.setEncryptionKey(context, trimmedKey)
            _uiState.value = _uiState.value.copy(encryptionKey = trimmedKey)
        }
    }

    fun setEncryptReceivedMessagesEnabled(context: Context, enabled: Boolean) {
        Settings.setEncryptReceivedMessages(context, enabled)
        _uiState.value = _uiState.value.copy(isEncryptReceivedMessagesEnabled = enabled)
    }

    fun logout(context: Context, onLogoutComplete: () -> Unit) {
        Settings.setApiKeyAsync(context, null)
        Settings.setSIM1PhoneNumber(context, null)
        Settings.setSIM2PhoneNumber(context, null)
        Settings.setActiveStatusAsync(context, true, Constants.SIM1)
        Settings.setActiveStatusAsync(context, true, Constants.SIM2)
        Settings.setIncomingActiveSIM1(context, true)
        Settings.setIncomingActiveSIM2(context, true)
        Settings.setUserID(context, null)
        Settings.setEncryptionKey(context, null)
        Settings.setEncryptReceivedMessages(context, false)
        Settings.setFcmTokenLastUpdateTimestampAsync(context, 0)
        Settings.setIncomingCallEventsEnabled(context, Constants.SIM1, false)
        Settings.setIncomingCallEventsEnabled(context, Constants.SIM2, false)
        onLogoutComplete()
    }
}
