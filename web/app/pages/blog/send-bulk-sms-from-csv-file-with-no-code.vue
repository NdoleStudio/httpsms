<script setup lang="ts">
import { useDisplay } from "vuetify";

const { mdAndUp } = useDisplay();

import { mdiCommentTextMultipleOutline } from "@mdi/js";

definePageMeta({ layout: "website" });

useHead({
  title: "Send bulk SMS messages from a CSV file with no code  - httpSMS",
  meta: [
    {
      property: "og:title",
      content: "Send bulk SMS messages from a CSV file with no code",
    },
    {
      property: "og:description",
      content:
        "Send personalized SMS messages from your CSV file instantly using your Android phone as the SMS gateway.",
    },
    {
      name: "twitter:card",
      content: "summary_large_image",
    },
    {
      property: "og:url",
      content:
        "https://httpsms.com/blog/send-bulk-sms-from-csv-file-with-no-code/",
    },
  ],
});
</script>

<template>
  <VContainer class="pt-8">
    <VRow class="mt-16">
      <VCol cols="12" md="9">
        <h1
          :class="
            mdAndUp ? 'text-display-medium mt-1' : 'text-display-small mt-1'
          "
        >
          Send multiple SMS messages from a CSV file with no code
        </h1>
        <BlogInfo date="October 29, 2023" readTime="7 min read" />

        <p class="text-body-large mt-2">
          Send personalized SMS messages to your users in bulk using your
          Android phone. The good news is, you don't have to write a single
          piece of code, just upload your CSV file and we will take care of the
          rest.
        </p>

        <h3 class="text-headline-large mt-8 mb-2">What is a CSV file</h3>
        <p>
          CSV is an abbreviation for comma-separated values. A CSV file allows
          data to be saved in a table structured format using a comma
          <code>,</code> to separate the various cells of a table and a new line
          to separate the various rows in the table. CSV files can be used with
          any spreadsheet program, such as Microsoft Excel, Open Office Calc, or
          Google Sheets.
        </p>

        <h3 class="text-headline-large mt-8 mb-2">Prerequisites</h3>
        <ul>
          <li>Basic understanding of CSV files.</li>
          <li>An Android phone.</li>
        </ul>

        <h3 class="text-headline-large mt-8 mb-2">Step 1: Get your API Key</h3>
        <p>
          Create an account on
          <NuxtLink class="text-decoration-none" to="/">httpsms.com</NuxtLink>
          and copy your API key from the settings page
          <NuxtLink class="text-decoration-none" to="/settings"
            >https://httpsms.com/settings</NuxtLink
          >
        </p>
        <VImg
          style="border-radius: 4px"
          alt="httpsms.com settings page"
          src="/img/blog/forward-incoming-sms-from-phone-to-webhook/settings.png"
        />

        <h3 class="text-headline-large mb-4 mt-16">
          Step 2: Install the httpSMS android app
        </h3>
        <p>
          <a
            class="text-decoration-none"
            href="https://github.com/NdoleStudio/httpsms/releases/latest/download/HttpSms.apk"
            >⬇️ Download and install</a
          >
          the httpSMS android app on your phone and sign in using your API KEY
          which you copied above. This app listens for SMS messages received on
          your android phone.
        </p>
        <VAlert type="info" variant="tonal">
          Make sure to enter your phone number in the international format e.g
          +18005550199 when authenticating with the httpSMS Android app.
        </VAlert>
        <VImg
          style="border-radius: 4px"
          alt="httpsms android app"
          height="800"
          src="/img/blog/forward-incoming-sms-from-phone-to-webhook/android-app.png"
        />

        <h3 class="text-headline-large mt-12">Step 3: Edit your CSV file</h3>
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
        <VAlert type="info" variant="tonal" class="mt-2 mb-4">
          Make sure to use the correct <code>FromPhoneNumber</code> from step 2
          above in your CSV file
        </VAlert>
        <VImg
          style="border-radius: 4px"
          alt="httpsms spreadsheet"
          height="800"
          src="/img/blog/send-bulk-sms-from-csv-file-with-no-code/httpms-spreedsheet.png"
        />

        <h3 class="text-headline-large mt-12">Step 4: Send the SMS Messages</h3>
        <p>
          Visit the
          <NuxtLink class="text-decoration-none" to="/bulk-messages">
            <VIcon
              color="primary"
              size="small"
              class="mr-1"
              :icon="mdiCommentTextMultipleOutline"
            />
            Bulk Messages
          </NuxtLink>
          page on httpSMS and upload your CSV file and send the your SMS
          messages.
        </p>
        <VImg
          style="border-radius: 4px"
          alt="bulk csv upload"
          height="800"
          src="/img/blog/send-bulk-sms-from-csv-file-with-no-code/bulk-csv-upload.png"
        />

        <p class="mt-12">
          Don't hesitate to
          <a class="text-decoration-none" href="mailto:arnold@httpsms.com"
            >contact us</a
          >
          if you face any issues sending bulk SMS messages from your CSV files
          by following this tutorial.
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
