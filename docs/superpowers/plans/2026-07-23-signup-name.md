# Signup Name Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Require a name during Firebase email/password signup, persist it to the Firebase profile, and clear stale form errors when switching authentication modes.

**Architecture:** Keep the change inside the existing `FirebaseAuth.vue` component. The component will conditionally render and validate the name, call Firebase `updateProfile` immediately after account creation, and use a dedicated mode-toggle handler to clear errors.

**Tech Stack:** Nuxt 4, Vue 3 Composition API, Vuetify 4, TypeScript, Firebase Authentication

## Global Constraints

- The visible field label must be `Name`.
- The name is required only for email/password signup.
- Persist the trimmed name as Firebase `displayName`.
- Do not add a new test framework; this web project currently has no configured tests.

---

### Task 1: Add signup name collection and profile persistence

**Files:**
- Modify: `web/app/components/FirebaseAuth.vue`

**Interfaces:**
- Consumes: Firebase `createUserWithEmailAndPassword(auth, email, password)` and `updateProfile(user, { displayName })`
- Produces: `toggleAuthMode(): void`, required signup `name` state, and Firebase users with a populated `displayName`

- [ ] **Step 1: Confirm the current web test command**

Run:

```bash
cd web
pnpm test
```

Expected: PASS with `No tests configured yet`. Do not introduce a test framework for this focused change.

- [ ] **Step 2: Add Firebase profile support and name state**

In `web/app/components/FirebaseAuth.vue`, add `updateProfile` to the existing
`firebase/auth` import:

```ts
import {
  getAuth,
  signInWithPopup,
  GoogleAuthProvider,
  GithubAuthProvider,
  signInWithEmailAndPassword,
  createUserWithEmailAndPassword,
  sendPasswordResetEmail,
  updateProfile,
} from 'firebase/auth'
```

Add the name state beside the existing email and password state:

```ts
const name = ref('')
const email = ref('')
const password = ref('')
```

- [ ] **Step 3: Require the name during signup validation**

At the start of `validateLoginForm`, after `let valid = true`, add:

```ts
if (isSignUp.value && !name.value.trim()) {
  errorMessages.value.add('name', 'Please provide your name')
  valid = false
}
```

This preserves sign-in validation while requiring a non-whitespace name only
in signup mode.

- [ ] **Step 4: Persist the name after Firebase account creation**

Replace the signup branch in `submitEmail` with:

```ts
if (isSignUp.value) {
  result = await createUserWithEmailAndPassword(
    auth,
    email.value.trim(),
    password.value,
  )
  await updateProfile(result.user, {
    displayName: name.value.trim(),
  })
} else {
  result = await signInWithEmailAndPassword(
    auth,
    email.value.trim(),
    password.value,
  )
}
```

Keep the existing `onSuccess(result.user, 'email')` call after the branch so
the auth store receives the updated Firebase user before redirecting.

- [ ] **Step 5: Clear errors when switching sign-in and signup modes**

Add this function beside the existing form navigation helpers:

```ts
function toggleAuthMode() {
  clearErrors()
  isSignUp.value = !isSignUp.value
}
```

Change the mode-toggle button handler from:

```vue
@click="isSignUp = !isSignUp"
```

to:

```vue
@click="toggleAuthMode"
```

- [ ] **Step 6: Render the signup-only Name field**

Before the email field in the sign-in/sign-up form, add:

```vue
<v-text-field
  v-if="isSignUp"
  v-model="name"
  label="Name"
  color="primary"
  type="text"
  variant="outlined"
  density="comfortable"
  class="mb-2"
  :error="errorMessages.has('name')"
  :error-messages="errorMessages.get('name')"
/>
```

- [ ] **Step 7: Run focused web validation**

Run:

```bash
cd web
pnpm exec eslint app/components/FirebaseAuth.vue
pnpm exec prettier --check app/components/FirebaseAuth.vue
pnpm exec stylelint app/components/FirebaseAuth.vue
```

Expected: all three commands exit successfully with no lint, formatting, or
style errors.

- [ ] **Step 8: Commit the implementation**

```bash
git add web/app/components/FirebaseAuth.vue
git commit -m "feat(web): collect name during signup"
```
