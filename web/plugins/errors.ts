import { AxiosError } from 'axios'
import Bag from '@/plugins/bag'
import capitalize from '@/plugins/capitalize'

export type ErrorMessagesSerialized = {
  [name: string]: Array<string>
}

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

export const getErrorMessages = (error: AxiosError): ErrorMessages => {
  const errors = new ErrorMessages()
  if (
    error === null ||
    typeof error.response?.data?.data !== 'object' ||
    error.response?.data?.data === null ||
    error.response?.status !== 422
  ) {
    return errors
  }

  Object.keys(error.response.data.data).forEach((key: string) => {
    errors.addMany(key, sanitize(key, error.response?.data.data[key]))
  })

  return errors
}
