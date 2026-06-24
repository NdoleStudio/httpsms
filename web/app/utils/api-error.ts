/**
 * Shape of the error object thrown by ofetch/$fetch for failed API requests.
 */
export interface ApiError {
  status?: number
  data?: {
    message?: string
    data?: Record<string, string[]>
  }
}

export function toApiError(error: unknown): ApiError {
  if (error !== null && typeof error === 'object') {
    return error as ApiError
  }
  return {}
}

export function getApiErrorMessage(error: unknown, fallback: string): string {
  return toApiError(error).data?.message ?? fallback
}
