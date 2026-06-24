<script setup lang="ts">
import { useDisplay } from 'vuetify'
const { mdAndDown } = useDisplay()
</script>

<template>
  <v-main>
    <v-container fluid class="fill-height">
      <v-row align="center" justify="center">
        <v-col
          cols="12"
          md="5"
          xl="3"
          class="text-center mt-16"
          :class="{ 'px-6': mdAndDown, 'px-16': !mdAndDown }"
        >
          <h2 class="text-headline-large text-medium-emphasis mt-16 mb-4">
            <img
              class="mx-auto d-inline-block"
              src="/img/logo.svg"
              style="max-width: 32px"
              alt="httpSMS Logo"
            />
            Loading the httpSMS dashboard
          </h2>
          <v-progress-circular
            indeterminate
            size="160"
            class="mt-8"
            color="primary"
          />
        </v-col>
      </v-row>
    </v-container>
  </v-main>
</template>
