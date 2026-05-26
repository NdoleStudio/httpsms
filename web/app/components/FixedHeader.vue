<script setup lang="ts">
import { useDisplay } from "vuetify";

const { lgAndUp, mdAndDown } = useDisplay();
</script>

<template>
  <v-app-bar elevation="0" color="#121212" height="70">
    <v-container>
      <v-row>
        <v-col class="w-full d-flex">
          <NuxtLink
            :to="{ name: 'index' }"
            class="text-on-surface text-h4 text-decoration-none"
            :class="{ 'mt-5': mdAndDown, 'mt-4': !mdAndDown }"
          >
            <v-avatar v-if="lgAndUp" :image="'/img/logo.svg'" :size="30" />
            HTTP SMS
          </NuxtLink>
          <v-spacer />
          <v-btn
            color="primary"
            class="mt-5 mb-5"
            :size="lgAndUp ? 'large' : 'default'"
            :to="{ name: 'login' }"
          >
            Get Started
            <span v-if="lgAndUp">&nbsp;For Free</span>
          </v-btn>
        </v-col>
      </v-row>
    </v-container>
  </v-app-bar>
</template>
