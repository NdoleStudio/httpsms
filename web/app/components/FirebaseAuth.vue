<script setup lang="ts">
import {
  getAuth,
  signInWithPopup,
  GoogleAuthProvider,
  GithubAuthProvider,
  signInWithEmailAndPassword,
  createUserWithEmailAndPassword,
  sendPasswordResetEmail,
} from 'firebase/auth'
import { mdiGoogle, mdiGithub, mdiEmail } from '@mdi/js'
import type { User as FirebaseUser } from 'firebase/auth'
import { ErrorMessages } from '~/utils/errors'

const props = withDefaults(
  defineProps<{
    to?: string
  }>(),
  { to: '/' },
)

const router = useRouter()
const authStore = useAuthStore()
const notificationsStore = useNotificationsStore()
const appStore = useAppStore()

const loading = ref(false)
const showEmailForm = ref(false)
const isSignUp = ref(false)
const showForgotPassword = ref(false)
const resetEmailSent = ref(false)
const email = ref('')
const password = ref('')
const generalError = ref('')
const errorMessages = ref(new ErrorMessages())

type LoginMethod = 'google' | 'github' | 'email'
const LAST_LOGIN_METHOD_KEY = 'httpsms_last_login_method'
const lastUsedMethod = ref<LoginMethod | null>(null)

onMounted(() => {
  try {
    const stored = localStorage.getItem(LAST_LOGIN_METHOD_KEY)
    if (stored === 'google' || stored === 'github' || stored === 'email') {
      lastUsedMethod.value = stored
    }
  } catch (error) {
    console.error(error)
  }
})

function clearErrors() {
  errorMessages.value = new ErrorMessages()
  generalError.value = ''
}

function validateEmail(): boolean {
  clearErrors()
  if (!email.value.trim()) {
    errorMessages.value.add('email', 'Please provide an email address')
    return false
  }
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!emailRegex.test(email.value.trim())) {
    errorMessages.value.add('email', 'Please enter a valid email address')
    return false
  }
  return true
}

function validateLoginForm(): boolean {
  clearErrors()
  let valid = true
  if (!email.value.trim()) {
    errorMessages.value.add('email', 'Please provide an email address')
    valid = false
  } else {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    if (!emailRegex.test(email.value.trim())) {
      errorMessages.value.add('email', 'Please enter a valid email address')
      valid = false
    }
  }
  if (!password.value) {
    errorMessages.value.add('password', 'Please enter your password')
    valid = false
  }
  return valid
}

async function signInWithGoogle() {
  loading.value = true
  try {
    const auth = getAuth()
    const result = await signInWithPopup(auth, new GoogleAuthProvider())
    onSuccess(result.user, 'google')
  } catch (error: unknown) {
    handleError(error, true)
  } finally {
    loading.value = false
  }
}

async function signInWithGithub() {
  loading.value = true
  try {
    const auth = getAuth()
    const result = await signInWithPopup(auth, new GithubAuthProvider())
    onSuccess(result.user, 'github')
  } catch (error: unknown) {
    handleError(error, true)
  } finally {
    loading.value = false
  }
}

async function submitEmail() {
  if (!validateLoginForm()) return
  loading.value = true
  try {
    const auth = getAuth()
    let result
    if (isSignUp.value) {
      result = await createUserWithEmailAndPassword(
        auth,
        email.value.trim(),
        password.value,
      )
    } else {
      result = await signInWithEmailAndPassword(
        auth,
        email.value.trim(),
        password.value,
      )
    }
    onSuccess(result.user, 'email')
  } catch (error: unknown) {
    handleError(error)
  } finally {
    loading.value = false
  }
}

async function submitPasswordReset() {
  if (!validateEmail()) return
  loading.value = true
  try {
    const auth = getAuth()
    await sendPasswordResetEmail(auth, email.value.trim())
    resetEmailSent.value = true
  } catch (error: unknown) {
    handleError(error)
  } finally {
    loading.value = false
  }
}

function showForgotPasswordForm() {
  clearErrors()
  resetEmailSent.value = false
  showForgotPassword.value = true
}

function backToSignIn() {
  clearErrors()
  resetEmailSent.value = false
  showForgotPassword.value = false
}

function onSuccess(user: FirebaseUser, method: LoginMethod) {
  try {
    localStorage.setItem(LAST_LOGIN_METHOD_KEY, method)
  } catch (error) {
    console.error(error)
  }
  notificationsStore.addNotification({
    message: 'Login successful!',
    type: 'success',
  })
  authStore.onAuthStateChanged(user)
  router.push({ path: props.to })
}

function handleError(error: unknown, isSocial = false) {
  const firebaseError = error as { code?: string; message?: string }
  const code = firebaseError.code || ''

  if (
    code === 'auth/popup-closed-by-user' ||
    code === 'auth/cancelled-popup-request'
  ) {
    return
  }

  if (isSocial) {
    const message = getGeneralErrorMessage(code, firebaseError.message)
    notificationsStore.addNotification({ message, type: 'error' })
    return
  }

  clearErrors()

  switch (code) {
    case 'auth/wrong-password':
      errorMessages.value.add('password', 'Incorrect password')
      break
    case 'auth/invalid-credential':
      errorMessages.value.add('email', 'Invalid email or password')
      errorMessages.value.add('password', 'Invalid email or password')
      break
    case 'auth/user-not-found':
      errorMessages.value.add(
        'email',
        'No account found with this email address',
      )
      break
    case 'auth/invalid-email':
      errorMessages.value.add('email', 'Please enter a valid email address')
      break
    case 'auth/email-already-in-use':
      errorMessages.value.add(
        'email',
        'An account already exists with this email',
      )
      break
    case 'auth/weak-password':
      errorMessages.value.add(
        'password',
        'Password should be at least 6 characters',
      )
      break
    case 'auth/user-disabled':
      errorMessages.value.add('email', 'This account has been disabled')
      break
    case 'auth/too-many-requests':
      generalError.value = 'Too many failed attempts. Please try again later'
      break
    case 'auth/network-request-failed':
      generalError.value =
        'Unable to connect to the server. Please check your internet connection'
      break
    case 'auth/missing-email':
      errorMessages.value.add('email', 'Please provide an email address')
      break
    default:
      generalError.value =
        firebaseError.message || 'An unexpected error occurred'
  }
}

function getGeneralErrorMessage(
  code: string,
  fallback: string | undefined,
): string {
  switch (code) {
    case 'auth/user-not-found':
      return 'No account found with this email address'
    case 'auth/wrong-password':
    case 'auth/invalid-credential':
      return 'The provided credentials are invalid.'
    case 'auth/user-disabled':
      return 'This account has been disabled'
    case 'auth/too-many-requests':
      return 'Too many failed attempts. Please try again later'
    case 'auth/network-request-failed':
      return 'Unable to connect to the server. Please check your internet connection'
    default:
      return fallback || 'An unexpected error occurred'
  }
}
</script>

<template>
  <div>
    <v-btn
      block
      color="white"
      size="large"
      class="mb-3 position-relative"
      :loading="loading"
      :disabled="loading"
      @click="signInWithGoogle"
    >
      <v-chip
        v-if="lastUsedMethod === 'google'"
        size="x-small"
        color="primary"
        label
        variant="flat"
        class="position-absolute last-used-chip"
      >
        Last Used
      </v-chip>
      <v-icon color="red" :icon="mdiGoogle" class="mr-2" />
      Continue with Google
    </v-btn>

    <v-btn
      block
      size="large"
      variant="flat"
      color="black"
      class="mb-3 position-relative"
      :loading="loading"
      :disabled="loading"
      @click="signInWithGithub"
    >
      <v-chip
        v-if="lastUsedMethod === 'github'"
        label
        size="x-small"
        color="primary"
        variant="flat"
        class="position-absolute last-used-chip"
      >
        Last Used
      </v-chip>
      <v-icon :icon="mdiGithub" class="mr-2" />
      Continue with GitHub
    </v-btn>

    <v-btn
      v-if="!showEmailForm"
      block
      size="large"
      variant="flat"
      color="red"
      class="mb-3 position-relative"
      :disabled="loading"
      @click="showEmailForm = true"
    >
      <v-chip
        v-if="lastUsedMethod === 'email'"
        label
        size="x-small"
        color="primary"
        variant="flat"
        class="position-absolute last-used-chip"
      >
        Last Used
      </v-chip>
      <v-icon :icon="mdiEmail" class="mr-2" />
      Continue with email
    </v-btn>

    <!-- Forgot Password Form -->
    <v-form
      v-if="showEmailForm && showForgotPassword"
      class="mt-4"
      @submit.prevent="submitPasswordReset"
    >
      <template v-if="!resetEmailSent">
        <p class="text-body-medium text-medium-emphasis mb-4">
          Enter your email address to reset your password
        </p>
        <v-text-field
          v-model="email"
          label="Email Address"
          color="primary"
          type="email"
          variant="outlined"
          density="comfortable"
          class="mb-2"
          :error="errorMessages.has('email')"
          :error-messages="errorMessages.get('email')"
        />
        <v-alert
          v-if="generalError"
          type="error"
          density="compact"
          class="mb-3"
        >
          {{ generalError }}
        </v-alert>
        <v-btn
          block
          size="large"
          color="primary"
          type="submit"
          :loading="loading"
        >
          Send Reset Link
        </v-btn>
      </template>
      <template v-else>
        <v-alert type="success" density="compact" class="mb-3">
          Check your email for password reset instructions
        </v-alert>
      </template>
      <v-btn
        block
        variant="text"
        size="small"
        color="warning"
        class="mt-2"
        @click="backToSignIn"
      >
        Back to Sign In
      </v-btn>
    </v-form>

    <!-- Sign In / Sign Up Form -->
    <v-form
      v-if="showEmailForm && !showForgotPassword"
      class="mt-4"
      @submit.prevent="submitEmail"
    >
      <v-text-field
        v-model="email"
        label="Email Address"
        color="primary"
        type="email"
        variant="outlined"
        density="comfortable"
        class="mb-2"
        :error="errorMessages.has('email')"
        :error-messages="errorMessages.get('email')"
      />
      <v-text-field
        v-model="password"
        label="Password"
        type="password"
        color="primary"
        variant="outlined"
        density="comfortable"
        class="mb-2"
        :error="errorMessages.has('password')"
        :error-messages="errorMessages.get('password')"
      />
      <v-alert v-if="generalError" type="error" density="compact" class="mb-3">
        {{ generalError }}
      </v-alert>
      <v-btn
        v-if="!isSignUp"
        variant="plain"
        size="small"
        color="primary"
        class="mb-3 px-0 mt-n4"
        @click="showForgotPasswordForm"
      >
        Forgot Password?
      </v-btn>
      <v-btn
        block
        size="large"
        color="primary"
        type="submit"
        :loading="loading"
      >
        {{ isSignUp ? 'Sign Up' : 'Sign In' }}
      </v-btn>
      <v-btn
        block
        variant="plain"
        size="small"
        color="primary"
        class="mt-2"
        @click="isSignUp = !isSignUp"
      >
        {{
          isSignUp ? 'Already have an account? Sign In' : 'No account? Sign Up'
        }}
      </v-btn>
    </v-form>

    <p class="text-body-small text-medium-emphasis mt-4">
      By continuing, you are indicating that you accept our
      <a
        :href="appStore.appData.url + '/terms-and-conditions'"
        class="text-decoration-none"
      >
        Terms of Service
      </a>
      and
      <a
        :href="appStore.appData.url + '/privacy-policy'"
        class="text-decoration-none"
      >
        Privacy Policy.</a
      >
    </p>
  </div>
</template>

<style scoped>
.last-used-chip {
  top: -8px;
  left: -8px;
  z-index: 1;
}
</style>
