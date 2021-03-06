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

	// MessageStatusDelivered means the mobile phone has delivered the message
	MessageStatusDelivered = "delivered"
)

// MessageEventName is the type of event generated by the mobile phone for a message
type MessageEventName string

const (
	// MessageEventNameSent is emitted when a message is sent by the mobile phone
	MessageEventNameSent = "SENT"

	// MessageEventNameDelivered is emitted when a message is delivered by the mobile phone
	MessageEventNameDelivered = "DELIVERED"

	// MessageEventNameFailed is emitted when a message is failed by the mobile phone
	MessageEventNameFailed = "FAILED"
)

// Message represents a message sent between 2 phone numbers
type Message struct {
	ID      uuid.UUID     `json:"id" gorm:"primaryKey;type:uuid;" example:"32343a19-da5e-4b1b-a767-3298a73703cb"`
	Owner   string        `json:"owner" gorm:"index:idx_messages_user_id__owner__contact" example:"+18005550199"`
	UserID  UserID        `json:"user_id" gorm:"index:idx_messages_user_id__owner__contact" example:"WB7DRDWrJZRGbYrv2CKGkqbzvqdC"`
	Contact string        `json:"contact" gorm:"index:idx_messages_user_id__owner__contact" example:"+18005550100"`
	Content string        `json:"content" example:"This is a sample text message"`
	Type    MessageType   `json:"type" example:"mobile-terminated"`
	Status  MessageStatus `json:"status" gorm:"index:idx_messages_status" example:"pending"`

	// SendDuration is the number of nanoseconds from when the request was received until when the mobile phone send the message
	SendDuration *int64 `json:"send_time" example:"133414"`

	RequestReceivedAt time.Time  `json:"request_received_at" example:"2022-06-05T14:26:01.520828+03:00"`
	CreatedAt         time.Time  `json:"created_at" example:"2022-06-05T14:26:02.302718+03:00"`
	UpdatedAt         time.Time  `json:"updated_at" example:"2022-06-05T14:26:10.303278+03:00"`
	OrderTimestamp    time.Time  `json:"order_timestamp" gorm:"index:idx_messages_order_timestamp" example:"2022-06-05T14:26:09.527976+03:00"`
	LastAttemptedAt   *time.Time `json:"last_attempted_at" example:"2022-06-05T14:26:09.527976+03:00"`
	SentAt            *time.Time `json:"sent_at" example:"2022-06-05T14:26:09.527976+03:00"`
	ReceivedAt        *time.Time `json:"received_at" example:"2022-06-05T14:26:09.527976+03:00"`
	FailureReason     *string    `json:"failure_reason"`
}

// IsSending determines if a message is being sent
func (message Message) IsSending() bool {
	return message.Status == MessageStatusSending
}

// IsSent determines if a message has been sent
func (message Message) IsSent() bool {
	return message.Status == MessageStatusSent
}

// Sent registers a message as sent
func (message *Message) Sent(timestamp time.Time) *Message {
	sendDuration := timestamp.UnixNano() - message.RequestReceivedAt.UnixNano()
	message.SentAt = &timestamp
	message.Status = MessageStatusSent
	message.updateOrderTimestamp(timestamp)
	message.SendDuration = &sendDuration
	return message
}

// Failed registers a message as failed
func (message *Message) Failed(timestamp time.Time) *Message {
	message.SentAt = &timestamp
	message.Status = MessageStatusFailed
	message.updateOrderTimestamp(timestamp)
	return message
}

// Delivered registers a message as delivered
func (message *Message) Delivered(timestamp time.Time) *Message {
	message.SentAt = &timestamp
	message.Status = MessageStatusDelivered
	message.updateOrderTimestamp(timestamp)
	return message
}

// AddSendAttempt configures a Message for sending
func (message *Message) AddSendAttempt(timestamp time.Time) *Message {
	message.Status = MessageStatusSending
	message.LastAttemptedAt = &timestamp
	message.updateOrderTimestamp(timestamp)
	return message
}

func (message *Message) updateOrderTimestamp(timestamp time.Time) {
	if timestamp.UnixNano() > message.OrderTimestamp.UnixNano() {
		message.OrderTimestamp = timestamp
	}
}
