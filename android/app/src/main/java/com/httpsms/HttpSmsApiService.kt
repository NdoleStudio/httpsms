package com.httpsms

import android.util.Log
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.net.URI
import java.text.SimpleDateFormat
import java.time.LocalDateTime
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter
import java.util.*


class HttpSmsApiService {
    private val baseURL = URI("https://httpsms.free.beeceptor.com")
    private val jsonMediaType = "application/json; charset=utf-8".toMediaType()

    fun getOutstandingMessages(): List<Message> {
        val client = OkHttpClient()

        val request: Request = Request.Builder()
            .url(baseURL.resolve("/v1/messages/outstanding").toURL())
            .build()

        val response = client.newCall(request).execute()
        if (response.isSuccessful) {
            val payload =  ResponseMessagesOutstanding.fromJson(response.body!!.string())?.data
            if (payload == null) {
                Log.e(TAG, "cannot decode payload [${response.body}]")
                return listOf();
            }
            return payload
        }

        Log.e(TAG, "invalid response with code [${response.code}] and payload [${response.body}]")
        return listOf()
    }

    fun sendDeliveredEvent(messageId: String, timestamp: ZonedDateTime) {
        sendEvent(messageId, "DELIVERED", timestamp)
    }

    fun sendSentEvent(messageId: String, timestamp: ZonedDateTime) {
        sendEvent(messageId, "SENT", timestamp)
    }

    fun sendFailedEvent(messageId: String, timestamp: ZonedDateTime, reason: String) {
        sendEvent(messageId, "FAILED", timestamp, reason)
    }


    private fun sendEvent(messageId: String, event: String, timestamp: ZonedDateTime, reason: String? = null) {
        val client = OkHttpClient()

        val formatter  = DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm:ss.SSS'000000'ZZZZZ");
        val timestampString = formatter.format(timestamp).replace("+", "Z")

        val body = """
            {
              "event_name": "$event",
              "reason": "$reason"
              "timestamp": "$timestampString"
            }
        """.trimIndent().toRequestBody(jsonMediaType)

        val request: Request = Request.Builder()
            .url(baseURL.resolve("/v1/messages/${messageId}/events").toURL())
            .post(body)
            .build()

        val response = client.newCall(request).execute()
        if (!response.isSuccessful) {
           Log.e(TAG, "error response [${response.body?.string()}] with code [${response.code}] while sending [${event}] event [${body}] for message with ID [${messageId}]")
            return
        }
        Log.i(TAG, "[$event] event sent successfully for message with ID [$messageId]" )
    }

    companion object {
        private val TAG = HttpSmsApiService::class.simpleName
    }
}
