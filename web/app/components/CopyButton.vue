<script setup lang="ts">
import { useDisplay } from 'vuetify'
import { mdiContentCopy } from '@mdi/js'

const props = withDefaults(
  defineProps<{
    value: string
    color?: string
    block?: boolean
    large?: boolean
    copyText?: string
    notificationText?: string
  }>(),
  {
    color: 'default',
    block: false,
    large: false,
    copyText: 'Copy',
    notificationText: 'Copied',
  },
)

const { smAndDown } = useDisplay()
const notificationsStore = useNotificationsStore()
const disabled = ref(false)

async function copy() {
  disabled.value = true
  await navigator.clipboard.writeText(props.value)
  notificationsStore.addNotification({
    message: props.notificationText,
    type: 'success',
  })
  setTimeout(() => {
    disabled.value = false
  }, 5000)
}
</script>

<template>
  <v-btn
    :disabled="disabled"
    :color="disabled ? 'default' : color"
    :size="smAndDown ? 'small' : large ? 'large' : 'default'"
    :block="block"
    variant="flat"
    @click="copy"
  >
    <v-icon start :icon="mdiContentCopy" />
    {{ copyText }}
  </v-btn>
</template>
