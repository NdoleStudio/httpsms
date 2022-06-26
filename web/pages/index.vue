<template>
  <div>
    <v-app-bar elevation="0" color="#121212" height="100px">
      <v-container>
        <v-row>
          <v-col class="w-full d-flex">
            <h2
              class="text--primary"
              :class="{
                'text-h4': $vuetify.breakpoint.lgAndUp,
                'text-h5': $vuetify.breakpoint.mdAndDown,
              }"
            >
              <v-icon :large="$vuetify.breakpoint.mdAndDown">mdi-forum</v-icon>
              Http SMS
            </h2>
            <v-spacer></v-spacer>
            <v-btn
              exact-path
              class="primary"
              :large="$vuetify.breakpoint.lgAndUp"
              :to="{ name: 'login' }"
            >
              Get Started
              <span v-if="$vuetify.breakpoint.lgAndUp">&nbsp;For Free</span>
            </v-btn>
          </v-col>
        </v-row>
      </v-container>
    </v-app-bar>
    <v-container class="pb-16">
      <v-row :class="{ 'py-16': $vuetify.breakpoint.lgAndUp }">
        <v-col
          cols="12"
          md="4"
          :class="{
            'text-center': $vuetify.breakpoint.mdAndDown,
            'py-16': $vuetify.breakpoint.lgAndUp,
          }"
        >
          <h1 class="text-h2" :class="{ 'mt-16': $vuetify.breakpoint.lgAndUp }">
            Convert your android phone into an SMS gateway.
          </h1>
          <h2 class="text--secondary text-h5 mt-8 mb-8">
            Use your android phone to send and receive SMS messages using an
            intuitive HTTP API.
          </h2>
          <div :class="{ 'text-center': $vuetify.breakpoint.mdAndDown }">
            <v-btn
              color="primary"
              large
              exact-path
              class="mt-4 mb-4"
              :to="{ name: 'login' }"
            >
              <v-icon v-if="$vuetify.breakpoint.lgAndUp">mdi-send</v-icon>
              Get Started
            </v-btn>
            <v-btn
              large
              color="default"
              target="_blank"
              class="mt-4 mb-4 ml-4"
              :href="$store.getters.getAppData.githubUrl"
            >
              <v-icon v-if="$vuetify.breakpoint.lgAndUp">mdi-github</v-icon>
              Open Source
            </v-btn>
          </div>
          <div
            class="mt-4"
            :class="{ 'text-center': $vuetify.breakpoint.mdAndDown }"
          >
            <v-icon color="success">mdi-check-circle</v-icon>
            Free to use
            <v-icon class="ml-4" color="success">mdi-check-circle</v-icon>
            100% Open Source
          </div>
          <v-divider
            v-if="$vuetify.breakpoint.mdAndDown"
            class="mt-6 mr-16 success"
            :class="{ 'ml-16': $vuetify.breakpoint.mdAndDown }"
          ></v-divider>
        </v-col>
        <v-col cols="12" md="8" class="py-16">
          <v-img
            class="mx-auto mb-4"
            max-height="700"
            max-width="98%"
            contain
            :src="require('assets/img/writing-code-phone.svg')"
          ></v-img>
        </v-col>
      </v-row>
    </v-container>
    <v-footer absolute>
      <v-container class="text-center">
        <v-row justify="center" no-gutters>
          <v-col cols="12">
            <v-btn
              text
              rounded
              class="my-2"
              :href="$store.getters.getAppData.documentationUrl"
              >Documentation</v-btn
            >
            <v-btn
              text
              rounded
              class="my-2"
              :href="$store.getters.getAppData.githubUrl"
              >Open Source</v-btn
            >
          </v-col>
          <v-col class="text-center mb-n4" cols="12">
            <p class="text--secondary text-subtitle-2">
              &copy; {{ new Date().getFullYear() }} -
              {{ $store.getters.getAppData.name }}
            </p>
          </v-col>
        </v-row>
      </v-container>
    </v-footer>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'

export default Vue.extend({
  name: 'LandingPage',
  middleware: ['guest'],
})
</script>
