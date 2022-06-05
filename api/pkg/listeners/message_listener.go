package listeners

import (
	"context"
	"fmt"

	"github.com/NdoleStudio/http-sms-manager/pkg/events"
	"github.com/NdoleStudio/http-sms-manager/pkg/repositories"
	"github.com/NdoleStudio/http-sms-manager/pkg/services"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/palantir/stacktrace"
)

// MessageListener handles cloud events which need to update the messages table
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
		events.EventTypeMessageAPISent:      l.OnMessageAPISent,
		events.EventTypeMessagePhoneSending: l.OnMessagePhoneSending,
	}
}

// OnMessageAPISent handles the events.EventTypeMessageAPISent event
func (listener *MessageListener) OnMessageAPISent(ctx context.Context, event cloudevents.Event) error {
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

	var payload events.MessageAPISentPayload
	if err = event.DataAs(&payload); err != nil {
		msg := fmt.Sprintf("cannot decode [%s] into [%T]", event.Data(), payload)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	storeParams := services.MessageStoreParams{
		From:              payload.From,
		To:                payload.To,
		Content:           payload.Content,
		ID:                payload.ID,
		Source:            event.Source(),
		RequestReceivedAt: payload.RequestReceivedAt,
	}

	if _, err = listener.service.StoreMessage(ctx, storeParams); err != nil {
		msg := fmt.Sprintf("cannot store message with ID [%s] for event with ID [%s]", storeParams.ID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return listener.storeEventListenerLog(ctx, listener.signature(event), event)
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

	handleParams := services.HandleMessageSendingParams{
		ID:        payload.ID,
		Timestamp: event.Time(),
	}

	if err = listener.service.HandleMessageSending(ctx, handleParams); err != nil {
		msg := fmt.Sprintf("cannot handle sending for message with ID [%s] for event with ID [%s]", handleParams.ID, event.ID())
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	return listener.storeEventListenerLog(ctx, listener.signature(event), event)
}

func (listener *MessageListener) signature(event cloudevents.Event) string {
	return listener.handlerSignature(listener, event)
}
