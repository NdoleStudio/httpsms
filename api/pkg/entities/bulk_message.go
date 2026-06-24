package entities

import "time"

// BulkMessage represents a summary of a bulk message batch
type BulkMessage struct {
	RequestID      string    `json:"request_id" example:"bulk-httpsms-file.csv"`
	Total          int64     `json:"total" example:"150"`
	ScheduledCount int64     `json:"scheduled_count" example:"50"`
	PendingCount   int64     `json:"pending_count" example:"30"`
	FailedCount    int64     `json:"failed_count" example:"5"`
	ExpiredCount   int64     `json:"expired_count" example:"3"`
	SentCount      int64     `json:"sent_count" example:"40"`
	DeliveredCount int64     `json:"delivered_count" example:"25"`
	CreatedAt      time.Time `json:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
}
