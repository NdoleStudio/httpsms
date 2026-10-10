package requests

import (
	"testing"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/stretchr/testify/assert"
)

func TestMessageIncomingToSearchParamsForcesMobileOriginated(t *testing.T) {
	request := MessageIncoming{
		Owners:         []string{"+18005550199"},
		Statuses:       []string{"received"},
		SortBy:         "created_at",
		SortDescending: true,
		Limit:          "25",
	}

	params := request.Sanitize().ToSearchParams(entities.UserID("user-id"))

	assert.Equal(t, []entities.MessageType{entities.MessageTypeMobileOriginated}, params.Types)
	assert.Equal(t, []entities.MessageStatus{entities.MessageStatusReceived}, params.Statuses)
	assert.Equal(t, 25, params.Limit)
}

func TestMessageIncomingSanitizeSetsDefaults(t *testing.T) {
	request := MessageIncoming{}

	sanitized := request.Sanitize()

	assert.Equal(t, "0", sanitized.Skip)
	assert.Equal(t, "100", sanitized.Limit)
	assert.Equal(t, "created_at", sanitized.SortBy)
	assert.True(t, sanitized.SortDescending)
}

func TestMessageIncomingToSearchParamsSetsUserIDAndOwners(t *testing.T) {
	request := MessageIncoming{
		Owners: []string{"+18005550199"},
		Limit:  "25",
	}

	params := request.Sanitize().ToSearchParams(entities.UserID("user-id"))

	assert.Equal(t, entities.UserID("user-id"), params.UserID)
	assert.Equal(t, []string{"+18005550199"}, params.Owners)
}
