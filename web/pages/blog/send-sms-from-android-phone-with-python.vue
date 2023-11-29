<template>
  <v-container class="pt-8">
    <v-row class="mt-16">
      <v-col cols="12" md="9">
        <v-img
          style="border-radius: 4px"
          alt="blog post header image"
          :src="
            require('@/static/img/blog/send-sms-from-android-phone-with-python/header.png')
          "
        ></v-img>
        <h1
          class="mt-1"
          :class="{
            'text-h2': $vuetify.breakpoint.mdAndUp,
            'text-h3': !$vuetify.breakpoint.mdAndUp,
          }"
        >
          Send an SMS from your Android phone with Python
        </h1>
        <p class="subtitle-2 mt-2">
          <span class="text-uppercase blue--text">{{ postDate }}</span>
          • <span class="text-uppercase">{{ readTime }}</span>
        </p>
        <p class="text--secondary subtitle-1 mt-2">
          In an era dominated by social media, instant messaging apps, and
          ever-evolving communication technologies, it's easy to overlook the
          humble yet remarkably resilient Short Message Service (SMS). Since its
          inception in the 1990s, SMS has stood the test of time, remaining one
          of the most widely used and reliable means of mobile communication.
        </p>
        <p>
          Whether you're a business owner looking to optimize your communication
          strategy, a developer seeking to integrate SMS functionality into your
          applications, or simply intrigued by the enduring charm of SMS, this
          article will explain how to setup your Android phone to send SMS
          messages.
        </p>
        <h3 class="text-h4 mt-8 mb-2">Prerequisites</h3>
        <ul>
          <li>Basic understanding of Python.</li>
          <li>An Android phone.</li>
          <li>
            <a href="https://www.python.org/" class="text-decoration-none"
              >Python</a
            >
            installed on your computer.
          </li>
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
        <h3 class="text-h4 mt-12">Step 3: Writing the code</h3>
        <p>
          Now that you have setup your android phone correctly on httpSMS, you
          can write the python code below in a new file named
          <code>send_sms.py</code>. This code will send and SMS and after
          running the script via your Android phone to the recipient phone
          number specified in the <code>payload</code>.
        </p>
        <v-alert type="info" outlined class="mt-2 mb-4">
          Make sure to use the correct <code>api_key</code> from step 1 and also
          use the correct <code>to</code> and <code>from</code> phone numbers in
          the <code>payload</code> variable.
        </v-alert>
        <pre v-highlight class="python w-full mb-n9">
<code>import requests
import json

api_key = "" # Get API Key from https://httpsms.com/settings

url = 'https://api.httpsms.com/v1/messages/send'

headers = {
    'x-api-key': api_key,
    'Accept': 'application/json',
    'Content-Type': 'application/json'
}

payload = {
    "content": "This is a sample text message sent via python",
    "from": "+18005550199", # This is the phone number of your android phone */
    "to": "+18005550100" # This is the recipient phone number */
}

response = requests.post(url, headers=headers, data=json.dumps(payload))

print(json.dumps(response.json(), indent=4))
</code>
        </pre>
        <p>
          Run the code above with the command
          <code>python send_sms.py</code> and check the phone specified in the
          <code>to</code> field of the <code>payload</code> to verify that the
          message has been received successfully.
        </p>
        <v-img
          style="border-radius: 4px"
          alt="httpsms android app"
          height="800"
          contain
          :src="
            require('@/static/img/blog/send-sms-from-android-phone-with-python/sms-sent.png')
          "
        ></v-img>
        <h3 class="text-h4 mt-12">Conclusion</h3>
        <p>
          Congratulations, you have successfully configured your android phone
          to send SMS messages via python. You can now reuse this code to send
          SMS messages from your python applications.
        </p>
        <p>
          If you are also interested in forwarding incoming SMS from your
          android phone to your server, checkout our
          <nuxt-link
            to="/blog/forward-incoming-sms-from-phone-to-webhook"
            class="text-decoration-none"
            >SMS forwarding guide.</nuxt-link
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
  name: 'SendSmsFromAndroidPhoneWithPython',
  layout: 'website',
  data() {
    return {
      mdiTwitter,
      authorImage: require('@/assets/img/arnold.png'),
      authorName: 'Acho Arnold',
      postDate: 'June 03, 2023',
      readTime: '6 min read',
      authorTwitter: 'acho_arnold',
    }
  },
  head() {
    return {
      title: 'Send an SMS from your Android phone with Python - httpSMS',
      meta: [
        {
          hid: 'og:title',
          property: 'og:title',
          content: 'Send an SMS from your Android phone with Python',
        },
        {
          hid: 'og:description',
          property: 'og:description',
          content:
            'Configure your Android phone as an SMS gateway to automate sending text messages with the Python programing language.',
        },
        {
          hid: 'og:image',
          property: 'og:image',
          content:
            'https://httpsms.com/img/blog/send-sms-from-android-phone-with-python/header.png',
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
            'https://httpsms.com/blog/send-sms-from-android-phone-with-python/',
        },
      ],
    }
  },
}
</script>
