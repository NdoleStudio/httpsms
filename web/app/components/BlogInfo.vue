<script setup lang="ts">
const props = defineProps<{
  date: string;
  readTime: string;
}>();
</script>

<template>
  <div class="d-flex flex-wrap ga-3 text-body-medium text-medium-emphasis mb-4">
    <span>{{ props.date }}</span>
    <span aria-hidden="true">•</span>
    <span>{{ props.readTime }}</span>
  </div>
</template>
