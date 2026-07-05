<script setup lang="ts">
import { useDisplay } from 'vuetify'

const { mdAndUp } = useDisplay()

definePageMeta({ layout: 'website' })

useSeoMeta({
  title:
    'Send an SMS message when a new row is added to Google Sheets using Zapier - httpSMS',
  description:
    'Automatically send a personalized SMS every time a new row is added to Google Sheets using Zapier and httpSMS — no code required.',
  ogTitle:
    'Send an SMS message when a new row is added to Google Sheets using Zapier',
  ogDescription:
    "Automate sending personalized SMS messages each time a new row is added to your Google Sheets document using Zapier. You don't need to write any code to make this happen and you can personalize the SMS messages which are sent out.",
  ogImage: 'https://httpsms.com/header.png',
  twitterCard: 'summary_large_image',
})
</script>

<template>
  <VContainer v-highlight class="pt-8">
    <VRow :class="{ 'mt-16': mdAndUp }">
      <VCol cols="12" md="9">
        <h1
          :class="
            mdAndUp ? 'text-display-medium mt-1' : 'text-display-small mt-n2'
          "
        >
          Send an SMS message when a new row is added to Google Sheets using
          Zapier
        </h1>
        <BlogInfo date="October 29, 2023" read-time="5 min read" />

        <p class="text-body-large mt-2">
          Automate sending personalized SMS messages each time a new row is
          added to your Google Sheets document using Zapier. You don't need to
          write any code to make this happen and you can personalize the SMS
          messages which are sent out.
        </p>

        <h3 class="text-headline-large mt-8 mb-2">Prerequisites</h3>
        <ul>
          <li>Basic understanding of Google Sheets.</li>
          <li>Basic understanding of Zapier.</li>
          <li>
            An account on
            <NuxtLink class="text-decoration-none" to="/login"
              >httpsms.com</NuxtLink
            >
          </li>
        </ul>

        <h3 class="text-headline-large mt-8 mb-2">
          Step 1: Create trigger on Zapier
        </h3>
        <p>
          Create a new Zap on Zapier and select Google Sheets as the trigger.
          The event name should be <b>"New Spreadsheet Row"</b> if you want to
          send an SMS message every time a new row is added to your Google
          Sheets document.
        </p>
        <VImg
          style="border-radius: 4px"
          alt="zapier trigger"
          src="/img/blog/send-sms-when-new-row-is-added-to-google-sheets-using-zapier/zapier-trigger.png"
        />
        <p class="mt-8">
          On the Zap, select the <b>Spreadsheet</b> which you have on google
          drive and make sure to select the correct <b>Worksheet</b>.
        </p>
        <VAlert type="info" variant="tonal">
          In the sample spreadsheet below, we are mimicking an e-commerce store.
          The first column contains the name of the customer, the second column
          is the name of the product which was bought and the third column is
          the phone number of the customer who made the purchase. You can use
          your own custom spreadsheet with your own set of columns.
        </VAlert>
        <VImg
          style="border-radius: 4px"
          alt="google sheets"
          src="/img/blog/send-sms-when-new-row-is-added-to-google-sheets-using-zapier/google-sheets.png"
        />

        <h3 class="text-headline-large mt-8 mb-2">
          Step 2: Create an action on Zapier
        </h3>
        <p>
          An action is what happens after the trigger. In this case, we want to
          send an SMS message to the customer who made the purchase. Select
          <b>Webhooks By Zapier</b> as the action app and select
          <b>Custom Request</b> as the action event.
        </p>
        <VImg
          style="border-radius: 4px"
          alt="zapier action event"
          src="/img/blog/send-sms-when-new-row-is-added-to-google-sheets-using-zapier/zapier-action-event.png"
        />
        <p class="mt-8">
          On the <b>Action</b> section in Zapier, set the Method to
          <code>Post</code>. Set the URL to
          <code>https://api.httpsms.com/v1/messages/send</code>. Set the
          <code>Data Pass-Through</code> to <code>false</code>. In the Data
          field, add the following JSON payload.
        </p>
        <pre
          class="pa-4 mb-6 rounded bg-surface-variant overflow-x-auto"
        ><code class="language-json text-body-medium">{
  "content": "Hello [Name]\nThanks for ordering [Product] via our shopify store. Your order will be shipped today!",
  "from": "+18005550199",
  "to": "[ToPhoneNumber]"
}</code></pre>
        <VAlert type="info" variant="tonal" class="mt-4">
          In the JSON message above, we are mimicking an e-commerce store. The
          <code>[Name]</code> variable contains the name of the customer on the
          spreadsheet. <code>[Product]</code> contains the name of the product
          which was bought and <code>[ToPhoneNumber]</code> contains the phone
          number of the customer who made the purchase. You can use your own
          custom message with your own set of variables according to your
          spreadsheet. Change the <code>from</code> field to the phone number
          which you registered on httpsms.com.
        </VAlert>
        <p class="mt-8">
          On the headers section add a new header called
          <code>x-api-key</code> and the value of this header should be your API
          key on
          <NuxtLink class="text-decoration-none" to="/settings"
            >httpsms.com</NuxtLink
          >
          and you can copy your API key from the settings page
          <NuxtLink class="text-decoration-none" to="/settings"
            >https://httpsms.com/settings</NuxtLink
          >.
        </p>
        <p>
          Also add a new header called <code>Content-Type</code> and the value
          of this header should be <code>application/json</code>
        </p>
        <p>
          The final configuration of the action should look like the screenshot
          below.
        </p>
        <VImg
          style="border-radius: 4px"
          alt="zapier action action"
          src="/img/blog/send-sms-when-new-row-is-added-to-google-sheets-using-zapier/zapier-action-action.png"
        />

        <h3 class="text-headline-large mb-4 mt-16">Conclusion</h3>
        <p>
          Publish your zap and you will automatically trigger httpsms to send an
          SMS to your customer when ever you add a new row in the google sheet.
          Don't hesitate to
          <a class="text-decoration-none" href="mailto:arnold@httpsms.com"
            >contact us</a
          >
          if you face any issues configuring your zap to send SMS messages from
          your Google Sheets by following this tutorial.
        </p>
        <p>Until the next time✌️</p>

        <BlogAuthorBio />
        <VDivider class="mx-16" />
        <div class="text-center mt-8 mb-4">
          <BackButton />
        </div>
      </VCol>
      <VCol v-if="$vuetify.display.mdAndUp" md="3">
        <BlogSidebar />
      </VCol>
    </VRow>
  </VContainer>
</template>
