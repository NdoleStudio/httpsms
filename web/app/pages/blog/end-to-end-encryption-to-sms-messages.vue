<script setup lang="ts">
import { useDisplay } from 'vuetify'

import { mdiLanguageGo, mdiLanguageJavascript } from '@mdi/js'
import { ref } from 'vue'

const { mdAndUp } = useDisplay()

const encryptTab = ref('javascript')
const sendTab = ref('javascript')
const receiveTab = ref('javascript')

definePageMeta({ layout: 'website' })

useSeoMeta({
  title:
    'Secure your conversations with end-to-end encryption for SMS messages - httpSMS',
  description:
    'Enable end-to-end encryption for your SMS messages with httpSMS so no one but you can read the texts you send and receive through your Android phone.',
  ogTitle:
    'Secure your conversations with end-to-end encryption for SMS messages',
  ogDescription:
    'We have added support for end-to-end encryption for SMS messages so that no one can see the content of the messages you send using httpSMS except you.',
  ogImage: 'https://httpsms.com/header.png',
  twitterCard: 'summary_large_image',
})
</script>

<template>
  <VContainer v-highlight class="pt-8">
    <VRow :class="{ 'mt-16': mdAndUp }">
      <VCol cols="12" md="9">
        <h1
          :class="
            mdAndUp ? 'text-display-medium mt-1' : 'text-display-small mt-n2'
          "
        >
          Secure your conversations by encrypting your SMS messages end-to-end
        </h1>
        <BlogInfo date="January 21, 2024" read-time="10 min read" />

        <p class="text-body-large mt-2">
          We have added support for end-to-end encryption for SMS messages so
          that no one can see the content of the messages you send using httpSMS
          except you.
        </p>
        <p>
          You setup an encryption key which you use to encrypt your messages
          before making an API request to httpSMS and you also use the same key
          to decrypt the messages you receive from httpSMS via our
          <a
            class="text-decoration-none"
            href="https://docs.httpsms.com/webhooks/introduction"
            >webhook events</a
          >. We are using the
          <a
            class="text-decoration-none"
            href="https://en.wikipedia.org/wiki/Advanced_Encryption_Standard"
            >AES 256</a
          >
          encryption algorithm to encrypt and decrypt the messages.
        </p>

        <h3 class="text-headline-large mt-8 mb-2">Setup your encryption key</h3>
        <p>
          <a
            class="text-decoration-none"
            href="https://github.com/NdoleStudio/httpsms/releases/latest/download/HttpSms.apk"
            >⬇️ Download and install</a
          >
          the httpSMS Android app on your phone and set you encryption key under
          the <b>App Settings</b> page of the app.
        </p>
        <VImg
          style="border-radius: 4px"
          alt="httpsms android app"
          height="800"
          src="/img/blog/end-to-end-encryption-to-sms-messages/encryption-key-android.png"
        />

        <h3 class="text-headline-large mb-4 mt-16">Encrypt your SMS message</h3>
        <p>
          We use the AES-256 encryption algorithm to encrypt the SMS messages.
          This algorithm requires a an encryption key which is 256 bits to work
          around this, we will hash any encryption key you set on the mobile app
          using the sha-256 algorithm so that it will always produce a key which
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

        <VTabs v-model="encryptTab" show-arrows>
          <VTab value="javascript">
            <VIcon color="#efd81d" class="mr-1" :icon="mdiLanguageJavascript" />
            Javascript
          </VTab>
          <VTab value="go">
            <VIcon color="#00aed8" class="mr-1" :icon="mdiLanguageGo" />
            Go
          </VTab>
        </VTabs>
        <VTabsWindow v-model="encryptTab">
          <VTabsWindowItem value="javascript">
            <pre
              class="pa-4 mb-6 rounded bg-surface overflow-x-auto"
            ><code class="language-javascript text-body-medium">import HttpSms from "httpsms"

const client = new HttpSms("" /* API Key from https://httpsms.com/settings */);

const key = "Password123";

const encryptedMessage = client.cipher.encrypt(key, "This is a sample text message");

// The encrypted message looks like this, note that you will get a different encrypted message when you run this code on your computer
// Qk3XGN5+Ax38Ig01m4AqaP6Y0b0wYpCXtx59sU23uVLWUU/c7axF7LozDg==</code></pre>
          </VTabsWindowItem>
          <VTabsWindowItem value="go">
            <pre
              class="pa-4 mb-6 rounded bg-surface overflow-x-auto"
            ><code class="language-go text-body-medium">import "github.com/NdoleStudio/httpsms-go"

client := htpsms.New(htpsms.WithAPIKey(""/* API Key from https://httpsms.com/settings */))

key := "Password123" // use the same key on the Android app
encryptedMessage := client.Cipher.Encrypt(key, "This is a test text message")

// The encrypted message looks like this, note that you will get a different encrypted message when you run this code on your computer
// Qk3XGN5+Ax38Ig01m4AqaP6Y0b0wYpCXtx59sU23uVLWUU/c7axF7LozDg==</code></pre>
          </VTabsWindowItem>
        </VTabsWindow>

        <h3 class="text-headline-large mt-6">Send an encrypted message</h3>
        <p>
          After generating the encrypted message payload, you can send it
          directly using the httpSMS API. Make sure to set
          <code>encrypted: true</code> in the JSON request payload so that
          httpSMS knows that the message is encrypted and it will be decoded in
          the Android app before sending to your recipient.
        </p>

        <VTabs v-model="sendTab" show-arrows>
          <VTab value="javascript">
            <VIcon color="#efd81d" class="mr-1" :icon="mdiLanguageJavascript" />
            Javascript
          </VTab>
          <VTab value="go">
            <VIcon color="#00aed8" class="mr-1" :icon="mdiLanguageGo" />
            Go
          </VTab>
        </VTabs>
        <VTabsWindow v-model="sendTab">
          <VTabsWindowItem value="javascript">
            <pre
              class="pa-4 mb-6 rounded bg-surface overflow-x-auto"
            ><code class="language-javascript text-body-medium">import HttpSms from "httpsms"

client.messages.postSend({
    content:   encryptedMessage,
    from:      '+18005550199',
    encrypted: true,
    to:        '+18005550100',
})
.then((message) =&gt; {
    console.log(message.id); // log the ID of the sent message
});</code></pre>
          </VTabsWindowItem>
          <VTabsWindowItem value="go">
            <pre
              class="pa-4 mb-6 rounded bg-surface overflow-x-auto"
            ><code class="language-go text-body-medium">import "github.com/NdoleStudio/httpsms-go"

client.Messages.Send(context.Background(), &amp;httpsms.MessageSendParams{
    Content:   encryptedMessage,
    From:      "+18005550199",
    To:        "+18005550100",
    Encrypted: true,
})</code></pre>
          </VTabsWindowItem>
        </VTabsWindow>

        <p class="mt-4">
          When you make the API request, the message will be decrypted before
          sending to the recipient. This is a screenshot of the SMS message
          which is sent to the recipient.
        </p>
        <VImg
          style="border-radius: 4px"
          alt="httpsms android app"
          height="800"
          src="/img/blog/end-to-end-encryption-to-sms-messages/send-sms-message.png"
        />

        <h3 class="text-headline-large mb-4 mt-16">
          Receiving an encrypted message
        </h3>
        <p>
          When your android phone receives a new message, it will be encrypted
          with the encryption Key on your Android phone before it is delivered
          to your server's webhook endpoint. You can configure webhooks by
          following
          <NuxtLink
            class="text-decoration-none"
            to="/blog/forward-incoming-sms-from-phone-to-webhook"
            >this guide.</NuxtLink
          >
        </p>

        <VTabs v-model="receiveTab" show-arrows>
          <VTab value="javascript">
            <VIcon color="#efd81d" class="mr-1" :icon="mdiLanguageJavascript" />
            Javascript
          </VTab>
          <VTab value="go">
            <VIcon color="#00aed8" class="mr-1" :icon="mdiLanguageGo" />
            Go
          </VTab>
        </VTabs>
        <VTabsWindow v-model="receiveTab">
          <VTabsWindowItem value="javascript">
            <pre
              class="pa-4 mb-6 rounded bg-surface overflow-x-auto"
            ><code class="language-javascript text-body-medium">import HttpSms from "httpsms"

const client = new HttpSms("" /* API Key from https://httpsms.com/settings */);

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

const encryptedMessage = "bdmZ7n6JVf/ST+SoNlSaOGUL1DcL5705ETw8GAB4llYBgE9HOOL+Pu/h+w==" // get the encrypted message from the request payload
const encryptionkey = "Password123" // use the same key on the Android app
const decryptedMessage = client.cipher.decrypt(encryptionkey, encryptedMessage)

// This is a test text message</code></pre>
          </VTabsWindowItem>
          <VTabsWindowItem value="go">
            <pre
              class="pa-4 mb-6 rounded bg-surface overflow-x-auto"
            ><code class="language-go text-body-medium">import "github.com/NdoleStudio/httpsms-go"

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
encryptionkey := "Password123" // use the same key on the Android app
decryptedMessage := client.Cipher.Decrypt(encryptionkey, encryptedMessage)

// This is a test text message</code></pre>
          </VTabsWindowItem>
        </VTabsWindow>

        <h3 class="text-headline-large mt-12">Conclusion</h3>
        <p>
          Congratulations, you have successfully configured your Android phone
          to send and receive SMS messages with end-to-end encryption. Don't
          hesitate to contact us if you face any problems while following this
          guide.
        </p>

        <BlogAuthorBio />
        <VDivider class="mx-16" />
        <div class="text-center mt-8 mb-4">
          <BackButton />
        </div>
      </VCol>
      <VCol v-if="$vuetify.display.mdAndUp" md="3">
        <BlogSidebar />
      </VCol>
    </VRow>
  </VContainer>
</template>
