<script setup lang="ts">
import {
  mdiSend,
  mdiDotsVertical,
  mdiArrowLeft,
  mdiCheckAll,
  mdiDelete,
  mdiCallMissed,
  mdiPaperclip,
  mdiCheck,
  mdiAlert,
  mdiPackageUp,
  mdiPackageDown,
  mdiAccount,
  mdiRefresh,
  mdiContentCopy,
} from '@mdi/js'
import Pusher from 'pusher-js'
import type { Channel } from 'pusher-js'
import { isValidPhoneNumber } from 'libphonenumber-js'
import type { EntitiesMessage } from '~~/shared/types/api'
import { startsWithLetter } from '~/utils/filters'

definePageMeta({
  middleware: ['auth'],
})

useHead({
  title: 'Messages - httpSMS',
})

const route = useRoute()
const router = useRouter()
const config = useRuntimeConfig()
const { lgAndUp, mdAndDown, mdAndUp } = useDisplay()
const { formatPhoneNumber } = useFilters()
const notificationsStore = useNotificationsStore()
const authStore = useAuthStore()
const phonesStore = usePhonesStore()
const threadsStore = useThreadsStore()
const messagesStore = useMessagesStore()

const formMessage = ref('')
const submitting = ref(false)
const loadingMessages = ref(false)
const hideMessages = ref(true)
const messages = ref<EntitiesMessage[]>([])
const selectedMenuItem = ref(-1)
const messageBody = ref<HTMLElement | null>(null)
const form = ref<{ validate: () => Promise<{ valid: boolean }> } | null>(null)

const formMessageRules = [
  (v: string) =>
    v === '' ||
    (v && v.length <= 320) ||
    'Message must be less than 320 characters',
]

let webhookChannel: Channel | null = null

const contactIsPhoneNumber = computed(() => {
  const thread = threadsStore.currentThread
  if (!thread) return false
  return isValidPhoneNumber(thread.contact) || !isNaN(Number(thread.contact))
})

const messageVisibility = computed(() =>
  hideMessages.value ? 'hidden' : 'visible',
)
const contact = computed(() => threadsStore.currentThread?.contact ?? '')

function isMT(message: EntitiesMessage): boolean {
  return message.type === 'mobile-terminated'
}

function isMo(message: EntitiesMessage): boolean {
  return message.type === 'mobile-originated'
}

function isMissedCall(message: EntitiesMessage): boolean {
  return message.type === 'call/missed'
}

function isPending(message: EntitiesMessage): boolean {
  return ['sending', 'pending', 'scheduled'].includes(message.status)
}

function statusColor(message: EntitiesMessage): string {
  if (message.status === 'sending') return 'warning'
  if (message.status === 'scheduled') return 'teal'
  return 'primary'
}

function canResend(message: EntitiesMessage): boolean {
  return (
    isMT(message) &&
    (message.status === 'expired' || message.status === 'failed')
  )
}

function formatAttachmentName(url: string): string {
  const parts = url.split('/')
  if (parts.length >= 2) return '/' + parts.slice(-2).join('/')
  return url
}

function scrollToElement() {
  const el = messageBody.value
  if (el) {
    el.scrollTop = el.scrollHeight + 120
  }
  hideMessages.value = false
}

function loadMessages(hide = true) {
  loadingMessages.value = true
  const threadId = route.params.id as string
  threadsStore
    .loadThreadMessages(threadId)
    .then((response: EntitiesMessage[]) => {
      messages.value = [...response].reverse()
    })
    .finally(() => {
      setTimeout(() => {
        loadingMessages.value = false
      }, 1100)
    })
  hideMessages.value = hide
  setTimeout(() => {
    scrollToElement()
  }, 950)
}

async function loadData() {
  await authStore.loadUser()
  await phonesStore.loadPhones()
  await threadsStore.loadThreads()

  if (!threadsStore.hasThreadId(route.params.id as string)) {
    await router.push('/threads')
    return
  }
  loadMessages()
}

async function archiveThread() {
  await threadsStore.updateThread({
    threadId: threadsStore.currentThread!.id,
    isArchived: true,
  })
  setTimeout(() => {
    selectedMenuItem.value = -1
  }, 1000)
}

async function unArchiveThread() {
  await threadsStore.updateThread({
    threadId: threadsStore.currentThread!.id,
    isArchived: false,
  })
  setTimeout(() => {
    selectedMenuItem.value = -1
  }, 1000)
}

async function resendMessage(message: EntitiesMessage) {
  await messagesStore.sendMessage({
    from: message.owner,
    to: message.contact,
    content: message.content,
    sim: 'DEFAULT',
  })
  setTimeout(() => {
    selectedMenuItem.value = -1
  }, 1000)
  loadMessages(false)
}

async function deleteMessage(message: EntitiesMessage) {
  await messagesStore.deleteMessage(message.id)
  setTimeout(() => {
    selectedMenuItem.value = -1
  }, 1000)
  loadMessages(false)
}

async function copyMessageId(message: EntitiesMessage) {
  await navigator.clipboard.writeText(message.id)
  notificationsStore.addNotification({
    message: 'Message ID copied to clipboard',
    type: 'success',
  })
  setTimeout(() => {
    selectedMenuItem.value = -1
  }, 1000)
}

async function deleteThread(threadID: string) {
  await threadsStore.deleteThread(threadID)
  await router.push('/threads')
}

async function sendMessage(event: KeyboardEvent | Event) {
  if (event instanceof KeyboardEvent && event.shiftKey) return
  if (!formMessage.value.trim()) return

  if (form.value) {
    const { valid } = await form.value.validate()
    if (!valid) return
  }

  submitting.value = true
  await messagesStore.sendMessage({
    from: phonesStore.owner!,
    to: threadsStore.currentThread!.contact,
    content: formMessage.value,
    sim: 'DEFAULT',
  })
  loadMessages(false)
  formMessage.value = ''
  submitting.value = false
}

onMounted(async () => {
  await loadData()

  const pusher = new Pusher(config.public.pusherKey as string, {
    cluster: config.public.pusherCluster as string,
  })

  webhookChannel = pusher.subscribe(authStore.user!.id)
  webhookChannel.bind('message.phone.sent', () => {
    if (!loadingMessages.value) loadMessages(false)
  })
  webhookChannel.bind('message.send.failed', () => {
    if (!loadingMessages.value) loadMessages(false)
  })
  webhookChannel.bind('message.phone.received', () => {
    if (!loadingMessages.value) loadMessages(false)
  })
})

onBeforeUnmount(() => {
  if (webhookChannel) webhookChannel.unsubscribe()
})
</script>

<template>
  <VContainer class="px-0" :class="{ 'fill-height': lgAndUp }">
    <div class="h-100">
      <VAppBar>
        <VBtn v-if="mdAndDown" icon to="/threads">
          <VIcon :icon="mdiArrowLeft" />
        </VBtn>
        <VToolbarTitle v-if="threadsStore.currentThread">
          {{ formatPhoneNumber(threadsStore.currentThread.contact) }}
        </VToolbarTitle>
        <VMenu>
          <template #activator="{ props }">
            <VBtn icon variant="text" class="mt-2" v-bind="props">
              <VIcon :icon="mdiDotsVertical" />
            </VBtn>
          </template>
          <VList
            class="pa-0"
            prepend-gap="24"
            :density="mdAndDown ? 'compact' : 'default'"
          >
            <VListItem
              v-if="
                threadsStore.currentThread &&
                !threadsStore.currentThread.is_archived
              "
              @click.prevent="archiveThread"
            >
              <template #prepend>
                <VIcon :icon="mdiPackageDown" />
              </template>
              <VListItemTitle>Archive</VListItemTitle>
            </VListItem>
            <VListItem
              v-if="
                threadsStore.currentThread &&
                threadsStore.currentThread.is_archived
              "
              @click.prevent="unArchiveThread"
            >
              <template #prepend>
                <VIcon :icon="mdiPackageUp" />
              </template>
              <VListItemTitle>Unarchive</VListItemTitle>
            </VListItem>
            <VListItem
              v-if="threadsStore.currentThread"
              @click.prevent="deleteThread(threadsStore.currentThread.id)"
            >
              <template #prepend>
                <VIcon :icon="mdiDelete" color="error" />
              </template>
              <VListItemTitle>Delete Thread</VListItemTitle>
            </VListItem>
          </VList>
        </VMenu>
      </VAppBar>
      <VProgressLinear
        v-if="loadingMessages"
        class="mt-n5 py-1"
        color="primary"
        indeterminate
      />
      <VContainer v-if="threadsStore.currentThread" class="pa-0">
        <div
          ref="messageBody"
          class="messages-body no-scrollbar w-100 pl-2"
          :class="{ 'pr-7': lgAndUp }"
        >
          <VRow
            v-for="message in messages"
            :key="message.id"
            :style="{ visibility: messageVisibility }"
          >
            <VCol
              class="d-flex"
              :class="{
                'pr-10': mdAndDown && !isMT(message),
                'pl-12 pr-4': mdAndDown && isMT(message),
                'pl-16 ml-16': lgAndUp && isMT(message),
                'pr-16 mr-16': lgAndUp && !isMT(message),
              }"
            >
              <VSpacer v-if="isMT(message)" />
              <v-avatar
                v-if="isMo(message)"
                :class="{
                  'ml-2': !mdAndUp,
                  'ml-4': mdAndUp,
                }"
                :color="threadsStore.currentThread!.color"
                size="40"
              >
                <v-icon
                  v-if="!startsWithLetter(message.contact)"
                  color="white"
                  >{{ mdiAccount }}</v-icon
                >
                <span v-else class="text-white text-headline-small">{{
                  message.contact.substring(0, 1)
                }}</span>
              </v-avatar>
              <VAvatar v-if="isMissedCall(message)" color="#1e1e1e">
                <VIcon size="large" color="red" :icon="mdiCallMissed" />
              </VAvatar>
              <!-- MT message menu -->
              <VMenu v-if="isMT(message)">
                <template #activator="{ props }">
                  <VBtn icon variant="text" class="mt-2" v-bind="props">
                    <VIcon :icon="mdiDotsVertical" />
                  </VBtn>
                </template>
                <VList class="pa-0" prepend-gap="24" density="compact">
                  <VListItem
                    v-if="canResend(message)"
                    @click.prevent="resendMessage(message)"
                  >
                    <template #prepend
                      ><VIcon size="small" :icon="mdiRefresh"
                    /></template>
                    <VListItemTitle class="text-body-medium"
                      >Resend Message</VListItemTitle
                    >
                  </VListItem>
                  <VListItem @click.prevent="copyMessageId(message)">
                    <template #prepend
                      ><VIcon size="small" :icon="mdiContentCopy"
                    /></template>
                    <VListItemTitle class="text-body-medium"
                      >Copy Message ID</VListItemTitle
                    >
                  </VListItem>
                  <VListItem @click.prevent="deleteMessage(message)">
                    <template #prepend
                      ><VIcon size="small" :icon="mdiDelete" color="error"
                    /></template>
                    <VListItemTitle class="text-body-medium"
                      >Delete Message</VListItemTitle
                    >
                  </VListItem>
                </VList>
              </VMenu>
              <div>
                <VCard
                  rounded="shaped"
                  :variant="isMT(message) ? 'flat' : 'tonal'"
                  :color="isMT(message) ? 'primary' : undefined"
                  :class="{ 'ml-2': !isMT(message) }"
                >
                  <VCardText
                    v-if="message.content"
                    class="text-break"
                    style="white-space: pre-line"
                  >
                    <span v-if="!isMissedCall(message)">{{
                      message.content
                    }}</span>
                    <span v-else class="text-medium-emphasis"
                      >Missed phone call</span
                    >
                  </VCardText>
                </VCard>
                <VCard
                  v-if="message.attachments?.length"
                  :class="{ 'ml-2': !isMT(message) }"
                >
                  <VCardText class="pb-2">
                    <a
                      v-for="(attachment, index) in message.attachments"
                      :key="index"
                      target="_blank"
                      rel="noopener noreferrer"
                      :href="attachment"
                      class="text-decoration-none hover:text-decoration-underline text-body-medium mb-2 d-flex w-100"
                    >
                      <VIcon
                        size="x-small"
                        class="text-medium-emphasis mt-1"
                        :icon="mdiPaperclip"
                      />
                      {{ formatAttachmentName(attachment) }}
                    </a>
                  </VCardText>
                </VCard>
                <div class="d-flex mt-n2">
                  <p class="ml-2 text-medium-emphasis text-body-small mr-2">
                    {{ new Date(message.order_timestamp).toLocaleString() }}
                  </p>
                  <VSpacer />
                  <VTooltip location="bottom">
                    <template #activator="{ props }">
                      <div v-bind="props">
                        <VIcon
                          v-if="message.status === 'expired'"
                          color="warning"
                          class="mt-2"
                          :icon="mdiAlert"
                        />
                        <VProgressCircular
                          v-else-if="isPending(message)"
                          indeterminate
                          :size="14"
                          :width="1"
                          class="mt-2"
                          :color="statusColor(message)"
                        />
                        <VIcon
                          v-else-if="message.status === 'delivered'"
                          color="primary"
                          class="mt-2"
                          :icon="mdiCheckAll"
                        />
                        <VIcon
                          v-else-if="message.status === 'sent'"
                          class="mt-2"
                          :icon="mdiCheck"
                        />
                        <VIcon
                          v-else-if="message.status === 'failed'"
                          color="error"
                          class="mt-2"
                          :icon="mdiAlert"
                        />
                      </div>
                    </template>
                    <span>{{ message.failure_reason || message.status }}</span>
                  </VTooltip>
                </div>
              </div>
              <!-- MO message menu -->
              <VMenu v-if="!isMT(message)">
                <template #activator="{ props }">
                  <VBtn icon variant="text" class="mt-2" v-bind="props">
                    <VIcon :icon="mdiDotsVertical" />
                  </VBtn>
                </template>
                <VList class="pa-0" prepend-gap="16" density="compact">
                  <VListItem
                    v-if="canResend(message)"
                    @click.prevent="resendMessage(message)"
                  >
                    <template #prepend
                      ><VIcon size="small" :icon="mdiRefresh"
                    /></template>
                    <VListItemTitle class="text-body-medium"
                      >Resend Message</VListItemTitle
                    >
                  </VListItem>
                  <VListItem @click.prevent="copyMessageId(message)">
                    <template #prepend
                      ><VIcon size="small" :icon="mdiContentCopy"
                    /></template>
                    <VListItemTitle class="text-body-medium"
                      >Copy Message ID</VListItemTitle
                    >
                  </VListItem>
                  <VListItem @click.prevent="deleteMessage(message)">
                    <template #prepend
                      ><VIcon size="small" :icon="mdiDelete" color="error"
                    /></template>
                    <VListItemTitle class="text-body-medium"
                      >Delete Message</VListItemTitle
                    >
                  </VListItem>
                </VList>
              </VMenu>
            </VCol>
          </VRow>
        </div>
        <VFooter app class="py-0" color="#121212">
          <VContainer class="pb-0 pt-0">
            <VForm ref="form" class="d-flex" @submit.prevent="sendMessage">
              <VTextField
                v-model="formMessage"
                :disabled="submitting || !contactIsPhoneNumber"
                variant="solo"
                :rules="formMessageRules"
                :placeholder="
                  contactIsPhoneNumber
                    ? 'Type your message here'
                    : 'You cannot send messages to ' + contact
                "
                rounded
                @keydown.enter="sendMessage"
              />
              <VBtn
                :disabled="submitting || !contactIsPhoneNumber"
                type="submit"
                color="primary"
                class="ml-2"
                icon
                size="large"
              >
                <VProgressCircular
                  v-if="submitting"
                  indeterminate
                  :size="20"
                  :width="3"
                  color="pink"
                />
                <VIcon :icon="mdiSend" />
              </VBtn>
            </VForm>
          </VContainer>
        </VFooter>
      </VContainer>
    </div>
  </VContainer>
</template>

<style lang="scss">
.messages-body {
  padding-top: 50px;
  max-height: calc(100vh - 170px);
  position: absolute;
  width: calc(100vw - 8px);
  bottom: 94px;
}

@media (min-width: 960px) {
  .messages-body {
    max-width: 900px;
  }
}
@media (min-width: 1280px) {
  .messages-body {
    max-width: 1185px;
  }
}

@media (min-width: 1545px) {
  .messages-body {
    max-width: 1280px;
  }
}

@media (min-width: 2138px) {
  .messages-body {
    max-width: 2000px;
  }
}

.no-scrollbar {
  overflow-x: hidden;
  -ms-overflow-style: none;
  overflow-y: scroll;
  &::-webkit-scrollbar {
    display: none;
  }
}
</style>
