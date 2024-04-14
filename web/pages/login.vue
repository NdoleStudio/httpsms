<template>
  <v-container fluid fill-height>
    <v-row align="center" justify="center">
      <v-col
        cols="12"
        md="4"
        xl="3"
        :class="{ 'mt-n16': $vuetify.breakpoint.lgAndUp }"
      >
        <div class="text-center mb-5">
          <v-avatar tile size="45" class="mt-n8 mr-4">
            <v-img contain :src="require('@/assets/img/logo.svg')"></v-img>
          </v-avatar>
          <span class="text-h3">Welcome</span>
        </div>
        <p class="subtitle-1 text-center text--secondary mt-1 mb-4">
          Join <b>3,587+</b> happy users who have sent or
          <br />
          received more than <b>816,171+</b> SMS messages
        </p>
        <v-card max-width="360" class="mx-auto">
          <v-card-text class="px-0">
            <client-only>
              <firebase-auth :to="to"></firebase-auth>
            </client-only>
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

export default Vue.extend({
  name: 'LoginIndex',
  middleware: ['guest'],
  data() {
    return {
      to: '/threads',
    }
  },
  head() {
    return {
      title: 'Login To Your Account - Http SMS',
    }
  },
  mounted() {
    if (this.$route.query.to) {
      this.to = this.$route.query.to as string
    }
  },
})
</script>
