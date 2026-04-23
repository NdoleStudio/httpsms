<template>
  <v-container
    fluid
    class="px-0 pt-0"
    :fill-height="$vuetify.breakpoint.lgAndUp"
  >
    <div class="w-full h-full">
      <v-app-bar height="60" fixed :dense="$vuetify.breakpoint.mdAndDown">
        <v-btn icon to="/settings">
          <v-icon>{{ mdiArrowLeft }}</v-icon>
        </v-btn>
        <v-toolbar-title>
          <div class="py-16">Send Schedules</div>
        </v-toolbar-title>
      </v-app-bar>

      <v-container class="mt-16">
        <v-row>
          <v-col cols="12" md="9" offset-md="1" xl="8" offset-xl="2">
            <p class="text--secondary mb-4">
              Send schedules allow you to set custom time intervals when
              outgoing messages will be sent. You can create one schedule and
              attach it to multiple phone numbers from the Settings page.
            </p>

            <div class="d-flex flex-wrap mb-6">
              <v-btn color="primary" @click="openCreateForm">
                <v-icon left>{{ mdiPlus }}</v-icon>
                Add Send Schedule
              </v-btn>
            </div>

            <div v-if="loading" class="mb-6">
              <v-progress-circular
                :size="60"
                :width="2"
                color="primary"
                indeterminate
              />
            </div>

            <template v-else>
              <v-row v-if="schedules.length" class="mb-6">
                <v-col
                  v-for="schedule in schedules"
                  :key="schedule.id"
                  cols="12"
                  md="6"
                >
                  <v-card outlined class="h-full">
                    <v-card-title class="d-flex align-center">
                      <span class="text-h6 text-break">{{ schedule.name }}</span>
                      <v-spacer />
                      <v-chip
                        small
                        :color="isScheduleActive(schedule) ? 'success' : 'grey'"
                        text-color="white"
                      >
                        {{ isScheduleActive(schedule) ? 'Active' : 'Inactive' }}
                      </v-chip>
                    </v-card-title>

                    <v-card-subtitle>{{ schedule.timezone }}</v-card-subtitle>

                    <v-card-text>
                      <div
                        v-for="line in scheduleSummary(schedule)"
                        :key="`${schedule.id}-${line}`"
                        class="mb-1"
                      >
                        {{ line }}
                      </div>
                    </v-card-text>

                    <v-card-actions>
                      <v-btn
                        small
                        color="primary"
                        @click="openEditForm(schedule)"
                      >
                        <v-icon left>{{ mdiSquareEditOutline }}</v-icon>
                        Edit
                      </v-btn>

                      <v-spacer />

                      <v-btn
                        small
                        text
                        color="error"
                        @click="confirmDelete(schedule)"
                      >
                        <v-icon left>{{ mdiDelete }}</v-icon>
                        Delete
                      </v-btn>
                    </v-card-actions>
                  </v-card>
                </v-col>
              </v-row>

              <v-alert v-else outlined type="info" class="mb-6">
                No schedules yet. Create your first send schedule.
              </v-alert>
            </template>

            <v-card outlined>
              <v-card-title class="d-flex align-center">
                <span>{{
                  activeSchedule.id ? 'Edit Send Schedule' : 'Add Send Schedule'
                }}</span>
                <v-spacer />
                <v-btn
                  v-if="hasEditorChanges"
                  text
                  color="secondary"
                  @click="resetEditor"
                >
                  Cancel
                </v-btn>
              </v-card-title>

              <v-card-text>
                <v-row>
                  <v-col cols="12" md="6">
                    <v-text-field
                      v-model="activeSchedule.name"
                      outlined
                      dense
                      label="Schedule name"
                      placeholder="Business Hours"
                      :error="errorMessages.has('name')"
                      :error-messages="errorMessages.get('name')"
                    />
                  </v-col>

                  <v-col cols="12" md="6">
                    <v-autocomplete
                      v-model="activeSchedule.timezone"
                      dense
                      outlined
                      :items="timezones"
                      label="Timezone"
                      :error="errorMessages.has('timezone')"
                      :error-messages="errorMessages.get('timezone')"
                    />
                  </v-col>

                  <v-col cols="12">
                    <v-switch
                      v-model="activeSchedule.is_active"
                      inset
                      label="Active"
                      class="mt-0"
                    />
                  </v-col>
                </v-row>

                <v-alert
                  v-if="errorMessages.has('windows')"
                  dense
                  outlined
                  type="error"
                  class="mb-4"
                >
                  <div
                    v-for="message in errorMessages.get('windows')"
                    :key="message"
                  >
                    {{ message }}
                  </div>
                </v-alert>

                <v-card
                  v-for="day in weekDays"
                  :key="day.value"
                  outlined
                  class="mb-4"
                >
                  <v-card-text class="pb-2">
                    <div class="d-flex align-center flex-wrap mb-3">
                      <div
                        class="font-weight-medium mr-4"
                        style="min-width: 110px"
                      >
                        {{ day.label }}
                      </div>

                      <v-switch
                        :input-value="dayEnabled(day.value)"
                        inset
                        dense
                        hide-details
                        class="mt-0 pt-0"
                        @change="toggleDay(day.value, $event)"
                      />

                      <v-spacer />

                      <v-btn
                        small
                        text
                        color="primary"
                        :disabled="!dayEnabled(day.value)"
                        @click="addWindow(day.value)"
                      >
                        <v-icon left small>{{ mdiPlus }}</v-icon>
                        Add window
                      </v-btn>
                    </div>

                    <div
                      v-if="windowsForDay(day.value).length === 0"
                      class="text--secondary"
                    >
                      Unavailable
                    </div>

                    <div
                      v-for="(window, index) in windowsForDay(day.value)"
                      :key="`${day.value}-${index}`"
                      class="d-flex align-center flex-wrap schedule-window-row"
                    >
                      <div class="schedule-time-field mr-2 mb-2">
                        <v-text-field
                          v-model="window.start_time"
                          dense
                          outlined
                          type="time"
                          label="Start"
                          hide-details="auto"
                        />
                      </div>

                      <div class="schedule-separator mb-2 mr-2">–</div>

                      <div class="schedule-time-field mr-2 mb-2">
                        <v-text-field
                          v-model="window.end_time"
                          dense
                          outlined
                          type="time"
                          label="End"
                          hide-details="auto"
                        />
                      </div>

                      <div class="mb-2">
                        <v-btn
                          icon
                          color="error"
                          @click="removeWindow(day.value, index)"
                        >
                          <v-icon>{{ mdiDelete }}</v-icon>
                        </v-btn>
                      </div>
                    </div>
                  </v-card-text>
                </v-card>
              </v-card-text>

              <v-card-actions>
                <v-btn
                  color="primary"
                  :loading="saving"
                  @click="saveSchedule"
                >
                  <v-icon left>{{ mdiContentSave }}</v-icon>
                  Save
                </v-btn>

                <v-spacer />

                <v-btn text @click="resetEditor">Reset</v-btn>
              </v-card-actions>
            </v-card>
          </v-col>
        </v-row>
      </v-container>
    </div>

    <v-dialog v-model="showDeleteDialog" max-width="500">
      <v-card>
        <v-card-title>Delete schedule</v-card-title>
        <v-card-text>
          Are you sure you want to delete <b>{{ scheduleToDelete?.name }}</b>?
          Phones attached to this schedule will no longer have schedule-based
          restrictions.
        </v-card-text>
        <v-card-actions>
          <v-btn color="error" :loading="saving" @click="deleteSchedule">
            Delete
          </v-btn>
          <v-spacer />
          <v-btn text @click="showDeleteDialog = false">Cancel</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<script>
import Vue from 'vue'
import {
  mdiArrowLeft,
  mdiContentSave,
  mdiDelete,
  mdiPlus,
  mdiSquareEditOutline,
} from '@mdi/js'
import { ErrorMessages } from '~/plugins/errors'

export default Vue.extend({
  name: 'SendSchedulesPage',
  middleware: ['auth'],
  data() {
    return {
      mdiArrowLeft,
      mdiContentSave,
      mdiDelete,
      mdiPlus,
      mdiSquareEditOutline,
      loading: false,
      saving: false,
      showDeleteDialog: false,
      schedules: [],
      scheduleToDelete: null,
      activeSchedule: {
        id: null,
        name: '',
        timezone: '',
        is_active: true,
        windows: [],
      },
      weekDays: [
        { value: 1, label: 'Monday' },
        { value: 2, label: 'Tuesday' },
        { value: 3, label: 'Wednesday' },
        { value: 4, label: 'Thursday' },
        { value: 5, label: 'Friday' },
        { value: 6, label: 'Saturday' },
        { value: 0, label: 'Sunday' },
      ],
      errorMessages: new ErrorMessages(),
    }
  },
  head() {
    return {
      title: 'Send Schedules - httpSMS',
    }
  },
  computed: {
    timezones() {
      try {
        return Intl.supportedValuesOf('timeZone')
      } catch {
        return []
      }
    },
    hasEditorChanges() {
      return (
        this.activeSchedule.id !== null ||
        this.activeSchedule.name !== '' ||
        this.activeSchedule.windows.length > 0 ||
        this.activeSchedule.is_active !== true ||
        this.activeSchedule.timezone !== this.defaultTimezone()
      )
    },
  },
  async mounted() {
    this.resetEditor()
    await this.loadSchedules()
  },
  methods: {
    resetErrors() {
      this.errorMessages = new ErrorMessages()
    },

    defaultTimezone() {
      return (
        this.$store.getters.getUser?.timezone ||
        Intl.DateTimeFormat().resolvedOptions().timeZone
      )
    },

    resetEditor() {
      this.resetErrors()
      this.activeSchedule = {
        id: null,
        name: '',
        timezone: this.defaultTimezone(),
        is_active: true,
        windows: [],
      }
    },

    minuteToClock(value) {
      const hours = String(Math.floor(value / 60)).padStart(2, '0')
      const minutes = String(value % 60).padStart(2, '0')
      return `${hours}:${minutes}`
    },

    clockToMinute(value) {
      if (!value || !value.includes(':')) {
        return 0
      }

      const [hours, minutes] = value.split(':').map((x) => parseInt(x, 10))
      return hours * 60 + minutes
    },

    isScheduleActive(schedule) {
      if (typeof schedule.is_active !== 'undefined') {
        return schedule.is_active
      }
      return Boolean(schedule.active)
    },

    windowsForDay(dayOfWeek) {
      return this.activeSchedule.windows.filter(
        (x) => x.day_of_week === dayOfWeek,
      )
    },

    dayEnabled(dayOfWeek) {
      return this.windowsForDay(dayOfWeek).length > 0
    },

    toggleDay(dayOfWeek, enabled) {
      if (enabled) {
        if (!this.dayEnabled(dayOfWeek)) {
          this.addWindow(dayOfWeek)
        }
        return
      }

      this.activeSchedule.windows = this.activeSchedule.windows.filter(
        (x) => x.day_of_week !== dayOfWeek,
      )
    },

    addWindow(dayOfWeek) {
      this.activeSchedule.windows.push({
        day_of_week: dayOfWeek,
        start_time: '09:00',
        end_time: '17:00',
      })
    },

    removeWindow(dayOfWeek, index) {
      const matches = this.activeSchedule.windows.filter(
        (x) => x.day_of_week === dayOfWeek,
      )
      const target = matches[index]
      this.activeSchedule.windows = this.activeSchedule.windows.filter(
        (x) => x !== target,
      )
    },

    scheduleSummary(schedule) {
      return this.weekDays.map((day) => {
        const windows = (schedule.windows || []).filter(
          (x) => x.day_of_week === day.value,
        )

        if (windows.length === 0) {
          return `${day.label}: Unavailable`
        }

        return `${day.label}: ${windows
          .map(
            (w) =>
              `${this.minuteToClock(w.start_minute)}-${this.minuteToClock(
                w.end_minute,
              )}`,
          )
          .join(', ')}`
      })
    },

    openCreateForm() {
      this.resetEditor()
      this.activeSchedule.windows = [
        { day_of_week: 1, start_time: '09:00', end_time: '17:00' },
        { day_of_week: 2, start_time: '09:00', end_time: '17:00' },
        { day_of_week: 3, start_time: '09:00', end_time: '17:00' },
        { day_of_week: 4, start_time: '09:00', end_time: '17:00' },
        { day_of_week: 5, start_time: '09:00', end_time: '17:00' },
      ]
      this.$vuetify.goTo(0)
    },

    openEditForm(schedule) {
      this.resetErrors()
      this.activeSchedule = {
        id: schedule.id,
        name: schedule.name,
        timezone: schedule.timezone,
        is_active: this.isScheduleActive(schedule),
        windows: (schedule.windows || []).map((x) => ({
          day_of_week: x.day_of_week,
          start_time: this.minuteToClock(x.start_minute),
          end_time: this.minuteToClock(x.end_minute),
        })),
      }

      this.$vuetify.goTo(0)
    },

    confirmDelete(schedule) {
      this.scheduleToDelete = schedule
      this.showDeleteDialog = true
    },

    async loadSchedules() {
      this.loading = true
      try {
        this.schedules = await this.$store.dispatch('getSendSchedules')
      } catch (errors) {
        this.$store.dispatch('addNotification', {
          type: 'error',
          message: 'Failed to load send schedules',
        })
        this.schedules = []
      } finally {
        this.loading = false
      }
    },

    buildPayload() {
      return {
        name: this.activeSchedule.name,
        timezone: this.activeSchedule.timezone,
        is_active: this.activeSchedule.is_active,
        windows: (this.activeSchedule.windows || []).map((window) => ({
          day_of_week: window.day_of_week,
          start_minute: this.clockToMinute(window.start_time),
          end_minute: this.clockToMinute(window.end_time),
        })),
      }
    },

    async saveSchedule() {
      this.resetErrors()
      this.saving = true

      try {
        const payload = this.buildPayload()

        if (this.activeSchedule.id) {
          await this.$store.dispatch('updateSendSchedule', {
            id: this.activeSchedule.id,
            ...payload,
          })
        } else {
          await this.$store.dispatch('createSendSchedule', payload)
        }

        this.$store.dispatch('addNotification', {
          type: 'success',
          message: 'Send schedule saved successfully',
        })

        await this.loadSchedules()
        this.resetEditor()
      } catch (errors) {
        this.errorMessages = errors
      } finally {
        this.saving = false
      }
    },

async deleteSchedule() {
  if (!this.scheduleToDelete?.id) {
    return
  }

  const deletedScheduleId = this.scheduleToDelete.id
  this.saving = true

  try {
    await this.$store.dispatch('deleteSendSchedule', deletedScheduleId)

    this.$store.dispatch('addNotification', {
      type: 'success',
      message: 'Send schedule deleted successfully',
    })

    this.showDeleteDialog = false
    this.scheduleToDelete = null

    await this.loadSchedules()

    if (this.activeSchedule.id === deletedScheduleId) {
      this.resetEditor()
    }
  } catch (error) {
    this.$store.dispatch('addNotification', {
      type: 'error',
      message: 'Failed to delete send schedule',
    })
  } finally {
    this.saving = false
  }
},
},
})
</script>

<style scoped>
.schedule-window-row {
  gap: 0;
}

.schedule-time-field {
  width: 170px;
  max-width: 100%;
}

.schedule-separator {
  font-size: 18px;
  line-height: 40px;
}

@media (max-width: 600px) {
  .schedule-time-field {
    width: 100%;
  }

  .schedule-separator {
    display: none;
  }
}
</style>
