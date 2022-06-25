<template>
  <v-snackbar
    v-model="notificationActive"
    text
    :color="notification.type"
    :timeout="notification.timeout"
  >
    <v-icon v-if="notification.type === 'success'" :color="notification.type"
      >mdi-check</v-icon
    >
    <v-icon v-if="notification.type === 'info'" :color="notification.type"
      >mdi-information</v-icon
    >
    {{ notification.message }}
    <template #action="{ attrs }">
      <v-btn
        v-if="$vuetify.breakpoint.lgAndUp"
        :color="notification.type"
        text
        v-bind="attrs"
        @click="disableNotification"
      >
        <span class="font-weight-bold">Close</span>
      </v-btn>
    </template>
  </v-snackbar>
</template>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'
import { Notification } from '~/store'

@Component
export default class Toast extends Vue {
  get notification(): Notification {
    return this.$store.getters.getNotification
  }

  get notificationActive(): boolean {
    return this.$store.getters.getNotification.active
  }

  set notificationActive(state: boolean) {
    this.disableNotification()
  }

  disableNotification() {
    this.$store.dispatch('disableNotification')
  }
}
</script>
