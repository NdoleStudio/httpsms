<script setup lang="ts">
import { useDisplay } from "vuetify";
import { getAuth, signOut } from "firebase/auth";
import {
  mdiPlus,
  mdiAccountCog,
  mdiLogout,
  mdiCellphoneKey,
  mdiDownload,
  mdiFinance,
  mdiBatteryChargingHigh,
  mdiPackageUp,
  mdiPackageDown,
  mdiDotsVertical,
  mdiMagnify,
  mdiCommentTextMultipleOutline,
  mdiCircle,
} from "@mdi/js";
import { formatPhoneNumber, phoneCountry, humanizeTime } from "~/utils/filters";
import type { EntitiesPhone } from "~~/shared/types/api";

const router = useRouter();
const route = useRoute();
const { mdAndDown, mdAndUp, lgAndUp } = useDisplay();
const authStore = useAuthStore();
const phonesStore = usePhonesStore();
const threadsStore = useThreadsStore();
const appStore = useAppStore();
const notificationsStore = useNotificationsStore();

const selectedMenuItem = ref(-1);

interface SelectItem {
  title: string;
  value: string;
}

const owners = computed<SelectItem[]>(() => {
  return phonesStore.phones.map((phone: EntitiesPhone) => ({
    title: formatPhoneNumber(phone.phone_number),
    value: phone.phone_number,
  }));
});

async function onOwnerChanged(owner: string) {
  await authStore.updateUser({ owner });
  if (route.name !== "threads") {
    threadsStore.setThreadId(null);
    await router.push({ name: "threads" });
    return;
  }
  await threadsStore.loadThreads();
}

async function toggleArchive() {
  threadsStore.toggleArchive();
  setTimeout(() => {
    selectedMenuItem.value = -1;
  }, 1000);
  if (route.name !== "threads") {
    threadsStore.setThreadId(null);
    await router.push({ name: "threads" });
    return;
  }
  await threadsStore.loadThreads();
}

async function logout() {
  const auth = getAuth();
  await signOut(auth);
  authStore.resetState();
  phonesStore.resetState();
  threadsStore.resetState();
  notificationsStore.addNotification({
    type: "info",
    message: "You have successfully logged out",
  });
  router.push({ name: "index" });
}
</script>

<template>
  <v-sheet
    class="pa-4 d-flex"
    :elevation="lgAndUp ? 0 : 2"
    color="black"
  >
    <div :class="{ 'px-2': mdAndDown }">
      <v-toolbar-title>
        <div class="d-flex pt-2" style="width: 245px">
          <v-select
            variant="outlined"
            density="compact"
            :disabled="owners.length === 0"
            placeholder="Phone Numbers"
            :class="{ 'mb-n5': !phonesStore.owner }"
            :items="owners"
            :model-value="phonesStore.owner"
            @update:model-value="onOwnerChanged"
          />
          <div style="width: 50px">
            <v-progress-circular
              v-if="appStore.polling"
              indeterminate
              :size="20"
              :width="1"
              class="mt-3 ml-2"
              color="success"
            />
          </div>
        </div>
      </v-toolbar-title>
      <div v-if="phonesStore.owner" class="d-flex mt-n4">
        <p class="text-medium-emphasis mb-n1">
          {{ phoneCountry(phonesStore.owner) }}
        </p>
        <v-tooltip v-if="phonesStore.heartbeat" location="end">
          <template #activator="{ props: tooltipProps }">
            <v-btn
              v-bind="tooltipProps"
              size="x-small"
              :to="{
                name: 'heartbeats-id',
                params: { id: phonesStore.owner },
              }"
              color="success"
              class="ml-2 mt-1 mb-n1"
              icon
            >
              <v-icon
                v-if="phonesStore.heartbeat.charging"
                size="small"
                class="mt-n1"
                :icon="mdiBatteryChargingHigh"
              />
              <v-icon v-else size="x-small" :icon="mdiCircle" />
            </v-btn>
          </template>
          <h4>Last Heartbeat</h4>
          {{ humanizeTime(phonesStore.heartbeat.timestamp) }} ago
        </v-tooltip>
      </div>
    </div>
    <v-spacer />
    <v-menu>
      <template #activator="{ props: menuProps }">
        <v-btn v-bind="menuProps" icon variant="text" class="mt-2">
          <v-icon :icon="mdiDotsVertical" />
        </v-btn>
      </template>
      <v-list class="pa-0" :density="mdAndDown ? 'compact' : 'default'" prepend-gap="16">
        <v-list-item @click.prevent="toggleArchive">
          <template #prepend>
            <v-icon
              v-if="!threadsStore.archivedThreads"
              :icon="mdiPackageDown"
            />
            <v-icon v-else :icon="mdiPackageUp" />
          </template>
          <v-list-item-title>
            {{ threadsStore.archivedThreads ? "Unarchived" : "Archived" }}
          </v-list-item-title>
        </v-list-item>
        <v-list-item v-if="phonesStore.owner" :to="{ name: 'messages' }">
          <template #prepend><v-icon :icon="mdiPlus" /></template>
          <v-list-item-title>New Message</v-list-item-title>
        </v-list-item>
        <v-list-item v-if="phonesStore.owner" :to="{ name: 'bulk-messages' }">
          <template #prepend
            ><v-icon :icon="mdiCommentTextMultipleOutline"
          /></template>
          <v-list-item-title>Bulk Messages</v-list-item-title>
        </v-list-item>
        <v-list-item v-if="phonesStore.owner" :to="{ name: 'search-messages' }">
          <template #prepend><v-icon :icon="mdiMagnify" /></template>
          <v-list-item-title>Search Messages</v-list-item-title>
        </v-list-item>
        <v-list-item :to="{ name: 'settings' }">
          <template #prepend><v-icon :icon="mdiAccountCog" /></template>
          <v-list-item-title>Settings</v-list-item-title>
        </v-list-item>
        <v-list-item :to="{ name: 'phone-api-keys' }">
          <template #prepend><v-icon :icon="mdiCellphoneKey" /></template>
          <v-list-item-title :class="{'pr-16': lgAndUp}">Phone API Keys</v-list-item-title>
        </v-list-item>
        <v-list-item
          v-if="phonesStore.owner"
          :href="appStore.appData.appDownloadUrl"
        >
          <template #prepend><v-icon :icon="mdiDownload" /></template>
          <v-list-item-title>Install App</v-list-item-title>
        </v-list-item>
        <v-list-item :to="{ name: 'billing' }">
          <template #prepend><v-icon :icon="mdiFinance" /></template>
          <v-list-item-title>Usage & Billing</v-list-item-title>
        </v-list-item>
        <v-list-item @click.prevent="logout">
          <template #prepend><v-icon :icon="mdiLogout" /></template>
          <v-list-item-title>Logout</v-list-item-title>
        </v-list-item>
      </v-list>
    </v-menu>
  </v-sheet>
</template>
