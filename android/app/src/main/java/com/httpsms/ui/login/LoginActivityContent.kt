package com.httpsms.ui.login

import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.ClickableText
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.QrCodeScanner
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
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.httpsms.R
import com.httpsms.ui.theme.Blue500
import com.httpsms.ui.theme.Pink500

@Composable
fun LoginScreen(
    viewModel: LoginViewModel,
    onQrScanClick: () -> Unit,
    onLoginClick: () -> Unit
) {
    val uiState by viewModel.uiState.collectAsState()
    val uriHandler = LocalUriHandler.current

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

        val annotatedString = buildAnnotatedString {
            val text = stringResource(id = R.string.get_your_api_key)
            val linkText = "httpsms.com/settings"
            val startIndex = text.indexOf(linkText)

            if (startIndex >= 0) {
                append(text.substring(0, startIndex))

                pushStringAnnotation(tag = "URL", annotation = "https://httpsms.com/settings")
                withStyle(style = SpanStyle(color = Blue500, fontWeight = FontWeight.Bold)) {
                    append(linkText)
                }
                pop()

                append(text.substring(startIndex + linkText.length))
            } else {
                append(text)
            }
        }

        ClickableText(
            text = annotatedString,
            onClick = { offset ->
                annotatedString.getStringAnnotations(tag = "URL", start = offset, end = offset)
                    .firstOrNull()?.let { annotation ->
                        uriHandler.openUri(annotation.item)
                    }
            },
            style = MaterialTheme.typography.bodyLarge.copy(
                textAlign = TextAlign.Center,
                fontSize = 20.sp,
                lineHeight = 28.sp,
                color = MaterialTheme.colorScheme.onBackground
            ),
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
                        imageVector = Icons.Default.QrCodeScanner,
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
            colors = ButtonDefaults.buttonColors(containerColor = Blue500),
            contentPadding = PaddingValues(horizontal = 32.dp, vertical = 16.dp)
        ) {
            Icon(
                painter = painterResource(id = R.drawable.ic_login),
                contentDescription = null,
                modifier = Modifier.padding(end = 8.dp),
                tint = Color.White
            )
            Text(
                text = stringResource(id = R.string.sign_in_button),
                color = Color.White,
                fontSize = 18.sp
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
