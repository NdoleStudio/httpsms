<template>
  <div>
    <v-app-bar elevation="0" color="#121212" height="100px">
      <v-container>
        <v-row>
          <v-col class="w-full d-flex">
            <h2
              class="text--primary"
              :class="{
                'text-h4': $vuetify.breakpoint.lgAndUp,
                'text-h5': $vuetify.breakpoint.mdAndDown,
              }"
            >
              <v-avatar
                v-if="$vuetify.breakpoint.lgAndUp"
                size="70"
                class="mt-4"
              >
                <v-img
                  class="mb-4"
                  contain
                  :src="require('assets/img/icon.svg')"
                ></v-img>
              </v-avatar>
              Http SMS
            </h2>
            <v-spacer></v-spacer>
            <v-btn
              exact-path
              class="primary mt-5"
              :large="$vuetify.breakpoint.lgAndUp"
              :to="{ name: 'login' }"
            >
              Get Started
              <span v-if="$vuetify.breakpoint.lgAndUp">&nbsp;For Free</span>
            </v-btn>
          </v-col>
        </v-row>
      </v-container>
    </v-app-bar>
    <v-container class="pb-16">
      <v-row :class="{ 'py-16': $vuetify.breakpoint.lgAndUp }">
        <v-col
          cols="12"
          md="4"
          :class="{
            'text-center': $vuetify.breakpoint.mdAndDown,
            'py-16': $vuetify.breakpoint.lgAndUp,
          }"
        >
          <h1 class="text-h2" :class="{ 'mt-16': $vuetify.breakpoint.lgAndUp }">
            Convert your android phone into an SMS gateway.
          </h1>
          <h2 class="text--secondary text-h5 mt-8 mb-8">
            Use your android phone to send and receive SMS messages using an
            intuitive HTTP API.
          </h2>
          <div :class="{ 'text-center': $vuetify.breakpoint.mdAndDown }">
            <v-btn
              color="primary"
              large
              exact-path
              class="mt-4 mb-4"
              :to="{ name: 'login' }"
            >
              <v-icon v-if="$vuetify.breakpoint.lgAndUp">{{ mdiSend }}</v-icon>
              Get Started
            </v-btn>
            <v-btn
              large
              color="default"
              target="_blank"
              class="mt-4 mb-4 ml-4"
              :href="$store.getters.getAppData.githubUrl"
            >
              <v-icon v-if="$vuetify.breakpoint.lgAndUp">
                {{ mdiGithub }}
              </v-icon>
              Open Source
            </v-btn>
          </div>
          <div
            class="mt-4"
            :class="{ 'text-center': $vuetify.breakpoint.mdAndDown }"
          >
            <v-icon color="success">
              {{ mdiCheckCircle }}
            </v-icon>
            Free to use
            <v-icon class="ml-4" color="success">
              {{ mdiCheckCircle }}
            </v-icon>
            100% Open Source
          </div>
          <v-divider
            v-if="$vuetify.breakpoint.mdAndDown"
            class="mt-6 mr-16 success"
            :class="{ 'ml-16': $vuetify.breakpoint.mdAndDown }"
          ></v-divider>
        </v-col>
        <v-col cols="12" md="8" class="py-16">
          <v-img
            class="mx-auto mb-4"
            max-height="700"
            max-width="98%"
            contain
            :src="require('assets/img/writing-code-phone.svg')"
          ></v-img>
        </v-col>
      </v-row>
    </v-container>
    <v-sheet class="py-16">
      <v-container>
        <v-row class="mb-16">
          <v-col cols="12" md="6" class="d-flex align-center" :order-lg="2">
            <div>
              <h3 class="text-h3 mb-1">Control Sending</h3>
              <h5 class="text-h6 font-weight-light">
                Send bulk messages without mobile carrier limitations. If you
                set a rate e.g 3 messages per minute, we will queue up your
                messages and send them at a rate of 1 message per 20 seconds.
              </h5>
            </div>
          </v-col>
          <v-col cols="12" md="6" :order-lg="1">
            <v-img
              class="mb-4"
              max-height="300"
              contain
              :src="require('assets/img/queue.svg')"
            ></v-img>
          </v-col>
        </v-row>
        <v-row class="mb-16">
          <v-col cols="12" md="6" class="d-flex align-center">
            <div>
              <h3 class="text-h3 mb-1">Monitoring</h3>
              <h5 class="text-h6 font-weight-light">
                If your android phone goes offline for some reason and it can't
                send SMS messages, we will send you a notification immediately.
              </h5>
            </div>
          </v-col>
          <v-col cols="12" md="6">
            <v-img
              class="mb-4"
              max-height="300"
              contain
              :src="require('assets/img/alert.svg')"
            ></v-img>
          </v-col>
        </v-row>
        <v-row class="mt-16">
          <v-col cols="12" md="6" class="d-flex align-center" :order-lg="2">
            <div>
              <h3 class="text-h3 mb-1">Open Source</h3>
              <h5 class="text-h6 font-weight-light">
                HttpSms is transparent and fully open source. The source code is
                available
                <a
                  target="_blank"
                  class="text-decoration-none"
                  :href="$store.getters.getAppData.githubUrl"
                  >on GitHub under the MIT License</a
                >. Feel free to fork it, verify it or submit a pull request.
              </h5>
            </div>
          </v-col>
          <v-col cols="12" md="6" :order-lg="1">
            <v-img
              class="mb-4"
              max-height="300"
              contain
              :src="require('assets/img/open-source.svg')"
            ></v-img>
          </v-col>
        </v-row>
      </v-container>
    </v-sheet>
    <v-container class="pb-16">
      <v-row class="mt-16">
        <v-col>
          <h2 class="text-h3 text-center">Get Started</h2>
        </v-col>
      </v-row>
      <v-row class="pb-16">
        <v-col cols="12">
          <v-row>
            <v-col cols="12" md="5" class="pr-4">
              <v-timeline dense class="mt-10 ml-n4">
                <v-timeline-item color="primary" :icon="mdiTallyMark1">
                  <v-card
                    class="elevation-2"
                    @mouseover="step1Hover = true"
                    @mouseleave="step1Hover = false"
                  >
                    <v-card-title class="text-h5"> Step 1 </v-card-title>
                    <v-card-text class="subtitle-1">
                      <router-link
                        class="font-weight-bold text-decoration-none"
                        :to="{ name: 'login' }"
                      >
                        Create an account
                      </router-link>
                      on httpsms.com and obtain you API key on the settings
                      page.
                    </v-card-text>
                  </v-card>
                </v-timeline-item>
                <v-timeline-item color="primary" :icon="mdiTallyMark2">
                  <v-card
                    class="elevation-2"
                    @mouseover="step2Hover = true"
                    @mouseleave="step2Hover = false"
                  >
                    <v-card-title class="text-h5"> Step 2 </v-card-title>
                    <v-card-text class="subtitle-1">
                      <a
                        download
                        class="font-weight-bold text-decoration-none"
                        :href="$store.getters.getAppData.appDownloadUrl"
                        >Download</a
                      >
                      and install the companion android application on your
                      phone and sign in using your API Key.
                    </v-card-text>
                  </v-card>
                </v-timeline-item>
                <v-timeline-item color="primary" :icon="mdiTallyMark3">
                  <v-card class="elevation-2">
                    <v-card-title class="text-h5"> Step 3 </v-card-title>
                    <v-card-text class="subtitle-1">
                      Start sending and receiving SMS messages using our rich
                      HTTP API. You can find the documentation on
                      <a
                        class="text-decoration-none"
                        :href="$store.getters.getAppData.documentationUrl"
                      >
                        {{ $store.getters.getAppData.documentationUrl }}
                      </a>
                    </v-card-text>
                  </v-card>
                </v-timeline-item>
              </v-timeline>
            </v-col>
            <v-col
              v-if="$vuetify.breakpoint.lgAndUp"
              cols="12"
              md="7"
              class="d-flex align-center"
              :class="{ 'min-h-650': $vuetify.breakpoint.lgAndUp }"
            >
              <transition name="fade">
                <v-img
                  v-if="step1Hover"
                  contain
                  max-height="600"
                  :src="require('assets/img/phone-api-key.png')"
                ></v-img>
              </transition>
              <transition name="fade">
                <v-img
                  v-if="step2Hover && !step1Hover"
                  contain
                  max-height="600"
                  :src="require('assets/img/phone-login.png')"
                ></v-img>
              </transition>
              <transition name="fade">
                <v-img
                  v-if="!step1Hover && !step2Hover"
                  contain
                  class="mt-n16"
                  :src="require('assets/img/code-snippet.png')"
                ></v-img>
              </transition>
            </v-col>
          </v-row>
        </v-col>
      </v-row>
    </v-container>
    <v-footer absolute>
      <v-container class="text-center">
        <v-row justify="center" no-gutters>
          <v-col cols="12">
            <v-btn
              text
              rounded
              class="my-2"
              :href="$store.getters.getAppData.documentationUrl"
              >Documentation</v-btn
            >
            <v-btn
              text
              rounded
              class="my-2"
              :href="$store.getters.getAppData.githubUrl"
              >Open Source</v-btn
            >
          </v-col>
          <v-col class="text-center mb-n4" cols="12">
            <p class="text--secondary text-subtitle-2">
              &copy; {{ new Date().getFullYear() }} -
              {{ $store.getters.getAppData.name }}
            </p>
          </v-col>
        </v-row>
      </v-container>
    </v-footer>
  </div>
</template>

<script lang="ts">
import Vue from 'vue'
import {
  mdiGithub,
  mdiCheckCircle,
  mdiSend,
  mdiForum,
  mdiNumeric1,
  mdiNumeric2,
  mdiNumeric3,
  mdiTallyMark1,
  mdiTallyMark3,
  mdiTallyMark2,
} from '@mdi/js'

export default Vue.extend({
  name: 'LandingPage',
  middleware: ['guest'],
  data() {
    return {
      mdiGithub,
      mdiCheckCircle,
      mdiSend,
      mdiForum,
      mdiNumeric1,
      mdiNumeric2,
      mdiNumeric3,
      mdiTallyMark1,
      mdiTallyMark2,
      mdiTallyMark3,
      step1Hover: false,
      step2Hover: false,
    }
  },
})
</script>

<style lang="scss">
.fade-enter-active {
  transition: opacity 0.5s;
}

.fade-enter,
.fade-leave-active {
  opacity: 0;
}

.min-h-650 {
  min-height: 650px;
}
</style>
