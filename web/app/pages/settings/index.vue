<script setup lang="ts">
import {
  mdiArrowLeft,
  mdiAccountCircle,
  mdiShieldCheck,
  mdiEye,
  mdiEyeOff,
  mdiQrcode,
  mdiRefresh,
  mdiLinkVariant,
  mdiSquareEditOutline,
  mdiDelete,
  mdiContentSave,
} from "@mdi/js";
import QRCode from "qrcode";

definePageMeta({
  middleware: ["auth"],
});

useHead({
  title: "Settings - httpSMS",
});

const config = useRuntimeConfig();
const { mdAndDown, mdAndUp, lgAndUp, xlAndUp } = useDisplay();
const authStore = useAuthStore();
const billingStore = useBillingStore();
const notificationsStore = useNotificationsStore();
const { useApi } = useApiComposable();

const apiKey = ref("");
const apiKeyShow = ref(false);
const showQrCodeDialog = ref(false);
const showRotateApiKey = ref(false);
const rotatingApiKey = ref(false);
const qrCodeCanvas = ref<HTMLCanvasElement | null>(null);

// Webhooks
const loadingWebhooks = ref(true);
const webhooks = ref<any[]>([]);
const updatingWebhook = ref(false);
const webhookDialog = ref(false);
const webhookForm = ref({
  id: "",
  url: "",
  events: [] as string[],
  phoneId: "",
});
const webhookEvents = [
  "message.phone.received",
  "message.phone.sent",
  "message.send.failed",
  "message.send.expired",
  "phone.heartbeat.offline",
];

// Discord
const loadingDiscordIntegrations = ref(true);
const discords = ref<any[]>([]);
const updatingDiscord = ref(false);
const discordDialog = ref(false);
const discordForm = ref({
  id: "",
  name: "",
  server_id: "",
  incoming_channel_id: "",
});

// Send Schedules
const loadingSendSchedules = ref(true);
const sendSchedules = ref<any[]>([]);

// Timezones
const timezones = Intl.supportedValuesOf("timeZone");

async function loadApiKey() {
  const key = await authStore.loadApiKey();
  apiKey.value = key;
}

async function rotateApiKey() {
  rotatingApiKey.value = true;
  try {
    const api = useApi();
    const response = await api<{ data: { api_key: string } }>(
      "/v1/users/api-keys",
      { method: "PUT" },
    );
    apiKey.value = response.data.api_key;
    showRotateApiKey.value = false;
    notificationsStore.addNotification({
      message: "API Key rotated successfully",
      type: "success",
    });
  } catch {
    notificationsStore.addNotification({
      message: "Failed to rotate API Key",
      type: "error",
    });
  } finally {
    rotatingApiKey.value = false;
  }
}

function generateQrCode() {
  showQrCodeDialog.value = true;
  nextTick(() => {
    if (qrCodeCanvas.value) {
      QRCode.toCanvas(qrCodeCanvas.value, apiKey.value, {
        width: 300,
        margin: 2,
      });
    }
  });
}

async function loadWebhooks() {
  loadingWebhooks.value = true;
  try {
    const api = useApi();
    const response = await api<{ data: any[] }>("/v1/webhooks");
    webhooks.value = response.data ?? [];
  } finally {
    loadingWebhooks.value = false;
  }
}

async function saveWebhook() {
  updatingWebhook.value = true;
  try {
    const api = useApi();
    if (webhookForm.value.id) {
      await api(`/v1/webhooks/${webhookForm.value.id}`, {
        method: "PUT",
        body: { url: webhookForm.value.url, events: webhookForm.value.events },
      });
    } else {
      await api("/v1/webhooks", {
        method: "POST",
        body: { url: webhookForm.value.url, events: webhookForm.value.events },
      });
    }
    webhookDialog.value = false;
    notificationsStore.addNotification({
      message: "Webhook saved",
      type: "success",
    });
    await loadWebhooks();
  } catch {
    notificationsStore.addNotification({
      message: "Failed to save webhook",
      type: "error",
    });
  } finally {
    updatingWebhook.value = false;
  }
}

async function deleteWebhook(id: string) {
  try {
    const api = useApi();
    await api(`/v1/webhooks/${id}`, { method: "DELETE" });
    notificationsStore.addNotification({
      message: "Webhook deleted",
      type: "success",
    });
    await loadWebhooks();
  } catch {
    notificationsStore.addNotification({
      message: "Failed to delete webhook",
      type: "error",
    });
  }
}

function onWebhookCreate() {
  webhookForm.value = { id: "", url: "", events: [], phoneId: "" };
  webhookDialog.value = true;
}

function onWebhookEdit(id: string) {
  const webhook = webhooks.value.find((w) => w.id === id);
  if (webhook) {
    webhookForm.value = {
      id: webhook.id,
      url: webhook.url,
      events: webhook.events ?? [],
      phoneId: "",
    };
    webhookDialog.value = true;
  }
}

async function loadDiscord() {
  loadingDiscordIntegrations.value = true;
  try {
    const api = useApi();
    const response = await api<{ data: any[] }>("/v1/discord-integrations");
    discords.value = response.data ?? [];
  } finally {
    loadingDiscordIntegrations.value = false;
  }
}

async function loadSendSchedules() {
  loadingSendSchedules.value = true;
  try {
    const api = useApi();
    const response = await api<{ data: any[] }>("/v1/message-send-schedules");
    sendSchedules.value = response.data ?? [];
  } finally {
    loadingSendSchedules.value = false;
  }
}

async function updateTimezone(timezone: string) {
  try {
    const api = useApi();
    await api("/v1/users", { method: "PUT", body: { timezone } });
    notificationsStore.addNotification({
      message: "Timezone updated",
      type: "success",
    });
  } catch {
    notificationsStore.addNotification({
      message: "Failed to update timezone",
      type: "error",
    });
  }
}

onMounted(async () => {
  await authStore.loadUser();
  await loadApiKey();
  loadWebhooks();
  loadDiscord();
  loadSendSchedules();
});
</script>

<template>
  <VContainer fluid class="px-0 pt-0" :class="{ 'fill-height': lgAndUp }">
    <div class="w-100 h-100">
      <VAppBar height="60" :density="mdAndDown ? 'compact' : 'default'">
        <VBtn icon to="/threads">
          <VIcon :icon="mdiArrowLeft" />
        </VBtn>
        <VToolbarTitle>Settings</VToolbarTitle>
      </VAppBar>
      <VContainer class="mt-16">
        <VRow>
          <VCol cols="12" md="9" offset-md="1" xl="8" offset-xl="2">
            <!-- Profile -->
            <div v-if="authStore.firebaseUser" class="text-center">
              <VAvatar size="100" color="indigo" class="mx-auto">
                <img
                  v-if="authStore.firebaseUser.photoURL"
                  :src="authStore.firebaseUser.photoURL"
                  :alt="authStore.firebaseUser.displayName ?? ''"
                />
                <VIcon v-else size="70" :icon="mdiAccountCircle" />
              </VAvatar>
              <h3 v-if="authStore.firebaseUser.displayName">
                {{ authStore.firebaseUser.displayName }}
              </h3>
              <h4 class="text-medium-emphasis">
                {{ authStore.firebaseUser.email }}
                <VIcon
                  v-if="authStore.firebaseUser.emailVerified"
                  size="small"
                  color="primary"
                  :icon="mdiShieldCheck"
                />
              </h4>
              <VAutocomplete
                v-if="authStore.user"
                density="compact"
                variant="outlined"
                :model-value="authStore.user.timezone"
                class="mx-auto mt-2"
                style="max-width: 250px"
                label="Timezone"
                :items="timezones"
                @update:model-value="updateTimezone"
              />
            </div>

            <!-- API Key -->
            <h5 class="text-headline-large mb-3 mt-3">API Key</h5>
            <p class="text-medium-emphasis">
              Use your API Key in the <code>x-api-key</code> HTTP Header when
              sending requests to
              <code>https://api.httpsms.com</code> endpoints.
            </p>
            <div v-if="apiKey === ''" class="mb-n9 pl-3 pt-5">
              <VProgressCircular
                :size="20"
                :width="2"
                color="primary"
                indeterminate
              />
            </div>
            <VTextField
              v-else
              :append-inner-icon="apiKeyShow ? mdiEye : mdiEyeOff"
              :type="apiKeyShow ? 'text' : 'password'"
              :model-value="apiKey"
              readonly
              name="api-key"
              variant="outlined"
              class="mb-n2"
              @click:append-inner="apiKeyShow = !apiKeyShow"
            />
            <div class="d-flex flex-wrap">
              <CopyButton
                :value="apiKey"
                color="primary"
                copy-text="Copy API Key"
                notification-text="API Key copied successfully"
              />
              <VBtn
                v-if="mdAndUp"
                color="primary"
                class="ml-4"
                @click="generateQrCode"
              >
                <VIcon start :icon="mdiQrcode" />
                Show QR Code
              </VBtn>
              <VDialog v-model="showQrCodeDialog" max-width="400px">
                <VCard>
                  <VCardTitle class="text-center">API Key QR Code</VCardTitle>
                  <VCardSubtitle class="mt-2 text-center">
                    Scan this QR code with the
                    <a :href="config.public.appDownloadUrl">httpSMS app</a>
                    on your Android phone to login.
                  </VCardSubtitle>
                  <VCardText class="text-center">
                    <canvas ref="qrCodeCanvas" />
                  </VCardText>
                  <VCardActions>
                    <VBtn
                      color="primary"
                      block
                      class="mb-4"
                      @click="showQrCodeDialog = false"
                      >Close</VBtn
                    >
                  </VCardActions>
                </VCard>
              </VDialog>
              <VBtn
                v-if="lgAndUp"
                class="ml-4"
                :href="config.public.appDocumentationUrl"
                >Documentation</VBtn
              >
              <VSpacer />
              <VDialog v-model="showRotateApiKey" max-width="550">
                <template #activator="{ props }">
                  <VBtn
                    :size="mdAndDown ? 'small' : 'default'"
                    :variant="lgAndUp ? 'text' : 'elevated'"
                    color="warning"
                    v-bind="props"
                  >
                    <VIcon start :icon="mdiRefresh" />
                    Rotate API Key
                  </VBtn>
                </template>
                <VCard>
                  <VCardTitle class="text-headline-medium text-break"
                    >Are you sure you want to rotate your API Key?</VCardTitle
                  >
                  <VCardText>
                    You will have to logout and login again on the
                    <b>httpSMS</b> Android app with your new API key after you
                    rotate it.
                  </VCardText>
                  <VCardActions class="pb-4">
                    <VBtn
                      color="primary"
                      :loading="rotatingApiKey"
                      @click="rotateApiKey"
                    >
                      <VIcon start :icon="mdiRefresh" />
                      Yes Rotate Key
                    </VBtn>
                    <VSpacer />
                    <VBtn variant="text" @click="showRotateApiKey = false"
                      >Close</VBtn
                    >
                  </VCardActions>
                </VCard>
              </VDialog>
            </div>

            <!-- Webhooks -->
            <h5 id="webhook-settings" class="text-headline-large mb-3 mt-12">
              Webhooks
            </h5>
            <p class="text-medium-emphasis">
              Webhooks allow us to send events to your server for example when
              the android phone receives an SMS message we can forward the
              message to your server.
            </p>
            <div v-if="loadingWebhooks">
              <VProgressCircular
                :size="60"
                :width="2"
                color="primary"
                class="mb-4"
                indeterminate
              />
            </div>
            <VTable v-else-if="webhooks.length" class="mb-4">
              <thead>
                <tr class="text-uppercase text-title-medium">
                  <th v-if="xlAndUp" class="text-left">ID</th>
                  <th class="text-left text-break">Callback URL</th>
                  <th v-if="lgAndUp" class="text-center">Events</th>
                  <th class="text-center">Action</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="webhook in webhooks" :key="webhook.id">
                  <td v-if="xlAndUp" class="text-left">{{ webhook.id }}</td>
                  <td class="text-break">{{ webhook.url }}</td>
                  <td v-if="lgAndUp" class="text-center">
                    <VChip
                      v-for="event in webhook.events"
                      :key="event"
                      class="ma-1"
                      size="small"
                      >{{ event }}</VChip
                    >
                  </td>
                  <td class="text-center">
                    <VBtn
                      :icon="mdAndDown"
                      size="small"
                      color="info"
                      :disabled="updatingWebhook"
                      @click.prevent="onWebhookEdit(webhook.id)"
                    >
                      <VIcon size="small" :icon="mdiSquareEditOutline" />
                      <span v-if="!mdAndDown" class="ml-1">Edit</span>
                    </VBtn>
                  </td>
                </tr>
              </tbody>
            </VTable>
            <div class="d-flex">
              <VBtn color="primary" @click="onWebhookCreate">
                <VIcon start :icon="mdiLinkVariant" />
                Add webhook
              </VBtn>
              <VBtn
                v-if="lgAndUp"
                class="ml-4"
                href="https://docs.httpsms.com/webhooks/introduction"
                >Documentation</VBtn
              >
            </div>

            <!-- Webhook Dialog -->
            <VDialog v-model="webhookDialog" max-width="600">
              <VCard>
                <VCardTitle
                  >{{ webhookForm.id ? "Edit" : "Add" }} Webhook</VCardTitle
                >
                <VCardText>
                  <VTextField
                    v-model="webhookForm.url"
                    variant="outlined"
                    label="Callback URL"
                    placeholder="https://example.com/webhook"
                  />
                  <VSelect
                    v-model="webhookForm.events"
                    :items="webhookEvents"
                    variant="outlined"
                    label="Events"
                    multiple
                    chips
                  />
                </VCardText>
                <VCardActions>
                  <VBtn
                    color="primary"
                    :loading="updatingWebhook"
                    @click="saveWebhook"
                  >
                    <VIcon start :icon="mdiContentSave" />
                    Save
                  </VBtn>
                  <VSpacer />
                  <VBtn
                    v-if="webhookForm.id"
                    color="error"
                    variant="text"
                    @click="deleteWebhook(webhookForm.id)"
                  >
                    <VIcon start :icon="mdiDelete" />
                    Delete
                  </VBtn>
                  <VBtn variant="text" @click="webhookDialog = false"
                    >Close</VBtn
                  >
                </VCardActions>
              </VCard>
            </VDialog>

            <!-- Discord Integration -->
            <h5 id="discord-settings" class="text-headline-large mb-3 mt-12">
              Discord Integration
            </h5>
            <p class="text-medium-emphasis">
              Send and receive SMS messages without leaving your discord server
              with the httpSMS discord app using the
              <code>/httpsms</code> command.
            </p>
            <div v-if="loadingDiscordIntegrations">
              <VProgressCircular
                :size="60"
                :width="2"
                color="primary"
                class="mb-4"
                indeterminate
              />
            </div>
            <VTable v-else-if="discords.length" class="mb-4">
              <thead>
                <tr class="text-uppercase text-title-medium">
                  <th class="text-left">Name</th>
                  <th class="text-left">Server ID</th>
                  <th class="text-left">Channel ID</th>
                  <th class="text-center">Action</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="discord in discords" :key="discord.id">
                  <td class="text-left">{{ discord.name }}</td>
                  <td class="text-left">{{ discord.server_id }}</td>
                  <td class="text-left">{{ discord.incoming_channel_id }}</td>
                  <td class="text-center">
                    <VBtn size="small" color="info">
                      <VIcon size="small" :icon="mdiSquareEditOutline" />
                      <span v-if="!mdAndDown" class="ml-1">Edit</span>
                    </VBtn>
                  </td>
                </tr>
              </tbody>
            </VTable>
            <VBtn
              color="primary"
              href="https://discord.com/api/oauth2/authorize?client_id=1049492914743599124&permissions=2048&scope=bot%20applications.commands"
            >
              Add Discord Integration
            </VBtn>

            <!-- Send Schedules -->
            <h5 id="send-schedules" class="text-headline-large mb-3 mt-12">
              Send Schedules
            </h5>
            <p class="text-medium-emphasis">
              Configure when your phone should send SMS messages. Messages sent
              outside the schedule will be queued and sent during the next
              active window.
            </p>
            <div v-if="loadingSendSchedules">
              <VProgressCircular
                :size="60"
                :width="2"
                color="primary"
                class="mb-4"
                indeterminate
              />
            </div>
            <VTable v-else-if="sendSchedules.length" class="mb-4">
              <thead>
                <tr class="text-uppercase text-title-medium">
                  <th class="text-left">Phone</th>
                  <th class="text-center">Days</th>
                  <th class="text-center">Start</th>
                  <th class="text-center">End</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="schedule in sendSchedules" :key="schedule.id">
                  <td>{{ useFilters().phoneNumber(schedule.phone_number) }}</td>
                  <td class="text-center">{{ schedule.days?.join(", ") }}</td>
                  <td class="text-center">{{ schedule.start_time }}</td>
                  <td class="text-center">{{ schedule.end_time }}</td>
                </tr>
              </tbody>
            </VTable>
          </VCol>
        </VRow>
      </VContainer>
    </div>
  </VContainer>
</template>
