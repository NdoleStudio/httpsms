import { useAuthStore } from "../stores/auth";

export default defineNuxtRouteMiddleware(async (to: { path: string }) => {
  const authStore = useAuthStore();
  await authStore.waitForAuthReady();
  if (authStore.authUser === null) {
    return navigateTo({ path: "/login", query: { to: to.path } });
  }
});
