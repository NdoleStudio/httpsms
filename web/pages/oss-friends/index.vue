<template>
  <v-container class="mt-16">
    <v-row>
      <v-col cols="10" offset="1" class="mt-16">
        <h1
          class="mb-4 text-center"
          :class="{
            'text-h2': $vuetify.breakpoint.lgAndUp,
            'text-h3': !$vuetify.breakpoint.lgAndUp,
          }"
        >
          Open Source Friends
        </h1>
        <p class="text-h5 text--secondary text-center">
          Here are some of our favorite open-source projects.
        </p>
        <v-row class="mb-8 mt-8">
          <v-col v-if="apps.length == 0" class="text-center my-16">
            <v-progress-circular :size="100" color="primary" indeterminate />
          </v-col>
          <v-col v-for="app in apps" :key="app.href" cols="12" md="4">
            <v-card>
              <v-card-text class="mb-0 pb-0">
                <h4 class="text-h5 text--primary">
                  {{ app.name }}
                </h4>
                <p class="mt-2 app-description mb-0">{{ app.description }}</p>
              </v-card-text>
              <v-card-actions>
                <v-btn text :href="app.href" color="primary">
                  Learn More
                  <v-icon right>{{ mdiOpenInNew }}</v-icon>
                </v-btn>
              </v-card-actions>
            </v-card>
          </v-col>
        </v-row>
        <div class="px-16">
          <v-divider />
        </div>
        <div class="text-center mt-8 mb-4">
          <back-button />
        </div>
      </v-col>
    </v-row>
  </v-container>
</template>

<script lang="ts">
import Vue from 'vue'
import { mdiOpenInNew } from '@mdi/js'

type AppData = {
  name: string
  description: string
  href: string
}

export default Vue.extend({
  name: 'OpenSourceFriendsIndex',
  layout: 'website',
  data: () => ({
    mdiOpenInNew,
    apps: [] as AppData[],
  }),
  head() {
    return {
      title: 'Open Source Friends - httpSMS',
    }
  },
  mounted() {
    fetch(
      'https://corsproxy.io/?https%3A%2F%2Fformbricks.com%2Fapi%2Foss-friends',
    )
      .then((res) => res.json())
      .then((response: any) => {
        this.apps = response.data
      })
  },
})
</script>

<style lang="scss" scoped>
.app-description {
  display: -webkit-box;
  -webkit-line-clamp: 3;
  line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
  height: 4.5em;
}
</style>
