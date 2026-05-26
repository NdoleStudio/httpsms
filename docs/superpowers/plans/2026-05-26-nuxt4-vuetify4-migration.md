# Nuxt 4 + Vuetify 4 Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the httpSMS frontend from Nuxt 2 + Vue 2 + Vuetify 2 to Nuxt 4 + Vue 3 + Vuetify 4 with full TypeScript, Pinia state management, and `<script setup>` Composition API.

**Architecture:** Fresh Nuxt 4 project replacing the existing `web/` directory. All Vue 2 class-based components (`vue-property-decorator`) will be rewritten as `<script setup lang="ts">` using Composition API. Vuex store will be split into domain-specific Pinia stores. Firebase auth via `nuxt-vuefire`. The Vuetify MCP tools (`vuetify-mcp-get_component_api_by_version`, `vuetify-mcp-get_v4_breaking_changes`) MUST be called for every component/page migration to validate Vuetify 4 compatibility.

**Tech Stack:** Nuxt 4, Vue 3, Vuetify 4, Pinia, TypeScript, nuxt-vuefire, Firebase JS SDK, Pusher.js, Chart.js, libphonenumber-js

---

## File Structure

```
web/
├── app/
│   ├── app.vue
│   ├── assets/
│   │   ├── img/              (copy from old assets/img/)
│   │   └── styles/
│   │       └── settings.scss
│   ├── components/
│   │   ├── BackButton.vue
│   │   ├── BlogAuthorBio.vue
│   │   ├── BlogInfo.vue
│   │   ├── CopyButton.vue
│   │   ├── FirebaseAuth.vue
│   │   ├── FixedHeader.vue
│   │   ├── LoadingButton.vue
│   │   ├── LoadingDashboard.vue
│   │   ├── MessageThread.vue
│   │   ├── MessageThreadHeader.vue
│   │   ├── NuxtLogo.vue
│   │   └── Toast.vue
│   ├── composables/
│   │   ├── useApi.ts
│   │   └── useFilters.ts
│   ├── layouts/
│   │   ├── default.vue
│   │   ├── error.vue
│   │   └── website.vue
│   ├── middleware/
│   │   ├── auth.ts
│   │   └── guest.ts
│   ├── pages/
│   │   ├── index.vue
│   │   ├── login.vue
│   │   ├── billing/index.vue
│   │   ├── blog/
│   │   │   ├── index.vue
│   │   │   ├── how-to-send-sms-messages-from-excel.vue
│   │   │   ├── grant-send-and-read-sms-permissions-on-android.vue
│   │   │   ├── forward-incoming-sms-from-phone-to-webhook.vue
│   │   │   ├── end-to-end-encryption-to-sms-messages.vue
│   │   │   ├── send-bulk-sms-from-csv-file-with-no-code.vue
│   │   │   ├── send-sms-from-android-phone-with-python.vue
│   │   │   └── send-sms-when-new-row-is-added-to-google-sheets-using-zapier.vue
│   │   ├── bulk-messages/index.vue
│   │   ├── heartbeats/[id].vue
│   │   ├── messages/index.vue
│   │   ├── phone-api-keys/index.vue
│   │   ├── privacy-policy/index.vue
│   │   ├── search-messages/index.vue
│   │   ├── settings/index.vue
│   │   ├── terms-and-conditions/index.vue
│   │   └── threads/
│   │       ├── index.vue
│   │       └── [id]/index.vue
│   ├── plugins/
│   │   ├── vuetify.ts
│   │   ├── chart.client.ts
│   │   └── vue-glow.client.ts
│   ├── stores/
│   │   ├── app.ts
│   │   ├── auth.ts
│   │   ├── billing.ts
│   │   ├── messages.ts
│   │   ├── notifications.ts
│   │   ├── phones.ts
│   │   └── threads.ts
│   └── utils/
│       ├── bag.ts
│       ├── capitalize.ts
│       ├── errors.ts
│       └── filters.ts
├── shared/
│   └── types/
│       ├── api.ts
│       ├── billing.ts
│       ├── heartbeat.ts
│       ├── message-thread.ts
│       ├── message.ts
│       └── user.ts
├── public/                   (copy from old static/)
├── nuxt.config.ts
├── package.json
├── tsconfig.json
└── .env
```

---

## Task 1: Backup Old Code & Scaffold Nuxt 4 Project

**Files:**
- Delete (after backup): `web/` (entire old structure)
- Create: `web/package.json`, `web/nuxt.config.ts`, `web/tsconfig.json`, `web/app/app.vue`

- [ ] **Step 1: Create a backup branch of the current web/ code**

```bash
cd C:\Users\Arnold\Work\NdoleStudio\httpsms.com
git stash
git checkout main
git checkout -b backup/web-nuxt2-vuetify2
git checkout feat/migrate-nuxt4-vuetify4
git stash pop
```

- [ ] **Step 2: Remove old web/ contents (except .env files and static assets)**

```bash
cd C:\Users\Arnold\Work\NdoleStudio\httpsms.com
# Remove old source files but keep .env and static for reference
Remove-Item -Recurse -Force web/node_modules, web/.nuxt, web/dist, web/coverage
# We'll scaffold fresh and copy what we need
```

- [ ] **Step 3: Initialize fresh Nuxt 4 project**

```bash
cd C:\Users\Arnold\Work\NdoleStudio\httpsms.com
# Remove old web directory entirely
Remove-Item -Recurse -Force web
# Create fresh Nuxt 4 project
npx nuxi@latest init web --template v4 --package-manager pnpm
cd web
```

- [ ] **Step 4: Install all dependencies**

```bash
cd C:\Users\Arnold\Work\NdoleStudio\httpsms.com\web
pnpm add vuetify@latest @mdi/js sass pusher-js firebase chart.js chartjs-adapter-moment vue-chartjs date-fns libphonenumber-js qrcode moment
pnpm add -D @types/qrcode vuetify-nuxt-module
pnpm add nuxt-vuefire
```

- [ ] **Step 5: Commit scaffold**

```bash
cd C:\Users\Arnold\Work\NdoleStudio\httpsms.com
git add -A
git commit -m "feat(web): scaffold fresh Nuxt 4 project for migration"
```

---

## Task 2: Configure nuxt.config.ts

**Files:**
- Create: `web/nuxt.config.ts`

- [ ] **Step 1: Write nuxt.config.ts with all modules and settings**

```typescript
// web/nuxt.config.ts
export default defineNuxtConfig({
  compatibilityDate: '2025-01-01',

  ssr: true,

  modules: [
    'vuetify-nuxt-module',
    'nuxt-vuefire',
    '@pinia/nuxt',
  ],

  css: [
    'vuetify/styles',
  ],

  build: {
    transpile: ['vuetify', 'chart.js', 'vue-chartjs'],
  },

  vite: {
    define: {
      'process.env.DEBUG': false,
    },
    css: {
      preprocessorOptions: {
        scss: {
          api: 'modern-compiler',
        },
      },
    },
  },

  vuetify: {
    moduleOptions: {
      styles: { configFile: 'app/assets/styles/settings.scss' },
    },
    vuetifyOptions: {
      theme: {
        defaultTheme: 'dark',
      },
      icons: {
        defaultSet: 'mdi-svg',
      },
      display: {
        thresholds: {
          md: 960,
          lg: 1280,
          xl: 1920,
          xxl: 2560,
        },
      },
    },
  },

  vuefire: {
    config: {
      apiKey: process.env.FIREBASE_API_KEY,
      authDomain: process.env.FIREBASE_AUTH_DOMAIN,
      projectId: process.env.FIREBASE_PROJECT_ID,
      storageBucket: process.env.FIREBASE_STORAGE_BUCKET,
      messagingSenderId: process.env.FIREBASE_MESSAGING_SENDER_ID,
      appId: process.env.FIREBASE_APP_ID,
      measurementId: process.env.FIREBASE_MEASUREMENT_ID,
    },
    auth: {
      enabled: true,
    },
  },

  runtimeConfig: {
    public: {
      apiBaseUrl: process.env.API_BASE_URL || 'http://localhost:8000',
      appUrl: process.env.APP_URL || 'https://httpsms.com',
      appName: process.env.APP_NAME || 'HTTP SMS',
      appGithubUrl: process.env.APP_GITHUB_URL || 'https://github.com/NdoleStudio/httpsms',
      appDocumentationUrl: process.env.APP_DOCUMENTATION_URL || 'https://docs.httpsms.com',
      appDownloadUrl: process.env.APP_DOWNLOAD_URL || 'https://apk.httpsms.com/HttpSms.apk',
      appEnv: process.env.APP_ENV || 'production',
      checkoutUrl: process.env.CHECKOUT_URL || '',
      enterpriseCheckoutUrl: process.env.ENTERPRISE_CHECKOUT_URL || '',
      cloudflareTurnstileSiteKey: process.env.CLOUDFLARE_TURNSTILE_SITE_KEY || '',
      pusherKey: process.env.PUSHER_KEY || '',
      pusherCluster: process.env.PUSHER_CLUSTER || '',
    },
  },

  nitro: {
    prerender: {
      routes: ['/'],
    },
  },

  routeRules: {
    '/messages': { ssr: false },
    '/settings': { ssr: false },
    '/threads/**': { ssr: false },
    '/billing': { ssr: false },
    '/bulk-messages': { ssr: false },
  },

  app: {
    head: {
      titleTemplate: '%s',
      title: 'Convert your android phone into an SMS gateway - httpSMS',
      htmlAttrs: { lang: 'en' },
      script: [
        { src: '/integrations.js', async: true, defer: true },
        { src: 'https://lmsqueezy.com/affiliate.js', async: true, defer: true },
        { src: 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit' },
      ],
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'Use your android phone to send and receive SMS messages using a simple HTTP API.' },
        { name: 'format-detection', content: 'telephone=no' },
        { name: 'twitter:site', content: '@NdoleStudio' },
        { name: 'twitter:card', content: 'summary_large_image' },
        { property: 'og:title', content: 'Convert your android phone into an SMS gateway - httpSMS' },
        { property: 'og:description', content: 'Use your android phone to send and receive SMS messages using a simple HTTP API.' },
        { property: 'og:image', content: 'https://httpsms.com/header.png' },
      ],
      link: [{ rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' }],
    },
  },
})
```

- [ ] **Step 2: Create Vuetify SASS settings file**

```scss
// web/app/assets/styles/settings.scss
@use 'vuetify/settings' with (
  $grid-breakpoints: (
    'md': 960px,
    'lg': 1280px,
    'xl': 1920px,
    'xxl': 2560px,
  ),
);
```

- [ ] **Step 3: Create app.vue**

```vue
<!-- web/app/app.vue -->
<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>
```

- [ ] **Step 4: Copy .env file**

```bash
# Copy from old .env (reference the backed-up content)
cd C:\Users\Arnold\Work\NdoleStudio\httpsms.com\web
```

Create `web/.env` with the same environment variables as the original.

- [ ] **Step 5: Commit configuration**

```bash
git add -A
git commit -m "feat(web): configure nuxt.config.ts with Vuetify 4, VueFire, Pinia"
```

---

## Task 3: Port Utility Files

**Files:**
- Create: `web/app/utils/bag.ts`
- Create: `web/app/utils/capitalize.ts`
- Create: `web/app/utils/errors.ts`
- Create: `web/app/utils/filters.ts`

- [ ] **Step 1: Create bag.ts**

```typescript
// web/app/utils/bag.ts
export default class Bag<T> {
  private items = new Map<string, Array<T>>()

  serialize(): Record<string, Array<T>> {
    const result: Record<string, Array<T>> = {}
    this.items.forEach((value: T[], key) => {
      result[key] = value
    })
    return result
  }

  static fromObject<T>(items: Record<string, Array<T>>): Bag<T> {
    const result = new Bag<T>()
    Object.keys(items).forEach((key) => {
      result.addMany(key, items[key])
    })
    return result
  }

  add(key: string, value: T): this {
    let messages: Array<T> | undefined = this.items.get(key)
    if (messages === undefined) {
      messages = []
    }

    if (!messages.includes(value)) {
      messages.push(value)
    }

    this.items.set(key, messages)
    return this
  }

  addMany(key: string, values: Array<T>): this {
    values.forEach((value: T) => {
      this.add(key, value)
    })
    return this
  }

  has(key: string): boolean {
    return this.items.has(key)
  }

  first(key: string): T | undefined {
    if (this.has(key)) {
      return this.get(key)[0] ?? undefined
    }
    return undefined
  }

  get(key: string): Array<T> {
    const result = this.items.get(key)
    if (result === undefined) {
      return []
    }
    return result
  }

  size(): number {
    return this.items.size
  }
}
```

- [ ] **Step 2: Create capitalize.ts**

```typescript
// web/app/utils/capitalize.ts
export function capitalize(value: string | null): string {
  if (!value) {
    return ''
  }
  return value.charAt(0).toUpperCase() + value.slice(1)
}
```

- [ ] **Step 3: Create errors.ts**

```typescript
// web/app/utils/errors.ts
import Bag from '~/utils/bag'
import { capitalize } from '~/utils/capitalize'

export class ErrorMessages extends Bag<string> {}

const sanitize = (key: string, values: Array<string>): Array<string> => {
  return values.map((value: string) => {
    return capitalize(
      value
        .split(key)
        .join(key.replace('_', ' '))
        .split('_')
        .join(' ')
        .split('-')
        .join(' ')
        .split(' char')
        .join(' character')
        .split(' field ')
        .join(' '),
    )
  })
}

interface AxiosLikeError {
  response?: {
    data?: { data?: Record<string, string[]> }
    status?: number
  }
}

export const getErrorMessages = (error: AxiosLikeError): ErrorMessages => {
  const errors = new ErrorMessages()
  if (
    error === null ||
    typeof error.response?.data?.data !== 'object' ||
    error.response?.data?.data === null ||
    error.response?.status !== 422
  ) {
    return errors
  }

  Object.keys(error.response.data.data).forEach((key: string) => {
    errors.addMany(key, sanitize(key, error.response!.data!.data![key]))
  })

  return errors
}
```

- [ ] **Step 4: Create filters.ts**

```typescript
// web/app/utils/filters.ts
import { intervalToDuration, formatDuration } from 'date-fns'
import { parsePhoneNumber, isValidPhoneNumber } from 'libphonenumber-js'

export function formatPhoneNumber(value: string): string {
  if (!isValidPhoneNumber(value)) {
    return value
  }
  const phoneNumber = parsePhoneNumber(value)
  if (phoneNumber) {
    return phoneNumber.formatInternational()
  }
  return value
}

export function phoneCountry(value: string): string {
  const phoneNumber = parsePhoneNumber(value)
  if (phoneNumber && phoneNumber.country) {
    const regionNames = new Intl.DisplayNames(undefined, { type: 'region' })
    return regionNames.of(phoneNumber.country) ?? 'Earth'
  }
  return 'Earth'
}

export function formatTimestamp(value: string): string {
  return new Date(value).toLocaleString()
}

export function formatMoney(value: string | number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(typeof value === 'string' ? parseInt(value) : value)
}

export function formatDecimal(value: string | number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'decimal',
  }).format(typeof value === 'string' ? parseInt(value) : value)
}

export function formatBillingPeriod(value: string): string {
  return new Date(value).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
  })
}

export function humanizeTime(value: string): string {
  const durations = intervalToDuration({
    start: new Date(),
    end: new Date(value),
  })
  return formatDuration(durations)
}
```

- [ ] **Step 5: Commit utilities**

```bash
git add -A
git commit -m "feat(web): port utility files (bag, capitalize, errors, filters)"
```

---

## Task 4: Port API Models/Types

**Files:**
- Create: `web/shared/types/message-thread.ts`
- Create: `web/shared/types/message.ts`
- Create: `web/shared/types/heartbeat.ts`
- Create: `web/shared/types/billing.ts`
- Create: `web/shared/types/user.ts`
- Copy: `web/shared/types/api.ts` (from old `web/models/api.ts`)

- [ ] **Step 1: Create shared types directory and copy model files**

Copy the existing model files from the old `web/models/` directory (available on the backup branch) into `web/shared/types/`. These files are pure TypeScript interfaces and need no changes:

```typescript
// web/shared/types/message-thread.ts
export interface MessageThread {
  color: string
  contact: string
  created_at: string
  id: string
  last_message_content: string
  last_message_id: string
  is_archived: boolean
  order_timestamp: string
  owner: string
  updated_at: string
}
```

```typescript
// web/shared/types/message.ts
export interface Message {
  contact: string
  content: string
  attachments: Array<string> | null
  created_at: string
  failure_reason: string
  id: string
  last_attempted_at: string | null
  order_timestamp: string
  owner: string
  received_at: string | null
  request_received_at: string | null
  send_time: number | null
  sent_at: string
  status: string
  type: string
  updated_at: string
}

export interface SearchMessagesRequest {
  owners: string[]
  types: string[]
  statuses: string[]
  query: string
  sort_by: string
  token?: string
  sort_descending: boolean
  skip: number
  limit: number
}
```

```typescript
// web/shared/types/heartbeat.ts
export interface Heartbeat {
  id: string
  owner: string
  phone_number: string
  charging: boolean
  timestamp: string
}
```

```typescript
// web/shared/types/billing.ts
export interface BillingUsage {
  id: string
  user_id: string
  period_start: string
  period_end: string
  sent_messages: number
  received_messages: number
}
```

```typescript
// web/shared/types/user.ts
export interface User {
  id: string
  email: string
  api_key: string
  active_phone_id: string | null
  timezone: string
  subscription_id: string | null
  subscription_name: string | null
  subscription_status: string | null
  notification_message_status_enabled: boolean
  notification_webhooks_enabled: boolean
}
```

- [ ] **Step 2: Copy the auto-generated api.ts**

Copy `web/models/api.ts` from the backup branch to `web/shared/types/api.ts`. This file is auto-generated from Swagger and needs no modification.

- [ ] **Step 3: Commit types**

```bash
git add -A
git commit -m "feat(web): port API model types to shared/types/"
```

---

## Task 5: Create useApi Composable

**Files:**
- Create: `web/app/composables/useApi.ts`

- [ ] **Step 1: Create the API composable**

```typescript
// web/app/composables/useApi.ts
import type { UseFetchOptions } from 'nuxt/app'

let authToken: string | null = null
let apiKey: string | null = null

export function setAuthHeader(token: string | null) {
  authToken = token
}

export function setApiKey(key: string | null) {
  apiKey = key
}

export function useApi() {
  const config = useRuntimeConfig()
  const baseURL = config.public.apiBaseUrl as string

  const apiFetch = $fetch.create({
    baseURL,
    headers: {
      'X-Client-Version': 'web',
    },
    onRequest({ options }) {
      const headers = (options.headers ||= {}) as Record<string, string>
      if (authToken) {
        headers.Authorization = `Bearer ${authToken}`
      }
      if (apiKey) {
        headers['x-api-key'] = apiKey
      }
    },
  })

  return { apiFetch, setAuthHeader, setApiKey }
}
```

- [ ] **Step 2: Create useFilters composable**

```typescript
// web/app/composables/useFilters.ts
import {
  formatPhoneNumber,
  phoneCountry,
  formatTimestamp,
  formatMoney,
  formatDecimal,
  formatBillingPeriod,
  humanizeTime,
} from '~/utils/filters'
import { capitalize } from '~/utils/capitalize'

export function useFilters() {
  return {
    formatPhoneNumber,
    phoneCountry,
    formatTimestamp,
    formatMoney,
    formatDecimal,
    formatBillingPeriod,
    humanizeTime,
    capitalize,
  }
}
```

- [ ] **Step 3: Commit composables**

```bash
git add -A
git commit -m "feat(web): create useApi and useFilters composables"
```

---

## Task 6: Create Pinia Stores — Notifications & App

**Files:**
- Create: `web/app/stores/notifications.ts`
- Create: `web/app/stores/app.ts`

- [ ] **Step 1: Create notifications store**

```typescript
// web/app/stores/notifications.ts
import { defineStore } from 'pinia'

export type NotificationType = 'error' | 'success' | 'info'

export interface Notification {
  message: string
  timeout: number
  active: boolean
  type: NotificationType
}

export interface NotificationRequest {
  message: string
  type: NotificationType
}

const DEFAULT_TIMEOUT = 3000

export const useNotificationsStore = defineStore('notifications', () => {
  const notification = ref<Notification>({
    active: false,
    message: '',
    type: 'success',
    timeout: DEFAULT_TIMEOUT,
  })

  function addNotification(request: NotificationRequest) {
    notification.value = {
      active: true,
      message: request.message,
      type: request.type,
      timeout: Math.floor(Math.random() * 100) + DEFAULT_TIMEOUT,
    }
  }

  function disableNotification() {
    notification.value.active = false
  }

  return {
    notification,
    addNotification,
    disableNotification,
  }
})
```

- [ ] **Step 2: Create app store**

```typescript
// web/app/stores/app.ts
import { defineStore } from 'pinia'

export interface AppData {
  url: string
  name: string
  env: string
  appDownloadUrl: string
  documentationUrl: string
  githubUrl: string
}

export const useAppStore = defineStore('app', () => {
  const config = useRuntimeConfig()
  const polling = ref(false)

  const appData = computed<AppData>(() => {
    let url = (config.public.appUrl as string) || ''
    if (url.length > 0 && url[url.length - 1] === '/') {
      url = url.substring(0, url.length - 1)
    }
    return {
      url,
      env: config.public.appEnv as string,
      appDownloadUrl: config.public.appDownloadUrl as string,
      documentationUrl: config.public.appDocumentationUrl as string,
      githubUrl: config.public.appGithubUrl as string,
      name: config.public.appName as string,
    }
  })

  const isLocal = computed(() => config.public.appEnv === 'local')

  function setPolling(value: boolean) {
    polling.value = value
  }

  return {
    polling,
    appData,
    isLocal,
    setPolling,
  }
})
```

- [ ] **Step 3: Commit stores**

```bash
git add -A
git commit -m "feat(web): create notifications and app Pinia stores"
```

---

## Task 7: Create Pinia Stores — Auth

**Files:**
- Create: `web/app/stores/auth.ts`

- [ ] **Step 1: Create auth store**

```typescript
// web/app/stores/auth.ts
import { defineStore } from 'pinia'
import { useCurrentUser } from 'vuefire'
import { setAuthHeader, setApiKey } from '~/composables/useApi'
import type { User } from '~~/shared/types/user'

export interface AuthUser {
  email: string | null
  displayName: string | null
  id: string
}

export const useAuthStore = defineStore('auth', () => {
  const authStateChanged = ref(false)
  const authUser = ref<AuthUser | null>(null)
  const user = ref<User | null>(null)
  const { apiFetch } = useApi()

  async function setAuthUserAction(newUser: AuthUser | null | undefined) {
    const userChanged = newUser?.id !== authUser.value?.id
    authUser.value = newUser ?? null
    authStateChanged.value = true

    if (userChanged && newUser !== null) {
      await Promise.all([loadUser(), loadPhones()])
    }
  }

  async function onAuthStateChanged(firebaseUser: any) {
    if (firebaseUser == null) {
      authUser.value = null
      user.value = null
      authStateChanged.value = true
      setApiKey('')
      return
    }
    setAuthHeader(await firebaseUser.getIdToken())
    const { uid, email, displayName } = firebaseUser
    authUser.value = { id: uid, email, displayName }
    authStateChanged.value = true
  }

  async function onIdTokenChanged(firebaseUser: any) {
    if (firebaseUser == null) {
      setApiKey('')
      return
    }
    setAuthHeader(await firebaseUser.getIdToken())
  }

  async function loadUser() {
    const response = await apiFetch<{ data: User }>('/v1/users/me')
    user.value = response.data
  }

  async function updateUser(payload: { owner?: string; timezone?: string }) {
    const phonesStore = usePhonesStore()
    if (payload.owner) {
      phonesStore.setOwner(payload.owner)
    }

    const activePhone = phonesStore.activePhone
    if (!activePhone) return

    const response = await apiFetch<{ data: User }>('/v1/users/me', {
      method: 'PUT',
      body: {
        active_phone_id: activePhone.id,
        timezone: payload.timezone ?? user.value?.timezone,
      },
    })

    setApiKey(response.data.api_key)
    user.value = response.data
  }

  async function deleteUserAccount(): Promise<string> {
    const response = await apiFetch<{ message: string }>('/v1/users/me', {
      method: 'DELETE',
    })
    return response.message
  }

  async function rotateApiKey(userId: string): Promise<User> {
    const response = await apiFetch<{ data: User }>(`/v1/users/${userId}/api-keys`, {
      method: 'DELETE',
    })
    user.value = response.data
    setApiKey(response.data.api_key)
    return response.data
  }

  function resetState() {
    user.value = null
    authUser.value = null
    authStateChanged.value = true
    setApiKey('')
  }

  // Import usePhonesStore lazily to avoid circular deps
  function loadPhones() {
    const phonesStore = usePhonesStore()
    return phonesStore.loadPhones(false)
  }

  return {
    authStateChanged,
    authUser,
    user,
    setAuthUserAction,
    onAuthStateChanged,
    onIdTokenChanged,
    loadUser,
    updateUser,
    deleteUserAccount,
    rotateApiKey,
    resetState,
  }
})

// Lazy import to avoid circular dependency
function usePhonesStore() {
  return (await import('~/stores/phones')).usePhonesStore()
}
```

Note: The circular dependency between auth and phones will be resolved by using dynamic imports. The actual implementation should use `const { usePhonesStore } = await import('~/stores/phones')` or restructure to avoid the cycle entirely.

- [ ] **Step 2: Commit**

```bash
git add -A
git commit -m "feat(web): create auth Pinia store"
```

---

## Task 8: Create Pinia Stores — Phones

**Files:**
- Create: `web/app/stores/phones.ts`

- [ ] **Step 1: Create phones store**

```typescript
// web/app/stores/phones.ts
import { defineStore } from 'pinia'
import type { EntitiesPhone } from '~~/shared/types/api'
import type { Heartbeat } from '~~/shared/types/heartbeat'

export const usePhonesStore = defineStore('phones', () => {
  const phones = ref<EntitiesPhone[]>([])
  const owner = ref<string | null>(null)
  const heartbeat = ref<Heartbeat | null>(null)
  const { apiFetch } = useApi()
  const notificationsStore = useNotificationsStore()

  const activePhone = computed<EntitiesPhone | null>(() => {
    return phones.value.find((x) => x.phone_number === owner.value) ?? null
  })

  function setOwner(value: string) {
    owner.value = value
  }

  async function loadPhones(force: boolean = false) {
    if (phones.value.length > 0 && !force) return

    const response = await apiFetch<{ data: EntitiesPhone[] }>('/v1/phones', {
      params: { limit: 100 },
    })
    phones.value = response.data

    const authStore = useAuthStore()
    if (authStore.user?.active_phone_id) {
      const phone = response.data.find((x) => x.id === authStore.user?.active_phone_id)
      if (phone) {
        owner.value = phone.phone_number
      }
    }

    if (!owner.value && phones.value.length > 0) {
      owner.value = phones.value[0].phone_number
    }
  }

  async function deletePhone(phoneID: string) {
    await apiFetch(`/v1/phones/${phoneID}`, { method: 'DELETE' })
    await loadPhones(true)
  }

  async function updatePhone(phone: EntitiesPhone) {
    try {
      const response = await apiFetch<{ message: string }>('/v1/phones', {
        method: 'PUT',
        body: {
          fcm_token: phone.fcm_token,
          sim: phone.sim,
          phone_number: phone.phone_number,
          message_expiration_seconds: parseInt(phone.message_expiration_seconds.toString()),
          missed_call_auto_reply: phone.missed_call_auto_reply,
          max_send_attempts: parseInt(phone.max_send_attempts.toString()),
          messages_per_minute: parseInt(phone.messages_per_minute.toString()),
          message_send_schedule_id: phone.message_send_schedule_id ?? null,
        },
      })
      notificationsStore.addNotification({ message: response.message, type: 'success' })
      await loadPhones(true)
    } catch (error: any) {
      notificationsStore.addNotification({
        message: error?.data?.message ?? 'Error while updating phone',
        type: 'error',
      })
    }
  }

  async function getHeartbeat(limit = 1): Promise<Heartbeat[]> {
    const response = await apiFetch<{ data: Heartbeat[] }>('/v1/heartbeats', {
      params: { limit, owner: owner.value },
    })
    if (response.data.length > 0) {
      heartbeat.value = response.data[0]
    } else {
      heartbeat.value = null
    }
    return response.data
  }

  function resetState() {
    phones.value = []
    owner.value = null
    heartbeat.value = null
  }

  return {
    phones,
    owner,
    heartbeat,
    activePhone,
    setOwner,
    loadPhones,
    deletePhone,
    updatePhone,
    getHeartbeat,
    resetState,
  }
})
```

- [ ] **Step 2: Commit**

```bash
git add -A
git commit -m "feat(web): create phones Pinia store"
```

---

## Task 9: Create Pinia Stores — Threads & Messages

**Files:**
- Create: `web/app/stores/threads.ts`
- Create: `web/app/stores/messages.ts`

- [ ] **Step 1: Create threads store**

```typescript
// web/app/stores/threads.ts
import { defineStore } from 'pinia'
import type { MessageThread } from '~~/shared/types/message-thread'
import type { Message } from '~~/shared/types/message'

export const useThreadsStore = defineStore('threads', () => {
  const threads = ref<MessageThread[]>([])
  const threadId = ref<string | null>(null)
  const loadingThreads = ref(true)
  const archivedThreads = ref(false)
  const { apiFetch } = useApi()
  const notificationsStore = useNotificationsStore()

  const currentThread = computed<MessageThread | null>(() => {
    return threads.value.find((x) => x.id === threadId.value) ?? null
  })

  const hasThread = computed(() => threadId.value != null && !loadingThreads.value)

  function hasThreadId(id: string): boolean {
    return threads.value.find((x) => x.id === id) !== undefined
  }

  async function loadThreads() {
    const phonesStore = usePhonesStore()
    if (phonesStore.owner === null && phonesStore.phones.length === 0) {
      loadingThreads.value = false
      return
    }

    const response = await apiFetch<{ data: MessageThread[] }>('/v1/message-threads', {
      params: {
        owner: phonesStore.owner ?? phonesStore.phones[0]?.phone_number,
        limit: 100,
        is_archived: archivedThreads.value,
      },
    })

    phonesStore.getHeartbeat().catch(console.error)
    threads.value = [...response.data]
    loadingThreads.value = false
  }

  async function loadThreadMessages(id: string | null): Promise<Message[]> {
    threadId.value = id
    const thread = currentThread.value
    if (!thread) throw new Error(`Cannot find thread with id ${id}`)

    const response = await apiFetch<{ data: Message[] }>('/v1/messages', {
      params: {
        contact: thread.contact,
        owner: thread.owner,
        limit: 50,
      },
    })
    return response.data
  }

  function setThreadId(id: string | null) {
    threadId.value = id
  }

  function toggleArchive() {
    archivedThreads.value = !archivedThreads.value
  }

  async function updateThread(payload: { threadId: string; isArchived: boolean }) {
    await apiFetch(`/v1/message-threads/${payload.threadId}`, {
      method: 'PUT',
      body: { is_archived: payload.isArchived },
    })
    archivedThreads.value = payload.isArchived
    await loadThreads()
  }

  async function deleteThread(id: string) {
    await apiFetch(`/v1/message-threads/${id}`, { method: 'DELETE' })
    threadId.value = null
    notificationsStore.addNotification({
      message: 'The message thread has been deleted successfully',
      type: 'success',
    })
  }

  function resetState() {
    threads.value = []
    threadId.value = null
    archivedThreads.value = false
    loadingThreads.value = true
  }

  return {
    threads,
    threadId,
    loadingThreads,
    archivedThreads,
    currentThread,
    hasThread,
    hasThreadId,
    loadThreads,
    loadThreadMessages,
    setThreadId,
    toggleArchive,
    updateThread,
    deleteThread,
    resetState,
  }
})
```

- [ ] **Step 2: Create messages store**

```typescript
// web/app/stores/messages.ts
import { defineStore } from 'pinia'
import type { EntitiesMessage } from '~~/shared/types/api'
import type { SearchMessagesRequest } from '~~/shared/types/message'

export type SIM = 'SIM1' | 'SIM2' | 'DEFAULT'

export interface SendMessageRequest {
  from: string
  to: string
  content: string
  sim: SIM
  request_id?: string
}

export const useMessagesStore = defineStore('messages', () => {
  const { apiFetch } = useApi()
  const notificationsStore = useNotificationsStore()

  async function sendMessage(request: SendMessageRequest) {
    try {
      const response = await apiFetch<{ message: string }>('/v1/messages/send', {
        method: 'POST',
        body: request,
      })
      notificationsStore.addNotification({ message: response.message, type: 'success' })
    } catch (e: any) {
      notificationsStore.addNotification({
        message: e?.data?.message ?? 'Error while sending message',
        type: 'error',
      })
    }
    const threadsStore = useThreadsStore()
    await threadsStore.loadThreads()
  }

  async function deleteMessage(messageId: string) {
    await apiFetch(`/v1/messages/${messageId}`, { method: 'DELETE' })
    notificationsStore.addNotification({
      message: 'The message has been deleted successfully',
      type: 'success',
    })
  }

  async function searchMessages(payload: SearchMessagesRequest): Promise<EntitiesMessage[]> {
    const token = payload.token
    const params = { ...payload }
    delete params.token

    const response = await apiFetch<{ data: EntitiesMessage[] }>('/v1/messages/search', {
      params,
      headers: token ? { token } : undefined,
    })
    return response.data
  }

  async function sendBulkMessages(document: File): Promise<void> {
    const formData = new FormData()
    formData.append('document', document)
    const response = await apiFetch<{ message?: string }>('/v1/bulk-messages', {
      method: 'POST',
      body: formData,
    })
    notificationsStore.addNotification({
      message: response.message ?? 'Bulk messages sent successfully',
      type: 'success',
    })
  }

  async function fetchBulkMessageOrders(): Promise<any[]> {
    const response = await apiFetch<{ data: any[] }>('/v1/bulk-messages')
    return response.data ?? []
  }

  return {
    sendMessage,
    deleteMessage,
    searchMessages,
    sendBulkMessages,
    fetchBulkMessageOrders,
  }
})
```

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "feat(web): create threads and messages Pinia stores"
```

---

## Task 10: Create Pinia Store — Billing

**Files:**
- Create: `web/app/stores/billing.ts`

- [ ] **Step 1: Create billing store**

```typescript
// web/app/stores/billing.ts
import { defineStore } from 'pinia'
import type { BillingUsage } from '~~/shared/types/billing'
import type {
  EntitiesWebhook,
  EntitiesDiscord,
  EntitiesMessageSendSchedule,
  EntitiesPhoneAPIKey,
  RequestsWebhookStore,
  RequestsWebhookUpdate,
  RequestsDiscordStore,
  RequestsDiscordUpdate,
  RequestsMessageSendScheduleStore,
  RequestsUserNotificationUpdate,
  RequestsUserPaymentInvoice,
  ResponsesUserSubscriptionPaymentsResponse,
} from '~~/shared/types/api'

export const useBillingStore = defineStore('billing', () => {
  const billingUsage = ref<BillingUsage | null>(null)
  const billingUsageHistory = ref<BillingUsage[]>([])
  const { apiFetch } = useApi()
  const notificationsStore = useNotificationsStore()

  async function loadBillingUsage() {
    const response = await apiFetch<{ data: BillingUsage }>('/v1/billing/usage')
    billingUsage.value = response.data
  }

  async function loadBillingUsageHistory() {
    const response = await apiFetch<{ data: BillingUsage[] }>('/v1/billing/usage-history')
    billingUsageHistory.value = response.data
  }

  async function getSubscriptionUpdateLink(): Promise<string> {
    const response = await apiFetch<{ data: string }>('/v1/users/subscription-update-url')
    return response.data
  }

  async function cancelSubscription(): Promise<string> {
    const response = await apiFetch<{ message: string }>('/v1/users/subscription', {
      method: 'DELETE',
    })
    return response.message
  }

  async function indexSubscriptionPayments(): Promise<ResponsesUserSubscriptionPaymentsResponse> {
    const response = await apiFetch<ResponsesUserSubscriptionPaymentsResponse>(
      '/v1/users/subscription/payments',
      { params: { limit: 100 } },
    )
    return response
  }

  async function generateSubscriptionPaymentInvoice(
    subscriptionInvoiceId: string,
    request: RequestsUserPaymentInvoice,
  ): Promise<void> {
    const response = await apiFetch(
      `/v1/users/subscription/invoices/${subscriptionInvoiceId}`,
      {
        method: 'POST',
        body: request,
        responseType: 'blob',
      },
    )

    const pdfBlob = new Blob([response as any], { type: 'application/pdf' })
    const url = window.URL.createObjectURL(pdfBlob)
    const tempLink = document.createElement('a')
    tempLink.href = url
    tempLink.setAttribute('download', 'Invoice.pdf')
    document.body.appendChild(tempLink)
    tempLink.click()
    document.body.removeChild(tempLink)
    window.URL.revokeObjectURL(url)
  }

  // Webhooks
  async function createWebhook(payload: RequestsWebhookStore): Promise<EntitiesWebhook> {
    const response = await apiFetch<{ data: EntitiesWebhook }>('/v1/webhooks', {
      method: 'POST',
      body: payload,
    })
    return response.data
  }

  async function getWebhooks(): Promise<EntitiesWebhook[]> {
    const response = await apiFetch<{ data: EntitiesWebhook[] }>('/v1/webhooks', {
      params: { limit: 100 },
    })
    return response.data
  }

  async function updateWebhook(payload: RequestsWebhookUpdate & { id: string }): Promise<EntitiesWebhook> {
    const response = await apiFetch<{ data: EntitiesWebhook }>(`/v1/webhooks/${payload.id}`, {
      method: 'PUT',
      body: payload,
    })
    return response.data
  }

  async function deleteWebhook(id: string): Promise<void> {
    await apiFetch(`/v1/webhooks/${id}`, { method: 'DELETE' })
  }

  // Discord
  async function createDiscord(payload: RequestsDiscordStore): Promise<EntitiesDiscord> {
    const response = await apiFetch<{ data: EntitiesDiscord }>('/v1/discord-integrations', {
      method: 'POST',
      body: payload,
    })
    return response.data
  }

  async function getDiscordIntegrations(): Promise<EntitiesDiscord[]> {
    const response = await apiFetch<{ data: EntitiesDiscord[] }>('/v1/discord-integrations', {
      params: { limit: 100 },
    })
    return response.data
  }

  async function updateDiscordIntegration(payload: RequestsDiscordUpdate & { id: string }): Promise<EntitiesDiscord> {
    const response = await apiFetch<{ data: EntitiesDiscord }>(`/v1/discord-integrations/${payload.id}`, {
      method: 'PUT',
      body: payload,
    })
    return response.data
  }

  async function deleteDiscordIntegration(id: string): Promise<void> {
    await apiFetch(`/v1/discord-integrations/${id}`, { method: 'DELETE' })
  }

  // Send Schedules
  async function getSendSchedules(): Promise<EntitiesMessageSendSchedule[]> {
    const response = await apiFetch<{ data: EntitiesMessageSendSchedule[] }>('/v1/send-schedules')
    return response.data
  }

  async function createSendSchedule(payload: RequestsMessageSendScheduleStore): Promise<EntitiesMessageSendSchedule> {
    const response = await apiFetch<{ data: EntitiesMessageSendSchedule }>('/v1/send-schedules', {
      method: 'POST',
      body: payload,
    })
    return response.data
  }

  async function updateSendSchedule(payload: RequestsMessageSendScheduleStore & { id: string }): Promise<EntitiesMessageSendSchedule> {
    const response = await apiFetch<{ data: EntitiesMessageSendSchedule }>(`/v1/send-schedules/${payload.id}`, {
      method: 'PUT',
      body: payload,
    })
    return response.data
  }

  async function deleteSendSchedule(id: string): Promise<void> {
    await apiFetch(`/v1/send-schedules/${id}`, { method: 'DELETE' })
  }

  // Phone API Keys
  async function storePhoneApiKey(name: string): Promise<EntitiesPhoneAPIKey> {
    const response = await apiFetch<{ data: EntitiesPhoneAPIKey; message: string }>('/v1/phone-api-keys', {
      method: 'POST',
      body: { name },
    })
    notificationsStore.addNotification({ message: response.message, type: 'success' })
    return response.data
  }

  async function indexPhoneApiKeys(): Promise<EntitiesPhoneAPIKey[]> {
    const response = await apiFetch<{ data: EntitiesPhoneAPIKey[] }>('/v1/phone-api-keys', {
      params: { limit: 100 },
    })
    return response.data
  }

  async function deletePhoneApiKey(id: string): Promise<void> {
    const response = await apiFetch<{ message: string }>(`/v1/phone-api-keys/${id}`, { method: 'DELETE' })
    notificationsStore.addNotification({ message: response.message, type: 'success' })
  }

  async function deletePhoneFromPhoneApiKey(phoneApiKeyId: string, phoneId: string): Promise<void> {
    const response = await apiFetch<{ message: string }>(
      `/v1/phone-api-keys/${phoneApiKeyId}/phones/${phoneId}`,
      { method: 'DELETE' },
    )
    notificationsStore.addNotification({ message: response.message, type: 'success' })
  }

  // Email notifications
  async function saveEmailNotifications(userId: string, payload: RequestsUserNotificationUpdate): Promise<void> {
    const authStore = useAuthStore()
    const response = await apiFetch<{ data: any }>(`/v1/users/${userId}/notifications`, {
      method: 'PUT',
      body: payload,
    })
    authStore.user = response.data
  }

  return {
    billingUsage,
    billingUsageHistory,
    loadBillingUsage,
    loadBillingUsageHistory,
    getSubscriptionUpdateLink,
    cancelSubscription,
    indexSubscriptionPayments,
    generateSubscriptionPaymentInvoice,
    createWebhook,
    getWebhooks,
    updateWebhook,
    deleteWebhook,
    createDiscord,
    getDiscordIntegrations,
    updateDiscordIntegration,
    deleteDiscordIntegration,
    getSendSchedules,
    createSendSchedule,
    updateSendSchedule,
    deleteSendSchedule,
    storePhoneApiKey,
    indexPhoneApiKeys,
    deletePhoneApiKey,
    deletePhoneFromPhoneApiKey,
    saveEmailNotifications,
  }
})
```

- [ ] **Step 2: Commit**

```bash
git add -A
git commit -m "feat(web): create billing Pinia store"
```

---

## Task 11: Create Middleware

**Files:**
- Create: `web/app/middleware/auth.ts`
- Create: `web/app/middleware/guest.ts`

- [ ] **Step 1: Create auth middleware**

```typescript
// web/app/middleware/auth.ts
export default defineNuxtRouteMiddleware((to) => {
  const authStore = useAuthStore()
  if (authStore.authUser === null) {
    return navigateTo({ path: '/login', query: { to: to.path } })
  }
})
```

- [ ] **Step 2: Create guest middleware**

```typescript
// web/app/middleware/guest.ts
export default defineNuxtRouteMiddleware(() => {
  const authStore = useAuthStore()
  if (authStore.authUser !== null) {
    return navigateTo('/threads')
  }
})
```

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "feat(web): create auth and guest route middleware"
```

---

## Task 12: Port Layouts

**Files:**
- Create: `web/app/layouts/default.vue`
- Create: `web/app/layouts/website.vue`
- Create: `web/app/layouts/error.vue`

**IMPORTANT:** Before writing each layout, call `vuetify-mcp-get_component_api_by_version` for every Vuetify component used (v-app, v-navigation-drawer, v-main, v-app-bar, v-footer, v-container, v-row, v-col, v-btn, v-snackbar, etc.) and `vuetify-mcp-get_v4_breaking_changes` to check for any breaking changes.

- [ ] **Step 1: Create default layout**

```vue
<!-- web/app/layouts/default.vue -->
<script setup lang="ts">
import Pusher from 'pusher-js'
import { useDisplay } from 'vuetify'
import { setAuthHeader } from '~/composables/useApi'
import { useCurrentUser } from 'vuefire'
import { getAuth } from 'firebase/auth'

const route = useRoute()
const config = useRuntimeConfig()
const { lgAndUp } = useDisplay()
const authStore = useAuthStore()
const phonesStore = usePhonesStore()
const threadsStore = useThreadsStore()
const appStore = useAppStore()
const firebaseUser = useCurrentUser()

let poller: ReturnType<typeof setInterval> | null = null
let canPoll = false

const hasDrawer = computed(() => {
  return ['threads', 'threads-id'].includes(route.name as string ?? '')
})

onMounted(() => {
  setTimeout(() => {
    const pusher = new Pusher(config.public.pusherKey as string, {
      cluster: config.public.pusherCluster as string,
    })

    if (authStore.authUser) {
      const channel = pusher.subscribe(authStore.authUser.id)
      channel.bind('phone.updated', () => {
        canPoll = true
      })
    }

    startPoller()
  }, 10_000)
})

onBeforeUnmount(() => {
  if (poller) clearInterval(poller)
})

function startPoller() {
  poller = setInterval(async () => {
    if (!canPoll || authStore.authUser == null) return

    appStore.setPolling(true)

    if (authStore.authUser && phonesStore.owner) {
      const auth = getAuth()
      const token = await auth.currentUser?.getIdToken()
      if (token) setAuthHeader(token)

      await Promise.all([
        phonesStore.loadPhones(true),
        threadsStore.loadThreads(),
        phonesStore.getHeartbeat(),
      ])
    }

    canPoll = false
    setTimeout(() => appStore.setPolling(false), 1000)
  }, 10_000)
}
</script>

<template>
  <v-app>
    <v-divider v-if="appStore.isLocal" class="py-1 bg-warning" />
    <v-navigation-drawer
      v-if="lgAndUp && hasDrawer"
      :width="400"
      permanent
    >
      <template #prepend>
        <v-divider v-if="appStore.isLocal" class="py-1 bg-warning" />
        <MessageThreadHeader />
        <div class="overflow-y-auto v-navigation-drawer__message-thread">
          <MessageThread />
        </div>
      </template>
    </v-navigation-drawer>
    <v-main :class="{ 'has-drawer': hasDrawer && lgAndUp }">
      <Toast />
      <slot v-if="authStore.authStateChanged" />
      <LoadingDashboard v-else />
    </v-main>
  </v-app>
</template>

<style lang="scss">
.v-application {
  .w-full {
    width: 100%;
  }
  .h-full {
    height: 100%;
  }
  .has-drawer {
    .v-snackbar {
      padding-left: 400px;
    }
  }
  .v-navigation-drawer__message-thread {
    height: calc(100vh - 120px);
    &::-webkit-scrollbar {
      width: 8px;
    }
    &::-webkit-scrollbar-track {
      background: #363636;
    }
    &::-webkit-scrollbar-thumb {
      background: #666666;
      border-radius: 8px;
    }
  }
  code.hljs {
    font-size: 16px;
  }
}
</style>
```

- [ ] **Step 2: Create website layout**

```vue
<!-- web/app/layouts/website.vue -->
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
    router.push('/#pricing')
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
              <h3 v-if="lgAndUp" class="text-h4 ml-1 text-on-surface">
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
              <h3 class="text-h4 ml-1 text-on-surface">httpSMS</h3>
            </NuxtLink>
            <div class="text-subtitle-2 mb-4 text-medium-emphasis">
              Made With <v-icon color="#cf1112" :icon="mdiHeart" /> in Tallinn
              <v-img
                class="d-inline-block"
                width="20"
                src="https://upload.wikimedia.org/wikipedia/commons/8/8f/Flag_of_Estonia.svg"
              />
            </div>
            <p class="mt-n3">
              <v-btn href="https://twitter.com/httpsmsHQ" icon color="#1DA1F2" :icon="mdiTwitter" />
              <v-btn :href="appStore.appData.githubUrl" icon size="large" color="#ffffff" :icon="mdiGithub" />
              <v-btn href="https://discord.gg/kGk8HVqeEZ" icon size="large" color="#5865f2">
                <v-img contain height="24" width="24" src="/img/discord-logo-blue.svg" />
              </v-btn>
            </p>
          </v-col>
          <v-col cols="12" md="3">
            <h2 class="text-h6 mb-2">Resources</h2>
            <ul style="list-style: none" class="pa-0">
              <li class="mb-2">
                <a class="text-on-surface text-decoration-none" @click.stop="goToPricing">
                  Pricing <v-icon size="small" :icon="mdiCreditCardOutline" />
                </a>
              </li>
              <li class="mb-2">
                <a href="https://httpsms.lemonsqueezy.com/affiliates" class="text-on-surface text-decoration-none">
                  Affiliates <v-icon color="warning" size="small" :icon="mdiShieldStar" />
                </a>
              </li>
              <li class="mb-2">
                <a href="https://status.httpsms.com" class="text-on-surface text-decoration-none">
                  Site status <v-icon color="success" size="x-small" :icon="mdiCircle" />
                </a>
              </li>
              <li class="mb-2">
                <NuxtLink class="text-on-surface text-decoration-none" to="/blog">
                  Blog <v-icon size="small" :icon="mdiPost" />
                </NuxtLink>
              </li>
            </ul>
          </v-col>
          <v-col cols="12" md="3">
            <h2 class="text-h6 mb-2">Developers</h2>
            <ul style="list-style: none" class="pa-0">
              <li class="mb-2">
                <a :href="appStore.appData.documentationUrl" class="text-on-surface text-decoration-none">
                  Documentation <v-icon size="small" :icon="mdiBookOpenVariant" />
                </a>
              </li>
              <li class="mb-2">
                <a :href="appStore.appData.githubUrl" class="text-on-surface text-decoration-none">
                  Github <v-icon size="small" :icon="mdiGithub" />
                </a>
              </li>
              <li class="mb-2">
                <a href="https://sandbox.httpsms.com" class="text-on-surface text-decoration-none">
                  Sandbox <v-icon size="small" color="pink" :icon="mdiCreation" />
                </a>
              </li>
              <li class="mb-2">
                <a href="https://httpsms.featurebase.app" class="text-on-surface text-decoration-none">
                  Request Feature <v-icon size="small" color="yellow" :icon="mdiLightbulbOn50" />
                </a>
              </li>
            </ul>
          </v-col>
          <v-col cols="12" md="3">
            <h2 class="text-h6 mb-2">Legal</h2>
            <ul style="list-style: none" class="pa-0">
              <li class="mb-2">
                <NuxtLink class="text-on-surface text-decoration-none" to="/terms-and-conditions">
                  Terms & Conditions <v-icon size="small" :icon="mdiScaleBalance" />
                </NuxtLink>
              </li>
              <li class="mb-2">
                <NuxtLink class="text-on-surface text-decoration-none" to="/privacy-policy">
                  Privacy Policy <v-icon size="small" :icon="mdiEyeOffOutline" />
                </NuxtLink>
              </li>
              <li class="mt-2">
                <a class="text-on-surface text-decoration-none" href="mailto:support@httpsms.com">
                  Contact Support <v-icon size="small" :icon="mdiEmailOutline" />
                </a>
              </li>
            </ul>
          </v-col>
        </v-row>
      </v-container>
    </v-footer>
  </v-app>
</template>
```

- [ ] **Step 3: Create error layout**

```vue
<!-- web/app/layouts/error.vue -->
<script setup lang="ts">
const props = defineProps<{
  error: {
    statusCode: number
    message: string
  }
}>()
</script>

<template>
  <v-app>
    <v-main>
      <v-container class="text-center mt-16">
        <h1 class="text-h1">{{ error.statusCode }}</h1>
        <p class="text-h5 mt-4">{{ error.message }}</p>
        <v-btn color="primary" class="mt-8" to="/">Go Home</v-btn>
      </v-container>
    </v-main>
  </v-app>
</template>
```

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat(web): port layouts to Nuxt 4 with Vuetify 4"
```

---

## Task 13: Port Components — Toast, LoadingDashboard, LoadingButton, BackButton

**Files:**
- Create: `web/app/components/Toast.vue`
- Create: `web/app/components/LoadingDashboard.vue`
- Create: `web/app/components/LoadingButton.vue`
- Create: `web/app/components/BackButton.vue`

**IMPORTANT:** Call `vuetify-mcp-get_component_api_by_version` for v-snackbar, v-btn, v-progress-circular, v-icon, v-container, v-row, v-col before writing these.

- [ ] **Step 1: Create Toast.vue**

```vue
<!-- web/app/components/Toast.vue -->
<script setup lang="ts">
import { useDisplay } from 'vuetify'
import { mdiCheck, mdiInformation } from '@mdi/js'

const { lgAndUp } = useDisplay()
const notificationsStore = useNotificationsStore()

const notificationActive = computed({
  get: () => notificationsStore.notification.active,
  set: () => notificationsStore.disableNotification(),
})
</script>

<template>
  <v-snackbar
    v-model="notificationActive"
    :color="notificationsStore.notification.type"
    :timeout="notificationsStore.notification.timeout"
  >
    <v-icon
      v-if="notificationsStore.notification.type === 'success'"
      :color="notificationsStore.notification.type"
      :icon="mdiCheck"
    />
    <v-icon
      v-if="notificationsStore.notification.type === 'info'"
      :color="notificationsStore.notification.type"
      :icon="mdiInformation"
    />
    {{ notificationsStore.notification.message }}
    <template #actions>
      <v-btn
        v-if="lgAndUp"
        :color="notificationsStore.notification.type"
        variant="text"
        @click="notificationsStore.disableNotification()"
      >
        <span class="font-weight-bold">Close</span>
      </v-btn>
    </template>
  </v-snackbar>
</template>
```

- [ ] **Step 2: Create LoadingDashboard.vue**

```vue
<!-- web/app/components/LoadingDashboard.vue -->
<script setup lang="ts">
import { useDisplay } from 'vuetify'
const { mdAndDown } = useDisplay()
</script>

<template>
  <v-main>
    <v-container fluid class="fill-height">
      <v-row align="center" justify="center">
        <v-col
          cols="12"
          md="5"
          xl="3"
          class="text-center mt-16"
          :class="{ 'px-6': mdAndDown, 'px-16': !mdAndDown }"
        >
          <h2 class="text-h4 text-medium-emphasis mt-16 mb-4">
            <img
              class="mx-auto d-inline-block"
              src="/img/logo.svg"
              style="max-width: 32px"
              alt="httpSMS Logo"
            />
            Loading the httpSMS dashboard
          </h2>
          <v-progress-circular
            indeterminate
            size="160"
            class="mt-8"
            color="primary"
          />
        </v-col>
      </v-row>
    </v-container>
  </v-main>
</template>
```

- [ ] **Step 3: Create LoadingButton.vue**

```vue
<!-- web/app/components/LoadingButton.vue -->
<script setup lang="ts">
const props = withDefaults(defineProps<{
  type?: string
  block?: boolean
  large?: boolean
  xLarge?: boolean
  tile?: boolean
  text?: boolean
  small?: boolean
  color?: string
  icon?: string | null
  loading: boolean
}>(), {
  type: 'submit',
  block: false,
  large: false,
  xLarge: false,
  tile: false,
  text: false,
  small: false,
  color: 'primary',
  icon: null,
})

const emit = defineEmits<{
  click: []
  'update:loading': [value: boolean]
}>()

const isClicked = ref(false)

watch(() => props.loading, (submitting) => {
  if (!submitting && isClicked.value) {
    isClicked.value = false
  }
})

function onClick() {
  isClicked.value = true
  emit('click')
}

const size = computed(() => {
  if (props.xLarge) return 'x-large'
  if (props.large) return 'large'
  if (props.small) return 'small'
  return 'default'
})
</script>

<template>
  <v-btn
    :block="block"
    :type="type"
    :size="size"
    :color="color"
    :variant="text ? 'text' : 'elevated'"
    :disabled="loading"
    @click.prevent="onClick"
  >
    <v-progress-circular
      v-if="isClicked"
      :size="small ? 20 : 25"
      color="grey"
      class="mr-2"
      indeterminate
    />
    <v-icon v-if="icon && !loading" start :icon="icon" />
    <slot />
  </v-btn>
</template>
```

- [ ] **Step 4: Create BackButton.vue**

```vue
<!-- web/app/components/BackButton.vue -->
<script setup lang="ts">
import { useDisplay } from 'vuetify'
import { mdiArrowLeft } from '@mdi/js'
import type { RouteLocationRaw } from 'vue-router'

const props = withDefaults(defineProps<{
  route?: RouteLocationRaw
  block?: boolean
}>(), {
  block: false,
})

const router = useRouter()
const { smAndDown } = useDisplay()

function goBack() {
  if (props.route) {
    router.push(props.route)
    return
  }
  if (window.history.length > 1) {
    router.back()
    return
  }
  router.push({ name: 'index' })
}
</script>

<template>
  <v-btn
    color="default"
    :size="smAndDown ? 'small' : 'default'"
    :block="block"
    @click="goBack"
  >
    <v-icon :icon="mdiArrowLeft" />
    Go Back
  </v-btn>
</template>
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat(web): port Toast, LoadingDashboard, LoadingButton, BackButton components"
```

---

## Task 14: Port Components — CopyButton, FixedHeader, BlogAuthorBio, BlogInfo, NuxtLogo

**Files:**
- Create: `web/app/components/CopyButton.vue`
- Create: `web/app/components/FixedHeader.vue`
- Create: `web/app/components/BlogAuthorBio.vue`
- Create: `web/app/components/BlogInfo.vue`
- Create: `web/app/components/NuxtLogo.vue`

**IMPORTANT:** Call `vuetify-mcp-get_component_api_by_version` for each Vuetify component used.

- [ ] **Step 1: Create CopyButton.vue**

```vue
<!-- web/app/components/CopyButton.vue -->
<script setup lang="ts">
import { useDisplay } from 'vuetify'
import { mdiContentCopy } from '@mdi/js'

const props = withDefaults(defineProps<{
  value: string
  color?: string
  block?: boolean
  large?: boolean
  copyText?: string
  notificationText?: string
}>(), {
  color: 'default',
  block: false,
  large: false,
  copyText: 'Copy',
  notificationText: 'Copied',
})

const { smAndDown } = useDisplay()
const notificationsStore = useNotificationsStore()
const disabled = ref(false)

async function copy() {
  disabled.value = true
  await navigator.clipboard.writeText(props.value)
  notificationsStore.addNotification({ message: props.notificationText, type: 'success' })
  setTimeout(() => { disabled.value = false }, 5000)
}
</script>

<template>
  <v-btn
    :disabled="disabled"
    :color="color"
    :size="smAndDown ? 'small' : (large ? 'large' : 'default')"
    :block="block"
    @click="copy"
  >
    <v-icon start :icon="mdiContentCopy" />
    {{ copyText }}
  </v-btn>
</template>
```

- [ ] **Step 2: Create FixedHeader.vue**

```vue
<!-- web/app/components/FixedHeader.vue -->
<script setup lang="ts">
import { useDisplay } from 'vuetify'

const { lgAndUp, mdAndDown } = useDisplay()
</script>

<template>
  <v-app-bar elevation="0" color="#121212" height="70">
    <v-container>
      <v-row>
        <v-col class="w-full d-flex">
          <NuxtLink
            :to="{ name: 'index' }"
            class="text-on-surface text-h4 text-decoration-none"
            :class="{ 'mt-5': mdAndDown, 'mt-4': !mdAndDown }"
          >
            <v-avatar v-if="lgAndUp" :image="'/img/logo.svg'" :size="30" />
            HTTP SMS
          </NuxtLink>
          <v-spacer />
          <v-btn
            color="primary"
            class="mt-5 mb-5"
            :size="lgAndUp ? 'large' : 'default'"
            :to="{ name: 'login' }"
          >
            Get Started
            <span v-if="lgAndUp">&nbsp;For Free</span>
          </v-btn>
        </v-col>
      </v-row>
    </v-container>
  </v-app-bar>
</template>
```

- [ ] **Step 3: Create BlogAuthorBio.vue**

```vue
<!-- web/app/components/BlogAuthorBio.vue -->
<script setup lang="ts">
import { mdiTwitter, mdiGithub } from '@mdi/js'
</script>

<template>
  <div class="d-flex mb-6 mt-8">
    <v-avatar image="/img/arnold.png" />
    <div class="ml-2">
      <p class="text-subtitle-1 mb-n1">Acho Arnold</p>
      <a class="mb-n4 text-decoration-none text-on-surface" href="https://twitter.com/acho_arnold">
        <v-icon color="#1DA1F2" :icon="mdiTwitter" />
      </a>
      <a class="ml-2 text-decoration-none text-on-surface" href="https://github.com/AchoArnold">
        <v-icon color="#FFFFFF" :icon="mdiGithub" />
      </a>
    </div>
  </div>
</template>
```

- [ ] **Step 4: Create BlogInfo.vue**

```vue
<!-- web/app/components/BlogInfo.vue -->
<script setup lang="ts">
import { mdiBookOpenVariant } from '@mdi/js'

const appStore = useAppStore()
</script>

<template>
  <div>
    <NuxtLink to="/" class="text-decoration-none d-flex">
      <v-avatar :image="'/img/logo.svg'" :size="33" class="mt-1" />
      <h3 class="text-h4 text-on-surface ml-1">httpSMS</h3>
    </NuxtLink>
    <p>
      httpSMS is an
      <a class="text-decoration-none" href="https://github.com/NdoleStudio/httpsms">open source</a>
      application that converts your android phone into an SMS gateway so you
      can send and receive SMS messages using a simple HTTP API.
    </p>
    <v-btn :href="appStore.appData.documentationUrl">
      <v-icon start :icon="mdiBookOpenVariant" />
      Documentation
    </v-btn>
  </div>
</template>
```

- [ ] **Step 5: Create NuxtLogo.vue (minimal placeholder)**

```vue
<!-- web/app/components/NuxtLogo.vue -->
<template>
  <img src="/img/logo.svg" alt="httpSMS Logo" style="max-width: 100px" />
</template>
```

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat(web): port CopyButton, FixedHeader, BlogAuthorBio, BlogInfo, NuxtLogo"
```

---

## Task 15: Port Components — FirebaseAuth, MessageThread, MessageThreadHeader

**Files:**
- Create: `web/app/components/FirebaseAuth.vue`
- Create: `web/app/components/MessageThread.vue`
- Create: `web/app/components/MessageThreadHeader.vue`

**IMPORTANT:** Call `vuetify-mcp-get_component_api_by_version` for v-select, v-list, v-list-item, v-menu, v-tooltip, v-sheet, v-progress-linear before writing.

- [ ] **Step 1: Create FirebaseAuth.vue**

This component uses FirebaseUI which must be loaded client-side only. In Nuxt 4, wrap with `<ClientOnly>` or use `.client.vue` suffix.

```vue
<!-- web/app/components/FirebaseAuth.vue -->
<script setup lang="ts">
import { getAuth, GoogleAuthProvider, GithubAuthProvider, EmailAuthProvider } from 'firebase/auth'

const props = withDefaults(defineProps<{
  to?: string
}>(), { to: '/' })

const router = useRouter()
const authStore = useAuthStore()
const notificationsStore = useNotificationsStore()
const appStore = useAppStore()
const authContainer = ref<HTMLElement | null>(null)
const firebaseUIInitialized = ref(false)
let ui: any = null

onMounted(async () => {
  if (!import.meta.client) return

  const firebaseui = await import('firebaseui')
  await import('firebaseui/dist/firebaseui.css')

  const auth = getAuth()
  ui = new firebaseui.auth.AuthUI(auth)
  ui.start('#firebaseui-auth-container', {
    callbacks: {
      signInSuccessWithAuthResult: (authResult: any) => {
        notificationsStore.addNotification({ message: 'Login successful!', type: 'success' })
        authStore.onAuthStateChanged(authResult.user)
        router.push({ path: props.to })
        return false
      },
      uiShown: () => {
        firebaseUIInitialized.value = true
        if (authContainer.value) {
          Array.from(authContainer.value.getElementsByClassName('firebaseui-idp-text-long'))
            .forEach((item: Element) => {
              item.textContent = item.textContent?.replace('Sign in with', 'Continue with') || null
            })
        }
      },
    },
    signInFlow: 'popup',
    signInSuccessUrl: window.location.href,
    signInOptions: [
      GoogleAuthProvider.PROVIDER_ID,
      GithubAuthProvider.PROVIDER_ID,
      EmailAuthProvider.PROVIDER_ID,
    ],
    tosUrl: appStore.appData.url + '/terms-and-conditions',
    privacyPolicyUrl: appStore.appData.url + '/privacy-policy',
  })
})

onBeforeUnmount(() => {
  if (ui) ui.delete()
})
</script>

<template>
  <div>
    <div id="firebaseui-auth-container" ref="authContainer" />
    <v-progress-circular
      v-if="!firebaseUIInitialized"
      class="mx-auto d-block my-16"
      :size="80"
      :width="5"
      color="primary"
      indeterminate
    />
  </div>
</template>
```

- [ ] **Step 2: Create MessageThread.vue**

Port from the existing component. This is a larger component — read the full source from the backup branch and rewrite using `<script setup>`, replacing `$store` with Pinia stores, `$vuetify.breakpoint` with `useDisplay()`, and Vue 2 filters with function calls.

```vue
<!-- web/app/components/MessageThread.vue -->
<script setup lang="ts">
import { useDisplay } from 'vuetify'
import { mdiPlus } from '@mdi/js'
import { formatPhoneNumber } from '~/utils/filters'

const { mdAndDown } = useDisplay()
const threadsStore = useThreadsStore()
const phonesStore = usePhonesStore()
</script>

<template>
  <div>
    <v-progress-linear
      v-if="threadsStore.loadingThreads"
      color="primary"
      indeterminate
    />
    <div
      v-if="!threadsStore.loadingThreads && threadsStore.archivedThreads"
      class="bg-warning py-1 text-center text-uppercase text-subtitle-1"
    >
      Archived Messages
    </div>
    <v-sheet
      v-if="!threadsStore.loadingThreads && threadsStore.threads.length === 0 && !threadsStore.archivedThreads"
      class="text-center mt-8 mx-3"
      :color="mdAndDown ? '#121212' : '#363636'"
    >
      <div v-if="mdAndDown">
        <v-img
          class="mx-auto mb-4"
          max-width="80%"
          src="/img/person-texting.svg"
        />
        <p v-if="phonesStore.owner" class="text-medium-emphasis">
          Start sending messages
        </p>
      </div>
      <v-btn
        v-if="phonesStore.owner && phonesStore.phones.length !== 0"
        color="primary"
        :to="{ name: 'messages' }"
      >
        <v-icon :icon="mdiPlus" />
        New Message
      </v-btn>
    </v-sheet>
    <!-- Thread list items would be ported from the full MessageThread.vue source -->
    <!-- Each thread item renders as a clickable list item navigating to /threads/[id] -->
  </div>
</template>
```

Note: The full MessageThread.vue content should be ported from the backup branch. The pattern above shows the migration approach.

- [ ] **Step 3: Create MessageThreadHeader.vue**

```vue
<!-- web/app/components/MessageThreadHeader.vue -->
<script setup lang="ts">
import { useDisplay } from 'vuetify'
import { getAuth, signOut } from 'firebase/auth'
import {
  mdiPlus, mdiAccountCog, mdiLogout, mdiCellphoneKey, mdiDownload,
  mdiFinance, mdiBatteryChargingHigh, mdiPackageUp, mdiPackageDown,
  mdiDotsVertical, mdiMagnify, mdiCommentTextMultipleOutline, mdiCircle,
} from '@mdi/js'
import { formatPhoneNumber, phoneCountry, humanizeTime } from '~/utils/filters'
import type { EntitiesPhone } from '~~/shared/types/api'

const router = useRouter()
const route = useRoute()
const { mdAndDown, mdAndUp, lgAndUp } = useDisplay()
const authStore = useAuthStore()
const phonesStore = usePhonesStore()
const threadsStore = useThreadsStore()
const appStore = useAppStore()
const notificationsStore = useNotificationsStore()

const selectedMenuItem = ref(-1)

interface SelectItem {
  title: string
  value: string
}

const owners = computed<SelectItem[]>(() => {
  return phonesStore.phones.map((phone: EntitiesPhone) => ({
    title: formatPhoneNumber(phone.phone_number),
    value: phone.phone_number,
  }))
})

async function onOwnerChanged(owner: string) {
  await authStore.updateUser({ owner })
  if (route.name !== 'threads') {
    threadsStore.setThreadId(null)
    await router.push({ name: 'threads' })
    return
  }
  await threadsStore.loadThreads()
}

async function toggleArchive() {
  threadsStore.toggleArchive()
  setTimeout(() => { selectedMenuItem.value = -1 }, 1000)
  if (route.name !== 'threads') {
    threadsStore.setThreadId(null)
    await router.push({ name: 'threads' })
    return
  }
  await threadsStore.loadThreads()
}

async function logout() {
  const auth = getAuth()
  await signOut(auth)
  authStore.resetState()
  phonesStore.resetState()
  threadsStore.resetState()
  notificationsStore.addNotification({ type: 'info', message: 'You have successfully logged out' })
  router.push({ name: 'index' })
}
</script>

<template>
  <v-sheet
    class="pa-4 d-flex"
    :elevation="lgAndUp ? 0 : 2"
    :color="lgAndUp ? 'grey-darken-4' : 'black'"
  >
    <div :class="{ 'px-2': mdAndDown }">
      <v-toolbar-title>
        <div class="d-flex pt-2" style="width: 245px">
          <v-select
            variant="outlined"
            density="compact"
            :disabled="owners.length === 0"
            placeholder="Phone Numbers"
            :class="{ 'mb-n6': !phonesStore.owner }"
            :items="owners"
            :model-value="phonesStore.owner"
            @update:model-value="onOwnerChanged"
          />
          <div style="width: 50px">
            <v-progress-circular
              v-if="appStore.polling"
              indeterminate
              :size="20"
              :width="1"
              class="mt-3 ml-2"
              color="success"
            />
          </div>
        </div>
      </v-toolbar-title>
      <div v-if="phonesStore.owner" class="d-flex mt-n4">
        <p class="text-medium-emphasis mb-n1">
          {{ phoneCountry(phonesStore.owner) }}
        </p>
        <v-tooltip
          v-if="phonesStore.heartbeat"
          location="end"
        >
          <template #activator="{ props: tooltipProps }">
            <v-btn
              v-bind="tooltipProps"
              size="x-small"
              :to="{ name: 'heartbeats-id', params: { id: phonesStore.owner } }"
              color="success"
              class="ml-2 mt-1 mb-n1"
              icon
            >
              <v-icon v-if="phonesStore.heartbeat.charging" size="small" class="mt-n1" :icon="mdiBatteryChargingHigh" />
              <v-icon v-else size="x-small" :icon="mdiCircle" />
            </v-btn>
          </template>
          <h4>Last Heartbeat</h4>
          {{ humanizeTime(phonesStore.heartbeat.timestamp) }} ago
        </v-tooltip>
      </div>
    </div>
    <v-spacer />
    <v-menu>
      <template #activator="{ props: menuProps }">
        <v-btn v-bind="menuProps" icon variant="text" class="mt-2">
          <v-icon :icon="mdiDotsVertical" />
        </v-btn>
      </template>
      <v-list class="px-2" nav :density="mdAndDown ? 'compact' : 'default'">
        <v-list-item @click.prevent="toggleArchive">
          <template #prepend>
            <v-icon v-if="!threadsStore.archivedThreads" :icon="mdiPackageDown" />
            <v-icon v-else :icon="mdiPackageUp" />
          </template>
          <v-list-item-title>
            {{ threadsStore.archivedThreads ? 'Unarchived' : 'Archived' }}
          </v-list-item-title>
        </v-list-item>
        <v-list-item v-if="phonesStore.owner" :to="{ name: 'messages' }">
          <template #prepend><v-icon :icon="mdiPlus" /></template>
          <v-list-item-title>New Message</v-list-item-title>
        </v-list-item>
        <v-list-item v-if="phonesStore.owner" :to="{ name: 'bulk-messages' }">
          <template #prepend><v-icon :icon="mdiCommentTextMultipleOutline" /></template>
          <v-list-item-title>Bulk Messages</v-list-item-title>
        </v-list-item>
        <v-list-item v-if="phonesStore.owner" :to="{ name: 'search-messages' }">
          <template #prepend><v-icon :icon="mdiMagnify" /></template>
          <v-list-item-title>Search Messages</v-list-item-title>
        </v-list-item>
        <v-list-item :to="{ name: 'settings' }">
          <template #prepend><v-icon :icon="mdiAccountCog" /></template>
          <v-list-item-title>Settings</v-list-item-title>
        </v-list-item>
        <v-list-item :to="{ name: 'phone-api-keys' }">
          <template #prepend><v-icon :icon="mdiCellphoneKey" /></template>
          <v-list-item-title>Phone API Keys</v-list-item-title>
        </v-list-item>
        <v-list-item v-if="phonesStore.owner" :href="appStore.appData.appDownloadUrl">
          <template #prepend><v-icon :icon="mdiDownload" /></template>
          <v-list-item-title>Install App</v-list-item-title>
        </v-list-item>
        <v-list-item :to="{ name: 'billing' }">
          <template #prepend><v-icon :icon="mdiFinance" /></template>
          <v-list-item-title>Usage & Billing</v-list-item-title>
        </v-list-item>
        <v-list-item @click.prevent="logout">
          <template #prepend><v-icon :icon="mdiLogout" /></template>
          <v-list-item-title>Logout</v-list-item-title>
        </v-list-item>
      </v-list>
    </v-menu>
  </v-sheet>
</template>
```

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat(web): port FirebaseAuth, MessageThread, MessageThreadHeader components"
```

---

## Task 16: Port Pages — Login & Index (Homepage)

**Files:**
- Create: `web/app/pages/login.vue`
- Create: `web/app/pages/index.vue`

**IMPORTANT:** Call `vuetify-mcp-get_component_api_by_version` for all Vuetify components used. Port from the backup branch source, converting class-based syntax to `<script setup>`, Vuex to Pinia, `$vuetify.breakpoint` to `useDisplay()`, `nuxt-link` to `NuxtLink`, `require()` for images to static `/img/` paths.

- [ ] **Step 1: Create login.vue**

Read the source from backup branch `web/pages/login.vue`, then rewrite with:
- `<script setup lang="ts">`
- `useDisplay()` for breakpoints
- `useAuthStore()` instead of `$store`
- `definePageMeta({ layout: 'website', middleware: ['guest'] })`
- Vuetify 4 component API

- [ ] **Step 2: Create index.vue (homepage)**

Read the source from backup branch `web/pages/index.vue`, then rewrite with:
- `<script setup lang="ts">`
- `definePageMeta({ layout: 'website' })`
- `useDisplay()` for all breakpoint references
- Static image paths (`/img/...`) instead of `require()`
- Vuetify 4 component API (v-btn `size` prop instead of `large`/`small`, `variant` instead of `text`)

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "feat(web): port login and homepage pages"
```

---

## Task 17: Port Pages — Threads, Messages, Search, Bulk Messages

**Files:**
- Create: `web/app/pages/threads/index.vue`
- Create: `web/app/pages/threads/[id]/index.vue`
- Create: `web/app/pages/messages/index.vue`
- Create: `web/app/pages/search-messages/index.vue`
- Create: `web/app/pages/bulk-messages/index.vue`

For each page, apply the same migration pattern:
1. Read original source from backup branch
2. Rewrite with `<script setup lang="ts">`
3. Replace `$store` with Pinia stores
4. Replace `$vuetify.breakpoint` with `useDisplay()`
5. Replace `nuxt-link` with `NuxtLink`
6. Replace filters (`| filterName`) with function calls
7. Use Vuetify 4 component API
8. Add `definePageMeta({ middleware: ['auth'] })` for authenticated pages

- [ ] **Step 1: Port threads/index.vue**
- [ ] **Step 2: Port threads/[id]/index.vue** (rename from `_id`)
- [ ] **Step 3: Port messages/index.vue**
- [ ] **Step 4: Port search-messages/index.vue**
- [ ] **Step 5: Port bulk-messages/index.vue**
- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat(web): port threads, messages, search, and bulk messages pages"
```

---

## Task 18: Port Pages — Settings, Billing, Heartbeats, Phone API Keys

**Files:**
- Create: `web/app/pages/settings/index.vue`
- Create: `web/app/pages/billing/index.vue`
- Create: `web/app/pages/heartbeats/[id].vue`
- Create: `web/app/pages/phone-api-keys/index.vue`

Same migration pattern as Task 17.

- [ ] **Step 1: Port settings/index.vue**
- [ ] **Step 2: Port billing/index.vue**
- [ ] **Step 3: Port heartbeats/[id].vue** (rename from `_id.vue`)
- [ ] **Step 4: Port phone-api-keys/index.vue**
- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat(web): port settings, billing, heartbeats, phone-api-keys pages"
```

---

## Task 19: Port Pages — Blog, Legal

**Files:**
- Create: `web/app/pages/blog/index.vue`
- Create: `web/app/pages/blog/how-to-send-sms-messages-from-excel.vue`
- Create: `web/app/pages/blog/grant-send-and-read-sms-permissions-on-android.vue`
- Create: `web/app/pages/blog/forward-incoming-sms-from-phone-to-webhook.vue`
- Create: `web/app/pages/blog/end-to-end-encryption-to-sms-messages.vue`
- Create: `web/app/pages/blog/send-bulk-sms-from-csv-file-with-no-code.vue`
- Create: `web/app/pages/blog/send-sms-from-android-phone-with-python.vue`
- Create: `web/app/pages/blog/send-sms-when-new-row-is-added-to-google-sheets-using-zapier.vue`
- Create: `web/app/pages/privacy-policy/index.vue`
- Create: `web/app/pages/terms-and-conditions/index.vue`

Same migration pattern. Blog pages use `layout: 'website'` and are mostly static content with Vuetify components.

- [ ] **Step 1: Port blog/index.vue**
- [ ] **Step 2: Port all blog article pages** (7 files)
- [ ] **Step 3: Port privacy-policy/index.vue**
- [ ] **Step 4: Port terms-and-conditions/index.vue**
- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat(web): port blog and legal pages"
```

---

## Task 20: Port Static Assets & Client Plugins

**Files:**
- Copy: `web/public/` (from old `web/static/`)
- Create: `web/app/plugins/chart.client.ts`
- Create: `web/app/plugins/vue-glow.client.ts`

- [ ] **Step 1: Copy static assets to public/**

```bash
# Copy from backup branch: static/ → public/
# Includes: favicon.ico, integrations.js, header.png, templates/, img/
```

- [ ] **Step 2: Move images from assets/img/ to public/img/**

Images referenced with static paths (`/img/logo.svg`, etc.) go in `public/img/`. Images that need Vite processing stay in `app/assets/img/`.

- [ ] **Step 3: Create chart.client.ts**

```typescript
// web/app/plugins/chart.client.ts
import {
  Chart,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  BarElement,
  Filler,
} from 'chart.js'

export default defineNuxtPlugin(() => {
  Chart.register(
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    BarElement,
    Title,
    Tooltip,
    Legend,
    Filler,
  )
})
```

- [ ] **Step 4: Create vue-glow.client.ts**

```typescript
// web/app/plugins/vue-glow.client.ts
import VueGlow from 'vue-glow'

export default defineNuxtPlugin((nuxtApp) => {
  nuxtApp.vueApp.use(VueGlow)
})
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat(web): port static assets and client plugins"
```

---

## Task 21: Update Docker & Build Config

**Files:**
- Modify: `web/Dockerfile`
- Modify: `web/nginx.conf`
- Create: `web/.env.production`

- [ ] **Step 1: Update Dockerfile for Nuxt 4**

```dockerfile
# web/Dockerfile
FROM node:20-alpine AS builder

RUN corepack enable && corepack prepare pnpm@latest --activate

WORKDIR /app
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY . .
RUN pnpm run generate

FROM nginx:alpine
COPY --from=builder /app/.output/public /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

- [ ] **Step 2: Keep nginx.conf** (should work as-is for static files)

- [ ] **Step 3: Create .env.production** (copy from backup, same vars)

- [ ] **Step 4: Update package.json scripts**

Ensure `package.json` has:
```json
{
  "scripts": {
    "dev": "nuxt dev",
    "build": "nuxt build",
    "generate": "nuxt generate",
    "preview": "nuxt preview",
    "lint": "eslint .",
    "typecheck": "nuxt typecheck"
  }
}
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat(web): update Dockerfile and build configuration for Nuxt 4"
```

---

## Task 22: Verify Build Compiles

- [ ] **Step 1: Install dependencies**

```bash
cd C:\Users\Arnold\Work\NdoleStudio\httpsms.com\web
pnpm install
```

- [ ] **Step 2: Run TypeScript check**

```bash
pnpm typecheck
```

Expected: No TypeScript errors. If there are errors, fix them.

- [ ] **Step 3: Run dev server**

```bash
pnpm dev
```

Expected: Dev server starts on port 3000 without errors.

- [ ] **Step 4: Run static generation**

```bash
pnpm generate
```

Expected: All pages generate successfully. Fix any SSR/prerender errors.

- [ ] **Step 5: Commit any fixes**

```bash
git add -A
git commit -m "fix(web): resolve TypeScript and build errors"
```

---

## Task 23: Verify ALL Layouts Render Correctly

- [ ] **Step 1: Start dev server**

```bash
cd C:\Users\Arnold\Work\NdoleStudio\httpsms.com\web
pnpm dev
```

- [ ] **Step 2: Verify default layout**

Navigate to `http://localhost:3000/threads` (or any auth-required page in logged-in state). Confirm:
- `v-app` renders with dark theme
- Navigation drawer appears on large screens
- Toast component is present
- No console errors

- [ ] **Step 3: Verify website layout**

Navigate to `http://localhost:3000/`. Confirm:
- App bar with logo, navigation buttons renders
- Footer with 4 columns renders
- Dark theme is active
- Responsive breakpoints work (resize browser)

- [ ] **Step 4: Verify error layout**

Navigate to `http://localhost:3000/nonexistent-page`. Confirm:
- Error page renders with status code and message
- "Go Home" button works

---

## Task 24: Verify ALL Components Render Correctly

- [ ] **Step 1: Verify Toast component**

Trigger a notification via the store (e.g., login success). Confirm snackbar appears with correct color, message, and auto-dismisses.

- [ ] **Step 2: Verify LoadingDashboard**

Load the app without being authenticated — confirm the loading spinner and text appear.

- [ ] **Step 3: Verify LoadingButton**

Navigate to any page with a form submit button. Confirm:
- Button shows spinner when clicked
- Disabled state works during loading

- [ ] **Step 4: Verify BackButton**

Navigate to a page with a back button. Confirm it navigates back correctly.

- [ ] **Step 5: Verify CopyButton**

Navigate to settings/API key page. Click copy. Confirm:
- Text is copied to clipboard
- Success notification appears
- Button is temporarily disabled

- [ ] **Step 6: Verify FixedHeader**

Navigate to a blog post page that uses FixedHeader. Confirm app bar renders with logo and "Get Started" button.

- [ ] **Step 7: Verify BlogAuthorBio**

Navigate to any blog post. Confirm author avatar, name, and social links render.

- [ ] **Step 8: Verify BlogInfo**

Navigate to any blog post. Confirm httpSMS description and documentation button render.

- [ ] **Step 9: Verify FirebaseAuth**

Navigate to `/login`. Confirm:
- FirebaseUI widget loads
- Shows Google, GitHub, Email auth options
- Loading spinner shows before widget init

- [ ] **Step 10: Verify MessageThread**

Navigate to `/threads` when logged in. Confirm:
- Thread list renders
- Empty state shows "New Message" button when no threads
- Loading indicator works

- [ ] **Step 11: Verify MessageThreadHeader**

Navigate to `/threads`. Confirm:
- Phone number selector works
- Heartbeat indicator shows
- Menu opens with all items (Archive, New Message, Settings, etc.)
- Logout works

---

## Task 25: Verify ALL Pages Render Correctly

- [ ] **Step 1: Verify pages/index.vue (homepage)**

Navigate to `/`. Confirm:
- Hero section with heading, buttons renders
- "Get Started" and "Live Demo" buttons are clickable
- Images load
- Pricing section exists
- Responsive layout works

- [ ] **Step 2: Verify pages/login.vue**

Navigate to `/login`. Confirm:
- FirebaseAuth component loads
- Guest middleware redirects if already logged in
- Login form appears

- [ ] **Step 3: Verify pages/threads/index.vue**

Navigate to `/threads` (authenticated). Confirm:
- Auth middleware protects the page
- Thread list loads
- Navigation drawer shows on desktop

- [ ] **Step 4: Verify pages/threads/[id]/index.vue**

Navigate to `/threads/some-id`. Confirm:
- Messages for the thread load
- Message input area renders
- Send button works

- [ ] **Step 5: Verify pages/messages/index.vue**

Navigate to `/messages`. Confirm:
- New message form renders
- Phone number inputs work
- Send functionality works

- [ ] **Step 6: Verify pages/search-messages/index.vue**

Navigate to `/search-messages`. Confirm:
- Search form renders
- Results display correctly

- [ ] **Step 7: Verify pages/bulk-messages/index.vue**

Navigate to `/bulk-messages`. Confirm:
- File upload area renders
- CSV template download link works
- Submit button works

- [ ] **Step 8: Verify pages/settings/index.vue**

Navigate to `/settings`. Confirm:
- All settings sections render
- API key display and rotation work
- Webhook management works

- [ ] **Step 9: Verify pages/billing/index.vue**

Navigate to `/billing`. Confirm:
- Usage chart renders
- Subscription info displays
- Payment history loads

- [ ] **Step 10: Verify pages/heartbeats/[id].vue**

Navigate to `/heartbeats/+1234567890`. Confirm:
- Heartbeat history chart renders
- Phone info displays

- [ ] **Step 11: Verify pages/phone-api-keys/index.vue**

Navigate to `/phone-api-keys`. Confirm:
- API keys list renders
- Create new key works
- Delete key works

- [ ] **Step 12: Verify pages/privacy-policy/index.vue**

Navigate to `/privacy-policy`. Confirm page content renders with website layout.

- [ ] **Step 13: Verify pages/terms-and-conditions/index.vue**

Navigate to `/terms-and-conditions`. Confirm page content renders with website layout.

- [ ] **Step 14: Verify pages/blog/index.vue**

Navigate to `/blog`. Confirm blog post listing renders.

- [ ] **Step 15: Verify all blog article pages**

Navigate to each blog article URL and confirm:
- Content renders
- Code highlighting works
- BlogAuthorBio component shows
- BlogInfo component shows
- Website layout is applied

Blog articles to check:
- `/blog/how-to-send-sms-messages-from-excel`
- `/blog/grant-send-and-read-sms-permissions-on-android`
- `/blog/forward-incoming-sms-from-phone-to-webhook`
- `/blog/end-to-end-encryption-to-sms-messages`
- `/blog/send-bulk-sms-from-csv-file-with-no-code`
- `/blog/send-sms-from-android-phone-with-python`
- `/blog/send-sms-when-new-row-is-added-to-google-sheets-using-zapier`

---

## Task 26: Final Build Verification & Commit

- [ ] **Step 1: Run full TypeScript check**

```bash
cd C:\Users\Arnold\Work\NdoleStudio\httpsms.com\web
pnpm typecheck
```

Expected: 0 errors.

- [ ] **Step 2: Run lint**

```bash
pnpm lint
```

Expected: No lint errors (or only auto-fixable ones).

- [ ] **Step 3: Run static generation**

```bash
pnpm generate
```

Expected: All pages pre-render successfully.

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "feat(web): complete Nuxt 4 + Vuetify 4 migration

Migrated from:
- Nuxt 2.18.1 → Nuxt 4.x
- Vue 2.7 → Vue 3.x
- Vuetify 2.7 → Vuetify 4.x
- Vuex 3 → Pinia
- vue-property-decorator → <script setup lang=\"ts\">
- @nuxtjs/firebase → nuxt-vuefire

All components and pages verified to render correctly."
```
