<script setup lang="ts">
import { mdiArrowLeft, mdiMicrosoftExcel, mdiSendCheck } from "@mdi/js";
import { ErrorMessages } from "~/utils/errors";
import capitalize from "~/utils/capitalize";

definePageMeta({
  middleware: ["auth"],
});

useHead({
  title: "Send Bulk Messages - httpSMS",
});

const router = useRouter();
const { mdAndUp, mdAndDown } = useDisplay();
const authStore = useAuthStore();
const notificationsStore = useNotificationsStore();
const { formatTimestamp } = useFilters();
const { useApi } = useApiComposable();

const formFile = ref<File | null>(null);
const loading = ref(true);
const loadingHistory = ref(true);
const errorTitle = ref("");
const errorMessages = ref(new ErrorMessages());
const bulkOrders = ref<any[]>([]);

function parseErrors(error: any): ErrorMessages {
  const bag = new ErrorMessages();
  const data = error?.data?.data;
  if (data && typeof data === "object") {
    Object.keys(data).forEach((key) => bag.addMany(key, data[key]));
  }
  return bag;
}

function cleanName(requestId: string): string {
  if (requestId.startsWith("bulk-csv-"))
    return requestId.replace(/^bulk-csv-/, "") + ".csv";
  if (requestId.startsWith("bulk-xls-"))
    return requestId.replace(/^bulk-xls-/, "") + ".xlsx";
  const newFormatMatch = requestId.match(/^bulk-[0-9A-Za-z]+-(.+)$/);
  if (newFormatMatch) return newFormatMatch[1];
  return requestId.replace(/^bulk-/, "");
}

async function fetchBulkOrders() {
  loadingHistory.value = true;
  try {
    const api = useApi();
    const response = await api<{ data: any[] }>("/v1/bulk-messages", {
      method: "GET",
    });
    bulkOrders.value = response.data ?? [];
  } catch {
    notificationsStore.addNotification({
      message: "Error while fetching bulk messages history",
      type: "error",
    });
  } finally {
    loadingHistory.value = false;
  }
}

async function sendBulkMessages() {
  loading.value = true;
  errorMessages.value = new ErrorMessages();
  errorTitle.value = "";

  try {
    const api = useApi();
    const formData = new FormData();
    if (formFile.value) formData.append("document", formFile.value);
    const response = await api<{ message?: string }>("/v1/bulk-messages", {
      method: "POST",
      body: formData,
    });
    notificationsStore.addNotification({
      message: response?.message ?? "Bulk messages sent successfully",
      type: "success",
    });
    loading.value = false;
    formFile.value = null;
    fetchBulkOrders();
  } catch (error: any) {
    errorTitle.value = capitalize(
      error?.data?.message ?? "Error while sending bulk messages",
    );
    errorMessages.value = parseErrors(error);
    notificationsStore.addNotification({
      message: error?.data?.message ?? "Errors while sending bulk messages",
      type: "error",
    });
    loading.value = false;
  }
}

onMounted(async () => {
  await authStore.loadUser();
  loading.value = false;
  fetchBulkOrders();
});
</script>

<template>
  <VContainer fluid class="px-0 pt-0" :class="{ 'fill-height': true }">
    <div class="w-100 h-100">
      <VAppBar height="60" :density="mdAndDown ? 'compact' : 'default'">
        <VBtn icon to="/threads">
          <VIcon :icon="mdiArrowLeft" />
        </VBtn>
        <VToolbarTitle>
          <div class="py-16">Bulk Messages</div>
        </VToolbarTitle>
        <VProgressLinear
          :active="loading"
          :indeterminate="loading"
          location="bottom"
          absolute
        />
      </VAppBar>
      <VContainer>
        <VRow>
          <VCol cols="12" md="10" offset-md="1" xxl="8" offset-xxl="2">
            <h5 class="text-headline-large mb-3 mt-3">Bulk Messages</h5>
            <p>
              Fill in our bulk SMS
              <a
                class="text-decoration-none"
                download
                href="/templates/httpsms-bulk.csv"
                >CSV template</a
              >
              or our
              <a
                class="text-decoration-none"
                download
                href="/templates/httpsms-bulk.xlsx"
                >Excel template</a
              >
              and upload it here to send your SMS messages to multiple
              recipients at once. You can also configure
              <NuxtLink
                class="text-decoration-none"
                to="/settings/#send-schedules"
                >send schedules</NuxtLink
              >
              on your phone to make sure messages are sent out at specific times
              of the day e.g
              <span class="text-medium-emphasis">Mon - Fri 9am - 5pm.</span>
            </p>
            <VAlert v-if="errorTitle" variant="tonal" type="warning" prominent>
              <h6 class="text-title-large font-weight-bold">
                {{ errorTitle }}
              </h6>
              <ul class="text-body-medium">
                <li
                  v-for="message in errorMessages.get('document')"
                  :key="message"
                >
                  {{ message }}
                </li>
              </ul>
            </VAlert>
            <form @submit.prevent="sendBulkMessages">
              <VFileInput
                v-model="formFile"
                label="File"
                color="primary"
                accept=".csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
                :error-messages="errorMessages.get('document')"
                :append-inner-icon="mdiMicrosoftExcel"
                variant="outlined"
              />
              <div class="d-flex">
                <VBtn
                  color="primary"
                  type="submit"
                  :loading="loading"
                  :disabled="loading"
                  size="large"
                >
                  <VIcon start :icon="mdiSendCheck" />
                  Send Bulk Messages
                </VBtn>
                <VSpacer />
                <VBtn
                  v-if="mdAndUp"
                  variant="plain"
                  color="info"
                  href="mailto:arnold@httpsms.com?subject=I'm having trouble with the bulk messages"
                >
                  I Need Help
                </VBtn>
              </div>
            </form>
          </VCol>
        </VRow>
        <VRow class="mt-8">
          <VCol cols="12" md="10" offset-md="1" xxl="8" offset-xxl="2">
            <h4 class="text-headline-large mb-3">Bulk Message History</h4>
            <p class="text-medium-emphasis">
              Your 10 most recent bulk SMS uploads are shown below, including a
              delivery status breakdown for each batch. Click on a row to see
              individual messages on the search page.
            </p>
            <VProgressLinear v-if="loadingHistory" indeterminate class="mb-4" />
            <VTable density="comfortable" v-else>
              <thead>
                <tr class="text-uppercase text-medium-emphasis">
                  <th class="text-left">Name</th>
                  <th class="text-center">Created At</th>
                  <th class="text-center">Total</th>
                  <th class="text-center">Pending</th>
                  <th class="text-center">Scheduled</th>
                  <th class="text-center">Sent</th>
                  <th class="text-center">Delivered</th>
                  <th class="text-center">Failed</th>
                  <th class="text-center">Expired</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="order in bulkOrders"
                  :key="order.request_id"
                  class="clickable-row"
                  @click="
                    router.push(`/search-messages?query=${order.request_id}`)
                  "
                >
                  <td class="text-left">{{ cleanName(order.request_id) }}</td>
                  <td class="text-center">
                    {{ formatTimestamp(order.created_at) }}
                  </td>
                  <td class="text-center">{{ order.total }}</td>
                  <td class="text-center">{{ order.pending_count }}</td>
                  <td class="text-center">{{ order.scheduled_count }}</td>
                  <td class="text-center">{{ order.sent_count }}</td>
                  <td class="text-center">{{ order.delivered_count }}</td>
                  <td class="text-center">{{ order.failed_count }}</td>
                  <td class="text-center">{{ order.expired_count }}</td>
                </tr>
              </tbody>
            </VTable>
          </VCol>
        </VRow>
      </VContainer>
    </div>
  </VContainer>
</template>

<style scoped>
.clickable-row {
  cursor: pointer;
}
.clickable-row:hover {
  background-color: rgb(0 0 0 / 4%);
}
</style>
