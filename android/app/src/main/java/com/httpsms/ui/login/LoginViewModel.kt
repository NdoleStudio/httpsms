package com.httpsms.ui.login

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import android.telephony.SubscriptionManager
import android.telephony.TelephonyManager
import android.webkit.URLUtil
import androidx.core.content.ContextCompat
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.httpsms.Constants
import com.httpsms.HttpSmsApiService
import com.httpsms.Settings
import com.httpsms.SmsManagerService
import com.httpsms.validators.PhoneNumberValidator
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import timber.log.Timber
import java.net.URI
import java.net.URISyntaxException

data class LoginUiState(
    val apiKey: String = "",
    val phoneNumberSIM1: String = "",
    val phoneNumberSIM2: String = "",
    val serverUrl: String = "",
    val isLoading: Boolean = false,
    val apiKeyError: String? = null,
    val phoneNumberSIM1Error: String? = null,
    val phoneNumberSIM2Error: String? = null,
    val serverUrlError: String? = null,
    val isDualSim: Boolean = false,
    val loginSuccess: Boolean = false
)

class LoginViewModel : ViewModel() {
    private val _uiState = MutableStateFlow(LoginUiState())
    val uiState = _uiState.asStateFlow()

    fun initialize(context: Context, defaultServerUrl: String) {
        val isDualSim = SmsManagerService.isDualSIM(context)
        val phoneNumberSIM1 = Settings.getSIM1PhoneNumber(context)
        val phoneNumberSIM2 = Settings.getSIM2PhoneNumber(context)
        
        _uiState.value = _uiState.value.copy(
            isDualSim = isDualSim,
            phoneNumberSIM1 = phoneNumberSIM1,
            phoneNumberSIM2 = phoneNumberSIM2,
            serverUrl = defaultServerUrl
        )
        
        // Try to auto-detect if fields are empty
        if (phoneNumberSIM1.isEmpty() || (isDualSim && phoneNumberSIM2.isEmpty())) {
            autoDetectPhoneNumbers(context)
        }
    }

    fun autoDetectPhoneNumbers(context: Context) {
        if (ContextCompat.checkSelfPermission(context, Manifest.permission.READ_PHONE_STATE) != PackageManager.PERMISSION_GRANTED &&
            ContextCompat.checkSelfPermission(context, Manifest.permission.READ_SMS) != PackageManager.PERMISSION_GRANTED) {
            Timber.d("Permissions not granted for auto-detecting phone numbers")
            return
        }

        val telephonyManager = context.getSystemService(Context.TELEPHONY_SERVICE) as TelephonyManager
        
        var detectedSIM1 = _uiState.value.phoneNumberSIM1
        var detectedSIM2 = _uiState.value.phoneNumberSIM2

        try {
            val subscriptionManager = if (Build.VERSION.SDK_INT >= 31) {
                context.getSystemService(SubscriptionManager::class.java)
            } else {
                SubscriptionManager.from(context)
            }
            
            val activeSubscriptions = try { subscriptionManager.activeSubscriptionInfoList } catch (e: Exception) { null }
            
            if (detectedSIM1.isEmpty()) {
                val line1Number = try { telephonyManager.line1Number } catch (e: Exception) { null }
                if (!line1Number.isNullOrEmpty()) {
                    detectedSIM1 = line1Number
                } else if (activeSubscriptions != null && activeSubscriptions.isNotEmpty()) {
                    detectedSIM1 = activeSubscriptions[0].number ?: ""
                }
            }

            if (detectedSIM2.isEmpty() && activeSubscriptions != null && activeSubscriptions.size >= 2) {
                detectedSIM2 = activeSubscriptions[1].number ?: ""
            }
            
            Timber.d("Auto-detected numbers - SIM1: $detectedSIM1, SIM2: $detectedSIM2")

        } catch (e: SecurityException) {
            Timber.e(e, "Security exception while auto-detecting phone numbers")
        }

        _uiState.value = _uiState.value.copy(
            phoneNumberSIM1 = detectedSIM1,
            phoneNumberSIM2 = detectedSIM2,
            isDualSim = SmsManagerService.isDualSIM(context)
        )
    }

    fun onApiKeyChange(value: String) {
        _uiState.value = _uiState.value.copy(apiKey = value, apiKeyError = null)
    }

    fun onPhoneNumberSIM1Change(value: String) {
        _uiState.value = _uiState.value.copy(phoneNumberSIM1 = value, phoneNumberSIM1Error = null)
    }

    fun onPhoneNumberSIM2Change(value: String) {
        _uiState.value = _uiState.value.copy(phoneNumberSIM2 = value, phoneNumberSIM2Error = null)
    }

    fun onServerUrlChange(value: String) {
        _uiState.value = _uiState.value.copy(serverUrl = value, serverUrlError = null)
    }

    fun login(context: Context, countryCode: String, onGooglePlayServicesError: (String) -> Unit, onFcmTokenMissing: () -> Unit) {
        val currentState = _uiState.value
        
        // Validation logic from LoginActivity.onLoginClick
        if (Settings.getFcmToken(context) == null) {
            onFcmTokenMissing()
            return
        }

        _uiState.value = currentState.copy(isLoading = true)

        viewModelScope.launch {
            val apiKey = currentState.apiKey.trim()
            val serverUrl = currentState.serverUrl.trim()
            val phone1 = currentState.phoneNumberSIM1.trim()
            val phone2 = currentState.phoneNumberSIM2.trim()

            if (!PhoneNumberValidator.isValidPhoneNumber(phone1, countryCode)) {
                _uiState.value = _uiState.value.copy(
                    isLoading = false,
                    phoneNumberSIM1Error = "Enter an international phone number in the E.164 format"
                )
                return@launch
            }

            if (currentState.isDualSim && !PhoneNumberValidator.isValidPhoneNumber(phone2, countryCode)) {
                _uiState.value = _uiState.value.copy(
                    isLoading = false,
                    phoneNumberSIM2Error = "Enter an international phone number in the E.164 format"
                )
                return@launch
            }

            if (!URLUtil.isValidUrl(serverUrl)) {
                 _uiState.value = _uiState.value.copy(
                    isLoading = false,
                    serverUrlError = "Server URL [$serverUrl] is invalid"
                )
                return@launch
            }

            if (!URLUtil.isHttpsUrl(serverUrl)) {
                _uiState.value = _uiState.value.copy(
                    isLoading = false,
                    serverUrlError = "Server URL [$serverUrl] must be HTTPS"
                )
                return@launch
            }

            val authResult = try {
                withContext(Dispatchers.IO) {
                    val service = HttpSmsApiService(apiKey, URI(serverUrl))
                    val e164Phone1 = PhoneNumberValidator.formatE164(phone1, countryCode)
                    val response1 = service.updateFcmToken(e164Phone1, Constants.SIM1, Settings.getFcmToken(context) ?: "")
                    
                    if (response1.second != null || response1.third != null) {
                        return@withContext Pair(response1.second, response1.third)
                    }

                    if (currentState.isDualSim) {
                        val e164Phone2 = PhoneNumberValidator.formatE164(phone2, countryCode)
                        val response2 = service.updateFcmToken(e164Phone2, Constants.SIM2, Settings.getFcmToken(context) ?: "")
                        return@withContext Pair(response2.second, response2.third)
                    }

                    Pair(null, null)
                }
            } catch (e: URISyntaxException) {
                Timber.e(e, "Invalid URI: $serverUrl")
                Pair(null, "Server URL [$serverUrl] is invalid")
            } catch (e: Exception) {
                Timber.e(e, "Login error")
                Pair(null, "An unexpected error occurred: ${e.message}")
            }

            if (authResult.first != null) {
                _uiState.value = _uiState.value.copy(isLoading = false, apiKeyError = authResult.first)
                return@launch
            }

            if (authResult.second != null) {
                _uiState.value = _uiState.value.copy(isLoading = false, serverUrlError = authResult.second)
                return@launch
            }

            // Save settings
            Settings.setApiKeyAsync(context, apiKey)
            Settings.setServerUrlAsync(context, serverUrl)
            Settings.setSIM1PhoneNumber(context, PhoneNumberValidator.formatE164(phone1, countryCode))
            if (currentState.isDualSim) {
                Settings.setSIM2PhoneNumber(context, PhoneNumberValidator.formatE164(phone2, countryCode))
            }

            _uiState.value = _uiState.value.copy(isLoading = false, loginSuccess = true)
        }
    }
}
