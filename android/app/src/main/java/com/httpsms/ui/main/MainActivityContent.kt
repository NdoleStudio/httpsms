package com.httpsms.ui.main

import android.telephony.PhoneNumberUtils
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.res.vectorResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.httpsms.R
import com.httpsms.ui.theme.Blue500
import com.httpsms.ui.theme.Pink500
import java.util.*

@Composable
fun MainScreen(
    viewModel: MainViewModel,
    onSettingsClick: () -> Unit,
    onSmsPermissionClick: () -> Unit,
    onBatteryOptimizationClick: () -> Unit,
    onHeartbeatClick: () -> Unit
) {
    val uiState by viewModel.uiState.collectAsState()
    val context = LocalContext.current

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Spacer(modifier = Modifier.height(16.dp))

        Image(
            painter = painterResource(id = R.drawable.logo_cropped),
            contentDescription = stringResource(id = R.string.img_http_sms_logo),
            modifier = Modifier
                .width(147.dp)
                .height(92.dp)
        )

        Spacer(modifier = Modifier.height(24.dp))

        PhoneCard(
            phoneNumber = uiState.phoneNumberSIM1,
            isActive = uiState.isActiveSIM1,
            refreshTime = uiState.lastHeartbeatTime
        )

        if (uiState.isDualSim) {
            Spacer(modifier = Modifier.height(24.dp))
            PhoneCard(
                phoneNumber = uiState.phoneNumberSIM2,
                isActive = uiState.isActiveSIM2,
                refreshTime = uiState.lastHeartbeatTime
            )
        }

        Spacer(modifier = Modifier.height(16.dp))

        if (!uiState.isSmsPermissionGranted || !uiState.isBatteryOptimizationDisabled) {
            Column(modifier = Modifier.fillMaxWidth()) {
                if (!uiState.isSmsPermissionGranted) {
                    Button(
                        onClick = onSmsPermissionClick,
                        modifier = Modifier.fillMaxWidth(),
                        colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF4CAF50))
                    ) {
                        Text(stringResource(id = R.string.enable_sms_permission), color = Color.White)
                        Spacer(modifier = Modifier.width(8.dp))
                        Icon(
                            painter = painterResource(id = R.drawable.open_in_new_24),
                            contentDescription = null,
                            tint = Color.White
                        )
                    }
                    Spacer(modifier = Modifier.height(8.dp))
                }

                if (!uiState.isBatteryOptimizationDisabled) {
                    Button(
                        onClick = onBatteryOptimizationClick,
                        modifier = Modifier.fillMaxWidth(),
                        colors = ButtonDefaults.buttonColors(containerColor = Pink500)
                    ) {
                        Icon(
                            imageVector = ImageVector.vectorResource(id = android.R.drawable.ic_lock_idle_low_battery),
                            contentDescription = null,
                            tint = Color.White
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(stringResource(id = R.string.disable_battery_optimization), color = Color.White)
                    }
                }
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        Button(
            onClick = onHeartbeatClick,
            modifier = Modifier.fillMaxWidth(),
            enabled = !uiState.isHeartbeatLoading,
            colors = ButtonDefaults.buttonColors(containerColor = Blue500)
        ) {
            Text(stringResource(id = R.string.send_heartbeat), color = Color.White)
        }

        if (uiState.isHeartbeatLoading) {
            LinearProgressIndicator(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 4.dp),
                color = Pink500
            )
        }

        Spacer(modifier = Modifier.height(16.dp))

        Text(
            text = uiState.appVersion,
            fontSize = 14.sp,
            color = MaterialTheme.colorScheme.onBackground.copy(alpha = 0.6f)
        )

        Spacer(modifier = Modifier.weight(1f))
        Spacer(modifier = Modifier.height(16.dp))

        Button(
            onClick = onSettingsClick,
            colors = ButtonDefaults.buttonColors(containerColor = Color.Black)
        ) {
            Icon(Icons.Default.Settings, contentDescription = null, tint = Color.White)
            Spacer(modifier = Modifier.width(8.dp))
            Text(stringResource(id = R.string.main_app_settings), color = Color.White)
        }
        
        Spacer(modifier = Modifier.height(16.dp))
    }
}

@Composable
fun PhoneCard(
    phoneNumber: String,
    isActive: Boolean,
    refreshTime: String
) {
        Card(
            modifier = Modifier.fillMaxWidth(),
            elevation = CardDefaults.cardElevation(defaultElevation = 8.dp)
        ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = PhoneNumberUtils.formatNumber(phoneNumber, Locale.getDefault().country) ?: phoneNumber,
                    fontSize = 28.sp,
                    fontWeight = FontWeight.Medium,
                    modifier = Modifier.weight(1f)
                )
                if (isActive) {
                    Icon(
                        imageVector = Icons.Default.CheckCircle,
                        contentDescription = "Active",
                        tint = Color(0xFF70AB5C),
                        modifier = Modifier.size(24.dp)
                    )
                }
            }
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                text = refreshTime,
                fontSize = 16.sp,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f)
            )
        }
    }
}
