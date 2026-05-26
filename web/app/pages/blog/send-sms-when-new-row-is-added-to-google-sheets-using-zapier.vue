<script setup lang="ts">
definePageMeta({ layout: "website" });
useHead({
  title:
    "Send SMS When New Row is Added to Google Sheets Using Zapier - httpSMS",
});
</script>

<template>
  <VContainer>
    <VRow>
      <VCol cols="12" md="8" offset-md="2">
        <h1 class="text-h3 mb-2">
          Send SMS When New Row is Added to Google Sheets Using Zapier
        </h1>
        <BlogInfo date="March 22, 2023" read-time="6 min read" />
        <VDivider class="my-6" />
        <p class="text-body-1 mb-4">
          Automate SMS sending by connecting Google Sheets to httpSMS through
          Zapier. Every time a new row is added to your spreadsheet, an SMS will
          be sent automatically.
        </p>
        <h2 class="text-h5 mt-6 mb-3">Setup Steps</h2>
        <ol class="ml-6 mb-4">
          <li class="mb-2">
            Create a Zapier account and connect Google Sheets
          </li>
          <li class="mb-2">Set the trigger to "New Spreadsheet Row"</li>
          <li class="mb-2">Add a Webhooks by Zapier action (POST request)</li>
          <li class="mb-2">
            Configure the URL as
            <code>https://api.httpsms.com/v1/messages/send</code>
          </li>
          <li class="mb-2">Add your API key in the headers</li>
          <li class="mb-2">
            Map the spreadsheet columns to the message fields
          </li>
        </ol>
        <p class="text-body-1 mb-4">
          Now every time a new row is added to your Google Sheet, Zapier will
          automatically send an SMS via httpSMS to the phone number in that row.
        </p>
        <BlogAuthorBio />
      </VCol>
    </VRow>
  </VContainer>
</template>
