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
        <h1 class="text-display-small mb-2">
          Send SMS When New Row is Added to Google Sheets Using Zapier
        </h1>
        <BlogInfo date="October 29, 2023" read-time="5 min read" />
        <VDivider class="my-6" />

        <p class="text-body-large mb-6">
          Automate sending personalized SMS messages each time a new row is
          added to your Google Sheets document using Zapier. You don't need to
          write any code to make this happen and you can personalize the SMS
          messages which are sent out.
        </p>

        <h2 class="text-headline-medium mb-4">Prerequisites</h2>
        <ul class="text-body-large pl-6 mb-6">
          <li>Basic understanding of Google Sheets</li>
          <li>Basic understanding of Zapier</li>
          <li>
            An account on
            <a href="https://httpsms.com" target="_blank" rel="noopener"
              >httpsms.com</a
            >
          </li>
        </ul>

        <h2 class="text-headline-medium mb-4">
          Step 1: Create trigger on Zapier
        </h2>
        <p class="text-body-large mb-6">
          In Zapier, create a new Zap and select Google Sheets as your trigger
          app. Choose the <strong>New Spreadsheet Row</strong> event, then
          select the Spreadsheet and the correct Worksheet you want to watch for
          new rows.
        </p>

        <h2 class="text-headline-medium mb-4">
          Step 2: Create an action on Zapier
        </h2>
        <p class="text-body-large mb-4">
          Add a new action step and select
          <strong>Webhooks by Zapier</strong> as the action app. Choose the
          <strong>Custom Request</strong> event and configure it with the
          following values:
        </p>
        <ul class="text-body-large pl-6 mb-4">
          <li>Method: Post</li>
          <li>URL: https://api.httpsms.com/v1/messages/send</li>
          <li>Data Pass-Through: false</li>
        </ul>
        <pre
          v-pre
          class="pa-4 mb-6 rounded bg-surface-variant overflow-x-auto"
        ><code class="language-json text-body-medium">{
  "content": "Hi {{1.Name}}, your order is ready for pickup.",
  "from": "{{1.FromPhoneNumber}}",
  "to": "{{1.ToPhoneNumber}}"
}</code></pre>
        <pre
          v-pre
          class="pa-4 mb-6 rounded bg-surface-variant overflow-x-auto"
        ><code class="language-json text-body-medium">{
  "x-api-key": "YOUR_API_KEY",
  "Content-Type": "application/json"
}</code></pre>

        <h2 class="text-headline-medium mb-4">Conclusion</h2>
        <p class="text-body-large mb-6">
          Publish your zap and you will automatically trigger httpsms to send an
          SMS to your customer when ever you add a new row in the google sheet.
          Don't hesitate to contact us if you face any issues configuring your
          zap to send SMS messages from your Google Sheets by following this
          tutorial. Until the next time ✌️
        </p>

        <BlogAuthorBio />
      </VCol>
    </VRow>
  </VContainer>
</template>
