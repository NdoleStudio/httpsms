<script setup lang="ts">
definePageMeta({ layout: "website" });
useHead({ title: "Grant Send and Read SMS Permissions on Android - httpSMS" });
</script>

<template>
  <VContainer>
    <VRow>
      <VCol cols="12" md="8" offset-md="2">
        <h1 class="text-h3 mb-2">
          Grant Send and Read SMS Permissions on Android
        </h1>
        <BlogInfo date="July 10, 2023" read-time="3 min read" />
        <VDivider class="my-6" />
        <p class="text-body-1 mb-4">
          The httpSMS Android app requires certain permissions to send and
          receive SMS messages. This guide walks you through granting the
          necessary permissions.
        </p>
        <h2 class="text-h5 mt-6 mb-3">Required Permissions</h2>
        <ul class="ml-6 mb-4">
          <li class="mb-2">
            <strong>SEND_SMS</strong> - Required to send SMS messages
          </li>
          <li class="mb-2">
            <strong>READ_SMS</strong> - Required to read incoming messages
          </li>
          <li class="mb-2">
            <strong>RECEIVE_SMS</strong> - Required to detect new messages
          </li>
        </ul>
        <h2 class="text-h5 mt-6 mb-3">How to Grant Permissions</h2>
        <ol class="ml-6 mb-4">
          <li class="mb-2">Open the httpSMS app</li>
          <li class="mb-2">When prompted, tap "Allow" for each permission</li>
          <li class="mb-2">
            If you denied earlier, go to Settings → Apps → httpSMS → Permissions
          </li>
          <li class="mb-2">Enable SMS permissions</li>
        </ol>
        <BlogAuthorBio />
      </VCol>
    </VRow>
  </VContainer>
</template>
