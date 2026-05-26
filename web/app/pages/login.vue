<script setup lang="ts">
import { mdiArrowLeft } from "@mdi/js";

definePageMeta({
  middleware: ["guest"],
  layout: "blank",
});

useHead({
  title: "Login To Your Account - HTTP SMS",
});

const route = useRoute();
const to = computed(() => (route.query.to as string) || "/threads");
</script>

<template>
  <VContainer fluid class="fill-height d-flex align-center justify-center">
    <VRow align="center" justify="center">
      <VCol cols="12" md="4" xl="3" class="mt-n16">
        <div class="text-center mb-5">
          <VAvatar size="45" class="mt-n8 mr-4">
            <VImg :src="'/img/logo.svg'" />
          </VAvatar>
          <span class="text-display-small">Welcome</span>
        </div>
        <p class="text-title-large text-center text-medium-emphasis mt-1 mb-4">
          Join <b>23,273+</b> users who send/receive more than
          <br />
          <b>500,000</b> messages per month
        </p>
        <VCard max-width="360" class="mx-auto">
          <VCardText class="px-0">
            <ClientOnly>
              <FirebaseAuth :to="to" />
            </ClientOnly>
          </VCardText>
        </VCard>
        <div class="text-center mt-4">
          <BackButton />
        </div>
      </VCol>
    </VRow>
  </VContainer>
</template>
