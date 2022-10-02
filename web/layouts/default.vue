<template>
  <v-app>
    <v-divider v-if="$store.getters.isLocal" class="py-1 warning"></v-divider>
    <v-navigation-drawer
      v-if="$vuetify.breakpoint.lgAndUp && hasDrawer"
      :width="400"
      app
      fixed
    >
      <template #prepend>
        <v-divider
          v-if="$store.getters.isLocal"
          class="py-1 warning"
        ></v-divider>
        <message-thread-header></message-thread-header>
        <div class="overflow-y-auto v-navigation-drawer__message-thread">
          <message-thread></message-thread>
        </div>
      </template>
    </v-navigation-drawer>
    <v-main :class="{ 'has-drawer': hasDrawer && $vuetify.breakpoint.lgAndUp }">
      <Nuxt />
      <toast></toast>
    </v-main>
  </v-app>
</template>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'

@Component
export default class DefaultLayout extends Vue {
  poller: number | null = null

  get hasDrawer(): boolean {
    return ['threads', 'threads-id'].includes(this.$route.name ?? '')
  }

  mounted() {
    if (this.$route.name !== 'index') {
      this.$store.dispatch('setNextRoute', this.$route.path)
      this.$router.push({ name: 'index' })
    }
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
      if (this.$store.getters.getAuthUser && this.$store.getters.getOwner) {
        promises.push(
          this.$store.dispatch('loadThreads'),
          this.$store.dispatch('getHeartbeat')
        )
      }

      if (this.$store.getters.getAuthUser) {
        promises.push(this.$store.dispatch('loadPhones', true))
      }

      if (this.$store.getters.hasThread && this.$store.getters.getUser) {
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
.v-application {
  .w-full {
    width: 100%;
  }
  .h-full {
    height: 100%;
  }

  .has-drawer {
    .v-snack {
      padding-left: 400px;
    }
  }

  .v-navigation-drawer__message-thread {
    height: calc(100vh - 120px);

    /* width */
    &::-webkit-scrollbar {
      width: 8px;
    }

    /* Track */
    &::-webkit-scrollbar-track {
      background: #363636;
    }

    /* Handle */
    &::-webkit-scrollbar-thumb {
      background: #666666;
      border-radius: 8px;
    }
  }
  code.hljs {
    font-size: 16px;
  }
}
</style>
