<template>
  <v-main>
    <v-container fluid fill-height>
      <v-row align="center" justify="center">
        <v-col
          cols="12"
          md="5"
          xl="3"
          class="text-center mt-16"
          :class="{
            'px-6': $vuetify.breakpoint.mdAndDown,
            'px-16': !$vuetify.breakpoint.mdAndDown,
          }"
        >
          <h2 class="text-h4 text--secondary mt-16 mb-4">
            <img
              class="mx-auto d-inline-block"
              :src="require('@/assets/img/logo.svg')"
              style="max-width: 32px"
              alt="httpSMS Logo"
            />
            Loading the httpSMS dashboard
          </h2>
          <v-progress-circular
            indeterminate
            size="160"
            class="mt-8"
            color="primary"
          ></v-progress-circular>
        </v-col>
      </v-row>
    </v-container>
  </v-main>
</template>

<script>
export default {
  name: 'LoadingDashboard',
}
</script>
