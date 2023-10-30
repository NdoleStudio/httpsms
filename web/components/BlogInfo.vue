<template>
  <div>
    <v-badge
      class="logo-badge"
      :class="{ 'logo-badge--mobile': $vuetify.breakpoint.mdAndDown }"
      color="primary"
      content="Beta"
    >
      <nuxt-link to="/" class="text-decoration-none d-flex">
        <v-avatar tile size="33" class="mt-1">
          <v-img contain :src="require('@/assets/img/logo.svg')"></v-img>
        </v-avatar>
        <h3 class="text-h4 text--primary ml-1">httpSMS</h3>
      </nuxt-link>
    </v-badge>
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
    <p class="body-2 mt-6 mb-4">
      Subscribe to my newsletter where I share the new features and updates on
      httpSMS.
    </p>
    <v-progress-circular
      v-if="!substackLoaded"
      indeterminate
      size="40"
      width="2"
      color="primary"
    ></v-progress-circular>
    <div id="custom-substack-embed" ref="custom-substack-embed"></div>
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
      substackLoaded: false,
    }
  },
  mounted() {
    setTimeout(() => {
      const s = document.createElement('script')
      s.type = 'text/javascript'
      s.src = 'https://substackapi.com/widget.js'
      document.getElementsByTagName('head')[0].appendChild(s)
      this.substackLoaded = true
    }, 5000)
  },
})
</script>
