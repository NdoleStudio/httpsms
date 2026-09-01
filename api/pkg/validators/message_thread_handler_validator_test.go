package validators

import (
	"context"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestValidateUpdateRequiresAtLeastOneStatusField(t *testing.T) {
	validator := &MessageThreadHandlerValidator{}
	request := requests.MessageThreadUpdate{
		MessageThreadID: uuid.NewString(),
	}

	errors := validator.ValidateUpdate(context.Background(), request)

	assert.NotEmpty(t, errors.Get("payload"))
}

func TestValidateUpdateAcceptsUnreadCountOnlyUpdate(t *testing.T) {
	validator := &MessageThreadHandlerValidator{}
	unreadCount := uint(0)
	request := requests.MessageThreadUpdate{
		MessageThreadID: uuid.NewString(),
		UnreadCount:     &unreadCount,
	}

	errors := validator.ValidateUpdate(context.Background(), request)

	assert.Empty(t, errors)
}
