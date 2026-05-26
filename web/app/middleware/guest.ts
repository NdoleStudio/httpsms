export default defineNuxtRouteMiddleware(() => {
  const authStore = useAuthStore();
  if (authStore.authUser !== null) {
    return navigateTo("/threads");
  }
});
