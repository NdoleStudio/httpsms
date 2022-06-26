<template>
  <v-container fluid class="pa-0" :fill-height="$vuetify.breakpoint.lgAndUp">
    <v-row v-if="$vuetify.breakpoint.lgAndUp" align="center" justify="center">
      <div>
        <v-img
          class="mx-auto mb-4"
          max-height="400"
          max-width="90%"
          contain
          :src="require('assets/img/person-texting.svg')"
        ></v-img>
        <div class="text-center">
          <h3 class="text-h5 mt-4">Select a thread</h3>
          <p class="text--secondary">Send and receive messages using our API</p>
        </div>
      </div>
    </v-row>
    <v-row v-else justify="end">
      <v-col class="px-0 py-0">
        <message-thread-header></message-thread-header>
        <message-thread></message-thread>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
export default {
  name: 'ThreadsIndex',
  middleware: ['auth'],
  mounted() {
    this.$store.dispatch('loadThreads')
  },
}
</script>
