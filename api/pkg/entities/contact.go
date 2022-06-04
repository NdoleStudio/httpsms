package entities

import (
	"time"

	"github.com/google/uuid"
)

// Contact stores a telephone contact
type Contact struct {
	ID          uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;"`
	PhoneNumber string    `json:"phone_number"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
