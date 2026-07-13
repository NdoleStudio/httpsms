# Root → /threads Redirect Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let logged-in users opt in (per-browser) to be redirected from the root landing page (`/`) straight to `/threads`, surfaced via a persistent popover under the "Dashboard" button.

**Architecture:** A dedicated Pinia store owns a localStorage-backed preference. A page-scoped middleware on `index.vue` reads the flag synchronously and redirects before render (optimistic; `/threads`' existing auth guard is the correctness backstop). A popover component in the `website` layout drives opt-in/dismiss. Logout clears store + localStorage.

**Tech Stack:** Nuxt 4 (client-only SPA, `ssr: false`), Vue 3 `<script setup>`, Pinia (setup stores), Vuetify 3/4, TypeScript.

## Global Constraints

- Client-only / per-browser: no API or backend changes. Persistence via `localStorage` only.
- localStorage key convention: prefix `httpsms_`. Use exactly `httpsms_redirect_to_threads`, value `'true'` when enabled.
- Wrap ALL `localStorage` access in `try/catch` (mirroring `web/app/components/FirebaseAuth.vue`). On error, degrade to "not opted in".
- No new test framework: `web/`'s `test` script is a stub. Validate with `pnpm lint` and manual dev-server checks.
- Pinia stores use setup style (`defineStore('name', () => { ... })`) — match `web/app/stores/threads.ts`.
- Hyperlinks carry classes `text-decoration-none hover:text-decoration-underline`.
- Redirect applies to the root route (`/`, route name `index`) ONLY.
- All work happens in the existing worktree `../httpsms-threads-redirect` on branch `feat/redirect-to-threads` (off `main`).

---

### Task 1: Preference store

**Files:**
- Create: `web/app/stores/redirectPreference.ts`

**Interfaces:**
- Consumes: nothing.
- Produces: `useRedirectPreferenceStore()` returning `{ enabled: Ref<boolean>, dismissedThisSession: Ref<boolean>, enable(): void, dismiss(): void, resetState(): void }`.
  - `enable()` sets `enabled=true`, persists to localStorage, then `navigateTo('/threads')`.
  - `dismiss()` sets `dismissedThisSession=true` (no persistence).
  - `resetState()` sets `enabled=false`, `dismissedThisSession=false`, removes the localStorage key.

- [ ] **Step 1: Create the store**

Create `web/app/stores/redirectPreference.ts`:

```ts
import { defineStore } from 'pinia'

const STORAGE_KEY = 'httpsms_redirect_to_threads'

function readFlag(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === 'true'
  } catch (error) {
    console.error(error)
    return false
  }
}

export const useRedirectPreferenceStore = defineStore(
  'redirectPreference',
  () => {
    const enabled = ref(readFlag())
    const dismissedThisSession = ref(false)

    function enable() {
      enabled.value = true
      try {
        localStorage.setItem(STORAGE_KEY, 'true')
      } catch (error) {
        console.error(error)
      }
      navigateTo('/threads')
    }

    function dismiss() {
      dismissedThisSession.value = true
    }

    function resetState() {
      enabled.value = false
      dismissedThisSession.value = false
      try {
        localStorage.removeItem(STORAGE_KEY)
      } catch (error) {
        console.error(error)
      }
    }

    return { enabled, dismissedThisSession, enable, dismiss, resetState }
  },
)
```

Notes: `ref`, `navigateTo`, and `defineStore` are auto-imported in Nuxt (no explicit import needed except `defineStore`, kept for parity with `threads.ts`). `readFlag()` runs at store creation; on the client (SPA) `localStorage` is available.

- [ ] **Step 2: Lint the new file**

Run: `cd web && pnpm lint:js`
Expected: PASS (no ESLint errors for the new file).

- [ ] **Step 3: Commit**

```bash
git add web/app/stores/redirectPreference.ts
git commit -m "feat(web): add redirectPreference store for per-browser threads redirect"
```

---

### Task 2: Optimistic redirect middleware on the root page

**Files:**
- Create: `web/app/middleware/redirectToThreads.ts`
- Modify: `web/app/pages/index.vue` (add middleware to `definePageMeta`)

**Interfaces:**
- Consumes: localStorage key `httpsms_redirect_to_threads` (written by Task 1's `enable()`).
- Produces: a named route middleware `redirectToThreads` applied only to `index.vue`.

- [ ] **Step 1: Create the middleware**

Create `web/app/middleware/redirectToThreads.ts`:

```ts
export default defineNuxtRouteMiddleware(() => {
  try {
    if (localStorage.getItem('httpsms_redirect_to_threads') === 'true') {
      return navigateTo('/threads', { replace: true })
    }
  } catch (error) {
    console.error(error)
  }
})
```

Notes: reads synchronously, does NOT wait for Firebase. If the session is invalid, `/threads`' existing `auth` middleware redirects to `/login`. `localStorage` is safe here because `web/` is `ssr: false` (client-only).

- [ ] **Step 2: Register the middleware on the root page**

In `web/app/pages/index.vue`, update the existing `definePageMeta` block:

```ts
definePageMeta({
  layout: 'website',
})
```

to:

```ts
definePageMeta({
  layout: 'website',
  middleware: ['redirect-to-threads'],
})
```

Note: Nuxt maps the file `redirectToThreads.ts` to the kebab-case name `redirect-to-threads`.

- [ ] **Step 3: Lint**

Run: `cd web && pnpm lint:js`
Expected: PASS.

- [ ] **Step 4: Manual verification**

Run: `cd web && pnpm dev`
Then in the browser console on `/`, run `localStorage.setItem('httpsms_redirect_to_threads','true')` and reload `/`.
Expected: immediately redirected to `/threads` (or to `/login` if not authenticated). Then run `localStorage.removeItem('httpsms_redirect_to_threads')` and reload `/`; expected: landing page renders normally. Stop the dev server when done.

- [ ] **Step 5: Commit**

```bash
git add web/app/middleware/redirectToThreads.ts web/app/pages/index.vue
git commit -m "feat(web): optimistically redirect root to /threads when opted in"
```

---

### Task 3: Persistent opt-in popover component

**Files:**
- Create: `web/app/components/RedirectPromptPopover.vue`

**Interfaces:**
- Consumes: `useAuthStore()` (`authUser`), `useRedirectPreferenceStore()` (`enabled`, `dismissedThisSession`, `enable`, `dismiss`), `useRoute()`.
- Produces: `<RedirectPromptPopover />` — self-contained; renders a positioned card only when it should be visible.

- [ ] **Step 1: Create the component**

Create `web/app/components/RedirectPromptPopover.vue`:

```vue
<script setup lang="ts">
import { mdiClose, mdiArrowRight } from '@mdi/js'

const route = useRoute()
const authStore = useAuthStore()
const redirectStore = useRedirectPreferenceStore()

const showPopover = computed(
  () =>
    route.name === 'index' &&
    authStore.authUser !== null &&
    !redirectStore.enabled &&
    !redirectStore.dismissedThisSession,
)
</script>

<template>
  <v-card
    v-if="showPopover"
    class="redirect-prompt pa-4"
    elevation="8"
    rounded="lg"
    max-width="280"
  >
    <div class="d-flex align-center justify-space-between">
      <span class="text-body-1">Skip this page next time?</span>
      <v-btn
        :icon="mdiClose"
        variant="text"
        size="small"
        color="warning"
        density="comfortable"
        aria-label="Dismiss"
        @click="redirectStore.dismiss()"
      />
    </div>
    <a
      class="text-primary text-decoration-none hover:text-decoration-underline d-inline-flex align-center mt-1"
      href="#"
      @click.prevent="redirectStore.enable()"
    >
      Always open dashboard
      <v-icon :icon="mdiArrowRight" size="small" class="ml-1" />
    </a>
  </v-card>
</template>

<style scoped>
.redirect-prompt {
  position: absolute;
  right: 0;
  top: 100%;
  margin-top: 8px;
  z-index: 10;
}
</style>
```

Notes: `v-btn` close control uses `color="warning"` per repo convention. The card is absolutely positioned under its anchor (Task 4 wraps it in a relatively-positioned container). `computed`, `useRoute` auto-imported.

- [ ] **Step 2: Lint**

Run: `cd web && pnpm lint:js && pnpm lint:style`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add web/app/components/RedirectPromptPopover.vue
git commit -m "feat(web): add redirect opt-in popover component"
```

---

### Task 4: Mount the popover under the Dashboard button

**Files:**
- Modify: `web/app/layouts/website.vue` (around lines 103-112, the Dashboard `v-btn`)

**Interfaces:**
- Consumes: `<RedirectPromptPopover />` (Task 3). Nuxt auto-imports components from `web/app/components`, so no explicit import is needed.
- Produces: the popover rendered relative to the Dashboard button.

- [ ] **Step 1: Wrap the Dashboard button with a positioned container hosting the popover**

In `web/app/layouts/website.vue`, replace the existing Dashboard button block:

```vue
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
```

with:

```vue
            <div
              v-show="authStore.authUser !== null"
              class="position-relative d-inline-block"
            >
              <v-btn
                color="primary"
                variant="flat"
                :class="{ 'mt-5': mdAndUp, 'mt-1': !mdAndUp }"
                :size="lgAndUp ? 'large' : 'default'"
                :to="{ name: 'threads' }"
              >
                Dashboard
              </v-btn>
              <RedirectPromptPopover />
            </div>
```

Note: `position-relative` (Vuetify utility) makes the popover's `position: absolute` anchor to this container.

- [ ] **Step 2: Lint**

Run: `cd web && pnpm lint:js && pnpm lint:style`
Expected: PASS.

- [ ] **Step 3: Manual verification**

Run: `cd web && pnpm dev`. Log in, then visit `/`.
Expected: the "Skip this page next time?" popover appears under the Dashboard button. Clicking ✕ hides it (reappears after reload). Clicking "Always open dashboard →" navigates to `/threads`, and a subsequent visit to `/` auto-redirects. Stop the dev server when done.

- [ ] **Step 4: Commit**

```bash
git add web/app/layouts/website.vue
git commit -m "feat(web): mount redirect opt-in popover under Dashboard button"
```

---

### Task 5: Clear the preference on logout

**Files:**
- Modify: `web/app/components/MessageThreadHeader.vue:68-73` (the `logout` function)
- Modify: `web/app/pages/settings/index.vue:773-775` (the sign-out handler)

**Interfaces:**
- Consumes: `useRedirectPreferenceStore()` (`resetState`).
- Produces: nothing new.

- [ ] **Step 1: Wire reset into MessageThreadHeader logout**

In `web/app/components/MessageThreadHeader.vue`, first ensure the store instance exists near the other store instances in `<script setup>` (add if absent):

```ts
const redirectPreferenceStore = useRedirectPreferenceStore()
```

Then in `logout()`, add the reset call alongside the existing resets:

```ts
  authStore.resetState()
  phonesStore.resetState()
  threadsStore.resetState()
  redirectPreferenceStore.resetState()
```

- [ ] **Step 2: Wire reset into settings sign-out**

In `web/app/pages/settings/index.vue`, ensure the store instance exists in `<script setup>` (add near the other store instances if absent):

```ts
const redirectPreferenceStore = useRedirectPreferenceStore()
```

Then after the existing resets in the sign-out handler:

```ts
    authStore.resetState()
    phonesStore.resetState()
    redirectPreferenceStore.resetState()
```

- [ ] **Step 3: Lint**

Run: `cd web && pnpm lint:js`
Expected: PASS.

- [ ] **Step 4: Manual verification**

Run: `cd web && pnpm dev`. Log in, opt in via the popover (redirect now active), then log out (from the app menu and/or settings page). Visit `/`.
Expected: landing page renders (no redirect); `localStorage` no longer contains `httpsms_redirect_to_threads`. Stop the dev server when done.

- [ ] **Step 5: Commit**

```bash
git add web/app/components/MessageThreadHeader.vue web/app/pages/settings/index.vue
git commit -m "feat(web): clear threads-redirect preference on logout"
```

---

### Task 6: Final validation

**Files:** none (validation only).

- [ ] **Step 1: Full lint**

Run: `cd web && pnpm lint`
Expected: PASS (js + style + prettier).

- [ ] **Step 2: Production build sanity check**

Run: `cd web && pnpm run generate`
Expected: build completes without errors (static generation succeeds).

- [ ] **Step 3: End-to-end manual pass**

Run: `cd web && pnpm dev` and verify the full flow: logged-out `/` shows landing page (no popover); logged-in `/` shows popover; ✕ dismisses for session; "Always open dashboard →" enables + navigates; return to `/` auto-redirects to `/threads`; logout clears the flag and restores the landing page. Stop the dev server when done.
