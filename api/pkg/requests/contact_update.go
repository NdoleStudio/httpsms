package requests

import (
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/lib/pq"
)

// ContactUpdateRequest updates an existing contact.
type ContactUpdateRequest struct {
	request
	Name         string            `json:"name" example:"Alice Smith"`
	Emails       []string          `json:"emails"`
	PhoneNumbers []string          `json:"phone_numbers"`
	Properties   map[string]string `json:"properties"`
}

// Sanitize trims and normalizes the update request.
func (input ContactUpdateRequest) Sanitize() ContactUpdateRequest {
	input.Name = strings.TrimSpace(input.Name)
	input.Emails = sanitizeUniqueStrings(input.Emails, func(value string) string {
		return strings.ToLower(strings.TrimSpace(value))
	})
	input.PhoneNumbers = sanitizeUniqueStrings(input.PhoneNumbers, func(value string) string {
		var base request
		return base.sanitizeAddress(value)
	})
	if input.Properties == nil {
		input.Properties = map[string]string{}
	}
	return input
}

// ApplyTo mutates an existing contact with the update values.
func (input *ContactUpdateRequest) ApplyTo(contact *entities.Contact) {
	contact.Name = input.Name
	contact.Emails = pq.StringArray(input.Emails)
	contact.PhoneNumbers = pq.StringArray(input.PhoneNumbers)
	contact.Properties = entities.ContactProperties(input.Properties)
	contact.UpdatedAt = time.Now().UTC()
}
