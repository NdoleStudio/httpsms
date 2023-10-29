<template>
  <div class="d-flex mb-6 mt-8">
    <v-avatar class="mb-n2">
      <v-img :src="require('@/assets/img/arnold.png')"></v-img>
    </v-avatar>
    <div class="ml-2">
      <p class="subtitle-1 mb-n1">Acho Arnold</p>
      <a
        class="mb-n4 text-decoration-none text--primary"
        href="https://twitter.com/acho_arnold"
      >
        <v-icon color="#1DA1F2">{{ mdiTwitter }}</v-icon>
      </a>
      <a
        class="ml-2 text-decoration-none text--primary"
        href="https://github.com/AchoArnold"
      >
        <v-icon color="#FFFFFF">{{ mdiGithub }}</v-icon>
      </a>
    </div>
  </div>
</template>

<script lang="ts">
import { mdiTwitter, mdiGithub } from '@mdi/js'

export default {
  name: 'BlogAuthorBio',
  data() {
    return {
      mdiTwitter,
      mdiGithub,
    }
  },
}
</script>
