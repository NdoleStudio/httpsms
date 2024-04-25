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

// Integration3CXListener sends 3CX events to users
type Integration3CXListener struct {
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.Integration3CXService
}

// NewIntegration3CXListener creates a new instance of Integration3CXListener
func NewIntegration3CXListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.Integration3CXService,
) (l *Integration3CXListener, routes map[string]events.EventListener) {
	l = &Integration3CXListener{
		logger:  logger.WithService(fmt.Sprintf("%T", l)),
		tracer:  tracer,
		service: service,
	}

	return l, map[string]events.EventListener{
		// events.EventTypeMessagePhoneReceived:  l.OnMessagePhoneReceived,
		// events.EventTypeMessagePhoneDelivered: l.OnMessagePhoneDelivered,
		// events.EventTypeMessageSendFailed:     l.OnMessageSendFailed,
		// events.EventTypeMessagePhoneSent:      l.OnMessagePhoneSent,
	}
}

// OnMessagePhoneReceived handles the events.EventTypeMessagePhoneReceived event
func (listener *Integration3CXListener) OnMessagePhoneReceived(ctx context.Context, event cloudevents.Event) error {
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

// OnMessageSendFailed handles the events.EventTypeMessageSendFailed event
func (listener *Integration3CXListener) OnMessageSendFailed(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageSendFailedPayload
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

// OnMessagePhoneSent handles the events.EventTypeMessagePhoneSent event
func (listener *Integration3CXListener) OnMessagePhoneSent(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessagePhoneSentPayload
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

// OnMessagePhoneDelivered handles the events.EventTypeMessagePhoneDelivered event
func (listener *Integration3CXListener) OnMessagePhoneDelivered(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessagePhoneDeliveredPayload
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
