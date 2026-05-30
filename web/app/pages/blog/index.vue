<script setup lang="ts">
definePageMeta({
  layout: "website",
});

useHead({
  title: "Blog Posts - httpSMS",
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
    title: "How to grant SMS permissions on Android 15+",
    description:
      "In Android 15 (Vanilla Ice Cream), the SEND_SMS and RECEIVE_SMS permissions are now hard restricted. Learn how to grant these permissions step by step.",
    date: "February 18, 2025",
    readTime: "5 min read",
    author: "Acho Arnold",
    sortDate: "2025-02-18",
  },
  {
    slug: "end-to-end-encryption-to-sms-messages",
    title:
      "Secure your conversations with end-to-end encryption for SMS messages",
    description:
      "Take control of your privacy by encrypting your SMS messages end-to-end. Safeguard your messages from prying eyes, ensuring absolute confidentiality.",
    date: "January 21, 2024",
    readTime: "10 min read",
    author: "Acho Arnold",
    sortDate: "2024-01-21",
  },
  {
    slug: "send-sms-when-new-row-is-added-to-google-sheets-using-zapier",
    title: "Send an SMS when a new row is added to Google Sheets using Zapier",
    description:
      "Automate sending personalized SMS messages each time a new row is added to your Google Sheets document using Zapier without writing code.",
    date: "November 29, 2023",
    readTime: "7 min read",
    author: "Acho Arnold",
    sortDate: "2023-11-29",
  },
  {
    slug: "how-to-send-sms-messages-from-excel",
    title: "How to send SMS messages to multiple phone numbers from Excel",
    description:
      "Send multiple SMS messages to your customers from a an Excel file using your Android phone as the SMS gateway.",
    date: "October 29, 2023",
    readTime: "5 min read",
    author: "Acho Arnold",
    sortDate: "2023-10-29",
  },
  {
    slug: "send-bulk-sms-from-csv-file-with-no-code",
    title: "Send bulk SMS messages from a CSV file with no code",
    description:
      "Send SMS messages to multiple recipients in your CSV file without writing a single piece of code.",
    date: "October 29, 2023",
    readTime: "7 min read",
    author: "Acho Arnold",
    sortDate: "2023-10-29",
  },
  {
    slug: "send-sms-from-android-phone-with-python",
    title: "Send an SMS from your Android phone with Python",
    description:
      "This article will show you how to configure your Android phone as an SMS gateway to automate sending text messages with the Python programing language.",
    date: "June 03, 2023",
    readTime: "6 min read",
    author: "Acho Arnold",
    sortDate: "2023-06-03",
  },
  {
    slug: "forward-incoming-sms-from-phone-to-webhook",
    title:
      "Forward a text message (SMS) from an Android phone into your webhook",
    description:
      "Program your android phone to forward messages received on your phone to your server and trigger powerful automations with tools like Zapier and IFTTT.",
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
          Learn more about httpSMS through our blog!
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
