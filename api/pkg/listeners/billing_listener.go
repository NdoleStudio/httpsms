package listeners

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/palantir/stacktrace"
)

// BillingListener handles cloud events which affect billing
type BillingListener struct {
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.BillingService
}

// NewBillingListener creates a new instance of UserListener
func NewBillingListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.BillingService,
) (l *BillingListener, routes map[string]events.EventListener) {
	l = &BillingListener{
		logger:  logger.WithService(fmt.Sprintf("%T", l)),
		tracer:  tracer,
		service: service,
	}

	return l, map[string]events.EventListener{
		events.EventTypeMessageAPISent:       l.OnMessageAPISent,
		events.EventTypeMessagePhoneReceived: l.OnMessagePhoneReceived,
	}
}

// OnMessageAPISent handles the events.EventTypeMessageAPISent event
func (listener *BillingListener) OnMessageAPISent(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageAPISentPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err := listener.service.RegisterSentMessage(ctx, payload.MessageID, payload.RequestReceivedAt, payload.UserID); err != nil {
		msg := fmt.Sprintf("cannot register sent message for event [%s] for event with ID [%s]", spew.Sdump(payload), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// OnMessagePhoneReceived handles the events.EventTypeMessagePhoneReceived event
func (listener *BillingListener) OnMessagePhoneReceived(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessagePhoneReceivedPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err := listener.service.RegisterReceivedMessage(ctx, payload.MessageID, payload.Timestamp, payload.UserID); err != nil {
		msg := fmt.Sprintf("cannot register received message for event [%s] for event with ID [%s]", spew.Sdump(payload), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
