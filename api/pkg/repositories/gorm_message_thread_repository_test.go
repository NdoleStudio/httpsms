package repositories

import (
	"testing"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMessageThreadActivityUpdatesOwnOnlyMessageColumns(t *testing.T) {
	messageID := uuid.New()
	updates := messageThreadActivityUpdates(MessageThreadActivityUpdate{
		Timestamp: time.Date(2026, 7, 18, 7, 0, 0, 0, time.UTC),
		MessageID: messageID,
		Content:   "hello",
		Status:    entities.MessageStatusReceived,
	})

	assert.Equal(t, map[string]any{
		"order_timestamp":      time.Date(2026, 7, 18, 7, 0, 0, 0, time.UTC),
		"last_message_id":      messageID,
		"last_message_content": "hello",
		"status":               entities.MessageStatusReceived,
	}, updates)
	assert.NotContains(t, updates, "is_read")
	assert.NotContains(t, updates, "is_archived")
	assert.NotContains(t, updates, "last_read_at")
}

func TestMessageThreadStatusUpdatesReadOnly(t *testing.T) {
	isRead := true
	readAt := time.Date(2026, 7, 18, 7, 1, 0, 0, time.UTC)

	updates := messageThreadStatusUpdates(MessageThreadStatusUpdate{
		IsRead: &isRead,
		ReadAt: readAt,
	})

	assert.Equal(t, map[string]any{
		"is_read":      true,
		"last_read_at": readAt,
	}, updates)
	assert.NotContains(t, updates, "is_archived")
}

func TestMessageThreadStatusUpdatesArchiveOnly(t *testing.T) {
	isArchived := true

	updates := messageThreadStatusUpdates(MessageThreadStatusUpdate{
		IsArchived: &isArchived,
	})

	assert.Equal(t, map[string]any{"is_archived": true}, updates)
	assert.NotContains(t, updates, "is_read")
	assert.NotContains(t, updates, "last_read_at")
}
