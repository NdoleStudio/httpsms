<template>
  <v-container fluid class="pa-0" :fill-height="$vuetify.breakpoint.lgAndUp">
    <div class="w-full h-full">
      <v-app-bar height="60" :dense="$vuetify.breakpoint.mdAndDown" fixed>
        <v-btn icon to="/threads">
          <v-icon>{{ mdiArrowLeft }}</v-icon>
        </v-btn>
        <v-toolbar-title
          >Heartbeats
          <v-icon x-small class="mx-2" color="primary">{{ mdiCircle }}</v-icon>
          <span v-if="$store.getters.getOwner">{{
            $store.getters.getOwner | phoneNumber
          }}</span></v-toolbar-title
        >
      </v-app-bar>
      <v-container class="mt-16">
        <v-row>
          <v-col cols="12">
            <p>
              Every 15 minutes, the httpSMS app on your Android phone sends a
              heartbeat event to the httpsms API to show that it is alive. The
              reason for this is because the Android operating system sometimes
              kills an application to save battery
              <a
                href="https://dontkillmyapp.com"
                class="text-decoration-none"
                target="_blank"
                >https://dontkillmyapp.com</a
              >.
            </p>
            <p>
              If httpSMS doesn't get any heartbeat event in a 1-hour interval,
              you will get an email notification about it so you can check if
              there is an issue with your Android phone.
            </p>
          </v-col>
          <v-col v-if="$vuetify.breakpoint.mdAndUp" cols="12" class="px-0">
            <bar-chart :data="chartData" :options="chartOptions"></bar-chart>
          </v-col>
          <v-col cols="12">
            <p>
              The table below shows the last 100 heartbeat events received from
              the httpSMS app on your Android phone.
            </p>
            <v-data-table
              :value="selected"
              hide-default-footer
              :headers="dataTableHeaders"
              :items="dataTableItems"
              sort-by="timestamp"
              sort-desc
              :items-per-page="100"
              class="heartbeat--table"
            >
              <template #item.interval="{ item }">
                {{ formatDuration(item.interval) }}
              </template>
              <template #item.owner="{ item }">
                {{ item.owner | phoneNumber }}
              </template>
              <template #item.timestamp="{ item }">
                {{ item.timestamp | timestamp }}
              </template>
            </v-data-table>
          </v-col>
        </v-row>
      </v-container>
    </div>
  </v-container>
</template>

<script>
import { mdiArrowLeft, mdiCircle } from '@mdi/js'
import 'chartjs-adapter-moment'
import { formatDuration, intervalToDuration } from 'date-fns'
import vueClassComponentEsm from 'vue-class-component'

export default {
  name: 'HeartbeatIndex',
  middleware: ['auth'],

  data() {
    return {
      mdiArrowLeft,
      mdiCircle,
      heartbeats: [],
      selected: [3, 6],
      dataTableHeaders: [
        {
          text: 'HEARTBEAT ID',
          align: 'start',
          sortable: false,
          value: 'id',
        },
        { text: 'PHONE NUMBER', value: 'owner', sortable: false },
        { text: 'RECEIVED AT', value: 'timestamp' },
        { text: 'TIME INTERVAL', value: 'interval' },
      ],
    }
  },

  head() {
    return {
      title: 'Heartbeats - Http SMS',
    }
  },

  computed: {
    vueClassComponentEsm() {
      return vueClassComponentEsm
    },
    dataTableItems() {
      return this.heartbeats.map((heartbeat, index) => {
        let interval = 0
        if (index < 99) {
          interval = this.getDiff(
            heartbeat.timestamp,
            this.heartbeats[index + 1].timestamp
          )
        }
        const item = {
          id: heartbeat.id,
          timestamp: heartbeat.timestamp,
          owner: heartbeat.owner,
          interval,
        }
        if (interval > 3600000) {
          this.selected.push(item)
        }
        return item
      })
    },
    chartOptions() {
      const minDate = new Date()
      minDate.setDate(minDate.getDate() - 1)
      return {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: {
            display: false,
          },
          tooltip: {
            callbacks: {
              label: function (context) {
                if (context.dataIndex === 99) {
                  return '-'
                }
                const duration = intervalToDuration({
                  start: new Date(
                    context.dataset.data[context.dataIndex + 1].x
                  ),
                  end: new Date(context.dataset.data[context.dataIndex].x),
                })
                return formatDuration(duration)
              },
            },
          },
        },
        scales: {
          x: {
            type: 'time',
          },
          y: {
            display: false,
          },
        },
      }
    },
    chartData() {
      const data = this.heartbeats.map((heartbeat) => {
        return {
          x: new Date(heartbeat.timestamp).toISOString(),
          y: 1,
        }
      })

      if (!data.length) {
        return {
          datasets: [
            {
              data,
              backgroundColor: '#2196f3',
            },
          ],
        }
      }

      let prev = new Date(data[0].x)
      const newData = []
      for (let i = 1; i < data.length; i++) {
        const current = new Date(data[i].x)
        const diff = prev - current
        if (diff > 600000) {
          // 10 minutes
          newData.push(data[i])
          prev = current
        }
      }

      return {
        datasets: [
          {
            data: newData,
            backgroundColor: '#2196f3',
          },
        ],
      }
    },
  },

  async mounted() {
    await this.$store.dispatch('loadUser')
    await this.$store.dispatch('loadPhones')
    this.getHeartbeat()
  },

  methods: {
    getDiff(a, b) {
      return new Date(a) - new Date(b)
    },

    formatDuration(duration) {
      if (duration === 0) {
        return '-'
      }
      const start = new Date()
      start.setMilliseconds(start.getMilliseconds() + duration)
      return (
        formatDuration(intervalToDuration({ start: new Date(), end: start })) ||
        '0 seconds'
      )
    },

    getHeartbeat() {
      this.$store.dispatch('getHeartbeat', 100).then((heartbeats) => {
        this.heartbeats = heartbeats
      })
    },
  },
}
</script>

<style lang="scss">
.v-application {
  .heartbeat--table.v-data-table tbody tr.v-data-table__selected {
    background: #b71c1c;
  }
}
</style>
