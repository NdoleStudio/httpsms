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
          <div class="py-16">Phone API Keys</div>
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
            <div class="d-flex mt-3 mb-4">
              <v-progress-circular
                :size="24"
                :width="2"
                v-if="loading"
                color="primary"
                class="mt-2 mr-2"
                indeterminate
              ></v-progress-circular>
              <h5 class="text-h4">Phone API Keys</h5>
              <v-btn
                color="primary"
                class="ml-4 mt-1"
                @click="showCreateAPIKeyDialog = true"
              >
                <v-icon left>{{ mdiPlus }}</v-icon>
                Create API Key
              </v-btn>
              <v-dialog
                v-model="showCreateAPIKeyDialog"
                overlay-opacity="0.9"
                max-width="600px"
              >
                <v-card>
                  <v-card-title>Create Phone API Key</v-card-title>
                  <v-card-subtitle class="mt-2"
                    >After creating the API key you can use it to login to the
                    httpSMS Android app on your phone</v-card-subtitle
                  >
                  <v-card-text>
                    <v-form>
                      <v-text-field
                        v-model="formPhoneApiKeyName"
                        label="Name"
                        placeholder="Enter a name for your API key"
                        name="api-key"
                        outlined
                        :disabled="loading"
                        :error="errorMessages.has('name')"
                        :error-messages="errorMessages.get('name')"
                        persistent-hint
                        class="mb-n2"
                      ></v-text-field>
                    </v-form>
                  </v-card-text>
                  <v-card-actions class="mt-n4">
                    <v-btn
                      color="primary"
                      :loading="loading"
                      @click="createPhoneApiKey"
                      >Create Key</v-btn
                    >
                    <v-spacer />
                    <v-btn
                      color="default"
                      text
                      @click="showCreateAPIKeyDialog = false"
                      >Close</v-btn
                    >
                  </v-card-actions>
                </v-card>
              </v-dialog>
            </div>
            <p class="text--secondary">
              If you have multiple phones, you can create a unique phone API
              keys for your different Android phones. These API keys can only be
              used on the specific mobile phone when it calls the httpSMS server
              for specific actions like sending heartbeats, registering received
              messages, delivery reports etc. If you want to interact with the
              full
              <a
                class="text-decoration-none"
                target="_blank"
                href="https://api.httpsms.com"
                >httpSMS API</a
              >, use the API key under your account settings page instead
              <router-link class="text-decoration-none" to="/settings"
                >https://httpsms.com/settings</router-link
              >.
            </p>
            <v-simple-table class="mb-4 api-key-table">
              <template #default>
                <thead>
                  <tr class="text-uppercase subtitle-2">
                    <th class="text-left">Name</th>
                    <th class="text-left">Created At</th>
                    <th class="text-left">Phone Numbers</th>
                    <th class="text-left">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="phoneApiKey in phoneApiKeys" :key="phoneApiKey.id">
                    <td class="text-left">
                      {{ phoneApiKey.name }}
                    </td>
                    <td>{{ phoneApiKey.created_at | timestamp }}</td>
                    <td>
                      <ul v-if="phoneApiKey.phone_numbers" class="ml-n3">
                        <li
                          v-for="phoneNumber in phoneApiKey.phone_numbers"
                          :key="phoneNumber"
                          class="my-3"
                        >
                          <b>{{ phoneNumber | phoneNumber }}</b>
                          <v-btn
                            class="ml-2 mt-n1"
                            small
                            color="error"
                            @click="
                              showRemovePhoneFromApiKeyDialog(
                                phoneApiKey,
                                phoneNumber,
                              )
                            "
                          >
                            Remove
                          </v-btn>
                        </li>
                      </ul>
                      <span v-else>-</span>
                    </td>
                    <td>
                      <v-btn
                        small
                        color="primary"
                        :disabled="loading"
                        :loading="loading"
                        @click="showPhoneApiKey(phoneApiKey)"
                      >
                        <v-icon left>{{ mdiEye }}</v-icon> View
                      </v-btn>
                      <v-btn
                        class="ml-2"
                        small
                        :disabled="loading"
                        color="error"
                        @click="showDeletePhoneApiKeyDialog(phoneApiKey)"
                      >
                        <v-icon left>{{ mdiDelete }}</v-icon> Delete
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
    <v-dialog
      v-model="showPhoneApiKeyQrCode"
      overlay-opacity="0.9"
      max-width="600"
    >
      <v-card>
        <v-card-title>Phone API Key QR Code</v-card-title>
        <v-card-subtitle class="mt-2"
          >Scan this QR code with the
          <a
            class="text-decoration-none"
            :href="$store.getters.getAppData.appDownloadUrl"
            >httpSMS app</a
          >
          on your Android phone to login.</v-card-subtitle
        >
        <v-card-text class="text-center">
          <v-text-field
            :value="activePhoneApiKey?.api_key"
            readonly
            name="api-key"
            outlined
            class="mb-n2"
          ></v-text-field>
          <canvas ref="qrCodeCanvas"></canvas>
        </v-card-text>
        <v-card-actions>
          <copy-button
            :value="activePhoneApiKey?.api_key"
            color="primary"
            copy-text="Copy API key"
            notification-text="Phone API Key copied successfully"
          />
          <v-spacer></v-spacer>
          <v-btn text class="mb-4" @click="showPhoneApiKeyQrCode = false"
            >Close</v-btn
          >
        </v-card-actions>
      </v-card>
    </v-dialog>
    <v-dialog
      v-model="deleteApiKeyDialog"
      overlay-opacity="0.9"
      max-width="600"
    >
      <v-card>
        <v-card-title class="text-h5 text-break">
          Are you sure you want to delete the
          <code>{{ activePhoneApiKey?.name }}</code> API Key?
        </v-card-title>
        <v-card-text>
          You will have to logout and login again on the <b>httpSMS</b> Android
          app on all of the phones which are currently using this API key.
        </v-card-text>
        <v-card-actions class="pb-4">
          <v-btn color="error" :loading="loading" @click="deleteApiKey">
            <v-icon left>{{ mdiDelete }}</v-icon>
            Delete API Key
          </v-btn>
          <v-spacer></v-spacer>
          <v-btn text @click="deleteApiKeyDialog = false"> Close </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
    <v-dialog
      v-model="removePhoneFromApiKeyDialog"
      overlay-opacity="0.9"
      max-width="600"
    >
      <v-card>
        <v-card-title class="text-h5 text-break">
          Are you sure you want to remove this phone number from the Phone API
          Key?
        </v-card-title>
        <v-card-text>
          This will remove the
          <code>{{ activePhoneNumber | phoneNumber }}</code> from your phone API
          key. You will have to logout and login again on the
          <b>httpSMS</b> Android app on the phone which is currently using this
          API key.
        </v-card-text>
        <v-card-actions class="pb-4">
          <v-btn
            color="error"
            :loading="loading"
            @click="removePhoneFromPhoneKey"
          >
            <v-icon left>{{ mdiDelete }}</v-icon>
            Remove Phone from key
          </v-btn>
          <v-spacer></v-spacer>
          <v-btn text @click="removePhoneFromApiKeyDialog = false">
            Close
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<script lang="ts">
import Vue from 'vue'
import * as QRCode from 'qrcode'
import {
  mdiArrowLeft,
  mdiAccountCircle,
  mdiShieldCheck,
  mdiDelete,
  mdiInformation,
  mdiPlus,
  mdiContentSave,
  mdiMicrosoftExcel,
  mdiEye,
  mdiEyeOff,
  mdiSendCheck,
  mdiCallReceived,
  mdiCallMade,
  mdiDotsVertical,
  mdiCreditCard,
  mdiSquareEditOutline,
} from '@mdi/js'
import Pusher, { Channel } from 'pusher-js'
import { AxiosError } from 'axios'
import { ErrorMessages, getErrorMessages } from '~/plugins/errors'
import {
  EntitiesPhone,
  EntitiesPhoneAPIKey,
  ResponsesUnprocessableEntity,
} from '~/models/api'

export default Vue.extend({
  name: 'PhoneApiKeysIndex',
  middleware: ['auth'],
  data() {
    return {
      mdiEye,
      mdiPlus,
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
      mdiDotsVertical,
      mdiSquareEditOutline,
      formFile: null,
      loading: true,
      errorTitle: '',
      formPhoneApiKeyName: '',
      showPhoneApiKeyQrCode: false,
      errorMessages: new ErrorMessages(),
      phoneApiKeys: new Array<EntitiesPhoneAPIKey>(),
      activePhoneApiKey: null as EntitiesPhoneAPIKey | null,
      activePhoneNumber: '',
      showCreateAPIKeyDialog: false,
      deleteApiKeyDialog: false,
      removePhoneFromApiKeyDialog: false,
      webhookChannel: null as Channel | null,
    }
  },
  head() {
    return {
      title: 'Phone API keys - httpSMS',
    }
  },
  computed: {},
  async mounted() {
    await this.$store.dispatch('loadUser')
    await this.$store.dispatch('loadPhones')
    this.loadPhoneApiKeys()
    this.loading = false

    const pusher = new Pusher(this.$config.pusherKey, {
      cluster: this.$config.pusherCluster,
    })
    this.webhookChannel = pusher.subscribe(this.$store.getters.getAuthUser.id)
    this.webhookChannel.bind('phone.updated', () => {
      this.loadPhoneApiKeys()
    })
  },
  beforeDestroy() {
    if (this.webhookChannel) {
      this.webhookChannel.unsubscribe()
    }
  },
  methods: {
    showPhoneApiKey(apiKey: EntitiesPhoneAPIKey) {
      this.activePhoneApiKey = apiKey
      this.showPhoneApiKeyQrCode = true
      this.$nextTick(() => {
        this.generateQrCode(apiKey.api_key)
      })
    },
    showDeletePhoneApiKeyDialog(apiKey: EntitiesPhoneAPIKey) {
      this.activePhoneApiKey = apiKey
      this.deleteApiKeyDialog = true
    },
    showRemovePhoneFromApiKeyDialog(
      apiKey: EntitiesPhoneAPIKey,
      phoneNumber: string,
    ) {
      this.activePhoneNumber = phoneNumber
      this.activePhoneApiKey = apiKey
      this.removePhoneFromApiKeyDialog = true
    },
    generateQrCode(text: string) {
      const canvas = this.$refs.qrCodeCanvas
      console.log(canvas)
      if (canvas) {
        QRCode.toCanvas(
          canvas,
          text,
          { errorCorrectionLevel: 'H' },
          (err: any) => {
            if (err) {
              this.$store.dispatch('addNotification', {
                message: 'Failed to generate phone API key QR code',
                type: 'error',
              })
            }
          },
        )
      }
    },
    loadPhoneApiKeys() {
      this.loading = true
      this.$store
        .dispatch('indexPhoneApiKeys')
        .then((phoneApiKeys) => {
          this.phoneApiKeys = phoneApiKeys
        })
        .finally(() => {
          this.loading = false
        })
    },
    removePhoneFromPhoneKey() {
      this.loading = true
      this.$store
        .dispatch('deletePhoneFromPhoneApiKey', {
          phoneApiKeyId: this.activePhoneApiKey?.id,
          phoneId: this.$store.getters.getPhones.find(
            (phone: EntitiesPhone) =>
              phone.phone_number === this.activePhoneNumber,
          )?.id,
        })
        .then(() => {
          this.deleteApiKeyDialog = false
          this.loadPhoneApiKeys()
        })
        .finally(() => {
          this.loading = false
        })
    },
    deleteApiKey() {
      this.loading = true
      this.$store
        .dispatch('deletePhoneApiKey', this.activePhoneApiKey?.id)
        .then(() => {
          this.deleteApiKeyDialog = false
          this.loadPhoneApiKeys()
        })
        .finally(() => {
          this.loading = false
        })
    },
    createPhoneApiKey() {
      this.errorMessages = new ErrorMessages()
      this.loading = true
      this.$store
        .dispatch('storePhoneApiKey', this.formPhoneApiKeyName)
        .then(() => {
          this.formPhoneApiKeyName = ''
          this.showCreateAPIKeyDialog = false
          this.loadPhoneApiKeys()
        })
        .catch((error: AxiosError<ResponsesUnprocessableEntity>) => {
          this.errorMessages = getErrorMessages(error)
          this.loading = false
        })
        .finally(() => {
          this.loading = false
        })
    },
  },
})
</script>
<style scoped lang="scss">
.api-key-table {
  tbody {
    tr:hover {
      background-color: transparent !important;
    }
  }
}
</style>
