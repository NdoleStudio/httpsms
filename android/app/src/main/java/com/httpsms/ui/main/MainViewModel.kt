package com.httpsms.ui.main

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.os.PowerManager
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.httpsms.Constants
import com.httpsms.HttpSmsApiService
import com.httpsms.Settings
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import timber.log.Timber
import java.time.Instant
import java.time.ZoneId
import java.time.ZoneOffset
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter

data class MainUiState(
    val phoneNumberSIM1: String = "",
    val isActiveSIM1: Boolean = false,
    val phoneNumberSIM2: String = "",
    val isActiveSIM2: Boolean = false,
    val isDualSim: Boolean = false,
    val lastHeartbeatTime: String = "--",
    val isSmsPermissionGranted: Boolean = true,
    val isBatteryOptimizationDisabled: Boolean = true,
    val isHeartbeatLoading: Boolean = false,
    val appVersion: String = ""
)

class MainViewModel : ViewModel() {
    private val _uiState = MutableStateFlow(MainUiState())
    val uiState = _uiState.asStateFlow()

    fun initialize(context: Context, appVersion: String) {
        updateState(context, appVersion)
    }

    fun updateState(context: Context, appVersion: String) {
        val isDualSim = Settings.isDualSIM(context)
        val phone1 = Settings.getSIM1PhoneNumber(context) ?: ""
        val active1 = Settings.getActiveStatus(context, Constants.SIM1)
        val phone2 = Settings.getSIM2PhoneNumber(context) ?: ""
        val active2 = Settings.getActiveStatus(context, Constants.SIM2)
        
        val timestamp = Settings.getHeartbeatTimestamp(context)
        val lastHeartbeat = if (timestamp == 0L) {
            "--"
        } else {
            val timestampZdt = ZonedDateTime.ofInstant(Instant.ofEpochMilli(timestamp), ZoneOffset.UTC)
            val localTime = timestampZdt.withZoneSameInstant(ZoneId.systemDefault())
            localTime.format(DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss"))
        }

        val smsPermissions = arrayOf(
            Manifest.permission.SEND_SMS,
            Manifest.permission.RECEIVE_SMS,
            Manifest.permission.READ_SMS
        )
        val allGranted = smsPermissions.all {
            context.checkSelfPermission(it) == PackageManager.PERMISSION_GRANTED
        }

        val pm = context.getSystemService(Context.POWER_SERVICE) as PowerManager
        val batteryOptimized = pm.isIgnoringBatteryOptimizations(context.packageName)

        _uiState.value = _uiState.value.copy(
            phoneNumberSIM1 = phone1,
            isActiveSIM1 = active1,
            phoneNumberSIM2 = phone2,
            isActiveSIM2 = active2,
            isDualSim = isDualSim,
            lastHeartbeatTime = lastHeartbeat,
            isSmsPermissionGranted = allGranted,
            isBatteryOptimizationDisabled = batteryOptimized,
            appVersion = appVersion
        )
    }

    fun sendHeartbeat(context: Context, onComplete: (String?) -> Unit) {
        _uiState.value = _uiState.value.copy(isHeartbeatLoading = true)
        
        viewModelScope.launch {
            val result = withContext(Dispatchers.IO) {
                val charging = Settings.isCharging(context)
                try {
                    val phoneNumbers = mutableListOf<String>()
                    phoneNumbers.add(Settings.getSIM1PhoneNumber(context))
                    if (Settings.getActiveStatus(context, Constants.SIM2)) {
                        phoneNumbers.add(Settings.getSIM2PhoneNumber(context))
                    }
                    val isStored = HttpSmsApiService.create(context).storeHeartbeat(phoneNumbers.toTypedArray(), charging)
                    if (!isStored) {
                        "Could not send heartbeat make sure the phone is connected to the internet"
                    } else {
                        Settings.setHeartbeatTimestampAsync(context, System.currentTimeMillis())
                        null
                    }
                } catch (exception: Exception) {
                    Timber.e(exception)
                    exception.javaClass.simpleName
                }
            }
            
            _uiState.value = _uiState.value.copy(isHeartbeatLoading = false)
            updateState(context, _uiState.value.appVersion)
            onComplete(result)
        }
    }
}
