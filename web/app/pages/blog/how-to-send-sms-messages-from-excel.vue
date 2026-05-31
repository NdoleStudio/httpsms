<script setup lang="ts">
import { useDisplay } from "vuetify";

const { mdAndUp } = useDisplay();

import { mdiCommentTextMultipleOutline } from "@mdi/js";

definePageMeta({ layout: "website" });

useHead({
  title:
    "How to send SMS messages to multiple phone numbers from Excel  - httpSMS",
  meta: [
    {
      property: "og:title",
      content: "How to send SMS messages to multiple phone numbers from Excel",
    },
    {
      property: "og:description",
      content:
        "Send personalized SMS messages to multiple phone numbers for less than $0.002 per SMS message using your Android phone as the SMS gateway.",
    },
    {
      name: "twitter:card",
      content: "summary_large_image",
    },
    {
      property: "og:url",
      content: "https://httpsms.com/blog/how-to-send-sms-messages-from-excel/",
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
          How to send SMS messages to multiple phone numbers from Excel
        </h1>
        <BlogInfo date="October 29, 2023" readTime="5 min read" />

        <p class="text-body-large mt-2">
          Send personalized SMS messages to multiple phone numbers for less than
          <b>$0.002 </b> per SMS message. You can also configure every SMS
          message in your Excel spreadsheet so they are unique for each
          recipient phone number.
        </p>

        <h3 class="text-headline-large mt-8 mb-2">Prerequisites</h3>
        <ul>
          <li>Basic understanding of Microsoft Excel or Google Sheets.</li>
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

        <h3 class="text-headline-large mt-12">Step 3: Edit your Excel file</h3>
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
        <VAlert type="info" variant="tonal" class="mt-2 mb-4">
          Make sure to use the correct <code>FromPhoneNumber</code> from step 2
          above in your Excel file
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
          page on httpSMS and upload your Excel file and send the your SMS
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
          if you face any issues sending bulk SMS messages from your Excel files
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
