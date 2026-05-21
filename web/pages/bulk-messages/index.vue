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
          <div class="py-16">Bulk Messages</div>
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
          <v-col cols="12">
            <h5 class="text-h4 mb-3 mt-3">Bulk Messages</h5>
            <p>
              Fill in our bulk SMS
              <a
                class="text-decoration-none"
                download
                href="/templates/httpsms-bulk.csv"
                >CSV template</a
              >
              or our
              <a
                class="text-decoration-none"
                download
                href="/templates/httpsms-bulk.xlsx"
                >Excel template</a
              >
              and upload it here to send your SMS messages to multiple
              recipients at once. You can also configure
              <nuxt-link
                class="text-decoration-none"
                to="/settings/#send-schedules"
                >send schedules</nuxt-link
              >
              on your phone to make sure messages are sent out at specific times
              of the day e.g
              <span class="text--secondary">Mon - Fri 9am - 5pm.</span>
            </p>
            <v-alert v-if="errorTitle" text prominent type="warning">
              <h6 class="subtitle-1 font-weight-bold">{{ errorTitle }}</h6>
              <ul class="body-2">
                <li
                  v-for="message in errorMessages.get('document')"
                  :key="message"
                >
                  {{ message }}
                </li>
              </ul>
            </v-alert>
            <v-form @submit.prevent="sendBulkMessages">
              <v-file-input
                v-model="formFile"
                label="File"
                :prepend-icon="null"
                accept=".csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
                :error-messages="errorMessages.get('document')"
                persistent-placeholder
                placeholder="Click here to upload your bulk SMS file."
                :append-icon="mdiMicrosoftExcel"
                outlined
              ></v-file-input>
              <div class="d-flex">
                <v-btn
                  color="primary"
                  type="submit"
                  :loading="loading"
                  :disabled="loading"
                  large
                >
                  <v-icon left>{{ mdiSendCheck }}</v-icon>
                  Send Bulk Messages
                </v-btn>
                <v-spacer></v-spacer>
                <v-btn
                  v-if="$vuetify.breakpoint.mdAndUp"
                  plain
                  color="info"
                  href="mailto:arnold@httpsms.com?subject=I'm having trouble with the bulk messages"
                >
                  I Need Help
                </v-btn>
              </div>
            </v-form>
          </v-col>
        </v-row>
        <v-row class="mt-8">
          <v-col cols="12">
            <h4 class="text-h4 mb-3">Bulk Message History</h4>
            <p class="text--secondary">
              Your 10 most recent bulk SMS uploads are shown below, including a
              delivery status breakdown for each batch. Click
              <code>View</code> to see individual messages.
            </p>
            <v-progress-linear
              v-if="loadingHistory"
              indeterminate
              class="mb-4"
            ></v-progress-linear>
            <v-simple-table>
              <template #default>
                <thead>
                  <tr class="text-uppercase subtitle-2">
                    <th class="text-left">Name</th>
                    <th class="text-center">Total</th>
                    <th class="text-center">Pending</th>
                    <th class="text-center">Scheduled</th>
                    <th class="text-center">Sent</th>
                    <th class="text-center">Delivered</th>
                    <th class="text-center">Failed</th>
                    <th class="text-center">Expired</th>
                    <th class="text-center">Created At</th>
                    <th class="text-center">Action</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="order in bulkOrders" :key="order.request_id">
                    <td class="text-left font-weight-medium">
                      {{ order.request_id }}
                    </td>
                    <td class="text-center">{{ order.total }}</td>
                    <td class="text-center">{{ order.pending_count }}</td>
                    <td class="text-center">{{ order.scheduled_count }}</td>
                    <td class="text-center">{{ order.sent_count }}</td>
                    <td class="text-center">{{ order.delivered_count }}</td>
                    <td class="text-center">{{ order.failed_count }}</td>
                    <td class="text-center">{{ order.expired_count }}</td>
                    <td class="text-center">
                      {{ order.created_at | timestamp }}
                    </td>
                    <td class="text-center">
                      <v-btn
                        small
                        color="primary"
                        text
                        :to="`/search-messages?query=${order.request_id}`"
                      >
                        <v-icon small left>{{ mdiEye }}</v-icon>
                        View
                      </v-btn>
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
  mdiInformation,
  mdiContentSave,
  mdiMicrosoftExcel,
  mdiEye,
  mdiEyeOff,
  mdiSendCheck,
  mdiCallReceived,
  mdiCallMade,
  mdiCreditCard,
  mdiSquareEditOutline,
} from '@mdi/js'
import { AxiosError } from 'axios'
import { ErrorMessages, getErrorMessages } from '~/plugins/errors'
import capitalize from '~/plugins/capitalize'
import { ResponsesUnprocessableEntity } from '~/models/api'

export default Vue.extend({
  name: 'BulkMessagesIndex',
  middleware: ['auth'],
  data() {
    return {
      mdiEye,
      mdiEyeOff,
      mdiMicrosoftExcel,
      mdiArrowLeft,
      mdiAccountCircle,
      mdiShieldCheck,
      mdiDelete,
      mdiSendCheck,
      mdiContentSave,
      mdiCallReceived,
      mdiCallMade,
      mdiCreditCard,
      mdiInformation,
      mdiSquareEditOutline,
      formFile: null,
      loading: true,
      loadingHistory: true,
      errorTitle: '',
      errorMessages: new ErrorMessages(),
      dialog: false,
      bulkOrders: [] as any[],
    }
  },
  head() {
    return {
      title: 'Send Bulk Messages - httpSMS',
    }
  },
  computed: {},
  async mounted() {
    await this.$store.dispatch('loadUser')
    this.loading = false
    this.fetchBulkOrders()
  },
  methods: {
    fetchBulkOrders() {
      this.loadingHistory = true
      this.$store
        .dispatch('fetchBulkMessageOrders')
        .then((orders: any[]) => {
          this.bulkOrders = orders
        })
        .catch(() => {
          // silently fail - the table will show "no data"
        })
        .finally(() => {
          this.loadingHistory = false
        })
    },
    sendBulkMessages() {
      this.loading = true
      this.errorMessages = new ErrorMessages()
      this.errorTitle = ''

      this.$store
        .dispatch('sendBulkMessages', this.formFile)
        .then(() => {
          setTimeout(() => {
            this.loading = false
            this.$router.push({ name: 'threads' })
          }, 2000)
        })
        .catch((error: AxiosError<ResponsesUnprocessableEntity>) => {
          this.errorTitle = capitalize(
            error.response?.data?.message ??
              'Error while sending bulk messages',
          )
          this.errorMessages = getErrorMessages(error)
          this.loading = false
        })
    },
  },
})
</script>
