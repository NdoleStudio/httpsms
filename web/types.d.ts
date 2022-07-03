import firebase from 'firebase/compat'
import Vuetify from 'vuetify/lib'

interface Firebase {
  auth: firebase.auth.Auth
  appCheck: firebase.appCheck.AppCheck
  analytics: firebase.analytics.Analytics
}

export interface SelectItem {
  text: string
  value: string | number
}

declare module 'vue/types/vue' {
  interface Vue {
    $vuetify: typeof Vuetify
    $fire: Firebase
  }
}
