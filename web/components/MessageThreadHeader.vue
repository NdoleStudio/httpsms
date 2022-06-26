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
        <v-tooltip
          v-if="$store.getters.getHeartbeat"
          right
          :open-on-focus="false"
          :open-on-hover="true"
          open-on-click
        >
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
    <v-menu offset-y>
      <template #activator="{ on }">
        <v-btn icon text v-on="on">
          <v-icon>mdi-dots-vertical</v-icon>
        </v-btn>
      </template>
      <v-list class="px-2" nav :dense="$vuetify.breakpoint.mdAndDown">
        <v-list-item-group>
          <v-list-item :to="{ name: 'index' }" exact>
            <v-list-item-icon class="pl-2">
              <v-icon dense>mdi-account-cog</v-icon>
            </v-list-item-icon>
            <v-list-item-content class="ml-n3">
              <v-list-item-title class="pr-16 py-1">
                <span :class="{ 'pr-16': $vuetify.breakpoint.mdAndUp }">
                  Settings
                </span>
              </v-list-item-title>
            </v-list-item-content>
          </v-list-item>
          <v-list-item @click.prevent="logout">
            <v-list-item-icon class="pl-2">
              <v-icon dense>mdi-logout</v-icon>
            </v-list-item-icon>
            <v-list-item-content class="ml-n3">
              <v-list-item-title class="pr-16 py-1">
                <span :class="{ 'pr-16': $vuetify.breakpoint.mdAndUp }">
                  Logout
                </span>
              </v-list-item-title>
            </v-list-item-content>
          </v-list-item>
        </v-list-item-group>
      </v-list>
    </v-menu>
  </v-sheet>
</template>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'

@Component
export default class MessageThreadHeader extends Vue {
  logout(): void {
    this.$fire.auth.signOut().then(() => {
      this.$store.dispatch('setUser', null)
      this.$store.dispatch('addNotification', {
        type: 'info',
        message: 'You have successfully logged out',
      })
      this.$router.push({ name: 'index' })
    })
  }
}
</script>
