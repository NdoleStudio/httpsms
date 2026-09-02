package entities

import (
	"net/url"
	"strings"
	"time"

	"github.com/NdoleStudio/stacktrace"
	"github.com/google/uuid"
)

// Phone represents an android phone which has installed the http sms app
type Phone struct {
	ID                    uuid.UUID  `json:"id" gorm:"primaryKey;type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	UserID                UserID     `json:"user_id" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	FcmToken              *string    `json:"fcm_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzd....." validate:"optional"`
	PhoneNumber           string     `json:"phone_number" example:"+18005550199"`
	MessagesPerMinute     uint       `json:"messages_per_minute" example:"1"`
	SIM                   SIM        `json:"sim" gorm:"default:SIM1"`
	MessageSendScheduleID *uuid.UUID `json:"message_send_schedule_id" gorm:"type:uuid" example:"32343a19-da5e-4b1b-a767-3298a73703cb" validate:"optional"`

	// MaxSendAttempts determines how many times to retry sending an SMS message
	MaxSendAttempts uint `json:"max_send_attempts" example:"2"`

	// MessageExpirationSeconds is the duration in seconds after sending a message when it is considered to be expired.
	MessageExpirationSeconds uint `json:"message_expiration_seconds"`

	MissedCallAutoReply *string `json:"missed_call_auto_reply" example:"This phone cannot receive calls. Please send an SMS instead." validate:"optional"`

	// UnarchiveThread moves an archived message thread back to the inbox when a new message is received on this phone.
	UnarchiveThread bool `json:"unarchive_thread" gorm:"default:false" example:"false"`

	CreatedAt time.Time `json:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt time.Time `json:"updated_at" example:"2022-06-05T14:26:10.303278+03:00"`
}

// NotificationTransport identifies how a phone receives wake-up notifications.
type NotificationTransport string

const (
	// NotificationTransportFCM sends notifications through Firebase.
	NotificationTransportFCM NotificationTransport = "fcm"
	// NotificationTransportHTTP sends notifications to a public HTTPS endpoint.
	NotificationTransportHTTP NotificationTransport = "http"
)

// MessageExpirationDuration returns the message expiration as time.Duration
func (phone *Phone) MessageExpirationDuration() time.Duration {
	return time.Duration(int(phone.MessageExpirationSecondsSanitized())) * time.Second
}

// MessageExpirationSecondsSanitized returns the message expiration seconds with default of 1 hour
func (phone *Phone) MessageExpirationSecondsSanitized() uint {
	if phone.MessageExpirationSeconds == 0 {
		return 10 * 60 // 10 minutes
	}
	return phone.MessageExpirationSeconds
}

// MaxSendAttemptsSanitized returns the max send attempts replacing 0 with 2
func (phone *Phone) MaxSendAttemptsSanitized() uint {
	if phone.MaxSendAttempts == 0 {
		return 2
	}
	return phone.MaxSendAttempts
}

// NotificationTransport returns the transport encoded by FcmToken.
func (phone *Phone) NotificationTransport() (NotificationTransport, error) {
	if phone == nil || phone.FcmToken == nil {
		return "", stacktrace.NewErrorf("phone has no notification token")
	}

	token := strings.TrimSpace(*phone.FcmToken)
	if token == "" {
		return "", stacktrace.NewErrorf("phone has no notification token")
	}

	if !strings.Contains(token, "://") {
		if strings.Contains(token, "/") {
			return "", stacktrace.NewErrorf("invalid notification token [%s]", token)
		}
		return NotificationTransportFCM, nil
	}

	endpoint, err := url.Parse(token)
	if err != nil {
		return "", stacktrace.Propagatef(err, "invalid notification URL [%s]", token)
	}

	if endpoint.Scheme != "https" {
		return "", stacktrace.NewErrorf("notification URL must use https")
	}
	if endpoint.Hostname() == "" {
		return "", stacktrace.NewErrorf("notification URL must include a hostname")
	}
	if endpoint.User != nil {
		return "", stacktrace.NewErrorf("notification URL must not contain user information")
	}

	return NotificationTransportHTTP, nil
}

// NotificationURL returns the parsed endpoint for an HTTP notification token.
func (phone *Phone) NotificationURL() (*url.URL, error) {
	transport, err := phone.NotificationTransport()
	if err != nil {
		return nil, err
	}

	if transport != NotificationTransportHTTP {
		return nil, stacktrace.NewErrorf("phone notification transport is [%s], not HTTP", transport)
	}

	endpoint, err := url.Parse(strings.TrimSpace(*phone.FcmToken))
	if err != nil {
		return nil, stacktrace.Propagatef(err, "cannot parse notification URL")
	}

	return endpoint, nil
}
