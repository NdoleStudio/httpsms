type ApiFetch = <T>(
  url: string,
  options?: Record<string, unknown>,
) => Promise<T>

declare global {
  const computed: typeof import('vue').computed
  const ref: typeof import('vue').ref

  function useApi(): {
    apiFetch: ApiFetch
  }

  function useNotificationsStore(): {
    addNotification(request: {
      message: string
      type: 'error' | 'success' | 'info'
    }): void
  }

  function usePhonesStore(): {
    owner: string | null
    phones: Array<{ phone_number: string }>
    getHeartbeat(): Promise<unknown[]>
  }
}

export {}
