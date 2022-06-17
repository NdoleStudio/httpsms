<template>
  <div class="w-full h-full">
    <v-app-bar :dense="$vuetify.breakpoint.mdAndDown">
      <v-btn v-if="$vuetify.breakpoint.mdAndDown" icon to="/">
        <v-icon>mdi-arrow-left</v-icon>
      </v-btn>
      <v-toolbar-title>
        <span v-if="$store.getters.hasThread">
          {{ $store.getters.getThread.contact }}
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
      <div ref="messageBody" class="messages-body">
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
                <v-card-text>{{ message.content }}</v-card-text>
              </v-card>
              <div class="d-flex">
                <p class="ml-2 text--secondary caption">
                  {{ new Date(message.order_timestamp).toLocaleString() }}
                </p>
                <v-spacer></v-spacer>
                <v-tooltip bottom>
                  <template #activator="{ on, attrs }">
                    <div v-bind="attrs" v-on="on">
                      <v-progress-circular
                        v-if="isPending(message)"
                        indeterminate
                        :size="16"
                        :width="1"
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
      <div class="d-flex fixed-bottom">
        <v-textarea
          :rows="2"
          placeholder="Type your message here"
          filled
          rounded
        ></v-textarea>
        <v-btn color="primary" class="pa-5 white--text ml-2 mt-1" fab>
          <v-icon>mdi-send</v-icon>
        </v-btn>
      </div>
    </v-container>
  </div>
</template>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'
import { Message } from '~/models/message'

@Component
export default class ThreadsIndex extends Vue {
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
  overflow-x: hidden;
  -ms-overflow-style: none; /* for Internet Explorer, Edge */
  overflow-y: scroll;
  position: absolute;
  bottom: 120px;
  &::-webkit-scrollbar {
    display: none; /* for Chrome, Safari, and Opera */
  }
}
</style>
