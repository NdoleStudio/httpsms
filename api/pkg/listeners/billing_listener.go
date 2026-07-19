package listeners

import (
	"context"
	"fmt"

	"github.com/davecgh/go-spew/spew"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	cloudevents "github.com/cloudevents/sdk-go/v2"
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
		events.UserAccountDeleted:            l.onUserAccountDeleted,
		events.EventTypeMessagePhoneReceived: l.OnMessagePhoneReceived,
	}
}

// OnMessageAPISent handles the events.EventTypeMessageAPISent event
func (listener *BillingListener) OnMessageAPISent(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageAPISentPayload
	if err := event.DataAs(&payload); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot decode [%s] into [%T]", event.Data(), payload))
	}

	if err := listener.service.RegisterSentMessage(ctx, payload.MessageID, payload.RequestReceivedAt, payload.UserID); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot register sent message for event [%s] for event with ID [%s]", spew.Sdump(payload), event.ID()))
	}

	return nil
}

// OnMessagePhoneReceived handles the events.EventTypeMessagePhoneReceived event
func (listener *BillingListener) OnMessagePhoneReceived(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessagePhoneReceivedPayload
	if err := event.DataAs(&payload); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot decode [%s] into [%T]", event.Data(), payload))
	}

	if err := listener.service.RegisterReceivedMessage(ctx, payload.MessageID, payload.Timestamp, payload.UserID); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot register received message for event [%s] for event with ID [%s]", spew.Sdump(payload), event.ID()))
	}

	return nil
}

func (listener *BillingListener) onUserAccountDeleted(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.UserAccountDeletedPayload
	if err := event.DataAs(&payload); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot decode [%s] into [%T]", event.Data(), payload))
	}

	if err := listener.service.DeleteAllForUser(ctx, payload.UserID); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagatef(err, "cannot delete [entities.BillingUsage] for user [%s] on [%s] event with ID [%s]", payload.UserID, event.Type(), event.ID()))
	}

	return nil
}
