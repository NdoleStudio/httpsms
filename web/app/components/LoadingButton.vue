<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    type?: string
    block?: boolean
    large?: boolean
    xLarge?: boolean
    tile?: boolean
    text?: boolean
    small?: boolean
    color?: string
    icon?: string | null
    loading: boolean
  }>(),
  {
    type: 'submit',
    block: false,
    large: false,
    xLarge: false,
    tile: false,
    text: false,
    small: false,
    color: 'primary',
    icon: null,
  },
)

const emit = defineEmits<{
  click: []
  'update:loading': [value: boolean]
}>()

const isClicked = ref(false)

watch(
  () => props.loading,
  (submitting) => {
    if (!submitting && isClicked.value) {
      isClicked.value = false
    }
  },
)

function onClick() {
  isClicked.value = true
  emit('click')
}

const size = computed(() => {
  if (props.xLarge) return 'x-large'
  if (props.large) return 'large'
  if (props.small) return 'small'
  return 'default'
})
</script>

<template>
  <v-btn
    :block="block"
    :type="type"
    :size="size"
    :color="color"
    :variant="text ? 'text' : 'elevated'"
    :disabled="loading"
    @click.prevent="onClick"
  >
    <v-progress-circular
      v-if="isClicked"
      :size="small ? 20 : 25"
      color="grey"
      class="mr-2"
      indeterminate
    />
    <v-icon v-if="icon && !loading" start :icon="icon" />
    <slot />
  </v-btn>
</template>
