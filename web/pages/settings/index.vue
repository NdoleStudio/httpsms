<template>
  <v-container fluid class="pa-0" :fill-height="$vuetify.breakpoint.lgAndUp">
    <div class="w-full h-full">
      <v-app-bar :dense="$vuetify.breakpoint.mdAndDown">
        <v-btn icon to="/">
          <v-icon>mdi-arrow-left</v-icon>
        </v-btn>
        <v-toolbar-title> Settings </v-toolbar-title>
      </v-app-bar>
      <v-container>
        <v-row>
          <v-col cols="12" md="9" offset-md="1" xl="8" offset-xl="2">
            <div v-if="$fire.auth.currentUser" class="text-center">
              <v-avatar size="100" color="indigo" class="mx-auto">
                <img
                  v-if="$fire.auth.currentUser.photoURL"
                  :src="$fire.auth.currentUser.photoURL"
                  :alt="$fire.auth.currentUser.displayName"
                />
                <v-icon v-else dark size="70"> mdi-account-circle </v-icon>
              </v-avatar>
              <h3 v-if="$fire.auth.currentUser.displayName">
                {{ $fire.auth.currentUser.displayName }}
              </h3>
              <h4 class="text--secondary">
                {{ $fire.auth.currentUser.email }}
                <v-icon
                  v-if="$fire.auth.currentUser.emailVerified"
                  small
                  color="primary"
                  >mdi-shield-check</v-icon
                >
              </h4>
            </div>
            <h5 class="text-h4 mb-3 mt-3">API Key</h5>
            <p>
              Use your API Key in the <code>x-api-key</code> HTTP Header when
              sending requests to
              <code>https://api.httpsms.com</code> endpoints.
            </p>
            <div v-if="apiKey === ''" class="mb-n9 pl-3 pt-5">
              <v-progress-circular
                :size="20"
                :width="2"
                color="primary"
                indeterminate
              ></v-progress-circular>
            </div>
            <v-text-field
              v-else
              :append-icon="apiKeyShow ? 'mdi-eye' : 'mdi-eye-off'"
              :type="apiKeyShow ? 'text' : 'password'"
              :value="apiKey"
              readonly
              name="api-key"
              outlined
              class="mb-n2"
              @click:append="apiKeyShow = !apiKeyShow"
            ></v-text-field>
            <copy-button
              :block="$vuetify.breakpoint.mdAndDown"
              :large="$vuetify.breakpoint.mdAndDown"
              :value="apiKey"
              copy-text="Copy API Key"
              notification-text="API Key copied successfully"
            ></copy-button>
            <h5 class="text-h4 mb-3 mt-12">Phones</h5>
            <p>
              List of mobile phones which are registered for sending and
              receiving SMS messages.
            </p>
            <v-simple-table>
              <template #default>
                <thead>
                  <tr class="text-uppercase subtitle-2">
                    <th v-if="$vuetify.breakpoint.lgAndUp" class="text-left">
                      ID
                    </th>
                    <th class="text-left">Phone Number</th>
                    <th v-if="$vuetify.breakpoint.lgAndUp" class="text-center">
                      Fcm Token
                    </th>
                    <th class="text-center">Rate</th>
                    <th class="text-center">Updated At</th>
                    <th class="text-center">Action</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="phone in $store.getters.getPhones" :key="phone.id">
                    <td v-if="$vuetify.breakpoint.lgAndUp" class="text-left">
                      {{ phone.id }}
                    </td>
                    <td>{{ phone.phone_number | phoneNumber }}</td>
                    <td v-if="$vuetify.breakpoint.lgAndUp">
                      <div class="d-flex justify-center">
                        <v-checkbox
                          readonly
                          class="mx-auto"
                          :input-value="true"
                          color="success"
                        ></v-checkbox>
                      </div>
                    </td>
                    <td class="text-center">
                      <span v-if="phone.messages_per_minute"
                        >{{ phone.messages_per_minute }}/min</span
                      >
                      <span v-else>Unlimited</span>
                    </td>
                    <td class="text-center">
                      {{ phone.updated_at | timestamp }}
                    </td>
                    <td class="text-center">
                      <v-btn
                        :icon="$vuetify.breakpoint.mdAndDown"
                        small
                        color="info"
                        :disabled="updatingPhone"
                        @click.prevent="showEditPhone(phone.id)"
                      >
                        <v-icon small>mdi-square-edit-outline</v-icon>
                        <span v-if="!$vuetify.breakpoint.mdAndDown">
                          Edit
                        </span>
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
    <v-dialog v-model="showPhoneEdit" max-width="500px">
      <v-card>
        <v-card-text v-if="activePhone" class="mt-6">
          <v-container>
            <v-row>
              <v-col>
                <v-text-field
                  outlined
                  dense
                  disabled
                  label="ID"
                  :value="activePhone.id"
                >
                </v-text-field>
                <v-text-field
                  outlined
                  disabled
                  dense
                  label="Phone Number"
                  :value="activePhone.phone_number"
                >
                </v-text-field>
                <v-textarea
                  outlined
                  disabled
                  dense
                  label="FCM Token"
                  :value="activePhone.fcm_token"
                >
                </v-textarea>
                <v-text-field
                  v-model="activePhone.messages_per_minute"
                  outlined
                  type="number"
                  dense
                  label="Messages Per Minute"
                >
                </v-text-field>
              </v-col>
            </v-row>
          </v-container>
        </v-card-text>
        <v-card-actions class="mt-n8">
          <v-btn small color="info" @click="updatePhone">
            <v-icon v-if="$vuetify.breakpoint.lgAndUp" small
              >mdi-content-save</v-icon
            >
            Update
          </v-btn>
          <v-spacer></v-spacer>
          <v-btn small color="error" text @click="deletePhone(activePhone.id)">
            <v-icon v-if="$vuetify.breakpoint.lgAndUp" small>mdi-delete</v-icon>
            Delete
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<script lang="ts">
import Vue from 'vue'
import { Phone } from '~/models/phone'

export default Vue.extend({
  name: 'SettingsIndex',
  middleware: ['auth'],
  data() {
    return {
      apiKeyShow: false,
      showPhoneEdit: false,
      activePhone: null,
      updatingPhone: false,
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
  mounted() {
    if (!this.$store.getters.getAuthUser) {
      this.$store.dispatch('setNextRoute', this.$route.path)
      this.$router.push({ name: 'index' })
      return
    }
    this.$store.dispatch('loadUser')
  },

  methods: {
    showEditPhone(phoneId: string) {
      const phone = this.$store.getters.getPhones.find(
        (x: Phone) => x.id === phoneId
      )
      if (!phone) {
        return
      }
      this.activePhone = { ...phone }
      this.showPhoneEdit = true
    },

    updatePhone() {
      this.updatingPhone = true
      this.$store.dispatch('updatePhone', this.activePhone).finally(() => {
        this.updatingPhone = false
        this.showPhoneEdit = false
        this.activePhone = null
      })
    },

    deletePhone(phoneId: string) {
      this.updatingPhone = true
      this.$store.dispatch('deletePhone', phoneId).finally(() => {
        this.updatingPhone = false
        this.showPhoneEdit = false
        this.activePhone = null
      })
    },
  },
})
</script>
