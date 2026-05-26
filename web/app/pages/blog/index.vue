<script setup lang="ts">
definePageMeta({
  layout: "website",
});

useHead({
  title: "Blog - httpSMS",
});

type Article = {
  slug: string;
  title: string;
  description: string;
  date: string;
  readTime: string;
  author: string;
  sortDate: string;
};

const articles: Article[] = [
  {
    slug: "grant-send-and-read-sms-permissions-on-android",
    title: "Grant Send and Read SMS Permissions on Android",
    description:
      "In Android 15, the SEND_SMS and RECEIVE_SMS permissions are now hard restricted. Learn how to grant these permissions.",
    date: "February 18, 2025",
    readTime: "5 min read",
    author: "Acho Arnold",
    sortDate: "2025-02-18",
  },
  {
    slug: "end-to-end-encryption-to-sms-messages",
    title: "End-to-End Encryption to SMS Messages",
    description:
      "We have added support for end-to-end encryption for SMS messages so that no one can see the content of the messages you send using httpSMS except you.",
    date: "January 21, 2024",
    readTime: "10 min read",
    author: "Acho Arnold",
    sortDate: "2024-01-21",
  },
  {
    slug: "send-sms-when-new-row-is-added-to-google-sheets-using-zapier",
    title: "Send SMS When New Row is Added to Google Sheets Using Zapier",
    description:
      "Automate sending personalized SMS messages each time a new row is added to your Google Sheets document using Zapier.",
    date: "October 29, 2023",
    readTime: "5 min read",
    author: "Acho Arnold",
    sortDate: "2023-10-29",
  },
  {
    slug: "how-to-send-sms-messages-from-excel",
    title: "How to Send SMS Messages from Excel",
    description:
      "Send personalized SMS messages to multiple phone numbers for less than $0.002 per SMS message.",
    date: "October 29, 2023",
    readTime: "5 min read",
    author: "Acho Arnold",
    sortDate: "2023-10-29",
  },
  {
    slug: "send-bulk-sms-from-csv-file-with-no-code",
    title: "Send Bulk SMS from CSV File with No Code",
    description:
      "Send personalized SMS messages to your users in bulk using your Android phone without writing any code.",
    date: "October 29, 2023",
    readTime: "7 min read",
    author: "Acho Arnold",
    sortDate: "2023-10-29",
  },
  {
    slug: "send-sms-from-android-phone-with-python",
    title: "Send SMS from Android Phone with Python",
    description:
      "Learn how to setup your Android phone to send SMS messages using Python and the httpSMS API.",
    date: "June 03, 2023",
    readTime: "6 min read",
    author: "Acho Arnold",
    sortDate: "2023-06-03",
  },
  {
    slug: "forward-incoming-sms-from-phone-to-webhook",
    title: "Forward Incoming SMS from Phone to Webhook",
    description:
      "Program your android phone to forward messages received on your phone to your server and trigger powerful automations.",
    date: "April 08, 2023",
    readTime: "5 min read",
    author: "Acho Arnold",
    sortDate: "2023-04-08",
  },
];

const sortedArticles = computed(() =>
  [...articles].sort((a, b) => b.sortDate.localeCompare(a.sortDate)),
);
</script>

<template>
  <VContainer>
    <VRow>
      <VCol cols="12" md="8" offset-md="2">
        <h1 class="text-display-small mb-4">Blog</h1>
        <p class="text-body-large mb-8">
          Guides, tutorials, and product updates to help you get the most out of
          httpSMS.
        </p>

        <div class="d-flex flex-column ga-4">
          <NuxtLink
            v-for="article in sortedArticles"
            :key="article.slug"
            :to="`/blog/${article.slug}`"
            class="text-decoration-none"
          >
            <VCard variant="outlined" class="pa-2">
              <VCardTitle class="text-headline-small text-wrap">{{
                article.title
              }}</VCardTitle>
              <VCardSubtitle class="pt-2">
                <div class="d-flex flex-wrap ga-2 text-body-medium">
                  <span>{{ article.date }}</span>
                  <span aria-hidden="true">•</span>
                  <span>{{ article.author }}</span>
                  <span aria-hidden="true">•</span>
                  <span>{{ article.readTime }}</span>
                </div>
              </VCardSubtitle>
              <VCardText class="text-body-large">{{
                article.description
              }}</VCardText>
            </VCard>
          </NuxtLink>
        </div>
      </VCol>
    </VRow>
  </VContainer>
</template>
