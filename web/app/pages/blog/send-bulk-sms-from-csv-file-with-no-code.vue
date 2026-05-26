<script setup lang="ts">
definePageMeta({ layout: "website" });
useHead({ title: "Send Bulk SMS from CSV File with No Code - httpSMS" });
</script>

<template>
  <VContainer>
    <VRow>
      <VCol cols="12" md="8" offset-md="2">
        <h1 class="text-h3 mb-2">Send Bulk SMS from CSV File with No Code</h1>
        <BlogInfo date="May 12, 2023" read-time="3 min read" />
        <VDivider class="my-6" />
        <p class="text-body-1 mb-4">
          Upload a simple CSV file to send SMS messages to hundreds of
          recipients at once.
        </p>
        <h2 class="text-h5 mt-6 mb-3">CSV Format</h2>
        <p class="text-body-1 mb-4">
          Your CSV file should have two columns: <code>to</code> and
          <code>content</code>. Download our
          <a href="/templates/httpsms-bulk.csv">CSV template</a> to get started.
        </p>
        <h2 class="text-h5 mt-6 mb-3">Upload and Send</h2>
        <ol class="ml-6 mb-4">
          <li class="mb-2">Navigate to the Bulk Messages page</li>
          <li class="mb-2">Click the file upload field</li>
          <li class="mb-2">Select your CSV file</li>
          <li class="mb-2">Click "Send Bulk Messages"</li>
        </ol>
        <BlogAuthorBio />
      </VCol>
    </VRow>
  </VContainer>
</template>
