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
	FromPhoneNumber string     `csv:"FromPhoneNumber"`
	ToPhoneNumber   string     `csv:"ToPhoneNumber"`
	Content         string     `csv:"Content"`
	SendTime        *time.Time `csv:"SendTime(optional)"`
	AttachmentURLs  string     `csv:"AttachmentURLs(optional)" validate:"optional"` // Comma separated list of URLs
}

// Sanitize sets defaults to BulkMessage
func (input *BulkMessage) Sanitize() *BulkMessage {
	input.ToPhoneNumber = input.sanitizeAddress(input.ToPhoneNumber)
	input.Content = strings.TrimSpace(input.Content)
	input.FromPhoneNumber = input.sanitizeAddress(input.FromPhoneNumber)
	input.AttachmentURLs = strings.TrimSpace(input.AttachmentURLs)
	return input
}

// ToMessageSendParams converts BulkMessage to services.MessageSendParams
func (input *BulkMessage) ToMessageSendParams(userID entities.UserID, requestID uuid.UUID, source string) services.MessageSendParams {
	from, _ := phonenumbers.Parse(input.FromPhoneNumber, phonenumbers.UNKNOWN_REGION)

	var attachments []entities.MessageAttachment
	if input.AttachmentURLs != "" {
		urls := strings.Split(input.AttachmentURLs, ",")
		for _, u := range urls {
			cleanURL := strings.TrimSpace(u)
			if cleanURL == "" {
				continue
			}

			// Since there's no easy way to set a type in the CSV, defaulting to octet-stream and then just checking the file extension in the URL
			contentType := "application/octet-stream"
			lowerURL := strings.ToLower(cleanURL)
			if strings.HasSuffix(lowerURL, ".jpg") || strings.HasSuffix(lowerURL, ".jpeg") {
				contentType = "image/jpeg"
			} else if strings.HasSuffix(lowerURL, ".png") {
				contentType = "image/png"
			} else if strings.HasSuffix(lowerURL, ".gif") {
				contentType = "image/gif"
			} else if strings.HasSuffix(lowerURL, ".mp4") {
				contentType = "video/mp4"
			}

			attachments = append(attachments, entities.MessageAttachment{
				ContentType: contentType,
				URL:         cleanURL,
			})
		}
	}

	return services.MessageSendParams{
		Source:            source,
		Owner:             from,
		RequestID:         input.sanitizeStringPointer(fmt.Sprintf("bulk-%s", requestID.String())),
		UserID:            userID,
		SendAt:            input.SendTime,
		RequestReceivedAt: time.Now().UTC(),
		Contact:           input.sanitizeAddress(input.ToPhoneNumber),
		Content:           input.Content,
		Attachments:       attachments,
	}
}
