<template>
  <v-sheet
    class="pa-4 d-flex"
    :elevation="$vuetify.breakpoint.lgAndUp ? 0 : 2"
    :color="$vuetify.breakpoint.lgAndUp ? 'grey darken-4' : 'primary'"
  >
    <div>
      <nuxt-link to="/" class="text-decoration-none text--primary">
        <v-toolbar-title>
          {{ $store.getters.getOwner | phoneNumber }}
          <v-progress-circular
            v-if="$store.getters.getPolling"
            indeterminate
            :size="14"
            :width="1"
            class="mt-n1"
            color="success"
          ></v-progress-circular>
        </v-toolbar-title>
      </nuxt-link>
      <div class="d-flex">
        <p class="text--secondary mb-n1">
          {{ $store.getters.getOwner | phoneCountry }}
        </p>
        <v-tooltip v-if="$store.getters.getHeartbeat" right>
          <template #activator="{ on, attrs }">
            <v-btn
              x-small
              v-bind="attrs"
              color="success"
              class="ml-2 mt-1 mb-n1"
              icon
              v-on="on"
            >
              <v-icon x-small>mdi-circle</v-icon>
            </v-btn>
          </template>
          <h4>Last Heartbeat</h4>
          {{ $store.getters.getHeartbeat.timestamp | timestamp }}
        </v-tooltip>
      </div>
    </div>
    <v-spacer></v-spacer>
    <v-tooltip bottom>
      <template #activator="{ on, attrs }">
        <v-btn icon text v-bind="attrs" v-on="on">
          <v-icon>mdi-dots-vertical</v-icon>
        </v-btn>
      </template>
      <span>More Options</span>
    </v-tooltip>
  </v-sheet>
</template>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'

@Component
export default class MessageThreadHeader extends Vue {}
</script>
