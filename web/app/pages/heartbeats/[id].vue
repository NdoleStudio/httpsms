<script setup lang="ts">
import { mdiArrowLeft, mdiCircle } from '@mdi/js'
import {
  formatDuration as formatDurationFns,
  intervalToDuration,
} from 'date-fns'
import { Bar } from 'vue-chartjs'
import type { ChartData, ChartOptions } from 'chart.js'
import type { EntitiesHeartbeat } from '~~/shared/types/api'

definePageMeta({
  middleware: ['auth'],
})

useHead({
  title: 'Heartbeats - httpSMS',
})

const route = useRoute()
const { mdAndDown, mdAndUp, lgAndUp } = useDisplay()
const authStore = useAuthStore()
const phonesStore = usePhonesStore()
const { formatPhoneNumber, formatTimestamp } = useFilters()

const loading = ref(true)
const heartbeats = ref<EntitiesHeartbeat[]>([])
const phoneId = computed(() => route.params.id as string)

interface HeartbeatTableItem {
  id: string
  owner: string
  timestamp: string
  interval: number
}

const dataTableHeaders = [
  { title: 'HEARTBEAT ID', key: 'id', sortable: false },
  { title: 'PHONE NUMBER', key: 'owner', sortable: false },
  { title: 'RECEIVED AT', key: 'timestamp' },
  { title: 'TIME INTERVAL', key: 'interval' },
] as const

function getDiff(a: string, b: string): number {
  return new Date(a).getTime() - new Date(b).getTime()
}

function formatInterval(duration: number): string {
  if (duration === 0) {
    return '-'
  }
  const start = new Date()
  start.setMilliseconds(start.getMilliseconds() + duration)
  return (
    formatDurationFns(intervalToDuration({ start: new Date(), end: start })) ||
    '0 seconds'
  )
}

const dataTableItems = computed<HeartbeatTableItem[]>(() => {
  return heartbeats.value.map((heartbeat, index) => {
    let interval = 0
    if (index < heartbeats.value.length - 1) {
      interval = getDiff(
        heartbeat.timestamp,
        heartbeats.value[index + 1]!.timestamp,
      )
    }
    return {
      id: heartbeat.id,
      timestamp: heartbeat.timestamp,
      owner: heartbeat.owner,
      interval,
    }
  })
})

const chartData = computed<ChartData<'bar'>>(() => {
  const data = heartbeats.value.map((heartbeat) => ({
    x: new Date(heartbeat.timestamp).toISOString(),
    y: 1,
  }))

  if (!data.length) {
    return {
      datasets: [{ data: [], backgroundColor: '#2196f3' }],
    } as unknown as ChartData<'bar'>
  }

  let prev = new Date(data[0]!.x)
  const newData = [] as Array<{ x: string; y: number }>
  for (let i = 1; i < data.length; i++) {
    const current = new Date(data[i]!.x)
    const diff = prev.getTime() - current.getTime()
    if (diff > 600000) {
      // 10 minutes
      newData.push(data[i]!)
      prev = current
    }
  }

  return {
    datasets: [{ data: newData, backgroundColor: '#2196f3' }],
  } as unknown as ChartData<'bar'>
})

const chartOptions = computed<ChartOptions<'bar'>>(() => {
  return {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: { display: false },
      tooltip: {
        callbacks: {
          label(context) {
            const dataset = context.dataset.data as unknown as Array<{
              x: string
            }>
            if (context.dataIndex === dataset.length - 1) {
              return '-'
            }
            const duration = intervalToDuration({
              start: new Date(dataset[context.dataIndex + 1]!.x),
              end: new Date(dataset[context.dataIndex]!.x),
            })
            return formatDurationFns(duration)
          },
        },
      },
    },
    scales: {
      x: { type: 'time' },
      y: { display: false },
    },
  } as ChartOptions<'bar'>
})

async function loadHeartbeats() {
  loading.value = true
  try {
    heartbeats.value = await phonesStore.getHeartbeat(100)
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await authStore.loadUser()
  await phonesStore.loadPhones()
  if (!phonesStore.owner) {
    phonesStore.setOwner(phoneId.value)
  }
  await loadHeartbeats()
})
</script>

<template>
  <VContainer fluid class="px-0 pt-0" :class="{ 'fill-height': lgAndUp }">
    <div class="w-100 h-100">
      <VAppBar height="60" :density="mdAndDown ? 'compact' : 'default'">
        <VBtn icon to="/threads">
          <VIcon :icon="mdiArrowLeft" />
        </VBtn>
        <VToolbarTitle>
          Heartbeats
          <VIcon size="12" class="mx-2" color="primary" :icon="mdiCircle" />
          <span v-if="phonesStore.owner">{{
            formatPhoneNumber(phonesStore.owner)
          }}</span>
        </VToolbarTitle>
      </VAppBar>
      <VContainer>
        <VRow>
          <VCol cols="12" class="mt-n8">
            <p>
              Every 15 minutes, the httpSMS app on your Android phone sends a
              heartbeat event to the httpsms API to show that it is alive. The
              reason for this is because the Android operating system sometimes
              kills an application to save battery
              <a
                href="https://dontkillmyapp.com"
                class="text-decoration-none hover:text-decoration-underline"
                target="_blank"
                >https://dontkillmyapp.com</a
              >.
            </p>
            <p>
              If httpSMS doesn't get any heartbeat event in a 1-hour interval,
              you will get an email notification about it so you can check if
              there is an issue with your Android phone.
            </p>
          </VCol>
          <VCol v-if="mdAndUp" cols="12" class="px-0">
            <div class="heartbeat--chart">
              <ClientOnly>
                <Bar :data="chartData" :options="chartOptions" />
              </ClientOnly>
            </div>
          </VCol>
          <VCol cols="12">
            <p>
              The table below shows the last 100 heartbeat events received from
              the httpSMS app on your Android phone.
            </p>
            <VProgressLinear
              v-if="loading"
              color="primary"
              indeterminate
              class="mb-4"
            />
            <VDataTable
              v-else
              hover
              :headers="dataTableHeaders"
              :items="dataTableItems"
              :items-per-page="100"
              :sort-by="[{ key: 'timestamp', order: 'desc' }]"
              hide-default-footer
              class="heartbeat--table"
              :row-props="
                ({ item }) =>
                  item.interval > 3600000 ? { class: 'heartbeat--missed' } : {}
              "
            >
              <template #[`item.interval`]="{ item }">
                {{ formatInterval(item.interval) }}
              </template>
              <template #[`item.owner`]="{ item }">
                {{ formatPhoneNumber(item.owner) }}
              </template>
              <template #[`item.timestamp`]="{ item }">
                {{ formatTimestamp(item.timestamp) }}
              </template>
            </VDataTable>
          </VCol>
        </VRow>
      </VContainer>
    </div>
  </VContainer>
</template>

<style lang="scss">
.heartbeat--chart {
  height: 200px;
}

.v-application {
  .heartbeat--table tbody tr.heartbeat--missed {
    background: #b71c1c;
  }
}
</style>
