export interface SearchMessagesRequest {
  owners: string[]
  types: string[]
  statuses: string[]
  query: string
  sort_by: string
  token?: string
  sort_descending: boolean
  skip: number
  limit: number
}
