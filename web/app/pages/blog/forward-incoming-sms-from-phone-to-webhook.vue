<script setup lang="ts">
import { useDisplay } from "vuetify";

const { mdAndUp } = useDisplay();

definePageMeta({ layout: "website" });

useHead({
  title:
    "How to forward a text message (SMS) from an android phone into your webhook - httpSMS",
  meta: [
    {
      property: "og:title",
      content:
        "How to forward a text message (SMS) from an android phone into your webhook",
    },
    {
      property: "og:description",
      content:
        "You can now program your android phone to forward messages received on your phone to your server and trigger powerful automations with tools like Zapier and IFTTT.",
    },
    {
      property: "og:image",
      content:
        "https://httpsms.com/img/blog/forward-incoming-sms-from-phone-to-webhook/header.png",
    },
    {
      name: "twitter:card",
      content: "summary_large_image",
    },
    {
      property: "og:url",
      content:
        "https://httpsms.com/blog/forward-incoming-sms-from-phone-to-webhook/",
    },
  ],
});
</script>

<template>
  <VContainer class="pt-8">
    <VRow class="mt-16">
      <VCol cols="12" md="9">
        <VImg
          style="border-radius: 4px"
          alt="blog post header image"
          src="/img/blog/forward-incoming-sms-from-phone-to-webhook/header.png"
        />

        <h1
          :class="
            mdAndUp ? 'text-display-medium mt-1' : 'text-display-small mt-1'
          "
        >
          How to forward a text message (SMS) from an android phone into your
          webhook
        </h1>
        <BlogInfo date="April 08, 2023" readTime="5 min read" />

        <p class="text-body-large mt-2">
          You can now program your android phone to forward messages received on
          your phone to your server and trigger powerful automations with tools
          like Zapier and IFTTT. I created an open source application called
          <NuxtLink class="text-decoration-none" to="/">httpSMS</NuxtLink>
          that helps you to set this up with ease
        </p>

        <h3 class="text-headline-large mt-2">Step 1: Get your API_KEY</h3>
        <p>
          Create an account on the httpSMS web application and copy your API key
          from the settings page.
          <NuxtLink class="text-decoration-none" to="/settings"
            >https://httpsms.com/settings</NuxtLink
          >
        </p>
        <VImg
          style="border-radius: 4px"
          alt="httpsms.com settings page"
          src="/img/blog/forward-incoming-sms-from-phone-to-webhook/settings.png"
        />

        <h3 class="text-headline-large mt-12">
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
        <VImg
          style="border-radius: 4px"
          alt="httpsms android app"
          height="800"
          src="/img/blog/forward-incoming-sms-from-phone-to-webhook/android-app.png"
        />

        <h3 class="text-headline-large mt-12">Step 3: Set up a webhook</h3>
        <p>
          Once the application has been installed, it will be listening for SMS
          messages received on the android phone. You can configure the
          application to sent this SMS message to your server URL using a
          webhook. You can configure this URL under the settings page in the
          httpSMS application
          <NuxtLink class="text-decoration-none" to="/settings"
            >https://httpsms.com/settings</NuxtLink
          >
        </p>
        <VImg
          style="border-radius: 4px"
          alt="httpSMS webhook configuration"
          src="/img/blog/forward-incoming-sms-from-phone-to-webhook/webhook.png"
        />

        <h3 class="text-headline-large mt-12">Conclusion</h3>
        <p>
          Congratulations, you have successfully set up SMS forwarding from your
          Android phone to a webhook! This powerful automation tool can help you
          streamline your business workflow and save you time and effort.
        </p>
        <p>
          You can also trigger the httpSMS application to send an SMS a simple
          API. You can find more information on the documentation page at
          <a class="text-decoration-none" href="https://docs.httpsms.com"
            >https://docs.httpsms.com</a
          >
        </p>
        <p>Until the next time✌️</p>

        <BlogAuthorBio />
        <VDivider class="mx-16" />
        <div class="text-center mt-8 mb-4">
          <BackButton />
        </div>
      </VCol>
    </VRow>
  </VContainer>
</template>
