package requests

import (
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/google/uuid"
)

// MessageOutstanding is the payload fetching outstanding entities.Message
type MessageOutstanding struct {
	request
	MessageID string `json:"message_id" query:"message_id"`
}

// Sanitize sets defaults to MessageOutstanding
func (input *MessageOutstanding) Sanitize() MessageOutstanding {
	input.MessageID = strings.TrimSpace(input.MessageID)
	return *input
}

// ToGetOutstandingParams converts MessageOutstanding into services.MessageGetOutstandingParams
func (input *MessageOutstanding) ToGetOutstandingParams(source string, userID entities.UserID, timestamp time.Time) services.MessageGetOutstandingParams {
	return services.MessageGetOutstandingParams{
		Source:    source,
		UserID:    userID,
		MessageID: uuid.MustParse(input.MessageID),
		Timestamp: timestamp,
	}
}
