package entities

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/NdoleStudio/stacktrace"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ContactProperties is a free-form key/value map persisted as a jsonb column.
type ContactProperties map[string]string

// Value implements driver.Valuer, serializing the map to JSON bytes.
func (p ContactProperties) Value() (driver.Value, error) {
	if p == nil {
		return []byte("{}"), nil
	}

	data, err := json.Marshal(p)
	if err != nil {
		return nil, stacktrace.Propagatef(err, "cannot marshal ContactProperties")
	}

	return data, nil
}

// Scan implements sql.Scanner, deserializing jsonb bytes/string into the map.
func (p *ContactProperties) Scan(src any) error {
	if src == nil {
		*p = ContactProperties{}
		return nil
	}

	var data []byte
	switch value := src.(type) {
	case []byte:
		data = value
	case string:
		data = []byte(value)
	default:
		return stacktrace.NewErrorf("unsupported type [%T] for ContactProperties", src)
	}

	if len(data) == 0 {
		*p = ContactProperties{}
		return nil
	}

	result := ContactProperties{}
	if err := json.Unmarshal(data, &result); err != nil {
		return stacktrace.Propagatef(err, "cannot unmarshal ContactProperties")
	}

	*p = result
	return nil
}

// Contact represents a saved contact belonging to a user.
type Contact struct {
	ID           uuid.UUID         `json:"id" gorm:"primaryKey;type:uuid" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	UserID       UserID            `json:"user_id" gorm:"index" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	Name         string            `json:"name" example:"Alice Smith"`
	Emails       pq.StringArray    `json:"emails" gorm:"type:text[]" swaggertype:"array,string" example:"alice@example.com"`
	PhoneNumbers pq.StringArray    `json:"phone_numbers" gorm:"type:text[]" swaggertype:"array,string" example:"+18005550199,+18005550100"`
	Properties   ContactProperties `json:"properties" gorm:"type:jsonb" swaggertype:"object,string"`
	CreatedAt    time.Time         `json:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt    time.Time         `json:"updated_at" example:"2022-06-05T14:26:02.302718+03:00"`
}

// TableName overrides the table name used by Contact.
func (Contact) TableName() string {
	return "contacts"
}
