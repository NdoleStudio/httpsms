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

// NotificationListener handles cloud events which sends notifications
type NotificationListener struct {
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.NotificationService
}

// NewNotificationListener creates a new instance of NotificationListener
func NewNotificationListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.NotificationService,
) (l *NotificationListener, routes map[string]events.EventListener) {
	l = &NotificationListener{
		logger:  logger.WithService(fmt.Sprintf("%T", l)),
		tracer:  tracer,
		service: service,
	}

	return l, map[string]events.EventListener{
		events.EventTypeMessageAPISent:               l.onMessageAPISent,
		events.EventTypeMessageNotificationScheduled: l.onMessageNotificationScheduled,
	}
}

// onMessageAPISent handles the events.EventTypeMessageAPISent event
func (listener *NotificationListener) onMessageAPISent(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageAPISentPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	sendParams := &services.NotificationScheduleParams{
		UserID:    payload.UserID,
		Owner:     payload.Owner,
		Source:    event.Source(),
		MessageID: payload.ID,
	}

	if err := listener.service.Schedule(ctx, sendParams); err != nil {
		msg := fmt.Sprintf("cannot send notification with params [%s] for event with ID [%s]", spew.Sdump(sendParams), event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// onMessageNotificationScheduled handles the events.EventTypeMessageNotificationScheduled event
func (listener *NotificationListener) onMessageNotificationScheduled(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageNotificationScheduledPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	scheduleParams := &services.NotificationSendParams{
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
