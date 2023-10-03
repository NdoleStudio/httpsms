package requests

import (
	"fmt"
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/google/uuid"
	"github.com/nyaruka/phonenumbers"
)

// BulkMessage represents a single message in a bulk SMS request
type BulkMessage struct {
	request
	FromPhoneNumber string `csv:"FromPhoneNumber"`
	ToPhoneNumber   string `csv:"ToPhoneNumber"`
	Content         string `csv:"Content"`
}

// Sanitize sets defaults to BulkMessage
func (input *BulkMessage) Sanitize() *BulkMessage {
	input.ToPhoneNumber = input.sanitizeAddress(input.ToPhoneNumber)
	input.Content = strings.TrimSpace(input.Content)
	input.FromPhoneNumber = input.sanitizeAddress(input.FromPhoneNumber)
	return input
}

// ToMessageSendParams converts BulkMessage to services.MessageSendParams
func (input *BulkMessage) ToMessageSendParams(userID entities.UserID, requestID uuid.UUID, source string) services.MessageSendParams {
	from, _ := phonenumbers.Parse(input.FromPhoneNumber, phonenumbers.UNKNOWN_REGION)
	return services.MessageSendParams{
		Source:            source,
		Owner:             from,
		RequestID:         input.sanitizeStringPointer(fmt.Sprintf("bulk-%s", requestID.String())),
		UserID:            userID,
		RequestReceivedAt: time.Now().UTC(),
		Contact:           input.sanitizeAddress(input.ToPhoneNumber),
		Content:           input.Content,
	}
}
