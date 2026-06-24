# Login "Last Used" Badge — Design

## Problem

The login page (`web/app/pages/login.vue` → `web/app/components/FirebaseAuth.vue`)
offers three authentication methods: Continue with Google, Continue with GitHub,
and Continue with email. Returning users don't get any hint about which method
they used last, so they may pick the wrong provider and end up creating a second
account or failing to sign in.

## Goal

Show a small "Last Used" badge on the button corresponding to the login method
the user most recently used successfully on this device, so they can quickly pick
the right one next time.

## Scope

Single component change: `web/app/components/FirebaseAuth.vue`. No API, store, or
backend changes.

## Approach

### Storage

- Persist the last successful method in `localStorage` under the key
  `httpsms_last_login_method`.
- Value is one of `'google' | 'github' | 'email'`.
- The value is written only **after a successful login** (inside `onSuccess`),
  never on click/attempt.

### Recording the method

- `onSuccess(user)` is currently shared by all three flows. Extend it to
  `onSuccess(user, method)` where `method` is `'google' | 'github' | 'email'`.
  - `signInWithGoogle` → `onSuccess(result.user, 'google')`
  - `signInWithGithub` → `onSuccess(result.user, 'github')`
  - `submitEmail` → `onSuccess(result.user, 'email')`
- Inside `onSuccess`, write the method to `localStorage` before redirecting.

### Reading the method

- A reactive `lastUsedMethod = ref<string | null>(null)`.
- Populated in `onMounted` from `localStorage` (client-only, SSR-safe; the
  component is already rendered inside `<ClientOnly>` on the login page).

### Display

- Each of the three method buttons gets `class="position-relative"`.
- A floating `v-chip` is rendered in the top-right corner of the matching button:
  - Vuetify `v-chip` with `label`, `size="x-small"`, `color="primary"`.
  - `class="position-absolute"` pinned to the top-right corner (slightly
    overlapping), text `Last Used`.
  - Shown via `v-if="lastUsedMethod === 'google'"` (and `'github'`, `'email'`
    respectively).

### Edge cases

- The email button is hidden once the inline email form opens
  (`v-if="!showEmailForm"`), so its badge hides with it automatically. No extra
  handling needed.
- An unknown/empty stored value shows no badge anywhere.
- `localStorage` access is guarded (wrapped in try/catch or `onMounted` only) so
  it never runs during SSR.

## Testing

- Manual verification: sign in with each method, confirm the badge appears on the
  correct button on the next visit to the login page.
- Confirm no SSR/hydration errors (badge logic runs client-side only).

## Out of scope

- Syncing the preference across devices.
- Remembering the specific email address used.
