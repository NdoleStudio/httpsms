# Redirect logged-in users from root to /threads — Design

Date: 2026-07-13
Component: `web/` (Nuxt 4 client-only SPA)

## Problem

When a logged-in user visits the root landing page (`/`), they currently see
the full marketing page with a "Dashboard" button that links to `/threads`.
Returning users generally want to go straight to their dashboard. We want an
opt-in, per-browser "always skip the landing page" behavior that redirects them
to `/threads` directly — with no backend changes and fast page loads.

## Goals

- Logged-in users who have opted in on a given browser are redirected from `/`
  to `/threads` immediately, before the heavy landing page renders.
- The opt-in is discoverable via a persistent popover under the "Dashboard"
  button, modeled on the reference UX: a card reading "Skip this page next
  time?" with a ✕ dismiss control and an "Always open dashboard →" link.
- Purely client-side / per-browser (localStorage). No API or backend changes.
- On logout, the preference is fully cleared (store + localStorage).

## Non-goals

- No server-side persistence or cross-device sync.
- No change to `/threads` or its auth guard behavior.
- No redirect on any page other than `/`.

## Context (current state)

- `web/` is a client-only Nuxt 4 SPA (`ssr: false`); Firebase resolves auth
  state on the client and populates `useAuthStore()` — `authUser` (nullable)
  and `authStateChanged` (bool).
- The **Dashboard** button lives in `web/app/layouts/website.vue`, shown when
  `authStore.authUser !== null`, linking to `{ name: 'threads' }`.
- Root `/` (`web/app/pages/index.vue`) uses the `website` layout.
- `/threads` already has an `auth` middleware that waits for
  `authStateChanged` and bounces unauthenticated users to `/login`.
- Existing localStorage convention: keys prefixed `httpsms_` (e.g.
  `httpsms_last_login_method` in `web/app/components/FirebaseAuth.vue`), all
  access wrapped in `try/catch`.
- Logout is triggered in `web/app/components/MessageThreadHeader.vue` and
  `web/app/pages/settings/index.vue`, both calling `authStore.resetState()`,
  `phonesStore.resetState()`, and `threadsStore.resetState()`.

## Design

### 1. Preference store — `web/app/stores/redirectPreference.ts`

A dedicated Pinia store (setup style, matching existing stores) owning the
per-browser preference.

- localStorage key: `httpsms_redirect_to_threads` (value `'true'` when enabled).
- State:
  - `enabled: Ref<boolean>` — hydrated from localStorage on store creation.
  - `dismissedThisSession: Ref<boolean>` — in-memory only, defaults `false`.
- Actions:
  - `enable()` — set `enabled = true`, write localStorage, then
    `navigateTo('/threads')` so the click also takes the user to the dashboard.
  - `dismiss()` — set `dismissedThisSession = true` (no persistence).
  - `resetState()` — set `enabled = false`, clear `dismissedThisSession`, and
    remove the localStorage key.
- All localStorage reads/writes wrapped in `try/catch` (mirroring
  `FirebaseAuth.vue`) so private-mode / disabled storage never throws.

### 2. Optimistic redirect — page middleware on `index.vue`

- A page-scoped middleware (via `definePageMeta({ middleware: [...] })` on
  `index.vue`) reads `localStorage['httpsms_redirect_to_threads']`
  **synchronously**. If truthy, it calls
  `navigateTo('/threads', { replace: true })` and returns.
- Runs before the landing page renders and does **not** wait for Firebase, so
  the redirect is immediate (optimistic). If the session turns out to be
  invalid, `/threads`' existing `auth` guard redirects to `/login`.
- localStorage access wrapped in `try/catch`; on any error, fall through and
  render the landing page normally.

### 3. Popover component — rendered in `website.vue`

A new component (e.g. `web/app/components/RedirectPromptPopover.vue`) anchored
under the Dashboard button in the app bar.

- Implemented with a Vuetify `v-menu` / positioned card (persistent, not
  hover-triggered) matching the reference: line 1 "Skip this page next time?"
  with a trailing ✕ icon button; line 2 an "Always open dashboard →" link.
- Visible only when **all** hold:
  - current route is `/` (root),
  - `authStore.authUser !== null`,
  - `!redirectPreferenceStore.enabled`,
  - `!redirectPreferenceStore.dismissedThisSession`.
- ✕ button → `redirectPreferenceStore.dismiss()` (hidden for the session,
  reappears next visit until opt-in).
- "Always open dashboard →" link → `redirectPreferenceStore.enable()`.
- Link carries `text-decoration-none hover:text-decoration-underline` per repo
  convention.

### 4. Logout wiring

At each existing logout site (`MessageThreadHeader.vue`,
`settings/index.vue`), add `redirectPreferenceStore.resetState()` alongside the
existing `resetState()` calls, so logout clears both the in-memory value and
the persisted localStorage flag. A fresh login then starts opted-out.

## Data flow

1. User (logged in, opted out) lands on `/` → sees landing page + popover.
2. Clicks "Always open dashboard →" → `enable()` writes localStorage +
   navigates to `/threads`.
3. Next visit to `/` → page middleware reads the flag synchronously →
   `navigateTo('/threads', { replace: true })` before render.
4. User logs out → `resetState()` clears store + localStorage → next `/` visit
   shows the landing page again (no redirect).

## Error handling

- All localStorage access is guarded with `try/catch`; failures degrade
  gracefully to "not opted in" (landing page renders, popover may show).
- Optimistic redirect relies on `/threads`' existing auth guard as the
  correctness backstop for stale/invalid sessions.

## Testing

- Follow existing Jest patterns in `web/` where present. At minimum, manual
  verification of: opt-in via popover, redirect on next `/` visit, ✕ session
  dismissal, and full clear on logout. Add unit coverage for the preference
  store's `enable`/`dismiss`/`resetState` and hydration if the repo's test
  setup supports Pinia stores.

## Rollout / isolation

- All work performed in a git worktree off `main`
  (`../httpsms-threads-redirect`, branch `feat/redirect-to-threads`), isolated
  from the current `AchoArnold/affiliates-page` branch.
