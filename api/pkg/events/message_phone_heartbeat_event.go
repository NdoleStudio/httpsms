package events

import (
	"time"
)

// EventTypeHeartbeatPhoneOutstanding is emitted when the phone requests for outstanding messages
const EventTypeHeartbeatPhoneOutstanding = "heartbeat.phone.outstanding"

// HeartbeatPhoneOutstandingPayload is the payload of the EventTypeHeartbeatPhoneOutstanding event
type HeartbeatPhoneOutstandingPayload struct {
	Owner     string    `json:"owner"`
	Timestamp time.Time `json:"timestamp"`
	Quantity  int       `json:"quantity"`
}
