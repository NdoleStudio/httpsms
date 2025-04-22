package listeners

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/entities"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/davecgh/go-spew/spew"
	"github.com/palantir/stacktrace"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
)

// PhoneAPIKeyListener handles cloud events that alter the state of entities.PhoneAPIKey
type PhoneAPIKeyListener struct {
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.PhoneAPIKeyService
}

// NewPhoneAPIKeyListener creates a new instance of PhoneAPIKeyListener
func NewPhoneAPIKeyListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.PhoneAPIKeyService,
) (l *PhoneAPIKeyListener, routes map[string]events.EventListener) {
	l = &PhoneAPIKeyListener{
		logger:  logger.WithService(fmt.Sprintf("%T", l)),
		tracer:  tracer,
		service: service,
	}

	return l, map[string]events.EventListener{
		events.EventTypePhoneUpdated: l.onPhoneUpdated,
		events.EventTypePhoneDeleted: l.onPhoneDeleted,
		events.UserAccountDeleted:    l.onUserAccountDeleted,
	}
}

// onPhoneUpdated handles the events.EventTypePhoneUpdated event
func (listener *PhoneAPIKeyListener) onPhoneUpdated(ctx context.Context, event cloudevents.Event) error {
	ctx, span, ctxLogger := listener.tracer.StartWithLogger(ctx, listener.logger)
	defer span.End()

	var payload events.PhoneUpdatedPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if payload.PhoneAPIKeyID == nil {
		ctxLogger.Info(fmt.Sprintf("phone API Key does not exist for [%s] event with ID [%s] and phone with ID [%s] for user [%S]", event.Type(), event.ID(), payload.PhoneID, payload.UserID))
		return nil
	}

	if err := listener.service.AddPhone(ctx, payload.UserID, *payload.PhoneAPIKeyID, payload.PhoneID); err != nil {
		msg := fmt.Sprintf("cannot store heartbeat monitor with params [%s] for event with ID [%s]", spew.Sdump(payload), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// onPhoneUpdated handles the events.EventTypePhoneUpdated event
func (listener *PhoneAPIKeyListener) onPhoneDeleted(ctx context.Context, event cloudevents.Event) error {
	ctx, span, _ := listener.tracer.StartWithLogger(ctx, listener.logger)
	defer span.End()

	var payload events.PhoneDeletedPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err := listener.service.RemovePhoneByID(ctx, payload.UserID, payload.PhoneID, payload.Owner); err != nil {
		msg := fmt.Sprintf("cannot remove phone with ID [%s] from phone api key for [%s] event with ID [%s]", payload.PhoneID, event.Type(), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// onUserAccountDeleted handles the events.EventTypePhoneUpdated event
func (listener *PhoneAPIKeyListener) onUserAccountDeleted(ctx context.Context, event cloudevents.Event) error {
	ctx, span, _ := listener.tracer.StartWithLogger(ctx, listener.logger)
	defer span.End()

	var payload events.UserAccountDeletedPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err := listener.service.DeleteAllForUser(ctx, payload.UserID); err != nil {
		msg := fmt.Sprintf("cannot delete all [%T] for user with ID [%s] for [%s] event with ID [%s]", entities.PhoneAPIKey{}, payload.UserID, event.Type(), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
