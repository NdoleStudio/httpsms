package listeners

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/http-sms-manager/pkg/events"
	"github.com/NdoleStudio/http-sms-manager/pkg/services"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
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
		events.EventTypeHeartbeatPhoneOutstanding: l.onHeartbeatPhoneOutstanding,
	}
}

// onHeartbeatPhoneOutstanding handles the events.EventTypeHeartbeatPhoneOutstanding event
func (listener *HeartbeatListener) onHeartbeatPhoneOutstanding(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.HeartbeatPhoneOutstandingPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	storeParams := services.HeartbeatStoreParams{
		Owner:     payload.Owner,
		Timestamp: payload.Timestamp,
		Quantity:  payload.Quantity,
	}

	if _, err := listener.service.Store(ctx, storeParams); err != nil {
		msg := fmt.Sprintf("cannot store heartbeat with params [%+#v] for event with ID [%s]", storeParams, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
