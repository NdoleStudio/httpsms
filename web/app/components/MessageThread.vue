<script setup lang="ts">
import {
  mdiPlus,
  mdiDownload,
  mdiCheckAll,
  mdiCheck,
  mdiAlert,
  mdiAccount,
} from '@mdi/js'
import { formatPhoneNumber, startsWithLetter } from '~/utils/filters'

const threadsStore = useThreadsStore()
const phonesStore = usePhonesStore()
const appStore = useAppStore()
const notificationsStore = useNotificationsStore()

function threadDate(date: string): string {
  return new Date(date).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
  })
}

function onInstallApp() {
  notificationsStore.addNotification({
    type: 'info',
    message: 'Downloading the httpSMS Android App',
  })
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
    <div
      v-if="
        !threadsStore.loadingThreads &&
        threadsStore.threads.length === 0 &&
        !threadsStore.archivedThreads
      "
      class="text-center mt-6"
    >
      <p v-if="phonesStore.owner" class="text-medium-emphasis text-center">
        Start sending messages
      </p>
      <v-btn
        v-if="phonesStore.owner && phonesStore.phones.length !== 0"
        color="primary"
        :to="{ name: 'messages' }"
      >
        <v-icon :icon="mdiPlus" start />
        New Message
      </v-btn>
    </div>
    <div
      v-if="phonesStore.phones.length === 0 && !threadsStore.loadingThreads"
      class="px-4 text-center"
    >
      <p>
        Install the mobile app on your Android phone to start sending messages.
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
        <v-icon :icon="mdiDownload" start />
        Download App
      </v-btn>
    </div>
    <v-list
      v-if="!threadsStore.loadingThreads && threadsStore.threads.length > 0"
      class="py-0"
      lines="three"
    >
      <v-list-item
        v-for="thread in threadsStore.threads"
        :key="thread.id"
        :to="{ name: 'threads-id', params: { id: thread.id } }"
        :active="threadsStore.threadId === thread.id"
      >
        <template #prepend>
          <v-avatar :color="thread.color" size="40">
            <v-icon v-if="!startsWithLetter(thread.contact)" color="white">{{
              mdiAccount
            }}</v-icon>
            <span v-else class="text-white text-headline-small">{{
              thread.contact.substring(0, 1)
            }}</span>
          </v-avatar>
        </template>
        <v-list-item-title>{{
          formatPhoneNumber(thread.contact)
        }}</v-list-item-title>
        <v-list-item-subtitle
          class="text-truncate mt-1"
          style="max-width: 250px"
        >
          {{ thread.last_message_content }}
        </v-list-item-subtitle>
        <template #append>
          <div class="d-flex flex-column align-end">
            <span class="text-body-small text-medium-emphasis">
              {{ threadDate(thread.order_timestamp) }}
            </span>
            <div class="mt-1">
              <v-icon
                v-if="thread.status === 'expired'"
                color="warning"
                size="x-small"
                :icon="mdiAlert"
              />
              <v-icon
                v-else-if="thread.status === 'delivered'"
                color="primary"
                size="x-small"
                :icon="mdiCheckAll"
              />
              <v-icon
                v-else-if="thread.status === 'received'"
                color="success"
                size="x-small"
                :icon="mdiCheckAll"
              />
              <v-icon
                v-else-if="thread.status === 'sent'"
                size="x-small"
                :icon="mdiCheck"
              />
              <v-icon
                v-else-if="thread.status === 'failed'"
                color="error"
                size="x-small"
                :icon="mdiAlert"
              />
            </div>
          </div>
        </template>
      </v-list-item>
    </v-list>
  </div>
</template>
