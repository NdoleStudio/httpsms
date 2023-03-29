export interface Phone {
  id: string
  user_id: string
  fcm_token: string
  phone_number: string
  is_dual_sim: boolean
  created_at: string
  updated_at: string
  max_send_attempts: number
  message_expiration_seconds: number
  messages_per_minute: number
}
