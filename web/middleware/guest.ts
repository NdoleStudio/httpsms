import { Context, Middleware } from '@nuxt/types'

const authMiddleware: Middleware = (context: Context) => {
  if (context.store.getters.getUser !== null) {
    context.redirect('/threads')
  }
}

export default authMiddleware
