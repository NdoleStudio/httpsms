export interface BillingUsage {
  id: string;
  user_id: string;
  period_start: string;
  period_end: string;
  sent_messages: number;
  received_messages: number;
}
