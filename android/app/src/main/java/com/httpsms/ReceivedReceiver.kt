package com.httpsms

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.provider.Telephony
import android.util.Base64
import androidx.work.Constraints
import androidx.work.Data
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequest
import androidx.work.WorkManager
import androidx.work.Worker
import androidx.work.WorkerParameters
import androidx.work.workDataOf
import com.google.android.mms.pdu_alt.CharacterSets
import com.google.android.mms.pdu_alt.MultimediaMessagePdu
import com.google.android.mms.pdu_alt.PduParser
import com.google.android.mms.pdu_alt.RetrieveConf
import timber.log.Timber
import java.io.File
import java.io.FileOutputStream
import java.time.ZoneOffset
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter

class ReceivedReceiver: BroadcastReceiver()
{
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action == Telephony.Sms.Intents.SMS_RECEIVED_ACTION) {
            handleSmsReceived(context, intent)
        } else if (intent.action == Telephony.Sms.Intents.WAP_PUSH_RECEIVED_ACTION) {
            handleMmsReceived(context, intent)
        } else {
            Timber.e("received invalid intent with action [${intent.action}]")
        }
    }

    private fun handleSmsReceived(context: Context, intent: Intent) {
        var smsSender = ""
        var smsBody = ""

        for (smsMessage in Telephony.Sms.Intents.getMessagesFromIntent(intent)) {
            smsSender = smsMessage.displayOriginatingAddress
            smsBody += smsMessage.messageBody
        }

        val (sim, owner) = getSimAndOwner(context, intent)

        if (!Settings.isIncomingMessageEnabled(context, sim)) {
            Timber.w("[${sim}] is not active for incoming messages")
            return
        }

        handleMessageReceived(
            context,
            sim,
            smsSender,
            owner,
            smsBody
        )
    }

    private fun handleMmsReceived(context: Context, intent: Intent) {
        val pushData = intent.getByteArrayExtra("data") ?: return
        val pdu = PduParser(pushData, true).parse() ?: return

        if (pdu !is MultimediaMessagePdu) {
            Timber.d("Received PDU is not a MultimediaMessagePdu, ignoring.")
            return
        }

        val from = pdu.from?.string ?: ""
        var content = ""
        val attachmentFiles = mutableListOf<String>()

        // Check if it's a RetrieveConf (which contains the actual message body)
        if (pdu is RetrieveConf) {
            val body = pdu.body
            if (body != null) {
                for (i in 0 until body.partsNum) {
                    val part = body.getPart(i)
                    val partData = part.data ?: continue
                    val contentType = String(part.contentType ?: "application/octet-stream".toByteArray())

                    if (contentType.startsWith("text/plain")) {
                        content += String(partData, charset(CharacterSets.getMimeName(part.charset)))
                    } else {
                        // Save attachment to a temporary file
                        val fileName = String(part.name ?: part.contentLocation ?: part.contentId ?: "attachment_$i".toByteArray())
                        val tempFile = File(context.cacheDir, "received_mms_${System.currentTimeMillis()}_$i")
                        FileOutputStream(tempFile).use { it.write(partData) }
                        attachmentFiles.add("${tempFile.absolutePath}|${contentType}|${fileName}")
                    }
                }
            }
        } else {
            Timber.d("Received PDU is of type [${pdu.javaClass.simpleName}], body extraction not implemented.")
        }

        val (sim, owner) = getSimAndOwner(context, intent)

        if (!Settings.isIncomingMessageEnabled(context, sim)) {
            Timber.w("[${sim}] is not active for incoming messages")
            return
        }

        handleMessageReceived(
            context,
            sim,
            from,
            owner,
            content,
            attachmentFiles.toTypedArray()
        )
    }

    private fun getSimAndOwner(context: Context, intent: Intent): Pair<String, String> {
        var sim = Constants.SIM1
        var owner = Settings.getSIM1PhoneNumber(context)
        if (intent.getIntExtra("android.telephony.extra.SLOT_INDEX", 0) > 0 && Settings.isDualSIM(context)) {
            owner = Settings.getSIM2PhoneNumber(context)
            sim = Constants.SIM2
        }
        return Pair(sim, owner)
    }

    private fun handleMessageReceived(context: Context, sim: String, from: String, to : String, content: String, attachments: Array<String>? = null) {
        val timestamp = ZonedDateTime.now(ZoneOffset.UTC)

        if (!Settings.isLoggedIn(context)) {
            Timber.w("[${sim}] user is not logged in")
            return
        }

        if (!Settings.getActiveStatus(context, sim)) {
            Timber.w("[${sim}] user is not active")
            return
        }

        var body = content;
        if (Settings.encryptReceivedMessages(context)) {
            body = Encrypter.encrypt(Settings.getEncryptionKey(context)!!, content)
        }

        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.CONNECTED)
            .build()

        val inputData: Data = workDataOf(
            Constants.KEY_MESSAGE_FROM to from,
            Constants.KEY_MESSAGE_TO to to,
            Constants.KEY_MESSAGE_SIM to sim,
            Constants.KEY_MESSAGE_CONTENT to body,
            Constants.KEY_MESSAGE_ENCRYPTED to Settings.encryptReceivedMessages(context),
            Constants.KEY_MESSAGE_TIMESTAMP to DateTimeFormatter.ofPattern(Constants.TIMESTAMP_PATTERN).format(timestamp).replace("+", "Z"),
            Constants.KEY_MESSAGE_ATTACHMENTS to attachments
        )

        val work = OneTimeWorkRequest
            .Builder(ReceivedSmsWorker::class.java)
            .setConstraints(constraints)
            .setInputData(inputData)
            .build()

        WorkManager
            .getInstance(context)
            .enqueue(work)

        Timber.d("work enqueued with ID [${work.id}] for received message from [${from}] to [${to}]")
    }

    internal class ReceivedSmsWorker(appContext: Context, workerParams: WorkerParameters) : Worker(appContext, workerParams) {
        override fun doWork(): Result {
            Timber.i("[${this.inputData.getString(Constants.KEY_MESSAGE_SIM)}] forwarding received message from [${this.inputData.getString(Constants.KEY_MESSAGE_FROM)}] to [${this.inputData.getString(Constants.KEY_MESSAGE_TO)}]")

            val sim = this.inputData.getString(Constants.KEY_MESSAGE_SIM)!!
            val from = this.inputData.getString(Constants.KEY_MESSAGE_FROM)!!
            val to = this.inputData.getString(Constants.KEY_MESSAGE_TO)!!
            val content = this.inputData.getString(Constants.KEY_MESSAGE_CONTENT)!!
            val encrypted = this.inputData.getBoolean(Constants.KEY_MESSAGE_ENCRYPTED, false)
            val timestamp = this.inputData.getString(Constants.KEY_MESSAGE_TIMESTAMP)!!

            val attachmentsData = inputData.getStringArray(Constants.KEY_MESSAGE_ATTACHMENTS)
            val attachments = attachmentsData?.mapNotNull {
                val parts = it.split("|")
                val file = File(parts[0])
                if (file.exists()) {
                    val bytes = file.readBytes()
                    val base64Content = Base64.encodeToString(bytes, Base64.NO_WRAP)
                    ReceivedAttachment(
                        name = parts[2],
                        contentType = parts[1],
                        content = base64Content
                    )
                } else {
                    null
                }
            }

            val request = ReceivedMessageRequest(
                sim = sim,
                from = from,
                to = to,
                content = content,
                encrypted = encrypted,
                timestamp = timestamp,
                attachments = attachments
            )

            val success = HttpSmsApiService.create(applicationContext).receive(request)

            // Cleanup temp files
            attachmentsData?.forEach {
                val path = it.split("|")[0]
                val file = File(path)
                if (file.exists()) {
                    file.delete()
                }
            }

            if (success) {
                return Result.success()
            }

            return Result.retry()
        }
    }
}
