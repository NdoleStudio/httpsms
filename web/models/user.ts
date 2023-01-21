export interface User {
  id: string
  email: string
  api_key: string
  active_phone_id: string | null
  subscription_ends_at: string
  /** @example "8f9c71b8-b84e-4417-8408-a62274f65a08" */
  subscription_id: string
  /** @example "free" */
  subscription_name: string
  /** @example "2022-06-05T14:26:02.302718+03:00" */
  subscription_renews_at: string
  /** @example "on_trial" */
  subscription_status: string
  created_at: string
  updated_at: string
}
