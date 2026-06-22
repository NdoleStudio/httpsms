<script setup lang="ts">
definePageMeta({
  middleware: ["auth"],
});

useHead({
  title: "Threads - httpSMS",
});

const { lgAndUp } = useDisplay();
const authStore = useAuthStore();
const phonesStore = usePhonesStore();
const threadsStore = useThreadsStore();

onMounted(async () => {
  await authStore.loadUser();
  await phonesStore.loadPhones();
  await threadsStore.loadThreads();
});
</script>

<template>
  <VContainer fluid :class="{ 'fill-height': lgAndUp }">
    <VRow v-if="lgAndUp" align="center" :class="{ 'fill-height': lgAndUp }" justify="center">
      <div :class="{ 'mt-n16': lgAndUp }">
        <VImg
          class="mx-auto mb-4"
          :class="{ 'mt-n16': lgAndUp }"
          max-height="400"
          max-width="90%"
          :src="'/img/person-texting.svg'"
        />
        <div class="text-center">
          <h3 class="text-headline-medium mt-4 mb-0">Select a Message</h3>
          <p class="text-medium-emphasis mt-0">
            Don't hesitate to
            <a
              href="https://discord.gg/kGk8HVqeEZ"
              target="_blank"
              class="text-decoration-none hover:text-decoration-underline"
              >message us on Discord</a
            >
            if you have any questions
          </p>
        </div>
      </div>
    </VRow>
    <VRow v-else justify="end">
      <VCol class="px-0 py-0">
        <MessageThreadHeader />
        <MessageThread />
      </VCol>
    </VRow>
  </VContainer>
</template>
