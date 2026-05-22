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

// GetSendTime parses the raw SendTime string into a *time.Time.
// For timezone-naive formats, the time is interpreted in the given location.
// For RFC3339 (which includes an offset), the embedded offset is used.
func (input *BulkMessage) GetSendTime(location *time.Location) *time.Time {
	raw := strings.TrimSpace(input.SendTime)
	if raw == "" {
		return nil
	}

	if location == nil {
		location = time.UTC
	}

	// RFC3339 already contains timezone offset, parse without location
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		utc := t.UTC()
		return &utc
	}

	// Naive formats: interpret in the user's location
	naiveFormats := []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	for _, format := range naiveFormats {
		if t, err := time.ParseInLocation(format, raw, location); err == nil {
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
func (input *BulkMessage) ToMessageSendParams(userID entities.UserID, requestID string, source string, index int, location *time.Location) services.MessageSendParams {
	from, _ := phonenumbers.Parse(input.FromPhoneNumber, phonenumbers.UNKNOWN_REGION)

	return services.MessageSendParams{
		Source:            source,
		Owner:             from,
		RequestID:         input.sanitizeStringPointer(requestID),
		UserID:            userID,
		SendAt:            input.GetSendTime(location),
		RequestReceivedAt: time.Now().UTC(),
		Contact:           input.sanitizeAddress(input.ToPhoneNumber),
		Content:           input.Content,
		Attachments:       input.removeEmptyStrings(strings.Split(input.AttachmentURLs, ",")),
		Index:             index,
	}
}
