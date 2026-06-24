<script setup lang="ts">
import {
  mdiArrowLeft,
  mdiCallReceived,
  mdiCallMade,
  mdiCheck,
  mdiAlert,
  mdiInvoice,
  mdiDownloadOutline,
} from '@mdi/js'
import type {
  RequestsUserPaymentInvoice,
  ResponsesUserSubscriptionPaymentsResponse,
} from '~~/shared/types/api'
import { formatBillingPeriodDateOrdinal } from '~/utils/filters'
import { countries, getStateOptions } from '~/utils/countries'

type SubscriptionPayment =
  ResponsesUserSubscriptionPaymentsResponse['data'][number]

definePageMeta({
  middleware: ['auth'],
})

useHead({
  title: 'Usage & Billing - httpSMS',
})

const config = useRuntimeConfig()
const { mdAndDown, lgAndUp } = useDisplay()
const authStore = useAuthStore()
const billingStore = useBillingStore()
const notificationsStore = useNotificationsStore()
const { formatDecimal, formatTimestamp } = useFilters()

const loading = ref(true)
const loadingSubscriptionPayments = ref(false)
const dialog = ref(false)
const subscriptionInvoiceDialog = ref(false)
const payments = ref<ResponsesUserSubscriptionPaymentsResponse | null>(null)
const selectedPayment = ref<SubscriptionPayment | null>(null)
const invoiceFormName = ref('')
const invoiceFormAddress = ref('')
const invoiceFormCity = ref('')
const invoiceFormState = ref('')
const invoiceFormZipCode = ref('')
const invoiceFormCountry = ref('')
const invoiceFormNotes = ref('')
const errorMessages = ref(new Map<string, string>())

type PaymentPlan = {
  name: string
  id: string
  price: number
  messagesPerMonth: number
}

const plans: PaymentPlan[] = [
  { name: 'Free', id: 'free', messagesPerMonth: 200, price: 0 },
  {
    name: 'PRO - Monthly',
    id: 'pro-monthly',
    messagesPerMonth: 5000,
    price: 10,
  },
  {
    name: 'PRO - Yearly',
    id: 'pro-yearly',
    messagesPerMonth: 5000,
    price: 100,
  },
  {
    name: 'Ultra - Monthly',
    id: 'ultra-monthly',
    messagesPerMonth: 10000,
    price: 20,
  },
  {
    name: 'Ultra - Yearly',
    id: 'ultra-yearly',
    messagesPerMonth: 10000,
    price: 200,
  },
  {
    name: '20k - Monthly',
    id: '20k-monthly',
    messagesPerMonth: 20000,
    price: 35,
  },
  {
    name: '20k - Yearly',
    id: '20k-yearly',
    messagesPerMonth: 20000,
    price: 350,
  },
  {
    name: '50k - Monthly',
    id: '50k-monthly',
    messagesPerMonth: 50000,
    price: 89,
  },
  {
    name: '100k - Monthly',
    id: '100k-monthly',
    messagesPerMonth: 100000,
    price: 175,
  },
  {
    name: '200k - Monthly',
    id: '200k-monthly',
    messagesPerMonth: 200000,
    price: 350,
  },
  {
    name: 'PRO - Lifetime',
    id: 'pro-lifetime',
    messagesPerMonth: 10000,
    price: 1000,
  },
]

const plan = computed<PaymentPlan>(() => {
  return (plans.find(
    (x) => x.id === (authStore.user?.subscription_name || 'free'),
  ) ?? plans[0])!
})

const isOnFreePlan = computed(() => plan.value.id === 'free')
const isOnLifetimePlan = computed(() => plan.value.id === 'pro-lifetime')
const subscriptionIsCancelled = computed(
  () => authStore.user?.subscription_status === 'cancelled',
)

const invoiceStateOptions = computed(() =>
  getStateOptions(invoiceFormCountry.value),
)

const totalMessages = computed(() => {
  if (!billingStore.billingUsage) return 0
  return (
    billingStore.billingUsage.sent_messages +
    billingStore.billingUsage.received_messages
  )
})

const checkoutURL = computed(() => {
  const url = new URL(config.public.checkoutUrl as string)
  const user = authStore.authUser
  if (user) {
    url.searchParams.append('checkout[custom][user_id]', user.id)
    url.searchParams.append('checkout[email]', user.email || '')
    url.searchParams.append('checkout[name]', user.displayName || '')
  }
  return url.toString()
})

const enterpriseCheckoutURL = computed(() => {
  const url = new URL(config.public.enterpriseCheckoutUrl as string)
  const user = authStore.authUser
  if (user) {
    url.searchParams.append('checkout[custom][user_id]', user.id)
    url.searchParams.append('checkout[email]', user.email || '')
    url.searchParams.append('checkout[name]', user.displayName || '')
  }
  return url.toString()
})

async function loadData() {
  await Promise.all([
    authStore.loadUser(),
    billingStore.loadBillingUsage(),
    billingStore.loadBillingUsageHistory(),
  ])
  loading.value = false
  loadSubscriptionInvoices()
}

async function loadSubscriptionInvoices() {
  if (!authStore.user?.subscription_id) return
  loadingSubscriptionPayments.value = true
  try {
    payments.value = await billingStore.indexSubscriptionPayments()
  } finally {
    loadingSubscriptionPayments.value = false
  }
}

async function updateDetails() {
  loading.value = true
  try {
    const link = await billingStore.getSubscriptionUpdateLink()
    window.location.href = link
  } catch {
    loading.value = false
  }
}

async function cancelPlan() {
  loading.value = true
  try {
    await billingStore.cancelSubscription()
    notificationsStore.addNotification({
      message: 'Subscription cancelled successfully',
      type: 'success',
    })
    navigateTo('/')
  } catch {
    loading.value = false
  }
}

async function generateInvoice() {
  errorMessages.value = new Map()
  loading.value = true
  try {
    await billingStore.generateSubscriptionPaymentInvoice(
      selectedPayment.value?.id || '',
      {
        name: invoiceFormName.value,
        address: invoiceFormAddress.value,
        city: invoiceFormCity.value,
        state: invoiceFormState.value,
        zip_code: invoiceFormZipCode.value,
        country: invoiceFormCountry.value,
        notes: invoiceFormNotes.value,
      } as RequestsUserPaymentInvoice,
    )
    subscriptionInvoiceDialog.value = false
  } catch (error: unknown) {
    if (error instanceof Map) {
      errorMessages.value = error
    }
  } finally {
    loading.value = false
  }
}

function showInvoiceDialog(payment: SubscriptionPayment) {
  selectedPayment.value = payment
  subscriptionInvoiceDialog.value = true
}

onMounted(async () => {
  await loadData()
})
</script>

<template>
  <VContainer fluid class="px-0 pt-0" :class="{ 'fill-height': lgAndUp }">
    <div class="w-100 h-100">
      <VAppBar height="60" :density="mdAndDown ? 'compact' : 'default'">
        <VBtn icon to="/threads">
          <VIcon :icon="mdiArrowLeft" />
        </VBtn>
        <VToolbarTitle>Account Usage</VToolbarTitle>
        <VProgressLinear
          color="primary"
          :active="loading"
          :indeterminate="loading"
          absolute
          location="bottom"
        />
      </VAppBar>
      <VContainer>
        <VRow>
          <VCol cols="12" md="9" offset-md="1" xl="8" offset-xl="2">
            <!-- Current Plan -->
            <h4 class="text-headline-large mb-3 mt-0">Current Plan</h4>
            <VRow v-if="authStore.user">
              <VCol md="6">
                <VAlert type="info" :icon="false" variant="tonal" prominent>
                  <div>
                    <h1
                      class="text-title-large mt-0 mb-0 font-weight-bold text-uppercase"
                    >
                      <span v-if="isOnFreePlan">{{ plan.name }}</span>
                      <span v-else-if="subscriptionIsCancelled">
                        <span class="text-warning">{{ plan.name }}</span> → Free
                      </span>
                      <span v-else>{{ plan.name }}</span>
                    </h1>
                    <p
                      v-if="
                        !isOnFreePlan &&
                        !isOnLifetimePlan &&
                        !subscriptionIsCancelled
                      "
                      class="text-medium-emphasis mt-1"
                    >
                      Your next bill is for <b>${{ plan.price }}</b> on
                      <b>{{
                        new Date(
                          authStore.user.subscription_renews_at!,
                        ).toLocaleDateString()
                      }}</b>
                    </p>
                    <p v-if="isOnLifetimePlan" class="text-medium-emphasis">
                      You are on the life time plan which costs
                      <b>${{ plan.price }}</b>
                    </p>
                    <p
                      v-else-if="subscriptionIsCancelled"
                      class="text-medium-emphasis"
                    >
                      You will be downgraded to the <b>FREE</b> plan on
                      <b>{{
                        new Date(
                          authStore.user.subscription_ends_at!,
                        ).toLocaleDateString()
                      }}</b>
                    </p>
                    <p v-else class="text-medium-emphasis mt-1">
                      {{ totalMessages }}/{{ plan.messagesPerMonth }} messages
                    </p>
                  </div>
                  <div class="d-flex mb-1 mt-1">
                    <VBtn
                      v-if="
                        !subscriptionIsCancelled &&
                        !isOnFreePlan &&
                        !isOnLifetimePlan
                      "
                      color="primary"
                      :loading="loading"
                      @click="updateDetails"
                    >
                      Update Plan
                    </VBtn>
                    <VBtn
                      v-else-if="!isOnLifetimePlan"
                      color="primary"
                      :href="checkoutURL"
                    >
                      Upgrade Plan
                    </VBtn>
                    <VSpacer />
                    <VDialog
                      v-if="
                        !subscriptionIsCancelled &&
                        !isOnFreePlan &&
                        !isOnLifetimePlan
                      "
                      v-model="dialog"
                      max-width="590"
                      opacity="0.9"
                    >
                      <template #activator="{ props: activatorProps }">
                        <VBtn
                          v-bind="activatorProps"
                          color="error"
                          variant="text"
                        >
                          Cancel Plan
                        </VBtn>
                      </template>
                      <VCard>
                        <VCardText class="pt-4">
                          <h2 class="text-headline-medium mb-2">
                            Are you sure you want to cancel your subscription?
                          </h2>
                          <p>
                            You will be downgraded to the free plan at the end
                            of the current billing period on
                            <b>{{
                              new Date(
                                authStore.user.subscription_renews_at!,
                              ).toLocaleDateString()
                            }}</b>
                          </p>
                        </VCardText>
                        <VCardActions>
                          <VBtn color="primary" @click="dialog = false">
                            Keep Subscription
                          </VBtn>
                          <VSpacer />
                          <VBtn
                            v-if="!isOnFreePlan"
                            variant="text"
                            :loading="loading"
                            color="error"
                            @click="cancelPlan"
                          >
                            Cancel Plan
                          </VBtn>
                        </VCardActions>
                      </VCard>
                    </VDialog>
                  </div>
                </VAlert>
              </VCol>
            </VRow>

            <!-- Upgrade Plan (only for free users) -->
            <template v-if="isOnFreePlan">
              <h2 class="text-headline-large mt-4 mb-2">Upgrade Plan</h2>
              <VRow>
                <VCol cols="12" md="6">
                  <VCard :href="checkoutURL" link>
                    <VCardText>
                      <VRow align="center">
                        <VCol class="flex-grow-1 flex-shrink-1">
                          <h1
                            class="text-title-large font-weight-bold text-uppercase mt-3"
                          >
                            Pro Plan
                          </h1>
                          <p class="text-medium-emphasis">
                            Send and receive up to 20,000 messages per month
                          </p>
                        </VCol>
                        <VCol class="flex-grow-0 flex-shrink-0">
                          <span class="text-headline-medium">$10</span>/month
                        </VCol>
                      </VRow>
                    </VCardText>
                  </VCard>
                </VCol>
                <VCol cols="12" md="6">
                  <VCard :href="enterpriseCheckoutURL" link>
                    <VCardText>
                      <VRow align="center">
                        <VCol class="flex-grow-1 flex-shrink-1">
                          <h1
                            class="text-title-large font-weight-bold text-uppercase mt-3"
                          >
                            Enterprise Plan
                          </h1>
                          <p class="text-medium-emphasis">
                            Send and receive up to 200,000 messages per month
                          </p>
                        </VCol>
                        <VCol class="flex-grow-0 flex-shrink-0">
                          <span class="text-headline-medium">$89</span>/month
                        </VCol>
                      </VRow>
                    </VCardText>
                  </VCard>
                </VCol>
              </VRow>
            </template>

            <!-- Overview -->
            <h4 class="text-headline-large mb-3 mt-8">Overview</h4>
            <p class="text-medium-emphasis">
              This is the summary of the sent messages and received messages
              from
              <v-code v-if="billingStore.billingUsage" class="font-weight-bold">
                <span
                  v-html="
                    formatBillingPeriodDateOrdinal(
                      billingStore.billingUsage.start_timestamp,
                    )
                  "
                />
              </v-code>
              to
              <v-code v-if="billingStore.billingUsage" class="font-weight-bold">
                <span
                  v-html="
                    formatBillingPeriodDateOrdinal(
                      billingStore.billingUsage.end_timestamp,
                    )
                  "
                /> </v-code
              >.
            </p>
            <VRow v-if="billingStore.billingUsage">
              <VCol cols="12" md="6">
                <VAlert
                  type="info"
                  variant="tonal"
                  :icon="mdiCallMade"
                  prominent
                >
                  <h2 class="text-headline-large my-0">
                    {{ formatDecimal(billingStore.billingUsage.sent_messages) }}
                  </h2>
                  <p class="text-medium-emphasis mt-n1">Messages Sent</p>
                </VAlert>
              </VCol>
              <VCol cols="12" md="6">
                <VAlert
                  type="warning"
                  variant="tonal"
                  :icon="mdiCallReceived"
                  prominent
                >
                  <h2 class="text-headline-large font-weight-bold my-0">
                    {{
                      formatDecimal(billingStore.billingUsage.received_messages)
                    }}
                  </h2>
                  <p class="text-medium-emphasis mt-n1">Messages Received</p>
                </VAlert>
              </VCol>
            </VRow>

            <!-- Subscription Payments -->
            <template v-if="authStore.user?.subscription_id != null">
              <h4 class="text-headline-large mb-3 mt-8">
                Subscription Payments
              </h4>
              <p class="text-medium-emphasis">
                This is a list of your last 10 subscription payments made using
                our payment provider
                <a
                  class="text-decoration-none"
                  href="https://www.lemonsqueezy.com"
                >
                  Lemon Squeezy</a
                >.
              </p>
              <VProgressCircular
                v-if="payments == null && loadingSubscriptionPayments"
                :size="20"
                :width="2"
                color="primary"
                indeterminate
              />
              <VTable v-if="payments">
                <thead>
                  <tr class="text-uppercase">
                    <th v-if="lgAndUp" class="text-left">ID</th>
                    <th class="text-left">Timestamp</th>
                    <th class="text-left">Status</th>
                    <th v-if="lgAndUp" class="text-left">Tax</th>
                    <th class="text-left">Total</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="payment in payments.data" :key="payment.id">
                    <td v-if="lgAndUp">{{ payment.id }}</td>
                    <td>
                      {{ formatTimestamp(payment.attributes.created_at) }}
                    </td>
                    <td>
                      <VChip
                        v-if="payment.attributes.status === 'paid'"
                        color="success"
                      >
                        <template #prepend>
                          <VIcon size="small" :icon="mdiCheck" />
                        </template>
                        {{ payment.attributes.status_formatted }}
                      </VChip>
                      <VChip v-else color="error">
                        <template #prepend>
                          <VIcon size="small" :icon="mdiAlert" />
                        </template>
                        {{ payment.attributes.status_formatted }}
                      </VChip>
                    </td>
                    <td v-if="lgAndUp">
                      {{ payment.attributes.tax_formatted }}
                    </td>
                    <td class="font-weight-bold">
                      {{ payment.attributes.total_formatted }}
                    </td>
                    <td class="text-right">
                      <VBtn
                        color="primary"
                        size="small"
                        @click="showInvoiceDialog(payment)"
                      >
                        <VIcon start :icon="mdiInvoice" />
                        Invoice
                      </VBtn>
                    </td>
                  </tr>
                </tbody>
              </VTable>
            </template>

            <!-- Usage History -->
            <h4 class="text-headline-large mb-3 mt-8">Usage History</h4>
            <p class="text-medium-emphasis">
              Summary of all the sent and received messages in the past 12
              billing periods
            </p>
            <VTable density="comfortable">
              <thead>
                <tr class="text-uppercase text-medium-emphasis">
                  <th class="text-left">Start Date</th>
                  <th class="text-left">End Date</th>
                  <th class="text-left">
                    Sent
                    <span v-if="lgAndUp">Messages</span>
                  </th>
                  <th class="text-left">
                    Received
                    <span v-if="lgAndUp">Messages</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="billingUsage in billingStore.billingUsageHistory"
                  :key="billingUsage.id"
                >
                  <td
                    v-html="
                      formatBillingPeriodDateOrdinal(
                        billingUsage.start_timestamp,
                      )
                    "
                  />
                  <td
                    v-html="
                      formatBillingPeriodDateOrdinal(billingUsage.end_timestamp)
                    "
                  />
                  <td>{{ formatDecimal(billingUsage.sent_messages) }}</td>
                  <td>{{ billingUsage.received_messages }}</td>
                </tr>
              </tbody>
            </VTable>
          </VCol>
        </VRow>
      </VContainer>
    </div>

    <!-- Invoice Dialog -->
    <VDialog
      v-model="subscriptionInvoiceDialog"
      persistent
      max-width="600"
      opacity="0.9"
    >
      <VCard>
        <VCardTitle class="text-headline-large">Generate Invoice</VCardTitle>
        <VCardSubtitle class="mt-n1">
          Create an invoice for your
          <b>{{ selectedPayment?.attributes.total_formatted }}</b> payment on
          {{ formatTimestamp(selectedPayment?.attributes.created_at ?? '') }}
        </VCardSubtitle>
        <VCardText class="pb-0">
          <VContainer>
            <VRow>
              <VCol cols="12">
                <VTextField
                  v-model="invoiceFormName"
                  density="compact"
                  color="primary"
                  :disabled="loading"
                  :error="errorMessages.has('name')"
                  :error-messages="errorMessages.get('name')"
                  label="Name"
                  placeholder="e.g Acme Corporation"
                  persistent-placeholder
                  variant="outlined"
                />
              </VCol>
              <VCol cols="12">
                <VTextField
                  v-model="invoiceFormAddress"
                  density="compact"
                  color="primary"
                  :disabled="loading"
                  :error="errorMessages.has('address')"
                  :error-messages="errorMessages.get('address')"
                  label="Address"
                  placeholder="e.g 221B Baker Street"
                  persistent-placeholder
                  variant="outlined"
                />
              </VCol>
            </VRow>
            <VRow>
              <VCol cols="6">
                <VTextField
                  v-model="invoiceFormCity"
                  density="compact"
                  :disabled="loading"
                  :error="errorMessages.has('city')"
                  :error-messages="errorMessages.get('city')"
                  label="City"
                  placeholder="e.g Los Angeles"
                  persistent-placeholder
                  variant="outlined"
                />
              </VCol>
              <VCol cols="6">
                <VTextField
                  v-if="invoiceStateOptions.length === 0"
                  v-model="invoiceFormState"
                  density="compact"
                  color="primary"
                  :disabled="loading"
                  :error="errorMessages.has('state')"
                  :error-messages="errorMessages.get('state')"
                  label="State"
                  placeholder="e.g CA"
                  persistent-placeholder
                  variant="outlined"
                />
                <VAutocomplete
                  v-else
                  v-model="invoiceFormState"
                  density="compact"
                  color="primary"
                  :disabled="loading"
                  :error="errorMessages.has('state')"
                  :error-messages="errorMessages.get('state')"
                  :items="invoiceStateOptions"
                  label="State"
                  placeholder="e.g CA"
                  persistent-placeholder
                  variant="outlined"
                />
              </VCol>
            </VRow>
            <VRow>
              <VCol cols="6">
                <VTextField
                  v-model="invoiceFormZipCode"
                  density="compact"
                  color="primary"
                  :disabled="loading"
                  :error="errorMessages.has('zip_code')"
                  :error-messages="errorMessages.get('zip_code')"
                  label="Zip Code"
                  placeholder="e.g 46001"
                  persistent-placeholder
                  variant="outlined"
                />
              </VCol>
              <VCol cols="6">
                <VAutocomplete
                  v-model="invoiceFormCountry"
                  density="compact"
                  color="primary"
                  :disabled="loading"
                  :error="errorMessages.has('country')"
                  :error-messages="errorMessages.get('country')"
                  :items="countries"
                  label="Country"
                  placeholder="e.g United States"
                  persistent-placeholder
                  variant="outlined"
                />
              </VCol>
            </VRow>
            <VRow>
              <VCol cols="12">
                <VTextarea
                  v-model="invoiceFormNotes"
                  density="compact"
                  color="primary"
                  :disabled="loading"
                  :error="errorMessages.has('notes')"
                  :error-messages="errorMessages.get('notes')"
                  rows="3"
                  label="Notes (optional)"
                  placeholder="e.g Thanks for doing business with us!"
                  persistent-placeholder
                  variant="outlined"
                />
              </VCol>
            </VRow>
          </VContainer>
        </VCardText>
        <VCardActions class="pb-4 mt-n4">
          <VBtn
            variant="flat"
            :loading="loading"
            color="primary"
            @click="generateInvoice"
          >
            <VIcon start :icon="mdiDownloadOutline" />
            Download Invoice
          </VBtn>
          <VSpacer />
          <VBtn
            color="warning"
            variant="text"
            @click="subscriptionInvoiceDialog = false"
          >
            Close
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>
  </VContainer>
</template>
