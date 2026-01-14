<template>
  <v-container
    fluid
    class="px-0 pt-0"
    :fill-height="$vuetify.breakpoint.lgAndUp"
  >
    <div class="w-full h-full">
      <v-app-bar height="60" :dense="$vuetify.breakpoint.mdAndDown">
        <v-btn icon to="/threads">
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
                        v-if="
                          !isOnFreePlan &&
                          !isOnLifetimePlan &&
                          !subscriptionIsCancelled
                        "
                        class="text--secondary"
                      >
                        Your next bill is for <b>${{ plan.price }}</b> on
                        <b>{{
                          new Date(
                            $store.getters.getUser.subscription_renews_at,
                          ).toLocaleDateString()
                        }}</b>
                      </p>
                      <p v-if="isOnLifetimePlan" class="text--secondary">
                        You are on the life time plan which costs
                        <b>${{ plan.price }}</b>
                      </p>
                      <p
                        v-else-if="subscriptionIsCancelled"
                        class="text--secondary"
                      >
                        You will be downgraded to the <b>FREE</b> plan on
                        <b>{{
                          new Date(
                            $store.getters.getUser.subscription_ends_at,
                          ).toLocaleDateString()
                        }}</b>
                      </p>
                      <p v-else class="text--secondary">
                        {{ totalMessages }}/{{ plan.messagesPerMonth }} messages
                      </p>
                    </v-col>
                    <v-col cols="12" class="d-flex mb-2 mt-n6">
                      <loading-button
                        v-if="
                          !subscriptionIsCancelled &&
                          !isOnFreePlan &&
                          !isOnLifetimePlan
                        "
                        color="primary"
                        :loading="loading"
                        @click="updateDetails"
                      >
                        Update Plan
                      </loading-button>
                      <v-btn
                        v-else-if="!isOnLifetimePlan"
                        color="primary"
                        :href="checkoutURL"
                        >Upgrade Plan</v-btn
                      >
                      <v-spacer></v-spacer>
                      <v-dialog
                        v-if="
                          !subscriptionIsCancelled &&
                          !isOnFreePlan &&
                          !isOnLifetimePlan
                        "
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
                                  $store.getters.getUser.subscription_renews_at,
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
                          <p class="text--secondary">5,000 messages monthly</p>
                        </v-col>
                        <v-col class="shrink">
                          <span class="text-h5 text--primary">$10</span>/month
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
                              >2 months free</v-chip
                            >
                          </h1>
                          <p class="text--secondary">5,000 messages monthly</p>
                        </v-col>
                        <v-col class="shrink">
                          <span class="text-h5 text--primary">$100</span>/year
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
                    :href="enterpriseCheckoutURL"
                    outlined
                  >
                    <v-card-text>
                      <v-row align="center">
                        <v-col class="grow">
                          <h1
                            class="subtitle-1 font-weight-bold text-uppercase mt-3"
                          >
                            100k - Monthly
                          </h1>
                          <p class="text--secondary">
                            100,000 messages monthly
                          </p>
                        </v-col>
                        <v-col class="shrink">
                          <span class="text-h5 text--primary">$175</span>/month
                        </v-col>
                      </v-row>
                    </v-card-text>
                  </v-card>
                </v-hover>
              </v-col>
            </v-row>
            <h5 class="text-h4 mb-3 mt-8">Overview</h5>
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
                        $store.getters.getBillingUsage.received_messages
                          | decimal
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
            <template v-if="$store.getters.getUser?.subscription_id != null">
              <h5 class="text-h4 mb-3 mt-8">Subscription Payments</h5>
              <p class="text--secondary">
                This is a list of your last 10 subscription payments made using
                our payment provider
                <a
                  class="text-decoration-none"
                  href="https://www.lemonsqueezy.com"
                  >Lemon Squeezy</a
                >.
              </p>
              <v-progress-circular
                v-if="payments == null && loadingSubscriptionPayments"
                :size="20"
                :width="2"
                color="primary"
                indeterminate
              ></v-progress-circular>
              <v-simple-table v-if="payments">
                <template #default>
                  <thead>
                    <tr class="text-uppercase">
                      <th v-if="$vuetify.breakpoint.lgAndUp" class="text-left">
                        ID
                      </th>
                      <th class="text-left">Timestamp</th>
                      <th class="text-left">Status</th>
                      <th v-if="$vuetify.breakpoint.lgAndUp" class="text-left">
                        Tax
                      </th>
                      <th class="text-left">Total</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="payment in payments.data" :key="payment.id">
                      <td v-if="$vuetify.breakpoint.lgAndUp">
                        {{ payment.id }}
                      </td>
                      <td>
                        {{ payment.attributes.created_at | timestamp }}
                      </td>
                      <td>
                        <v-chip
                          v-if="payment.attributes.status === 'paid'"
                          color="success"
                        >
                          <v-avatar size="4" left class="green darken-4">
                            <v-icon small>{{ mdiCheck }}</v-icon>
                          </v-avatar>
                          {{ payment.attributes.status_formatted }}
                        </v-chip>
                        <v-chip v-else color="error">
                          <v-avatar size="4" left class="red darken-4">
                            <v-icon small>{{ mdiAlert }}</v-icon>
                          </v-avatar>
                          {{ payment.attributes.status_formatted }}
                        </v-chip>
                      </td>
                      <td v-if="$vuetify.breakpoint.lgAndUp">
                        {{ payment.attributes.tax_formatted }}
                      </td>
                      <td class="font-weight-bold">
                        {{ payment.attributes.total_formatted }}
                      </td>
                      <td class="text-right">
                        <v-btn
                          color="primary"
                          small
                          @click="showInvoiceDialog(payment)"
                        >
                          <v-icon left>{{ mdiInvoice }}</v-icon>
                          Invoice
                        </v-btn>
                      </td>
                    </tr>
                  </tbody>
                </template>
              </v-simple-table>
            </template>
            <h5 class="text-h4 mb-3 mt-8">Usage History</h5>
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
    <v-dialog
      v-model="subscriptionInvoiceDialog"
      persistent
      overlay-opacity="0.9"
      max-width="600px"
    >
      <v-card>
        <v-card-title class="text-h4"> Generate Invoice </v-card-title>
        <v-card-subtitle class="mt-n1">
          Create an invoice for your
          <b>{{ selectedPayment?.attributes.total_formatted }}</b> payment on
          {{ selectedPayment?.attributes.created_at | timestamp }}
        </v-card-subtitle>
        <v-card-text>
          <v-container>
            <v-row>
              <v-col cols="12">
                <v-text-field
                  v-model="invoiceFormName"
                  dense
                  :disabled="loading"
                  :error="errorMessages.has('name')"
                  :error-messages="errorMessages.get('name')"
                  label="Name"
                  placeholder="e.g Acme Corporation"
                  persistent-placeholder
                  outlined
                ></v-text-field>
              </v-col>
              <v-col cols="12">
                <v-text-field
                  v-model="invoiceFormAddress"
                  dense
                  :disabled="loading"
                  :error="errorMessages.has('address')"
                  :error-messages="errorMessages.get('address')"
                  label="Address"
                  placeholder="e.g 221B Baker Street"
                  persistent-placeholder
                  outlined
                ></v-text-field>
              </v-col>
            </v-row>
            <v-row>
              <v-col cols="6">
                <v-text-field
                  v-model="invoiceFormCity"
                  dense
                  :disabled="loading"
                  :error="errorMessages.has('city')"
                  :error-messages="errorMessages.get('city')"
                  label="City"
                  placeholder="e.g Los Angeles"
                  persistent-placeholder
                  outlined
                ></v-text-field>
              </v-col>
              <v-col cols="6">
                <v-text-field
                  v-if="invoiceStateOptions.length === 0"
                  v-model="invoiceFormState"
                  dense
                  :disabled="loading"
                  :error="errorMessages.has('state')"
                  :error-messages="errorMessages.get('state')"
                  label="State"
                  placeholder="e.g CA"
                  persistent-placeholder
                  outlined
                ></v-text-field>
                <v-autocomplete
                  v-else
                  v-model="invoiceFormState"
                  dense
                  :disabled="loading"
                  :error="errorMessages.has('state')"
                  :error-messages="errorMessages.get('state')"
                  :items="invoiceStateOptions"
                  label="State"
                  outlined
                  placeholder="e.g CA"
                  persistent-placeholder
                ></v-autocomplete>
              </v-col>
            </v-row>
            <v-row>
              <v-col cols="6">
                <v-text-field
                  v-model="invoiceFormZipCode"
                  dense
                  :disabled="loading"
                  :error="errorMessages.has('zip_code')"
                  :error-messages="errorMessages.get('zip_code')"
                  label="Zip Code"
                  placeholder="e.g 46001"
                  persistent-placeholder
                  outlined
                ></v-text-field>
              </v-col>
              <v-col cols="6">
                <v-autocomplete
                  v-model="invoiceFormCountry"
                  dense
                  :disabled="loading"
                  :error="errorMessages.has('country')"
                  :error-messages="errorMessages.get('country')"
                  :items="countries"
                  label="Country"
                  placeholder="e.g United States"
                  outlined
                  persistent-placeholder
                ></v-autocomplete>
              </v-col>
            </v-row>
            <v-row>
              <v-col cols="12">
                <v-textarea
                  v-model="invoiceFormNotes"
                  dense
                  :disabled="loading"
                  :error="errorMessages.has('notes')"
                  :error-messages="errorMessages.get('notes')"
                  rows="3"
                  label="Notes (optional)"
                  placeholder="e.g Thanks for doing business with us!"
                  persistent-placeholder
                  outlined
                ></v-textarea>
              </v-col>
            </v-row>
          </v-container>
        </v-card-text>
        <v-card-actions class="mt-n8 pb-4">
          <v-btn :loading="loading" color="primary" @click="generateInvoice">
            <v-icon left>{{ mdiDownloadOutline }}</v-icon>
            Download Invoice
          </v-btn>
          <v-spacer></v-spacer>
          <v-btn color="error" text @click="subscriptionInvoiceDialog = false">
            Close
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<script lang="ts">
import Vue from 'vue'
import {
  mdiArrowLeft,
  mdiAccountCircle,
  mdiShieldCheck,
  mdiDelete,
  mdiDownloadOutline,
  mdiCog,
  mdiContentSave,
  mdiCheck,
  mdiAlert,
  mdiInvoice,
  mdiEye,
  mdiEyeOff,
  mdiCallReceived,
  mdiCallMade,
  mdiCreditCard,
  mdiSquareEditOutline,
} from '@mdi/js'
import {
  RequestsUserPaymentInvoice,
  ResponsesUserSubscriptionPaymentsResponse,
} from '~/models/api'
import { ErrorMessages } from '~/plugins/errors'

type PaymentPlan = {
  name: string
  id: string
  price: number
  messagesPerMonth: number
}

type subscriptionPayment = {
  attributes: {
    created_at: string
    total_formatted: string
  }
  id: string
}

export default Vue.extend({
  name: 'BillingIndex',
  middleware: ['auth'],
  data() {
    return {
      mdiEye,
      mdiEyeOff,
      mdiArrowLeft,
      mdiDownloadOutline,
      mdiAccountCircle,
      mdiCheck,
      mdiAlert,
      mdiInvoice,
      mdiShieldCheck,
      mdiDelete,
      mdiCog,
      mdiContentSave,
      mdiCallReceived,
      mdiCallMade,
      mdiCreditCard,
      mdiSquareEditOutline,
      loading: true,
      loadingSubscriptionPayments: false,
      dialog: false,
      payments: null as ResponsesUserSubscriptionPaymentsResponse | null,
      selectedPayment: null as subscriptionPayment | null,
      errorMessages: new ErrorMessages(),
      invoiceFormName: '',
      invoiceFormAddress: '',
      invoiceFormCity: '',
      invoiceFormState: '',
      invoiceFormZipCode: '',
      invoiceFormCountry: '',
      invoiceFormNotes: '',
      subscriptionInvoiceDialog: false,
      countries: [
        { text: 'Afghanistan', value: 'AF' },
        { text: 'Åland Islands', value: 'AX' },
        { text: 'Albania', value: 'AL' },
        { text: 'Algeria', value: 'DZ' },
        { text: 'American Samoa', value: 'AS' },
        { text: 'Andorra', value: 'AD' },
        { text: 'Angola', value: 'AO' },
        { text: 'Anguilla', value: 'AI' },
        { text: 'Antarctica', value: 'AQ' },
        { text: 'Antigua and Barbuda', value: 'AG' },
        { text: 'Argentina', value: 'AR' },
        { text: 'Armenia', value: 'AM' },
        { text: 'Aruba', value: 'AW' },
        { text: 'Australia', value: 'AU' },
        { text: 'Austria', value: 'AT' },
        { text: 'Azerbaijan', value: 'AZ' },
        { text: 'Bahamas', value: 'BS' },
        { text: 'Bahrain', value: 'BH' },
        { text: 'Bangladesh', value: 'BD' },
        { text: 'Barbados', value: 'BB' },
        { text: 'Belarus', value: 'BY' },
        { text: 'Belgium', value: 'BE' },
        { text: 'Belize', value: 'BZ' },
        { text: 'Benin', value: 'BJ' },
        { text: 'Bermuda', value: 'BM' },
        { text: 'Bhutan', value: 'BT' },
        { text: 'Bolivia', value: 'BO' },
        { text: 'Bonaire', value: 'BQ' },
        { text: 'Bosnia and Herzegovina', value: 'BA' },
        { text: 'Botswana', value: 'BW' },
        { text: 'Bouvet Island', value: 'BV' },
        { text: 'Brazil', value: 'BR' },
        { text: 'British Indian Ocean', value: 'IO' },
        { text: 'Brunei Darussalam', value: 'BN' },
        { text: 'Bulgaria', value: 'BG' },
        { text: 'Burkina Faso', value: 'BF' },
        { text: 'Burundi', value: 'BI' },
        { text: 'Cabo Verde', value: 'CV' },
        { text: 'Cambodia', value: 'KH' },
        { text: 'Cameroon', value: 'CM' },
        { text: 'Canada', value: 'CA' },
        { text: 'Cayman Islands', value: 'KY' },
        { text: 'Central African Republic', value: 'CF' },
        { text: 'Chad', value: 'TD' },
        { text: 'Chile', value: 'CL' },
        { text: 'China', value: 'CN' },
        { text: 'Christmas Island', value: 'CX' },
        { text: 'Cocos (Keeling) Islands', value: 'CC' },
        { text: 'Colombia', value: 'CO' },
        { text: 'Comoros', value: 'KM' },
        { text: 'Congo', value: 'CG' },
        { text: 'Congo', value: 'CD' },
        { text: 'Cook Islands', value: 'CK' },
        { text: 'Costa Rica', value: 'CR' },
        { text: "Côte d'Ivoire", value: 'CI' },
        { text: 'Cuba', value: 'CU' },
        { text: 'Curaçao', value: 'CW' },
        { text: 'Cyprus', value: 'CY' },
        { text: 'Czechia', value: 'CZ' },
        { text: 'Denmark', value: 'DK' },
        { text: 'Djibouti', value: 'DJ' },
        { text: 'Dominica', value: 'DM' },
        { text: 'Dominican Republic', value: 'DO' },
        { text: 'Ecuador', value: 'EC' },
        { text: 'Egypt', value: 'EG' },
        { text: 'El Salvador', value: 'SV' },
        { text: 'Equatorial Guinea', value: 'GQ' },
        { text: 'Eritrea', value: 'ER' },
        { text: 'Estonia', value: 'EE' },
        { text: 'Eswatini', value: 'SZ' },
        { text: 'Ethiopia', value: 'ET' },
        { text: 'Falkland Islands', value: 'FK' },
        { text: 'Faroe Islands', value: 'FO' },
        { text: 'Fiji', value: 'FJ' },
        { text: 'Finland', value: 'FI' },
        { text: 'France', value: 'FR' },
        { text: 'French Guiana', value: 'GF' },
        { text: 'French Polynesia', value: 'PF' },
        { text: 'French Southern Territories', value: 'TF' },
        { text: 'Gabon', value: 'GA' },
        { text: 'Gambia', value: 'GM' },
        { text: 'Georgia', value: 'GE' },
        { text: 'Germany', value: 'DE' },
        { text: 'Ghana', value: 'GH' },
        { text: 'Gibraltar', value: 'GI' },
        { text: 'Greece', value: 'GR' },
        { text: 'Greenland', value: 'GL' },
        { text: 'Grenada', value: 'GD' },
        { text: 'Guadeloupe', value: 'GP' },
        { text: 'Guam', value: 'GU' },
        { text: 'Guatemala', value: 'GT' },
        { text: 'Guernsey', value: 'GG' },
        { text: 'Guinea', value: 'GN' },
        { text: 'Guinea-Bissau', value: 'GW' },
        { text: 'Guyana', value: 'GY' },
        { text: 'Haiti', value: 'HT' },
        { text: 'Heard Island and McDonald Islands', value: 'HM' },
        { text: 'Holy See', value: 'VA' },
        { text: 'Honduras', value: 'HN' },
        { text: 'Hong Kong', value: 'HK' },
        { text: 'Hungary', value: 'HU' },
        { text: 'Iceland', value: 'IS' },
        { text: 'India', value: 'IN' },
        { text: 'Indonesia', value: 'ID' },
        { text: 'Iran', value: 'IR' },
        { text: 'Iraq', value: 'IQ' },
        { text: 'Ireland', value: 'IE' },
        { text: 'Isle of Man', value: 'IM' },
        { text: 'Israel', value: 'IL' },
        { text: 'Italy', value: 'IT' },
        { text: 'Jamaica', value: 'JM' },
        { text: 'Japan', value: 'JP' },
        { text: 'Jersey', value: 'JE' },
        { text: 'Jordan', value: 'JO' },
        { text: 'Kazakhstan', value: 'KZ' },
        { text: 'Kenya', value: 'KE' },
        { text: 'Kiribati', value: 'KI' },
        { text: 'North Korea', value: 'KP' },
        { text: 'South Korea', value: 'KR' },
        { text: 'Kuwait', value: 'KW' },
        { text: 'Kyrgyzstan', value: 'KG' },
        { text: 'Lao People’s Democratic Republic', value: 'LA' },
        { text: 'Latvia', value: 'LV' },
        { text: 'Lebanon', value: 'LB' },
        { text: 'Lesotho', value: 'LS' },
        { text: 'Liberia', value: 'LR' },
        { text: 'Libya', value: 'LY' },
        { text: 'Liechtenstein', value: 'LI' },
        { text: 'Lithuania', value: 'LT' },
        { text: 'Luxembourg', value: 'LU' },
        { text: 'Macao', value: 'MO' },
        { text: 'Madagascar', value: 'MG' },
        { text: 'Malawi', value: 'MW' },
        { text: 'Malaysia', value: 'MY' },
        { text: 'Maldives', value: 'MV' },
        { text: 'Mali', value: 'ML' },
        { text: 'Malta', value: 'MT' },
        { text: 'Marshall Islands', value: 'MH' },
        { text: 'Martinique', value: 'MQ' },
        { text: 'Mauritania', value: 'MR' },
        { text: 'Mauritius', value: 'MU' },
        { text: 'Mayotte', value: 'YT' },
        { text: 'Mexico', value: 'MX' },
        { text: 'Micronesia', value: 'FM' },
        { text: 'Moldova', value: 'MD' },
        { text: 'Monaco', value: 'MC' },
        { text: 'Mongolia', value: 'MN' },
        { text: 'Montenegro', value: 'ME' },
        { text: 'Montserrat', value: 'MS' },
        { text: 'Morocco', value: 'MA' },
        { text: 'Mozambique', value: 'MZ' },
        { text: 'Myanmar', value: 'MM' },
        { text: 'Namibia', value: 'NA' },
        { text: 'Nauru', value: 'NR' },
        { text: 'Nepal', value: 'NP' },
        { text: 'Netherlands', value: 'NL' },
        { text: 'New Caledonia', value: 'NC' },
        { text: 'New Zealand', value: 'NZ' },
        { text: 'Nicaragua', value: 'NI' },
        { text: 'Niger', value: 'NE' },
        { text: 'Nigeria', value: 'NG' },
        { text: 'Niue', value: 'NU' },
        { text: 'Norfolk Island', value: 'NF' },
        { text: 'North Macedonia', value: 'MK' },
        { text: 'Northern Mariana Islands', value: 'MP' },
        { text: 'Norway', value: 'NO' },
        { text: 'Oman', value: 'OM' },
        { text: 'Pakistan', value: 'PK' },
        { text: 'Palau', value: 'PW' },
        { text: 'Panama', value: 'PA' },
        { text: 'Papua New Guinea', value: 'PG' },
        { text: 'Paraguay', value: 'PY' },
        { text: 'Peru', value: 'PE' },
        { text: 'Philippines', value: 'PH' },
        { text: 'Pitcairn', value: 'PN' },
        { text: 'Poland', value: 'PL' },
        { text: 'Portugal', value: 'PT' },
        { text: 'Puerto Rico', value: 'PR' },
        { text: 'Qatar', value: 'QA' },
        { text: 'Réunion', value: 'RE' },
        { text: 'Romania', value: 'RO' },
        { text: 'Russian Federation', value: 'RU' },
        { text: 'Rwanda', value: 'RW' },
        { text: 'Saint Barthélemy', value: 'BL' },
        { text: 'Saint Helena, Ascension and Tristan da Cunha', value: 'SH' },
        { text: 'Saint Kitts and Nevis', value: 'KN' },
        { text: 'Saint Lucia', value: 'LC' },
        { text: 'Saint Martin (French part)', value: 'MF' },
        { text: 'Saint Pierre and Miquelon', value: 'PM' },
        { text: 'Saint Vincent and the Grenadines', value: 'VC' },
        { text: 'Samoa', value: 'WS' },
        { text: 'San Marino', value: 'SM' },
        { text: 'Sao Tome and Principe', value: 'ST' },
        { text: 'Saudi Arabia', value: 'SA' },
        { text: 'Senegal', value: 'SN' },
        { text: 'Serbia', value: 'RS' },
        { text: 'Seychelles', value: 'SC' },
        { text: 'Sierra Leone', value: 'SL' },
        { text: 'Singapore', value: 'SG' },
        { text: 'Slovakia', value: 'SK' },
        { text: 'Slovenia', value: 'SI' },
        { text: 'Solomon Islands', value: 'SB' },
        { text: 'Somalia', value: 'SO' },
        { text: 'South Africa', value: 'ZA' },
        { text: 'South Georgia and the South Sandwich Islands', value: 'GS' },
        { text: 'South Sudan', value: 'SS' },
        { text: 'Spain', value: 'ES' },
        { text: 'Sri Lanka', value: 'LK' },
        { text: 'Sudan', value: 'SD' },
        { text: 'Suriname', value: 'SR' },
        { text: 'Svalbard and Jan Mayen', value: 'SJ' },
        { text: 'Sweden', value: 'SE' },
        { text: 'Switzerland', value: 'CH' },
        { text: 'Syrian Arab Republic', value: 'SY' },
        { text: 'Taiwan, Province of China', value: 'TW' },
        { text: 'Tajikistan', value: 'TJ' },
        { text: 'Tanzania, United Republic of', value: 'TZ' },
        { text: 'Thailand', value: 'TH' },
        { text: 'Timor-Leste', value: 'TL' },
        { text: 'Togo', value: 'TG' },
        { text: 'Tokelau', value: 'TK' },
        { text: 'Tonga', value: 'TO' },
        { text: 'Trinidad and Tobago', value: 'TT' },
        { text: 'Tunisia', value: 'TN' },
        { text: 'Turkey', value: 'TR' },
        { text: 'Turkmenistan', value: 'TM' },
        { text: 'Turks and Caicos Islands', value: 'TC' },
        { text: 'Tuvalu', value: 'TV' },
        { text: 'Uganda', value: 'UG' },
        { text: 'Ukraine', value: 'UA' },
        { text: 'United Arab Emirates', value: 'AE' },
        { text: 'United Kingdom', value: 'GB' },
        { text: 'United States', value: 'US' },
        { text: 'United States Minor Outlying Islands', value: 'UM' },
        { text: 'Uruguay', value: 'UY' },
        { text: 'Uzbekistan', value: 'UZ' },
        { text: 'Vanuatu', value: 'VU' },
        { text: 'Venezuela', value: 'VE' },
        { text: 'Viet Nam', value: 'VN' },
        { text: 'Virgin Islands (British)', value: 'VG' },
        { text: 'Virgin Islands (U.S.)', value: 'VI' },
        { text: 'Wallis and Futuna', value: 'WF' },
        { text: 'Western Sahara', value: 'EH' },
        { text: 'Yemen', value: 'YE' },
        { text: 'Zambia', value: 'ZM' },
        { text: 'Zimbabwe', value: 'ZW' },
      ],
      plans: [
        {
          name: 'Free',
          id: 'free',
          messagesPerMonth: 200,
          price: 0,
        },
        {
          name: 'PRO - Monthly',
          id: 'pro-monthly',
          messagesPerMonth: 5000,
          price: 10,
        },
        {
          name: 'PRO - Yearly',
          id: 'pro-yearly',
          messagesPerMonth: 5000,
          price: 100,
        },
        {
          name: 'Ultra - Monthly',
          id: 'ultra-monthly',
          messagesPerMonth: 10000,
          price: 20,
        },
        {
          name: 'Ultra - Yearly',
          id: 'ultra-yearly',
          messagesPerMonth: 10000,
          price: 200,
        },
        {
          name: '20k - Monthly',
          id: '20k-monthly',
          messagesPerMonth: 20000,
          price: 35,
        },
        {
          name: '20k - Yearly',
          id: '20k-yearly',
          messagesPerMonth: 20000,
          price: 350,
        },
        {
          name: '50k - Monthly',
          id: '50k-monthly',
          messagesPerMonth: 50000,
          price: 89,
        },
        {
          name: '100k - Monthly',
          id: '100k-monthly',
          messagesPerMonth: 100000,
          price: 175,
        },
        {
          name: '200k - Monthly',
          id: '200k-monthly',
          messagesPerMonth: 200000,
          price: 350,
        },
        {
          name: 'PRO - Lifetime',
          id: 'pro-lifetime',
          messagesPerMonth: 10000,
          price: 1000,
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
    invoiceStateOptions() {
      if (this.invoiceFormCountry === 'US') {
        return [
          { text: 'Alabama', value: 'AL' },
          { text: 'Alaska', value: 'AK' },
          { text: 'Arizona', value: 'AZ' },
          { text: 'Arkansas', value: 'AR' },
          { text: 'California', value: 'CA' },
          { text: 'Colorado', value: 'CO' },
          { text: 'Connecticut', value: 'CT' },
          { text: 'Delaware', value: 'DE' },
          { text: 'Florida', value: 'FL' },
          { text: 'Georgia', value: 'GA' },
          { text: 'Hawaii', value: 'HI' },
          { text: 'Idaho', value: 'ID' },
          { text: 'Illinois', value: 'IL' },
          { text: 'Indiana', value: 'IN' },
          { text: 'Iowa', value: 'IA' },
          { text: 'Kansas', value: 'KS' },
          { text: 'Kentucky', value: 'KY' },
          { text: 'Louisiana', value: 'LA' },
          { text: 'Maine', value: 'ME' },
          { text: 'Maryland', value: 'MD' },
          { text: 'Massachusetts', value: 'MA' },
          { text: 'Michigan', value: 'MI' },
          { text: 'Minnesota', value: 'MN' },
          { text: 'Mississippi', value: 'MS' },
          { text: 'Missouri', value: 'MO' },
          { text: 'Montana', value: 'MT' },
          { text: 'Nebraska', value: 'NE' },
          { text: 'Nevada', value: 'NV' },
          { text: 'New Hampshire', value: 'NH' },
          { text: 'New Jersey', value: 'NJ' },
          { text: 'New Mexico', value: 'NM' },
          { text: 'New York', value: 'NY' },
          { text: 'North Carolina', value: 'NC' },
          { text: 'North Dakota', value: 'ND' },
          { text: 'Ohio', value: 'OH' },
          { text: 'Oklahoma', value: 'OK' },
          { text: 'Oregon', value: 'OR' },
          { text: 'Pennsylvania', value: 'PA' },
          { text: 'Rhode Island', value: 'RI' },
          { text: 'South Carolina', value: 'SC' },
          { text: 'South Dakota', value: 'SD' },
          { text: 'Tennessee', value: 'TN' },
          { text: 'Texas', value: 'TX' },
          { text: 'Utah', value: 'UT' },
          { text: 'Vermont', value: 'VT' },
          { text: 'Virginia', value: 'VA' },
          { text: 'Washington', value: 'WA' },
          { text: 'West Virginia', value: 'WV' },
          { text: 'Wisconsin', value: 'WI' },
          { text: 'Wyoming', value: 'WY' },
          { text: 'District of Columbia', value: 'DC' },
        ]
      }
      if (this.invoiceFormCountry === 'CA') {
        return [
          { text: 'Alberta', value: 'AB' },
          { text: 'British Columbia', value: 'BC' },
          { text: 'Manitoba', value: 'MB' },
          { text: 'New Brunswick', value: 'NB' },
          { text: 'Newfoundland and Labrador', value: 'NL' },
          { text: 'Nova Scotia', value: 'NS' },
          { text: 'Ontario', value: 'ON' },
          { text: 'Prince Edward Island', value: 'PE' },
          { text: 'Quebec', value: 'QC' },
          { text: 'Saskatchewan', value: 'SK' },
          { text: 'Northwest Territories', value: 'NT' },
          { text: 'Nunavut', value: 'NU' },
          { text: 'Yukon', value: 'YT' },
        ]
      }
      return []
    },
    checkoutURL() {
      const url = new URL(this.$config.checkoutURL)
      const user = this.$store.getters.getAuthUser
      url.searchParams.append('checkout[custom][user_id]', user?.id)
      url.searchParams.append('checkout[email]', user?.email)
      url.searchParams.append('checkout[name]', user?.displayName)
      return url.toString()
    },
    enterpriseCheckoutURL() {
      const url = new URL(this.$config.enterpriseCheckoutURL)
      const user = this.$store.getters.getAuthUser
      url.searchParams.append('checkout[custom][user_id]', user?.id)
      url.searchParams.append('checkout[email]', user?.email)
      url.searchParams.append('checkout[name]', user?.displayName)
      return url.toString()
    },

    plan(): PaymentPlan {
      return this.plans.find(
        (x) =>
          x.id === (this.$store.getters.getUser?.subscription_name || 'free'),
      )!
    },
    isOnFreePlan(): boolean {
      return this.plan.id === 'free'
    },
    isOnLifetimePlan(): boolean {
      return this.plan.id === 'pro-lifetime'
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
      this.loadSubscriptionInvoices()
    },

    loadSubscriptionInvoices() {
      this.loadingSubscriptionPayments = true
      this.$store
        .dispatch('indexSubscriptionPayments')
        .then((response: ResponsesUserSubscriptionPaymentsResponse) => {
          this.payments = response
        })
        .finally(() => {
          this.loadingSubscriptionPayments = false
        })
    },

    generateInvoice() {
      this.errorMessages = new ErrorMessages()
      this.loading = true
      this.$store
        .dispatch('generateSubscriptionPaymentInvoice', {
          subscriptionInvoiceId: this.selectedPayment?.id || '',
          request: {
            name: this.invoiceFormName,
            address: this.invoiceFormAddress,
            city: this.invoiceFormCity,
            state: this.invoiceFormState,
            zip_code: this.invoiceFormZipCode,
            country: this.invoiceFormCountry,
            notes: this.invoiceFormNotes,
          },
        } as {
          subscriptionInvoiceId: string
          request: RequestsUserPaymentInvoice
        })
        .then(() => {
          this.subscriptionInvoiceDialog = false
        })
        .catch((error: ErrorMessages) => {
          this.errorMessages = error
        })
        .finally(() => {
          this.loading = false
        })
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

    showInvoiceDialog(payment: subscriptionPayment) {
      this.selectedPayment = payment
      this.subscriptionInvoiceDialog = true
    },
  },
})
</script>
