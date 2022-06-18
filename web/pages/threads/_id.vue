<template>
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
      <v-tooltip left>
        <template #activator="{ on, attrs }">
          <v-btn icon text v-bind="attrs" v-on="on">
            <v-icon>mdi-dots-vertical</v-icon>
          </v-btn>
        </template>
        <span>Message Options</span>
      </v-tooltip>
    </v-app-bar>
    <v-container>
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
      <v-form
        ref="form"
        class="fixed-bottom d-flex"
        lazy-validation
        @submit.prevent="sendMessage"
      >
        <v-textarea
          v-model="formMessage"
          :disabled="submitting"
          :rows="2"
          filled
          class="no-scrollbar"
          :rules="formMessageRules"
          placeholder="Type your message here"
          rounded
          @keydown.enter="sendMessage"
        ></v-textarea>
        <v-btn
          :disabled="submitting"
          type="submit"
          color="primary"
          class="pa-5 white--text ml-2 mt-1"
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
  </div>
</template>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'
import { InputValidationRules } from 'vuetify'
import { Message } from '~/models/message'
import { SendMessageRequest } from '~/store'

@Component
export default class ThreadsIndex extends Vue {
  formMessage = ''
  formMessageRules: InputValidationRules = [
    (v) => !!v || 'Message is required',
    (v) => (v && v.length <= 320) || 'Message must be less than 320 characters',
  ]

  submitting = false

  async mounted(): Promise<void> {
    await this.$store.dispatch('loadThreads')
    await this.$store.dispatch('loadThreadMessages', this.$route.params.id)
    this.scrollToElement()
  }

  get messages(): Array<Message> {
    return [...this.$store.getters.getThreadMessages].reverse()
  }

  isPending(message: Message): boolean {
    return ['sending', 'pending'].includes(message.status)
  }

  isMT(message: Message): boolean {
    return message.type === 'mobile-terminated'
  }

  scrollToElement() {
    const el: Element = this.$refs.messageBody as Element
    el.scrollTop = el.scrollHeight + 120
  }

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
    ;(this.$refs.form as any).reset()

    this.$nextTick(this.scrollToElement)
  }
}
</script>

<style lang="scss">
.fixed-bottom {
  width: 96%;
  max-width: 1761px;
  position: fixed;
  bottom: 0;
}

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
