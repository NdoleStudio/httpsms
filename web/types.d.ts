import firebase from 'firebase/compat'
import Vuetify from '@/plugins/vuetify'

interface Firebase {
  auth: firebase.auth.Auth
  appCheck: firebase.appCheck.AppCheck
  analytics: firebase.analytics.Analytics
}

declare module 'vue/types/vue' {
  interface Vue {
    $vuetify: typeof Vuetify
    $fire: Firebase
  }
}
