<script setup lang="ts">
import {
  mdiAccount,
  mdiAlertCircleOutline,
  mdiClose,
  mdiContentSaveCheck,
  mdiEmailOutline,
  mdiPlus,
} from '@mdi/js'
import { parsePhoneNumberFromString } from 'libphonenumber-js'
import type { EntitiesContact } from '~~/shared/types/api'
import { useContactsStore, type ContactInput } from '~/stores/contacts'
import { toApiError } from '~/utils/api-error'
import { ErrorMessages } from '~/utils/errors'

interface PropertyRow {
  key: string
  value: string
}

interface PhoneNumberRow {
  value: string
  country: string
}

interface ContactForm {
  name: string
  phoneNumbers: PhoneNumberRow[]
  emails: string[]
  properties: PropertyRow[]
}

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    contact?: EntitiesContact | null
    initialPhoneNumber?: string
    refreshContacts?: boolean
  }>(),
  {
    contact: null,
    initialPhoneNumber: '',
    refreshContacts: true,
  },
)

const emit = defineEmits<{
  saved: []
  'update:modelValue': [value: boolean]
}>()

const contactsStore = useContactsStore()
const saving = ref(false)
const form = ref<ContactForm>(emptyForm())
const formErrors = ref(new ErrorMessages())

const dialogTitle = computed(() =>
  props.contact ? 'Edit Contact' : 'Add Contact',
)

function phoneNumberRow(value = ''): PhoneNumberRow {
  return {
    value,
    country: parsePhoneNumberFromString(value)?.country ?? 'US',
  }
}

function emptyForm(): ContactForm {
  return {
    name: '',
    phoneNumbers: [phoneNumberRow(props?.initialPhoneNumber)],
    emails: [''],
    properties: [{ key: '', value: '' }],
  }
}

function resetForm() {
  const contact = props.contact
  if (!contact) {
    form.value = emptyForm()
  } else {
    const propertyEntries = Object.entries(contact.properties ?? {}).map(
      ([key, value]) => ({ key, value }),
    )
    form.value = {
      name: contact.name ?? '',
      phoneNumbers: contact.phone_numbers?.length
        ? contact.phone_numbers.map((value) => phoneNumberRow(value))
        : [phoneNumberRow()],
      emails: contact.emails?.length ? [...contact.emails] : [''],
      properties: propertyEntries.length
        ? propertyEntries
        : [{ key: '', value: '' }],
    }
  }
  formErrors.value = new ErrorMessages()
}

function closeDialog() {
  emit('update:modelValue', false)
}

function addPhoneNumber() {
  form.value.phoneNumbers.push(phoneNumberRow())
}

function removePhoneNumber(index: number) {
  form.value.phoneNumbers.splice(index, 1)
  if (form.value.phoneNumbers.length === 0) {
    form.value.phoneNumbers.push(phoneNumberRow())
  }
}

function addEmail() {
  form.value.emails.push('')
}

function removeEmail(index: number) {
  form.value.emails.splice(index, 1)
  if (form.value.emails.length === 0) {
    form.value.emails.push('')
  }
}

function addProperty() {
  form.value.properties.push({ key: '', value: '' })
}

function removeProperty(index: number) {
  form.value.properties.splice(index, 1)
  if (form.value.properties.length === 0) {
    form.value.properties.push({ key: '', value: '' })
  }
}

function buildPayload(): ContactInput {
  const properties: Record<string, string> = {}
  form.value.properties.forEach((row) => {
    const key = row.key.trim()
    if (key) {
      properties[key] = row.value
    }
  })
  return {
    name: form.value.name.trim(),
    phone_numbers: form.value.phoneNumbers
      .map(({ value }) => value.trim())
      .filter((value) => value.length > 0),
    emails: form.value.emails
      .map((value) => value.trim())
      .filter((value) => value.length > 0),
    properties,
  }
}

function validateForm(): boolean {
  const bag = new ErrorMessages()
  if (form.value.name.trim() === '') {
    bag.add('name', 'The name is required.')
  }
  const hasPhone = form.value.phoneNumbers.some(
    ({ value }) => value.trim().length > 0,
  )
  if (!hasPhone) {
    bag.add('phone_numbers', 'At least one phone number is required.')
  }
  formErrors.value = bag
  return bag.size() === 0
}

function fieldErrorsFromApi(error: unknown): string[] {
  const data = toApiError(error).data?.data
  if (!data || typeof data !== 'object') {
    return []
  }
  return Object.values(data).flat()
}

async function submitForm() {
  if (!validateForm()) {
    return
  }
  const payload = buildPayload()
  saving.value = true
  try {
    if (props.contact) {
      await contactsStore.updateContact(props.contact.id, payload, {
        refresh: props.refreshContacts,
      })
    } else {
      await contactsStore.saveContacts([payload], {
        refresh: props.refreshContacts,
      })
    }
    closeDialog()
    emit('saved')
  } catch (error: unknown) {
    const bag = new ErrorMessages()
    const messages = fieldErrorsFromApi(error)
    if (messages.length > 0) {
      bag.addMany('contacts', messages)
    }
    formErrors.value = bag
  } finally {
    saving.value = false
  }
}

watch(
  () => props.modelValue,
  (isOpen) => {
    if (isOpen) {
      resetForm()
    }
  },
  { immediate: true },
)
</script>

<template>
  <VDialog
    :model-value="modelValue"
    max-width="640"
    opacity="0.9"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <VCard>
      <VCardTitle class="d-flex align-center">
        <span>{{ dialogTitle }}</span>
        <VSpacer />
        <VBtn
          :icon="mdiClose"
          variant="text"
          color="warning"
          size="small"
          aria-label="Close dialog"
          @click="closeDialog"
        />
      </VCardTitle>
      <VCardText>
        <VAlert
          v-if="formErrors.get('contacts').length"
          type="error"
          variant="tonal"
          density="comfortable"
          class="mb-4"
          :icon="mdiAlertCircleOutline"
        >
          <ul class="pl-4 mb-0">
            <li v-for="message in formErrors.get('contacts')" :key="message">
              {{ message }}
            </li>
          </ul>
        </VAlert>

        <VTextField
          v-model="form.name"
          label="Name"
          variant="outlined"
          density="comfortable"
          persistent-placeholder
          placeholder="e.g John Doe"
          :prepend-inner-icon="mdiAccount"
          :error="formErrors.has('name')"
          :error-messages="formErrors.get('name')"
          class="mb-2"
        />

        <div class="d-flex align-center mt-2 mb-1">
          <span class="text-subtitle-2">Phone Numbers</span>
          <VSpacer />
          <VBtn
            variant="text"
            color="primary"
            size="small"
            :prepend-icon="mdiPlus"
            @click="addPhoneNumber"
          >
            Add
          </VBtn>
        </div>
        <div
          v-for="(phone, index) in form.phoneNumbers"
          :key="`phone-${index}`"
          class="d-flex align-start ga-2"
        >
          <div class="mt-2" style="flex: 0 0 92%">
            <v-phone-input
              v-model="phone.value"
              v-model:country="phone.country"
              :label="`Phone number ${index + 1}`"
              country-label="Country"
              placeholder="Phone number e.g 18005550199"
              variant="outlined"
              density="comfortable"
              color="primary"
              persistent-placeholder
              :error="index === 0 && formErrors.has('phone_numbers')"
              :error-messages="
                index === 0 ? formErrors.get('phone_numbers') : []
              "
            />
          </div>
          <div style="flex: 0 0 8%">
            <VBtn
              :icon="mdiClose"
              variant="text"
              size="small"
              class="mt-1"
              aria-label="Remove phone number"
              @click="removePhoneNumber(index)"
            />
          </div>
        </div>

        <div class="d-flex align-center mt-2 mb-1">
          <span class="text-subtitle-2">Email Addresses</span>
          <VSpacer />
          <VBtn
            variant="text"
            color="primary"
            size="small"
            :prepend-icon="mdiPlus"
            @click="addEmail"
          >
            Add
          </VBtn>
        </div>
        <div
          v-for="(email, index) in form.emails"
          :key="`email-${email}-${index}`"
          class="d-flex align-start ga-2"
        >
          <VTextField
            v-model="form.emails[index]"
            :label="`Email ${index + 1}`"
            placeholder="e.g alice@example.com"
            variant="outlined"
            autocomplete="email"
            type="email"
            persistent-placeholder
            density="comfortable"
            :prepend-inner-icon="mdiEmailOutline"
          />
          <VBtn
            :icon="mdiClose"
            variant="text"
            size="small"
            class="mt-1"
            aria-label="Remove email"
            @click="removeEmail(index)"
          />
        </div>

        <div class="d-flex align-center mt-2 mb-1">
          <span class="text-subtitle-2">Properties</span>
          <VSpacer />
          <VBtn
            variant="text"
            color="primary"
            size="small"
            :prepend-icon="mdiPlus"
            @click="addProperty"
          >
            Add
          </VBtn>
        </div>
        <div
          v-for="(property, index) in form.properties"
          :key="`property-${index}`"
          class="d-flex align-start ga-2"
        >
          <VTextField
            v-model="property.key"
            label="Key"
            variant="outlined"
            density="comfortable"
          />
          <VTextField
            v-model="property.value"
            label="Value"
            variant="outlined"
            density="comfortable"
          />
          <VBtn
            :icon="mdiClose"
            variant="text"
            size="small"
            class="mt-1"
            aria-label="Remove property"
            @click="removeProperty(index)"
          />
        </div>
      </VCardText>
      <VCardActions class="pb-4">
        <VBtn
          color="primary"
          variant="flat"
          :loading="saving"
          :disabled="saving"
          :prepend-icon="mdiContentSaveCheck"
          @click="submitForm"
        >
          Save Contact
        </VBtn>
        <VSpacer />
        <VBtn color="warning" variant="text" @click="closeDialog">Close</VBtn>
      </VCardActions>
    </VCard>
  </VDialog>
</template>
