<template>
  <v-container
    fluid
    class="px-0 pt-0"
    :fill-height="$vuetify.breakpoint.lgAndUp"
  >
    <div class="w-full h-full">
      <v-app-bar height="60" fixed :dense="$vuetify.breakpoint.mdAndDown">
        <v-btn icon to="/threads">
          <v-icon>{{ mdiArrowLeft }}</v-icon>
        </v-btn>
        <v-toolbar-title>
          <div class="py-16">Settings</div>
        </v-toolbar-title>
      </v-app-bar>
      <v-container class="mt-16">
        <v-row>
          <v-col cols="12" md="9" offset-md="1" xl="8" offset-xl="2">
            <div v-if="$fire.auth.currentUser" class="text-center">
              <v-avatar size="100" color="indigo" class="mx-auto">
                <img
                  v-if="$fire.auth.currentUser.photoURL"
                  :src="$fire.auth.currentUser.photoURL"
                  :alt="$fire.auth.currentUser.displayName"
                />
                <v-icon v-else dark size="70">{{ mdiAccountCircle }}</v-icon>
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
                >
                  {{ mdiShieldCheck }}
                </v-icon>
              </h4>
              <v-autocomplete
                v-if="$store.getters.getUser"
                dense
                outlined
                :value="$store.getters.getUser.timezone"
                class="mx-auto mt-2"
                style="max-width: 250px"
                label="Timezone"
                :items="timezones"
                @change="updateUser"
              ></v-autocomplete>
            </div>
            <h5 class="text-h4 mb-3 mt-3">API Key</h5>
            <p class="text--secondary">
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
              :append-icon="apiKeyShow ? mdiEye : mdiEyeOff"
              :type="apiKeyShow ? 'text' : 'password'"
              :value="apiKey"
              readonly
              name="api-key"
              outlined
              class="mb-n2"
              @click:append="apiKeyShow = !apiKeyShow"
            ></v-text-field>
            <div class="d-flex">
              <copy-button
                :block="$vuetify.breakpoint.mdAndDown"
                :large="$vuetify.breakpoint.mdAndDown"
                :value="apiKey"
                color="primary"
                copy-text="Copy API Key"
                notification-text="API Key copied successfully"
              ></copy-button>
              <v-btn
                v-if="$vuetify.breakpoint.lgAndUp"
                class="ml-4"
                :href="$store.getters.getAppData.documentationUrl"
                >Documentation</v-btn
              >
            </div>
            <h5 class="text-h4 mb-3 mt-12">Webhooks</h5>
            <p class="text--secondary">
              Webhooks allow us to send events to your server for example when
              the android phone receives an SMS message we can forward the
              message to your server.
            </p>
            <div v-if="loadingWebhooks">
              <v-progress-circular
                :size="60"
                :width="2"
                color="primary"
                class="mb-4"
                indeterminate
              ></v-progress-circular>
            </div>
            <v-simple-table v-else-if="webhooks.length" class="mb-4">
              <template #default>
                <thead>
                  <tr class="text-uppercase subtitle-2">
                    <th v-if="$vuetify.breakpoint.xlOnly" class="text-left">
                      ID
                    </th>
                    <th class="text-left">Callback URL</th>
                    <th v-if="$vuetify.breakpoint.lgAndUp" class="text-center">
                      Events
                    </th>
                    <th class="text-center">Action</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="webhook in webhooks" :key="webhook.id">
                    <td v-if="$vuetify.breakpoint.xlOnly" class="text-left">
                      {{ webhook.id }}
                    </td>
                    <td>{{ webhook.url }}</td>
                    <td v-if="$vuetify.breakpoint.lgAndUp" class="text-center">
                      <v-chip
                        v-for="event in webhook.events"
                        :key="event"
                        small
                        >{{ event }}</v-chip
                      >
                    </td>
                    <td class="text-center">
                      <v-btn
                        :icon="$vuetify.breakpoint.mdAndDown"
                        small
                        color="info"
                        :disabled="updatingWebhook"
                        @click.prevent="onWebhookEdit(webhook.id)"
                      >
                        <v-icon small>{{ mdiSquareEditOutline }}</v-icon>
                        <span v-if="!$vuetify.breakpoint.mdAndDown">
                          Edit
                        </span>
                      </v-btn>
                    </td>
                  </tr>
                </tbody>
              </template>
            </v-simple-table>
            <v-btn color="primary" @click="onWebhookCreate">
              <v-icon left>{{ mdiLinkVariant }}</v-icon>
              Add webhook
            </v-btn>
            <h5 class="text-h4 mb-3 mt-12">Discord Integration</h5>
            <p class="text--secondary">
              Send and receive SMS messages without leaving your discord server
              with the httpsms discord app using the
              <code>/httpsms</code> command.
            </p>
            <div v-if="loadingDiscordIntegrations">
              <v-progress-circular
                :size="60"
                :width="2"
                color="primary"
                class="mb-4"
                indeterminate
              ></v-progress-circular>
            </div>
            <v-simple-table v-else-if="discords.length" class="mb-4">
              <template #default>
                <thead>
                  <tr class="text-uppercase subtitle-2">
                    <th class="text-left">Name</th>
                    <th class="text-left">Server ID</th>
                    <th class="text-left">Channel ID</th>
                    <th class="text-center">Action</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="discord in discords" :key="discord.id">
                    <td class="text-left">
                      {{ discord.name }}
                    </td>
                    <td class="text-left">
                      {{ discord.server_id }}
                    </td>
                    <td class="text-left">
                      {{ discord.incoming_channel_id }}
                    </td>
                    <td class="text-center">
                      <v-btn
                        :icon="$vuetify.breakpoint.mdAndDown"
                        small
                        color="info"
                        :disabled="updatingDiscord"
                        @click.prevent="onDiscordEdit(discord.id)"
                      >
                        <v-icon small>{{ mdiSquareEditOutline }}</v-icon>
                        <span v-if="!$vuetify.breakpoint.mdAndDown">
                          Edit
                        </span>
                      </v-btn>
                    </td>
                  </tr>
                </tbody>
              </template>
            </v-simple-table>
            <v-btn color="#5865f2" @click="onDiscordCreate">
              <v-img
                contain
                height="24"
                width="24"
                class="mr-2"
                :src="require('assets/img/discord-logo.svg')"
              ></v-img>
              Add Discord
            </v-btn>
            <h5 class="text-h4 mb-3 mt-12">Phones</h5>
            <p class="text--secondary">
              List of mobile phones which are registered for sending and
              receiving SMS messages.
            </p>
            <v-simple-table>
              <template #default>
                <thead>
                  <tr class="text-uppercase subtitle-2">
                    <th v-if="$vuetify.breakpoint.xlOnly" class="text-left">
                      ID
                    </th>
                    <th class="text-left">Phone Number</th>
                    <th v-if="$vuetify.breakpoint.lgAndUp" class="text-center">
                      Fcm Token
                    </th>
                    <th v-if="$vuetify.breakpoint.lgAndUp" class="text-center">
                      Retries
                    </th>
                    <th class="text-center">Rate</th>
                    <th class="text-center">Updated At</th>
                    <th class="text-center">Action</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="phone in $store.getters.getPhones" :key="phone.id">
                    <td v-if="$vuetify.breakpoint.xlOnly" class="text-left">
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
                    <td v-if="$vuetify.breakpoint.lgAndUp">
                      <div class="d-flex justify-center">
                        {{
                          phone.max_send_attempts ? phone.max_send_attempts : 1
                        }}
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
                        color="info"
                        :disabled="updatingPhone"
                        @click.prevent="showEditPhone(phone.id)"
                      >
                        <v-icon small>{{ mdiSquareEditOutline }}</v-icon>
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
    <v-dialog v-model="showPhoneEdit" max-width="600px">
      <v-card>
        <v-card-title>Edit Phone</v-card-title>
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
                <v-text-field
                  outlined
                  disabled
                  dense
                  label="SIM"
                  :value="activePhone.sim"
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
                  v-model="activePhone.message_expiration_seconds"
                  outlined
                  type="number"
                  dense
                  label="Message Expiration (seconds)"
                >
                </v-text-field>
                <v-text-field
                  v-model="activePhone.messages_per_minute"
                  outlined
                  type="number"
                  dense
                  label="Messages Per Minute"
                >
                </v-text-field>
                <v-text-field
                  v-model="activePhone.max_send_attempts"
                  outlined
                  type="number"
                  dense
                  placeholder="How many retries when sending an SMS"
                  label="Max Send Attempts"
                >
                </v-text-field>
              </v-col>
            </v-row>
          </v-container>
        </v-card-text>
        <v-card-actions class="mt-n8">
          <v-btn small color="info" @click="updatePhone">
            <v-icon v-if="$vuetify.breakpoint.lgAndUp" small>
              {{ mdiContentSave }}
            </v-icon>
            Update
          </v-btn>
          <v-spacer></v-spacer>
          <v-btn small color="error" text @click="deletePhone(activePhone.id)">
            <v-icon v-if="$vuetify.breakpoint.lgAndUp" small>
              {{ mdiDelete }}
            </v-icon>
            Delete
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
    <v-dialog v-model="showWebhookEdit" max-width="600px">
      <v-card>
        <v-card-title>
          <span v-if="!activeWebhook.id">Add a new&nbsp;</span>
          <span v-else>Edit&nbsp;</span>
          webhook
        </v-card-title>
        <v-card-text>
          <v-row>
            <v-col class="pt-8">
              <v-text-field
                v-if="activeWebhook.id"
                outlined
                dense
                disabled
                label="ID"
                :value="activeWebhook.id"
              >
              </v-text-field>
              <v-text-field
                v-model="activeWebhook.url"
                outlined
                dense
                label="Callback URL"
                persistent-placeholder
                persistent-hint
                :error="errorMessages.has('url')"
                :error-messages="errorMessages.get('url')"
                hint="A POST request will be sent to this URL every time an event is triggered in httpSMS."
                placeholder="https://example.com/webhook"
              >
              </v-text-field>
              <v-text-field
                v-model="activeWebhook.signing_key"
                outlined
                dense
                class="mt-6"
                persistent-placeholder
                persistent-hint
                label="Signing Key"
                placeholder="******************"
                :error="errorMessages.has('signing_key')"
                :error-messages="errorMessages.get('signing_key')"
                hint="The signing key is used to verify the webhook is sent from httpSMS."
              >
              </v-text-field>
              <v-select
                v-model="activeWebhook.events"
                :items="events"
                label="Events"
                multiple
                outlined
                persistent-placeholder
                class="mt-6"
                dense
                :error="errorMessages.has('events')"
                :error-messages="errorMessages.get('events')"
                hint="Select multiple httpSMS events to watch for"
                persistent-hint
              ></v-select>
            </v-col>
          </v-row>
        </v-card-text>
        <v-card-actions class="mt-n4 pb-4">
          <loading-button
            v-if="!activeWebhook.id"
            :icon="mdiContentSave"
            :loading="updatingWebhook"
            small
            @click="createWebhook"
          >
            Save Webhook
          </loading-button>
          <loading-button
            v-else
            small
            color="info"
            :loading="updatingWebhook"
            @click="updateWebhook"
          >
            <v-icon v-if="$vuetify.breakpoint.lgAndUp" small>
              {{ mdiContentSave }}
            </v-icon>
            Update Webhook
          </loading-button>
          <v-spacer></v-spacer>
          <v-btn
            v-if="activeWebhook.id"
            :disabled="updatingWebhook"
            small
            color="error"
            text
            @click="deleteWebhook(activeWebhook.id)"
          >
            <v-icon v-if="$vuetify.breakpoint.lgAndUp" small>
              {{ mdiDelete }}
            </v-icon>
            Delete
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
    <v-dialog v-model="showDiscordEdit" max-width="700px">
      <v-card>
        <v-card-title>
          <span v-if="!activeDiscord.id">Add a new&nbsp;</span>
          <span v-else>Edit&nbsp;</span>
          discord integration
        </v-card-title>
        <v-card-text>
          <v-row>
            <v-col class="pt-8">
              <p class="mt-n4 subtitle-1">
                Click the button below to add the httpSMS bot to your discord
                server. You need to do this so we can have permission to send
                and receive messages on your discord server.
              </p>
              <v-btn
                color="#5865f2"
                class="mb-6"
                target="_blank"
                href="https://discord.com/api/oauth2/authorize?client_id=1095780203256627291&permissions=2147485760&scope=bot%20applications.commands"
              >
                <v-icon left>{{ mdiConnection }}</v-icon>
                Add Discord Bot
              </v-btn>
              <v-text-field
                v-if="activeDiscord.id"
                outlined
                dense
                disabled
                label="ID"
                :value="activeDiscord.id"
              >
              </v-text-field>
              <v-text-field
                v-model="activeDiscord.name"
                outlined
                dense
                label="Name"
                persistent-placeholder
                persistent-hint
                :error="errorMessages.has('name')"
                :error-messages="errorMessages.get('name')"
                hint="The name of the discord integration"
                placeholder="e.g Game Server"
              >
              </v-text-field>
              <v-text-field
                v-model="activeDiscord.server_id"
                outlined
                dense
                class="mt-6"
                persistent-placeholder
                persistent-hint
                label="Discord Server ID"
                placeholder="e.g 1095778291488653372"
                :error="errorMessages.has('server_id')"
                :error-messages="errorMessages.get('server_id')"
                hint="You can get this by right clicking on your server and clicking Copy Server ID."
              >
              </v-text-field>
              <v-text-field
                v-model="activeDiscord.incoming_channel_id"
                outlined
                dense
                class="mt-6"
                persistent-placeholder
                persistent-hint
                label="Discord Incoming Channel ID"
                placeholder="e.g 1095778291488653372"
                :error="errorMessages.has('incoming_channel_id')"
                :error-messages="errorMessages.get('incoming_channel_id')"
                hint="You can get this by right clicking on your discord channel and clicking Copy Chanel ID."
              >
              </v-text-field>
            </v-col>
          </v-row>
        </v-card-text>
        <v-card-actions class="mt-n4 pb-4 pl-6">
          <loading-button
            v-if="!activeDiscord.id"
            :icon="mdiContentSave"
            :loading="updatingDiscord"
            @click="createDiscord"
          >
            Save Discord Integration
          </loading-button>
          <loading-button
            v-else
            color="info"
            :loading="updatingDiscord"
            @click="updateDiscord"
          >
            <v-icon v-if="$vuetify.breakpoint.lgAndUp" small>
              {{ mdiContentSave }}
            </v-icon>
            Update Discord Integration
          </loading-button>
          <v-spacer></v-spacer>
          <v-btn
            v-if="activeDiscord.id"
            :disabled="updatingDiscord"
            color="error"
            text
            @click="deleteDiscord(activeDiscord.id)"
          >
            <v-icon v-if="$vuetify.breakpoint.lgAndUp" small>
              {{ mdiDelete }}
            </v-icon>
            Delete
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<script>
import Vue from 'vue'
import {
  mdiArrowLeft,
  mdiAccountCircle,
  mdiShieldCheck,
  mdiDelete,
  mdiContentSave,
  mdiConnection,
  mdiEye,
  mdiLinkVariant,
  mdiEyeOff,
  mdiSquareEditOutline,
} from '@mdi/js'
import { ErrorMessages } from '~/plugins/errors'
import LoadingButton from '~/components/LoadingButton.vue'

export default Vue.extend({
  name: 'SettingsIndex',
  components: { LoadingButton },
  middleware: ['auth'],
  data() {
    return {
      mdiEye,
      mdiEyeOff,
      mdiArrowLeft,
      mdiAccountCircle,
      mdiShieldCheck,
      mdiDelete,
      mdiLinkVariant,
      mdiContentSave,
      mdiSquareEditOutline,
      mdiConnection,
      errorMessages: new ErrorMessages(),
      apiKeyShow: false,
      showPhoneEdit: false,
      showDiscordEdit: false,
      activeWebhook: {
        id: null,
        url: '',
        signing_key: '',
        events: ['message.phone.received'],
      },
      activeDiscord: {
        id: null,
        name: '',
        server_id: '',
        incoming_channel_id: '',
      },
      updatingWebhook: false,
      loadingWebhooks: false,
      discords: [],
      webhooks: [],
      showWebhookEdit: false,
      activePhone: null,
      updatingPhone: false,
      updatingDiscord: false,
      loadingDiscordIntegrations: false,
      events: ['message.phone.received'],
    }
  },
  head() {
    return {
      title: 'Settings - Http SMS',
    }
  },
  computed: {
    apiKey() {
      if (this.$store.getters.getUser === null) {
        return ''
      }
      return this.$store.getters.getUser.api_key
    },
    timezones() {
      return Intl.supportedValuesOf('timeZone')
    },
  },
  mounted() {
    this.$store.dispatch('clearAxiosError')
    this.$store.dispatch('loadUser')
    this.$store.dispatch('loadPhones')
    this.loadWebhooks()
    this.loadDiscordIntegrations()
  },

  methods: {
    showEditPhone(phoneId) {
      const phone = this.$store.getters.getPhones.find((x) => x.id === phoneId)
      if (!phone) {
        return
      }
      this.activePhone = { ...phone }
      this.showPhoneEdit = true
      this.resetErrors()
    },

    onWebhookEdit(webhookId) {
      const webhook = this.webhooks.find((x) => x.id === webhookId)
      if (!webhook) {
        return
      }
      this.activeWebhook = {
        id: webhook.id,
        url: webhook.url,
        signing_key: webhook.signing_key,
        events: webhook.events,
      }
      this.showWebhookEdit = true
      this.resetErrors()
    },

    onDiscordEdit(discordId) {
      const discord = this.discords.find((x) => x.id === discordId)
      if (!discord) {
        return
      }
      this.activeDiscord = {
        id: discord.id,
        name: discord.name,
        server_id: discord.server_id,
        incoming_channel_id: discord.incoming_channel_id,
      }
      this.showDiscordEdit = true
      this.resetErrors()
    },

    onWebhookCreate() {
      this.activeWebhook = {
        id: null,
        url: '',
        signing_key: '',
        events: ['message.phone.received'],
      }
      this.showWebhookEdit = true
      this.resetErrors()
    },

    onDiscordCreate() {
      this.activeDiscord = {
        id: null,
        name: '',
        server_id: '',
        incoming_channel_id: '',
      }
      this.showDiscordEdit = true
      this.resetErrors()
    },

    async updatePhone() {
      this.updatingPhone = true
      await this.$store.dispatch('clearAxiosError')
      this.$store.dispatch('updatePhone', this.activePhone).finally(() => {
        if (!this.$store.getters.getAxiosError) {
          this.updatingPhone = false
          this.showPhoneEdit = false
          this.activePhone = null
        }
      })
    },

    resetErrors() {
      this.errorMessages = new ErrorMessages()
    },

    createDiscord() {
      this.resetErrors()
      this.updatingDiscord = true
      this.$store
        .dispatch('createDiscord', this.activeDiscord)
        .then(() => {
          this.$store.dispatch('addNotification', {
            message: 'Discord integration created successfully',
            type: 'success',
          })
          this.showDiscordEdit = false
          this.loadDiscordIntegrations()
        })
        .catch((errors) => {
          this.errorMessages = errors
        })
        .finally(() => {
          this.updatingDiscord = false
        })
    },

    updateDiscord() {
      this.resetErrors()
      this.updatingDiscord = true
      this.$store
        .dispatch('updateDiscordIntegration', this.activeDiscord)
        .then(() => {
          this.$store.dispatch('addNotification', {
            message: 'Discord integration updated successfully',
            type: 'success',
          })
          this.showDiscordEdit = false
          this.loadDiscordIntegrations()
        })
        .catch((errors) => {
          this.errorMessages = errors
        })
        .finally(() => {
          this.updatingDiscord = false
        })
    },

    deleteDiscord(discordId) {
      this.updatingDiscord = true
      this.$store
        .dispatch('deleteDiscordIntegration', discordId)
        .then(() => {
          this.$store.dispatch('addNotification', {
            message: 'Discord integration deleted successfully',
            type: 'success',
          })
          this.showDiscordEdit = false
          this.loadDiscordIntegrations()
        })
        .finally(() => {
          this.updatingDiscord = false
        })
    },

    createWebhook() {
      this.resetErrors()
      this.updatingWebhook = true
      this.$store
        .dispatch('createWebhook', this.activeWebhook)
        .then(() => {
          this.$store.dispatch('addNotification', {
            message: 'Webhook created successfully',
            type: 'success',
          })
          this.showWebhookEdit = false
          this.loadWebhooks()
        })
        .catch((errors) => {
          this.errorMessages = errors
        })
        .finally(() => {
          this.updatingWebhook = false
        })
    },

    updateUser(timezone) {
      this.resetErrors()
      this.$store
        .dispatch('updateUser', {
          owner: this.$store.getters.getOwner,
          timezone,
        })
        .then(() => {
          this.$store.dispatch('addNotification', {
            message: 'Timezone updated successfully',
            type: 'success',
          })
        })
        .catch(() => {
          this.$store.dispatch('addNotification', {
            message: 'Failed to update timezone',
            type: 'error',
          })
        })
    },

    updateWebhook() {
      this.resetErrors()
      this.updatingWebhook = true
      this.$store
        .dispatch('updateWebhook', this.activeWebhook)
        .then(() => {
          this.$store.dispatch('addNotification', {
            message: 'Webhook updated successfully',
            type: 'success',
          })
          this.showWebhookEdit = false
          this.loadWebhooks()
        })
        .catch((errors) => {
          this.errorMessages = errors
        })
        .finally(() => {
          this.updatingWebhook = false
        })
    },

    deleteWebhook(webhookId) {
      this.updatingWebhook = true
      this.$store
        .dispatch('deleteWebhook', webhookId)
        .then(() => {
          this.$store.dispatch('addNotification', {
            message: 'Webhook deleted successfully',
            type: 'success',
          })
          this.showWebhookEdit = false
          this.loadWebhooks()
        })
        .finally(() => {
          this.updatingWebhook = false
        })
    },

    loadWebhooks() {
      this.loadingWebhooks = true
      this.$store
        .dispatch('getWebhooks')
        .then((webhooks) => {
          this.webhooks = webhooks
        })
        .finally(() => {
          this.loadingWebhooks = false
        })
    },

    loadDiscordIntegrations() {
      this.loadingDiscordIntegrations = true
      this.$store
        .dispatch('getDiscordIntegrations')
        .then((discords) => {
          this.discords = discords
        })
        .finally(() => {
          this.loadingDiscordIntegrations = false
        })
    },

    deletePhone(phoneId) {
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
