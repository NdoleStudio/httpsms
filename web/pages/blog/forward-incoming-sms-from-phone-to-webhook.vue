<template>
  <v-container class="pt-8">
    <v-row class="mt-16">
      <v-col cols="12" md="9">
        <v-img
          style="border-radius: 4px"
          alt="blog post header image"
          :src="
            require('@/static/img/blog/forward-incoming-sms-from-phone-to-webhook/header.png')
          "
        ></v-img>
        <h1
          class="mt-1"
          :class="{
            'text-h2': $vuetify.breakpoint.mdAndUp,
            'text-h3': !$vuetify.breakpoint.mdAndUp,
          }"
        >
          How to forward a text message (SMS) from an android phone into your
          webhook
        </h1>
        <p class="subtitle-2 mt-2">
          <span class="text-uppercase blue--text">{{ postDate }}</span>
          • <span class="text-uppercase">{{ readTime }}</span>
        </p>
        <p class="text--secondary subtitle-1 mt-2">
          You can now program your android phone to forward messages received on
          your phone to your server and trigger powerful automations with tools
          like Zapier and IFTTT. I created an open source application called
          <nuxt-link to="/" class="text-decoration-none">httpSMS</nuxt-link>
          that helps you to set this up with ease
        </p>
        <h3 class="text-h4 mt-2">Step 1: Get your API_KEY</h3>
        <p>
          Create an account on the httpSMS web application and copy your API key
          from the settings page.
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
        <h3 class="text-h4 mt-12">Step 2: Install the httpSMS android app</h3>
        <p>
          Install the
          <a
            target="_blank"
            class="text-decoration-none"
            href="https://github.com/NdoleStudio/httpsms/releases/latest/download/HttpSms.apk"
            >httpSMS</a
          >
          android app on your phone and sign in using your API KEY which you
          copied above. This app listens for SMS messages received on your
          android phone. <br /><span v-if="$vuetify.breakpoint.mdAndUp"
            >👉</span
          >
          <a
            target="_blank"
            class="text-decoration-none text-break"
            href="https://github.com/NdoleStudio/httpsms/releases/latest/download/HttpSms.apk"
            >https://github.com/NdoleStudio/httpsms/releases/latest/download/HttpSms.apk</a
          >
        </p>
        <v-img
          style="border-radius: 4px"
          alt="httpsms android app"
          height="800"
          contain
          :src="
            require('@/static/img/blog/forward-incoming-sms-from-phone-to-webhook/android-app.png')
          "
        ></v-img>
        <h3 class="text-h4 mt-12">Step 3: Set up a webhook</h3>
        <p>
          Once the application has been installed, it will be listening for SMS
          messages received on the android phone. You can configure the
          application to sent this SMS message to your server URL using a
          webhook. You can configure this URL under the settings page in the
          httpSMS application
          <nuxt-link to="/settings" class="text-decoration-none"
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
            alt="httpSMS webhook configuration"
            :src="
              require('@/static/img/blog/forward-incoming-sms-from-phone-to-webhook/webhook.png')
            "
          ></v-img>
        </vue-glow>
        <h3 class="text-h4 mt-12">Conclusion</h3>
        <p>
          Congratulations, you have successfully set up SMS forwarding from your
          Android phone to a webhook! This powerful automation tool can help you
          streamline your business workflow and save you time and effort.
        </p>
        <p>
          You can also trigger the httpSMS application to send an SMS a simple
          API. You can find more information on the documentation page at
          <a
            target="_blank"
            class="text-decoration-none"
            href="https://docs.httpsms.com"
            >https://docs.httpsms.com</a
          >
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
import { mdiTwitter } from '@mdi/js'
export default {
  name: 'ForwardIncomingSMSFromPhoneToWebhook',
  layout: 'website',
  data() {
    return {
      mdiTwitter,
      authorImage: require('@/assets/img/arnold.png'),
      authorName: 'Acho Arnold',
      postDate: 'April 08, 2023',
      readTime: '5 min read',
      authorTwitter: 'acho_arnold',
    }
  },
  head() {
    return {
      title:
        'How to forward a text message (SMS) from an android phone into your webhook - httpSMS',
      meta: [
        {
          hid: 'og:title',
          property: 'og:title',
          content:
            'How to forward a text message (SMS) from an android phone into your webhook',
        },
        {
          hid: 'og:description',
          property: 'og:description',
          content:
            'You can now program your android phone to forward messages received on your phone to your server and trigger powerful automations with tools like Zapier and IFTTT.',
        },
        {
          hid: 'og:image',
          property: 'og:image',
          content:
            'https://httpsms.com/img/blog/forward-incoming-sms-from-phone-to-webhook/header.png',
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
            'https://httpsms.com/blog/forward-incoming-sms-from-phone-to-webhook',
        },
      ],
    }
  },
}
</script>
