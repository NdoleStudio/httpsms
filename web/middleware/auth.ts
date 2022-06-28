import { Context, Middleware } from '@nuxt/types'
import { NotificationRequest } from '~/store'

const authMiddleware: Middleware = async (context: Context) => {
  if (context.store.getters.getAuthUser === null) {
    const notification: NotificationRequest = {
      message: 'Login to continue',
      type: 'info',
    }
    await context.store.dispatch('addNotification', notification)

    context.redirect('/login', { to: context.route.path })
  }
}

export default authMiddleware
