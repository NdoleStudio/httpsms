package requests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContactStoreRequest_UnmarshalArrayForm(t *testing.T) {
	var request ContactStoreRequest

	require.NoError(t, json.Unmarshal([]byte(`[{"name":"Alice","phone_numbers":["+18005550199"]}]`), &request))

	require.Len(t, request.Contacts, 1)
	assert.Equal(t, "Alice", request.Contacts[0].Name)
	assert.Equal(t, []string{"+18005550199"}, request.Contacts[0].PhoneNumbers)
}

func TestContactStoreRequest_UnmarshalObjectForm(t *testing.T) {
	var request ContactStoreRequest

	require.NoError(t, json.Unmarshal([]byte(`{"contacts":[{"name":"Bob","phone_numbers":["+18005550100"]}]}`), &request))

	require.Len(t, request.Contacts, 1)
	assert.Equal(t, "Bob", request.Contacts[0].Name)
	assert.Equal(t, []string{"+18005550100"}, request.Contacts[0].PhoneNumbers)
}

func TestContactStoreRequest_UnmarshalMalformedJSON(t *testing.T) {
	var request ContactStoreRequest

	err := json.Unmarshal([]byte(`{"contacts":[{"name":"Alice"`), &request)

	require.Error(t, err)
}

func TestContactStoreRequest_SanitizeNormalizesAndDeduplicates(t *testing.T) {
	request := ContactStoreRequest{Contacts: []ContactItem{{
		Name:         "  Alice  ",
		PhoneNumbers: []string{"18005550199", "+18005550199", " ", "+18005550100"},
		Emails:       []string{" Alice@Example.com ", "", "alice@example.com", "B@example.com", "b@example.com"},
		Properties:   map[string]string{"company": "Acme", "role": "CTO"},
	}}}

	request = request.Sanitize()

	require.Len(t, request.Contacts, 1)
	assert.Equal(t, "Alice", request.Contacts[0].Name)
	assert.Equal(t, []string{"+18005550199", "+18005550100"}, request.Contacts[0].PhoneNumbers)
	assert.Equal(t, []string{"alice@example.com", "b@example.com"}, request.Contacts[0].Emails)
	assert.Equal(t, map[string]string{"company": "Acme", "role": "CTO"}, request.Contacts[0].Properties)
}

func TestContactStoreRequest_SanitizeInitializesNilProperties(t *testing.T) {
	request := ContactStoreRequest{Contacts: []ContactItem{{
		Name:         "Alice",
		PhoneNumbers: []string{"+18005550199"},
	}}}

	request = request.Sanitize()

	require.Len(t, request.Contacts, 1)
	assert.NotNil(t, request.Contacts[0].Properties)
	assert.Empty(t, request.Contacts[0].Properties)
}

func TestContactStoreRequest_ToContactsPreservesPropertiesAndOwnership(t *testing.T) {
	request := ContactStoreRequest{Contacts: []ContactItem{{
		Name:         "Alice",
		Emails:       []string{"alice@example.com"},
		PhoneNumbers: []string{"+18005550199"},
		Properties:   map[string]string{"company": "Acme"},
	}}}.Sanitize()

	contacts := request.ToContacts(entities.UserID("user-1"))

	require.Len(t, contacts, 1)
	assert.NotZero(t, contacts[0].ID)
	assert.Equal(t, entities.UserID("user-1"), contacts[0].UserID)
	assert.Equal(t, "Alice", contacts[0].Name)
	assert.Equal(t, []string{"alice@example.com"}, []string(contacts[0].Emails))
	assert.Equal(t, []string{"+18005550199"}, []string(contacts[0].PhoneNumbers))
	assert.Equal(t, entities.ContactProperties{"company": "Acme"}, contacts[0].Properties)
	assert.False(t, contacts[0].CreatedAt.IsZero())
	assert.False(t, contacts[0].UpdatedAt.IsZero())
	assert.True(t, contacts[0].CreatedAt.Equal(contacts[0].UpdatedAt))
}

func TestContactUpdateRequest_SanitizeAndApplyTo(t *testing.T) {
	request := ContactUpdateRequest{
		Name:         "  Alice  ",
		Emails:       []string{" Alice@Example.com ", "", "alice@example.com"},
		PhoneNumbers: []string{"18005550199", "+18005550199"},
		Properties:   map[string]string{"nickname": "Al"},
	}.Sanitize()

	contact := &entities.Contact{
		ID:           uuid.MustParse("32343a19-da5e-4b1b-a767-3298a73703cb"),
		UserID:       entities.UserID("user-1"),
		Name:         "Before",
		Emails:       nil,
		PhoneNumbers: nil,
		Properties:   entities.ContactProperties{"old": "value"},
		CreatedAt:    time.Unix(1, 0).UTC(),
		UpdatedAt:    time.Unix(2, 0).UTC(),
	}

	request.ApplyTo(contact)

	assert.Equal(t, "Alice", contact.Name)
	assert.Equal(t, []string{"alice@example.com"}, []string(contact.Emails))
	assert.Equal(t, []string{"+18005550199"}, []string(contact.PhoneNumbers))
	assert.Equal(t, entities.ContactProperties{"nickname": "Al"}, contact.Properties)
	assert.Equal(t, time.Unix(1, 0).UTC(), contact.CreatedAt)
	assert.True(t, contact.UpdatedAt.After(time.Unix(2, 0).UTC()))
}

func TestContactUpdateRequest_InitializesNilProperties(t *testing.T) {
	request := ContactUpdateRequest{Name: "Alice", PhoneNumbers: []string{"+18005550199"}}.Sanitize()

	assert.NotNil(t, request.Properties)
	assert.Empty(t, request.Properties)
}

func TestContactIndex_SanitizeUsesDefaultsAndToIndexParams(t *testing.T) {
	request := ContactIndex{}

	sanitized := request.Sanitize()
	params := sanitized.ToIndexParams()

	assert.Equal(t, "0", sanitized.Skip)
	assert.Equal(t, "20", sanitized.Limit)
	assert.Equal(t, "", sanitized.Query)
	assert.Equal(t, 0, params.Skip)
	assert.Equal(t, 20, params.Limit)
	assert.Equal(t, "", params.Query)
}

func TestContactIndex_SanitizeTrimsAndConverts(t *testing.T) {
	request := ContactIndex{Skip: " 15 ", Limit: " 50 ", Query: " alice "}

	sanitized := request.Sanitize()
	params := sanitized.ToIndexParams()

	assert.Equal(t, "15", sanitized.Skip)
	assert.Equal(t, "50", sanitized.Limit)
	assert.Equal(t, "alice", sanitized.Query)
	assert.Equal(t, 15, params.Skip)
	assert.Equal(t, 50, params.Limit)
	assert.Equal(t, "alice", params.Query)
}
