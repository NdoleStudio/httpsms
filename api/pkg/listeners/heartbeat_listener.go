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

// HeartbeatListener handles cloud events which need to register entities.Heartbeat
type HeartbeatListener struct {
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.HeartbeatService
}

// NewHeartbeatListener creates a new instance of HeartbeatListener
func NewHeartbeatListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.HeartbeatService,
) (l *HeartbeatListener, routes map[string]events.EventListener) {
	l = &HeartbeatListener{
		logger:  logger.WithService(fmt.Sprintf("%T", l)),
		tracer:  tracer,
		service: service,
	}

	return l, map[string]events.EventListener{
		events.EventTypeMessagePhoneSending: l.onMessagePhoneSending,
	}
}

// onMessagePhoneSending handles the events.EventTypeMessagePhoneSending event
func (listener *HeartbeatListener) onMessagePhoneSending(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessagePhoneSendingPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	storeParams := services.HeartbeatStoreParams{
		Owner:     payload.Owner,
		Timestamp: payload.Timestamp,
		UserID:    payload.UserID,
		MessageID: payload.ID,
	}

	if _, err := listener.service.Store(ctx, storeParams); err != nil {
		msg := fmt.Sprintf("cannot store heartbeat with params [%+#v] for event with MessageID [%s]", storeParams, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
