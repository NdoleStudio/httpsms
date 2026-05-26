<script setup lang="ts">
definePageMeta({
  layout: "website",
});

useHead({
  title: "Blog - httpSMS",
});

const articles = [
  {
    slug: "end-to-end-encryption-to-sms-messages",
    title: "How to Add End-to-End Encryption to SMS Messages",
    description:
      "Learn how to encrypt your SMS messages using AES-256 encryption with the httpSMS platform.",
    date: "2023-09-15",
  },
  {
    slug: "forward-incoming-sms-from-phone-to-webhook",
    title: "Forward Incoming SMS from Phone to Webhook",
    description:
      "Setup automatic forwarding of incoming SMS messages to your server using webhooks.",
    date: "2023-08-20",
  },
  {
    slug: "grant-send-and-read-sms-permissions-on-android",
    title: "Grant Send and Read SMS Permissions on Android",
    description:
      "Step by step guide to grant the necessary SMS permissions on your Android phone.",
    date: "2023-07-10",
  },
  {
    slug: "how-to-send-sms-messages-from-excel",
    title: "How to Send SMS Messages from Excel",
    description: "Send bulk SMS messages using an Excel spreadsheet template.",
    date: "2023-06-05",
  },
  {
    slug: "send-bulk-sms-from-csv-file-with-no-code",
    title: "Send Bulk SMS from CSV File with No Code",
    description:
      "Upload a CSV file to send SMS messages to multiple recipients without writing code.",
    date: "2023-05-12",
  },
  {
    slug: "send-sms-from-android-phone-with-python",
    title: "Send SMS from Android Phone with Python",
    description:
      "Use Python to send SMS messages through your Android phone via the httpSMS API.",
    date: "2023-04-18",
  },
  {
    slug: "send-sms-when-new-row-is-added-to-google-sheets-using-zapier",
    title: "Send SMS When New Row is Added to Google Sheets Using Zapier",
    description:
      "Automate SMS sending with Zapier and Google Sheets integration.",
    date: "2023-03-22",
  },
];
</script>

<template>
  <VContainer>
    <VRow>
      <VCol cols="12" md="8" offset-md="2">
        <h1 class="text-h3 mb-6">Blog</h1>
        <p class="text-medium-emphasis mb-8">
          Guides and tutorials to help you get the most out of httpSMS.
        </p>
        <VCard
          v-for="article in articles"
          :key="article.slug"
          class="mb-4"
          :to="`/blog/${article.slug}`"
          variant="outlined"
        >
          <VCardTitle>{{ article.title }}</VCardTitle>
          <VCardSubtitle>{{ article.date }}</VCardSubtitle>
          <VCardText>{{ article.description }}</VCardText>
        </VCard>
      </VCol>
    </VRow>
  </VContainer>
</template>
