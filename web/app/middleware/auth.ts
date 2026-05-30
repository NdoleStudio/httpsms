import { useAuthStore } from "../stores/auth";

export default defineNuxtRouteMiddleware((to: { path: string }) => {
  const authStore = useAuthStore();
  if (authStore.authUser === null) {
    return navigateTo({ path: "/login", query: { to: to.path } });
  }
});
