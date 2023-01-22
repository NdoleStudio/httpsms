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

// WebhookListener sends webhook events to users
type WebhookListener struct {
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.WebhookService
}

// NewWebhookListener creates a new instance of WebhookListener
func NewWebhookListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.WebhookService,
) (l *WebhookListener, routes map[string]events.EventListener) {
	l = &WebhookListener{
		logger:  logger.WithService(fmt.Sprintf("%T", l)),
		tracer:  tracer,
		service: service,
	}

	return l, map[string]events.EventListener{
		events.EventTypeMessagePhoneReceived: l.OnMessagePhoneReceived,
	}
}

// OnMessagePhoneReceived handles the events.EventTypeMessagePhoneReceived event
func (listener *WebhookListener) OnMessagePhoneReceived(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessagePhoneReceivedPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err := listener.service.Send(ctx, payload.UserID, event); err != nil {
		msg := fmt.Sprintf("cannot process [%s] event with ID [%s]", event.Type(), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
