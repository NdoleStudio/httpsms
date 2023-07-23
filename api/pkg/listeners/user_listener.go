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
		events.UserSubscriptionCreated:     l.OnUserSubscriptionCreated,
		events.UserSubscriptionCancelled:   l.OnUserSubscriptionCancelled,
		events.UserSubscriptionExpired:     l.OnUserSubscriptionExpired,
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

// OnUserSubscriptionCreated handles the events.UserSubscriptionCreated event
func (listener *UserListener) OnUserSubscriptionCreated(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.UserSubscriptionCreatedPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err := listener.service.StartSubscription(ctx, &payload); err != nil {
		msg := fmt.Sprintf("cannot start subscription for user with ID [%s] for event with ID [%s]", payload.UserID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// OnUserSubscriptionCancelled handles the events.UserSubscriptionCancelled event
func (listener *UserListener) OnUserSubscriptionCancelled(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.UserSubscriptionCancelledPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err := listener.service.CancelSubscription(ctx, &payload); err != nil {
		msg := fmt.Sprintf("cannot cancell subscription for user with ID [%s] for event with ID [%s]", payload.UserID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// OnUserSubscriptionExpired handles the events.UserSubscriptionExpired event
func (listener *UserListener) OnUserSubscriptionExpired(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.UserSubscriptionExpiredPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err := listener.service.ExpireSubscription(ctx, &payload); err != nil {
		msg := fmt.Sprintf("cannot expire subscription for user with ID [%s] for event with ID [%s]", payload.UserID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
