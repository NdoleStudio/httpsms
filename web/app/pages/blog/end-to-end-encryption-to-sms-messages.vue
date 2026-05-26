<script setup lang="ts">
definePageMeta({ layout: "website" });
useHead({
  title: "How to Add End-to-End Encryption to SMS Messages - httpSMS",
});
</script>

<template>
  <VContainer>
    <VRow>
      <VCol cols="12" md="8" offset-md="2">
        <h1 class="text-h3 mb-2">
          How to Add End-to-End Encryption to SMS Messages
        </h1>
        <BlogInfo date="September 15, 2023" read-time="5 min read" />
        <VDivider class="my-6" />
        <p class="text-body-1 mb-4">
          Privacy is fundamental. With httpSMS, you can add end-to-end
          encryption to your SMS messages using military-grade AES-256
          encryption. This ensures that only you and the intended recipient can
          read the message content.
        </p>
        <h2 class="text-h5 mt-6 mb-3">How It Works</h2>
        <p class="text-body-1 mb-4">
          When you enable encryption in the httpSMS Android app, all messages
          are encrypted before being sent. The encryption key is derived from a
          passphrase that you set in the app settings. Both sender and receiver
          must use the same passphrase.
        </p>
        <h2 class="text-h5 mt-6 mb-3">Setup Steps</h2>
        <ol class="ml-6 mb-4">
          <li class="mb-2">Open the httpSMS app on your Android phone</li>
          <li class="mb-2">Go to Settings → Encryption</li>
          <li class="mb-2">Enable encryption and set your passphrase</li>
          <li class="mb-2">Share the passphrase securely with the recipient</li>
          <li class="mb-2">
            The recipient enables encryption with the same passphrase
          </li>
        </ol>
        <p class="text-body-1 mb-4">
          Messages are encrypted using AES-256 in CFB mode with a SHA-256
          derived key. The encryption happens on-device before the message is
          sent to the httpSMS API.
        </p>
        <BlogAuthorBio />
      </VCol>
    </VRow>
  </VContainer>
</template>
