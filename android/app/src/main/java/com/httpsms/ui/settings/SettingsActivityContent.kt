package com.httpsms.ui.settings

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.httpsms.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(
    viewModel: SettingsViewModel,
    onBackClick: () -> Unit,
    onLogoutClick: () -> Unit
) {
    val uiState by viewModel.uiState.collectAsState()
    val context = LocalContext.current

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("App Settings") },
                navigationIcon = {
                    IconButton(onClick = onBackClick) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = Color.Black,
                    titleContentColor = Color.White,
                    navigationIconContentColor = Color.White
                )
            )
        }
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .verticalScroll(rememberScrollState())
                .padding(16.dp)
        ) {
            // SIM 1 Settings
            OutlinedTextField(
                value = uiState.phoneNumberSIM1,
                onValueChange = { },
                label = { Text(stringResource(id = R.string.settings_sim1)) },
                modifier = Modifier.fillMaxWidth(),
                enabled = false
            )

            SwitchSetting(
                text = stringResource(id = R.string.settings_outgoing_messages_sim1),
                checked = uiState.isActiveSIM1,
                onCheckedChange = { viewModel.setActiveSIM1(context, it) }
            )

            SwitchSetting(
                text = stringResource(id = R.string.settings_incoming_messages_sim1),
                checked = uiState.isIncomingSIM1Enabled,
                onCheckedChange = { viewModel.setIncomingSIM1Enabled(context, it) }
            )

            SwitchSetting(
                text = stringResource(id = R.string.enable_incoming_call_events_sim1),
                checked = uiState.isIncomingCallEventsSIM1Enabled,
                onCheckedChange = { viewModel.setIncomingCallEventsSIM1Enabled(context, it) }
            )

            if (uiState.isDualSim) {
                Spacer(modifier = Modifier.height(16.dp))
                // SIM 2 Settings
                OutlinedTextField(
                    value = uiState.phoneNumberSIM2,
                    onValueChange = { },
                    label = { Text(stringResource(id = R.string.settings_sim_2)) },
                    modifier = Modifier.fillMaxWidth(),
                    enabled = false
                )

                SwitchSetting(
                    text = stringResource(id = R.string.settings_outgoing_messages_sim2),
                    checked = uiState.isActiveSIM2,
                    onCheckedChange = { viewModel.setActiveSIM2(context, it) }
                )

                SwitchSetting(
                    text = stringResource(id = R.string.settings_incoming_messages_sim2),
                    checked = uiState.isIncomingSIM2Enabled,
                    onCheckedChange = { viewModel.setIncomingSIM2Enabled(context, it) }
                )

                SwitchSetting(
                    text = stringResource(id = R.string.enable_sim2_incoming_call_events),
                    checked = uiState.isIncomingCallEventsSIM2Enabled,
                    onCheckedChange = { viewModel.setIncomingCallEventsSIM2Enabled(context, it) }
                )
            }

            Spacer(modifier = Modifier.height(16.dp))

            OutlinedTextField(
                value = uiState.encryptionKey,
                onValueChange = { viewModel.setEncryptionKey(context, it) },
                label = { Text(stringResource(id = R.string.encryption_key)) },
                modifier = Modifier.fillMaxWidth()
            )

            SwitchSetting(
                text = stringResource(id = R.string.encrypt_received_messages),
                checked = uiState.isEncryptReceivedMessagesEnabled,
                onCheckedChange = { viewModel.setEncryptReceivedMessagesEnabled(context, it) },
                enabled = uiState.encryptionKey.isNotEmpty()
            )

            SwitchSetting(
                text = stringResource(id = R.string.enable_debug_logs),
                checked = uiState.isDebugLogEnabled,
                onCheckedChange = { viewModel.setDebugLogEnabled(context, it) }
            )

            Spacer(modifier = Modifier.height(24.dp))

            Button(
                onClick = onLogoutClick,
                modifier = Modifier.fillMaxWidth(),
                colors = ButtonDefaults.buttonColors(containerColor = Color.Black)
            ) {
                Text(stringResource(id = R.string.main_log_out), color = Color.White)
            }
        }
    }
}

@Composable
fun SwitchSetting(
    text: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
    enabled: Boolean = true
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = text,
            modifier = Modifier.weight(1f),
            fontSize = 18.sp,
            color = if (enabled) MaterialTheme.colorScheme.onBackground else MaterialTheme.colorScheme.onBackground.copy(alpha = 0.5f)
        )
        Switch(
            checked = checked,
            onCheckedChange = onCheckedChange,
            enabled = enabled
        )
    }
}
