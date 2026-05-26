<script setup lang="ts">
import { mdiArrowLeft, mdiMagnify } from "@mdi/js";
import type { Message } from "~/shared/types/message";

definePageMeta({
  middleware: ["auth"],
});

useHead({
  title: "Search Messages - httpSMS",
});

const route = useRoute();
const { mdAndDown, lgAndUp } = useDisplay();
const messagesStore = useMessagesStore();
const authStore = useAuthStore();

const loading = ref(false);
const searchQuery = ref((route.query.query as string) || "");
const messages = ref<Message[]>([]);

async function searchMessages() {
  if (!searchQuery.value.trim()) return;
  loading.value = true;
  try {
    messages.value = await messagesStore.searchMessages(searchQuery.value);
  } finally {
    loading.value = false;
  }
}

onMounted(async () => {
  await authStore.loadUser();
  if (searchQuery.value) {
    await searchMessages();
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
          :indeterminate="loading"
          location="bottom"
          absolute
        />
      </VAppBar>
      <VContainer>
        <VRow>
          <VCol cols="12" md="8" offset-md="2">
            <form @submit.prevent="searchMessages">
              <VTextField
                v-model="searchQuery"
                variant="outlined"
                persistent-placeholder
                placeholder="Search by message ID, request ID, or content"
                label="Search"
                :append-inner-icon="mdiMagnify"
                @click:append-inner="searchMessages"
              />
            </form>
          </VCol>
        </VRow>
        <VRow v-if="messages.length > 0">
          <VCol cols="12">
            <VTable>
              <thead>
                <tr class="text-uppercase text-title-medium">
                  <th class="text-left">Contact</th>
                  <th class="text-left">Content</th>
                  <th class="text-center">Type</th>
                  <th class="text-center">Status</th>
                  <th class="text-center">Date</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="message in messages" :key="message.id">
                  <td class="text-left">
                    {{ useFilters().phoneNumber(message.contact) }}
                  </td>
                  <td class="text-left">
                    {{ message.content?.substring(0, 50)
                    }}{{ (message.content?.length ?? 0) > 50 ? "..." : "" }}
                  </td>
                  <td class="text-center">
                    {{
                      message.type === "mobile-terminated" ? "Sent" : "Received"
                    }}
                  </td>
                  <td class="text-center">
                    <VChip
                      :color="
                        message.status === 'delivered'
                          ? 'success'
                          : message.status === 'failed'
                          ? 'error'
                          : 'default'
                      "
                      size="small"
                    >
                      {{ message.status }}
                    </VChip>
                  </td>
                  <td class="text-center">
                    {{ useFilters().timestamp(message.order_timestamp) }}
                  </td>
                </tr>
              </tbody>
            </VTable>
          </VCol>
        </VRow>
        <VRow v-else-if="!loading && searchQuery">
          <VCol cols="12" class="text-center">
            <p class="text-medium-emphasis mt-8">
              No messages found for "{{ searchQuery }}"
            </p>
          </VCol>
        </VRow>
      </VContainer>
    </div>
  </VContainer>
</template>
