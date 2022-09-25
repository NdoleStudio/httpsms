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

// PhoneNotificationListener handles cloud events which sends notifications
type PhoneNotificationListener struct {
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.PhoneNotificationService
}

// NewNotificationListener creates a new instance of PhoneNotificationListener
func NewNotificationListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.PhoneNotificationService,
) (l *PhoneNotificationListener, routes map[string]events.EventListener) {
	l = &PhoneNotificationListener{
		logger:  logger.WithService(fmt.Sprintf("%T", l)),
		tracer:  tracer,
		service: service,
	}

	return l, map[string]events.EventListener{
		events.EventTypeMessageAPISent:          l.onMessageAPISent,
		events.EventTypeMessageSendRetry:        l.onMessageSendRetry,
		events.EventTypeMessageNotificationSend: l.onMessageNotificationSend,
	}
}

// onMessageAPISent handles the events.EventTypeMessageAPISent event
func (listener *PhoneNotificationListener) onMessageAPISent(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageAPISentPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	sendParams := &services.PhoneNotificationScheduleParams{
		UserID:    payload.UserID,
		Owner:     payload.Owner,
		Contact:   payload.Contact,
		Content:   payload.Content,
		Source:    event.Source(),
		MessageID: payload.ID,
	}

	if err := listener.service.Schedule(ctx, sendParams); err != nil {
		msg := fmt.Sprintf("cannot send notification with params [%s] for event with ID [%s]", spew.Sdump(sendParams), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// onMessageSendRetry handles the events.EventTypeMessageSendRetry event
func (listener *PhoneNotificationListener) onMessageSendRetry(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageSendRetryPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	sendParams := &services.PhoneNotificationScheduleParams{
		UserID:    payload.UserID,
		Owner:     payload.Owner,
		Contact:   payload.Contact,
		Content:   payload.Content,
		Source:    event.Source(),
		MessageID: payload.MessageID,
	}

	if err := listener.service.Schedule(ctx, sendParams); err != nil {
		msg := fmt.Sprintf("cannot send notification with params [%s] for event with ID [%s]", spew.Sdump(sendParams), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// onMessageNotificationSend handles the events.EventTypeMessageNotificationSend event
func (listener *PhoneNotificationListener) onMessageNotificationSend(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageNotificationSendPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	scheduleParams := &services.PhoneNotificationSendParams{
		UserID:              payload.UserID,
		PhoneID:             payload.PhoneID,
		Source:              event.Source(),
		ScheduledAt:         payload.ScheduledAt,
		PhoneNotificationID: payload.NotificationID,
		MessageID:           payload.MessageID,
	}

	if err := listener.service.Send(ctx, scheduleParams); err != nil {
		msg := fmt.Sprintf("cannot schedule notification with params [%s] for event with ID [%s]", spew.Sdump(scheduleParams), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}
