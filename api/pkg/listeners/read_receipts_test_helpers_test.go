package listeners

import (
	"context"

	"github.com/NdoleStudio/httpsms/pkg/entities"
	"github.com/NdoleStudio/httpsms/pkg/repositories"
	"github.com/NdoleStudio/httpsms/pkg/telemetry"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

type noopListenerLogger struct{}

func (logger *noopListenerLogger) Error(error)                                 {}
func (logger *noopListenerLogger) WithService(string) telemetry.Logger         { return logger }
func (logger *noopListenerLogger) WithString(string, string) telemetry.Logger  { return logger }
func (logger *noopListenerLogger) WithSpan(trace.SpanContext) telemetry.Logger { return logger }
func (logger *noopListenerLogger) Trace(string)                                {}
func (logger *noopListenerLogger) Info(string)                                 {}
func (logger *noopListenerLogger) Warn(error)                                  {}
func (logger *noopListenerLogger) Debug(string)                                {}
func (logger *noopListenerLogger) Fatal(error)                                 {}
func (logger *noopListenerLogger) Printf(string, ...interface{})               {}

type listenerMessageThreadRepository struct {
	activity repositories.MessageThreadActivityUpdate
}

func (repository *listenerMessageThreadRepository) Store(context.Context, *entities.MessageThread) error {
	return nil
}

func (repository *listenerMessageThreadRepository) UpdateActivity(_ context.Context, params repositories.MessageThreadActivityUpdate) error {
	repository.activity = params
	return nil
}

func (repository *listenerMessageThreadRepository) UpdateStatus(_ context.Context, _ entities.UserID, threadID uuid.UUID, _ repositories.MessageThreadStatusUpdate) (*entities.MessageThread, error) {
	return &entities.MessageThread{ID: threadID}, nil
}

func (repository *listenerMessageThreadRepository) UpdateAfterDeletedMessage(context.Context, repositories.MessageThreadDeletedUpdate) error {
	return nil
}

func (repository *listenerMessageThreadRepository) LoadByOwnerContact(context.Context, entities.UserID, string, string) (*entities.MessageThread, error) {
	return &entities.MessageThread{ID: uuid.New()}, nil
}

func (repository *listenerMessageThreadRepository) Load(context.Context, entities.UserID, uuid.UUID) (*entities.MessageThread, error) {
	return &entities.MessageThread{}, nil
}

func (repository *listenerMessageThreadRepository) Index(context.Context, entities.UserID, string, bool, repositories.IndexParams) (*[]entities.MessageThread, error) {
	threads := []entities.MessageThread{}
	return &threads, nil
}

func (repository *listenerMessageThreadRepository) Delete(context.Context, entities.UserID, uuid.UUID) error {
	return nil
}

func (repository *listenerMessageThreadRepository) DeleteAllForUser(context.Context, entities.UserID) error {
	return nil
}
