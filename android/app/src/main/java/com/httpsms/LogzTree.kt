package com.httpsms

import android.content.Context
import android.os.Build
import com.beust.klaxon.Json
import com.beust.klaxon.Klaxon
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import timber.log.Timber
import java.time.ZoneOffset
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter
import io.sentry.Sentry

class LogzTree(val context: Context): Timber.DebugTree() {
    private val client = OkHttpClient()

    override fun log(priority: Int, tag: String?, message: String, t: Throwable?) {
        val formatter: DateTimeFormatter = DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm:ss.SSS'Z'")
        val logEntry = LogEntry(
            BuildConfig.APPLICATION_ID,
            BuildConfig.VERSION_NAME,
            priority,
            severity(priority),
            tag,
            message,
            Build.MODEL,
            Build.BRAND,
            Build.DEVICE,
            Build.VERSION.SDK_INT,
            ZonedDateTime.now(ZoneOffset.UTC).format(formatter),
            Settings.getUserID(context),
            t
        )

        val body = Klaxon().toJsonString(listOf(logEntry)).toRequestBody("application/json".toMediaType())
        val request: Request = Request.Builder()
            .url("https://api.axiom.co/v1/datasets/production/ingest")
            .post(body)
            .header("Content-Type", "application/json")
            .header("Authorization", "Bearer xaat-2a2e0b73-3702-4971-a80f-be3956934950")
            .build()

        Thread {
            try {
                val response = client.newCall(request).execute()
                response.body?.close()
            } catch(ex: Exception) {
                Sentry.captureException(ex)
            }
        }.start()
    }

    private fun severity(priority: Int): String {
        return when(priority) {
            3 -> "DEBUG"
            4 -> "INFO"
            5 -> "WARNING"
            6 -> "ERROR"
            7 -> "ASSERT"
            else -> "VERBOSE"
        }
    }

    class LogEntry(
        val name: String,
        val release: String,
        val priority: Int,
        val severity: String,
        val tag: String?,
        val message: String,
        val model: String,
        val brand: String,
        val device: String,
        val version: Int,
        @Json(name = "@timestamp")
        val dt: String,
        val userID: String,
        val throwable: Throwable?)
}
