<template>
  <v-container fluid class="px-0 pt-0" :fill-height="$vuetify.breakpoint.lgAndUp">
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
            <p class="text--secondary">
              Create reusable availability schedules and attach them to phones
              from the Settings page. Outgoing messages respect both the phone
              send rate and the selected schedule.
            </p>

            <div class="d-flex mb-4">
              <v-btn color="primary" @click="openCreateDialog">
                <v-icon left>{{ mdiPlus }}</v-icon>
                Add Schedule
              </v-btn>
            </div>

            <div v-if="loading" class="mb-4">
              <v-progress-circular :size="60" :width="2" color="primary" indeterminate></v-progress-circular>
            </div>

            <v-row v-else-if="schedules.length">
              <v-col v-for="schedule in schedules" :key="schedule.id" cols="12" md="6">
                <v-card outlined>
                  <v-card-title class="d-flex align-center">
                    <span class="text-h6">{{ schedule.name }}</span>
                    <v-spacer></v-spacer>
                    <v-chip small :color="schedule.is_active ? 'success' : 'grey'" text-color="white">
                      {{ schedule.is_active ? 'Active' : 'Inactive' }}
                    </v-chip>
                  </v-card-title>
                  <v-card-subtitle>{{ schedule.timezone }}</v-card-subtitle>
                  <v-card-text>
                    <div v-for="line in scheduleSummary(schedule)" :key="line">{{ line }}</div>
                  </v-card-text>
                  <v-card-actions>
                    <v-btn small color="primary" @click="openEditDialog(schedule)">
                      <v-icon left>{{ mdiSquareEditOutline }}</v-icon>
                      Edit
                    </v-btn>
                    <v-spacer></v-spacer>
                    <v-btn small text color="error" @click="confirmDelete(schedule)">
                      <v-icon left>{{ mdiDelete }}</v-icon>
                      Delete
                    </v-btn>
                  </v-card-actions>
                </v-card>
              </v-col>
            </v-row>

            <v-alert v-else outlined type="info">
              No schedules yet. Create your first availability schedule.
            </v-alert>
          </v-col>
        </v-row>
      </v-container>
    </div>

    <v-dialog v-model="showDialog" :fullscreen="$vuetify.breakpoint.smAndDown" max-width="900">
      <v-card>
        <v-card-title>{{ activeSchedule.id ? 'Edit Schedule' : 'Add Schedule' }}</v-card-title>
        <v-card-text>
          <v-row>
            <v-col cols="12" md="6">
              <v-text-field v-model="activeSchedule.name" outlined dense label="Schedule name"></v-text-field>
            </v-col>
            <v-col cols="12" md="6">
              <v-autocomplete v-model="activeSchedule.timezone" dense outlined :items="timezones" label="Timezone"></v-autocomplete>
            </v-col>
            <v-col cols="12" md="3">
              <v-switch v-model="activeSchedule.is_active" inset label="Active"></v-switch>
            </v-col>
          </v-row>

          <div v-for="day in weekDays" :key="day.value" class="mb-4 pa-3 rounded schedule-day">
            <div class="d-flex align-center mb-2">
              <div class="font-weight-medium">{{ day.label }}</div>
              <v-spacer></v-spacer>
              <v-btn small text color="primary" @click="addWindow(day.value)">Add window</v-btn>
            </div>
            <div v-if="windowsForDay(day.value).length === 0" class="text--secondary">Unavailable</div>
            <v-row v-for="(window, index) in windowsForDay(day.value)" :key="`${day.value}-${index}`">
              <v-col cols="5" sm="4">
                <v-text-field v-model="window.start_time" dense outlined type="time" label="Start"></v-text-field>
              </v-col>
              <v-col cols="5" sm="4">
                <v-text-field v-model="window.end_time" dense outlined type="time" label="End"></v-text-field>
              </v-col>
              <v-col cols="2" sm="4" class="d-flex align-center">
                <v-btn icon color="error" @click="removeWindow(day.value, index)">
                  <v-icon>{{ mdiDelete }}</v-icon>
                </v-btn>
              </v-col>
            </v-row>
          </div>
        </v-card-text>
        <v-card-actions>
          <v-btn color="primary" :loading="saving" @click="saveSchedule">Save</v-btn>
          <v-spacer></v-spacer>
          <v-btn text @click="showDialog = false">Close</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="showDeleteDialog" max-width="500">
      <v-card>
        <v-card-title>Delete schedule</v-card-title>
        <v-card-text>
          Are you sure you want to delete <b>{{ activeSchedule.name }}</b>?
          Phones attached to this schedule will no longer have schedule-based
          restrictions.
        </v-card-text>
        <v-card-actions>
          <v-btn color="error" :loading="saving" @click="deleteSchedule">Delete</v-btn>
          <v-spacer></v-spacer>
          <v-btn text @click="showDeleteDialog = false">Cancel</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<script>
import Vue from 'vue'
import { mdiArrowLeft, mdiDelete, mdiPlus, mdiSquareEditOutline } from '@mdi/js'
import axios from '~/plugins/axios'

export default Vue.extend({
  name: 'SendSchedulesPage',
  middleware: ['auth'],
  data() {
    return {
      mdiArrowLeft,
      mdiDelete,
      mdiPlus,
      mdiSquareEditOutline,
      loading: false,
      saving: false,
      showDialog: false,
      showDeleteDialog: false,
      schedules: [],
      activeSchedule: {
        id: null,
        name: 'Business Hours',
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
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
    }
  },
  head() {
    return { title: 'Send Schedules - httpSMS' }
  },
  computed: {
    timezones() {
      return Intl.supportedValuesOf('timeZone')
    },
  },
  async mounted() {
    await this.loadSchedules()
  },
  methods: {
    minuteToClock(value) {
      const hours = String(Math.floor(value / 60)).padStart(2, '0')
      const minutes = String(value % 60).padStart(2, '0')
      return `${hours}:${minutes}`
    },
    clockToMinute(value) {
      const [hours, minutes] = value.split(':').map((x) => parseInt(x, 10))
      return hours * 60 + minutes
    },
    windowsForDay(dayOfWeek) {
      return this.activeSchedule.windows.filter((x) => x.day_of_week === dayOfWeek)
    },
    addWindow(dayOfWeek) {
      this.activeSchedule.windows.push({ day_of_week: dayOfWeek, start_time: '09:00', end_time: '17:00' })
    },
    removeWindow(dayOfWeek, index) {
      const matches = this.activeSchedule.windows.filter((x) => x.day_of_week === dayOfWeek)
      const target = matches[index]
      this.activeSchedule.windows = this.activeSchedule.windows.filter((x) => x !== target)
    },
    scheduleSummary(schedule) {
      return this.weekDays.map((day) => {
        const windows = (schedule.windows || []).filter((x) => x.day_of_week === day.value)
        if (windows.length === 0) return `${day.label}: Unavailable`
        return `${day.label}: ${windows.map((w) => `${this.minuteToClock(w.start_minute)}-${this.minuteToClock(w.end_minute)}`).join(', ')}`
      })
    },
    openCreateDialog() {
      this.activeSchedule = {
        id: null,
        name: 'Business Hours',
        timezone: this.$store.getters.getUser?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone,
        is_active: true,
        windows: [
          { day_of_week: 1, start_time: '09:00', end_time: '17:00' },
          { day_of_week: 2, start_time: '09:00', end_time: '17:00' },
          { day_of_week: 3, start_time: '09:00', end_time: '17:00' },
          { day_of_week: 4, start_time: '09:00', end_time: '17:00' },
          { day_of_week: 5, start_time: '09:00', end_time: '17:00' },
        ],
      }
      this.showDialog = true
    },
    openEditDialog(schedule) {
      this.activeSchedule = {
        id: schedule.id,
        name: schedule.name,
        timezone: schedule.timezone,
        is_active: schedule.is_active,
        windows: (schedule.windows || []).map((x) => ({ day_of_week: x.day_of_week, start_time: this.minuteToClock(x.start_minute), end_time: this.minuteToClock(x.end_minute) })),
      }
      this.showDialog = true
    },
    confirmDelete(schedule) {
      this.activeSchedule = { ...schedule }
      this.showDeleteDialog = true
    },
    async loadSchedules() {
      this.loading = true
      try {
        const response = await axios.get('/v1/send-schedules')
        this.schedules = response.data?.data || []
      } catch (error) {
        this.$store.dispatch('addNotification', {
          type: 'error',
          message: 'Failed to load send schedules',
        })
      } finally {
        this.loading = false
      }
    },
    async saveSchedule() {
      this.saving = true
      try {
        const payload = {
          name: this.activeSchedule.name,
          timezone: this.activeSchedule.timezone,
          is_active: this.activeSchedule.is_active,
          windows: (this.activeSchedule.windows || []).map((window) => ({
            day_of_week: window.day_of_week,
            start_minute: this.clockToMinute(window.start_time),
            end_minute: this.clockToMinute(window.end_time),
          })),
        }
        if (this.activeSchedule.id) {
          await axios.put(`/v1/send-schedules/${this.activeSchedule.id}`, payload)
        } else {
          await axios.post('/v1/send-schedules', payload)
        }
        this.$store.dispatch('addNotification', { type: 'success', message: 'Send schedule saved successfully' })
        this.showDialog = false
        await this.loadSchedules()
      } catch (error) {
        this.$store.dispatch('addNotification', {
          type: 'error',
          message: error?.response?.data?.message || 'Failed to save send schedule',
        })
      } finally {
        this.saving = false
      }
    },
    async deleteSchedule() {
      this.saving = true
      try {
        await axios.delete(`/v1/send-schedules/${this.activeSchedule.id}`)
        this.$store.dispatch('addNotification', { type: 'success', message: 'Send schedule deleted successfully' })
        this.showDeleteDialog = false
        await this.loadSchedules()
      } catch (error) {
        this.$store.dispatch('addNotification', { type: 'error', message: 'Failed to delete send schedule' })
      } finally {
        this.saving = false
      }
    },
  },
})
</script>

<style scoped>
.schedule-day {
  border: 1px solid rgba(0, 0, 0, 0.12);
}
</style>
