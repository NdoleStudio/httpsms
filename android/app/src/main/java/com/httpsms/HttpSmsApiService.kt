package com.httpsms

import android.content.Context
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.apache.commons.text.StringEscapeUtils
import timber.log.Timber
import java.net.URI
import java.net.URL
import java.util.logging.Level
import java.util.logging.Logger.getLogger


class HttpSmsApiService(private val apiKey: String, private val baseURL: URI) {
    private val apiKeyHeader = "x-api-key"
    private val clientVersionHeader = "X-Client-Version"
    private val jsonMediaType = "application/json; charset=utf-8".toMediaType()
    private val client = OkHttpClient.Builder().retryOnConnectionFailure(true).build()

    init {
        getLogger(OkHttpClient::class.java.name).level = Level.FINE
    }

    companion object {
        fun create(context: Context): HttpSmsApiService {
            return HttpSmsApiService(
                Settings.getApiKeyOrDefault(context),
                Settings.getServerUrlOrDefault(context)
            )
        }
    }

    fun getOutstandingMessage(messageID: String): Message? {
        val request: Request = Request.Builder()
            .url(resolveURL("/v1/messages/outstanding?message_id=${messageID}"))
            .header(apiKeyHeader, apiKey)
            .header(clientVersionHeader, BuildConfig.VERSION_NAME)
            .build()

        val response = client.newCall(request).execute()
        if (response.isSuccessful) {
            val payload = ResponseMessage.fromJson(response.body!!.string())?.data
            if (payload == null) {
                response.close()
                Timber.e("cannot decode payload [${response.body}]")
                return null
            }
            response.close()
            return payload
        }

        Timber.e("invalid response with code [${response.code}]")
        response.close()
        return null
    }

    fun sendDeliveredEvent(messageId: String, timestamp: String): Boolean {
        return sendEvent(messageId, "DELIVERED", timestamp)
    }

    fun sendSentEvent(messageId: String, timestamp: String): Boolean {
        return sendEvent(messageId, "SENT", timestamp)
    }

    fun sendFailedEvent(messageId: String, timestamp: String, reason: String): Boolean {
        return sendEvent(messageId, "FAILED", timestamp, reason)
    }

    fun receive(sim: String, from: String, to: String, content: String, encrypted: Boolean, timestamp: String): Boolean {
        val body = """
            {
              "content": "${StringEscapeUtils.escapeJson(content)}",
              "sim": "$sim",
              "from": "$from",
              "timestamp": "$timestamp",
              "encrypted": $encrypted,
              "to": "$to"
            }
        """.trimIndent()

        val request: Request = Request.Builder()
            .url(resolveURL("/v1/messages/receive"))
            .post(body.toRequestBody(jsonMediaType))
            .header(apiKeyHeader, apiKey)
            .header(clientVersionHeader, BuildConfig.VERSION_NAME)
            .build()

        val response = client.newCall(request).execute()
        if (!response.isSuccessful) {
            Timber.e("error response [${response.body?.string()}] with code [${response.code}] while receiving message [${body}]")
            response.close()
            return response.code in 400..499
        }

        val message = ResponseMessage.fromJson(response.body!!.string())
        response.close()
        Timber.i("received message stored successfully for message with ID [${message?.data?.id}]" )
        return true
    }

    fun sendMissedCallEvent(sim: String, from: String, to: String, timestamp: String): Boolean {
        val body = """
            {
              "sim": "$sim",
              "from": "$from",
              "timestamp": "$timestamp",
              "to": "$to"
            }
        """.trimIndent()

        val request: Request = Request.Builder()
            .url(resolveURL("/v1/messages/calls/missed"))
            .post(body.toRequestBody(jsonMediaType))
            .header(apiKeyHeader, apiKey)
            .header(clientVersionHeader, BuildConfig.VERSION_NAME)
            .build()

        val response = client.newCall(request).execute()
        if (!response.isSuccessful) {
            Timber.e("error response [${response.body?.string()}] with code [${response.code}] while sending missed call event [${body}]")
            response.close()
            return response.code in 400..499
        }

        response.close()
        Timber.i("missed call from [${from}] to [${to}] sent successfully with timestamp [${timestamp}]" )
        return true
    }

    fun storeHeartbeat(phoneNumber: String, charging: Boolean) {
        val body = """
            {
              "charging": $charging,
              "owner": "$phoneNumber"
            }
        """.trimIndent()

        val request: Request = Request.Builder()
            .url(resolveURL("/v1/heartbeats"))
            .post(body.toRequestBody(jsonMediaType))
            .header(apiKeyHeader, apiKey)
            .header(clientVersionHeader, BuildConfig.VERSION_NAME)
            .build()

        val response = client.newCall(request).execute()
        if (!response.isSuccessful) {
            Timber.e("error response [${response.body?.string()}] with code [${response.code}] while sending heartbeat [$body] for owner [$phoneNumber]")
            response.close()
            return
        }

        response.close()
        Timber.i( "heartbeat stored successfully for owner [$phoneNumber]" )
    }


    private fun sendEvent(messageId: String, event: String, timestamp: String, reason: String? = null): Boolean {
        var reasonString = "null"
        if (reason != null) {
            reasonString = "\"$reason\""
        }

        val body = """
            {
              "event_name": "$event",
              "reason": $reasonString,
              "timestamp": "$timestamp"
            }
        """.trimIndent()

        val request: Request = Request.Builder()
            .url(resolveURL("/v1/messages/${messageId}/events"))
            .post(body.toRequestBody(jsonMediaType))
            .header(apiKeyHeader, apiKey)
            .header(clientVersionHeader, BuildConfig.VERSION_NAME)
            .build()

        val response = client.newCall(request).execute()
        if (response.code == 404) {
            response.close()
            Timber.i( "[$event] event sent successfully but message with ID [$messageId] has been deleted" )
            return true
        }

        if (!response.isSuccessful) {
            Timber.e("error response [${response.body?.string()}] with code [${response.code}] while sending [${event}] event [${body}] for message with ID [${messageId}]")
            response.close()
            return false
        }

        response.close()
        Timber.i( "[$event] event sent successfully for message with ID [$messageId]" )
        return true
    }


    fun updatePhone(phoneNumber: String, fcmToken: String, sim: String): Phone?  {
        val body = """
            {
              "fcm_token": "$fcmToken",
              "phone_number": "$phoneNumber",
              "sim": "$sim"
            }
        """.trimIndent()

        val request: Request = Request.Builder()
            .url(resolveURL("/v1/phones"))
            .put(body.toRequestBody(jsonMediaType))
            .header(apiKeyHeader, apiKey)
            .header(clientVersionHeader, BuildConfig.VERSION_NAME)
            .build()

        val response = client.newCall(request).execute()
        if (!response.isSuccessful) {
            Timber.e("error response [${response.body?.string()}] with code [${response.code}] while sending fcm token [${body}]")
            response.close()
            return null
        }

        val payload = ResponsePhone.fromJson(response.body!!.string())?.data
        response.close()
        Timber.i("fcm token sent successfully for phone [$phoneNumber] and id [${payload?.id}]" )
        return  payload
    }


    fun validateApiKey(): Pair<String?, String?> {
        val request: Request = Request.Builder()
            .url(resolveURL("/v1/users/me"))
            .header(apiKeyHeader, apiKey)
            .header(clientVersionHeader, BuildConfig.VERSION_NAME)
            .get()
            .build()

        try {
            val response = client.newCall(request).execute()
            if (!response.isSuccessful) {
                Timber.e("error response [${response.body?.string()}] with code [${response.code}] while verifying apiKey [$apiKey]")
                response.close()
                return Pair("Cannot validate the API key. Check if it is correct and try again.", null)
            }

            response.close()
            Timber.i("api key [$apiKey] and server url [$baseURL] are valid" )
            return Pair(null, null)
        } catch (ex: Exception) {
            return Pair(null, ex.message)
        }
    }

    private fun resolveURL(path: String): URL {
        return baseURL.resolve(baseURL.path + path).toURL()
    }
}
