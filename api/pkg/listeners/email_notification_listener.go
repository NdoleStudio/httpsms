package listeners

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/palantir/stacktrace"
)

// EmailNotificationListener listens for events about failed and expired messages
type EmailNotificationListener struct {
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.EmailNotificationService
}

// NewEmailNotificationListener creates a new instance of emailNotificationListener
func NewEmailNotificationListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.EmailNotificationService,
) (l *EmailNotificationListener, routes map[string]events.EventListener) {
	l = &EmailNotificationListener{
		logger:  logger.WithService(fmt.Sprintf("%T", l)),
		tracer:  tracer,
		service: service,
	}

	return l, map[string]events.EventListener{
		events.EventTypeMessageSendExpired: l.OnMessageSendExpired,
		events.EventTypeMessageSendFailed:  l.OnMessageSendFailed,
	}
}

// OnMessageSendExpired handles the events.EventTypeMessageSendExpired event
func (listener *EmailNotificationListener) OnMessageSendExpired(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	payload := new(events.MessageSendExpiredPayload)
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err := listener.service.NotifyMessageExpired(ctx, payload); err != nil {
		msg := fmt.Sprintf("cannot process [%s] event with ID [%s]", event.Type(), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// OnMessageSendFailed handles the events.EventTypeMessageSendFailed event
func (listener *EmailNotificationListener) OnMessageSendFailed(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	payload := new(events.MessageSendFailedPayload)
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err := listener.service.NotifyMessageFailed(ctx, payload); err != nil {
		msg := fmt.Sprintf("cannot process [%s] event with ID [%s]", event.Type(), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
