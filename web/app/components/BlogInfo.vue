<script setup lang="ts">
import { mdiBookOpenVariant } from "@mdi/js";

const appStore = useAppStore();
</script>

<template>
  <div>
    <NuxtLink to="/" class="text-decoration-none d-flex">
      <v-avatar :image="'/img/logo.svg'" :size="33" class="mt-1" />
      <h3 class="text-h4 text-on-surface ml-1">httpSMS</h3>
    </NuxtLink>
    <p>
      httpSMS is an
      <a
        class="text-decoration-none"
        href="https://github.com/NdoleStudio/httpsms"
        >open source</a
      >
      application that converts your android phone into an SMS gateway so you
      can send and receive SMS messages using a simple HTTP API.
    </p>
    <v-btn :href="appStore.appData.documentationUrl">
      <v-icon start :icon="mdiBookOpenVariant" />
      Documentation
    </v-btn>
  </div>
</template>
