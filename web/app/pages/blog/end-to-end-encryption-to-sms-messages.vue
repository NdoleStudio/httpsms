<script setup lang="ts">
definePageMeta({ layout: "website" });

useHead({
  title: "End-to-End Encryption to SMS Messages - httpSMS",
});
</script>

<template>
  <VContainer>
    <VRow>
      <VCol cols="12" md="8" offset-md="2">
        <h1 class="text-display-small mb-2">
          End-to-End Encryption to SMS Messages
        </h1>
        <BlogInfo date="January 21, 2024" read-time="10 min read" />
        <VDivider class="my-6" />

        <p class="text-body-large mb-6">
          We have added support for end-to-end encryption for SMS messages so
          that no one can see the content of the messages you send using httpSMS
          except you. You setup an encryption key which you use to encrypt your
          messages before making an API request to httpSMS and you also use the
          same key to decrypt the messages you receive from httpSMS via our
          webhook events. We are using the AES 256 encryption algorithm to
          encrypt and decrypt the messages.
        </p>

        <h2 class="text-headline-medium mb-4">Setup your encryption key</h2>
        <p class="text-body-large mb-6">
          Download and install the httpSMS Android app on your phone and set
          your encryption key under the App Settings page of the app.
          <a
            href="https://github.com/NdoleStudio/httpsms/releases/latest/download/HttpSms.apk"
            target="_blank"
            rel="noopener"
            >Download the Android app</a
          >.
        </p>

        <h2 class="text-headline-medium mb-4">Encrypt your SMS message</h2>
        <p class="text-body-large mb-4">
          We are using AES-256 to encrypt your SMS messages. The encryption key
          is hashed with SHA-256 before use and the initialization vector (IV)
          is generated for each message. The IV is appended to the encrypted
          payload before the final value is base64 encoded.
        </p>
        <pre
          class="pa-4 mb-6 rounded bg-surface-variant overflow-x-auto"
        ><code class="language-javascript text-body-medium">import HttpSms from "httpsms"

const client = new HttpSms("" /* API Key from https://httpsms.com/settings */)

const key = "Password123"

const encryptedMessage = client.cipher.encrypt(key, "This is a sample text message")

// The encrypted message looks like this
// Qk3XGN5+Ax38Ig01m4AqaP6Y0b0wYpCXtx59sU23uVLWUU/c7axF7LozDg==</code></pre>

        <h2 class="text-headline-medium mb-4">Send an encrypted message</h2>
        <p class="text-body-large mb-4">
          Once you have encrypted the content, send it to the API with
          <code>encrypted: true</code>. This flag tells the Android app to
          decrypt the message before sending it to the recipient.
        </p>
        <pre
          class="pa-4 mb-6 rounded bg-surface-variant overflow-x-auto"
        ><code class="language-javascript text-body-medium">import HttpSms from "httpsms"

client.messages.postSend({
    content:   encryptedMessage,
    from:      '+18005550199',
    encrypted: true,
    to:        '+18005550100',
})
.then((message) =&gt; {
    console.log(message.id) // log the ID of the sent message
})</code></pre>

        <h2 class="text-headline-medium mb-4">
          Receiving an encrypted message
        </h2>
        <p class="text-body-large mb-4">
          When your Android phone receives an SMS message, the app encrypts the
          message with the encryption key configured on the phone before
          delivering it to your webhook endpoint.
        </p>
        <pre
          class="pa-4 mb-6 rounded bg-surface-variant overflow-x-auto"
        ><code class="language-json text-body-medium">{
  "event": "message.phone.received",
  "data": {
    "content": "bdmZ7n6JVf/ST+SoNlSaOGUL1DcL5705ETw8GAB4llYBgE9HOOL+Pu/h+w==",
    "encrypted": true,
    "from": "+18005550100",
    "id": "msg_123",
    "to": "+18005550199"
  }
}</code></pre>
        <pre
          class="pa-4 mb-6 rounded bg-surface-variant overflow-x-auto"
        ><code class="language-javascript text-body-medium">import HttpSms from "httpsms"

const client = new HttpSms("" /* API Key from https://httpsms.com/settings */)

const encryptedMessage = "bdmZ7n6JVf/ST+SoNlSaOGUL1DcL5705ETw8GAB4llYBgE9HOOL+Pu/h+w=="
const encryptionkey = "Password123"
const decryptedMessage = client.cipher.decrypt(encryptionkey, encryptedMessage)

// This is a test text message</code></pre>

        <p class="text-body-large mb-6">
          Congratulations, you have successfully configured your Android phone
          to send and receive SMS messages with end-to-end encryption. Don't
          hesitate to contact us if you face any problems while following this
          guide.
        </p>

        <BlogAuthorBio />
      </VCol>
    </VRow>
  </VContainer>
</template>
