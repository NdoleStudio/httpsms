package services

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/events"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
)

// MessageThreadService is handles message requests
type MessageThreadService struct {
	service
	logger          telemetry.Logger
	tracer          telemetry.Tracer
	repository      repositories.MessageThreadRepository
	eventDispatcher *EventDispatcher
}

// NewMessageThreadService creates a new MessageThreadService
func NewMessageThreadService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.MessageThreadRepository,
	eventDispatcher *EventDispatcher,
) (s *MessageThreadService) {
	return &MessageThreadService{
		logger:          logger.WithService(fmt.Sprintf("%T", s)),
		tracer:          tracer,
		eventDispatcher: eventDispatcher,
		repository:      repository,
	}
}

// MessageThreadUpdateParams are parameters for updating a thread
type MessageThreadUpdateParams struct {
	Owner     string
	Status    entities.MessageStatus
	Contact   string
	Content   string
	UserID    entities.UserID
	MessageID uuid.UUID
	Timestamp time.Time
}

// UpdateThread updates a thread between 2 parties when a timestamp changes
func (service *MessageThreadService) UpdateThread(ctx context.Context, params MessageThreadUpdateParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	thread, err := service.repository.LoadByOwnerContact(ctx, params.UserID, params.Owner, params.Contact)
	if stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		ctxLogger.Info(fmt.Sprintf("cannot find thread with owner [%s], and contact [%s]. creating new thread", params.Owner, params.Contact))
		return service.createThread(ctx, params)
	}

	if err != nil {
		msg := fmt.Sprintf("cannot find thread with owner [%s], and contact [%s]. creating new thread", params.Owner, params.Contact)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if thread.OrderTimestamp.Unix() > params.Timestamp.Unix() && thread.Status != entities.MessageStatusSending && thread.HasLastMessage(params.MessageID) {
		ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("thread [%s] has timestamp [%s] and status [%s] which is greater than timestamp [%s] for message [%s] and status [%s]", thread.ID, thread.OrderTimestamp, thread.Status, params.Timestamp, params.MessageID, params.Status)))
		return nil
	}

	if thread.Status == entities.MessageStatusDelivered && thread.LastMessageID != nil && thread.HasLastMessage(params.MessageID) {
		ctxLogger.Warn(stacktrace.NewError(fmt.Sprintf("thread [%s] already has status [%s] not updating with status [%s] for message [%s]", thread.ID, thread.Status, params.Status, params.MessageID)))
		return nil
	}

	if err = service.repository.Update(ctx, thread.Update(params.Timestamp, params.MessageID, params.Content, params.Status)); err != nil {
		msg := fmt.Sprintf("cannot update message thread with id [%s] after adding message [%s]", thread.ID, params.MessageID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("thread with id [%s] updated with last message [%s] and status [%s]", thread.ID, thread.LastMessageID, thread.Status))
	return nil
}

// MessageThreadStatusParams are parameters for updating a thread status
type MessageThreadStatusParams struct {
	IsArchived      bool
	UserID          entities.UserID
	MessageThreadID uuid.UUID
}

// UpdateStatus updates a thread between an owner and a contact
func (service *MessageThreadService) UpdateStatus(ctx context.Context, params MessageThreadStatusParams) (*entities.MessageThread, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	thread, err := service.repository.Load(ctx, params.UserID, params.MessageThreadID)
	if err != nil {
		msg := fmt.Sprintf("cannot find thread with id [%s]", params.MessageThreadID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if err = service.repository.Update(ctx, thread.UpdateArchive(params.IsArchived)); err != nil {
		msg := fmt.Sprintf("cannot update message thread with id [%s] with archive status [%t]", thread.ID, params.IsArchived)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("thread with id [%s] updated with archive status [%t]", thread.ID, thread.IsArchived))
	return thread, nil
}

// UpdateAfterDeletedMessage updates a thread after the last message has been deleted
func (service *MessageThreadService) UpdateAfterDeletedMessage(ctx context.Context, payload *events.MessageAPIDeletedPayload) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	thread, err := service.repository.LoadByOwnerContact(ctx, payload.UserID, payload.Owner, payload.Contact)
	if err != nil {
		msg := fmt.Sprintf("cannot find thread for user [%s] with owner [%s], and contact [%s]", payload.UserID, payload.Owner, payload.Contact)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if payload.PreviousMessageID == nil {
		if err = service.repository.Delete(ctx, thread.UserID, thread.ID); err != nil {
			msg := fmt.Sprintf("cannot delete thread with ID [%s] for user [%s] and owner [%s]", thread.ID, thread.UserID, thread.Owner)
			ctxLogger.Error(stacktrace.Propagate(err, msg))
			return nil
		}
		msg := fmt.Sprintf("previous message ID is nil for thread with ID [%s] and user [%s]", thread.ID, thread.UserID)
		ctxLogger.Info(msg)
		return nil
	}

	if thread.LastMessageID != nil && *thread.LastMessageID != payload.MessageID {
		msg := fmt.Sprintf("last message ID [%s] does not match message ID [%s] for thread with ID [%s]", *thread.LastMessageID, payload.MessageID, thread.ID)
		ctxLogger.Info(msg)
		return nil
	}

	thread.LastMessageContent = payload.PreviousMessageContent
	thread.LastMessageID = payload.PreviousMessageID
	thread.Status = *payload.PreviousMessageStatus
	thread.UpdatedAt = time.Now().UTC()

	if err = service.repository.Update(ctx, thread); err != nil {
		msg := fmt.Sprintf("cannot update thread with ID [%s] for user with ID [%s]", thread.ID, thread.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("last message has been removed from thread with ID [%s] and userID [%s]", thread.ID, thread.UserID))
	return nil
}

func (service *MessageThreadService) createThread(ctx context.Context, params MessageThreadUpdateParams) error {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	thread := &entities.MessageThread{
		ID:                 uuid.New(),
		Owner:              params.Owner,
		Contact:            params.Contact,
		UserID:             params.UserID,
		IsArchived:         false,
		Color:              service.getColor(),
		LastMessageContent: &params.Content,
		Status:             params.Status,
		LastMessageID:      &params.MessageID,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
		OrderTimestamp:     params.Timestamp,
	}

	if err := service.repository.Store(ctx, thread); err != nil {
		msg := fmt.Sprintf("cannot store thread with id [%s] for message with ID [%s]", thread.ID, params.MessageID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf(
		"created thread [%s] for message ID [%s] with owner [%s] and contact [%s]",
		thread.ID,
		thread.LastMessageID,
		thread.Owner,
		thread.Contact,
	))

	return nil
}

func (service *MessageThreadService) getColor() string {
	colors := []string{
		"deep-purple",
		"indigo",
		"blue",
		"red",
		"pink",
		"purple",
		"light-blue",
		"cyan",
		"teal",
		"green",
		"light-green",
		"lime",
		"yellow",
		"amber",
		"orange",
		"deep-orange",
		"brown",
	}
	generator := rand.New(rand.NewSource(time.Now().UnixNano()))
	return colors[generator.Intn(len(colors))]
}

// MessageThreadGetParams parameters fetching threads
type MessageThreadGetParams struct {
	repositories.IndexParams
	IsArchived bool
	UserID     entities.UserID
	Owner      string
}

// GetThreads fetches threads for an owner
func (service *MessageThreadService) GetThreads(ctx context.Context, params MessageThreadGetParams) (*[]entities.MessageThread, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	ctxLogger := service.tracer.CtxLogger(service.logger, span)

	threads, err := service.repository.Index(ctx, params.UserID, params.Owner, params.IsArchived, params.IndexParams)
	if err != nil {
		msg := fmt.Sprintf("could not fetch messages threads for params [%+#v]", params)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("fetched [%d] threads with params [%+#v]", len(*threads), params))
	return threads, nil
}

// GetThread fetches an entities.MessageThread  message thread by the ID
func (service *MessageThreadService) GetThread(ctx context.Context, userID entities.UserID, messageThreadID uuid.UUID) (*entities.MessageThread, error) {
	ctx, span := service.tracer.Start(ctx)
	defer span.End()

	thread, err := service.repository.Load(ctx, userID, messageThreadID)
	if err != nil {
		msg := fmt.Sprintf("could not fetch thread with ID [%s] for user [%s]", messageThreadID, userID)
		return nil, service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	return thread, nil
}

// DeleteThread deletes an entities.MessageThread from the database
func (service *MessageThreadService) DeleteThread(ctx context.Context, source string, thread *entities.MessageThread) error {
	ctx, span, ctxLogger := service.tracer.StartWithLogger(ctx, service.logger)
	defer span.End()

	if err := service.repository.Delete(ctx, thread.UserID, thread.ID); err != nil {
		msg := fmt.Sprintf("could not delete message thread with ID [%s] for user with ID [%s]", thread.ID, thread.UserID)
		return service.tracer.WrapErrorSpan(span, stacktrace.PropagateWithCode(err, stacktrace.GetCode(err), msg))
	}

	event, err := service.createEvent(events.MessageThreadAPIDeleted, source, &events.MessageThreadAPIDeletedPayload{
		MessageThreadID: thread.ID,
		UserID:          thread.UserID,
		Owner:           thread.Owner,
		Contact:         thread.Contact,
		IsArchived:      thread.IsArchived,
		Color:           thread.Color,
		Status:          thread.Status,
		Timestamp:       time.Now().UTC(),
	})
	if err != nil {
		msg := fmt.Sprintf("cannot create [%T] for message thread dleted with ID [%s]", event, thread.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("created event [%s] with id [%s] for message thread [%s]", event.Type(), event.ID(), thread.ID))
	if err = service.eventDispatcher.Dispatch(ctx, event); err != nil {
		msg := fmt.Sprintf("cannot dispatch event [%s] with id [%s] for message thread [%s]", event.Type(), event.ID(), thread.ID)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	ctxLogger.Info(fmt.Sprintf("dispatched [%s] event with id [%s] for message thread [%s]", event.Type(), event.ID(), thread.ID))
	return nil
}
