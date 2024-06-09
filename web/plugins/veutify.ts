import Vue from 'vue'
import { Route } from 'vue-router'
import { DataOptions } from 'vuetify'

export type VForm = Vue & {
  validate: () => boolean
  resetValidation: () => boolean
  reset: () => void
}

export type FormInputType = string | null | File

export type FormValidationRule = (value: FormInputType) => string | boolean

export type FormValidationRules = Array<FormValidationRule>

export interface SelectItem {
  text: string
  value: string | number | boolean
}

export interface DatatableFooterProps {
  itemsPerPage: number
  itemsPerPageOptions: Array<number>
}

export const DefaultFooterProps: DatatableFooterProps = {
  itemsPerPage: 100,
  itemsPerPageOptions: [10, 50, 100, 200],
}

export type ParseParamsResponse = {
  options: DataOptions
  query: string | null
}

export const parseFilterOptionsFromParams = (
  route: Route,
  options: DataOptions,
): ParseParamsResponse => {
  let query = null
  Object.keys(route.query).forEach((value: string) => {
    if (value === 'itemsPerPage') {
      options.itemsPerPage = parseInt(
        (route.query[value] as string) ?? options.itemsPerPage.toString(),
      )
    }

    if (value === 'sortBy') {
      options.sortBy = [(route.query[value] as string) ?? options.sortBy[0]]
    }

    if (value === 'sortDesc') {
      options.sortDesc = [!(route.query[value] === 'false')]
    }

    if (value === 'page') {
      options.page = parseInt(
        (route.query[value] as string) ?? options.page.toString(),
      )
    }
    if (value === 'query') {
      query = route.query[value]
    }
  })
  return { options, query }
}
