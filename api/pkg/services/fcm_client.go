package services

import (
	"context"

	"firebase.google.com/go/messaging"
)

// FCMClient is the interface for sending Firebase Cloud Messaging notifications.
type FCMClient interface {
	// Send sends a message via FCM and returns the message name on success.
	Send(ctx context.Context, message *messaging.Message) (string, error)
}

// FirebaseFCMClient wraps the real Firebase messaging.Client.
type FirebaseFCMClient struct {
	client *messaging.Client
}

// NewFirebaseFCMClient creates a new FirebaseFCMClient.
func NewFirebaseFCMClient(client *messaging.Client) *FirebaseFCMClient {
	return &FirebaseFCMClient{client: client}
}

// Send sends a message via the real Firebase SDK.
func (c *FirebaseFCMClient) Send(ctx context.Context, message *messaging.Message) (string, error) {
	return c.client.Send(ctx, message)
}
