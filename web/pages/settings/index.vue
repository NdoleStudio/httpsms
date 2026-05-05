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
                @change="updateTimezone"
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
            <div class="d-flex flex-wrap">
              <copy-button
                :value="apiKey"
                color="primary"
                copy-text="Copy API Key"
                notification-text="API Key copied successfully"
              ></copy-button>
              <v-btn
                v-if="$vuetify.breakpoint.mdAndUp"
                color="primary"
                class="ml-4"
                @click="showQrCodeDialog = true"
              >
                <v-icon left>{{ mdiQrcode }}</v-icon>
                Show QR Code
              </v-btn>
              <v-dialog
                v-model="showQrCodeDialog"
                overlay-opacity="0.9"
                max-width="400px"
              >
                <v-card>
                  <v-card-title class="justify-center"
                    >API Key QR Code</v-card-title
                  >
                  <v-card-subtitle class="mt-2 text-center"
                    >Scan this QR code with the
                    <a :href="$store.getters.getAppData.appDownloadUrl"
                      >httpSMS app</a
                    >
                    on your Android phone to login.</v-card-subtitle
                  >
                  <v-card-text class="text-center">
                    <canvas ref="qrCodeCanvas"></canvas>
                  </v-card-text>
                  <v-card-actions>
                    <v-btn
                      color="primary"
                      block
                      class="mb-4"
                      @click="showQrCodeDialog = false"
                      >Close</v-btn
                    >
                  </v-card-actions>
                </v-card>
              </v-dialog>
              <v-btn
                v-if="$vuetify.breakpoint.lgAndUp"
                class="ml-4"
                :href="$store.getters.getAppData.documentationUrl"
                >Documentation</v-btn
              >
              <v-spacer></v-spacer>
              <v-dialog
                v-model="showRotateApiKey"
                overlay-opacity="0.9"
                max-width="550"
              >
                <template #activator="{ on, attrs }">
                  <v-btn
                    :small="$vuetify.breakpoint.mdAndDown"
                    :text="$vuetify.breakpoint.lgAndUp"
                    color="warning"
                    v-bind="attrs"
                    v-on="on"
                  >
                    <v-icon left>{{ mdiRefresh }}</v-icon>
                    Rotate API Key
                  </v-btn>
                </template>
                <v-card>
                  <v-card-title class="text-h5 text-break">
                    Are you sure you want to rotate your API Key?
                  </v-card-title>
                  <v-card-text>
                    You will have to logout and login again on the
                    <b>httpSMS</b> Android app with your new API key after you
                    rotate it.
                  </v-card-text>
                  <v-card-actions class="pb-4">
                    <v-btn
                      color="primary"
                      :loading="rotatingApiKey"
                      @click="rotateApiKey"
                    >
                      <v-icon left>{{ mdiRefresh }}</v-icon>
                      Yes Rotate Key
                    </v-btn>
                    <v-spacer></v-spacer>
                    <v-btn text @click="showRotateApiKey = false">
                      Close
                    </v-btn>
                  </v-card-actions>
                </v-card>
              </v-dialog>
            </div>
            <h5 id="webhook-settings" class="text-h4 mb-3 mt-12">Webhooks</h5>
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
                    <th class="text-left text-break">Callback URL</th>
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
                    <td class="text-break">{{ webhook.url }}</td>
                    <td v-if="$vuetify.breakpoint.lgAndUp" class="text-center">
                      <v-chip
                        v-for="event in webhook.events"
                        :key="event"
                        class="ma-1"
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
            <div class="d-flex">
              <v-btn color="primary" @click="onWebhookCreate">
                <v-icon left>{{ mdiLinkVariant }}</v-icon>
                Add webhook
              </v-btn>
              <v-btn
                v-if="$vuetify.breakpoint.lgAndUp"
                class="ml-4"
                href="https://docs.httpsms.com/webhooks/introduction"
                >Documentation</v-btn
              >
            </div>

            <h5 id="discord-settings" class="text-h4 mb-3 mt-12">
              Discord Integration
            </h5>
            <p class="text--secondary">
              Send and receive SMS messages without leaving your discord server
              with the httpSMS discord app using the
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

            <h5 id="phones" class="text-h4 mb-3 mt-12">Phones</h5>
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
                        small
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

            <h5 id="send-schedules" class="text-h4 mb-3 mt-12">
              Send Schedules
            </h5>
            <p class="text--secondary">
              Create availability schedules and attach them to each phone.
              Outgoing messages sent outside the schedule window are queued and
              delivered when the schedule opens according to your
              <a
                class="text-decoration-none"
                href="https://docs.httpsms.com/features/control-sms-send-rate"
                >configured send rate</a
              >.
            </p>
            <v-simple-table>
              <template #default>
                <thead>
                  <tr class="text-uppercase subtitle-2">
                    <th class="text-left">Name</th>
                    <th class="text-left">Timezone</th>
                    <th class="text-left">Schedule</th>
                    <th class="text-center">Action</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="schedule in sendSchedules" :key="schedule.id">
                    <td class="text-left pt-2" style="vertical-align: top">
                      {{ schedule.name }}
                    </td>
                    <td class="pt-2" style="vertical-align: top">
                      {{ schedule.timezone }}
                    </td>
                    <td class="py-2">
                      <div
                        v-for="line in scheduleSummary(schedule)"
                        :key="`${schedule.id}-${line}`"
                        class="mb-1"
                      >
                        {{ line[0] }}:
                        <span class="text--secondary">{{ line[1] }}</span>
                      </div>
                    </td>
                    <td class="text-center pt-2" style="vertical-align: top">
                      <v-btn
                        :icon="$vuetify.breakpoint.mdAndDown"
                        color="info"
                        small
                        :disabled="loadingSendSchedules"
                        @click.prevent="openEditSchedule(schedule)"
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
            <v-btn color="primary" class="mt-4" @click="openCreateSchedule">
              <v-icon left>{{ mdiCalendarClock }}</v-icon>
              Create Send Schedule
            </v-btn>

            <h5 id="email-notifications" class="text-h4 mb-3 mt-12">
              Email Notifications
            </h5>
            <p class="text--secondary">
              Manage the email notifications which you receive from httpSMS.
              Feel free to turn on/off individual notifications anytime so you
              don't get overloaded with emails
            </p>
            <v-switch
              v-model="notificationSettings.heartbeat_enabled"
              label="Heartbeat emails"
              :disabled="updatingEmailNotifications"
              hint="This switch controls email notifications we send when we don't receive a heartbeat from your phone for 1 hour."
              persistent-hint
            ></v-switch>
            <v-switch
              v-model="notificationSettings.webhook_enabled"
              label="Webhook and discord emails"
              :disabled="updatingEmailNotifications"
              hint="This switch controls email notifications we send when we can't forward events to your discord server or to your webhook."
              persistent-hint
            ></v-switch>
            <v-switch
              v-model="notificationSettings.message_status_enabled"
              label="Message status emails"
              :disabled="updatingEmailNotifications"
              hint="This switch controls email notifications we send when we your message is failed or expired."
              persistent-hint
            ></v-switch>
            <v-switch
              v-model="notificationSettings.newsletter_enabled"
              label="Newsletter emails"
              :disabled="updatingEmailNotifications"
              hint="This switch controls newsletter emails about new features, updates, and promotions."
              persistent-hint
            ></v-switch>
            <v-btn
              color="primary"
              :loading="updatingEmailNotifications"
              class="mt-4"
              @click="saveEmailNotifications"
            >
              <v-icon left>{{ mdiContentSave }}</v-icon>
              Save Notification Settings
            </v-btn>

            <h5 id="email-notifications" class="text-h4 error--text mb-3 mt-12">
              Delete Account
            </h5>
            <p v-if="hasActiveSubscription" class="text--secondary">
              You cannot delete your account because you have an active
              subscription on httpSMS.
              <router-link class="text-decoration-none" to="/billing"
                >Cancel your subscription</router-link
              >
              before deleting your account.
            </p>
            <p v-else class="text--secondary">
              You can delete all your data on httpSMS by clicking the button
              below. This action is <b>irreversible</b> and all your data will
              be permanently deleted from the httpSMS database instantly and it
              cannot be recovered.
            </p>
            <v-btn
              color="error"
              :loading="deletingAccount"
              class="mt-4"
              :disabled="hasActiveSubscription"
              @click="showDeleteAccountDialog = true"
            >
              <v-icon left>{{ mdiDelete }}</v-icon>
              Delete your Account
            </v-btn>
            <v-dialog
              v-model="showDeleteAccountDialog"
              overlay-opacity="0.9"
              max-width="600px"
            >
              <v-card>
                <v-card-title class="justify-center text-center"
                  >Delete your httpSMS account</v-card-title
                >
                <v-card-text class="mt-2 text-center">
                  Are you sure you want to delete your account? This action is
                  <b>irreversible</b> and all your data will be permanently
                  deleted from the httpSMS database instantly.
                </v-card-text>
                <v-card-actions>
                  <v-btn
                    color="error"
                    text
                    :loading="deletingAccount"
                    @click="deleteUserAccount"
                  >
                    <v-icon v-if="$vuetify.breakpoint.lgAndUp" left>{{
                      mdiDelete
                    }}</v-icon>
                    Delete My Account
                  </v-btn>
                  <v-spacer></v-spacer>
                  <v-btn
                    color="primary"
                    @click="showDeleteAccountDialog = false"
                  >
                    <span v-if="$vuetify.breakpoint.lgAndUp"
                      >Keep My account</span
                    >
                    <span v-else>Close</span>
                  </v-btn>
                </v-card-actions>
              </v-card>
            </v-dialog>
          </v-col>
        </v-row>
      </v-container>
    </div>

    <v-dialog v-model="showPhoneEdit" overlay-opacity="0.9" max-width="700px">
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
                  min="1"
                  max="5"
                  :rules="[
                    (v) =>
                      (v >= 1 && v <= 5) ||
                      'Max send attempts must be between 1 and 5',
                  ]"
                >
                </v-text-field>
                <v-autocomplete
                  v-model="activePhone.message_send_schedule_id"
                  outlined
                  :readonly="sendSchedules.length === 0"
                  dense
                  clearable
                  label="Send Schedule"
                  :items="sendSchedules"
                  item-text="name"
                  item-value="id"
                  :multiple="false"
                  hint="Attach a send schedule to this phone"
                  persistent-hint
                ></v-autocomplete>
                <v-textarea
                  v-model="activePhone.missed_call_auto_reply"
                  outlined
                  dense
                  label="Missed Call AutoReply"
                  persistent-placeholder
                  persistent-hint
                  placeholder="We are currently closed at the moment, please send us a text message from  09:00 to 17:00"
                  hint="Here you can configure an automated SMS message which is sent to the caller when this phone has a missed call"
                >
                </v-textarea>
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
          <v-btn
            small
            color="error"
            text
            @click="deletePhone(activePhone?.id ?? '')"
          >
            <v-icon v-if="$vuetify.breakpoint.lgAndUp" small>
              {{ mdiDelete }}
            </v-icon>
            Delete
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="showWebhookEdit" overlay-opacity="0.9" max-width="600px">
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
                label="Signing Key (optional)"
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
              <v-select
                v-model="activeWebhook.phone_numbers"
                :items="phoneNumbers"
                label="Phone Numbers"
                multiple
                outlined
                persistent-placeholder
                class="mt-6"
                dense
                :error="errorMessages.has('phone_numbers')"
                :error-messages="errorMessages.get('phone_numbers')"
                hint="Select multiple phone numbers to watch for events"
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

    <v-dialog v-model="showDiscordEdit" overlay-opacity="0.9" max-width="700px">
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

    <v-dialog
      v-model="showScheduleEdit"
      overlay-opacity="0.9"
      max-width="800px"
    >
      <v-card>
        <v-card-title>
          <span v-if="!activeSchedule.id">Create Message Send Schedule</span>
          <span v-else>Edit Message Send Schedule</span>
        </v-card-title>
        <v-card-text
          class="mt-4"
          :class="{ 'px-2': $vuetify.breakpoint.mdAndDown }"
        >
          <v-row>
            <v-col cols="12" md="6">
              <v-text-field
                v-model="activeSchedule.name"
                outlined
                dense
                persistent-placeholder
                label="Schedule Name"
                placeholder="e.g Business Hours"
                :error="errorMessages.has('name')"
                :error-messages="errorMessages.get('name')"
              />
            </v-col>
            <v-col cols="12" md="6">
              <v-autocomplete
                v-model="activeSchedule.timezone"
                dense
                outlined
                :items="timezones"
                label="Timezone"
                :class="{ 'mt-n6 mb-0': $vuetify.breakpoint.mdAndDown }"
                :error="errorMessages.has('timezone')"
                :error-messages="errorMessages.get('timezone')"
              />
            </v-col>
          </v-row>
          <v-card
            :outlined="!$vuetify.breakpoint.mdAndDown"
            elevation="0"
            class="px-0"
          >
            <v-card-text
              :class="{
                'px-2 mt-n8': $vuetify.breakpoint.mdAndDown,
                'px-4': !$vuetify.breakpoint.mdAndDown,
              }"
            >
              <div
                v-for="day in weekDays"
                :key="day.value"
                :class="[
                  $vuetify.breakpoint.smAndUp ? 'd-flex align-start' : '',
                  'mb-4',
                ]"
              >
                <div
                  :class="[$vuetify.breakpoint.smAndUp ? 'pr-4' : '', 'pt-2']"
                  :style="$vuetify.breakpoint.smAndUp ? 'min-width: 160px' : ''"
                >
                  <v-switch
                    :input-value="scheduleDayEnabled(day.value)"
                    inset
                    dense
                    :label="day.label"
                    class="mt-0 pt-0 text--primary"
                    @change="scheduleToggleDay(day.value, $event)"
                  />
                </div>
                <div class="pt-2 flex-grow-1">
                  <div
                    v-for="(window, index) in scheduleWindowsForDay(day.value)"
                    :key="`${day.value}-${index}`"
                    class="d-flex align-center flex-wrap mb-2"
                  >
                    <div
                      class="mr-2 mb-2"
                      style="width: 110px; max-width: 100%"
                    >
                      <v-text-field
                        v-model="window.start_time"
                        dense
                        outlined
                        :error="scheduleWindowError(day.value)"
                        type="time"
                        label="Start"
                        hide-details="auto"
                      />
                    </div>
                    <div class="mb-2 mr-2">–</div>
                    <div
                      class="mr-2 mb-2"
                      style="width: 110px; max-width: 100%"
                    >
                      <v-text-field
                        v-model="window.end_time"
                        dense
                        outlined
                        :error="scheduleWindowError(day.value)"
                        type="time"
                        label="End"
                        hide-details="auto"
                      />
                    </div>
                    <div class="mb-2">
                      <v-btn
                        v-if="index == 0"
                        icon
                        color="primary"
                        @click="scheduleAddWindow(day.value)"
                      >
                        <v-icon>{{ mdiPlus }}</v-icon>
                      </v-btn>
                      <v-btn
                        icon
                        color="error"
                        @click="scheduleRemoveWindow(day.value, index)"
                      >
                        <v-icon>{{ mdiDelete }}</v-icon>
                      </v-btn>
                    </div>
                  </div>
                  <div
                    v-if="scheduleWindowError(day.value)"
                    class="w-full error--text mt-n2 mb-4"
                  >
                    {{ scheduleWindowError(day.value) }}
                  </div>
                </div>
              </div>
            </v-card-text>
          </v-card>
        </v-card-text>
        <v-card-actions class="pb-4">
          <loading-button
            v-if="!activeSchedule.id"
            :icon="mdiContentSave"
            :loading="savingSchedule"
            @click="saveSchedule"
          >
            Save Schedule
          </loading-button>
          <loading-button
            v-else
            color="info"
            :loading="savingSchedule"
            @click="saveSchedule"
          >
            <v-icon v-if="$vuetify.breakpoint.lgAndUp" small>
              {{ mdiContentSave }}
            </v-icon>
            Update Schedule
          </loading-button>
          <v-spacer></v-spacer>
          <v-btn
            v-if="activeSchedule.id"
            :disabled="savingSchedule"
            color="error"
            text
            @click="confirmDeleteSchedule"
          >
            <v-icon v-if="$vuetify.breakpoint.lgAndUp" small left>
              {{ mdiDelete }}
            </v-icon>
            Delete
          </v-btn>
          <v-btn v-else text color="error" @click="showScheduleEdit = false">
            Close
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="showScheduleDelete" max-width="500">
      <v-card>
        <v-card-title>Delete schedule</v-card-title>
        <v-card-text>
          Are you sure you want to delete <b>{{ activeSchedule.name }}</b
          >? Phones attached to this schedule will no longer have schedule-based
          restrictions.
        </v-card-text>
        <v-card-actions>
          <v-btn
            color="error"
            :loading="savingSchedule"
            @click="deleteSchedule"
          >
            Delete
          </v-btn>
          <v-spacer />
          <v-btn text @click="showScheduleDelete = false">Cancel</v-btn>
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
  mdiCalendarClock,
  mdiShieldCheck,
  mdiDelete,
  mdiContentSave,
  mdiConnection,
  mdiEye,
  mdiRefresh,
  mdiLinkVariant,
  mdiEyeOff,
  mdiPlus,
  mdiSquareEditOutline,
  mdiQrcode,
} from '@mdi/js'
import { toCanvas } from 'qrcode'
import { ErrorMessages } from '~/plugins/errors'
import LoadingButton from '~/components/LoadingButton.vue'
import {
  EntitiesDiscord,
  EntitiesMessageSendSchedule,
  EntitiesPhone,
  EntitiesWebhook,
} from '~/models/api'

export default Vue.extend({
  name: 'SettingsIndex',
  components: { LoadingButton },
  middleware: ['auth'],
  data() {
    return {
      mdiEye,
      mdiEyeOff,
      mdiRefresh,
      mdiArrowLeft,
      mdiAccountCircle,
      mdiCalendarClock,
      mdiShieldCheck,
      mdiDelete,
      mdiPlus,
      mdiQrcode,
      mdiLinkVariant,
      mdiContentSave,
      mdiSquareEditOutline,
      mdiConnection,
      errorMessages: new ErrorMessages(),
      apiKeyShow: false,
      showPhoneEdit: false,
      showDiscordEdit: false,
      showScheduleEdit: false,
      showScheduleDelete: false,
      savingSchedule: false,
      showRotateApiKey: false,
      rotatingApiKey: false,
      showQrCodeDialog: false,
      deletingAccount: false,
      showDeleteAccountDialog: false,
      activeWebhook: {
        id: null,
        url: '',
        signing_key: '',
        phone_numbers: [] as string[],
        events: ['message.phone.received'],
      } as {
        id: string | null
        url: string
        signing_key: string
        phone_numbers: string[]
        events: string[]
      },
      activeDiscord: {
        id: null,
        name: '',
        server_id: '',
        incoming_channel_id: '',
      } as {
        id: string | null
        name: string
        server_id: string
        incoming_channel_id: string
      },
      updatingEmailNotifications: false,
      notificationSettings: {
        webhook_enabled: true,
        message_status_enabled: true,
        newsletter_enabled: true,
        heartbeat_enabled: true,
      },
      updatingWebhook: false,
      loadingWebhooks: false,
      loadingSendSchedules: false,
      discords: [] as EntitiesDiscord[],
      webhooks: [] as EntitiesWebhook[],
      showWebhookEdit: false,
      activePhone: null as EntitiesPhone | null,
      updatingPhone: false,
      updatingDiscord: false,
      loadingDiscordIntegrations: false,
      sendSchedules: [] as EntitiesMessageSendSchedule[],
      events: [
        'message.phone.received',
        'message.phone.sent',
        'message.phone.delivered',
        'message.send.failed',
        'message.send.expired',
        'message.call.missed',
        'phone.heartbeat.offline',
        'phone.heartbeat.online',
      ],
      activeSchedule: {
        id: null,
        name: '',
        timezone: '',
        windows: [],
      } as {
        id: string | null
        name: string
        timezone: string
        windows: Array<{
          day_of_week: number
          start_time: string
          end_time: string
        }>
      },
      weekDays: [
        { value: 1, label: 'Monday' },
        { value: 2, label: 'Tuesday' },
        { value: 3, label: 'Wednesday' },
        { value: 4, label: 'Thursday' },
        { value: 5, label: 'Friday' },
        { value: 6, label: 'Saturday' },
        { value: 0, label: 'Sunday' },
      ],
    }
  },
  head() {
    return {
      title: 'Settings - httpSMS',
    }
  },
  computed: {
    apiKey() {
      if (this.$store.getters.getUser === null) {
        return ''
      }
      return this.$store.getters.getUser.api_key
    },
    hasActiveSubscription() {
      if (this.$store.getters.getUser === null) {
        return true
      }
      return this.$store.getters.getUser.subscription_renews_at != null
    },
    timezones() {
      try {
        // @ts-ignore
        return Intl.supportedValuesOf('timeZone')
      } catch {
        return []
      }
    },
    phoneNumbers() {
      return this.$store.getters.getPhones.map((phone: EntitiesPhone) => {
        return phone.phone_number
      })
    },
  },
  watch: {
    showQrCodeDialog(newVal) {
      if (newVal && this.apiKey) {
        this.$nextTick(() => {
          this.generateQrCode(this.apiKey)
        })
      }
    },
  },
  async mounted() {
    await Promise.all([
      this.$store.dispatch('clearAxiosError'),
      this.$store.dispatch('loadUser'),
      this.$store.dispatch('loadPhones'),
    ])
    this.loadWebhooks()
    this.loadDiscordIntegrations()
    this.loadSendSchedules()
    this.updateEmailNotifications()
    if (this.$route.hash) {
      await this.$vuetify.goTo(this.$route.hash)
    }
  },

  methods: {
    generateQrCode(text: string) {
      const canvas = this.$refs.qrCodeCanvas
      if (canvas) {
        toCanvas(canvas, text, { errorCorrectionLevel: 'H' }, (err) => {
          if (err) {
            this.$store.dispatch('addNotification', {
              message: 'Failed to generate API key QR code',
              type: 'error',
            })
          }
        })
      }
    },
    updateEmailNotifications() {
      this.notificationSettings = {
        webhook_enabled:
          this.$store.getters.getUser.notification_webhook_enabled,
        message_status_enabled:
          this.$store.getters.getUser.notification_message_status_enabled,
        heartbeat_enabled:
          this.$store.getters.getUser.notification_heartbeat_enabled,
        newsletter_enabled:
          this.$store.getters.getUser.notification_newsletter_enabled,
      }
    },
    showEditPhone(phoneId: string) {
      const phone = this.$store.getters.getPhones.find(
        (x: EntitiesPhone) => x.id === phoneId,
      )
      if (!phone) {
        return
      }
      this.activePhone = { ...phone }
      this.showPhoneEdit = true
      this.resetErrors()
    },

    onWebhookEdit(webhookId: string) {
      const webhook = this.webhooks.find(
        (x: EntitiesWebhook) => x.id === webhookId,
      ) as EntitiesWebhook | undefined
      if (!webhook) {
        return
      }
      this.activeWebhook = {
        id: webhook.id,
        url: webhook.url,
        phone_numbers: webhook.phone_numbers.filter(
          (x: string) =>
            this.phoneNumbers.find((y: string) => y === x) !== undefined,
        ),
        signing_key: webhook.signing_key,
        events: webhook.events,
      }
      this.showWebhookEdit = true
      this.resetErrors()
    },

    onDiscordEdit(discordId: string) {
      const discord = this.discords.find(
        (x: EntitiesDiscord) => x.id === discordId,
      ) as EntitiesDiscord | undefined
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
        phone_numbers: this.$store.getters.getPhones.map(
          (phone: EntitiesPhone) => phone.phone_number,
        ),
        events: [
          'message.phone.received',
          'message.phone.sent',
          'message.phone.delivered',
          'message.send.failed',
          'message.send.expired',
        ],
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
        .catch((errors: ErrorMessages) => {
          this.errorMessages = errors
        })
        .finally(() => {
          this.updatingDiscord = false
        })
    },

    saveEmailNotifications() {
      this.updatingEmailNotifications = true
      this.$store
        .dispatch('saveEmailNotifications', this.notificationSettings)
        .then(() => {
          this.$store.dispatch('addNotification', {
            message: 'Email notifications saved successfully',
            type: 'success',
          })
          this.updateEmailNotifications()
        })
        .finally(() => {
          this.updatingEmailNotifications = false
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
        .catch((errors: ErrorMessages) => {
          this.errorMessages = errors
        })
        .finally(() => {
          this.updatingDiscord = false
        })
    },

    deleteDiscord(discordId: string) {
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
        .catch((errors: ErrorMessages) => {
          this.errorMessages = errors
        })
        .finally(() => {
          this.updatingWebhook = false
        })
    },

    updateTimezone(timezone: string) {
      this.resetErrors()
      this.$store
        .dispatch('updateTimezone', timezone)
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
        .catch((errors: ErrorMessages) => {
          this.errorMessages = errors
        })
        .finally(() => {
          this.updatingWebhook = false
        })
    },

    rotateApiKey() {
      this.rotatingApiKey = true
      this.$store
        .dispatch('rotateApiKey', this.$store.getters.getUser.id)
        .finally(() => {
          this.rotatingApiKey = false
          this.showRotateApiKey = false
        })
    },

    deleteWebhook(webhookId: string) {
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

    loadSendSchedules() {
      this.loadingSendSchedules = true
      this.$store
        .dispatch('getSendSchedules')
        .then((sendSchedules: EntitiesMessageSendSchedule[]) => {
          this.sendSchedules = sendSchedules
        })
        .finally(() => {
          this.loadingSendSchedules = false
        })
    },

    loadWebhooks() {
      this.loadingWebhooks = true
      this.$store
        .dispatch('getWebhooks')
        .then((webhooks: EntitiesWebhook[]) => {
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
        .then((discords: EntitiesDiscord[]) => {
          this.discords = discords
        })
        .finally(() => {
          this.loadingDiscordIntegrations = false
        })
    },

    deleteUserAccount() {
      this.deletingAccount = true
      this.$store
        .dispatch('deleteUserAccount')
        .then((message: string) => {
          this.$store.dispatch('addNotification', {
            message: message ?? 'Your account has been deleted successfully',
            type: 'success',
          })
          this.$fire.auth.signOut().then(() => {
            this.$store.dispatch('setAuthUser', null)
            this.$store.dispatch('resetState')
            this.$store.dispatch('addNotification', {
              type: 'info',
              message: 'You have successfully logged out',
            })
            this.$router.push({ name: 'index' })
          })
        })
        .finally(() => {
          this.deletingAccount = false
          this.showDeleteAccountDialog = false
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

    minuteToClock(value: number) {
      const hours = String(Math.floor(value / 60)).padStart(2, '0')
      const minutes = String(value % 60).padStart(2, '0')
      return `${hours}:${minutes}`
    },

    clockToMinute(value: string) {
      if (!value || !value.includes(':')) {
        return 0
      }

      const [hours, minutes] = value.split(':').map((x) => parseInt(x, 10))
      return hours * 60 + minutes
    },

    getWeekday(index: number): string {
      return this.weekDays.find((x) => x.value === index)?.label ?? ''
    },

    scheduleSummary(schedule: EntitiesMessageSendSchedule) {
      return this.weekDays
        .map((day) => {
          const windows = (schedule.windows || []).filter(
            (x) => x.day_of_week === day.value,
          )

          if (windows.length === 0) {
            return []
          }
          return [
            day.label,
            windows
              .map(
                (w) =>
                  `${this.minuteToClock(w.start_minute)} - ${this.minuteToClock(
                    w.end_minute,
                  )}`,
              )
              .join(', '),
          ]
        })
        .filter((x: string[]) => x.length > 0)
    },

    defaultTimezone(): string {
      return (
        this.$store.getters.getUser?.timezone ||
        Intl.DateTimeFormat().resolvedOptions().timeZone
      )
    },

    openCreateSchedule() {
      this.resetErrors()
      this.activeSchedule = {
        id: null,
        name: '',
        timezone: this.defaultTimezone(),
        windows: [
          { day_of_week: 1, start_time: '09:00', end_time: '17:00' },
          { day_of_week: 2, start_time: '09:00', end_time: '17:00' },
          { day_of_week: 3, start_time: '09:00', end_time: '17:00' },
          { day_of_week: 4, start_time: '09:00', end_time: '17:00' },
          { day_of_week: 5, start_time: '09:00', end_time: '17:00' },
        ],
      }
      this.showScheduleEdit = true
    },

    openEditSchedule(schedule: EntitiesMessageSendSchedule) {
      this.resetErrors()
      this.activeSchedule = {
        id: schedule.id,
        name: schedule.name,
        timezone: schedule.timezone,
        windows: (schedule.windows || []).map((x) => ({
          day_of_week: x.day_of_week,
          start_time: this.minuteToClock(x.start_minute),
          end_time: this.minuteToClock(x.end_minute),
        })),
      }
      this.showScheduleEdit = true
    },

    scheduleWindowsForDay(dayOfWeek: number) {
      return this.activeSchedule.windows.filter(
        (x) => x.day_of_week === dayOfWeek,
      )
    },

    scheduleDayEnabled(dayOfWeek: number) {
      return this.scheduleWindowsForDay(dayOfWeek).length > 0
    },

    scheduleToggleDay(dayOfWeek: number, enabled: boolean) {
      if (enabled) {
        if (!this.scheduleDayEnabled(dayOfWeek)) {
          this.scheduleAddWindow(dayOfWeek)
        }
        return
      }
      this.activeSchedule.windows = this.activeSchedule.windows.filter(
        (x) => x.day_of_week !== dayOfWeek,
      )
    },

    scheduleAddWindow(dayOfWeek: number) {
      this.activeSchedule.windows.push({
        day_of_week: dayOfWeek,
        start_time: '09:00',
        end_time: '17:00',
      })
    },

    scheduleWindowError(index: number): string | null {
      const messages = this.errorMessages.has('windows')
        ? this.errorMessages.get('windows')
        : []
      if (messages.length === 0) {
        return null
      }

      const message = messages.find((x: string) =>
        x.includes(`Day of week ${index}`),
      )
      return message
        ? message.replace(`Day of week ${index}`, this.getWeekday(index))
        : null
    },

    scheduleRemoveWindow(dayOfWeek: number, index: number) {
      const matches = this.activeSchedule.windows.filter(
        (x) => x.day_of_week === dayOfWeek,
      )
      const target = matches[index]
      this.activeSchedule.windows = this.activeSchedule.windows.filter(
        (x) => x !== target,
      )
    },

    async saveSchedule() {
      this.resetErrors()
      this.savingSchedule = true

      try {
        const payload = {
          name: this.activeSchedule.name,
          timezone: this.activeSchedule.timezone,
          windows: (this.activeSchedule.windows || []).map((window) => ({
            day_of_week: window.day_of_week,
            start_minute: this.clockToMinute(window.start_time),
            end_minute: this.clockToMinute(window.end_time),
          })),
        }

        if (this.activeSchedule.id) {
          await this.$store.dispatch('updateSendSchedule', {
            id: this.activeSchedule.id,
            ...payload,
          })
        } else {
          await this.$store.dispatch('createSendSchedule', payload)
        }

        this.$store.dispatch('addNotification', {
          type: 'success',
          message: 'Send schedule saved successfully',
        })

        this.showScheduleEdit = false
        this.loadSendSchedules()
      } catch (errors) {
        if (errors instanceof ErrorMessages) {
          this.errorMessages = errors
        } else {
          this.$store.dispatch('addNotification', {
            type: 'error',
            message: 'Failed to save send schedule',
          })
        }
      } finally {
        this.savingSchedule = false
      }
    },

    confirmDeleteSchedule() {
      this.showScheduleDelete = true
    },

    async deleteSchedule() {
      if (!this.activeSchedule.id) {
        return
      }

      this.savingSchedule = true

      try {
        await this.$store.dispatch('deleteSendSchedule', this.activeSchedule.id)

        this.$store.dispatch('addNotification', {
          type: 'success',
          message: 'Send schedule deleted successfully',
        })

        this.showScheduleDelete = false
        this.showScheduleEdit = false
        this.loadSendSchedules()
      } catch (error) {
        this.$store.dispatch('addNotification', {
          type: 'error',
          message: 'Failed to delete send schedule',
        })
      } finally {
        this.savingSchedule = false
      }
    },
  },
})
</script>
