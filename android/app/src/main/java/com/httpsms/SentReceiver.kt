package com.httpsms

import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log
import android.telephony.SmsManager

internal class SentReceiver : BroadcastReceiver() {
    companion object {
        private val TAG = SentReceiver::class.simpleName
    }

    override fun onReceive(context: Context, intent: Intent) {
        when (resultCode) {
            Activity.RESULT_OK -> Log.i(TAG, "sending successful with intent [${intent.getStringExtra(Constants.KEY_MESSAGE_ID)}]")
            SmsManager.RESULT_ERROR_GENERIC_FAILURE -> Log.i(TAG, "sending failed with RESULT_ERROR_GENERIC_FAILURE intent [${intent.getStringExtra(Constants.KEY_MESSAGE_ID)}]")
            SmsManager.RESULT_ERROR_NO_SERVICE -> Log.i(TAG, "sending failed with RESULT_ERROR_NO_SERVICE intent [${intent.getStringExtra(Constants.KEY_MESSAGE_ID)}]")
            SmsManager.RESULT_ERROR_NULL_PDU -> Log.i(TAG, "sending failed with RESULT_ERROR_NULL_PDU intent [${intent.getStringExtra(Constants.KEY_MESSAGE_ID)}]")
            SmsManager.RESULT_ERROR_RADIO_OFF -> Log.i(TAG, "sending failed with RESULT_ERROR_RADIO_OFF intent [${intent.getStringExtra(Constants.KEY_MESSAGE_ID)}]")
        }
    }
}
