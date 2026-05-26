<script setup lang="ts">
import { useDisplay } from "vuetify";
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
} from "@mdi/js";

const router = useRouter();
const route = useRoute();
const { lgAndUp, mdAndUp } = useDisplay();
const authStore = useAuthStore();
const appStore = useAppStore();

function goToPricing() {
  if (route.name === "index") {
    document.getElementById("pricing")?.scrollIntoView({ behavior: "smooth" });
  } else {
    router.push("/#pricing");
  }
}
</script>

<template>
  <v-app>
    <v-app-bar elevation="2" color="#121212" height="70">
      <v-container>
        <v-row>
          <v-col class="w-full d-flex">
            <NuxtLink
              to="/"
              class="text-decoration-none d-flex"
              :class="{ 'mt-5': mdAndUp }"
            >
              <v-avatar :image="'/img/logo.svg'" :size="33" class="mt-1" />
              <h3
                v-if="lgAndUp"
                class="text-headline-large ml-1 text-on-surface"
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
      <Toast />
      <slot />
    </v-main>
    <v-footer class="pt-4">
      <v-container>
        <v-row>
          <v-col cols="12" md="3">
            <NuxtLink to="/" class="text-decoration-none d-flex">
              <v-avatar :image="'/img/logo.svg'" :size="33" class="mt-1" />
              <h3 class="text-headline-large ml-1 text-on-surface">httpSMS</h3>
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
                icon
                color="#1DA1F2"
                :icon="mdiTwitter"
              />
              <v-btn
                :href="appStore.appData.githubUrl"
                icon
                size="large"
                color="#ffffff"
                :icon="mdiGithub"
              />
              <v-btn
                href="https://discord.gg/kGk8HVqeEZ"
                icon
                size="large"
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
          </v-col>
          <v-col cols="12" md="3">
            <h2 class="text-headline-small mb-2">Resources</h2>
            <ul style="list-style: none" class="pa-0">
              <li class="mb-2">
                <a
                  class="text-on-surface text-decoration-none"
                  @click.stop="goToPricing"
                >
                  Pricing
                  <v-icon size="small" :icon="mdiCreditCardOutline" />
                </a>
              </li>
              <li class="mb-2">
                <a
                  href="https://httpsms.lemonsqueezy.com/affiliates"
                  class="text-on-surface text-decoration-none"
                >
                  Affiliates
                  <v-icon color="warning" size="small" :icon="mdiShieldStar" />
                </a>
              </li>
              <li class="mb-2">
                <a
                  href="https://status.httpsms.com"
                  class="text-on-surface text-decoration-none"
                >
                  Site status
                  <v-icon color="success" size="x-small" :icon="mdiCircle" />
                </a>
              </li>
              <li class="mb-2">
                <NuxtLink
                  class="text-on-surface text-decoration-none"
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
                  class="text-on-surface text-decoration-none"
                >
                  Documentation
                  <v-icon size="small" :icon="mdiBookOpenVariant" />
                </a>
              </li>
              <li class="mb-2">
                <a
                  :href="appStore.appData.githubUrl"
                  class="text-on-surface text-decoration-none"
                >
                  Github <v-icon size="small" :icon="mdiGithub" />
                </a>
              </li>
              <li class="mb-2">
                <a
                  href="https://sandbox.httpsms.com"
                  class="text-on-surface text-decoration-none"
                >
                  Sandbox
                  <v-icon size="small" color="pink" :icon="mdiCreation" />
                </a>
              </li>
              <li class="mb-2">
                <a
                  href="https://httpsms.featurebase.app"
                  class="text-on-surface text-decoration-none"
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
                  class="text-on-surface text-decoration-none"
                  to="/terms-and-conditions"
                >
                  Terms & Conditions
                  <v-icon size="small" :icon="mdiScaleBalance" />
                </NuxtLink>
              </li>
              <li class="mb-2">
                <NuxtLink
                  class="text-on-surface text-decoration-none"
                  to="/privacy-policy"
                >
                  Privacy Policy
                  <v-icon size="small" :icon="mdiEyeOffOutline" />
                </NuxtLink>
              </li>
              <li class="mt-2">
                <a
                  class="text-on-surface text-decoration-none"
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
