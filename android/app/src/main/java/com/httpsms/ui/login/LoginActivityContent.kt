package com.httpsms.ui.login

import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Info
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.res.vectorResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.httpsms.R
import com.httpsms.ui.theme.Blue500
import com.httpsms.ui.theme.Pink500
import androidx.compose.foundation.layout.*

@Composable
fun LoginScreen(
    viewModel: LoginViewModel,
    onQrScanClick: () -> Unit,
    onLoginClick: () -> Unit
) {
    val uiState by viewModel.uiState.collectAsState()

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Spacer(modifier = Modifier.height(64.dp))

        Image(
            painter = painterResource(id = R.drawable.logo_cropped),
            contentDescription = stringResource(id = R.string.img_http_sms_logo),
            modifier = Modifier.size(100.dp)
        )

        Spacer(modifier = Modifier.height(32.dp))

        Text(
            text = stringResource(id = R.string.get_your_api_key),
            textAlign = TextAlign.Center,
            fontSize = 20.sp,
            lineHeight = 28.sp,
            modifier = Modifier.fillMaxWidth()
        )

        Spacer(modifier = Modifier.height(24.dp))

        OutlinedTextField(
            value = uiState.apiKey,
            onValueChange = { viewModel.onApiKeyChange(it) },
            label = { Text(stringResource(id = R.string.text_area_api_key)) },
            modifier = Modifier.fillMaxWidth(),
            isError = uiState.apiKeyError != null,
            supportingText = uiState.apiKeyError?.let { { Text(it) } },
            trailingIcon = {
                IconButton(onClick = onQrScanClick) {
                    Icon(
                        imageVector = ImageVector.vectorResource(id = android.R.drawable.ic_menu_camera),
                        contentDescription = "Scan QR Code"
                    )
                }
            },
            keyboardOptions = KeyboardOptions(
                keyboardType = KeyboardType.Text,
                imeAction = ImeAction.Done
            ),
            enabled = !uiState.isLoading
        )

        Spacer(modifier = Modifier.height(8.dp))

        OutlinedTextField(
            value = uiState.phoneNumberSIM1,
            onValueChange = { viewModel.onPhoneNumberSIM1Change(it) },
            label = { Text(stringResource(id = R.string.login_phone_number_sim1)) },
            placeholder = { Text(stringResource(id = R.string.login_phone_number_hint)) },
            modifier = Modifier.fillMaxWidth(),
            isError = uiState.phoneNumberSIM1Error != null,
            supportingText = uiState.phoneNumberSIM1Error?.let { { Text(it) } },
            keyboardOptions = KeyboardOptions(
                keyboardType = KeyboardType.Phone,
                imeAction = ImeAction.Done
            ),
            enabled = !uiState.isLoading
        )

        if (uiState.isDualSim) {
            Spacer(modifier = Modifier.height(8.dp))
            OutlinedTextField(
                value = uiState.phoneNumberSIM2,
                onValueChange = { viewModel.onPhoneNumberSIM2Change(it) },
                label = { Text(stringResource(id = R.string.login_phone_number_sim2)) },
                placeholder = { Text(stringResource(id = R.string.login_phone_number_hint)) },
                modifier = Modifier.fillMaxWidth(),
                isError = uiState.phoneNumberSIM2Error != null,
                supportingText = uiState.phoneNumberSIM2Error?.let { { Text(it) } },
                keyboardOptions = KeyboardOptions(
                    keyboardType = KeyboardType.Phone,
                    imeAction = ImeAction.Done
                ),
                enabled = !uiState.isLoading
            )
        }

        Spacer(modifier = Modifier.height(8.dp))

        OutlinedTextField(
            value = uiState.serverUrl,
            onValueChange = { viewModel.onServerUrlChange(it) },
            label = { Text(stringResource(id = R.string.server_url)) },
            placeholder = { Text(stringResource(id = R.string.login_server_url_hint)) },
            modifier = Modifier.fillMaxWidth(),
            isError = uiState.serverUrlError != null,
            supportingText = uiState.serverUrlError?.let { { Text(it) } },
            keyboardOptions = KeyboardOptions(
                keyboardType = KeyboardType.Uri,
                imeAction = ImeAction.Done
            ),
            enabled = !uiState.isLoading
        )

        Spacer(modifier = Modifier.height(16.dp))

        Button(
            onClick = onLoginClick,
            enabled = !uiState.isLoading,
            modifier = Modifier.align(Alignment.CenterHorizontally),
            colors = ButtonDefaults.buttonColors(containerColor = Blue500)
        ) {
            Icon(
                painter = painterResource(id = R.drawable.ic_login),
                contentDescription = null,
                modifier = Modifier.padding(end = 8.dp)
            )
            Text(
                text = stringResource(id = R.string.sign_in_button),
                color = Color.White,
                fontSize = 16.sp
            )
        }

        if (uiState.isLoading) {
            Spacer(modifier = Modifier.height(16.dp))
            CircularProgressIndicator(
                modifier = Modifier.size(24.dp),
                color = Pink500
            )
        }
    }
}
