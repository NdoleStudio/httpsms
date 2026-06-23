<script setup lang="ts">
import { mdiArrowLeft, mdiPlus, mdiDelete, mdiEye } from '@mdi/js'
import QRCode from 'qrcode'
import Pusher from 'pusher-js'
import type { Channel } from 'pusher-js'
import { ErrorMessages } from '~/utils/errors'
import { toApiError } from '~/utils/api-error'
import type { EntitiesPhoneAPIKey } from '~~/shared/types/api'

definePageMeta({
  middleware: ['auth'],
})

useHead({
  title: 'Phone API Keys - httpSMS',
})

const config = useRuntimeConfig()
const { mdAndDown, lgAndUp } = useDisplay()
const authStore = useAuthStore()
const appStore = useAppStore()
const phonesStore = usePhonesStore()
const notificationsStore = useNotificationsStore()
const { formatTimestamp, formatPhoneNumber } = useFilters()
const { useApi } = useApiComposable()

const loading = ref(true)
const phoneApiKeys = ref<EntitiesPhoneAPIKey[]>([])
const errorMessages = ref(new ErrorMessages())

const showCreateApiKeyDialog = ref(false)
const formPhoneApiKeyName = ref('')

const showPhoneApiKeyQrCode = ref(false)
const deleteApiKeyDialog = ref(false)
const removePhoneFromApiKeyDialog = ref(false)
const activePhoneApiKey = ref<EntitiesPhoneAPIKey | null>(null)
const activePhoneNumber = ref('')

const qrCodeCanvas = ref<HTMLCanvasElement | null>(null)
let webhookChannel: Channel | null = null

function parseErrors(error: unknown): ErrorMessages {
  const bag = new ErrorMessages()
  const data = toApiError(error).data?.data
  if (data && typeof data === 'object') {
    Object.keys(data).forEach((key) => bag.addMany(key, data[key]))
  }
  return bag
}

async function loadPhoneApiKeys() {
  loading.value = true
  try {
    const api = useApi()
    const response = await api<{ data: EntitiesPhoneAPIKey[] }>(
      '/v1/phone-api-keys',
      { query: { limit: 100 } },
    )
    phoneApiKeys.value = response.data ?? []
  } catch {
    notificationsStore.addNotification({
      message: 'Failed to load Phone API Keys',
      type: 'error',
    })
  } finally {
    loading.value = false
  }
}

async function createPhoneApiKey() {
  errorMessages.value = new ErrorMessages()
  loading.value = true
  try {
    const api = useApi()
    await api('/v1/phone-api-keys', {
      method: 'POST',
      body: { name: formPhoneApiKeyName.value },
    })
    notificationsStore.addNotification({
      message: 'Phone API Key created successfully',
      type: 'success',
    })
    formPhoneApiKeyName.value = ''
    showCreateApiKeyDialog.value = false
    await loadPhoneApiKeys()
  } catch (error: unknown) {
    errorMessages.value = parseErrors(error)
    if (errorMessages.value.size() === 0) {
      notificationsStore.addNotification({
        message: 'Failed to create Phone API Key',
        type: 'error',
      })
    }
  } finally {
    loading.value = false
  }
}

function generateQrCode(text: string) {
  const canvas = qrCodeCanvas.value
  if (!canvas) {
    return
  }
  QRCode.toCanvas(
    canvas,
    text,
    { errorCorrectionLevel: 'H' },
    (err: Error | null | undefined) => {
      if (err) {
        notificationsStore.addNotification({
          message: 'Failed to generate phone API key QR code',
          type: 'error',
        })
      }
    },
  )
}

function showPhoneApiKey(apiKey: EntitiesPhoneAPIKey) {
  activePhoneApiKey.value = apiKey
  showPhoneApiKeyQrCode.value = true
  nextTick(() => {
    generateQrCode(apiKey.api_key)
  })
}

function showDeletePhoneApiKeyDialog(apiKey: EntitiesPhoneAPIKey) {
  activePhoneApiKey.value = apiKey
  deleteApiKeyDialog.value = true
}

function showRemovePhoneFromApiKeyDialog(
  apiKey: EntitiesPhoneAPIKey,
  phoneNumber: string,
) {
  activePhoneApiKey.value = apiKey
  activePhoneNumber.value = phoneNumber
  removePhoneFromApiKeyDialog.value = true
}

async function deleteApiKey() {
  if (!activePhoneApiKey.value) {
    return
  }
  loading.value = true
  try {
    const api = useApi()
    await api(`/v1/phone-api-keys/${activePhoneApiKey.value.id}`, {
      method: 'DELETE',
    })
    notificationsStore.addNotification({
      message: 'The phone API key has been deleted successfully',
      type: 'success',
    })
    deleteApiKeyDialog.value = false
    await loadPhoneApiKeys()
  } catch {
    notificationsStore.addNotification({
      message: 'Failed to delete Phone API Key',
      type: 'error',
    })
    loading.value = false
  }
}

async function removePhoneFromPhoneKey() {
  if (!activePhoneApiKey.value) {
    return
  }
  const phoneId = phonesStore.phones.find(
    (phone) => phone.phone_number === activePhoneNumber.value,
  )?.id
  if (!phoneId) {
    notificationsStore.addNotification({
      message: 'Could not find the phone to remove from the API key',
      type: 'error',
    })
    return
  }
  loading.value = true
  try {
    const api = useApi()
    await api(
      `/v1/phone-api-keys/${activePhoneApiKey.value.id}/phones/${phoneId}`,
      { method: 'DELETE' },
    )
    notificationsStore.addNotification({
      message: 'The phone has been removed from the phone API key successfully',
      type: 'success',
    })
    removePhoneFromApiKeyDialog.value = false
    await loadPhoneApiKeys()
  } catch {
    notificationsStore.addNotification({
      message: 'Failed to remove the phone from the Phone API Key',
      type: 'error',
    })
    loading.value = false
  }
}

onMounted(async () => {
  await authStore.loadUser()
  await phonesStore.loadPhones()
  await loadPhoneApiKeys()

  const pusherKey = config.public.pusherKey as string
  const pusherCluster = config.public.pusherCluster as string
  if (pusherKey && authStore.user?.id) {
    const pusher = new Pusher(pusherKey, { cluster: pusherCluster })
    webhookChannel = pusher.subscribe(authStore.user.id)
    webhookChannel.bind('phone.updated', () => {
      if (!loading.value) {
        loadPhoneApiKeys()
      }
    })
  }
})

onBeforeUnmount(() => {
  if (webhookChannel) {
    webhookChannel.unsubscribe()
  }
})
</script>

<template>
  <VContainer fluid class="px-0 pt-0" :class="{ 'fill-height': lgAndUp }">
    <div class="w-100 h-100">
      <VAppBar height="60" :density="mdAndDown ? 'compact' : 'default'">
        <VBtn icon to="/threads">
          <VIcon :icon="mdiArrowLeft" />
        </VBtn>
        <VToolbarTitle> Phone API Keys </VToolbarTitle>
        <VProgressLinear
          color="primary"
          :active="loading"
          :indeterminate="loading"
          absolute
          location="bottom"
        />
      </VAppBar>
      <VContainer class="pt-0">
        <VRow>
          <VCol cols="12" md="9" offset-md="1" xl="8" offset-xl="2">
            <div class="d-flex align-center flex-wrap mt-3 mb-4">
              <VProgressCircular
                v-if="loading"
                :size="24"
                :width="2"
                color="primary"
                class="mt-1 mr-2"
                indeterminate
              />
              <h5 class="text-display-small my-0">Phone API Keys</h5>
              <VBtn
                color="primary"
                class="ml-4 mt-1"
                @click="showCreateApiKeyDialog = true"
              >
                <VIcon start :icon="mdiPlus" />
                Create API Key
              </VBtn>
              <VSpacer />
              <VBtn
                v-if="lgAndUp"
                href="https://docs.httpsms.com/features/phone-api-keys"
                target="_blank"
                variant="tonal"
                class="mt-1"
              >
                Documentation
              </VBtn>
            </div>
            <p class="text-medium-emphasis">
              If you have multiple phones, you can create unique phone API keys
              for your different Android phones. These API keys can only be used
              on the specific mobile phone when it calls the httpSMS server for
              specific actions like sending heartbeats, registering received
              messages, delivery reports etc. If you want to interact with the
              full
              <a
                class="text-decoration-none hover:text-decoration-underline"
                target="_blank"
                href="https://api.httpsms.com"
                >httpSMS API</a
              >, use the API key under your account settings page instead
              <NuxtLink
                class="text-decoration-none hover:text-decoration-underline"
                to="/settings"
                >https://httpsms.com/settings</NuxtLink
              >.
            </p>
            <VTable class="mb-4 api-key-table" density="comfortable">
              <thead>
                <tr class="text-uppercase text-medium-emphasis">
                  <th class="text-left">Name</th>
                  <th class="text-left">Created At</th>
                  <th class="text-left">Phone Numbers</th>
                  <th class="text-left">Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="phoneApiKey in phoneApiKeys" :key="phoneApiKey.id">
                  <td class="text-left">{{ phoneApiKey.name }}</td>
                  <td>{{ formatTimestamp(phoneApiKey.created_at) }}</td>
                  <td>
                    <ul v-if="phoneApiKey.phone_numbers.length" class="ml-n3">
                      <li
                        v-for="phoneNumber in phoneApiKey.phone_numbers"
                        :key="phoneNumber"
                        class="my-3"
                      >
                        <b>{{ formatPhoneNumber(phoneNumber) }}</b>
                        <VBtn
                          class="ml-2"
                          size="small"
                          color="error"
                          @click="
                            showRemovePhoneFromApiKeyDialog(
                              phoneApiKey,
                              phoneNumber,
                            )
                          "
                        >
                          Remove
                        </VBtn>
                      </li>
                    </ul>
                    <span v-else class="text-medium-emphasis">-</span>
                  </td>
                  <td>
                    <VBtn
                      size="small"
                      color="primary"
                      :disabled="loading"
                      @click="showPhoneApiKey(phoneApiKey)"
                    >
                      <VIcon start :icon="mdiEye" /> View
                    </VBtn>
                    <VBtn
                      class="ml-2"
                      size="small"
                      color="error"
                      :disabled="loading"
                      @click="showDeletePhoneApiKeyDialog(phoneApiKey)"
                    >
                      <VIcon start :icon="mdiDelete" /> Delete
                    </VBtn>
                  </td>
                </tr>
              </tbody>
            </VTable>
          </VCol>
        </VRow>
      </VContainer>
    </div>

    <VDialog v-model="showCreateApiKeyDialog" max-width="600" opacity="0.9">
      <VCard>
        <VCardTitle>Create Phone API Key</VCardTitle>
        <VCardSubtitle class="mt-2" style="white-space: normal">
          After creating the API key you can use it to login to the httpSMS
          Android app on your phone
        </VCardSubtitle>
        <VCardText>
          <VForm @submit.prevent="createPhoneApiKey">
            <VTextField
              v-model="formPhoneApiKeyName"
              variant="outlined"
              label="Name"
              class="mt-4"
              persistent-placeholder
              placeholder="Enter a name for your phone API key"
              name="api-key"
              :disabled="loading"
              :error="errorMessages.has('name')"
              :error-messages="errorMessages.get('name')"
            />
          </VForm>
        </VCardText>
        <VCardActions class="mt-n6 mb-1">
          <VBtn
            color="primary"
            variant="flat"
            :loading="loading"
            @click="createPhoneApiKey"
          >
            Create<span v-if="lgAndUp" class="mx-1">Phone API</span>Key
          </VBtn>
          <VSpacer />
          <VBtn
            variant="text"
            color="warning"
            @click="showCreateApiKeyDialog = false"
          >
            Close
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>

    <VDialog v-model="showPhoneApiKeyQrCode" max-width="600" opacity="0.9">
      <VCard>
        <VCardTitle>Phone API Key QR Code</VCardTitle>
        <VCardSubtitle class="mt-2" style="white-space: normal">
          Scan this QR code with the
          <a
            class="text-decoration-none hover:text-decoration-underline"
            target="_blank"
            :href="appStore.appData.appDownloadUrl"
            >httpSMS app</a
          >
          on your Android phone to login.
        </VCardSubtitle>
        <VCardText class="text-center">
          <VTextField
            :model-value="activePhoneApiKey?.api_key"
            readonly
            name="api-key"
            variant="outlined"
          />
          <canvas ref="qrCodeCanvas"></canvas>
        </VCardText>
        <VCardActions>
          <CopyButton
            :value="activePhoneApiKey?.api_key ?? ''"
            color="primary"
            copy-text="Copy API key"
            notification-text="Phone API Key copied successfully"
          />
          <VSpacer />
          <VBtn
            color="warning"
            variant="text"
            @click="showPhoneApiKeyQrCode = false"
          >
            Close
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>

    <VDialog v-model="deleteApiKeyDialog" max-width="600" opacity="0.9">
      <VCard>
        <VCardTitle class="text-h5 text-break">
          Are you sure you want to delete the phone API Key?
        </VCardTitle>
        <VCardText class="text-medium-emphasis">
          You will have to logout and login again on the <b>httpSMS</b> Android
          app on all of the phones which are currently using this API key.
        </VCardText>
        <VCardActions class="pb-2 mt-n2">
          <VBtn
            color="error"
            variant="flat"
            :loading="loading"
            @click="deleteApiKey"
          >
            <VIcon start :icon="mdiDelete" />
            Delete API Key
          </VBtn>
          <VSpacer />
          <VBtn
            variant="text"
            color="warning"
            @click="deleteApiKeyDialog = false"
            >Close</VBtn
          >
        </VCardActions>
      </VCard>
    </VDialog>

    <VDialog
      v-model="removePhoneFromApiKeyDialog"
      max-width="600"
      opacity="0.9"
    >
      <VCard>
        <VCardTitle class="text-h5 text-break">
          Are you sure you want to remove this phone number from the Phone API
          Key?
        </VCardTitle>
        <VCardText>
          This will remove the
          <code>{{ formatPhoneNumber(activePhoneNumber) }}</code> from your
          phone API key. You will have to logout and login again on the
          <b>httpSMS</b> Android app on the phone which is currently using this
          API key.
        </VCardText>
        <VCardActions class="pb-4">
          <VBtn
            color="error"
            :loading="loading"
            @click="removePhoneFromPhoneKey"
          >
            <VIcon start :icon="mdiDelete" />
            Remove Phone from key
          </VBtn>
          <VSpacer />
          <VBtn
            variant="text"
            color="warning"
            @click="removePhoneFromApiKeyDialog = false"
          >
            Close
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>
  </VContainer>
</template>

<style scoped lang="scss">
.api-key-table {
  tbody {
    tr:hover {
      background-color: transparent !important;
    }
  }
}
</style>
