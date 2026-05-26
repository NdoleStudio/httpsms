<script setup lang="ts">
definePageMeta({ layout: "website" });

useHead({
  title: "Send SMS from Android Phone with Python - httpSMS",
});
</script>

<template>
  <VContainer>
    <VRow>
      <VCol cols="12" md="8" offset-md="2">
        <h1 class="text-display-small mb-2">
          Send SMS from Android Phone with Python
        </h1>
        <BlogInfo date="June 03, 2023" read-time="6 min read" />
        <VDivider class="my-6" />

        <p class="text-body-large mb-6">
          In an era dominated by social media, instant messaging apps, and
          ever-evolving communication technologies, it's easy to overlook the
          humble yet remarkably resilient Short Message Service (SMS). Since its
          inception in the 1990s, SMS has stood the test of time, remaining one
          of the most widely used and reliable means of mobile communication.
          Whether you're a business owner looking to optimize your communication
          strategy, a developer seeking to integrate SMS functionality into your
          applications, or simply intrigued by the enduring charm of SMS, this
          article will explain how to setup your Android phone to send SMS
          messages.
        </p>

        <h2 class="text-headline-medium mb-4">Prerequisites</h2>
        <ul class="text-body-large pl-6 mb-6">
          <li>Basic understanding of Python</li>
          <li>An Android phone</li>
          <li>Python installed on your computer</li>
        </ul>

        <h2 class="text-headline-medium mb-4">Step 1: Get your API Key</h2>
        <p class="text-body-large mb-6">
          Create an account on
          <a href="https://httpsms.com" target="_blank" rel="noopener"
            >httpsms.com</a
          >
          and copy your API key from the
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

        <h2 class="text-headline-medium mb-4">Step 3: Writing the code</h2>
        <pre
          class="pa-4 mb-6 rounded bg-surface-variant overflow-x-auto"
        ><code class="language-python text-body-medium">import requests
import json

api_key = "" # Get API Key from https://httpsms.com/settings

url = 'https://api.httpsms.com/v1/messages/send'

headers = {
    'x-api-key': api_key,
    'Accept': 'application/json',
    'Content-Type': 'application/json'
}

payload = {
    "content": "This is a sample text message sent via python",
    "from": "+18005550199", # This is the phone number of your android phone
    "to": "+18005550100" # This is the recipient phone number
}

response = requests.post(url, headers=headers, data=json.dumps(payload))

print(json.dumps(response.json(), indent=4))</code></pre>
        <p class="text-body-large mb-6">
          Run the script with <code>python send_sms.py</code>.
        </p>

        <p class="text-body-large mb-6">
          Congratulations, you have successfully configured your android phone
          to send SMS messages via python. You can now reuse this code to send
          SMS messages from your python applications. If you are also interested
          in forwarding incoming SMS from your android phone to your server,
          checkout our SMS forwarding guide
          <NuxtLink to="/blog/forward-incoming-sms-from-phone-to-webhook"
            >here</NuxtLink
          >. Until the next time ✌️
        </p>

        <BlogAuthorBio />
      </VCol>
    </VRow>
  </VContainer>
</template>
