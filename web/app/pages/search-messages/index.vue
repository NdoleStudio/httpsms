<script setup lang="ts">
import {
  mdiAlert,
  mdiArrowLeft,
  mdiCallMade,
  mdiCallMissed,
  mdiCallReceived,
  mdiCheck,
  mdiCheckAll,
  mdiDelete,
  mdiExport,
  mdiMagnify,
  mdiProgressCheck,
  mdiRefresh,
} from "@mdi/js";
import type { EntitiesMessage, EntitiesPhone } from "~~/shared/types/api";
import type { SearchMessagesRequest } from "~~/shared/types/message";
import { ErrorMessages } from "~/utils/errors";

interface Turnstile {
  ready(callback: () => void): void;
  render(
    container: string | HTMLElement,
    params?: {
      sitekey: string;
      action: string;
      callback?: (token: string) => void;
      "error-callback"?: (error: string) => void;
    },
  ): string | null | undefined;
  remove(widgetId?: string): void;
}

definePageMeta({
  middleware: ["auth"],
});

useHead({
  title: "Search your Messages - httpSMS",
});

const route = useRoute();
const config = useRuntimeConfig();
const { mdAndDown, mdAndUp, smAndDown, lgAndUp } = useDisplay();
const messagesStore = useMessagesStore();
const phonesStore = usePhonesStore();
const authStore = useAuthStore();
const notificationsStore = useNotificationsStore();
const { useApi } = useApiComposable();
const { formatPhoneNumber, formatTimestamp, capitalize } = useFilters();

const loading = ref(true);
const initialLoadComplete = ref(false);
const errorTitle = ref("");
const showDeleteDialog = ref(false);
const showResendDialog = ref(false);
const errorMessages = ref(new ErrorMessages());

const formOwners = ref<string[]>([]);
const formTypes = ref<string[]>([]);
const formStatuses = ref<string[]>([]);
const formQuery = ref("");

const messages = ref<EntitiesMessage[]>([]);
const totalMessages = ref(0);
const selectedIds = ref<string[]>([]);
let turnstileWidgetId: string | null = null;

const page = ref(1);
const itemsPerPage = ref(100);
const sortBy = ref<{ key: string; order: "asc" | "desc" }[]>([
  { key: "created_at", order: "desc" },
]);

const itemsPerPageOptions = [
  { value: 10, title: "10" },
  { value: 50, title: "50" },
  { value: 100, title: "100" },
  { value: 200, title: "200" },
];

const headers = [
  { title: "Created At", key: "created_at" },
  { title: "Owner", key: "owner" },
  { title: "Contact", key: "contact" },
  { title: "Message Type", key: "type" },
  { title: "Status", key: "status" },
  { title: "Message Content", key: "content", sortable: false },
];

const selectedMessages = computed<EntitiesMessage[]>(() =>
  messages.value.filter((message) => selectedIds.value.includes(message.id)),
);

const canResendSelected = computed<boolean>(
  () =>
    selectedMessages.value.length > 0 &&
    selectedMessages.value.every(
      (message) =>
        message.type === "mobile-terminated" &&
        (message.status === "expired" || message.status === "failed"),
    ),
);

const phoneNumberSelectItems = computed(() =>
  phonesStore.phones.map((phone: EntitiesPhone) => ({
    title: formatPhoneNumber(phone.phone_number),
    value: phone.phone_number,
  })),
);

const messageTypeSelectItems = [
  { title: "Outbound", value: "mobile-terminated" },
  { title: "Inbound", value: "mobile-originated" },
  { title: "Missed Calls", value: "call/missed" },
];

const messageStatusSelectItems = [
  { value: "pending", title: "Pending" },
  { value: "sent", title: "Sent" },
  { value: "delivered", title: "Delivered" },
  { value: "failed", title: "Failed" },
  { value: "expired", title: "Expired" },
  { value: "received", title: "Received" },
];

function getCaptcha(): Promise<string> {
  return new Promise<string>((resolve, reject) => {
    const turnstile: Turnstile = (
      window as unknown as { turnstile?: Turnstile }
    ).turnstile!;
    turnstile.ready(() => {
      if (turnstileWidgetId) {
        turnstile.remove(turnstileWidgetId);
        turnstileWidgetId = null;
      }

      turnstileWidgetId =
        turnstile.render("#cloudflare-turnstile", {
          sitekey: (config.public as Record<string, string>)
            .cloudflareTurnstileSiteKey!,
          action: "search_messages",
          callback: (token) => resolve(token),
          "error-callback": (error: string) => reject(error),
        }) ?? null;
    });
  });
}

function parseErrors(error: any): ErrorMessages {
  const bag = new ErrorMessages();
  const data = error?.data?.data;
  if (data && typeof data === "object") {
    Object.keys(data).forEach((key) => bag.addMany(key, data[key]));
  }
  return bag;
}

async function fetchMessages(reset = false) {
  loading.value = true;
  errorMessages.value = new ErrorMessages();
  errorTitle.value = "";

  if (reset) {
    page.value = 1;
  }

  try {
    const token = await getCaptcha();
    const sort = sortBy.value[0];
    const results = await messagesStore.searchMessages({
      token,
      owners: formOwners.value,
      types: formTypes.value,
      statuses: formStatuses.value,
      query: formQuery.value,
      sort_by: sort?.key ?? "created_at",
      sort_descending: sort ? sort.order === "desc" : true,
      skip: (page.value - 1) * itemsPerPage.value,
      limit: itemsPerPage.value,
    } as SearchMessagesRequest);

    messages.value = results;
    totalMessages.value =
      (page.value - 1) * itemsPerPage.value + results.length;
    if (results.length === itemsPerPage.value) {
      totalMessages.value += 1;
    }
  } catch (error: any) {
    errorTitle.value = capitalize(
      error?.data?.message ??
        "Error while searching messages. Contact us via email",
    );
    errorMessages.value = parseErrors(error);
  } finally {
    loading.value = false;
  }
}

function onUpdateOptions(options: {
  page: number;
  itemsPerPage: number;
  sortBy: { key: string; order: "asc" | "desc" }[];
}) {
  page.value = options.page;
  itemsPerPage.value = options.itemsPerPage;
  sortBy.value = options.sortBy.length
    ? options.sortBy
    : [{ key: "created_at", order: "desc" }];

  if (!initialLoadComplete.value) {
    return;
  }
  fetchMessages();
}

function sanitizeContent(content: string): string {
  content = content.replaceAll('"', '""');
  return content.includes(",") ? '"' + content + '"' : content;
}

function exportMessages() {
  let csvContent = "data:text/csv;charset=utf-8,";
  csvContent +=
    "Message ID,Created At,Owner,Contact,Message Type,Status,Message Content\n";
  selectedMessages.value.forEach((message) => {
    csvContent += `${message.id},${new Date(
      message.created_at,
    ).toLocaleString()},${message.owner},${message.contact},${message.type},${
      message.status
    },${sanitizeContent(message.content)}\n`;
  });

  const encodedUri = encodeURI(csvContent);
  const link = document.createElement("a");
  link.setAttribute("href", encodedUri);
  link.setAttribute(
    "download",
    `httpsms-${new Date().toJSON().slice(0, 10)}.csv`,
  );
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);

  notificationsStore.addNotification({
    message: "The selected messages have been exported successfully",
    type: "success",
  });
}

async function deleteMessages() {
  loading.value = true;
  try {
    await Promise.all(
      selectedMessages.value.map((message) =>
        messagesStore.deleteMessage(message.id),
      ),
    );
    notificationsStore.addNotification({
      message: "The selected messages have been deleted successfully",
      type: "success",
    });
    selectedIds.value = [];
  } catch {
    notificationsStore.addNotification({
      message: "Error while deleting the selected messages",
      type: "error",
    });
  } finally {
    loading.value = false;
    showDeleteDialog.value = false;
    fetchMessages();
  }
}

async function resendMessages() {
  loading.value = true;
  const api = useApi();
  try {
    const results = await Promise.allSettled(
      selectedMessages.value.map((message) =>
        api("/v1/messages/send", {
          method: "POST",
          body: {
            from: message.owner,
            to: message.contact,
            content: message.content,
            sim: message.sim,
            request_id: message.request_id,
          },
        }),
      ),
    );

    const failed = results.filter((r) => r.status === "rejected");
    if (failed.length === 0) {
      notificationsStore.addNotification({
        message: "The selected messages have been queued for resending",
        type: "success",
      });
      selectedIds.value = [];
    } else if (failed.length === results.length) {
      notificationsStore.addNotification({
        message: "Error while resending the selected messages",
        type: "error",
      });
    } else {
      notificationsStore.addNotification({
        message: `${results.length - failed.length} messages resent, ${
          failed.length
        } failed`,
        type: "info",
      });
      selectedIds.value = [];
    }
  } finally {
    loading.value = false;
    showResendDialog.value = false;
    fetchMessages();
  }
}

onMounted(async () => {
  await authStore.loadUser();
  await phonesStore.loadPhones();

  const queryParam = route.query.query;
  if (queryParam && typeof queryParam === "string") {
    formQuery.value = queryParam;
  }

  loading.value = false;
  initialLoadComplete.value = true;

  if (formQuery.value) {
    await fetchMessages(true);
  }
});

onBeforeUnmount(() => {
  const turnstile = (window as unknown as { turnstile?: Turnstile }).turnstile;
  if (turnstile && turnstileWidgetId) {
    turnstile.remove(turnstileWidgetId);
    turnstileWidgetId = null;
  }
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
          <div class="py-16">Search Messages</div>
        </VToolbarTitle>
        <VProgressLinear
          :active="loading"
          color="primary"
          :indeterminate="loading"
          location="bottom"
          absolute
        />
      </VAppBar>
      <VContainer>
        <VRow>
          <VCol cols="12">
            <h5 class="text-headline-large mb-3 mt-0">Search Messages</h5>
            <p>
              On this page, you can search all your messages by phone number,
              message type, and message status and even using the content of the
              SMS message. You will also be able to bulk delete messages and
              even export your messages in a CSV file.
            </p>
            <VAlert v-if="errorTitle" variant="tonal" prominent type="warning">
              <h6 class="text-title-large font-weight-bold">
                {{ errorTitle }}
              </h6>
            </VAlert>
          </VCol>
        </VRow>
        <VCard>
          <VCardText class="pt-4 pb-0">
            <VRow>
              <VCol cols="12" md="4">
                <VSelect
                  v-model="formOwners"
                  color="primary"
                  :error="errorMessages.has('owners')"
                  :error-messages="errorMessages.get('owners')"
                  :items="phoneNumberSelectItems"
                  multiple
                  density="compact"
                  label="Phone Numbers"
                  variant="outlined"
                />
              </VCol>
              <VCol cols="12" md="4">
                <VSelect
                  v-model="formTypes"
                  color="primary"
                  :error="errorMessages.has('types')"
                  :error-messages="errorMessages.get('types')"
                  :items="messageTypeSelectItems"
                  density="compact"
                  multiple
                  label="Message Types"
                  variant="outlined"
                />
              </VCol>
              <VCol cols="12" md="4">
                <VSelect
                  v-model="formStatuses"
                  color="primary"
                  :error="errorMessages.has('statuses')"
                  :error-messages="errorMessages.get('statuses')"
                  :items="messageStatusSelectItems"
                  density="compact"
                  multiple
                  label="Message Status"
                  variant="outlined"
                />
              </VCol>
            </VRow>
            <VRow class="mt-n3">
              <VCol cols="12" md="8">
                <VTextField
                  v-model="formQuery"
                  color="primary"
                  :error="errorMessages.has('query')"
                  :error-messages="errorMessages.get('query')"
                  label="Search Query"
                  variant="outlined"
                  density="compact"
                  clearable
                  @keyup.enter="fetchMessages(true)"
                />
              </VCol>
              <VCol cols="12" md="4">
                <div id="cloudflare-turnstile" class="d-none"></div>
                <VBtn
                  :loading="loading"
                  :disabled="loading"
                  color="primary"
                  class="py-5"
                  @click="fetchMessages(true)"
                >
                  <VIcon v-if="mdAndUp" start :icon="mdiMagnify" />
                  <span v-if="smAndDown">SEARCH</span>
                  <span v-else>Search Messages</span>
                </VBtn>
              </VCol>
            </VRow>
          </VCardText>
        </VCard>
        <VRow>
          <VCol cols="12" class="mt-16 mb-n2 d-flex align-baseline">
            <h2 class="text-headline-large mb-0">Search Results</h2>
            <VDialog v-model="showDeleteDialog" opacity="0.9" max-width="550">
              <template #activator="{ props }">
                <VBtn
                  :loading="loading"
                  :disabled="loading || selectedMessages.length < 1"
                  size="small"
                  class="ml-2"
                  color="error"
                  v-bind="props"
                >
                  <VIcon v-if="mdAndUp" start :icon="mdiDelete" />
                  <span v-if="smAndDown">DELETE</span>
                  <span v-else>Delete messages</span>
                </VBtn>
              </template>
              <VCard>
                <VCardTitle>
                  Delete <v-code>{{ selectedMessages.length }}</v-code> selected
                  messages?
                </VCardTitle>
                <VCardText class="text-medium-emphasis">
                  The messages will be deleted permanently from the httpSMS
                  server and cannot be recovered.
                </VCardText>
                <VCardActions class="pb-4">
                  <VBtn
                    color="error"
                    :loading="loading"
                    variant="flat"
                    @click="deleteMessages"
                  >
                    Delete Messages
                  </VBtn>
                  <VSpacer />
                  <VBtn color="warning" @click="showDeleteDialog = false">
                    Close
                  </VBtn>
                </VCardActions>
              </VCard>
            </VDialog>
            <VDialog v-model="showResendDialog" opacity="0.9" max-width="550">
              <template #activator="{ props }">
                <VBtn
                  :loading="loading"
                  :disabled="loading || !canResendSelected"
                  size="small"
                  class="ml-2 mt-2 d-none d-md-inline-flex"
                  v-bind="props"
                >
                  <VIcon start :icon="mdiRefresh" />
                  Resend Messages
                </VBtn>
              </template>
              <VCard>
                <VCardTitle class="text-headline-medium text-break">
                  Resend <v-code>{{ selectedMessages.length }}</v-code> selected
                  messages?
                </VCardTitle>
                <VCardText class="text-medium-emphasis">
                  The selected messages will be queued for sending again using
                  the original sender, recipient, and content.
                </VCardText>
                <VCardActions class="pb-4">
                  <VBtn
                    color="primary"
                    variant="flat"
                    :loading="loading"
                    @click="resendMessages"
                  >
                    Resend Messages
                  </VBtn>
                  <VSpacer />
                  <VBtn color="warning" @click="showResendDialog = false">
                    Close
                  </VBtn>
                </VCardActions>
              </VCard>
            </VDialog>
            <VSpacer />
            <VBtn
              :loading="loading"
              :disabled="loading || selectedMessages.length < 1"
              size="small"
              color="primary"
              class="mt-2"
              @click="exportMessages"
            >
              <VIcon v-if="mdAndUp" start :icon="mdiExport" />
              <span v-if="smAndDown">EXPORT</span>
              <span v-else>Export to CSV</span>
            </VBtn>
          </VCol>
          <VCol cols="12">
            <VDataTableServer
              v-model="selectedIds"
              v-model:items-per-page="itemsPerPage"
              v-model:page="page"
              v-model:sort-by="sortBy"
              color="primary"
              item-value="id"
              :headers="headers"
              :items="messages"
              :items-length="totalMessages"
              :items-per-page-options="itemsPerPageOptions"
              :loading="loading"
              show-select
              loading-text="Loading... Please wait"
              no-data-text="You don't have any messages yet"
              class="elevation-1"
              @update:options="onUpdateOptions"
            >
              <template #[`item.created_at`]="{ item }">
                {{ formatTimestamp(item.created_at) }}
              </template>
              <template #[`item.type`]="{ item }">
                <span v-if="item.type === 'call/missed'">
                  <VIcon size="small" color="error" :icon="mdiCallMissed" />
                  missed call
                </span>
                <span v-else-if="item.type === 'mobile-originated'">
                  <VIcon size="small" :icon="mdiCallReceived" />
                  inbound
                </span>
                <span v-else-if="item.type === 'mobile-terminated'">
                  <VIcon size="small" color="secondary" :icon="mdiCallMade" />
                  outbound
                </span>
              </template>
              <template #[`item.status`]="{ item }">
                <VChip
                  v-if="item.status === 'expired'"
                  color="warning"
                  size="small"
                  variant="outlined"
                >
                  <VIcon size="small" start :icon="mdiAlert" />
                  Expired
                </VChip>
                <VChip
                  v-else-if="item.status === 'delivered'"
                  color="primary"
                  size="small"
                  variant="outlined"
                >
                  <VIcon size="small" start :icon="mdiCheckAll" />
                  Delivered
                </VChip>
                <VChip
                  v-else-if="item.status === 'received'"
                  color="success"
                  size="small"
                  variant="outlined"
                >
                  <VIcon size="small" start :icon="mdiCheckAll" />
                  Received
                </VChip>
                <VChip
                  v-else-if="item.status === 'sent'"
                  size="small"
                  variant="outlined"
                >
                  <VIcon size="small" start :icon="mdiCheck" />
                  Sent
                </VChip>
                <VChip
                  v-else-if="item.status === 'failed'"
                  color="error"
                  size="small"
                  variant="outlined"
                >
                  <VIcon size="small" start :icon="mdiAlert" />
                  Failed
                </VChip>
                <VChip v-else size="small" color="cyan" variant="outlined">
                  <VIcon size="small" start :icon="mdiProgressCheck" />
                  {{ capitalize(item.status) }}
                </VChip>
              </template>
              <template #[`item.content`]="{ item }">
                <pre
                  style="
                    white-space: pre-wrap;
                    max-width: 300px;
                    word-break: break-all;
                  "
                  >{{ item.content }}</pre
                >
              </template>
            </VDataTableServer>
          </VCol>
        </VRow>
      </VContainer>
    </div>
  </VContainer>
</template>
