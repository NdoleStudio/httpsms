<template>
  <v-container>
    <v-row class="mt-16">
      <v-col cols="12" md="9">
        <v-row>
          <v-col v-for="post in blogPosts" :key="post.route" cols="12">
            <v-hover v-slot="{ hover }">
              <vue-glow
                color="#329ef4"
                mode="hex"
                elevation="7"
                :intensity="hover ? 0.7 : 0"
                intense
              >
                <v-card :to="post.route">
                  <v-card-title class="text-h4 text-break">{{
                    post.title
                  }}</v-card-title>
                  <v-card-subtitle>
                    <span class="text-uppercase blue--text">{{
                      post.date
                    }}</span>
                    • <span class="text-uppercase">{{ post.readTime }}</span>
                  </v-card-subtitle>
                  <v-card-text class="mt-n2">
                    <p class="text--primary subtitle-1">
                      {{ post.description }}
                    </p>
                    <div class="d-flex mt-n2">
                      <v-avatar class="mb-n2">
                        <v-img :src="post.authorImage"></v-img>
                      </v-avatar>
                      <div class="ml-2">
                        <p class="subtitle-1">{{ post.authorName }}</p>
                        <p class="mt-n5 mb-n4">
                          <v-icon small color="#1DA1F2">{{
                            mdiTwitter
                          }}</v-icon>
                          <v-icon small color="#ffffff">{{ mdiGithub }}</v-icon>
                        </p>
                      </div>
                    </div>
                  </v-card-text>
                </v-card>
              </vue-glow>
            </v-hover>
          </v-col>
        </v-row>
      </v-col>
      <v-col v-if="$vuetify.breakpoint.mdAndUp" md="3">
        <blog-info></blog-info>
      </v-col>
    </v-row>
  </v-container>
</template>

<script lang="ts">
import { mdiGithub, mdiTwitter } from '@mdi/js'

export default {
  name: 'BlogIndex',
  layout: 'website',
  data() {
    return {
      mdiTwitter,
      mdiGithub,
      blogPosts: [
        {
          route: '/blog/end-to-end-encryption-to-sms-messages',
          title:
            'Secure your conversations with end-to-end encryption for SMS messages',
          date: 'January 21, 2024',
          readTime: '10 min read',
          authorImage: require('@/assets/img/arnold.png'),
          description:
            'Take control of your privacy by encrypting your SMS messages end-to-end. Safeguard your messages from prying eyes, ensuring absolute confidentiality.',
          authorName: 'Acho Arnold',
          authorTwitter: 'acho_arnold',
        },
        {
          route:
            '/blog/send-sms-when-new-row-is-added-to-google-sheets-using-zapier',
          title:
            'Send an SMS when a new row is added to Google Sheets using Zapier',
          date: 'November 29, 2023',
          readTime: '7 min read',
          authorImage: require('@/assets/img/arnold.png'),
          description:
            'Automate sending personalized SMS messages each time a new row is added to your Google Sheets document using Zapier without writing code.',
          authorName: 'Acho Arnold',
          authorTwitter: 'acho_arnold',
        },
        {
          route: '/blog/how-to-send-sms-messages-from-excel',
          title:
            'How to send SMS messages to multiple phone numbers from Excel',
          date: 'October 29, 2023',
          readTime: '5 min read',
          authorImage: require('@/assets/img/arnold.png'),
          description:
            'Send multiple SMS messages to your customers from a an Excel file using your Android phone as the SMS gateway.',
          authorName: 'Acho Arnold',
          authorTwitter: 'acho_arnold',
        },
        {
          route: '/blog/send-bulk-sms-from-csv-file-with-no-code',
          title: 'Send bulk SMS messages from a CSV file with no code',
          date: 'October 29, 2023',
          readTime: '7 min read',
          authorImage: require('@/assets/img/arnold.png'),
          description:
            'Send SMS messages to multiple recipients in your CSV file without writing a single piece of code.',
          authorName: 'Acho Arnold',
          authorTwitter: 'acho_arnold',
        },
        {
          route: '/blog/send-sms-from-android-phone-with-python',
          title: 'Send an SMS from your Android phone with Python',
          date: 'June 03, 2023',
          readTime: '6 min read',
          authorImage: require('@/assets/img/arnold.png'),
          description:
            'This article will show you how to configure your Android phone as an SMS gateway to automate sending text messages with the Python programing language.',
          authorName: 'Acho Arnold',
          authorTwitter: 'acho_arnold',
        },
        {
          route: '/blog/forward-incoming-sms-from-phone-to-webhook',
          title:
            'Forward a text message (SMS) from an Android phone into your webhook',
          date: 'April 08, 2023',
          readTime: '5 min read',
          authorImage: require('@/assets/img/arnold.png'),
          description:
            'Program your android phone to forward messages received on your phone to your server and trigger powerful automations with tools like zapper and IFTTFT.',
          authorName: 'Acho Arnold',
          authorTwitter: 'acho_arnold',
        },
      ],
    }
  },
  head() {
    return {
      title: 'Blog Posts - httpSMS',
    }
  },
}
</script>
