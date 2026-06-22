<script setup lang="ts">
import { useDisplay } from "vuetify";
import {
  mdiPlus,
  mdiDownload,
  mdiCheckAll,
  mdiCheck,
  mdiAlert,
} from "@mdi/js";
import { formatPhoneNumber } from "~/utils/filters";
import type { MessageThread } from "~~/shared/types/message-thread";

const { mdAndDown } = useDisplay();
const threadsStore = useThreadsStore();
const phonesStore = usePhonesStore();
const appStore = useAppStore();
const notificationsStore = useNotificationsStore();

function getInitials(contact: string): string {
  const formatted = formatPhoneNumber(contact);
  return formatted.substring(0, 2);
}

function threadDate(date: string): string {
  return new Date(date).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
  });
}

function onInstallApp() {
  notificationsStore.addNotification({
    type: "info",
    message: "Downloading the httpSMS Android App",
  });
}
</script>

<template>
  <div>
    <v-progress-linear
      v-if="threadsStore.loadingThreads"
      color="primary"
      indeterminate
    />
    <div
      v-if="!threadsStore.loadingThreads && threadsStore.archivedThreads"
      class="bg-warning py-1 text-center text-uppercase text-title-medium"
    >
      Archived Messages
    </div>
    <v-sheet
      v-if="
        !threadsStore.loadingThreads &&
        threadsStore.threads.length === 0 &&
        !threadsStore.archivedThreads
      "
      class="text-center mt-8 mx-3"
      :color="mdAndDown ? '#121212' : '#363636'"
    >
      <div v-if="mdAndDown">
        <v-img
          class="mx-auto mb-4"
          max-width="80%"
          src="/img/person-texting.svg"
        />
        <p v-if="phonesStore.owner" class="text-medium-emphasis">
          Start sending messages
        </p>
      </div>
      <v-btn
        v-if="phonesStore.owner && phonesStore.phones.length !== 0"
        color="primary"
        :to="{ name: 'messages' }"
      >
        <v-icon :icon="mdiPlus" />
        New Message
      </v-btn>
    </v-sheet>
    <div
      v-if="phonesStore.phones.length === 0 && !threadsStore.loadingThreads"
      class="px-4 text-center"
    >
      <p>
        Install the mobile app on your android phone to start sending messages.
        You can also
        <a
          href="https://discord.gg/kGk8HVqeEZ"
          target="_blank"
          class="text-decoration-none hover:text-decoration-underline"
          >message us on Discord</a
        >
        to help set things up.
      </p>
      <v-btn
        color="primary"
        :href="appStore.appData.appDownloadUrl"
        @click="onInstallApp"
      >
        <v-icon :icon="mdiDownload" />
        Install App
      </v-btn>
    </div>
    <v-list
      v-if="!threadsStore.loadingThreads && threadsStore.threads.length > 0"
      class="pa-0"
    >
      <v-list-item
        v-for="thread in threadsStore.threads"
        :key="thread.id"
        :to="{ name: 'threads-id', params: { id: thread.id } }"
        :active="threadsStore.threadId === thread.id"
      >
        <template #prepend>
          <v-avatar :color="thread.color" size="40">
            <span class="text-white text-body-medium">{{
              getInitials(thread.contact)
            }}</span>
          </v-avatar>
        </template>
        <v-list-item-title>{{
          formatPhoneNumber(thread.contact)
        }}</v-list-item-title>
        <v-list-item-subtitle class="text-truncate" style="max-width: 250px">
          {{ thread.last_message_content }}
        </v-list-item-subtitle>
        <template #append>
          <div class="d-flex flex-column align-end">
            <span class="text-caption text-medium-emphasis">
              {{ threadDate(thread.order_timestamp) }}
            </span>
            <v-icon
              v-if="thread.status === 'expired'"
              color="warning"
              size="small"
              :icon="mdiAlert"
            />
            <v-icon
              v-else-if="thread.status === 'delivered'"
              color="primary"
              size="small"
              :icon="mdiCheckAll"
            />
            <v-icon
              v-else-if="thread.status === 'received'"
              color="success"
              size="small"
              :icon="mdiCheckAll"
            />
            <v-icon
              v-else-if="thread.status === 'sent'"
              size="small"
              :icon="mdiCheck"
            />
            <v-icon
              v-else-if="thread.status === 'failed'"
              color="error"
              size="small"
              :icon="mdiAlert"
            />
          </div>
        </template>
      </v-list-item>
    </v-list>
  </div>
</template>

