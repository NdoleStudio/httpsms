package entities

import (
	"time"

	"github.com/google/uuid"
)

// MessageType is the type of message if it is incoming or outgoing
type MessageType string

const (
	// MessageTypeMobileTerminated means the message it sent to a mobile phone
	MessageTypeMobileTerminated = "mobile-terminated"

	// MessageTypeMobileOriginated means the message comes directly from a mobile phone
	MessageTypeMobileOriginated = "mobile-originated"
)

// MessageStatus is the status of the message
type MessageStatus string

const (
	// MessageStatusPending means the message has been queued to be sent
	MessageStatusPending = "pending"

	// MessageStatusSending means a phone has picked up the message and is currently sending it
	MessageStatusSending = "sending"

	// MessageStatusSent means the message has already sent by the mobile phone
	MessageStatusSent = "sent"

	// MessageStatusReceived means the message was received by tne mobile phone (MO)
	MessageStatusReceived = "received"

	// MessageStatusFailed means the mobile phone could not send the message
	MessageStatusFailed = "failed"
)

// Message represents a message sent between 2 phone numbers
type Message struct {
	ID      uuid.UUID     `json:"id" gorm:"primaryKey;type:uuid;"`
	From    string        `json:"from"`
	To      string        `json:"to"`
	Content string        `json:"content"`
	Type    MessageType   `json:"type"`
	Status  MessageStatus `json:"status"`

	// SendDuration is the number of nanoseconds from when the request was received until when the mobile phone send the message
	SendDuration *int64 `json:"send_time"`

	RequestReceivedAt time.Time  `json:"request_received_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	OrderTimestamp    time.Time  `json:"order_timestamp"`
	LastAttemptedAt   *time.Time `json:"last_attempted_at"`
	SentAt            *time.Time `json:"sent_at"`
	ReceivedAt        *time.Time `json:"received_at"`

	FailureReason *string `json:"failure_reason"`
}

// IsSending determines if a message is being sent
func (message Message) IsSending() bool {
	return message.Status == MessageStatusSending
}

// AddSendAttempt configures a Message for sending
func (message *Message) AddSendAttempt(timestamp time.Time) *Message {
	message.Status = MessageStatusSending
	message.LastAttemptedAt = &timestamp
	message.OrderTimestamp = timestamp
	return message
}
