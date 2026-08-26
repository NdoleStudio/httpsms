<script setup lang="ts">
import {
  mdiAccountGroupOutline,
  mdiAccountPlus,
  mdiAlertCircleOutline,
  mdiArrowLeft,
  mdiClose,
  mdiDelete,
  mdiFileUpload,
  mdiMagnify,
  mdiPhoneCheck,
  mdiEmailCheckOutline,
  mdiSquareEditOutline,
} from '@mdi/js'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { EntitiesContact } from '~~/shared/types/api'
import { useContactsStore } from '~/stores/contacts'
import { useFilters } from '~/composables/useFilters'
import { toApiError } from '~/utils/api-error'

definePageMeta({
  middleware: ['auth'],
})

useHead({
  title: 'Contacts - httpSMS',
})

const contactsStore = useContactsStore()
const { formatPhoneNumber, formatTimestamp, humanizeTimeShort } = useFilters()

const headers = [
  { title: 'Name', key: 'name', sortable: true },
  { title: 'Phone Numbers', key: 'phone_numbers', sortable: false },
  { title: 'Emails', key: 'emails', sortable: false },
  { title: 'Created', key: 'created_at', sortable: false },
  { title: 'Updated', key: 'updated_at', sortable: true },
  { title: 'Actions', key: 'actions', sortable: false, align: 'end' as const },
]

const itemsPerPageOptions = [
  { value: 10, title: '10' },
  { value: 25, title: '25' },
  { value: 50, title: '50' },
  { value: 100, title: '100' },
]

const contactDialog = ref(false)
const deleteDialog = ref(false)
const importDialog = ref(false)
const saving = ref(false)

// Server-driven pagination state for VDataTableServer.
const page = ref(1)
const itemsPerPage = ref(10)
const sortBy = ref<{ key: string; order: 'asc' | 'desc' }[]>([
  { key: 'updated_at', order: 'desc' },
])
// initialLoadComplete gates the table's initial @update:options emit so the
// first fetch is driven by onMounted rather than firing twice on mount.
const initialLoadComplete = ref(false)

const editingContact = ref<EntitiesContact | null>(null)
const pendingDelete = ref<EntitiesContact | null>(null)

const importFile = ref<File | null>(null)
const importErrors = ref<string[]>([])

let searchTimer: ReturnType<typeof setTimeout> | undefined

const searchTerm = computed({
  get: () => contactsStore.search,
  set: (value: string | null) => {
    contactsStore.search = value ?? ''
  },
})

function openAdd() {
  editingContact.value = null
  contactDialog.value = true
}

function openEdit(contact: EntitiesContact) {
  editingContact.value = contact
  contactDialog.value = true
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

function fieldErrorsFromApi(error: unknown): string[] {
  const data = toApiError(error).data?.data
  if (!data || typeof data !== 'object') {
    return []
  }
  return Object.values(data).flat()
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
  const sort = sortBy.value[0]
  return contactsStore
    .loadContacts({
      force: true,
      skip,
      limit: itemsPerPage.value,
      sortBy: sort?.key === 'name' ? 'name' : 'updated_at',
      sortDescending: sort ? sort.order === 'desc' : true,
    })
    .catch(() => {
      // The store already surfaced the failure via a notification.
    })
}

function onUpdateOptions(options: {
  page: number
  itemsPerPage: number
  sortBy: { key: string; order: 'asc' | 'desc' }[]
}) {
  page.value = options.page
  itemsPerPage.value = options.itemsPerPage
  sortBy.value = options.sortBy.length
    ? options.sortBy
    : [{ key: 'updated_at', order: 'desc' }]

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
        <VCol cols="12">
          <div class="d-flex align-center">
            <h1 class="text-display-large mb-1">Contacts</h1>
            <VSpacer />
            <div class="mt-12">
              <VBtn
                variant="tonal"
                class="ml-4 mb-4"
                :prepend-icon="mdiFileUpload"
                @click="openImport"
              >
                Import CSV
              </VBtn>
              <VBtn
                class="ml-4 mb-4"
                color="primary"
                variant="flat"
                :prepend-icon="mdiAccountPlus"
                @click="openAdd"
              >
                Add Contact
              </VBtn>
            </div>
          </div>
          <p class="text-medium-emphasis mb-6">
            Use httpSMS as a lightweight CRM by adding your contacts here. Your
            message threads will show contact names instead of phone numbers,
            making conversations easier to recognize and manage. Add contacts
            individually, or fill in our
            <a
              class="text-decoration-none hover:text-decoration-underline"
              href="/templates/httpsms-contacts.csv"
              download
              >CSV template</a
            >
            and upload it to import your contact list in bulk.
          </p>

          <VTextField
            v-model="searchTerm"
            :prepend-inner-icon="mdiMagnify"
            label="Search by name, phone number or email"
            variant="outlined"
            density="compact"
            autocomplete="search-query"
            clearable
            color="primary"
            hide-details
            class="mb-4"
          />

          <VDataTableServer
            v-model:page="page"
            v-model:items-per-page="itemsPerPage"
            v-model:sort-by="sortBy"
            :headers="headers"
            :header-props="{
              class: 'text-uppercase text-medium-emphasis',
            }"
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
                <span class="font-weight-medium">{{ item.name }}</span>
              </div>
            </template>

            <template #[`item.phone_numbers`]="{ item }">
              <div
                v-if="item.phone_numbers?.length"
                class="d-flex flex-column ga-1 py-2"
              >
                <div
                  v-for="phone in item.phone_numbers ?? []"
                  :key="phone"
                  class="d-flex"
                >
                  <VIcon
                    :icon="mdiPhoneCheck"
                    size="small"
                    color="info"
                    class="mr-1 text-medium-emphasis"
                  />
                  {{ formatPhoneNumber(phone) }}
                </div>
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
                    :icon="mdiEmailCheckOutline"
                    size="small"
                    color="info"
                    class="mr-1 text-medium-emphasis"
                  />
                  {{ email }}
                </span>
              </div>
              <span v-else class="text-medium-emphasis">—</span>
            </template>

            <template #[`item.created_at`]="{ item }">
              <VTooltip :text="formatTimestamp(item.created_at)">
                <template #activator="{ props }">
                  <span v-bind="props" class="text-medium-emphasis">
                    {{ humanizeTimeShort(item.created_at) }}
                  </span>
                </template>
              </VTooltip>
            </template>

            <template #[`item.updated_at`]="{ item }">
              <VTooltip :text="formatTimestamp(item.updated_at)">
                <template #activator="{ props }">
                  <span v-bind="props" class="text-medium-emphasis">
                    {{ humanizeTimeShort(item.updated_at) }}
                  </span>
                </template>
              </VTooltip>
            </template>

            <template #[`item.actions`]="{ item }">
              <div class="d-flex justify-end">
                <VBtn
                  :icon="mdiSquareEditOutline"
                  variant="text"
                  class="mr-2"
                  density="comfortable"
                  aria-label="Edit contact"
                  @click="openEdit(item)"
                />
                <VBtn
                  :icon="mdiDelete"
                  variant="text"
                  density="comfortable"
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
        </VCol>
      </VRow>
    </VContainer>

    <ContactDialog v-model="contactDialog" :contact="editingContact" />

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
        <VCardText class="mt-n2 text-medium-emphasis">
          Are you sure you want to delete
          <v-code>{{ pendingDelete?.name }}</v-code
          >? This action cannot be undone.
        </VCardText>
        <VCardActions class="mb-2">
          <VBtn
            color="error"
            variant="flat"
            :prepend-icon="mdiDelete"
            :loading="saving"
            :disabled="saving"
            @click="confirmDelete"
          >
            Delete Contact
          </VBtn>
          <VSpacer />
          <VBtn color="warning" variant="text" @click="deleteDialog = false">
            Close
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
          <p class="mb-4 mt-n2 text-medium-emphasis">
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
        <VCardActions class="pb-4">
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
          <v-spacer />
          <VBtn color="warning" variant="text" @click="importDialog = false">
            Close
          </VBtn>
        </VCardActions>
      </VCard>
    </VDialog>
  </VContainer>
</template>
