<script setup lang="ts">
import {
  mdiArrowLeft,
  mdiCellphoneKey,
  mdiPlus,
  mdiDelete,
  mdiContentCopy,
} from "@mdi/js";

definePageMeta({
  middleware: ["auth"],
});

useHead({
  title: "Phone API Keys - httpSMS",
});

const { mdAndDown, lgAndUp } = useDisplay();
const authStore = useAuthStore();
const notificationsStore = useNotificationsStore();
const { useApi } = useApiComposable();

const loading = ref(true);
const phoneApiKeys = ref<any[]>([]);
const createDialog = ref(false);
const formName = ref("");
const creating = ref(false);

async function loadPhoneApiKeys() {
  loading.value = true;
  try {
    const api = useApi();
    const response = await api<{ data: any[] }>("/v1/phone-api-keys");
    phoneApiKeys.value = response.data ?? [];
  } finally {
    loading.value = false;
  }
}

async function createPhoneApiKey() {
  creating.value = true;
  try {
    const api = useApi();
    await api("/v1/phone-api-keys", {
      method: "POST",
      body: { name: formName.value },
    });
    notificationsStore.addNotification({
      message: "Phone API Key created",
      type: "success",
    });
    createDialog.value = false;
    formName.value = "";
    await loadPhoneApiKeys();
  } catch {
    notificationsStore.addNotification({
      message: "Failed to create Phone API Key",
      type: "error",
    });
  } finally {
    creating.value = false;
  }
}

async function deletePhoneApiKey(id: string) {
  try {
    const api = useApi();
    await api(`/v1/phone-api-keys/${id}`, { method: "DELETE" });
    notificationsStore.addNotification({
      message: "Phone API Key deleted",
      type: "success",
    });
    await loadPhoneApiKeys();
  } catch {
    notificationsStore.addNotification({
      message: "Failed to delete Phone API Key",
      type: "error",
    });
  }
}

async function copyApiKey(key: string) {
  await navigator.clipboard.writeText(key);
  notificationsStore.addNotification({
    message: "API Key copied to clipboard",
    type: "success",
  });
}

onMounted(async () => {
  await authStore.loadUser();
  await loadPhoneApiKeys();
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
          <VIcon :icon="mdiCellphoneKey" class="mr-2" />
          Phone API Keys
        </VToolbarTitle>
      </VAppBar>
      <VContainer class="mt-16">
        <VRow>
          <VCol cols="12" md="8" offset-md="2">
            <h4 class="text-h4 mb-3">Phone API Keys</h4>
            <p class="text-medium-emphasis">
              Create separate API keys for each phone. This allows you to manage
              multiple phones independently and securely.
            </p>
            <VProgressLinear v-if="loading" indeterminate class="mb-4" />
            <VTable v-else-if="phoneApiKeys.length" class="mb-4">
              <thead>
                <tr class="text-uppercase text-subtitle-2">
                  <th class="text-left">Name</th>
                  <th class="text-left">API Key</th>
                  <th class="text-center">Created</th>
                  <th class="text-center">Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="key in phoneApiKeys" :key="key.id">
                  <td>{{ key.name }}</td>
                  <td class="text-break">
                    {{ key.api_key?.substring(0, 12) }}...
                  </td>
                  <td class="text-center">
                    {{ useFilters().timestamp(key.created_at) }}
                  </td>
                  <td class="text-center">
                    <VBtn icon size="small" @click="copyApiKey(key.api_key)">
                      <VIcon size="small" :icon="mdiContentCopy" />
                    </VBtn>
                    <VBtn
                      icon
                      size="small"
                      color="error"
                      @click="deletePhoneApiKey(key.id)"
                    >
                      <VIcon size="small" :icon="mdiDelete" />
                    </VBtn>
                  </td>
                </tr>
              </tbody>
            </VTable>
            <VBtn color="primary" @click="createDialog = true">
              <VIcon start :icon="mdiPlus" />
              Create Phone API Key
            </VBtn>

            <VDialog v-model="createDialog" max-width="500">
              <VCard>
                <VCardTitle>Create Phone API Key</VCardTitle>
                <VCardText>
                  <VTextField
                    v-model="formName"
                    variant="outlined"
                    label="Name"
                    placeholder="My Phone"
                  />
                </VCardText>
                <VCardActions>
                  <VBtn
                    color="primary"
                    :loading="creating"
                    @click="createPhoneApiKey"
                    >Create</VBtn
                  >
                  <VSpacer />
                  <VBtn variant="text" @click="createDialog = false"
                    >Cancel</VBtn
                  >
                </VCardActions>
              </VCard>
            </VDialog>
          </VCol>
        </VRow>
      </VContainer>
    </div>
  </VContainer>
</template>
