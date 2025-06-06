<template>
  <v-container
    fluid
    class="px-0 pt-0 pb-0"
    :fill-height="$vuetify.breakpoint.lgAndUp"
  >
    <div class="w-full h-full">
      <v-app-bar height="60" :dense="$vuetify.breakpoint.mdAndDown">
        <v-btn v-if="$vuetify.breakpoint.mdAndDown" icon to="/threads">
          <v-icon>{{ mdiArrowLeft }}</v-icon>
        </v-btn>
        <v-toolbar-title>
          <span v-if="$store.getters.hasThread">
            {{ $store.getters.getThread.contact | phoneNumber }}
          </span>
        </v-toolbar-title>
        <v-spacer></v-spacer>
        <v-menu offset-y>
          <template #activator="{ on }">
            <v-btn icon text class="mt-2" v-on="on">
              <v-icon>{{ mdiDotsVertical }}</v-icon>
            </v-btn>
          </template>
          <v-list class="px-2" nav :dense="$vuetify.breakpoint.mdAndDown">
            <v-list-item-group v-model="selectedMenuItem">
              <v-list-item
                v-if="
                  $store.getters.hasThread &&
                  !$store.getters.getThread.is_archived
                "
                @click.prevent="archiveThread"
              >
                <v-list-item-icon class="pl-2">
                  <v-icon dense>{{ mdiPackageDown }}</v-icon>
                </v-list-item-icon>
                <v-list-item-content class="ml-n3">
                  <v-list-item-title class="pr-16 py-1">
                    <span :class="{ 'pr-16': $vuetify.breakpoint.mdAndUp }">
                      Archive
                    </span>
                  </v-list-item-title>
                </v-list-item-content>
              </v-list-item>
              <v-list-item
                v-if="
                  $store.getters.hasThread &&
                  $store.getters.getThread.is_archived
                "
                @click.prevent="unArchiveThread"
              >
                <v-list-item-icon class="pl-2">
                  <v-icon dense>{{ mdiPackageUp }}</v-icon>
                </v-list-item-icon>
                <v-list-item-content class="ml-n3">
                  <v-list-item-title class="pr-16 py-1">
                    <span :class="{ 'pr-16': $vuetify.breakpoint.mdAndUp }">
                      Unarchive
                    </span>
                  </v-list-item-title>
                </v-list-item-content>
              </v-list-item>
              <v-list-item
                v-if="$store.getters.hasThread"
                @click.prevent="deleteThread($store.getters.getThread.id)"
              >
                <v-list-item-icon class="pl-2">
                  <v-icon dense color="error">{{ mdiDelete }}</v-icon>
                </v-list-item-icon>
                <v-list-item-content class="ml-n3">
                  <v-list-item-title class="pr-16 py-1">
                    Delete Thread
                  </v-list-item-title>
                </v-list-item-content>
              </v-list-item>
            </v-list-item-group>
          </v-list>
        </v-menu>
      </v-app-bar>
      <v-progress-linear
        v-if="loadingMessages"
        color="primary"
        indeterminate
      ></v-progress-linear>
      <v-container v-if="$store.getters.hasThread">
        <div
          ref="messageBody"
          class="messages-body no-scrollbar"
          :class="{ 'pr-7': $vuetify.breakpoint.lgAndUp }"
        >
          <v-row
            v-for="message in messages"
            :key="message.id"
            :style="{ visibility: messageVisibility }"
          >
            <v-col
              class="d-flex"
              :class="{
                'pr-12': $vuetify.breakpoint.mdAndDown && !isMT(message),
                'pl-12 pr-8': $vuetify.breakpoint.mdAndDown && isMT(message),
                'pl-16 ml-16': $vuetify.breakpoint.lgAndUp && isMT(message),
                'pr-16 mr-16': $vuetify.breakpoint.lgAndUp && !isMT(message),
              }"
            >
              <v-spacer v-if="isMT(message)"></v-spacer>
              <v-avatar
                v-if="isMo(message)"
                :color="$store.getters.getThread.color"
              >
                <v-icon>{{ mdiAccount }}</v-icon>
              </v-avatar>
              <v-avatar v-if="isMissedCall(message)" color="#1e1e1e">
                <v-icon large color="red">{{ mdiCallMissed }}</v-icon>
              </v-avatar>
              <v-menu v-if="isMT(message)" offset-y>
                <template #activator="{ on }">
                  <v-btn icon text class="mt-2" v-on="on">
                    <v-icon>{{ mdiDotsVertical }}</v-icon>
                  </v-btn>
                </template>
                <v-list class="px-2" nav dense>
                  <v-list-item-group v-model="selectedMenuItem">
                    <v-list-item
                      v-if="canResend(message)"
                      @click.prevent="resendMessage(message)"
                    >
                      <v-list-item-icon class="pl-2">
                        <v-icon dense>{{ mdiRefresh }}</v-icon>
                      </v-list-item-icon>
                      <v-list-item-content class="ml-n3">
                        <v-list-item-title class="pr-16 py-1">
                          Resend Message
                        </v-list-item-title>
                      </v-list-item-content>
                    </v-list-item>
                    <v-list-item @click.prevent="copyMessageId(message)">
                      <v-list-item-icon class="pl-2">
                        <v-icon dense>{{ mdiContentCopy }}</v-icon>
                      </v-list-item-icon>
                      <v-list-item-content class="ml-n3">
                        <v-list-item-title class="pr-16 py-1">
                          Copy Message ID
                        </v-list-item-title>
                      </v-list-item-content>
                    </v-list-item>
                    <v-list-item @click.prevent="deleteMessage(message)">
                      <v-list-item-icon class="pl-2">
                        <v-icon dense color="error">{{ mdiDelete }}</v-icon>
                      </v-list-item-icon>
                      <v-list-item-content class="ml-n3">
                        <v-list-item-title class="pr-16 py-1">
                          Delete Message
                        </v-list-item-title>
                      </v-list-item-content>
                    </v-list-item>
                  </v-list-item-group>
                </v-list>
              </v-menu>
              <div>
                <v-card
                  class="ml-2"
                  shaped
                  :color="isMT(message) ? 'primary' : 'default'"
                >
                  <v-card-text
                    class="text--primary text-break"
                    style="white-space: pre-line"
                  >
                    <span v-if="!isMissedCall(message)">{{
                      message.content
                    }}</span>
                    <span v-else class="text--secondary"
                      >Missed phone call</span
                    >
                  </v-card-text>
                </v-card>
                <div class="d-flex">
                  <p class="ml-2 text--secondary caption mr-2">
                    {{ new Date(message.order_timestamp).toLocaleString() }}
                  </p>
                  <v-spacer></v-spacer>
                  <v-tooltip bottom>
                    <template #activator="{ on, attrs }">
                      <div v-bind="attrs" v-on="on">
                        <v-icon
                          v-if="message.status === 'expired'"
                          color="warning"
                          class="mt-n2"
                          >{{ mdiAlert }}</v-icon
                        >
                        <v-progress-circular
                          v-else-if="isPending(message)"
                          indeterminate
                          :size="14"
                          :width="1"
                          class="mt-n2"
                          :color="statusColor(message)"
                        ></v-progress-circular>
                        <v-icon
                          v-else-if="message.status === 'delivered'"
                          color="primary"
                          class="mt-n6"
                        >
                          {{ mdiCheckAll }}
                        </v-icon>
                        <v-icon
                          v-else-if="message.status === 'sent'"
                          class="mt-n6"
                        >
                          {{ mdiCheck }}
                        </v-icon>
                        <v-icon
                          v-else-if="message.status === 'failed'"
                          color="error"
                          class="mt-n2"
                        >
                          {{ mdiAlert }}
                        </v-icon>
                      </div>
                    </template>
                    <span>
                      {{
                        message.failure_reason
                          ? message.failure_reason
                          : message.status
                      }}
                    </span>
                  </v-tooltip>
                </div>
              </div>
              <v-menu v-if="!isMT(message)" offset-y>
                <template #activator="{ on }">
                  <v-btn icon text class="mt-2" v-on="on">
                    <v-icon>{{ mdiDotsVertical }}</v-icon>
                  </v-btn>
                </template>
                <v-list class="px-2" nav dense>
                  <v-list-item-group v-model="selectedMenuItem">
                    <v-list-item
                      v-if="canResend(message)"
                      @click.prevent="resendMessage(message)"
                    >
                      <v-list-item-icon class="pl-2">
                        <v-icon dense>{{ mdiRefresh }}</v-icon>
                      </v-list-item-icon>
                      <v-list-item-content class="ml-n3">
                        <v-list-item-title class="pr-16 py-1">
                          Resend Message
                        </v-list-item-title>
                      </v-list-item-content>
                    </v-list-item>
                    <v-list-item @click.prevent="copyMessageId(message)">
                      <v-list-item-icon class="pl-2">
                        <v-icon dense>{{ mdiContentCopy }}</v-icon>
                      </v-list-item-icon>
                      <v-list-item-content class="ml-n3">
                        <v-list-item-title class="pr-16 py-1">
                          Copy Message ID
                        </v-list-item-title>
                      </v-list-item-content>
                    </v-list-item>
                    <v-list-item @click.prevent="deleteMessage(message)">
                      <v-list-item-icon class="pl-2">
                        <v-icon dense color="error">{{ mdiDelete }}</v-icon>
                      </v-list-item-icon>
                      <v-list-item-content class="ml-n3">
                        <v-list-item-title class="pr-16 py-1">
                          Delete Message
                        </v-list-item-title>
                      </v-list-item-content>
                    </v-list-item>
                  </v-list-item-group>
                </v-list>
              </v-menu>
            </v-col>
          </v-row>
        </div>
        <v-footer absolute padless color="#121212">
          <v-container class="pb-0">
            <v-form
              ref="form"
              class="d-flex"
              lazy-validation
              @submit.prevent="sendMessage"
            >
              <v-text-field
                ref="messageInput"
                v-model="formMessage"
                :disabled="submitting || !contactIsPhoneNumber"
                :rows="1"
                filled
                class="no-scrollbar ml-2"
                :rules="formMessageRules"
                :placeholder="
                  contactIsPhoneNumber
                    ? 'Type your message here'
                    : 'You cannot send messages to ' + contact
                "
                rounded
                @keydown.enter="sendMessage"
              ></v-text-field>
              <v-btn
                :disabled="submitting || !contactIsPhoneNumber"
                type="submit"
                color="primary"
                class="white--text ml-2"
                fab
              >
                <v-progress-circular
                  v-if="submitting"
                  indeterminate
                  style="position: absolute"
                  :size="20"
                  :width="3"
                  color="pink"
                ></v-progress-circular>
                <v-icon>{{ mdiSend }}</v-icon>
              </v-btn>
            </v-form>
          </v-container>
        </v-footer>
      </v-container>
    </div>
  </v-container>
</template>

<script lang="ts">
import Vue from 'vue'
import { InputValidationRules } from 'vuetify'
import { isValidPhoneNumber } from 'libphonenumber-js'
import {
  mdiSend,
  mdiDotsVertical,
  mdiArrowLeft,
  mdiCheckAll,
  mdiDelete,
  mdiCallMissed,
  mdiCheck,
  mdiAlert,
  mdiPackageUp,
  mdiPackageDown,
  mdiAccount,
  mdiRefresh,
  mdiSim,
  mdiContentCopy,
} from '@mdi/js'
import Pusher, { Channel } from 'pusher-js'
import { Message } from '~/models/message'
import { NotificationRequest, SendMessageRequest, SIM } from '~/store'

export default Vue.extend({
  middleware: ['auth'],
  data() {
    const formMessageRules: InputValidationRules = [
      (v) =>
        v === '' ||
        (v && v.length <= 320) ||
        'Message must be less than 320 characters',
    ]
    return {
      mdiSend,
      mdiDotsVertical,
      mdiArrowLeft,
      mdiCheckAll,
      mdiCallMissed,
      mdiCheck,
      mdiAlert,
      mdiDelete,
      hideMessages: true,
      loadingMessages: false,
      messages: [] as Message[],
      mdiPackageUp,
      mdiPackageDown,
      mdiAccount,
      mdiContentCopy,
      mdiRefresh,
      mdiSim,
      simOptions: [
        { title: 'Default', code: 'DEFAULT', value: 0 },
        { title: 'SIM 1', code: 'SIM1', value: 1 },
        { title: 'SIM 2', code: 'SIM2', value: 2 },
      ],
      simSelected: { title: 'Default', code: 'DEFAULT', value: 0 },
      formMessage: '',
      formMessageRules,
      submitting: false,
      webhookChannel: null as Channel | null,
      selectedMenuItem: -1,
    }
  },
  head() {
    return {
      title: 'Messages - httpSMS',
    }
  },
  computed: {
    contactIsPhoneNumber(): boolean {
      return (
        isValidPhoneNumber(this.$store.getters.getThread.contact) ||
        !isNaN(Number(this.$store.getters.getThread.contact))
      )
    },
    messageVisibility(): string {
      if (this.hideMessages) {
        return 'hidden'
      }
      return 'visible'
    },
    contact(): string {
      return this.$store.getters.getThread.contact
    },
  },

  async mounted() {
    await this.loadData()

    const pusher = new Pusher(this.$config.pusherKey, {
      cluster: this.$config.pusherCluster,
    })
    this.webhookChannel = pusher.subscribe(this.$store.getters.getAuthUser.id)
    this.webhookChannel.bind('message.phone.sent', () => {
      if (!this.loadingMessages) {
        this.loadMessages(false)
      }
    })
    this.webhookChannel.bind('message.send.failed', () => {
      if (!this.loadingMessages) {
        this.loadMessages(false)
      }
    })
  },

  beforeDestroy() {
    if (this.webhookChannel) {
      this.webhookChannel.unsubscribe()
    }
  },

  methods: {
    isPending(message: Message): boolean {
      return ['sending', 'pending', 'scheduled'].includes(message.status)
    },

    statusColor(message: Message): string {
      if (message.status === 'sending') {
        return 'warning'
      }

      if (message.status === 'scheduled') {
        return 'teal'
      }

      return 'primary'
    },

    canResend(message: Message): boolean {
      return (
        this.isMT(message) &&
        (message.status === 'expired' || message.status === 'failed')
      )
    },

    loadMessages(hideMessages = true) {
      this.loadingMessages = true
      this.$store
        .dispatch('loadThreadMessages', this.$route.params.id)
        .then((messages: Array<Message>) => {
          this.messages = [...messages].reverse()
        })
        .finally(() => {
          setTimeout(() => {
            this.loadingMessages = false
          }, 1100)
        })
      this.hideMessages = hideMessages
      setTimeout(() => {
        this.scrollToElement()
      }, 950)
    },

    async loadData() {
      await this.$store.dispatch('loadUser')
      await this.$store.dispatch('loadPhones')
      await this.$store.dispatch('loadThreads')

      if (!this.$store.getters.hasThreadId(this.$route.params.id)) {
        await this.$router.push({ name: 'threads' })
      }
      this.loadMessages()
    },

    isMT(message: Message): boolean {
      return message.type === 'mobile-terminated'
    },

    isMo(message: Message): boolean {
      return message.type === 'mobile-originated'
    },

    isMissedCall(message: Message): boolean {
      return message.type === 'call/missed'
    },

    scrollToElement() {
      const el: Element = this.$refs.messageBody as Element
      if (el) {
        el.scrollTop = el.scrollHeight + 120
      }
      this.hideMessages = false
    },

    archiveThread() {
      this.$store.dispatch('updateThread', {
        threadId: this.$store.getters.getThread.id,
        isArchived: true,
      })
      setTimeout(() => {
        this.selectedMenuItem = -1
      }, 1000)
    },

    unArchiveThread() {
      this.$store.dispatch('updateThread', {
        threadId: this.$store.getters.getThread.id,
        isArchived: false,
      })
      setTimeout(() => {
        this.selectedMenuItem = -1
      }, 1000)
    },

    async resendMessage(message: Message) {
      await this.$store.dispatch('sendMessage', {
        from: message.owner,
        to: message.contact,
        content: message.content,
      })

      setTimeout(() => {
        this.selectedMenuItem = -1
      }, 1000)

      this.loadMessages(false)
    },

    async deleteMessage(message: Message) {
      await this.$store.dispatch('deleteMessage', message.id)

      setTimeout(() => {
        this.selectedMenuItem = -1
      }, 1000)

      this.loadMessages(false)
    },

    async copyMessageId(message: Message) {
      await navigator.clipboard.writeText(message.id).then(() => {
        this.$store.dispatch('addNotification', {
          message: 'Message ID copied to clipboard',
          type: 'success',
        } as NotificationRequest)
      })

      setTimeout(() => {
        this.selectedMenuItem = -1
      }, 1000)
    },

    async deleteThread(threadID: string) {
      await this.$store.dispatch('deleteThread', threadID)
      await this.$router.push({ name: 'threads' })
    },

    async sendMessage(event: KeyboardEvent) {
      if (event.shiftKey) {
        return
      }

      if (!(this.$refs.form as any).validate()) {
        return
      }

      this.submitting = true

      const request: SendMessageRequest = {
        from: this.$store.getters.getOwner,
        to: this.$store.getters.getThread.contact,
        content: this.formMessage,
        sim: this.simSelected.code as SIM,
      }

      await this.$store.dispatch('sendMessage', request)
      this.loadMessages(false)

      this.formMessage = ''
      this.submitting = false
    },
  },
})
</script>

<style lang="scss">
.messages-body {
  padding-top: 50px;
  max-height: calc(100vh - 200px);
  position: absolute;
  width: 100%;
  bottom: 120px;
}

@media (min-width: 960px) {
  .messages-body {
    max-width: 900px;
  }
}
@media (min-width: 1264px) {
  .messages-body {
    max-width: 1185px;
  }
}
@media (min-width: 1904px) {
  .messages-body {
    max-width: 1785px;
  }
}

.no-scrollbar,
.no-scrollbar textarea {
  overflow-x: hidden;
  -ms-overflow-style: none; /* for Internet Explorer, Edge */
  overflow-y: scroll;
  &::-webkit-scrollbar {
    display: none; /* for Chrome, Safari, and Opera */
  }
}
</style>
