package entities

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageThreadUnreadCountHasDefault(t *testing.T) {
	threadType := reflect.TypeOf(MessageThread{})

	unreadCount, ok := threadType.FieldByName("UnreadCount")
	require.True(t, ok)
	assert.Contains(t, unreadCount.Tag.Get("gorm"), "not null")
	assert.Contains(t, unreadCount.Tag.Get("gorm"), "default:0")
	assert.Equal(t, "unread_count", unreadCount.Tag.Get("json"))
}

func TestMessageThreadHasUniqueOwnerContactPerUser(t *testing.T) {
	threadType := reflect.TypeOf(MessageThread{})

	expectedTags := map[string]string{
		"UserID":  "uniqueIndex:idx_message_threads_user_owner_contact,priority:1",
		"Owner":   "uniqueIndex:idx_message_threads_user_owner_contact,priority:2",
		"Contact": "uniqueIndex:idx_message_threads_user_owner_contact,priority:3",
	}
	for fieldName, expectedTag := range expectedTags {
		field, ok := threadType.FieldByName(fieldName)
		require.True(t, ok)
		assert.Contains(t, field.Tag.Get("gorm"), expectedTag)
	}
}

func TestMessageThreadContactDetailsAreTransientAndOmittedWhenNil(t *testing.T) {
	threadType := reflect.TypeOf(MessageThread{})

	contactDetails, ok := threadType.FieldByName("ContactDetails")
	require.True(t, ok)
	assert.Equal(t, "*entities.Contact", contactDetails.Type.String())
	assert.Equal(t, "contact_details,omitempty", contactDetails.Tag.Get("json"))
	assert.Equal(t, "-", contactDetails.Tag.Get("gorm"))

	data, err := json.Marshal(MessageThread{})
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(data, &payload))
	assert.NotContains(t, payload, "contact_details")
}
