<script setup lang="ts">
import { mdiArrowLeft, mdiSend, mdiCircle } from '@mdi/js'
import {
  isValidPhoneNumber,
  getCountryCallingCode,
  parsePhoneNumber,
  type CountryCode,
} from 'libphonenumber-js'
import { toApiError } from '~/utils/api-error'

definePageMeta({
  middleware: ['auth'],
})

useHead({
  title: 'New Message - httpSMS',
})

const router = useRouter()
const { mdAndDown } = useDisplay()
const notificationsStore = useNotificationsStore()
const phonesStore = usePhonesStore()
const { useApi } = useApiComposable()
const { formatPhoneNumber } = useFilters()

const sending = ref(false)
const formPhoneNumber = ref('')
const phoneCountry = ref('US')
const formContent = ref('')
const formAttachments = ref('')
const errors = ref(new Map<string, string[]>())

function getRecipientNumber(): string {
  const phone = formPhoneNumber.value
  if (isValidPhoneNumber(phone)) {
    return phone
  }
  // Short code — strip the country dial code prefix
  const dialCode = getCountryCallingCode(
    phoneCountry.value.toUpperCase() as CountryCode,
  )
  const prefix = `+${dialCode}`
  if (phone.startsWith(prefix)) {
    return phone.slice(prefix.length)
  }
  return phone
}

async function sendMessage() {
  errors.value = new Map()
  sending.value = true

  try {
    const api = useApi()
    await api('/v1/messages/send', {
      method: 'POST',
      body: {
        to: getRecipientNumber(),
        from: phonesStore.owner,
        content: formContent.value,
        sim: 'DEFAULT',
        attachments: formAttachments.value
          .trim()
          .split(',')
          .filter((x) => x.trim() !== '')
          .map((x) => x.trim()),
      },
    })
    notificationsStore.addNotification({
      message: 'Message Sent!',
      type: 'success',
    })
    await router.push('/threads')
  } catch (err: unknown) {
    const data = toApiError(err).data?.data
    if (data) {
      const newErrors = new Map<string, string[]>()
      if (data.content) newErrors.set('content', data.content)
      if (data.to)
        newErrors.set(
          'to',
          data.to.map((x: string) =>
            x.replace('to field', 'phone number field'),
          ),
        )
      if (data.attachments) newErrors.set('attachments', data.attachments)
      if (data.from) {
        notificationsStore.addNotification({
          message: data.from[0],
          type: 'error',
        })
      }
      errors.value = newErrors
    }
  } finally {
    sending.value = false
  }
}

onMounted(async () => {
  await phonesStore.loadPhones()
  if (phonesStore.owner) {
    const country = parsePhoneNumber(phonesStore.owner)?.country
    if (country) {
      phoneCountry.value = country
    }
  }
})
</script>

<template>
  <VContainer fluid class="pa-0" :class="{ 'fill-height': true }">
    <div class="w-100 h-100">
      <VAppBar height="60" :density="mdAndDown ? 'compact' : 'default'">
        <VBtn icon to="/threads">
          <VIcon :icon="mdiArrowLeft" />
        </VBtn>
        <VToolbarTitle>
          New Message
          <template v-if="phonesStore.owner">
            <VIcon size="12" class="mx-2" color="primary" :icon="mdiCircle" />
            {{ formatPhoneNumber(phonesStore.owner) }}
          </template>
        </VToolbarTitle>
      </VAppBar>
      <VContainer>
        <VRow>
          <VCol cols="12" md="8" offset-md="2" xl="6" offset-xl="3">
            <p class="mb-8 mt-0">
              Enter the recipient's phone number and your message below, and
              we'll deliver a real SMS it through your connected Android phone.
              You can also text a short code like
              <v-code>24273</v-code> without entering a full phone number.
            </p>
            <form @submit.prevent="sendMessage">
              <v-phone-input
                v-model="formPhoneNumber"
                v-model:country="phoneCountry"
                :disabled="sending"
                :error="errors.has('to')"
                :error-messages="errors.get('to')"
                variant="outlined"
                color="primary"
                density="compact"
                persistent-placeholder
                placeholder="Recipient phone number e.g 18005550199"
                label="Phone Number"
                country-label="Country"
              />
              <VTextarea
                v-model="formContent"
                :error="errors.has('content')"
                :error-messages="errors.get('content')"
                :disabled="sending"
                variant="outlined"
                density="compact"
                color="primary"
                persistent-placeholder
                placeholder="Enter your message here"
                label="Content"
              />
              <loading-button
                :disabled="sending"
                :block="mdAndDown"
                :loading="sending"
                :icon="mdiSend"
              >
                Send Message
              </loading-button>
            </form>
          </VCol>
        </VRow>
      </VContainer>
    </div>
  </VContainer>
</template>
