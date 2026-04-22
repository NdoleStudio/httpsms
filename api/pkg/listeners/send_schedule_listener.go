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

// SendScheduleListener handles cloud events related to message send schedules.
type SendScheduleListener struct {
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.SendScheduleService
}

// NewSendScheduleListener creates a new instance of SendScheduleListener.
func NewSendScheduleListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.SendScheduleService,
) (l *SendScheduleListener, routes map[string]events.EventListener) {
	l = &SendScheduleListener{
		logger:  logger.WithService(fmt.Sprintf("%T", &SendScheduleListener{})),
		tracer:  tracer,
		service: service,
	}

	return l, map[string]events.EventListener{
		events.UserAccountDeleted: l.onUserAccountDeleted,
	}
}

// onUserAccountDeleted removes all message send schedules for a deleted user account.
func (listener *SendScheduleListener) onUserAccountDeleted(
	ctx context.Context,
	event cloudevents.Event,
) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.UserAccountDeletedPayload
	if err := event.DataAs(&payload); err != nil {
		return listener.tracer.WrapErrorSpan(
			span,
			stacktrace.Propagate(
				err,
				"cannot decode [%s] into [%T]",
				event.Data(),
				payload,
			),
		)
	}

	if err := listener.service.DeleteAllForUser(ctx, payload.UserID); err != nil {
		return listener.tracer.WrapErrorSpan(
			span,
			stacktrace.Propagate(
				err,
				"cannot delete [entities.MessageSendSchedule] for user [%s] on [%s] event with ID [%s]",
				payload.UserID,
				event.Type(),
				event.ID(),
			),
		)
	}

	return nil
}
