<script setup lang="ts">
definePageMeta({ layout: "website" });
useHead({ title: "Send SMS from Android Phone with Python - httpSMS" });
</script>

<template>
  <VContainer>
    <VRow>
      <VCol cols="12" md="8" offset-md="2">
        <h1 class="text-h3 mb-2">Send SMS from Android Phone with Python</h1>
        <BlogInfo date="April 18, 2023" read-time="5 min read" />
        <VDivider class="my-6" />
        <p class="text-body-1 mb-4">
          Use Python and the httpSMS API to send SMS messages through your
          Android phone.
        </p>
        <h2 class="text-h5 mt-6 mb-3">Prerequisites</h2>
        <ul class="ml-6 mb-4">
          <li class="mb-2">Python 3.7+</li>
          <li class="mb-2">httpSMS account with API key</li>
          <li class="mb-2">Android phone with httpSMS app installed</li>
        </ul>
        <h2 class="text-h5 mt-6 mb-3">Code Example</h2>
        <pre class="pa-4 bg-surface-variant rounded mb-4"><code>import requests
import json

api_key = "your-api-key"
url = 'https://api.httpsms.com/v1/messages/send'

headers = {
    'x-api-key': api_key,
    'Accept': 'application/json',
    'Content-Type': 'application/json'
}

payload = {
    "content": "Hello from Python!",
    "from": "+18005550199",
    "to": "+18005550100"
}

response = requests.post(url, headers=headers, data=json.dumps(payload))
print(response.json())</code></pre>
        <BlogAuthorBio />
      </VCol>
    </VRow>
  </VContainer>
</template>
