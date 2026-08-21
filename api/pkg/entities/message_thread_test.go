package entities

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageThreadUnreadFields(t *testing.T) {
	threadType := reflect.TypeOf(MessageThread{})

	_, hasIsRead := threadType.FieldByName("IsRead")
	assert.False(t, hasIsRead)

	unreadCount, ok := threadType.FieldByName("UnreadCount")
	require.True(t, ok)
	assert.Equal(t, "unread_count", unreadCount.Tag.Get("json"))
	assert.Contains(t, unreadCount.Tag.Get("gorm"), "not null")
	assert.Contains(t, unreadCount.Tag.Get("gorm"), "default:0")

	lastReadAt, ok := threadType.FieldByName("LastReadAt")
	require.True(t, ok)
	assert.Equal(t, "-", lastReadAt.Tag.Get("json"))
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

func TestMessageThreadUnreadItemUsesMessageIDAsPrimaryKey(t *testing.T) {
	itemType := reflect.TypeOf(MessageThreadUnreadItem{})

	messageID, ok := itemType.FieldByName("MessageID")
	require.True(t, ok)
	assert.Contains(t, messageID.Tag.Get("gorm"), "primaryKey")
}
