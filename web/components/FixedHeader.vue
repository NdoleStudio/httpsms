<template>
  <v-app-bar elevation="0" color="#121212" height="70" fixed>
    <v-container>
      <v-row>
        <v-col class="w-full d-flex">
          <nuxt-link
            :to="{ name: 'index' }"
            class="text--primary text-h4 text-decoration-none"
            :class="{
              'mt-5': $vuetify.breakpoint.mdAndDown,
              'mt-4': !$vuetify.breakpoint.mdAndDown,
            }"
          >
            <v-avatar v-if="$vuetify.breakpoint.lgAndUp" tile size="30">
              <v-img contain :src="require('@/assets/img/logo.svg')"></v-img>
            </v-avatar>
            HTTP SMS
          </nuxt-link>
          <v-spacer></v-spacer>
          <v-btn
            exact-path
            class="primary mt-5 mb-5"
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
</template>
