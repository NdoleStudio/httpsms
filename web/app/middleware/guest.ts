export default defineNuxtRouteMiddleware(async () => {
  const authStore = useAuthStore();
  await authStore.waitForAuthReady();
  if (authStore.authUser !== null) {
    return navigateTo("/threads");
  }
});
