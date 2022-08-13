package entities

import (
	"time"

	"github.com/google/uuid"
)

// UserID is the MessageID of a user
type UserID string

// User stores information about a user
type User struct {
	ID                UserID     `json:"id" gorm:"primaryKey;type:string;" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	Email             string     `json:"email" example:"name@email.com"` // gorm:"uniqueIndex"
	APIKey            string     `json:"api_key" example:"xyz"`          // gorm:"uniqueIndex"
	ActivePhoneID     *uuid.UUID `json:"active_phone_id" gorm:"type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	NotificationEmail *string    `json:"notification_email" example:"john@example.com"`
	CreatedAt         time.Time  `json:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt         time.Time  `json:"updated_at" example:"2022-06-05T14:26:10.303278+03:00"`
}
