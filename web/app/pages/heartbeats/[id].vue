<script setup lang="ts">
import { mdiArrowLeft, mdiHeartPulse } from "@mdi/js";

definePageMeta({
  middleware: ["auth"],
});

useHead({
  title: "Heartbeats - httpSMS",
});

const route = useRoute();
const { mdAndDown, lgAndUp } = useDisplay();
const authStore = useAuthStore();
const { useApi } = useApiComposable();

const loading = ref(true);
const heartbeats = ref<any[]>([]);
const phoneId = computed(() => route.params.id as string);

async function loadHeartbeats() {
  loading.value = true;
  try {
    const api = useApi();
    const response = await api<{ data: any[] }>(
      `/v1/heartbeats?owner=${phoneId.value}`,
    );
    heartbeats.value = response.data ?? [];
  } finally {
    loading.value = false;
  }
}

onMounted(async () => {
  await authStore.loadUser();
  await loadHeartbeats();
});
</script>

<template>
  <VContainer fluid class="px-0 pt-0" :class="{ 'fill-height': lgAndUp }">
    <div class="w-100 h-100">
      <VAppBar height="60" :density="mdAndDown ? 'compact' : 'default'">
        <VBtn icon to="/threads">
          <VIcon :icon="mdiArrowLeft" />
        </VBtn>
        <VToolbarTitle>
          <VIcon :icon="mdiHeartPulse" class="mr-2" />
          Heartbeats
        </VToolbarTitle>
      </VAppBar>
      <VContainer class="mt-16">
        <VRow>
          <VCol cols="12" md="8" offset-md="2">
            <h4 class="text-h4 mb-3">Phone Heartbeats</h4>
            <p class="text-medium-emphasis">
              Monitor the connectivity status of your Android phones. Heartbeats
              are sent every 15 minutes when the phone is online.
            </p>
            <VProgressLinear v-if="loading" indeterminate class="mb-4" />
            <VTable v-else-if="heartbeats.length">
              <thead>
                <tr class="text-uppercase text-subtitle-2">
                  <th class="text-left">Phone</th>
                  <th class="text-center">Status</th>
                  <th class="text-center">Last Heartbeat</th>
                  <th class="text-center">Charging</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="heartbeat in heartbeats" :key="heartbeat.id">
                  <td>{{ useFilters().phoneNumber(heartbeat.owner) }}</td>
                  <td class="text-center">
                    <VChip
                      :color="heartbeat.is_online ? 'success' : 'error'"
                      size="small"
                    >
                      {{ heartbeat.is_online ? "Online" : "Offline" }}
                    </VChip>
                  </td>
                  <td class="text-center">
                    {{ useFilters().timestamp(heartbeat.timestamp) }}
                  </td>
                  <td class="text-center">
                    {{ heartbeat.charging ? "Yes" : "No" }}
                  </td>
                </tr>
              </tbody>
            </VTable>
            <p v-else class="text-medium-emphasis text-center mt-8">
              No heartbeats found. Make sure the httpSMS app is running on your
              phone.
            </p>
          </VCol>
        </VRow>
      </VContainer>
    </div>
  </VContainer>
</template>
