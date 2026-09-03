package services

import (
	"context"

	"firebase.google.com/go/messaging"
	"github.com/NdoleStudio/stacktrace"
	"github.com/google/uuid"
)

// NotificationSender delivers a notification to a transport-specific destination.
type NotificationSender interface {
	Send(ctx context.Context, message *messaging.Message, notificationID uuid.UUID) (string, error)
}

// FCMNotificationSender delivers gateway notifications through Firebase Cloud Messaging.
type FCMNotificationSender struct {
	client FCMClient
}

// NewFCMNotificationSender creates a Firebase notification sender.
func NewFCMNotificationSender(client FCMClient) *FCMNotificationSender {
	return &FCMNotificationSender{client: client}
}

// Send delivers a gateway notification through Firebase Cloud Messaging.
func (sender *FCMNotificationSender) Send(
	ctx context.Context,
	message *messaging.Message,
	_ uuid.UUID,
) (string, error) {
	if message == nil {
		return "", stacktrace.Propagatef(
			stacktrace.NewErrorf("notification message is nil"),
			"cannot send Firebase notification",
		)
	}

	result, err := sender.client.Send(ctx, message)
	if err != nil {
		return "", stacktrace.Propagatef(err, "cannot send Firebase notification")
	}
	return result, nil
}
