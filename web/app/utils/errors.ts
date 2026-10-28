import Bag from './bag'
import { capitalize } from './capitalize'
import { toApiError } from './api-error'

export class ErrorMessages extends Bag<string> {}

const sanitize = (key: string, values: Array<string>): Array<string> => {
  return values.map((value: string) => {
    return capitalize(
      value
        .split(key)
        .join(key.replace('_', ' '))
        .split('_')
        .join(' ')
        .split('-')
        .join(' ')
        .split(' char')
        .join(' character')
        .split(' field ')
        .join(' '),
    )
  })
}

export const getErrorMessages = (error: unknown): ErrorMessages => {
  const errors = new ErrorMessages()
  const apiError = toApiError(error)
  const response = 'response' in apiError ? toApiError(apiError.response) : {}
  const data = apiError.data?.data ?? response.data?.data
  const status = apiError.status ?? response.status
  if (typeof data !== 'object' || data === null || status !== 422) {
    return errors
  }

  Object.entries(data).forEach(([key, values]) => {
    errors.addMany(key, sanitize(key, values))
  })

  return errors
}
