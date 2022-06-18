<template>
  <v-sheet
    class="pa-4 d-flex"
    :elevation="$vuetify.breakpoint.lgAndUp ? 0 : 2"
    :color="$vuetify.breakpoint.lgAndUp ? 'grey darken-4' : 'primary'"
  >
    <div>
      <nuxt-link to="/">
        <v-toolbar-title>
          {{ $store.getters.getOwner | phoneNumber }}
        </v-toolbar-title>
      </nuxt-link>
      <p class="text--secondary mb-n1">Estonia</p>
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
