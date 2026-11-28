<script setup lang="ts">
import { useDisplay } from 'vuetify'
import { mdiArrowLeft } from '@mdi/js'
import type { RouteLocationRaw } from 'vue-router'

const props = withDefaults(
  defineProps<{
    route?: RouteLocationRaw
    block?: boolean
  }>(),
  {
    route: undefined,
    block: false,
  },
)

const router = useRouter()
const { smAndDown } = useDisplay()

function goBack() {
  if (props.route) {
    router.push(props.route)
    return
  }
  const { position } = router.options.history.state
  if (typeof position === 'number' ? position > 0 : window.history.length > 1) {
    router.back()
    return
  }
  router.replace({ name: 'index' })
}
</script>

<template>
  <v-btn
    variant="tonal"
    :size="smAndDown ? 'small' : 'default'"
    :block="block"
    @click="goBack"
  >
    <v-icon :icon="mdiArrowLeft" start />
    Go Back
  </v-btn>
</template>
