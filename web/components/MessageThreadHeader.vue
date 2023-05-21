<template>
  <v-sheet
    class="pa-4 d-flex"
    :elevation="$vuetify.breakpoint.lgAndUp ? 0 : 2"
    :color="$vuetify.breakpoint.lgAndUp ? 'grey darken-4' : 'black'"
  >
    <div :class="{ 'px-2': $vuetify.breakpoint.mdAndDown }">
      <v-toolbar-title>
        <div class="d-flex pt-2" style="width: 245px">
          <v-select
            outlined
            dense
            :disabled="owners.length === 0"
            placeholder="Phone Numbers"
            :class="{ 'mb-n6': !$store.getters.getOwner }"
            :items="owners"
            :value="$store.getters.getOwner"
            @change="onOwnerChanged"
          >
          </v-select>
          <div style="width: 50px">
            <v-progress-circular
              v-if="$store.getters.getPolling"
              indeterminate
              :size="20"
              :width="1"
              class="mt-3 ml-2"
              color="success"
            ></v-progress-circular>
          </div>
        </div>
      </v-toolbar-title>
      <div v-if="$store.getters.getOwner" class="d-flex mt-n4">
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
              :to="{
                name: 'heartbeats-id',
                params: { id: $store.getters.getOwner },
              }"
              color="success"
              class="ml-2 mt-1 mb-n1"
              icon
              v-on="on"
            >
              <v-icon x-small>{{ mdiCircle }}</v-icon>
            </v-btn>
          </template>
          <h4>Last Heartbeat</h4>
          {{ $store.getters.getHeartbeat.timestamp | humanizeTime }} ago
        </v-tooltip>
      </div>
    </div>
    <v-spacer></v-spacer>
    <v-menu offset-y>
      <template #activator="{ on }">
        <v-btn icon text class="mt-2" v-on="on">
          <v-icon>{{ mdiDotsVertical }}</v-icon>
        </v-btn>
      </template>
      <v-list class="px-2" nav :dense="$vuetify.breakpoint.mdAndDown">
        <v-list-item-group v-model="selectedMenuItem">
          <v-list-item @click.prevent="toggleArchive">
            <v-list-item-icon class="pl-2">
              <v-icon v-if="!$store.getters.getIsArchived" dense>
                {{ mdiPackageDown }}
              </v-icon>
              <v-icon v-if="$store.getters.getIsArchived" dense>
                {{ mdiPackageUp }}
              </v-icon>
            </v-list-item-icon>
            <v-list-item-content class="ml-n3">
              <v-list-item-title class="pr-16 py-1">
                <span
                  v-if="!$store.getters.getIsArchived"
                  :class="{ 'pr-16': $vuetify.breakpoint.mdAndUp }"
                >
                  Archived
                </span>
                <span
                  v-if="$store.getters.getIsArchived"
                  :class="{ 'pr-16': $vuetify.breakpoint.mdAndUp }"
                >
                  Unarchived
                </span>
              </v-list-item-title>
            </v-list-item-content>
          </v-list-item>
          <v-list-item
            v-if="$store.getters.getOwner"
            :to="{ name: 'messages' }"
            exact
          >
            <v-list-item-icon class="pl-2">
              <v-icon dense>{{ mdiPlus }}</v-icon>
            </v-list-item-icon>
            <v-list-item-content class="ml-n3">
              <v-list-item-title class="pr-16 py-1">
                <span :class="{ 'pr-16': $vuetify.breakpoint.mdAndUp }">
                  New Message
                </span>
              </v-list-item-title>
            </v-list-item-content>
          </v-list-item>
          <v-list-item :to="{ name: 'settings' }" exact>
            <v-list-item-icon class="pl-2">
              <v-icon dense>{{ mdiAccountCog }}</v-icon>
            </v-list-item-icon>
            <v-list-item-content class="ml-n3">
              <v-list-item-title class="pr-16 py-1">
                <span :class="{ 'pr-16': $vuetify.breakpoint.mdAndUp }">
                  Settings
                </span>
              </v-list-item-title>
            </v-list-item-content>
          </v-list-item>
          <v-list-item
            v-if="$store.getters.getOwner"
            :href="$store.getters.getAppData.appDownloadUrl"
            exact
          >
            <v-list-item-icon class="pl-2">
              <v-icon dense>{{ mdiDownload }}</v-icon>
            </v-list-item-icon>
            <v-list-item-content class="ml-n3">
              <v-list-item-title class="pr-16 py-1">
                <span :class="{ 'pr-16': $vuetify.breakpoint.mdAndUp }">
                  Install App
                </span>
              </v-list-item-title>
            </v-list-item-content>
          </v-list-item>
          <v-list-item :to="{ name: 'billing' }" exact>
            <v-list-item-icon class="pl-2">
              <v-icon dense>{{ mdiFinance }}</v-icon>
            </v-list-item-icon>
            <v-list-item-content class="ml-n3">
              <v-list-item-title class="pr-16 py-1">
                <span :class="{ 'pr-16': $vuetify.breakpoint.mdAndUp }">
                  Usage & Billing
                </span>
              </v-list-item-title>
            </v-list-item-content>
          </v-list-item>
          <v-list-item @click.prevent="logout">
            <v-list-item-icon class="pl-2">
              <v-icon dense>{{ mdiLogout }}</v-icon>
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
import {
  mdiPlus,
  mdiAccountCog,
  mdiLogout,
  mdiDownload,
  mdiFinance,
  mdiPackageUp,
  mdiPackageDown,
  mdiDotsVertical,
  mdiCircle,
} from '@mdi/js'
import { SelectItem } from '~/types'
import { Phone } from '~/models/phone'

@Component
export default class MessageThreadHeader extends Vue {
  selectedMenuItem = -1
  mdiPlus = mdiPlus
  mdiAccountCog = mdiAccountCog
  mdiLogout = mdiLogout
  mdiDownload = mdiDownload
  mdiPackageUp = mdiPackageUp
  mdiFinance = mdiFinance
  mdiPackageDown = mdiPackageDown
  mdiDotsVertical = mdiDotsVertical
  mdiCircle = mdiCircle

  get owners(): Array<SelectItem> {
    return this.$store.getters.getPhones.map((phone: Phone): SelectItem => {
      return {
        text: this.$options.filters?.phoneNumber(phone.phone_number),
        value: phone.phone_number,
      }
    })
  }

  async onOwnerChanged(owner: string) {
    await this.$store.dispatch('setOwner', owner)
    if (this.$route.name !== 'threads') {
      await this.$store.dispatch('setThreadId', null)
      await this.$router.push({ name: 'threads' })
      return
    }

    await this.$store.dispatch('loadThreads')
  }

  async toggleArchive() {
    await this.$store.dispatch('toggleArchive')

    setTimeout(() => {
      this.selectedMenuItem = -1
    }, 1000)

    if (this.$route.name !== 'threads') {
      await this.$store.dispatch('setThreadId', null)
      await this.$router.push({ name: 'threads' })
      return
    }
    await this.$store.dispatch('loadThreads')
  }

  logout(): void {
    this.$fire.auth.signOut().then(() => {
      this.$store.dispatch('setAuthUser', null)
      this.$store.dispatch('resetState')
      this.$store.dispatch('addNotification', {
        type: 'info',
        message: 'You have successfully logged out',
      })
      this.$router.push({ name: 'index' })
    })
  }
}
</script>
