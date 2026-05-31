<script setup lang="ts">
import { mdiBookOpenVariant } from "@mdi/js";

const appStore = useAppStore();
</script>

<template>
  <div>
    <NuxtLink to="/" class="text-decoration-none d-flex align-center">
      <VAvatar color="#121212" class="pa-1" size="36" rounded="0">
        <VImg height="33" width="33" :src="'/img/logo.svg'" />
      </VAvatar>
      <h3 class="text-headline-large ml-1 mb-3 mt-3 text-white">httpSMS</h3>
    </NuxtLink>
    <p class="mt-0">
      httpSMS is an
      <a
        class="text-decoration-none"
        href="https://github.com/NdoleStudio/httpsms"
      >open source</a>
      application that converts your android phone into an SMS gateway so you
      can send and receive SMS messages using a simple HTTP API.
    </p>
    <VBtn variant="tonal" :href="appStore.appData.documentationUrl">
      <VIcon start :icon="mdiBookOpenVariant" />
      Documentation
    </VBtn>
  </div>
</template>
