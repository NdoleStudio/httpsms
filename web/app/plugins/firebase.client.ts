import { initializeApp, getApps } from "firebase/app";
import { getAuth, onAuthStateChanged } from "firebase/auth";

export default defineNuxtPlugin(() => {
  const config = useRuntimeConfig();

  // Skip initialization if no API key is configured
  if (!config.public.firebaseApiKey) {
    console.warn(
      "[firebase] No FIREBASE_API_KEY configured. Auth will not work.",
    );
    return;
  }

  const firebaseConfig = {
    apiKey: config.public.firebaseApiKey,
    authDomain: config.public.firebaseAuthDomain,
    projectId: config.public.firebaseProjectId,
    storageBucket: config.public.firebaseStorageBucket,
    messagingSenderId: config.public.firebaseMessagingSenderId,
    appId: config.public.firebaseAppId,
    measurementId: config.public.firebaseMeasurementId,
  };

  // Initialize Firebase (only once)
  const app =
    getApps().length === 0 ? initializeApp(firebaseConfig) : getApps()[0];
  const auth = getAuth(app);

  // Also initialize the compat SDK for FirebaseUI
  if (import.meta.client) {
    import("firebase/compat/app").then((firebase) => {
      if (!firebase.default.apps.length) {
        firebase.default.initializeApp(firebaseConfig);
      }
    });
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
