<script setup lang="ts">
import { useDisplay } from "vuetify";
import { mdiCheck, mdiInformation } from "@mdi/js";

const { lgAndUp } = useDisplay();
const notificationsStore = useNotificationsStore();

const notificationActive = computed({
  get: () => notificationsStore.notification.active,
  set: () => notificationsStore.disableNotification(),
});
</script>

<template>
  <v-snackbar
    v-model="notificationActive"
    :color="notificationsStore.notification.type"
    :timeout="notificationsStore.notification.timeout"
  >
    <v-icon
      v-if="notificationsStore.notification.type === 'success'"
      :color="notificationsStore.notification.type"
      :icon="mdiCheck"
    />
    <v-icon
      v-if="notificationsStore.notification.type === 'info'"
      :color="notificationsStore.notification.type"
      :icon="mdiInformation"
    />
    {{ notificationsStore.notification.message }}
    <template #actions>
      <v-btn
        v-if="lgAndUp"
        :color="notificationsStore.notification.type"
        variant="text"
        @click="notificationsStore.disableNotification()"
      >
        <span class="font-weight-bold">Close</span>
      </v-btn>
    </template>
  </v-snackbar>
</template>
