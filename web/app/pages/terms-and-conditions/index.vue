<script setup lang="ts">
definePageMeta({
  layout: "website",
});

useHead({
  title: "Terms and Conditions - httpSMS",
});
</script>

<template>
  <VContainer>
    <VRow>
      <VCol cols="12" md="8" offset-md="2">
        <h1 class="text-h3 mb-6">Terms and Conditions</h1>
        <p class="text-medium-emphasis mb-4">Last updated: January 1, 2024</p>

        <h2 class="text-h5 mt-6 mb-3">1. Acceptance of Terms</h2>
        <p>
          By accessing or using the httpSMS service, you agree to be bound by
          these Terms and Conditions. If you do not agree to these terms, do not
          use the service.
        </p>

        <h2 class="text-h5 mt-6 mb-3">2. Service Description</h2>
        <p>
          httpSMS provides a platform that converts your Android phone into an
          SMS gateway, allowing you to send and receive SMS messages
          programmatically via an HTTP API.
        </p>

        <h2 class="text-h5 mt-6 mb-3">3. Acceptable Use</h2>
        <p>
          You agree not to use the service for spam, illegal activities, or any
          purpose that violates applicable laws. We reserve the right to suspend
          accounts that violate these terms.
        </p>

        <h2 class="text-h5 mt-6 mb-3">4. Limitations of Liability</h2>
        <p>
          httpSMS is provided "as is" without warranties of any kind. We are not
          liable for any damages resulting from the use or inability to use the
          service.
        </p>

        <h2 class="text-h5 mt-6 mb-3">5. Contact</h2>
        <p>
          For questions about these terms, contact us at
          <a href="mailto:arnold@httpsms.com">arnold@httpsms.com</a>.
        </p>
      </VCol>
    </VRow>
  </VContainer>
</template>
