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
            <h5 class="text-h4 mb-3 mt-3">Current Plan</h5>
            <v-row v-if="$store.getters.getUser">
              <v-col md="6" xl="4">
                <v-alert dense text prominent color="info">
                  <v-row align="center">
                    <v-col cols="12">
                      <h1
                        class="subtitle-1 font-weight-bold text-uppercase mt-3"
                      >
                        <span v-if="isOnFreePlan">{{ plan.name }}</span>
                        <span v-else-if="subscriptionIsCancelled"
                          ><span class="warning--text">{{ plan.name }}</span> →
                          Free</span
                        >
                        <span v-else>{{ plan.name }}</span>
                      </h1>
                      <p
                        v-if="!isOnFreePlan && !subscriptionIsCancelled"
                        class="text--secondary"
                      >
                        Your next bill is for <b>${{ plan.price }}</b> on
                        <b>{{
                          new Date(
                            $store.getters.getUser.subscription_renews_at
                          ).toLocaleDateString()
                        }}</b>
                      </p>
                      <p
                        v-else-if="subscriptionIsCancelled"
                        class="text--secondary"
                      >
                        You will be downgraded to the <b>FREE</b> plan on
                        <b>{{
                          new Date(
                            $store.getters.getUser.subscription_ends_at
                          ).toLocaleDateString()
                        }}</b>
                      </p>
                      <p v-else class="text--secondary">
                        {{ totalMessages }}/{{ plan.messagesPerMonth }} messages
                      </p>
                    </v-col>
                    <v-col cols="12" class="d-flex mb-2 mt-n6">
                      <loading-button
                        v-if="!subscriptionIsCancelled && !isOnFreePlan"
                        color="primary"
                        :loading="loading"
                        @click="updateDetails"
                      >
                        Update Details
                      </loading-button>
                      <v-btn v-else color="primary" :href="checkoutURL"
                        >Upgrade Plan</v-btn
                      >
                      <v-spacer></v-spacer>
                      <v-dialog
                        v-if="!subscriptionIsCancelled && !isOnFreePlan"
                        v-model="dialog"
                        max-width="590"
                      >
                        <template #activator="{ on, attrs }">
                          <v-btn v-bind="attrs" color="error" text v-on="on">
                            Cancel Plan
                          </v-btn>
                        </template>
                        <v-card>
                          <v-card-text class="pt-4 mb-n6">
                            <h2 class="text--primary text-h5 mb-2">
                              Are you sure you want to cancel your subscription?
                            </h2>
                            <p>
                              You will be downgraded to the free plan at the end
                              of the current billing period on
                              <b>{{
                                new Date(
                                  $store.getters.getUser.subscription_renews_at
                                ).toLocaleDateString()
                              }}</b>
                            </p>
                          </v-card-text>
                          <v-card-actions>
                            <v-btn color="primary" @click="dialog = false">
                              Keep Subscription
                            </v-btn>
                            <v-spacer></v-spacer>
                            <loading-button
                              v-if="!isOnFreePlan"
                              :text="true"
                              :loading="loading"
                              color="error"
                              @click="cancelPlan"
                            >
                              Cancel Plan
                            </loading-button>
                          </v-card-actions>
                        </v-card>
                      </v-dialog>
                    </v-col>
                  </v-row>
                </v-alert>
              </v-col>
            </v-row>
            <h2 v-if="isOnFreePlan" class="text-h4 mt-4 mb-2">Upgrade Plan</h2>
            <v-row v-if="isOnFreePlan">
              <v-col cols="12" md="6" xl="4">
                <v-hover v-slot="{ hover }">
                  <v-card
                    :color="hover ? 'black' : 'default'"
                    :href="checkoutURL"
                    outlined
                  >
                    <v-card-text>
                      <v-row align="center">
                        <v-col class="grow">
                          <h1
                            class="subtitle-1 font-weight-bold text-uppercase mt-3"
                          >
                            Pro - Monthly
                          </h1>
                          <p class="text--secondary">5,000 messages</p>
                        </v-col>
                        <v-col class="shrink">
                          <span class="text-h5 text--primary">$6</span>/month
                        </v-col>
                      </v-row>
                    </v-card-text>
                  </v-card>
                </v-hover>
              </v-col>
              <v-col cols="12" md="6" xl="4">
                <v-hover v-slot="{ hover }">
                  <v-card
                    :color="hover ? 'black' : 'default'"
                    :href="checkoutURL"
                    outlined
                  >
                    <v-card-text>
                      <v-row align="center">
                        <v-col class="grow">
                          <h1
                            class="subtitle-1 font-weight-bold text-uppercase mt-3"
                          >
                            Pro - Yearly
                            <v-chip small color="primary" class="mt-n1"
                              >Save 20%</v-chip
                            >
                          </h1>
                          <p class="text--secondary">5,000 messages</p>
                        </v-col>
                        <v-col class="shrink">
                          <span class="text-h5 text--primary">$5</span>/month
                        </v-col>
                      </v-row>
                    </v-card-text>
                  </v-card>
                </v-hover>
              </v-col>
            </v-row>
            <h5 class="text-h4 mb-3 mt-4">Overview</h5>
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

type PaymentPlan = {
  name: string
  id: string
  price: number
  messagesPerMonth: number
}

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
      dialog: false,
      plans: [
        {
          name: 'Free',
          id: 'free',
          messagesPerMonth: 1000,
          price: 0,
        },
        {
          name: 'PRO - Monthly',
          id: 'pro-monthly',
          messagesPerMonth: 5000,
          price: 6,
        },
        {
          name: 'PRO - Yearly',
          id: 'pro-yearly',
          messagesPerMonth: 5000,
          price: 60,
        },
      ],
    }
  },
  head() {
    return {
      title: 'Usage & Billing - httpSMS',
    }
  },
  computed: {
    checkoutURL() {
      const url = new URL(this.$config.checkoutURL)
      const user = this.$store.getters.getAuthUser
      url.searchParams.append('checkout[custom][user_id]', user?.id)
      url.searchParams.append('checkout[email]', user?.email)
      url.searchParams.append('checkout[name]', user?.displayName)
      return url.toString()
    },
    plan(): PaymentPlan {
      return this.plans.find(
        (x) =>
          x.id === (this.$store.getters.getUser?.subscription_name || 'free')
      )!
    },
    isOnFreePlan(): boolean {
      return this.plan.id === 'free'
    },
    subscriptionIsCancelled(): boolean {
      return this.$store.getters.getUser?.subscription_status === 'cancelled'
    },
    totalMessages(): number {
      if (this.$store.getters.getBillingUsage == null) {
        return 0
      }
      return (
        this.$store.getters.getBillingUsage.sent_messages +
        this.$store.getters.getBillingUsage.received_messages
      )
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
        this.$store.dispatch('loadUser'),
        this.$store.dispatch('loadBillingUsage'),
        this.$store.dispatch('loadBillingUsageHistory'),
      ])
      this.loading = false
    },
    updateDetails() {
      this.loading = true
      this.$store
        .dispatch('getSubscriptionUpdateLink')
        .then((link: string) => {
          window.location.href = link
        })
        .catch(() => {
          this.loading = false
        })
    },
    cancelPlan() {
      this.loading = true
      this.$store
        .dispatch('cancelSubscription')
        .then(() => {
          this.$store.dispatch('addNotification', {
            message: 'Subscription cancelled successfully',
            type: 'success',
          })
          this.$router.push('/')
        })
        .catch(() => {
          this.loading = false
        })
    },
  },
})
</script>
