export interface BulkMessageOrder {
  request_id: string
  created_at: string
  total: number
  pending_count: number
  scheduled_count: number
  sent_count: number
  delivered_count: number
  failed_count: number
  expired_count: number
}
