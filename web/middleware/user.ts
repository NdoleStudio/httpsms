import { Context } from '@nuxt/types'
import { User, Auth } from 'firebase/auth'
import { User as StateUser } from '~/store'
import { setAuthHeader } from '~/plugins/axios'

export default async function (context: Context) {
  await setUser(context)
}

const setUser = (context: Context): Promise<User | null> => {
  return new Promise((resolve, reject) => {
    const unsubscribe = (context.app.$fire.auth as Auth).onAuthStateChanged(
      async (user) => {
        unsubscribe()
        let stateUser: StateUser | null = null
        if (user) {
          stateUser = {
            id: user.uid,
          }
          setAuthHeader(await user.getIdToken())
        }
        context.store.dispatch('setUser', stateUser).finally(() => {
          resolve(user)
        })
      },
      reject
    )
  })
}
