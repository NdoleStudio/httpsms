<script setup lang="ts">
definePageMeta({ layout: "website" });

useHead({
  title: "Forward Incoming SMS from Phone to Webhook - httpSMS",
});
</script>

<template>
  <VContainer>
    <VRow>
      <VCol cols="12" md="8" offset-md="2">
        <h1 class="text-display-small mb-2">
          Forward Incoming SMS from Phone to Webhook
        </h1>
        <BlogInfo date="April 08, 2023" read-time="5 min read" />
        <VDivider class="my-6" />

        <p class="text-body-large mb-6">
          You can now program your android phone to forward messages received on
          your phone to your server and trigger powerful automations with tools
          like Zapier and IFTTT. I created an open source application called
          httpSMS that helps you to set this up with ease.
        </p>

        <h2 class="text-headline-medium mb-4">Step 1: Get your API_KEY</h2>
        <p class="text-body-large mb-6">
          Create an account on the httpSMS web app and copy your API key from
          the
          <a href="https://httpsms.com/settings" target="_blank" rel="noopener"
            >settings page</a
          >.
        </p>

        <h2 class="text-headline-medium mb-4">
          Step 2: Install the httpSMS android app
        </h2>
        <p class="text-body-large mb-6">
          Download the Android app from
          <a
            href="https://github.com/NdoleStudio/httpsms/releases/latest/download/HttpSms.apk"
            target="_blank"
            rel="noopener"
            >https://github.com/NdoleStudio/httpsms/releases/latest/download/HttpSms.apk</a
          >
          and sign in using your API KEY.
        </p>

        <h2 class="text-headline-medium mb-4">Step 3: Set up a webhook</h2>
        <p class="text-body-large mb-6">
          Once installed, the app listens for SMS messages received on your
          phone. Configure your webhook URL under the
          <a href="https://httpsms.com/settings" target="_blank" rel="noopener"
            >settings page</a
          >
          so the messages can be delivered to your server.
        </p>

        <p class="text-body-large mb-6">
          Congratulations, you have successfully set up SMS forwarding from your
          Android phone to a webhook! This powerful automation tool can help you
          streamline your business workflow and save you time and effort. You
          can also trigger the httpSMS application to send an SMS a simple API.
          You can find more information on the documentation page at
          <a href="https://docs.httpsms.com" target="_blank" rel="noopener"
            >https://docs.httpsms.com</a
          >. Until the next time ✌️
        </p>

        <BlogAuthorBio />
      </VCol>
    </VRow>
  </VContainer>
</template>
