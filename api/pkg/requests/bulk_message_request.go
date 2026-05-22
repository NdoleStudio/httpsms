package requests

import (
	"strings"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/nyaruka/phonenumbers"
)

// BulkMessage represents a single message in a bulk SMS request
type BulkMessage struct {
	request
	FileType        string `json:"type"`
	FromPhoneNumber string `csv:"FromPhoneNumber"`
	ToPhoneNumber   string `csv:"ToPhoneNumber"`
	Content         string `csv:"Content"`
	SendTime        string `csv:"SendTime(optional)"`
	AttachmentURLs  string `csv:"AttachmentURLs(optional)" validate:"optional"` // Comma separated list of URLs
}

// GetSendTime parses the raw SendTime string into a *time.Time
func (input *BulkMessage) GetSendTime() *time.Time {
	raw := strings.TrimSpace(input.SendTime)
	if raw == "" {
		return nil
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, raw); err == nil {
			utc := t.UTC()
			return &utc
		}
	}
	return nil
}

// Sanitize sets defaults to BulkMessage
func (input *BulkMessage) Sanitize() *BulkMessage {
	input.ToPhoneNumber = input.sanitizeAddress(input.ToPhoneNumber)
	input.Content = strings.TrimSpace(input.Content)
	input.FromPhoneNumber = input.sanitizeAddress(input.FromPhoneNumber)

	var attachments []string
	for _, attachment := range strings.Split(input.AttachmentURLs, ",") {
		if strings.TrimSpace(attachment) != "" {
			attachments = append(attachments, strings.TrimSpace(attachment))
		}
	}
	input.AttachmentURLs = strings.Join(attachments, ",")
	return input
}

// ToMessageSendParams converts BulkMessage to services.MessageSendParams
func (input *BulkMessage) ToMessageSendParams(userID entities.UserID, requestID string, source string, index int) services.MessageSendParams {
	from, _ := phonenumbers.Parse(input.FromPhoneNumber, phonenumbers.UNKNOWN_REGION)

	return services.MessageSendParams{
		Source:            source,
		Owner:             from,
		RequestID:         input.sanitizeStringPointer(requestID),
		UserID:            userID,
		SendAt:            input.GetSendTime(),
		RequestReceivedAt: time.Now().UTC(),
		Contact:           input.sanitizeAddress(input.ToPhoneNumber),
		Content:           input.Content,
		Attachments:       input.removeEmptyStrings(strings.Split(input.AttachmentURLs, ",")),
		Index:             index,
	}
}
