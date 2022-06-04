package listeners

import (
	"context"
	"fmt"
	"time"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/events"
	"github.com/NdoleStudio/http-sms-manager/pkg/repositories"
	"github.com/NdoleStudio/http-sms-manager/pkg/services"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
)

// MessageListener handles cloud events which need to update the messages table
type MessageListener struct {
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	service    *services.MessageService
	repository repositories.EventListenerLogRepository
}

// NewMessageListener creates a new instance of MessageListener
func NewMessageListener(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	service *services.MessageService,
	repository repositories.EventListenerLogRepository,
) (listener *MessageListener, routes map[string]events.EventListener) {
	listener = &MessageListener{
		logger:     logger.WithService(fmt.Sprintf("%T", listener)),
		tracer:     tracer,
		service:    service,
		repository: repository,
	}

	return listener, map[string]events.EventListener{
		events.EventTypeMessageAPISent: listener.OnMessageAPISent,
	}
}

// OnMessageAPISent handles the events.EventTypeMessageAPISent event
func (listener *MessageListener) OnMessageAPISent(ctx context.Context, event cloudevents.Event) error {
	ctx, span := listener.tracer.Start(ctx)
	defer span.End()

	handlerName := fmt.Sprintf("%s.%T", event.Type(), listener)
	listener.logger.Warn(stacktrace.NewError(handlerName))

	handled, err := listener.repository.Has(ctx, event.ID(), handlerName)
	if err != nil {
		msg := fmt.Sprintf("cannot test if event [%s] has been handled by [%T]", event.ID(), handlerName)
		return listener.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger := listener.tracer.CtxLogger(listener.logger, span)

	if handled {
		ctxLogger.Info(fmt.Sprintf("event [%s] has already been handled by [%s]", event.ID(), handlerName))
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

	return listener.repository.Save(ctx, &entities.EventListenerLog{
		ID:        uuid.New(),
		EventID:   event.ID(),
		EventType: event.Type(),
		Handler:   handlerName,
		Duration:  time.Now().Sub(event.Time()),
		HandledAt: time.Now().UTC(),
		CreatedAt: time.Now().UTC(),
	})
}
