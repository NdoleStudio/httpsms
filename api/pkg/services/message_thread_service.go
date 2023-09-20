package services

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"github.com/palantir/stacktrace"
)

// MessageThreadService is handles message requests
type MessageThreadService struct {
	logger     telemetry.Logger
	tracer     telemetry.Tracer
	repository repositories.MessageThreadRepository
}

// NewMessageThreadService creates a new MessageThreadService
func NewMessageThreadService(
	logger telemetry.Logger,
	tracer telemetry.Tracer,
	repository repositories.MessageThreadRepository,
) (s *MessageThreadService) {
	return &MessageThreadService{
		logger:     logger.WithService(fmt.Sprintf("%T", s)),
		tracer:     tracer,
		repository: repository,
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
	if err != nil && stacktrace.GetCode(err) == repositories.ErrCodeNotFound {
		ctxLogger.Info(fmt.Sprintf("cannot find thread with owner [%s], and contact [%s]. creating new thread", params.Owner, params.Contact))
		return service.createThread(ctx, params)
	}

	if err != nil {
		msg := fmt.Sprintf("cannot find thread with owner [%s], and contact [%s]. creating new thread", params.Owner, params.Contact)
		return service.tracer.WrapErrorSpan(span, stacktrace.Propagate(err, msg))
	}

	if thread.OrderTimestamp.Unix() > params.Timestamp.Unix() && thread.Status != entities.MessageStatusSending {
		ctxLogger.Info(fmt.Sprintf("thread [%s] has timestamp [%s] and status [%s] which is greater than timestamp [%s] for message [%s] and status [%s]", thread.ID, thread.OrderTimestamp, thread.Status, params.Timestamp, params.MessageID, params.Status))
		return nil
	}

	thread = thread.Update(params.Timestamp, params.MessageID, params.Content, params.Status)
	if err = service.repository.Update(ctx, thread); err != nil {
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
		LastMessageContent: params.Content,
		Status:             params.Status,
		LastMessageID:      params.MessageID,
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
	rand.Seed(time.Now().UnixNano())
	return colors[rand.Intn(len(colors))]
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
