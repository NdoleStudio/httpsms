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

// MessageSendScheduleListener handles cloud events related to message send schedules.
type MessageSendScheduleListener struct {
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.MessageSendScheduleService
}

// NewMessageSendScheduleListener creates a new instance of MessageSendScheduleListener.
func NewMessageSendScheduleListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.MessageSendScheduleService,
) (l *MessageSendScheduleListener, routes map[string]events.EventListener) {
	l = &MessageSendScheduleListener{
		logger:  logger.WithService(fmt.Sprintf("%T", &MessageSendScheduleListener{})),
		tracer:  tracer,
		service: service,
	}

	return l, map[string]events.EventListener{
		events.UserAccountDeleted: l.onUserAccountDeleted,
	}
}

// onUserAccountDeleted removes all message send schedules for a deleted user account.
func (listener *MessageSendScheduleListener) onUserAccountDeleted(
	ctx context.Context,
	event cloudevents.Event,
) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.UserAccountDeletedPayload
	if err := event.DataAs(&payload); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)))
	}

	if err := listener.service.DeleteAllForUser(ctx, payload.UserID); err != nil {
		msg := fmt.Sprintf("cannot delete [entities.MessageSendSchedule] for user [%s] on [%s] event with ID [%s]", payload.UserID, event.Type(), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
