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

// MessageListener handles cloud events which need to update entities.Message
type MessageListener struct {
	listener
	logger  telemetry.Logger
	tracer  telemetry.Tracer
	service *services.MessageService
}

// NewMessageListener creates a new instance of MessageListener
func NewMessageListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.MessageService,
	repository repositories.EventListenerLogRepository,
) (l *MessageListener, routes map[string]events.EventListener) {
	l = &MessageListener{
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
		events.EventTypeMessagePhoneFailed:    l.OnMessagePhoneFailed,
		events.EventTypeMessagePhoneReceived:  l.OnMessagePhoneReceived,
	}
}

// OnMessageAPISent handles the events.EventTypeMessageAPISent event
func (listener *MessageListener) OnMessageAPISent(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	var payload events.MessageAPISentPayload
	if err := event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	storeParams := services.MessageStoreParams{
		Owner:     payload.Owner,
		Contact:   payload.Contact,
		Content:   payload.Content,
		UserID:    payload.UserID,
		ID:        payload.ID,
		Timestamp: payload.RequestReceivedAt,
	}

	if _, err := listener.service.StoreSentMessage(ctx, storeParams); err != nil {
		msg := fmt.Sprintf("cannot store message with ID [%s] for event with ID [%s]", storeParams.ID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return nil
}

// OnMessagePhoneSending handles the events.EventTypeMessagePhoneSending event
func (listener *MessageListener) OnMessagePhoneSending(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	handled, err := listener.repository.Has(ctx, event.ID(), listener.signature(event))
	if err != nil {
		msg := fmt.Sprintf("cannot verify if event [%s] has been handled by [%T]", event.ID(), listener.signature(event))
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger := listener.tracer.CtxLogger(listener.logger, span)

	if handled {
		ctxLogger.Info(fmt.Sprintf("event [%s] has already been handled by [%s]", event.ID(), listener.signature(event)))
		return nil
	}

	var payload events.MessagePhoneSendingPayload
	if err = event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	handleParams := services.HandleMessageParams{
		ID:        payload.ID,
		UserID:    payload.UserID,
		Timestamp: event.Time(),
	}

	if err = listener.service.HandleMessageSending(ctx, handleParams); err != nil {
		msg := fmt.Sprintf("cannot handle sending for message with ID [%s] for event with ID [%s]", handleParams.ID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return listener.storeEventListenerLog(ctx, listener.signature(event), event)
}

// OnMessagePhoneSent handles the events.EventTypeMessagePhoneSent event
func (listener *MessageListener) OnMessagePhoneSent(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	handled, err := listener.repository.Has(ctx, event.ID(), listener.signature(event))
	if err != nil {
		msg := fmt.Sprintf("cannot verify if event [%s] has been handled by [%T]", event.ID(), listener.signature(event))
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger := listener.tracer.CtxLogger(listener.logger, span)

	if handled {
		ctxLogger.Info(fmt.Sprintf("event [%s] has already been handled by [%s]", event.ID(), listener.signature(event)))
		return nil
	}

	var payload events.MessagePhoneSentPayload
	if err = event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	handleParams := services.HandleMessageParams{
		ID:        payload.ID,
		UserID:    payload.UserID,
		Timestamp: payload.Timestamp,
	}

	if err = listener.service.HandleMessageSent(ctx, handleParams); err != nil {
		msg := fmt.Sprintf("cannot handle [%s] for message with ID [%s] for event with ID [%s]", event.Type(), handleParams.ID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return listener.storeEventListenerLog(ctx, listener.signature(event), event)
}

// OnMessagePhoneDelivered handles the events.EventTypeMessagePhoneDelivered event
func (listener *MessageListener) OnMessagePhoneDelivered(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	handled, err := listener.repository.Has(ctx, event.ID(), listener.signature(event))
	if err != nil {
		msg := fmt.Sprintf("cannot verify if event [%s] has been handled by [%T]", event.ID(), listener.signature(event))
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger := listener.tracer.CtxLogger(listener.logger, span)

	if handled {
		ctxLogger.Info(fmt.Sprintf("event [%s] has already been handled by [%s]", event.ID(), listener.signature(event)))
		return nil
	}

	var payload events.MessagePhoneDeliveredPayload
	if err = event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	handleParams := services.HandleMessageParams{
		ID:        payload.ID,
		UserID:    payload.UserID,
		Timestamp: payload.Timestamp,
	}

	if err = listener.service.HandleMessageDelivered(ctx, handleParams); err != nil {
		msg := fmt.Sprintf("cannot handle [%s] for message with ID [%s] for event with ID [%s]", event.Type(), handleParams.ID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return listener.storeEventListenerLog(ctx, listener.signature(event), event)
}

// OnMessagePhoneFailed handles the events.EventTypeMessagePhoneFailed event
func (listener *MessageListener) OnMessagePhoneFailed(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	handled, err := listener.repository.Has(ctx, event.ID(), listener.signature(event))
	if err != nil {
		msg := fmt.Sprintf("cannot verify if event [%s] has been handled by [%T]", event.ID(), listener.signature(event))
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger := listener.tracer.CtxLogger(listener.logger, span)

	if handled {
		ctxLogger.Info(fmt.Sprintf("event [%s] has already been handled by [%s]", event.ID(), listener.signature(event)))
		return nil
	}

	var payload events.MessagePhoneFailedPayload
	if err = event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	handleParams := services.HandleMessageParams{
		ID:        payload.ID,
		UserID:    payload.UserID,
		Timestamp: payload.Timestamp,
	}

	if err = listener.service.HandleMessageFailed(ctx, handleParams); err != nil {
		msg := fmt.Sprintf("cannot handle [%s] for message with ID [%s] for event with ID [%s]", event.Type(), handleParams.ID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return listener.storeEventListenerLog(ctx, listener.signature(event), event)
}

// OnMessagePhoneReceived handles the events.EventTypeMessageAPISent event
func (listener *MessageListener) OnMessagePhoneReceived(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	handled, err := listener.repository.Has(ctx, event.ID(), listener.signature(event))
	if err != nil {
		msg := fmt.Sprintf("cannot verify if event [%s] has been handled by [%T]", event.ID(), listener.signature(event))
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger := listener.tracer.CtxLogger(listener.logger, span)

	if handled {
		ctxLogger.Info(fmt.Sprintf("event [%s] has already been handled by [%s]", event.ID(), listener.signature(event)))
		return nil
	}

	var payload events.MessagePhoneReceivedPayload
	if err = event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	storeParams := services.MessageStoreParams{
		Owner:     payload.Owner,
		Contact:   payload.Contact,
		Content:   payload.Content,
		UserID:    payload.UserID,
		ID:        payload.ID,
		Timestamp: payload.Timestamp,
	}

	if _, err = listener.service.StoreReceivedMessage(ctx, storeParams); err != nil {
		msg := fmt.Sprintf("cannot store message with ID [%s] for event with ID [%s]", storeParams.ID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return listener.storeEventListenerLog(ctx, listener.signature(event), event)
}

func (listener *MessageListener) signature(event cloudevents.Event) string {
	return listener.handlerSignature(listener, event)
}
