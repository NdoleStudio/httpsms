import { Context, Middleware } from '@nuxt/types'

const guestMiddleware: Middleware = (context: Context) => {
  if (context.store.getters.getAuthUser !== null) {
    context.redirect('/threads')
  }
}

export default guestMiddleware
