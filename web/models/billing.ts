export interface BillingUsage {
  id: string
  start_timestamp: string
  end_timestamp: string
  user_id: string
  sent_messages: number
  received_messages: number
  total_cost: number
  created_at: string
}
