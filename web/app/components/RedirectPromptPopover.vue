<script setup lang="ts">
import { computed } from 'vue'
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

const menuOpen = computed({
  get: () => showPopover.value,
  set: (value: boolean) => {
    if (!value) redirectStore.dismiss()
  },
})
</script>

<template>
  <v-menu
    v-model="menuOpen"
    activator="parent"
    location="bottom end"
    offset="8"
    :open-on-click="false"
    :close-on-content-click="false"
  >
    <v-list width="280" class="py-0" rounded="lg" elevation="8">
      <v-list-item>
        <v-list-item-title class="text-body-1">
          Skip this page next time?
        </v-list-item-title>
        <template #append>
          <v-btn
            :icon="mdiClose"
            variant="text"
            size="small"
            color="warning"
            density="comfortable"
            aria-label="Dismiss"
            @click="redirectStore.dismiss()"
          />
        </template>
      </v-list-item>
      <v-list-item class="text-primary" @click="redirectStore.enable()">
        <v-list-item-title>Always open dashboard</v-list-item-title>
        <template #append>
          <v-icon :icon="mdiArrowRight" size="small" />
        </template>
      </v-list-item>
    </v-list>
  </v-menu>
</template>
