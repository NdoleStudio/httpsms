<script setup lang="ts">
import { computed } from 'vue'
import { formatBillingPeriodDateOrdinalParts } from '~/utils/filters'

const props = defineProps<{ value: string }>()

const parts = computed(() => formatBillingPeriodDateOrdinalParts(props.value))
</script>

<template>
  <span
    >{{ parts.leading }}<sup>{{ parts.suffix }}</sup
    >{{ parts.trailing }}</span
  >
</template>
