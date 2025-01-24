export interface Message {
  contact: string
  content: string
  created_at: string
  failure_reason: string
  id: string
  last_attempted_at: string | null
  order_timestamp: string
  owner: string
  received_at: string | null
  request_received_at: string | null
  send_time: number | null
  sent_at: string
  status: string
  type: string
  updated_at: string
}

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
