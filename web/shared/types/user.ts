export interface User {
  id: string;
  email: string;
  api_key: string;
  active_phone_id: string | null;
  timezone: string;
  subscription_id: string | null;
  subscription_name: string | null;
  subscription_status: string | null;
  notification_message_status_enabled: boolean;
  notification_webhooks_enabled: boolean;
}
