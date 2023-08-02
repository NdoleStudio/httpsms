<template>
  <v-container fluid class="pa-0" :fill-height="$vuetify.breakpoint.lgAndUp">
    <div class="w-full h-full">
      <v-app-bar height="60" :dense="$vuetify.breakpoint.mdAndDown" fixed>
        <v-btn icon to="/threads">
          <v-icon>{{ mdiArrowLeft }}</v-icon>
        </v-btn>
        <v-toolbar-title
          >New Message
          <v-icon x-small class="mx-2" color="primary">{{ mdiCircle }}</v-icon>
          {{ $store.getters.getOwner | phoneNumber }}</v-toolbar-title
        >
      </v-app-bar>
      <v-container class="mt-16">
        <v-row>
          <v-col cols="12" md="8" offset-md="2" xl="6" offset-xl="3">
            <v-form @submit.prevent="sendMessage">
              <v-text-field
                v-model="formPhoneNumber"
                :disabled="sending"
                :error="errors.has('to')"
                :error-messages="errors.get('to')"
                outlined
                placeholder="Recipient phone number e.g +18005550199"
                label="Phone Number"
              ></v-text-field>
              <v-textarea
                v-model="formContent"
                :error="errors.has('content')"
                :error-messages="errors.get('content')"
                :disabled="sending"
                outlined
                placeholder="Enter your message here"
                label="Content"
              ></v-textarea>
              <v-btn
                type="submit"
                class="primary"
                :disabled="sending"
                :block="$vuetify.breakpoint.mdAndDown"
              >
                <v-icon>{{ mdiSend }}</v-icon>
                Send Message
              </v-btn>
            </v-form>
          </v-col>
        </v-row>
      </v-container>
    </div>
  </v-container>
</template>

<script>
import { mdiArrowLeft, mdiSend, mdiSim, mdiCircle } from '@mdi/js'
import axios from '@/plugins/axios'

export default {
  name: 'MessagesIndex',
  middleware: ['auth'],
  data() {
    return {
      mdiArrowLeft,
      mdiSend,
      mdiCircle,
      mdiSim,
      simOptions: [
        { title: 'Default', code: 'DEFAULT' },
        { title: 'SIM 1', code: 'SIM1' },
        { title: 'SIM 2', code: 'SIM2' },
      ],
      simSelected: { title: 'Default', code: 'DEFAULT' },
      sending: false,
      formPhoneNumber: '',
      formContent: '',
      errors: new Map(),
    }
  },
  head() {
    return {
      title: 'New Message - Http SMS',
    }
  },

  methods: {
    sendMessage() {
      this.errors = new Map()
      this.sending = true
      axios
        .post('/v1/messages/send', {
          to: this.formPhoneNumber,
          from: this.$store.getters.getOwner,
          content: this.formContent,
          sim: this.simSelected.code,
        })
        .then(() => {
          this.$store.dispatch('addNotification', {
            message: 'Message Sent!',
            type: 'success',
          })
          this.$router.push({ name: 'threads' })
        })
        .catch((axiosError) => {
          const errors = new Map()
          const response = axiosError.response
          if (response.data.data.content) {
            errors.set('content', response.data.data.content)
          }
          if (response.data.data.to) {
            errors.set(
              'to',
              response.data.data.to.map((x) =>
                x.replace('to field', 'phone number field'),
              ),
            )
          }
          if (response.data.data.from) {
            this.$store.dispatch('addNotification', {
              message: response.data.data.from[0],
              type: 'error',
            })
          }
          this.errors = errors
        })
        .finally(() => {
          this.sending = false
        })
    },
  },
}
</script>
