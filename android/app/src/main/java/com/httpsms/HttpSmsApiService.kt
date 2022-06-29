package com.httpsms

import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import timber.log.Timber
import java.net.URI
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter
import java.util.logging.Level
import java.util.logging.Logger.getLogger


class HttpSmsApiService(private val apiKey: String) {
    private val apiKeyHeader = "X-API-KEY"
    private val baseURL = URI("https://api.httpsms.com")
    private val jsonMediaType = "application/json; charset=utf-8".toMediaType()

    init {
        getLogger(OkHttpClient::class.java.name).level = Level.FINE
    }

    fun getOutstandingMessages(owner: String): List<Message> {
        val client = OkHttpClient()

        val request: Request = Request.Builder()
            .url(baseURL.resolve("/v1/messages/outstanding?owner=${owner}").toURL())
            .header(apiKeyHeader, apiKey)
            .build()

        val response = client.newCall(request).execute()
        if (response.isSuccessful) {
            val payload =  ResponseMessagesOutstanding.fromJson(response.body!!.string())?.data
            if (payload == null) {
                Timber.e("cannot decode payload [${response.body}]")
                return listOf()
            }
            response.close()
            return payload
        }

        Timber.e("invalid response with code [${response.code}] and payload [${response.body}]")
        response.close()
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

    fun receive(from: String, to: String, content: String, timestamp: ZonedDateTime) {
        val client = OkHttpClient()

        val formatter  = DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm:ss.SSS'000000'ZZZZZ")
        val timestampString = formatter.format(timestamp).replace("+", "Z")

        val body = """
            {
              "content": "$content",
              "from": "$from",
              "timestamp": "$timestampString",
              "to": "$to"
            }
        """.trimIndent()

        val request: Request = Request.Builder()
            .url(baseURL.resolve("/v1/messages/receive").toURL())
            .post(body.toRequestBody(jsonMediaType))
            .header(apiKeyHeader, apiKey)
            .build()

        val response = client.newCall(request).execute()
        if (!response.isSuccessful) {
            Timber.e("error response [${response.body?.string()}] with code [${response.code}] while receiving message [${body}]}]")
            return
        }

        val message = ResponseMessage.fromJson(response.body!!.string())
        response.close()
        Timber.i("received message stored successfully for message with ID [${message?.data?.id}]" )
    }


    private fun sendEvent(messageId: String, event: String, timestamp: ZonedDateTime, reason: String? = null) {
        val client = OkHttpClient()

        val formatter  = DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm:ss.SSS'000000'ZZZZZ")
        val timestampString = formatter.format(timestamp).replace("+", "Z")

        val body = """
            {
              "event_name": "$event",
              "reason": "$reason"
              "timestamp": "$timestampString"
            }
        """.trimIndent()

        val request: Request = Request.Builder()
            .url(baseURL.resolve("/v1/messages/${messageId}/events").toURL())
            .post(body.toRequestBody(jsonMediaType))
            .header(apiKeyHeader, apiKey)
            .build()

        val response = client.newCall(request).execute()
        if (!response.isSuccessful) {
           Timber.e("error response [${response.body?.string()}] with code [${response.code}] while sending [${event}] event [${body}] for message with ID [${messageId}]")
            return
        }

        response.close()
        Timber.i( "[$event] event sent successfully for message with ID [$messageId]" )
    }


    fun updatePhone(phoneNumber: String, fcmToken: String) {
        val client = OkHttpClient()

        val body = """
            {
              "fcm_token": "$fcmToken",
              "phone_number": "$phoneNumber"
            }
        """.trimIndent()

        val request: Request = Request.Builder()
            .url(baseURL.resolve("/v1/phones").toURL())
            .put(body.toRequestBody(jsonMediaType))
            .header(apiKeyHeader, apiKey)
            .build()

        val response = client.newCall(request).execute()
        if (!response.isSuccessful) {
            Timber.e("error response [${response.body?.string()}] with code [${response.code}] while sending fcm token [${body}]")
            return
        }

        response.close()
        Timber.i("fcm token sent successfully for phone [$phoneNumber]" )
    }


    fun validateApiKey(): String? {
        val client = OkHttpClient()

        val request: Request = Request.Builder()
            .url(baseURL.resolve("/v1/users/me").toURL())
            .header(apiKeyHeader, apiKey)
            .get()
            .build()

        val response = client.newCall(request).execute()
        if (!response.isSuccessful) {
            Timber.e("error response [${response.body?.string()}] with code [${response.code}] while verifying apiKey [$apiKey]")
            return "Cannot validate the API key. Check if it is correct and try again."
        }

        response.close()
        Timber.i("api key [$apiKey] is valid" )
        return null
    }
}
