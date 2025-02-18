<template>
  <div>
    <nuxt-link to="/" class="text-decoration-none d-flex">
      <v-avatar tile size="33" class="mt-1">
        <v-img contain :src="require('@/assets/img/logo.svg')"></v-img>
      </v-avatar>
      <h3 class="text-h4 text--primary ml-1">httpSMS</h3>
    </nuxt-link>
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
    <v-btn :href="$store.getters.getAppData.documentationUrl">
      <v-icon left>{{ mdiBookOpenVariant }}</v-icon>
      Documentation
    </v-btn>
  </div>
</template>

<script lang="ts">
import { mdiBookOpenVariant } from '@mdi/js'
import Vue from 'vue'

export default Vue.extend({
  name: 'BlogInfo',
  data() {
    return {
      mdiBookOpenVariant,
    }
  },
})
</script>
