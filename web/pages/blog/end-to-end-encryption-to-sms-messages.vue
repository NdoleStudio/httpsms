<template>
  <v-container class="pt-8">
    <v-row class="mt-16">
      <v-col cols="12" md="9">
        <h1
          class="mt-1"
          :class="{
            'text-h2': $vuetify.breakpoint.mdAndUp,
            'text-h3': !$vuetify.breakpoint.mdAndUp,
          }"
        >
          Secure your conversations by encrypting your SMS messages end-to-end
        </h1>
        <p class="subtitle-2 mt-2">
          <span class="text-uppercase blue--text">{{ postDate }}</span>
          • <span class="text-uppercase">{{ readTime }}</span>
        </p>
        <p class="text--secondary subtitle-1 mt-2">
          We have added support for end-to-end encryption for SMS messages so
          that no one can see the content of the messages you send using httpSMS
          except you.
        </p>
        <p>
          The way it works is that you set up an encryption key which you use to
          encrypt your messages before making an API request to httpSMS and you
          also use the same key to decrypt the messages you receive from httpSMS
          via our
          <a href="https://docs.httpsms.com/webhooks/introduction"
            >webhook events</a
          >. We are using the
          <a href="https://en.wikipedia.org/wiki/Advanced_Encryption_Standard"
            >AES 265</a
          >
          encryption algorithm to encrypt and decrypt the messages.
        </p>
        <h3 class="text-h4 mt-8 mb-2">Setup your encryption key</h3>
        <p>
          <a
            target="_blank"
            class="text-decoration-none"
            href="https://github.com/NdoleStudio/httpsms/releases/latest/download/HttpSms.apk"
            >⬇️ Download and install</a
          >
          the httpSMS Android app on your phone and set you encryption key under
          the <b>App Settings</b> page of the app.
        </p>
        <v-img
          style="border-radius: 4px"
          alt="httpsms android app"
          height="800"
          contain
          :src="
            require('@/static/img/blog/end-to-end-encryption-to-sms-messages/encryption-key-android.png')
          "
        ></v-img>
        <h3 class="text-h4 mb-4 mt-16">Encrypt your SMS message</h3>
        <p>
          We use the AES-265 encryption algorithm to encrypt the SMS messages.
          This algorithm requires a an encryption key which is 256 bits to work
          around this, we will hash any encryption key you set on the mobile app
          using the sha-265 algorithm so that it will always produce a key which
          is 256 bits.
        </p>
        <p>
          The AES algorithm also has an initialization vector (IV) parameter
          which is used to ensure that the same value encrypted multiple times
          will not produce the same encrypted value. The IV is 16 bits and it is
          appended to the encrypted message before encoding it in base64.
        </p>
        <p>
          When you use our client libraries it will automatically take care of
          encrypting your message so you don't have to deal with creating the
          initialization vector and encoding the payload yourself.
        </p>
        <v-tabs v-model="selectedTab" show-arrows>
          <v-tab href="#go">
            <v-icon color="#efd81d" class="mr-1">{{ mdiLanguageGo }}</v-icon>
            Go
          </v-tab>
        </v-tabs>
        <v-tabs-items v-model="selectedTab">
          <v-tab-item value="go">
            <pre v-highlight class="go w-full mb-n12">
<code>import "github.com/NdoleStudio/httpsms-go"

client := htpsms.New(htpsms.WithAPIKey(/* API Key from https://httpsms.com/settings */))

key := "Password123" // use the same key on the Android app
encryptedMessage := client.Cipher.Encrypt(key, "This is a test text message")

// The encrypted message looks like this
// Qk3XGN5+Ax38Ig01m4AqaP6Y0b0wYpCXtx59sU23uVLWUU/c7axF7LozDg==
</code>
        </pre>
          </v-tab-item>
        </v-tabs-items>
        <h3 class="text-h4 mt-6">Send an encrypted message</h3>
        <p>
          After generating the encrypted message payload, you can send it
          directly using the httpSMS API. Make sure to set
          <code>encrypted: true</code> in the JSON request payload so that
          httpSMS knows that the message is encrypted and it will be decoded in
          the Android app before sending to your recipient.
        </p>
        <v-tabs v-model="selectedTab" show-arrows>
          <v-tab href="#go">
            <v-icon color="#efd81d" class="mr-1">{{ mdiLanguageGo }}</v-icon>
            Go
          </v-tab>
        </v-tabs>
        <v-tabs-items v-model="selectedTab">
          <v-tab-item value="go">
            <pre v-highlight class="go w-full mb-n12">
<code>import "github.com/NdoleStudio/httpsms-go"

client.Messages.Send(context.Background(), &httpsms.MessageSendParams{
    Content:   encryptedMessage,
    From:      "+18005550199",
    To:        "+18005550100",
    Encrypted: true,
})
</code>
        </pre>
          </v-tab-item>
        </v-tabs-items>
        <p class="mt-4">
          When you make the API request, the message will be decrypted before
          sending to the recipient. This is a screenshot of the SMS message
          which is sent to the recipient.
        </p>
        <v-img
          style="border-radius: 4px"
          alt="httpsms android app"
          height="800"
          contain
          :src="
            require('@/static/img/blog/end-to-end-encryption-to-sms-messages/send-sms-message.png')
          "
        ></v-img>
        <h3 class="text-h4 mb-4 mt-16">Receiving an encrypted message</h3>
        <p>
          When your android phone receives a new message, it will be encrypted
          with the encryption Key on your Android phone before it is delivered
          to your server's webhook endpoint. You can configure webhooks by
          following
          <a
            href="https://httpsms.com/blog/forward-incoming-sms-from-phone-to-webhook"
            >this guide.</a
          >
        </p>
        <v-tabs v-model="selectedTab" show-arrows>
          <v-tab href="#go">
            <v-icon color="#efd81d" class="mr-1">{{ mdiLanguageGo }}</v-icon>
            Go
          </v-tab>
        </v-tabs>
        <v-tabs-items v-model="selectedTab">
          <v-tab-item value="go">
            <pre v-highlight class="go w-full mb-n12">
<code>import "github.com/NdoleStudio/httpsms-go"

client := htpsms.New(htpsms.WithAPIKey(/* API Key from https://httpsms.com/settings */))

// The payload in the webhook HTTP request looks like this
/*
{
  "specversion": "1.0",
  "id": "8dca3b0a-446a-4a5d-8d2a-95314926c4ed",
  "source": "/v1/messages/receive",
  "type": "message.phone.received",
  "datacontenttype": "application/json",
  "time": "2024-01-21T12:27:29.1605708Z",
  "data": {
    "message_id": "0681b838-4157-44bb-a4ea-721e40ee7ca7",
    "user_id": "XtABz6zdeFMoBLoltz6SREDvRSh2",
    "owner": "+37253920216",
    "encrypted": true,
    "contact": "+37253920216",
    "timestamp": "2024-01-21T12:27:17.949Z",
    "content": "bdmZ7n6JVf/ST+SoNlSaOGUL1DcL5705ETw8GAB4llYBgE9HOOL+Pu/h+w==",
    "sim": "SIM1"
  }
}
*/

encryptedMessage = "bdmZ7n6JVf/ST+SoNlSaOGUL1DcL5705ETw8GAB4llYBgE9HOOL+Pu/h+w==" // get the encrypted message from the request payload
key := "Password123" // use the same key on the Android app
decryptedMessage := client.Cipher.Decrypt(key, encryptedMessage)

// This is a test text message
</code>
        </pre>
          </v-tab-item>
        </v-tabs-items>
        <h3 class="text-h4 mt-12">Conclusion</h3>
        <p>
          Congratulations, you have successfully configured your Android phone
          to send and receive SMS messages with end-to-end encryption. Don't
          hesitate to contact us if you face any problems while following this
          guide.
        </p>
        <blog-author-bio></blog-author-bio>
        <v-divider class="mx-16"></v-divider>
        <div class="text-center mt-8 mb-4">
          <back-button></back-button>
        </div>
      </v-col>
      <v-col v-if="$vuetify.breakpoint.mdAndUp" md="3">
        <blog-info></blog-info>
      </v-col>
    </v-row>
  </v-container>
</template>

<script lang="ts">
import { mdiLanguageGo, mdiTwitter } from '@mdi/js'
export default {
  name: 'EndToEndEncryptionToSmsMessages',
  layout: 'website',
  data() {
    return {
      mdiTwitter,
      mdiLanguageGo,
      selectedTab: 'go',
      authorImage: require('@/assets/img/arnold.png'),
      authorName: 'Acho Arnold',
      postDate: 'January 21, 2024',
      readTime: '10 min read',
      authorTwitter: 'acho_arnold',
    }
  },
  head() {
    return {
      title:
        'Secure your conversations with end-to-end encryption for SMS messages - httpSMS',
      meta: [
        {
          hid: 'og:title',
          property: 'og:title',
          content:
            'Secure your conversations with end-to-end encryption for SMS messages',
        },
        {
          hid: 'og:description',
          property: 'og:description',
          content:
            'Configure your Android phone as an SMS gateway to automate sending text messages with the Python programing language.',
        },
      ],
    }
  },
}
</script>
