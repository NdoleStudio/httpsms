<script setup lang="ts">
definePageMeta({ layout: "website" });
useHead({ title: "Forward Incoming SMS from Phone to Webhook - httpSMS" });
</script>

<template>
  <VContainer>
    <VRow>
      <VCol cols="12" md="8" offset-md="2">
        <h1 class="text-h3 mb-2">Forward Incoming SMS from Phone to Webhook</h1>
        <BlogInfo date="August 20, 2023" read-time="4 min read" />
        <VDivider class="my-6" />
        <p class="text-body-1 mb-4">
          Configure httpSMS to automatically forward incoming SMS messages to
          your server via webhooks. This allows you to build powerful automation
          workflows.
        </p>
        <h2 class="text-h5 mt-6 mb-3">Setting Up a Webhook</h2>
        <ol class="ml-6 mb-4">
          <li class="mb-2">Log in to your httpSMS dashboard</li>
          <li class="mb-2">Navigate to Settings → Webhooks</li>
          <li class="mb-2">Click "Add Webhook"</li>
          <li class="mb-2">Enter your server's callback URL</li>
          <li class="mb-2">Select the "message.phone.received" event</li>
          <li class="mb-2">Save the webhook</li>
        </ol>
        <p class="text-body-1 mb-4">
          When an SMS is received on your phone, httpSMS will send a POST
          request to your callback URL with the message details in JSON format.
        </p>
        <BlogAuthorBio />
      </VCol>
    </VRow>
  </VContainer>
</template>
