package listeners

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/httpsms/pkg/events"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/services"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/palantir/stacktrace"
)

// MessageThreadListener handles cloud events which need to update entities.MessageThread
type MessageThreadListener struct {
	listener
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.MessageThreadService
}

// NewMessageThreadListener creates a new instance of MessageThreadListener
func NewMessageThreadListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.MessageThreadService,
	repository repositories.EventListenerLogRepository,
) (l *MessageThreadListener, routes map[string]events.EventListener) {
	l = &MessageThreadListener{
		logger:  logger.WithService(fmt.Sprintf("%T", l)),
		tracer:  tracer,
		service: service,
		listener: listener{
			repository: repository,
		},
	}

	return l, map[string]events.EventListener{
		events.EventTypeMessageAPISent:        l.OnMessageAPISent,
		events.EventTypeMessagePhoneSending:   l.OnMessagePhoneSending,
		events.EventTypeMessagePhoneSent:      l.OnMessagePhoneSent,
		events.EventTypeMessagePhoneDelivered: l.OnMessagePhoneDelivered,
		events.EventTypeMessageSendFailed:     l.OnMessagePhoneFailed,
		events.EventTypeMessagePhoneReceived:  l.OnMessagePhoneReceived,
	}
}

// OnMessageAPISent handles the events.EventTypeMessageAPISent event
func (listener *MessageThreadListener) OnMessageAPISent(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageAPISentPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	updateParams := services.MessageThreadUpdateParams{
		Owner:     payload.Owner,
		Contact:   payload.Contact,
		UserID:    payload.UserID,
		Timestamp: payload.RequestReceivedAt,
		Content:   payload.Content,
		MessageID: payload.ID,
	}

	if err := listener.service.UpdateThread(ctx, updateParams); err != nil {
		msg := fmt.Sprintf("cannot update thread for message with ID [%s] for event with ID [%s]", updateParams.MessageID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// OnMessagePhoneSending handles the events.EventTypeMessagePhoneSending event
func (listener *MessageThreadListener) OnMessagePhoneSending(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessagePhoneSendingPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	updateParams := services.MessageThreadUpdateParams{
		Owner:     payload.Owner,
		UserID:    payload.UserID,
		Contact:   payload.Contact,
		Timestamp: event.Time(),
		Content:   payload.Content,
		MessageID: payload.ID,
	}

	if err := listener.service.UpdateThread(ctx, updateParams); err != nil {
		msg := fmt.Sprintf("cannot update thread for message with ID [%s] for event with ID [%s]", updateParams.MessageID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// OnMessagePhoneSent handles the events.EventTypeMessagePhoneSent event
func (listener *MessageThreadListener) OnMessagePhoneSent(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessagePhoneSentPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	updateParams := services.MessageThreadUpdateParams{
		Owner:     payload.Owner,
		Contact:   payload.Contact,
		UserID:    payload.UserID,
		Timestamp: payload.Timestamp,
		Content:   payload.Content,
		MessageID: payload.ID,
	}

	if err := listener.service.UpdateThread(ctx, updateParams); err != nil {
		msg := fmt.Sprintf("cannot update thread for message with ID [%s] for event with ID [%s]", updateParams.MessageID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// OnMessagePhoneDelivered handles the events.EventTypeMessagePhoneDelivered event
func (listener *MessageThreadListener) OnMessagePhoneDelivered(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessagePhoneDeliveredPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	updateParams := services.MessageThreadUpdateParams{
		Owner:     payload.Owner,
		UserID:    payload.UserID,
		Contact:   payload.Contact,
		Timestamp: payload.Timestamp,
		Content:   payload.Content,
		MessageID: payload.ID,
	}

	if err := listener.service.UpdateThread(ctx, updateParams); err != nil {
		msg := fmt.Sprintf("cannot update thread for message with ID [%s] for event with ID [%s]", updateParams.MessageID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// OnMessagePhoneFailed handles the events.EventTypeMessageSendFailed event
func (listener *MessageThreadListener) OnMessagePhoneFailed(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessagePhoneDeliveredPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	updateParams := services.MessageThreadUpdateParams{
		Owner:     payload.Owner,
		Contact:   payload.Contact,
		UserID:    payload.UserID,
		Timestamp: payload.Timestamp,
		Content:   payload.Content,
		MessageID: payload.ID,
	}

	if err := listener.service.UpdateThread(ctx, updateParams); err != nil {
		msg := fmt.Sprintf("cannot update thread for message with ID [%s] for event with ID [%s]", updateParams.MessageID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// OnMessagePhoneReceived handles the events.EventTypeMessagePhoneReceived event
func (listener *MessageThreadListener) OnMessagePhoneReceived(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessagePhoneReceivedPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	updateParams := services.MessageThreadUpdateParams{
		Owner:     payload.Owner,
		Contact:   payload.Contact,
		Timestamp: event.Time(),
		UserID:    payload.UserID,
		Content:   payload.Content,
		MessageID: payload.ID,
	}

	if err := listener.service.UpdateThread(ctx, updateParams); err != nil {
		msg := fmt.Sprintf("cannot update thread for message with ID [%s] for event with ID [%s]", updateParams.MessageID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

func (listener *MessageThreadListener) updateThread(ctx context.Context, params services.MessageThreadUpdateParams) error {
	return listener.service.UpdateThread(ctx, params)
}
