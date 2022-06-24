package com.httpsms

import android.util.Log
import okhttp3.OkHttpClient
import okhttp3.Request;
import java.net.URI


class HttpSmsApiService {
    private val baseURL = URI("https://httpsms.free.beeceptor.com")

    fun getOutstandingMessages(): List<Message> {
        val client = OkHttpClient()

        val request: Request = Request.Builder()
            .url(baseURL.resolve("/v1/messages/outstanding").toURL())
            .build()

        val response = client.newCall(request).execute()
        if (response.isSuccessful) {
            val payload =  MessagesOutstanding.fromJson(response.body!!.string())?.data
            if (payload == null) {
                Log.e(TAG, "cannot decode payload [${response.body}]")
                return listOf();
            }
            return payload
        }

        Log.e(TAG, "invalid response with code [${response.code}] and payload [${response.body}]")
        return listOf()
    }

    companion object {
        private val TAG = HttpSmsApiService::class.simpleName
    }
}
