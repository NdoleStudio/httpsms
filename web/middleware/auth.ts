import { Context, Middleware } from '@nuxt/types'

const authMiddleware: Middleware = (context: Context) => {
  if (context.store.getters.getAuthUser === null) {
    context.redirect('/login')
  }
}

export default authMiddleware
