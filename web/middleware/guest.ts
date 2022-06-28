import { Context, Middleware } from '@nuxt/types'

const authMiddleware: Middleware = (context: Context) => {
  if (context.store.getters.getAuthUser !== null) {
    context.redirect('/threads')
  }
}

export default authMiddleware
