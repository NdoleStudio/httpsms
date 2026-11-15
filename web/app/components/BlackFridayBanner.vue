<script setup lang="ts">
import { mdiChevronRight, mdiClose } from '@mdi/js'
import {
  BLACK_FRIDAY_END,
  getBlackFridayCountdown,
  isBlackFridayEligible,
} from '~/utils/black-friday'

const route = useRoute()
const authStore = useAuthStore()
const dismissed = useCookie<boolean>('httpsms_bf2026_dismissed', {
  default: () => false,
  expires: new Date(BLACK_FRIDAY_END),
  sameSite: 'lax',
  path: '/',
})
const now = ref(Date.now())
const countdown = computed(() => getBlackFridayCountdown(now.value))
const visible = computed(
  () =>
    !dismissed.value &&
    countdown.value !== null &&
    isBlackFridayEligible(
      String(route.name),
      authStore.authStateChanged,
      authStore.authUser !== null,
      authStore.user?.subscription_name,
    ),
)
const bannerElement = useTemplateRef<HTMLElement>('bannerElement')
const bannerHeight = ref(0)
const contentColumns = computed(() => {
  switch (route.name) {
    case 'settings':
    case 'billing':
    case 'phone-api-keys':
      return { md: 9, offsetMd: 1, xl: 8, offsetXl: 2 }
    case 'bulk-messages':
      return { md: 10, offsetMd: 1, xxl: 8, offsetXxl: 2 }
    case 'messages':
      return { md: 8, offsetMd: 2, xl: 6, offsetXl: 3 }
    default:
      return {}
  }
})
const hasPagePadding = computed(() =>
  ['settings', 'messages'].includes(String(route.name)),
)
let resizeObserver: ResizeObserver | undefined
let timer: ReturnType<typeof setInterval> | undefined

watch(bannerElement, (element) => {
  resizeObserver?.disconnect()
  if (!element) return

  const updateHeight = () => {
    bannerHeight.value = element.getBoundingClientRect().height
  }
  updateHeight()
  resizeObserver = new ResizeObserver(updateHeight)
  resizeObserver.observe(element)
})

watch(
  visible,
  (show) => {
    clearInterval(timer)
    if (show) {
      now.value = Date.now()
      timer = setInterval(() => {
        now.value = Date.now()
      }, 1000)
    }
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  clearInterval(timer)
  resizeObserver?.disconnect()
})
</script>

<template>
  <v-layout-item
    v-if="visible"
    position="top"
    :order="-1"
    :size="bannerHeight"
    model-value
  >
    <div ref="bannerElement">
      <v-toolbar
        class="black-friday-banner"
        color="transparent"
        tag="aside"
        aria-label="Black Friday promotion"
        flat
      >
        <v-container fluid class="py-0" :class="{ 'px-0': !hasPagePadding }">
          <v-container :class="route.name === 'settings' ? 'pa-0' : 'py-0'">
            <v-row class="my-0">
              <v-col cols="12" v-bind="contentColumns" class="py-3">
                <div class="d-flex align-center ga-3">
                  <div
                    class="black-friday-banner-offer d-flex align-center ga-3"
                  >
                    <strong class="text-warning text-uppercase"
                      >Black Friday</strong
                    >
                    <span>
                      Get <strong>40% off</strong> the yearly plans with
                      discount code
                      <strong class="text-warning">BF2026</strong> at checkout.
                    </span>
                    <v-btn
                      to="/billing"
                      class="d-none d-sm-inline-flex"
                      color="warning"
                      size="small"
                      :append-icon="mdiChevronRight"
                    >
                      Use now
                    </v-btn>
                  </div>
                  <div
                    class="black-friday-banner-timer d-none d-sm-block text-right"
                  >
                    <strong
                      class="text-no-wrap"
                      role="timer"
                      aria-label="Time remaining"
                      aria-live="off"
                    >
                      {{ countdown }}
                    </strong>
                  </div>
                  <v-btn
                    :icon="mdiClose"
                    class="align-self-start align-self-sm-center flex-shrink-0"
                    color="error"
                    density="compact"
                    size="small"
                    variant="text"
                    aria-label="Dismiss Black Friday promotion"
                    @click="dismissed = true"
                  />
                </div>
              </v-col>
            </v-row>
          </v-container>
        </v-container>
      </v-toolbar>
    </div>
  </v-layout-item>
</template>

<style scoped>
.black-friday-banner {
  background: linear-gradient(110deg, #090909 40%, #10545b);
}

.black-friday-banner :deep([class~='v-toolbar__content']) {
  height: auto !important;
  min-height: 64px;
}

.black-friday-banner-offer {
  flex: 1;
  flex-wrap: wrap;
  min-width: 0;
}

.black-friday-banner-timer {
  font-variant-numeric: tabular-nums;
}
</style>
