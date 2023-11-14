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
              recipients at once.
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
      errorTitle: '',
      errorMessages: new ErrorMessages(),
      dialog: false,
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
  },
  methods: {
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
