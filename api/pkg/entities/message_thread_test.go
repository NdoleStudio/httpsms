package entities

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageThreadReadFieldsHaveBackwardCompatibleDefaults(t *testing.T) {
	threadType := reflect.TypeOf(MessageThread{})

	isRead, ok := threadType.FieldByName("IsRead")
	require.True(t, ok)
	assert.Contains(t, isRead.Tag.Get("gorm"), "not null")
	assert.Contains(t, isRead.Tag.Get("gorm"), "default:true")
	assert.Equal(t, "is_read", isRead.Tag.Get("json"))

	lastReadAt, ok := threadType.FieldByName("LastReadAt")
	require.True(t, ok)
	assert.Contains(t, lastReadAt.Tag.Get("gorm"), "not null")
	assert.Contains(t, lastReadAt.Tag.Get("gorm"), "default:CURRENT_TIMESTAMP")
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
