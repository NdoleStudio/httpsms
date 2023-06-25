// To parse the JSON, install Klaxon and do:
//
//   val welcome4 = Welcome4.fromJson(jsonString)

package com.httpsms

import com.beust.klaxon.Json
import com.beust.klaxon.Klaxon

private val klaxon = Klaxon()

data class ResponseMessage (
    val data: Message,
    val message: String,
    val status: String
) {
    companion object {
        fun fromJson(json: String) = klaxon.parse<ResponseMessage>(json)
    }
}
data class ResponsePhone (
    val data: Phone,
    val message: String,
    val status: String,
) {
    companion object {
        fun fromJson(json: String) = klaxon.parse<ResponsePhone>(json)
    }
}

data class Phone (
    val id: String,

    @Json(name = "user_id")
    val userID: String,
)

data class Message (
    val contact: String,
    val content: String,
    val sim: String,

    @Json(name = "created_at")
    val createdAt: String,

    @Json(name = "failure_reason")
    val failureReason: String?,

    val id: String,

    @Json(name = "last_attempted_at")
    val lastAttemptedAt: String?,

    @Json(name = "order_timestamp")
    val orderTimestamp: String,

    val owner: String,

    @Json(name = "received_at")
    val receivedAt: String?,

    @Json(name = "request_received_at")
    val requestReceivedAt: String,

    @Json(name = "send_time")
    val sendTime: Long?,

    @Json(name = "sent_at")
    val sentAt: String?,

    val status: String,
    val type: String,

    @Json(name = "updated_at")
    val updatedAt: String
)
