package requests

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ContactItem is a single contact in a create request.
type ContactItem struct {
	Name         string            `json:"name" example:"Alice Smith"`
	Emails       []string          `json:"emails"`
	PhoneNumbers []string          `json:"phone_numbers"`
	Properties   map[string]string `json:"properties"`
}

// ContactStoreRequest creates one or many contacts.
type ContactStoreRequest struct {
	request
	Contacts []ContactItem `json:"contacts"`
}

// UnmarshalJSON accepts either a JSON array of contacts or {"contacts":[...]}.
func (input *ContactStoreRequest) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "[") {
		var items []ContactItem
		if err := json.Unmarshal(data, &items); err != nil {
			return err
		}
		input.Contacts = items
		return nil
	}

	type alias ContactStoreRequest
	var wrapper alias
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return err
	}

	input.Contacts = wrapper.Contacts
	return nil
}

// Sanitize trims and normalizes each contact item.
func (input ContactStoreRequest) Sanitize() ContactStoreRequest {
	for index := range input.Contacts {
		input.Contacts[index] = sanitizeContactItem(input.Contacts[index])
	}
	return input
}

// ToContacts converts the request into persistable entities.Contact records.
func (input *ContactStoreRequest) ToContacts(userID entities.UserID) []*entities.Contact {
	now := time.Now().UTC()
	contacts := make([]*entities.Contact, 0, len(input.Contacts))
	for _, item := range input.Contacts {
		properties := item.Properties
		if properties == nil {
			properties = map[string]string{}
		}

		contacts = append(contacts, &entities.Contact{
			ID:           uuid.New(),
			UserID:       userID,
			Name:         item.Name,
			Emails:       pq.StringArray(item.Emails),
			PhoneNumbers: pq.StringArray(item.PhoneNumbers),
			Properties:   entities.ContactProperties(properties),
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}
	return contacts
}

func sanitizeContactItem(item ContactItem) ContactItem {
	item.Name = strings.TrimSpace(item.Name)
	item.Emails = sanitizeUniqueStrings(item.Emails, func(value string) string {
		return strings.ToLower(strings.TrimSpace(value))
	})
	item.PhoneNumbers = sanitizeUniqueStrings(item.PhoneNumbers, func(value string) string {
		var base request
		return base.sanitizeAddress(value)
	})
	if item.Properties == nil {
		item.Properties = map[string]string{}
	}
	return item
}
