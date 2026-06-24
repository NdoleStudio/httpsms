<script setup lang="ts">
import { mdiTwitter, mdiGithub } from '@mdi/js'
</script>

<template>
  <div class="d-flex mb-6 mt-8">
    <v-avatar image="/img/arnold.png" />
    <div class="ml-2 mt-n7">
      <p class="text-title-large mb-n1">Acho Arnold</p>
      <a
        class="mb-n4 text-decoration-none text-on-surface"
        href="https://twitter.com/acho_arnold"
      >
        <v-icon color="#1DA1F2" :icon="mdiTwitter" />
      </a>
      <a
        class="ml-2 text-decoration-none text-on-surface"
        href="https://github.com/AchoArnold"
      >
        <v-icon color="#FFFFFF" :icon="mdiGithub" />
      </a>
    </div>
  </div>
</template>
