<template>
  <v-app dark>
    <v-navigation-drawer
      v-if="$vuetify.breakpoint.lgAndUp && hasDrawer"
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
  poller: number | null = null

  get hasDrawer(): boolean {
    return !['login', 'index'].includes(this.$route.name ?? '')
  }

  mounted() {
    this.startPoller()
  }

  beforeDestroy(): void {
    if (this.poller) {
      clearInterval(this.poller)
    }
  }

  startPoller() {
    this.poller = window.setInterval(async () => {
      await this.$store.dispatch('setPolling', true)

      const promises = []
      if (this.$store.getters.getOwner) {
        promises.push(
          this.$store.dispatch('loadThreads'),
          this.$store.dispatch('getHeartbeat')
        )
      }

      if (this.$store.getters.hasThread) {
        promises.push(
          this.$store.dispatch(
            'loadThreadMessages',
            this.$store.getters.getThread.id
          )
        )
      }
      await Promise.all(promises)

      setTimeout(() => {
        this.$store.dispatch('setPolling', false)
      }, 1000)
    }, 10000)
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
