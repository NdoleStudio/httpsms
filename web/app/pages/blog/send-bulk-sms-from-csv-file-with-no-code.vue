<script setup lang="ts">
definePageMeta({ layout: "website" });

useHead({
  title: "Send Bulk SMS from CSV File with No Code - httpSMS",
});
</script>

<template>
  <VContainer>
    <VRow>
      <VCol cols="12" md="8" offset-md="2">
        <h1 class="text-display-small mb-2">
          Send Bulk SMS from CSV File with No Code
        </h1>
        <BlogInfo date="October 29, 2023" read-time="7 min read" />
        <VDivider class="my-6" />

        <p class="text-body-large mb-6">
          Send personalized SMS messages to your users in bulk using your
          Android phone. The good news is, you don't have to write a single
          piece of code, just upload your CSV file and we will take care of the
          rest.
        </p>

        <h2 class="text-headline-medium mb-4">What is a CSV file?</h2>
        <p class="text-body-large mb-6">
          CSV stands for Comma-Separated Values. It is a simple file format
          where each row is on a new line and each column is separated by a
          comma. CSV files can be opened and edited in tools like Microsoft
          Excel, OpenOffice, Google Sheets, or even a plain text editor.
        </p>

        <h2 class="text-headline-medium mb-4">Prerequisites</h2>
        <ul class="text-body-large pl-6 mb-6">
          <li>Basic understanding of CSV files</li>
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

        <h2 class="text-headline-medium mb-4">Step 3: Edit your CSV file</h2>
        <p class="text-body-large mb-6">
          Download the httpSMS CSV template from
          <a
            href="https://httpsms.com/templates/httpsms-bulk.csv"
            target="_blank"
            rel="noopener"
            >https://httpsms.com/templates/httpsms-bulk.csv</a
          >
          and edit it with spreadsheet software or a text editor. Fill in the
          <strong>FromPhoneNumber</strong>, <strong>ToPhoneNumber</strong>, and
          <strong>Content</strong> columns.
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
          and upload your CSV file.
        </p>

        <p class="text-body-large mb-6">
          Don't hesitate to contact us if you face any issues sending bulk SMS
          messages from your CSV files by following this tutorial. Until the
          next time ✌️
        </p>

        <BlogAuthorBio />
      </VCol>
    </VRow>
  </VContainer>
</template>
