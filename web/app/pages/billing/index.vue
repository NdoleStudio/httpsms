<script setup lang="ts">
import { mdiArrowLeft, mdiCreditCard } from "@mdi/js";

definePageMeta({
  middleware: ["auth"],
});

useHead({
  title: "Billing - httpSMS",
});

const config = useRuntimeConfig();
const { mdAndDown, lgAndUp } = useDisplay();
const authStore = useAuthStore();
const billingStore = useBillingStore();
const notificationsStore = useNotificationsStore();
const { useApi } = useApiComposable();

const loading = ref(true);
const usage = ref<any>(null);

async function loadBillingUsage() {
  loading.value = true;
  try {
    const api = useApi();
    const response = await api<{ data: any }>("/v1/billing/usage");
    usage.value = response.data;
  } catch {
    // silently fail
  } finally {
    loading.value = false;
  }
}

function getCheckoutUrl() {
  window.open(config.public.checkoutUrl as string, "_blank");
}

function getEnterpriseCheckoutUrl() {
  window.open(config.public.enterpriseCheckoutUrl as string, "_blank");
}

onMounted(async () => {
  await authStore.loadUser();
  await loadBillingUsage();
});
</script>

<template>
  <VContainer fluid class="px-0 pt-0" :class="{ 'fill-height': lgAndUp }">
    <div class="w-100 h-100">
      <VAppBar height="60" :density="mdAndDown ? 'compact' : 'default'">
        <VBtn icon to="/threads">
          <VIcon :icon="mdiArrowLeft" />
        </VBtn>
        <VToolbarTitle>Billing</VToolbarTitle>
      </VAppBar>
      <VContainer class="mt-16">
        <VRow>
          <VCol cols="12" md="8" offset-md="2">
            <h4 class="text-headline-large mb-3">Usage</h4>
            <p class="text-medium-emphasis">
              Your current billing period usage and limits.
            </p>

            <VProgressLinear v-if="loading" indeterminate class="mb-4" />

            <template v-else-if="usage">
              <VCard class="mb-6">
                <VCardText>
                  <VRow>
                    <VCol cols="6" md="3">
                      <div class="text-center">
                        <p class="text-headline-large text-primary">
                          {{ usage.sent_messages ?? 0 }}
                        </p>
                        <p class="text-medium-emphasis text-title-medium">
                          Messages Sent
                        </p>
                      </div>
                    </VCol>
                    <VCol cols="6" md="3">
                      <div class="text-center">
                        <p class="text-headline-large text-primary">
                          {{ usage.received_messages ?? 0 }}
                        </p>
                        <p class="text-medium-emphasis text-title-medium">
                          Messages Received
                        </p>
                      </div>
                    </VCol>
                    <VCol cols="6" md="3">
                      <div class="text-center">
                        <p class="text-headline-large">
                          {{ usage.total_messages ?? 0 }}
                        </p>
                        <p class="text-medium-emphasis text-title-medium">
                          Total Messages
                        </p>
                      </div>
                    </VCol>
                    <VCol cols="6" md="3">
                      <div class="text-center">
                        <p class="text-headline-large">
                          {{ usage.message_limit ?? 200 }}
                        </p>
                        <p class="text-medium-emphasis text-title-medium">
                          Monthly Limit
                        </p>
                      </div>
                    </VCol>
                  </VRow>
                  <VProgressLinear
                    :model-value="
                      ((usage.total_messages ?? 0) /
                        (usage.message_limit ?? 200)) *
                      100
                    "
                    color="primary"
                    height="8"
                    rounded
                    class="mt-4"
                  />
                </VCardText>
              </VCard>

              <h4 class="text-headline-large mb-3 mt-8">Upgrade Plan</h4>
              <p class="text-medium-emphasis mb-4">
                Upgrade your plan to send and receive more SMS messages per
                month.
              </p>

              <VRow>
                <VCol cols="12" md="6">
                  <VCard>
                    <VCardTitle>Pro Plan</VCardTitle>
                    <VCardSubtitle>Up to 5,000 messages/month</VCardSubtitle>
                    <VCardText>
                      <p class="text-headline-large text-primary mb-2">
                        $10<span class="text-body-large">/month</span>
                      </p>
                      <ul class="ml-4">
                        <li>5,000 SMS messages per month</li>
                        <li>Priority support</li>
                        <li>Webhook integrations</li>
                      </ul>
                    </VCardText>
                    <VCardActions>
                      <VBtn color="primary" block @click="getCheckoutUrl">
                        <VIcon start :icon="mdiCreditCard" />
                        Upgrade to Pro
                      </VBtn>
                    </VCardActions>
                  </VCard>
                </VCol>
                <VCol cols="12" md="6">
                  <VCard>
                    <VCardTitle>Enterprise Plan</VCardTitle>
                    <VCardSubtitle>Custom message limits</VCardSubtitle>
                    <VCardText>
                      <p class="text-headline-large mb-2">Custom</p>
                      <ul class="ml-4">
                        <li>Up to 200,000+ SMS messages per month</li>
                        <li>Dedicated support</li>
                        <li>Custom integrations</li>
                      </ul>
                    </VCardText>
                    <VCardActions>
                      <VBtn
                        color="secondary"
                        block
                        @click="getEnterpriseCheckoutUrl"
                      >
                        Contact Sales
                      </VBtn>
                    </VCardActions>
                  </VCard>
                </VCol>
              </VRow>
            </template>
          </VCol>
        </VRow>
      </VContainer>
    </div>
  </VContainer>
</template>
