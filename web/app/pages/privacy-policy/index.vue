<script setup lang="ts">
definePageMeta({
  layout: "website",
});

useHead({
  title: "Privacy Policy - httpSMS",
});
</script>

<template>
  <VContainer>
    <VRow>
      <VCol cols="12" md="8" offset-md="2">
        <h1 class="text-h3 mb-6">Privacy Policy</h1>
        <p class="text-medium-emphasis mb-4">Last updated: January 1, 2024</p>

        <h2 class="text-h5 mt-6 mb-3">1. Information We Collect</h2>
        <p>
          We collect information you provide directly to us, including your
          email address, phone number, and any SMS messages processed through
          our service. We also collect usage data such as IP address, browser
          type, and device information.
        </p>

        <h2 class="text-h5 mt-6 mb-3">2. How We Use Your Information</h2>
        <p>
          We use the information we collect to provide, maintain, and improve
          our services, process SMS messages, send you technical notices, and
          respond to your requests.
        </p>

        <h2 class="text-h5 mt-6 mb-3">3. Data Security</h2>
        <p>
          We take reasonable measures to protect your personal information. SMS
          messages can be encrypted end-to-end using AES-256 encryption when you
          enable the encryption feature.
        </p>

        <h2 class="text-h5 mt-6 mb-3">4. Data Retention</h2>
        <p>
          We retain your SMS messages and account data as long as your account
          is active. You can request deletion of your data at any time by
          contacting us.
        </p>

        <h2 class="text-h5 mt-6 mb-3">5. Contact Us</h2>
        <p>
          If you have questions about this Privacy Policy, please contact us at
          <a href="mailto:arnold@httpsms.com">arnold@httpsms.com</a>.
        </p>
      </VCol>
    </VRow>
  </VContainer>
</template>
