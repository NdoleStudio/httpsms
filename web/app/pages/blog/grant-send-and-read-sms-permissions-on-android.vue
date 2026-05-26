<script setup lang="ts">
definePageMeta({ layout: "website" });

useHead({
  title: "Grant Send and Read SMS Permissions on Android - httpSMS",
});
</script>

<template>
  <VContainer>
    <VRow>
      <VCol cols="12" md="8" offset-md="2">
        <h1 class="text-display-small mb-2">
          Grant Send and Read SMS Permissions on Android
        </h1>
        <BlogInfo date="February 18, 2025" read-time="5 min read" />
        <VDivider class="my-6" />

        <p class="text-body-large mb-6">
          In Android 15 (Vanilla Ice Cream), the android.permission.SEND_SMS and
          android.permission.RECEIVE_SMS permissions are now hard restricted and
          cannot be granted via the runtime permissions interface.
        </p>

        <h2 class="text-headline-medium mb-4">Step 1: Open App Info</h2>
        <p class="text-body-large mb-6">
          Long press the icon of the android app which you want to grant the
          permission and select 'App info'
        </p>

        <h2 class="text-headline-medium mb-4">
          Step 2: Allow Restricted Permissions
        </h2>
        <p class="text-body-large mb-6">
          On the App Info page, click on the menu button and select the 'Allow
          restricted settings' option
        </p>

        <h2 class="text-headline-medium mb-4">Step 3: Allow SMS Permissions</h2>
        <p class="text-body-large mb-6">
          Once you have allowed the restricted settings from step 2 above, You
          can navigate to Permissions ➡️ SMS and tap the Allow button to grant
          SMS permissions to the android app.
        </p>

        <p class="text-body-large mb-6">
          Congratulations, you have successfully configured SMS permissions on
          your Android app. Don't hesitate to contact us if you face any
          problems while following this guide.
        </p>

        <BlogAuthorBio />
      </VCol>
    </VRow>
  </VContainer>
</template>
