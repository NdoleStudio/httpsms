package listeners

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/NdoleStudio/stacktrace"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// DiscordListener sends messages to discord
type DiscordListener struct {
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.DiscordService
}

// NewDiscordListener creates a new instance of DiscordListener
func NewDiscordListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.DiscordService,
) (l *DiscordListener, routes map[string]events.EventListener) {
	l = &DiscordListener{
		logger:  logger.WithService(fmt.Sprintf("%T", l)),
		tracer:  tracer,
		service: service,
	}

	return l, map[string]events.EventListener{
		events.EventTypeMessagePhoneReceived: l.OnMessagePhoneReceived,
		events.UserAccountDeleted:            l.onUserAccountDeleted,
	}
}

// OnMessagePhoneReceived handles the events.EventTypeMessagePhoneReceived event
func (listener *DiscordListener) OnMessagePhoneReceived(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessagePhoneReceivedPayload
	if err := event.DataAs(&payload); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot decode [%s] into [%T]", event.Data(), payload))
	}

	if err := listener.service.HandleMessageReceived(ctx, payload.UserID, event); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot process [%s] event with ID [%s]", event.Type(), event.ID()))
	}

	return nil
}

func (listener *DiscordListener) onUserAccountDeleted(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.UserAccountDeletedPayload
	if err := event.DataAs(&payload); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot decode [%s] into [%T]", event.Data(), payload))
	}

	if err := listener.service.DeleteAllForUser(ctx, payload.UserID); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot delete [entities.Discord] for user [%s] on [%s] event with ID [%s]", payload.UserID, event.Type(), event.ID()))
	}

	return nil
}
