package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/NdoleStudio/http-sms-manager/pkg/events"
	"github.com/NdoleStudio/http-sms-manager/pkg/repositories"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"

	"github.com/NdoleStudio/http-sms-manager/pkg/entities"
	"github.com/NdoleStudio/http-sms-manager/pkg/telemetry"
)

// MessageService is handles message requests
type MessageService struct {
	logger          telemetry.Logger
	tracer          telemetry.Tracer
	eventDispatcher *EventDispatcher
	repository      repositories.MessageRepository
}

// NewMessageService creates a new MessageService
func NewMessageService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.MessageRepository,
	eventDispatcher *EventDispatcher,
) (s *MessageService) {
	return &MessageService{
		logger:          logger.WithService(fmt.Sprintf("%T", s)),
		tracer:          tracer,
		repository:      repository,
		eventDispatcher: eventDispatcher,
	}
}

// MessageGetOutstandingParams parameters for sending a new message
type MessageGetOutstandingParams struct {
	Source string
	Limit  int
}

// GetOutstanding fetches messages that still to be sent to the phone
func (service *MessageService) GetOutstanding(ctx context.Context, params MessageGetOutstandingParams) (*[]entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	messages, err := service.repository.GetOutstanding(ctx, params.Limit)
	if err != nil {
		msg := fmt.Sprintf("could not fetch [%d] outstanding messages", params.Limit)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("fetched [%d] outstanding messages", len(*messages)))
	return service.handleOutstandingMessages(ctx, params.Source, messages), nil
}

// MessageGetParams parameters for sending a new message
type MessageGetParams struct {
	repositories.IndexParams
	From string
	To   string
}

// GetMessages fetches sent between 2 phone numbers
func (service *MessageService) GetMessages(ctx context.Context, params MessageGetParams) (*[]entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	messages, err := service.repository.Index(ctx, params.From, params.To, params.IndexParams)
	if err != nil {
		msg := fmt.Sprintf("could not fetch messages with parms [%+#v]", params)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("fetched [%d] messages with prams [%+#v]", len(*messages), params))
	return messages, nil
}

func (service *MessageService) handleOutstandingMessages(ctx context.Context, source string, messages *[]entities.Message) *[]entities.Message {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	var wg sync.WaitGroup
	results := make([]entities.Message, 0, len(*messages))
	var lock sync.Mutex

	for _, message := range *messages {
		wg.Add(1)
		go func(ctx context.Context, message entities.Message) {
			defer wg.Done()

			event, err := service.createMessagePhoneSendingEvent(source, events.MessagePhoneSendingPayload{
				ID:      message.ID,
				From:    message.From,
				To:      message.To,
				Content: message.Content,
			})
			if err != nil {
				msg := fmt.Sprintf("cannot create [%T] for message with ID [%s]", event, message.ID)
				ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
				return
			}

			ctxLogger.Info(fmt.Sprintf("created event [%s] with id [%s] for message [%s]", event.Type(), event.ID(), message.ID))

			if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
				msg := fmt.Sprintf("cannot dispatch event [%s] with id [%s] for message [%s]", event.Type(), event.ID(), message.ID)
				ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
				return
			}

			ctxLogger.Info(fmt.Sprintf("dispatched event [%s] with id [%s] for message [%s]", event.Type(), event.ID(), message.ID))

			resultMessage, err := service.repository.Load(ctx, message.ID)
			if err != nil {
				msg := fmt.Sprintf("cannot load message with id [%s]", message.ID)
				ctxLogger.Error(service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg)))
				return
			}

			ctxLogger.Info(fmt.Sprintf("loaded message [%s]", message.ID))

			lock.Lock()
			defer lock.Unlock()
			results = append(results, *resultMessage)
		}(ctx, message)
	}

	wg.Wait()
	return &results
}

// MessageSendParams parameters for sending a new message
type MessageSendParams struct {
	From              string
	To                string
	Content           string
	Source            string
	RequestReceivedAt time.Time
}

// SendMessage a new message
func (service *MessageService) SendMessage(ctx context.Context, params MessageSendParams) (*entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	eventPayload := events.MessageAPISentPayload{
		ID:                uuid.New(),
		From:              params.From,
		To:                params.To,
		RequestReceivedAt: params.RequestReceivedAt,
		Content:           params.Content,
	}

	ctxLogger.Info(fmt.Sprintf("creating cloud event for message with ID [%s]", eventPayload.ID))

	event, err := service.createMessageAPISentEvent(params.Source, eventPayload)
	if err != nil {
		msg := fmt.Sprintf("cannot create %T from payload with message id [%s]", event)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("created event [%s] with id [%s] and message id [%s]", event.Type(), event.ID(), eventPayload.ID))

	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch event type [%s] and id [%s]", event.Type(), event.ID())
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("event [%s] dispatched succesfully", event.ID()))

	message, err := service.repository.Load(ctx, eventPayload.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot load message with ID [%s] in the repository", eventPayload.ID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("fetched message with id [%s] from the repository", message.ID))

	return message, nil
}

// MessageStoreParams are parameters for creating a new message
type MessageStoreParams struct {
	From              string
	To                string
	Content           string
	ID                uuid.UUID
	Source            string
	RequestReceivedAt time.Time
}

// StoreMessage a new message
func (service *MessageService) StoreMessage(ctx context.Context, params MessageStoreParams) (*entities.Message, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	message := &entities.Message{
		ID:                params.ID,
		From:              params.From,
		To:                params.To,
		Content:           params.Content,
		Type:              entities.MessageTypeMobileTerminated,
		Status:            entities.MessageStatusPending,
		RequestReceivedAt: params.RequestReceivedAt,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
		OrderTimestamp:    params.RequestReceivedAt,
		SendDuration:      nil,
		LastAttemptedAt:   nil,
		SentAt:            nil,
		ReceivedAt:        nil,
	}

	if err := service.repository.Store(ctx, message); err != nil {
		msg := fmt.Sprintf("cannot save message with id [%s]", params.ID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("message saved with id [%s] in the repository", message.ID))
	return message, nil
}

// HandleMessageSendingParams are parameters for registering a new message being sent
type HandleMessageSendingParams struct {
	ID        uuid.UUID
	Timestamp time.Time
}

// HandleMessageSending handles when a message is being sent
func (service *MessageService) HandleMessageSending(ctx context.Context, params HandleMessageSendingParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	message, err := service.repository.Load(ctx, params.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot find message with id [%s]", params.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if !message.IsSending() {
		msg := fmt.Sprintf("message has wrong status [%s]. expected %s", message.Status, entities.MessageStatusSending)
		return service.tracer.WrapErrorSpan(span, stacktrace.NewError(msg))
	}

	if err = service.repository.Update(ctx, message.AddSendAttempt(params.Timestamp)); err != nil {
		msg := fmt.Sprintf("cannot update message with id [%s] after sending", message.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("message with id [%s] in the repository after adding send attempt", message.ID))
	return nil
}

func (service *MessageService) createMessageAPISentEvent(source string, payload events.MessageAPISentPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypeMessageAPISent, source, payload)
}

func (service *MessageService) createMessagePhoneSendingEvent(source string, payload events.MessagePhoneSendingPayload) (cloudevents.Event, error) {
	return service.createEvent(events.EventTypeMessagePhoneSending, source, payload)
}

func (service *MessageService) createEvent(eventType string, source string, payload any) (cloudevents.Event, error) {
	event := cloudevents.NewEvent()

	event.SetSource(source)
	event.SetType(eventType)
	event.SetTime(time.Now().UTC())
	event.SetID(uuid.New().String())

	if err := event.SetData(cloudevents.ApplicationJSON, payload); err != nil {
		msg := fmt.Sprintf("cannot encode %T [%#+v] as JSON", payload, payload)
		return event, stacktrace.Propagate(err, msg)
	}

	return event, nil
}
