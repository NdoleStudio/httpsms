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

// UserListener handles cloud events which sends notifications
type UserListener struct {
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.UserService
}

// NewUserListener creates a new instance of UserListener
func NewUserListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.UserService,
) (l *UserListener, routes map[string]events.EventListener) {
	l = &UserListener{
		logger:  logger.WithService(fmt.Sprintf("%T", l)),
		tracer:  tracer,
		service: service,
	}

	return l, map[string]events.EventListener{
		events.EventTypePhoneHeartbeatDead: l.onPhoneHeartbeatDead,
	}
}

// onPhoneHeartbeatDead handles the events.EventTypePhoneHeartbeatDead event
func (listener *UserListener) onPhoneHeartbeatDead(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.PhoneHeartbeatDeadPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	sendParams := &services.UserSendPhoneDeadEmailParams{
		UserID:                 payload.UserID,
		PhoneID:                payload.PhoneID,
		Owner:                  payload.Owner,
		LastHeartbeatTimestamp: payload.LastHeartbeatTimestamp,
	}

	if err := listener.service.SendPhoneDeadEmail(ctx, sendParams); err != nil {
		msg := fmt.Sprintf("cannot send notification with params [%s] for event with ID [%s]", spew.Sdump(sendParams), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
