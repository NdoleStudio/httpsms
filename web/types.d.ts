import Vuetify from '@/plugins/vuetify'

declare module 'vue/types/vue' {
  interface Vue {
    $vuetify: typeof Vuetify
  }
}
