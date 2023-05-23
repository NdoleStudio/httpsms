<template>
  <v-app dark>
    <v-app-bar elevate-on-scroll color="#121212" height="70" fixed>
      <v-container>
        <v-row>
          <v-col class="w-full d-flex">
            <v-badge class="logo-badge mt-5" color="primary" content="Beta">
              <nuxt-link to="/" class="text-decoration-none d-flex">
                <v-avatar
                  v-if="$vuetify.breakpoint.lgAndUp"
                  tile
                  size="33"
                  class="mt-1"
                >
                  <v-img
                    contain
                    :src="require('@/assets/img/logo.svg')"
                  ></v-img>
                </v-avatar>
                <h3 class="text-h4 ml-1 text--primary">httpSMS</h3>
              </nuxt-link>
            </v-badge>
            <v-spacer></v-spacer>
            <v-btn
              v-if="$vuetify.breakpoint.lgAndUp"
              large
              text
              color="primary"
              class="my-5 mr-2"
              @click="goToPricing"
            >
              Pricing
            </v-btn>
            <v-btn
              v-if="$vuetify.breakpoint.lgAndUp"
              large
              text
              color="primary"
              class="my-5 mr-2"
              :to="{ name: 'blog' }"
            >
              Blog
            </v-btn>
            <v-btn
              v-if="$vuetify.breakpoint.lgAndUp"
              large
              text
              color="primary"
              class="my-5 mr-2"
              :to="{ name: 'login' }"
            >
              Login
            </v-btn>
            <v-btn
              exact-path
              color="primary"
              class="my-5"
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
    <v-main>
      <Nuxt />
    </v-main>
    <v-footer class="pt-4">
      <v-container>
        <v-row>
          <v-col cols="12" md="3">
            <v-badge
              class="logo-badge"
              :class="{ 'logo-badge--mobile': $vuetify.breakpoint.mdAndDown }"
              color="primary"
              content="Beta"
            >
              <nuxt-link to="/" class="text-decoration-none d-flex">
                <v-avatar tile size="33" class="mt-1">
                  <v-img
                    contain
                    :src="require('@/assets/img/logo.svg')"
                  ></v-img>
                </v-avatar>
                <h3 class="text-h4 ml-1 text--primary">httpSMS</h3>
              </nuxt-link>
            </v-badge>
            <p class="subtitle-2 text--secondary">
              Made With <v-icon color="#cf1112">{{ mdiHeart }}</v-icon> in
              Tallinn
              <v-img
                class="d-inline-block"
                width="20"
                src="https://upload.wikimedia.org/wikipedia/commons/8/8f/Flag_of_Estonia.svg"
              ></v-img>
            </p>
            <p class="mt-n3">
              <v-btn
                href="https://twitter.com/acho_arnold"
                icon
                color="#1DA1F2"
              >
                <v-icon>{{ mdiTwitter }}</v-icon>
              </v-btn>
              <v-btn
                :href="$store.getters.getAppData.githubUrl"
                icon
                large
                color="#ffffff"
              >
                <v-icon>{{ mdiGithub }}</v-icon>
              </v-btn>
              <v-btn
                href="https://discord.gg/6qdbY6XN"
                icon
                large
                color="#5865f2"
              >
                <v-img
                  contain
                  height="24"
                  width="24"
                  :src="require('assets/img/discord-logo-blue.svg')"
                ></v-img>
              </v-btn>
            </p>
          </v-col>
          <v-col cols="12" md="3">
            <h2 class="text-h6 mb-2">Resources</h2>
            <ul style="list-style: none" class="pa-0">
              <li class="mb-2">
                <v-hover v-slot="{ hover }">
                  <a
                    class="text--primary text-decoration-none"
                    :class="{ 'text-decoration-underline': hover }"
                    @click.stop="goToPricing"
                  >
                    Pricing
                    <v-icon small>{{ mdiCreditCardOutline }}</v-icon>
                  </a>
                </v-hover>
              </li>
              <li class="mb-2">
                <v-hover v-slot="{ hover }">
                  <a
                    href="https://status.httpsms.com"
                    class="text--primary text-decoration-none"
                    :class="{ 'text-decoration-underline': hover }"
                  >
                    Site status
                    <v-icon color="success" x-small>{{ mdiCircle }}</v-icon>
                  </a>
                </v-hover>
              </li>
              <li class="mb-2">
                <v-hover v-slot="{ hover }">
                  <nuxt-link
                    class="text--primary text-decoration-none"
                    :class="{ 'text-decoration-underline': hover }"
                    to="/blog"
                  >
                    Blog
                    <v-icon small>{{ mdiPost }}</v-icon>
                  </nuxt-link>
                </v-hover>
              </li>
            </ul>
          </v-col>
          <v-col cols="12" md="3">
            <h2 class="text-h6 mb-2">Developers</h2>
            <ul style="list-style: none" class="pa-0">
              <li class="mb-2">
                <v-hover v-slot="{ hover }">
                  <a
                    :href="$store.getters.getAppData.documentationUrl"
                    class="text--primary text-decoration-none"
                    :class="{ 'text-decoration-underline': hover }"
                  >
                    Documentation
                    <v-icon small>{{ mdiBookOpenVariant }}</v-icon>
                  </a>
                </v-hover>
              </li>
              <li class="mb-2">
                <v-hover v-slot="{ hover }">
                  <a
                    :href="$store.getters.getAppData.githubUrl"
                    class="text--primary text-decoration-none"
                    :class="{ 'text-decoration-underline': hover }"
                  >
                    Github
                    <v-icon small>{{ mdiGithub }}</v-icon>
                  </a>
                </v-hover>
              </li>
            </ul>
          </v-col>
          <v-col cols="12" md="3">
            <h2 class="text-h6 mb-2">Legal</h2>
            <ul style="list-style: none" class="pa-0">
              <li class="mb-2">
                <v-hover v-slot="{ hover }">
                  <nuxt-link
                    class="text--primary text-decoration-none"
                    :class="{ 'text-decoration-underline': hover }"
                    to="/terms-and-conditions"
                  >
                    Terms & Conditions
                    <v-icon small>{{ mdiScaleBalance }}</v-icon>
                  </nuxt-link>
                </v-hover>
              </li>
              <li>
                <v-hover v-slot="{ hover }">
                  <nuxt-link
                    class="text--primary text-decoration-none"
                    :class="{ 'text-decoration-underline': hover }"
                    to="/privacy-policy"
                  >
                    Privacy Policy
                    <v-icon small>{{ mdiEyeOffOutline }}</v-icon>
                  </nuxt-link>
                </v-hover>
              </li>
              <li class="mt-2">
                <v-hover v-slot="{ hover }">
                  <a
                    class="text--primary text-decoration-none"
                    :class="{ 'text-decoration-underline': hover }"
                    href="mailto:support@httpsms.com"
                  >
                    Contact Support
                    <v-icon small>{{ mdiEmailOutline }}</v-icon>
                  </a>
                </v-hover>
              </li>
            </ul>
          </v-col>
        </v-row>
      </v-container>
    </v-footer>
  </v-app>
</template>

<script lang="ts">
import Vue from 'vue'
import {
  mdiLoginVariant,
  mdiArrowRight,
  mdiGithub,
  mdiCircle,
  mdiTwitter,
  mdiHeart,
  mdiEyeOffOutline,
  mdiPost,
  mdiCreditCardOutline,
  mdiScaleBalance,
  mdiEmailOutline,
  mdiBookOpenVariant,
} from '@mdi/js'

export default Vue.extend({
  name: 'WebsiteLayout',
  data() {
    return {
      mdiLoginVariant,
      mdiArrowRight,
      mdiGithub,
      mdiTwitter,
      mdiHeart,
      mdiScaleBalance,
      mdiEyeOffOutline,
      mdiCreditCardOutline,
      mdiEmailOutline,
      mdiPost,
      mdiCircle,
      mdiBookOpenVariant,
    }
  },
  methods: {
    goToPricing() {
      if (this.$route.name === 'index') {
        this.$vuetify.goTo('#pricing')
      } else {
        this.$router.push('/#pricing')
      }
    },
  },
})
</script>

<style lang="scss">
.v-application {
  .logo-badge {
    .v-badge__wrapper {
      span {
        margin-bottom: -8px;
      }
    }
  }
}
</style>
