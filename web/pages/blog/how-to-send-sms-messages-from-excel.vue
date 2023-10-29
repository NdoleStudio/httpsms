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
          How to send SMS messages to multiple phone numbers from Excel
        </h1>
        <p class="subtitle-2 mt-2">
          <span class="text-uppercase blue--text">{{ postDate }}</span>
          • <span class="text-uppercase">{{ readTime }}</span>
        </p>
        <p class="text--secondary subtitle-1 mt-2">
          Send personalized SMS messages to multiple phone numbers for less than
          <b>$0.002 </b> per SMS message. You can also configure every SMS
          message in your Excel spreadsheet so they are unique for each
          recipient phone number.
        </p>
        <h3 class="text-h4 mt-8 mb-2">Prerequisites</h3>
        <ul>
          <li>Basic understanding of Microsoft Excel or Google Sheets.</li>
          <li>An Android phone.</li>
        </ul>
        <h3 class="text-h4 mt-8 mb-2">Step 1: Get your API Key</h3>
        <p>
          Create an account on
          <nuxt-link class="text-decoration-none" to="/">httpsms.com</nuxt-link>
          and copy your API key from the settings page
          <nuxt-link class="text-decoration-none" to="/settings"
            >https://httpsms.com/settings</nuxt-link
          >
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
              require('@/static/img/blog/forward-incoming-sms-from-phone-to-webhook/settings.png')
            "
          ></v-img>
        </vue-glow>
        <h3 class="text-h4 mb-4 mt-16">
          Step 2: Install the httpSMS android app
        </h3>
        <p>
          <a
            target="_blank"
            class="text-decoration-none"
            href="https://github.com/NdoleStudio/httpsms/releases/latest/download/HttpSms.apk"
            >⬇️ Download and install</a
          >
          the httpSMS android app on your phone and sign in using your API KEY
          which you copied above. This app listens for SMS messages received on
          your android phone.
        </p>
        <v-alert type="info" outlined>
          Make sure to enter your phone number in the international format e.g
          +18005550199 when authenticating with the httpSMS Android app.
        </v-alert>
        <v-img
          style="border-radius: 4px"
          alt="httpsms android app"
          height="800"
          contain
          :src="
            require('@/static/img/blog/forward-incoming-sms-from-phone-to-webhook/android-app.png')
          "
        ></v-img>
        <h3 class="text-h4 mt-12">Step 3: Edit your Excel file</h3>
        <p>
          Download the
          <a
            class="text-decoration-none"
            href="/templates/httpsms-bulk.xlsx"
            download
            >httpSMS Excel file template</a
          >
          and edit it with your favorite spreadsheet software e.g Excel, Google
          Sheets, Open Office etc. Fill in the phone number which you registered
          in httpSMS in the <code>FromPhoneNumber</code> column and fill in the
          number of the recipient of the SMS in the
          <code>ToPhoneNumber</code> column. Also add the SMS which you want to
          send in the message in the <code>Content</code> column.
        </p>
        <v-alert type="info" outlined class="mt-2 mb-4">
          Make sure to use the correct <code>FromPhoneNumber</code> from step 2
          above in your Excel file
        </v-alert>
        <v-img
          style="border-radius: 4px"
          alt="httpsms android app"
          height="800"
          contain
          :src="
            require('@/static/img/blog/send-bulk-sms-from-csv-file-with-no-code/httpms-spreedsheet.png')
          "
        ></v-img>
        <h3 class="text-h4 mt-12">Step 3: Send the SMS Messages</h3>
        <p>
          Visit the
          <router-link class="text-decoration-none" to="/bulk-messages"
            ><v-icon color="primary" x-small>{{
              mdiCommentTextMultipleOutline
            }}</v-icon>
            Bulk Messages</router-link
          >
          page on httpSMS and upload your Excel file and send the your SMS
          messages.
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
            alt="httpsms android app"
            height="800"
            contain
            :src="
              require('@/static/img/blog/send-bulk-sms-from-csv-file-with-no-code/bulk-csv-upload.png')
            "
          ></v-img>
        </vue-glow>
        <p class="mt-12">
          Don't hesitate to
          <a class="text-decoration-none" href="mailto:arnold@httpsms.com"
            >contact us</a
          >
          if you face any issues sending bulk SMS messages from your Excel files
          by following this tutorial.
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
  name: 'SendBulkSmsToMultiplePhoneNumbersFromExcel',
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
