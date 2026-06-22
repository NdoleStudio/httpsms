<script setup lang="ts">
import {
  getAuth,
  signInWithPopup,
  GoogleAuthProvider,
  GithubAuthProvider,
  signInWithEmailAndPassword,
  createUserWithEmailAndPassword,
} from "firebase/auth";
import { mdiGoogle, mdiGithub, mdiEmail } from "@mdi/js";

const props = withDefaults(
  defineProps<{
    to?: string;
  }>(),
  { to: "/" },
);

const router = useRouter();
const authStore = useAuthStore();
const notificationsStore = useNotificationsStore();
const appStore = useAppStore();

const loading = ref(false);
const showEmailForm = ref(false);
const isSignUp = ref(false);
const email = ref("");
const password = ref("");
const emailError = ref("");

async function signInWithGoogle() {
  loading.value = true;
  try {
    const auth = getAuth();
    const result = await signInWithPopup(auth, new GoogleAuthProvider());
    onSuccess(result.user);
  } catch (error: unknown) {
    handleError(error, true);
  } finally {
    loading.value = false;
  }
}

async function signInWithGithub() {
  loading.value = true;
  try {
    const auth = getAuth();
    const result = await signInWithPopup(auth, new GithubAuthProvider());
    onSuccess(result.user);
  } catch (error: unknown) {
    handleError(error, true);
  } finally {
    loading.value = false;
  }
}

async function submitEmail() {
  emailError.value = "";
  loading.value = true;
  try {
    const auth = getAuth();
    let result;
    if (isSignUp.value) {
      result = await createUserWithEmailAndPassword(
        auth,
        email.value,
        password.value,
      );
    } else {
      result = await signInWithEmailAndPassword(
        auth,
        email.value,
        password.value,
      );
    }
    onSuccess(result.user);
  } catch (error: unknown) {
    handleError(error);
  } finally {
    loading.value = false;
  }
}

function onSuccess(user: unknown) {
  notificationsStore.addNotification({
    message: "Login successful!",
    type: "success",
  });
  authStore.onAuthStateChanged(user);
  router.push({ path: props.to });
}

function handleError(error: unknown, isSocial = false) {
  const firebaseError = error as { code?: string; message?: string };
  const code = firebaseError.code || "";
  let message = "";
  if (code === "auth/user-not-found" || code === "auth/invalid-credential") {
    message = "Invalid email or password";
  } else if (code === "auth/email-already-in-use") {
    message = "An account with this email already exists";
  } else if (code === "auth/weak-password") {
    message = "Password must be at least 6 characters";
  } else if (
    code === "auth/popup-closed-by-user" ||
    code === "auth/cancelled-popup-request"
  ) {
    // User closed the popup, no error to show
    return;
  } else {
    message = firebaseError.message || "An error occurred";
  }

  if (isSocial) {
    notificationsStore.addNotification({ message, type: "error" });
  } else {
    emailError.value = message;
  }
}
</script>

<template>
  <div class="text-center">
    <v-btn
      block
      color="white"
      size="large"
      class="mb-3"
      :loading="loading"
      :disabled="loading"
      @click="signInWithGoogle"
    >
      <v-icon  color="red" :icon="mdiGoogle" class="mr-2" />
      Continue with Google
    </v-btn>

    <v-btn
      block
      size="large"
      variant="flat"
      color="black"
      class="mb-3"
      :loading="loading"
      :disabled="loading"
      @click="signInWithGithub"
    >
      <v-icon :icon="mdiGithub" class="mr-2" />
      Continue with GitHub
    </v-btn>

    <v-btn
      v-if="!showEmailForm"
      block
      size="large"
      variant="flat"
      color="red"
      class="mb-3"
      :disabled="loading"
      @click="showEmailForm = true"
    >
      <v-icon :icon="mdiEmail" class="mr-2" />
      Continue with email
    </v-btn>

    <v-form v-if="showEmailForm" class="mt-4" @submit.prevent="submitEmail">
      <v-text-field
        v-model="email"
        label="Email"
        type="email"
        variant="outlined"
        density="comfortable"
        class="mb-2"
        required
      />
      <v-text-field
        v-model="password"
        label="Password"
        type="password"
        variant="outlined"
        density="comfortable"
        class="mb-2"
        required
      />
      <v-alert v-if="emailError" type="error" density="compact" class="mb-3">
        {{ emailError }}
      </v-alert>
      <v-btn
        block
        size="large"
        color="primary"
        type="submit"
        :loading="loading"
      >
        {{ isSignUp ? "Sign Up" : "Sign In" }}
      </v-btn>
      <v-btn
        block
        variant="text"
        size="small"
        class="mt-2"
        @click="isSignUp = !isSignUp"
      >
        {{
          isSignUp ? "Already have an account? Sign In" : "No account? Sign Up"
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
      <a :href="appStore.appData.url + '/privacy-policy'" class="text-decoration-none">
        Privacy Policy.</a
      >
    </p>
  </div>
</template>
