package com.httpsms

import android.app.Activity
import android.content.Context
import android.content.Intent
import android.content.BroadcastReceiver
import android.util.Log


internal class DeliveredReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        when (resultCode) {
            Activity.RESULT_OK -> Log.i(TAG, "delivered message with intent [${intent.extras?.getString(Constants.KEY_MESSAGE_ID)}]")
            Activity.RESULT_CANCELED -> Log.e(TAG, "message not delivered [${intent.getStringExtra(Constants.KEY_MESSAGE_ID)}}]")
        }
    }

    companion object {
        private val TAG = DeliveredReceiver::class.simpleName
    }
}
