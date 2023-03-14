<template>
  <v-container fluid fill-height>
    <v-progress-circular
      indeterminate
      size="160"
      class="mt-8 mx-auto"
      color="primary"
    ></v-progress-circular>
  </v-container>
</template>

<script lang="ts">
import Vue from 'vue'

export default Vue.extend({
  name: 'Router',
  mounted() {
    console.log('routing to new page', this.$route.path)
    this.$router.push({
      path: this.$route.path,
    })
  },
})
</script>
