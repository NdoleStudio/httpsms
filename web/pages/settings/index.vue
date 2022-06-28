<template>
  <v-container fluid class="pa-0" :fill-height="$vuetify.breakpoint.lgAndUp">
    <div class="w-full h-full">
      <v-app-bar :dense="$vuetify.breakpoint.mdAndDown">
        <v-btn icon to="/">
          <v-icon>mdi-arrow-left</v-icon>
        </v-btn>
        <v-toolbar-title> Settings </v-toolbar-title>
      </v-app-bar>
      <v-container>
        <v-row>
          <v-col cols="12" md="8" offset-md="2" xl="6" offset-xl="3">
            <h5 class="text-h4 mb-3 mt-3">API Key</h5>
            <p>
              Use your API Key in the <code>X-API-Key</code> HTTP Header when
              sending requests to
              <code>https://api.httpsms.com</code> endpoints.
            </p>
            <div v-if="apiKey === ''" class="mb-n9 pl-3 pt-5">
              <v-progress-circular
                :size="20"
                :width="2"
                color="primary"
                indeterminate
              ></v-progress-circular>
            </div>
            <v-text-field
              v-else
              :append-icon="apiKeyShow ? 'mdi-eye' : 'mdi-eye-off'"
              :type="apiKeyShow ? 'text' : 'password'"
              :value="apiKey"
              readonly
              name="api-key"
              outlined
              class="mb-n2"
              @click:append="apiKeyShow = !apiKeyShow"
            ></v-text-field>
            <copy-button
              :block="$vuetify.breakpoint.mdAndDown"
              :large="$vuetify.breakpoint.mdAndDown"
              :value="apiKey"
              copy-text="Copy API Key"
              notification-text="API Key copied successfully"
            ></copy-button>
          </v-col>
        </v-row>
      </v-container>
    </div>
  </v-container>
</template>

<script>
export default {
  name: 'SettingsIndex',
  middleware: ['auth'],
  data() {
    return {
      apiKeyShow: false,
    }
  },
  computed: {
    apiKey() {
      if (this.$store.getters.getUser === null) {
        return ''
      }
      return this.$store.getters.getUser.api_key
    },
  },
  mounted() {
    this.$store.dispatch('loadUser')
  },
}
</script>
