package validators

import (
	"context"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateUpdateRequiresAtLeastOneStatusField(t *testing.T) {
	validator := &MessageThreadHandlerValidator{}
	request := requests.MessageThreadUpdate{
		MessageThreadID: uuid.NewString(),
	}

	errors := validator.ValidateUpdate(context.Background(), request)

	assert.NotEmpty(t, errors.Get("payload"))
}

func TestValidateUpdateAcceptsUnreadCountReset(t *testing.T) {
	validator := &MessageThreadHandlerValidator{}
	zero := uint(0)
	request := requests.MessageThreadUpdate{
		MessageThreadID: uuid.NewString(),
		UnreadCount:     &zero,
	}

	errors := validator.ValidateUpdate(context.Background(), request)

	assert.Empty(t, errors)
}

func TestValidateUpdateRejectsUnreadCountValuesOtherThanZero(t *testing.T) {
	validator := &MessageThreadHandlerValidator{}
	one := uint(1)
	request := requests.MessageThreadUpdate{
		MessageThreadID: uuid.NewString(),
		UnreadCount:     &one,
	}

	errors := validator.ValidateUpdate(context.Background(), request)

	require.NotNil(t, errors)
	assert.Contains(t, errors, "unread_count")
	assert.Equal(t, "must be 0", errors.Get("unread_count"))
}

func TestValidateUpdateAcceptsArchiveOnlyAndCombinedPayloads(t *testing.T) {
	validator := &MessageThreadHandlerValidator{}
	isArchived := true
	zero := uint(0)

	testCases := map[string]requests.MessageThreadUpdate{
		"archive only": {
			MessageThreadID: uuid.NewString(),
			IsArchived:      &isArchived,
		},
		"combined payload": {
			MessageThreadID: uuid.NewString(),
			IsArchived:      &isArchived,
			UnreadCount:     &zero,
		},
	}

	for name, request := range testCases {
		t.Run(name, func(t *testing.T) {
			errors := validator.ValidateUpdate(context.Background(), request)

			assert.Empty(t, errors)
		})
	}
}
