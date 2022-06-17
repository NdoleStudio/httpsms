<template>
  <v-app dark>
    <v-navigation-drawer
      v-if="$vuetify.breakpoint.lgAndUp"
      :width="400"
      fixed
      app
    >
      <template #prepend>
        <message-thread-header></message-thread-header>
        <message-thread></message-thread>
      </template>
    </v-navigation-drawer>
    <v-main>
      <v-container
        fluid
        class="pa-0"
        :fill-height="$vuetify.breakpoint.lgAndUp"
      >
        <Nuxt />
      </v-container>
    </v-main>
  </v-app>
</template>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'

@Component
export default class DefaultLayout extends Vue {
  mounted() {
    this.$store.dispatch('loadThreads')
  }
}
</script>

<style lang="scss">
html {
  overflow-y: auto;
}

.v-application {
  .w-full {
    width: 100%;
  }
  .h-full {
    height: 100%;
  }
}
</style>
