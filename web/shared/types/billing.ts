export interface BillingUsage {
  id: string;
  user_id: string;
  start_timestamp: string;
  end_timestamp: string;
  sent_messages: number;
  received_messages: number;
  total_cost: number;
  created_at: string;
}
