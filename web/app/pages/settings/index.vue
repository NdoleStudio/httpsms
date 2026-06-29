<script setup lang="ts">
import {
  mdiArrowLeft,
  mdiAccountCircle,
  mdiShieldCheck,
  mdiEye,
  mdiEyeOff,
  mdiQrcode,
  mdiRefresh,
  mdiLinkVariant,
  mdiSquareEditOutline,
  mdiDelete,
  mdiContentSave,
  mdiConnection,
  mdiCalendarClock,
  mdiPlus,
} from '@mdi/js'
import {
  getAuth,
  sendEmailVerification,
  signOut,
  type User as FirebaseUser,
} from 'firebase/auth'
import QRCode from 'qrcode'
import { ErrorMessages } from '~/utils/errors'
import { toApiError } from '~/utils/api-error'
import type {
  EntitiesPhone,
  EntitiesWebhook,
  EntitiesDiscord,
  EntitiesMessageSendSchedule,
} from '~~/shared/types/api'

definePageMeta({
  middleware: ['auth'],
})

useHead({
  title: 'Settings - httpSMS',
})

const config = useRuntimeConfig()
const route = useRoute()
const router = useRouter()
const { mdAndDown, mdAndUp, lgAndUp, xlAndUp, smAndUp } = useDisplay()
const authStore = useAuthStore()
const phonesStore = usePhonesStore()
const billingStore = useBillingStore()
const notificationsStore = useNotificationsStore()

const firebaseUser = ref<FirebaseUser | null>(null)
const gravatarUrl = ref<string | null>(null)
const sendingVerificationEmail = ref(false)
const verificationEmailSent = ref(false)

async function sendVerificationEmail() {
  if (!firebaseUser.value) return
  sendingVerificationEmail.value = true
  try {
    await sendEmailVerification(firebaseUser.value)
    verificationEmailSent.value = true
    notificationsStore.addNotification({
      message: 'Verification email sent. Please check your inbox.',
      type: 'success',
    })
  } catch (error) {
    console.error('sendEmailVerification failed:', error)
    notificationsStore.addNotification({
      message: 'Failed to send verification email. Please try again later.',
      type: 'error',
    })
  } finally {
    sendingVerificationEmail.value = false
  }
}

const computeGravatarUrl = async (email: string): Promise<string> => {
  const normalized = email.trim().toLowerCase()
  const data = new TextEncoder().encode(normalized)
  const digest = await crypto.subtle.digest('SHA-256', data)
  const hash = Array.from(new Uint8Array(digest))
    .map((byte) => byte.toString(16).padStart(2, '0'))
    .join('')
  return `https://www.gravatar.com/avatar/${hash}?d=identicon&s=200`
}

const avatarUrl = computed(
  () => firebaseUser.value?.photoURL ?? gravatarUrl.value,
)

const apiKeyShow = ref(false)
const showQrCodeDialog = ref(false)
const showRotateApiKey = ref(false)
const rotatingApiKey = ref(false)
const qrCodeCanvas = ref<HTMLCanvasElement | null>(null)

const errorMessages = ref(new ErrorMessages())

// Timezones
const timezones = (() => {
  try {
    const intlWithTimeZones = Intl as typeof Intl & {
      supportedValuesOf(key: string): string[]
    }
    return intlWithTimeZones.supportedValuesOf('timeZone')
  } catch {
    return [] as string[]
  }
})()

const apiKey = computed(() => authStore.user?.api_key ?? '')

const hasActiveSubscription = computed(() => {
  if (authStore.user === null) return true
  return authStore.user.subscription_renews_at != null
})

const phoneNumbers = computed(() =>
  phonesStore.phones.map((phone) => phone.phone_number),
)

const webhookEventOptions = [
  'message.phone.received',
  'message.phone.sent',
  'message.phone.delivered',
  'message.send.failed',
  'message.send.expired',
  'message.call.missed',
  'phone.heartbeat.offline',
  'phone.heartbeat.online',
]

function resetErrors() {
  errorMessages.value = new ErrorMessages()
}

function parseErrors(error: unknown): ErrorMessages {
  const bag = new ErrorMessages()
  const data = toApiError(error).data?.data
  if (data && typeof data === 'object') {
    Object.keys(data).forEach((key) => bag.addMany(key, data[key]!))
  }
  return bag
}

// ---------------------------------------------------------------------------
// API Key
// ---------------------------------------------------------------------------
async function rotateApiKey() {
  if (!authStore.user) return
  rotatingApiKey.value = true
  try {
    await authStore.rotateApiKey(authStore.user.id)
    showRotateApiKey.value = false
    notificationsStore.addNotification({
      message: 'API Key rotated successfully',
      type: 'success',
    })
  } catch {
    notificationsStore.addNotification({
      message: 'Failed to rotate API Key',
      type: 'error',
    })
  } finally {
    rotatingApiKey.value = false
  }
}

function generateQrCode() {
  showQrCodeDialog.value = true
  nextTick(() => {
    if (qrCodeCanvas.value) {
      QRCode.toCanvas(
        qrCodeCanvas.value,
        apiKey.value,
        { errorCorrectionLevel: 'H', width: 300, margin: 2 },
        (err) => {
          if (err) {
            notificationsStore.addNotification({
              message: 'Failed to generate API key QR code',
              type: 'error',
            })
          }
        },
      )
    }
  })
}

// ---------------------------------------------------------------------------
// Timezone
// ---------------------------------------------------------------------------
async function updateTimezone(timezone: string) {
  try {
    await authStore.updateUser({ timezone })
    notificationsStore.addNotification({
      message: 'Timezone updated successfully',
      type: 'success',
    })
  } catch {
    notificationsStore.addNotification({
      message: 'Failed to update timezone',
      type: 'error',
    })
  }
}

// ---------------------------------------------------------------------------
// Webhooks
// ---------------------------------------------------------------------------
const loadingWebhooks = ref(true)
const webhooks = ref<EntitiesWebhook[]>([])
const updatingWebhook = ref(false)
const showWebhookEdit = ref(false)
const activeWebhook = ref<{
  id: string | null
  url: string
  signing_key: string
  phone_numbers: string[]
  events: string[]
}>({
  id: null,
  url: '',
  signing_key: '',
  phone_numbers: [],
  events: ['message.phone.received'],
})

async function loadWebhooks() {
  loadingWebhooks.value = true
  try {
    webhooks.value = await billingStore.getWebhooks()
  } finally {
    loadingWebhooks.value = false
  }
}

function onWebhookCreate() {
  resetErrors()
  activeWebhook.value = {
    id: null,
    url: '',
    signing_key: '',
    phone_numbers: phoneNumbers.value.slice(),
    events: [
      'message.phone.received',
      'message.phone.sent',
      'message.phone.delivered',
      'message.send.failed',
      'message.send.expired',
    ],
  }
  showWebhookEdit.value = true
}

function onWebhookEdit(webhookId: string) {
  const webhook = webhooks.value.find((x) => x.id === webhookId)
  if (!webhook) return
  resetErrors()
  activeWebhook.value = {
    id: webhook.id,
    url: webhook.url,
    signing_key: webhook.signing_key,
    phone_numbers: (webhook.phone_numbers ?? []).filter((x) =>
      phoneNumbers.value.includes(x),
    ),
    events: webhook.events ?? [],
  }
  showWebhookEdit.value = true
}

async function saveWebhook() {
  resetErrors()
  updatingWebhook.value = true
  try {
    const payload = {
      url: activeWebhook.value.url,
      signing_key: activeWebhook.value.signing_key,
      phone_numbers: activeWebhook.value.phone_numbers,
      events: activeWebhook.value.events,
    }
    if (activeWebhook.value.id) {
      await billingStore.updateWebhook({
        id: activeWebhook.value.id,
        ...payload,
      })
    } else {
      await billingStore.createWebhook(payload)
    }
    notificationsStore.addNotification({
      message: `Webhook ${activeWebhook.value.id ? 'updated' : 'created'} successfully`,
      type: 'success',
    })
    showWebhookEdit.value = false
    await loadWebhooks()
  } catch (error: unknown) {
    errorMessages.value = parseErrors(error)
    if (errorMessages.value.size() === 0) {
      notificationsStore.addNotification({
        message: 'Failed to save webhook',
        type: 'error',
      })
    }
  } finally {
    updatingWebhook.value = false
  }
}

async function deleteWebhook(id: string) {
  updatingWebhook.value = true
  try {
    await billingStore.deleteWebhook(id)
    notificationsStore.addNotification({
      message: 'Webhook deleted successfully',
      type: 'success',
    })
    showWebhookEdit.value = false
    await loadWebhooks()
  } catch {
    notificationsStore.addNotification({
      message: 'Failed to delete webhook',
      type: 'error',
    })
  } finally {
    updatingWebhook.value = false
  }
}

// ---------------------------------------------------------------------------
// Discord
// ---------------------------------------------------------------------------
const loadingDiscordIntegrations = ref(true)
const discords = ref<EntitiesDiscord[]>([])
const updatingDiscord = ref(false)
const showDiscordEdit = ref(false)
const activeDiscord = ref<{
  id: string | null
  name: string
  server_id: string
  incoming_channel_id: string
}>({
  id: null,
  name: '',
  server_id: '',
  incoming_channel_id: '',
})

async function loadDiscordIntegrations() {
  loadingDiscordIntegrations.value = true
  try {
    discords.value = await billingStore.getDiscordIntegrations()
  } finally {
    loadingDiscordIntegrations.value = false
  }
}

function onDiscordCreate() {
  resetErrors()
  activeDiscord.value = {
    id: null,
    name: '',
    server_id: '',
    incoming_channel_id: '',
  }
  showDiscordEdit.value = true
}

function onDiscordEdit(discordId: string) {
  const discord = discords.value.find((x) => x.id === discordId)
  if (!discord) return
  resetErrors()
  activeDiscord.value = {
    id: discord.id,
    name: discord.name,
    server_id: discord.server_id,
    incoming_channel_id: discord.incoming_channel_id,
  }
  showDiscordEdit.value = true
}

async function saveDiscord() {
  resetErrors()
  updatingDiscord.value = true
  try {
    const payload = {
      name: activeDiscord.value.name,
      server_id: activeDiscord.value.server_id,
      incoming_channel_id: activeDiscord.value.incoming_channel_id,
    }
    if (activeDiscord.value.id) {
      await billingStore.updateDiscordIntegration({
        id: activeDiscord.value.id,
        ...payload,
      })
    } else {
      await billingStore.createDiscord(payload)
    }
    notificationsStore.addNotification({
      message: `Discord integration ${activeDiscord.value.id ? 'updated' : 'created'} successfully`,
      type: 'success',
    })
    showDiscordEdit.value = false
    await loadDiscordIntegrations()
  } catch (error: unknown) {
    errorMessages.value = parseErrors(error)
    if (errorMessages.value.size() === 0) {
      notificationsStore.addNotification({
        message: 'Failed to save discord integration',
        type: 'error',
      })
    }
  } finally {
    updatingDiscord.value = false
  }
}

async function deleteDiscord(id: string) {
  updatingDiscord.value = true
  try {
    await billingStore.deleteDiscordIntegration(id)
    notificationsStore.addNotification({
      message: 'Discord integration deleted successfully',
      type: 'success',
    })
    showDiscordEdit.value = false
    await loadDiscordIntegrations()
  } catch {
    notificationsStore.addNotification({
      message: 'Failed to delete discord integration',
      type: 'error',
    })
  } finally {
    updatingDiscord.value = false
  }
}

// ---------------------------------------------------------------------------
// Phones
// ---------------------------------------------------------------------------
const updatingPhone = ref(false)
const showPhoneEdit = ref(false)
const activePhone = ref<EntitiesPhone | null>(null)

function showEditPhone(phoneId: string) {
  const phone = phonesStore.phones.find((x) => x.id === phoneId)
  if (!phone) return
  resetErrors()
  activePhone.value = { ...phone }
  showPhoneEdit.value = true
}

async function updatePhone() {
  if (!activePhone.value) return
  updatingPhone.value = true
  try {
    await phonesStore.updatePhone(activePhone.value)
    showPhoneEdit.value = false
    activePhone.value = null
  } finally {
    updatingPhone.value = false
  }
}

async function deletePhone(phoneId: string) {
  updatingPhone.value = true
  try {
    await phonesStore.deletePhone(phoneId)
    notificationsStore.addNotification({
      message: 'Phone deleted successfully',
      type: 'success',
    })
    showPhoneEdit.value = false
    activePhone.value = null
  } catch {
    notificationsStore.addNotification({
      message: 'Failed to delete phone',
      type: 'error',
    })
  } finally {
    updatingPhone.value = false
  }
}

// ---------------------------------------------------------------------------
// Send Schedules
// ---------------------------------------------------------------------------
const loadingSendSchedules = ref(true)
const sendSchedules = ref<EntitiesMessageSendSchedule[]>([])
const showScheduleEdit = ref(false)
const showScheduleDelete = ref(false)
const savingSchedule = ref(false)
const activeSchedule = ref<{
  id: string | null
  name: string
  timezone: string
  windows: Array<{
    day_of_week: number
    start_time: string
    end_time: string
  }>
}>({
  id: null,
  name: '',
  timezone: '',
  windows: [],
})

const weekDays = [
  { value: 1, label: 'Monday' },
  { value: 2, label: 'Tuesday' },
  { value: 3, label: 'Wednesday' },
  { value: 4, label: 'Thursday' },
  { value: 5, label: 'Friday' },
  { value: 6, label: 'Saturday' },
  { value: 0, label: 'Sunday' },
]

async function loadSendSchedules() {
  loadingSendSchedules.value = true
  try {
    sendSchedules.value = await billingStore.getSendSchedules()
  } finally {
    loadingSendSchedules.value = false
  }
}

function minuteToClock(value: number): string {
  const hours = String(Math.floor(value / 60)).padStart(2, '0')
  const minutes = String(value % 60).padStart(2, '0')
  return `${hours}:${minutes}`
}

function clockToMinute(value: string): number {
  if (!value || !value.includes(':')) return 0
  const [hours = 0, minutes = 0] = value.split(':').map((x) => parseInt(x, 10))
  return hours * 60 + minutes
}

function getWeekday(index: number): string {
  return weekDays.find((x) => x.value === index)?.label ?? ''
}

function scheduleSummary(schedule: EntitiesMessageSendSchedule): string[][] {
  return weekDays
    .map((day) => {
      const windows = (schedule.windows || []).filter(
        (x) => x.day_of_week === day.value,
      )
      if (windows.length === 0) return []
      return [
        day.label,
        windows
          .map(
            (w) =>
              `${minuteToClock(w.start_minute)} - ${minuteToClock(w.end_minute)}`,
          )
          .join(', '),
      ]
    })
    .filter((x) => x.length > 0)
}

function defaultTimezone(): string {
  return (
    authStore.user?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone
  )
}

function openCreateSchedule() {
  resetErrors()
  activeSchedule.value = {
    id: null,
    name: '',
    timezone: defaultTimezone(),
    windows: [1, 2, 3, 4, 5].map((day) => ({
      day_of_week: day,
      start_time: '09:00',
      end_time: '17:00',
    })),
  }
  showScheduleEdit.value = true
}

function openEditSchedule(schedule: EntitiesMessageSendSchedule) {
  resetErrors()
  activeSchedule.value = {
    id: schedule.id,
    name: schedule.name,
    timezone: schedule.timezone,
    windows: (schedule.windows || []).map((x) => ({
      day_of_week: x.day_of_week,
      start_time: minuteToClock(x.start_minute),
      end_time: minuteToClock(x.end_minute),
    })),
  }
  showScheduleEdit.value = true
}

function scheduleWindowsForDay(dayOfWeek: number) {
  return activeSchedule.value.windows.filter((x) => x.day_of_week === dayOfWeek)
}

function scheduleDayEnabled(dayOfWeek: number): boolean {
  return scheduleWindowsForDay(dayOfWeek).length > 0
}

function scheduleToggleDay(dayOfWeek: number, enabled: boolean | null) {
  if (enabled) {
    if (!scheduleDayEnabled(dayOfWeek)) {
      scheduleAddWindow(dayOfWeek)
    }
    return
  }
  activeSchedule.value.windows = activeSchedule.value.windows.filter(
    (x) => x.day_of_week !== dayOfWeek,
  )
}

function scheduleAddWindow(dayOfWeek: number) {
  activeSchedule.value.windows.push({
    day_of_week: dayOfWeek,
    start_time: '09:00',
    end_time: '17:00',
  })
}

function scheduleRemoveWindow(dayOfWeek: number, index: number) {
  const matches = activeSchedule.value.windows.filter(
    (x) => x.day_of_week === dayOfWeek,
  )
  const target = matches[index]
  activeSchedule.value.windows = activeSchedule.value.windows.filter(
    (x) => x !== target,
  )
}

function scheduleWindowError(index: number): string | null {
  const messages = errorMessages.value.has('windows')
    ? errorMessages.value.get('windows')
    : []
  if (messages.length === 0) return null
  const message = messages.find((x) => x.includes(`day_of_week ${index}`))
  return message
    ? message.replace(`day_of_week ${index}`, getWeekday(index))
    : null
}

async function saveSchedule() {
  resetErrors()
  savingSchedule.value = true
  try {
    const payload = {
      name: activeSchedule.value.name,
      timezone: activeSchedule.value.timezone,
      windows: (activeSchedule.value.windows || []).map((window) => ({
        day_of_week: window.day_of_week,
        start_minute: clockToMinute(window.start_time),
        end_minute: clockToMinute(window.end_time),
      })),
    }
    if (activeSchedule.value.id) {
      await billingStore.updateSendSchedule({
        id: activeSchedule.value.id,
        ...payload,
      })
    } else {
      await billingStore.createSendSchedule(payload)
    }
    notificationsStore.addNotification({
      type: 'success',
      message: 'Send schedule saved successfully',
    })
    showScheduleEdit.value = false
    await loadSendSchedules()
  } catch (error: unknown) {
    errorMessages.value = parseErrors(error)
    if (errorMessages.value.size() != 0) {
      notificationsStore.addNotification({
        type: 'error',
        message: 'Failed to save send schedule',
      })
    }
  } finally {
    savingSchedule.value = false
  }
}

function confirmDeleteSchedule() {
  showScheduleDelete.value = true
}

async function deleteSchedule() {
  if (!activeSchedule.value.id) return
  savingSchedule.value = true
  try {
    await billingStore.deleteSendSchedule(activeSchedule.value.id)
    notificationsStore.addNotification({
      type: 'success',
      message: 'Send schedule deleted successfully',
    })
    showScheduleDelete.value = false
    showScheduleEdit.value = false
    await loadSendSchedules()
  } catch {
    notificationsStore.addNotification({
      type: 'error',
      message: 'Failed to delete send schedule',
    })
  } finally {
    savingSchedule.value = false
  }
}

// ---------------------------------------------------------------------------
// Email Notifications
// ---------------------------------------------------------------------------
const updatingEmailNotifications = ref(false)
const notificationSettings = ref({
  webhook_enabled: true,
  message_status_enabled: true,
  newsletter_enabled: true,
  heartbeat_enabled: true,
})

function syncEmailNotifications() {
  if (!authStore.user) return
  notificationSettings.value = {
    webhook_enabled: authStore.user.notification_webhook_enabled,
    message_status_enabled: authStore.user.notification_message_status_enabled,
    heartbeat_enabled: authStore.user.notification_heartbeat_enabled,
    newsletter_enabled: authStore.user.notification_newsletter_enabled,
  }
}

async function saveEmailNotifications() {
  if (!authStore.user) return
  updatingEmailNotifications.value = true
  try {
    await billingStore.saveEmailNotifications(
      authStore.user.id,
      notificationSettings.value,
    )
    notificationsStore.addNotification({
      message: 'Email notifications saved successfully',
      type: 'success',
    })
    syncEmailNotifications()
  } catch {
    notificationsStore.addNotification({
      message: 'Failed to save email notifications',
      type: 'error',
    })
  } finally {
    updatingEmailNotifications.value = false
  }
}

// ---------------------------------------------------------------------------
// Delete account
// ---------------------------------------------------------------------------
const deletingAccount = ref(false)
const showDeleteAccountDialog = ref(false)

async function deleteUserAccount() {
  deletingAccount.value = true
  try {
    const message = await authStore.deleteUserAccount()
    notificationsStore.addNotification({
      message: message ?? 'Your account has been deleted successfully',
      type: 'success',
    })
    const auth = getAuth()
    await signOut(auth)
    authStore.resetState()
    phonesStore.resetState()
    notificationsStore.addNotification({
      type: 'info',
      message: 'You have successfully logged out',
    })
    await router.push({ name: 'index' })
  } catch {
    notificationsStore.addNotification({
      message: 'Failed to delete your account',
      type: 'error',
    })
  } finally {
    deletingAccount.value = false
    showDeleteAccountDialog.value = false
  }
}

watch(showQrCodeDialog, (open) => {
  if (open && apiKey.value) {
    nextTick(() => generateQrCode())
  }
})

onMounted(async () => {
  firebaseUser.value = getAuth().currentUser
  if (firebaseUser.value?.email) {
    gravatarUrl.value = await computeGravatarUrl(firebaseUser.value.email)
  }
  await Promise.all([authStore.loadUser(), phonesStore.loadPhones()])
  syncEmailNotifications()
  loadWebhooks()
  loadDiscordIntegrations()
  loadSendSchedules()
  if (route.hash) {
    nextTick(() => {
      const el = document.querySelector(route.hash)
      if (el) el.scrollIntoView({ behavior: 'smooth' })
    })
  }
})
</script>

<template>
  <VContainer fluid :class="{ 'fill-height': lgAndUp }">
    <div class="w-100 h-100">
      <VAppBar>
        <VBtn icon to="/threads">
          <VIcon :icon="mdiArrowLeft" />
        </VBtn>
        <VToolbarTitle>Settings</VToolbarTitle>
      </VAppBar>
      <VContainer class="pa-0">
        <VRow>
          <VCol cols="12" md="9" offset-md="1" xl="8" offset-xl="2">
            <!-- Profile -->
            <div v-if="firebaseUser" class="text-center">
              <VAvatar v-if="avatarUrl" size="100" :image="avatarUrl" />
              <v-avatar v-else size="100">
                <VIcon size="80" :icon="mdiAccountCircle" />
              </v-avatar>

              <h3
                v-if="firebaseUser.displayName"
                class="text-title-large mt-2 mb-0"
              >
                {{ firebaseUser.displayName }}
              </h3>
              <h4 class="text-medium-emphasis mb-2 mt-0">
                {{ firebaseUser.email }}
                <VIcon
                  v-if="firebaseUser.emailVerified"
                  size="small"
                  color="primary"
                  :icon="mdiShieldCheck"
                />
                <VBtn
                  v-else
                  size="x-small"
                  variant="tonal"
                  color="warning"
                  :loading="sendingVerificationEmail"
                  :disabled="verificationEmailSent"
                  @click="sendVerificationEmail"
                >
                  Verify Email
                </VBtn>
              </h4>
              <VAutocomplete
                v-if="authStore.user"
                density="compact"
                variant="outlined"
                :model-value="authStore.user.timezone"
                class="mx-auto mt-2"
                style="max-width: 250px"
                label="Timezone"
                :items="timezones"
                @update:model-value="updateTimezone"
              />
            </div>

            <!-- API Key -->
            <h5 class="text-headline-large mb-3 mt-0">API Key</h5>
            <p class="text-medium-emphasis">
              Use your API Key in the <v-code>x-api-key</v-code> HTTP Header
              when sending requests to
              <v-code>https://api.httpsms.com</v-code> endpoints.
            </p>
            <div v-if="apiKey === ''" class="mb-n9 pl-3 pt-5">
              <VProgressCircular
                :size="20"
                :width="2"
                color="primary"
                indeterminate
              />
            </div>
            <VTextField
              v-else
              :append-inner-icon="apiKeyShow ? mdiEye : mdiEyeOff"
              :type="apiKeyShow ? 'text' : 'password'"
              :model-value="apiKey"
              readonly
              name="api-key"
              variant="outlined"
              class="mb-n2"
              @click:append-inner="apiKeyShow = !apiKeyShow"
            />
            <div class="d-flex flex-wrap">
              <CopyButton
                :value="apiKey"
                color="primary"
                copy-text="Copy API Key"
                notification-text="API Key copied successfully"
              />
              <VBtn
                v-if="mdAndUp"
                color="primary"
                class="ml-4"
                @click="generateQrCode"
              >
                <VIcon start :icon="mdiQrcode" />
                Show QR Code
              </VBtn>
              <VDialog
                v-model="showQrCodeDialog"
                max-width="400px"
                opacity="0.9"
              >
                <VCard>
                  <VCardTitle class="text-center">API Key QR Code</VCardTitle>
                  <VCardText class="text-center">
                    <p class="text-body-large mt-0">
                      Scan this QR code with the
                      <a
                        class="text-decoration-none hover:text-decoration-underline"
                        :href="config.public.appDownloadUrl"
                        >httpSMS app</a
                      >
                      on your Android phone to login.
                    </p>
                    <canvas ref="qrCodeCanvas" />
                  </VCardText>
                  <VCardActions>
                    <VBtn
                      color="primary"
                      block
                      variant="flat"
                      class="mt-n4"
                      @click="showQrCodeDialog = false"
                      >Close</VBtn
                    >
                  </VCardActions>
                </VCard>
              </VDialog>
              <VBtn
                v-if="lgAndUp"
                class="ml-4"
                :href="config.public.appDocumentationUrl"
                >Documentation</VBtn
              >
              <VSpacer />
              <VDialog v-model="showRotateApiKey" max-width="550">
                <template #activator="{ props }">
                  <VBtn
                    :size="mdAndDown ? 'small' : 'default'"
                    :variant="lgAndUp ? 'text' : 'elevated'"
                    color="warning"
                    v-bind="props"
                  >
                    <VIcon start :icon="mdiRefresh" />
                    Rotate API Key
                  </VBtn>
                </template>
                <VCard>
                  <VCardTitle class="text-headline-medium text-break"
                    >Are you sure you want to rotate your API Key?</VCardTitle
                  >
                  <VCardText>
                    You will have to logout and login again on the
                    <b>httpSMS</b> Android app with your new API key after you
                    rotate it.
                  </VCardText>
                  <VCardActions class="pb-4">
                    <VBtn
                      color="primary"
                      :loading="rotatingApiKey"
                      @click="rotateApiKey"
                    >
                      <VIcon start :icon="mdiRefresh" />
                      Yes Rotate Key
                    </VBtn>
                    <VSpacer />
                    <VBtn variant="text" @click="showRotateApiKey = false"
                      >Close</VBtn
                    >
                  </VCardActions>
                </VCard>
              </VDialog>
            </div>

            <!-- Webhooks -->
            <h5 id="webhook-settings" class="text-headline-large mb-3 mt-12">
              Webhooks
            </h5>
            <p class="text-medium-emphasis">
              Webhooks allow us to send events to your server for example when
              the android phone receives an SMS message we can forward the
              message to your server.
            </p>
            <div v-if="loadingWebhooks">
              <VProgressCircular
                :size="60"
                :width="2"
                color="primary"
                class="mb-4"
                indeterminate
              />
            </div>
            <VTable v-else-if="webhooks.length" class="mb-4">
              <thead>
                <tr class="text-uppercase text-title-medium">
                  <th v-if="xlAndUp" class="text-left">ID</th>
                  <th class="text-left text-break">Callback URL</th>
                  <th v-if="lgAndUp" class="text-center">Events</th>
                  <th class="text-center">Action</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="webhook in webhooks" :key="webhook.id">
                  <td v-if="xlAndUp" class="text-left">{{ webhook.id }}</td>
                  <td class="text-break">{{ webhook.url }}</td>
                  <td v-if="lgAndUp" class="text-center">
                    <VChip
                      v-for="event in webhook.events"
                      :key="event"
                      class="ma-1"
                      size="small"
                      >{{ event }}</VChip
                    >
                  </td>
                  <td class="text-center">
                    <VBtn
                      :icon="mdAndDown"
                      size="small"
                      color="info"
                      :disabled="updatingWebhook"
                      @click.prevent="onWebhookEdit(webhook.id)"
                    >
                      <VIcon size="small" :icon="mdiSquareEditOutline" />
                      <span v-if="!mdAndDown" class="ml-1">Edit</span>
                    </VBtn>
                  </td>
                </tr>
              </tbody>
            </VTable>
            <div class="d-flex">
              <VBtn color="primary" @click="onWebhookCreate">
                <VIcon start :icon="mdiLinkVariant" />
                Add webhook
              </VBtn>
              <VBtn
                v-if="lgAndUp"
                class="ml-4"
                href="https://docs.httpsms.com/webhooks/introduction"
                >Documentation</VBtn
              >
            </div>

            <!-- Discord Integration -->
            <h5 id="discord-settings" class="text-headline-large mb-3 mt-12">
              Discord Integration
            </h5>
            <p class="text-medium-emphasis">
              Send and receive SMS messages without leaving your discord server
              with the httpSMS discord app using the
              <v-code>/httpsms</v-code> command.
            </p>
            <div v-if="loadingDiscordIntegrations">
              <VProgressCircular
                :size="60"
                :width="2"
                color="primary"
                class="mb-4"
                indeterminate
              />
            </div>
            <VTable v-else-if="discords.length" class="mb-4">
              <thead>
                <tr class="text-uppercase text-title-medium">
                  <th class="text-left">Name</th>
                  <th class="text-left">Server ID</th>
                  <th class="text-left">Channel ID</th>
                  <th class="text-center">Action</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="discord in discords" :key="discord.id">
                  <td class="text-left">{{ discord.name }}</td>
                  <td class="text-left">{{ discord.server_id }}</td>
                  <td class="text-left">{{ discord.incoming_channel_id }}</td>
                  <td class="text-center">
                    <VBtn
                      :icon="mdAndDown"
                      size="small"
                      color="info"
                      :disabled="updatingDiscord"
                      @click.prevent="onDiscordEdit(discord.id)"
                    >
                      <VIcon size="small" :icon="mdiSquareEditOutline" />
                      <span v-if="!mdAndDown" class="ml-1">Edit</span>
                    </VBtn>
                  </td>
                </tr>
              </tbody>
            </VTable>
            <VBtn color="primary" @click="onDiscordCreate">
              <VIcon start :icon="mdiConnection" />
              Add Discord Integration
            </VBtn>

            <!-- Phones -->
            <h5 id="phones" class="text-headline-large mb-3 mt-12">Phones</h5>
            <p class="text-medium-emphasis">
              List of mobile phones which are registered for sending and
              receiving SMS messages.
            </p>
            <VTable class="mb-4" density="comfortable">
              <thead>
                <tr class="text-uppercase text-medium-emphasis">
                  <th v-if="xlAndUp" class="text-left">ID</th>
                  <th class="text-left">Phone Number</th>
                  <th v-if="lgAndUp" class="text-center">Retries</th>
                  <th class="text-center">Rate</th>
                  <th class="text-center">Updated At</th>
                  <th class="text-center">Action</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="phone in phonesStore.phones" :key="phone.id">
                  <td v-if="xlAndUp" class="text-left">{{ phone.id }}</td>
                  <td>
                    {{ useFilters().formatPhoneNumber(phone.phone_number) }}
                  </td>
                  <td v-if="lgAndUp" class="text-center">
                    {{ phone.max_send_attempts ? phone.max_send_attempts : 1 }}
                  </td>
                  <td class="text-center">
                    <span v-if="phone.messages_per_minute"
                      >{{ phone.messages_per_minute }}/min</span
                    >
                    <span v-else>Unlimited</span>
                  </td>
                  <td class="text-center">
                    {{ useFilters().formatTimestamp(phone.updated_at) }}
                  </td>
                  <td class="text-center">
                    <VBtn
                      :icon="mdAndDown"
                      size="small"
                      color="info"
                      :disabled="updatingPhone"
                      @click.prevent="showEditPhone(phone.id)"
                    >
                      <VIcon size="small" :icon="mdiSquareEditOutline" />
                      <span v-if="!mdAndDown" class="ml-1">Edit</span>
                    </VBtn>
                  </td>
                </tr>
              </tbody>
            </VTable>

            <!-- Send Schedules -->
            <h5 id="send-schedules" class="text-headline-large mb-3 mt-12">
              Send Schedules
            </h5>
            <p class="text-medium-emphasis">
              Create availability schedules and attach them to each phone.
              Outgoing messages sent outside the schedule window are queued and
              delivered when the schedule opens according to your
              <a
                class="text-decoration-none"
                href="https://docs.httpsms.com/features/outgoing-message-queue#id-3.-send-schedule-window"
                >configured send rate</a
              >.
            </p>
            <div v-if="loadingSendSchedules">
              <VProgressCircular
                :size="60"
                :width="2"
                color="primary"
                class="mb-4"
                indeterminate
              />
            </div>
            <VTable class="mb-4" density="comfortable">
              <thead>
                <tr class="text-uppercase text-medium-emphasis">
                  <th class="text-left">Name</th>
                  <th class="text-left">Timezone</th>
                  <th class="text-left">Schedule</th>
                  <th class="text-center">Action</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="schedule in sendSchedules" :key="schedule.id">
                  <td class="text-left pt-2" style="vertical-align: top">
                    {{ schedule.name }}
                  </td>
                  <td class="pt-2" style="vertical-align: top">
                    {{ schedule.timezone }}
                  </td>
                  <td class="py-2">
                    <div
                      v-for="line in scheduleSummary(schedule)"
                      :key="`${schedule.id}-${line[0]}`"
                      class="mb-1"
                    >
                      {{ line[0] }}:
                      <span class="text-medium-emphasis">{{ line[1] }}</span>
                    </div>
                  </td>
                  <td class="text-center pt-2" style="vertical-align: top">
                    <VBtn
                      :icon="mdAndDown"
                      size="small"
                      color="info"
                      :disabled="loadingSendSchedules"
                      @click.prevent="openEditSchedule(schedule)"
                    >
                      <VIcon size="small" :icon="mdiSquareEditOutline" />
                      <span v-if="!mdAndDown" class="ml-1">Edit</span>
                    </VBtn>
                  </td>
                </tr>
              </tbody>
            </VTable>
            <div class="d-flex mt-4">
              <VBtn color="primary" @click="openCreateSchedule">
                <VIcon start :icon="mdiCalendarClock" />
                Create Send Schedule
              </VBtn>
              <VBtn
                v-if="lgAndUp"
                class="ml-4"
                href="https://docs.httpsms.com/features/outgoing-message-queue"
                >Documentation</VBtn
              >
            </div>

            <!-- Email Notifications -->
            <h5 id="email-notifications" class="text-headline-large mb-3 mt-12">
              Email Notifications
            </h5>
            <p class="text-medium-emphasis">
              Manage the email notifications which you receive from httpSMS.
              Feel free to turn on/off individual notifications anytime so you
              don't get overloaded with emails
            </p>
            <VSwitch
              v-model="notificationSettings.heartbeat_enabled"
              color="primary"
              label="Heartbeat emails"
              :disabled="updatingEmailNotifications"
              hint="This switch controls email notifications we send when we don't receive a heartbeat from your phone for 1 hour."
              persistent-hint
            />
            <VSwitch
              v-model="notificationSettings.webhook_enabled"
              color="primary"
              label="Webhook and discord emails"
              :disabled="updatingEmailNotifications"
              hint="This switch controls email notifications we send when we can't forward events to your discord server or to your webhook."
              persistent-hint
            />
            <VSwitch
              v-model="notificationSettings.message_status_enabled"
              color="primary"
              label="Message status emails"
              :disabled="updatingEmailNotifications"
              hint="This switch controls email notifications we send when your message is failed or expired."
              persistent-hint
            />
            <VSwitch
              v-model="notificationSettings.newsletter_enabled"
              color="primary"
              label="Newsletter emails"
              :disabled="updatingEmailNotifications"
              hint="This switch controls newsletter emails about new features, updates, and promotions."
              persistent-hint
            />
            <VBtn
              color="primary"
              :loading="updatingEmailNotifications"
              class="mt-4"
              @click="saveEmailNotifications"
            >
              <VIcon start :icon="mdiContentSave" />
              Save Notification Settings
            </VBtn>

            <!-- Message Data Retention -->
            <h5 class="text-headline-large mb-3 mt-12">
              Message Data Retention
            </h5>
            <p class="text-medium-emphasis">
              Your messages are permanently deleted once they exceed the max
              retention period below, counted from when the message was sent or
              received. You can always delete your messages manually on the
              <NuxtLink class="text-decoration-none" to="/search-messages"
                >message search page.</NuxtLink
              >
            </p>
            <VSelect
              :items="['1 Year']"
              model-value="1 Year"
              label="Retention Period"
              variant="outlined"
              density="compact"
              class="mt-4"
              style="max-width: 300px"
            />

            <!-- Delete Account -->
            <h5 class="text-headline-large text-error mb-3 mt-10">
              Delete Account
            </h5>
            <p v-if="hasActiveSubscription" class="text-medium-emphasis">
              You cannot delete your account because you have an active
              subscription on httpSMS.
              <NuxtLink class="text-decoration-none" to="/billing"
                >Cancel your subscription</NuxtLink
              >
              before deleting your account.
            </p>
            <p v-else class="text-medium-emphasis">
              You can delete all your data on httpSMS by clicking the button
              below. This action is <b>irreversible</b> and all your data will
              be permanently deleted from the httpSMS database instantly and it
              cannot be recovered.
            </p>
            <VBtn
              color="error"
              :loading="deletingAccount"
              class="mt-2"
              :disabled="hasActiveSubscription"
              @click="showDeleteAccountDialog = true"
            >
              <VIcon start :icon="mdiDelete" />
              Delete your Account
            </VBtn>
            <VDialog v-model="showDeleteAccountDialog" max-width="600px">
              <VCard>
                <VCardTitle class="text-center"
                  >Delete your httpSMS account</VCardTitle
                >
                <VCardText class="mt-2 text-center text-medium-emphasis">
                  Are you sure you want to delete your account? This action is
                  <b>irreversible</b> and all your data will be permanently
                  deleted from the httpSMS database instantly.
                </VCardText>
                <VCardActions>
                  <VBtn
                    color="error"
                    variant="text"
                    :loading="deletingAccount"
                    @click="deleteUserAccount"
                  >
                    <VIcon v-if="lgAndUp" start :icon="mdiDelete" />
                    Delete My Account
                  </VBtn>
                  <VSpacer />
                  <VBtn
                    color="primary"
                    variant="flat"
                    @click="showDeleteAccountDialog = false"
                  >
                    <span v-if="lgAndUp">Keep My account</span>
                    <span v-else>Close</span>
                  </VBtn>
                </VCardActions>
              </VCard>
            </VDialog>
          </VCol>
        </VRow>
      </VContainer>
    </div>

    <!-- Webhook Edit Dialog -->
    <VDialog v-model="showWebhookEdit" max-width="600px" opacity="0.9">
      <VCard>
        <VCardTitle>
          <span v-if="!activeWebhook.id">Add a new&nbsp;</span>
          <span v-else>Edit&nbsp;</span>
          webhook
        </VCardTitle>
        <VCardText>
          <VRow>
            <VCol>
              <VTextField
                v-if="activeWebhook.id"
                variant="outlined"
                density="compact"
                disabled
                label="ID"
                :model-value="activeWebhook.id"
              />
              <VTextField
                v-model="activeWebhook.url"
                variant="outlined"
                density="compact"
                label="Callback URL"
                persistent-placeholder
                persistent-hint
                :error="errorMessages.has('url')"
                :error-messages="errorMessages.get('url')"
                hint="A POST request will be sent to this URL every time an event is triggered in httpSMS."
                placeholder="https://example.com/webhook"
              />
              <VTextField
                v-model="activeWebhook.signing_key"
                variant="outlined"
                density="compact"
                class="mt-6"
                persistent-placeholder
                persistent-hint
                label="Signing Key (optional)"
                placeholder="******************"
                :error="errorMessages.has('signing_key')"
                :error-messages="errorMessages.get('signing_key')"
                hint="The signing key is used to verify the webhook is sent from httpSMS."
              />
              <VSelect
                v-model="activeWebhook.events"
                :items="webhookEventOptions"
                label="Events"
                multiple
                chips
                variant="outlined"
                persistent-placeholder
                class="mt-6"
                density="compact"
                :error="errorMessages.has('events')"
                :error-messages="errorMessages.get('events')"
                hint="Select multiple httpSMS events to watch for"
                persistent-hint
              />
              <VSelect
                v-model="activeWebhook.phone_numbers"
                :items="phoneNumbers"
                label="Phone Numbers"
                multiple
                chips
                variant="outlined"
                persistent-placeholder
                class="mt-6"
                density="compact"
                :error="errorMessages.has('phone_numbers')"
                :error-messages="errorMessages.get('phone_numbers')"
                hint="Select multiple phone numbers to watch for events"
                persistent-hint
              />
            </VCol>
          </VRow>
        </VCardText>
        <VCardActions class="pb-4 px-4">
          <LoadingButton
            :icon="mdiContentSave"
            :loading="updatingWebhook"
            @click="saveWebhook"
          >
            {{ activeWebhook.id ? 'Update Webhook' : 'Save Webhook' }}
          </LoadingButton>
          <VSpacer />
          <VBtn
            v-if="activeWebhook.id"
            :disabled="updatingWebhook"
            size="small"
            color="error"
            variant="text"
            @click="deleteWebhook(activeWebhook.id)"
          >
            <VIcon v-if="lgAndUp" start :icon="mdiDelete" />
            Delete
          </VBtn>
          <VBtn
            v-else
            variant="text"
            color="warning"
            @click="showWebhookEdit = false"
            >Close</VBtn
          >
        </VCardActions>
      </VCard>
    </VDialog>

    <!-- Discord Edit Dialog -->
    <VDialog v-model="showDiscordEdit" max-width="700px">
      <VCard>
        <VCardTitle>
          <span v-if="!activeDiscord.id">Add a new&nbsp;</span>
          <span v-else>Edit&nbsp;</span>
          discord integration
        </VCardTitle>
        <VCardText>
          <VRow>
            <VCol class="pt-8">
              <p class="mt-n4 text-body-1">
                Click the button below to add the httpSMS bot to your discord
                server. You need to do this so we can have permission to send
                and receive messages on your discord server.
              </p>
              <VBtn
                color="#5865f2"
                class="mb-6"
                target="_blank"
                href="https://discord.com/api/oauth2/authorize?client_id=1095780203256627291&permissions=2147485760&scope=bot%20applications.commands"
              >
                <VIcon start :icon="mdiConnection" />
                Add Discord Bot
              </VBtn>
              <VTextField
                v-if="activeDiscord.id"
                variant="outlined"
                density="compact"
                disabled
                label="ID"
                :model-value="activeDiscord.id"
              />
              <VTextField
                v-model="activeDiscord.name"
                variant="outlined"
                density="compact"
                label="Name"
                persistent-placeholder
                persistent-hint
                :error="errorMessages.has('name')"
                :error-messages="errorMessages.get('name')"
                hint="The name of the discord integration"
                placeholder="e.g Game Server"
              />
              <VTextField
                v-model="activeDiscord.server_id"
                variant="outlined"
                density="compact"
                class="mt-6"
                persistent-placeholder
                persistent-hint
                label="Discord Server ID"
                placeholder="e.g 1095778291488653372"
                :error="errorMessages.has('server_id')"
                :error-messages="errorMessages.get('server_id')"
                hint="You can get this by right clicking on your server and clicking Copy Server ID."
              />
              <VTextField
                v-model="activeDiscord.incoming_channel_id"
                variant="outlined"
                density="compact"
                class="mt-6"
                persistent-placeholder
                persistent-hint
                label="Discord Incoming Channel ID"
                placeholder="e.g 1095778291488653372"
                :error="errorMessages.has('incoming_channel_id')"
                :error-messages="errorMessages.get('incoming_channel_id')"
                hint="You can get this by right clicking on your discord channel and clicking Copy Channel ID."
              />
            </VCol>
          </VRow>
        </VCardText>
        <VCardActions class="pb-4 pl-6">
          <LoadingButton
            :icon="mdiContentSave"
            :loading="updatingDiscord"
            @click="saveDiscord"
          >
            {{
              activeDiscord.id
                ? 'Update Discord Integration'
                : 'Save Discord Integration'
            }}
          </LoadingButton>
          <VSpacer />
          <VBtn
            v-if="activeDiscord.id"
            :disabled="updatingDiscord"
            color="error"
            variant="text"
            @click="deleteDiscord(activeDiscord.id)"
          >
            <VIcon v-if="lgAndUp" start :icon="mdiDelete" />
            Delete
          </VBtn>
          <VBtn
            v-else
            variant="text"
            color="warning"
            @click="showDiscordEdit = false"
            >Close</VBtn
          >
        </VCardActions>
      </VCard>
    </VDialog>

    <!-- Phone Edit Dialog -->
    <VDialog v-model="showPhoneEdit" max-width="700px" opacity="0.9">
      <VCard>
        <VCardTitle>Edit Phone</VCardTitle>
        <VCardText v-if="activePhone">
          <VContainer>
            <VRow>
              <VCol>
                <VTextField
                  variant="outlined"
                  density="compact"
                  disabled
                  label="ID"
                  :model-value="activePhone.id"
                />
                <VTextField
                  variant="outlined"
                  disabled
                  density="compact"
                  label="Phone Number"
                  :model-value="activePhone.phone_number"
                />
                <VTextField
                  variant="outlined"
                  disabled
                  density="compact"
                  label="SIM"
                  :model-value="activePhone.sim"
                />
                <VTextarea
                  variant="outlined"
                  disabled
                  density="compact"
                  label="FCM Token"
                  :model-value="activePhone.fcm_token"
                />
                <VTextField
                  v-model="activePhone.message_expiration_seconds"
                  variant="outlined"
                  type="number"
                  density="compact"
                  label="Message Expiration (seconds)"
                />
                <VTextField
                  v-model="activePhone.messages_per_minute"
                  variant="outlined"
                  type="number"
                  density="compact"
                  label="Messages Per Minute"
                />
                <VTextField
                  v-model="activePhone.max_send_attempts"
                  variant="outlined"
                  type="number"
                  density="compact"
                  placeholder="How many retries when sending an SMS"
                  label="Max Send Attempts"
                  min="1"
                  max="5"
                  :rules="[
                    (v: number) =>
                      (v >= 1 && v <= 5) ||
                      'Max send attempts must be between 1 and 5',
                  ]"
                />
                <VAutocomplete
                  v-model="activePhone.message_send_schedule_id"
                  variant="outlined"
                  :readonly="sendSchedules.length === 0"
                  density="compact"
                  clearable
                  label="Send Schedule"
                  :items="sendSchedules"
                  item-title="name"
                  item-value="id"
                  hint="Attach a send schedule to this phone"
                  persistent-hint
                />
                <VTextarea
                  v-model="activePhone.missed_call_auto_reply"
                  variant="outlined"
                  density="compact"
                  class="mt-6"
                  label="Missed Call AutoReply"
                  persistent-placeholder
                  persistent-hint
                  placeholder="We are currently closed at the moment, please send us a text message from 09:00 to 17:00"
                  hint="Here you can configure an automated SMS message which is sent to the caller when this phone has a missed call"
                />
              </VCol>
            </VRow>
          </VContainer>
        </VCardText>
        <VCardActions class="pb-4 px-4 mt-n4">
          <loading-button :loading="updatingPhone" @click="updatePhone">
            <VIcon v-if="lgAndUp" start :icon="mdiContentSave" />
            Update Phone
          </loading-button>
          <VSpacer />
          <VBtn
            color="error"
            variant="text"
            :disabled="updatingPhone"
            @click="deletePhone(activePhone?.id ?? '')"
          >
            <VIcon v-if="lgAndUp" start :icon="mdiDelete" />
            Delete
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>

    <!-- Send Schedule Edit Dialog -->
    <VDialog v-model="showScheduleEdit" max-width="800px" opacity="0.9">
      <VCard>
        <VCardTitle>
          <span v-if="!activeSchedule.id">Create Message Send Schedule</span>
          <span v-else>Edit Message Send Schedule</span>
        </VCardTitle>
        <VCardText class="mt-4" :class="{ 'px-2': mdAndDown }">
          <VRow>
            <VCol cols="12" md="6">
              <VTextField
                v-model="activeSchedule.name"
                variant="outlined"
                density="compact"
                persistent-placeholder
                label="Schedule Name"
                placeholder="e.g Business Hours"
                :error="errorMessages.has('name')"
                :error-messages="errorMessages.get('name')"
              />
            </VCol>
            <VCol cols="12" md="6">
              <VAutocomplete
                v-model="activeSchedule.timezone"
                density="compact"
                variant="outlined"
                :items="timezones"
                label="Timezone"
                :error="errorMessages.has('timezone')"
                :error-messages="errorMessages.get('timezone')"
              />
            </VCol>
          </VRow>
          <VCard variant="flat" :border="lgAndUp" class="px-0">
            <VCardText :class="mdAndDown ? 'px-2 mt-n4' : 'px-4'">
              <div
                v-for="day in weekDays"
                :key="day.value"
                :class="[smAndUp ? 'd-flex align-start' : '', 'mb-4']"
              >
                <div
                  :class="[smAndUp ? 'pr-4' : '', 'pt-2']"
                  :style="smAndUp ? 'min-width: 160px' : ''"
                >
                  <VSwitch
                    :model-value="scheduleDayEnabled(day.value)"
                    inset
                    density="compact"
                    color="primary"
                    :label="day.label"
                    hide-details
                    class="mt-0 pt-0"
                    @update:model-value="scheduleToggleDay(day.value, $event)"
                  />
                </div>
                <div class="pt-2 flex-grow-1">
                  <div
                    v-for="(window, index) in scheduleWindowsForDay(day.value)"
                    :key="`${day.value}-${index}`"
                    class="d-flex align-center flex-wrap mb-2"
                  >
                    <div
                      class="mr-2 mb-2"
                      style="width: 130px; max-width: 100%"
                    >
                      <VTextField
                        v-model="window.start_time"
                        density="compact"
                        variant="outlined"
                        :error="!!scheduleWindowError(day.value)"
                        type="time"
                        label="Start"
                        hide-details="auto"
                      />
                    </div>
                    <div class="mb-2 mr-2">–</div>
                    <div
                      class="mr-2 mb-2"
                      style="width: 130px; max-width: 100%"
                    >
                      <VTextField
                        v-model="window.end_time"
                        density="compact"
                        variant="outlined"
                        :error="!!scheduleWindowError(day.value)"
                        type="time"
                        label="End"
                        hide-details="auto"
                      />
                    </div>
                    <div class="mb-2">
                      <VBtn
                        v-if="index == 0"
                        icon
                        variant="text"
                        density="comfortable"
                        color="primary"
                        @click="scheduleAddWindow(day.value)"
                      >
                        <VIcon :icon="mdiPlus" />
                      </VBtn>
                      <VBtn
                        icon
                        density="comfortable"
                        variant="text"
                        class="ml-1"
                        color="error"
                        @click="scheduleRemoveWindow(day.value, index)"
                      >
                        <VIcon :icon="mdiDelete" />
                      </VBtn>
                    </div>
                  </div>
                  <div
                    v-if="scheduleWindowError(day.value)"
                    class="w-100 text-error mt-n2 mb-4"
                  >
                    {{ scheduleWindowError(day.value) }}
                  </div>
                </div>
              </div>
            </VCardText>
          </VCard>
        </VCardText>
        <VCardActions class="pb-4 mt-n2">
          <LoadingButton
            :icon="mdiContentSave"
            :loading="savingSchedule"
            @click="saveSchedule"
          >
            {{ activeSchedule.id ? 'Update Schedule' : 'Save Schedule' }}
          </LoadingButton>
          <VSpacer />
          <VBtn
            v-if="activeSchedule.id"
            :disabled="savingSchedule"
            color="error"
            variant="text"
            @click="confirmDeleteSchedule"
          >
            <VIcon v-if="lgAndUp" start :icon="mdiDelete" />
            Delete
          </VBtn>
          <VBtn
            v-else
            variant="text"
            color="warning"
            @click="showScheduleEdit = false"
          >
            Close
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>

    <!-- Send Schedule Delete Confirmation -->
    <VDialog v-model="showScheduleDelete" max-width="500" opacity="0.9">
      <VCard>
        <VCardTitle>Delete schedule</VCardTitle>
        <VCardText class="text-medium-emphasis">
          Are you sure you want to delete <b>{{ activeSchedule.name }}</b
          >? Phones attached to this schedule will no longer have schedule-based
          restrictions.
        </VCardText>
        <VCardActions>
          <VBtn
            variant="flat"
            color="error"
            :loading="savingSchedule"
            @click="deleteSchedule"
          >
            Delete
          </VBtn>
          <VSpacer />
          <VBtn variant="text" @click="showScheduleDelete = false">Cancel</VBtn>
        </VCardActions>
      </VCard>
    </VDialog>
  </VContainer>
</template>
