package listeners

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/http-sms-manager/pkg/events"
	"github.com/NdoleStudio/http-sms-manager/pkg/services"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/palantir/stacktrace"
)

// NotificationListener handles cloud events which sends notifications
type NotificationListener struct {
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.NotificationService
}

// NewNotificationListener creates a new instance of NotificationListener
func NewNotificationListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.NotificationService,
) (l *NotificationListener, routes map[string]events.EventListener) {
	l = &NotificationListener{
		logger:  logger.WithService(fmt.Sprintf("%T", l)),
		tracer:  tracer,
		service: service,
	}

	return l, map[string]events.EventListener{
		events.EventTypeMessageAPISent: l.onMessageAPISent,
	}
}

// onMessageAPISent handles the events.EventTypeHeartbeatPhoneOutstanding event
func (listener *NotificationListener) onMessageAPISent(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageAPISentPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	sentParams := &services.NotificationMessageSentParams{
		UserID:    payload.UserID,
		Owner:     payload.Owner,
		MessageID: payload.ID,
	}

	if err := listener.service.MessageSent(ctx, sentParams); err != nil {
		msg := fmt.Sprintf("cannot send notification with params [%+#v] for event with ID [%s]", sentParams, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
