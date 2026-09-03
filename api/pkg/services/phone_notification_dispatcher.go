package services

import (
	"context"
	"strings"

	"firebase.google.com/go/messaging"
	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/stacktrace"
	"github.com/google/uuid"
)

// PhoneNotificationDispatcher routes gateway notifications to the phone's configured transport.
type PhoneNotificationDispatcher struct {
	fcmSender  NotificationSender
	httpSender NotificationSender
}

// NewPhoneNotificationDispatcher creates a dispatcher for FCM and HTTP notification transports.
func NewPhoneNotificationDispatcher(
	fcmSender NotificationSender,
	httpSender NotificationSender,
) *PhoneNotificationDispatcher {
	return &PhoneNotificationDispatcher{
		fcmSender:  fcmSender,
		httpSender: httpSender,
	}
}

// Send delivers a notification using the phone's configured notification transport.
func (dispatcher *PhoneNotificationDispatcher) Send(
	ctx context.Context,
	phone *entities.Phone,
	message *messaging.Message,
	notificationID uuid.UUID,
) (string, error) {
	transport, err := phone.NotificationTransport()
	if err != nil {
		return "", stacktrace.Propagatef(err, "cannot determine notification transport for phone [%s]", phone.ID)
	}
	if message == nil {
		return "", stacktrace.Propagatef(
			stacktrace.NewErrorf("notification message is nil"),
			"cannot dispatch notification for phone [%s]",
			phone.ID,
		)
	}

	message.Token = strings.TrimSpace(*phone.FcmToken)
	switch transport {
	case entities.NotificationTransportFCM:
		return dispatcher.fcmSender.Send(ctx, message, notificationID)
	case entities.NotificationTransportHTTP:
		return dispatcher.httpSender.Send(ctx, message, notificationID)
	default:
		return "", stacktrace.NewErrorf("unsupported notification transport [%s]", transport)
	}
}
