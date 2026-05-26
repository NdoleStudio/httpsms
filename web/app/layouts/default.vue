<script setup lang="ts">
import Pusher from "pusher-js";
import { useDisplay } from "vuetify";
import { setAuthHeader } from "~/composables/useApi";
import { getAuth } from "firebase/auth";

const route = useRoute();
const config = useRuntimeConfig();
const { lgAndUp } = useDisplay();
const authStore = useAuthStore();
const phonesStore = usePhonesStore();
const threadsStore = useThreadsStore();
const appStore = useAppStore();

let poller: ReturnType<typeof setInterval> | null = null;
let canPoll = false;

const hasDrawer = computed(() => {
  return ["threads", "threads-id"].includes((route.name as string) ?? "");
});

onMounted(() => {
  setTimeout(() => {
    const pusher = new Pusher(config.public.pusherKey as string, {
      cluster: config.public.pusherCluster as string,
    });

    if (authStore.authUser) {
      const channel = pusher.subscribe(authStore.authUser.id);
      channel.bind("phone.updated", () => {
        canPoll = true;
      });
    }

    startPoller();
  }, 10_000);
});

onBeforeUnmount(() => {
  if (poller) clearInterval(poller);
});

function startPoller() {
  poller = setInterval(async () => {
    if (!canPoll || authStore.authUser == null) return;

    appStore.setPolling(true);

    if (authStore.authUser && phonesStore.owner) {
      const auth = getAuth();
      const token = await auth.currentUser?.getIdToken();
      if (token) setAuthHeader(token);

      await Promise.all([
        phonesStore.loadPhones(true),
        threadsStore.loadThreads(),
        phonesStore.getHeartbeat(),
      ]);
    }

    canPoll = false;
    setTimeout(() => appStore.setPolling(false), 1000);
  }, 10_000);
}
</script>

<template>
  <v-app>
    <v-divider v-if="appStore.isLocal" class="py-1 bg-warning" />
    <v-navigation-drawer v-if="lgAndUp && hasDrawer" :width="400" permanent>
      <template #prepend>
        <v-divider v-if="appStore.isLocal" class="py-1 bg-warning" />
        <MessageThreadHeader />
        <div class="overflow-y-auto v-navigation-drawer__message-thread">
          <MessageThread />
        </div>
      </template>
    </v-navigation-drawer>
    <v-main :class="{ 'has-drawer': hasDrawer && lgAndUp }">
      <Toast />
      <slot v-if="authStore.authStateChanged" />
      <LoadingDashboard v-else />
    </v-main>
  </v-app>
</template>

<style lang="scss">
.v-application {
  .w-full {
    width: 100%;
  }
  .h-full {
    height: 100%;
  }
  .has-drawer {
    .v-snackbar {
      padding-left: 400px;
    }
  }
  .v-navigation-drawer__message-thread {
    height: calc(100vh - 120px);
    &::-webkit-scrollbar {
      width: 8px;
    }
    &::-webkit-scrollbar-track {
      background: #363636;
    }
    &::-webkit-scrollbar-thumb {
      background: #666666;
      border-radius: 8px;
    }
  }
  code.hljs {
    font-size: 16px;
  }
}
</style>
