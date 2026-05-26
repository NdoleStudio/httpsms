export default defineNuxtRouteMiddleware((to) => {
  const authStore = useAuthStore();
  if (authStore.authUser === null) {
    return navigateTo({ path: "/login", query: { to: to.path } });
  }
});
