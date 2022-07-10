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
          <v-col cols="12" md="9" offset-md="1" xl="8" offset-xl="2">
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
            <h5 class="text-h4 mb-3 mt-12">Phones</h5>
            <p>
              List of mobile phones which are registered for sending and
              receiving SMS messages.
            </p>
            <v-simple-table>
              <template #default>
                <thead>
                  <tr class="text-uppercase">
                    <th v-if="$vuetify.breakpoint.lgAndUp" class="text-left">
                      ID
                    </th>
                    <th class="text-left">Phone Number</th>
                    <th class="text-center">Fcm Token</th>
                    <th class="text-center">Updated At</th>
                    <th class="text-center">Action</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="phone in $store.getters.getPhones" :key="phone.id">
                    <td v-if="$vuetify.breakpoint.lgAndUp" class="text-left">
                      {{ phone.id }}
                    </td>
                    <td>{{ phone.phone_number | phoneNumber }}</td>
                    <td>
                      <div class="d-flex justify-center">
                        <v-checkbox
                          readonly
                          class="mx-auto"
                          :input-value="true"
                          color="success"
                        ></v-checkbox>
                      </div>
                    </td>
                    <td class="text-center">
                      {{ phone.updated_at | timestamp }}
                    </td>
                    <td class="text-center">
                      <v-btn
                        :icon="$vuetify.breakpoint.mdAndDown"
                        small
                        color="error"
                        @click.prevent="deletePhone(phone.id)"
                      >
                        <v-icon small>mdi-delete</v-icon>
                        <span v-if="!$vuetify.breakpoint.mdAndDown">
                          Delete
                        </span>
                      </v-btn>
                    </td>
                  </tr>
                </tbody>
              </template>
            </v-simple-table>
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
    if (!this.$store.getters.getAuthUser) {
      this.$store.dispatch('setNextRoute', this.$route.path)
      this.$router.push({ name: 'index' })
      return
    }
    this.$store.dispatch('loadUser')
  },

  methods: {
    deletePhone(phoneId) {
      this.$store.dispatch('deletePhone', phoneId)
    },
  },
}
</script>
