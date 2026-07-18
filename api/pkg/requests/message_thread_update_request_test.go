package requests

import (
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMessageThreadUpdateToUpdateParamsPreservesOptionalFields(t *testing.T) {
	threadID := uuid.New()
	isRead := true
	input := MessageThreadUpdate{
		MessageThreadID: threadID.String(),
		IsRead:          &isRead,
	}

	params := input.ToUpdateParams(entities.UserID("user-id"))

	assert.Equal(t, threadID, params.MessageThreadID)
	assert.Equal(t, entities.UserID("user-id"), params.UserID)
	assert.Nil(t, params.IsArchived)
	assert.Same(t, &isRead, params.IsRead)
}
