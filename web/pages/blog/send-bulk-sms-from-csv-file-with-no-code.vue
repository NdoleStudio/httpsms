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
          Send multiple SMS messages from a CSV file with no code
        </h1>
        <p class="subtitle-2 mt-2">
          <span class="text-uppercase blue--text">{{ postDate }}</span>
          • <span class="text-uppercase">{{ readTime }}</span>
        </p>
        <p class="text--secondary subtitle-1 mt-2">
          Send personalized SMS messages to your users in bulk using your
          Android phone. The good news is, you don't have to write a single
          piece of code, just upload your CSV file and we will take care of the
          rest.
        </p>
        <h3 class="text-h4 mt-8 mb-2">What is a CSV file</h3>
        <p>
          CSV is an abbreviation for comma-separated values. A CSV file allows
          data to be saved in a table structured format using a comma
          <code>,</code> to separate the various cells of a table and a new line
          to separate the various rows in the table. CSV files can be used with
          any spreadsheet program, such as Microsoft Excel, Open Office Calc, or
          Google Sheets.
        </p>
        <h3 class="text-h4 mt-8 mb-2">Prerequisites</h3>
        <ul>
          <li>Basic understanding of CSV files.</li>
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
        <h3 class="text-h4 mt-12">Step 3: Edit your CSV file</h3>
        <p>
          Download the
          <a
            class="text-decoration-none"
            href="/templates/httpsms-bulk.csv"
            download
            >httpSMS CSV file template</a
          >
          and edit it with your favorite spreadsheet software e.g Excel, Google
          Sheets or even a text editor like notepad. Fill in the phone number
          which you registered in httpSMS in the
          <code>FromPhoneNumber</code> column and fill in the number of the
          recipient of the SMS in the <code>ToPhoneNumber</code> column. Also
          add the SMS which you want to send in the message in the
          <code>Content</code> column.
        </p>
        <v-alert type="info" outlined class="mt-2 mb-4">
          Make sure to use the correct <code>FromPhoneNumber</code> from step 2
          above in your CSV file
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
          page on httpSMS and upload your CSV file and send the your SMS
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
          if you face any issues sending bulk SMS messages from your CSV files
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
  name: 'SendBulkSmsFromCsvFileWithNoCode',
  layout: 'website',
  data() {
    return {
      mdiTwitter,
      mdiCommentTextMultipleOutline,
      authorImage: require('@/assets/img/arnold.png'),
      authorName: 'Acho Arnold',
      postDate: 'October 29, 2023',
      readTime: '7 min read',
      authorTwitter: 'acho_arnold',
    }
  },
  head() {
    return {
      title: 'Send bulk SMS messages from a CSV file with no code  - httpSMS',
      meta: [
        {
          hid: 'og:title',
          property: 'og:title',
          content: 'Send bulk SMS messages from a CSV file with no code',
        },
        {
          hid: 'og:description',
          property: 'og:description',
          content:
            'Send personalized SMS messages from your CSV file instantly using your Android phone as the SMS gateway.',
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
            'https://httpsms.com/blog/send-bulk-sms-from-csv-file-with-no-code',
        },
      ],
    }
  },
}
</script>
