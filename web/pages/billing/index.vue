<template>
  <v-container fluid class="pa-0" :fill-height="$vuetify.breakpoint.lgAndUp">
    <div class="w-full h-full">
      <v-app-bar height="60" :dense="$vuetify.breakpoint.mdAndDown">
        <v-btn icon to="/">
          <v-icon>{{ mdiArrowLeft }}</v-icon>
        </v-btn>
        <v-toolbar-title>
          <div class="py-16">Account Usage</div>
        </v-toolbar-title>
        <v-progress-linear
          :active="loading"
          :indeterminate="loading"
          absolute
          bottom
        ></v-progress-linear>
      </v-app-bar>
      <v-container>
        <v-row>
          <v-col cols="12" md="9" offset-md="1" xl="8" offset-xl="2">
            <h5 class="text-h4 mb-3 mt-3">Overview</h5>
            <p class="text--secondary">
              This is the summary of the sent messages and received messages in
              <code
                v-if="$store.getters.getBillingUsage"
                class="font-weight-bold"
                >{{
                  $store.getters.getBillingUsage.start_timestamp | billingPeriod
                }}</code
              >.
            </p>
            <v-row v-if="$store.getters.getBillingUsage">
              <v-col cols="12" md="4">
                <v-alert
                  dark
                  dense
                  :icon="mdiCallMade"
                  prominent
                  type="info"
                  text
                >
                  <h2 class="text-h4 font-weight-bold mt-4">
                    {{ $store.getters.getBillingUsage.sent_messages | decimal }}
                  </h2>
                  <p class="text--secondary mt-n1">Messages Sent</p>
                </v-alert>
              </v-col>
              <v-col cols="12" md="4">
                <v-alert
                  dark
                  dense
                  :icon="mdiCallReceived"
                  prominent
                  type="warning"
                  text
                >
                  <div class="d-flex">
                    <h2 class="text-h4 font-weight-bold mt-4">
                      {{
                        $store.getters.getBillingUsage.received_messages |
                          decimal
                      }}
                    </h2>
                  </div>
                  <p class="text--secondary mt-n1">Messages Received</p>
                </v-alert>
              </v-col>
              <v-col cols="12" md="4">
                <v-alert
                  dense
                  :icon="mdiCreditCard"
                  prominent
                  type="success"
                  text
                >
                  <h2 class="text-h4 font-weight-bold mt-4">
                    {{ $store.getters.getBillingUsage.total_cost | money }}
                  </h2>
                  <p class="text--secondary mt-n1">Total Cost</p>
                </v-alert>
              </v-col>
            </v-row>
            <h5 class="text-h4 mb-3 mt-12">Usage History</h5>
            <p class="text--secondary">
              Summary of all the sent and received messages in the past 12
              months
            </p>
            <v-simple-table>
              <template #default>
                <thead>
                  <tr class="text-uppercase">
                    <th class="text-left">Period</th>
                    <th class="text-left">
                      Sent
                      <span v-if="$vuetify.breakpoint.lgAndUp">Messages</span>
                    </th>
                    <th class="text-left">
                      Received
                      <span v-if="$vuetify.breakpoint.lgAndUp">Messages</span>
                    </th>
                    <th class="text-right">
                      <span v-if="$vuetify.breakpoint.lgAndUp">Total</span> Cost
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="billingUsage in $store.getters
                      .getBillingUsageHistory"
                    :key="billingUsage.id"
                  >
                    <td>
                      {{ billingUsage.start_timestamp | billingPeriod }}
                    </td>
                    <td>
                      {{ billingUsage.sent_messages | decimal }}
                    </td>
                    <td>
                      {{ billingUsage.received_messages }}
                    </td>
                    <td class="text-right font-weight-bold">
                      {{ billingUsage.total_cost | money }}
                    </td>
                  </tr>
                </tbody>
              </template>
            </v-simple-table>
          </v-col>
        </v-row>
      </v-container>
    </div>
  </v-container>
</template>

<script lang="ts">
import Vue from 'vue'
import {
  mdiArrowLeft,
  mdiAccountCircle,
  mdiShieldCheck,
  mdiDelete,
  mdiContentSave,
  mdiEye,
  mdiEyeOff,
  mdiCallReceived,
  mdiCallMade,
  mdiCreditCard,
  mdiSquareEditOutline,
} from '@mdi/js'

export default Vue.extend({
  name: 'BillingIndex',
  middleware: ['auth'],
  data() {
    return {
      mdiEye,
      mdiEyeOff,
      mdiArrowLeft,
      mdiAccountCircle,
      mdiShieldCheck,
      mdiDelete,
      mdiContentSave,
      mdiCallReceived,
      mdiCallMade,
      mdiCreditCard,
      mdiSquareEditOutline,
      loading: true,
    }
  },
  head() {
    return {
      title: 'Usage & Billing - Http SMS',
    }
  },
  computed: {
    apiKey() {
      if (this.$store.getters.getUser === null) {
        return ''
      }
      return this.$store.getters.getUser.api_key
    },
  },
  async mounted() {
    if (!this.$store.getters.getAuthUser) {
      await this.$store.dispatch('setNextRoute', this.$route.path)
      await this.$router.push({ name: 'index' })
      setTimeout(this.loadData, 2000)
      return
    }
    await this.loadData()
  },

  methods: {
    async loadData() {
      await Promise.all([
        this.$store.dispatch('loadBillingUsage'),
        this.$store.dispatch('loadBillingUsageHistory'),
      ])
      this.loading = false
    },
  },
})
</script>
