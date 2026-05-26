<script setup lang="ts">
import firebase from "firebase/compat/app";
import "firebase/compat/auth";
import {
  GoogleAuthProvider,
  GithubAuthProvider,
  EmailAuthProvider,
} from "firebase/auth";

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
const authContainer = ref<HTMLElement | null>(null);
const firebaseUIInitialized = ref(false);
let ui: any = null;

onMounted(async () => {
  if (!import.meta.client) return;

  const firebaseui = await import("firebaseui");
  await import("firebaseui/dist/firebaseui.css");

  // FirebaseUI requires the compat auth instance
  const auth = firebase.auth();
  ui = firebaseui.auth.AuthUI.getInstance() || new firebaseui.auth.AuthUI(auth);
  ui.start("#firebaseui-auth-container", {
    callbacks: {
      signInSuccessWithAuthResult: (authResult: any) => {
        notificationsStore.addNotification({
          message: "Login successful!",
          type: "success",
        });
        authStore.onAuthStateChanged(authResult.user);
        router.push({ path: props.to });
        return false;
      },
      uiShown: () => {
        firebaseUIInitialized.value = true;
        if (authContainer.value) {
          Array.from(
            authContainer.value.getElementsByClassName(
              "firebaseui-idp-text-long",
            ),
          ).forEach((item: Element) => {
            item.textContent =
              item.textContent?.replace("Sign in with", "Continue with") ||
              null;
          });
        }
      },
    },
    signInFlow: "popup",
    signInSuccessUrl: window.location.href,
    signInOptions: [
      GoogleAuthProvider.PROVIDER_ID,
      GithubAuthProvider.PROVIDER_ID,
      EmailAuthProvider.PROVIDER_ID,
    ],
    tosUrl: appStore.appData.url + "/terms-and-conditions",
    privacyPolicyUrl: appStore.appData.url + "/privacy-policy",
  });
});

onBeforeUnmount(() => {
  if (ui) ui.delete();
});
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
