export interface Phone {
  id: string
  user_id: string
  fcm_token: string
  phone_number: string
  created_at: string
  updated_at: string
  message_expiration_timeout: number
  messages_per_minute: number
}
