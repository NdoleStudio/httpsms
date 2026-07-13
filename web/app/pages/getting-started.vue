<script setup lang="ts">
import {
  mdiDownload,
  mdiAccountPlusOutline,
  mdiOpenInNew,
  mdiCheckCircle,
  mdiLanguagePython,
  mdiMicrosoftExcel,
  mdiFileDelimitedOutline,
  mdiLightningBolt,
  mdiWebhook,
  mdiLockOutline,
} from '@mdi/js'
import { useDisplay } from 'vuetify'

const { mdAndUp } = useDisplay()

const apkUrl =
  'https://github.com/NdoleStudio/httpsms/releases/latest/download/HttpSms.apk'

definePageMeta({ layout: 'website' })

useSeoMeta({
  title: 'Getting Started - Send your first SMS with httpSMS',
  description:
    'Turn your Android phone into an SMS gateway in 5 minutes. Download the app, sign in with your API key, and send your first text message with httpSMS.',
  ogTitle: 'Getting Started with httpSMS - Send your first SMS in 5 minutes',
  ogDescription:
    'Turn your Android phone into an SMS gateway in 5 minutes. Download the app, sign in with your API key, and send your first text message.',
  ogImage: 'https://httpsms.com/header.png',
  twitterCard: 'summary_large_image',
})

type AutomationLink = {
  title: string
  description: string
  to: string
  icon: string
}

const automations: AutomationLink[] = [
  {
    title: 'Send SMS with Python',
    description: 'Automate text messages from your scripts in a few lines.',
    to: '/blog/send-sms-from-android-phone-with-python',
    icon: mdiLanguagePython,
  },
  {
    title: 'Send SMS from Excel',
    description:
      'Message a whole list of contacts straight from a spreadsheet.',
    to: '/blog/how-to-send-sms-messages-from-excel',
    icon: mdiMicrosoftExcel,
  },
  {
    title: 'Send bulk SMS from a CSV',
    description: 'Upload a CSV and text everyone on it — no code needed.',
    to: '/blog/send-bulk-sms-from-csv-file-with-no-code',
    icon: mdiFileDelimitedOutline,
  },
  {
    title: 'Send SMS with Zapier',
    description: 'Text a contact when a new row lands in Google Sheets.',
    to: '/blog/send-sms-when-new-row-is-added-to-google-sheets-using-zapier',
    icon: mdiLightningBolt,
  },
  {
    title: 'Forward SMS to a webhook',
    description: 'Push every incoming message to your own server in real time.',
    to: '/blog/forward-incoming-sms-from-phone-to-webhook',
    icon: mdiWebhook,
  },
  {
    title: 'Encrypt your messages',
    description: 'Add end-to-end encryption so only you can read your texts.',
    to: '/blog/end-to-end-encryption-to-sms-messages',
    icon: mdiLockOutline,
  },
]
</script>

<template>
  <VContainer class="pt-8">
    <VRow :class="{ 'mt-16': mdAndUp }">
      <VCol cols="12" md="9">
        <!-- Hero -->
        <span class="text-overline text-primary">Getting started</span>
        <h1
          :class="
            mdAndUp ? 'text-display-medium mt-1' : 'text-display-small mt-n2'
          "
        >
          Send your first SMS in 5 minutes
        </h1>
        <p class="text-body-large mt-4 text-medium-emphasis">
          httpSMS turns the Android phone in your pocket into an SMS gateway. No
          new SIM card, no per-message contracts — just your phone, the app, and
          the four steps below.
        </p>

        <div class="d-flex flex-wrap ga-3 mt-6">
          <VBtn
            color="primary"
            size="large"
            variant="flat"
            :href="apkUrl"
            :prepend-icon="mdiDownload"
          >
            Download the Android app
          </VBtn>
          <VBtn
            color="primary"
            size="large"
            variant="outlined"
            to="/login"
            :prepend-icon="mdiAccountPlusOutline"
          >
            Create a free account
          </VBtn>
        </div>

        <VDivider class="my-10" />

        <!-- Step 1 -->
        <div class="d-flex align-center mb-3">
          <VAvatar color="primary" size="40" class="mr-3">
            <span class="text-title-large font-weight-bold">1</span>
          </VAvatar>
          <h2 class="text-headline-large">
            Create your account &amp; copy your API key
          </h2>
        </div>
        <p>
          Sign up at
          <NuxtLink
            class="text-decoration-none hover:text-decoration-underline"
            to="/login"
            >httpsms.com</NuxtLink
          >, then open
          <NuxtLink
            class="text-decoration-none hover:text-decoration-underline"
            to="/settings"
            >Settings</NuxtLink
          >
          and copy your API key. You'll paste it into the app in Step 3, so keep
          it handy.
        </p>
        <VImg
          class="rounded mt-4"
          alt="httpSMS settings page showing the API key"
          src="/img/blog/forward-incoming-sms-from-phone-to-webhook/settings.png"
        />

        <VDivider class="my-10" />

        <!-- Step 2 -->
        <div class="d-flex align-center mb-3">
          <VAvatar color="primary" size="40" class="mr-3">
            <span class="text-title-large font-weight-bold">2</span>
          </VAvatar>
          <h2 class="text-headline-large">Download &amp; install the app</h2>
        </div>
        <p>
          <a
            class="text-decoration-none hover:text-decoration-underline"
            :href="apkUrl"
            >⬇️ Download the httpSMS app</a
          >
          onto your Android phone and open the file to install it. If Android
          asks, allow installs from this source — that's normal for apps
          downloaded outside the Play Store.
        </p>

        <!-- Placeholder: replace with a GIF of installing the APK -->
        <div class="media-placeholder rounded mt-4">
          <div class="text-h4">📹</div>
          <p class="text-title-medium mt-2 mb-0">
            GIF: Downloading &amp; installing the app
          </p>
          <p class="text-body-small text-medium-emphasis mb-0">
            /img/getting-started/install-app.gif
          </p>
        </div>

        <VDivider class="my-10" />

        <!-- Step 3 -->
        <div class="d-flex align-center mb-3">
          <VAvatar color="primary" size="40" class="mr-3">
            <span class="text-title-large font-weight-bold">3</span>
          </VAvatar>
          <h2 class="text-headline-large">Sign in on your phone</h2>
        </div>
        <p>
          Open the app, paste the API key from Step 1, and enter the phone
          number of this Android phone. This is the phone that will send your
          messages.
        </p>
        <VAlert type="info" variant="tonal" class="my-4">
          Enter your phone number in international format, e.g.
          <code>+18005550199</code>.
        </VAlert>

        <VRow class="mt-2">
          <VCol cols="12" sm="6">
            <VImg
              class="rounded"
              alt="httpSMS Android app sign-in screen"
              max-height="520"
              src="/img/blog/forward-incoming-sms-from-phone-to-webhook/android-app.png"
            />
          </VCol>
          <VCol cols="12" sm="6">
            <!-- Placeholder: replace with a GIF of signing in -->
            <div class="media-placeholder media-placeholder-tall rounded">
              <div class="text-h4">📹</div>
              <p class="text-title-medium mt-2 mb-0">GIF: Signing in</p>
              <p class="text-body-small text-medium-emphasis mb-0">
                /img/getting-started/sign-in.gif
              </p>
            </div>
          </VCol>
        </VRow>

        <VAlert
          type="info"
          variant="outlined"
          class="mt-6"
          :icon="mdiCheckCircle"
        >
          On Android 15+ you'll need to grant SMS permissions before the app can
          send texts. Follow our short guide:
          <NuxtLink
            class="text-decoration-none hover:text-decoration-underline font-weight-bold"
            to="/blog/grant-send-and-read-sms-permissions-on-android"
            >How to grant SMS permissions on Android 15+</NuxtLink
          >.
        </VAlert>

        <VDivider class="my-10" />

        <!-- Step 4 -->
        <div class="d-flex align-center mb-3">
          <VAvatar color="primary" size="40" class="mr-3">
            <span class="text-title-large font-weight-bold">4</span>
          </VAvatar>
          <h2 class="text-headline-large">Send your first SMS</h2>
        </div>
        <p>
          Head to your dashboard, start a new message, and text yourself a quick
          hello. Your phone sends it over the network, and you'll see it arrive
          on the other device. That's it — you're up and running.
        </p>

        <!-- Placeholder: replace with a GIF of composing and sending a message -->
        <div class="media-placeholder rounded mt-4">
          <div class="text-h4">📹</div>
          <p class="text-title-medium mt-2 mb-0">
            GIF: Composing &amp; sending a message
          </p>
          <p class="text-body-small text-medium-emphasis mb-0">
            /img/getting-started/send-sms.gif
          </p>
        </div>

        <div class="mt-6">
          <VBtn
            color="primary"
            size="large"
            variant="flat"
            to="/threads"
            :append-icon="mdiOpenInNew"
          >
            Open your dashboard
          </VBtn>
        </div>

        <VDivider class="my-10" />

        <!-- Automate it -->
        <h2 class="text-headline-large mb-1">Ready to automate?</h2>
        <p class="text-body-large text-medium-emphasis mt-0 mb-4">
          Now that your phone is sending texts, wire it into the tools you
          already use.
        </p>
        <VRow>
          <VCol v-for="item in automations" :key="item.to" cols="12" sm="6">
            <NuxtLink
              :to="item.to"
              class="text-decoration-none hover:text-decoration-underline"
            >
              <VHover v-slot="{ isHovering, props: hoverProps }">
                <VCard
                  v-bind="hoverProps"
                  :elevation="isHovering ? 8 : 2"
                  :color="isHovering ? 'blue-darken-4' : undefined"
                  class="automation-card d-flex align-center pa-4"
                >
                  <VAvatar
                    color="primary"
                    variant="tonal"
                    size="44"
                    class="mr-4"
                  >
                    <VIcon :icon="item.icon" />
                  </VAvatar>
                  <div>
                    <div class="text-title-medium">{{ item.title }}</div>
                    <div class="text-body-small text-medium-emphasis">
                      {{ item.description }}
                    </div>
                  </div>
                </VCard>
              </VHover>
            </NuxtLink>
          </VCol>
        </VRow>
      </VCol>
      <VCol v-if="mdAndUp" md="3">
        <BlogSidebar />
      </VCol>
    </VRow>
  </VContainer>
</template>

<style scoped>
.media-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  min-height: 220px;
  padding: 24px;
  border: 2px dashed rgb(var(--v-theme-primary));
  background: rgb(var(--v-theme-primary), 0.04);
}

.media-placeholder-tall {
  min-height: 520px;
  height: 100%;
}

.automation-card {
  transition: all 0.3s ease;
}
</style>
