<template>
  <v-container class="pt-8">
    <v-row class="mt-16">
      <v-col cols="12" md="9">
        <h1
          class="mt-1"
          :class="{
            'text-h2': $vuetify.breakpoint.mdAndUp,
            'text-h3': !$vuetify.breakpoint.mdAndUp,
          }"
        >
          Send an SMS message when a new row is added to Google Sheets using
          Zapier
        </h1>
        <p class="subtitle-2 mt-2">
          <span class="text-uppercase blue--text">{{ postDate }}</span>
          • <span class="text-uppercase">{{ readTime }}</span>
        </p>
        <p class="text--secondary subtitle-1 mt-2">
          Automate sending personalized SMS messages each time a new row is
          added to your Google Sheets document using Zapier. You don't need to
          write any code to make this happen and you can personalize the SMS
          messages which are sent out.
        </p>
        <h3 class="text-h4 mt-8 mb-2">Prerequisites</h3>
        <ul>
          <li>Basic understanding of Google Sheets.</li>
          <li>Basic understanding of Zapier.</li>
          <li>
            An account on
            <router-link to="/login" class="text-decoration-none"
              >httpsms.com</router-link
            >
          </li>
        </ul>
        <h3 class="text-h4 mt-8 mb-2">Step 1: Create trigger on Zapier</h3>
        <p>
          Create a new Zap on Zapier and select Google Sheets as the trigger.
          The event name should be <b>"New Spreadsheet Row"</b> if you want to
          send an SMS message every time a new row is added to your Google
          Sheets document.
        </p>
        <vue-glow
          color="#329ef4"
          mode="hex"
          elevation="14"
          :intensity="1.07"
          intense
        >
          <v-img
            style="border-radius: 4px"
            alt="httpsms.com settings page"
            :src="
              require('~/static/img/blog/send-sms-when-new-row-is-added-to-google-sheets-using-zapier/zapier-trigger.png')
            "
          ></v-img>
        </vue-glow>
        <p class="mt-8">
          On the Zap, select the <b>Spreadsheet</b> which you have on google
          drive and make sure to select the correct <b>Worksheet</b>.
        </p>
        <v-alert type="info" outlined>
          In the sample spreadsheet below, we are mimicking an e-commerce store.
          The first column contains the name of the customer, the second column
          is the name of the product which was bought and the third column is
          the phone number of the customer who made the purchase. You can use
          your own custom spreadsheet with your own set of columns.
        </v-alert>
        <vue-glow
          color="#329ef4"
          mode="hex"
          elevation="14"
          :intensity="1.07"
          intense
        >
          <v-img
            style="border-radius: 4px"
            alt="httpsms.com settings page"
            :src="
              require('@/static/img/blog/send-sms-when-new-row-is-added-to-google-sheets-using-zapier/google-sheets.png')
            "
          ></v-img>
        </vue-glow>
        <h3 class="text-h4 mt-8 mb-2">Step 2: Create an action on Zapier</h3>
        <p>
          An action is what happens after the trigger. In this case, we want to
          send an SMS message to the customer who made the purchase. Select
          <b>Webhooks By Zapier</b> as the action app and select
          <b>Custom Request</b> as the action event.
        </p>
        <vue-glow
          color="#329ef4"
          mode="hex"
          elevation="14"
          :intensity="1.07"
          intense
        >
          <v-img
            style="border-radius: 4px"
            alt="httpsms.com settings page"
            :src="
              require('@/static/img/blog/send-sms-when-new-row-is-added-to-google-sheets-using-zapier/zapier-action-event.png')
            "
          ></v-img>
        </vue-glow>
        <p class="mt-8">
          On the <b>Action</b> section in Zapier, set the Method to
          <code>Post</code>. Set the URL to
          <code>https://api.httpsms.com/v1/messages/send</code>. Set the `Data
          Pass-Through` to <code>false</code>. In the Data field, add the
          following JSON payload.
        </p>
        <pre v-highlight class="json w-full mt-n2 mb-n13">
<code>
{
  "content": "Hello [Name]\nThanks for ordering [Product] via our shopify store. Your order will be shipped today!",
  "from": "+18005550199",
  "to": "[ToPhoneNumber]"
}
</code>
        </pre>
        <v-alert type="info" class="mt-4" outlined>
          In the JSON message above, we are mimicking an e-commerce store. The
          <code>[Name]</code> variable contains the name of the customer on the
          spreadsheet. <code>[Product]</code> contains the name of the product
          which was bought and <code>[ToPhoneNumber]</code> contains the phone
          number of the customer who made the purchase. You can use your own
          custom message with your own set of variables according to your
          spreadsheet. Change the <code>from</code> field to the phone number
          which you registered on httpsms.com.
        </v-alert>
        <p class="mt-8">
          On the headers section add a new header called
          <code>x-api-key</code> and the value of this header should be your API
          key on
          <nuxt-link class="text-decoration-none" to="/settings"
            >httpsms.com</nuxt-link
          >
          and you can copy your API key from the settings page<nuxt-link
            class="text-decoration-none"
            to="/settings"
            >https://httpsms.com/settings</nuxt-link
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
        <vue-glow
          color="#329ef4"
          mode="hex"
          elevation="14"
          :intensity="1.07"
          intense
        >
          <v-img
            style="border-radius: 4px"
            alt="httpsms.com settings page"
            :src="
              require('@/static/img/blog/send-sms-when-new-row-is-added-to-google-sheets-using-zapier/zapier-action-action.png')
            "
          ></v-img>
        </vue-glow>

        <h3 class="text-h4 mb-4 mt-16">Conclusion</h3>
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
        <blog-author-bio></blog-author-bio>
        <v-divider class="mx-16"></v-divider>
        <div class="text-center mt-8 mb-4">
          <back-button></back-button>
        </div>
      </v-col>
      <v-col v-if="$vuetify.breakpoint.mdAndUp" md="3">
        <blog-info></blog-info>
      </v-col>
    </v-row>
  </v-container>
</template>

<script lang="ts">
import { mdiTwitter, mdiCommentTextMultipleOutline } from '@mdi/js'
export default {
  name: 'SendSmsWhenNewRowIsAddedToGoogleSheetsUsingZapier',
  layout: 'website',
  data() {
    return {
      mdiTwitter,
      mdiCommentTextMultipleOutline,
      authorImage: require('@/assets/img/arnold.png'),
      authorName: 'Acho Arnold',
      postDate: 'October 29, 2023',
      readTime: '5 min read',
      authorTwitter: 'acho_arnold',
    }
  },
  head() {
    return {
      title:
        'How to send SMS messages to multiple phone numbers from Excel  - httpSMS',
      meta: [
        {
          hid: 'og:title',
          property: 'og:title',
          content:
            'How to send SMS messages to multiple phone numbers from Excel',
        },
        {
          hid: 'og:description',
          property: 'og:description',
          content:
            'Configure your Android phone as an SMS gateway to automate sending text messages with the Python programing language.',
        },
        {
          hid: 'twitter:card',
          name: 'twitter:card',
          content: 'summary_large_image',
        },
        {
          hid: 'og:url',
          property: 'og:url',
          content:
            'https://httpsms.com/blog/how-to-send-sms-messages-from-excel',
        },
      ],
    }
  },
}
</script>
