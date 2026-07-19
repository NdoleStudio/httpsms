package listeners

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/stacktrace"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/davecgh/go-spew/spew"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
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
		events.EventTypePhoneUpdated:          l.onPhoneUpdated,
		events.EventTypePhoneDeleted:          l.onPhoneDeleted,
		events.EventTypePhoneHeartbeatCheck:   l.onPhoneHeartbeatCheck,
		events.EventTypePhoneHeartbeatOffline: l.onPhoneHeartbeatOffline,
		events.UserAccountDeleted:             l.onUserAccountDeleted,
	}
}

// onPhoneUpdated handles the events.EventTypePhoneUpdated event
func (listener *HeartbeatListener) onPhoneUpdated(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.PhoneUpdatedPayload
	if err := event.DataAs(&payload); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot decode [%s] into [%T]", event.Data(), payload))
	}

	storeParams := &services.HeartbeatMonitorStoreParams{
		Owner:   payload.Owner,
		PhoneID: payload.PhoneID,
		UserID:  payload.UserID,
		Source:  event.Source(),
	}

	if _, err := listener.service.StoreMonitor(ctx, storeParams); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot store heartbeat monitor with params [%s] for event with ID [%s]", spew.Sdump(storeParams), event.ID()))
	}

	return nil
}

// onPhoneDeleted handles the events.EventTypePhoneDeleted event
func (listener *HeartbeatListener) onPhoneDeleted(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.PhoneDeletedPayload
	if err := event.DataAs(&payload); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot decode [%s] into [%T]", event.Data(), payload))
	}

	if err := listener.service.DeleteMonitor(ctx, payload.UserID, payload.Owner); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot delete heartbeat monitor with userID [%s] and owner [%s] for event with ID [%s]", payload.UserID, payload.Owner, event.ID()))
	}

	return nil
}

// onPhoneHeartbeatCheck handles the events.EventTypePhoneHeartbeatCheck event
func (listener *HeartbeatListener) onPhoneHeartbeatCheck(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.PhoneHeartbeatCheckPayload
	if err := event.DataAs(&payload); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot decode [%s] into [%T]", event.Data(), payload))
	}

	monitorParams := &services.HeartbeatMonitorParams{
		Owner:     payload.Owner,
		PhoneID:   payload.PhoneID,
		MonitorID: payload.MonitorID,
		UserID:    payload.UserID,
		Source:    event.Source(),
	}

	if err := listener.service.Monitor(ctx, monitorParams); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot monitor heartbeats for userID [%s] and owner [%s] for event with ID [%s]", payload.UserID, payload.Owner, event.ID()))
	}

	return nil
}

// onPhoneDeleted handles the events.EventTypePhoneDeleted event
func (listener *HeartbeatListener) onPhoneHeartbeatOffline(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.PhoneHeartbeatOfflinePayload
	if err := event.DataAs(&payload); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot decode [%s] into [%T]", event.Data(), payload))
	}

	if err := listener.service.UpdatePhoneOnline(ctx, payload.UserID, payload.MonitorID, false); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot delete heartbeat monitor with userID [%s] and owner [%s] for event with ID [%s]", payload.UserID, payload.Owner, event.ID()))
	}

	return nil
}

func (listener *HeartbeatListener) onUserAccountDeleted(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.UserAccountDeletedPayload
	if err := event.DataAs(&payload); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot decode [%s] into [%T]", event.Data(), payload))
	}

	if err := listener.service.DeleteAllForUser(ctx, payload.UserID); err != nil {
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, "cannot delete [entities.Heartbeat] for user [%s] on [%s] event with ID [%s]", payload.UserID, event.Type(), event.ID()))
	}

	return nil
}
