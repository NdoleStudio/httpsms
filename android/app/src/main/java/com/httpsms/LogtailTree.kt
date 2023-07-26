package com.httpsms

import android.content.Context
import android.os.Build
import com.beust.klaxon.Klaxon
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import timber.log.Timber
import java.time.ZoneOffset
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter
import java.util.concurrent.ConcurrentLinkedQueue

class LogtailTree(val context: Context): Timber.DebugTree() {
    private val client = OkHttpClient()
    private val jsonMediaType = "application/json; charset=utf-8".toMediaType()
    private val queue: ConcurrentLinkedQueue<LogEntry> = ConcurrentLinkedQueue<LogEntry>()

    override fun log(priority: Int, tag: String?, message: String, t: Throwable?) {
        val formatter: DateTimeFormatter = DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm:ss.SSS'Z'")
        val logEntry = LogEntry(
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
        queue.add(logEntry)
        if (queue.size < 100) {
            return
        }

        val logEntries = queue.toArray()
        queue.clear()

        val request: Request = Request.Builder()
            .url("https://in.logtail.com")
            .post(Klaxon().toJsonString(logEntries).toRequestBody(jsonMediaType))
            .header("Authorization", "Bearer m7ZoA8u5KRYNe6RnEdWeZqsZ")
            .build()

        Thread {
            try {
                val response = client.newCall(request).execute()
                response.body?.close()
            } catch(_: Exception) {
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
        val release: String,
        val priority: Int,
        val severity: String,
        val tag: String?,
        val message: String,
        val model: String,
        val brand: String,
        val device: String,
        val version: Int,
        val dt: String,
        val userID: String,
        val throwable: Throwable?)
}
