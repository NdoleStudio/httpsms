package entities

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// PhoneAPIKey represents the API key for a phone
type PhoneAPIKey struct {
	ID           uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	Name         string         `json:"name" example:"Business Phone Key"`
	UserID       UserID         `json:"user_id" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	UserEmail    string         `json:"user_email" example:"user@gmail.com"`
	PhoneNumbers pq.StringArray `json:"phone_numbers" example:"[+18005550199,+18005550100]" gorm:"type:text[]" swaggertype:"array,string"`
	PhoneIDs     pq.StringArray `json:"phone_ids" example:"[32343a19-da5e-4b1b-a767-3298a73703cb,32343a19-da5e-4b1b-a767-3298a73703cc]" gorm:"type:text[]" swaggertype:"array,string"`
	APIKey       string         `json:"api_key" example:"pk_DGW8NwQp7mxKaSZ72Xq9v67SLqSbWQvckzzmK8D6rvd7NywSEkdMJtuxKyEkYnCY"`
	CreatedAt    time.Time      `json:"created_at"  example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt    time.Time      `json:"updated_at"  example:"2022-06-05T14:26:02.302718+03:00"`
}

// TableName overrides the table name used by PhoneAPIKey
func (PhoneAPIKey) TableName() string {
	return "phone_api_keys"
}
