package requests

import (
	"github.com/google/uuid"
)

// PhoneDelete is the payload for deleting
type PhoneDelete struct {
	request
	PhoneID string `json:"phoneID" swaggerignore:"true"` // used internally for validation
}

// PhoneIDUuid returns the phoneID as uuid.UUID
func (input *PhoneDelete) PhoneIDUuid() uuid.UUID {
	return uuid.MustParse(input.PhoneID)
}
