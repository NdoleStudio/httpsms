import firebase from 'firebase/compat'
import { Framework } from 'vuetify'

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
    $vuetify: Framework
    $fire: Firebase
  }
}
