import { initializeApp, getApps } from "firebase/app";
import type { FirebaseApp } from "firebase/app";
import { getAuth, onAuthStateChanged } from "firebase/auth";
import { useAuthStore } from "../stores/auth";

export default defineNuxtPlugin(async () => {
  const config = useRuntimeConfig();
  const publicConfig = config.public as Record<string, string>;

  // Skip initialization if no API key is configured
  if (!publicConfig.firebaseApiKey) {
    console.warn(
      "[firebase] No FIREBASE_API_KEY configured. Auth will not work.",
    );
    return;
  }

  const firebaseConfig = {
    apiKey: publicConfig.firebaseApiKey,
    authDomain: publicConfig.firebaseAuthDomain,
    projectId: publicConfig.firebaseProjectId,
    storageBucket: publicConfig.firebaseStorageBucket,
    messagingSenderId: publicConfig.firebaseMessagingSenderId,
    appId: publicConfig.firebaseAppId,
    measurementId: publicConfig.firebaseMeasurementId,
  };

  // Initialize Firebase (only once)
  const app: FirebaseApp =
    getApps().length === 0 ? initializeApp(firebaseConfig) : getApps()[0];
  const auth = getAuth(app);

  // Also initialize the compat SDK for FirebaseUI
  if (import.meta.client) {
    const firebase = (await import("firebase/compat/app")) as {
      default: {
        apps: unknown[];
        initializeApp: (config: typeof firebaseConfig) => void;
      };
    };
    if (!firebase.default.apps.length) {
      firebase.default.initializeApp(firebaseConfig);
    }
  }

  // Listen for auth state changes and update the auth store
  const authStore = useAuthStore();
  onAuthStateChanged(auth, (user) => {
    authStore.onAuthStateChanged(user);
  });

  return {
    provide: {
      firebaseApp: app,
      firebaseAuth: auth,
    },
  };
});
