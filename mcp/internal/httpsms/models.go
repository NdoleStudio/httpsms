// Package httpsms is a typed client for the httpSMS HTTP API
// (api.httpsms.com), used by every MCP tool that needs to call it.
//
// The client never mints, caches, or inspects the delegated API bearer
// token it is given: callers (the MCP tool handlers) mint a short-lived,
// scope- and operation-bound token per call with auth.KeySet and pass it in
// as a plain string. This package is deliberately isolated from the rest of
// the MCP server (auth, oauth, config) so it can be developed, tested, and
// reused independently of OAuth, token minting, and tool registration.
package httpsms

import (
	"fmt"
	"time"
)

// Response is the standard httpSMS API success envelope every 2xx response
// is wrapped in.
type Response[T any] struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// APIError is a non-2xx httpSMS API response. It is always safe to log or
// include in a tool error: it never carries the request body, the bearer
// token, or SMS content, only the response status code, the API's own
// message, any field validation errors, and the request ID this client
// generated for the call.
type APIError struct {
	// StatusCode is the HTTP status code the API responded with.
	StatusCode int

	// Message is the API's own top-level "message" field.
	Message string

	// Fields are per-field validation errors from a 422 response, if any.
	Fields map[string][]string

	// RequestID is the value this client sent as the request's X-Request-Id
	// header. The httpSMS API does not currently echo it back, but it is
	// still useful for correlating a returned error with the client-side
	// log line and trace span that issued the request.
	RequestID string
}

// Error implements the error interface. It never includes the request body
// or bearer token.
func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("httpsms: request [%s] failed with status %d: %s", e.RequestID, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("httpsms: request failed with status %d: %s", e.StatusCode, e.Message)
}

// Phone is one of the user's registered httpSMS sending phones.
type Phone struct {
	ID                string    `json:"id"`
	PhoneNumber       string    `json:"phone_number"`
	SIM               string    `json:"sim"`
	MessagesPerMinute uint      `json:"messages_per_minute"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Message is a single SMS/MMS message sent or received through httpSMS,
// mobile-originated (incoming) or mobile-terminated (outgoing).
type Message struct {
	ID             string     `json:"id"`
	RequestID      *string    `json:"request_id"`
	Owner          string     `json:"owner"`
	Contact        string     `json:"contact"`
	Content        string     `json:"content"`
	Attachments    []string   `json:"attachments"`
	Encrypted      bool       `json:"encrypted"`
	Type           string     `json:"type"`
	Status         string     `json:"status"`
	SIM            string     `json:"sim"`
	OrderTimestamp time.Time  `json:"order_timestamp"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	SentAt         *time.Time `json:"sent_at"`
	DeliveredAt    *time.Time `json:"delivered_at"`
	ReceivedAt     *time.Time `json:"received_at"`
	FailedAt       *time.Time `json:"failed_at"`
	FailureReason  *string    `json:"failure_reason"`
}

// MessageThread is a conversation between one of the user's phones (Owner)
// and a Contact.
type MessageThread struct {
	ID                 string    `json:"id"`
	Owner              string    `json:"owner"`
	Contact            string    `json:"contact"`
	IsArchived         bool      `json:"is_archived"`
	UnreadCount        uint      `json:"unread_count"`
	Status             string    `json:"status"`
	LastMessageContent *string   `json:"last_message_content"`
	LastMessageID      *string   `json:"last_message_id"`
	OrderTimestamp     time.Time `json:"order_timestamp"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// PhoneAPIKey authenticates the httpSMS Android app for a subset of the
// user's phones. APIKey is a secret, one-time display value: callers must
// never log, trace, or persist it beyond returning it to the user once.
type PhoneAPIKey struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	PhoneNumbers []string  `json:"phone_numbers"`
	PhoneIDs     []string  `json:"phone_ids"`
	APIKey       string    `json:"api_key"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// User is the authenticated httpSMS user. APIKey is a secret, one-time
// display value after rotation: callers must never log, trace, or persist
// it beyond returning it to the user once.
type User struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	APIKey string `json:"api_key"`
}

// ListPhonesParams are the supported filters for GET /v1/phones. Skip and
// Limit of zero are omitted from the request so the API applies its own
// default.
type ListPhonesParams struct {
	Skip  int
	Query string
	Limit int
}

// SendSMSParams is the payload for POST /v1/messages/send.
type SendSMSParams struct {
	From        string
	To          string
	Content     string
	Attachments []string
	Encrypted   bool
	RequestID   string
	SendAt      *time.Time
}

// ListMessageThreadsParams are the supported filters for
// GET /v1/message-threads. IsArchived is a pointer so "not set" (let the API
// default to false) is distinguishable from an explicit false. Skip and
// Limit of zero are omitted so the API applies its own default.
type ListMessageThreadsParams struct {
	Owner        string
	IsArchived   *bool
	WithContacts bool
	Query        string
	Skip         int
	Limit        int
}

// ListThreadMessagesParams are the supported filters for GET /v1/messages.
// Owner and Contact are required by the API.
type ListThreadMessagesParams struct {
	Owner   string
	Contact string
	Query   string
	Skip    int
	Limit   int
}

// ListIncomingMessagesParams are the supported filters for
// GET /v1/messages/incoming. SortDescending is a pointer so "not set" (let
// the API pick its own default sort order) is distinguishable from an
// explicit false.
type ListIncomingMessagesParams struct {
	Owners         []string
	Statuses       []string
	Query          string
	SortBy         string
	SortDescending *bool
	Skip           int
	Limit          int
}

// CreatePhoneAPIKeyParams is the payload for POST /v1/phone-api-keys.
type CreatePhoneAPIKeyParams struct {
	Name string
}
