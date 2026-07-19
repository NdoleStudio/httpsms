<script setup lang="ts">
import {
  mdiAccount,
  mdiAccountGroupOutline,
  mdiAccountPlus,
  mdiAlertCircleOutline,
  mdiArrowLeft,
  mdiClose,
  mdiDelete,
  mdiEmailOutline,
  mdiFileUpload,
  mdiMagnify,
  mdiPencil,
  mdiPhone,
  mdiPlus,
} from '@mdi/js'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { EntitiesContact } from '~~/shared/types/api'
import { useContactsStore, type ContactInput } from '~/stores/contacts'
import { useFilters } from '~/composables/useFilters'
import { ErrorMessages } from '~/utils/errors'
import { toApiError } from '~/utils/api-error'

definePageMeta({
  middleware: ['auth'],
})

useHead({
  title: 'Contacts - httpSMS',
})

interface PropertyRow {
  key: string
  value: string
}

interface ContactForm {
  name: string
  phoneNumbers: string[]
  emails: string[]
  properties: PropertyRow[]
}

const contactsStore = useContactsStore()
const { formatPhoneNumber, humanizeTime, formatTimestamp } = useFilters()

// Avatar backgrounds are picked deterministically from the Vuetify theme
// palette so the table reads like an address book without inventing colors.
const avatarPalette = ['primary', 'secondary', 'info', 'success']

const headers = [
  { title: 'Name', key: 'name', sortable: true },
  { title: 'Phone Numbers', key: 'phone_numbers', sortable: false },
  { title: 'Emails', key: 'emails', sortable: false },
  { title: 'Created', key: 'created_at', sortable: true },
  { title: 'Updated', key: 'updated_at', sortable: true },
  { title: 'Actions', key: 'actions', sortable: false, align: 'end' as const },
]

const itemsPerPageOptions = [
  { value: 10, title: '10' },
  { value: 25, title: '25' },
  { value: 50, title: '50' },
  { value: 100, title: '100' },
]

const editDialog = ref(false)
const deleteDialog = ref(false)
const importDialog = ref(false)
const saving = ref(false)

// Server-driven pagination state for VDataTableServer.
const page = ref(1)
const itemsPerPage = ref(10)
// initialLoadComplete gates the table's initial @update:options emit so the
// first fetch is driven by onMounted rather than firing twice on mount.
const initialLoadComplete = ref(false)

const editingId = ref<string | null>(null)
const pendingDelete = ref<EntitiesContact | null>(null)

const form = ref<ContactForm>({
  name: '',
  phoneNumbers: [''],
  emails: [''],
  properties: [],
})
const formErrors = ref(new ErrorMessages())

const importFile = ref<File | null>(null)
const importErrors = ref<string[]>([])

let searchTimer: ReturnType<typeof setTimeout> | undefined

const searchTerm = computed({
  get: () => contactsStore.search,
  set: (value: string | null) => {
    contactsStore.search = value ?? ''
  },
})

const dialogTitle = computed(() =>
  editingId.value ? 'Edit Contact' : 'Add Contact',
)

function emptyForm(): ContactForm {
  return {
    name: '',
    phoneNumbers: [''],
    emails: [''],
    properties: [],
  }
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) {
    return ''
  }
  const first = parts[0] ?? ''
  const last = parts.length > 1 ? (parts[parts.length - 1] ?? '') : ''
  return (first.charAt(0) + last.charAt(0)).toUpperCase()
}

function avatarColor(name: string): string {
  const key = name.trim()
  if (!key) {
    return 'primary'
  }
  let hash = 0
  for (let index = 0; index < key.length; index += 1) {
    hash = (hash + key.charCodeAt(index)) % avatarPalette.length
  }
  return avatarPalette[hash] ?? 'primary'
}

function relativeTime(value: string): string {
  const humanized = humanizeTime(value)
  return humanized ? `${humanized} ago` : 'just now'
}

function openAdd() {
  editingId.value = null
  form.value = emptyForm()
  formErrors.value = new ErrorMessages()
  editDialog.value = true
}

function openEdit(contact: EntitiesContact) {
  editingId.value = contact.id
  form.value = {
    name: contact.name ?? '',
    phoneNumbers: contact.phone_numbers?.length
      ? [...contact.phone_numbers]
      : [''],
    emails: contact.emails?.length ? [...contact.emails] : [''],
    properties: Object.entries(contact.properties ?? {}).map(
      ([key, value]) => ({ key, value }),
    ),
  }
  formErrors.value = new ErrorMessages()
  editDialog.value = true
}

function openDelete(contact: EntitiesContact) {
  pendingDelete.value = contact
  deleteDialog.value = true
}

function openImport() {
  importFile.value = null
  importErrors.value = []
  importDialog.value = true
}

function addPhoneNumber() {
  form.value.phoneNumbers.push('')
}

function removePhoneNumber(index: number) {
  form.value.phoneNumbers.splice(index, 1)
  if (form.value.phoneNumbers.length === 0) {
    form.value.phoneNumbers.push('')
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
      .map((value) => value.trim())
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
    (value) => value.trim().length > 0,
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
    if (editingId.value) {
      await contactsStore.updateContact(editingId.value, payload)
    } else {
      await contactsStore.saveContacts([payload])
    }
    editDialog.value = false
  } catch (error: unknown) {
    // The store already surfaced a toast; retain the API field messages so the
    // user can correct them inline instead of only seeing a transient toast.
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

async function confirmDelete() {
  if (!pendingDelete.value?.id) {
    return
  }
  saving.value = true
  try {
    await contactsStore.deleteContact(pendingDelete.value.id)
    deleteDialog.value = false
    pendingDelete.value = null
  } catch {
    // The store already surfaced the failure via a notification.
  } finally {
    saving.value = false
  }
}

async function submitImport() {
  if (!importFile.value) {
    return
  }
  importErrors.value = []
  saving.value = true
  try {
    await contactsStore.uploadCsv(importFile.value)
    importDialog.value = false
    importFile.value = null
  } catch (error: unknown) {
    // The store already surfaced a toast; keep the row-indexed messages inline
    // so the user can see exactly which CSV rows failed validation.
    importErrors.value = fieldErrorsFromApi(error)
  } finally {
    saving.value = false
  }
}

function fetchContacts() {
  const skip = (page.value - 1) * itemsPerPage.value
  return contactsStore
    .loadContacts({ force: true, skip, limit: itemsPerPage.value })
    .catch(() => {
      // The store already surfaced the failure via a notification.
    })
}

function onUpdateOptions(options: { page: number; itemsPerPage: number }) {
  page.value = options.page
  itemsPerPage.value = options.itemsPerPage

  // Ignore the initial emit fired while the table mounts; onMounted owns the
  // first fetch so the request is not duplicated.
  if (!initialLoadComplete.value) {
    return
  }
  fetchContacts()
}

watch(
  () => contactsStore.search,
  () => {
    if (searchTimer) {
      clearTimeout(searchTimer)
    }
    searchTimer = setTimeout(() => {
      // A new query always resets to the first page. If we are already on
      // page 1 the page ref does not change, so no @update:options fires and
      // we fetch directly; otherwise resetting the page drives the fetch via
      // onUpdateOptions. Either way exactly one request is made.
      if (page.value !== 1) {
        page.value = 1
      } else {
        fetchContacts()
      }
    }, 350)
  },
)

onMounted(() => {
  initialLoadComplete.value = true
  fetchContacts()
})

onBeforeUnmount(() => {
  if (searchTimer) {
    clearTimeout(searchTimer)
  }
})
</script>

<template>
  <VContainer fluid class="px-0 pt-0">
    <VAppBar>
      <VBtn icon to="/threads" aria-label="Back to messages">
        <VIcon :icon="mdiArrowLeft" />
      </VBtn>
      <VToolbarTitle>Contacts</VToolbarTitle>
      <VProgressLinear
        :active="contactsStore.loading"
        :indeterminate="contactsStore.loading"
        color="primary"
        location="bottom"
        absolute
      />
    </VAppBar>

    <VContainer class="pt-0">
      <VRow>
        <VCol cols="12" md="10" offset-md="1" xxl="8" offset-xxl="2">
          <div class="d-flex flex-column flex-md-row align-md-center mb-6 mt-3">
            <div>
              <h1 class="text-display-large mb-1">Contacts</h1>
              <p class="text-medium-emphasis mb-0">
                Manage your contacts. {{ contactsStore.total }} total
              </p>
            </div>
            <VSpacer />
            <div class="d-flex flex-column flex-sm-row ga-3 mt-4 mt-md-0">
              <VBtn
                variant="outlined"
                color="primary"
                :prepend-icon="mdiFileUpload"
                @click="openImport"
              >
                Import CSV
              </VBtn>
              <VBtn
                color="primary"
                variant="flat"
                :prepend-icon="mdiAccountPlus"
                @click="openAdd"
              >
                Add Contact
              </VBtn>
            </div>
          </div>

          <VTextField
            v-model="searchTerm"
            :prepend-inner-icon="mdiMagnify"
            label="Search by name, phone number or email"
            variant="outlined"
            density="comfortable"
            clearable
            hide-details
            class="mb-4"
          />

          <VCard variant="outlined">
            <VDataTableServer
              v-model:page="page"
              v-model:items-per-page="itemsPerPage"
              :headers="headers"
              :items="contactsStore.contacts"
              :items-length="contactsStore.total"
              :loading="contactsStore.loading"
              :items-per-page-options="itemsPerPageOptions"
              item-value="id"
              hover
              loading-text="Loading contacts…"
              @update:options="onUpdateOptions"
            >
              <template #[`item.name`]="{ item }">
                <div class="d-flex align-center py-2">
                  <VAvatar
                    :color="avatarColor(item.name)"
                    size="40"
                    class="mr-3"
                  >
                    <span
                      v-if="initials(item.name)"
                      class="text-body-1 font-weight-medium"
                      >{{ initials(item.name) }}</span
                    >
                    <VIcon v-else :icon="mdiAccount" />
                  </VAvatar>
                  <span class="font-weight-medium">{{ item.name }}</span>
                </div>
              </template>

              <template #[`item.phone_numbers`]="{ item }">
                <div
                  v-if="item.phone_numbers?.length"
                  class="d-flex flex-column ga-1 py-2"
                >
                  <VChip
                    v-for="phone in item.phone_numbers ?? []"
                    :key="phone"
                    size="small"
                    variant="tonal"
                    color="primary"
                    :prepend-icon="mdiPhone"
                  >
                    {{ formatPhoneNumber(phone) }}
                  </VChip>
                </div>
                <span v-else class="text-medium-emphasis">—</span>
              </template>

              <template #[`item.emails`]="{ item }">
                <div
                  v-if="item.emails?.length"
                  class="d-flex flex-column ga-1 py-2"
                >
                  <span
                    v-for="email in item.emails ?? []"
                    :key="email"
                    class="d-flex align-center text-body-2"
                  >
                    <VIcon
                      :icon="mdiEmailOutline"
                      size="x-small"
                      class="mr-1 text-medium-emphasis"
                    />
                    {{ email }}
                  </span>
                </div>
                <span v-else class="text-medium-emphasis">—</span>
              </template>

              <template #[`item.created_at`]="{ item }">
                <span :title="formatTimestamp(item.created_at)">{{
                  relativeTime(item.created_at)
                }}</span>
              </template>

              <template #[`item.updated_at`]="{ item }">
                <span :title="formatTimestamp(item.updated_at)">{{
                  relativeTime(item.updated_at)
                }}</span>
              </template>

              <template #[`item.actions`]="{ item }">
                <div class="d-flex justify-end">
                  <VBtn
                    :icon="mdiPencil"
                    variant="text"
                    size="small"
                    aria-label="Edit contact"
                    @click="openEdit(item)"
                  />
                  <VBtn
                    :icon="mdiDelete"
                    variant="text"
                    size="small"
                    color="error"
                    aria-label="Delete contact"
                    @click="openDelete(item)"
                  />
                </div>
              </template>

              <template #no-data>
                <div class="text-center py-12">
                  <VIcon
                    :icon="mdiAccountGroupOutline"
                    size="64"
                    class="text-medium-emphasis mb-3"
                  />
                  <p class="text-title-medium mb-1">
                    {{
                      searchTerm
                        ? 'No contacts match your search'
                        : 'No contacts yet'
                    }}
                  </p>
                  <p class="text-medium-emphasis mb-4">
                    {{
                      searchTerm
                        ? 'Try a different name, phone number or email.'
                        : 'Add your first contact or import them from a CSV file.'
                    }}
                  </p>
                  <VBtn
                    v-if="!searchTerm"
                    color="primary"
                    variant="flat"
                    :prepend-icon="mdiAccountPlus"
                    @click="openAdd"
                  >
                    Add Contact
                  </VBtn>
                </div>
              </template>
            </VDataTableServer>
          </VCard>
        </VCol>
      </VRow>
    </VContainer>

    <!-- Add / Edit contact dialog -->
    <VDialog v-model="editDialog" max-width="640" opacity="0.9">
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
            @click="editDialog = false"
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
            <VTextField
              v-model="form.phoneNumbers[index]"
              :label="`Phone number ${index + 1}`"
              placeholder="+18005550199"
              variant="outlined"
              density="comfortable"
              :prepend-inner-icon="mdiPhone"
              :error="index === 0 && formErrors.has('phone_numbers')"
              :error-messages="
                index === 0 ? formErrors.get('phone_numbers') : []
              "
            />
            <VBtn
              :icon="mdiClose"
              variant="text"
              size="small"
              class="mt-1"
              aria-label="Remove phone number"
              @click="removePhoneNumber(index)"
            />
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
            :key="`email-${index}`"
            class="d-flex align-start ga-2"
          >
            <VTextField
              v-model="form.emails[index]"
              :label="`Email ${index + 1}`"
              placeholder="alice@example.com"
              variant="outlined"
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
          <p
            v-if="!form.properties.length"
            class="text-medium-emphasis text-body-2 mb-2"
          >
            Add custom key/value details such as company or address.
          </p>
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
        <VCardActions>
          <VSpacer />
          <VBtn color="warning" variant="text" @click="editDialog = false">
            Close
          </VBtn>
          <VBtn
            color="primary"
            variant="flat"
            :loading="saving"
            :disabled="saving"
            @click="submitForm"
          >
            Save
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>

    <!-- Delete contact dialog -->
    <VDialog v-model="deleteDialog" max-width="480" opacity="0.9">
      <VCard>
        <VCardTitle class="d-flex align-center">
          <span>Delete Contact</span>
          <VSpacer />
          <VBtn
            :icon="mdiClose"
            variant="text"
            color="warning"
            size="small"
            aria-label="Close dialog"
            @click="deleteDialog = false"
          />
        </VCardTitle>
        <VCardText>
          Are you sure you want to delete
          <strong>{{ pendingDelete?.name }}</strong
          >? This action cannot be undone.
        </VCardText>
        <VCardActions>
          <VSpacer />
          <VBtn color="warning" variant="text" @click="deleteDialog = false">
            Close
          </VBtn>
          <VBtn
            color="error"
            variant="flat"
            :prepend-icon="mdiDelete"
            :loading="saving"
            :disabled="saving"
            @click="confirmDelete"
          >
            Delete
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>

    <!-- Import CSV dialog -->
    <VDialog v-model="importDialog" max-width="600" opacity="0.9">
      <VCard>
        <VCardTitle class="d-flex align-center">
          <span>Import Contacts from CSV</span>
          <VSpacer />
          <VBtn
            :icon="mdiClose"
            variant="text"
            color="warning"
            size="small"
            aria-label="Close dialog"
            @click="importDialog = false"
          />
        </VCardTitle>
        <VCardText>
          <p class="mb-4">
            Download the
            <a
              class="text-decoration-none hover:text-decoration-underline"
              href="/templates/httpsms-contacts.csv"
              download
              >CSV template</a
            >, fill it in and upload it here. Separate multiple emails or phone
            numbers within a cell using a semicolon (<code>;</code>).
          </p>
          <VFileInput
            v-model="importFile"
            label="CSV file"
            color="primary"
            accept=".csv,text/csv"
            variant="outlined"
            density="comfortable"
            :prepend-icon="mdiFileUpload"
            hide-details="auto"
          />
          <VAlert
            v-if="importErrors.length"
            type="error"
            variant="tonal"
            density="comfortable"
            class="mt-4"
            :icon="mdiAlertCircleOutline"
          >
            <p class="font-weight-medium mb-1">We couldn't import your file:</p>
            <ul class="pl-4 mb-0">
              <li v-for="message in importErrors" :key="message">
                {{ message }}
              </li>
            </ul>
          </VAlert>
        </VCardText>
        <VCardActions>
          <VSpacer />
          <VBtn color="warning" variant="text" @click="importDialog = false">
            Close
          </VBtn>
          <VBtn
            color="primary"
            variant="flat"
            :prepend-icon="mdiFileUpload"
            :loading="saving"
            :disabled="saving || !importFile"
            @click="submitImport"
          >
            Import
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>
  </VContainer>
</template>
