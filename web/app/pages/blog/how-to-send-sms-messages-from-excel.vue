<script setup lang="ts">
definePageMeta({ layout: "website" });
useHead({ title: "How to Send SMS Messages from Excel - httpSMS" });
</script>

<template>
  <VContainer>
    <VRow>
      <VCol cols="12" md="8" offset-md="2">
        <h1 class="text-h3 mb-2">How to Send SMS Messages from Excel</h1>
        <BlogInfo date="June 5, 2023" read-time="4 min read" />
        <VDivider class="my-6" />
        <p class="text-body-1 mb-4">
          Send bulk SMS messages to multiple recipients using a simple Excel
          template. No coding required!
        </p>
        <h2 class="text-h5 mt-6 mb-3">Steps</h2>
        <ol class="ml-6 mb-4">
          <li class="mb-2">
            Download the
            <a href="/templates/httpsms-bulk.xlsx">Excel template</a>
          </li>
          <li class="mb-2">Fill in the phone numbers and messages</li>
          <li class="mb-2">Go to Bulk Messages in your httpSMS dashboard</li>
          <li class="mb-2">Upload the file and click Send</li>
        </ol>
        <p class="text-body-1 mb-4">
          The template has two columns: "to" (phone number in international
          format) and "content" (the message text). You can send to up to 1,000
          recipients per file.
        </p>
        <BlogAuthorBio />
      </VCol>
    </VRow>
  </VContainer>
</template>
