package validators

import (
	"context"
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/requests"
	"github.com/stretchr/testify/assert"
)

func TestValidateMessageIncomingDoesNotRequireTurnstileToken(t *testing.T) {
	validator := &MessageHandlerValidator{}
	request := requests.MessageIncoming{
		Owners: []string{"+18005550199"},
		Limit:  "25",
		Skip:   "0",
	}

	errors := validator.ValidateMessageIncoming(context.Background(), request.Sanitize())

	assert.Empty(t, errors)
}

func TestValidateMessageIncomingRejectsInvalidOwner(t *testing.T) {
	validator := &MessageHandlerValidator{}
	request := requests.MessageIncoming{
		Owners: []string{"not-a-phone-number"},
		Limit:  "25",
		Skip:   "0",
	}

	errors := validator.ValidateMessageIncoming(context.Background(), request.Sanitize())

	assert.NotEmpty(t, errors.Get("owners"))
}

func TestValidateMessageIncomingRejectsStatusOtherThanReceived(t *testing.T) {
	validator := &MessageHandlerValidator{}
	request := requests.MessageIncoming{
		Owners:   []string{"+18005550199"},
		Statuses: []string{"pending"},
		Limit:    "25",
		Skip:     "0",
	}

	errors := validator.ValidateMessageIncoming(context.Background(), request.Sanitize())

	assert.NotEmpty(t, errors.Get("statuses"))
}
