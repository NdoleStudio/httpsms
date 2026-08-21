package requests

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageThreadUpdateJSONDistinguishesUnreadCountZeroFromOmitted(t *testing.T) {
	t.Run("zero value is preserved as a pointer", func(t *testing.T) {
		var input MessageThreadUpdate

		err := json.Unmarshal([]byte(`{"unread_count":0}`), &input)

		require.NoError(t, err)
		require.NotNil(t, input.UnreadCount)
		assert.Equal(t, uint(0), *input.UnreadCount)
	})

	t.Run("omitted unread count stays nil", func(t *testing.T) {
		var input MessageThreadUpdate

		err := json.Unmarshal([]byte(`{"is_archived":true}`), &input)

		require.NoError(t, err)
		assert.Nil(t, input.UnreadCount)
	})
}

func TestMessageThreadUpdateToUpdateParamsPreservesOptionalFields(t *testing.T) {
	threadID := uuid.New()
	isArchived := true
	unreadCount := uint(0)
	input := MessageThreadUpdate{
		MessageThreadID: threadID.String(),
		IsArchived:      &isArchived,
		UnreadCount:     &unreadCount,
	}

	params := input.ToUpdateParams(entities.UserID("user-id"))

	assert.Equal(t, threadID, params.MessageThreadID)
	assert.Equal(t, entities.UserID("user-id"), params.UserID)
	assert.Same(t, &isArchived, params.IsArchived)
	assert.Same(t, &unreadCount, params.UnreadCount)
}

func TestMessageThreadUpdateUnreadCountSwaggerAllowsExactlyZero(t *testing.T) {
	field, ok := reflect.TypeOf(MessageThreadUpdate{}).FieldByName("UnreadCount")
	require.True(t, ok)
	assert.Equal(t, "0", field.Tag.Get("minimum"))
	assert.Equal(t, "0", field.Tag.Get("maximum"))
}
