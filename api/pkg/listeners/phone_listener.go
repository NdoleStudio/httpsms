package listeners

import (
	"context"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/palantir/stacktrace"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
)

// PhoneListener handles cloud events that alter the state of entities.Phone
type PhoneListener struct {
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.PhoneService
}

// NewPhoneListener creates a new instance of PhoneListener
func NewPhoneListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.PhoneService,
) (l *PhoneListener, routes map[string]events.EventListener) {
	l = &PhoneListener{
		logger:  logger.WithService(fmt.Sprintf("%T", l)),
		tracer:  tracer,
		service: service,
	}

	return l, map[string]events.EventListener{
		events.EventTypeMessageSendScheduleDeleted: l.onMessageSendScheduleDeleted,
		events.UserAccountDeleted:                  l.onUserAccountDeleted,
	}
}

// onMessageSendScheduleDeleted handles the events.EventTypeMessageSendScheduleDeleted event
func (listener *PhoneListener) onMessageSendScheduleDeleted(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageSendScheduleDeletedPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err := listener.service.NullifyScheduleID(ctx, payload.UserID, payload.ScheduleID); err != nil {
		msg := fmt.Sprintf("cannot nullify schedule ID [%s] for user [%s] on [%s] event with ID [%s]", payload.ScheduleID, payload.UserID, event.Type(), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// onUserAccountDeleted handles the events.UserAccountDeleted event
func (listener *PhoneListener) onUserAccountDeleted(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.UserAccountDeletedPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err := listener.service.DeleteAllForUser(ctx, payload.UserID); err != nil {
		msg := fmt.Sprintf("cannot delete all [entities.Phone] for user [%s] on [%s] event with ID [%s]", payload.UserID, event.Type(), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
