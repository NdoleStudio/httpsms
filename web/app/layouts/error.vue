<script setup lang="ts">
const props = defineProps<{
  error: {
    statusCode: number;
    message: string;
  };
}>();
</script>

<template>
  <v-app>
    <v-main>
      <v-container class="text-center mt-16">
        <h1 class="text-display-large">{{ error.statusCode }}</h1>
        <p class="text-headline-medium mt-4">{{ error.message }}</p>
        <v-btn color="primary" class="mt-8" to="/">Go Home</v-btn>
      </v-container>
    </v-main>
  </v-app>
</template>
