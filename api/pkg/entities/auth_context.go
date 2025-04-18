package entities

import "github.com/google/uuid"

// AuthContext is the user gotten from an auth request
type AuthContext struct {
	ID            UserID     `json:"id"`
	PhoneAPIKeyID *uuid.UUID `json:"phone_api_key_id"`
	PhoneNumbers  []string   `json:"phone_numbers"`
	Email         string     `json:"email"`
}

// IsNoop checks if a user is empty
func (user AuthContext) IsNoop() bool {
	return user.ID == "" || user.Email == ""
}
