<script setup lang="ts">
import { mdiDotsVertical } from "@mdi/js";
import { useDisplay } from "vuetify";

const { mdAndUp } = useDisplay();

definePageMeta({ layout: "website" });

useHead({
  title:
    "How to grant SEND_SMS and RECEIVE_SMS permissions on Android 14+ - httpSMS",
  meta: [
    {
      property: "og:title",
      content:
        "How to grant SEND_SMS and RECEIVE_SMS permissions on Android 14+",
    },
    {
      property: "og:description",
      content:
        "We have added support for end-to-end encryption for SMS messages so that no one can see the content of the messages you send using httpSMS except you.",
    },
  ],
});
</script>

<template>
  <VContainer class="pt-8">
    <VRow class="mt-16">
      <VCol cols="12" md="9">
        <h1 class="text-h3 text-md-h2 mt-1">
          How to grant SMS permissions on Android 15+
        </h1>
        <BlogInfo date="February 18, 2025" read-time="5 min read" />

        <p class="text-subtitle-1 mt-2">
          In Android 15 (Vanilla Ice Cream), the
          <code>android.permission.SEND_SMS</code> and
          <code>android.permission.RECEIVE_SMS</code> permissions are now hard
          restricted and cannot be granted
          <a
            class="text-decoration-none"
            href="https://developer.android.com/training/permissions/requesting"
            >via the runtime permissions interface</a
          >.
        </p>

        <VAlert type="warning" variant="outlined" class="my-4">
          Granting the <code>SEND_SMS</code> and
          <code>RECEIVE_SMS</code> permissions will allow an Android app to be
          able to read and send SMS messages on your phone. Make sure you trust
          the application before allowing these permissions.
        </VAlert>

        <h3 class="text-h4 mt-8 mb-2">Step1: Open App Info</h3>
        <p>
          Long press the icon of the android app which you want to grant the
          permission and select <b>"App info"</b>
        </p>
        <VImg
          style="border-radius: 4px"
          alt="httpsms android app"
          height="800"
          src="/img/blog/grant-send-and-read-sms-permissions-on-android/app-info.png"
        />

        <h3 class="text-h4 mb-4 mt-16">Step 2: Allow Restricted Permissions</h3>
        <p>
          On the App Info page, click on the menu button
          <VIcon :icon="mdiDotsVertical" /> and select the
          <b>"Allow restricted settings"</b> option
        </p>
        <VImg
          style="border-radius: 4px"
          alt="httpsms android app"
          height="800"
          src="/img/blog/grant-send-and-read-sms-permissions-on-android/allow-restricted-settings.png"
        />

        <h3 class="text-h4 mb-4 mt-16">Step 3: Allow SMS Permissions</h3>
        <p>
          Once you have allowed the restricted settings from step 2 above, You
          can navigate to <b>Permissions ➡️ SMS</b> and tap the Allow button to
          grant SMS permissions to the android app.
        </p>
        <VImg
          style="border-radius: 4px"
          alt="httpsms android app"
          height="800"
          src="/img/blog/grant-send-and-read-sms-permissions-on-android/allow.png"
        />

        <h3 class="text-h4 mt-12">Conclusion</h3>
        <p>
          Congratulations, you have successfully configured SMS permissions on
          your Android app. Don't hesitate to contact us if you face any
          problems while following this guide.
        </p>

        <BlogAuthorBio />
        <VDivider class="mx-16" />
        <div class="text-center mt-8 mb-4">
          <BackButton />
        </div>
      </VCol>
    </VRow>
  </VContainer>
</template>
