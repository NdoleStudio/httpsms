<template>
  <div>
    <v-progress-linear
      v-if="$store.getters.getLoadingThreads"
      color="primary"
      indeterminate
    ></v-progress-linear>
    <div
      v-if="!$store.getters.getLoadingThreads && $store.getters.getIsArchived"
      class="warning py-1 text-center text-uppercase text-subtitle-1"
    >
      Archived Messages
    </div>
    <v-sheet
      v-if="
        !$store.getters.getLoadingThreads &&
        threads.length === 0 &&
        !$store.getters.getIsArchived
      "
      class="text-center mt-8 mx-3"
      :color="$vuetify.breakpoint.mdAndDown ? '#121212' : '#363636'"
    >
      <div v-if="$vuetify.breakpoint.mdAndDown">
        <v-img
          class="mx-auto mb-4"
          max-width="80%"
          contain
          :src="require('assets/img/person-texting.svg')"
        ></v-img>
        <p v-if="$store.getters.getOwner" class="text--secondary">
          Start sending messages
        </p>
      </div>
      <v-btn
        v-if="$store.getters.getOwner && $store.getters.getPhones.length !== 0"
        class="primary"
        :to="{ name: 'messages' }"
      >
        <v-icon>
          {{ mdiPlus }}
        </v-icon>
        New Message
      </v-btn>
    </v-sheet>
    <div v-if="$store.getters.getPhones.length === 0" class="px-4 text-center">
      <p>
        Install the mobile app on your android phone to start sending messages
      </p>
      <v-btn
        class="primary"
        :href="$store.getters.getAppData.appDownloadUrl"
        @click="
          $store.dispatch('addNotification', {
            type: 'info',
            message: 'Downloading the httpSMS Android App',
          })
        "
      >
        <v-icon>
          {{ mdiDownload }}
        </v-icon>
        Install App
      </v-btn>
    </div>
    <v-list two-line class="px-0 py-0" subheader>
      <v-list-item-group>
        <template v-for="thread in threads">
          <v-list-item
            :key="thread.id"
            :to="'/threads/' + thread.id"
            class="py-1"
            :class="{
              'px-6': $vuetify.breakpoint.mdAndDown,
              'px-3': $vuetify.breakpoint.lgAndUp,
            }"
          >
            <v-list-item-avatar :color="thread.color">
              <v-icon dark>{{ mdiAccount }}</v-icon>
            </v-list-item-avatar>
            <v-list-item-content>
              <v-list-item-title>
                {{ thread.contact | phoneNumber }}
              </v-list-item-title>
              <v-list-item-subtitle>
                {{ thread.last_message_content }}
              </v-list-item-subtitle>
            </v-list-item-content>
            <v-list-item-action>
              <v-list-item-action-text>
                {{ threadDate(thread.order_timestamp) }}
              </v-list-item-action-text>
            </v-list-item-action>
          </v-list-item>
        </template>
      </v-list-item-group>
    </v-list>
  </div>
</template>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'
import { mdiPlus, mdiDownload, mdiAccount } from '@mdi/js'

@Component
export default class MessageThread extends Vue {
  mdiPlus = mdiPlus
  mdiDownload = mdiDownload
  mdiAccount = mdiAccount

  get threads(): Array<MessageThread> {
    return this.$store.getters.getThreads
  }

  threadDate(date: string): string {
    return new Date(date).toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
    })
  }
}
</script>
