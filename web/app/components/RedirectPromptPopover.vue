<script setup lang="ts">
import { mdiClose, mdiArrowRight } from '@mdi/js'

const route = useRoute()
const authStore = useAuthStore()
const redirectStore = useRedirectPreferenceStore()

const showPopover = computed(
  () =>
    route.name === 'index' &&
    authStore.authUser !== null &&
    !redirectStore.enabled &&
    !redirectStore.dismissedThisSession,
)
</script>

<template>
  <v-card
    v-if="showPopover"
    class="redirect-prompt pa-4"
    elevation="8"
    rounded="lg"
    max-width="280"
  >
    <div class="d-flex align-center justify-space-between">
      <span class="text-body-1">Skip this page next time?</span>
      <v-btn
        :icon="mdiClose"
        variant="text"
        size="small"
        color="warning"
        density="comfortable"
        aria-label="Dismiss"
        @click="redirectStore.dismiss()"
      />
    </div>
    <a
      class="text-primary text-decoration-none hover:text-decoration-underline d-inline-flex align-center mt-1"
      href="#"
      @click.prevent="redirectStore.enable()"
    >
      Always open dashboard
      <v-icon :icon="mdiArrowRight" size="small" class="ml-1" />
    </a>
  </v-card>
</template>

<style scoped>
.redirect-prompt {
  position: absolute;
  right: 0;
  top: 100%;
  margin-top: 8px;
  z-index: 10;
}
</style>
