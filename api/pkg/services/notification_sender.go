package services

import (
	"context"
	"strings"
	"time"

	"firebase.google.com/go/messaging"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/stacktrace"
	"github.com/google/uuid"
)

// GatewayNotification is a transport-neutral notification for a phone gateway.
type GatewayNotification struct {
	Data           map[string]string
	Priority       string
	TTL            *time.Duration
	NotificationID uuid.UUID
}

// NotificationSender delivers a notification to a transport-specific destination.
type NotificationSender interface {
	Send(ctx context.Context, destination string, notification GatewayNotification) (string, error)
}

// NotificationDispatcher routes gateway notifications to the phone's configured transport.
type NotificationDispatcher struct {
	fcmSender  NotificationSender
	httpSender NotificationSender
}

// NewNotificationDispatcher creates a dispatcher for FCM and HTTP notification transports.
func NewNotificationDispatcher(fcmSender NotificationSender, httpSender NotificationSender) *NotificationDispatcher {
	return &NotificationDispatcher{
		fcmSender:  fcmSender,
		httpSender: httpSender,
	}
}

// Send delivers a notification using the phone's configured notification transport.
func (dispatcher *NotificationDispatcher) Send(
	ctx context.Context,
	phone *entities.Phone,
	notification GatewayNotification,
) (string, error) {
	transport, err := phone.NotificationTransport()
	if err != nil {
		return "", stacktrace.Propagatef(err, "cannot determine notification transport for phone [%s]", phone.ID)
	}

	destination := strings.TrimSpace(*phone.FcmToken)
	switch transport {
	case entities.NotificationTransportFCM:
		return dispatcher.fcmSender.Send(ctx, destination, notification)
	case entities.NotificationTransportHTTP:
		return dispatcher.httpSender.Send(ctx, destination, notification)
	default:
		return "", stacktrace.NewErrorf("unsupported notification transport [%s]", transport)
	}
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
	destination string,
	notification GatewayNotification,
) (string, error) {
	message := &messaging.Message{
		Token: destination,
		Data:  notification.Data,
		Android: &messaging.AndroidConfig{
			Priority: notification.Priority,
			TTL:      notification.TTL,
		},
	}

	result, err := sender.client.Send(ctx, message)
	if err != nil {
		return "", stacktrace.Propagatef(err, "cannot send Firebase notification")
	}
	return result, nil
}
