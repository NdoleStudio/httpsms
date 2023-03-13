import { Context, Middleware } from '@nuxt/types'

const authMiddleware: Middleware = (context: Context) => {
  if (context.store.getters.getAuthUser === null) {
    context.redirect('/login', { to: context.route.path })
  }
}

export default authMiddleware
