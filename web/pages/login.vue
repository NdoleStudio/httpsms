<template>
  <v-container fill-height fluid>
    <v-row align="center" justify="center">
      <v-col
        cols="12"
        md="4"
        xl="3"
        :class="{ 'mt-n16': $vuetify.breakpoint.lgAndUp }"
      >
        <h3 class="text-h2 text-center mb-4">Login</h3>
        <v-card max-width="360" class="mx-auto">
          <v-card-text class="px-0">
            <no-ssr>
              <firebase-auth></firebase-auth>
            </no-ssr>
          </v-card-text>
        </v-card>
        <div class="text-center mt-4">
          <back-button></back-button>
        </div>
      </v-col>
    </v-row>
  </v-container>
</template>

<script lang="ts">
import Vue from 'vue'
import { Component } from 'vue-property-decorator'

@Component
export default class Login extends Vue {
  layout: string = 'auth'
}
</script>
