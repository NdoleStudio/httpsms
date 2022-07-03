<template>
  <v-container fluid class="pa-0" :fill-height="$vuetify.breakpoint.lgAndUp">
    <div class="w-full h-full">
      <v-app-bar :dense="$vuetify.breakpoint.mdAndDown">
        <v-btn v-if="$vuetify.breakpoint.mdAndDown" icon to="/">
          <v-icon>mdi-arrow-left</v-icon>
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
              <v-icon>mdi-dots-vertical</v-icon>
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
                  <v-icon dense>mdi-package-down</v-icon>
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
                  <v-icon dense>mdi-package-up</v-icon>
                </v-list-item-icon>
                <v-list-item-content class="ml-n3">
                  <v-list-item-title class="pr-16 py-1">
                    <span :class="{ 'pr-16': $vuetify.breakpoint.mdAndUp }">
                      Unarchive
                    </span>
                  </v-list-item-title>
                </v-list-item-content>
              </v-list-item>
            </v-list-item-group>
          </v-list>
        </v-menu>
      </v-app-bar>
      <v-progress-linear
        v-if="$store.getters.getLoadingMessages"
        color="primary"
        indeterminate
      ></v-progress-linear>
      <v-container v-if="$store.getters.hasThread">
        <div ref="messageBody" class="messages-body no-scrollbar">
          <v-row v-for="message in messages" :key="message.id">
            <v-col
              class="d-flex"
              :class="{
                'pr-12': $vuetify.breakpoint.mdAndDown && !isMT(message),
                'pl-12 pr-5': $vuetify.breakpoint.mdAndDown && isMT(message),
              }"
            >
              <v-spacer v-if="isMT(message)"></v-spacer>
              <v-avatar
                v-if="!isMT(message)"
                :color="$store.getters.getThread.color"
              >
                <v-icon> mdi-account</v-icon>
              </v-avatar>
              <div>
                <v-card
                  class="ml-2"
                  shaped
                  :color="isMT(message) ? 'primary' : 'default'"
                >
                  <v-card-text
                    class="text--primary"
                    style="white-space: pre-line"
                    >{{ message.content }}</v-card-text
                  >
                </v-card>
                <div class="d-flex">
                  <p class="ml-2 text--secondary caption mr-2">
                    {{ new Date(message.order_timestamp).toLocaleString() }}
                  </p>
                  <v-spacer></v-spacer>
                  <v-tooltip bottom>
                    <template #activator="{ on, attrs }">
                      <div v-bind="attrs" v-on="on">
                        <v-progress-circular
                          v-if="isPending(message)"
                          indeterminate
                          :size="14"
                          :width="1"
                          class="mt-n2"
                          color="primary"
                        ></v-progress-circular>
                        <v-icon
                          v-if="message.status === 'delivered'"
                          color="primary"
                          class="mt-n6"
                          >mdi-check-all</v-icon
                        >
                        <v-icon v-if="message.status === 'sent'" class="mt-n6">
                          mdi-check
                        </v-icon>
                      </div>
                    </template>
                    <span>{{ message.status }}</span>
                  </v-tooltip>
                </div>
              </div>
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
                :disabled="submitting"
                :rows="1"
                filled
                class="no-scrollbar"
                :rules="formMessageRules"
                placeholder="Type your message here"
                rounded
                @keydown.enter="sendMessage"
              ></v-text-field>
              <v-btn
                :disabled="submitting"
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
                <v-icon>mdi-send</v-icon>
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
import { Message } from '~/models/message'
import { SendMessageRequest } from '~/store'

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
      formMessage: '',
      formMessageRules,
      submitting: false,
      selectedMenuItem: -1,
    }
  },

  computed: {
    messages(): Array<Message> {
      return [...this.$store.getters.getThreadMessages].reverse()
    },
  },

  async mounted() {
    await this.$store.dispatch('loadPhones')
    await this.$store.dispatch('loadThreads')

    if (!this.$store.getters.hasThreadId(this.$route.params.id)) {
      await this.$router.push({ name: 'threads' })
      return
    }

    await this.$store.dispatch('loadThreadMessages', this.$route.params.id)
    this.scrollToElement()
  },

  methods: {
    isPending(message: Message): boolean {
      return ['sending', 'pending'].includes(message.status)
    },

    isMT(message: Message): boolean {
      return message.type === 'mobile-terminated'
    },

    scrollToElement() {
      const el: Element = this.$refs.messageBody as Element
      el.scrollTop = el.scrollHeight + 120
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
      }

      await this.$store.dispatch('sendMessage', request)

      this.formMessage = ''
      this.submitting = false

      this.$nextTick(() => {
        ;(this.$refs.messageInput as any).$refs.input.focus()
        this.scrollToElement()
        ;(this.$refs.form as any).reset()
      })
    },
  },
})
</script>

<style lang="scss">
.messages-body {
  padding-top: 50px;
  width: 96%;
  max-height: calc(100vh - 200px);
  max-width: 1761px;
  position: absolute;
  bottom: 120px;
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
