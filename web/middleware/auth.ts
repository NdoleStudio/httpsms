import { Context, Middleware } from '@nuxt/types'

const authMiddleware: Middleware = (context: Context) => {
  if (context.store.getters.getAuthUser === null) {
    if (context.store.getters.getNextRoute) {
      context.redirect('/login', { to: context.route.path })
      context.store.dispatch('setNextRoute', null)
    }
  }
}

export default authMiddleware
