export interface User {
  id: string;
  email: string;
  api_key: string;
  active_phone_id: string | null;
  timezone: string;
  subscription_id: string | null;
  subscription_name: string | null;
  subscription_status: string | null;
  subscription_renews_at: string | null;
  subscription_ends_at: string | null;
  notification_heartbeat_enabled: boolean;
  notification_message_status_enabled: boolean;
  notification_newsletter_enabled: boolean;
  notification_webhook_enabled: boolean;
}
