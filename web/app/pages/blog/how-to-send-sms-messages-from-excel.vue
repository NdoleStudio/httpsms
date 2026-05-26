<script setup lang="ts">
definePageMeta({ layout: "website" });

useHead({
  title: "How to Send SMS Messages from Excel - httpSMS",
});
</script>

<template>
  <VContainer>
    <VRow>
      <VCol cols="12" md="8" offset-md="2">
        <h1 class="text-display-small mb-2">
          How to Send SMS Messages from Excel
        </h1>
        <BlogInfo date="October 29, 2023" read-time="5 min read" />
        <VDivider class="my-6" />

        <p class="text-body-large mb-6">
          Send personalized SMS messages to multiple phone numbers for less than
          $0.002 per SMS message. You can also configure every SMS message in
          your Excel spreadsheet so they are unique for each recipient phone
          number.
        </p>

        <h2 class="text-headline-medium mb-4">Prerequisites</h2>
        <ul class="text-body-large pl-6 mb-6">
          <li>Basic understanding of Microsoft Excel or Google Sheets</li>
          <li>An Android phone</li>
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

        <h2 class="text-headline-medium mb-4">Step 3: Edit your Excel file</h2>
        <p class="text-body-large mb-6">
          Download the httpSMS Excel file template from
          <a
            href="https://httpsms.com/templates/httpsms-bulk.xlsx"
            target="_blank"
            rel="noopener"
            >https://httpsms.com/templates/httpsms-bulk.xlsx</a
          >
          and edit it with Microsoft Excel, Google Sheets, or any spreadsheet
          software. Fill in the <strong>FromPhoneNumber</strong>,
          <strong>ToPhoneNumber</strong>, and <strong>Content</strong> columns.
        </p>

        <h2 class="text-headline-medium mb-4">Send your SMS messages</h2>
        <p class="text-body-large mb-6">
          Visit the
          <a
            href="https://httpsms.com/bulk-messages"
            target="_blank"
            rel="noopener"
            >Bulk Messages page</a
          >
          and upload your Excel file.
        </p>

        <p class="text-body-large mb-6">
          Don't hesitate to contact us if you face any issues sending bulk SMS
          messages from your Excel files by following this tutorial. Until the
          next time ✌️
        </p>

        <BlogAuthorBio />
      </VCol>
    </VRow>
  </VContainer>
</template>
