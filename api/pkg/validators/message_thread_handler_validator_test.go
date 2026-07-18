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

func TestValidateUpdateAcceptsReadOnlyUpdate(t *testing.T) {
	validator := &MessageThreadHandlerValidator{}
	isRead := true
	request := requests.MessageThreadUpdate{
		MessageThreadID: uuid.NewString(),
		IsRead:          &isRead,
	}

	errors := validator.ValidateUpdate(context.Background(), request)

	assert.Empty(t, errors)
}
