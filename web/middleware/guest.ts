import { Context, Middleware } from '@nuxt/types'

const authMiddleware: Middleware = (context: Context) => {
  if (context.store.getters.getAuthUser !== null) {
    if (context.store.getters.getNextRoute) {
      context.redirect(context.store.getters.getNextRoute)
      context.store.dispatch('setNextRoute', null)
    }
    context.redirect('/threads')
  }
}

export default authMiddleware
