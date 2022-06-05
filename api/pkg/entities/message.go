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
	ID      uuid.UUID     `json:"id" gorm:"primaryKey;type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	From    string        `json:"from" gorm:"index:idx__messages__from__to" example:"+18005550199"`
	To      string        `json:"to" gorm:"index:idx__messages__from__to" example:"+18005550100"`
	Content string        `json:"content" example:"This is a sample text message"`
	Type    MessageType   `json:"type" example:"mobile-terminated"`
	Status  MessageStatus `json:"status" gorm:"index:idx__messages__status" example:"pending"`

	// SendDuration is the number of nanoseconds from when the request was received until when the mobile phone send the message
	SendDuration *int64 `json:"send_time" example:"133414"`

	RequestReceivedAt time.Time  `json:"request_received_at" example:"2022-06-05T14:26:01.520828+03:00"`
	CreatedAt         time.Time  `json:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt         time.Time  `json:"updated_at" example:"2022-06-05T14:26:10.303278+03:00"`
	OrderTimestamp    time.Time  `json:"order_timestamp" gorm:"index:idx__messages__order_timestamp" example:"2022-06-05T14:26:09.527976+03:00"`
	LastAttemptedAt   *time.Time `json:"last_attempted_at" example:"2022-06-05T14:26:09.527976+03:00"`
	SentAt            *time.Time `json:"sent_at" example:"2022-06-05T14:26:09.527976+03:00"`
	ReceivedAt        *time.Time `json:"received_at" example:"2022-06-05T14:26:09.527976+03:00"`
	FailureReason     *string    `json:"failure_reason"`
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
