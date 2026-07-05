<script setup lang="ts">
import { useDisplay } from 'vuetify'
import {
  mdiGithub,
  mdiCircle,
  mdiTwitter,
  mdiHeart,
  mdiShieldStar,
  mdiLightbulbOn50,
  mdiCreation,
  mdiEyeOffOutline,
  mdiPost,
  mdiCreditCardOutline,
  mdiScaleBalance,
  mdiEmailOutline,
  mdiBookOpenVariant,
} from '@mdi/js'

const router = useRouter()
const route = useRoute()
const { lgAndUp, mdAndUp } = useDisplay()
const authStore = useAuthStore()
const appStore = useAppStore()

function goToPricing() {
  if (route.name === 'index') {
    document.getElementById('pricing')?.scrollIntoView({ behavior: 'smooth' })
  } else {
    router.push('/').then(() => {
      setTimeout(() => {
        document
          .getElementById('pricing')
          ?.scrollIntoView({ behavior: 'smooth' })
      }, 300)
    })
  }
}
</script>

<template>
  <v-app>
    <v-app-bar color="#121212" elevation="0">
      <v-container>
        <v-row>
          <v-col class="w-full d-flex">
            <NuxtLink to="/" class="text-decoration-none d-flex align-baseline">
              <v-avatar
                color="#121212"
                class="pa-1"
                rounded="0"
                :image="'/img/logo.svg'"
                :size="38"
              />
              <h3
                v-if="lgAndUp"
                class="text-headline-large mb-0 ml-1 mt-6 text-white"
              >
                httpSMS
              </h3>
            </NuxtLink>
            <v-spacer />
            <v-btn
              v-show="lgAndUp"
              size="large"
              variant="text"
              color="primary"
              class="my-5 mr-2"
              @click="goToPricing"
            >
              Pricing
            </v-btn>
            <v-btn
              v-show="lgAndUp"
              size="large"
              variant="text"
              color="primary"
              class="my-5 mr-2"
              :to="{ name: 'blog' }"
            >
              Blog
            </v-btn>
            <v-btn
              v-show="lgAndUp && authStore.authUser === null"
              size="large"
              variant="text"
              color="primary"
              class="my-5 mr-2"
              :to="{ name: 'login' }"
            >
              Login
            </v-btn>
            <v-btn
              v-show="authStore.authUser === null"
              color="primary"
              variant="flat"
              :class="{ 'mt-5': mdAndUp, 'mt-1': !mdAndUp }"
              :size="lgAndUp ? 'large' : 'default'"
              :to="{ name: 'login' }"
            >
              Get Started
              <span v-show="lgAndUp">&nbsp;For Free</span>
            </v-btn>
            <v-btn
              v-show="authStore.authUser !== null"
              color="primary"
              variant="flat"
              :class="{ 'mt-5': mdAndUp, 'mt-1': !mdAndUp }"
              :size="lgAndUp ? 'large' : 'default'"
              :to="{ name: 'threads' }"
            >
              Dashboard
            </v-btn>
          </v-col>
        </v-row>
      </v-container>
    </v-app-bar>
    <v-main>
      <AppToast />
      <slot />
    </v-main>
    <v-footer>
      <v-container>
        <v-row>
          <v-col cols="12" md="3">
            <NuxtLink to="/" class="text-decoration-none d-flex mt-n6">
              <v-avatar
                color="#212121"
                class="mt-8 pa-1"
                rounded="0"
                :image="'/img/logo.svg'"
                :size="38"
              />
              <h3 class="text-headline-large ml-1 mb-0 text-white">httpSMS</h3>
            </NuxtLink>
            <div class="text-title-medium mb-4 text-medium-emphasis">
              Made With
              <v-icon color="#cf1112" :icon="mdiHeart" /> in Tallinn
              <v-img
                class="d-inline-block"
                width="20"
                src="https://upload.wikimedia.org/wikipedia/commons/8/8f/Flag_of_Estonia.svg"
              />
            </div>
            <p class="mt-n3">
              <v-btn
                href="https://twitter.com/httpsmsHQ"
                color="#1DA1F2"
                class="ml-n3"
                variant="text"
                :icon="mdiTwitter"
              />
              <v-btn
                :href="appStore.appData.githubUrl"
                color="#ffffff"
                variant="text"
                :icon="mdiGithub"
              />
              <v-btn
                href="https://discord.gg/kGk8HVqeEZ"
                icon
                variant="text"
                color="#5865f2"
              >
                <v-img
                  contain
                  height="24"
                  width="24"
                  src="/img/discord-logo-blue.svg"
                />
              </v-btn>
            </p>
            <a
              href="https://www.saashub.com/httpsms?utm_source=badge&utm_campaign=badge&utm_content=httpsms&badge_variant=color&badge_kind=approved"
              target="_blank"
            >
              <img
                src="https://cdn-b.saashub.com/img/badges/approved-color.png?v=1"
                alt="httpSMS badge"
                style="max-width: 150px"
              />
            </a>
          </v-col>
          <v-col cols="12" md="3">
            <h2 class="text-headline-small mb-2">Resources</h2>
            <ul style="list-style: none" class="pa-0">
              <li class="mb-2">
                <a
                  class="text-white text-decoration-none footer-link"
                  style="cursor: pointer"
                  @click.stop="goToPricing"
                >
                  Pricing
                  <v-icon size="small" :icon="mdiCreditCardOutline" />
                </a>
              </li>
              <li class="mb-2">
                <NuxtLink
                  to="/affiliates"
                  class="text-white text-decoration-none footer-link"
                >
                  Affiliates
                  <v-icon color="warning" size="small" :icon="mdiShieldStar" />
                </NuxtLink>
              </li>
              <li class="mb-2">
                <a
                  href="https://status.httpsms.com"
                  class="text-white text-decoration-none footer-link"
                >
                  Site status
                  <v-icon color="success" size="x-small" :icon="mdiCircle" />
                </a>
              </li>
              <li class="mb-2">
                <NuxtLink
                  class="text-white text-decoration-none footer-link"
                  to="/blog"
                >
                  Blog <v-icon size="small" :icon="mdiPost" />
                </NuxtLink>
              </li>
            </ul>
          </v-col>
          <v-col cols="12" md="3">
            <h2 class="text-headline-small mb-2">Developers</h2>
            <ul style="list-style: none" class="pa-0">
              <li class="mb-2">
                <a
                  :href="appStore.appData.documentationUrl"
                  class="text-white text-decoration-none footer-link"
                >
                  Documentation
                  <v-icon size="small" :icon="mdiBookOpenVariant" />
                </a>
              </li>
              <li class="mb-2">
                <a
                  :href="appStore.appData.githubUrl"
                  class="text-white text-decoration-none footer-link"
                >
                  Github <v-icon size="small" :icon="mdiGithub" />
                </a>
              </li>
              <li class="mb-2">
                <a
                  href="https://sandbox.httpsms.com"
                  class="text-white text-decoration-none footer-link"
                >
                  Sandbox
                  <v-icon size="small" color="pink" :icon="mdiCreation" />
                </a>
              </li>
              <li class="mb-2">
                <a
                  href="https://httpsms.featurebase.app"
                  class="text-white text-decoration-none footer-link"
                >
                  Request Feature
                  <v-icon
                    size="small"
                    color="yellow"
                    :icon="mdiLightbulbOn50"
                  />
                </a>
              </li>
            </ul>
          </v-col>
          <v-col cols="12" md="3">
            <h2 class="text-headline-small mb-2">Legal</h2>
            <ul style="list-style: none" class="pa-0">
              <li class="mb-2">
                <NuxtLink
                  class="text-white text-decoration-none footer-link"
                  to="/terms-and-conditions"
                >
                  Terms & Conditions
                  <v-icon size="small" :icon="mdiScaleBalance" />
                </NuxtLink>
              </li>
              <li class="mb-2">
                <NuxtLink
                  class="text-white text-decoration-none footer-link"
                  to="/privacy-policy"
                >
                  Privacy Policy
                  <v-icon size="small" :icon="mdiEyeOffOutline" />
                </NuxtLink>
              </li>
              <li class="mt-2">
                <a
                  class="text-white text-decoration-none footer-link"
                  href="mailto:support@httpsms.com"
                >
                  Contact Support
                  <v-icon size="small" :icon="mdiEmailOutline" />
                </a>
              </li>
            </ul>
          </v-col>
        </v-row>
      </v-container>
    </v-footer>
  </v-app>
</template>

<style scoped>
.footer-link:hover {
  text-decoration: underline !important;
}
</style>
